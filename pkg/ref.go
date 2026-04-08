package pkg

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/codingconcepts/edg/pkg/random"
)

const (
	// rejectionSamplingFactor is the ratio threshold for switching between
	// rejection sampling and Fisher-Yates in refN. When n < len(data)/factor,
	// rejection sampling is used to avoid allocating a full indices slice.
	rejectionSamplingFactor = 4
)

func (e *Env) global(name string) any {
	return e.request.Globals[name]
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
	return data[random.Rng.IntN(len(data))]
}

// refN picks a random count N in [min, max], selects N unique random
// rows from the named dataset, extracts the specified field from each,
// and returns a comma-separated string (e.g. "42,17,93") for portable
// use across database drivers.
func (e *Env) refN(name string, field string, lo, hi int) string {
	raw, ok := e.env[name]
	if !ok {
		return ""
	}
	data, ok := raw.([]map[string]any)
	if !ok || len(data) == 0 {
		return ""
	}

	n := min(lo+random.Rng.IntN(hi-lo+1), len(data))

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
			idx := random.Rng.IntN(len(data))
			if _, ok := seen[idx]; !ok {
				seen[idx] = struct{}{}
				parts[i] = fmt.Sprint(data[idx][field])
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
		j := i + random.Rng.IntN(len(indices)-i)
		indices[i], indices[j] = indices[j], indices[i]
		parts[i] = fmt.Sprint(data[indices[i]][field])
	}
	return parts
}

func (e *Env) refSame(name string) map[string]any {
	return e.refCached(name, "ref_same", &e.oneCacheMutex, e.oneCache)
}

// refPerm picks a random row from a named dataset on first call and
// returns that same row for every subsequent call with that name,
// lasting the entire lifetime of the worker.
func (e *Env) refPerm(name string) map[string]any {
	return e.refCached(name, "ref_perm", &e.permCacheMutex, e.permCache)
}

// refCached picks a random row from a named dataset on first call
// and caches it for subsequent calls with the same name.
func (e *Env) refCached(name, label string, mu *sync.RWMutex, cache map[string]any) map[string]any {
	mu.Lock()
	defer mu.Unlock()

	if cached, exists := cache[name]; exists {
		if m, ok := cached.(map[string]any); ok {
			return m
		}
		slog.Warn(label+": cached value has unexpected type", "name", name, "type", fmt.Sprintf("%T", cached))
		return nil
	}

	raw, ok := e.env[name]
	if !ok {
		return nil
	}
	data, ok := raw.([]map[string]any)
	if !ok || len(data) == 0 {
		return nil
	}

	result := data[random.Rng.IntN(len(data))]
	cache[name] = result
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

	i := random.Rng.IntN(len(data)-e.uniqIndex) + e.uniqIndex

	// Swap in place; data shares its backing array with e.env[name].
	data[i], data[e.uniqIndex] = data[e.uniqIndex], data[i]

	val := data[e.uniqIndex]

	e.uniqIndex++

	return val
}

// refEach executes a SQL query and returns all rows as [][]any,
// where each inner slice contains one row's column values in order.
// Used in arg expressions to drive batched query execution.
func (e *Env) refEach(query string) ([][]any, error) {
	rows, err := e.db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("ref_each: query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("ref_each: failed to get columns: %w", err)
	}

	var batches [][]any
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("ref_each: failed to scan row: %w", err)
		}
		batch := make([]any, len(values))
		copy(batch, values)
		batches = append(batches, batch)
	}

	return batches, nil
}

// getNurandC returns the run-time constant C for a given A value,
// generating it on first access. C is fixed for the lifetime of the
// worker, per the TPC-C spec.
func (e *Env) getNurandC(A int) int {
	e.nurandCMutex.Lock()
	defer e.nurandCMutex.Unlock()

	if c, ok := e.nurandC[A]; ok {
		return c
	}

	c := random.Rng.IntN(A + 1)
	e.nurandC[A] = c
	return c
}

// nuRand implements the TPC-C Non-Uniform Random number generator:
//
//	NURand(A, x, y) = (((random(0, A) | random(x, y)) + C) % (y - x + 1)) + x
func (e *Env) nuRand(rawA, rawX, rawY any) (int, error) {
	A, err := toInt(rawA)
	if err != nil {
		return 0, fmt.Errorf("nurand A: %w", err)
	}
	x, err := toInt(rawX)
	if err != nil {
		return 0, fmt.Errorf("nurand x: %w", err)
	}
	y, err := toInt(rawY)
	if err != nil {
		return 0, fmt.Errorf("nurand y: %w", err)
	}
	C := e.getNurandC(A)
	return (((random.Rng.IntN(A+1) | (random.Rng.IntN(y-x+1) + x)) + C) % (y - x + 1)) + x, nil
}

// nuRandN generates N unique NURand values as a comma-separated string,
// where N is chosen randomly in [min, max]. Used for multi-item order
// lines in New-Order transactions.
func (e *Env) nuRandN(rawA, rawX, rawY, rawMinN, rawMaxN any) (string, error) {
	A, err := toInt(rawA)
	if err != nil {
		return "", fmt.Errorf("nurand_n A: %w", err)
	}
	x, err := toInt(rawX)
	if err != nil {
		return "", fmt.Errorf("nurand_n x: %w", err)
	}
	y, err := toInt(rawY)
	if err != nil {
		return "", fmt.Errorf("nurand_n y: %w", err)
	}
	minN, err := toInt(rawMinN)
	if err != nil {
		return "", fmt.Errorf("nurand_n minN: %w", err)
	}
	maxN, err := toInt(rawMaxN)
	if err != nil {
		return "", fmt.Errorf("nurand_n maxN: %w", err)
	}
	n := minN + random.Rng.IntN(maxN-minN+1)

	seen := make(map[int]bool, n)
	parts := make([]string, 0, n)
	for range random.MaxIter {
		if len(parts) >= n {
			break
		}
		v, err := e.nuRand(A, x, y)
		if err != nil {
			return "", err
		}
		if !seen[v] {
			seen[v] = true
			parts = append(parts, fmt.Sprintf("%d", v))
		}
	}
	if len(parts) < n {
		return "", fmt.Errorf("nurand_n: could not find %d unique values after %d iterations", n, random.MaxIter)
	}
	return strings.Join(parts, ","), nil
}

