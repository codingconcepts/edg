package env

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"
	edgdb "github.com/codingconcepts/edg/pkg/db"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunSection_Exec(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("CREATE TABLE").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        edgdb.NewSQDB(db),
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &config.Request{},
	}

	queries := []*config.Query{
		{Name: "create_t", Type: config.QueryTypeExec, Query: "CREATE TABLE t (id INT)"},
	}

	require.NoError(t, env.runSection(context.Background(), queries, config.ConfigSectionUp, edgdb.NewSQDB(db)))
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
		db:        edgdb.NewSQDB(db),
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

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionSeed, edgdb.NewSQDB(db)))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunSection_SeedCapturesNamedArgs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("INSERT INTO employees").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        edgdb.NewSQDB(db),
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{"const": convert.Constant},
		request:   &config.Request{},
	}

	q := &config.Query{
		Name:  "ceo",
		Type:  config.QueryTypeExec,
		Query: "INSERT INTO employees (id, title) VALUES ($1, $2)",
		Args: config.QueryArgs{
			Exprs: []string{"const(1)", "const('CEO')"},
			Names: map[string]int{"id": 0, "title": 1},
		},
	}
	require.NoError(t, q.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionSeed, edgdb.NewSQDB(db)))
	require.NoError(t, mock.ExpectationsWereMet())

	data, ok := env.env["ceo"].([]map[string]any)
	require.True(t, ok, "seed query did not capture dataset")
	require.Len(t, data, 1)
	assert.Equal(t, 1, data[0]["id"])
	assert.Equal(t, "CEO", data[0]["title"])
}

func TestRunSection_SeedCapturesBatchArgs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	for range 3 {
		mock.ExpectExec("INSERT INTO employees").WillReturnResult(driver.ResultNoRows)
	}

	env := &Env{
		db:        edgdb.NewSQDB(db),
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{"const": convert.Constant},
		request:   &config.Request{},
	}

	q := &config.Query{
		Name:  "vps",
		Type:  config.QueryTypeExecBatch,
		Count: 3,
		Query: "INSERT INTO employees (id, title) VALUES ($1, $2)",
		Args: config.QueryArgs{
			Exprs: []string{"const(10)", "const('VP')"},
			Names: map[string]int{"id": 0, "title": 1},
		},
	}
	require.NoError(t, q.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionSeed, edgdb.NewSQDB(db)))
	require.NoError(t, mock.ExpectationsWereMet())

	data, ok := env.env["vps"].([]map[string]any)
	require.True(t, ok, "seed batch query did not capture dataset")
	require.Len(t, data, 3)
	for i, row := range data {
		assert.Equal(t, 10, row["id"], "row %d id", i)
		assert.Equal(t, "VP", row["title"], "row %d title", i)
	}
}

func TestRunSection_SeedCaptureHierarchical(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("INSERT INTO employees").WillReturnResult(driver.ResultNoRows)
	for range 3 {
		mock.ExpectExec("INSERT INTO employees").WillReturnResult(driver.ResultNoRows)
	}

	env := &Env{
		db:        edgdb.NewSQDB(db),
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{"const": convert.Constant},
		request:   &config.Request{},
	}
	env.env["ref_rand"] = env.refRand

	ceo := &config.Query{
		Name:  "ceo",
		Type:  config.QueryTypeExec,
		Query: "INSERT INTO employees (id, title) VALUES ($1, $2)",
		Args: config.QueryArgs{
			Exprs: []string{"const(1)", "const('CEO')"},
			Names: map[string]int{"id": 0, "title": 1},
		},
	}
	require.NoError(t, ceo.CompileArgs(env.env))

	vps := &config.Query{
		Name:  "vps",
		Type:  config.QueryTypeExecBatch,
		Count: 3,
		Query: "INSERT INTO employees (id, manager_id) VALUES ($1, $2)",
		Args: config.QueryArgs{
			Exprs: []string{"const(10)", "ref_rand('ceo').id"},
			Names: map[string]int{"id": 0, "manager_id": 1},
		},
	}
	require.NoError(t, vps.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{ceo, vps}, config.ConfigSectionSeed, edgdb.NewSQDB(db)))
	require.NoError(t, mock.ExpectationsWereMet())

	// CEO captured
	ceoData, ok := env.env["ceo"].([]map[string]any)
	require.True(t, ok, "ceo not captured")
	require.Len(t, ceoData, 1)
	assert.Equal(t, 1, ceoData[0]["id"])

	// VPs captured and all reference the CEO
	vpData, ok := env.env["vps"].([]map[string]any)
	require.True(t, ok, "vps not captured")
	require.Len(t, vpData, 3)
	for i, row := range vpData {
		assert.Equal(t, 1, row["manager_id"], "vp %d should reference ceo id=1", i)
	}
}

