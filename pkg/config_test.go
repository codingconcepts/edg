package pkg

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestCompileArgs(t *testing.T) {
	env := testEnv(map[string][]map[string]any{
		"items": sampleRows(),
	})
	env.env["const"] = constant
	env.env["gen"] = gen
	env.env["ref_rand"] = env.refRand

	q := &Query{
		Args: []string{"const(42)", "gen('number:1,10')"},
	}

	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	if len(q.CompiledArgs) != 2 {
		t.Fatalf("expected 2 compiled args, got %d", len(q.CompiledArgs))
	}
}

func TestCompileArgs_InvalidExpr(t *testing.T) {
	env := testEnv(nil)

	q := &Query{
		Args: []string{"invalid_func()"},
	}

	if err := q.CompileArgs(env); err == nil {
		t.Error("expected error for invalid expression, got nil")
	}
}

func TestCompileArgs_Empty(t *testing.T) {
	env := testEnv(nil)

	q := &Query{}

	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs with no args failed: %v", err)
	}

	if len(q.CompiledArgs) != 0 {
		t.Errorf("expected 0 compiled args, got %d", len(q.CompiledArgs))
	}
}

func TestGenerateArgs_NoArgs(t *testing.T) {
	env := testEnv(nil)

	q := &Query{}

	argSets, err := q.GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if len(argSets) != 1 || argSets[0] != nil {
		t.Errorf("expected [[nil]], got %v", argSets)
	}
}

func TestGenerateArgs_WithConst(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = constant

	q := &Query{
		Args: []string{"const(42)", "const('hello')"},
	}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, err := q.GenerateArgs(env)
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
	env.env["const"] = constant
	env.oneCache["test"] = "data"

	q := &Query{Args: []string{"const(1)"}}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	if _, err := q.GenerateArgs(env); err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if len(env.oneCache) != 0 {
		t.Error("GenerateArgs did not clear oneCache")
	}
}

func TestGenerateArgs_ResetsUniqIndexAfter(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = constant
	env.uniqIndex = 5

	q := &Query{Args: []string{"const(1)"}}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	if _, err := q.GenerateArgs(env); err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if env.uniqIndex != 0 {
		t.Errorf("GenerateArgs did not reset uniqIndex: got %d", env.uniqIndex)
	}
}

func TestRequestParsesSeedSection(t *testing.T) {
	input := `
up:
  - name: create_table
    query: CREATE TABLE t (id INT PRIMARY KEY)
seed:
  - name: populate_table
    args:
      - items
    query: INSERT INTO t SELECT s FROM generate_series(1, $1) AS s
down:
  - name: drop_table
    query: DROP TABLE t
`
	var req Request
	if err := yaml.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(req.Up) != 1 {
		t.Fatalf("expected 1 up query, got %d", len(req.Up))
	}
	if len(req.Seed) != 1 {
		t.Fatalf("expected 1 seed query, got %d", len(req.Seed))
	}
	if req.Seed[0].Name != "populate_table" {
		t.Errorf("seed query name = %q, want %q", req.Seed[0].Name, "populate_table")
	}
	if len(req.Seed[0].Args) != 1 {
		t.Fatalf("expected 1 seed arg, got %d", len(req.Seed[0].Args))
	}
	if len(req.Down) != 1 {
		t.Fatalf("expected 1 down query, got %d", len(req.Down))
	}
}

func TestRequestParsesDeseedSection(t *testing.T) {
	input := `
deseed:
  - name: truncate_items
    type: exec
    query: TRUNCATE TABLE item
  - name: truncate_warehouse
    type: exec
    query: TRUNCATE TABLE warehouse
`
	var req Request
	if err := yaml.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(req.Deseed) != 2 {
		t.Fatalf("expected 2 deseed queries, got %d", len(req.Deseed))
	}
	if req.Deseed[0].Name != "truncate_items" {
		t.Errorf("deseed query name = %q, want %q", req.Deseed[0].Name, "truncate_items")
	}
	if req.Deseed[0].Type != QueryTypeExec {
		t.Errorf("deseed query type = %q, want %q", req.Deseed[0].Type, QueryTypeExec)
	}
}

func TestRequestParsesEmptySeed(t *testing.T) {
	input := `
up:
  - name: create_table
    query: CREATE TABLE t (id INT PRIMARY KEY)
`
	var req Request
	if err := yaml.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(req.Seed) != 0 {
		t.Errorf("expected 0 seed queries, got %d", len(req.Seed))
	}
}

func TestNewEnv_RejectsUnknownQueryType(t *testing.T) {
	tests := []struct {
		name    string
		req     *Request
		wantErr bool
	}{
		{
			name:    "valid exec",
			req:     &Request{Run: []*Query{{Name: "q", Type: QueryTypeExec, Query: "SELECT 1"}}},
			wantErr: false,
		},
		{
			name:    "valid query",
			req:     &Request{Run: []*Query{{Name: "q", Type: QueryTypeQuery, Query: "SELECT 1"}}},
			wantErr: false,
		},
		{
			name:    "valid query_batch",
			req:     &Request{Run: []*Query{{Name: "q", Type: QueryTypeQueryBatch, Query: "SELECT 1"}}},
			wantErr: false,
		},
		{
			name:    "valid exec_batch",
			req:     &Request{Run: []*Query{{Name: "q", Type: QueryTypeExecBatch, Query: "SELECT 1"}}},
			wantErr: false,
		},
		{
			name:    "empty defaults to query",
			req:     &Request{Run: []*Query{{Name: "q", Type: "", Query: "SELECT 1"}}},
			wantErr: false,
		},
		{
			name:    "unknown type",
			req:     &Request{Run: []*Query{{Name: "q", Type: "bogus", Query: "SELECT 1"}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEnv(nil, tt.req)
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
			env.env["const"] = constant
			env.env["ref_rand"] = env.refRand
			b.ResetTimer()
			for range b.N {
				q := &Query{Args: tc.args}
				q.CompileArgs(env)
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
				env.env["const"] = constant
				env.env["ref_rand"] = env.refRand
			},
		},
		{
			name:    "batch",
			envSize: 0,
			args:    []string{"batch(3)", "const(10)"},
			extra: func(env *Env) {
				env.env["const"] = constant
				env.env["batch"] = batch
			},
		},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			env := benchEnv(tc.envSize)
			tc.extra(env)
			q := &Query{Args: tc.args}
			if err := q.CompileArgs(env); err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			for range b.N {
				q.GenerateArgs(env)
			}
		})
	}
}

