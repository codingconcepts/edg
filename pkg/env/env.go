package env

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"
	"github.com/codingconcepts/edg/pkg/gen"
	"github.com/codingconcepts/edg/pkg/output"
	"github.com/codingconcepts/edg/pkg/random"
	"github.com/codingconcepts/edg/pkg/seq"

	"github.com/expr-lang/expr"
)

// ErrConditionalRollback is returned when a transaction's rollback_if
// condition evaluates to true. It triggers a rollback without being
// treated as an error.
var ErrConditionalRollback = errors.New("conditional rollback")

type Env struct {
	db      *sql.DB
	driver  string
	request *config.Request

	envMutex sync.RWMutex
	env      map[string]any

	oneCacheMutex sync.RWMutex
	oneCache      map[string]any

	permCacheMutex sync.RWMutex
	permCache      map[string]any

	uniqIndexMutex sync.RWMutex
	uniqIndex      int

	nurandCMutex sync.RWMutex
	nurandC      map[int]int

	vectorCentroidsMutex sync.RWMutex
	vectorCentroids      map[string][][]float64

	seqCounter atomic.Int64
	seqManager *seq.Manager

	computedArgs     []any
	computedArgNames map[string]int
	lastPrintValues  []string

	txLocals map[string]any

	stmtCache map[*config.Query]*sql.Stmt

	Output  output.Writer
	Results chan<- config.QueryResult
}

