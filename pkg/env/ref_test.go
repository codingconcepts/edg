package env

import (
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRefRand(t *testing.T) {
	rows := sampleRows()
	env := testEnv(map[string][]map[string]any{"items": rows})

	result := env.refRand("items")
	if result == nil {
		t.Fatal("refRand returned nil")
	}

	if _, ok := result["id"]; !ok {
		t.Error("refRand result missing 'id' key")
	}
}

func TestRefRand_UnknownName(t *testing.T) {
	env := testEnv(nil)
	if result := env.refRand("nonexistent"); result != nil {
		t.Errorf("refRand for unknown name = %v, want nil", result)
	}
}

func TestRefRand_EmptyData(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"empty": {}})
	if result := env.refRand("empty"); result != nil {
		t.Errorf("refRand for empty data = %v, want nil", result)
	}
}

func TestRefN(t *testing.T) {
	rows := sampleRows()
	env := testEnv(map[string][]map[string]any{"items": rows})

	result := env.refN("items", "id", 2, 3)
	if result == "" {
		t.Fatal("refN returned empty string")
	}

	parts := strings.Split(result, ",")
	if len(parts) < 2 || len(parts) > 3 {
		t.Errorf("refN returned %d items, want 2-3", len(parts))
	}

	// All values should be unique.
	seen := map[string]bool{}
	for _, v := range parts {
		if seen[v] {
			t.Errorf("refN returned duplicate value: %v", v)
		}
		seen[v] = true
	}
}

func TestRefN_ClampsToDataSize(t *testing.T) {
	rows := sampleRows()
	env := testEnv(map[string][]map[string]any{"items": rows})

	result := env.refN("items", "id", 5, 10)
	parts := strings.Split(result, ",")
	if len(parts) != 3 {
		t.Errorf("refN returned %d items, want 3 (clamped to data size)", len(parts))
	}
}

func TestRefN_UnknownName(t *testing.T) {
	env := testEnv(nil)
	if result := env.refN("nonexistent", "id", 1, 3); result != "" {
		t.Errorf("refN for unknown name = %v, want empty string", result)
	}
}

func TestRefSame_ReturnsSameRow(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"users": sampleRows()})

	first := env.refSame("users")
	second := env.refSame("users")

	if first["id"] != second["id"] {
		t.Errorf("refSame returned different rows: %v vs %v", first["id"], second["id"])
	}
}

func TestRefSame_ClearedBetweenCycles(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"users": sampleRows()})

	first := env.refSame("users")
	env.clearOneCache()

	// After clearing, a new random row is picked. Run enough times to
	// confirm it doesn't always match (statistically near-certain with 3 rows).
	different := false
	for range 20 {
		second := env.refSame("users")
		if first["id"] != second["id"] {
			different = true
			break
		}
		env.clearOneCache()
	}
	if !different {
		t.Error("refSame returned the same row 20 times after cache clears; expected variation")
	}
}

func TestRefSame_UnknownName(t *testing.T) {
	env := testEnv(nil)
	if result := env.refSame("nonexistent"); result != nil {
		t.Errorf("refSame for unknown name = %v, want nil", result)
	}
}

func TestRefSame_EmptyData(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"empty": {}})
	if result := env.refSame("empty"); result != nil {
		t.Errorf("refSame for empty data = %v, want nil", result)
	}
}

func TestRefPerm_ReturnsSameRowForever(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"warehouses": sampleRows()})

	first := env.refPerm("warehouses")
	if first == nil {
		t.Fatal("refPerm returned nil")
	}

	for range 10 {
		got := env.refPerm("warehouses")
		if got["id"] != first["id"] {
			t.Errorf("refPerm changed: got %v, want %v", got["id"], first["id"])
		}
	}
}

func TestRefPerm_SurvivesCacheClear(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"warehouses": sampleRows()})

	first := env.refPerm("warehouses")

	// oneCache clear should NOT affect permCache
	env.clearOneCache()

	got := env.refPerm("warehouses")
	if got["id"] != first["id"] {
		t.Errorf("refPerm changed after clearOneCache: got %v, want %v", got["id"], first["id"])
	}
}

func TestRefPerm_UnknownName(t *testing.T) {
	env := testEnv(nil)
	if result := env.refPerm("nonexistent"); result != nil {
		t.Errorf("refPerm for unknown name = %v, want nil", result)
	}
}

func TestRefDiff_ReturnsUniqueRows(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"items": sampleRows()})

	seen := map[any]bool{}
	for range 3 {
		row := env.refDiff("items")
		if row == nil {
			t.Fatal("refDiff returned nil")
		}
		id := row["id"]
		if seen[id] {
			t.Errorf("refDiff returned duplicate id: %v", id)
		}
		seen[id] = true
	}

	if len(seen) != 3 {
		t.Errorf("expected 3 unique rows, got %d", len(seen))
	}
}

