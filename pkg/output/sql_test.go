package output

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLWriter_Add_RawSQL(t *testing.T) {
	w := newSQLWriter("pgx", t.TempDir())

	err := w.Add(WriteRow{
		Section: "up",
		Name:    "create_table",
		SQL:     "CREATE TABLE users (id INT)",
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"CREATE TABLE users (id INT);"}, w.sections["up"])
}

func TestSQLWriter_Add_Insert(t *testing.T) {
	w := newSQLWriter("pgx", t.TempDir())

	err := w.Add(WriteRow{
		Section: "seed",
		Name:    "users",
		SQL:     "INSERT INTO users (name, age) VALUES ($1, $2)",
		Columns: []string{"name", "age"},
		Args:    []any{"Alice", 30},
	})
	require.NoError(t, err)

	require.Len(t, w.sections["seed"], 1)
	assert.Equal(t, "INSERT INTO users (name, age) VALUES ('Alice', 30);", w.sections["seed"][0])
}

func TestSQLWriter_Add_Update(t *testing.T) {
	w := newSQLWriter("pgx", t.TempDir())

	err := w.Add(WriteRow{
		Section: "seed",
		Name:    "users",
		SQL:     "UPDATE users SET name = $1 WHERE id = $2",
		Columns: []string{"name", "id"},
		Args:    []any{"Bob", 1},
	})
	require.NoError(t, err)

	require.Len(t, w.sections["seed"], 1)
	assert.Equal(t, "UPDATE users SET name = 'Bob' WHERE id = 1;", w.sections["seed"][0])
}

func TestSQLWriter_Add_Upsert(t *testing.T) {
	w := newSQLWriter("pgx", t.TempDir())

	err := w.Add(WriteRow{
		Section: "seed",
		Name:    "users",
		SQL:     "INSERT INTO users (id, name) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name",
		Columns: []string{"id", "name"},
		Args:    []any{1, "Alice"},
	})
	require.NoError(t, err)

	require.Len(t, w.sections["seed"], 1)
	assert.Equal(t, "INSERT INTO users (id, name) VALUES (1, 'Alice') ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;", w.sections["seed"][0])
}

func TestSQLWriter_Flush(t *testing.T) {
	dir := t.TempDir()
	w := newSQLWriter("pgx", dir)

	w.sections["seed"] = []string{
		"INSERT INTO users VALUES ('Alice', 30);",
		"INSERT INTO users VALUES ('Bob', 25);",
	}

	err := w.Flush()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "seed.sql"))
	require.NoError(t, err)
	assert.Equal(t, "INSERT INTO users VALUES ('Alice', 30);\nINSERT INTO users VALUES ('Bob', 25);\n", string(data))
}
