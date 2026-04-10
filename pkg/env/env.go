package env

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"strings"
	"sync"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"
	"github.com/codingconcepts/edg/pkg/gen"
	"github.com/codingconcepts/edg/pkg/random"

	"github.com/expr-lang/expr"
)

type Env struct {
	db      *sql.DB
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

	seqCounter int64

	computedArgs []any

	Results chan<- config.QueryResult
}

func NewEnv(db *sql.DB, r *config.Request) (*Env, error) {
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	env := Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		nurandC:   map[int]int{},
		request:   r,
	}

	env.env = map[string]any{
		"arg":               env.arg,             // Reference a previously evaluated arg by index.
		"array":             gen.GenArray,         // CockroachDB/PostgreSQL array literal.
		"avg":               env.aggAvg,           // Average a numeric field across all rows in a dataset.
		"batch":             convert.Batch,        // Generate sequential batch indices [0, n) for batched execution.
		"bit":               gen.GenBit,           // Random fixed-length bit string.
		"bool":              gen.GenBool,           // Generate either a true or a false value.
		"bytes":             gen.GenBytes,          // Random bytes as hex-encoded string.
		"coalesce":          convert.Coalesce,      // First non-nil value from arguments.
		"cond":              convert.Cond,          // Conditional: if predicate then trueVal else falseVal.
		"const":             convert.Constant,      // Use a constant value.
		"count":             env.aggCount,          // Number of rows in a dataset.
		"date_offset":       gen.DateOffset,        // Timestamp offset from now.
		"date":              gen.DateRand,           // Random date with custom format.
		"distinct":          env.aggDistinct,       // Number of distinct values for a field in a dataset.
		"duration":          gen.RandDuration,      // Random duration between min and max.
		"exp_f":             gen.ExpRandF,          // Exponential-distribution random float with precision.
		"exp":               gen.ExpRand,           // Exponential-distribution random float in [min, max].
		"expr":              convert.Constant,      // Evaluate an arithmetic expression (e.g. expr(warehouses * 10)).
		"gen_batch":         gen.GenBatch,          // Generate N values in batches, returns [][]any of comma-separated strings.
		"gen":               gen.Gen,               // Generate a random value using gofakeit.
		"global":            env.global,            // Use a value in the global config section.
		"inet":              gen.GenInet,           // Random IP address within a CIDR block.
		"json_arr":          gen.JsonArr,           // Build a JSON array of N random values.
		"json_obj":          gen.JsonObj,           // Build a JSON object from key-value pairs.
		"lognorm_f":         gen.LognormRandF,      // Log-normal-distribution random float with precision.
		"lognorm":           gen.LognormRand,       // Log-normal-distribution random float in [min, max].
		"max":               env.aggMax,            // Maximum value of a numeric field in a dataset.
		"min":               env.aggMin,            // Minimum value of a numeric field in a dataset.
		"norm_f":            env.normRandF,         // Normal-distribution random float with precision.
		"norm_n":            env.normRandN,         // N unique normal-distribution random values (comma-separated).
		"norm":              env.normRand,          // Normal-distribution random integer in [min, max].
		"nullable":          convert.Nullable,      // Return NULL with given probability, otherwise the value.
		"nurand_n":          env.nuRandN,           // N unique Non-Uniform Random values (comma-separated).
		"nurand":            env.nuRand,            // Non-Uniform Random per TPC-C spec.
		"point_wkt":         gen.GenPointWKT,       // Random geographic point as WKT string.
		"point":             gen.GenPoint,          // Random geographic point within a radius.
		"ref_diff":          env.refDiff,           // Use unique rows across multiple arguments.
		"ref_each":          env.refEach,           // Cycles through each row.
		"ref_n":             env.refN,              // Pick N unique random field values from a dataset.
		"ref_perm":          env.refPerm,           // Use the same random row for the worker's lifetime.
		"ref_rand":          env.refRand,           // Use a random row.
		"ref_same":          env.refSame,           // Use the same random row across multiple arguments.
		"regex":             gen.GenRegex,          // Generate a string matching a regex pattern.
		"seq":               env.seq,               // Auto-incrementing sequence (start + counter * step).
		"set_exp":           setExp,                // Pick from a set using exponential distribution.
		"set_lognorm":       setLognormal,          // Pick from a set using log-normal distribution.
		"set_norm":          setNormal,             // Pick from a set using normal distribution.
		"set_rand":          setRand,               // Pick from a set (uniform or weighted random).
		"set_zipf":          setZipfian,            // Pick from a set using Zipfian distribution.
		"sum":               env.aggSum,            // Sum a numeric field across all rows in a dataset.
		"template":          convert.Tmpl,          // Format string interpolation (fmt.Sprintf).
		"time":              gen.GenTime,           // Random time of day (HH:MM:SS).
		"timestamp":         gen.RandTimestamp,     // Random timestamp between min and max (RFC3339).
		"timez":             gen.GenTimez,          // Random time of day with timezone (HH:MM:SS+00:00).
		"uniform_f":         gen.FloatRand,         // Uniform random float in [min, max] with precision.
		"uniform":           gen.UniformRand,       // Uniform random float in [min, max].
		"uuid_v1":           gen.GenUUIDv1,         // Generate a Version 1 UUID (timestamp + node ID).
		"uuid_v4":           gen.GenUUIDv4,         // Generate a Version 4 UUID (random).
		"uuid_v6":           gen.GenUUIDv6,         // Generate a Version 6 UUID (reordered timestamp).
		"uuid_v7":           gen.GenUUIDv7,         // Generate a Version 7 UUID (Unix timestamp + random).
		"varbit":            gen.GenVarBit,         // Random variable-length bit string.
		"weighted_sample_n": env.weightedSampleN,   // N weighted random field values (comma-separated).
		"zipf":              gen.ZipfRand,          // Zipfian-distributed random integer in [0, max].
	}

	// Check that globals don't shadow built-in functions.
	for name := range r.Globals {
		if _, exists := env.env[name]; exists {
			return nil, fmt.Errorf("global %q shadows a built-in function", name)
		}
	}

	// Add each global variable to map itself for cleaner access.
	maps.Copy(env.env, r.Globals)

	// Load reference datasets into the environment so they're available
	// via ref_rand, ref_same, etc. without a database query.
	for name, rows := range r.Reference {
		if _, exists := env.env[name]; exists {
			return nil, fmt.Errorf("reference dataset %q shadows a built-in function or global", name)
		}
		env.SetEnv(name, slices.Clone(rows))
	}

	// Register user-defined expressions as callable functions.
	// First pass: add stubs so the compiler sees all expression names.
	for name := range r.Expressions {
		if _, exists := env.env[name]; exists {
			return nil, fmt.Errorf("expression %q shadows a built-in function, global, or reference dataset", name)
		}
		env.env[name] = func(args ...any) (any, error) {
			return nil, fmt.Errorf("expression %q used before compilation", name)
		}
	}
	// Second pass: compile bodies and replace stubs with real functions.
	for name, body := range r.Expressions {
		compileEnv := maps.Clone(env.env)
		compileEnv["args"] = []any{}

		program, err := expr.Compile(body, expr.Env(compileEnv))
		if err != nil {
			return nil, fmt.Errorf("compiling expression %q: %w", name, err)
		}

		p := program
		env.env[name] = func(args ...any) (any, error) {
			runEnv := maps.Clone(env.env)
			runEnv["args"] = args
			return expr.Run(p, runEnv)
		}
	}

	// Expand row references into args before compilation.
	allQueries := [][]*config.Query{r.Up, r.Seed, r.Deseed, r.Down, r.Init, r.Run}
	for _, queries := range allQueries {
		for _, query := range queries {
			if query.Row != "" {
				query.Args = slices.Clone(r.Rows[query.Row])
			}
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
		{"run", r.Run},
	} {
		for i, query := range group.queries {
			switch query.Type {
			case config.QueryTypeQuery, config.QueryTypeExec, config.QueryTypeQueryBatch, config.QueryTypeExecBatch, "":
			default:
				return nil, fmt.Errorf("unknown query type %q in %s query %d (%s)", query.Type, group.name, i, query.Name)
			}
			if err := query.CompileArgs(env.env); err != nil {
				return nil, fmt.Errorf("failed to compile %s query %d: %w", group.name, i, err)
			}
		}
	}

	return &env, nil
}

