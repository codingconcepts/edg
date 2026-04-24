package env

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNURand_InRange(t *testing.T) {
	env := testEnv(nil)
	env.nurandC = map[int]int{}

	for range 1000 {
		v, err := env.nuRand(1023, 1, 3000)
		require.NoError(t, err)
		require.True(t, v >= 1 && v <= 3000, "nurand(1023, 1, 3000) = %d, out of range [1, 3000]", v)
	}
}

func TestNURand_NonUniform(t *testing.T) {
	env := testEnv(nil)
	env.nurandC = map[int]int{}

	// With A=1023, x=1, y=3000, NURand should produce a non-uniform
	// distribution. Bucket into 10 bins and verify they aren't all equal.
	bins := make([]int, 10)
	for range 10000 {
		v, err := env.nuRand(1023, 1, 3000)
		require.NoError(t, err)
		bins[(v-1)*10/3000]++
	}

	allSame := true
	for _, b := range bins {
		if b != bins[0] {
			allSame = false
			break
		}
	}
	assert.False(t, allSame, "nurand produced perfectly uniform distribution; expected non-uniform")
}

func TestNURand_ConstantC(t *testing.T) {
	env := testEnv(nil)
	env.nurandC = map[int]int{}

	_, err := env.nuRand(1023, 1, 3000)
	require.NoError(t, err)
	c1 := env.nurandC[1023]

	// Subsequent calls should use the same C.
	for range 100 {
		_, _ = env.nuRand(1023, 1, 3000)
	}
	assert.Equal(t, c1, env.nurandC[1023], "NURand C changed")
}

func TestNURandN(t *testing.T) {
	env := testEnv(nil)
	env.nurandC = map[int]int{}

	result, err := env.nuRandN(8191, 1, 100000, 5, 15)
	require.NoError(t, err)
	parts := strings.Split(result, ",")

	assert.True(t, len(parts) >= 5 && len(parts) <= 15, "nurandN returned %d items, want 5-15", len(parts))

	seen := map[string]bool{}
	for _, v := range parts {
		assert.False(t, seen[v], "nurandN returned duplicate value: %v", v)
		seen[v] = true
	}
}

func TestNormRand_Distribution(t *testing.T) {
	env := testEnv(nil)

	const (
		mean   = 500
		stddev = 50
		min    = 1
		max    = 1000
		n      = 50000
	)

	sum := 0.0
	within1 := 0
	within2 := 0
	within3 := 0

	for range n {
		v, err := env.normRand(mean, stddev, min, max)
		require.NoError(t, err)
		require.True(t, v >= min && v <= max, "normRand value %v outside [%d, %d]", v, min, max)
		sum += v

		dist := v - mean
		if dist < 0 {
			dist = -dist
		}
		switch {
		case dist <= stddev:
			within1++
			within2++
			within3++
		case dist <= 2*stddev:
			within2++
			within3++
		case dist <= 3*stddev:
			within3++
		}
	}

	// Observed mean should be close to the configured mean.
	observedMean := sum / n
	assert.True(t, observedMean >= mean-2 && observedMean <= mean+2, "observed mean = %.1f, want ~%d", observedMean, mean)

	// Empirical rule: ~68 / ~95 / ~99.7 within 1 / 2 / 3 stddevs.
	pct1 := float64(within1) / n
	pct2 := float64(within2) / n
	pct3 := float64(within3) / n

	assert.True(t, pct1 >= 0.65 && pct1 <= 0.71, "within 1 stddev = %.1f%%, want ~68%%", pct1*100)
	assert.True(t, pct2 >= 0.93 && pct2 <= 0.97, "within 2 stddevs = %.1f%%, want ~95%%", pct2*100)
	assert.True(t, pct3 >= 0.99, "within 3 stddevs = %.1f%%, want ~99.7%%", pct3*100)
}

func TestNormRandN(t *testing.T) {
	env := testEnv(nil)

	result, err := env.normRandN(500, 50, 1, 1000, 5, 15)
	require.NoError(t, err)
	parts := strings.Split(result, ",")

	assert.True(t, len(parts) >= 5 && len(parts) <= 15, "normRandN returned %d items, want 5-15", len(parts))

	seen := map[string]bool{}
	for _, v := range parts {
		assert.False(t, seen[v], "normRandN returned duplicate value: %v", v)
		seen[v] = true

		var num int
		fmt.Sscanf(v, "%d", &num)
		assert.True(t, num >= 1 && num <= 1000, "normRandN value %d outside [1, 1000]", num)
	}
}

func TestSeq(t *testing.T) {
	cases := []struct {
		name  string
		start int
		step  int
		calls int
		want  []int64
	}{
		{
			name:  "start 1 step 1",
			start: 1, step: 1, calls: 5,
			want: []int64{1, 2, 3, 4, 5},
		},
		{
			name:  "start 100 step 10",
			start: 100, step: 10, calls: 3,
			want: []int64{100, 110, 120},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := testEnv(nil)
			for i, want := range c.want {
				got, err := env.seq(c.start, c.step)
				require.NoError(t, err)
				assert.Equal(t, want, got, "call %d", i)
			}
		})
	}
}

