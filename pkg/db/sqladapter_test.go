package db

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQDB_QueryContext(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name"}).
			AddRow(1, "alice").
			AddRow(2, "bob"),
	)

	db := NewSQDB(sqlDB)
	iter, err := db.QueryContext(context.Background(), "SELECT id, name FROM users")
	require.NoError(t, err)
	defer iter.Close()

	cols, err := iter.Columns()
	require.NoError(t, err)
	assert.Equal(t, []string{"id", "name"}, cols)

	require.True(t, iter.Next())
	var id int
	var name string
	require.NoError(t, iter.Scan(&id, &name))
	assert.Equal(t, 1, id)
	assert.Equal(t, "alice", name)

	require.True(t, iter.Next())
	require.NoError(t, iter.Scan(&id, &name))
	assert.Equal(t, 2, id)
	assert.Equal(t, "bob", name)

	require.False(t, iter.Next())
	require.NoError(t, iter.Err())
}

func TestSQDB_ExecContext(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	mock.ExpectExec("INSERT").WillReturnResult(driver.ResultNoRows)

	db := NewSQDB(sqlDB)
	err = db.ExecContext(context.Background(), "INSERT INTO users VALUES (1, 'alice')")
	require.NoError(t, err)
}

func TestSQDB_Transaction(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WillReturnResult(driver.ResultNoRows)
	mock.ExpectCommit()

	db := NewSQDB(sqlDB)
	tx, err := db.BeginTx(context.Background())
	require.NoError(t, err)

	err = tx.ExecContext(context.Background(), "INSERT INTO users VALUES (1, 'alice')")
	require.NoError(t, err)

	require.NoError(t, tx.Commit())
}

func TestSQDB_PreparedStatement(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	mock.ExpectPrepare("SELECT").
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "alice"))

	db := NewSQDB(sqlDB)
	stmt, err := db.PrepareContext(context.Background(), "SELECT id, name FROM users WHERE id = ?")
	require.NoError(t, err)
	defer stmt.Close()

	iter, err := stmt.QueryContext(context.Background(), 1)
	require.NoError(t, err)
	defer iter.Close()

	require.True(t, iter.Next())
	var id int
	var name string
	require.NoError(t, iter.Scan(&id, &name))
	assert.Equal(t, 1, id)
	assert.Equal(t, "alice", name)
}

func TestSQDB_ColumnTypes(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "alice"),
	)

	db := NewSQDB(sqlDB)
	iter, err := db.QueryContext(context.Background(), "SELECT id, name FROM users")
	require.NoError(t, err)
	defer iter.Close()

	cts, err := iter.ColumnTypes()
	require.NoError(t, err)
	require.Len(t, cts, 2)
}

func TestNewSQLRowIterator(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"x"}).AddRow(42),
	)

	rows, err := sqlDB.Query("SELECT x")
	require.NoError(t, err)

	iter := NewSQLRowIterator(rows)
	defer iter.Close()

	require.True(t, iter.Next())
	var x int
	require.NoError(t, iter.Scan(&x))
	assert.Equal(t, 42, x)
}
