package env

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/codingconcepts/edg/pkg/config"
	edgdb "github.com/codingconcepts/edg/pkg/db"
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
		stmtCache:       map[*config.Query]edgdb.PreparedStatement{},
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

func TestIter(t *testing.T) {
	env := testEnv(nil)

	assert.Equal(t, 1, env.iter())
	assert.Equal(t, 2, env.iter())
	assert.Equal(t, 3, env.iter())

	env.resetUniqIndex()
	assert.Equal(t, 1, env.iter())
}

func TestEnvClose(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectPrepare("SELECT 1")
	mock.ExpectClose()

	q := &config.Query{Name: "test", Query: "SELECT 1"}
	wrapped := edgdb.NewSQDB(db)
	stmt, err := wrapped.PrepareContext(context.Background(), "SELECT 1")
	require.NoError(t, err)

	env := &Env{
		stmtCache: map[*config.Query]edgdb.PreparedStatement{q: stmt},
	}

	env.Close()

	assert.Empty(t, env.stmtCache)
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
