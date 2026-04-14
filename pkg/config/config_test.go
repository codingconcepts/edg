package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codingconcepts/edg/pkg/test"
	"gopkg.in/yaml.v3"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
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
	if err := yaml.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(req.Up) != 1 {
		t.Fatalf("expected 1 up query, got %d", len(req.Up))
	}
	if len(req.Seed) != 1 {
		t.Fatalf("expected 1 seed query, got %d", len(req.Seed))
	}
	if req.Seed[0].Name != "populate_table" {
		t.Errorf("seed query name = %q, want %q", req.Seed[0].Name, "populate_table")
	}
	if len(req.Seed[0].Args) != 1 {
		t.Fatalf("expected 1 seed arg, got %d", len(req.Seed[0].Args))
	}
	if len(req.Down) != 1 {
		t.Fatalf("expected 1 down query, got %d", len(req.Down))
	}
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
	if err := yaml.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(req.Deseed) != 2 {
		t.Fatalf("expected 2 deseed queries, got %d", len(req.Deseed))
	}
	if req.Deseed[0].Name != "truncate_items" {
		t.Errorf("deseed query name = %q, want %q", req.Deseed[0].Name, "truncate_items")
	}
	if req.Deseed[0].Type != QueryTypeExec {
		t.Errorf("deseed query type = %q, want %q", req.Deseed[0].Type, QueryTypeExec)
	}
}

func TestRequestParsesEmptySeed(t *testing.T) {
	input := `
up:
  - name: create_table
    query: CREATE TABLE t (id INT PRIMARY KEY)
`
	var req Request
	if err := yaml.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(req.Seed) != 0 {
		t.Errorf("expected 0 seed queries, got %d", len(req.Seed))
	}
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
	if err := yaml.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(req.Expectations) != 2 {
		t.Fatalf("expected 2 expectations, got %d", len(req.Expectations))
	}
	if req.Expectations[0] != "error_rate < 1" {
		t.Errorf("expectation[0] = %q, want %q", req.Expectations[0], "error_rate < 1")
	}
	if req.Expectations[1] != "check_balance.p99 < 100" {
		t.Errorf("expectation[1] = %q, want %q", req.Expectations[1], "check_balance.p99 < 100")
	}
}

func TestDurationUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"seconds", "wait: 5s", 5 * time.Second, false},
		{"milliseconds", "wait: 250ms", 250 * time.Millisecond, false},
		{"minutes", "wait: 2m", 2 * time.Minute, false},
		{"complex", "wait: 1m30s", 90 * time.Second, false},
		{"invalid", "wait: notaduration", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out struct {
				Wait Duration `yaml:"wait"`
			}
			err := yaml.Unmarshal([]byte(tt.input), &out)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if time.Duration(out.Wait) != tt.want {
				t.Errorf("got %v, want %v", time.Duration(out.Wait), tt.want)
			}
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
	if err := yaml.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(req.Seed) != 1 {
		t.Fatalf("expected 1 seed query, got %d", len(req.Seed))
	}
	q := req.Seed[0]
	if q.Type != QueryTypeExecBatch {
		t.Errorf("type = %q, want %q", q.Type, QueryTypeExecBatch)
	}
	// Count/Size are parsed as any from YAML, typically int.
	if q.Count != 100 {
		t.Errorf("count = %v, want 100", q.Count)
	}
	if q.Size != 50 {
		t.Errorf("size = %v, want 50", q.Size)
	}
}

func TestConfigSectionSeedValue(t *testing.T) {
	if ConfigSectionSeed != "seed" {
		t.Errorf("ConfigSectionSeed = %q, want %q", ConfigSectionSeed, "seed")
	}
}

