package pkg

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

	"github.com/codingconcepts/edg/pkg/random"

	"github.com/expr-lang/expr"
)

type Env struct {
	db      *sql.DB
	request *Request

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

	Results chan<- QueryResult
}

func NewEnv(db *sql.DB, r *Request) (*Env, error) {
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
		"array":             genArray,            // CockroachDB/PostgreSQL array literal.
		"avg":               env.aggAvg,          // Average a numeric field across all rows in a dataset.
		"batch":             batch,               // Generate sequential batch indices [0, n) for batched execution.
		"bit":               genBit,              // Random fixed-length bit string.
		"bool":              genBool,             // Generate either a true or a false value.
		"bytes":             genBytes,            // Random bytes as hex-encoded string.
		"coalesce":          coalesce,            // First non-nil value from arguments.
		"cond":              cond,                // Conditional: if predicate then trueVal else falseVal.
		"const":             constant,            // Use a constant value.
		"count":             env.aggCount,        // Number of rows in a dataset.
		"date_offset":       dateOffset,          // Timestamp offset from now.
		"date":              dateRand,            // Random date with custom format.
		"distinct":          env.aggDistinct,     // Number of distinct values for a field in a dataset.
		"duration":          randDuration,        // Random duration between min and max.
		"exp_f":             expRandF,            // Exponential-distribution random float with precision.
		"exp":               expRand,             // Exponential-distribution random float in [min, max].
		"expr":              constant,            // Evaluate an arithmetic expression (e.g. expr(warehouses * 10)).
		"gen_batch":         genBatch,            // Generate N values in batches, returns [][]any of comma-separated strings.
		"gen":               gen,                 // Generate a random value using gofakeit.
		"global":            env.global,          // Use a value in the global config section.
		"inet":              genInet,             // Random IP address within a CIDR block.
		"json_arr":          jsonArr,             // Build a JSON array of N random values.
		"json_obj":          jsonObj,             // Build a JSON object from key-value pairs.
		"lognorm_f":         lognormRandF,        // Log-normal-distribution random float with precision.
		"lognorm":           lognormRand,         // Log-normal-distribution random float in [min, max].
		"max":               env.aggMax,          // Maximum value of a numeric field in a dataset.
		"min":               env.aggMin,          // Minimum value of a numeric field in a dataset.
		"norm_f":            env.normRandF,       // Normal-distribution random float with precision.
		"norm_n":            env.normRandN,       // N unique normal-distribution random values (comma-separated).
		"norm":              env.normRand,        // Normal-distribution random integer in [min, max].
		"nullable":          nullable,            // Return NULL with given probability, otherwise the value.
		"nurand_n":          env.nuRandN,         // N unique Non-Uniform Random values (comma-separated).
		"nurand":            env.nuRand,          // Non-Uniform Random per TPC-C spec.
		"point_wkt":         genPointWKT,         // Random geographic point as WKT string.
		"point":             genPoint,            // Random geographic point within a radius.
		"ref_diff":          env.refDiff,         // Use unique rows across multiple arguments.
		"ref_each":          env.refEach,         // Cycles through each row.
		"ref_n":             env.refN,            // Pick N unique random field values from a dataset.
		"ref_perm":          env.refPerm,         // Use the same random row for the worker's lifetime.
		"ref_rand":          env.refRand,         // Use a random row.
		"ref_same":          env.refSame,         // Use the same random row across multiple arguments.
		"regex":             genRegex,            // Generate a string matching a regex pattern.
		"seq":               env.seq,             // Auto-incrementing sequence (start + counter * step).
		"set_exp":           setExp,              // Pick from a set using exponential distribution.
		"set_lognorm":       setLognormal,        // Pick from a set using log-normal distribution.
		"set_norm":          setNormal,           // Pick from a set using normal distribution.
		"set_rand":          setRand,             // Pick from a set (uniform or weighted random).
		"set_zipf":          setZipfian,          // Pick from a set using Zipfian distribution.
		"sum":               env.aggSum,          // Sum a numeric field across all rows in a dataset.
		"template":          tmpl,                // Format string interpolation (fmt.Sprintf).
		"time":              genTime,             // Random time of day (HH:MM:SS).
		"timestamp":         randTimestamp,       // Random timestamp between min and max (RFC3339).
		"timez":             genTimez,            // Random time of day with timezone (HH:MM:SS+00:00).
		"uniform_f":         floatRand,           // Uniform random float in [min, max] with precision.
		"uniform":           uniformRand,         // Uniform random float in [min, max].
		"uuid_v1":           genUUIDv1,           // Generate a Version 1 UUID (timestamp + node ID).
		"uuid_v4":           genUUIDv4,           // Generate a Version 4 UUID (random).
		"uuid_v6":           genUUIDv6,           // Generate a Version 6 UUID (reordered timestamp).
		"uuid_v7":           genUUIDv7,           // Generate a Version 7 UUID (Unix timestamp + random).
		"varbit":            genVarBit,           // Random variable-length bit string.
		"weighted_sample_n": env.weightedSampleN, // N weighted random field values (comma-separated).
		"zipf":              zipfRand,            // Zipfian-distributed random integer in [0, max].
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
	allQueries := [][]*Query{r.Up, r.Seed, r.Deseed, r.Down, r.Init, r.Run}
	for _, queries := range allQueries {
		for _, query := range queries {
			if query.Row != "" {
				query.Args = slices.Clone(r.Rows[query.Row])
			}
		}
	}

	for _, group := range []struct {
		name    string
		queries []*Query
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
			case QueryTypeQuery, QueryTypeExec, QueryTypeQueryBatch, QueryTypeExecBatch, "":
			default:
				return nil, fmt.Errorf("unknown query type %q in %s query %d (%s)", query.Type, group.name, i, query.Name)
			}
			if err := query.CompileArgs(&env); err != nil {
				return nil, fmt.Errorf("failed to compile %s query %d: %w", group.name, i, err)
			}
		}
	}

	return &env, nil
}