func NewEnv(db *sql.DB, driver string, r *config.Request) (*Env, error) {
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	env := Env{
		db:              db,
		driver:          driver,
		oneCache:        map[string]any{},
		permCache:       map[string]any{},
		nurandC:         map[int]int{},
		vectorCentroids: map[string][][]float64{},
		stmtCache:       map[*config.Query]*sql.Stmt{},
		request:         r,
	}

	env.env = map[string]any{
		"arg":               env.arg,             // Reference a previously evaluated arg by index or name.
		"array":             gen.GenArray,        // CockroachDB/PostgreSQL array literal.
		"avg":               env.aggAvg,          // Average a numeric field across all rows in a dataset.
		"batch":             convert.Batch,       // Generate sequential batch indices [0, n) for batched execution.
		"bit":               gen.GenBit,          // Random fixed-length bit string.
		"blob":              gen.GenBlob,         // Random bytes as raw []byte for BLOB/BYTEA columns.
		"bool":              gen.GenBool,         // Generate either a true or a false value.
		"bytes":             gen.GenBytes,        // Random bytes as hex-encoded string (PostgreSQL/CockroachDB only).
		"coalesce":          convert.Coalesce,    // First non-nil value from arguments.
		"cond":              convert.Cond,        // Conditional: if predicate then trueVal else falseVal.
		"const":             convert.Constant,    // Use a constant value.
		"count":             env.aggCount,        // Number of rows in a dataset.
		"date_offset":       gen.DateOffset,      // Timestamp offset from now.
		"date":              gen.DateRand,        // Random date with custom format.
		"distinct":          env.aggDistinct,     // Number of distinct values for a field in a dataset.
		"duration":          gen.RandDuration,    // Random duration between min and max.
		"env":               environ,             // Use a value in the environment and return err if missing.
		"env_nil":           environNil,          // Use a value in the environment and return nil if missing.
		"exp_f":             gen.ExpRandF,        // Exponential-distribution random float with precision.
		"exp":               gen.ExpRand,         // Exponential-distribution random float in [min, max].
		"expr":              convert.Constant,    // Evaluate an arithmetic expression (e.g. expr(warehouses * 10)).
		"gen_batch":         gen.GenBatch,        // Generate N values in batches, returns [][]any of comma-separated strings.
		"gen":               gen.Gen,             // Generate a random value using gofakeit.
		"global":            env.global,          // Use a value in the global config section.
		"local":             env.local,           // Use a transaction-scoped local variable.
		"inet":              gen.GenInet,         // Random IP address within a CIDR block.
		"json_arr":          gen.JsonArr,         // Build a JSON array of N random values.
		"json_obj":          gen.JsonObj,         // Build a JSON object from key-value pairs.
		"lognorm_f":         gen.LognormRandF,    // Log-normal-distribution random float with precision.
		"lognorm":           gen.LognormRand,     // Log-normal-distribution random float in [min, max].
		"max":               env.aggMax,          // Maximum value of a numeric field in a dataset.
		"min":               env.aggMin,          // Minimum value of a numeric field in a dataset.
		"norm_f":            env.normRandF,       // Normal-distribution random float with precision.
		"norm_n":            env.normRandN,       // N unique normal-distribution random values (comma-separated).
		"norm":              env.normRand,        // Normal-distribution random integer in [min, max].
		"nullable":          convert.Nullable,    // Return NULL with given probability, otherwise the value.
		"nurand_n":          env.nuRandN,         // N unique Non-Uniform Random values (comma-separated).
		"nurand":            env.nuRand,          // Non-Uniform Random per TPC-C spec.
		"point_wkt":         gen.GenPointWKT,     // Random geographic point as WKT string.
		"point":             gen.GenPoint,        // Random geographic point within a radius.
		"ref_diff":          env.refDiff,         // Use unique rows across multiple arguments.
		"ref_each":          env.refEach,         // Cycles through each row.
		"ref_n":             env.refN,            // Pick N unique random field values from a dataset.
		"ref_perm":          env.refPerm,         // Use the same random row for the worker's lifetime.
		"ref_rand":          env.refRand,         // Use a random row.
		"ref_same":          env.refSame,         // Use the same random row across multiple arguments.
		"regex":             gen.GenRegex,        // Generate a string matching a regex pattern.
		"seq":               env.seq,             // Auto-incrementing sequence (start + counter * step).
		"seq_global":        env.seqGlobal,       // Shared auto-incrementing sequence across all workers.
		"seq_rand":          env.seqRand,         // Uniform random value from an already-generated global sequence.
		"seq_zipf":          env.seqZipf,         // Zipfian-distributed value from a global sequence.
		"seq_norm":          env.seqNorm,         // Normal-distributed value from a global sequence.
		"seq_exp":           env.seqExp,          // Exponential-distributed value from a global sequence.
		"seq_lognorm":       env.seqLognorm,      // Log-normal-distributed value from a global sequence.
		"set_exp":           setExp,              // Pick from a set using exponential distribution.
		"set_lognorm":       setLognormal,        // Pick from a set using log-normal distribution.
		"set_norm":          setNormal,           // Pick from a set using normal distribution.
		"set_rand":          setRand,             // Pick from a set (uniform or weighted random).
		"set_zipf":          setZipfian,          // Pick from a set using Zipfian distribution.
		"sum":               env.aggSum,          // Sum a numeric field across all rows in a dataset.
		"template":          convert.Tmpl,        // Format string interpolation (fmt.Sprintf).
		"time":              gen.GenTime,         // Random time of day (HH:MM:SS).
		"timestamp":         gen.RandTimestamp,   // Random timestamp between min and max (RFC3339).
		"timez":             gen.GenTimez,        // Random time of day with timezone (HH:MM:SS+00:00).
		"uniform_f":         gen.FloatRand,       // Uniform random float in [min, max] with precision.
		"uniform":           gen.UniformRand,     // Uniform random float in [min, max].
		"uuid_v1":           gen.GenUUIDv1,       // Generate a Version 1 UUID (timestamp + node ID).
		"uuid_v4":           gen.GenUUIDv4,       // Generate a Version 4 UUID (random).
		"uuid_v6":           gen.GenUUIDv6,       // Generate a Version 6 UUID (reordered timestamp).
		"uuid_v7":           gen.GenUUIDv7,       // Generate a Version 7 UUID (Unix timestamp + random).
		"varbit":            gen.GenVarBit,       // Random variable-length bit string.
		"vector":            env.vector,          // pgvector-compatible clustered vector literal (uniform).
		"vector_norm":       env.vectorNorm,      // pgvector vector with normal centroid selection.
		"vector_zipf":       env.vectorZipf,      // pgvector vector with Zipfian centroid selection.
		"weighted_sample_n": env.weightedSampleN, // N weighted random field values (comma-separated).
		"zipf":              gen.ZipfRand,        // Zipfian-distributed random integer in [0, max].
	}

	if err := env.loadGlobals(r); err != nil {
		return nil, err
	}
	if err := env.loadReferences(r); err != nil {
		return nil, err
	}
	if err := env.registerExpressions(r); err != nil {
		return nil, err
	}
	if err := env.compileQueries(r); err != nil {
		return nil, err
	}

	return &env, nil
}