// GenerateArgs evaluates compiled arg expressions and returns one or more
// arg sets. When a single arg evaluates to [][]any (from ref_each), each
// inner slice becomes a separate arg set, causing the query to run once
// per batch row. Otherwise a single arg set is returned.
//
// For batch queries (type: query_batch or exec_batch), args are evaluated
// repeatedly per row, with values collected into comma-separated strings
// per arg position.
func (e *Env) GenerateArgs(q *config.Query) ([][]any, error) {
	defer e.clearOneCache()
	defer e.resetUniqIndex()

	if q.Type == config.QueryTypeQueryBatch || q.Type == config.QueryTypeExecBatch {
		return e.generateBatchArgs(q)
	}

	if len(q.CompiledArgs) == 0 {
		return [][]any{nil}, nil
	}

	e.computedArgs = e.computedArgs[:0]
	var completeArgs []any
	for _, cq := range q.CompiledArgs {
		compiledArg, err := expr.Run(cq, e.env)
		if err != nil {
			return nil, fmt.Errorf("error running expr: %w", err)
		}
		completeArgs = append(completeArgs, compiledArg)
		e.computedArgs = append(e.computedArgs, compiledArg)
	}

	// Find a ref_each batch arg ([][]any) and expand it into multiple
	// arg sets, merging with any scalar args at their positions.
	for i, arg := range completeArgs {
		batches, ok := arg.([][]any)
		if !ok {
			continue
		}

		result := make([][]any, len(batches))
		for b, batchRow := range batches {
			row := make([]any, 0, len(completeArgs)-1+len(batchRow))
			row = append(row, completeArgs[:i]...)
			row = append(row, batchRow...)
			row = append(row, completeArgs[i+1:]...)
			result[b] = row
		}
		return result, nil
	}

	return [][]any{completeArgs}, nil
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
				perArg[i][row] = convert.SQLFormatValue(v)
				e.computedArgs = append(e.computedArgs, v)
			}
		}

		args := make([]any, len(q.CompiledArgs))
		for i, parts := range perArg {
			args[i] = strings.Join(parts, ",")
		}
		result[b] = args
	}

	return result, nil
}