// normRand returns a normally-distributed random float in [min, max],
// rounded to 0 decimal places by default.
//
//	norm(mean, stddev, min, max)
func (e *Env) normRand(rawMean, rawStddev, rawMin, rawMax any) (float64, error) {
	return e.normRandF(rawMean, rawStddev, rawMin, rawMax, 0)
}

// normRandF returns a normally-distributed random float in [min, max],
// rounded to the given number of decimal places.
//
//	norm_f(mean, stddev, min, max, precision)
func (e *Env) normRandF(rawMean, rawStddev, rawMin, rawMax, rawPrecision any) (float64, error) {
	mean, err := toFloat(rawMean)
	if err != nil {
		return 0, fmt.Errorf("norm_f mean: %w", err)
	}
	stddev, err := toFloat(rawStddev)
	if err != nil {
		return 0, fmt.Errorf("norm_f stddev: %w", err)
	}
	mn, err := toFloat(rawMin)
	if err != nil {
		return 0, fmt.Errorf("norm_f min: %w", err)
	}
	mx, err := toFloat(rawMax)
	if err != nil {
		return 0, fmt.Errorf("norm_f max: %w", err)
	}
	p, err := toInt(rawPrecision)
	if err != nil {
		return 0, fmt.Errorf("norm_f precision: %w", err)
	}
	return random.Norm(mean, stddev, mn, mx, p)
}

// normRandN generates N unique normally-distributed random integers as a
// comma-separated string, where N is chosen randomly in [minN, maxN].
//
//	norm_n(mean, stddev, min, max, minN, maxN)
func (e *Env) normRandN(rawMean, rawStddev, rawMin, rawMax, rawMinN, rawMaxN any) (string, error) {
	minN, err := toInt(rawMinN)
	if err != nil {
		return "", fmt.Errorf("norm_n minN: %w", err)
	}
	maxN, err := toInt(rawMaxN)
	if err != nil {
		return "", fmt.Errorf("norm_n maxN: %w", err)
	}
	n := minN + random.Rng.IntN(maxN-minN+1)

	seen := make(map[float64]bool, n)
	parts := make([]string, 0, n)
	for range random.MaxIter {
		if len(parts) >= n {
			break
		}
		v, err := e.normRand(rawMean, rawStddev, rawMin, rawMax)
		if err != nil {
			return "", err
		}
		if !seen[v] {
			seen[v] = true
			parts = append(parts, fmt.Sprintf("%g", v))
		}
	}
	if len(parts) < n {
		return "", fmt.Errorf("norm_n: could not find %d unique values after %d iterations", n, random.MaxIter)
	}
	return strings.Join(parts, ","), nil
}

// seq returns a monotonically increasing value: start + counter * step.
// The counter is shared across all seq calls for a worker.
//
//	seq(start, step)
func (e *Env) seq(rawStart, rawStep any) (int64, error) {
	s, err := toInt(rawStart)
	if err != nil {
		return 0, fmt.Errorf("seq start: %w", err)
	}
	st, err := toInt(rawStep)
	if err != nil {
		return 0, fmt.Errorf("seq step: %w", err)
	}
	counter := atomic.AddInt64(&e.seqCounter, 1) - 1
	return int64(s) + counter*int64(st), nil
}

// weightedSampleN picks N unique random rows from a named dataset
// using weighted selection based on a weight column, extracts the
// specified field, and returns a comma-separated string.
//
//	weighted_sample_n('products', 'id', 'popularity', 3, 8)
func (e *Env) weightedSampleN(name, field, weightField string, rawMinN, rawMaxN any) (string, error) {
	raw, ok := e.env[name]
	if !ok {
		return "", nil
	}
	data, ok := raw.([]map[string]any)
	if !ok || len(data) == 0 {
		return "", nil
	}

	lo, err := toInt(rawMinN)
	if err != nil {
		return "", fmt.Errorf("weighted_sample_n minN: %w", err)
	}
	hi, err := toInt(rawMaxN)
	if err != nil {
		return "", fmt.Errorf("weighted_sample_n maxN: %w", err)
	}
	n := min(lo+random.Rng.IntN(hi-lo+1), len(data))

	items := make([]weightedItem, len(data))
	for i, row := range data {
		w, err := toInt(row[weightField])
		if err != nil {
			return "", fmt.Errorf("weighted_sample_n weight for row %d: %w", i, err)
		}
		items[i] = weightedItem{
			Value:  i,
			Weight: w,
		}
	}
	wi := makeWeightedItems(items)
	if wi.totalWeight == 0 {
		return "", nil
	}

	seen := make(map[int]bool, n)
	parts := make([]string, 0, n)
	for range random.MaxIter {
		if len(parts) >= n {
			break
		}
		idx, err := toInt(wi.choose())
		if err != nil {
			return "", err
		}
		if !seen[idx] {
			seen[idx] = true
			parts = append(parts, fmt.Sprint(data[idx][field]))
		}
	}
	if len(parts) < n {
		return "", fmt.Errorf("weighted_sample_n: could not find %d unique values after %d iterations", n, random.MaxIter)
	}
	return strings.Join(parts, ","), nil
}
