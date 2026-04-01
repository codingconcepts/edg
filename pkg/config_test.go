package pkg

import (
	"testing"

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
	env.oneCache["stale"] = "data"

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
