package env

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/codingconcepts/edg/pkg/random"
)

const (
	// rejectionSamplingFactor is the ratio threshold for switching between
	// rejection sampling and Fisher-Yates in refN. When n < len(data)/factor,
	// rejection sampling is used to avoid allocating a full indices slice.
	rejectionSamplingFactor = 4
)

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

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("ref_each: failed to get column types: %w", err)
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
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				batch[i] = normalizeBytes(b, columnTypes[i].DatabaseTypeName())
			} else {
				batch[i] = v
			}
		}
		batches = append(batches, batch)
	}

	return batches, nil
}
