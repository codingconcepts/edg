package pkg

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func testEnv(data map[string][]map[string]any) *Env {
	env := &Env{
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
	}
	for name, rows := range data {
		env.env[name] = rows
	}
	return env
}

func sampleRows() []map[string]any {
	return []map[string]any{
		{"id": 1, "name": "a"},
		{"id": 2, "name": "b"},
		{"id": 3, "name": "c"},
	}
}

func TestConstant(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"string", "hello"},
		{"int", 42},
		{"nil", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := constant(tt.input)
			if got != tt.input {
				t.Errorf("constant(%v) = %v, want %v", tt.input, got, tt.input)
			}
		})
	}
}

func TestExpr(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = constant
	env.env["expr"] = constant
	env.env["warehouses"] = 5

	q := &Query{
		Args: []string{"expr(warehouses * 10)", "expr(warehouses + 1)"},
	}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, err := q.GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	args := argSets[0]
	if args[0] != 50 {
		t.Errorf("expr(warehouses * 10) = %v, want 50", args[0])
	}
	if args[1] != 6 {
		t.Errorf("expr(warehouses + 1) = %v, want 6", args[1])
	}
}

func TestBareArithmetic(t *testing.T) {
	env := testEnv(nil)
	env.env["orders"] = 30000
	env.env["districts"] = 10

	q := &Query{
		Args: []string{"orders / districts"},
	}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, err := q.GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	got, ok := argSets[0][0].(float64)
	if !ok {
		t.Fatalf("orders / districts = %v (%T), want float64", argSets[0][0], argSets[0][0])
	}
	if got != 3000 {
		t.Errorf("orders / districts = %v, want 3000", got)
	}
}

