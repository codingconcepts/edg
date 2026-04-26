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
	cases := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr string
	}{
		{"seconds", "wait: 5s", 5 * time.Second, ""},
		{"milliseconds", "wait: 250ms", 250 * time.Millisecond, ""},
		{"minutes", "wait: 2m", 2 * time.Minute, ""},
		{"complex", "wait: 1m30s", 90 * time.Second, ""},
		{"invalid", "wait: notaduration", 0, "invalid duration"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out struct {
				Wait Duration `yaml:"wait"`
			}
			err := yaml.Unmarshal([]byte(c.input), &out)
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, time.Duration(out.Wait))
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

func TestValidate_Transactions(t *testing.T) {
	cases := []struct {
		name    string
		req     *Request
		wantErr string
	}{
		{
			name: "local shadows query name",
			req: &Request{
				Run: []*RunItem{
					{Transaction: &Transaction{
						Name:   "tx",
						Locals: map[string]string{"debit": "const(1)"},
						Queries: []*Query{
							{Name: "debit", Type: QueryTypeExec, Query: "UPDATE t SET x = 1"},
						},
					}},
				},
			},
			wantErr: `local "debit" shadows query name`,
		},
		{
			name: "empty queries",
			req: &Request{
				Run: []*RunItem{
					{Transaction: &Transaction{Name: "empty_tx", Queries: []*Query{}}},
				},
			},
			wantErr: "must contain at least one query",
		},
		{
			name: "batch not allowed",
			req: &Request{
				Run: []*RunItem{
					{Transaction: &Transaction{
						Name:    "tx",
						Queries: []*Query{{Name: "q1", Type: QueryTypeExecBatch, Query: "INSERT INTO t VALUES ($1)"}},
					}},
				},
			},
			wantErr: "cannot be a batch type inside a transaction",
		},
		{
			name: "prepared not allowed",
			req: &Request{
				Run: []*RunItem{
					{Transaction: &Transaction{
						Name:    "tx",
						Queries: []*Query{{Name: "q1", Type: QueryTypeExec, Prepared: true, Query: "INSERT INTO t VALUES ($1)"}},
					}},
				},
			},
			wantErr: "cannot use prepared statements inside a transaction",
		},
		{
			name: "rollback_if with extra fields",
			req: &Request{
				Run: []*RunItem{
					{Transaction: &Transaction{
						Name: "tx",
						Queries: []*Query{
							{Name: "q1", Type: QueryTypeExec, Query: "INSERT INTO t VALUES (1)"},
							{RollbackIf: "true", Name: "bad", Query: "SELECT 1"},
						},
					}},
				},
			},
			wantErr: "must not have name, type, args, or query",
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

func TestCompileLocals(t *testing.T) {
	cases := []struct {
		name    string
		locals  map[string]string
		env     map[string]any
		wantErr string
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
			wantErr: "failed to compile local",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tx := &Transaction{Locals: c.locals}
			err := tx.CompileLocals(c.env)
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
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
		wantErr string
	}{
		{
			name:    "valid single quote",
			env:     map[string]string{"ABC": "123"},
			pattern: `env('ABC')`,
		},
		{
			name:    "valid double quote",
			env:     map[string]string{"ABC": "123"},
			pattern: `env("ABC")`,
		},
		{
			name:    "missing env var",
			env:     map[string]string{"ABC": "123"},
			pattern: `env("DEF")`,
			wantErr: "missing environment variable",
		},
		{
			name:    "mismatched quotes not matched",
			env:     map[string]string{"ABC": "123"},
			pattern: `env('ABC")`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test.CleanupEnv(t, "ABC")
			for k, v := range c.env {
				t.Setenv(k, v)
			}

			req := &Request{
				Rows: map[string][]string{
					"value": {c.pattern},
				},
			}
			err := req.Validate()
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
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

func TestRateUnmarshalYAML(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		wantTimes    int
		wantInterval time.Duration
		wantTicker   time.Duration
		wantErr      string
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
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
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

func TestValidate_Workers(t *testing.T) {
	cases := []struct {
		name    string
		req     *Request
		wantErr string
	}{
		{
			name: "missing name",
			req: &Request{
				Workers: []*Worker{
					{Query: Query{Type: QueryTypeExec, Query: "SELECT 1"}, Rate: Rate{Times: 1, Interval: time.Second, tickerInterval: time.Second}},
				},
			},
			wantErr: "missing a name",
		},
		{
			name: "bad rate",
			req: &Request{
				Workers: []*Worker{
					{Query: Query{Name: "bad", Type: QueryTypeExec, Query: "SELECT 1"}, Rate: Rate{Times: 0, Interval: time.Second}},
				},
			},
			wantErr: "rate must have positive",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.req.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.wantErr)
		})
	}
}

func TestValidate_Seq(t *testing.T) {
	cases := []struct {
		name    string
		req     *Request
		wantErr string
	}{
		{
			name: "duplicate name",
			req: &Request{
				Seq: []seq.Config{
					{Name: "a", Start: 1, Step: 1},
					{Name: "a", Start: 1, Step: 1},
				},
			},
			wantErr: "duplicate seq name",
		},
		{
			name: "missing name",
			req: &Request{
				Seq: []seq.Config{{Start: 1, Step: 1}},
			},
			wantErr: "missing a name",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.req.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.wantErr)
		})
	}
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

func TestValidate_ExpectationGlobals(t *testing.T) {
	cases := []struct {
		name    string
		req     *Request
		wantErr string
	}{
		{
			name: "shadows success_count",
			req: &Request{
				Globals:      map[string]any{MetricSuccessCount: 42},
				Expectations: []Expectation{{Expr: "true"}},
			},
			wantErr: "shadows a built-in expectation metric",
		},
		{
			name: "shadows total_errors",
			req: &Request{
				Globals:      map[string]any{MetricTotalErrors: 42},
				Expectations: []Expectation{{Expr: "true"}},
			},
			wantErr: "shadows a built-in expectation metric",
		},
		{
			name: "shadows error_rate",
			req: &Request{
				Globals:      map[string]any{MetricErrorRate: 42},
				Expectations: []Expectation{{Expr: "true"}},
			},
			wantErr: "shadows a built-in expectation metric",
		},
		{
			name: "shadows tpm",
			req: &Request{
				Globals:      map[string]any{MetricTPM: 42},
				Expectations: []Expectation{{Expr: "true"}},
			},
			wantErr: "shadows a built-in expectation metric",
		},
		{
			name: "non-reserved global allowed",
			req: &Request{
				Globals:      map[string]any{"accounts": 100},
				Expectations: []Expectation{{Expr: "true"}},
			},
		},
		{
			name: "skipped without expectations",
			req: &Request{
				Globals: map[string]any{MetricErrorRate: 42},
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

func TestCompilePrint(t *testing.T) {
	cases := []struct {
		name    string
		print   []PrintExpr
		env     map[string]any
		wantErr string
		wantLen int
	}{
		{
			name:    "empty print",
			print:   nil,
			env:     map[string]any{},
			wantLen: 0,
		},
		{
			name:    "valid expression",
			print:   []PrintExpr{{Expr: "x + 1"}},
			env:     map[string]any{"x": 1},
			wantLen: 1,
		},
		{
			name:    "multiple expressions",
			print:   []PrintExpr{{Expr: "x"}, {Expr: "y"}},
			env:     map[string]any{"x": 1, "y": 2},
			wantLen: 2,
		},
		{
			name:    "invalid syntax",
			print:   []PrintExpr{{Expr: "invalid +++"}},
			env:     map[string]any{},
			wantErr: "failed to compile print",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q := &Query{Print: c.print}
			err := q.CompilePrint(c.env)
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, q.CompiledPrint, c.wantLen)
		})
	}
}

func TestPrintExprUnmarshalYAML(t *testing.T) {
	t.Run("scalar string", func(t *testing.T) {
		input := `print: latency`
		var out struct {
			Print PrintExpr `yaml:"print"`
		}
		require.NoError(t, yaml.Unmarshal([]byte(input), &out))
		assert.Equal(t, "latency", out.Print.Expr)
		assert.Empty(t, out.Print.Agg)
	})

	t.Run("object with agg", func(t *testing.T) {
		input := `
print:
  expr: latency
  agg: avg
`
		var out struct {
			Print PrintExpr `yaml:"print"`
		}
		require.NoError(t, yaml.Unmarshal([]byte(input), &out))
		assert.Equal(t, "latency", out.Print.Expr)
		assert.Equal(t, "avg", out.Print.Agg)
	})

	t.Run("list of mixed", func(t *testing.T) {
		input := `
print:
  - latency
  - expr: count
    agg: sum
`
		var out struct {
			Print []PrintExpr `yaml:"print"`
		}
		require.NoError(t, yaml.Unmarshal([]byte(input), &out))
		require.Len(t, out.Print, 2)
		assert.Equal(t, "latency", out.Print[0].Expr)
		assert.Equal(t, "count", out.Print[1].Expr)
		assert.Equal(t, "sum", out.Print[1].Agg)
	})
}

func TestCompileArgs(t *testing.T) {
	env := map[string]any{
		"x":     1,
		"y":     "hello",
		"const": func(v any) any { return v },
	}

	cases := []struct {
		name    string
		query   *Query
		env     map[string]any
		wantErr string
	}{
		{
			name:  "positional args",
			query: &Query{Args: PositionalArgs("x", "y")},
			env:   env,
		},
		{
			name:  "with count",
			query: &Query{Args: PositionalArgs("x"), Count: 10},
			env:   env,
		},
		{
			name:  "with size",
			query: &Query{Args: PositionalArgs("x"), Size: 5},
			env:   env,
		},
		{
			name:  "with count and size",
			query: &Query{Args: PositionalArgs("x"), Count: 10, Size: 5},
			env:   env,
		},
		{
			name:    "invalid arg expression",
			query:   &Query{Args: PositionalArgs("bad +++")},
			env:     map[string]any{},
			wantErr: "failed to compile arg",
		},
		{
			name:    "invalid count expression",
			query:   &Query{Args: PositionalArgs("x"), Count: "bad +++"},
			env:     env,
			wantErr: "failed to compile count",
		},
		{
			name:    "invalid size expression",
			query:   &Query{Args: PositionalArgs("x"), Size: "bad +++"},
			env:     env,
			wantErr: "failed to compile size",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.query.CompileArgs(c.env)
			if c.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), c.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, c.query.CompiledArgs, c.query.Args.Len())
			if c.query.Count != nil {
				assert.NotNil(t, c.query.CompiledCount)
			}
			if c.query.Size != nil {
				assert.NotNil(t, c.query.CompiledSize)
			}
		})
	}
}

func TestValidate_RunWeights(t *testing.T) {
	cases := []struct {
		name    string
		req     *Request
		wantErr string
	}{
		{
			name: "missing name",
			req: &Request{
				RunWeights: map[string]int{"q1": 1},
				Run:        []*RunItem{{Query: &Query{Type: QueryTypeExec, Query: "SELECT 1"}}},
			},
			wantErr: "missing a name",
		},
		{
			name: "unknown query",
			req: &Request{
				RunWeights: map[string]int{"nonexistent": 5},
				Run:        []*RunItem{{Query: &Query{Name: "q1", Type: QueryTypeExec, Query: "SELECT 1"}}},
			},
			wantErr: `run_weights references unknown query "nonexistent"`,
		},
		{
			name: "valid",
			req: &Request{
				RunWeights: map[string]int{"q1": 5, "q2": 3},
				Run: []*RunItem{
					{Query: &Query{Name: "q1", Type: QueryTypeExec, Query: "SELECT 1"}},
					{Query: &Query{Name: "q2", Type: QueryTypeExec, Query: "SELECT 2"}},
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

func TestValidate_RowRefs(t *testing.T) {
	cases := []struct {
		name    string
		req     *Request
		wantErr string
	}{
		{
			name: "row and args mutually exclusive",
			req: &Request{
				Rows: map[string][]string{"user": {"gen('email')"}},
				Seed: []*Query{{Name: "q1", Row: "user", Args: PositionalArgs("1")}},
			},
			wantErr: "row and args are mutually exclusive",
		},
		{
			name: "unknown row ref",
			req: &Request{
				Rows: map[string][]string{"user": {"gen('email')"}},
				Seed: []*Query{{Name: "q1", Row: "nonexistent"}},
			},
			wantErr: `references unknown row "nonexistent"`,
		},
		{
			name: "valid row ref",
			req: &Request{
				Rows: map[string][]string{"user": {"gen('email')"}},
				Seed: []*Query{{Name: "q1", Row: "user", Query: "INSERT INTO users VALUES ($1)"}},
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

func TestValidate_DuplicateRunItemName(t *testing.T) {
	req := &Request{
		Run: []*RunItem{
			{Transaction: &Transaction{Name: "dup", Queries: []*Query{{Name: "q1", Type: QueryTypeExec, Query: "SELECT 1"}}}},
			{Transaction: &Transaction{Name: "dup", Queries: []*Query{{Name: "q2", Type: QueryTypeExec, Query: "SELECT 2"}}}},
		},
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `duplicate name "dup" in run`)
}

func TestRunItem_NilQueryAndTransaction(t *testing.T) {
	ri := &RunItem{}
	assert.Equal(t, "", ri.Name())
	assert.Nil(t, ri.AllQueries())
}

func TestQueryArgsUnmarshalYAML_InvalidKind(t *testing.T) {
	input := `args: 42`
	var out struct {
		Args QueryArgs `yaml:"args"`
	}
	err := yaml.Unmarshal([]byte(input), &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "args must be a list or map")
}

func TestValidate_PrintExpressionInValidation(t *testing.T) {
	req := &Request{
		Run: []*RunItem{
			{Query: &Query{
				Name:  "q1",
				Print: []PrintExpr{{Expr: "gen('email')"}},
			}},
		},
	}
	require.NoError(t, req.Validate())
}
