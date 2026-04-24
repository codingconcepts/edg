package env

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/codingconcepts/edg/pkg/convert"
	"github.com/codingconcepts/edg/pkg/random"
	"github.com/codingconcepts/edg/pkg/seq"
)

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
	A, err := convert.ToInt(rawA)
	if err != nil {
		return 0, fmt.Errorf("nurand A: %w", err)
	}
	x, err := convert.ToInt(rawX)
	if err != nil {
		return 0, fmt.Errorf("nurand x: %w", err)
	}
	y, err := convert.ToInt(rawY)
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
	A, err := convert.ToInt(rawA)
	if err != nil {
		return "", fmt.Errorf("nurand_n A: %w", err)
	}
	x, err := convert.ToInt(rawX)
	if err != nil {
		return "", fmt.Errorf("nurand_n x: %w", err)
	}
	y, err := convert.ToInt(rawY)
	if err != nil {
		return "", fmt.Errorf("nurand_n y: %w", err)
	}
	minN, err := convert.ToInt(rawMinN)
	if err != nil {
		return "", fmt.Errorf("nurand_n minN: %w", err)
	}
	maxN, err := convert.ToInt(rawMaxN)
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
	mean, err := convert.ToFloat(rawMean)
	if err != nil {
		return 0, fmt.Errorf("norm_f mean: %w", err)
	}
	stddev, err := convert.ToFloat(rawStddev)
	if err != nil {
		return 0, fmt.Errorf("norm_f stddev: %w", err)
	}
	mn, err := convert.ToFloat(rawMin)
	if err != nil {
		return 0, fmt.Errorf("norm_f min: %w", err)
	}
	mx, err := convert.ToFloat(rawMax)
	if err != nil {
		return 0, fmt.Errorf("norm_f max: %w", err)
	}
	p, err := convert.ToInt(rawPrecision)
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
	minN, err := convert.ToInt(rawMinN)
	if err != nil {
		return "", fmt.Errorf("norm_n minN: %w", err)
	}
	maxN, err := convert.ToInt(rawMaxN)
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
	s, err := convert.ToInt(rawStart)
	if err != nil {
		return 0, fmt.Errorf("seq start: %w", err)
	}
	st, err := convert.ToInt(rawStep)
	if err != nil {
		return 0, fmt.Errorf("seq step: %w", err)
	}
	counter := e.seqCounter.Add(1) - 1
	return int64(s) + counter*int64(st), nil
}

func (e *Env) seqGlobal(name any) (int64, error) {
	n, ok := name.(string)
	if !ok {
		return 0, fmt.Errorf("seq_global: name must be string, got %T", name)
	}
	if e.seqManager == nil {
		return 0, fmt.Errorf("seq_global(%q): no sequences configured", n)
	}
	return e.seqManager.Next(n)
}

func (e *Env) seqRand(name any) (int64, error) {
	n, ok := name.(string)
	if !ok {
		return 0, fmt.Errorf("seq_rand: name must be string, got %T", name)
	}
	if e.seqManager == nil {
		return 0, fmt.Errorf("seq_rand(%q): no sequences configured", n)
	}
	return e.seqManager.Rand(n)
}

func (e *Env) seqZipf(name, rawS, rawV any) (int64, error) {
	n, ok := name.(string)
	if !ok {
		return 0, fmt.Errorf("seq_zipf: name must be string, got %T", name)
	}
	if e.seqManager == nil {
		return 0, fmt.Errorf("seq_zipf(%q): no sequences configured", n)
	}
	s, err := convert.ToFloat(rawS)
	if err != nil {
		return 0, fmt.Errorf("seq_zipf s: %w", err)
	}
	v, err := convert.ToFloat(rawV)
	if err != nil {
		return 0, fmt.Errorf("seq_zipf v: %w", err)
	}
	return e.seqManager.Zipf(n, s, v)
}

func (e *Env) seqNorm(name, rawMean, rawStddev any) (int64, error) {
	n, ok := name.(string)
	if !ok {
		return 0, fmt.Errorf("seq_norm: name must be string, got %T", name)
	}
	if e.seqManager == nil {
		return 0, fmt.Errorf("seq_norm(%q): no sequences configured", n)
	}
	mean, err := convert.ToFloat(rawMean)
	if err != nil {
		return 0, fmt.Errorf("seq_norm mean: %w", err)
	}
	stddev, err := convert.ToFloat(rawStddev)
	if err != nil {
		return 0, fmt.Errorf("seq_norm stddev: %w", err)
	}
	return e.seqManager.Norm(n, mean, stddev)
}

