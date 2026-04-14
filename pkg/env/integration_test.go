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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	err := yaml.Unmarshal(data, &req)
	require.NoError(t, err, "parsing config")
	return &req
}

func rowCount(t *testing.T, table string) int {
	t.Helper()
	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
	require.NoError(t, err, "counting %s", table)
	return count
}

func tableExists(t *testing.T, query, table string) bool {
	t.Helper()
	var count int
	err := db.QueryRow(query, table).Scan(&count)
	require.NoError(t, err, "checking table %s", table)
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
	require.NoError(t, err, "creating env")

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
	err := env.Up(ctx)
	require.NoError(t, err, "up")
	for _, table := range allTables {
		assert.True(t, tableExists(t, queries["table_exists"], table), "table %s was not created", table)
	}
}

func testSeed(t *testing.T, env *Env, ctx context.Context, queries map[string]string) {
	err := env.Seed(ctx)
	require.NoError(t, err, "seed")

	t.Run("ref_source", func(t *testing.T) { testSeedRefSource(t) })
	t.Run("gen_batch", func(t *testing.T) { testSeedGenBatch(t) })
	t.Run("batch", func(t *testing.T) { testSeedBatch(t, queries) })
	t.Run("ref_each", func(t *testing.T) { testSeedRefEach(t) })
}

func testSeedRefSource(t *testing.T) {
	assert.Equal(t, 20, rowCount(t, "test_ref_source"))
}

func testSeedGenBatch(t *testing.T) {
	assert.Equal(t, 10, rowCount(t, "test_gen_batch"))
}

func testSeedBatch(t *testing.T, queries map[string]string) {
	assert.Equal(t, 5, rowCount(t, "test_batch"))

	// Values should be 0..4.
	rows, err := db.Query(queries["batch_vals"])
	require.NoError(t, err, "querying test_batch")
	defer rows.Close()
	var vals []int
	for rows.Next() {
		var v int
		rows.Scan(&v)
		vals = append(vals, v)
	}
	for i, v := range vals {
		assert.Equal(t, i, v, "test_batch[%d]", i)
	}
}

func testSeedRefEach(t *testing.T) {
	assert.Equal(t, 5, rowCount(t, "test_ref_each"))
}

func testInit(t *testing.T, env *Env, ctx context.Context) {
	err := env.Init(ctx)
	require.NoError(t, err, "init")
}

