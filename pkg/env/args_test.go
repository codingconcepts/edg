package env

import (
	"strings"
	"testing"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"
	"github.com/codingconcepts/edg/pkg/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileArgs(t *testing.T) {
	cases := []struct {
		name    string
		data    map[string][]map[string]any
		setup   func(*Env)
		args    config.QueryArgs
		wantErr bool
		wantLen int
	}{
		{
			name: "valid expressions",
			data: map[string][]map[string]any{"items": sampleRows()},
			setup: func(e *Env) {
				e.env["const"] = convert.Constant
				e.env["gen"] = gen.Gen
				e.env["ref_rand"] = e.refRand
			},
			args:    config.PositionalArgs("const(42)", "gen('number:1,10')"),
			wantLen: 2,
		},
		{
			name:    "invalid expression",
			args:    config.PositionalArgs("invalid_func()"),
			wantErr: true,
		},
		{
			name:    "empty args",
			wantLen: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := testEnv(c.data)
			if c.setup != nil {
				c.setup(env)
			}

			q := &config.Query{Args: c.args}
			err := q.CompileArgs(env.env)

			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, q.CompiledArgs, c.wantLen)
		})
	}
}

func TestGenerateArgs_NoArgs(t *testing.T) {
	env := testEnv(nil)

	q := &config.Query{}

	argSets, _, err := env.GenerateArgs(q)
	require.NoError(t, err)

	require.Len(t, argSets, 1)
	assert.Nil(t, argSets[0])
}

