package env

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/codingconcepts/edg/pkg/config"

	"github.com/DATA-DOG/go-sqlmock"
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
			if err != nil {
				t.Fatalf("creating sqlmock: %v", err)
			}
			defer db.Close()

			mock.ExpectQuery("SELECT").WillReturnRows(buildMockRows(tt.columns, tt.rows))

			rows, err := db.Query("SELECT")
			if err != nil {
				t.Fatalf("querying: %v", err)
			}

			got, err := ReadRows(rows)
			if err != nil {
				t.Fatalf("ReadRows() error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}

			assertMaps(t, got, tt.expected)
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

	if len(got) != len(expected) {
		t.Fatalf("got %d rows, want %d", len(got), len(expected))
	}
	for i, want := range expected {
		for k, v := range want {
			gv, ok := got[i][k]
			if !ok {
				t.Errorf("row %d: missing key %q", i, k)
				continue
			}
			if fmt.Sprint(gv) != fmt.Sprint(v) {
				t.Errorf("row %d[%q] = %v, want %v", i, k, gv, v)
			}
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
				ReadRows(rows)
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
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectPrepare("SELECT id, name FROM users").
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "alice"))

	stmt, err := db.Prepare("SELECT id, name FROM users WHERE id = ?")
	if err != nil {
		t.Fatalf("preparing: %v", err)
	}
	defer stmt.Close()

	env := newTestEnv(t)
	q := &config.Query{Name: "users", Query: "SELECT id, name FROM users WHERE id = ?"}

	if err := env.QueryPrepared(context.Background(), stmt, q, 1); err != nil {
		t.Fatalf("QueryPrepared error: %v", err)
	}

	data, ok := env.env["users"].([]map[string]any)
	if !ok {
		t.Fatal("QueryPrepared did not store results")
	}
	if len(data) != 1 || data[0]["name"] != "alice" {
		t.Errorf("got %v, want [{id:1 name:alice}]", data)
	}
}

func TestExecPrepared(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectPrepare("INSERT INTO users").
		ExpectExec().
		WithArgs(1, "alice").
		WillReturnResult(driver.ResultNoRows)

	stmt, err := db.Prepare("INSERT INTO users VALUES (?, ?)")
	if err != nil {
		t.Fatalf("preparing: %v", err)
	}
	defer stmt.Close()

	env := newTestEnv(t)
	q := &config.Query{Name: "insert_user", Query: "INSERT INTO users VALUES (?, ?)"}

	if err := env.ExecPrepared(context.Background(), stmt, q, 1, "alice"); err != nil {
		t.Fatalf("ExecPrepared error: %v", err)
	}
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
		expectErr   string
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
			expectErr:   "running statement: connection refused",
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
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("creating sqlmock: %v", err)
			}
			defer db.Close()

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
			err = env.Query(context.Background(), db, &tt.query, tt.args...)

			if tt.expectErr != "" {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if got := err.Error(); got != tt.expectErr {
					t.Fatalf("error = %q, want %q", got, tt.expectErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Query() error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}

			data, ok := env.env[tt.query.Name].([]map[string]any)
			if !ok {
				t.Fatalf("env[%q] not set or wrong type: %v", tt.query.Name, env.env[tt.query.Name])
			}

			assertMaps(t, data, tt.expected)
		})
	}
}