func TestGen(t *testing.T) {
	result := gen("number:1,100")
	if result == nil {
		t.Fatal("gen returned nil for valid pattern")
	}

	result = gen("{number:1,100}")
	if result == nil {
		t.Fatal("gen returned nil for already-wrapped pattern")
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"number:1,10", "{number:1,10}"},
		{"{number:1,10}", "{number:1,10}"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := wrap(tt.input); got != tt.want {
				t.Errorf("wrap(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

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

func TestNURand_InRange(t *testing.T) {
	env := testEnv(nil)
	env.nurandC = map[int]int{}

	for range 1000 {
		v := env.nurand(1023, 1, 3000)
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
		v := env.nurand(1023, 1, 3000)
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

	_ = env.nurand(1023, 1, 3000)
	c1 := env.nurandC[1023]

	// Subsequent calls should use the same C.
	for range 100 {
		_ = env.nurand(1023, 1, 3000)
	}
	if env.nurandC[1023] != c1 {
		t.Errorf("NURand C changed: got %d, want %d", env.nurandC[1023], c1)
	}
}

func TestNURandN(t *testing.T) {
	env := testEnv(nil)
	env.nurandC = map[int]int{}

	result := env.nurandN(8191, 1, 100000, 5, 15)
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

func TestRefSame_ReturnsSameRow(t *testing.T) {
	rows := sampleRows()
	env := testEnv(nil)

	first := env.refSame(rows)
	second := env.refSame(rows)

	if first["id"] != second["id"] {
		t.Errorf("refSame returned different rows: %v vs %v", first["id"], second["id"])
	}
}

func TestRefSame_ClearedBetweenCycles(t *testing.T) {
	rows := sampleRows()
	env := testEnv(nil)

	first := env.refSame(rows)
	env.clearOneCache()

	// After clearing, a new random row is picked. Run enough times to
	// confirm it doesn't always match (statistically near-certain with 3 rows).
	different := false
	for range 20 {
		second := env.refSame(rows)
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

func TestSeedArgsCompiled(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = constant
	env.env["items"] = 100

	seedQuery := &Query{
		Name: "populate_items",
		Args: []string{"items"},
	}
	if err := seedQuery.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs for seed query failed: %v", err)
	}

	if len(seedQuery.CompiledArgs) != 1 {
		t.Fatalf("expected 1 compiled arg, got %d", len(seedQuery.CompiledArgs))
	}

	argSets, err := seedQuery.GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if len(argSets) != 1 {
		t.Fatalf("expected 1 arg set, got %d", len(argSets))
	}
	if argSets[0][0] != 100 {
		t.Errorf("seed arg = %v, want 100", argSets[0][0])
	}
}

func TestConfigSectionSeedValue(t *testing.T) {
	if ConfigSectionSeed != "seed" {
		t.Errorf("ConfigSectionSeed = %q, want %q", ConfigSectionSeed, "seed")
	}
}

func TestConfigSectionDeseedValue(t *testing.T) {
	if ConfigSectionDeseed != "deseed" {
		t.Errorf("ConfigSectionDeseed = %q, want %q", ConfigSectionDeseed, "deseed")
	}
}

func TestExpressions(t *testing.T) {
	req := &Request{
		Globals: map[string]any{
			"customers": 30000,
			"districts": 10,
		},
		Expressions: map[string]string{
			"cust_per_district": "customers / districts",
		},
		Run: []*Query{
			{Args: []string{"cust_per_district()"}},
		},
	}

	env, err := NewEnv(nil, req)
	if err != nil {
		t.Fatalf("NewEnv failed: %v", err)
	}

	argSets, err := req.Run[0].GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	got, ok := argSets[0][0].(float64)
	if !ok {
		t.Fatalf("cust_per_district() = %v (%T), want float64", argSets[0][0], argSets[0][0])
	}
	if got != 3000 {
		t.Errorf("cust_per_district() = %v, want 3000", got)
	}
}

func TestExpressions_WithArgs(t *testing.T) {
	req := &Request{
		Globals: map[string]any{
			"customers": 30000,
		},
		Expressions: map[string]string{
			"divide": "customers / args[0]",
		},
		Run: []*Query{
			{Args: []string{"divide(10)"}},
		},
	}

	env, err := NewEnv(nil, req)
	if err != nil {
		t.Fatalf("NewEnv failed: %v", err)
	}

	argSets, err := req.Run[0].GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	got, ok := argSets[0][0].(float64)
	if !ok {
		t.Fatalf("divide(10) = %v (%T), want float64", argSets[0][0], argSets[0][0])
	}
	if got != 3000 {
		t.Errorf("divide(10) = %v, want 3000", got)
	}
}

func TestExpressions_InvalidBody(t *testing.T) {
	req := &Request{
		Expressions: map[string]string{
			"bad": "undefined_var +",
		},
	}

	_, err := NewEnv(nil, req)
	if err == nil {
		t.Fatal("expected error for invalid expression, got nil")
	}
}

func TestBatch(t *testing.T) {
	result := batch(3)
	if len(result) != 3 {
		t.Fatalf("batch(3) returned %d sets, want 3", len(result))
	}
	for i, row := range result {
		if len(row) != 1 {
			t.Fatalf("batch row %d has %d values, want 1", i, len(row))
		}
		if row[0] != i {
			t.Errorf("batch row %d = %v, want %d", i, row[0], i)
		}
	}
}

func TestBatch_Zero(t *testing.T) {
	result := batch(0)
	if len(result) != 0 {
		t.Errorf("batch(0) returned %d sets, want 0", len(result))
	}
}

func TestGenerateArgs_Batch(t *testing.T) {
	env := testEnv(nil)
	env.env["batch"] = batch
	env.env["const"] = constant
	env.env["items"] = 30

	q := &Query{Args: []string{"batch(items / 10)", "const(10)"}}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, err := q.GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if len(argSets) != 3 {
		t.Fatalf("expected 3 arg sets, got %d", len(argSets))
	}

	for i, args := range argSets {
		if args[0] != i {
			t.Errorf("arg set %d: args[0] = %v, want %d", i, args[0], i)
		}
		if args[1] != 10 {
			t.Errorf("arg set %d: args[1] = %v, want 10", i, args[1])
		}
	}
}

func TestGenBatch(t *testing.T) {
	result := genBatch(25, 10, "email")
	if len(result) != 3 {
		t.Fatalf("genBatch(25, 10) returned %d batches, want 3", len(result))
	}

	// First two batches should have 10 emails each.
	for _, i := range []int{0, 1} {
		csv := result[i][0].(string)
		parts := strings.Split(csv, ",")
		if len(parts) != 10 {
			t.Errorf("batch %d has %d values, want 10", i, len(parts))
		}
	}

	// Last batch should have 5 emails (remainder).
	csv := result[2][0].(string)
	parts := strings.Split(csv, ",")
	if len(parts) != 5 {
		t.Errorf("last batch has %d values, want 5", len(parts))
	}

	// All emails across all batches should be unique.
	seen := map[string]bool{}
	for _, row := range result {
		for _, v := range strings.Split(row[0].(string), ",") {
			if seen[v] {
				t.Errorf("genBatch produced duplicate: %s", v)
			}
			seen[v] = true
		}
	}
	if len(seen) != 25 {
		t.Errorf("genBatch produced %d unique values, want 25", len(seen))
	}
}

func TestGenBatch_ExactMultiple(t *testing.T) {
	result := genBatch(20, 10, "email")
	if len(result) != 2 {
		t.Fatalf("genBatch(20, 10) returned %d batches, want 2", len(result))
	}
	for i, row := range result {
		parts := strings.Split(row[0].(string), ",")
		if len(parts) != 10 {
			t.Errorf("batch %d has %d values, want 10", i, len(parts))
		}
	}
}

func TestSetEnv(t *testing.T) {
	env := testEnv(nil)
	data := sampleRows()

	env.SetEnv("test_data", data)

	raw, ok := env.env["test_data"]
	if !ok {
		t.Fatal("SetEnv did not set the key")
	}

	got := raw.([]map[string]any)
	if len(got) != len(data) {
		t.Errorf("SetEnv stored %d rows, want %d", len(got), len(data))
	}
}

func TestPickWeighted(t *testing.T) {
	queries := []*Query{
		{Name: "heavy"},
		{Name: "light"},
	}
	env := &Env{
		request: &Request{
			Run: queries,
			RunWeights: map[string]int{
				"heavy": 90,
				"light": 10,
			},
		},
	}

	counts := map[string]int{}
	for range 1000 {
		q := env.pickWeighted()
		if q == nil {
			t.Fatal("pickWeighted returned nil")
		}
		counts[q.Name]++
	}

	// With 90/10 weights over 1000 iterations, "heavy" should
	// appear significantly more than "light".
	if counts["heavy"] < 800 {
		t.Errorf("heavy picked %d/1000 times, expected ~900", counts["heavy"])
	}
	if counts["light"] < 50 {
		t.Errorf("light picked %d/1000 times, expected ~100", counts["light"])
	}
}

func TestPickWeighted_NoWeights(t *testing.T) {
	env := &Env{
		request: &Request{
			Run:        []*Query{{Name: "a"}},
			RunWeights: nil,
		},
	}

	if q := env.pickWeighted(); q != nil {
		t.Errorf("pickWeighted with no weights returned %v, want nil", q.Name)
	}
}

func TestClearOneCache(t *testing.T) {
	env := testEnv(nil)
	env.oneCache[1] = "value"

	env.clearOneCache()

	if len(env.oneCache) != 0 {
		t.Errorf("clearOneCache left %d entries", len(env.oneCache))
	}
}

func TestResetUniqIndex(t *testing.T) {
	env := testEnv(nil)
	env.uniqIndex = 5

	env.resetUniqIndex()

	if env.uniqIndex != 0 {
		t.Errorf("resetUniqIndex left index at %d", env.uniqIndex)
	}
}

func benchEnv(dataSize int) *Env {
	rows := make([]map[string]any, dataSize)
	for i := range rows {
		rows[i] = map[string]any{"id": i, "name": fmt.Sprintf("item_%d", i)}
	}
	env := &Env{
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		nurandC:   map[int]int{},
		env:       map[string]any{},
		request:   &Request{},
	}
	env.env["items"] = rows
	return env
}

func BenchmarkToInt(b *testing.B) {
	cases := []struct {
		name  string
		input any
	}{
		{"int", 42},
		{"float64", 42.0},
		{"int64", int64(42)},
		{"unsupported", "42"},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				toInt(tc.input)
			}
		})
	}
}

func BenchmarkWrap(b *testing.B) {
	cases := []struct {
		name  string
		input string
	}{
		{"needs_wrap", "number:1,100"},
		{"already_wrapped", "{number:1,100}"},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				wrap(tc.input)
			}
		})
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
		env := testEnv(nil)
		env.refSame(rows)
		b.ResetTimer()
		for range b.N {
			env.refSame(rows)
		}
	})

	b.Run("cache_miss", func(b *testing.B) {
		env := testEnv(nil)
		b.ResetTimer()
		for range b.N {
			env.refSame(rows)
			env.clearOneCache()
		}
	})

	b.Run("parallel", func(b *testing.B) {
		env := testEnv(nil)
		env.refSame(rows)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				env.refSame(rows)
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
	env.nurand(1023, 1, 3000)

	b.Run("sequential", func(b *testing.B) {
		for range b.N {
			env.nurand(1023, 1, 3000)
		}
	})

	b.Run("parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				env.nurand(1023, 1, 3000)
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
				env.nurandN(8191, 1, 100000, tc.n, tc.n)
			}
		})
	}
}

