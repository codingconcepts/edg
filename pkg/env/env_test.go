package env

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"
	"github.com/codingconcepts/edg/pkg/test"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testEnv(data map[string][]map[string]any) *Env {
	env := &Env{
		oneCache:        map[string]any{},
		permCache:       map[string]any{},
		vectorCentroids: map[string][][]float64{},
		stmtCache:       map[*config.Query]*sql.Stmt{},
		env:             map[string]any{},
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

func benchEnv(dataSize int) *Env {
	rows := make([]map[string]any, dataSize)
	for i := range rows {
		rows[i] = map[string]any{"id": i, "name": fmt.Sprintf("item_%d", i)}
	}
	env := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		nurandC:   map[int]int{},
		env:       map[string]any{},
		request:   &config.Request{},
	}
	env.env["items"] = rows
	return env
}

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

func TestPickWeighted(t *testing.T) {
	items := []*config.RunItem{
		{Query: &config.Query{Name: "heavy"}},
		{Query: &config.Query{Name: "light"}},
	}
	env := &Env{
		request: &config.Request{
			Run: items,
			RunWeights: map[string]int{
				"heavy": 90,
				"light": 10,
			},
		},
	}

	counts := map[string]int{}
	for range 1000 {
		item := env.pickWeighted()
		require.NotNil(t, item)
		counts[item.Name()]++
	}

	// With 90/10 weights over 1000 iterations, "heavy" should
	// appear significantly more than "light".
	assert.GreaterOrEqual(t, counts["heavy"], 800, "heavy picked %d/1000 times, expected ~900", counts["heavy"])
	assert.GreaterOrEqual(t, counts["light"], 50, "light picked %d/1000 times, expected ~100", counts["light"])
}

func TestPickWeighted_NoWeights(t *testing.T) {
	env := &Env{
		request: &config.Request{
			Run:        []*config.RunItem{{Query: &config.Query{Name: "a"}}},
			RunWeights: nil,
		},
	}

	assert.Nil(t, env.pickWeighted(), "pickWeighted with no weights should return nil")
}

func TestPickWeighted_SkipsUnweightedQueries(t *testing.T) {
	env := &Env{
		request: &config.Request{
			Run: []*config.RunItem{
				{Query: &config.Query{Name: "weighted"}},
				{Query: &config.Query{Name: "unweighted"}},
			},
			RunWeights: map[string]int{
				"weighted": 100,
			},
		},
	}

	for range 100 {
		item := env.pickWeighted()
		require.NotNil(t, item)
		assert.Equal(t, "weighted", item.Name(), "pickWeighted should only return 'weighted'")
	}
}

func TestClearOneCache(t *testing.T) {
	env := testEnv(nil)
	env.oneCache["test"] = "value"

	env.clearOneCache()

	assert.Empty(t, env.oneCache)
}

func TestResetUniqIndex(t *testing.T) {
	env := testEnv(nil)
	env.uniqIndex = 5

	env.resetUniqIndex()

	assert.Equal(t, 0, env.uniqIndex)
}

func TestRunIteration_NoWeights(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("INSERT INTO t1").WillReturnResult(driver.ResultNoRows)
	mock.ExpectExec("INSERT INTO t2").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &config.Request{
			Run: []*config.RunItem{
				{Query: &config.Query{Name: "q1", Type: config.QueryTypeExec, Query: "INSERT INTO t1 VALUES (1)"}},
				{Query: &config.Query{Name: "q2", Type: config.QueryTypeExec, Query: "INSERT INTO t2 VALUES (2)"}},
			},
		},
	}

	require.NoError(t, env.RunIteration(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunIteration_WithWeights(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Only one query should run per call.
	mock.ExpectExec("INSERT").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &config.Request{
			Run: []*config.RunItem{
				{Query: &config.Query{Name: "q1", Type: config.QueryTypeExec, Query: "INSERT INTO t1 VALUES (1)"}},
				{Query: &config.Query{Name: "q2", Type: config.QueryTypeExec, Query: "INSERT INTO t2 VALUES (2)"}},
			},
			RunWeights: map[string]int{
				"q1": 50,
				"q2": 50,
			},
		},
	}

	require.NoError(t, env.RunIteration(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInitFrom(t *testing.T) {
	sourceRows := []map[string]any{
		{"id": 1, "name": "a"},
		{"id": 2, "name": "b"},
	}

	source := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{"load_items": sourceRows},
		request:   &config.Request{},
	}

	target := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &config.Request{
			Init: []*config.Query{
				{Name: "load_items", Type: config.QueryTypeQuery},
			},
		},
	}

	target.InitFrom(source)

	raw, ok := target.env["load_items"]
	require.True(t, ok, "InitFrom did not copy data")
	copied := raw.([]map[string]any)
	require.Len(t, copied, 2)
	assert.Equal(t, 1, copied[0]["id"])
}

func TestInitFrom_SkipsExecQueries(t *testing.T) {
	source := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &config.Request{},
	}

	target := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &config.Request{
			Init: []*config.Query{
				{Name: "setup", Type: config.QueryTypeExec},
			},
		},
	}

	target.InitFrom(source)

	_, ok := target.env["setup"]
	assert.False(t, ok, "InitFrom should skip exec-type queries")
}

func TestInitFrom_IndependentCopies(t *testing.T) {
	sourceRows := []map[string]any{
		{"id": 1},
		{"id": 2},
		{"id": 3},
	}

	source := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{"items": sourceRows},
		request:   &config.Request{},
	}

	target := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &config.Request{
			Init: []*config.Query{
				{Name: "items", Type: config.QueryTypeQuery},
			},
		},
	}

	target.InitFrom(source)

	// Modifying the target's copy should not affect the source.
	targetData := target.env["items"].([]map[string]any)
	targetData[0] = map[string]any{"id": 999}

	assert.Equal(t, 1, sourceRows[0]["id"], "InitFrom did not create an independent copy; source was mutated")
}