func TestConfigSectionDeseedValue(t *testing.T) {
	if ConfigSectionDeseed != "deseed" {
		t.Errorf("ConfigSectionDeseed = %q, want %q", ConfigSectionDeseed, "deseed")
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
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if req.Globals["batch_size"] != 100 {
		t.Errorf("globals.batch_size = %v, want 100", req.Globals["batch_size"])
	}
	if len(req.Up) != 1 || req.Up[0].Name != "create_table" {
		t.Errorf("unexpected up queries: %v", req.Up)
	}
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
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if req.Globals["batch_size"] != 500 {
		t.Errorf("globals.batch_size = %v, want 500", req.Globals["batch_size"])
	}
	if req.Globals["workers"] != 4 {
		t.Errorf("globals.workers = %v, want 4", req.Globals["workers"])
	}
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
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(req.Up) != 2 {
		t.Fatalf("expected 2 up queries, got %d", len(req.Up))
	}
	if req.Up[0].Name != "create_users" {
		t.Errorf("up[0].name = %q, want %q", req.Up[0].Name, "create_users")
	}
	if req.Up[1].Name != "create_orders" {
		t.Errorf("up[1].name = %q, want %q", req.Up[1].Name, "create_orders")
	}
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
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(req.Run) != 2 {
		t.Fatalf("expected 2 run queries, got %d", len(req.Run))
	}
	if req.Run[0].Name() != "check_balance" {
		t.Errorf("run[0].name = %q, want %q", req.Run[0].Name(), "check_balance")
	}
	if req.Run[1].Name() != "make_transfer" {
		t.Errorf("run[1].name = %q, want %q", req.Run[1].Name(), "make_transfer")
	}
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
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if req.Globals["batch_size"] != 42 {
		t.Errorf("globals.batch_size = %v, want 42", req.Globals["batch_size"])
	}
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
	if err == nil {
		t.Fatal("expected circular include error, got nil")
	}
}

func TestLoadConfig_MissingInclude(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.yaml", `
globals: !include nonexistent.yaml
`)

	_, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	if err == nil {
		t.Fatal("expected error for missing include, got nil")
	}
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
	if err := yaml.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(req.Run) != 1 {
		t.Fatalf("expected 1 run item, got %d", len(req.Run))
	}
	if !req.Run[0].IsTransaction() {
		t.Fatal("expected run item to be a transaction")
	}

	tx := req.Run[0].Transaction
	if tx.Name != "make_transfer" {
		t.Errorf("transaction name = %q, want %q", tx.Name, "make_transfer")
	}
	if len(tx.Queries) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(tx.Queries))
	}
	if !tx.Queries[1].IsRollbackIf() {
		t.Fatal("expected second element to be a rollback_if")
	}
	if tx.Queries[1].RollbackIf != "ref_same('read_source').balance < 100" {
		t.Errorf("rollback_if = %q, want %q", tx.Queries[1].RollbackIf, "ref_same('read_source').balance < 100")
	}
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
	if err := yaml.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	for _, q := range req.Run[0].Transaction.Queries {
		if q.IsRollbackIf() {
			t.Error("unexpected rollback_if element")
		}
	}
}

func TestCompileRollbackIf_Valid(t *testing.T) {
	q := &Query{RollbackIf: "balance < 100"}

	envMap := map[string]any{"balance": 50}
	if err := q.CompileRollbackIf(envMap); err != nil {
		t.Fatalf("CompileRollbackIf failed: %v", err)
	}
	if q.CompiledRollbackIf == nil {
		t.Fatal("CompiledRollbackIf should not be nil")
	}
}

func TestCompileRollbackIf_Empty(t *testing.T) {
	q := &Query{}

	if err := q.CompileRollbackIf(map[string]any{}); err != nil {
		t.Fatalf("CompileRollbackIf failed: %v", err)
	}
	if q.CompiledRollbackIf != nil {
		t.Fatal("CompiledRollbackIf should be nil when rollback_if is empty")
	}
}

func TestCompileRollbackIf_Invalid(t *testing.T) {
	q := &Query{RollbackIf: "invalid +++"}

	err := q.CompileRollbackIf(map[string]any{})
	if err == nil {
		t.Fatal("expected error for invalid expression, got nil")
	}
}

func TestCompileRollbackIf_NonBoolReturnsError(t *testing.T) {
	q := &Query{RollbackIf: "42"}

	err := q.CompileRollbackIf(map[string]any{})
	if err == nil {
		t.Fatal("expected error for non-boolean expression, got nil")
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
	if err := yaml.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	tx := req.Run[0].Transaction
	if len(tx.Locals) != 2 {
		t.Fatalf("expected 2 locals, got %d", len(tx.Locals))
	}
	if tx.Locals["amount"] != "gen('number:1,100')" {
		t.Errorf("locals[amount] = %q, want %q", tx.Locals["amount"], "gen('number:1,100')")
	}
	if tx.Locals["fee"] != "const(5)" {
		t.Errorf("locals[fee] = %q, want %q", tx.Locals["fee"], "const(5)")
	}
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
	if err == nil {
		t.Fatal("expected error when local shadows query name, got nil")
	}
	if !strings.Contains(err.Error(), `local "debit" shadows query name`) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCompileLocals_Valid(t *testing.T) {
	envMap := map[string]any{
		"const": func(v any) any { return v },
	}
	tx := &Transaction{
		Locals: map[string]string{"amount": "const(42)"},
	}

	if err := tx.CompileLocals(envMap); err != nil {
		t.Fatalf("CompileLocals failed: %v", err)
	}
	if len(tx.CompiledLocals) != 1 {
		t.Fatalf("expected 1 compiled local, got %d", len(tx.CompiledLocals))
	}
	if tx.CompiledLocals["amount"] == nil {
		t.Fatal("compiled local 'amount' is nil")
	}
}

func TestCompileLocals_Invalid(t *testing.T) {
	tx := &Transaction{
		Locals: map[string]string{"bad": "invalid +++"},
	}

	err := tx.CompileLocals(map[string]any{})
	if err == nil {
		t.Fatal("expected error for invalid expression, got nil")
	}
}

func TestValidate_GenPatternValid(t *testing.T) {
	req := &Request{
		Run: []*RunItem{
			{Query: &Query{Name: "q1", Args: []string{"gen('email')", "gen('number:1,100')"}}},
		},
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_GenPatternInvalid(t *testing.T) {
	req := &Request{
		Run: []*RunItem{
			{Query: &Query{Name: "q1", Args: []string{"gen('notafunction')"}}},
		},
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for invalid gen pattern, got nil")
	}
	if !strings.Contains(err.Error(), "notafunction") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_GenPatternInRow(t *testing.T) {
	req := &Request{
		Rows: map[string][]string{
			"user": {"gen('email')", "gen('bogusxyz')"},
		},
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for invalid gen pattern in row, got nil")
	}
	if !strings.Contains(err.Error(), "bogusxyz") {
		t.Errorf("unexpected error: %v", err)
	}
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
	if err == nil {
		t.Fatal("expected error for invalid gen pattern in locals, got nil")
	}
	if !strings.Contains(err.Error(), "notreal") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_EnvPattern(t *testing.T) {
	cases := []struct {
		name    string
		env     map[string]string
		pattern string
		wantErr bool
	}{
		{
			name: "valid single quote",
			env: map[string]string{
				"ABC": "123",
			},
			pattern: `env('ABC')`,
			wantErr: false,
		},
		{
			name: "valid double quote",
			env: map[string]string{
				"ABC": "123",
			},
			pattern: `env("ABC")`,
			wantErr: false,
		},
		{
			name: "missing env var",
			env: map[string]string{
				"ABC": "123",
			},
			pattern: `env("DEF")`,
			wantErr: true,
		},
		{
			name: "mismatched quotes not matched",
			env: map[string]string{
				"ABC": "123",
			},
			pattern: `env('ABC")`,
			wantErr: false, // regex won't match, so no validation error (expr-lang catches syntax)
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test.CleanupEnv(t, "ABC")

			for k, v := range c.env {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("error setting env var for test: %v", err)
				}
			}

			req := &Request{
				Rows: map[string][]string{
					"value": {c.pattern},
				},
			}

			err := req.Validate()

			if err != nil {
				if !c.wantErr {
					t.Fatalf("expected no error but got: %v", err)
				}
				return
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
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if req.Globals["batch_size"] != 100 {
		t.Errorf("globals.batch_size = %v, want 100", req.Globals["batch_size"])
	}
	if len(req.Up) != 1 || req.Up[0].Name != "create_table" {
		t.Errorf("unexpected up: %v", req.Up)
	}
	if len(req.Down) != 1 || req.Down[0].Name != "drop_table" {
		t.Errorf("unexpected down: %v", req.Down)
	}
}
