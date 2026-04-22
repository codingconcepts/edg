package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdoutWriter_Add(t *testing.T) {
	cases := []struct {
		name string
		row  WriteRow
		want string
	}{
		{
			name: "raw SQL without semicolon",
			row:  WriteRow{SQL: "CREATE TABLE users (id INT)"},
			want: "CREATE TABLE users (id INT);\n",
		},
		{
			name: "raw SQL with semicolon",
			row:  WriteRow{SQL: "CREATE TABLE users (id INT);"},
			want: "CREATE TABLE users (id INT);\n",
		},
		{
			name: "insert with args",
			row: WriteRow{
				SQL:     "INSERT INTO users (name, age) VALUES ($1, $2)",
				Columns: []string{"name", "age"},
				Args:    []any{"Alice", 30},
			},
			want: "INSERT INTO users (name, age) VALUES ('Alice', 30);\n",
		},
		{
			name: "update with args",
			row: WriteRow{
				SQL:     "UPDATE users SET name = $1 WHERE id = $2",
				Columns: []string{"name", "id"},
				Args:    []any{"Bob", 1},
			},
			want: "UPDATE users SET name = 'Bob' WHERE id = 1;\n",
		},
		{
			name: "upsert with args",
			row: WriteRow{
				SQL:     "INSERT INTO users (id, name) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name",
				Columns: []string{"id", "name"},
				Args:    []any{1, "Alice"},
			},
			want: "INSERT INTO users (id, name) VALUES (1, 'Alice') ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := &stdoutWriter{driver: "pgx", w: &buf}

			err := w.Add(tc.row)
			require.NoError(t, err)
			assert.Equal(t, tc.want, buf.String())
		})
	}
}

func TestStdoutWriter_Add_Drivers(t *testing.T) {
	cases := []struct {
		name   string
		driver string
		row    WriteRow
		want   string
	}{
		{
			name:   "mysql insert",
			driver: "mysql",
			row: WriteRow{
				SQL:     "INSERT INTO users (name) VALUES ($1)",
				Columns: []string{"name"},
				Args:    []any{"Alice"},
			},
			want: "INSERT INTO users (name) VALUES ('Alice');\n",
		},
		{
			name:   "mysql upsert",
			driver: "mysql",
			row: WriteRow{
				SQL:     "INSERT INTO users (id, name) VALUES ($1, $2) ON DUPLICATE KEY UPDATE name = VALUES(name)",
				Columns: []string{"id", "name"},
				Args:    []any{1, "Alice"},
			},
			want: "INSERT INTO users (id, name) VALUES (1, 'Alice') ON DUPLICATE KEY UPDATE name = VALUES(name);\n",
		},
		{
			name:   "mssql insert",
			driver: "mssql",
			row: WriteRow{
				SQL:     "INSERT INTO users (name, data) VALUES ($1, $2)",
				Columns: []string{"name", "data"},
				Args:    []any{"Alice", []byte{0xde, 0xad}},
			},
			want: "INSERT INTO users (name, data) VALUES ('Alice', 0xdead);\n",
		},
		{
			name:   "oracle merge",
			driver: "oracle",
			row: WriteRow{
				SQL:     "MERGE INTO users u USING (SELECT $1 AS id, $2 AS name FROM DUAL) src ON (u.id = src.id) WHEN MATCHED THEN UPDATE SET u.name = src.name WHEN NOT MATCHED THEN INSERT (id, name) VALUES (src.id, src.name)",
				Columns: []string{"id", "name"},
				Args:    []any{1, "Alice"},
			},
			want: "MERGE INTO users u USING (SELECT 1 AS id, 'Alice' AS name FROM DUAL) src ON (u.id = src.id) WHEN MATCHED THEN UPDATE SET u.name = src.name WHEN NOT MATCHED THEN INSERT (id, name) VALUES (src.id, src.name);\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := &stdoutWriter{driver: tc.driver, w: &buf}

			err := w.Add(tc.row)
			require.NoError(t, err)
			assert.Equal(t, tc.want, buf.String())
		})
	}
}

func TestStdoutWriter_Flush(t *testing.T) {
	w := newStdoutWriter("pgx")
	assert.NoError(t, w.Flush())
}
