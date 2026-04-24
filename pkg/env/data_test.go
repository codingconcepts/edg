package env

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/codingconcepts/edg/pkg/config"
	edgdb "github.com/codingconcepts/edg/pkg/db"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadRows(t *testing.T) {
	tests := []struct {
		name     string
		columns  []string
		rows     [][]any
		expected []map[string]any
	}{
		{
			name:     "no rows",
			columns:  []string{"id", "name"},
			rows:     nil,
			expected: nil,
		},
		{
			name:    "single row",
			columns: []string{"id", "name"},
			rows:    [][]any{{1, "alice"}},
			expected: []map[string]any{
				{"id": 1, "name": "alice"},
			},
		},
		{
			name:    "multiple rows",
			columns: []string{"id", "name"},
			rows: [][]any{
				{1, "alice"},
				{2, "bob"},
				{3, "charlie"},
			},
			expected: []map[string]any{
				{"id": 1, "name": "alice"},
				{"id": 2, "name": "bob"},
				{"id": 3, "name": "charlie"},
			},
		},
		{
			name:    "columns lowercased",
			columns: []string{"ID", "FirstName", "LAST_NAME"},
			rows:    [][]any{{1, "alice", "smith"}},
			expected: []map[string]any{
				{"id": 1, "firstname": "alice", "last_name": "smith"},
			},
		},
		{
			name:    "nil values",
			columns: []string{"id", "name"},
			rows:    [][]any{{1, nil}},
			expected: []map[string]any{
				{"id": 1, "name": nil},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			mock.ExpectQuery("SELECT").WillReturnRows(buildMockRows(tt.columns, tt.rows))

			rows, err := db.Query("SELECT")
			require.NoError(t, err)

			got, err := ReadRows(edgdb.NewSQLRowIterator(rows))
			require.NoError(t, err)

			err = mock.ExpectationsWereMet()
			require.NoError(t, err)

			assertMaps(t, got, tt.expected)
		})
	}
}

func TestNormalizeBytes(t *testing.T) {
	tests := []struct {
		name       string
		input      []byte
		dbType     string
		expectType string
	}{
		{"BYTEA preserved", []byte{0xDE, 0xAD}, "BYTEA", "[]uint8"},
		{"BLOB preserved", []byte{0xCA, 0xFE}, "BLOB", "[]uint8"},
		{"BYTES preserved", []byte{0x01}, "BYTES", "[]uint8"},
		{"BINARY preserved", []byte{0xFF}, "BINARY", "[]uint8"},
		{"VARBINARY preserved", []byte{0xAB}, "VARBINARY", "[]uint8"},
		{"RAW preserved", []byte{0x00}, "RAW", "[]uint8"},
		{"IMAGE preserved", []byte{0x89}, "IMAGE", "[]uint8"},
		{"TINYBLOB preserved", []byte{0x01}, "TINYBLOB", "[]uint8"},
		{"MEDIUMBLOB preserved", []byte{0x01}, "MEDIUMBLOB", "[]uint8"},
		{"LONGBLOB preserved", []byte{0x01}, "LONGBLOB", "[]uint8"},
		{"LONG RAW preserved", []byte{0x01}, "LONG RAW", "[]uint8"},
		{"VARCHAR becomes string", []byte("hello"), "VARCHAR", "string"},
		{"TEXT becomes string", []byte("world"), "TEXT", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeBytes(tt.input, tt.dbType)
			got := fmt.Sprintf("%T", result)
			assert.Equal(t, tt.expectType, got, "normalizeBytes(%v, %q)", tt.input, tt.dbType)
		})
	}
}

func buildMockRows(columns []string, rows [][]any) *sqlmock.Rows {
	mockRows := sqlmock.NewRows(columns)
	for _, r := range rows {
		vals := make([]driver.Value, len(r))
		for i, v := range r {
			vals[i] = v
		}
		mockRows = mockRows.AddRow(vals...)
	}
	return mockRows
}

func assertMaps(t *testing.T, got, expected []map[string]any) {
	t.Helper()

	require.Equal(t, len(expected), len(got), "row count mismatch")
	for i, want := range expected {
		for k, v := range want {
			gv, ok := got[i][k]
			require.True(t, ok, "row %d: missing key %q", i, k)
			assert.Equal(t, fmt.Sprint(v), fmt.Sprint(gv), "row %d[%q]", i, k)
		}
	}
}