// runSection runs a list of queries. Args are inlined into the SQL
// (string replacement of $N placeholders) when the query is a batch
// type or when batch-expanded (multiple arg sets from gen_batch/batch/
// ref_each). This provides cross-driver placeholder compatibility and
// avoids pgx-stdlib DECIMAL type issues. All other queries use native
// driver bind params.
func (e *Env) runSection(ctx context.Context, queries []*Query, section ConfigSection) error {
	verbose := section != ConfigSectionInit && section != ConfigSectionRun

	for _, q := range queries {
		if verbose {
			slog.Info("running query", "section", section, "query", q.Name)
		}

		argSets, err := q.GenerateArgs(e)
		if err != nil {
			err = fmt.Errorf("building args for %s query %s: %w", section, q.Name, err)
			e.sendResult(QueryResult{Name: q.Name, Section: section, Err: err})
			return err
		}

		queryStart := time.Now()

		// Inline $N placeholders when the query is a batch type or
		// when batch-expanded (multiple arg sets from gen_batch/batch/ref_each).
		// Simple scalar queries always use native bind params.
		shouldInline := q.isBatch() || len(argSets) > 1

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

				inlinedQuery := &Query{
					Name:  q.Name,
					Type:  q.Type,
					Query: inlined,
				}
				if err := inlinedQuery.Run(ctx, e); err != nil {
					err = fmt.Errorf("running %s query %s: %w", section, q.Name, err)
					e.sendResult(QueryResult{Name: q.Name, Section: section, Latency: time.Since(queryStart), Err: err, Count: i})
					return err
				}
				continue
			}

			if err := q.Run(ctx, e, args...); err != nil {
				err = fmt.Errorf("running %s query %s: %w", section, q.Name, err)
				e.sendResult(QueryResult{Name: q.Name, Section: section, Latency: time.Since(queryStart), Err: err, Count: i})
				return err
			}
		}

		e.sendResult(QueryResult{Name: q.Name, Section: section, Latency: time.Since(queryStart), Count: len(argSets)})

		if section == ConfigSectionRun && q.Wait > 0 {
			select {
			case <-time.After(time.Duration(q.Wait)):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func (e *Env) sendResult(r QueryResult) {
	if e.Results != nil {
		e.Results <- r
	}
}

// Up runs the schema-up queries to create tables.
func (e *Env) Up(ctx context.Context) error {
	return e.runSection(ctx, e.request.Up, ConfigSectionUp)
}

// Seed runs the seed queries to populate tables with data.
func (e *Env) Seed(ctx context.Context) error {
	return e.runSection(ctx, e.request.Seed, ConfigSectionSeed)
}

// Deseed runs the deseed queries to delete data from tables.
func (e *Env) Deseed(ctx context.Context) error {
	return e.runSection(ctx, e.request.Deseed, ConfigSectionDeseed)
}

// Down runs the schema-down queries to tear down tables.
func (e *Env) Down(ctx context.Context) error {
	return e.runSection(ctx, e.request.Down, ConfigSectionDown)
}

// Init runs the init queries once (e.g. to seed reference data).
func (e *Env) Init(ctx context.Context) error {
	return e.runSection(ctx, e.request.Init, ConfigSectionInit)
}

// InitFrom copies init query results from another Env, avoiding
// redundant database queries when multiple workers need the same
// initial dataset. Each query-type result gets its own slice copy
// so that refDiff's in-place swaps don't interfere across workers.
func (e *Env) InitFrom(source *Env) {
	for _, q := range e.request.Init {
		if q.Type != QueryTypeQuery {
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
		return e.runSection(ctx, e.request.Run, ConfigSectionRun)
	}

	q := e.pickWeighted()
	if q == nil {
		return e.runSection(ctx, e.request.Run, ConfigSectionRun)
	}
	return e.runSection(ctx, []*Query{q}, ConfigSectionRun)
}

// pickWeighted selects a single run query using the cumulative
// weights from run_weights. Queries not listed in run_weights
// are excluded.
func (e *Env) pickWeighted() *Query {
	type entry struct {
		query      *Query
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
