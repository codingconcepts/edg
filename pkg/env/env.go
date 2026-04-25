package env

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"
	"github.com/codingconcepts/edg/pkg/db"
	"github.com/codingconcepts/edg/pkg/gen"
	"github.com/codingconcepts/edg/pkg/output"
	"github.com/codingconcepts/edg/pkg/random"
	"github.com/codingconcepts/edg/pkg/seq"

	"github.com/expr-lang/expr/vm"
)

// ErrConditionalRollback is returned when a transaction's rollback_if
// condition evaluates to true. It triggers a rollback without being
// treated as an error.
var ErrConditionalRollback = errors.New("conditional rollback")

type Env struct {
	db      db.DB
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

	iterCounter int

	uniqSeenMutex sync.Mutex
	uniqSeen      map[string]map[any]struct{}
	uniqProg      map[string]*vm.Program

	nurandCMutex sync.RWMutex
	nurandC      map[int]int

	vectorCentroidsMutex sync.RWMutex
	vectorCentroids      map[string][][]float64

	seqCounter atomic.Int64
	seqManager *seq.Manager

	computedArgs     []any
	computedArgNames map[string]int
	capturedRows     []map[string]any
	lastPrintValues  []string

	txLocals map[string]any

	stmtCache map[*config.Query]db.PreparedStatement

	Retries int
	Output  output.Writer
	Results chan<- config.QueryResult
}

