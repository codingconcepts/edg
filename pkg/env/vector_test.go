package env

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVector_Format(t *testing.T) {
	env := testEnv(nil)

	result, err := env.vector(4, 3, 0.1)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(result, "["), "vector = %q, want []-wrapped", result)
	require.True(t, strings.HasSuffix(result, "]"), "vector = %q, want []-wrapped", result)
	inner := result[1 : len(result)-1]
	parts := strings.Split(inner, ",")
	require.Equal(t, 4, len(parts), "vector(4,...) produced %d dimensions, want 4", len(parts))
}

func TestVector_UnitLength(t *testing.T) {
	env := testEnv(nil)

	for range 100 {
		result, err := env.vector(16, 3, 0.1)
		require.NoError(t, err)
		inner := result[1 : len(result)-1]
		parts := strings.Split(inner, ",")

		var norm float64
		for _, p := range parts {
			var v float64
			fmt.Sscanf(p, "%f", &v)
			norm += v * v
		}
		// Should be approximately 1.0 (unit vector).
		require.InDelta(t, 1.0, norm, 0.01, "vector norm = %f, want ~1.0", norm)
	}
}

func TestVector_Clustering(t *testing.T) {
	env := testEnv(nil)

	// Generate many vectors with 2 clusters and tight spread.
	// Vectors from the same cluster should be closer to each other
	// than vectors from different clusters.
	const (
		dims     = 32
		n        = 200
		spread   = 0.05
		clusters = 2
	)

	vecs := make([][]float64, n)
	for i := range n {
		result, err := env.vector(dims, clusters, spread)
		require.NoError(t, err)
		inner := result[1 : len(result)-1]
		parts := strings.Split(inner, ",")
		vec := make([]float64, dims)
		for j, p := range parts {
			fmt.Sscanf(p, "%f", &vec[j])
		}
		vecs[i] = vec
	}

	// Compute all pairwise cosine similarities.
	// With 2 clusters, we expect a bimodal distribution:
	// high similarity (same cluster) and lower similarity (different clusters).
	var sims []float64
	for i := range n {
		for j := i + 1; j < n; j++ {
			var dot float64
			for k := range dims {
				dot += vecs[i][k] * vecs[j][k]
			}
			sims = append(sims, dot)
		}
	}

	// With tight spread and 2 clusters, there should be a mix of
	// high similarities (>0.9, same cluster) and lower ones (<0.5, different clusters).
	highCount := 0
	lowCount := 0
	for _, s := range sims {
		switch {
		case s > 0.9:
			highCount++
		case s < 0.5:
			lowCount++
		}
	}

	assert.NotZero(t, highCount, "no high-similarity pairs found; clustering may not be working")
	assert.NotZero(t, lowCount, "no low-similarity pairs found; vectors may not be clustered")
}

func TestVector_InvalidArgs(t *testing.T) {
	env := testEnv(nil)

	_, err := env.vector(0, 3, 0.1)
	assert.Error(t, err, "vector(0,...) should error")

	_, err = env.vector(4, 0, 0.1)
	assert.Error(t, err, "vector(...,0,...) should error")

	_, err = env.vector("bad", 3, 0.1)
	assert.Error(t, err, "vector with non-int dims should error")
}

func TestVector_CentroidsCached(t *testing.T) {
	env := testEnv(nil)

	// First call creates centroids.
	_, err := env.vector(8, 3, 0.1)
	require.NoError(t, err)

	c1 := env.vectorCentroids["8:3"]
	require.NotNil(t, c1, "centroids not cached after first call")

	// Second call should reuse same centroids.
	_, err = env.vector(8, 3, 0.1)
	require.NoError(t, err)

	c2 := env.vectorCentroids["8:3"]
	require.Equal(t, len(c1), len(c2), "centroids changed between calls")
	for i := range c1 {
		for j := range c1[i] {
			require.Equal(t, c1[i][j], c2[i][j], "centroid values changed between calls")
		}
	}
}

