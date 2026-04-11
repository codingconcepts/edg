package env

import (
	"context"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/random"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/sijms/go-ora/v2"
	"gopkg.in/yaml.v3"
)

//go:embed testdata/crdb.yaml
var crdbConfig []byte

//go:embed testdata/mysql.yaml
var mysqlConfig []byte

//go:embed testdata/oracle.yaml
var oracleConfig []byte

//go:embed testdata/mssql.yaml
var mssqlConfig []byte

const runIterations = 5

var (
	dbTests    *bool
	rngSeed    *uint64
	db         *sql.DB
	driverName string

	allTables = []string{
		"test_ref_source", "test_scalars", "test_uuids", "test_numbers",
		"test_sets", "test_json", "test_geo", "test_time",
		"test_distributions", "test_refs", "test_ref_diff", "test_ref_perm",
		"test_binary", "test_arrays",
		"test_gen_batch", "test_batch", "test_ref_each",
		"test_agg",
	}
)

func TestMain(m *testing.M) {
	dbTests = flag.Bool("db", false, "run database integration tests")
	rngSeed = flag.Uint64("rng-seed", 0, "PRNG seed for deterministic output")
	flag.Parse()

	var rngSeedSet bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "rng-seed" {
			rngSeedSet = true
		}
	})
	if rngSeedSet {
		random.Seed(*rngSeed)
	}

	if *dbTests {
		setupDatabase()
	}

	code := m.Run()

	if *dbTests {
		teardownDatabase()
	}

	os.Exit(code)
}

func setupDatabase() {
	connStr, ok := os.LookupEnv("URL")
	if !ok {
		log.Fatal("missing URL env var")
	}

	driverName, ok = os.LookupEnv("DRIVER")
	if !ok {
		log.Fatal("missing DRIVER env var")
	}

	var err error

	// Retry connection for up to 30 seconds to allow the database to start.
	for range 30 {
		db, err = sql.Open(driverName, connStr)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			return
		}
		time.Sleep(time.Second)
	}

	log.Fatalf("connecting to database after retries: %v", err)
}

func teardownDatabase() {
	if db != nil {
		db.Close()
	}
}

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if !*dbTests {
		t.Skip("skipping database test (use -db flag to enable)")
	}
}

func loadConfig(t *testing.T, data []byte) *config.Request {
	t.Helper()
	var req config.Request
	if err := yaml.Unmarshal(data, &req); err != nil {
		t.Fatalf("parsing config: %v", err)
	}
	return &req
}

func rowCount(t *testing.T, table string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count); err != nil {
		t.Fatalf("counting %s: %v", table, err)
	}
	return count
}

func tableExists(t *testing.T, query, table string) bool {
	t.Helper()
	var count int
	if err := db.QueryRow(query, table).Scan(&count); err != nil {
		t.Fatalf("checking table %s: %v", table, err)
	}
	return count > 0
}

func TestIntegration_CockroachDB(t *testing.T) {
	skipIfNoDB(t)
	if driverName != "pgx" {
		t.Skip("skipping CockroachDB test (DRIVER != pgx)")
	}

	queries := map[string]string{
		"table_exists":  "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = $1",
		"scalars":       "SELECT const_val, global_val, expr_val, gen_val, regex_val, tmpl_val, cond_val, coal_val, expr_fn_val FROM test_scalars",
		"uuids":         "SELECT v1, v4, v6, v7 FROM test_uuids",
		"numbers":       "SELECT float_val, uniform_val, norm_val, norm_f_val, zipf_val FROM test_numbers",
		"sets":          "SELECT rand_val, weighted_val, normal_val, exp_val, lognorm_val, zipfian_val FROM test_sets",
		"json":          "SELECT obj::STRING, arr::STRING FROM test_json",
		"geo":           "SELECT lat, lon, wkt FROM test_geo",
		"time":          "SELECT ts, dur, date_str, offset_ts, time_val, timez_val FROM test_time",
		"distributions": "SELECT nu_val, nu_vals, norm_vals, exp_val, lognorm_val, exp_int_val, lognorm_int_val FROM test_distributions",
		"refs":          "SELECT rand_id, same_id, same_name, ref_n_ids, weighted_ids FROM test_refs",
		"ref_diff":      "SELECT id1, id2 FROM test_ref_diff",
		"ref_perm":      "SELECT COUNT(DISTINCT perm_id) FROM test_ref_perm",
		"binary":        "SELECT bytes_val, bit_val, varbit_val, inet_val FROM test_binary",
		"arrays":        "SELECT arr_val FROM test_arrays",
		"batch_vals":    "SELECT val FROM test_batch ORDER BY val",
		"agg":           "SELECT sum_val, avg_val, min_val, max_val, count_val, distinct_val FROM test_agg",
	}

	runIntegrationTests(t, crdbConfig, queries)
}