func testRun(t *testing.T, env *Env, ctx context.Context, queries map[string]string) {
	for i := range runIterations {
		err := env.RunIteration(ctx)
		require.NoError(t, err, "run iteration %d", i)
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
	assert.Equal(t, runIterations, rowCount(t, "test_scalars"))

	rows, err := db.Query(queries["scalars"])
	require.NoError(t, err, "querying")
	defer rows.Close()

	regexPat := regexp.MustCompile(`^[A-Z]{3}-[0-9]{4}$`)
	tmplPat := regexp.MustCompile(`^ITEM-\d{5}$`)

	for rows.Next() {
		var constVal, globalVal, exprVal int
		var genVal, regexVal, tmplVal, condVal, coalVal string
		var exprFnVal float64
		rows.Scan(&constVal, &globalVal, &exprVal, &genVal, &regexVal, &tmplVal, &condVal, &coalVal, &exprFnVal)

		assert.Equal(t, 42, constVal, "const_val")
		assert.Equal(t, 42, globalVal, "global_val")
		assert.Equal(t, 84, exprVal, "expr_val")
		assert.NotEmpty(t, genVal, "gen_val")
		assert.Regexp(t, regexPat, regexVal, "regex_val")
		assert.Regexp(t, tmplPat, tmplVal, "tmpl_val")
		assert.Equal(t, "yes", condVal, "cond_val")
		assert.Equal(t, "fallback", coalVal, "coal_val")
		assert.Equal(t, float64(10), exprFnVal, "expr_fn_val")
	}
}

func testRunUUIDs(t *testing.T, queries map[string]string) {
	assert.Equal(t, runIterations, rowCount(t, "test_uuids"))

	uuidPat := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	rows, err := db.Query(queries["uuids"])
	require.NoError(t, err, "querying")
	defer rows.Close()
	for rows.Next() {
		var v1, v4, v6, v7 string
		rows.Scan(&v1, &v4, &v6, &v7)
		for _, v := range []string{v1, v4, v6, v7} {
			assert.Regexp(t, uuidPat, v, "invalid UUID")
		}
	}
}

func testRunNumbers(t *testing.T, queries map[string]string) {
	assert.Equal(t, runIterations, rowCount(t, "test_numbers"))

	rows, err := db.Query(queries["numbers"])
	require.NoError(t, err, "querying")
	defer rows.Close()
	for rows.Next() {
		var floatVal, uniformVal, normVal, normFVal float64
		var zipfVal int
		rows.Scan(&floatVal, &uniformVal, &normVal, &normFVal, &zipfVal)

		assert.True(t, floatVal >= 1.0 && floatVal <= 100.0, "float_val %v out of [1, 100]", floatVal)
		assert.True(t, uniformVal >= 1.0 && uniformVal <= 100.0, "uniform_val %v out of [1, 100]", uniformVal)
		assert.True(t, normVal >= 1 && normVal <= 100, "norm_val %v out of [1, 100]", normVal)
		assert.True(t, normFVal >= 1 && normFVal <= 100, "norm_f_val %v out of [1, 100]", normFVal)
		assert.True(t, zipfVal >= 0 && zipfVal <= 99, "zipf_val %d out of [0, 99]", zipfVal)
	}
}

func testRunSets(t *testing.T, queries map[string]string) {
	assert.Equal(t, runIterations, rowCount(t, "test_sets"))

	validRand := map[string]bool{"red": true, "green": true, "blue": true}
	validWeighted := map[string]bool{"visa": true, "mastercard": true, "amex": true}
	validDist := map[string]bool{"a": true, "b": true, "c": true, "d": true, "e": true}

	rows, err := db.Query(queries["sets"])
	require.NoError(t, err, "querying")
	defer rows.Close()
	for rows.Next() {
		var randVal, weightedVal, normalVal, expVal, lognormVal, zipfianVal string
		rows.Scan(&randVal, &weightedVal, &normalVal, &expVal, &lognormVal, &zipfianVal)

		assert.True(t, validRand[randVal], "rand_val %q not in valid set", randVal)
		assert.True(t, validWeighted[weightedVal], "weighted_val %q not in valid set", weightedVal)
		assert.True(t, validDist[normalVal], "normal_val %q not in valid set", normalVal)
		assert.True(t, validDist[expVal], "exp_val %q not in valid set", expVal)
		assert.True(t, validDist[lognormVal], "lognorm_val %q not in valid set", lognormVal)
		assert.True(t, validDist[zipfianVal], "zipfian_val %q not in valid set", zipfianVal)
	}
}

func testRunJSON(t *testing.T, queries map[string]string) {
	assert.Equal(t, runIterations, rowCount(t, "test_json"))

	rows, err := db.Query(queries["json"])
	require.NoError(t, err, "querying")
	defer rows.Close()
	for rows.Next() {
		var obj, arr string
		rows.Scan(&obj, &arr)

		assert.True(t, strings.HasPrefix(obj, "{"), "obj %q is not a JSON object", obj)
		assert.True(t, strings.HasPrefix(arr, "["), "arr %q is not a JSON array", arr)
	}
}

func testRunGeo(t *testing.T, queries map[string]string) {
	assert.Equal(t, runIterations, rowCount(t, "test_geo"))

	rows, err := db.Query(queries["geo"])
	require.NoError(t, err, "querying")
	defer rows.Close()
	for rows.Next() {
		var lat, lon float64
		var wkt string
		rows.Scan(&lat, &lon, &wkt)

		assert.True(t, lat >= -90 && lat <= 90, "lat %v out of [-90, 90]", lat)
		assert.True(t, lon >= -180 && lon <= 180, "lon %v out of [-180, 180]", lon)
		assert.True(t, strings.HasPrefix(wkt, "POINT("), "wkt %q does not start with POINT(", wkt)
	}
}

func testRunTime(t *testing.T, queries map[string]string) {
	assert.Equal(t, runIterations, rowCount(t, "test_time"))

	timePat := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}$`)
	timezPat := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}\+00:00$`)

	rows, err := db.Query(queries["time"])
	require.NoError(t, err, "querying")
	defer rows.Close()
	for rows.Next() {
		var ts, dur, dateStr, offsetTs, timeVal, timezVal string
		rows.Scan(&ts, &dur, &dateStr, &offsetTs, &timeVal, &timezVal)

		_, err := time.Parse(time.RFC3339, ts)
		assert.NoError(t, err, "ts %q is not valid RFC3339", ts)
		_, err = time.ParseDuration(dur)
		assert.NoError(t, err, "dur %q is not valid duration", dur)
		_, err = time.Parse("2006-01-02", dateStr)
		assert.NoError(t, err, "date_str %q is not valid date", dateStr)
		_, err = time.Parse(time.RFC3339, offsetTs)
		assert.NoError(t, err, "offset_ts %q is not valid RFC3339", offsetTs)
		assert.Regexp(t, timePat, timeVal, "time_val %q does not match HH:MM:SS", timeVal)
		assert.Regexp(t, timezPat, timezVal, "timez_val %q does not match HH:MM:SS+00:00", timezVal)
	}
}

