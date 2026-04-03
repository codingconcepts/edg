package pkg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/expr-lang/expr"
)

const (
	// rejectionSamplingFactor is the ratio threshold for switching between
	// rejection sampling and Fisher-Yates in refN. When n < len(data)/factor,
	// rejection sampling is used to avoid allocating a full indices slice.
	rejectionSamplingFactor = 4
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
		"const":       constant,      // Use a constant value.
		"expr":        constant,      // Evaluate an arithmetic expression (e.g. expr(warehouses * 10)).
		"gen":         gen,           // Generate a random value using gofakeit.
		"gen_batch":   genBatch,      // Generate N values in batches, returns [][]any of comma-separated strings.
		"batch":       batch,         // Generate sequential batch indices [0, n) for batched execution.
		"global":      env.global,    // Use a value in the global config section.
		"ref_rand":    env.refRand,   // Use a random row.
		"ref_same":    env.refSame,   // Use the same random row across multiple arguments.
		"ref_perm":    env.refPerm,   // Use the same random row for the worker's lifetime.
		"ref_diff":    env.refDiff,   // Use unique rows across multiple arguments.
		"ref_each":    env.refEach,   // Cycles through each row.
		"ref_n":       env.refN,      // Pick N unique random field values from a dataset.
		"nurand":      env.nuRand,    // Non-Uniform Random per TPC-C spec.
		"nurand_n":    env.nuRandN,   // N unique Non-Uniform Random values (comma-separated).
		"norm_rand":   env.normRand,  // Normal-distribution random integer in [min, max].
		"norm_rand_n": env.normRandN, // N unique normal-distribution random integers (comma-separated).
		"set_rand":    setRand,       // Pick from a set (uniform or weighted random).
		"set_normal":  setNormal,     // Pick from a set using normal distribution.
	}

	// Add each global variable to map itself for cleaner access.
	maps.Copy(env.env, r.Globals)

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
			return fmt.Errorf("building args for %s query %s: %w", section, q.Name, err)
		}

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
					return fmt.Errorf("running %s query %s: %w", section, q.Name, err)
				}
				continue
			}

			if err := q.Run(ctx, e, args...); err != nil {
				return fmt.Errorf("running %s query %s: %w", section, q.Name, err)
			}
		}

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
		copy(copied, data)
		e.SetEnv(q.Name, copied)
	}
}