func BenchmarkPickWeighted(b *testing.B) {
	cases := []struct {
		name  string
		count int
	}{
		{"queries_2", 2},
		{"queries_5", 5},
		{"queries_10", 10},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			queries := make([]*Query, tc.count)
			weights := make(map[string]int, tc.count)
			for i := range tc.count {
				name := fmt.Sprintf("q%d", i)
				queries[i] = &Query{Name: name}
				weights[name] = i + 1
			}
			env := &Env{
				request: &Request{
					Run:        queries,
					RunWeights: weights,
				},
			}
			b.ResetTimer()
			for range b.N {
				env.pickWeighted()
			}
		})
	}
}

func BenchmarkGenBatch(b *testing.B) {
	cases := []struct {
		name  string
		total int
		batch int
	}{
		{"total_10/batch_10", 10, 10},
		{"total_100/batch_10", 100, 10},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				genBatch(tc.total, tc.batch, "number:1,1000")
			}
		})
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
	got := env.refEach("SELECT id, name FROM items")

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
	got := env.refEach("SELECT 1")

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
	got := env.refEach("SELECT id, name FROM empty_table")

	if len(got) != 0 {
		t.Errorf("refEach with no rows = %v, want empty", got)
	}
}

func TestPickWeighted_SkipsUnweightedQueries(t *testing.T) {
	env := &Env{
		request: &Request{
			Run: []*Query{{Name: "weighted"}, {Name: "unweighted"}},
			RunWeights: map[string]int{
				"weighted": 100,
			},
		},
	}

	for range 100 {
		q := env.pickWeighted()
		if q == nil {
			t.Fatal("pickWeighted returned nil")
		}
		if q.Name != "weighted" {
			t.Errorf("pickWeighted returned %q, want only 'weighted'", q.Name)
		}
	}
}