func testRunDistributions(t *testing.T, queries map[string]string) {
	assert.Equal(t, runIterations, rowCount(t, "test_distributions"))

	rows, err := db.Query(queries["distributions"])
	require.NoError(t, err, "querying")
	defer rows.Close()
	for rows.Next() {
		var nuVal int
		var nuVals, normVals string
		var expVal, lognormVal, expIntVal, lognormIntVal float64
		rows.Scan(&nuVal, &nuVals, &normVals, &expVal, &lognormVal, &expIntVal, &lognormIntVal)

		assert.True(t, nuVal >= 1 && nuVal <= 1000, "nu_val %d out of [1, 1000]", nuVal)

		nuParts := strings.Split(nuVals, ",")
		assert.True(t, len(nuParts) >= 3 && len(nuParts) <= 5, "nu_vals has %d parts, want 3-5", len(nuParts))

		normParts := strings.Split(normVals, ",")
		assert.True(t, len(normParts) >= 3 && len(normParts) <= 5, "norm_vals has %d parts, want 3-5", len(normParts))

		assert.True(t, expVal >= 0 && expVal <= 100, "exp_val %v out of [0, 100]", expVal)
		assert.True(t, lognormVal >= 1 && lognormVal <= 100, "lognorm_val %v out of [1, 100]", lognormVal)
		assert.True(t, expIntVal >= 0 && expIntVal <= 100, "exp_int_val %v out of [0, 100]", expIntVal)
		assert.True(t, lognormIntVal >= 1 && lognormIntVal <= 100, "lognorm_int_val %v out of [1, 100]", lognormIntVal)
	}
}

func testRunBinary(t *testing.T, queries map[string]string) {
	assert.Equal(t, runIterations, rowCount(t, "test_binary"))

	bytesPat := regexp.MustCompile(`^\\x[0-9a-f]{32}$`)
	bitPat := regexp.MustCompile(`^[01]{8}$`)
	varbitPat := regexp.MustCompile(`^[01]{1,16}$`)

	rows, err := db.Query(queries["binary"])
	require.NoError(t, err, "querying")
	defer rows.Close()
	for rows.Next() {
		var bytesVal, bitVal, varbitVal, inetVal string
		rows.Scan(&bytesVal, &bitVal, &varbitVal, &inetVal)

		assert.Regexp(t, bytesPat, bytesVal, "bytes_val %q does not match \\x + 32 hex chars", bytesVal)
		assert.Regexp(t, bitPat, bitVal, "bit_val %q does not match 8 bits", bitVal)
		assert.Regexp(t, varbitPat, varbitVal, "varbit_val %q does not match 1-16 bits", varbitVal)
		assert.NotNil(t, net.ParseIP(inetVal), "inet_val %q is not a valid IP address", inetVal)
	}
}