func (e *Env) seqExp(name, rawRate any) (int64, error) {
	n, ok := name.(string)
	if !ok {
		return 0, fmt.Errorf("seq_exp: name must be string, got %T", name)
	}
	if e.seqManager == nil {
		return 0, fmt.Errorf("seq_exp(%q): no sequences configured", n)
	}
	rate, err := convert.ToFloat(rawRate)
	if err != nil {
		return 0, fmt.Errorf("seq_exp rate: %w", err)
	}
	return e.seqManager.Exp(n, rate)
}

func (e *Env) seqLognorm(name, rawMu, rawSigma any) (int64, error) {
	n, ok := name.(string)
	if !ok {
		return 0, fmt.Errorf("seq_lognorm: name must be string, got %T", name)
	}
	if e.seqManager == nil {
		return 0, fmt.Errorf("seq_lognorm(%q): no sequences configured", n)
	}
	mu, err := convert.ToFloat(rawMu)
	if err != nil {
		return 0, fmt.Errorf("seq_lognorm mu: %w", err)
	}
	sigma, err := convert.ToFloat(rawSigma)
	if err != nil {
		return 0, fmt.Errorf("seq_lognorm sigma: %w", err)
	}
	return e.seqManager.Lognorm(n, mu, sigma)
}

// SetSeqManager sets the shared sequence manager for cross-worker sequences.
func (e *Env) SetSeqManager(m *seq.Manager) {
	e.seqManager = m
}