func TestGenerateArgs_MixedBatchAndScalar(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = constant

	// Simulate ref_each returning batch arg sets.
	batchData := [][]any{{1}, {2}, {3}}
	env.env["test_batch"] = func() [][]any { return batchData }

	q := &Query{Args: []string{"test_batch()", "const(10)", "const(3000)"}}
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

func TestGenerateArgs_BatchType(t *testing.T) {
	env := testEnv(map[string][]map[string]any{
		"categories": {
			{"name": "electronics", "markup": 1.5},
			{"name": "clothing", "markup": 1.3},
			{"name": "books", "markup": 1.1},
		},
	})
	env.env["gen"] = gen
	env.env["ref_same"] = env.refSame

	q := &Query{
		Type:  QueryTypeExecBatch,
		Count: 10,
		Size:  4,
		Args:  []string{"gen('noun')", "ref_same('categories').name", "ref_same('categories').markup"},
	}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, err := q.GenerateArgs(env)
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
		csv := args[0].(string)
		parts := strings.Split(csv, ",")
		if len(parts) != 4 {
			t.Errorf("expected 4 CSV values, got %d: %q", len(parts), csv)
		}
	}
	lastCSV := argSets[2][0].(string)
	parts := strings.Split(lastCSV, ",")
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
		names := strings.Split(args[1].(string), ",")
		markups := strings.Split(args[2].(string), ",")
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
	env.env["const"] = constant
	env.env["products"] = 7
	env.env["batch_size"] = 3

	q := &Query{
		Type:  QueryTypeExecBatch,
		Count: "products",
		Size:  "batch_size",
		Args:  []string{"const(42)"},
	}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, err := q.GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	// 7 / 3 = 3 batches (3, 3, 1).
	if len(argSets) != 3 {
		t.Fatalf("expected 3 batches, got %d", len(argSets))
	}

	// First two batches: 3 values each.
	for _, args := range argSets[:2] {
		parts := strings.Split(args[0].(string), ",")
		if len(parts) != 3 {
			t.Errorf("expected 3 CSV values, got %d", len(parts))
		}
	}
	// Last batch: 1 value.
	parts := strings.Split(argSets[2][0].(string), ",")
	if len(parts) != 1 {
		t.Errorf("last batch: expected 1 CSV value, got %d", len(parts))
	}
}

func TestGenerateArgs_BatchType_SizeDefaultsToCount(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = constant

	q := &Query{
		Type:  QueryTypeExecBatch,
		Count: 5,
		Args:  []string{"const('x')"},
	}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, err := q.GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	// No size set, so all 5 in one batch.
	if len(argSets) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(argSets))
	}

	parts := strings.Split(argSets[0][0].(string), ",")
	if len(parts) != 5 {
		t.Errorf("expected 5 CSV values, got %d", len(parts))
	}
}

func TestRequestParsesBatchType(t *testing.T) {
	input := `
seed:
  - name: populate_product
    type: exec_batch
    count: 100
    size: 50
    args:
      - gen('productname')
    query: |-
      INSERT INTO product (name) SELECT unnest(string_to_array('$1', ','))
`
	var req Request
	if err := yaml.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(req.Seed) != 1 {
		t.Fatalf("expected 1 seed query, got %d", len(req.Seed))
	}
	q := req.Seed[0]
	if q.Type != QueryTypeExecBatch {
		t.Errorf("type = %q, want %q", q.Type, QueryTypeExecBatch)
	}
	gotCount, err := toInt(q.Count)
	if err != nil {
		t.Fatal(err)
	}
	if gotCount != 100 {
		t.Errorf("count = %v, want 100", q.Count)
	}
	gotSize, err := toInt(q.Size)
	if err != nil {
		t.Fatal(err)
	}
	if gotSize != 50 {
		t.Errorf("size = %v, want 50", q.Size)
	}
}

func TestDurationUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"seconds", "wait: 5s", 5 * time.Second, false},
		{"milliseconds", "wait: 250ms", 250 * time.Millisecond, false},
		{"minutes", "wait: 2m", 2 * time.Minute, false},
		{"complex", "wait: 1m30s", 90 * time.Second, false},
		{"invalid", "wait: notaduration", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out struct {
				Wait Duration `yaml:"wait"`
			}
			err := yaml.Unmarshal([]byte(tt.input), &out)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if time.Duration(out.Wait) != tt.want {
				t.Errorf("got %v, want %v", time.Duration(out.Wait), tt.want)
			}
		})
	}
}
