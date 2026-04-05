package pkg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

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

	Results chan<- QueryResult
}

func NewEnv(db *sql.DB, r *Request) (*Env, error) {
	env := Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		nurandC:   map[int]int{},
		request:   r,
	}

	env.env = map[string]any{
		"const":             constant,            // Use a constant value.
		"expr":              constant,            // Evaluate an arithmetic expression (e.g. expr(warehouses * 10)).
		"gen":               gen,                 // Generate a random value using gofakeit.
		"gen_batch":         genBatch,            // Generate N values in batches, returns [][]any of comma-separated strings.
		"batch":             batch,               // Generate sequential batch indices [0, n) for batched execution.
		"global":            env.global,          // Use a value in the global config section.
		"ref_rand":          env.refRand,         // Use a random row.
		"ref_same":          env.refSame,         // Use the same random row across multiple arguments.
		"ref_perm":          env.refPerm,         // Use the same random row for the worker's lifetime.
		"ref_diff":          env.refDiff,         // Use unique rows across multiple arguments.
		"ref_each":          env.refEach,         // Cycles through each row.
		"ref_n":             env.refN,            // Pick N unique random field values from a dataset.
		"nurand":            env.nuRand,          // Non-Uniform Random per TPC-C spec.
		"nurand_n":          env.nuRandN,         // N unique Non-Uniform Random values (comma-separated).
		"norm":              env.normRand,        // Normal-distribution random integer in [min, max].
		"norm_f":            env.normRandF,       // Normal-distribution random float with precision.
		"norm_n":            env.normRandN,       // N unique normal-distribution random values (comma-separated).
		"exp":               expRand,             // Exponential-distribution random float in [min, max].
		"exp_f":             expRandF,            // Exponential-distribution random float with precision.
		"lognorm":           lognormRand,         // Log-normal-distribution random float in [min, max].
		"lognorm_f":         lognormRandF,        // Log-normal-distribution random float with precision.
		"set_rand":          setRand,             // Pick from a set (uniform or weighted random).
		"set_norm":          setNormal,           // Pick from a set using normal distribution.
		"set_exp":           setExp,              // Pick from a set using exponential distribution.
		"set_lognorm":       setLognormal,        // Pick from a set using log-normal distribution.
		"set_zipf":          setZipfian,          // Pick from a set using Zipfian distribution.
		"uuid_v1":           genUUIDv1,           // Generate a Version 1 UUID (timestamp + node ID).
		"uuid_v4":           genUUIDv4,           // Generate a Version 4 UUID (random).
		"uuid_v6":           genUUIDv6,           // Generate a Version 6 UUID (reordered timestamp).
		"uuid_v7":           genUUIDv7,           // Generate a Version 7 UUID (Unix timestamp + random).
		"uniform_f":         floatRand,           // Uniform random float in [min, max] with precision.
		"uniform":           uniformRand,         // Uniform random float in [min, max].
		"seq":               env.seq,             // Auto-incrementing sequence (start + counter * step).
		"zipf":              zipfRand,            // Zipfian-distributed random integer in [0, max].
		"cond":              cond,                // Conditional: if predicate then trueVal else falseVal.
		"coalesce":          coalesce,            // First non-nil value from arguments.
		"template":          tmpl,                // Format string interpolation (fmt.Sprintf).
		"regex":             genRegex,            // Generate a string matching a regex pattern.
		"json_obj":          jsonObj,             // Build a JSON object from key-value pairs.
		"json_arr":          jsonArr,             // Build a JSON array of N random values.
		"point":             genPoint,            // Random geographic point within a radius.
		"point_wkt":         genPointWKT,         // Random geographic point as WKT string.
		"timestamp":         randTimestamp,       // Random timestamp between min and max (RFC3339).
		"duration":          randDuration,        // Random duration between min and max.
		"date":              dateRand,            // Random date with custom format.
		"date_offset":       dateOffset,          // Timestamp offset from now.
		"weighted_sample_n": env.weightedSampleN, // N weighted random field values (comma-separated).
		"bytes":             genBytes,            // Random bytes as hex-encoded string.
		"bit":               genBit,              // Random fixed-length bit string.
		"varbit":            genVarBit,           // Random variable-length bit string.
		"inet":              genInet,             // Random IP address within a CIDR block.
		"array":             genArray,            // CockroachDB/PostgreSQL array literal.
		"time":              genTime,             // Random time of day (HH:MM:SS).
		"timez":             genTimez,            // Random time of day with timezone (HH:MM:SS+00:00).
		"sum":               env.aggSum,          // Sum a numeric field across all rows in a dataset.
		"avg":               env.aggAvg,          // Average a numeric field across all rows in a dataset.
		"min":               env.aggMin,          // Minimum value of a numeric field in a dataset.
		"max":               env.aggMax,          // Maximum value of a numeric field in a dataset.
		"count":             env.aggCount,        // Number of rows in a dataset.
		"distinct":          env.aggDistinct,     // Number of distinct values for a field in a dataset.
	}

	// Add each global variable to map itself for cleaner access.
	maps.Copy(env.env, r.Globals)

	// Load reference datasets into the environment so they're available
	// via ref_rand, ref_same, etc. without a database query.
	for name, rows := range r.Reference {
		env.SetEnv(name, slices.Clone(rows))
	}

	// Register user-defined expressions as callable functions.
	// First pass: add stubs so the compiler sees all expression names.
	for name := range r.Expressions {
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
			case QueryTypeQuery, QueryTypeExec, QueryTypeBatch, "":
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

// runSection runs a list of queries, handling batched args from ref_each.
// Batch args are inlined directly into the SQL to avoid pgx-stdlib
// sending numeric values as DECIMAL, which CockroachDB can't mix
// with INT in arithmetic expressions.
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

		for i, args := range argSets {
			if verbose && len(argSets) > 1 {
				slog.Info("running query", "section", section, "query", q.Name, "batch", fmt.Sprintf("%d/%d", i+1, len(argSets)))
			}

			// Non-run sections use $N as framework-level placeholders
			// that get inlined before execution. Run-section queries
			// use driver-native bind params (:N for Oracle, $N for pgx).
			if section != ConfigSectionRun && len(args) > 0 {
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
		copied := slices.Clone(data)
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

	r := rand.IntN(total)
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