func TestRefDiff_ResetsAfterCycle(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"items": sampleRows()})

	// Exhaust all 3 rows.
	for range 3 {
		env.refDiff("items")
	}

	// Reset and verify we can get rows again.
	env.resetUniqIndex()

	row := env.refDiff("items")
	if row == nil {
		t.Fatal("refDiff returned nil after reset")
	}
}

func TestRefDiff_UnknownName(t *testing.T) {
	env := testEnv(nil)
	if result := env.refDiff("nonexistent"); result != nil {
		t.Errorf("refDiff for unknown name = %v, want nil", result)
	}
}

func TestRefEach(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name"}).
			AddRow(1, "alice").
			AddRow(2, "bob").
			AddRow(3, "charlie"),
	)

	env := &Env{db: db}
	got, err := env.refEach("SELECT id, name FROM items")
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 3 {
		t.Fatalf("refEach returned %d rows, want 3", len(got))
	}
	for i, row := range got {
		if len(row) != 2 {
			t.Errorf("row %d has %d columns, want 2", i, len(row))
		}
	}
	if got[0][0] != int64(1) {
		t.Errorf("row 0 col 0 = %v, want 1", got[0][0])
	}
	if got[2][1] != "charlie" {
		t.Errorf("row 2 col 1 = %v, want charlie", got[2][1])
	}
}

func TestRefEach_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("connection refused"))

	env := &Env{db: db}
	got, err := env.refEach("SELECT 1")

	if err == nil {
		t.Fatal("refEach with query error should return error")
	}
	if got != nil {
		t.Errorf("refEach with query error = %v, want nil", got)
	}
}

func TestRefEach_NoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name"}),
	)

	env := &Env{db: db}
	got, err := env.refEach("SELECT id, name FROM empty_table")
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 0 {
		t.Errorf("refEach with no rows = %v, want empty", got)
	}
}

func TestNURand_InRange(t *testing.T) {
	env := testEnv(nil)
	env.nurandC = map[int]int{}

	for range 1000 {
		v, err := env.nuRand(1023, 1, 3000)
		if err != nil {
			t.Fatal(err)
		}
		if v < 1 || v > 3000 {
			t.Fatalf("nurand(1023, 1, 3000) = %d, out of range [1, 3000]", v)
		}
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
		if err != nil {
			t.Fatal(err)
		}
		bins[(v-1)*10/3000]++
	}

	allSame := true
	for _, b := range bins {
		if b != bins[0] {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("nurand produced perfectly uniform distribution; expected non-uniform")
	}
}

func TestNURand_ConstantC(t *testing.T) {
	env := testEnv(nil)
	env.nurandC = map[int]int{}

	_, err := env.nuRand(1023, 1, 3000)
	if err != nil {
		t.Fatal(err)
	}
	c1 := env.nurandC[1023]

	// Subsequent calls should use the same C.
	for range 100 {
		_, _ = env.nuRand(1023, 1, 3000)
	}
	if env.nurandC[1023] != c1 {
		t.Errorf("NURand C changed: got %d, want %d", env.nurandC[1023], c1)
	}
}

func TestNURandN(t *testing.T) {
	env := testEnv(nil)
	env.nurandC = map[int]int{}

	result, err := env.nuRandN(8191, 1, 100000, 5, 15)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(result, ",")

	if len(parts) < 5 || len(parts) > 15 {
		t.Errorf("nurandN returned %d items, want 5-15", len(parts))
	}

	seen := map[string]bool{}
	for _, v := range parts {
		if seen[v] {
			t.Errorf("nurandN returned duplicate value: %v", v)
		}
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
		if err != nil {
			t.Fatal(err)
		}
		if v < min || v > max {
			t.Fatalf("normRand value %v outside [%d, %d]", v, min, max)
		}
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
	if observedMean < mean-2 || observedMean > mean+2 {
		t.Errorf("observed mean = %.1f, want ~%d", observedMean, mean)
	}

	// Empirical rule: ~68 / ~95 / ~99.7 within 1 / 2 / 3 stddevs.
	pct1 := float64(within1) / n
	pct2 := float64(within2) / n
	pct3 := float64(within3) / n

	if pct1 < 0.65 || pct1 > 0.71 {
		t.Errorf("within 1 stddev = %.1f%%, want ~68%%", pct1*100)
	}
	if pct2 < 0.93 || pct2 > 0.97 {
		t.Errorf("within 2 stddevs = %.1f%%, want ~95%%", pct2*100)
	}
	if pct3 < 0.99 {
		t.Errorf("within 3 stddevs = %.1f%%, want ~99.7%%", pct3*100)
	}
}

func TestNormRandN(t *testing.T) {
	env := testEnv(nil)

	result, err := env.normRandN(500, 50, 1, 1000, 5, 15)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(result, ",")

	if len(parts) < 5 || len(parts) > 15 {
		t.Errorf("normRandN returned %d items, want 5-15", len(parts))
	}

	seen := map[string]bool{}
	for _, v := range parts {
		if seen[v] {
			t.Errorf("normRandN returned duplicate value: %v", v)
		}
		seen[v] = true

		var num int
		fmt.Sscanf(v, "%d", &num)
		if num < 1 || num > 1000 {
			t.Errorf("normRandN value %d outside [1, 1000]", num)
		}
	}
}

func TestSeq(t *testing.T) {
	env := testEnv(nil)

	for i := range 5 {
		got, err := env.seq(1, 1)
		if err != nil {
			t.Fatal(err)
		}
		want := int64(1 + i)
		if got != want {
			t.Errorf("seq(1, 1) call %d = %d, want %d", i, got, want)
		}
	}
}

func TestSeq_StepAndStart(t *testing.T) {
	env := testEnv(nil)

	for i := range 3 {
		got, err := env.seq(100, 10)
		if err != nil {
			t.Fatal(err)
		}
		want := int64(100 + i*10)
		if got != want {
			t.Errorf("seq(100, 10) call %d = %d, want %d", i, got, want)
		}
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
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Fatal("weightedSampleN returned empty string")
	}

	parts := strings.Split(result, ",")
	if len(parts) != 2 {
		t.Errorf("weightedSampleN returned %d items, want 2", len(parts))
	}

	// Verify uniqueness.
	if len(parts) == 2 && parts[0] == parts[1] {
		t.Error("weightedSampleN returned duplicate values")
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
		if err != nil {
			t.Fatal(err)
		}
		counts[result]++
	}

	if counts["heavy"] < 900 {
		t.Errorf("heavy picked %d/1000, expected ~999", counts["heavy"])
	}
}

func TestWeightedSampleN_UnknownName(t *testing.T) {
	env := testEnv(nil)
	result, err := env.weightedSampleN("nonexistent", "id", "weight", 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Errorf("weightedSampleN for unknown name = %v, want empty", result)
	}
}

func TestWeightedSampleN_ZeroWeights(t *testing.T) {
	rows := []map[string]any{
		{"id": 1, "weight": 0},
		{"id": 2, "weight": 0},
	}
	env := testEnv(map[string][]map[string]any{"items": rows})

	result, err := env.weightedSampleN("items", "id", "weight", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Errorf("weightedSampleN with zero weights = %v, want empty", result)
	}
}

func BenchmarkRefRand(b *testing.B) {
	cases := []struct {
		name string
		rows int
	}{
		{"rows_10", 10},
		{"rows_100", 100},
		{"rows_1000", 1000},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			env := benchEnv(tc.rows)
			b.ResetTimer()
			for range b.N {
				env.refRand("items")
			}
		})
	}
}

