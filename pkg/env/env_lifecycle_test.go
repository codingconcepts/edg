package env

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/codingconcepts/edg/pkg/config"
	edgdb "github.com/codingconcepts/edg/pkg/db"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunIteration_NoWeights(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("INSERT INTO t1").WillReturnResult(driver.ResultNoRows)
	mock.ExpectExec("INSERT INTO t2").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        edgdb.NewSQDB(db),
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
		db:        edgdb.NewSQDB(db),
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