func TestIntegration_MySQL(t *testing.T) {
	skipIfNoDB(t)
	if driverName != "mysql" {
		t.Skip("skipping MySQL test (DRIVER != mysql)")
	}

	queries := map[string]string{
		"table_exists":  "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = DATABASE()",
		"scalars":       "SELECT const_val, global_val, expr_val, gen_val, regex_val, tmpl_val, cond_val, coal_val, expr_fn_val FROM test_scalars",
		"uuids":         "SELECT v1, v4, v6, v7 FROM test_uuids",
		"numbers":       "SELECT float_val, uniform_val, norm_val, norm_f_val, zipf_val FROM test_numbers",
		"sets":          "SELECT rand_val, weighted_val, normal_val, exp_val, lognorm_val, zipfian_val FROM test_sets",
		"json":          "SELECT CAST(obj AS CHAR), CAST(arr AS CHAR) FROM test_json",
		"geo":           "SELECT lat, lon, wkt FROM test_geo",
		"time":          "SELECT ts, dur, date_str, offset_ts, time_val, timez_val FROM test_time",
		"distributions": "SELECT nu_val, nu_vals, norm_vals, exp_val, lognorm_val, exp_int_val, lognorm_int_val FROM test_distributions",
		"refs":          "SELECT rand_id, same_id, same_name, ref_n_ids, weighted_ids FROM test_refs",
		"ref_diff":      "SELECT id1, id2 FROM test_ref_diff",
		"ref_perm":      "SELECT COUNT(DISTINCT perm_id) FROM test_ref_perm",
		"binary":        "SELECT bytes_val, bit_val, varbit_val, inet_val FROM test_binary",
		"arrays":        "SELECT arr_val FROM test_arrays",
		"batch_vals":    "SELECT val FROM test_batch ORDER BY val",
		"agg":           "SELECT sum_val, avg_val, min_val, max_val, count_val, distinct_val FROM test_agg",
	}

	runIntegrationTests(t, mysqlConfig, queries)
}

func TestIntegration_Oracle(t *testing.T) {
	skipIfNoDB(t)
	if driverName != "oracle" {
		t.Skip("skipping Oracle test (DRIVER != oracle)")
	}

	queries := map[string]string{
		"table_exists":  "SELECT COUNT(*) FROM user_tables WHERE table_name = UPPER(:1)",
		"scalars":       "SELECT const_val, global_val, expr_val, gen_val, regex_val, tmpl_val, cond_val, coal_val, expr_fn_val FROM test_scalars",
		"uuids":         "SELECT v1, v4, v6, v7 FROM test_uuids",
		"numbers":       "SELECT float_val, uniform_val, norm_val, norm_f_val, zipf_val FROM test_numbers",
		"sets":          "SELECT rand_val, weighted_val, normal_val, exp_val, lognorm_val, zipfian_val FROM test_sets",
		"json":          "SELECT obj, arr FROM test_json",
		"geo":           "SELECT lat, lon, wkt FROM test_geo",
		"time":          "SELECT ts, dur, date_str, offset_ts, time_val, timez_val FROM test_time",
		"distributions": "SELECT nu_val, nu_vals, norm_vals, exp_val, lognorm_val, exp_int_val, lognorm_int_val FROM test_distributions",
		"refs":          "SELECT rand_id, same_id, same_name, ref_n_ids, weighted_ids FROM test_refs",
		"ref_diff":      "SELECT id1, id2 FROM test_ref_diff",
		"ref_perm":      "SELECT COUNT(DISTINCT perm_id) FROM test_ref_perm",
		"binary":        "SELECT bytes_val, bit_val, varbit_val, inet_val FROM test_binary",
		"arrays":        "SELECT arr_val FROM test_arrays",
		"batch_vals":    "SELECT val FROM test_batch ORDER BY val",
		"agg":           "SELECT sum_val, avg_val, min_val, max_val, count_val, distinct_val FROM test_agg",
	}

	runIntegrationTests(t, oracleConfig, queries)
}