// RunQuery executes a query against the database via the appropriate
// method (Query for reads, Exec for writes).
func (e *Env) RunQuery(ctx context.Context, q *config.Query, args ...any) error {
	switch q.Type {
	case config.QueryTypeExec, config.QueryTypeExecBatch:
		if err := e.Exec(ctx, e.db, q, args...); err != nil {
			return fmt.Errorf("executing exec %s: %w", q.Name, err)
		}
	case config.QueryTypeQuery, config.QueryTypeQueryBatch, "":
		if err := e.Query(ctx, e.db, q, args...); err != nil {
			return fmt.Errorf("executing query %s: %w", q.Name, err)
		}
	}

	return nil
}

// runSection runs a list of queries. Args are inlined into the SQL
// (string replacement of $N placeholders) when the query is a batch
// type or when batch-expanded (multiple arg sets from gen_batch/batch/
// ref_each). This provides cross-driver placeholder compatibility and
// avoids pgx-stdlib DECIMAL type issues. All other queries use native
// driver bind params.
func (e *Env) runSection(ctx context.Context, queries []*config.Query, section config.ConfigSection) error {
	verbose := section != config.ConfigSectionInit && section != config.ConfigSectionRun

	for _, q := range queries {
		if verbose {
			slog.Info("running query", "section", section, "query", q.Name)
		}

		argSets, err := e.GenerateArgs(q)
		if err != nil {
			err = fmt.Errorf("building args for %s query %s: %w", section, q.Name, err)
			e.sendResult(config.QueryResult{Name: q.Name, Section: section, Err: err})
			return err
		}

		queryStart := time.Now()

		// Inline $N placeholders when the query is a batch type or
		// when batch-expanded (multiple arg sets from gen_batch/batch/ref_each).
		// Simple scalar queries always use native bind params.
		shouldInline := q.IsBatch() || len(argSets) > 1

		for i, args := range argSets {
			if verbose && len(argSets) > 1 {
				slog.Info("running query", "section", section, "query", q.Name, "batch", fmt.Sprintf("%d/%d", i+1, len(argSets)))
			}

			if shouldInline && len(args) > 0 {
				inlined := q.Query
				for j := len(args) - 1; j >= 0; j-- {
					placeholder := fmt.Sprintf("$%d", j+1)
					inlined = strings.ReplaceAll(inlined, placeholder, fmt.Sprintf("%v", args[j]))
				}

				inlinedQuery := &config.Query{
					Name:  q.Name,
					Type:  q.Type,
					Query: inlined,
				}
				if err := e.RunQuery(ctx, inlinedQuery); err != nil {
					err = fmt.Errorf("running %s query %s: %w", section, q.Name, err)
					e.sendResult(config.QueryResult{Name: q.Name, Section: section, Latency: time.Since(queryStart), Err: err, Count: i})
					return err
				}
				continue
			}

			if err := e.RunQuery(ctx, q, args...); err != nil {
				err = fmt.Errorf("running %s query %s: %w", section, q.Name, err)
				e.sendResult(config.QueryResult{Name: q.Name, Section: section, Latency: time.Since(queryStart), Err: err, Count: i})
				return err
			}
		}

		e.sendResult(config.QueryResult{Name: q.Name, Section: section, Latency: time.Since(queryStart), Count: len(argSets)})

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

func (e *Env) sendResult(r config.QueryResult) {
	if e.Results != nil {
		e.Results <- r
	}
}

// Up runs the schema-up queries to create tables.
func (e *Env) Up(ctx context.Context) error {
	return e.runSection(ctx, e.request.Up, config.ConfigSectionUp)
}

// Seed runs the seed queries to populate tables with data.
func (e *Env) Seed(ctx context.Context) error {
	return e.runSection(ctx, e.request.Seed, config.ConfigSectionSeed)
}

// Deseed runs the deseed queries to delete data from tables.
func (e *Env) Deseed(ctx context.Context) error {
	return e.runSection(ctx, e.request.Deseed, config.ConfigSectionDeseed)
}

// Down runs the schema-down queries to tear down tables.
func (e *Env) Down(ctx context.Context) error {
	return e.runSection(ctx, e.request.Down, config.ConfigSectionDown)
}

// Init runs the init queries once (e.g. to seed reference data).
func (e *Env) Init(ctx context.Context) error {
	return e.runSection(ctx, e.request.Init, config.ConfigSectionInit)
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
// is configured, a single transaction is chosen by weighted random
// selection. Otherwise all run queries execute sequentially.
func (e *Env) RunIteration(ctx context.Context) error {
	if len(e.request.RunWeights) == 0 {
		return e.runSection(ctx, e.request.Run, config.ConfigSectionRun)
	}

	q := e.pickWeighted()
	if q == nil {
		return e.runSection(ctx, e.request.Run, config.ConfigSectionRun)
	}
	return e.runSection(ctx, []*config.Query{q}, config.ConfigSectionRun)
}

// pickWeighted selects a single run query using the cumulative
// weights from run_weights. Queries not listed in run_weights
// are excluded.
func (e *Env) pickWeighted() *config.Query {
	type entry struct {
		query      *config.Query
		cumulative int
	}

	var entries []entry
	total := 0
	for _, q := range e.request.Run {
		w, ok := e.request.RunWeights[q.Name]
		if !ok {
			continue
		}
		total += w
		entries = append(entries, entry{query: q, cumulative: total})
	}

	if total == 0 {
		return nil
	}

	r := random.Rng.IntN(total)
	for _, e := range entries {
		if r < e.cumulative {
			return e.query
		}
	}

	return entries[len(entries)-1].query
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

func (e *Env) arg(index int) (any, error) {
	if index < 0 || index >= len(e.computedArgs) {
		return nil, fmt.Errorf("arg(%d): index out of range (%d args computed so far)", index, len(e.computedArgs))
	}
	return e.computedArgs[index], nil
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
