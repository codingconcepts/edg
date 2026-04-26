package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLoadConfig_NoIncludes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.yaml", `
globals:
  batch_size: 100
up:
  - name: create_table
    query: CREATE TABLE t (id INT)
`)

	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	require.NoError(t, err)

	assert.Equal(t, 100, req.Globals["batch_size"])
	require.Len(t, req.Up, 1)
	assert.Equal(t, "create_table", req.Up[0].Name)
}

func TestLoadConfig_NonexistentFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bad.yaml", ":\n\t- invalid yaml {{{")
	_, err := LoadConfig(filepath.Join(dir, "bad.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing")
}

func TestLoadConfig_DecodeError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bad.yaml", "up: not_a_list")
	_, err := LoadConfig(filepath.Join(dir, "bad.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decoding")
}

func TestParseConfig_GlobalsOrder(t *testing.T) {
	input := `
globals:
  warehouses: 1
  districts: 10
  customers: 30000
  batch_size: 500
`
	req, err := ParseConfig([]byte(input))
	require.NoError(t, err)

	assert.Equal(t, []string{"warehouses", "districts", "customers", "batch_size"}, req.GlobalsOrder)
}

func TestParseConfig_GlobalsOrder_NoGlobals(t *testing.T) {
	input := `
up:
  - name: t
    query: CREATE TABLE t (id INT)
`
	req, err := ParseConfig([]byte(input))
	require.NoError(t, err)

	assert.Nil(t, req.GlobalsOrder)
}

func TestParseConfig_InvalidYAML(t *testing.T) {
	_, err := ParseConfig([]byte(":\n  :\n  - :\n  invalid"))
	require.Error(t, err)
}

func TestParseConfig_DecodeError(t *testing.T) {
	_, err := ParseConfig([]byte("up: not_a_list"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decoding config")
}

func TestExtractGlobalsOrder(t *testing.T) {
	cases := []struct {
		name string
		node *yaml.Node
		want []string
	}{
		{
			name: "non-document node",
			node: &yaml.Node{Kind: yaml.ScalarNode, Value: "hello"},
		},
		{
			name: "non-mapping root",
			node: &yaml.Node{
				Kind:    yaml.DocumentNode,
				Content: []*yaml.Node{{Kind: yaml.SequenceNode}},
			},
		},
		{
			name: "non-mapping globals",
			node: func() *yaml.Node {
				var doc yaml.Node
				_ = yaml.Unmarshal([]byte("globals:\n  - item1\n  - item2\n"), &doc)
				return &doc
			}(),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Nil(t, extractGlobalsOrder(c.node))
		})
	}
}