func TestIntegration_MSSQL(t *testing.T) {
	skipIfNoDB(t)
	if driverName != "mssql" {
		t.Skip("skipping MSSQL test (DRIVER != mssql)")
	}

	queries := map[string]string{
		"table_exists":  "SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = @p1",
		"scalars":       "SELECT const_val, global_val, expr_val, gen_val, regex_val, tmpl_val, cond_val, coal_val, expr_fn_val FROM test_scalars",
		"uuids":         "SELECT v1, v4, v6, v7 FROM test_uuids",
		"numbers":       "SELECT float_val, uniform_val, norm_val, norm_f_val, zipf_val FROM test_numbers",
		"sets":          "SELECT rand_val, weighted_val, normal_val, exp_val, lognorm_val, zipfian_val FROM test_sets",
		"json":          "SELECT obj, arr FROM test_json",
		"geo":           "SELECT lat, lon, wkt FROM test_geo",
		"time":          "SELECT ts, dur, date_str, offset_ts, time_val, timez_val FROM test_time",
		"distributions": "SELECT nu_val, nu_vals, norm_vals, exp_val, lognorm_val, exp_int_val, lognorm_int_val FROM test_distributions",
		"refs":          "SELECT rand_id, same_id, same_name, ref_n_ids, weighted_ids FROM test_refs",
		"ref_diff":      "SELECT id1, id2 FROM test_ref_diff",
		"ref_perm":      "SELECT COUNT(DISTINCT perm_id) FROM test_ref_perm",
		"binary":        "SELECT bytes_val, bit_val, varbit_val, inet_val FROM test_binary",
		"arrays":        "SELECT arr_val FROM test_arrays",
		"batch_vals":    "SELECT val FROM test_batch ORDER BY val",
		"agg":           "SELECT sum_val, avg_val, min_val, max_val, count_val, distinct_val FROM test_agg",
	}

	runIntegrationTests(t, mssqlConfig, queries)
}

func runIntegrationTests(t *testing.T, config []byte, queries map[string]string) {
	skipIfNoDB(t)

	req := loadConfig(t, config)
	ctx := context.Background()

	env, err := NewEnv(db, driverName, req)
	if err != nil {
		t.Fatalf("creating env: %v", err)
	}

	// Tear down tables when the test finishes.
	t.Cleanup(func() {
		env.Down(ctx)
	})

	t.Run("up", func(t *testing.T) { testUp(t, env, ctx, queries) })
	t.Run("seed", func(t *testing.T) { testSeed(t, env, ctx, queries) })
	t.Run("init", func(t *testing.T) { testInit(t, env, ctx) })
	t.Run("run", func(t *testing.T) { testRun(t, env, ctx, queries) })
	t.Run("deseed", func(t *testing.T) { testDeseed(t, env, ctx) })
	t.Run("down", func(t *testing.T) { testDown(t, env, ctx, queries) })
}

func testUp(t *testing.T, env *Env, ctx context.Context, queries map[string]string) {
	if err := env.Up(ctx); err != nil {
		t.Fatalf("up: %v", err)
	}
	for _, table := range allTables {
		if !tableExists(t, queries["table_exists"], table) {
			t.Errorf("table %s was not created", table)
		}
	}
}

func testSeed(t *testing.T, env *Env, ctx context.Context, queries map[string]string) {
	if err := env.Seed(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}

	t.Run("ref_source", func(t *testing.T) { testSeedRefSource(t) })
	t.Run("gen_batch", func(t *testing.T) { testSeedGenBatch(t) })
	t.Run("batch", func(t *testing.T) { testSeedBatch(t, queries) })
	t.Run("ref_each", func(t *testing.T) { testSeedRefEach(t) })
}

