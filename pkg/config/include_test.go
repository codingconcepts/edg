package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_IncludeMapping(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "shared/globals.yaml", `
batch_size: 500
workers: 4
`)
	writeFile(t, dir, "main.yaml", `
globals: !include shared/globals.yaml
up:
  - name: create_table
    query: CREATE TABLE t (id INT)
`)

	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	require.NoError(t, err)

	assert.Equal(t, 500, req.Globals["batch_size"])
	assert.Equal(t, 4, req.Globals["workers"])
}

func TestLoadConfig_IncludeSequence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "shared/schema.yaml", `
- name: create_users
  query: CREATE TABLE users (id INT)
- name: create_orders
  query: CREATE TABLE orders (id INT)
`)
	writeFile(t, dir, "main.yaml", `
up: !include shared/schema.yaml
`)

	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	require.NoError(t, err)

	require.Len(t, req.Up, 2)
	assert.Equal(t, "create_users", req.Up[0].Name)
	assert.Equal(t, "create_orders", req.Up[1].Name)
}

func TestLoadConfig_IncludeSequenceItem(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "shared/transfer.yaml", `
- name: make_transfer
  type: exec
  query: UPDATE account SET balance = balance + 1
`)
	writeFile(t, dir, "main.yaml", `
run:
  - name: check_balance
    type: query
    query: SELECT balance FROM account WHERE id = 1
  - !include shared/transfer.yaml
`)

	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	require.NoError(t, err)

	require.Len(t, req.Run, 2)
	assert.Equal(t, "check_balance", req.Run[0].Name())
	assert.Equal(t, "make_transfer", req.Run[1].Name())
}

func TestLoadConfig_NestedIncludes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "level2.yaml", `
batch_size: 42
`)
	writeFile(t, dir, "main.yaml", `
globals: !include level2.yaml
up:
  - name: t
    query: CREATE TABLE t (id INT)
`)

	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	require.NoError(t, err)

	assert.Equal(t, 42, req.Globals["batch_size"])
}

func TestLoadConfig_CircularInclude(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.yaml", `
globals: !include b.yaml
`)
	writeFile(t, dir, "b.yaml", `
batch_size: !include a.yaml
`)

	_, err := LoadConfig(filepath.Join(dir, "a.yaml"))
	require.Error(t, err)
}

func TestLoadConfig_MissingInclude(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.yaml", `
globals: !include nonexistent.yaml
`)

	_, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	require.Error(t, err)
}

func TestLoadConfig_MultipleIncludes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "shared/globals.yaml", `
batch_size: 100
`)
	writeFile(t, dir, "shared/schema.yaml", `
- name: create_table
  query: CREATE TABLE t (id INT)
`)
	writeFile(t, dir, "shared/teardown.yaml", `
- name: drop_table
  type: exec
  query: DROP TABLE t
`)
	writeFile(t, dir, "main.yaml", `
globals: !include shared/globals.yaml
up: !include shared/schema.yaml
down: !include shared/teardown.yaml
`)

	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	require.NoError(t, err)

	assert.Equal(t, 100, req.Globals["batch_size"])
	require.Len(t, req.Up, 1)
	assert.Equal(t, "create_table", req.Up[0].Name)
	require.Len(t, req.Down, 1)
	assert.Equal(t, "drop_table", req.Down[0].Name)
}

func TestLoadConfig_IncludeNonSequenceInSequence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "single.yaml", `
name: q1
query: SELECT 1
`)
	writeFile(t, dir, "main.yaml", `
up:
  - !include single.yaml
`)
	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	require.NoError(t, err)
	require.Len(t, req.Up, 1)
	assert.Equal(t, "q1", req.Up[0].Name)
}

func TestLoadConfig_InvalidIncludeYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bad.yaml", ":\n\t{{{invalid")
	writeFile(t, dir, "main.yaml", `
globals: !include bad.yaml
`)
	_, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing include")
}

func TestResolveIncludes_NilNode(t *testing.T) {
	err := resolveIncludes(nil, ".", map[string]bool{})
	require.NoError(t, err)
}

func TestLoadConfig_NestedIncludeInSequenceItem(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "shared/inner.yaml", `
batch_size: 99
`)
	writeFile(t, dir, "shared/query.yaml", `
- name: q1
  type: exec
  query: SELECT 1
`)
	writeFile(t, dir, "main.yaml", `
globals: !include shared/inner.yaml
run:
  - !include shared/query.yaml
`)
	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	require.NoError(t, err)
	assert.Equal(t, 99, req.Globals["batch_size"])
	require.Len(t, req.Run, 1)
}