func BenchmarkRefN(b *testing.B) {
	cases := []struct {
		name string
		rows int
		n    int
	}{
		{"rows_100/n_5", 100, 5},
		{"rows_100/n_15", 100, 15},
		{"rows_100/n_50", 100, 50},
		{"rows_1000/n_5", 1000, 5},
		{"rows_1000/n_15", 1000, 15},
		{"rows_1000/n_50", 1000, 50},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			env := benchEnv(tc.rows)
			b.ResetTimer()
			for range b.N {
				env.refN("items", "id", tc.n, tc.n)
			}
		})
	}
}

func BenchmarkRefSame(b *testing.B) {
	rows := make([]map[string]any, 100)
	for i := range rows {
		rows[i] = map[string]any{"id": i}
	}

	b.Run("cache_hit", func(b *testing.B) {
		env := testEnv(map[string][]map[string]any{"items": rows})
		env.refSame("items")
		b.ResetTimer()
		for range b.N {
			env.refSame("items")
		}
	})

	b.Run("cache_miss", func(b *testing.B) {
		env := testEnv(map[string][]map[string]any{"items": rows})
		b.ResetTimer()
		for range b.N {
			env.refSame("items")
			env.clearOneCache()
		}
	})

	b.Run("parallel", func(b *testing.B) {
		env := testEnv(map[string][]map[string]any{"items": rows})
		env.refSame("items")
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				env.refSame("items")
			}
		})
	})
}

func BenchmarkRefPerm(b *testing.B) {
	b.Run("cache_hit", func(b *testing.B) {
		env := benchEnv(100)
		env.refPerm("items")
		b.ResetTimer()
		for range b.N {
			env.refPerm("items")
		}
	})

	b.Run("cache_miss", func(b *testing.B) {
		env := benchEnv(100)
		b.ResetTimer()
		for range b.N {
			env.refPerm("items")
			env.permCacheMutex.Lock()
			clear(env.permCache)
			env.permCacheMutex.Unlock()
		}
	})

	b.Run("parallel", func(b *testing.B) {
		env := benchEnv(100)
		env.refPerm("items")
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				env.refPerm("items")
			}
		})
	})
}

func BenchmarkRefDiff(b *testing.B) {
	cases := []struct {
		name string
		rows int
	}{
		{"rows_100", 100},
		{"rows_1000", 1000},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			env := benchEnv(tc.rows)
			count := 0
			b.ResetTimer()
			for range b.N {
				if count >= tc.rows {
					env.resetUniqIndex()
					count = 0
				}
				env.refDiff("items")
				count++
			}
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