func testSeedRefSource(t *testing.T) {
	if got := rowCount(t, "test_ref_source"); got != 20 {
		t.Errorf("test_ref_source rows = %d, want 20", got)
	}
}

func testSeedGenBatch(t *testing.T) {
	if got := rowCount(t, "test_gen_batch"); got != 10 {
		t.Errorf("test_gen_batch rows = %d, want 10", got)
	}
}

func testSeedBatch(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_batch"); got != 5 {
		t.Errorf("test_batch rows = %d, want 5", got)
	}

	// Values should be 0..4.
	rows, err := db.Query(queries["batch_vals"])
	if err != nil {
		t.Fatalf("querying test_batch: %v", err)
	}
	defer rows.Close()
	var vals []int
	for rows.Next() {
		var v int
		rows.Scan(&v)
		vals = append(vals, v)
	}
	for i, v := range vals {
		if v != i {
			t.Errorf("test_batch[%d] = %d, want %d", i, v, i)
		}
	}
}

func testSeedRefEach(t *testing.T) {
	if got := rowCount(t, "test_ref_each"); got != 5 {
		t.Errorf("test_ref_each rows = %d, want 5", got)
	}
}

func testInit(t *testing.T, env *Env, ctx context.Context) {
	if err := env.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}
}

func testRun(t *testing.T, env *Env, ctx context.Context, queries map[string]string) {
	for i := range runIterations {
		if err := env.RunIteration(ctx); err != nil {
			t.Fatalf("run iteration %d: %v", i, err)
		}
	}

	t.Run("scalars", func(t *testing.T) { testRunScalars(t, queries) })
	t.Run("uuids", func(t *testing.T) { testRunUUIDs(t, queries) })
	t.Run("numbers", func(t *testing.T) { testRunNumbers(t, queries) })
	t.Run("sets", func(t *testing.T) { testRunSets(t, queries) })
	t.Run("json", func(t *testing.T) { testRunJSON(t, queries) })
	t.Run("geo", func(t *testing.T) { testRunGeo(t, queries) })
	t.Run("time", func(t *testing.T) { testRunTime(t, queries) })
	t.Run("distributions", func(t *testing.T) { testRunDistributions(t, queries) })
	t.Run("binary", func(t *testing.T) { testRunBinary(t, queries) })
	t.Run("arrays", func(t *testing.T) { testRunArrays(t, queries) })
	t.Run("refs", func(t *testing.T) { testRunRefs(t, queries) })
	t.Run("ref_diff", func(t *testing.T) { testRunRefDiff(t, queries) })
	t.Run("ref_perm", func(t *testing.T) { testRunRefPerm(t, queries) })
	t.Run("agg", func(t *testing.T) { testRunAgg(t, queries) })
}

func testRunScalars(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_scalars"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	rows, err := db.Query(queries["scalars"])
	if err != nil {
		t.Fatalf("querying: %v", err)
	}
	defer rows.Close()

	regexPat := regexp.MustCompile(`^[A-Z]{3}-[0-9]{4}$`)
	tmplPat := regexp.MustCompile(`^ITEM-\d{5}$`)

	for rows.Next() {
		var constVal, globalVal, exprVal int
		var genVal, regexVal, tmplVal, condVal, coalVal string
		var exprFnVal float64
		rows.Scan(&constVal, &globalVal, &exprVal, &genVal, &regexVal, &tmplVal, &condVal, &coalVal, &exprFnVal)

		if constVal != 42 {
			t.Errorf("const_val = %d, want 42", constVal)
		}
		if globalVal != 42 {
			t.Errorf("global_val = %d, want 42", globalVal)
		}
		if exprVal != 84 {
			t.Errorf("expr_val = %d, want 84", exprVal)
		}
		if genVal == "" {
			t.Error("gen_val is empty")
		}
		if !regexPat.MatchString(regexVal) {
			t.Errorf("regex_val %q does not match pattern", regexVal)
		}
		if !tmplPat.MatchString(tmplVal) {
			t.Errorf("tmpl_val %q does not match pattern", tmplVal)
		}
		if condVal != "yes" {
			t.Errorf("cond_val = %q, want yes", condVal)
		}
		if coalVal != "fallback" {
			t.Errorf("coal_val = %q, want fallback", coalVal)
		}
		if exprFnVal != 10 {
			t.Errorf("expr_fn_val = %v, want 10", exprFnVal)
		}
	}
}