func testRunArrays(t *testing.T, queries map[string]string) {
	assert.Equal(t, runIterations, rowCount(t, "test_arrays"))

	rows, err := db.Query(queries["arrays"])
	require.NoError(t, err, "querying")
	defer rows.Close()
	for rows.Next() {
		var arrVal string
		rows.Scan(&arrVal)

		assert.True(t, strings.HasPrefix(arrVal, "{") && strings.HasSuffix(arrVal, "}"),
			"arr_val %q is not a valid array literal", arrVal)
	}
}

func testRunRefs(t *testing.T, queries map[string]string) {
	assert.Equal(t, runIterations, rowCount(t, "test_refs"))

	rows, err := db.Query(queries["refs"])
	require.NoError(t, err, "querying")
	defer rows.Close()
	for rows.Next() {
		var randID, sameID int
		var sameName, refNIDs, weightedIDs string
		rows.Scan(&randID, &sameID, &sameName, &refNIDs, &weightedIDs)

		assert.True(t, randID >= 1 && randID <= 20, "rand_id %d out of [1, 20]", randID)
		assert.True(t, sameID >= 1 && sameID <= 20, "same_id %d out of [1, 20]", sameID)

		// ref_same should return consistent id and name.
		expectedName := fmt.Sprintf("item-%d", sameID)
		assert.Equal(t, expectedName, sameName, "same_name inconsistent with same_id %d", sameID)

		refNParts := strings.Split(refNIDs, ",")
		assert.True(t, len(refNParts) >= 2 && len(refNParts) <= 4, "ref_n_ids has %d parts, want 2-4", len(refNParts))

		weightedParts := strings.Split(weightedIDs, ",")
		assert.True(t, len(weightedParts) >= 2 && len(weightedParts) <= 4, "weighted_ids has %d parts, want 2-4", len(weightedParts))
	}
}

func testRunRefDiff(t *testing.T, queries map[string]string) {
	assert.Equal(t, runIterations, rowCount(t, "test_ref_diff"))

	rows, err := db.Query(queries["ref_diff"])
	require.NoError(t, err, "querying")
	defer rows.Close()
	for rows.Next() {
		var id1, id2 int
		rows.Scan(&id1, &id2)
		assert.NotEqual(t, id1, id2, "ref_diff returned same id for both args: %d", id1)
	}
}

func testRunAgg(t *testing.T, queries map[string]string) {
	assert.Equal(t, runIterations, rowCount(t, "test_agg"))

	// ref_source has ids 1..20 and weight = id * 10.
	// All aggregation values are deterministic.
	rows, err := db.Query(queries["agg"])
	require.NoError(t, err, "querying")
	defer rows.Close()
	for rows.Next() {
		var sumVal, avgVal, minVal, maxVal float64
		var countVal, distinctVal int
		rows.Scan(&sumVal, &avgVal, &minVal, &maxVal, &countVal, &distinctVal)

		assert.Equal(t, float64(210), sumVal, "sum_val")
		assert.Equal(t, 10.5, avgVal, "avg_val")
		assert.Equal(t, float64(1), minVal, "min_val")
		assert.Equal(t, float64(20), maxVal, "max_val")
		assert.Equal(t, 20, countVal, "count_val")
		assert.Equal(t, 20, distinctVal, "distinct_val")
	}
}

func testRunRefPerm(t *testing.T, queries map[string]string) {
	assert.Equal(t, runIterations, rowCount(t, "test_ref_perm"))

	// All rows should have the same perm_id since the same Env
	// is used for all iterations.
	var distinct int
	db.QueryRow(queries["ref_perm"]).Scan(&distinct)
	assert.Equal(t, 1, distinct, "ref_perm produced %d distinct values, want 1", distinct)
}

func testDeseed(t *testing.T, env *Env, ctx context.Context) {
	err := env.Deseed(ctx)
	require.NoError(t, err, "deseed")
	for _, table := range allTables {
		assert.Equal(t, 0, rowCount(t, table), "%s has rows after deseed", table)
	}
}

func testDown(t *testing.T, env *Env, ctx context.Context, queries map[string]string) {
	err := env.Down(ctx)
	require.NoError(t, err, "down")
	for _, table := range allTables {
		assert.False(t, tableExists(t, queries["table_exists"], table), "table %s still exists after down", table)
	}
}