func TestVectorZipf_Format(t *testing.T) {
	env := testEnv(nil)

	result, err := env.vectorZipf(8, 5, 0.1, 2.0, 1.0)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(result, "["), "vector_zipf = %q, want []-wrapped", result)
	require.True(t, strings.HasSuffix(result, "]"), "vector_zipf = %q, want []-wrapped", result)
	inner := result[1 : len(result)-1]
	parts := strings.Split(inner, ",")
	require.Equal(t, 8, len(parts), "vector_zipf(8,...) produced %d dimensions, want 8", len(parts))
}

func TestVectorZipf_Skewed(t *testing.T) {
	env := testEnv(nil)

	// Generate many vectors with strong Zipfian skew (s=2.0).
	// Cluster 0 should dominate. We test by checking that most
	// vectors are close to centroid 0.
	const (
		dims     = 16
		clusters = 5
		n        = 500
	)

	centroids := env.getOrCreateCentroids(dims, clusters)

	clusterCounts := make([]int, clusters)
	for range n {
		result, err := env.vectorZipf(dims, clusters, 0.05, 2.0, 1.0)
		require.NoError(t, err)
		inner := result[1 : len(result)-1]
		parts := strings.Split(inner, ",")
		vec := make([]float64, dims)
		for j, p := range parts {
			fmt.Sscanf(p, "%f", &vec[j])
		}

		// Find closest centroid.
		bestIdx := 0
		bestDot := -2.0
		for ci, c := range centroids {
			var dot float64
			for j := range dims {
				dot += vec[j] * c[j]
			}
			if dot > bestDot {
				bestDot = dot
				bestIdx = ci
			}
		}
		clusterCounts[bestIdx]++
	}

	// With s=2.0, cluster 0 should have significantly more vectors.
	assert.GreaterOrEqual(t, clusterCounts[0], n/3, "vector_zipf: cluster 0 got %d/%d, expected dominant (>%d)", clusterCounts[0], n, n/3)
}

func TestVectorNorm_Format(t *testing.T) {
	env := testEnv(nil)

	result, err := env.vectorNorm(8, 5, 0.1, 2.0, 1.0)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(result, "["), "vector_norm = %q, want []-wrapped", result)
	require.True(t, strings.HasSuffix(result, "]"), "vector_norm = %q, want []-wrapped", result)
	inner := result[1 : len(result)-1]
	parts := strings.Split(inner, ",")
	require.Equal(t, 8, len(parts), "vector_norm(8,...) produced %d dimensions, want 8", len(parts))
}

func TestVectorNorm_Centered(t *testing.T) {
	env := testEnv(nil)

	// Generate many vectors with normal centroid selection centered
	// on cluster 2. Clusters near the center should be picked more.
	const (
		dims     = 16
		clusters = 5
		n        = 500
	)

	centroids := env.getOrCreateCentroids(dims, clusters)

	clusterCounts := make([]int, clusters)
	for range n {
		result, err := env.vectorNorm(dims, clusters, 0.05, 2.0, 0.8)
		require.NoError(t, err)
		inner := result[1 : len(result)-1]
		parts := strings.Split(inner, ",")
		vec := make([]float64, dims)
		for j, p := range parts {
			fmt.Sscanf(p, "%f", &vec[j])
		}

		bestIdx := 0
		bestDot := -2.0
		for ci, c := range centroids {
			var dot float64
			for j := range dims {
				dot += vec[j] * c[j]
			}
			if dot > bestDot {
				bestDot = dot
				bestIdx = ci
			}
		}
		clusterCounts[bestIdx]++
	}

	// Cluster 2 (the mean) should be the most popular.
	for i, c := range clusterCounts {
		if i != 2 {
			assert.LessOrEqual(t, c, clusterCounts[2], "vector_norm: cluster %d (%d) > center cluster 2 (%d)", i, c, clusterCounts[2])
		}
	}
}