func NewEnv(database db.DB, driver string, r *config.Request, sections ...config.ConfigSection) (*Env, error) {
	if err := r.Validate(sections...); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	env := Env{
		db:              database,
		driver:          driver,
		oneCache:        map[string]any{},
		uniqSeen:        map[string]map[any]struct{}{},
		uniqProg:        map[string]*vm.Program{},
		permCache:       map[string]any{},
		nurandC:         map[int]int{},
		vectorCentroids: map[string][][]float64{},
		stmtCache:       map[*config.Query]db.PreparedStatement{},
		request:         r,
	}

	env.env = map[string]any{
		"arg":                 env.arg,                // Reference a previously evaluated arg by index or name.
		"array":               gen.GenArray,           // CockroachDB/PostgreSQL array literal.
		"avg":                 env.aggAvg,             // Average a numeric field across all rows in a dataset.
		"batch":               convert.Batch,          // Generate sequential batch indices [0, n) for batched execution.
		"bit":                 gen.GenBit,             // Random fixed-length bit string.
		"blob":                gen.GenBlob,            // Random bytes as raw []byte for BLOB/BYTEA columns.
		"bool":                gen.GenBool,            // Generate either a true or a false value.
		"bytes":               gen.GenBytes,           // Random bytes as hex-encoded string (PostgreSQL/CockroachDB only).
		"coalesce":            convert.Coalesce,       // First non-nil value from arguments.
		"cond":                convert.Cond,           // Conditional: if predicate then trueVal else falseVal.
		"const":               convert.Constant,       // Use a constant value.
		"count":               env.aggCount,           // Number of rows in a dataset.
		"date_offset":         gen.DateOffset,         // Timestamp offset from now.
		"date":                gen.DateRand,           // Random date with custom format.
		"distinct":            env.aggDistinct,        // Number of distinct values for a field in a dataset.
		"distribute_sum":      env.distributeSum,      // Partition a total into N random parts that sum exactly to it.
		"distribute_weighted": env.distributeWeighted, // Partition a total by proportional weights with optional noise.
		"duration":            gen.RandDuration,       // Random duration between min and max.
		"env_nil":             environNil,             // Use a value in the environment and return nil if missing.
		"env":                 environ,                // Use a value in the environment and return err if missing.
		"exp_f":               gen.ExpRandF,           // Exponential-distribution random float with precision.
		"exp":                 gen.ExpRand,            // Exponential-distribution random float in [min, max].
		"expr":                convert.Constant,       // Evaluate an arithmetic expression (e.g. expr(warehouses * 10)).
		"fail":                fail,                   // Return an error that stops the worker gracefully.
		"fatal":               fatal,                  // Terminate the process immediately.
		"gen_batch":           gen.GenBatch,           // Generate N values in batches, returns [][]any of comma-separated strings.
		"gen_locale":          gen.GenLocale,          // Generate locale-aware PII (name, address, phone, etc.).
		"uniq":               env.uniq,               // Retry expression until unique value found.
		"gen":                 gen.Gen,                // Generate a random value using gofakeit.
		"global":              env.global,             // Use a value in the global config section.
		"inet":                gen.GenInet,            // Random IP address within a CIDR block.
		"iter":                env.iter,               // 1-based row counter for exec_batch queries.
		"json_arr":            gen.JsonArr,            // Build a JSON array of N random values.
		"json_obj":            gen.JsonObj,            // Build a JSON object from key-value pairs.
		"local":               env.local,              // Use a transaction-scoped local variable.
		"lognorm_f":           gen.LognormRandF,       // Log-normal-distribution random float with precision.
		"lognorm":             gen.LognormRand,        // Log-normal-distribution random float in [min, max].
		"ltree":               gen.GenLtree,           // PostgreSQL ltree path from dot-joined parts.
		"mask":                gen.Mask,               // Deterministic pseudonymization token for PII masking.
		"max":                 env.aggMax,             // Maximum value of a numeric field in a dataset.
		"min":                 env.aggMin,             // Minimum value of a numeric field in a dataset.
		"norm_f":              env.normRandF,          // Normal-distribution random float with precision.
		"norm_n":              env.normRandN,          // N unique normal-distribution random values (comma-separated).
		"norm":                env.normRand,           // Normal-distribution random integer in [min, max].
		"nullable":            convert.Nullable,       // Return NULL with given probability, otherwise the value.
		"nurand_n":            env.nuRandN,            // N unique Non-Uniform Random values (comma-separated).
		"nurand":              env.nuRand,             // Non-Uniform Random per TPC-C spec.
		"objectid":            gen.GenObjectID,        // Generate a MongoDB ObjectID.
		"point_wkt":           gen.GenPointWKT,        // Random geographic point as WKT string.
		"point":               gen.GenPoint,           // Random geographic point within a radius.
		"ref_diff":            env.refDiff,            // Use unique rows across multiple arguments.
		"ref_each":            env.refEach,            // Cycles through each row.
		"ref_n":               env.refN,               // Pick N unique random field values from a dataset.
		"ref_perm":            env.refPerm,            // Use the same random row for the worker's lifetime.
		"ref_rand":            env.refRand,            // Use a random row.
		"ref_same":            env.refSame,            // Use the same random row across multiple arguments.
		"regex":               gen.GenRegex,           // Generate a string matching a regex pattern.
		"seq_exp":             env.seqExp,             // Exponential-distributed value from a global sequence.
		"seq_global":          env.seqGlobal,          // Shared auto-incrementing sequence across all workers.
		"seq_lognorm":         env.seqLognorm,         // Log-normal-distributed value from a global sequence.
		"seq_norm":            env.seqNorm,            // Normal-distributed value from a global sequence.
		"seq_rand":            env.seqRand,            // Uniform random value from an already-generated global sequence.
		"seq_zipf":            env.seqZipf,            // Zipfian-distributed value from a global sequence.
		"seq":                 env.seq,                // Auto-incrementing sequence (start + counter * step).
		"set_exp":             setExp,                 // Pick from a set using exponential distribution.
		"set_lognorm":         setLognormal,           // Pick from a set using log-normal distribution.
		"set_norm":            setNormal,              // Pick from a set using normal distribution.
		"set_rand":            setRand,                // Pick from a set (uniform or weighted random).
		"set_zipf":            setZipfian,             // Pick from a set using Zipfian distribution.
		"sum":                 env.aggSum,             // Sum a numeric field across all rows in a dataset.
		"template":            convert.Tmpl,           // Format string interpolation (fmt.Sprintf).
		"time":                gen.GenTime,            // Random time of day (HH:MM:SS).
		"timestamp":           gen.RandTimestamp,      // Random timestamp between min and max (RFC3339).
		"timez":               gen.GenTimez,           // Random time of day with timezone (HH:MM:SS+00:00).
		"uniform_f":           gen.FloatRand,          // Uniform random float in [min, max] with precision.
		"uniform":             gen.UniformRand,        // Uniform random float in [min, max].
		"uuid_v1":             gen.GenUUIDv1,          // Generate a Version 1 UUID (timestamp + node ID).
		"uuid_v4":             gen.GenUUIDv4,          // Generate a Version 4 UUID (random).
		"uuid_v6":             gen.GenUUIDv6,          // Generate a Version 6 UUID (reordered timestamp).
		"uuid_v7":             gen.GenUUIDv7,          // Generate a Version 7 UUID (Unix timestamp + random).
		"varbit":              gen.GenVarBit,          // Random variable-length bit string.
		"vector_norm":         env.vectorNorm,         // pgvector vector with normal centroid selection.
		"vector_zipf":         env.vectorZipf,         // pgvector vector with Zipfian centroid selection.
		"vector":              env.vector,             // pgvector-compatible clustered vector literal (uniform).
		"weighted_sample_n":   env.weightedSampleN,    // N weighted random field values (comma-separated).
		"zipf":                gen.ZipfRand,           // Zipfian-distributed random integer in [0, max].
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

func (e *Env) clearOneCache() {
	e.oneCacheMutex.Lock()
	defer e.oneCacheMutex.Unlock()

	clear(e.oneCache)
}

func (e *Env) resetUniqIndex() {
	e.uniqIndexMutex.Lock()
	defer e.uniqIndexMutex.Unlock()

	e.uniqIndex = 0
	e.iterCounter = 0

	e.uniqSeenMutex.Lock()
	clear(e.uniqSeen)
	clear(e.uniqProg)
	e.uniqSeenMutex.Unlock()
}

func (e *Env) iter() int {
	e.iterCounter++
	return e.iterCounter
}

// ensure imports are used
var (
	_ = random.Rng
	_ *seq.Manager
)