func TestRunSection_ExecBatchValues(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("INSERT INTO employees").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        edgdb.NewSQDB(db),
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{"const": convert.Constant},
		request:   &config.Request{},
	}

	q := &config.Query{
		Name:  "insert_emp",
		Type:  config.QueryTypeExecBatch,
		Count: 3,
		Query: "INSERT INTO employees (id, title) __values__",
		Args: config.QueryArgs{
			Exprs: []string{"const(10)", "const('VP')"},
			Names: map[string]int{"id": 0, "title": 1},
		},
	}
	require.NoError(t, q.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionSeed, edgdb.NewSQDB(db)))
	require.NoError(t, mock.ExpectationsWereMet())

	data, ok := env.env["insert_emp"].([]map[string]any)
	require.True(t, ok, "seed batch __values__ should capture dataset")
	require.Len(t, data, 3)
	for i, row := range data {
		assert.Equal(t, 10, row["id"], "row %d", i)
		assert.Equal(t, "VP", row["title"], "row %d", i)
	}
}

func TestRunSection_SeedCaptureNotInRunSection(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("INSERT INTO t").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        edgdb.NewSQDB(db),
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{"const": convert.Constant},
		request:   &config.Request{},
	}

	q := &config.Query{
		Name:  "insert_t",
		Type:  config.QueryTypeExec,
		Query: "INSERT INTO t (id) VALUES ($1)",
		Args: config.QueryArgs{
			Exprs: []string{"const(1)"},
			Names: map[string]int{"id": 0},
		},
	}
	require.NoError(t, q.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, edgdb.NewSQDB(db)))
	require.NoError(t, mock.ExpectationsWereMet())

	_, ok := env.env["insert_t"].([]map[string]any)
	assert.False(t, ok, "run section should not capture datasets")
}

func TestRunSection_RunSectionPassesArgs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("INSERT INTO orders").
		WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        edgdb.NewSQDB(db),
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

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, edgdb.NewSQDB(db)))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunSection_WaitRespectsContextCancel(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("INSERT").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        edgdb.NewSQDB(db),
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

	err = env.runSection(ctx, []*config.Query{q}, config.ConfigSectionRun, edgdb.NewSQDB(db))
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
		db:        edgdb.NewSQDB(db),
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &config.Request{},
	}

	queries := []*config.Query{
		{Name: "items", Type: config.QueryTypeQuery, Query: "SELECT id FROM items"},
	}

	require.NoError(t, env.runSection(context.Background(), queries, config.ConfigSectionInit, edgdb.NewSQDB(db)))

	data, ok := env.env["items"].([]map[string]any)
	require.True(t, ok, "runSection did not store query results")
	assert.Len(t, data, 2)
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
		db:        edgdb.NewSQDB(db),
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		stmtCache: map[*config.Query]edgdb.PreparedStatement{},
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

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, edgdb.NewSQDB(db)))
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
		db:        edgdb.NewSQDB(db),
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		stmtCache: map[*config.Query]edgdb.PreparedStatement{},
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

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, edgdb.NewSQDB(db)))

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
		db:        edgdb.NewSQDB(db),
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		stmtCache: map[*config.Query]edgdb.PreparedStatement{},
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

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, edgdb.NewSQDB(db)))

	// Change arg for second call, re-compile.
	q.Args = config.PositionalArgs("const(2)")
	require.NoError(t, q.CompileArgs(env.env))

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, edgdb.NewSQDB(db)))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunSection_PreparedIgnoredForBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Batch queries should NOT prepare. Expect a regular exec.
	mock.ExpectExec("INSERT INTO t").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        edgdb.NewSQDB(db),
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		stmtCache: map[*config.Query]edgdb.PreparedStatement{},
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

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, edgdb.NewSQDB(db)))
	require.NoError(t, mock.ExpectationsWereMet())
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
		db:        edgdb.NewSQDB(db),
		driver:    "mysql",
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		stmtCache: map[*config.Query]edgdb.PreparedStatement{},
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

	require.NoError(t, env.runSection(context.Background(), []*config.Query{q}, config.ConfigSectionRun, edgdb.NewSQDB(db)))
	require.NoError(t, mock.ExpectationsWereMet())
}
