package pkg

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func testEnv(data map[string][]map[string]any) *Env {
	env := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
	}
	for name, rows := range data {
		env.env[name] = rows
	}
	return env
}

func sampleRows() []map[string]any {
	return []map[string]any{
		{"id": 1, "name": "a"},
		{"id": 2, "name": "b"},
		{"id": 3, "name": "c"},
	}
}

func benchEnv(dataSize int) *Env {
	rows := make([]map[string]any, dataSize)
	for i := range rows {
		rows[i] = map[string]any{"id": i, "name": fmt.Sprintf("item_%d", i)}
	}
	env := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		nurandC:   map[int]int{},
		env:       map[string]any{},
		request:   &Request{},
	}
	env.env["items"] = rows
	return env
}

func TestExpr(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = constant
	env.env["expr"] = constant
	env.env["warehouses"] = 5

	q := &Query{
		Args: []string{"expr(warehouses * 10)", "expr(warehouses + 1)"},
	}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, err := q.GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	args := argSets[0]
	if args[0] != 50 {
		t.Errorf("expr(warehouses * 10) = %v, want 50", args[0])
	}
	if args[1] != 6 {
		t.Errorf("expr(warehouses + 1) = %v, want 6", args[1])
	}
}

func TestBareArithmetic(t *testing.T) {
	env := testEnv(nil)
	env.env["orders"] = 30000
	env.env["districts"] = 10

	q := &Query{
		Args: []string{"orders / districts"},
	}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, err := q.GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	got, ok := argSets[0][0].(float64)
	if !ok {
		t.Fatalf("orders / districts = %v (%T), want float64", argSets[0][0], argSets[0][0])
	}
	if got != 3000 {
		t.Errorf("orders / districts = %v, want 3000", got)
	}
}