func (e *Env) loadGlobals(r *config.Request) error {
	for name := range r.Globals {
		if _, exists := e.env[name]; exists {
			return fmt.Errorf("global %q shadows a built-in function", name)
		}
	}

	maps.Copy(e.env, r.Globals)

	for _, name := range r.GlobalsOrder {
		strVal, ok := r.Globals[name].(string)
		if !ok {
			continue
		}
		program, err := expr.Compile(strVal, expr.Env(e.env))
		if err != nil {
			continue
		}
		result, err := expr.Run(program, e.env)
		if err != nil {
			return fmt.Errorf("evaluating global %q: %w", name, err)
		}
		e.env[name] = result
		r.Globals[name] = result
	}
	return nil
}

func (e *Env) loadReferences(r *config.Request) error {
	for name, rows := range r.Reference {
		if _, exists := e.env[name]; exists {
			return fmt.Errorf("reference dataset %q shadows a built-in function or global", name)
		}
		e.SetEnv(name, slices.Clone(rows))
	}
	return nil
}

func (e *Env) registerExpressions(r *config.Request) error {
	for name := range r.Expressions {
		if _, exists := e.env[name]; exists {
			return fmt.Errorf("expression %q shadows a built-in function, global, or reference dataset", name)
		}
		e.env[name] = func(args ...any) (any, error) {
			return nil, fmt.Errorf("expression %q used before compilation", name)
		}
	}
	for name, body := range r.Expressions {
		compileEnv := maps.Clone(e.env)
		compileEnv["args"] = []any{}

		program, err := expr.Compile(body, expr.Env(compileEnv))
		if err != nil {
			return fmt.Errorf("compiling expression %q: %w", name, err)
		}

		p := program
		e.env[name] = func(args ...any) (any, error) {
			runEnv := maps.Clone(e.env)
			runEnv["args"] = args
			return expr.Run(p, runEnv)
		}
	}
	return nil
}

func (e *Env) compileQueries(r *config.Request) error {
	runQueries := r.RunAllQueries()
	workerQueries := r.WorkerQueries()
	allQueries := [][]*config.Query{r.Up, r.Seed, r.Deseed, r.Down, r.Init, runQueries, workerQueries}

	for _, queries := range allQueries {
		for _, query := range queries {
			if query.Row != "" {
				query.Args = config.QueryArgs{Exprs: slices.Clone(r.Rows[query.Row])}
			}
		}
	}

	sepSQL := string(e.sep())
	for _, queries := range allQueries {
		for _, query := range queries {
			query.Query = strings.ReplaceAll(query.Query, "__sep__", sepSQL)
		}
	}

	for _, group := range []struct {
		name    string
		queries []*config.Query
	}{
		{"up", r.Up},
		{"seed", r.Seed},
		{"deseed", r.Deseed},
		{"down", r.Down},
		{"init", r.Init},
		{"run", runQueries},
		{"workers", workerQueries},
	} {
		for i, query := range group.queries {
			if query.IsRollbackIf() {
				if err := query.CompileRollbackIf(e.env); err != nil {
					return fmt.Errorf("failed to compile %s rollback_if %d: %w", group.name, i, err)
				}
				continue
			}
			switch query.Type {
			case config.QueryTypeQuery, config.QueryTypeExec, config.QueryTypeQueryBatch, config.QueryTypeExecBatch, "":
			default:
				return fmt.Errorf("unknown query type %q in %s query %d (%s)", query.Type, group.name, i, query.Name)
			}
			if err := query.CompileArgs(e.env); err != nil {
				return fmt.Errorf("failed to compile %s query %d: %w", group.name, i, err)
			}
			if err := query.CompilePrint(e.env); err != nil {
				return fmt.Errorf("failed to compile %s query %d print: %w", group.name, i, err)
			}
		}
	}

	for _, item := range r.Run {
		if item.Transaction != nil && len(item.Transaction.Locals) > 0 {
			if err := item.Transaction.CompileLocals(e.env); err != nil {
				return fmt.Errorf("transaction %s: %w", item.Transaction.Name, err)
			}
		}
	}
	return nil
}

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

	// Determine the formatting function per arg position based on the
	// SQL template. Placeholders inside quotes ('$N') are part of a
	// string literal (e.g. string_to_array) and need unquoted values.
	// Bare placeholders ($N, e.g. ARRAY[$N]) need SQL-quoted values.
	// When batch_format is "json", all values use BatchFormatValue
	// (raw text) since JSON escaping is handled by BatchJoinJSON.
	formatters := make([]func(any) string, len(q.CompiledArgs))
	for i := range q.CompiledArgs {
		if useJSON {
			formatters[i] = convert.BatchFormatValue
		} else {
			quoted := fmt.Sprintf("'$%d'", i+1)
			if strings.Contains(q.Query, quoted) {
				formatters[i] = convert.BatchFormatValue
			} else {
				formatters[i] = func(v any) string { return convert.SQLFormatValue(v, e.driver) }
			}
		}
	}

	e.computedArgNames = q.Args.Names

	for b := range batches {
		n := size
		if remaining := count - b*size; remaining < size {
			n = remaining
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
	}

	return result, nil
}