func TestWeightedSampleN(t *testing.T) {
	rows := []map[string]any{
		{"id": 1, "name": "a", "weight": 100},
		{"id": 2, "name": "b", "weight": 1},
		{"id": 3, "name": "c", "weight": 1},
	}
	env := testEnv(map[string][]map[string]any{"items": rows})

	result, err := env.weightedSampleN("items", "name", "weight", 2, 2)
	require.NoError(t, err)
	require.NotEmpty(t, result, "weightedSampleN returned empty string")

	parts := strings.Split(result, ",")
	assert.Equal(t, 2, len(parts), "weightedSampleN returned %d items, want 2", len(parts))

	// Verify uniqueness.
	if len(parts) == 2 {
		assert.NotEqual(t, parts[0], parts[1], "weightedSampleN returned duplicate values")
	}
}

func TestWeightedSampleN_Weighted(t *testing.T) {
	rows := []map[string]any{
		{"id": 1, "name": "heavy", "weight": 1000},
		{"id": 2, "name": "light", "weight": 1},
	}
	env := testEnv(map[string][]map[string]any{"items": rows})

	counts := map[string]int{}
	for range 1000 {
		result, err := env.weightedSampleN("items", "name", "weight", 1, 1)
		require.NoError(t, err)
		counts[result]++
	}

	assert.GreaterOrEqual(t, counts["heavy"], 900, "heavy picked %d/1000, expected ~999", counts["heavy"])
}

func TestWeightedSampleN_EdgeCases(t *testing.T) {
	cases := []struct {
		name  string
		data  map[string][]map[string]any
		table string
		field string
		minN  int
		maxN  int
	}{
		{
			name:  "unknown name",
			table: "nonexistent", field: "id",
			minN: 1, maxN: 3,
		},
		{
			name: "zero weights",
			data: map[string][]map[string]any{
				"items": {{"id": 1, "weight": 0}, {"id": 2, "weight": 0}},
			},
			table: "items", field: "id",
			minN: 1, maxN: 1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := testEnv(c.data)
			result, err := env.weightedSampleN(c.table, c.field, "weight", c.minN, c.maxN)
			require.NoError(t, err)
			assert.Empty(t, result)
		})
	}
}

func TestDistributeSum_Basic(t *testing.T) {
	env := testEnv(nil)

	result, err := env.distributeSum(100.0, 5, 5, 2)
	require.NoError(t, err)

	parts := strings.Split(result, ",")
	assert.Equal(t, 5, len(parts), "expected 5 parts, got %d", len(parts))

	sum := 0.0
	for _, p := range parts {
		var v float64
		fmt.Sscanf(p, "%f", &v)
		assert.True(t, v >= 0, "part %s is negative", p)
		sum += v
	}
	assert.InDelta(t, 100.0, sum, 0.005, "parts sum to %f, want 100.00", sum)
}

func TestDistributeSum_VariableN(t *testing.T) {
	env := testEnv(nil)

	counts := map[int]bool{}
	for range 100 {
		result, err := env.distributeSum(500.0, 3, 7, 2)
		require.NoError(t, err)

		parts := strings.Split(result, ",")
		n := len(parts)
		assert.True(t, n >= 3 && n <= 7, "got %d parts, want 3-7", n)
		counts[n] = true

		sum := 0.0
		for _, p := range parts {
			var v float64
			fmt.Sscanf(p, "%f", &v)
			sum += v
		}
		assert.InDelta(t, 500.0, sum, 0.005, "parts sum to %f, want 500.00", sum)
	}
	assert.True(t, len(counts) > 1, "expected varying N, got only %v", counts)
}

func TestDistributeSum_SinglePart(t *testing.T) {
	env := testEnv(nil)

	result, err := env.distributeSum(42.50, 1, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, "42.50", result)
}

func TestDistributeSum_ZeroPrecision(t *testing.T) {
	env := testEnv(nil)

	result, err := env.distributeSum(1000, 4, 4, 0)
	require.NoError(t, err)

	parts := strings.Split(result, ",")
	sum := 0.0
	for _, p := range parts {
		var v float64
		fmt.Sscanf(p, "%f", &v)
		assert.Equal(t, v, float64(int(v)), "expected integer, got %s", p)
		sum += v
	}
	assert.InDelta(t, 1000.0, sum, 0.5, "parts sum to %f, want 1000", sum)
}

func TestDistributeSum_Errors(t *testing.T) {
	env := testEnv(nil)

	_, err := env.distributeSum(-10.0, 3, 5, 2)
	assert.Error(t, err, "negative total should error")

	_, err = env.distributeSum(100.0, 0, 5, 2)
	assert.Error(t, err, "minN < 1 should error")

	_, err = env.distributeSum(100.0, 5, 3, 2)
	assert.Error(t, err, "maxN < minN should error")
}