// RunOnce executes one iteration of the run queries. When run_weights
// is configured, a single transaction is chosen by weighted random
// selection. Otherwise all run queries execute sequentially.
func (e *Env) RunOnce(ctx context.Context) error {
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

// refRand returns a random row from a named dataset.
func (e *Env) refRand(name string) map[string]any {
	raw, ok := e.env[name]
	if !ok {
		return nil
	}
	data, ok := raw.([]map[string]any)
	if !ok || len(data) == 0 {
		return nil
	}
	return data[rand.IntN(len(data))]
}

// refN picks a random count N in [min, max], selects N unique random
// rows from the named dataset, extracts the specified field from each,
// and returns a comma-separated string (e.g. "42,17,93") for portable
// use across database drivers.
func (e *Env) refN(name string, field string, min, max int) string {
	raw, ok := e.env[name]
	if !ok {
		return ""
	}
	data, ok := raw.([]map[string]any)
	if !ok || len(data) == 0 {
		return ""
	}

	n := min + rand.IntN(max-min+1)
	if n > len(data) {
		n = len(data)
	}

	var parts []string
	if n*rejectionSamplingFactor < len(data) {
		parts = rejection(data, field, n)
	} else {
		parts = fisherYates(data, field, n)
	}

	return strings.Join(parts, ",")
}

// rejection selects n unique random items from data using rejection
// sampling. Efficient when n is small relative to len(data).
func rejection(data []map[string]any, field string, n int) []string {
	parts := make([]string, n)
	seen := make(map[int]struct{}, n)
	for i := range n {
		for {
			idx := rand.IntN(len(data))
			if _, ok := seen[idx]; !ok {
				seen[idx] = struct{}{}
				parts[i] = fmt.Sprintf("%v", data[idx][field])
				break
			}
		}
	}
	return parts
}

// fisherYates selects n unique random items from data using a partial
// Fisher-Yates shuffle on a copy of indices.
func fisherYates(data []map[string]any, field string, n int) []string {
	indices := make([]int, len(data))
	for i := range indices {
		indices[i] = i
	}
	parts := make([]string, n)
	for i := range n {
		j := i + rand.IntN(len(indices)-i)
		indices[i], indices[j] = indices[j], indices[i]
		parts[i] = fmt.Sprintf("%v", data[indices[i]][field])
	}
	return parts
}

// getNurandC returns the run-time constant C for a given A value,
// generating it on first access. C is fixed for the lifetime of the
// worker, per the TPC-C spec.
func (e *Env) getNurandC(A int) int {
	e.nurandCMutex.RLock()
	if c, ok := e.nurandC[A]; ok {
		e.nurandCMutex.RUnlock()
		return c
	}
	e.nurandCMutex.RUnlock()

	c := rand.IntN(A + 1)

	e.nurandCMutex.Lock()
	e.nurandC[A] = c
	e.nurandCMutex.Unlock()

	return c
}

// nuRand implements the TPC-C Non-Uniform Random number generator:
//
//	NURand(A, x, y) = (((random(0, A) | random(x, y)) + C) % (y - x + 1)) + x
func (e *Env) nuRand(a, b, c any) int {
	A, x, y := toInt(a), toInt(b), toInt(c)
	C := e.getNurandC(A)
	return (((rand.IntN(A+1) | (rand.IntN(y-x+1) + x)) + C) % (y - x + 1)) + x
}

// nuRandN generates N unique NURand values as a comma-separated string,
// where N is chosen randomly in [min, max]. Used for multi-item order
// lines in New-Order transactions.
func (e *Env) nuRandN(a, b, c, d, f any) string {
	A, x, y, min, max := toInt(a), toInt(b), toInt(c), toInt(d), toInt(f)
	n := min + rand.IntN(max-min+1)

	seen := make(map[int]bool, n)
	parts := make([]string, 0, n)
	for len(parts) < n {
		v := e.nuRand(A, x, y)
		if !seen[v] {
			seen[v] = true
			parts = append(parts, fmt.Sprintf("%d", v))
		}
	}
	return strings.Join(parts, ",")
}

// normRandOne generates a single normally-distributed random integer
// clamped to [min, max]. Uses the Box-Muller transform via rand.NormFloat64.
func normRandOne(mean, stddev, min, max float64) int {
	for {
		v := mean + stddev*rand.NormFloat64()
		clamped := int(math.Round(v))
		if clamped >= int(min) && clamped <= int(max) {
			return clamped
		}
	}
}

// normRand returns a normally-distributed random integer in [min, max].
//
//	norm_rand(mean, stddev, min, max)
func (e *Env) normRand(a, b, c, d any) int {
	mean, stddev := toFloat(a), toFloat(b)
	min, max := toFloat(c), toFloat(d)
	return normRandOne(mean, stddev, min, max)
}

// normRandN generates N unique normally-distributed random integers as a
// comma-separated string, where N is chosen randomly in [minN, maxN].
//
//	norm_rand_n(mean, stddev, min, max, minN, maxN)
func (e *Env) normRandN(a, b, c, d, f, g any) string {
	mean, stddev := toFloat(a), toFloat(b)
	min, max := toFloat(c), toFloat(d)
	minN, maxN := toInt(f), toInt(g)
	n := minN + rand.IntN(maxN-minN+1)

	seen := make(map[int]bool, n)
	parts := make([]string, 0, n)
	for len(parts) < n {
		v := normRandOne(mean, stddev, min, max)
		if !seen[v] {
			seen[v] = true
			parts = append(parts, fmt.Sprintf("%d", v))
		}
	}
	return strings.Join(parts, ",")
}

// refEach executes a SQL query and returns all rows as [][]any,
// where each inner slice contains one row's column values in order.
// Used in arg expressions to drive batched query execution.
func (e *Env) refEach(query string) [][]any {
	rows, err := e.db.QueryContext(context.Background(), query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil
	}

	var batches [][]any
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil
		}
		batch := make([]any, len(values))
		copy(batch, values)
		batches = append(batches, batch)
	}

	return batches
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

func (e *Env) global(name string) any {
	return e.request.Globals[name]
}

func constant(v any) any {
	return v
}

// batch returns sequential integers [0, n) as a [][]any batch set,
// driving batched query execution without requiring a SQL query.
func batch(n any) [][]any {
	count := toInt(n)
	result := make([][]any, count)
	for i := range count {
		result[i] = []any{i}
	}
	return result
}

// genBatch generates totalCount values using the given gofakeit pattern,
// split into groups of batchSize. Returns [][]any where each inner slice
// contains a comma-separated string of generated values, acting as a
// batch driver for GenerateArgs.
func genBatch(totalCount, batchSize any, pattern string) [][]any {
	total := toInt(totalCount)
	size := toInt(batchSize)
	if size <= 0 {
		size = total
	}
	batches := (total + size - 1) / size
	result := make([][]any, batches)
	for i := range batches {
		n := size
		if remaining := total - i*size; remaining < size {
			n = remaining
		}
		parts := make([]string, n)
		for j := range n {
			val := gen(pattern)
			if val != nil {
				parts[j] = fmt.Sprintf("%v", val)
			}
		}
		result[i] = []any{strings.Join(parts, ",")}
	}
	return result
}

func gen(s string) any {
	val, err := gofakeit.Generate(wrap(s))
	if err != nil {
		return nil
	}
	return val
}