func testRunUUIDs(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_uuids"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	uuidPat := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	rows, err := db.Query(queries["uuids"])
	if err != nil {
		t.Fatalf("querying: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v1, v4, v6, v7 string
		rows.Scan(&v1, &v4, &v6, &v7)
		for _, v := range []string{v1, v4, v6, v7} {
			if !uuidPat.MatchString(v) {
				t.Errorf("invalid UUID: %q", v)
			}
		}
	}
}

func testRunNumbers(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_numbers"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	rows, err := db.Query(queries["numbers"])
	if err != nil {
		t.Fatalf("querying: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var floatVal, uniformVal, normVal, normFVal float64
		var zipfVal int
		rows.Scan(&floatVal, &uniformVal, &normVal, &normFVal, &zipfVal)

		if floatVal < 1.0 || floatVal > 100.0 {
			t.Errorf("float_val %v out of [1, 100]", floatVal)
		}
		if uniformVal < 1.0 || uniformVal > 100.0 {
			t.Errorf("uniform_val %v out of [1, 100]", uniformVal)
		}
		if normVal < 1 || normVal > 100 {
			t.Errorf("norm_val %v out of [1, 100]", normVal)
		}
		if normFVal < 1 || normFVal > 100 {
			t.Errorf("norm_f_val %v out of [1, 100]", normFVal)
		}
		if zipfVal < 0 || zipfVal > 99 {
			t.Errorf("zipf_val %d out of [0, 99]", zipfVal)
		}
	}
}

func testRunSets(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_sets"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	validRand := map[string]bool{"red": true, "green": true, "blue": true}
	validWeighted := map[string]bool{"visa": true, "mastercard": true, "amex": true}
	validDist := map[string]bool{"a": true, "b": true, "c": true, "d": true, "e": true}

	rows, err := db.Query(queries["sets"])
	if err != nil {
		t.Fatalf("querying: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var randVal, weightedVal, normalVal, expVal, lognormVal, zipfianVal string
		rows.Scan(&randVal, &weightedVal, &normalVal, &expVal, &lognormVal, &zipfianVal)

		if !validRand[randVal] {
			t.Errorf("rand_val %q not in valid set", randVal)
		}
		if !validWeighted[weightedVal] {
			t.Errorf("weighted_val %q not in valid set", weightedVal)
		}
		if !validDist[normalVal] {
			t.Errorf("normal_val %q not in valid set", normalVal)
		}
		if !validDist[expVal] {
			t.Errorf("exp_val %q not in valid set", expVal)
		}
		if !validDist[lognormVal] {
			t.Errorf("lognorm_val %q not in valid set", lognormVal)
		}
		if !validDist[zipfianVal] {
			t.Errorf("zipfian_val %q not in valid set", zipfianVal)
		}
	}
}

func testRunJSON(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_json"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	rows, err := db.Query(queries["json"])
	if err != nil {
		t.Fatalf("querying: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var obj, arr string
		rows.Scan(&obj, &arr)

		if !strings.HasPrefix(obj, "{") {
			t.Errorf("obj %q is not a JSON object", obj)
		}
		if !strings.HasPrefix(arr, "[") {
			t.Errorf("arr %q is not a JSON array", arr)
		}
	}
}

func testRunGeo(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_geo"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	rows, err := db.Query(queries["geo"])
	if err != nil {
		t.Fatalf("querying: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var lat, lon float64
		var wkt string
		rows.Scan(&lat, &lon, &wkt)

		if lat < -90 || lat > 90 {
			t.Errorf("lat %v out of [-90, 90]", lat)
		}
		if lon < -180 || lon > 180 {
			t.Errorf("lon %v out of [-180, 180]", lon)
		}
		if !strings.HasPrefix(wkt, "POINT(") {
			t.Errorf("wkt %q does not start with POINT(", wkt)
		}
	}
}

func testRunTime(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_time"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	timePat := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}$`)
	timezPat := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}\+00:00$`)

	rows, err := db.Query(queries["time"])
	if err != nil {
		t.Fatalf("querying: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ts, dur, dateStr, offsetTs, timeVal, timezVal string
		rows.Scan(&ts, &dur, &dateStr, &offsetTs, &timeVal, &timezVal)

		if _, err := time.Parse(time.RFC3339, ts); err != nil {
			t.Errorf("ts %q is not valid RFC3339: %v", ts, err)
		}
		if _, err := time.ParseDuration(dur); err != nil {
			t.Errorf("dur %q is not valid duration: %v", dur, err)
		}
		if _, err := time.Parse("2006-01-02", dateStr); err != nil {
			t.Errorf("date_str %q is not valid date: %v", dateStr, err)
		}
		if _, err := time.Parse(time.RFC3339, offsetTs); err != nil {
			t.Errorf("offset_ts %q is not valid RFC3339: %v", offsetTs, err)
		}
		if !timePat.MatchString(timeVal) {
			t.Errorf("time_val %q does not match HH:MM:SS", timeVal)
		}
		if !timezPat.MatchString(timezVal) {
			t.Errorf("timez_val %q does not match HH:MM:SS+00:00", timezVal)
		}
	}
}

func testRunDistributions(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_distributions"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	rows, err := db.Query(queries["distributions"])
	if err != nil {
		t.Fatalf("querying: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var nuVal int
		var nuVals, normVals string
		var expVal, lognormVal, expIntVal, lognormIntVal float64
		rows.Scan(&nuVal, &nuVals, &normVals, &expVal, &lognormVal, &expIntVal, &lognormIntVal)

		if nuVal < 1 || nuVal > 1000 {
			t.Errorf("nu_val %d out of [1, 1000]", nuVal)
		}

		nuParts := strings.Split(nuVals, ",")
		if len(nuParts) < 3 || len(nuParts) > 5 {
			t.Errorf("nu_vals has %d parts, want 3-5", len(nuParts))
		}

		normParts := strings.Split(normVals, ",")
		if len(normParts) < 3 || len(normParts) > 5 {
			t.Errorf("norm_vals has %d parts, want 3-5", len(normParts))
		}

		if expVal < 0 || expVal > 100 {
			t.Errorf("exp_val %v out of [0, 100]", expVal)
		}

		if lognormVal < 1 || lognormVal > 100 {
			t.Errorf("lognorm_val %v out of [1, 100]", lognormVal)
		}

		if expIntVal < 0 || expIntVal > 100 {
			t.Errorf("exp_int_val %v out of [0, 100]", expIntVal)
		}

		if lognormIntVal < 1 || lognormIntVal > 100 {
			t.Errorf("lognorm_int_val %v out of [1, 100]", lognormIntVal)
		}
	}
}

func testRunBinary(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_binary"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	bytesPat := regexp.MustCompile(`^\\x[0-9a-f]{32}$`)
	bitPat := regexp.MustCompile(`^[01]{8}$`)
	varbitPat := regexp.MustCompile(`^[01]{1,16}$`)

	rows, err := db.Query(queries["binary"])
	if err != nil {
		t.Fatalf("querying: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var bytesVal, bitVal, varbitVal, inetVal string
		rows.Scan(&bytesVal, &bitVal, &varbitVal, &inetVal)

		if !bytesPat.MatchString(bytesVal) {
			t.Errorf("bytes_val %q does not match \\x + 32 hex chars", bytesVal)
		}
		if !bitPat.MatchString(bitVal) {
			t.Errorf("bit_val %q does not match 8 bits", bitVal)
		}
		if !varbitPat.MatchString(varbitVal) {
			t.Errorf("varbit_val %q does not match 1-16 bits", varbitVal)
		}
		if ip := net.ParseIP(inetVal); ip == nil {
			t.Errorf("inet_val %q is not a valid IP address", inetVal)
		}
	}
}

func testRunArrays(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_arrays"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	rows, err := db.Query(queries["arrays"])
	if err != nil {
		t.Fatalf("querying: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var arrVal string
		rows.Scan(&arrVal)

		if !strings.HasPrefix(arrVal, "{") || !strings.HasSuffix(arrVal, "}") {
			t.Errorf("arr_val %q is not a valid array literal", arrVal)
		}
	}
}

func testRunRefs(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_refs"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	rows, err := db.Query(queries["refs"])
	if err != nil {
		t.Fatalf("querying: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var randID, sameID int
		var sameName, refNIDs, weightedIDs string
		rows.Scan(&randID, &sameID, &sameName, &refNIDs, &weightedIDs)

		if randID < 1 || randID > 20 {
			t.Errorf("rand_id %d out of [1, 20]", randID)
		}
		if sameID < 1 || sameID > 20 {
			t.Errorf("same_id %d out of [1, 20]", sameID)
		}

		// ref_same should return consistent id and name.
		expectedName := fmt.Sprintf("item-%d", sameID)
		if sameName != expectedName {
			t.Errorf("same_name %q inconsistent with same_id %d (want %q)", sameName, sameID, expectedName)
		}

		refNParts := strings.Split(refNIDs, ",")
		if len(refNParts) < 2 || len(refNParts) > 4 {
			t.Errorf("ref_n_ids has %d parts, want 2-4", len(refNParts))
		}

		weightedParts := strings.Split(weightedIDs, ",")
		if len(weightedParts) < 2 || len(weightedParts) > 4 {
			t.Errorf("weighted_ids has %d parts, want 2-4", len(weightedParts))
		}
	}
}

func testRunRefDiff(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_ref_diff"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	rows, err := db.Query(queries["ref_diff"])
	if err != nil {
		t.Fatalf("querying: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id1, id2 int
		rows.Scan(&id1, &id2)
		if id1 == id2 {
			t.Errorf("ref_diff returned same id for both args: %d", id1)
		}
	}
}

func testRunAgg(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_agg"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	// ref_source has ids 1..20 and weight = id * 10.
	// All aggregation values are deterministic.
	rows, err := db.Query(queries["agg"])
	if err != nil {
		t.Fatalf("querying: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sumVal, avgVal, minVal, maxVal float64
		var countVal, distinctVal int
		rows.Scan(&sumVal, &avgVal, &minVal, &maxVal, &countVal, &distinctVal)

		if sumVal != 210 {
			t.Errorf("sum_val = %v, want 210", sumVal)
		}
		if avgVal != 10.5 {
			t.Errorf("avg_val = %v, want 10.5", avgVal)
		}
		if minVal != 1 {
			t.Errorf("min_val = %v, want 1", minVal)
		}
		if maxVal != 20 {
			t.Errorf("max_val = %v, want 20", maxVal)
		}
		if countVal != 20 {
			t.Errorf("count_val = %d, want 20", countVal)
		}
		if distinctVal != 20 {
			t.Errorf("distinct_val = %d, want 20", distinctVal)
		}
	}
}

func testRunRefPerm(t *testing.T, queries map[string]string) {
	if got := rowCount(t, "test_ref_perm"); got != runIterations {
		t.Errorf("rows = %d, want %d", got, runIterations)
	}

	// All rows should have the same perm_id since the same Env
	// is used for all iterations.
	var distinct int
	db.QueryRow(queries["ref_perm"]).Scan(&distinct)
	if distinct != 1 {
		t.Errorf("ref_perm produced %d distinct values, want 1", distinct)
	}
}

func testDeseed(t *testing.T, env *Env, ctx context.Context) {
	if err := env.Deseed(ctx); err != nil {
		t.Fatalf("deseed: %v", err)
	}
	for _, table := range allTables {
		if got := rowCount(t, table); got != 0 {
			t.Errorf("%s has %d rows after deseed, want 0", table, got)
		}
	}
}

func testDown(t *testing.T, env *Env, ctx context.Context, queries map[string]string) {
	if err := env.Down(ctx); err != nil {
		t.Fatalf("down: %v", err)
	}
	for _, table := range allTables {
		if tableExists(t, queries["table_exists"], table) {
			t.Errorf("table %s still exists after down", table)
		}
	}
}
