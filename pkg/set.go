package pkg

import (
	"errors"
	"fmt"

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
	r := random.Rng.IntN(wi.totalWeight) + 1
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
		return values[random.Rng.IntN(len(values))], nil
	}

	if len(weights) != len(values) {
		return nil, fmt.Errorf("set_rand: values and weights length mismatch (%d vs %d)", len(values), len(weights))
	}

	intWeights := make([]int, len(weights))
	for i, w := range weights {
		iw, err := toInt(w)
		if err != nil {
			return nil, fmt.Errorf("set_rand weight %d: %w", i, err)
		}
		intWeights[i] = iw
	}

	wi := buildWeightedItems(values, intWeights)
	return wi.choose(), nil
}

// pickFromSet handles the common guard and index-to-value lookup
// shared by the distribution-based set functions.
func pickFromSet(name string, values []any, indexFn func(max float64) (int, error)) (any, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("%s requires at least one value", name)
	}
	if len(values) == 1 {
		return values[0], nil
	}
	idx, err := indexFn(float64(len(values) - 1))
	if err != nil {
		return nil, err
	}
	return values[idx], nil
}

// setNormal picks an item from a set using normal distribution.
// mean is the index that will be selected most often, and stddev
// controls the spread: ~68% of picks fall within mean +/- stddev
// indices, ~95% within mean +/- 2*stddev.
//
//	set_norm(['a', 'b', 'c', 'd', 'e'], 2, 0.8)
func setNormal(values []any, rawMean, rawStddev any) (any, error) {
	m, err := toFloat(rawMean)
	if err != nil {
		return nil, fmt.Errorf("set_norm mean: %w", err)
	}
	s, err := toFloat(rawStddev)
	if err != nil {
		return nil, fmt.Errorf("set_norm stddev: %w", err)
	}
	return pickFromSet("set_norm", values, func(max float64) (int, error) {
		v, err := random.Norm(m, s, 0, max)
		return int(v), err
	})
}

// setExp picks an item from a set using exponential distribution.
// rate controls how steeply the distribution favours early indices:
// higher rate means stronger concentration on the first items.
//
//	set_exp(['a', 'b', 'c', 'd', 'e'], 0.5)
func setExp(values []any, rawRate any) (any, error) {
	r, err := toFloat(rawRate)
	if err != nil {
		return nil, fmt.Errorf("set_exp rate: %w", err)
	}
	return pickFromSet("set_exp", values, func(max float64) (int, error) {
		v, err := random.Exp(r, 0, max)
		return int(v), err
	})
}

// setLognormal picks an item from a set using log-normal distribution.
// mu and sigma are the mean and standard deviation of the underlying
// normal distribution, mapped onto the set's indices.
//
//	set_lognorm(['a', 'b', 'c', 'd', 'e'], 1.0, 0.5)
func setLognormal(values []any, rawMu, rawSigma any) (any, error) {
	m, err := toFloat(rawMu)
	if err != nil {
		return nil, fmt.Errorf("set_lognorm mu: %w", err)
	}
	s, err := toFloat(rawSigma)
	if err != nil {
		return nil, fmt.Errorf("set_lognorm sigma: %w", err)
	}
	return pickFromSet("set_lognorm", values, func(max float64) (int, error) {
		v, err := random.LogNorm(m, s, 0, max)
		return int(v), err
	})
}

// setZipfian picks an item from a set using Zipfian distribution.
// s (> 1) and v (>= 1) control the distribution shape; lower indices
// are selected exponentially more often.
//
//	set_zipf(['a', 'b', 'c', 'd', 'e'], 2.0, 1.0)
func setZipfian(values []any, rawS, rawV any) (any, error) {
	s, err := toFloat(rawS)
	if err != nil {
		return nil, fmt.Errorf("set_zipf s: %w", err)
	}
	v, err := toFloat(rawV)
	if err != nil {
		return nil, fmt.Errorf("set_zipf v: %w", err)
	}
	return pickFromSet("set_zipf", values, func(max float64) (int, error) {
		return random.Zipf(s, v, int(max))
	})
}
