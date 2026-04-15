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
