package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/codingconcepts/edg/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestRequestParsesSeedSection(t *testing.T) {
	input := `
up:
  - name: create_table
    query: CREATE TABLE t (id INT PRIMARY KEY)
seed:
  - name: populate_table
    args:
      - items
    query: INSERT INTO t SELECT s FROM generate_series(1, $1) AS s
down:
  - name: drop_table
    query: DROP TABLE t
`
	var req Request
	require.NoError(t, yaml.Unmarshal([]byte(input), &req))

	require.Len(t, req.Up, 1)
	require.Len(t, req.Seed, 1)
	assert.Equal(t, "populate_table", req.Seed[0].Name)
	require.Len(t, req.Seed[0].Args, 1)
	require.Len(t, req.Down, 1)
}

func TestRequestParsesDeseedSection(t *testing.T) {
	input := `
deseed:
  - name: truncate_items
    type: exec
    query: TRUNCATE TABLE item
  - name: truncate_warehouse
    type: exec
    query: TRUNCATE TABLE warehouse
`
	var req Request
	require.NoError(t, yaml.Unmarshal([]byte(input), &req))

	require.Len(t, req.Deseed, 2)
	assert.Equal(t, "truncate_items", req.Deseed[0].Name)
	assert.Equal(t, QueryTypeExec, req.Deseed[0].Type)
}

func TestRequestParsesEmptySeed(t *testing.T) {
	input := `
up:
  - name: create_table
    query: CREATE TABLE t (id INT PRIMARY KEY)
`
	var req Request
	require.NoError(t, yaml.Unmarshal([]byte(input), &req))

	assert.Empty(t, req.Seed)
}

func TestRequestParsesExpectations(t *testing.T) {
	input := `
run:
  - name: check_balance
    query: SELECT 1
expectations:
  - error_rate < 1
  - check_balance.p99 < 100
`
	var req Request
	require.NoError(t, yaml.Unmarshal([]byte(input), &req))

	require.Len(t, req.Expectations, 2)
	assert.Equal(t, "error_rate < 1", req.Expectations[0])
	assert.Equal(t, "check_balance.p99 < 100", req.Expectations[1])
}

func TestDurationUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   time.Duration
		expErr string
	}{
		{"seconds", "wait: 5s", 5 * time.Second, ""},
		{"milliseconds", "wait: 250ms", 250 * time.Millisecond, ""},
		{"minutes", "wait: 2m", 2 * time.Minute, ""},
		{"complex", "wait: 1m30s", 90 * time.Second, ""},
		{"invalid", "wait: notaduration", 0, "invalid duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out struct {
				Wait Duration `yaml:"wait"`
			}
			err := yaml.Unmarshal([]byte(tt.input), &out)
			if tt.expErr != "" {
				require.ErrorContains(t, err, tt.expErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, time.Duration(out.Wait))
		})
	}
}

func TestRequestParsesBatchType(t *testing.T) {
	input := `
seed:
  - name: populate_product
    type: exec_batch
    count: 100
    size: 50
    args:
      - gen('productname')
    query: |-
      INSERT INTO product (name) SELECT unnest(string_to_array('$1', ','))
`
	var req Request
	require.NoError(t, yaml.Unmarshal([]byte(input), &req))

	require.Len(t, req.Seed, 1)
	q := req.Seed[0]
	assert.Equal(t, QueryTypeExecBatch, q.Type)
	// Count/Size are parsed as any from YAML, typically int.
	assert.Equal(t, 100, q.Count)
	assert.Equal(t, 50, q.Size)
}

func TestConfigSectionSeedValue(t *testing.T) {
	assert.Equal(t, ConfigSection("seed"), ConfigSectionSeed)
}

func TestConfigSectionDeseedValue(t *testing.T) {
	assert.Equal(t, ConfigSection("deseed"), ConfigSectionDeseed)
}

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

func TestTransactionParsesRollbackIf(t *testing.T) {
	input := `
run:
  - transaction: make_transfer
    queries:
      - name: read_source
        type: query
        query: SELECT id, balance FROM account WHERE id = 1
      - rollback_if: "ref_same('read_source').balance < 100"
`
	var req Request
	require.NoError(t, yaml.Unmarshal([]byte(input), &req))

	require.Len(t, req.Run, 1)
	require.True(t, req.Run[0].IsTransaction())

	tx := req.Run[0].Transaction
	assert.Equal(t, "make_transfer", tx.Name)
	require.Len(t, tx.Queries, 2)
	require.True(t, tx.Queries[1].IsRollbackIf())
	assert.Equal(t, "ref_same('read_source').balance < 100", tx.Queries[1].RollbackIf)
}

func TestTransactionParsesWithoutRollbackIf(t *testing.T) {
	input := `
run:
  - transaction: simple
    queries:
      - name: q1
        type: exec
        query: INSERT INTO t VALUES (1)
`
	var req Request
	require.NoError(t, yaml.Unmarshal([]byte(input), &req))

	for _, q := range req.Run[0].Transaction.Queries {
		assert.False(t, q.IsRollbackIf())
	}
}

