package output

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSVWriter_Add_SkipsNoArgs(t *testing.T) {
	w := newCSVWriter(t.TempDir())

	err := w.Add(WriteRow{
		Section: "up",
		Name:    "create_table",
		SQL:     "CREATE TABLE users (id INT)",
	})
	require.NoError(t, err)
	assert.Empty(t, w.files)
}

func TestCSVWriter_Add_AccumulatesRows(t *testing.T) {
	w := newCSVWriter(t.TempDir())

	rows := []WriteRow{
		{Section: "seed", Name: "users", Columns: []string{"name", "age"}, Args: []any{"Alice", 30}},
		{Section: "seed", Name: "users", Columns: []string{"name", "age"}, Args: []any{"Bob", 25}},
	}

	for _, r := range rows {
		require.NoError(t, w.Add(r))
	}

	f := w.files["seed_users"]
	require.NotNil(t, f)
	assert.Equal(t, []string{"name", "age"}, f.columns)
	assert.Len(t, f.rows, 2)
	assert.Equal(t, []string{"Alice", "30"}, f.rows[0])
	assert.Equal(t, []string{"Bob", "25"}, f.rows[1])
}

func TestCSVWriter_Flush(t *testing.T) {
	dir := t.TempDir()
	w := newCSVWriter(dir)

	w.files["seed_users"] = &csvFile{
		columns: []string{"name", "age"},
		rows: [][]string{
			{"Alice", "30"},
			{"Bob", "25"},
		},
	}

	err := w.Flush()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "seed_users.csv"))
	require.NoError(t, err)
	assert.Equal(t, "name,age\nAlice,30\nBob,25\n", string(data))
}

func TestFormatCSVValue(t *testing.T) {
	cases := []struct {
		name string
		val  any
		want string
	}{
		{name: "nil", val: nil, want: ""},
		{name: "string", val: "hello", want: "hello"},
		{name: "int", val: 42, want: "42"},
		{name: "bytes", val: []byte{0xde, 0xad}, want: "dead"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatCSVValue(tc.val))
		})
	}
}
