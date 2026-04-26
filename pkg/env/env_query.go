package env

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"
	"github.com/codingconcepts/edg/pkg/db"
	"github.com/codingconcepts/edg/pkg/output"
	"github.com/expr-lang/expr"
)

// RunQuery executes a query against the given executor via the
// appropriate method (Query for reads, Exec for writes).
func (e *Env) RunQuery(ctx context.Context, ex db.Executor, q *config.Query, args ...any) error {
	switch q.Type {
	case config.QueryTypeExec, config.QueryTypeExecBatch:
		if err := e.Exec(ctx, ex, q, args...); err != nil {
			return fmt.Errorf("executing exec %s: %w", q.Name, err)
		}
	case config.QueryTypeQuery, config.QueryTypeQueryBatch, "":
		if err := e.Query(ctx, ex, q, args...); err != nil {
			return fmt.Errorf("executing query %s: %w", q.Name, err)
		}
	}

	return nil
}

// RunQueryPrepared executes a query using a prepared statement.
func (e *Env) RunQueryPrepared(ctx context.Context, stmt db.PreparedStatement, q *config.Query, args ...any) error {
	switch q.Type {
	case config.QueryTypeExec:
		if err := e.ExecPrepared(ctx, stmt, q, args...); err != nil {
			return fmt.Errorf("executing prepared exec %s: %w", q.Name, err)
		}
	case config.QueryTypeQuery, "":
		if err := e.QueryPrepared(ctx, stmt, q, args...); err != nil {
			return fmt.Errorf("executing prepared query %s: %w", q.Name, err)
		}
	}

	return nil
}