func TestCompileRollbackIf_Valid(t *testing.T) {
	q := &Query{RollbackIf: "balance < 100"}

	envMap := map[string]any{"balance": 50}
	require.NoError(t, q.CompileRollbackIf(envMap))
	require.NotNil(t, q.CompiledRollbackIf)
}

func TestCompileRollbackIf_Empty(t *testing.T) {
	q := &Query{}

	require.NoError(t, q.CompileRollbackIf(map[string]any{}))
	require.Nil(t, q.CompiledRollbackIf)
}

func TestCompileRollbackIf_Invalid(t *testing.T) {
	q := &Query{RollbackIf: "invalid +++"}

	err := q.CompileRollbackIf(map[string]any{})
	require.Error(t, err)
}

func TestCompileRollbackIf_NonBoolReturnsError(t *testing.T) {
	q := &Query{RollbackIf: "42"}

	err := q.CompileRollbackIf(map[string]any{})
	require.Error(t, err)
}

func TestTransactionParsesLocals(t *testing.T) {
	input := `
run:
  - transaction: make_transfer
    locals:
      amount: gen('number:1,100')
      fee: const(5)
    queries:
      - name: debit
        type: exec
        query: UPDATE account SET balance = balance - 1
`
	var req Request
	require.NoError(t, yaml.Unmarshal([]byte(input), &req))

	tx := req.Run[0].Transaction
	require.Len(t, tx.Locals, 2)
	assert.Equal(t, "gen('number:1,100')", tx.Locals["amount"])
	assert.Equal(t, "const(5)", tx.Locals["fee"])
}

func TestValidate_LocalShadowsQueryName(t *testing.T) {
	req := &Request{
		Run: []*RunItem{
			{Transaction: &Transaction{
				Name:   "tx",
				Locals: map[string]string{"debit": "const(1)"},
				Queries: []*Query{
					{Name: "debit", Type: QueryTypeExec, Query: "UPDATE t SET x = 1"},
				},
			}},
		},
	}

	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `local "debit" shadows query name`)
}

func TestCompileLocals_Valid(t *testing.T) {
	envMap := map[string]any{
		"const": func(v any) any { return v },
	}
	tx := &Transaction{
		Locals: map[string]string{"amount": "const(42)"},
	}

	require.NoError(t, tx.CompileLocals(envMap))
	require.Len(t, tx.CompiledLocals, 1)
	require.NotNil(t, tx.CompiledLocals["amount"])
}

func TestCompileLocals_Invalid(t *testing.T) {
	tx := &Transaction{
		Locals: map[string]string{"bad": "invalid +++"},
	}

	err := tx.CompileLocals(map[string]any{})
	require.Error(t, err)
}

func TestValidate_GenPatternValid(t *testing.T) {
	req := &Request{
		Run: []*RunItem{
			{Query: &Query{Name: "q1", Args: []string{"gen('email')", "gen('number:1,100')"}}},
		},
	}
	require.NoError(t, req.Validate())
}

func TestValidate_GenPatternInvalid(t *testing.T) {
	req := &Request{
		Run: []*RunItem{
			{Query: &Query{Name: "q1", Args: []string{"gen('notafunction')"}}},
		},
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notafunction")
}

func TestValidate_GenPatternInRow(t *testing.T) {
	req := &Request{
		Rows: map[string][]string{
			"user": {"gen('email')", "gen('bogusxyz')"},
		},
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogusxyz")
}

func TestValidate_GenPatternInLocals(t *testing.T) {
	req := &Request{
		Run: []*RunItem{
			{Transaction: &Transaction{
				Name:   "tx",
				Locals: map[string]string{"amount": "gen('notreal')"},
				Queries: []*Query{
					{Name: "q1", Type: QueryTypeExec, Query: "SELECT 1"},
				},
			}},
		},
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notreal")
}

func TestValidate_EnvPattern(t *testing.T) {
	cases := []struct {
		name    string
		env     map[string]string
		pattern string
		expErr  string
	}{
		{
			name: "valid single quote",
			env: map[string]string{
				"ABC": "123",
			},
			pattern: `env('ABC')`,
			expErr:  "",
		},
		{
			name: "valid double quote",
			env: map[string]string{
				"ABC": "123",
			},
			pattern: `env("ABC")`,
			expErr:  "",
		},
		{
			name: "missing env var",
			env: map[string]string{
				"ABC": "123",
			},
			pattern: `env("DEF")`,
			expErr:  "missing environment variable",
		},
		{
			name: "mismatched quotes not matched",
			env: map[string]string{
				"ABC": "123",
			},
			pattern: `env('ABC")`,
			expErr:  "", // regex won't match, so no validation error (expr-lang catches syntax)
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test.CleanupEnv(t, "ABC")

			for k, v := range c.env {
				require.NoError(t, os.Setenv(k, v))
			}

			req := &Request{
				Rows: map[string][]string{
					"value": {c.pattern},
				},
			}

			err := req.Validate()

			if c.expErr != "" {
				require.ErrorContains(t, err, c.expErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
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