func TestSeedArgsCompiled(t *testing.T) {
	env := testEnv(nil)
	env.env["const"] = constant
	env.env["items"] = 100

	seedQuery := &Query{
		Name: "populate_items",
		Args: []string{"items"},
	}
	if err := seedQuery.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs for seed query failed: %v", err)
	}

	if len(seedQuery.CompiledArgs) != 1 {
		t.Fatalf("expected 1 compiled arg, got %d", len(seedQuery.CompiledArgs))
	}

	argSets, err := seedQuery.GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if len(argSets) != 1 {
		t.Fatalf("expected 1 arg set, got %d", len(argSets))
	}
	if argSets[0][0] != 100 {
		t.Errorf("seed arg = %v, want 100", argSets[0][0])
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

func TestExpressions(t *testing.T) {
	req := &Request{
		Globals: map[string]any{
			"customers": 30000,
			"districts": 10,
		},
		Expressions: map[string]string{
			"cust_per_district": "customers / districts",
		},
		Run: []*Query{
			{Args: []string{"cust_per_district()"}},
		},
	}

	env, err := NewEnv(nil, req)
	if err != nil {
		t.Fatalf("NewEnv failed: %v", err)
	}

	argSets, err := req.Run[0].GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	got, ok := argSets[0][0].(float64)
	if !ok {
		t.Fatalf("cust_per_district() = %v (%T), want float64", argSets[0][0], argSets[0][0])
	}
	if got != 3000 {
		t.Errorf("cust_per_district() = %v, want 3000", got)
	}
}

func TestExpressions_WithArgs(t *testing.T) {
	req := &Request{
		Globals: map[string]any{
			"customers": 30000,
		},
		Expressions: map[string]string{
			"divide": "customers / args[0]",
		},
		Run: []*Query{
			{Args: []string{"divide(10)"}},
		},
	}

	env, err := NewEnv(nil, req)
	if err != nil {
		t.Fatalf("NewEnv failed: %v", err)
	}

	argSets, err := req.Run[0].GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	got, ok := argSets[0][0].(float64)
	if !ok {
		t.Fatalf("divide(10) = %v (%T), want float64", argSets[0][0], argSets[0][0])
	}
	if got != 3000 {
		t.Errorf("divide(10) = %v, want 3000", got)
	}
}

func TestExpressions_InvalidBody(t *testing.T) {
	req := &Request{
		Expressions: map[string]string{
			"bad": "undefined_var +",
		},
	}

	_, err := NewEnv(nil, req)
	if err == nil {
		t.Fatal("expected error for invalid expression, got nil")
	}
}

func TestGenerateArgs_Batch(t *testing.T) {
	env := testEnv(nil)
	env.env["batch"] = batch
	env.env["const"] = constant
	env.env["items"] = 30

	q := &Query{Args: []string{"batch(items / 10)", "const(10)"}}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	argSets, err := q.GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if len(argSets) != 3 {
		t.Fatalf("expected 3 arg sets, got %d", len(argSets))
	}

	for i, args := range argSets {
		if args[0] != i {
			t.Errorf("arg set %d: args[0] = %v, want %d", i, args[0], i)
		}
		if args[1] != 10 {
			t.Errorf("arg set %d: args[1] = %v, want 10", i, args[1])
		}
	}
}

func TestSetEnv(t *testing.T) {
	env := testEnv(nil)
	data := sampleRows()

	env.SetEnv("test_data", data)

	raw, ok := env.env["test_data"]
	if !ok {
		t.Fatal("SetEnv did not set the key")
	}

	got := raw.([]map[string]any)
	if len(got) != len(data) {
		t.Errorf("SetEnv stored %d rows, want %d", len(got), len(data))
	}
}

func TestPickWeighted(t *testing.T) {
	queries := []*Query{
		{Name: "heavy"},
		{Name: "light"},
	}
	env := &Env{
		request: &Request{
			Run: queries,
			RunWeights: map[string]int{
				"heavy": 90,
				"light": 10,
			},
		},
	}

	counts := map[string]int{}
	for range 1000 {
		q := env.pickWeighted()
		if q == nil {
			t.Fatal("pickWeighted returned nil")
		}
		counts[q.Name]++
	}

	// With 90/10 weights over 1000 iterations, "heavy" should
	// appear significantly more than "light".
	if counts["heavy"] < 800 {
		t.Errorf("heavy picked %d/1000 times, expected ~900", counts["heavy"])
	}
	if counts["light"] < 50 {
		t.Errorf("light picked %d/1000 times, expected ~100", counts["light"])
	}
}

func TestPickWeighted_NoWeights(t *testing.T) {
	env := &Env{
		request: &Request{
			Run:        []*Query{{Name: "a"}},
			RunWeights: nil,
		},
	}

	if q := env.pickWeighted(); q != nil {
		t.Errorf("pickWeighted with no weights returned %v, want nil", q.Name)
	}
}

func TestPickWeighted_SkipsUnweightedQueries(t *testing.T) {
	env := &Env{
		request: &Request{
			Run: []*Query{{Name: "weighted"}, {Name: "unweighted"}},
			RunWeights: map[string]int{
				"weighted": 100,
			},
		},
	}

	for range 100 {
		q := env.pickWeighted()
		if q == nil {
			t.Fatal("pickWeighted returned nil")
		}
		if q.Name != "weighted" {
			t.Errorf("pickWeighted returned %q, want only 'weighted'", q.Name)
		}
	}
}

func TestClearOneCache(t *testing.T) {
	env := testEnv(nil)
	env.oneCache["test"] = "value"

	env.clearOneCache()

	if len(env.oneCache) != 0 {
		t.Errorf("clearOneCache left %d entries", len(env.oneCache))
	}
}

func TestResetUniqIndex(t *testing.T) {
	env := testEnv(nil)
	env.uniqIndex = 5

	env.resetUniqIndex()

	if env.uniqIndex != 0 {
		t.Errorf("resetUniqIndex left index at %d", env.uniqIndex)
	}
}

func TestRunIteration_NoWeights(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO t1").WillReturnResult(driver.ResultNoRows)
	mock.ExpectExec("INSERT INTO t2").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &Request{
			Run: []*Query{
				{Name: "q1", Type: QueryTypeExec, Query: "INSERT INTO t1 VALUES (1)"},
				{Name: "q2", Type: QueryTypeExec, Query: "INSERT INTO t2 VALUES (2)"},
			},
		},
	}

	if err := env.RunIteration(context.Background()); err != nil {
		t.Fatalf("RunIteration error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunIteration_WithWeights(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	// Only one query should run per call.
	mock.ExpectExec("INSERT").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &Request{
			Run: []*Query{
				{Name: "q1", Type: QueryTypeExec, Query: "INSERT INTO t1 VALUES (1)"},
				{Name: "q2", Type: QueryTypeExec, Query: "INSERT INTO t2 VALUES (2)"},
			},
			RunWeights: map[string]int{
				"q1": 50,
				"q2": 50,
			},
		},
	}

	if err := env.RunIteration(context.Background()); err != nil {
		t.Fatalf("RunIteration error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestInitFrom(t *testing.T) {
	sourceRows := []map[string]any{
		{"id": 1, "name": "a"},
		{"id": 2, "name": "b"},
	}

	source := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{"load_items": sourceRows},
		request:   &Request{},
	}

	target := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &Request{
			Init: []*Query{
				{Name: "load_items", Type: QueryTypeQuery},
			},
		},
	}

	target.InitFrom(source)

	raw, ok := target.env["load_items"]
	if !ok {
		t.Fatal("InitFrom did not copy data")
	}
	copied := raw.([]map[string]any)
	if len(copied) != 2 {
		t.Fatalf("InitFrom copied %d rows, want 2", len(copied))
	}
	if copied[0]["id"] != 1 {
		t.Errorf("copied row 0 id = %v, want 1", copied[0]["id"])
	}
}

func TestInitFrom_SkipsExecQueries(t *testing.T) {
	source := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &Request{},
	}

	target := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &Request{
			Init: []*Query{
				{Name: "setup", Type: QueryTypeExec},
			},
		},
	}

	target.InitFrom(source)

	if _, ok := target.env["setup"]; ok {
		t.Error("InitFrom should skip exec-type queries")
	}
}