// RunQuery executes a query against the given executor via the
// appropriate method (Query for reads, Exec for writes).
func (e *Env) RunQuery(ctx context.Context, ex Executor, q *config.Query, args ...any) error {
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
func (e *Env) RunQueryPrepared(ctx context.Context, stmt *sql.Stmt, q *config.Query, args ...any) error {
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
func (e *Env) runSection(ctx context.Context, queries []*config.Query, section config.ConfigSection, ex Executor) error {
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

		var stmt *sql.Stmt
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
					if _, isRaw := args[j].(convert.RawSQL); !isRaw {
						quotedPlaceholder := "'" + placeholder + "'"
						inlined = strings.ReplaceAll(inlined, quotedPlaceholder, formatted)
					}
					inlined = strings.ReplaceAll(inlined, placeholder, formatted)
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

// Close releases resources held by the Env, including any cached
// prepared statements. Safe to call multiple times.
func (e *Env) Close() {
	for q, stmt := range e.stmtCache {
		stmt.Close()
		delete(e.stmtCache, q)
	}
}

func (e *Env) SetOutput(w output.Writer) {
	e.Output = w
}

func (e *Env) captureOutput(q *config.Query, section config.ConfigSection) error {
	columns := output.ExtractColumns(q)

	if q.IsBatch() {
		defer e.clearOneCache()
		defer e.resetUniqIndex()
		defer e.evalPrint(q)
		return e.captureOutputBatch(q, section, columns)
	}

	argSets, _, err := e.GenerateArgs(q)
	if err != nil {
		return fmt.Errorf("generating args for %s query %s: %w", section, q.Name, err)
	}

	var envRows []map[string]any
	for _, args := range argSets {
		resolvedSQL := q.Query
		if len(args) > 0 {
			resolvedSQL = inlineArgs(q.Query, args, e.driver)
		}
		if err := e.Output.Add(output.WriteRow{Section: string(section), Name: q.Name, SQL: resolvedSQL, Columns: columns, Args: args}); err != nil {
			return err
		}
		if len(columns) > 0 && len(args) > 0 {
			row := make(map[string]any, len(columns))
			for i, col := range columns {
				if i < len(args) {
					row[col] = args[i]
				}
			}
			envRows = append(envRows, row)
		}
	}

	if len(envRows) > 0 {
		e.SetEnv(q.Name, envRows)
	}

	e.sendResult(config.QueryResult{Name: q.Name, Section: section, Count: len(argSets)})
	return nil
}

func (e *Env) captureOutputBatch(q *config.Query, section config.ConfigSection, columns []string) error {
	count := 1
	if q.CompiledCount != nil {
		v, err := expr.Run(q.CompiledCount, e.env)
		if err != nil {
			return fmt.Errorf("evaluating count: %w", err)
		}
		c, err := convert.ToInt(v)
		if err != nil {
			return fmt.Errorf("count: %w", err)
		}
		count = c
	}

	e.computedArgNames = q.Args.Names

	envRows := make([]map[string]any, 0, count)
	for row := range count {
		e.clearOneCache()
		e.computedArgs = e.computedArgs[:0]
		args := make([]any, len(q.CompiledArgs))
		for i, cq := range q.CompiledArgs {
			v, err := expr.Run(cq, e.env)
			if err != nil {
				return fmt.Errorf("evaluating arg %d row %d: %w", i, row, err)
			}
			args[i] = v
			e.computedArgs = append(e.computedArgs, v)
		}

		if err := e.Output.Add(output.WriteRow{Section: string(section), Name: q.Name, SQL: q.Query, Columns: columns, Args: args}); err != nil {
			return err
		}

		r := make(map[string]any, len(columns))
		for i, col := range columns {
			if i < len(args) {
				r[col] = args[i]
			}
		}
		envRows = append(envRows, r)
	}

	if len(envRows) > 0 {
		e.SetEnv(q.Name, envRows)
	}

	e.sendResult(config.QueryResult{Name: q.Name, Section: section, Count: count})
	return nil
}

func inlineArgs(query string, args []any, driver string) string {
	for j := len(args) - 1; j >= 0; j-- {
		placeholder := fmt.Sprintf("$%d", j+1)
		formatted := convert.SQLFormatValue(args[j], driver)
		if _, isRaw := args[j].(convert.RawSQL); !isRaw {
			quotedPlaceholder := "'" + placeholder + "'"
			query = strings.ReplaceAll(query, quotedPlaceholder, formatted)
		}
		query = strings.ReplaceAll(query, placeholder, formatted)
	}
	return query
}

// getOrPrepare returns a cached prepared statement for q, creating
// one if it doesn't exist yet. The query's $N placeholders are
// translated to the driver's native format before preparing.
func (e *Env) getOrPrepare(ctx context.Context, q *config.Query) (*sql.Stmt, error) {
	if stmt, ok := e.stmtCache[q]; ok {
		return stmt, nil
	}
	query := translatePlaceholders(q.Query, e.driver)
	stmt, err := e.db.PrepareContext(ctx, query)
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
	case "mysql":
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

// Up runs the schema-up queries to create tables.
func (e *Env) Up(ctx context.Context) error {
	return e.runSection(ctx, e.request.Up, config.ConfigSectionUp, e.db)
}

// Seed runs the seed queries to populate tables with data.
func (e *Env) Seed(ctx context.Context) error {
	return e.runSection(ctx, e.request.Seed, config.ConfigSectionSeed, e.db)
}

// Deseed runs the deseed queries to delete data from tables.
func (e *Env) Deseed(ctx context.Context) error {
	return e.runSection(ctx, e.request.Deseed, config.ConfigSectionDeseed, e.db)
}

// Down runs the schema-down queries to tear down tables.
func (e *Env) Down(ctx context.Context) error {
	return e.runSection(ctx, e.request.Down, config.ConfigSectionDown, e.db)
}

// Init runs the init queries once (e.g. to seed reference data).
func (e *Env) Init(ctx context.Context) error {
	return e.runSection(ctx, e.request.Init, config.ConfigSectionInit, e.db)
}

// InitFrom copies init query results from another Env, avoiding
// redundant database queries when multiple workers need the same
// initial dataset. Each query-type result gets its own slice copy
// so that refDiff's in-place swaps don't interfere across workers.
func (e *Env) InitFrom(source *Env) {
	for _, q := range e.request.Init {
		if q.Type != config.QueryTypeQuery {
			continue
		}
		source.envMutex.RLock()
		data, ok := source.env[q.Name].([]map[string]any)
		source.envMutex.RUnlock()
		if !ok {
			continue
		}
		copied := make([]map[string]any, len(data))
		for i, row := range data {
			copied[i] = maps.Clone(row)
		}
		e.SetEnv(q.Name, copied)
	}
}

// RunIteration executes one iteration of the run queries. When run_weights
// is configured, a single item is chosen by weighted random selection.
// Otherwise all run items execute sequentially.
func (e *Env) RunIteration(ctx context.Context) error {
	if len(e.request.RunWeights) == 0 {
		return e.runRunItems(ctx, e.request.Run)
	}

	item := e.pickWeighted()
	if item == nil {
		return e.runRunItems(ctx, e.request.Run)
	}
	return e.runRunItems(ctx, []*config.RunItem{item})
}

func (e *Env) RunWorker(ctx context.Context, w *config.Worker) {
	ticker := time.NewTicker(w.Rate.TickerInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.runSection(ctx, []*config.Query{&w.Query}, config.ConfigSectionWorker, e.db); err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Error("worker error", "worker", w.Name, "error", err)
			}
		}
	}
}

// runRunItems dispatches each run item as either a standalone query
// or a multi-statement transaction.
func (e *Env) runRunItems(ctx context.Context, items []*config.RunItem) error {
	for _, item := range items {
		switch {
		case item.IsTransaction():
			if err := e.runTransaction(ctx, item.Transaction); err != nil {
				return err
			}
		default:
			if err := e.runSection(ctx, []*config.Query{item.Query}, config.ConfigSectionRun, e.db); err != nil {
				return err
			}
		}
	}
	return nil
}

// runTransaction wraps the queries of a Transaction in an explicit
// BEGIN/COMMIT block. On error the transaction is rolled back.
// When a rollback_if condition is set, it is evaluated after each
// query; if it returns true the transaction is rolled back without
// being treated as an error.
func (e *Env) runTransaction(ctx context.Context, tx *config.Transaction) error {
	start := time.Now()

	dbTx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("beginning transaction %s: %w", tx.Name, err)
		e.sendResult(config.QueryResult{Name: tx.Name, Section: config.ConfigSectionRun, Err: err, IsTransaction: true})
		return err
	}

	if len(tx.CompiledLocals) > 0 {
		if err := e.evalLocals(tx); err != nil {
			_ = dbTx.Rollback()
			e.sendResult(config.QueryResult{Name: tx.Name, Section: config.ConfigSectionRun, Err: err, IsTransaction: true})
			return err
		}
		defer e.clearLocals()
	}

	if err := e.runTransactionQueries(ctx, tx, dbTx); err != nil {
		_ = dbTx.Rollback()
		if errors.Is(err, ErrConditionalRollback) {
			e.sendResult(config.QueryResult{Name: tx.Name, Section: config.ConfigSectionRun, Latency: time.Since(start), Count: 1, IsTransaction: true, Rollback: true})
			return nil
		}
		e.sendResult(config.QueryResult{Name: tx.Name, Section: config.ConfigSectionRun, Err: err, IsTransaction: true})
		return err
	}

	if err := dbTx.Commit(); err != nil {
		err = fmt.Errorf("committing transaction %s: %w", tx.Name, err)
		e.sendResult(config.QueryResult{Name: tx.Name, Section: config.ConfigSectionRun, Err: err, IsTransaction: true})
		return err
	}

	e.sendResult(config.QueryResult{Name: tx.Name, Section: config.ConfigSectionRun, Latency: time.Since(start), Count: 1, IsTransaction: true})
	return nil
}

// runTransactionQueries executes each query in the transaction.
// When a rollback_if element is encountered, its condition is
// evaluated; if true, ErrConditionalRollback is returned.
func (e *Env) runTransactionQueries(ctx context.Context, tx *config.Transaction, dbTx *sql.Tx) error {
	for _, q := range tx.Queries {
		if q.IsRollbackIf() {
			result, err := expr.Run(q.CompiledRollbackIf, e.env)
			if err != nil {
				return fmt.Errorf("evaluating rollback_if in transaction %s: %w", tx.Name, err)
			}
			if b, ok := result.(bool); ok && b {
				return ErrConditionalRollback
			}
			continue
		}

		if err := e.runSection(ctx, []*config.Query{q}, config.ConfigSectionRun, dbTx); err != nil {
			return err
		}
	}
	return nil
}

// pickWeighted selects a single run item using the cumulative
// weights from run_weights. Items not listed in run_weights
// are excluded.
func (e *Env) pickWeighted() *config.RunItem {
	type entry struct {
		item       *config.RunItem
		cumulative int
	}

	var entries []entry
	total := 0
	for _, item := range e.request.Run {
		w, ok := e.request.RunWeights[item.Name()]
		if !ok {
			continue
		}
		total += w
		entries = append(entries, entry{item: item, cumulative: total})
	}

	if total == 0 {
		return nil
	}

	r := random.Rng.IntN(total)
	for _, e := range entries {
		if r < e.cumulative {
			return e.item
		}
	}

	return entries[len(entries)-1].item
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

func (e *Env) clearOneCache() {
	e.oneCacheMutex.Lock()
	defer e.oneCacheMutex.Unlock()

	clear(e.oneCache)
}

func (e *Env) resetUniqIndex() {
	e.uniqIndexMutex.Lock()
	defer e.uniqIndexMutex.Unlock()

	e.uniqIndex = 0
}
