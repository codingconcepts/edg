package env

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"

	"github.com/expr-lang/expr/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpr(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant
	env.env["expr"] = convert.Constant
	env.env["warehouses"] = 5

	q := &config.Query{
		Args: config.PositionalArgs("expr(warehouses * 10)", "expr(warehouses + 1)"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	argSets, _, err := env.GenerateArgs(q)
	require.NoError(t, err)

	args := argSets[0]
	assert.Equal(t, 50, args[0])
	assert.Equal(t, 6, args[1])
}

func TestBareArithmetic(t *testing.T) {
	env := testEnv(nil)
	env.env["orders"] = 30000
	env.env["districts"] = 10

	q := &config.Query{
		Args: config.PositionalArgs("orders / districts"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	argSets, _, err := env.GenerateArgs(q)
	require.NoError(t, err)

	got, ok := argSets[0][0].(float64)
	require.True(t, ok, "orders / districts = %v (%T), want float64", argSets[0][0], argSets[0][0])
	assert.Equal(t, float64(3000), got)
}

func TestGenerateArgs_Batch(t *testing.T) {
	env := testEnv(nil)
	env.env["batch"] = convert.Batch
	env.env["const"] = convert.Constant
	env.env["items"] = 30

	q := &config.Query{Args: config.PositionalArgs("batch(items / 10)", "const(10)")}
	require.NoError(t, q.CompileArgs(env.env))

	argSets, _, err := env.GenerateArgs(q)
	require.NoError(t, err)

	require.Len(t, argSets, 3)

	for i, args := range argSets {
		assert.Equal(t, i, args[0], "arg set %d: args[0]", i)
		assert.Equal(t, 10, args[1], "arg set %d: args[1]", i)
	}
}

func TestSetEnv(t *testing.T) {
	env := testEnv(nil)
	data := sampleRows()

	env.SetEnv("test_data", data)

	raw, ok := env.env["test_data"]
	require.True(t, ok, "SetEnv did not set the key")

	got := raw.([]map[string]any)
	assert.Len(t, got, len(data))
}

func TestUniq(t *testing.T) {
	env := testEnv(nil)
	env.env["regex"] = func(pattern string) (string, error) {
		return "ABC", nil
	}
	env.uniqSeen = map[string]map[any]struct{}{}
	env.uniqProg = map[string]*vm.Program{}

	v, err := env.uniq("regex('[A-Z]{3}')")
	require.NoError(t, err)
	assert.Equal(t, "ABC", v)

	_, err = env.uniq("regex('[A-Z]{3}')", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate unique value after 5 attempts")
}

func TestUniq_MultipleValues(t *testing.T) {
	env := testEnv(nil)
	counter := 0
	env.env["const"] = func(v any) any { return v }
	env.env["gen"] = func(pattern string) string {
		counter++
		return fmt.Sprintf("val_%d", counter)
	}
	env.uniqSeen = map[string]map[any]struct{}{}
	env.uniqProg = map[string]*vm.Program{}

	seen := map[any]bool{}
	for range 10 {
		v, err := env.uniq("gen('word')")
		require.NoError(t, err)
		assert.False(t, seen[v], "duplicate value: %v", v)
		seen[v] = true
	}
}

func TestUniq_ResetBetweenQueries(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = func(v any) any { return v }
	env.env["regex"] = func(pattern string) (string, error) {
		return "FIXED", nil
	}
	env.uniqSeen = map[string]map[any]struct{}{}
	env.uniqProg = map[string]*vm.Program{}

	v, err := env.uniq("regex('[A-Z]{5}')")
	require.NoError(t, err)
	assert.Equal(t, "FIXED", v)

	env.resetUniqIndex()

	v, err = env.uniq("regex('[A-Z]{5}')")
	require.NoError(t, err)
	assert.Equal(t, "FIXED", v)
}

func TestArg_DependentColumn(t *testing.T) {
	req := &config.Request{
		Run: []*config.RunItem{
			{Query: &config.Query{Args: config.PositionalArgs(
				"gen('firstname')",
				"gen('lastname')",
				`arg(0) + " " + arg(1)`,
			)}},
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	argSets, _, err := env.GenerateArgs(req.Run[0].Query)
	require.NoError(t, err)

	first := argSets[0][0].(string)
	last := argSets[0][1].(string)
	full := argSets[0][2].(string)
	want := first + " " + last
	assert.Equal(t, want, full)
}

func TestArg_OutOfRange(t *testing.T) {
	req := &config.Request{
		Run: []*config.RunItem{
			{Query: &config.Query{Args: config.PositionalArgs("arg(0)")}},
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	_, _, err = env.GenerateArgs(req.Run[0].Query)
	require.Error(t, err)
}

func TestArg_Named(t *testing.T) {
	req := &config.Request{
		Run: []*config.RunItem{
			{Query: &config.Query{Args: config.QueryArgs{
				Exprs: []string{
					"gen('firstname')",
					"gen('lastname')",
					`arg('first') + " " + arg('last')`,
				},
				Names: map[string]int{"first": 0, "last": 1, "full": 2},
			}}},
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	argSets, _, err := env.GenerateArgs(req.Run[0].Query)
	require.NoError(t, err)

	first := argSets[0][0].(string)
	last := argSets[0][1].(string)
	full := argSets[0][2].(string)
	assert.Equal(t, first+" "+last, full)
}

func TestArg_NamedUnknown(t *testing.T) {
	req := &config.Request{
		Run: []*config.RunItem{
			{Query: &config.Query{Args: config.QueryArgs{
				Exprs: []string{`arg('missing')`},
				Names: map[string]int{},
			}}},
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	_, _, err = env.GenerateArgs(req.Run[0].Query)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `arg("missing")`)
}

func TestArg_Batch(t *testing.T) {
	req := &config.Request{
		Seed: []*config.Query{
			{
				Name:  "seed",
				Type:  config.QueryTypeExecBatch,
				Count: 3,
				Args: config.PositionalArgs(
					"gen('firstname')",
					"gen('lastname')",
					`arg(0) + " " + arg(1)`,
				),
				Query: "INSERT INTO t VALUES ($1, $2, $3)",
			},
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	argSets, _, err := env.GenerateArgs(req.Seed[0])
	require.NoError(t, err)

	// Single batch of 3 rows, each arg is a CSV string.
	// sqlFormatValue wraps strings in quotes, so the full name
	// is computed from raw values then formatted: 'First Last'.
	firsts := strings.Split(string(argSets[0][0].(convert.RawSQL)), convert.Sep)
	lasts := strings.Split(string(argSets[0][1].(convert.RawSQL)), convert.Sep)
	fulls := strings.Split(string(argSets[0][2].(convert.RawSQL)), convert.Sep)

	for i := range 3 {
		// Strip quotes added by sqlFormatValue.
		first := strings.Trim(firsts[i], "'")
		last := strings.Trim(lasts[i], "'")
		full := strings.Trim(fulls[i], "'")
		want := first + " " + last
		assert.Equal(t, want, full, "row %d", i)
	}
}

func TestLocal_ReturnsValue(t *testing.T) {
	env := testEnv(nil)
	env.txLocals = map[string]any{"amount": 42}

	got, err := env.local("amount")
	require.NoError(t, err)
	assert.Equal(t, 42, got)
}

func TestLocal_NotInTransaction(t *testing.T) {
	env := testEnv(nil)

	_, err := env.local("amount")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not inside a transaction")
}

func TestLocal_Undefined(t *testing.T) {
	env := testEnv(nil)
	env.txLocals = map[string]any{}

	_, err := env.local("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not defined")
}

func TestEvalLocals(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant

	tx := &config.Transaction{
		Name:   "test_tx",
		Locals: map[string]string{"amount": "const(99)"},
	}
	require.NoError(t, tx.CompileLocals(env.env))

	require.NoError(t, env.evalLocals(tx))

	assert.Equal(t, 99, env.txLocals["amount"])
}

func TestClearLocals(t *testing.T) {
	env := testEnv(nil)
	env.txLocals = map[string]any{"amount": 42}

	env.clearLocals()

	assert.Nil(t, env.txLocals)
}