func TestInitFrom_IndependentCopies(t *testing.T) {
	sourceRows := []map[string]any{
		{"id": 1},
		{"id": 2},
		{"id": 3},
	}

	source := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{"items": sourceRows},
		request:   &Request{},
	}

	target := &Env{
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request: &Request{
			Init: []*Query{
				{Name: "items", Type: QueryTypeQuery},
			},
		},
	}

	target.InitFrom(source)

	// Modifying the target's copy should not affect the source.
	targetData := target.env["items"].([]map[string]any)
	targetData[0] = map[string]any{"id": 999}

	if sourceRows[0]["id"] != 1 {
		t.Error("InitFrom did not create an independent copy; source was mutated")
	}
}

func TestNewEnv_GlobalShadowsBuiltin(t *testing.T) {
	req := &Request{
		Globals: map[string]any{
			"ref_rand": "oops",
		},
	}

	_, err := NewEnv(nil, req)
	if err == nil {
		t.Fatal("expected error when global shadows a built-in, got nil")
	}

	want := `global "ref_rand" shadows a built-in function`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestReference_LoadedIntoEnv(t *testing.T) {
	req := &Request{
		Reference: map[string][]map[string]any{
			"regions": {
				{"name": "eu", "region": "eu-west-2"},
				{"name": "us", "region": "us-east-1"},
			},
		},
	}

	env, err := NewEnv(nil, req)
	if err != nil {
		t.Fatalf("NewEnv failed: %v", err)
	}

	raw, ok := env.env["regions"]
	if !ok {
		t.Fatal("reference data not loaded into env")
	}

	rows := raw.([]map[string]any)
	if len(rows) != 2 {
		t.Fatalf("loaded %d rows, want 2", len(rows))
	}
	if rows[0]["name"] != "eu" {
		t.Errorf("row 0 name = %v, want eu", rows[0]["name"])
	}
	if rows[1]["region"] != "us-east-1" {
		t.Errorf("row 1 region = %v, want us-east-1", rows[1]["region"])
	}
}

func TestReference_IndependentCopies(t *testing.T) {
	req := &Request{
		Reference: map[string][]map[string]any{
			"items": {
				{"id": 1},
				{"id": 2},
			},
		},
	}

	env1, err := NewEnv(nil, req)
	if err != nil {
		t.Fatalf("NewEnv env1 failed: %v", err)
	}
	env2, err := NewEnv(nil, req)
	if err != nil {
		t.Fatalf("NewEnv env2 failed: %v", err)
	}

	// Mutating env1's copy should not affect env2.
	data1 := env1.env["items"].([]map[string]any)
	data1[0] = map[string]any{"id": 999}

	data2 := env2.env["items"].([]map[string]any)
	if data2[0]["id"] != 1 {
		t.Error("reference data is shared between envs; expected independent copies")
	}
}

func TestReference_NilIsNoOp(t *testing.T) {
	req := &Request{}

	env, err := NewEnv(nil, req)
	if err != nil {
		t.Fatalf("NewEnv failed: %v", err)
	}

	// Should not panic or add unexpected keys.
	if _, ok := env.env["regions"]; ok {
		t.Error("unexpected 'regions' key in env with nil reference")
	}
}

func TestReference_RefRand(t *testing.T) {
	req := &Request{
		Reference: map[string][]map[string]any{
			"colors": {
				{"name": "red"},
				{"name": "blue"},
				{"name": "green"},
			},
		},
		Run: []*Query{
			{Args: []string{"ref_rand('colors').name"}},
		},
	}

	env, err := NewEnv(nil, req)
	if err != nil {
		t.Fatalf("NewEnv failed: %v", err)
	}

	argSets, err := req.Run[0].GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	got, ok := argSets[0][0].(string)
	if !ok {
		t.Fatalf("ref_rand('colors').name = %v (%T), want string", argSets[0][0], argSets[0][0])
	}

	valid := got == "red" || got == "blue" || got == "green"
	if !valid {
		t.Errorf("ref_rand('colors').name = %q, want one of red/blue/green", got)
	}
}

func TestRunSection_Exec(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &Request{},
	}

	queries := []*Query{
		{Name: "create_t", Type: QueryTypeExec, Query: "CREATE TABLE t (id INT)"},
	}

	if err := env.runSection(context.Background(), queries, ConfigSectionUp); err != nil {
		t.Fatalf("runSection error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunSection_SeedUsesBindParams(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	// Non-batch seed queries use bind params, same as run.
	mock.ExpectExec("INSERT INTO items SELECT generate_series").
		WithArgs(100).
		WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env: map[string]any{
			"const": constant,
			"items": 100,
		},
		request: &Request{},
	}

	q := &Query{
		Name:  "seed_items",
		Type:  QueryTypeExec,
		Query: "INSERT INTO items SELECT generate_series(1, $1)",
		Args:  []string{"items"},
	}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	if err := env.runSection(context.Background(), []*Query{q}, ConfigSectionSeed); err != nil {
		t.Fatalf("runSection error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunSection_RunSectionPassesArgs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO orders").
		WithArgs(42).
		WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env: map[string]any{
			"const": constant,
		},
		request: &Request{},
	}

	q := &Query{
		Name:  "insert_order",
		Type:  QueryTypeExec,
		Query: "INSERT INTO orders VALUES ($1)",
		Args:  []string{"const(42)"},
	}
	if err := q.CompileArgs(env); err != nil {
		t.Fatalf("CompileArgs failed: %v", err)
	}

	if err := env.runSection(context.Background(), []*Query{q}, ConfigSectionRun); err != nil {
		t.Fatalf("runSection error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunSection_WaitRespectsContextCancel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT").WillReturnResult(driver.ResultNoRows)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &Request{},
	}

	q := &Query{
		Name:  "slow",
		Type:  QueryTypeExec,
		Query: "INSERT INTO t VALUES (1)",
		Wait:  Duration(10 * time.Second),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err = env.runSection(ctx, []*Query{q}, ConfigSectionRun)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runSection error = %v, want context.Canceled", err)
	}
}

func TestRunSection_QueryStoresResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2),
	)

	env := &Env{
		db:        db,
		oneCache:  map[string]any{},
		permCache: map[string]any{},
		env:       map[string]any{},
		request:   &Request{},
	}

	queries := []*Query{
		{Name: "items", Type: QueryTypeQuery, Query: "SELECT id FROM items"},
	}

	if err := env.runSection(context.Background(), queries, ConfigSectionInit); err != nil {
		t.Fatalf("runSection error: %v", err)
	}

	data, ok := env.env["items"].([]map[string]any)
	if !ok {
		t.Fatal("runSection did not store query results")
	}
	if len(data) != 2 {
		t.Errorf("stored %d rows, want 2", len(data))
	}
}

