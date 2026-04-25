package env

import (
	"testing"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedArgsCompiled(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = convert.Constant
	env.env["items"] = 100

	seedQuery := &config.Query{
		Name: "populate_items",
		Args: config.PositionalArgs("items"),
	}
	require.NoError(t, seedQuery.CompileArgs(env.env))

	require.Len(t, seedQuery.CompiledArgs, 1)

	argSets, _, err := env.GenerateArgs(seedQuery)
	require.NoError(t, err)

	require.Len(t, argSets, 1)
	assert.Equal(t, 100, argSets[0][0])
}

func TestExpressions(t *testing.T) {
	req := &config.Request{
		Globals: map[string]any{
			"customers": 30000,
			"districts": 10,
		},
		Expressions: map[string]string{
			"cust_per_district": "customers / districts",
		},
		Run: []*config.RunItem{
			{Query: &config.Query{Args: config.PositionalArgs("cust_per_district()")}},
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	argSets, _, err := env.GenerateArgs(req.Run[0].Query)
	require.NoError(t, err)

	got, ok := argSets[0][0].(float64)
	require.True(t, ok, "cust_per_district() = %v (%T), want float64", argSets[0][0], argSets[0][0])
	assert.Equal(t, float64(3000), got)
}

func TestExpressions_WithArgs(t *testing.T) {
	req := &config.Request{
		Globals: map[string]any{
			"customers": 30000,
		},
		Expressions: map[string]string{
			"divide": "customers / args[0]",
		},
		Run: []*config.RunItem{
			{Query: &config.Query{Args: config.PositionalArgs("divide(10)")}},
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	argSets, _, err := env.GenerateArgs(req.Run[0].Query)
	require.NoError(t, err)

	got, ok := argSets[0][0].(float64)
	require.True(t, ok, "divide(10) = %v (%T), want float64", argSets[0][0], argSets[0][0])
	assert.Equal(t, float64(3000), got)
}

func TestExpressions_InvalidBody(t *testing.T) {
	req := &config.Request{
		Expressions: map[string]string{
			"bad": "undefined_var +",
		},
	}

	_, err := NewEnv(nil, "", req)
	require.Error(t, err)
}

func TestReference_LoadedIntoEnv(t *testing.T) {
	req := &config.Request{
		Reference: map[string][]map[string]any{
			"regions": {
				{"name": "eu", "region": "eu-west-2"},
				{"name": "us", "region": "us-east-1"},
			},
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	raw, ok := env.env["regions"]
	require.True(t, ok, "reference data not loaded into env")

	rows := raw.([]map[string]any)
	require.Len(t, rows, 2)
	assert.Equal(t, "eu", rows[0]["name"])
	assert.Equal(t, "us-east-1", rows[1]["region"])
}

func TestReference_IndependentCopies(t *testing.T) {
	req := &config.Request{
		Reference: map[string][]map[string]any{
			"items": {
				{"id": 1},
				{"id": 2},
			},
		},
	}

	env1, err := NewEnv(nil, "", req)
	require.NoError(t, err)
	env2, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	// Mutating env1's copy should not affect env2.
	data1 := env1.env["items"].([]map[string]any)
	data1[0] = map[string]any{"id": 999}

	data2 := env2.env["items"].([]map[string]any)
	assert.Equal(t, 1, data2[0]["id"], "reference data is shared between envs; expected independent copies")
}

func TestReference_NilIsNoOp(t *testing.T) {
	req := &config.Request{}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	// Should not panic or add unexpected keys.
	_, ok := env.env["regions"]
	assert.False(t, ok, "unexpected 'regions' key in env with nil reference")
}

func TestReference_RefRand(t *testing.T) {
	req := &config.Request{
		Reference: map[string][]map[string]any{
			"colors": {
				{"name": "red"},
				{"name": "blue"},
				{"name": "green"},
			},
		},
		Run: []*config.RunItem{
			{Query: &config.Query{Args: config.PositionalArgs("ref_rand('colors').name")}},
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	argSets, _, err := env.GenerateArgs(req.Run[0].Query)
	require.NoError(t, err)

	got, ok := argSets[0][0].(string)
	require.True(t, ok, "ref_rand('colors').name = %v (%T), want string", argSets[0][0], argSets[0][0])

	valid := got == "red" || got == "blue" || got == "green"
	assert.True(t, valid, "ref_rand('colors').name = %q, want one of red/blue/green", got)
}

func TestRow_ExpandsIntoArgs(t *testing.T) {
	req := &config.Request{
		Rows: map[string][]string{
			"customer": {"gen('email')", "gen('name')"},
		},
		Run: []*config.RunItem{
			{Query: &config.Query{Name: "insert_customer", Row: "customer", Query: "INSERT INTO customer (email, name) VALUES ($1, $2)"}},
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	argSets, _, err := env.GenerateArgs(req.Run[0].Query)
	require.NoError(t, err)

	require.Len(t, argSets, 1)
	require.Len(t, argSets[0], 2)

	email, ok := argSets[0][0].(string)
	assert.True(t, ok && email != "", "arg 0 = %v (%T), want non-empty string", argSets[0][0], argSets[0][0])
	name, ok := argSets[0][1].(string)
	assert.True(t, ok && name != "", "arg 1 = %v (%T), want non-empty string", argSets[0][1], argSets[0][1])
}

func TestRow_UsedAcrossSections(t *testing.T) {
	req := &config.Request{
		Rows: map[string][]string{
			"customer": {"gen('email')"},
		},
		Seed: []*config.Query{
			{Name: "seed_customer", Row: "customer", Query: "INSERT INTO customer (email) VALUES ($1)"},
		},
		Run: []*config.RunItem{
			{Query: &config.Query{Name: "insert_customer", Row: "customer", Query: "INSERT INTO customer (email) VALUES ($1)"}},
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	// Both queries should have compiled args from the row.
	for _, q := range []*config.Query{req.Seed[0], req.Run[0].Query} {
		assert.Len(t, q.CompiledArgs, 1, "query %s", q.Name)

		argSets, _, err := env.GenerateArgs(q)
		require.NoError(t, err)
		assert.Len(t, argSets[0], 1, "query %s", q.Name)
	}
}

func TestRow_UnknownRowName(t *testing.T) {
	req := &config.Request{
		Run: []*config.RunItem{
			{Query: &config.Query{Name: "bad", Row: "nonexistent"}},
		},
	}

	_, err := NewEnv(nil, "", req)
	require.Error(t, err)
}

func TestRow_MutuallyExclusiveWithArgs(t *testing.T) {
	req := &config.Request{
		Rows: map[string][]string{
			"customer": {"gen('email')"},
		},
		Run: []*config.RunItem{
			{Query: &config.Query{Name: "bad", Row: "customer", Args: config.PositionalArgs("gen('name')")}},
		},
	}

	_, err := NewEnv(nil, "", req)
	require.Error(t, err)
}