func TestRunOnce_NoWeights(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO t1").WillReturnResult(driver.ResultNoRows)
	mock.ExpectExec("INSERT INTO t2").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &Request{
			Run: []*Query{
				{Name: "q1", Type: QueryTypeExec, Query: "INSERT INTO t1 VALUES (1)"},
				{Name: "q2", Type: QueryTypeExec, Query: "INSERT INTO t2 VALUES (2)"},
			},
		},
	}

	if err := env.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunOnce_WithWeights(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	// Only one query should run per call.
	mock.ExpectExec("INSERT").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &Request{
			Run: []*Query{
				{Name: "q1", Type: QueryTypeExec, Query: "INSERT INTO t1 VALUES (1)"},
				{Name: "q2", Type: QueryTypeExec, Query: "INSERT INTO t2 VALUES (2)"},
			},
			RunWeights: map[string]int{
				"q1": 50,
				"q2": 50,
			},
		},
	}

	if err := env.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestInitFrom(t *testing.T) {
	sourceRows := []map[string]any{
		{"id": 1, "name": "a"},
		{"id": 2, "name": "b"},
	}

	source := &Env{
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env:       map[string]any{"load_items": sourceRows},
		request:   &Request{},
	}

	target := &Env{
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &Request{
			Init: []*Query{
				{Name: "load_items", Type: QueryTypeQuery},
			},
		},
	}

	target.InitFrom(source)

	raw, ok := target.env["load_items"]
	if !ok {
		t.Fatal("InitFrom did not copy data")
	}
	copied := raw.([]map[string]any)
	if len(copied) != 2 {
		t.Fatalf("InitFrom copied %d rows, want 2", len(copied))
	}
	if copied[0]["id"] != 1 {
		t.Errorf("copied row 0 id = %v, want 1", copied[0]["id"])
	}
}

