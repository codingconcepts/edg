package pkg

import (
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/codingconcepts/edg/pkg/random"
)

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
		return nil, errors.New("set_rand requires at least one value")
	}

	if len(weights) == 0 {
		return values[rand.IntN(len(values))], nil
	}

	if len(weights) != len(values) {
		return nil, fmt.Errorf("set_rand: values and weights length mismatch (%d vs %d)", len(values), len(weights))
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
//
//   - index 2 ('c') is picked most often
//
//   - ~68% of picks land in indices 1-3 ('b','c','d')
//
//   - ~95% of picks land in indices 0-4 ('a'..'e')
//
//   - a smaller stddev (e.g. 0.3) concentrates picks more tightly around the mean
//
//   - a larger stddev (e.g. 2.0) spreads picks more evenly across the set
//
//     set_norm(['a', 'b', 'c', 'd', 'e'], 2, 0.8)
func setNormal(values []any, mean, stddev any) (any, error) {
	if len(values) == 0 {
		return nil, errors.New("set_norm requires at least one value")
	}

	if len(values) == 1 {
		return values[0], nil
	}

	m := toFloat(mean)
	s := toFloat(stddev)

	idx := int(random.Norm(m, s, 0, float64(len(values)-1)))
	return values[idx], nil
}

// setExp picks an item from a set using exponential distribution.
// rate controls how steeply the distribution favours early indices:
// higher rate means stronger concentration on the first items.
//
//	set_exp(['a', 'b', 'c', 'd', 'e'], 0.5)
func setExp(values []any, rate any) (any, error) {
	if len(values) == 0 {
		return nil, errors.New("set_exp requires at least one value")
	}

	if len(values) == 1 {
		return values[0], nil
	}

	r := toFloat(rate)
	idx := int(random.Exp(r, 0, float64(len(values)-1)))
	return values[idx], nil
}

// setLognormal picks an item from a set using log-normal distribution.
// mu and sigma are the mean and standard deviation of the underlying
// normal distribution, mapped onto the set's indices.
//
//	set_lognorm(['a', 'b', 'c', 'd', 'e'], 1.0, 0.5)
func setLognormal(values []any, mu, sigma any) (any, error) {
	if len(values) == 0 {
		return nil, errors.New("set_lognorm requires at least one value")
	}

	if len(values) == 1 {
		return values[0], nil
	}

	m := toFloat(mu)
	s := toFloat(sigma)
	idx := int(random.LogNorm(m, s, 0, float64(len(values)-1)))
	return values[idx], nil
}

// setZipfian picks an item from a set using Zipfian distribution.
// s (> 1) and v (>= 1) control the distribution shape; lower indices
// are selected exponentially more often.
//
//	set_zipf(['a', 'b', 'c', 'd', 'e'], 2.0, 1.0)
func setZipfian(values []any, s, v any) (any, error) {
	if len(values) == 0 {
		return nil, errors.New("set_zipf requires at least one value")
	}

	if len(values) == 1 {
		return values[0], nil
	}

	idx := random.Zipf(toFloat(s), toFloat(v), len(values)-1)
	return values[idx], nil
}