// runSection runs a list of queries against the given executor. Args
// are inlined into the SQL (string replacement of $N placeholders)
// when the query is a batch type or when batch-expanded (multiple arg
// sets from gen_batch/batch/ref_each). This provides cross-driver
// placeholder compatibility and avoids pgx-stdlib DECIMAL type issues.
// All other queries use native driver bind params.
func (e *Env) runSection(ctx context.Context, queries []*config.Query, section config.ConfigSection, ex db.Executor) error {
	verbose := section != config.ConfigSectionInit && section != config.ConfigSectionRun && section != config.ConfigSectionWorker

	for _, q := range queries {
		if verbose {
			verb := "running"
			if e.Output != nil {
				verb = "capturing"
			}
			slog.Info(verb+" query", "section", section, "query", q.Name)
		}

		if e.Output != nil {
			if err := e.captureOutput(q, section); err != nil {
				return err
			}
			continue
		}

		argSets, batchExpanded, err := e.GenerateArgs(q)
		if err != nil {
			err = fmt.Errorf("building args for %s query %s: %w", section, q.Name, err)
			e.sendResult(config.QueryResult{Name: q.Name, Section: section, Err: err})
			return err
		}

		queryStart := time.Now()

		// Use a prepared statement when the user opted in and the query
		// is not a batch type (batch queries change SQL text each time).
		usePrepared := q.Prepared && !q.IsBatch() && !batchExpanded

		var stmt db.PreparedStatement
		if usePrepared {
			stmt, err = e.getOrPrepare(ctx, q)
			if err != nil {
				err = fmt.Errorf("preparing %s query %s: %w", section, q.Name, err)
				e.sendResult(config.QueryResult{Name: q.Name, Section: section, Err: err})
				return err
			}
		}

		// Inline $N placeholders when batch expansion occurred
		// (gen_batch/batch/ref_each/query_batch/exec_batch), when
		// placeholders appear inside quoted strings (e.g.
		// string_to_array('$1', __sep__)) where the driver can't see them,
		// or when the query uses $N placeholders at all. The last case
		// ensures cross-driver compatibility: only PostgreSQL/CockroachDB
		// understand $N natively; MySQL, Oracle, and SQL Server do not.
		// When using prepared statements, skip inlining so args go
		// through the driver as native bind params.
		shouldInline := !usePrepared && (batchExpanded || strings.Contains(q.Query, "$1"))

		for i, args := range argSets {
			if verbose && len(argSets) > 1 {
				slog.Info("running query", "section", section, "query", q.Name, "batch", fmt.Sprintf("%d/%d", i+1, len(argSets)))
			}

			if shouldInline && len(args) > 0 {
				inlined := q.Query
				for j := len(args) - 1; j >= 0; j-- {
					placeholder := fmt.Sprintf("$%d", j+1)
					formatted := convert.SQLFormatValue(args[j], e.driver)
					// For non-RawSQL values (strings, []byte), replace quoted
					// placeholders '$N' as a unit first. SQLFormatValue already
					// wraps these in quotes, so this avoids double-quoting.
					// RawSQL values (batch CSVs) are unquoted by design and
					// must preserve the surrounding quotes in the query.
					_, isRaw := args[j].(convert.RawSQL)
					if !isRaw {
						singleQuoted := "'" + placeholder + "'"
						inlined = strings.ReplaceAll(inlined, singleQuoted, formatted)
						doubleQuoted := `"` + placeholder + `"`
						inlined = strings.ReplaceAll(inlined, doubleQuoted, formatted)
					}
					if isRaw && e.driver == "mongodb" {
						doubleQuoted := `"` + placeholder + `"`
						inlined = strings.ReplaceAll(inlined, doubleQuoted, formatted)
					}
					inlined = strings.ReplaceAll(inlined, placeholder, formatted)
				}

				if raw, ok := args[0].(convert.RawSQL); ok {
					inlined = valuesTokenRe.ReplaceAllLiteralString(inlined, string(raw))
				}

				inlinedQuery := &config.Query{
					Name:  q.Name,
					Type:  q.Type,
					Query: inlined,
				}
				if err := e.RunQuery(ctx, ex, inlinedQuery); err != nil {
					err = fmt.Errorf("running %s query %s: %w", section, q.Name, err)
					e.sendResult(config.QueryResult{Name: q.Name, Section: section, Latency: time.Since(queryStart), Err: err, Count: i})
					return err
				}
				continue
			}

			if usePrepared {
				if err := e.RunQueryPrepared(ctx, stmt, q, args...); err != nil {
					err = fmt.Errorf("running %s query %s: %w", section, q.Name, err)
					e.sendResult(config.QueryResult{Name: q.Name, Section: section, Latency: time.Since(queryStart), Err: err, Count: i})
					return err
				}
			} else {
				if err := e.RunQuery(ctx, ex, q, args...); err != nil {
					err = fmt.Errorf("running %s query %s: %w", section, q.Name, err)
					e.sendResult(config.QueryResult{Name: q.Name, Section: section, Latency: time.Since(queryStart), Err: err, Count: i})
					return err
				}
			}
		}

		if section == config.ConfigSectionSeed && q.Name != "" && len(q.CompiledArgs) > 0 {
			if q.IsBatch() {
				if len(e.capturedRows) > 0 {
					e.SetEnv(q.Name, e.capturedRows)
					e.capturedRows = nil
				}
			} else {
				columns := output.ExtractColumns(q)
				rows := make([]map[string]any, 0, len(argSets))
				for _, args := range argSets {
					if len(args) == 0 {
						continue
					}
					row := make(map[string]any, len(columns))
					for ci, col := range columns {
						if ci < len(args) {
							row[col] = args[ci]
						}
					}
					rows = append(rows, row)
				}
				if len(rows) > 0 {
					e.SetEnv(q.Name, rows)
				}
			}
		}

		var printAggs []string
		if len(q.Print) > 0 {
			printAggs = make([]string, len(q.Print))
			for i, pe := range q.Print {
				printAggs[i] = pe.Agg
			}
		}
		e.sendResult(config.QueryResult{Name: q.Name, Section: section, Latency: time.Since(queryStart), Count: len(argSets), PrintAggs: printAggs, PrintValues: e.lastPrintValues})

		if section == config.ConfigSectionRun && q.Wait > 0 {
			select {
			case <-time.After(time.Duration(q.Wait)):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func (e *Env) evalPrint(q *config.Query) {
	if len(q.CompiledPrint) == 0 {
		e.lastPrintValues = nil
		return
	}
	values := make([]string, len(q.CompiledPrint))
	for i, p := range q.CompiledPrint {
		if p == nil {
			continue
		}
		result, err := expr.Run(p, e.env)
		if err != nil {
			values[i] = fmt.Sprintf("<error: %v>", err)
			continue
		}
		values[i] = fmt.Sprintf("%v", result)
	}
	e.lastPrintValues = values
}

func (e *Env) sendResult(r config.QueryResult) {
	if e.Results != nil {
		e.Results <- r
	}
}

func inlineArgs(query string, args []any, driver string) string {
	for j := len(args) - 1; j >= 0; j-- {
		placeholder := fmt.Sprintf("$%d", j+1)
		formatted := convert.SQLFormatValue(args[j], driver)
		_, isRaw := args[j].(convert.RawSQL)
		if !isRaw {
			quotedPlaceholder := "'" + placeholder + "'"
			query = strings.ReplaceAll(query, quotedPlaceholder, formatted)
		}
		if driver == "mongodb" {
			dqPlaceholder := `"` + placeholder + `"`
			query = strings.ReplaceAll(query, dqPlaceholder, formatted)
		}
		query = strings.ReplaceAll(query, placeholder, formatted)
	}
	return query
}

// getOrPrepare returns a cached prepared statement for q, creating
// one if it doesn't exist yet. The query's $N placeholders are
// translated to the driver's native format before preparing.
func (e *Env) getOrPrepare(ctx context.Context, q *config.Query) (db.PreparedStatement, error) {
	if stmt, ok := e.stmtCache[q]; ok {
		return stmt, nil
	}
	preparer, ok := e.db.(db.Preparer)
	if !ok {
		return nil, fmt.Errorf("driver does not support prepared statements")
	}
	query := translatePlaceholders(q.Query, e.driver)
	stmt, err := preparer.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("preparing statement %s: %w", q.Name, err)
	}
	e.stmtCache[q] = stmt
	return stmt, nil
}

// translatePlaceholders rewrites $1, $2, ... placeholders to the
// native format expected by the given driver. pgx and dsql use $N
// natively, so the query is returned unchanged for those drivers.
func translatePlaceholders(query, driver string) string {
	var replaceFn func(int) string
	switch driver {
	case "mysql", "cassandra":
		replaceFn = func(int) string { return "?" }
	case "oracle":
		replaceFn = func(i int) string { return fmt.Sprintf(":%d", i) }
	case "mssql", "spanner":
		replaceFn = func(i int) string { return fmt.Sprintf("@p%d", i) }
	default:
		return query
	}
	for i := 1; strings.Contains(query, fmt.Sprintf("$%d", i)); i++ {
		query = strings.Replace(query, fmt.Sprintf("$%d", i), replaceFn(i), 1)
	}
	return query
}
