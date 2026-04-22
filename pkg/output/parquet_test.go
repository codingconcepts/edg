package output

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParquetWriter_Add_SkipsNoArgs(t *testing.T) {
	w := newParquetWriter(t.TempDir())

	err := w.Add(WriteRow{
		Section: "up",
		Name:    "create_table",
		SQL:     "CREATE TABLE users (id INT)",
	})
	require.NoError(t, err)
	assert.Empty(t, w.files)
}

func TestParquetWriter_Add_AccumulatesRows(t *testing.T) {
	w := newParquetWriter(t.TempDir())

	rows := []WriteRow{
		{Section: "seed", Name: "users", Columns: []string{"name", "age"}, Args: []any{"Alice", 30}},
		{Section: "seed", Name: "users", Columns: []string{"name", "age"}, Args: []any{"Bob", 25}},
	}

	for _, r := range rows {
		require.NoError(t, w.Add(r))
	}

	f := w.files["seed_users"]
	require.NotNil(t, f)
	assert.Len(t, f.rows, 2)
	assert.Equal(t, []any{"Alice", 30}, f.rows[0])
}

func TestParquetWriter_Flush(t *testing.T) {
	dir := t.TempDir()
	w := newParquetWriter(dir)

	w.files["seed_users"] = &parquetFile{
		columns: []string{"name", "age"},
		rows: [][]any{
			{"Alice", 30},
			{"Bob", 25},
		},
	}

	err := w.Flush()
	require.NoError(t, err)

	filename := filepath.Join(dir, "seed_users.parquet")
	f, err := os.Open(filename)
	require.NoError(t, err)
	defer f.Close()

	stat, err := f.Stat()
	require.NoError(t, err)

	pf, err := parquet.OpenFile(f, stat.Size())
	require.NoError(t, err)
	assert.Equal(t, 2, int(pf.NumRows()))
}