func TestGenerateArgs_WithConst(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant

	q := &config.Query{
		Args: config.PositionalArgs("const(42)", "const('hello')"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	argSets, _, err := env.GenerateArgs(q)
	require.NoError(t, err)

	require.Len(t, argSets, 1)

	args := argSets[0]
	require.Len(t, args, 2)
	assert.Equal(t, 42, args[0])
	assert.Equal(t, "hello", args[1])
}

func TestGenerateArgs_ClearsOneCacheAfter(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant
	env.oneCache["test"] = "data"

	q := &config.Query{Args: config.PositionalArgs("const(1)")}
	require.NoError(t, q.CompileArgs(env.env))

	_, _, err := env.GenerateArgs(q)
	require.NoError(t, err)

	assert.Empty(t, env.oneCache)
}

func TestGenerateArgs_ResetsUniqIndexAfter(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant
	env.uniqIndex = 5

	q := &config.Query{Args: config.PositionalArgs("const(1)")}
	require.NoError(t, q.CompileArgs(env.env))

	_, _, err := env.GenerateArgs(q)
	require.NoError(t, err)

	assert.Equal(t, 0, env.uniqIndex)
}

func TestNewEnv_RejectsUnknownQueryType(t *testing.T) {
	tests := []struct {
		name   string
		req    *config.Request
		expErr string
	}{
		{
			name:   "valid exec",
			req:    &config.Request{Run: []*config.RunItem{{Query: &config.Query{Name: "q", Type: config.QueryTypeExec, Query: "SELECT 1"}}}},
			expErr: "",
		},
		{
			name:   "valid query",
			req:    &config.Request{Run: []*config.RunItem{{Query: &config.Query{Name: "q", Type: config.QueryTypeQuery, Query: "SELECT 1"}}}},
			expErr: "",
		},
		{
			name:   "valid query_batch",
			req:    &config.Request{Run: []*config.RunItem{{Query: &config.Query{Name: "q", Type: config.QueryTypeQueryBatch, Query: "SELECT 1"}}}},
			expErr: "",
		},
		{
			name:   "valid exec_batch",
			req:    &config.Request{Run: []*config.RunItem{{Query: &config.Query{Name: "q", Type: config.QueryTypeExecBatch, Query: "SELECT 1"}}}},
			expErr: "",
		},
		{
			name:   "empty defaults to query",
			req:    &config.Request{Run: []*config.RunItem{{Query: &config.Query{Name: "q", Type: "", Query: "SELECT 1"}}}},
			expErr: "",
		},
		{
			name:   "unknown type",
			req:    &config.Request{Run: []*config.RunItem{{Query: &config.Query{Name: "q", Type: "bogus", Query: "SELECT 1"}}}},
			expErr: `unknown query type "bogus" in run query 0 (q)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEnv(nil, "", tt.req)
			if tt.expErr != "" {
				require.EqualError(t, err, tt.expErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func BenchmarkCompileArgs(b *testing.B) {
	cases := []struct {
		name string
		args config.QueryArgs
	}{
		{"args_1", config.PositionalArgs("const(42)")},
		{"args_3", config.PositionalArgs("const(42)", "ref_rand('items')", "const(42)")},
		{"args_5", config.PositionalArgs("const(42)", "ref_rand('items')", "const(42)", "ref_rand('items')", "const(42)")},
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
		args    config.QueryArgs
		extra   func(env *Env)
	}{
		{
			name:    "scalar",
			envSize: 100,
			args:    config.PositionalArgs("const(42)", "ref_rand('items')"),
			extra: func(env *Env) {
				env.env["const"] = convert.Constant
				env.env["ref_rand"] = env.refRand
			},
		},
		{
			name:    "batch",
			envSize: 0,
			args:    config.PositionalArgs("batch(3)", "const(10)"),
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

	q := &config.Query{Args: config.PositionalArgs("test_batch()", "const(10)", "const(3000)")}
	require.NoError(t, q.CompileArgs(env.env))

	argSets, _, err := env.GenerateArgs(q)
	require.NoError(t, err)

	require.Len(t, argSets, 3)

	for i, args := range argSets {
		require.Len(t, args, 3, "arg set %d", i)
		assert.Equal(t, i+1, args[0], "arg set %d: args[0]", i)
		assert.Equal(t, 10, args[1], "arg set %d: args[1]", i)
		assert.Equal(t, 3000, args[2], "arg set %d: args[2]", i)
	}
}

func TestGenerateArgs_CartesianProduct(t *testing.T) {
	env := testEnv(nil)

	batchA := [][]any{{1}, {2}}
	batchB := [][]any{{"x"}, {"y"}, {"z"}}
	env.env["batchA"] = func() [][]any { return batchA }
	env.env["batchB"] = func() [][]any { return batchB }

	q := &config.Query{Args: config.PositionalArgs("batchA()", "batchB()")}
	require.NoError(t, q.CompileArgs(env.env))

	argSets, expanded, err := env.GenerateArgs(q)
	require.NoError(t, err)

	require.True(t, expanded, "expected batch expansion")

	// 2 x 3 = 6 cartesian product rows.
	want := [][]any{
		{1, "x"},
		{1, "y"},
		{1, "z"},
		{2, "x"},
		{2, "y"},
		{2, "z"},
	}

	require.Len(t, argSets, len(want))

	for i, args := range argSets {
		require.Len(t, args, len(want[i]), "arg set %d", i)
		for j, v := range args {
			assert.Equal(t, want[i][j], v, "arg set %d[%d]", i, j)
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

	q := &config.Query{Args: config.PositionalArgs("const(99)", "batchA()", "batchB()", "const(42)")}
	require.NoError(t, q.CompileArgs(env.env))

	argSets, _, err := env.GenerateArgs(q)
	require.NoError(t, err)

	want := [][]any{
		{99, 1, "x", 42},
		{99, 1, "y", 42},
		{99, 2, "x", 42},
		{99, 2, "y", 42},
	}

	require.Len(t, argSets, len(want))

	for i, args := range argSets {
		for j, v := range args {
			assert.Equal(t, want[i][j], v, "arg set %d[%d]", i, j)
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
		Args:  config.PositionalArgs("gen('noun')", "ref_same('categories').name", "ref_same('categories').markup"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	argSets, _, err := env.GenerateArgs(q)
	require.NoError(t, err)

	// 10 total, batches of 4: expect 3 batches (4, 4, 2).
	require.Len(t, argSets, 3)

	// Each arg set should have 3 args (one per expression).
	for i, args := range argSets {
		require.Len(t, args, 3, "arg set %d", i)
	}

	// First two batches should have 4 CSV values, last should have 2.
	for _, args := range argSets[:2] {
		csv := string(args[0].(convert.RawSQL))
		parts := strings.Split(csv, convert.Sep)
		assert.Len(t, parts, 4)
	}
	lastCSV := string(argSets[2][0].(convert.RawSQL))
	parts := strings.Split(lastCSV, convert.Sep)
	assert.Len(t, parts, 2)

	// Verify ref_same correlation: name and markup should match per row.
	validPairs := map[string]string{
		"'electronics'": "1.5",
		"'clothing'":    "1.3",
		"'books'":       "1.1",
	}
	for _, args := range argSets {
		names := strings.Split(string(args[1].(convert.RawSQL)), convert.Sep)
		markups := strings.Split(string(args[2].(convert.RawSQL)), convert.Sep)
		require.Len(t, names, len(markups), "name/markup length mismatch")
		for j := range names {
			want, ok := validPairs[names[j]]
			if assert.True(t, ok, "unexpected category name: %q", names[j]) {
				assert.Equal(t, want, markups[j], "row %d: name=%q", j, names[j])
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
		Args:  config.PositionalArgs("const(42)"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	argSets, _, err := env.GenerateArgs(q)
	require.NoError(t, err)

	// 7 / 3 = 3 batches (3, 3, 1).
	require.Len(t, argSets, 3)

	// First two batches: 3 values each.
	for _, args := range argSets[:2] {
		parts := strings.Split(string(args[0].(convert.RawSQL)), convert.Sep)
		assert.Len(t, parts, 3)
	}
	// Last batch: 1 value.
	parts := strings.Split(string(argSets[2][0].(convert.RawSQL)), convert.Sep)
	assert.Len(t, parts, 1)
}

func TestGenerateArgs_BatchType_SizeDefaultsToCount(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant

	q := &config.Query{
		Type:  config.QueryTypeExecBatch,
		Count: 5,
		Args:  config.PositionalArgs("const('x')"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	argSets, _, err := env.GenerateArgs(q)
	require.NoError(t, err)

	// No size set, so all 5 in one batch.
	require.Len(t, argSets, 1)

	parts := strings.Split(string(argSets[0][0].(convert.RawSQL)), convert.Sep)
	assert.Len(t, parts, 5)
}