func (e *Env) refSame(name string) map[string]any {
	e.oneCacheMutex.RLock()
	if cached, exists := e.oneCache[name]; exists {
		e.oneCacheMutex.RUnlock()
		return cached.(map[string]any)
	}
	e.oneCacheMutex.RUnlock()

	raw, ok := e.env[name]
	if !ok {
		return nil
	}
	data, ok := raw.([]map[string]any)
	if !ok || len(data) == 0 {
		return nil
	}

	result := data[rand.IntN(len(data))]

	e.oneCacheMutex.Lock()
	e.oneCache[name] = result
	e.oneCacheMutex.Unlock()

	return result
}

// refPerm picks a random row from a named dataset on first call and
// returns that same row for every subsequent call with that name,
// lasting the entire lifetime of the worker.
func (e *Env) refPerm(name string) map[string]any {
	e.permCacheMutex.RLock()
	if cached, exists := e.permCache[name]; exists {
		e.permCacheMutex.RUnlock()
		return cached.(map[string]any)
	}
	e.permCacheMutex.RUnlock()

	raw, ok := e.env[name]
	if !ok {
		return nil
	}
	data, ok := raw.([]map[string]any)
	if !ok || len(data) == 0 {
		return nil
	}

	result := data[rand.IntN(len(data))]

	e.permCacheMutex.Lock()
	e.permCache[name] = result
	e.permCacheMutex.Unlock()

	return result
}

func (e *Env) refDiff(name string) map[string]any {
	raw, ok := e.env[name]
	if !ok {
		return nil
	}
	data, ok := raw.([]map[string]any)
	if !ok || len(data) == 0 {
		return nil
	}

	e.uniqIndexMutex.Lock()
	defer e.uniqIndexMutex.Unlock()

	i := rand.IntN(len(data)-e.uniqIndex) + e.uniqIndex

	// Swap in place; data shares its backing array with e.env[name].
	data[i], data[e.uniqIndex] = data[e.uniqIndex], data[i]

	val := data[e.uniqIndex]

	e.uniqIndex++

	return val
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	default:
		return 0
	}
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

// weightedItem pairs a value with a selection weight.
type weightedItem struct {
	Value  any
	Weight int
}

// weightedItems supports weighted random selection from a set of items.
type weightedItems struct {
	items       []weightedItem
	totalWeight int
}

func makeWeightedItems(items []weightedItem) weightedItems {
	wi := weightedItems{items: items}
	for _, item := range items {
		wi.totalWeight += item.Weight
	}
	return wi
}

func (wi weightedItems) choose() any {
	r := rand.IntN(wi.totalWeight) + 1
	for _, item := range wi.items {
		r -= item.Weight
		if r <= 0 {
			return item.Value
		}
	}
	return nil
}

func buildWeightedItems(values []any, weights []int) weightedItems {
	items := make([]weightedItem, len(values))
	for i, v := range values {
		items[i] = weightedItem{Value: v, Weight: weights[i]}
	}
	return makeWeightedItems(items)
}

// setRand picks a random item from a set. If weights are provided,
// weighted random selection is used; otherwise uniform random.
//
//	set_rand(['visa', 'mastercard', 'amex'], [])
//	set_rand(['visa', 'mastercard', 'amex'], [60, 30, 10])
func setRand(values []any, weights []any) (any, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("set_rand requires at least one value")
	}

	if len(weights) == 0 {
		return values[rand.IntN(len(values))], nil
	}

	if len(weights) != len(values) {
		return nil, fmt.Errorf("set_rand values and weights must be the same length (got %d values and %d weights)", len(values), len(weights))
	}

	intWeights := make([]int, len(weights))
	for i, w := range weights {
		intWeights[i] = toInt(w)
	}

	wi := buildWeightedItems(values, intWeights)
	return wi.choose(), nil
}

// setNormal picks an item from a set using normal distribution.
// mean is the index that will be selected most often, and stddev
// controls the spread: ~68% of picks fall within mean +/- stddev
// indices, ~95% within mean +/- 2*stddev.
//
// For example, with values ['a','b','c','d','e'], mean=2, stddev=0.8:
//   - index 2 ('c') is picked most often
//   - ~68% of picks land in indices 1-3 ('b','c','d')
//   - ~95% of picks land in indices 0-4 ('a'..'e')
//   - a smaller stddev (e.g. 0.3) concentrates picks more tightly around the mean
//   - a larger stddev (e.g. 2.0) spreads picks more evenly across the set
//
//	set_normal(['a', 'b', 'c', 'd', 'e'], 2, 0.8)
func setNormal(values []any, mean, stddev any) (any, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("set_normal requires at least one value")
	}

	if len(values) == 1 {
		return values[0], nil
	}

	m := toFloat(mean)
	s := toFloat(stddev)

	idx := normRandOne(m, s, 0, float64(len(values)-1))
	return values[idx], nil
}

func wrap(s string) string {
	if strings.HasPrefix(s, "{") {
		return s
	}
	return "{" + s + "}"
}
