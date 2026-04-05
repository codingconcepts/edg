package pkg

import (
	"math"
)

// getDataset retrieves a named dataset from the environment as []map[string]any.
func (e *Env) getDataset(name string) ([]map[string]any, bool) {
	raw, ok := e.env[name]
	if !ok {
		return nil, false
	}
	data, ok := raw.([]map[string]any)
	if !ok || len(data) == 0 {
		return nil, false
	}
	return data, true
}

// aggSum returns the sum of a numeric field across all rows in a named dataset.
//
//	sum("orders", "amount")
func (e *Env) aggSum(name, field string) float64 {
	data, ok := e.getDataset(name)
	if !ok {
		return 0
	}
	var total float64
	for _, row := range data {
		total += toFloat(row[field])
	}
	return total
}

// aggAvg returns the average of a numeric field across all rows in a named dataset.
//
//	avg("orders", "amount")
func (e *Env) aggAvg(name, field string) float64 {
	count := e.aggCount(name)
	if count == 0 {
		return 0
	}
	return e.aggSum(name, field) / float64(count)
}

// aggMin returns the minimum value of a numeric field across all rows in a named dataset.
//
//	min("orders", "amount")
func (e *Env) aggMin(name, field string) float64 {
	data, ok := e.getDataset(name)
	if !ok {
		return 0
	}
	result := math.Inf(1)
	for _, row := range data {
		v := toFloat(row[field])
		if v < result {
			result = v
		}
	}
	return result
}

// aggMax returns the maximum value of a numeric field across all rows in a named dataset.
//
//	max("orders", "amount")
func (e *Env) aggMax(name, field string) float64 {
	data, ok := e.getDataset(name)
	if !ok {
		return 0
	}
	result := math.Inf(-1)
	for _, row := range data {
		v := toFloat(row[field])
		if v > result {
			result = v
		}
	}
	return result
}

// aggCount returns the number of rows in a named dataset.
//
//	count("orders")
func (e *Env) aggCount(name string) int {
	data, ok := e.getDataset(name)
	if !ok {
		return 0
	}
	return len(data)
}

// aggDistinct returns the number of distinct values for a field in a named dataset.
//
//	distinct("orders", "status")
func (e *Env) aggDistinct(name, field string) int {
	data, ok := e.getDataset(name)
	if !ok {
		return 0
	}
	seen := make(map[any]struct{}, len(data))
	for _, row := range data {
		seen[row[field]] = struct{}{}
	}
	return len(seen)
}
