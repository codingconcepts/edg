package env

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"
	edgdb "github.com/codingconcepts/edg/pkg/db"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		db:        edgdb.NewSQDB(db),
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
		db:        edgdb.NewSQDB(db),
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
		db:        edgdb.NewSQDB(db),
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

func TestRunTransaction_WithLocals(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WillReturnResult(driver.ResultNoRows)
	mock.ExpectCommit()

	results := make(chan config.QueryResult, 10)
	env := &Env{
		db:        edgdb.NewSQDB(db),
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