func TestRow_ExpandsIntoArgs(t *testing.T) {
	req := &Request{
		Rows: map[string][]string{
			"customer": {"gen('email')", "gen('name')"},
		},
		Run: []*Query{
			{Name: "insert_customer", Row: "customer", Query: "INSERT INTO customer (email, name) VALUES ($1, $2)"},
		},
	}

	env, err := NewEnv(nil, req)
	if err != nil {
		t.Fatalf("NewEnv failed: %v", err)
	}

	argSets, err := req.Run[0].GenerateArgs(env)
	if err != nil {
		t.Fatalf("GenerateArgs failed: %v", err)
	}

	if len(argSets) != 1 {
		t.Fatalf("expected 1 arg set, got %d", len(argSets))
	}
	if len(argSets[0]) != 2 {
		t.Fatalf("expected 2 args, got %d", len(argSets[0]))
	}

	email, ok := argSets[0][0].(string)
	if !ok || email == "" {
		t.Errorf("arg 0 = %v (%T), want non-empty string", argSets[0][0], argSets[0][0])
	}
	name, ok := argSets[0][1].(string)
	if !ok || name == "" {
		t.Errorf("arg 1 = %v (%T), want non-empty string", argSets[0][1], argSets[0][1])
	}
}

func TestRow_UsedAcrossSections(t *testing.T) {
	req := &Request{
		Rows: map[string][]string{
			"customer": {"gen('email')"},
		},
		Seed: []*Query{
			{Name: "seed_customer", Row: "customer", Query: "INSERT INTO customer (email) VALUES ($1)"},
		},
		Run: []*Query{
			{Name: "insert_customer", Row: "customer", Query: "INSERT INTO customer (email) VALUES ($1)"},
		},
	}

	env, err := NewEnv(nil, req)
	if err != nil {
		t.Fatalf("NewEnv failed: %v", err)
	}

	// Both queries should have compiled args from the row.
	for _, q := range []*Query{req.Seed[0], req.Run[0]} {
		if len(q.CompiledArgs) != 1 {
			t.Errorf("query %s: expected 1 compiled arg, got %d", q.Name, len(q.CompiledArgs))
		}

		argSets, err := q.GenerateArgs(env)
		if err != nil {
			t.Fatalf("GenerateArgs for %s failed: %v", q.Name, err)
		}
		if len(argSets[0]) != 1 {
			t.Errorf("query %s: expected 1 arg, got %d", q.Name, len(argSets[0]))
		}
	}
}