func TestDistributeWeighted_ExactProportions(t *testing.T) {
	env := testEnv(nil)

	result, err := env.distributeWeighted(1000.0, []any{50, 30, 20}, 0.0, 2)
	require.NoError(t, err)

	parts := strings.Split(result, ",")
	assert.Equal(t, 3, len(parts))

	var values [3]float64
	sum := 0.0
	for i, p := range parts {
		fmt.Sscanf(p, "%f", &values[i])
		sum += values[i]
	}
	assert.InDelta(t, 1000.0, sum, 0.005)
	assert.InDelta(t, 500.0, values[0], 0.01, "expected ~500 for weight 50")
	assert.InDelta(t, 300.0, values[1], 0.01, "expected ~300 for weight 30")
	assert.InDelta(t, 200.0, values[2], 0.01, "expected ~200 for weight 20")
}

func TestDistributeWeighted_WithNoise(t *testing.T) {
	env := testEnv(nil)

	sums := [3]float64{}
	const iterations = 1000
	for range iterations {
		result, err := env.distributeWeighted(1000.0, []any{50, 30, 20}, 0.3, 2)
		require.NoError(t, err)

		parts := strings.Split(result, ",")
		require.Equal(t, 3, len(parts))

		sum := 0.0
		for i, p := range parts {
			var v float64
			fmt.Sscanf(p, "%f", &v)
			sums[i] += v
			sum += v
		}
		assert.InDelta(t, 1000.0, sum, 0.005)
	}

	// Averages should still approximate the proportions.
	avg0 := sums[0] / iterations
	avg1 := sums[1] / iterations
	avg2 := sums[2] / iterations
	assert.True(t, avg0 > avg1, "weight 50 avg (%f) should exceed weight 30 avg (%f)", avg0, avg1)
	assert.True(t, avg1 > avg2, "weight 30 avg (%f) should exceed weight 20 avg (%f)", avg1, avg2)
}

func TestDistributeWeighted_FullNoise(t *testing.T) {
	env := testEnv(nil)

	result, err := env.distributeWeighted(500.0, []any{1, 1, 1, 1}, 1.0, 2)
	require.NoError(t, err)

	parts := strings.Split(result, ",")
	assert.Equal(t, 4, len(parts))

	sum := 0.0
	for _, p := range parts {
		var v float64
		fmt.Sscanf(p, "%f", &v)
		sum += v
	}
	assert.InDelta(t, 500.0, sum, 0.005)
}

func TestDistributeWeighted_SingleWeight(t *testing.T) {
	env := testEnv(nil)

	result, err := env.distributeWeighted(42.50, []any{1}, 0.0, 2)
	require.NoError(t, err)
	assert.Equal(t, "42.50", result)
}

func TestDistributeWeighted_Errors(t *testing.T) {
	env := testEnv(nil)

	_, err := env.distributeWeighted(-10.0, []any{1, 2}, 0.0, 2)
	assert.Error(t, err, "negative total")

	_, err = env.distributeWeighted(100.0, []any{}, 0.0, 2)
	assert.Error(t, err, "empty weights")

	_, err = env.distributeWeighted(100.0, []any{0, 0}, 0.0, 2)
	assert.Error(t, err, "all-zero weights")

	_, err = env.distributeWeighted(100.0, []any{1, 2}, 1.5, 2)
	assert.Error(t, err, "noise > 1")

	_, err = env.distributeWeighted(100.0, []any{-1, 2}, 0.0, 2)
	assert.Error(t, err, "negative weight")
}

func BenchmarkDistributeSum(b *testing.B) {
	env := benchEnv(0)
	b.Run("n_5", func(b *testing.B) {
		for range b.N {
			_, _ = env.distributeSum(1000.0, 5, 5, 2)
		}
	})
	b.Run("n_3_7", func(b *testing.B) {
		for range b.N {
			_, _ = env.distributeSum(1000.0, 3, 7, 2)
		}
	})
}

func BenchmarkNurand(b *testing.B) {
	env := benchEnv(0)
	_, _ = env.nuRand(1023, 1, 3000)
	b.Run("sequential", func(b *testing.B) {
		for range b.N {
			_, _ = env.nuRand(1023, 1, 3000)
		}
	})

	b.Run("parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = env.nuRand(1023, 1, 3000)
			}
		})
	})
}

func BenchmarkNurandN(b *testing.B) {
	cases := []struct {
		name string
		n    int
	}{
		{"n_5", 5},
		{"n_15", 15},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			env := benchEnv(0)
			b.ResetTimer()
			for range b.N {
				_, _ = env.nuRandN(8191, 1, 100000, tc.n, tc.n)
			}
		})
	}
}

func BenchmarkNormRand(b *testing.B) {
	env := benchEnv(0)

	b.Run("sequential", func(b *testing.B) {
		for range b.N {
			_, _ = env.normRand(500, 50, 1, 1000)
		}
	})

	b.Run("narrow_range", func(b *testing.B) {
		for range b.N {
			_, _ = env.normRand(50, 100, 40, 60)
		}
	})
}

func BenchmarkNormRandN(b *testing.B) {
	cases := []struct {
		name string
		n    int
	}{
		{"n_5", 5},
		{"n_15", 15},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			env := benchEnv(0)
			b.ResetTimer()
			for range b.N {
				_, _ = env.normRandN(500, 50, 1, 1000, tc.n, tc.n)
			}
		})
	}
}
