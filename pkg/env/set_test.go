package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetRand_Uniform(t *testing.T) {
	counts := map[any]int{}
	for range 3000 {
		v, err := setRand([]any{"a", "b", "c"}, []any{})
		require.NoError(t, err)
		counts[v]++
	}

	for _, key := range []string{"a", "b", "c"} {
		assert.GreaterOrEqual(t, counts[key], 800, "%q picked %d/3000 times, expected ~1000", key, counts[key])
		assert.LessOrEqual(t, counts[key], 1200, "%q picked %d/3000 times, expected ~1000", key, counts[key])
	}
}

func TestSetRand_Weighted(t *testing.T) {
	counts := map[any]int{}
	for range 10000 {
		v, err := setRand([]any{"heavy", "light"}, []any{90, 10})
		require.NoError(t, err)
		counts[v]++
	}

	assert.GreaterOrEqual(t, counts["heavy"], 8500, "heavy picked %d/10000 times, expected ~9000", counts["heavy"])
	assert.GreaterOrEqual(t, counts["light"], 500, "light picked %d/10000 times, expected ~1000", counts["light"])
}

func TestSetRand_EdgeCases(t *testing.T) {
	cases := []struct {
		name    string
		items   []any
		weights []any
		want    any
		wantErr bool
	}{
		{"single item", []any{"only"}, []any{}, "only", false},
		{"empty", []any{}, []any{}, nil, true},
		{"mismatched weights", []any{"a", "b", "c"}, []any{50, 30}, nil, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := setRand(c.items, c.weights)
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, v)
		})
	}
}

func TestSetNormal_CenterBias(t *testing.T) {
	items := []any{"a", "b", "c", "d", "e"}
	counts := map[any]int{}
	for range 10000 {
		v, err := setNormal(items, 2.0, 0.8)
		require.NoError(t, err)
		counts[v]++
	}

	// Middle item "c" (index 2) should be picked most often.
	assert.Greater(t, counts["c"], counts["a"], "center item 'c' (%d) should be picked more than edge 'a' (%d)", counts["c"], counts["a"])
	assert.Greater(t, counts["c"], counts["e"], "center item 'c' (%d) should be picked more than edge 'e' (%d)", counts["c"], counts["e"])
}

func TestSetNormal_EdgeCases(t *testing.T) {
	cases := []struct {
		name    string
		items   []any
		want    any
		wantErr bool
	}{
		{"single item", []any{"only"}, "only", false},
		{"empty", []any{}, nil, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := setNormal(c.items, 0, 1)
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, v)
		})
	}
}

func TestSetExp_LowIndexBias(t *testing.T) {
	items := []any{"a", "b", "c", "d", "e"}
	counts := map[any]int{}
	for range 10000 {
		v, err := setExp(items, 0.5)
		require.NoError(t, err)
		counts[v]++
	}

	// First item "a" (index 0) should be picked most often.
	assert.Greater(t, counts["a"], counts["e"], "first item 'a' (%d) should be picked more than last 'e' (%d)", counts["a"], counts["e"])
}

func TestSetExp_EdgeCases(t *testing.T) {
	cases := []struct {
		name    string
		items   []any
		want    any
		wantErr bool
	}{
		{"single item", []any{"only"}, "only", false},
		{"empty", []any{}, nil, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := setExp(c.items, 0.5)
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, v)
		})
	}
}

func TestSetLognormal_LowIndexBias(t *testing.T) {
	items := []any{"a", "b", "c", "d", "e"}
	counts := map[any]int{}
	for range 10000 {
		v, err := setLognormal(items, 0.5, 0.5)
		require.NoError(t, err)
		counts[v]++
	}

	// Lower indices should be picked more than higher ones.
	assert.Greater(t, counts["a"]+counts["b"], counts["d"]+counts["e"],
		"lower items a+b (%d) should be picked more than upper d+e (%d)",
		counts["a"]+counts["b"], counts["d"]+counts["e"])
}

func TestSetLognormal_EdgeCases(t *testing.T) {
	cases := []struct {
		name    string
		items   []any
		want    any
		wantErr bool
	}{
		{"single item", []any{"only"}, "only", false},
		{"empty", []any{}, nil, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := setLognormal(c.items, 0, 1)
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, v)
		})
	}
}

func TestSetZipfian_LowIndexBias(t *testing.T) {
	items := []any{"a", "b", "c", "d", "e"}
	counts := map[any]int{}
	for range 10000 {
		v, err := setZipfian(items, 2.0, 1.0)
		require.NoError(t, err)
		counts[v]++
	}

	// First item "a" (index 0) should dominate.
	assert.Greater(t, counts["a"], counts["e"], "first item 'a' (%d) should be picked more than last 'e' (%d)", counts["a"], counts["e"])
}

func TestSetZipfian_EdgeCases(t *testing.T) {
	cases := []struct {
		name    string
		items   []any
		want    any
		wantErr bool
	}{
		{"single item", []any{"only"}, "only", false},
		{"empty", []any{}, nil, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := setZipfian(c.items, 2.0, 1.0)
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, v)
		})
	}
}

func TestWeightedItems_Choose(t *testing.T) {
	wi := buildWeightedItems(
		[]any{"heavy", "light"},
		[]int{90, 10},
	)

	counts := map[any]int{}
	for range 1000 {
		counts[wi.choose()]++
	}

	assert.GreaterOrEqual(t, counts["heavy"], 800, "heavy picked %d/1000, expected ~900", counts["heavy"])
}
