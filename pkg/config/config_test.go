package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/codingconcepts/edg/pkg/seq"
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
	require.Equal(t, 1, req.Seed[0].Args.Len())
	require.Len(t, req.Down, 1)
}

func TestRequestParsesNamedArgs(t *testing.T) {
	input := `
run:
  - name: insert_order
    type: exec
    args:
      email: gen('email')
      region: ref_same('regions').name
      city: set_rand(ref_same('regions').cities, [])
      amount: uniform(1, 500)
    query: INSERT INTO t VALUES ($1, $2, $3, $4)
`
	var req Request
	require.NoError(t, yaml.Unmarshal([]byte(input), &req))

	require.Equal(t, 4, req.Run[0].Query.Args.Len())
	require.True(t, req.Run[0].Query.Args.IsNamed())
	assert.Equal(t, "gen('email')", req.Run[0].Query.Args.Exprs[0])
	assert.Equal(t, "uniform(1, 500)", req.Run[0].Query.Args.Exprs[3])
	assert.Equal(t, 0, req.Run[0].Query.Args.Names["email"])
	assert.Equal(t, 3, req.Run[0].Query.Args.Names["amount"])
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
  - query: SELECT COUNT(*) AS cnt FROM account
    expr: cnt > 0
`
	var req Request
	require.NoError(t, yaml.Unmarshal([]byte(input), &req))

	require.Len(t, req.Expectations, 3)
	assert.Equal(t, Expectation{Expr: "error_rate < 1"}, req.Expectations[0])
	assert.Equal(t, Expectation{Expr: "check_balance.p99 < 100"}, req.Expectations[1])
	assert.Equal(t, Expectation{Query: "SELECT COUNT(*) AS cnt FROM account", Expr: "cnt > 0"}, req.Expectations[2])
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

func TestConfigSectionValues(t *testing.T) {
	cases := []struct {
		name string
		got  ConfigSection
		want ConfigSection
	}{
		{"seed", ConfigSection("seed"), ConfigSectionSeed},
		{"deseed", ConfigSection("deseed"), ConfigSectionDeseed},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.got)
		})
	}
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
	cases := []struct {
		name         string
		input        string
		wantRollback bool
		rollbackExpr string
	}{
		{
			name: "with rollback_if",
			input: `
run:
  - transaction: make_transfer
    queries:
      - name: read_source
        type: query
        query: SELECT id, balance FROM account WHERE id = 1
      - rollback_if: "ref_same('read_source').balance < 100"
`,
			wantRollback: true,
			rollbackExpr: "ref_same('read_source').balance < 100",
		},
		{
			name: "without rollback_if",
			input: `
run:
  - transaction: simple
    queries:
      - name: q1
        type: exec
        query: INSERT INTO t VALUES (1)
`,
			wantRollback: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var req Request
			require.NoError(t, yaml.Unmarshal([]byte(c.input), &req))

			require.Len(t, req.Run, 1)
			require.True(t, req.Run[0].IsTransaction())

			tx := req.Run[0].Transaction
			if c.wantRollback {
				require.True(t, tx.Queries[len(tx.Queries)-1].IsRollbackIf())
				assert.Equal(t, c.rollbackExpr, tx.Queries[len(tx.Queries)-1].RollbackIf)
			} else {
				for _, q := range tx.Queries {
					assert.False(t, q.IsRollbackIf())
				}
			}
		})
	}
}

func TestCompileRollbackIf(t *testing.T) {
	cases := []struct {
		name       string
		rollbackIf string
		env        map[string]any
		wantErr    bool
		wantNil    bool
	}{
		{
			name:       "valid expression",
			rollbackIf: "balance < 100",
			env:        map[string]any{"balance": 50},
		},
		{
			name:    "empty expression",
			env:     map[string]any{},
			wantNil: true,
		},
		{
			name:       "invalid syntax",
			rollbackIf: "invalid +++",
			env:        map[string]any{},
			wantErr:    true,
		},
		{
			name:       "non-bool return",
			rollbackIf: "42",
			env:        map[string]any{},
			wantErr:    true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q := &Query{RollbackIf: c.rollbackIf}
			err := q.CompileRollbackIf(c.env)

			switch {
			case c.wantErr:
				require.Error(t, err)
			case c.wantNil:
				require.NoError(t, err)
				require.Nil(t, q.CompiledRollbackIf)
			default:
				require.NoError(t, err)
				require.NotNil(t, q.CompiledRollbackIf)
			}
		})
	}
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

func TestCompileLocals(t *testing.T) {
	cases := []struct {
		name    string
		locals  map[string]string
		env     map[string]any
		wantErr bool
		wantLen int
	}{
		{
			name:    "valid expression",
			locals:  map[string]string{"amount": "const(42)"},
			env:     map[string]any{"const": func(v any) any { return v }},
			wantLen: 1,
		},
		{
			name:    "invalid syntax",
			locals:  map[string]string{"bad": "invalid +++"},
			env:     map[string]any{},
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tx := &Transaction{Locals: c.locals}
			err := tx.CompileLocals(c.env)

			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, tx.CompiledLocals, c.wantLen)
		})
	}
}

func TestValidate_GenPattern(t *testing.T) {
	cases := []struct {
		name    string
		req     *Request
		wantErr string
	}{
		{
			name: "valid args",
			req: &Request{
				Run: []*RunItem{
					{Query: &Query{Name: "q1", Args: PositionalArgs("gen('email')", "gen('number:1,100')")}},
				},
			},
		},
		{
			name: "invalid arg",
			req: &Request{
				Run: []*RunItem{
					{Query: &Query{Name: "q1", Args: PositionalArgs("gen('notafunction')")}},
				},
			},
			wantErr: "notafunction",
		},
		{
			name: "invalid row",
			req: &Request{
				Rows: map[string][]string{
					"user": {"gen('email')", "gen('bogusxyz')"},
				},
			},
			wantErr: "bogusxyz",
		},
		{
			name: "invalid local",
			req: &Request{
				Run: []*RunItem{
					{Transaction: &Transaction{
						Name:   "tx",
						Locals: map[string]string{"amount": "gen('notreal')"},
						Queries: []*Query{
							{Name: "q1", Type: QueryTypeExec, Query: "SELECT 1"},
						},
					}},
				},
			},
			wantErr: "notreal",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.req.Validate()
			if c.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), c.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
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

func TestValidate_SectionFiltering(t *testing.T) {
	req := &Request{
		Up: []*Query{
			{Name: "create_table", Query: "CREATE TABLE t (id INT)"},
		},
		Run: []*RunItem{
			{Query: &Query{
				Name: "insert",
				Type: QueryTypeExec,
				Args: PositionalArgs("env('FLY_REGION')"),
			}},
		},
	}

	cases := []struct {
		name     string
		env      map[string]string
		sections []ConfigSection
		wantErr  string
	}{
		{
			name:    "all sections fails without env",
			wantErr: "missing environment variable",
		},
		{
			name:     "up only passes without env",
			sections: []ConfigSection{ConfigSectionUp},
		},
		{
			name:     "run fails without env",
			sections: []ConfigSection{ConfigSectionRun},
			wantErr:  "missing environment variable",
		},
		{
			name:     "run passes with env set",
			env:      map[string]string{"FLY_REGION": "fra"},
			sections: []ConfigSection{ConfigSectionRun},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test.CleanupEnv(t, "FLY_REGION")
			for k, v := range c.env {
				t.Setenv(k, v)
			}

			err := req.Validate(c.sections...)
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
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

func TestRateUnmarshalYAML(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		wantTimes    int
		wantInterval time.Duration
		wantTicker   time.Duration
		expErr       string
	}{
		{"once per 10s", "rate: 1/10s", 1, 10 * time.Second, 10 * time.Second, ""},
		{"3 per 10s", "rate: 3/10s", 3, 10 * time.Second, 10 * time.Second / 3, ""},
		{"once per minute", "rate: 1/1m", 1, time.Minute, time.Minute, ""},
		{"5 per 90s", "rate: 5/1m30s", 5, 90 * time.Second, 18 * time.Second, ""},
		{"bad format", "rate: nope", 0, 0, 0, "invalid rate format"},
		{"bad times", "rate: x/10s", 0, 0, 0, "parsing times"},
		{"bad interval", "rate: 1/xyz", 0, 0, 0, "parsing interval"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out struct {
				Rate Rate `yaml:"rate"`
			}
			err := yaml.Unmarshal([]byte(c.input), &out)
			if c.expErr != "" {
				require.ErrorContains(t, err, c.expErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.wantTimes, out.Rate.Times)
			assert.Equal(t, c.wantInterval, out.Rate.Interval)
			assert.Equal(t, c.wantTicker, out.Rate.TickerInterval())
		})
	}
}

func TestRateString(t *testing.T) {
	r := Rate{Times: 3, Interval: 10 * time.Second}
	assert.Equal(t, "3/10s", r.String())
}

func TestRequestParsesWorkers(t *testing.T) {
	input := `
workers:
  - name: reaper
    rate: 1/10s
    type: exec
    query: UPDATE runs SET status = 'pending' WHERE lease_expires_at < now()
  - name: counter
    rate: 5/1m
    type: query
    args:
      - const(1)
    query: SELECT count(*) FROM events WHERE id > $1
`
	var req Request
	require.NoError(t, yaml.Unmarshal([]byte(input), &req))

	require.Len(t, req.Workers, 2)
	assert.Equal(t, "reaper", req.Workers[0].Name)
	assert.Equal(t, 1, req.Workers[0].Rate.Times)
	assert.Equal(t, 10*time.Second, req.Workers[0].Rate.Interval)
	assert.Equal(t, QueryTypeExec, req.Workers[0].Type)

	assert.Equal(t, "counter", req.Workers[1].Name)
	assert.Equal(t, 5, req.Workers[1].Rate.Times)
	assert.Equal(t, time.Minute, req.Workers[1].Rate.Interval)
	require.Equal(t, 1, req.Workers[1].Args.Len())
}

func TestValidate_WorkerMissingName(t *testing.T) {
	req := &Request{
		Workers: []*Worker{
			{Query: Query{Type: QueryTypeExec, Query: "SELECT 1"}, Rate: Rate{Times: 1, Interval: time.Second, tickerInterval: time.Second}},
		},
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing a name")
}

func TestValidate_WorkerBadRate(t *testing.T) {
	req := &Request{
		Workers: []*Worker{
			{Query: Query{Name: "bad", Type: QueryTypeExec, Query: "SELECT 1"}, Rate: Rate{Times: 0, Interval: time.Second}},
		},
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate must have positive")
}

func TestValidate_SeqDuplicateName(t *testing.T) {
	req := &Request{
		Seq: []seq.Config{
			{Name: "a", Start: 1, Step: 1},
			{Name: "a", Start: 1, Step: 1},
		},
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate seq name")
}

func TestValidate_SeqMissingName(t *testing.T) {
	req := &Request{
		Seq: []seq.Config{
			{Start: 1, Step: 1},
		},
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing a name")
}

func TestValidate_SeqReference(t *testing.T) {
	cases := []struct {
		name    string
		req     *Request
		wantErr string
	}{
		{
			name: "valid seq_global reference",
			req: &Request{
				Seq: []seq.Config{{Name: "order_id", Start: 1, Step: 1}},
				Seed: []*Query{
					{Name: "q1", Args: PositionalArgs(`seq_global("order_id")`)},
				},
			},
		},
		{
			name: "valid seq_rand reference",
			req: &Request{
				Seq: []seq.Config{{Name: "order_id", Start: 1, Step: 1}},
				Seed: []*Query{
					{Name: "q1", Args: PositionalArgs(`seq_rand("order_id")`)},
				},
			},
		},
		{
			name: "valid seq_zipf reference",
			req: &Request{
				Seq: []seq.Config{{Name: "order_id", Start: 1, Step: 1}},
				Run: []*RunItem{
					{Query: &Query{Name: "q1", Args: PositionalArgs(`seq_zipf("order_id", 2.0, 1.0)`)}},
				},
			},
		},
		{
			name: "valid seq_norm reference",
			req: &Request{
				Seq: []seq.Config{{Name: "order_id", Start: 1, Step: 1}},
				Run: []*RunItem{
					{Query: &Query{Name: "q1", Args: PositionalArgs(`seq_norm("order_id", 500, 100)`)}},
				},
			},
		},
		{
			name: "valid seq_exp reference",
			req: &Request{
				Seq: []seq.Config{{Name: "order_id", Start: 1, Step: 1}},
				Run: []*RunItem{
					{Query: &Query{Name: "q1", Args: PositionalArgs(`seq_exp("order_id", 0.5)`)}},
				},
			},
		},
		{
			name: "valid seq_lognorm reference",
			req: &Request{
				Seq: []seq.Config{{Name: "order_id", Start: 1, Step: 1}},
				Run: []*RunItem{
					{Query: &Query{Name: "q1", Args: PositionalArgs(`seq_lognorm("order_id", 1, 0.5)`)}},
				},
			},
		},
		{
			name: "unknown seq_global reference",
			req: &Request{
				Seed: []*Query{
					{Name: "q1", Args: PositionalArgs(`seq_global("nope")`)},
				},
			},
			wantErr: `unknown sequence "nope"`,
		},
		{
			name: "unknown seq_rand reference",
			req: &Request{
				Run: []*RunItem{
					{Query: &Query{Name: "q1", Args: PositionalArgs(`seq_rand("missing")`)}},
				},
			},
			wantErr: `unknown sequence "missing"`,
		},
		{
			name: "unknown seq_zipf reference",
			req: &Request{
				Run: []*RunItem{
					{Query: &Query{Name: "q1", Args: PositionalArgs(`seq_zipf('bad', 2.0, 1.0)`)}},
				},
			},
			wantErr: `unknown sequence "bad"`,
		},
		{
			name: "single-quoted seq reference",
			req: &Request{
				Seq: []seq.Config{{Name: "order_id", Start: 1, Step: 1}},
				Seed: []*Query{
					{Name: "q1", Args: PositionalArgs(`seq_global('order_id')`)},
				},
			},
		},
		{
			name: "seq reference in print",
			req: &Request{
				Seq: []seq.Config{{Name: "order_id", Start: 1, Step: 1}},
				Run: []*RunItem{
					{Query: &Query{Name: "q1", Print: []PrintExpr{{Expr: `seq_rand("order_id")`}}}},
				},
			},
		},
		{
			name: "seq reference in expression",
			req: &Request{
				Seq:         []seq.Config{{Name: "order_id", Start: 1, Step: 1}},
				Expressions: map[string]string{"next_order": `seq_global("order_id")`},
			},
		},
		{
			name: "seq reference in transaction local",
			req: &Request{
				Seq: []seq.Config{{Name: "order_id", Start: 1, Step: 1}},
				Run: []*RunItem{
					{Transaction: &Transaction{
						Name:   "tx",
						Locals: map[string]string{"oid": `seq_global("order_id")`},
						Queries: []*Query{
							{Name: "q1", Type: QueryTypeExec, Query: "SELECT 1"},
						},
					}},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.req.Validate()
			if c.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), c.wantErr)
				return
			}
			require.NoError(t, err)
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

func TestValidate_ExpectationGlobalShadowsMetric(t *testing.T) {
	for _, metric := range []string{MetricSuccessCount, MetricTotalErrors, MetricErrorRate, MetricTPM} {
		t.Run(metric, func(t *testing.T) {
			req := &Request{
				Globals:      map[string]any{metric: 42},
				Expectations: []Expectation{{Expr: "true"}},
			}
			err := req.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "shadows a built-in expectation metric")
		})
	}
}

func TestValidate_ExpectationGlobalAllowed(t *testing.T) {
	req := &Request{
		Globals:      map[string]any{"accounts": 100},
		Expectations: []Expectation{{Expr: "true"}},
	}
	require.NoError(t, req.Validate())
}

func TestValidate_GlobalShadowSkippedWithoutExpectations(t *testing.T) {
	req := &Request{
		Globals: map[string]any{MetricErrorRate: 42},
	}
	require.NoError(t, req.Validate())
}
