package env

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/codingconcepts/edg/pkg/convert"
	"github.com/codingconcepts/edg/pkg/random"
)

// vector generates a pgvector-compatible vector literal with clustered
// values so that similarity search produces meaningful results.
// Centroids are lazily generated on first call and cached for the
// lifetime of the Env (keyed by dims+clusters). Centroid selection
// is uniform random.
//
//	vector(dimensions, clusters, spread)
func (e *Env) vector(rawDims, rawClusters, rawSpread any) (string, error) {
	dims, clusters, spread, err := e.parseVectorArgs(rawDims, rawClusters, rawSpread)
	if err != nil {
		return "", err
	}
	return e.vectorWithPicker(dims, clusters, spread, func(n int) int {
		return random.Rng.IntN(n)
	})
}

// vectorZipf generates a pgvector-compatible vector literal with
// Zipfian centroid selection. Cluster 0 is the "hottest" (most vectors),
// with frequency dropping off according to s and v.
//
//	vector_zipf(dimensions, clusters, spread, s, v)
func (e *Env) vectorZipf(rawDims, rawClusters, rawSpread, rawS, rawV any) (string, error) {
	dims, clusters, spread, err := e.parseVectorArgs(rawDims, rawClusters, rawSpread)
	if err != nil {
		return "", err
	}
	s, err := convert.ToFloat(rawS)
	if err != nil {
		return "", fmt.Errorf("vector_zipf s: %w", err)
	}
	v, err := convert.ToFloat(rawV)
	if err != nil {
		return "", fmt.Errorf("vector_zipf v: %w", err)
	}
	return e.vectorWithPicker(dims, clusters, spread, func(n int) int {
		idx, err := random.Zipf(s, v, n-1)
		if err != nil {
			return 0
		}
		return idx
	})
}

// vectorNorm generates a pgvector-compatible vector literal with
// normally-distributed centroid selection. The mean and stddev
// control which cluster index is picked most often.
//
//	vector_norm(dimensions, clusters, spread, mean, stddev)
func (e *Env) vectorNorm(rawDims, rawClusters, rawSpread, rawMean, rawStddev any) (string, error) {
	dims, clusters, spread, err := e.parseVectorArgs(rawDims, rawClusters, rawSpread)
	if err != nil {
		return "", err
	}
	mean, err := convert.ToFloat(rawMean)
	if err != nil {
		return "", fmt.Errorf("vector_norm mean: %w", err)
	}
	stddev, err := convert.ToFloat(rawStddev)
	if err != nil {
		return "", fmt.Errorf("vector_norm stddev: %w", err)
	}
	return e.vectorWithPicker(dims, clusters, spread, func(n int) int {
		idx, err := random.Norm(mean, stddev, 0, float64(n-1))
		if err != nil {
			return n / 2
		}
		return int(idx)
	})
}

// parseVectorArgs validates and converts the common vector parameters.
func (e *Env) parseVectorArgs(rawDims, rawClusters, rawSpread any) (int, int, float64, error) {
	dims, err := convert.ToInt(rawDims)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("vector dims: %w", err)
	}
	clusters, err := convert.ToInt(rawClusters)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("vector clusters: %w", err)
	}
	spread, err := convert.ToFloat(rawSpread)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("vector spread: %w", err)
	}
	if dims <= 0 {
		return 0, 0, 0, fmt.Errorf("vector: dims must be > 0, got %d", dims)
	}
	if clusters <= 0 {
		return 0, 0, 0, fmt.Errorf("vector: clusters must be > 0, got %d", clusters)
	}
	return dims, clusters, spread, nil
}

// vectorWithPicker generates a vector using the given cluster picker function.
func (e *Env) vectorWithPicker(dims, clusters int, spread float64, pickCluster func(int) int) (string, error) {
	centroids := e.getOrCreateCentroids(dims, clusters)

	centroid := centroids[pickCluster(clusters)]
	vec := make([]float64, dims)
	var norm float64
	for i := range dims {
		v := centroid[i] + spread*random.Rng.NormFloat64()
		vec[i] = v
		norm += v * v
	}

	// Normalize to unit length (matches real embedding models).
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}

	// Format as pgvector literal: [0.012345,-0.234567,...]
	parts := make([]string, dims)
	for i, v := range vec {
		parts[i] = strconv.FormatFloat(v, 'f', 6, 64)
	}
	return "[" + strings.Join(parts, ",") + "]", nil
}

// getOrCreateCentroids returns cached centroids for the given dims+clusters
// key, creating them if they don't exist yet. Each centroid is a random
// unit vector.
func (e *Env) getOrCreateCentroids(dims, clusters int) [][]float64 {
	key := fmt.Sprintf("%d:%d", dims, clusters)

	e.vectorCentroidsMutex.RLock()
	c, ok := e.vectorCentroids[key]
	e.vectorCentroidsMutex.RUnlock()
	if ok {
		return c
	}

	e.vectorCentroidsMutex.Lock()
	defer e.vectorCentroidsMutex.Unlock()

	// Double-check after acquiring write lock.
	if c, ok := e.vectorCentroids[key]; ok {
		return c
	}

	centroids := make([][]float64, clusters)
	for i := range clusters {
		v := make([]float64, dims)
		var norm float64
		for j := range dims {
			v[j] = random.Rng.NormFloat64()
			norm += v[j] * v[j]
		}
		norm = math.Sqrt(norm)
		for j := range v {
			v[j] /= norm
		}
		centroids[i] = v
	}
	e.vectorCentroids[key] = centroids
	return centroids
}
