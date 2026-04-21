package env

import (
	"fmt"
	"strings"
	"sync/atomic"

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
	counter := atomic.AddInt64(&e.seqCounter, 1) - 1
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
