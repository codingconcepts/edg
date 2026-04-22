package output

import (
	"testing"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/expr-lang/expr/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFormat(t *testing.T) {
	cases := []struct {
		input string
		want  Format
		err   bool
	}{
		{input: "sql", want: FormatSQL},
		{input: "SQL", want: FormatSQL},
		{input: "json", want: FormatJSON},
		{input: "csv", want: FormatCSV},
		{input: "parquet", want: FormatParquet},
		{input: "stdout", want: FormatStdout},
		{input: "xml", err: true},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseFormat(tc.input)
			if tc.err {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNew(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		format Format
		err    bool
	}{
		{format: FormatSQL},
		{format: FormatJSON},
		{format: FormatCSV},
		{format: FormatParquet},
		{format: FormatStdout},
		{format: Format("unknown"), err: true},
	}

	for _, tc := range cases {
		t.Run(string(tc.format), func(t *testing.T) {
			w, err := New(tc.format, "pgx", dir)
			if tc.err {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, w)
		})
	}
}

func TestExtractColumns_Named(t *testing.T) {
	q := &config.Query{
		Args: config.QueryArgs{
			Names: map[string]int{"name": 0, "age": 1},
		},
	}

	cols := ExtractColumns(q)
	assert.Equal(t, []string{"name", "age"}, cols)
}

func TestExtractColumns_FromInsert(t *testing.T) {
	q := &config.Query{
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		CompiledArgs: make([]*vm.Program, 2),
	}

	cols := ExtractColumns(q)
	assert.Equal(t, []string{"name", "age"}, cols)
}

func TestExtractColumns_Fallback(t *testing.T) {
	q := &config.Query{
		Query:        "SELECT 1",
		CompiledArgs: make([]*vm.Program, 2),
	}

	cols := ExtractColumns(q)
	assert.Equal(t, []string{"col_1", "col_2"}, cols)
}

func TestResolveArgs(t *testing.T) {
	cases := []struct {
		name   string
		query  string
		driver string
		args   []any
		want   string
	}{
		{
			name:   "insert",
			query:  "INSERT INTO users (name, age) VALUES ($1, $2)",
			driver: "pgx",
			args:   []any{"Alice", 30},
			want:   "INSERT INTO users (name, age) VALUES ('Alice', 30)",
		},
		{
			name:   "update",
			query:  "UPDATE users SET name = $1 WHERE id = $2",
			driver: "pgx",
			args:   []any{"Bob", 1},
			want:   "UPDATE users SET name = 'Bob' WHERE id = 1",
		},
		{
			name:   "upsert",
			query:  "INSERT INTO users (id, name) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name",
			driver: "pgx",
			args:   []any{1, "Alice"},
			want:   "INSERT INTO users (id, name) VALUES (1, 'Alice') ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name",
		},
		{
			name:   "quoted placeholder",
			query:  "SELECT unnest(string_to_array('$1', chr(31)))",
			driver: "pgx",
			args:   []any{"Alice"},
			want:   "SELECT unnest(string_to_array('Alice', chr(31)))",
		},
		{
			name:   "mysql insert",
			query:  "INSERT INTO users (name, age) VALUES ($1, $2)",
			driver: "mysql",
			args:   []any{"Alice", 30},
			want:   "INSERT INTO users (name, age) VALUES ('Alice', 30)",
		},
		{
			name:   "mysql upsert",
			query:  "INSERT INTO users (id, name) VALUES ($1, $2) ON DUPLICATE KEY UPDATE name = VALUES(name)",
			driver: "mysql",
			args:   []any{1, "Alice"},
			want:   "INSERT INTO users (id, name) VALUES (1, 'Alice') ON DUPLICATE KEY UPDATE name = VALUES(name)",
		},
		{
			name:   "mssql insert",
			query:  "INSERT INTO users (name, age) VALUES ($1, $2)",
			driver: "mssql",
			args:   []any{"Alice", 30},
			want:   "INSERT INTO users (name, age) VALUES ('Alice', 30)",
		},
		{
			name:   "mssql bytes",
			query:  "INSERT INTO users (data) VALUES ($1)",
			driver: "mssql",
			args:   []any{[]byte{0xde, 0xad}},
			want:   "INSERT INTO users (data) VALUES (0xdead)",
		},
		{
			name:   "oracle insert",
			query:  "INSERT INTO users (name, age) VALUES ($1, $2)",
			driver: "oracle",
			args:   []any{"Alice", 30},
			want:   "INSERT INTO users (name, age) VALUES ('Alice', 30)",
		},
		{
			name:   "oracle merge",
			query:  "MERGE INTO users u USING (SELECT $1 AS id, $2 AS name FROM DUAL) src ON (u.id = src.id) WHEN MATCHED THEN UPDATE SET u.name = src.name WHEN NOT MATCHED THEN INSERT (id, name) VALUES (src.id, src.name)",
			driver: "oracle",
			args:   []any{1, "Alice"},
			want:   "MERGE INTO users u USING (SELECT 1 AS id, 'Alice' AS name FROM DUAL) src ON (u.id = src.id) WHEN MATCHED THEN UPDATE SET u.name = src.name WHEN NOT MATCHED THEN INSERT (id, name) VALUES (src.id, src.name)",
		},
		{
			name:   "pgx bytes",
			query:  "INSERT INTO users (data) VALUES ($1)",
			driver: "pgx",
			args:   []any{[]byte{0xca, 0xfe}},
			want:   "INSERT INTO users (data) VALUES (X'cafe')",
		},
		{
			name:   "null arg",
			query:  "INSERT INTO users (name) VALUES ($1)",
			driver: "pgx",
			args:   []any{nil},
			want:   "INSERT INTO users (name) VALUES (NULL)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, resolveArgs(tc.query, tc.driver, tc.args))
		})
	}
}