// weightedSampleN picks N unique random rows from a named dataset
// using weighted selection based on a weight column, extracts the
// specified field, and returns a comma-separated string.
//
//	weighted_sample_n('products', 'id', 'popularity', 3, 8)
func (e *Env) weightedSampleN(name, field, weightField string, rawMinN, rawMaxN any) (string, error) {
	data, ok := e.getDataset(name)
	if !ok {
		return "", nil
	}

	lo, err := convert.ToInt(rawMinN)
	if err != nil {
		return "", fmt.Errorf("weighted_sample_n minN: %w", err)
	}
	hi, err := convert.ToInt(rawMaxN)
	if err != nil {
		return "", fmt.Errorf("weighted_sample_n maxN: %w", err)
	}
	n := min(lo+random.Rng.IntN(hi-lo+1), len(data))

	items := make([]weightedItem, len(data))
	for i, row := range data {
		w, err := convert.ToInt(row[weightField])
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
		idx, err := convert.ToInt(wi.choose())
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

// distributeSum partitions a total value into N random positive parts
// that sum exactly to total, where N is chosen randomly in [minN, maxN].
// Each part is rounded to the given number of decimal places. Returns
// a comma-separated string (e.g. "25.50,14.30,10.20").
//
//	distribute_sum(100.00, 3, 7, 2)
func (e *Env) distributeSum(rawTotal, rawMinN, rawMaxN, rawPrecision any) (string, error) {
	total, err := convert.ToFloat(rawTotal)
	if err != nil {
		return "", fmt.Errorf("distribute_sum total: %w", err)
	}
	minN, err := convert.ToInt(rawMinN)
	if err != nil {
		return "", fmt.Errorf("distribute_sum minN: %w", err)
	}
	maxN, err := convert.ToInt(rawMaxN)
	if err != nil {
		return "", fmt.Errorf("distribute_sum maxN: %w", err)
	}
	p, err := convert.ToInt(rawPrecision)
	if err != nil {
		return "", fmt.Errorf("distribute_sum precision: %w", err)
	}

	if minN < 1 {
		return "", fmt.Errorf("distribute_sum: minN must be >= 1, got %d", minN)
	}
	if maxN < minN {
		return "", fmt.Errorf("distribute_sum: maxN must be >= minN, got %d < %d", maxN, minN)
	}
	if total <= 0 {
		return "", fmt.Errorf("distribute_sum: total must be > 0, got %g", total)
	}

	n := minN
	if maxN > minN {
		n += random.Rng.IntN(maxN - minN + 1)
	}

	shift := math.Pow(10, float64(p))
	format := fmt.Sprintf("%%.%df", p)

	if n == 1 {
		return fmt.Sprintf(format, math.Round(total*shift)/shift), nil
	}

	// Random breakpoints in (0, total), sorted.
	breaks := make([]float64, n-1)
	for i := range breaks {
		breaks[i] = random.Rng.Float64() * total
	}
	sort.Float64s(breaks)

	// Parts are differences between consecutive breakpoints.
	parts := make([]float64, n)
	prev := 0.0
	for i, bp := range breaks {
		parts[i] = math.Round((bp-prev)*shift) / shift
		prev = bp
	}
	parts[n-1] = math.Round((total-prev)*shift) / shift

	// Correct rounding drift: add residual to last part.
	sum := 0.0
	for _, v := range parts {
		sum += v
	}
	residual := math.Round((total-sum)*shift) / shift
	parts[n-1] += residual

	strs := make([]string, n)
	for i, v := range parts {
		strs[i] = fmt.Sprintf(format, v)
	}
	return strings.Join(strs, ","), nil
}

// distributeWeighted partitions a total into parts proportional to the
// given weights, with controlled randomness. noise (0-1) blends between
// exact proportions (0) and fully random (1). Returns a comma-separated
// string whose values sum exactly to total.
//
//	distribute_weighted(1000, [50, 30, 20], 0.2, 2)
func (e *Env) distributeWeighted(rawTotal, rawWeights, rawNoise, rawPrecision any) (string, error) {
	total, err := convert.ToFloat(rawTotal)
	if err != nil {
		return "", fmt.Errorf("distribute_weighted total: %w", err)
	}
	weights, err := toFloatSlice(rawWeights)
	if err != nil {
		return "", fmt.Errorf("distribute_weighted weights: %w", err)
	}
	noise, err := convert.ToFloat(rawNoise)
	if err != nil {
		return "", fmt.Errorf("distribute_weighted noise: %w", err)
	}
	p, err := convert.ToInt(rawPrecision)
	if err != nil {
		return "", fmt.Errorf("distribute_weighted precision: %w", err)
	}

	n := len(weights)
	if n < 1 {
		return "", fmt.Errorf("distribute_weighted: weights must have at least 1 element")
	}
	if total <= 0 {
		return "", fmt.Errorf("distribute_weighted: total must be > 0, got %g", total)
	}
	if noise < 0 || noise > 1 {
		return "", fmt.Errorf("distribute_weighted: noise must be in [0, 1], got %g", noise)
	}

	sumW := 0.0
	for _, w := range weights {
		if w < 0 {
			return "", fmt.Errorf("distribute_weighted: weights must be >= 0")
		}
		sumW += w
	}
	if sumW == 0 {
		return "", fmt.Errorf("distribute_weighted: weights must not all be zero")
	}

	// Blend each weight's proportion with a uniform random value.
	blended := make([]float64, n)
	blendedSum := 0.0
	for i, w := range weights {
		proportion := w / sumW
		uniform := random.Rng.Float64()
		blended[i] = (1-noise)*proportion + noise*uniform
		blendedSum += blended[i]
	}

	// Scale to total.
	parts := make([]float64, n)
	shift := math.Pow(10, float64(p))
	format := fmt.Sprintf("%%.%df", p)

	if n == 1 {
		return fmt.Sprintf(format, math.Round(total*shift)/shift), nil
	}

	runningSum := 0.0
	for i := 0; i < n-1; i++ {
		parts[i] = math.Round(blended[i]/blendedSum*total*shift) / shift
		runningSum += parts[i]
	}
	parts[n-1] = math.Round((total-runningSum)*shift) / shift

	strs := make([]string, n)
	for i, v := range parts {
		strs[i] = fmt.Sprintf(format, v)
	}
	return strings.Join(strs, ","), nil
}

func toFloatSlice(v any) ([]float64, error) {
	switch s := v.(type) {
	case []any:
		result := make([]float64, len(s))
		for i, item := range s {
			f, err := convert.ToFloat(item)
			if err != nil {
				return nil, fmt.Errorf("element[%d]: %w", i, err)
			}
			result[i] = f
		}
		return result, nil
	case []float64:
		return s, nil
	case []int:
		result := make([]float64, len(s))
		for i, item := range s {
			result[i] = float64(item)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("expected array, got %T", v)
	}
}
