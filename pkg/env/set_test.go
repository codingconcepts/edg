package env

import (
	"testing"
)

func TestSetRand_Uniform(t *testing.T) {
	counts := map[any]int{}
	for range 3000 {
		v, err := setRand([]any{"a", "b", "c"}, []any{})
		if err != nil {
			t.Fatalf("setRand error: %v", err)
		}
		counts[v]++
	}

	for _, key := range []string{"a", "b", "c"} {
		if counts[key] < 800 || counts[key] > 1200 {
			t.Errorf("%q picked %d/3000 times, expected ~1000", key, counts[key])
		}
	}
}

func TestSetRand_Weighted(t *testing.T) {
	counts := map[any]int{}
	for range 10000 {
		v, err := setRand([]any{"heavy", "light"}, []any{90, 10})
		if err != nil {
			t.Fatalf("setRand error: %v", err)
		}
		counts[v]++
	}

	if counts["heavy"] < 8500 {
		t.Errorf("heavy picked %d/10000 times, expected ~9000", counts["heavy"])
	}
	if counts["light"] < 500 {
		t.Errorf("light picked %d/10000 times, expected ~1000", counts["light"])
	}
}

func TestSetRand_SingleItem(t *testing.T) {
	v, err := setRand([]any{"only"}, []any{})
	if err != nil {
		t.Fatalf("setRand error: %v", err)
	}
	if v != "only" {
		t.Errorf("setRand single item = %v, want 'only'", v)
	}
}

func TestSetRand_Empty(t *testing.T) {
	_, err := setRand([]any{}, []any{})
	if err == nil {
		t.Fatal("setRand with no values should return error")
	}
}

func TestSetRand_MismatchedWeights(t *testing.T) {
	_, err := setRand([]any{"a", "b", "c"}, []any{50, 30})
	if err == nil {
		t.Fatal("setRand with mismatched weights should return error")
	}
}

func TestSetNormal_CenterBias(t *testing.T) {
	items := []any{"a", "b", "c", "d", "e"}
	counts := map[any]int{}
	for range 10000 {
		v, err := setNormal(items, 2.0, 0.8)
		if err != nil {
			t.Fatalf("setNormal error: %v", err)
		}
		counts[v]++
	}

	// Middle item "c" (index 2) should be picked most often.
	if counts["c"] < counts["a"] {
		t.Errorf("center item 'c' (%d) should be picked more than edge 'a' (%d)", counts["c"], counts["a"])
	}
	if counts["c"] < counts["e"] {
		t.Errorf("center item 'c' (%d) should be picked more than edge 'e' (%d)", counts["c"], counts["e"])
	}
}

func TestSetNormal_SingleItem(t *testing.T) {
	v, err := setNormal([]any{"only"}, 0, 1)
	if err != nil {
		t.Fatalf("setNormal error: %v", err)
	}
	if v != "only" {
		t.Errorf("setNormal single item = %v, want 'only'", v)
	}
}

func TestSetNormal_Empty(t *testing.T) {
	_, err := setNormal([]any{}, 0, 1)
	if err == nil {
		t.Fatal("setNormal with no values should return error")
	}
}

func TestSetExp_LowIndexBias(t *testing.T) {
	items := []any{"a", "b", "c", "d", "e"}
	counts := map[any]int{}
	for range 10000 {
		v, err := setExp(items, 0.5)
		if err != nil {
			t.Fatalf("setExp error: %v", err)
		}
		counts[v]++
	}

	// First item "a" (index 0) should be picked most often.
	if counts["a"] < counts["e"] {
		t.Errorf("first item 'a' (%d) should be picked more than last 'e' (%d)", counts["a"], counts["e"])
	}
}

func TestSetExp_SingleItem(t *testing.T) {
	v, err := setExp([]any{"only"}, 0.5)
	if err != nil {
		t.Fatalf("setExp error: %v", err)
	}
	if v != "only" {
		t.Errorf("setExp single item = %v, want 'only'", v)
	}
}

func TestSetExp_Empty(t *testing.T) {
	_, err := setExp([]any{}, 0.5)
	if err == nil {
		t.Fatal("setExp with no values should return error")
	}
}

func TestSetLognormal_LowIndexBias(t *testing.T) {
	items := []any{"a", "b", "c", "d", "e"}
	counts := map[any]int{}
	for range 10000 {
		v, err := setLognormal(items, 0.5, 0.5)
		if err != nil {
			t.Fatalf("setLognormal error: %v", err)
		}
		counts[v]++
	}

	// Lower indices should be picked more than higher ones.
	if counts["a"]+counts["b"] < counts["d"]+counts["e"] {
		t.Errorf("lower items a+b (%d) should be picked more than upper d+e (%d)",
			counts["a"]+counts["b"], counts["d"]+counts["e"])
	}
}

func TestSetLognormal_SingleItem(t *testing.T) {
	v, err := setLognormal([]any{"only"}, 0, 1)
	if err != nil {
		t.Fatalf("setLognormal error: %v", err)
	}
	if v != "only" {
		t.Errorf("setLognormal single item = %v, want 'only'", v)
	}
}

func TestSetLognormal_Empty(t *testing.T) {
	_, err := setLognormal([]any{}, 0, 1)
	if err == nil {
		t.Fatal("setLognormal with no values should return error")
	}
}

func TestSetZipfian_LowIndexBias(t *testing.T) {
	items := []any{"a", "b", "c", "d", "e"}
	counts := map[any]int{}
	for range 10000 {
		v, err := setZipfian(items, 2.0, 1.0)
		if err != nil {
			t.Fatalf("setZipfian error: %v", err)
		}
		counts[v]++
	}

	// First item "a" (index 0) should dominate.
	if counts["a"] < counts["e"] {
		t.Errorf("first item 'a' (%d) should be picked more than last 'e' (%d)", counts["a"], counts["e"])
	}
}

func TestSetZipfian_SingleItem(t *testing.T) {
	v, err := setZipfian([]any{"only"}, 2.0, 1.0)
	if err != nil {
		t.Fatalf("setZipfian error: %v", err)
	}
	if v != "only" {
		t.Errorf("setZipfian single item = %v, want 'only'", v)
	}
}

func TestSetZipfian_Empty(t *testing.T) {
	_, err := setZipfian([]any{}, 2.0, 1.0)
	if err == nil {
		t.Fatal("setZipfian with no values should return error")
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

	if counts["heavy"] < 800 {
		t.Errorf("heavy picked %d/1000, expected ~900", counts["heavy"])
	}
}
