package env

import (
	"strings"
	"testing"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"
	"github.com/codingconcepts/edg/pkg/gen"
)

func TestCompileArgs(t *testing.T) {
	env := testEnv(map[string][]map[string]any{
		"items": sampleRows(),
	})
	env.env["const"] = convert.Constant
	env.env["gen"] = gen.Gen
	env.env["ref_rand"] = env.refRand

	q := &config.Query{
		Args: []string{"const(42)", "gen('number:1,10')"},
	}

	if err := q.CompileArgs(env.env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	if len(q.CompiledArgs) != 2 {
		t.Fatalf("expected 2 compiled args, got %d", len(q.CompiledArgs))
	}
}

func TestCompileArgs_InvalidExpr(t *testing.T) {
	env := testEnv(nil)

	q := &config.Query{
		Args: []string{"invalid_func()"},
	}

	if err := q.CompileArgs(env.env); err == nil {
		t.Error("expected error for invalid expression, got nil")
	}
}

func TestCompileArgs_Empty(t *testing.T) {
	env := testEnv(nil)

	q := &config.Query{}

	if err := q.CompileArgs(env.env); err != nil {
		t.Fatalf("CompileArgs with no args failed: %v", err)
	}

	if len(q.CompiledArgs) != 0 {
		t.Errorf("expected 0 compiled args, got %d", len(q.CompiledArgs))
	}
}

func TestGenerateArgs_NoArgs(t *testing.T) {
	env := testEnv(nil)

	q := &config.Query{}

	argSets, _, err := env.GenerateArgs(q)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if len(argSets) != 1 || argSets[0] != nil {
		t.Errorf("expected [[nil]], got %v", argSets)
	}
}

func TestGenerateArgs_WithConst(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant

	q := &config.Query{
		Args: []string{"const(42)", "const('hello')"},
	}
	if err := q.CompileArgs(env.env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, _, err := env.GenerateArgs(q)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if len(argSets) != 1 {
		t.Fatalf("expected 1 arg set, got %d", len(argSets))
	}

	args := argSets[0]
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != 42 {
		t.Errorf("arg[0] = %v, want 42", args[0])
	}
	if args[1] != "hello" {
		t.Errorf("arg[1] = %v, want 'hello'", args[1])
	}
}

func TestGenerateArgs_ClearsOneCacheAfter(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant
	env.oneCache["test"] = "data"

	q := &config.Query{Args: []string{"const(1)"}}
	if err := q.CompileArgs(env.env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	if _, _, err := env.GenerateArgs(q); err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if len(env.oneCache) != 0 {
		t.Error("GenerateArgs did not clear oneCache")
	}
}

func TestGenerateArgs_ResetsUniqIndexAfter(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant
	env.uniqIndex = 5

	q := &config.Query{Args: []string{"const(1)"}}
	if err := q.CompileArgs(env.env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	if _, _, err := env.GenerateArgs(q); err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if env.uniqIndex != 0 {
		t.Errorf("GenerateArgs did not reset uniqIndex: got %d", env.uniqIndex)
	}
}

func TestNewEnv_RejectsUnknownQueryType(t *testing.T) {
	tests := []struct {
		name    string
		req     *config.Request
		wantErr bool
	}{
		{
			name:    "valid exec",
			req:     &config.Request{Run: []*config.RunItem{{Query: &config.Query{Name: "q", Type: config.QueryTypeExec, Query: "SELECT 1"}}}},
			wantErr: false,
		},
		{
			name:    "valid query",
			req:     &config.Request{Run: []*config.RunItem{{Query: &config.Query{Name: "q", Type: config.QueryTypeQuery, Query: "SELECT 1"}}}},
			wantErr: false,
		},
		{
			name:    "valid query_batch",
			req:     &config.Request{Run: []*config.RunItem{{Query: &config.Query{Name: "q", Type: config.QueryTypeQueryBatch, Query: "SELECT 1"}}}},
			wantErr: false,
		},
		{
			name:    "valid exec_batch",
			req:     &config.Request{Run: []*config.RunItem{{Query: &config.Query{Name: "q", Type: config.QueryTypeExecBatch, Query: "SELECT 1"}}}},
			wantErr: false,
		},
		{
			name:    "empty defaults to query",
			req:     &config.Request{Run: []*config.RunItem{{Query: &config.Query{Name: "q", Type: "", Query: "SELECT 1"}}}},
			wantErr: false,
		},
		{
			name:    "unknown type",
			req:     &config.Request{Run: []*config.RunItem{{Query: &config.Query{Name: "q", Type: "bogus", Query: "SELECT 1"}}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEnv(nil, "", tt.req)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func BenchmarkCompileArgs(b *testing.B) {
	cases := []struct {
		name string
		args []string
	}{
		{"args_1", []string{"const(42)"}},
		{"args_3", []string{"const(42)", "ref_rand('items')", "const(42)"}},
		{"args_5", []string{"const(42)", "ref_rand('items')", "const(42)", "ref_rand('items')", "const(42)"}},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			env := benchEnv(100)
			env.env["const"] = convert.Constant
			env.env["ref_rand"] = env.refRand
			b.ResetTimer()
			for range b.N {
				q := &config.Query{Args: tc.args}
				q.CompileArgs(env.env)
			}
		})
	}
}

func BenchmarkGenerateArgs(b *testing.B) {
	cases := []struct {
		name    string
		envSize int
		args    []string
		extra   func(env *Env)
	}{
		{
			name:    "scalar",
			envSize: 100,
			args:    []string{"const(42)", "ref_rand('items')"},
			extra: func(env *Env) {
				env.env["const"] = convert.Constant
				env.env["ref_rand"] = env.refRand
			},
		},
		{
			name:    "batch",
			envSize: 0,
			args:    []string{"batch(3)", "const(10)"},
			extra: func(env *Env) {
				env.env["const"] = convert.Constant
				env.env["batch"] = convert.Batch
			},
		},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			env := benchEnv(tc.envSize)
			tc.extra(env)
			q := &config.Query{Args: tc.args}
			if err := q.CompileArgs(env.env); err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			for range b.N {
				env.GenerateArgs(q)
			}
		})
	}
}

func TestGenerateArgs_MixedBatchAndScalar(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant

	// Simulate ref_each returning batch arg sets.
	batchData := [][]any{{1}, {2}, {3}}
	env.env["test_batch"] = func() [][]any { return batchData }

	q := &config.Query{Args: []string{"test_batch()", "const(10)", "const(3000)"}}
	if err := q.CompileArgs(env.env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, _, err := env.GenerateArgs(q)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if len(argSets) != 3 {
		t.Fatalf("expected 3 arg sets, got %d", len(argSets))
	}

	for i, args := range argSets {
		if len(args) != 3 {
			t.Fatalf("arg set %d: expected 3 args, got %d", i, len(args))
		}
		if args[0] != i+1 {
			t.Errorf("arg set %d: args[0] = %v, want %d", i, args[0], i+1)
		}
		if args[1] != 10 {
			t.Errorf("arg set %d: args[1] = %v, want 10", i, args[1])
		}
		if args[2] != 3000 {
			t.Errorf("arg set %d: args[2] = %v, want 3000", i, args[2])
		}
	}
}

func TestGenerateArgs_CartesianProduct(t *testing.T) {
	env := testEnv(nil)

	batchA := [][]any{{1}, {2}}
	batchB := [][]any{{"x"}, {"y"}, {"z"}}
	env.env["batchA"] = func() [][]any { return batchA }
	env.env["batchB"] = func() [][]any { return batchB }

	q := &config.Query{Args: []string{"batchA()", "batchB()"}}
	if err := q.CompileArgs(env.env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, expanded, err := env.GenerateArgs(q)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if !expanded {
		t.Fatal("expected batch expansion")
	}

	// 2 x 3 = 6 cartesian product rows.
	want := [][]any{
		{1, "x"},
		{1, "y"},
		{1, "z"},
		{2, "x"},
		{2, "y"},
		{2, "z"},
	}

	if len(argSets) != len(want) {
		t.Fatalf("expected %d arg sets, got %d", len(want), len(argSets))
	}

	for i, args := range argSets {
		if len(args) != len(want[i]) {
			t.Fatalf("arg set %d: expected %d args, got %d", i, len(want[i]), len(args))
		}
		for j, v := range args {
			if v != want[i][j] {
				t.Errorf("arg set %d[%d] = %v, want %v", i, j, v, want[i][j])
			}
		}
	}
}

func TestGenerateArgs_CartesianWithScalars(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant

	batchA := [][]any{{1}, {2}}
	batchB := [][]any{{"x"}, {"y"}}
	env.env["batchA"] = func() [][]any { return batchA }
	env.env["batchB"] = func() [][]any { return batchB }

	q := &config.Query{Args: []string{"const(99)", "batchA()", "batchB()", "const(42)"}}
	if err := q.CompileArgs(env.env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, _, err := env.GenerateArgs(q)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	want := [][]any{
		{99, 1, "x", 42},
		{99, 1, "y", 42},
		{99, 2, "x", 42},
		{99, 2, "y", 42},
	}

	if len(argSets) != len(want) {
		t.Fatalf("expected %d arg sets, got %d", len(want), len(argSets))
	}

	for i, args := range argSets {
		for j, v := range args {
			if v != want[i][j] {
				t.Errorf("arg set %d[%d] = %v, want %v", i, j, v, want[i][j])
			}
		}
	}
}

func TestGenerateArgs_BatchType(t *testing.T) {
	env := testEnv(map[string][]map[string]any{
		"categories": {
			{"name": "electronics", "markup": 1.5},
			{"name": "clothing", "markup": 1.3},
			{"name": "books", "markup": 1.1},
		},
	})
	env.env["gen"] = gen.Gen
	env.env["ref_same"] = env.refSame

	q := &config.Query{
		Type:  config.QueryTypeExecBatch,
		Count: 10,
		Size:  4,
		Args:  []string{"gen('noun')", "ref_same('categories').name", "ref_same('categories').markup"},
	}
	if err := q.CompileArgs(env.env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, _, err := env.GenerateArgs(q)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	// 10 total, batches of 4: expect 3 batches (4, 4, 2).
	if len(argSets) != 3 {
		t.Fatalf("expected 3 arg sets, got %d", len(argSets))
	}

	// Each arg set should have 3 args (one per expression).
	for i, args := range argSets {
		if len(args) != 3 {
			t.Fatalf("arg set %d: expected 3 args, got %d", i, len(args))
		}
	}

	// First two batches should have 4 CSV values, last should have 2.
	for _, args := range argSets[:2] {
		csv := string(args[0].(convert.RawSQL))
		parts := strings.Split(csv, "\x1f")
		if len(parts) != 4 {
			t.Errorf("expected 4 CSV values, got %d: %q", len(parts), csv)
		}
	}
	lastCSV := string(argSets[2][0].(convert.RawSQL))
	parts := strings.Split(lastCSV, "\x1f")
	if len(parts) != 2 {
		t.Errorf("last batch: expected 2 CSV values, got %d: %q", len(parts), lastCSV)
	}

	// Verify ref_same correlation: name and markup should match per row.
	validPairs := map[string]string{
		"'electronics'": "1.5",
		"'clothing'":    "1.3",
		"'books'":       "1.1",
	}
	for _, args := range argSets {
		names := strings.Split(string(args[1].(convert.RawSQL)), "\x1f")
		markups := strings.Split(string(args[2].(convert.RawSQL)), "\x1f")
		if len(names) != len(markups) {
			t.Fatalf("name/markup length mismatch: %d vs %d", len(names), len(markups))
		}
		for j := range names {
			want, ok := validPairs[names[j]]
			if !ok {
				t.Errorf("unexpected category name: %q", names[j])
			} else if markups[j] != want {
				t.Errorf("row %d: name=%q markup=%q, want markup=%q", j, names[j], markups[j], want)
			}
		}
	}
}

func TestGenerateArgs_BatchType_GlobalRefs(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant
	env.env["products"] = 7
	env.env["batch_size"] = 3

	q := &config.Query{
		Type:  config.QueryTypeExecBatch,
		Count: "products",
		Size:  "batch_size",
		Args:  []string{"const(42)"},
	}
	if err := q.CompileArgs(env.env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, _, err := env.GenerateArgs(q)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	// 7 / 3 = 3 batches (3, 3, 1).
	if len(argSets) != 3 {
		t.Fatalf("expected 3 batches, got %d", len(argSets))
	}

	// First two batches: 3 values each.
	for _, args := range argSets[:2] {
		parts := strings.Split(string(args[0].(convert.RawSQL)), "\x1f")
		if len(parts) != 3 {
			t.Errorf("expected 3 CSV values, got %d", len(parts))
		}
	}
	// Last batch: 1 value.
	parts := strings.Split(string(argSets[2][0].(convert.RawSQL)), "\x1f")
	if len(parts) != 1 {
		t.Errorf("last batch: expected 1 CSV value, got %d", len(parts))
	}
}

func TestGenerateArgs_BatchType_SizeDefaultsToCount(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant

	q := &config.Query{
		Type:  config.QueryTypeExecBatch,
		Count: 5,
		Args:  []string{"const('x')"},
	}
	if err := q.CompileArgs(env.env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, _, err := env.GenerateArgs(q)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	// No size set, so all 5 in one batch.
	if len(argSets) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(argSets))
	}

	parts := strings.Split(string(argSets[0][0].(convert.RawSQL)), "\x1f")
	if len(parts) != 5 {
		t.Errorf("expected 5 CSV values, got %d", len(parts))
	}
}
