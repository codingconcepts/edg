package env

import (
	"fmt"
	"log/slog"
	"maps"
	"regexp"
	"strings"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"
	"github.com/codingconcepts/edg/pkg/output"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// valuesTokenRe matches __values__ with optional Oracle params:
//
//	__values__                        → standard multi-row VALUES
//	__values__(product(name, price))  → Oracle INSERT ALL
var valuesTokenRe = regexp.MustCompile(`__values__(?:\((\w+)\(([^)]+)\)\))?`)

// GenerateArgs evaluates compiled arg expressions and returns one or more
// arg sets. The returned bool indicates whether batch expansion occurred
// (from gen_batch/batch/ref_each/query_batch/exec_batch), signalling that
// the caller should inline placeholders rather than using native bind params.
func (e *Env) GenerateArgs(q *config.Query) ([][]any, bool, error) {
	defer e.clearOneCache()
	defer e.resetUniqIndex()
	defer e.evalPrint(q)

	if q.Type == config.QueryTypeQueryBatch || q.Type == config.QueryTypeExecBatch {
		r, err := e.generateBatchArgs(q)
		return r, true, err
	}

	if len(q.CompiledArgs) == 0 {
		return [][]any{nil}, false, nil
	}

	e.computedArgs = e.computedArgs[:0]
	e.computedArgNames = q.Args.Names
	var completeArgs []any
	for _, cq := range q.CompiledArgs {
		compiledArg, err := expr.Run(cq, e.env)
		if err != nil {
			return nil, false, fmt.Errorf("error running expr: %w", err)
		}
		completeArgs = append(completeArgs, compiledArg)
		e.computedArgs = append(e.computedArgs, compiledArg)
	}

	// Collect all ref_each batch args ([][]any) and their positions.
	type batchArg struct {
		pos  int
		rows [][]any
	}
	var batches []batchArg
	for i, arg := range completeArgs {
		if b, ok := arg.([][]any); ok {
			batches = append(batches, batchArg{pos: i, rows: b})
		}
	}

	if len(batches) == 0 {
		return [][]any{completeArgs}, false, nil
	}

	// Compute cartesian product of all batch args, keeping scalar
	// args in their original positions.
	totalRows := 1
	for _, b := range batches {
		totalRows *= len(b.rows)
	}

	result := make([][]any, totalRows)
	for idx := range totalRows {
		var row []any
		bi := 0
		for i, arg := range completeArgs {
			if bi < len(batches) && batches[bi].pos == i {
				// stride = product of lengths of all subsequent batches
				stride := 1
				for _, sb := range batches[bi+1:] {
					stride *= len(sb.rows)
				}
				batchRowIdx := (idx / stride) % len(batches[bi].rows)
				row = append(row, batches[bi].rows[batchRowIdx]...)
				bi++
			} else {
				row = append(row, arg)
			}
		}
		result[idx] = row
	}

	return result, true, nil
}

// generateBatchArgs handles type: query_batch/exec_batch queries. It evaluates each arg
// expression repeatedly (once per row), collecting values into CSV strings.
// Count and Size control the total rows and batch grouping.
func (e *Env) generateBatchArgs(q *config.Query) ([][]any, error) {
	count := 1
	if q.CompiledCount != nil {
		v, err := expr.Run(q.CompiledCount, e.env)
		if err != nil {
			return nil, fmt.Errorf("error evaluating count: %w", err)
		}
		c, err := convert.ToInt(v)
		if err != nil {
			return nil, fmt.Errorf("count: %w", err)
		}
		count = c
	}

	size := count
	if q.CompiledSize != nil {
		v, err := expr.Run(q.CompiledSize, e.env)
		if err != nil {
			return nil, fmt.Errorf("error evaluating size: %w", err)
		}
		s, err := convert.ToInt(v)
		if err != nil {
			return nil, fmt.Errorf("size: %w", err)
		}
		size = s
	}
	if size <= 0 {
		size = count
	}

	batches := (count + size - 1) / size
	result := make([][]any, batches)

	useJSON := q.BatchFormat == "json"

	valuesMatch := valuesTokenRe.FindStringSubmatch(q.Query)
	useValues := valuesMatch != nil
	var oracleTable, oracleCols string
	if useValues && len(valuesMatch) > 2 {
		oracleTable = valuesMatch[1]
		oracleCols = valuesMatch[2]
	}

	// Determine the formatting function per arg position based on the
	// SQL template. Placeholders inside quotes ('$N') are part of a
	// string literal (e.g. string_to_array) and need unquoted values.
	// Bare placeholders ($N, e.g. ARRAY[$N]) need SQL-quoted values.
	// When batch_format is "json", all values use BatchFormatValue
	// (raw text) since JSON escaping is handled by BatchJoinJSON.
	// When __values__ is used, formatters are not needed (SQLFormatValue
	// is called directly per value in the tuple-building loop).
	var formatters []func(any) string
	if !useValues {
		formatters = make([]func(any) string, len(q.CompiledArgs))
		for i := range q.CompiledArgs {
			if useJSON {
				formatters[i] = func(v any) string { return convert.BatchFormatValue(v, e.driver) }
			} else {
				singleQuoted := fmt.Sprintf("'$%d'", i+1)
				doubleQuoted := fmt.Sprintf(`"$%d"`, i+1)
				if strings.Contains(q.Query, singleQuoted) {
					formatters[i] = func(v any) string { return convert.BatchFormatValue(v, e.driver) }
				} else if strings.Contains(q.Query, doubleQuoted) && e.driver == "mongodb" {
					formatters[i] = func(v any) string { return convert.SQLFormatValue(v, e.driver) }
				} else if strings.Contains(q.Query, doubleQuoted) {
					formatters[i] = func(v any) string { return convert.BatchFormatValue(v, e.driver) }
				} else {
					formatters[i] = func(v any) string { return convert.SQLFormatValue(v, e.driver) }
				}
			}
		}
	}

	e.computedArgNames = q.Args.Names

	var captureColumns []string
	if q.Name != "" {
		captureColumns = output.ExtractColumns(q)
		e.capturedRows = nil
	}

	for b := range batches {
		n := size
		if remaining := count - b*size; remaining < size {
			n = remaining
		}

		if useValues {
			tuples := make([]string, n)
			for row := range n {
				e.clearOneCache()
				e.computedArgs = e.computedArgs[:0]
				vals := make([]string, len(q.CompiledArgs))
				for i, cq := range q.CompiledArgs {
					v, err := expr.Run(cq, e.env)
					if err != nil {
						return nil, fmt.Errorf("error running batch arg %d row %d: %w", i, b*size+row, err)
					}
					vals[i] = convert.SQLFormatValue(v, e.driver)
					e.computedArgs = append(e.computedArgs, v)
				}
				if captureColumns != nil {
					r := make(map[string]any, len(captureColumns))
					for ci, col := range captureColumns {
						if ci < len(e.computedArgs) {
							r[col] = e.computedArgs[ci]
						}
					}
					e.capturedRows = append(e.capturedRows, r)
				}
				tuples[row] = "(" + strings.Join(vals, ", ") + ")"
			}

			var valuesSQL string
			switch {
			case oracleTable != "":
				var sb strings.Builder
				for _, t := range tuples {
					sb.WriteString("INTO ")
					sb.WriteString(oracleTable)
					sb.WriteString(" (")
					sb.WriteString(oracleCols)
					sb.WriteString(") VALUES ")
					sb.WriteString(t)
					sb.WriteByte('\n')
				}
				sb.WriteString("SELECT 1 FROM DUAL")
				valuesSQL = sb.String()
			default:
				valuesSQL = "VALUES " + strings.Join(tuples, ", ")
			}
			result[b] = []any{convert.RawSQL(valuesSQL)}
			rowsDone := b*size + n
			slog.Info("generating rows", "query", q.Name, "progress", fmt.Sprintf("%d/%d", rowsDone, count))
			continue
		}

		perArg := make([][]string, len(q.CompiledArgs))
		for i := range perArg {
			perArg[i] = make([]string, n)
		}

		for row := range n {
			e.clearOneCache()
			e.computedArgs = e.computedArgs[:0]
			for i, cq := range q.CompiledArgs {
				v, err := expr.Run(cq, e.env)
				if err != nil {
					return nil, fmt.Errorf("error running batch arg %d row %d: %w", i, b*size+row, err)
				}
				perArg[i][row] = formatters[i](v)
				e.computedArgs = append(e.computedArgs, v)
			}
			if captureColumns != nil {
				r := make(map[string]any, len(captureColumns))
				for ci, col := range captureColumns {
					if ci < len(e.computedArgs) {
						r[col] = e.computedArgs[ci]
					}
				}
				e.capturedRows = append(e.capturedRows, r)
			}
		}

		args := make([]any, len(q.CompiledArgs))
		for i, parts := range perArg {
			if useJSON {
				args[i] = convert.RawSQL(convert.BatchJoinJSON(parts))
			} else {
				args[i] = convert.RawSQL(strings.Join(parts, convert.Sep))
			}
		}
		result[b] = args

		rowsDone := b*size + n
		slog.Info("generating rows", "query", q.Name, "progress", fmt.Sprintf("%d/%d", rowsDone, count))
	}

	return result, nil
}

// Eval compiles and runs an arbitrary expression against the env.
func (e *Env) Eval(expression string) (any, error) {
	e.envMutex.RLock()
	envCopy := maps.Clone(e.env)
	e.envMutex.RUnlock()

	program, err := expr.Compile(expression, expr.Env(envCopy))
	if err != nil {
		return nil, err
	}
	return expr.Run(program, envCopy)
}

func (e *Env) arg(key any) (any, error) {
	switch k := key.(type) {
	case int:
		if k < 0 || k >= len(e.computedArgs) {
			return nil, fmt.Errorf("arg(%d): index out of range (%d args computed so far)", k, len(e.computedArgs))
		}
		return e.computedArgs[k], nil
	case string:
		idx, ok := e.computedArgNames[k]
		if !ok {
			return nil, fmt.Errorf("arg(%q): unknown arg name", k)
		}
		if idx >= len(e.computedArgs) {
			return nil, fmt.Errorf("arg(%q): not yet computed (index %d, %d args computed so far)", k, idx, len(e.computedArgs))
		}
		return e.computedArgs[idx], nil
	default:
		return nil, fmt.Errorf("arg(): expected int or string, got %T", key)
	}
}

func (e *Env) local(name string) (any, error) {
	if e.txLocals == nil {
		return nil, fmt.Errorf("local(%q): not inside a transaction", name)
	}
	v, ok := e.txLocals[name]
	if !ok {
		return nil, fmt.Errorf("local(%q): not defined", name)
	}
	return v, nil
}

// evalLocals evaluates all compiled locals for a transaction and
// stores the results in txLocals for access via local().
func (e *Env) evalLocals(tx *config.Transaction) error {
	e.txLocals = make(map[string]any, len(tx.CompiledLocals))
	for name, p := range tx.CompiledLocals {
		result, err := expr.Run(p, e.env)
		if err != nil {
			return fmt.Errorf("evaluating local %q in transaction %s: %w", name, tx.Name, err)
		}
		e.txLocals[name] = result
	}
	return nil
}

func (e *Env) clearLocals() {
	e.txLocals = nil
}

func (e *Env) SetEnv(name string, data []map[string]any) {
	e.envMutex.Lock()
	defer e.envMutex.Unlock()

	e.env[name] = data
}

func (e *Env) uniq(first any, args ...any) (any, error) {
	switch v := first.(type) {
	case []any:
		seen := map[any]struct{}{}
		var result []any
		for _, item := range v {
			if _, dup := seen[item]; !dup {
				seen[item] = struct{}{}
				result = append(result, item)
			}
		}
		return result, nil
	case string:
		return e.uniqExpr(v, args...)
	default:
		return nil, fmt.Errorf("uniq: expected string expression or array, got %T", first)
	}
}

func (e *Env) uniqExpr(expression string, args ...any) (any, error) {
	maxRetries := 100
	if len(args) > 0 {
		v, err := convert.ToInt(args[0])
		if err != nil {
			return nil, fmt.Errorf("uniq: max retries must be an integer: %w", err)
		}
		maxRetries = v
	}

	e.uniqSeenMutex.Lock()
	prog, ok := e.uniqProg[expression]
	if !ok {
		var err error
		prog, err = expr.Compile(expression, expr.Env(e.env))
		if err != nil {
			e.uniqSeenMutex.Unlock()
			return nil, fmt.Errorf("uniq: compiling %q: %w", expression, err)
		}
		e.uniqProg[expression] = prog
	}
	seen := e.uniqSeen[expression]
	if seen == nil {
		seen = map[any]struct{}{}
		e.uniqSeen[expression] = seen
	}
	e.uniqSeenMutex.Unlock()

	for range maxRetries {
		v, err := expr.Run(prog, e.env)
		if err != nil {
			return nil, fmt.Errorf("uniq: evaluating %q: %w", expression, err)
		}
		e.uniqSeenMutex.Lock()
		if _, dup := seen[v]; !dup {
			seen[v] = struct{}{}
			e.uniqSeenMutex.Unlock()
			return v, nil
		}
		e.uniqSeenMutex.Unlock()
	}

	return nil, fmt.Errorf("uniq(%q): failed to generate unique value after %d attempts", expression, maxRetries)
}

// ensure vm import is used
var _ *vm.Program