func TestRow_UnknownRowName(t *testing.T) {
	req := &Request{
		Run: []*Query{
			{Name: "bad", Row: "nonexistent"},
		},
	}

	_, err := NewEnv(nil, req)
	if err == nil {
		t.Fatal("expected error for unknown row name, got nil")
	}
}

func TestRow_MutuallyExclusiveWithArgs(t *testing.T) {
	req := &Request{
		Rows: map[string][]string{
			"customer": {"gen('email')"},
		},
		Run: []*Query{
			{Name: "bad", Row: "customer", Args: []string{"gen('name')"}},
		},
	}

	_, err := NewEnv(nil, req)
	if err == nil {
		t.Fatal("expected error when both row and args are set, got nil")
	}
}

func BenchmarkPickWeighted(b *testing.B) {
	cases := []struct {
		name  string
		count int
	}{
		{"queries_2", 2},
		{"queries_5", 5},
		{"queries_10", 10},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			queries := make([]*Query, tc.count)
			weights := make(map[string]int, tc.count)
			for i := range tc.count {
				name := fmt.Sprintf("q%d", i)
				queries[i] = &Query{Name: name}
				weights[name] = i + 1
			}
			env := &Env{
				request: &Request{
					Run:        queries,
					RunWeights: weights,
				},
			}
			b.ResetTimer()
			for range b.N {
				env.pickWeighted()
			}
		})
	}
}
