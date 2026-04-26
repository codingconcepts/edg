---
title: Expressions
weight: 4
bookCollapseSection: true
---

# Expressions

Query arguments are written as expressions compiled at startup using [expr-lang/expr](https://github.com/expr-lang/expr). Each expression has access to the built-in functions, globals, and any user-defined expressions.

> **Tip:** Use `edg repl` to try any expression interactively without a database connection. See [REPL]({{< relref "../cli-reference/repl" >}}) for details.

## Functions

These are edg's built-in functions, available in any expression context (`args:`, `expressions:`, globals). They generate data, reference datasets, aggregate values, and control execution flow.

| Function | Returns | Description |
|---|---|---|
| `__sep__` | `string` | Driver-aware batch field separator. A query-text token that is replaced with the SQL function producing the ASCII unit separator character (char 31) used to delimit values within batch-expanded placeholders. Resolves to `chr(31)` for pgx, `CHAR(31)` for MySQL and MSSQL, `codepoints-to-string(31)` for Oracle, `CODE_POINTS_TO_STRING([31])` for Spanner. Can be used in any argument position within SQL. **Always use `__sep__` instead of a literal comma. Generated values may contain commas, which would silently corrupt your data.**<br><br>`string_to_array('$1', __sep__)` |
| `arg(index)` | `any` | Returns the value of a previously evaluated arg by its zero-based index or name. Enables dependent columns where later args reference earlier ones.<br><br>`arg(0)` -> `"Alice"`<br>`arg('email')` -> `"alice@example.com"` (with [named args]({{< relref "configuration#named-args" >}})) |
| `array(minN, maxN, pattern)` | `string` | PostgreSQL/CockroachDB array literal with a random number of elements.<br><br>`array(2, 4, 'email')` -> `{a@b.com,c@d.com,d@e.com}` |
| `avg(name, field)` | `float64` | Average of a numeric field across all rows in a named dataset.<br><br>`avg('fetch_products', 'price')` -> `19.39` |
| `batch(n)` | `[][]any` | Returns sequential integers `[0, n)` as batch arg sets,<br><br>`batch(3)` -> `[[0], [1], [2]]` |
| `bit(n)` | `string` | Random fixed-length bit string of exactly `n` bits.<br><br>`bit(8)` -> `10110011` |
| `blob(n)` | `[]byte` | Random `n` bytes as raw binary data. Works across all databases (PostgreSQL, MySQL, Oracle, MSSQL) via bind parameters. Use this for BLOB, BYTEA, VARBINARY, and RAW columns.<br><br>`blob(1024)` -> `(1024 random bytes)` |
| `bool()` | `bool` | Random `true` or `false`. Useful as a coin flip with `cond()` and `arg()` for mutually exclusive columns.<br><br>`bool()` -> `true` |
| `bytes(n)` | `string` | Random `n` bytes as a hex-encoded string with `\x` prefix. PostgreSQL/CockroachDB only. For cross-database binary data, use `blob(n)` instead.<br><br>`bytes(4)` -> `\x1a2b3c4d` |
| `coalesce(v1, v2, ...)` | `any` | Returns the first non-nil value from arguments.<br><br>`coalesce(nil, 'default')` -> `default` |
| `cond(predicate, trueVal, falseVal)` | `any` | Returns `trueVal` if `predicate` is true, `falseVal` otherwise.<br><br>`cond(true, 'yes', 'no')` -> `yes` |
| `const(value)` | `any` | Returns the value as-is. Useful for literal constants.<br><br>`const(42)` -> `42` |
| `count(name)` | `int` | Number of rows in a named dataset.<br><br>`count('fetch_products')` -> `5` |
| `date_offset(duration)` | `string` | Returns the current time offset by `duration`, formatted as RFC3339.<br><br>`date_offset('-72h')` -> `2026-04-08T10:00:00Z` |
| `date(format, min, max)` | `string` | Random timestamp formatted using a Go time format string.<br><br>`date('2006-01-02', '2020-01-01T00:00:00Z', '2025-01-01T00:00:00Z')` -> `2023-07-15` |
| `distinct(name, field)` | `int` | Number of distinct values for a field in a named dataset.<br><br>`distinct('fetch_products', 'category')` -> `3` |
| `duration(min, max)` | `string` | Random duration between `min` and `max` (Go duration strings).<br><br>`duration('1h', '24h')` -> `14h32m17s` |
| `env_nil(name)` | `any` | Returns the value of an environment variable as a string, or `nil` if unset. Unlike `env()`, does not error on missing variables. Designed for use with `coalesce()` to provide defaults: `int(coalesce(env_nil('PORT'), 8080))`. Always returns a string when the variable exists, so wrap with `int()` or `float()` when arithmetic is needed.<br><br>`env_nil('MISSING')` -> `nil`<br>`env_nil('HOST')` -> `localhost` |
| `env(name)` | `string` | Returns the value of a given environment variable (or an error if one doesn't exist with that name). Missing variables are caught at config load time, before any queries run. Can be composed with other functions, e.g. `upper(env('HOST'))`. For numeric values, use expr-lang conversion: `int(env('PORT'))`, `float(env('RATE'))`.<br><br>`env('API_KEY')` -> `ca3864628a8f29d644e1...` |
| `exp_f(rate, min, max, precision)` | `float64` | Exponentially-distributed random number in [min, max], rounded to `precision` decimal places.<br><br>`exp_f(0.5, 0, 100, 2)` -> `3.72` |
| `exp(rate, min, max)` | `float64` | Exponentially-distributed random number in [min, max], rounded to 0 decimal places.<br><br>`exp(0.5, 0, 100)` -> `4` |
| `expr(expression)` | `any` | Evaluates an arithmetic expression. Alias for `const`, the expr engine handles the arithmetic.<br><br>`expr(2 + 3)` -> `5` |
| `fail(message)` | `error` | Returns an error that stops the current worker gracefully. Useful with `??` to catch unexpected values: `{'a': 1}['x'] ?? fail('unknown key')`.<br><br>`fail('unexpected region')` -> *(worker stops with error)* |
| `fatal(message)` | `void` | Terminates the entire process immediately. Use when an unexpected value should halt all workers, not just the current one.<br><br>`fatal('missing required config')` -> *(process exits)* |
| `gen_batch(total, batchSize, pattern)` | `[][]any` | Generates `total` values using [gofakeit](https://github.com/brianvoe/gofakeit) `pattern`, grouped into batches of `batchSize`. Each batch arg is a string of generated values delimited by the ASCII unit separator (char 31, `\x1f`).<br><br>`gen_batch(4, 2, 'firstname')` -> `[["Alice\x1fBob"], ["Carol\x1fDave"]]` |
| `uniq(expression)` | `any` | Evaluates a string expression repeatedly until a unique (not previously seen) value is produced. Defaults to 100 retry attempts. Pass an optional second argument to override: `uniq("regex('[A-Z]{2}')", 500)`. Seen values persist across rows within a query and reset between queries.<br><br>`uniq("gen('airlineairportiata')")` -> `LAX` |
| `gen(pattern)` | `string` | Generates a random value using [gofakeit](https://github.com/brianvoe/gofakeit) patterns (e.g. `gen('number:1,100')`).<br><br>`gen('number:1,10')` -> `7` |
| `global(name)` | `any` | Looks up a value from the `globals` section by name. Globals are also available directly as variables, so `global('warehouses')` and `warehouses` are equivalent.<br><br>`global('warehouses')` -> `10` |
| `inet(cidr)` | `string` | Random IP address within the given CIDR block.<br><br>`inet('192.168.1.0/24')` -> `192.168.1.42` |
| `iter()` | `int` | 1-based row counter for `exec_batch` / `query_batch` queries. Returns 1 for the first row, 2 for the second, etc. Resets at the start of each batch query. Useful for generating sequential IDs without a global sequence.<br><br>`iter()` -> `1` |
| `json_arr(minN, maxN, pattern)` | `string` | Builds a JSON array of N random values (N in [minN, maxN]) generated by a gofakeit `pattern`.<br><br>`json_arr(1, 3, 'word')` -> `["foo","bar"]` |
| `json_obj(k1, v1, k2, v2, ...)` | `string` | Builds a JSON object string from key-value pair arguments.<br><br>`json_obj('key', 'val')` -> `{"key":"val"}` |
| `lognorm_f(mu, sigma, min, max, precision)` | `float64` | Log-normally-distributed random number in [min, max], rounded to `precision` decimal places.<br><br>`lognorm_f(1.0, 0.5, 1, 1000, 2)` -> `3.42` |
| `lognorm(mu, sigma, min, max)` | `float64` | Log-normally-distributed random number in [min, max], rounded to 0 decimal places.<br><br>`lognorm(1.0, 0.5, 1, 1000)` -> `3` |
| `max(name, field)` | `float64` | Maximum value of a numeric field in a named dataset.<br><br>`max('fetch_products', 'price')` -> `49.99` |
| `min(name, field)` | `float64` | Minimum value of a numeric field in a named dataset.<br><br>`min('fetch_products', 'price')` -> `1.99` |
| `norm_f(mean, stddev, min, max, precision)` | `float64` | Normally-distributed random number in [min, max], rounded to `precision` decimal places.<br><br>`norm_f(50.0, 15.0, 1.0, 100.0, 2)` -> `52.37` |
| `norm_n(mean, stddev, min, max, minN, maxN)` | `string` | N unique normally-distributed values (N in [minN, maxN]) as a comma-separated string.<br><br>`norm_n(50.0, 10.0, 1, 100, 2, 4)` -> `47,53,61` |
| `norm(mean, stddev, min, max)` | `float64` | Normally-distributed random number in [min, max], rounded to 0 decimal places.<br><br>`norm(4, 1, 1, 5)` -> `4` |
| `nullable(expr, probability)` | `any` | Returns NULL with `probability` (0.0–1.0), otherwise returns the expression result.<br><br>`nullable(gen('email'), 0.3)` -> `NULL` |
| `nurand_n(A, x, y, min, max)` | `string` | Generates N unique NURand values (N in [min, max]) as a comma-separated string.<br><br>`nurand_n(255, 1, 100, 3, 5)` -> `42,87,13,61` |
| `nurand(A, x, y)` | `int` | TPC-C Non-Uniform Random: `(((random(0,A) \| random(x,y)) + C) / (y-x+1)) + x`.<br><br>`nurand(255, 1, 100)` -> `42` |
| `point_wkt(lat, lon, radiusKM)` | `string` | Generates a random geographic point as a WKT string: `POINT(lon lat)`.<br><br>`point_wkt(51.5, -0.1, 10.0)` -> `POINT(-0.082 51.513)` |
| `point(lat, lon, radiusKM)` | `map` | Generates a random geographic point within `radiusKM` of (`lat`, `lon`). Access fields with `.lat` and `.lon`.<br><br>`point(51.5, -0.1, 10.0).lat` -> `51.513` |
| `ref_diff(name)` | `map` | Returns unique rows across multiple calls within the same query execution. Uses a swap-based index to avoid repeats.<br><br>`ref_diff('products').name` -> `Widget` |
| `ref_each(query)` | `[][]any` | Executes a SQL query and returns all rows. Each row becomes a separate arg set.<br><br>`ref_each('SELECT id FROM t')` -> `[[1], [2], [3]]` |
| `ref_n(name, field, min, max)` | `string` | Picks N unique random rows (N in [min, max]) from a named dataset, extracts `field` from each, and returns a comma-separated string.<br><br>`ref_n('products', 'name', 2, 3)` -> `Widget,Gadget` |
| `ref_perm(name)` | `map` | Returns a random row on first call, then the same row for the entire lifetime of the worker.<br><br>`ref_perm('products').name` -> `Widget` |
| `ref_rand(name)` | `map` | Returns a random row from a named dataset (populated by an `init` query). Access fields with dot notation: `ref_rand('fetch_warehouses').w_id`.<br><br>`ref_rand('products').name` -> `Gadget` |
| `ref_same(name)` | `map` | Returns a random row, but the same row is reused across all `ref_same` calls within a single query execution. Cleared between iterations.<br><br>`ref_same('products').name` -> `Widget` |
| `regex(pattern)` | `string` | Generates a random string matching the given regular expression.<br><br>`regex('[A-Z]{3}-[0-9]{4}')` -> `ABK-7291` |
| `seq_exp(name, rate)` | `int` | Exponentially-distributed value from a global sequence. Lower indices are selected more frequently.<br><br>`seq_exp("order_id", 0.5)` -> `7` |
| `seq_global(name)` | `int` | Shared auto-incrementing sequence across all workers. Returns the next value from a named sequence defined in the [`seq`]({{< relref "configuration#seq" >}}) config section. Thread-safe via atomic counters.<br><br>`seq_global("order_id")` -> `1` |
| `seq_lognorm(name, mu, sigma)` | `int` | Log-normally-distributed value from a global sequence.<br><br>`seq_lognorm("order_id", 2, 0.5)` -> `8` |
| `seq_norm(name, mean, stddev)` | `int` | Normally-distributed value from a global sequence. `mean` and `stddev` are index positions (0-based).<br><br>`seq_norm("order_id", 500, 100)` -> `487` |
| `seq_rand(name)` | `int` | Uniform random value from the already-generated values of a global sequence. Computes valid values from the sequence's start, step, and current counter (no values stored in memory).<br><br>`seq_rand("order_id")` -> `42` |
| `seq_zipf(name, s, v)` | `int` | Zipfian-distributed value from a global sequence. Lower indices (earlier values) are selected more frequently. `s` (> 1) and `v` (>= 1) control the distribution shape.<br><br>`seq_zipf("order_id", 2.0, 1.0)` -> `3` |
| `seq(start, step)` | `int` | Auto-incrementing sequence per worker. Returns `start + counter * step`.<br><br>`seq(1, 1)` -> `1` |
| `set_exp(values, rate)` | `any` | Picks an item from a set using exponential distribution.<br><br>`set_exp(['low', 'med', 'high'], 0.5)` -> `low` |
| `set_lognorm(values, mu, sigma)` | `any` | Picks an item from a set using log-normal distribution.<br><br>`set_lognorm(['free', 'basic', 'pro'], 0.5, 0.5)` -> `free` |
| `set_norm(values, mean, stddev)` | `any` | Picks an item from a set using normal distribution.<br><br>`set_norm([1, 2, 3, 4, 5], 2, 0.8)` -> `3` |
| `set_rand(values, weights)` | `any` | Picks a random item from a set. If weights are provided, weighted random selection is used; otherwise uniform.<br><br>`set_rand(['a', 'b', 'c'], [])` -> `b` |
| `set_zipf(values, s, v)` | `any` | Picks an item from a set using Zipfian distribution.<br><br>`set_zipf(['a', 'b', 'c'], 2.0, 1.0)` -> `a` |
| `sum(name, field)` | `float64` | Sum of a numeric field across all rows in a named dataset.<br><br>`sum('fetch_products', 'price')` -> `96.95` |
| `template(format, args...)` | `string` | Formats a string using Go's `fmt.Sprintf` syntax.<br><br>`template('ORD-%05d', seq(1, 1))` -> `ORD-00001` |
| `time(min, max)` | `string` | Random time of day between `min` and `max` (HH:MM:SS format).<br><br>`time('08:00:00', '18:00:00')` -> `14:32:07` |
| `timestamp(min, max)` | `string` | Random timestamp between `min` and `max` (RFC3339).<br><br>`timestamp('2020-01-01T00:00:00Z', '2025-01-01T00:00:00Z')` -> `2023-07-15T14:32:07Z` |
| `timez(min, max)` | `string` | Random time of day with `+00:00` timezone suffix.<br><br>`timez('09:00:00', '17:00:00')` -> `14:32:07+00:00` |
| `uniform_f(min, max, precision)` | `float64` | Uniform random float in [min, max] rounded to `precision` decimal places.<br><br>`uniform_f(0.01, 999.99, 2)` -> `347.82` |
| `uniform(min, max)` | `float64` | Uniform random float in [min, max].<br><br>`uniform(1, 100)` -> `73.12` |
| `uuid_v1()` | `string` | Generates a Version 1 UUID (timestamp + node ID).<br><br>`uuid_v1()` -> `6ba7b810-9dad-11d1-80b4-00c04fd430c8` |
| `uuid_v4()` | `string` | Generates a Version 4 UUID (random).<br><br>`uuid_v4()` -> `550e8400-e29b-41d4-a716-446655440000` |
| `uuid_v6()` | `string` | Generates a Version 6 UUID (reordered timestamp).<br><br>`uuid_v6()` -> `1ef21d2f-6ba7-6810-9dad-00c04fd430c8` |
| `uuid_v7()` | `string` | Generates a Version 7 UUID (Unix timestamp + random, sortable).<br><br>`uuid_v7()` -> `018ef4c9-7f3a-7b3c-8d1a-2b4c5d6e7f8a` |
| `varbit(n)` | `string` | Random variable-length bit string of 1 to `n` bits.<br><br>`varbit(8)` -> `10110` |
| `vector_norm(dims, clusters, spread, mean, stddev)` | `string` | Like `vector` but picks centroids using a normal distribution over cluster indices. `mean` is the center cluster index, `stddev` controls spread.<br><br>`vector_norm(32, 5, 0.1, 2.0, 0.8)` |
| `vector_zipf(dims, clusters, spread, s, v)` | `string` | Like `vector` but picks centroids using a Zipfian distribution. Cluster 0 is the "hottest", with frequency dropping off according to `s` (skew) and `v` (>= 1). Simulates real-world data where some categories have far more embeddings.<br><br>`vector_zipf(32, 5, 0.1, 2.0, 1.0)` |
| `vector(dims, clusters, spread)` | `string` | pgvector-compatible vector literal with uniform centroid selection. Generates clustered, unit-length vectors for realistic similarity search. `dims` is the number of dimensions, `clusters` is the number of cluster centroids, and `spread` controls intra-cluster noise (Gaussian σ).<br><br>`vector(4, 3, 0.1)` -> `[0.512340,-0.234567,0.678901,0.456789]` |
| `weighted_sample_n(name, field, weightField, minN, maxN)` | `string` | Picks N unique rows using weighted selection, returns a comma-separated string.<br><br>`weighted_sample_n('products', 'name', 'stock', 2, 3)` -> `Widget,Pen` |
| `zipf(s, v, max)` | `int` | Zipfian-distributed random integer in [0, max].<br><br>`zipf(2.0, 1.0, 999)` -> `3` |

## Function Lifecycle

Several functions maintain state. Understanding when that state resets is important for getting correct results:

| Function | Scope | Resets |
|---|---|---|
| `arg(index)` / `arg('name')` | Per-query | Returns the value of arg at `index` (or by name when using [named args]({{< relref "configuration#named-args" >}})). Cleared before the next query. In batch queries, resets per row. |
| `uniq(expression)` | Per-query | Tracks seen values across all rows within a query. Resets between queries. |
| `iter()` | Per-query | Returns 1 for the first row, 2 for the second, etc. Resets to 0 at the start of each batch query. |
| `nurand(A, x, y)` | Per-worker | The TPC-C constant C is generated once per worker per A value and stays fixed for the worker's lifetime. |
| `ref_diff(name)` | Per-query | Returns a unique row on each call within a query (no repeats). Index resets before the next query. |
| `ref_perm(name)` | Per-worker | Picks a row on first call and returns that same row for the entire lifetime of the worker. Never resets. |
| `ref_rand(name)` | None | Fresh random row on every call |
| `ref_same(name)` | Per-query | Picks a row on first call within a query; all subsequent `ref_same` calls for the same dataset within that query return the same row. **Cleared before the next query.** |
| `seq_global(name)` | Global | Single counter shared across all workers via atomic increment. Values are **globally unique**. Configured in the [`seq`]({{< relref "configuration#seq" >}}) config section. |
| `seq_rand` / `seq_zipf` / `seq_norm` / `seq_exp` / `seq_lognorm` | Global | Pick from already-generated sequence values using the named distribution. The valid value set grows as `seq_global` advances the counter. No values are stored in memory. Valid values are computed from `start + index * step`. |
| `seq(start, step)` | Per-worker | Counter starts at 0 for each worker and increments on every call. Two workers both calling `seq(1, 1)` will produce the same sequence independently -- values are **not globally unique**. |
| `vector` / `vector_zipf` / `vector_norm` | Per-worker | Cluster centroids are generated on first call (keyed by dims+clusters) and reused for the worker's lifetime. Each call picks a centroid (uniform, Zipfian, or normal) and adds noise. |
