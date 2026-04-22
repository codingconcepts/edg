package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONWriter_Add_SkipsNoArgs(t *testing.T) {
	w := newJSONWriter(t.TempDir())

	err := w.Add(WriteRow{
		Section: "up",
		Name:    "create_table",
		SQL:     "CREATE TABLE users (id INT)",
	})
	require.NoError(t, err)
	assert.Empty(t, w.data)
}

func TestJSONWriter_Add_AccumulatesRows(t *testing.T) {
	w := newJSONWriter(t.TempDir())

	rows := []WriteRow{
		{Section: "seed", Name: "users", Columns: []string{"name", "age"}, Args: []any{"Alice", 30}},
		{Section: "seed", Name: "users", Columns: []string{"name", "age"}, Args: []any{"Bob", 25}},
	}

	for _, r := range rows {
		require.NoError(t, w.Add(r))
	}

	require.Len(t, w.data["seed"]["users"], 2)
	assert.Equal(t, map[string]any{"name": "Alice", "age": 30}, w.data["seed"]["users"][0])
	assert.Equal(t, map[string]any{"name": "Bob", "age": 25}, w.data["seed"]["users"][1])
}

func TestJSONWriter_Flush(t *testing.T) {
	dir := t.TempDir()
	w := newJSONWriter(dir)

	w.data["seed"] = map[string][]map[string]any{
		"users": {
			{"name": "Alice", "age": 30},
		},
	}

	err := w.Flush()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "seed.json"))
	require.NoError(t, err)

	var parsed map[string][]map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Equal(t, "Alice", parsed["users"][0]["name"])
}

func TestNormalizeValue(t *testing.T) {
	cases := []struct {
		name string
		val  any
		want any
	}{
		{name: "nil", val: nil, want: nil},
		{name: "string", val: "hello", want: "hello"},
		{name: "int", val: 42, want: 42},
		{name: "bytes", val: []byte{0xca, 0xfe}, want: "cafe"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, normalizeValue(tc.val))
		})
	}
}
