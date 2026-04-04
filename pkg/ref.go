package pkg

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
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
	return data[rand.IntN(len(data))]
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

	n := min(lo+rand.IntN(hi-lo+1), len(data))

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
		j := i + rand.IntN(len(indices)-i)
		indices[i], indices[j] = indices[j], indices[i]
		parts[i] = fmt.Sprintf("%v", data[indices[i]][field])
	}
	return parts
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
func (e *Env) nuRand(rawA, rawX, rawY any) int {
	A, x, y := toInt(rawA), toInt(rawX), toInt(rawY)
	C := e.getNurandC(A)
	return (((rand.IntN(A+1) | (rand.IntN(y-x+1) + x)) + C) % (y - x + 1)) + x
}

// nuRandN generates N unique NURand values as a comma-separated string,
// where N is chosen randomly in [min, max]. Used for multi-item order
// lines in New-Order transactions.
func (e *Env) nuRandN(rawA, rawX, rawY, rawMinN, rawMaxN any) string {
	A, x, y := toInt(rawA), toInt(rawX), toInt(rawY)
	minN, maxN := toInt(rawMinN), toInt(rawMaxN)
	n := minN + rand.IntN(maxN-minN+1)

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

// normRand returns a normally-distributed random float in [min, max],
// rounded to 0 decimal places by default.
//
//	norm_rand(mean, stddev, min, max)
func (e *Env) normRand(rawMean, rawStddev, rawMin, rawMax any) float64 {
	return random.Norm(toFloat(rawMean), toFloat(rawStddev), toFloat(rawMin), toFloat(rawMax))
}

// normRandF returns a normally-distributed random float in [min, max],
// rounded to the given number of decimal places.
//
//	norm_rand_f(mean, stddev, min, max, precision)
func (e *Env) normRandF(rawMean, rawStddev, rawMin, rawMax, rawPrecision any) float64 {
	return random.Norm(toFloat(rawMean), toFloat(rawStddev), toFloat(rawMin), toFloat(rawMax), toInt(rawPrecision))
}

// normRandN generates N unique normally-distributed random integers as a
// comma-separated string, where N is chosen randomly in [minN, maxN].
//
//	norm_rand_n(mean, stddev, min, max, minN, maxN)
func (e *Env) normRandN(rawMean, rawStddev, rawMin, rawMax, rawMinN, rawMaxN any) string {
	mean, stddev := toFloat(rawMean), toFloat(rawStddev)
	lo, hi := toFloat(rawMin), toFloat(rawMax)
	minN, maxN := toInt(rawMinN), toInt(rawMaxN)
	n := minN + rand.IntN(maxN-minN+1)

	seen := make(map[float64]bool, n)
	parts := make([]string, 0, n)
	for len(parts) < n {
		v := random.Norm(mean, stddev, lo, hi)
		if !seen[v] {
			seen[v] = true
			parts = append(parts, fmt.Sprintf("%g", v))
		}
	}
	return strings.Join(parts, ",")
}

// seq returns a monotonically increasing value: start + counter * step.
// The counter is shared across all seq calls for a worker.
//
//	seq(start, step)
func (e *Env) seq(start, step any) int64 {
	s := int64(toInt(start))
	st := int64(toInt(step))
	counter := atomic.AddInt64(&e.seqCounter, 1) - 1
	return s + counter*st
}

// weightedSampleN picks N unique random rows from a named dataset
// using weighted selection based on a weight column, extracts the
// specified field, and returns a comma-separated string.
//
//	weighted_sample_n('products', 'id', 'popularity', 3, 8)
func (e *Env) weightedSampleN(name, field, weightField string, minN, maxN any) string {
	raw, ok := e.env[name]
	if !ok {
		return ""
	}
	data, ok := raw.([]map[string]any)
	if !ok || len(data) == 0 {
		return ""
	}

	lo := toInt(minN)
	hi := toInt(maxN)
	n := min(lo+rand.IntN(hi-lo+1), len(data))

	items := make([]weightedItem, len(data))
	for i, row := range data {
		items[i] = weightedItem{
			Value:  i,
			Weight: toInt(row[weightField]),
		}
	}
	wi := makeWeightedItems(items)
	if wi.totalWeight == 0 {
		return ""
	}

	seen := make(map[int]bool, n)
	parts := make([]string, 0, n)
	for len(parts) < n {
		idx := toInt(wi.choose())
		if !seen[idx] {
			seen[idx] = true
			parts = append(parts, fmt.Sprint(data[idx][field]))
		}
	}

	return strings.Join(parts, ",")
}