func BenchmarkReadRows(b *testing.B) {
	cases := []struct {
		name string
		rows int
		cols int
	}{
		{"rows_1/cols_2", 1, 2},
		{"rows_1/cols_5", 1, 5},
		{"rows_10/cols_2", 10, 2},
		{"rows_10/cols_5", 10, 5},
		{"rows_100/cols_2", 100, 2},
		{"rows_100/cols_5", 100, 5},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			columns := make([]string, tc.cols)
			for i := range tc.cols {
				columns[i] = fmt.Sprintf("Col%d", i)
			}

			rowData := make([][]any, tc.rows)
			for i := range tc.rows {
				row := make([]any, tc.cols)
				for j := range tc.cols {
					row[j] = i*tc.cols + j
				}
				rowData[i] = row
			}

			b.ResetTimer()
			for range b.N {
				db, mock, err := sqlmock.New()
				if err != nil {
					b.Fatalf("creating sqlmock: %v", err)
				}

				mockRows := sqlmock.NewRows(columns)
				for _, r := range rowData {
					vals := make([]driver.Value, len(r))
					for i, v := range r {
						vals[i] = v
					}
					mockRows.AddRow(vals...)
				}
				mock.ExpectQuery("SELECT").WillReturnRows(mockRows)

				rows, _ := db.Query("SELECT")
				ReadRows(edgdb.NewSQLRowIterator(rows))
				db.Close()
			}
		})
	}
}

func newTestEnv(t *testing.T) *Env {
	t.Helper()

	env := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		nurandC:   map[int]int{},
		request:   &config.Request{},
	}

	env.env = map[string]any{}
	return env
}

func TestQueryPrepared(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	mock.ExpectPrepare("SELECT id, name FROM users").
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "alice"))

	wrapped := edgdb.NewSQDB(sqlDB)
	stmt, err := wrapped.PrepareContext(context.Background(), "SELECT id, name FROM users WHERE id = ?")
	require.NoError(t, err)
	defer stmt.Close()

	env := newTestEnv(t)
	q := &config.Query{Name: "users", Query: "SELECT id, name FROM users WHERE id = ?"}

	err = env.QueryPrepared(context.Background(), stmt, q, 1)
	require.NoError(t, err)

	data, ok := env.env["users"].([]map[string]any)
	require.True(t, ok, "QueryPrepared did not store results")
	require.Equal(t, 1, len(data))
	assert.Equal(t, "alice", data[0]["name"])
}

func TestExecPrepared(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	mock.ExpectPrepare("INSERT INTO users").
		ExpectExec().
		WithArgs(1, "alice").
		WillReturnResult(driver.ResultNoRows)

	wrapped := edgdb.NewSQDB(sqlDB)
	stmt, err := wrapped.PrepareContext(context.Background(), "INSERT INTO users VALUES (?, ?)")
	require.NoError(t, err)
	defer stmt.Close()

	env := newTestEnv(t)
	q := &config.Query{Name: "insert_user", Query: "INSERT INTO users VALUES (?, ?)"}

	err = env.ExecPrepared(context.Background(), stmt, q, 1, "alice")
	require.NoError(t, err)
}

func TestQuery(t *testing.T) {
	tests := []struct {
		name        string
		mockPattern string
		columns     []string
		rows        [][]any
		queryErr    error
		query       config.Query
		args        []any
		expErr      string
		expected    []map[string]any
	}{
		{
			name:        "stores results in env",
			mockPattern: "SELECT id, name FROM users",
			columns:     []string{"id", "name"},
			rows:        [][]any{{1, "alice"}, {2, "bob"}},
			query:       config.Query{Name: "users", Query: "SELECT id, name FROM users"},
			expected: []map[string]any{
				{"id": 1, "name": "alice"},
				{"id": 2, "name": "bob"},
			},
		},
		{
			name:        "query error",
			mockPattern: "SELECT",
			queryErr:    fmt.Errorf("connection refused"),
			query:       config.Query{Name: "users", Query: "SELECT 1"},
			expErr:      "running statement: connection refused",
		},
		{
			name:        "passes args to query",
			mockPattern: "SELECT id, name FROM users WHERE id = \\$1",
			columns:     []string{"id", "name"},
			rows:        [][]any{{42, "charlie"}},
			query:       config.Query{Name: "users", Query: "SELECT id, name FROM users WHERE id = $1"},
			args:        []any{42},
			expected: []map[string]any{
				{"id": 42, "name": "charlie"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer sqlDB.Close()

			eq := mock.ExpectQuery(tt.mockPattern)
			if len(tt.args) > 0 {
				driverArgs := make([]driver.Value, len(tt.args))
				for i, a := range tt.args {
					driverArgs[i] = a
				}
				eq = eq.WithArgs(driverArgs...)
			}
			if tt.queryErr != nil {
				eq.WillReturnError(tt.queryErr)
			} else {
				eq.WillReturnRows(buildMockRows(tt.columns, tt.rows))
			}

			env := newTestEnv(t)
			err = env.Query(context.Background(), edgdb.NewSQDB(sqlDB), &tt.query, tt.args...)

			if tt.expErr != "" {
				require.EqualError(t, err, tt.expErr)
				return
			}

			require.NoError(t, err)

			err = mock.ExpectationsWereMet()
			require.NoError(t, err)

			data, ok := env.env[tt.query.Name].([]map[string]any)
			require.True(t, ok, "env[%q] not set or wrong type: %v", tt.query.Name, env.env[tt.query.Name])

			assertMaps(t, data, tt.expected)
		})
	}
}