func TestInitFrom_SkipsExecQueries(t *testing.T) {
	source := &Env{
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &Request{},
	}

	target := &Env{
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &Request{
			Init: []*Query{
				{Name: "setup", Type: QueryTypeExec},
			},
		},
	}

	target.InitFrom(source)

	if _, ok := target.env["setup"]; ok {
		t.Error("InitFrom should skip exec-type queries")
	}
}

func TestInitFrom_IndependentCopies(t *testing.T) {
	sourceRows := []map[string]any{
		{"id": 1},
		{"id": 2},
		{"id": 3},
	}

	source := &Env{
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env:       map[string]any{"items": sourceRows},
		request:   &Request{},
	}

	target := &Env{
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &Request{
			Init: []*Query{
				{Name: "items", Type: QueryTypeQuery},
			},
		},
	}

	target.InitFrom(source)

	// Modifying the target's copy should not affect the source.
	targetData := target.env["items"].([]map[string]any)
	targetData[0] = map[string]any{"id": 999}

	if sourceRows[0]["id"] != 1 {
		t.Error("InitFrom did not create an independent copy; source was mutated")
	}
}

func TestRunSection_Exec(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &Request{},
	}

	queries := []*Query{
		{Name: "create_t", Type: QueryTypeExec, Query: "CREATE TABLE t (id INT)"},
	}

	if err := env.runSection(context.Background(), queries, ConfigSectionUp); err != nil {
		t.Fatalf("runSection error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunSection_InlinesArgsForNonRunSection(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	// The seed section inlines $1 → 100, so the executed query should
	// contain the literal value, not a bind parameter.
	mock.ExpectExec("INSERT INTO items SELECT generate_series\\(1, 100\\)").
		WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env: map[string]any{
			"const": constant,
			"items": 100,
		},
		request: &Request{},
	}

	q := &Query{
		Name:  "seed_items",
		Type:  QueryTypeExec,
		Query: "INSERT INTO items SELECT generate_series(1, $1)",
		Args:  []string{"items"},
	}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	if err := env.runSection(context.Background(), []*Query{q}, ConfigSectionSeed); err != nil {
		t.Fatalf("runSection error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunSection_RunSectionPassesArgs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO orders").
		WithArgs(42).
		WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env: map[string]any{
			"const": constant,
		},
		request: &Request{},
	}

	q := &Query{
		Name:  "insert_order",
		Type:  QueryTypeExec,
		Query: "INSERT INTO orders VALUES ($1)",
		Args:  []string{"const(42)"},
	}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	if err := env.runSection(context.Background(), []*Query{q}, ConfigSectionRun); err != nil {
		t.Fatalf("runSection error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunSection_WaitRespectsContextCancel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &Request{},
	}

	q := &Query{
		Name:  "slow",
		Type:  QueryTypeExec,
		Query: "INSERT INTO t VALUES (1)",
		Wait:  Duration(10 * time.Second),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err = env.runSection(ctx, []*Query{q}, ConfigSectionRun)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runSection error = %v, want context.Canceled", err)
	}
}

func TestRunSection_QueryStoresResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2),
	)

	env := &Env{
		db:        db,
		oneCache:  map[uintptr]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &Request{},
	}

	queries := []*Query{
		{Name: "items", Type: QueryTypeQuery, Query: "SELECT id FROM items"},
	}

	if err := env.runSection(context.Background(), queries, ConfigSectionInit); err != nil {
		t.Fatalf("runSection error: %v", err)
	}

	data, ok := env.env["items"].([]map[string]any)
	if !ok {
		t.Fatal("runSection did not store query results")
	}
	if len(data) != 2 {
		t.Errorf("stored %d rows, want 2", len(data))
	}
}