func TestNewEnv_ExpressionGlobals(t *testing.T) {
	req := &config.Request{
		Globals: map[string]any{
			"warehouses": 1,
			"districts":  "warehouses * 10",
			"customers":  "districts * 3000",
		},
		GlobalsOrder: []string{"warehouses", "districts", "customers"},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	assert.Equal(t, 1, env.env["warehouses"])
	assert.Equal(t, 10, env.env["districts"])
	assert.Equal(t, 30000, env.env["customers"])
}

func TestNewEnv_ExpressionGlobals_LiteralStringKept(t *testing.T) {
	req := &config.Request{
		Globals: map[string]any{
			"city": "new york",
		},
		GlobalsOrder: []string{"city"},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	// "new york" is not a valid expression, so it stays as a literal string.
	assert.Equal(t, "new york", env.env["city"])
}

func TestNewEnv_ExpressionGlobals_NoOrder(t *testing.T) {
	req := &config.Request{
		Globals: map[string]any{
			"total": 42,
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	// Without GlobalsOrder, non-string globals still work.
	assert.Equal(t, 42, env.env["total"])
}

func TestNewEnv_ExpressionGlobals_UsedInArgs(t *testing.T) {
	req := &config.Request{
		Globals: map[string]any{
			"warehouses": 2,
			"districts":  "warehouses * 10",
		},
		GlobalsOrder: []string{"warehouses", "districts"},
		Run: []*config.RunItem{
			{Query: &config.Query{Args: config.PositionalArgs("districts")}},
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	argSets, _, err := env.GenerateArgs(req.Run[0].Query)
	require.NoError(t, err)

	assert.Equal(t, 20, argSets[0][0])
}

func TestNewEnv_ExpressionGlobals_Env(t *testing.T) {
	test.CleanupEnv(t, "EDG_TEST_BATCH")
	require.NoError(t, os.Setenv("EDG_TEST_BATCH", "250"))

	req := &config.Request{
		Globals: map[string]any{
			"batch_size": "env('EDG_TEST_BATCH')",
		},
		GlobalsOrder: []string{"batch_size"},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	assert.Equal(t, "250", env.env["batch_size"])
}

func TestNewEnv_ExpressionGlobals_EnvMissing(t *testing.T) {
	test.CleanupEnv(t, "EDG_TEST_MISSING")
	os.Unsetenv("EDG_TEST_MISSING")

	req := &config.Request{
		Globals: map[string]any{
			"val": "env('EDG_TEST_MISSING')",
		},
		GlobalsOrder: []string{"val"},
	}

	_, err := NewEnv(nil, "", req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evaluating global")
	assert.Contains(t, err.Error(), "EDG_TEST_MISSING")
}

func TestNewEnv_GlobalShadowsBuiltin(t *testing.T) {
	req := &config.Request{
		Globals: map[string]any{
			"ref_rand": "oops",
		},
	}

	_, err := NewEnv(nil, "", req)
	require.Error(t, err)

	assert.EqualError(t, err, `global "ref_rand" shadows a built-in function`)
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

func TestRunSection_Exec(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("CREATE TABLE").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &config.Request{},
	}

	queries := []*config.Query{
		{Name: "create_t", Type: config.QueryTypeExec, Query: "CREATE TABLE t (id INT)"},
	}

	require.NoError(t, env.runSection(context.Background(), queries, config.ConfigSectionUp, db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunSection_SeedUsesBindParams(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// $N placeholders are always inlined for cross-driver compatibility.
	mock.ExpectExec("INSERT INTO items SELECT generate_series").
		WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env: map[string]any{
			"const": convert.Constant,
			"items": 100,
		},
		request: &config.Request{},
	}

	q := &config.Query{
		Name:  "seed_items",
		Type:  config.QueryTypeExec,
		Query: "INSERT INTO items SELECT generate_series(1, $1)",
		Args:  config.PositionalArgs("items"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionSeed, db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunSection_RunSectionPassesArgs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("INSERT INTO orders").
		WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env: map[string]any{
			"const": convert.Constant,
		},
		request: &config.Request{},
	}

	q := &config.Query{
		Name:  "insert_order",
		Type:  config.QueryTypeExec,
		Query: "INSERT INTO orders VALUES ($1)",
		Args:  config.PositionalArgs("const(42)"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunSection_WaitRespectsContextCancel(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("INSERT").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &config.Request{},
	}

	q := &config.Query{
		Name:  "slow",
		Type:  config.QueryTypeExec,
		Query: "INSERT INTO t VALUES (1)",
		Wait:  config.Duration(10 * time.Second),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err = env.runSection(ctx, []*config.Query{q}, config.ConfigSectionRun, db)
	require.True(t, errors.Is(err, context.Canceled), "runSection error = %v, want context.Canceled", err)
}

func TestRunSection_QueryStoresResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2),
	)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &config.Request{},
	}

	queries := []*config.Query{
		{Name: "items", Type: config.QueryTypeQuery, Query: "SELECT id FROM items"},
	}

	require.NoError(t, env.runSection(context.Background(), queries, config.ConfigSectionInit, db))

	data, ok := env.env["items"].([]map[string]any)
	require.True(t, ok, "runSection did not store query results")
	assert.Len(t, data, 2)
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

func TestRunSection_PreparedExec(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectPrepare("INSERT INTO t").
		ExpectExec().
		WithArgs(42).
		WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		stmtCache: map[*config.Query]*sql.Stmt{},
		env:       map[string]any{"const": convert.Constant},
		request:   &config.Request{},
	}

	q := &config.Query{
		Name:     "insert_t",
		Type:     config.QueryTypeExec,
		Prepared: true,
		Query:    "INSERT INTO t VALUES ($1)",
		Args:     config.PositionalArgs("const(42)"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunSection_PreparedQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectPrepare("SELECT id, name FROM t").
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "alice"))

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		stmtCache: map[*config.Query]*sql.Stmt{},
		env:       map[string]any{"const": convert.Constant},
		request:   &config.Request{},
	}

	q := &config.Query{
		Name:     "lookup",
		Type:     config.QueryTypeQuery,
		Prepared: true,
		Query:    "SELECT id, name FROM t WHERE id = $1",
		Args:     config.PositionalArgs("const(1)"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, db))

	data, ok := env.env["lookup"].([]map[string]any)
	require.True(t, ok, "prepared query did not store results")
	require.Len(t, data, 1)
	assert.Equal(t, "alice", data[0]["name"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunSection_PreparedCachesStmt(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect a single Prepare, but two Exec calls.
	prep := mock.ExpectPrepare("INSERT INTO t")
	prep.ExpectExec().WithArgs(1).WillReturnResult(driver.ResultNoRows)
	prep.ExpectExec().WithArgs(2).WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		stmtCache: map[*config.Query]*sql.Stmt{},
		env:       map[string]any{"const": convert.Constant},
		request:   &config.Request{},
	}

	q := &config.Query{
		Name:     "insert_t",
		Type:     config.QueryTypeExec,
		Prepared: true,
		Query:    "INSERT INTO t VALUES ($1)",
		Args:     config.PositionalArgs("const(1)"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, db))

	// Change arg for second call, re-compile.
	q.Args = config.PositionalArgs("const(2)")
	require.NoError(t, q.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, db))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunSection_PreparedIgnoredForBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Batch queries should NOT prepare. Expect a regular exec.
	mock.ExpectExec("INSERT INTO t").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		stmtCache: map[*config.Query]*sql.Stmt{},
		env:       map[string]any{"const": convert.Constant},
		request:   &config.Request{},
	}

	q := &config.Query{
		Name:     "batch_insert",
		Type:     config.QueryTypeExecBatch,
		Prepared: true,
		Count:    1,
		Query:    "INSERT INTO t VALUES ($1)",
		Args:     config.PositionalArgs("const(42)"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEnvClose(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectPrepare("SELECT 1")
	mock.ExpectClose()

	q := &config.Query{Name: "test", Query: "SELECT 1"}
	stmt, err := db.Prepare("SELECT 1")
	require.NoError(t, err)

	env := &Env{
		stmtCache: map[*config.Query]*sql.Stmt{q: stmt},
	}

	env.Close()

	assert.Empty(t, env.stmtCache)
}

func TestTranslatePlaceholders(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		driver string
		want   string
	}{
		{name: "pgx unchanged", query: "SELECT * FROM t WHERE id = $1", driver: "pgx", want: "SELECT * FROM t WHERE id = $1"},
		{name: "dsql unchanged", query: "SELECT * FROM t WHERE id = $1", driver: "dsql", want: "SELECT * FROM t WHERE id = $1"},
		{name: "empty driver unchanged", query: "SELECT * FROM t WHERE id = $1", driver: "", want: "SELECT * FROM t WHERE id = $1"},
		{name: "mysql single", query: "SELECT * FROM t WHERE id = $1", driver: "mysql", want: "SELECT * FROM t WHERE id = ?"},
		{name: "mysql multi", query: "UPDATE t SET a = $2 WHERE id = $1", driver: "mysql", want: "UPDATE t SET a = ? WHERE id = ?"},
		{name: "oracle single", query: "SELECT * FROM t WHERE id = $1", driver: "oracle", want: "SELECT * FROM t WHERE id = :1"},
		{name: "oracle multi", query: "UPDATE t SET a = $2 WHERE id = $1", driver: "oracle", want: "UPDATE t SET a = :2 WHERE id = :1"},
		{name: "mssql single", query: "SELECT * FROM t WHERE id = $1", driver: "mssql", want: "SELECT * FROM t WHERE id = @p1"},
		{name: "mssql multi", query: "UPDATE t SET a = $2 WHERE id = $1", driver: "mssql", want: "UPDATE t SET a = @p2 WHERE id = @p1"},
		{name: "no placeholders", query: "SELECT 1", driver: "mysql", want: "SELECT 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translatePlaceholders(tt.query, tt.driver)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRunSection_PreparedTranslatesPlaceholders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// sqlmock sees the translated query (? for mysql).
	mock.ExpectPrepare("INSERT INTO t").
		ExpectExec().
		WithArgs(42).
		WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		driver:    "mysql",
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		stmtCache: map[*config.Query]*sql.Stmt{},
		env:       map[string]any{"const": convert.Constant},
		request:   &config.Request{},
	}

	q := &config.Query{
		Name:     "insert_t",
		Type:     config.QueryTypeExec,
		Prepared: true,
		Query:    "INSERT INTO t VALUES ($1)",
		Args:     config.PositionalArgs("const(42)"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunTransaction_RollbackIfTrue(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "balance"}).AddRow(1, 50),
	)
	mock.ExpectRollback()

	results := make(chan config.QueryResult, 10)
	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &config.Request{},
		Results:   results,
	}

	env.env["ref_same"] = env.refSame

	rollbackCheck := &config.Query{RollbackIf: "ref_same('read_source').balance < 100"}
	require.NoError(t, rollbackCheck.CompileRollbackIf(env.env))

	tx := &config.Transaction{
		Name: "test_tx",
		Queries: []*config.Query{
			{Name: "read_source", Type: config.QueryTypeQuery, Query: "SELECT id, balance FROM account WHERE id = 1"},
			rollbackCheck,
		},
	}

	err = env.runTransaction(context.Background(), tx)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())

	// Check result was reported as a rollback.
	close(results)
	var gotRollback bool
	for r := range results {
		if r.IsTransaction && r.Name == "test_tx" {
			gotRollback = r.Rollback
		}
	}
	assert.True(t, gotRollback, "expected result with Rollback=true")
}

func TestRunTransaction_RollbackIfFalse(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "balance"}).AddRow(1, 500),
	)
	mock.ExpectCommit()

	results := make(chan config.QueryResult, 10)
	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &config.Request{},
		Results:   results,
	}

	env.env["ref_same"] = env.refSame

	rollbackCheck := &config.Query{RollbackIf: "ref_same('read_source').balance < 100"}
	require.NoError(t, rollbackCheck.CompileRollbackIf(env.env))

	tx := &config.Transaction{
		Name: "test_tx",
		Queries: []*config.Query{
			{Name: "read_source", Type: config.QueryTypeQuery, Query: "SELECT id, balance FROM account WHERE id = 1"},
			rollbackCheck,
		},
	}

	err = env.runTransaction(context.Background(), tx)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())

	// Check result was NOT a rollback.
	close(results)
	for r := range results {
		if r.IsTransaction && r.Name == "test_tx" {
			assert.False(t, r.Rollback, "expected result with Rollback=false, got true")
		}
	}
}

func TestRunTransaction_NoRollbackIf(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WillReturnResult(driver.ResultNoRows)
	mock.ExpectCommit()

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &config.Request{},
	}

	tx := &config.Transaction{
		Name: "simple_tx",
		Queries: []*config.Query{
			{Name: "insert", Type: config.QueryTypeExec, Query: "INSERT INTO t VALUES (1)"},
		},
	}

	require.NoError(t, env.runTransaction(context.Background(), tx))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNewEnv_CompilesTransactionRollbackIf(t *testing.T) {
	req := &config.Request{
		Globals: map[string]any{"threshold": 100},
		Run: []*config.RunItem{
			{Transaction: &config.Transaction{
				Name: "test_tx",
				Queries: []*config.Query{
					{Name: "q1", Type: config.QueryTypeExec, Query: "INSERT INTO t VALUES (1)"},
					{RollbackIf: "threshold > 50"},
				},
			}},
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	rollbackQ := env.request.Run[0].Transaction.Queries[1]
	require.NotNil(t, rollbackQ.CompiledRollbackIf)
}

func TestNewEnv_InvalidTransactionRollbackIf(t *testing.T) {
	req := &config.Request{
		Run: []*config.RunItem{
			{Transaction: &config.Transaction{
				Name: "bad_tx",
				Queries: []*config.Query{
					{Name: "q1", Type: config.QueryTypeExec, Query: "INSERT INTO t VALUES (1)"},
					{RollbackIf: "invalid +++"},
				},
			}},
		},
	}

	_, err := NewEnv(nil, "", req)
	require.Error(t, err)
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

func TestNewEnv_CompilesTransactionLocals(t *testing.T) {
	req := &config.Request{
		Globals: map[string]any{"fee": 5},
		Run: []*config.RunItem{
			{Transaction: &config.Transaction{
				Name:   "test_tx",
				Locals: map[string]string{"amount": "fee * 2"},
				Queries: []*config.Query{
					{Name: "q1", Type: config.QueryTypeExec, Query: "INSERT INTO t VALUES (1)"},
				},
			}},
		},
	}

	_, err := NewEnv(nil, "", req)
	require.NoError(t, err)

	tx := req.Run[0].Transaction
	require.Len(t, tx.CompiledLocals, 1)
	require.NotNil(t, tx.CompiledLocals["amount"])
}

func TestNewEnv_InvalidTransactionLocal(t *testing.T) {
	req := &config.Request{
		Run: []*config.RunItem{
			{Transaction: &config.Transaction{
				Name:   "bad_tx",
				Locals: map[string]string{"bad": "invalid +++"},
				Queries: []*config.Query{
					{Name: "q1", Type: config.QueryTypeExec, Query: "INSERT INTO t VALUES (1)"},
				},
			}},
		},
	}

	_, err := NewEnv(nil, "", req)
	require.Error(t, err)
}

func TestRunTransaction_WithLocals(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WillReturnResult(driver.ResultNoRows)
	mock.ExpectCommit()

	results := make(chan config.QueryResult, 10)
	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{"const": convert.Constant},
		request:   &config.Request{},
		Results:   results,
	}
	env.env["local"] = env.local

	q := &config.Query{
		Name:  "insert",
		Type:  config.QueryTypeExec,
		Query: "INSERT INTO t VALUES ($1)",
		Args:  config.PositionalArgs("local('amount')"),
	}
	require.NoError(t, q.CompileArgs(env.env))

	tx := &config.Transaction{
		Name:    "test_tx",
		Locals:  map[string]string{"amount": "const(77)"},
		Queries: []*config.Query{q},
	}
	require.NoError(t, tx.CompileLocals(env.env))

	require.NoError(t, env.runTransaction(context.Background(), tx))
	require.NoError(t, mock.ExpectationsWereMet())

	// Verify locals were cleared after transaction.
	assert.Nil(t, env.txLocals, "txLocals not cleared after transaction")
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
			items := make([]*config.RunItem, tc.count)
			weights := make(map[string]int, tc.count)
			for i := range tc.count {
				name := fmt.Sprintf("q%d", i)
				items[i] = &config.RunItem{Query: &config.Query{Name: name}}
				weights[name] = i + 1
			}
			env := &Env{
				request: &config.Request{
					Run:        items,
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
