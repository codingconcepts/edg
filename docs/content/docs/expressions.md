---
title: Expressions
weight: 5
---

# Expressions

Query arguments are written as expressions compiled at startup using [expr-lang/expr](https://github.com/expr-lang/expr). Each expression has access to the built-in functions, globals, and any user-defined expressions.

> **Tip:** Use `edg repl` to try any expression interactively without a database connection. See [REPL]({{< relref "repl" >}}) for details.

## Functions

These are edg's built-in functions, available in any expression context (`args:`, `expressions:`, globals). They generate data, reference datasets, aggregate values, and control execution flow.

<details>
<summary>All built-in functions</summary>

| Function | Returns | Description |
|---|---|---|
| `arg(index)` | `any` | Returns the value of a previously evaluated arg by its zero-based index. Enables dependent columns where later args reference earlier ones.<br><br>`arg(0)` -> `"Alice"` |
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
| `date(format, min, max)` | `string` | Random timestamp formatted using a Go time format string.<br><br>`date('2006-01-02', '2020-01-01T00:00:00Z', '2025-01-01T00:00:00Z')` -> `2023-07-15` |
| `date_offset(duration)` | `string` | Returns the current time offset by `duration`, formatted as RFC3339.<br><br>`date_offset('-72h')` -> `2026-04-08T10:00:00Z` |
| `distinct(name, field)` | `int` | Number of distinct values for a field in a named dataset.<br><br>`distinct('fetch_products', 'category')` -> `3` |
| `duration(min, max)` | `string` | Random duration between `min` and `max` (Go duration strings).<br><br>`duration('1h', '24h')` -> `14h32m17s` |
| `env(name)` | `string` | Returns the value of a given environment variable (or an error if one doesn't exist with that name). Missing variables are caught at config load time, before any queries run. Can be composed with other functions, e.g. `upper(env('HOST'))`. For numeric values, use expr-lang conversion: `int(env('PORT'))`, `float(env('RATE'))`.<br><br>`env('API_KEY')` -> `ca3864628a8f29d644e1...` |
| `exp(rate, min, max)` | `float64` | Exponentially-distributed random number in [min, max], rounded to 0 decimal places.<br><br>`exp(0.5, 0, 100)` -> `4` |
| `exp_f(rate, min, max, precision)` | `float64` | Exponentially-distributed random number in [min, max], rounded to `precision` decimal places.<br><br>`exp_f(0.5, 0, 100, 2)` -> `3.72` |
| `expr(expression)` | `any` | Evaluates an arithmetic expression. Alias for `const`, the expr engine handles the arithmetic.<br><br>`expr(2 + 3)` -> `5` |
| `gen(pattern)` | `string` | Generates a random value using [gofakeit](https://github.com/brianvoe/gofakeit) patterns (e.g. `gen('number:1,100')`).<br><br>`gen('number:1,10')` -> `7` |
| `gen_batch(total, batchSize, pattern)` | `[][]any` | Generates `total` values using [gofakeit](https://github.com/brianvoe/gofakeit) `pattern`, grouped into batches of `batchSize`. Each batch arg is a string of generated values delimited by the ASCII unit separator (char 31, `\x1f`).<br><br>`gen_batch(4, 2, 'firstname')` -> `[["Alice\x1fBob"], ["Carol\x1fDave"]]` |
| `global(name)` | `any` | Looks up a value from the `globals` section by name. Globals are also available directly as variables, so `global('warehouses')` and `warehouses` are equivalent.<br><br>`global('warehouses')` -> `10` |
| `inet(cidr)` | `string` | Random IP address within the given CIDR block.<br><br>`inet('192.168.1.0/24')` -> `192.168.1.42` |
| `json_arr(minN, maxN, pattern)` | `string` | Builds a JSON array of N random values (N in [minN, maxN]) generated by a gofakeit `pattern`.<br><br>`json_arr(1, 3, 'word')` -> `["foo","bar"]` |
| `json_obj(k1, v1, k2, v2, ...)` | `string` | Builds a JSON object string from key-value pair arguments.<br><br>`json_obj('key', 'val')` -> `{"key":"val"}` |
| `lognorm(mu, sigma, min, max)` | `float64` | Log-normally-distributed random number in [min, max], rounded to 0 decimal places.<br><br>`lognorm(1.0, 0.5, 1, 1000)` -> `3` |
| `lognorm_f(mu, sigma, min, max, precision)` | `float64` | Log-normally-distributed random number in [min, max], rounded to `precision` decimal places.<br><br>`lognorm_f(1.0, 0.5, 1, 1000, 2)` -> `3.42` |
| `max(name, field)` | `float64` | Maximum value of a numeric field in a named dataset.<br><br>`max('fetch_products', 'price')` -> `49.99` |
| `min(name, field)` | `float64` | Minimum value of a numeric field in a named dataset.<br><br>`min('fetch_products', 'price')` -> `1.99` |
| `norm(mean, stddev, min, max)` | `float64` | Normally-distributed random number in [min, max], rounded to 0 decimal places.<br><br>`norm(4, 1, 1, 5)` -> `4` |
| `norm_f(mean, stddev, min, max, precision)` | `float64` | Normally-distributed random number in [min, max], rounded to `precision` decimal places.<br><br>`norm_f(50.0, 15.0, 1.0, 100.0, 2)` -> `52.37` |
| `norm_n(mean, stddev, min, max, minN, maxN)` | `string` | N unique normally-distributed values (N in [minN, maxN]) as a comma-separated string.<br><br>`norm_n(50.0, 10.0, 1, 100, 2, 4)` -> `47,53,61` |
| `nullable(expr, probability)` | `any` | Returns NULL with `probability` (0.0–1.0), otherwise returns the expression result.<br><br>`nullable(gen('email'), 0.3)` -> `NULL` |
| `nurand(A, x, y)` | `int` | TPC-C Non-Uniform Random: `(((random(0,A) \| random(x,y)) + C) / (y-x+1)) + x`.<br><br>`nurand(255, 1, 100)` -> `42` |
| `nurand_n(A, x, y, min, max)` | `string` | Generates N unique NURand values (N in [min, max]) as a comma-separated string.<br><br>`nurand_n(255, 1, 100, 3, 5)` -> `42,87,13,61` |
| `point(lat, lon, radiusKM)` | `map` | Generates a random geographic point within `radiusKM` of (`lat`, `lon`). Access fields with `.lat` and `.lon`.<br><br>`point(51.5, -0.1, 10.0).lat` -> `51.513` |
| `point_wkt(lat, lon, radiusKM)` | `string` | Generates a random geographic point as a WKT string: `POINT(lon lat)`.<br><br>`point_wkt(51.5, -0.1, 10.0)` -> `POINT(-0.082 51.513)` |
| `ref_diff(name)` | `map` | Returns unique rows across multiple calls within the same query execution. Uses a swap-based index to avoid repeats.<br><br>`ref_diff('products').name` -> `Widget` |
| `ref_each(query)` | `[][]any` | Executes a SQL query and returns all rows. Each row becomes a separate arg set.<br><br>`ref_each('SELECT id FROM t')` -> `[[1], [2], [3]]` |
| `ref_n(name, field, min, max)` | `string` | Picks N unique random rows (N in [min, max]) from a named dataset, extracts `field` from each, and returns a comma-separated string.<br><br>`ref_n('products', 'name', 2, 3)` -> `Widget,Gadget` |
| `ref_perm(name)` | `map` | Returns a random row on first call, then the same row for the entire lifetime of the worker.<br><br>`ref_perm('products').name` -> `Widget` |
| `ref_rand(name)` | `map` | Returns a random row from a named dataset (populated by an `init` query). Access fields with dot notation: `ref_rand('fetch_warehouses').w_id`.<br><br>`ref_rand('products').name` -> `Gadget` |
| `ref_same(name)` | `map` | Returns a random row, but the same row is reused across all `ref_same` calls within a single query execution. Cleared between iterations.<br><br>`ref_same('products').name` -> `Widget` |
| `regex(pattern)` | `string` | Generates a random string matching the given regular expression.<br><br>`regex('[A-Z]{3}-[0-9]{4}')` -> `ABK-7291` |
| `seq(start, step)` | `int` | Auto-incrementing sequence per worker. Returns `start + counter * step`.<br><br>`seq(1, 1)` -> `1` |
| `sep` | `string` | Driver-aware batch field separator. Emits the SQL function that produces the ASCII unit separator character (char 31) used to delimit values within batch-expanded placeholders. Resolves to `chr(31)` for pgx and Oracle, `CHAR(31)` for MySQL and MSSQL. Typically used inside `string_to_array` to split a batch arg back into rows. **Always use `sep` instead of a literal comma — generated values may contain commas, which would silently corrupt your data.**<br><br>`string_to_array('$1', sep)` |
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
| `uniform(min, max)` | `float64` | Uniform random float in [min, max].<br><br>`uniform(1, 100)` -> `73.12` |
| `uniform_f(min, max, precision)` | `float64` | Uniform random float in [min, max] rounded to `precision` decimal places.<br><br>`uniform_f(0.01, 999.99, 2)` -> `347.82` |
| `uuid_v1()` | `string` | Generates a Version 1 UUID (timestamp + node ID).<br><br>`uuid_v1()` -> `6ba7b810-9dad-11d1-80b4-00c04fd430c8` |
| `uuid_v4()` | `string` | Generates a Version 4 UUID (random).<br><br>`uuid_v4()` -> `550e8400-e29b-41d4-a716-446655440000` |
| `uuid_v6()` | `string` | Generates a Version 6 UUID (reordered timestamp).<br><br>`uuid_v6()` -> `1ef21d2f-6ba7-6810-9dad-00c04fd430c8` |
| `uuid_v7()` | `string` | Generates a Version 7 UUID (Unix timestamp + random, sortable).<br><br>`uuid_v7()` -> `018ef4c9-7f3a-7b3c-8d1a-2b4c5d6e7f8a` |
| `varbit(n)` | `string` | Random variable-length bit string of 1 to `n` bits.<br><br>`varbit(8)` -> `10110` |
| `vector(dims, clusters, spread)` | `string` | pgvector-compatible vector literal with uniform centroid selection. Generates clustered, unit-length vectors for realistic similarity search. `dims` is the number of dimensions, `clusters` is the number of cluster centroids, and `spread` controls intra-cluster noise (Gaussian σ).<br><br>`vector(4, 3, 0.1)` -> `[0.512340,-0.234567,0.678901,0.456789]` |
| `vector_norm(dims, clusters, spread, mean, stddev)` | `string` | Like `vector` but picks centroids using a normal distribution over cluster indices. `mean` is the center cluster index, `stddev` controls spread.<br><br>`vector_norm(32, 5, 0.1, 2.0, 0.8)` |
| `vector_zipf(dims, clusters, spread, s, v)` | `string` | Like `vector` but picks centroids using a Zipfian distribution. Cluster 0 is the "hottest", with frequency dropping off according to `s` (skew) and `v` (>= 1). Simulates real-world data where some categories have far more embeddings.<br><br>`vector_zipf(32, 5, 0.1, 2.0, 1.0)` |
| `weighted_sample_n(name, field, weightField, minN, maxN)` | `string` | Picks N unique rows using weighted selection, returns a comma-separated string.<br><br>`weighted_sample_n('products', 'name', 'stock', 2, 3)` -> `Widget,Pen` |
| `zipf(s, v, max)` | `int` | Zipfian-distributed random integer in [0, max].<br><br>`zipf(2.0, 1.0, 999)` -> `3` |

</details>

## Function Lifecycle

Several functions maintain state. Understanding when that state resets is important for getting correct results:

| Function | Scope | Resets |
|---|---|---|
| `arg(index)` | Per-query | Returns the value of arg at `index` from the current query execution. Cleared before the next query. In batch queries, resets per row. |
| `nurand(A, x, y)` | Per-worker | The TPC-C constant C is generated once per worker per A value and stays fixed for the worker's lifetime. |
| `ref_diff(name)` | Per-query | Returns a unique row on each call within a query (no repeats). Index resets before the next query. |
| `ref_perm(name)` | Per-worker | Picks a row on first call and returns that same row for the entire lifetime of the worker. Never resets. |
| `ref_rand(name)` | None | Fresh random row on every call |
| `ref_same(name)` | Per-query | Picks a row on first call within a query; all subsequent `ref_same` calls for the same dataset within that query return the same row. **Cleared before the next query.** |
| `seq(start, step)` | Per-worker | Counter starts at 0 for each worker and increments on every call. Two workers both calling `seq(1, 1)` will produce the same sequence independently -- values are **not globally unique**. |
| `vector` / `vector_zipf` / `vector_norm` | Per-worker | Cluster centroids are generated on first call (keyed by dims+clusters) and reused for the worker's lifetime. Each call picks a centroid (uniform, Zipfian, or normal) and adds noise. |

## User-Defined Expressions

The `expressions` section lets you define named functions from [expr-lang](https://expr-lang.org/docs/language-definition) expression strings. Each expression becomes a callable function available in any query arg. Expressions can reference globals, built-in functions, and other expressions. Arguments passed to the function are available via the `args` slice.

```yaml
globals:
  total_rows: 10000
  num_buckets: 10

expressions:
  # No-arg expressions (computed from globals).
  rows_per_bucket: "total_rows / num_buckets"
  ten_percent: "int(ceil(total_rows * 0.1))"

  # Parameterized expressions (use args[0], args[1], etc.).
  clamp: "max(min(args[0], args[1]), 0)"
  pct_of: "int(ceil(args[0] * args[1] / 100))"
  like_prefix: "string(args[0]) + '%'"
  pick_label: "args[0] > 1000 ? 'large' : 'small'"
  wrapped_offset: "abs(args[0] - args[1]) % args[2]"
  power_scale: "int(floor(float(args[0]) ** 2))"
  is_active: "not (args[0] == 0) and (args[0] != args[1] or args[0] < args[2])"
  safe_val: "args[0] ?? args[1]"
  round_ratio: "round(float(args[0]) / float(args[1]))"
  normalize: "lower(trim(replace(args[0], ' ', '_')))"
  shout: "upper(split(args[0], ',')[0])"
  add_piped: "(args[0] + args[1]) | int"

run:
  - name: example_query
    query: >-
      SELECT * FROM t
      WHERE bucket = $1
        AND label = $2
        AND name LIKE $3
        AND score >= $4
        AND active = $5
        AND tag = $6
        AND rank >= $7
        AND tier = $8
        AND fallback = $9
        AND weight = $10
        AND grp = $11
        AND max_id <= $12
      LIMIT $13
      OFFSET $14
    args:
      - gen('number:1,' + num_buckets)          # $1  random int 1-num_buckets
      - pick_label(total_rows)                  # $2  "large"
      - like_prefix('foo')                      # $3  "foo%"
      - pct_of(total_rows, 5)                   # $4  500
      - is_active(1, 2, 100)                    # $5  true
      - normalize(' Foo Bar ')                  # $6  "foo_bar"
      - power_scale(3)                          # $7  9
      - shout('hello,world')                    # $8  "HELLO"
      - safe_val('premium', 'basic')            # $9  "premium"
      - round_ratio(total_rows, num_buckets)    # $10 1000
      - wrapped_offset(7, 20, num_buckets)      # $11 3
      - add_piped(rows_per_bucket, ten_percent) # $12 2000
      - clamp(rows_per_bucket, 500)             # $13 500
      - ten_percent                             # $14 1000
```

## expr examples

Expressions support the full expr-lang feature set, including:

| Category | Examples |
|---|---|
| Arithmetic | `+`, `-`, `*`, `/`, `%`, `**` |
| Array functions | `filter()`, `map()`, `reduce()`, `sort()`, `sortBy()`, `reverse()`, `first()`, `last()`, `take()`, `flatten()`, `uniq()`, `concat()`, `join()`, `find()`, `findIndex()`, `findLast()`, `findLastIndex()`, `all()`, `any()`, `one()`, `none()`, `groupBy()` |
| Bitwise | `bitand()`, `bitor()`, `bitxor()`, `bitnand()`, `bitnot()`, `bitshl()`, `bitshr()`, `bitushr()` |
| Comparison & logic | `==`, `!=`, `<`, `>`, `and`, `or`, `not` |
| Conditionals | `args[0] > 10 ? 'yes' : 'no'`, `args[0] ?? 0` (nil coalescing), `if`/`else` |
| Language | `let` bindings, `#` predicates (current element in closures), `len()`, `get()` |
| Map functions | `keys()`, `values()` |
| Math functions | `abs()`, `ceil()`, `floor()`, `round()`, `mean()`, `median()` |
| Operators | `\|` (pipe), `in` (membership), `..` (range), `[:]` (slice), `?.` (optional chaining) |
| String functions | `upper()`, `lower()`, `trim()`, `trimPrefix()`, `trimSuffix()`, `split()`, `splitAfter()`, `replace()`, `repeat()`, `indexOf()`, `lastIndexOf()`, `hasPrefix()`, `hasSuffix()` |
| String operators | `contains`, `startsWith`, `endsWith`, `matches` (regex) |
| Type conversion | `int()`, `float()`, `string()`, `type()`, `toJSON()`, `fromJSON()`, `toBase64()`, `fromBase64()`, `toPairs()`, `fromPairs()` |

These examples show expr-lang expressions used with edg's reference data. Each expression can appear anywhere an edg expression is accepted (e.g. in `args:`, `expressions:`, or inline globals). Visit the [Expressions example](https://github.com/codingconcepts/edg/tree/main/_examples/expression/) for a complete runnable demonstration.

<details>
<summary>expr-lang expression examples by category</summary>

The examples below assume the following in-memory reference data:

```yaml
reference:
  products:
    - {name: Widget, category: electronics, price: 29.99, stock: 150, active: true}
    - {name: Gadget, category: electronics, price: 49.99, stock: 80, active: true}
    - {name: Notebook, category: stationery, price: 4.99, stock: 500, active: true}
    - {name: Pen, category: stationery, price: 1.99, stock: 1000, active: true}
    - {name: Cable, category: electronics, price: 9.99, stock: 0, active: false}
  regions:
    - {code: us-east, zone: us, cities: [new_york, boston, miami]}
    - {code: eu-west, zone: eu, cities: [london, paris, dublin]}
    - {code: ap-south, zone: ap, cities: [mumbai, singapore, tokyo]}
```

| Category | Expression | Example output |
|---|---|---|
| Arithmetic | `10 % 3`<br><br>`ref_rand('products').stock % 7` | `1` -> `3` |
| Arithmetic | `2 ** 8`<br><br>`len(ref_same('regions').cities) ** 3` | `256` -> `27` |
| Arithmetic | `3 + 4 * 2`<br><br>`ref_rand('products').price + ref_rand('products').stock * 2` | `11` -> `329.99` |
| Array functions | `all(ref_same('regions').cities, {len(#) > 3})` | `true` |
| Array functions | `any(ref_same('regions').cities, {# == 'miami'})` | `true` |
| Array functions | `concat(ref_same('regions').cities, ['london', 'paris'])` | `[new_york, boston, miami, london, paris]` |
| Array functions | `filter(ref_same('regions').cities, {# startsWith 'b'})` | `[boston]` |
| Array functions | `find(ref_same('regions').cities, {# startsWith 'b'})` | `boston` |
| Array functions | `findIndex(ref_same('regions').cities, {# startsWith 'b'})` | `1` |
| Array functions | `findLast(ref_same('regions').cities, {# endsWith 'i'})` | `miami` |
| Array functions | `findLastIndex(ref_same('regions').cities, {# endsWith 'i'})` | `2` |
| Array functions | `first(ref_same('regions').cities)` | `new_york` |
| Array functions | `flatten([['new_york', 'boston'], ['london', 'paris']])`<br><br>`flatten([ref_same('regions').cities, ['sydney', 'tokyo']])` | `[new_york, boston, london, paris]` -> `[new_york, boston, miami, sydney, tokyo]` |
| Array functions | `groupBy(ref_same('regions').cities, {len(#) > 5})` | `{false: [miami, boston], true: [new_york]}` |
| Array functions | `join(ref_same('regions').cities, ', ')` | `new_york, boston, miami` |
| Array functions | `last(ref_same('regions').cities)` | `miami` |
| Array functions | `map(ref_same('regions').cities, {upper(#)})` | `[NEW_YORK, BOSTON, MIAMI]` |
| Array functions | `none(ref_same('regions').cities, {# == 'tokyo'})` | `true` |
| Array functions | `one(ref_same('regions').cities, {# == 'miami'})` | `true` |
| Array functions | `reduce([29.99, 49.99, 4.99, 1.99, 9.99], {#acc + #}, 0)`<br><br>`reduce(ref_same('regions').cities, {#acc + len(#)}, 0)` | `96.95` -> `19` |
| Array functions | `reverse(ref_same('regions').cities)` | `[miami, boston, new_york]` |
| Array functions | `sort(ref_same('regions').cities)` | `[boston, miami, new_york]` |
| Array functions | `sortBy(['Pen', 'Widget', 'Cable'], {len(#)})`<br><br>`sortBy(ref_same('regions').cities, {len(#)})` | `[Pen, Cable, Widget]` -> `[miami, boston, new_york]` |
| Array functions | `take(ref_same('regions').cities, 2)` | `[new_york, boston]` |
| Array functions | `uniq(['electronics', 'stationery', 'electronics'])`<br><br>`uniq(concat(ref_same('regions').cities, ref_same('regions').cities))` | `[electronics, stationery]` -> `[new_york, boston, miami]` |
| Bitwise | `bitand(0b1100, 0b1010)`<br><br>`bitand(ref_rand('products').stock, 0xFF)` | `8` -> `150` |
| Bitwise | `bitnot(0b1100)`<br><br>`bitnot(ref_rand('products').stock)` | `-13` -> `-151` |
| Bitwise | `bitor(0b1100, 0b1010)`<br><br>`bitor(ref_rand('products').stock, 1)` | `14` -> `151` |
| Bitwise | `bitshl(1, 4)`<br><br>`bitshl(1, len(ref_same('regions').cities))` | `16` -> `8` |
| Bitwise | `bitshr(16, 4)`<br><br>`bitshr(ref_rand('products').stock, 1)` | `1` -> `75` |
| Bitwise | `bitxor(0b1100, 0b1010)`<br><br>`bitxor(ref_rand('products').stock, 0xFF)` | `6` -> `105` |
| Comparison & logic | `not (ref_rand('products').category == 'stationery')` | `true` |
| Comparison & logic | `ref_rand('products').stock > 0 and ref_rand('products').active` | `true` |
| Conditionals | `nil ?? 'unknown'`<br><br>`ref_rand('products')?.description ?? 'none'` | `unknown` -> `none` |
| Conditionals | `ref_rand('products').stock > 0 ? 'in_stock' : 'sold_out'` | `in_stock` |
| Language | `all(ref_same('regions').cities, {len(#) > 0})` | `true` |
| Language | `get(ref_rand('products'), 'name')` | `Widget` |
| Language | `len(ref_same('regions').cities)` | `3` |
| Language | `let p = ref_rand('products'); p.price + 10` | `39.99` |
| Map functions | `keys(ref_rand('products'))` | `[name, category, price, stock, active]` |
| Map functions | `values(ref_rand('products'))` | `[Widget, electronics, 29.99, 150, true]` |
| Math functions | `abs(-7)`<br><br>`abs(ref_rand('products').stock - 200)` | `7` -> `50` |
| Math functions | `ceil(3.2)`<br><br>`ceil(ref_rand('products').price)` | `4` -> `30` |
| Math functions | `floor(ref_rand('products').price)` | `29` |
| Math functions | `mean([29.99, 49.99, 4.99, 1.99, 9.99])`<br><br>`mean(map(ref_same('regions').cities, {len(#)}))` | `19.39` -> `6.33` |
| Math functions | `median([1.99, 4.99, 9.99, 29.99, 49.99])`<br><br>`median(map(ref_same('regions').cities, {len(#)}))` | `9.99` -> `6` |
| Math functions | `round(ref_rand('products').price)` | `30` |
| Operators | `1..5`<br><br>`1..len(ref_same('regions').cities) + 1` | `[1, 2, 3, 4]` -> `[1, 2, 3]` |
| Operators | `ref_rand('products')?.name` | `Widget` |
| Operators | `ref_rand('products').price \| int` | `29` |
| Operators | `ref_same('regions').cities[0:2]` | `[new_york, boston]` |
| Operators | `ref_same('regions').zone in ['us', 'eu', 'ap']` | `true` |
| String functions | `hasPrefix(ref_same('regions').code, 'us')` | `true` |
| String functions | `hasSuffix(ref_same('regions').code, 'east')` | `true` |
| String functions | `indexOf('london', 'on')`<br><br>`indexOf(ref_rand('products').category, 'c')` | `1` -> `3` |
| String functions | `lastIndexOf('london', 'on')`<br><br>`lastIndexOf(ref_rand('products').category, 'c')` | `4` -> `9` |
| String functions | `lower(ref_rand('products').category)` | `electronics` |
| String functions | `repeat('*', 5)`<br><br>`repeat(ref_same('regions').zone, 3)` | `*****` -> `ususus` |
| String functions | `replace(ref_same('regions').code, '-', '_')` | `us_east` |
| String functions | `split('new_york,boston,miami', ',')`<br><br>`split(ref_same('regions').code, '-')` | `[new_york, boston, miami]` -> `[us, east]` |
| String functions | `splitAfter('us,eu,ap', ',')`<br><br>`splitAfter(ref_same('regions').code, '-')` | `[us,, eu,, ap]` -> `[us-, east]` |
| String functions | `trim('  gadget  ')`<br><br>`trim(ref_rand('products').name)` | `gadget` -> `Widget` |
| String functions | `trimPrefix(ref_same('regions').code, ref_same('regions').zone + '-')` | `east` |
| String functions | `trimSuffix(ref_same('regions').code, '-' + trimPrefix(ref_same('regions').code, ref_same('regions').zone + '-'))` | `us` |
| String functions | `upper(ref_rand('products').name)` | `WIDGET` |
| String operators | `ref_rand('products').category contains 'electron'` | `true` |
| String operators | `ref_same('regions').code endsWith 'east'` | `true` |
| String operators | `ref_same('regions').code matches '[a-z]+-[a-z]+'` | `true` |
| String operators | `ref_same('regions').code startsWith 'us'` | `true` |
| Type conversion | `float(ref_rand('products').stock)` | `150.0` |
| Type conversion | `fromBase64('V2lkZ2V0')`<br><br>`fromBase64(toBase64(ref_rand('products').name))` | `Widget` -> `Widget` |
| Type conversion | `fromJSON('{"code":"us-east","zone":"us"}')`<br><br>`fromJSON('{"id":' + string(ref_rand('products').stock) + '}')` | `{code: us-east, zone: us}` -> `{id: 150}` |
| Type conversion | `fromPairs([['name', 'Widget'], ['price', 29.99]])`<br><br>`fromPairs([['product', ref_rand('products').name], ['zone', ref_same('regions').zone]])` | `{name: Widget, price: 29.99}` -> `{product: Widget, zone: us}` |
| Type conversion | `int('42')`<br><br>`int(ref_rand('products').price)` | `42` -> `29` |
| Type conversion | `string(ref_rand('products').price)` | `29.99` |
| Type conversion | `toBase64(ref_rand('products').name)` | `V2lkZ2V0` |
| Type conversion | `toJSON(ref_rand('products'))` | `{"active":true,"category":"electronics","name":"Widget","price":29.99,"stock":150}` |
| Type conversion | `toPairs({name: 'Widget', price: 29.99})`<br><br>`toPairs(ref_rand('products'))` | `[[name, Widget], [price, 29.99]]` -> `[[active, true], [category, electronics], [name, Widget], [price, 29.99], [stock, 150]]` |
| Type conversion | `type(ref_rand('products').price)` | `float` |

### Advanced expressions

These combine multiple functions and reference lookups for more complex use cases.

| Description | Expression | Example output |
|---|---|---|
| Chained array ops | `join(take(sort(map(ref_same('regions').cities, {upper(#)})), 2), ', ')` | `BOSTON, MIAMI` |
| Conditional formatting | `let p = ref_rand('products'); p.stock > 100 ? upper(p.name) : lower(p.name)` | `WIDGET` |
| Conditional string ops | `let r = ref_same('regions'); hasPrefix(r.code, 'us') ? upper(first(r.cities)) : lower(last(r.cities))` | `NEW_YORK` |
| Derived metric | `let p = ref_rand('products'); int(ceil(p.price * float(p.stock) / 100))` | `45` |
| Derived slug | `let p = ref_rand('products'); replace(lower(p.name + '_' + p.category), ' ', '_')` | `widget_electronics` |
| Discount pricing | `let p = ref_rand('products'); p.active ? round(p.price * 0.9) : 0` | `27` |
| Filtered count | `len(filter(ref_same('regions').cities, {len(#) > 5}))` | `2` |
| JSON from refs | `toJSON(fromPairs([['product', ref_rand('products').name], ['zone', ref_same('regions').zone]]))` | `{"product":"Widget","zone":"us"}` |
| Multi-condition classification | `let p = ref_rand('products'); p.price > 10 and p.stock > 0 ? 'premium' : (p.active ? 'basic' : 'discontinued')` | `premium` |
| Reduce mapped values | `reduce(map(ref_same('regions').cities, {len(#)}), {#acc + #}, 0)` | `19` |

</details>

## Distributions

Several functions generate values using statistical distributions, giving you control over the shape of your random data.

### Numeric Distributions

| Function | Signature | Description |
|---|---|---|
| `exp_f` | `exp_f(rate, min, max, precision)` | Exponential decay from min |
| `lognorm_f` | `lognorm_f(mu, sigma, min, max, precision)` | Right-skewed with a long tail |
| `norm_f` | `norm_f(mean, stddev, min, max, precision)` | Bell curve centered on mean |
| `uniform` | `uniform(min, max)` | Flat distribution, every value equally likely |
| `zipf` | `zipf(s, v, max)` | Power-law skew, low values dominate |

### Set Distributions

Pick from a predefined set of values using a distribution to control which items are selected most often.

| Function | Signature | Description |
|---|---|---|
| `set_exp` | `set_exp(values, rate)` | Exponential distribution over indices; lower indices picked most often |
| `set_lognorm` | `set_lognorm(values, mu, sigma)` | Log-normal distribution over indices; right-skewed selection |
| `set_norm` | `set_norm(values, mean, stddev)` | Normal distribution over indices; `mean` index picked most often |
| `set_rand` | `set_rand(values, weights)` | Uniform or weighted random selection from a set |
| `set_zipf` | `set_zipf(values, s, v)` | Zipfian distribution over indices; strong power-law skew toward first items |

## Argument Expression Examples

These expressions are used in the `args:` list of a `run` query. Each entry in `args:` generates a value that is bound to a query parameter (`$1`, `$2`, etc.).

<details>
<summary>All argument expression examples by category</summary>

### Aggregation

| Expression | Description |
|---|---|
| `avg('fetch_products', 'price')` | Average price across all products |
| `count('fetch_products')` | Total number of rows in the dataset |
| `distinct('fetch_products', 'category_id')` | Number of distinct category IDs across all products |
| `max('fetch_products', 'price')` | Maximum price in the dataset |
| `min('fetch_products', 'price')` | Minimum price in the dataset |
| `sum('fetch_products', 'price')` | Sum of the price field across all rows |

### Batch operations

| Expression | Description |
|---|---|
| `batch(customers / batch_size)` | Drives batched execution: the parent query runs N times with $1 = 0..N-1 |
| `gen_batch(customers, batch_size, 'email')` | Generates unique emails via gofakeit, split into batches |
| `string_to_array('$1', sep)` | Splits a batch-expanded placeholder back into rows using the driver-aware separator |

> [!NOTE]
> Always use `sep` instead of a literal comma delimiter when splitting batch args. Generated values (names, addresses, etc.) can contain commas, which would silently split a single value into multiple rows and corrupt your data. `sep` uses the ASCII unit separator (char 31), which never appears in generated text.

### Binary

| Expression | Description |
|---|---|
| `bit(8)` | Random fixed-length bit string of 8 bits (e.g. `10110011`) |
| `blob(1024)` | Random 1KB blob as raw binary data (works across all databases) |
| `bytes(16)` | Random 16 bytes as a hex-encoded CockroachDB/PostgreSQL BYTES literal |
| `varbit(16)` | Random variable-length bit string of 1-16 bits |

### Conditionals & dependent columns

| Expression | Description |
|---|---|
| `arg(0) * float(arg(1))` | Compute a total from previously generated price and quantity |
| `arg(0) + " " + arg(1)` | Concatenate previously generated firstname and lastname |
| `coalesce(ref_rand('optional_data').value, 'default')` | First non-nil fallback value |
| `cond(arg(0), gen('email'), nil)` | Email if coin flip is true, NULL if false |
| `cond(gen('number:1,100') > 95, 'premium', 'standard')` | Conditional value based on a random roll |
| `nullable(gen('email'), 0.3)` | 30% chance of NULL, otherwise a random email |

### Constants & globals

| Expression | Description |
|---|---|
| `const(42)` | Always passes the integer 42 |
| `expr(warehouses * 10)` | Evaluates an arithmetic expression using globals |
| `global('warehouses')` | Looks up a global by name (equivalent to using the variable directly) |
| `warehouses * 10` | Direct global reference in an expression (equivalent to `expr(...)`) |

### Dates & times

| Expression | Description |
|---|---|
| `date('2006-01-02', '2020-01-01T00:00:00Z', '2025-01-01T00:00:00Z')` | Random date formatted as YYYY-MM-DD |
| `date_offset('-72h')` | Timestamp 72 hours in the past (e.g. for TTL or expiry columns) |
| `duration('1h', '24h')` | Random duration between 1 hour and 24 hours |
| `time('08:00:00', '18:00:00')` | Random time of day between 08:00 and 18:00 (HH:MM:SS format) |
| `timestamp('2020-01-01T00:00:00Z', '2025-01-01T00:00:00Z')` | Random timestamp between two dates (RFC3339 format) |
| `timez('09:00:00', '17:00:00')` | Random time of day with timezone suffix (for TIMETZ columns) |

### Geographic

| Expression | Description |
|---|---|
| `point(51.5074, -0.1278, 10.0).lat` | Random geographic point within 10km of London, latitude |
| `point(51.5074, -0.1278, 10.0).lon` | Random geographic point within 10km of London, longitude |
| `point_wkt(51.5074, -0.1278, 10.0)` | Random geographic point as WKT for native geometry columns |

### Identifiers

| Expression | Description |
|---|---|
| `seq(1, 1)` | Auto-incrementing sequence: 1, 2, 3, ... (per worker) |
| `seq(100, 10)` | Auto-incrementing with custom start and step: 100, 110, 120, ... |
| `uuid_v1()` | Random UUID v1 (timestamp + node ID) |
| `uuid_v4()` | Random UUID v4 (random) |
| `uuid_v6()` | Random UUID v6 (reordered timestamp) |
| `uuid_v7()` | Random UUID v7 (time-ordered, sortable) |

### JSON & arrays

| Expression | Description |
|---|---|
| `array(2, 5, 'email')` | PostgreSQL/CockroachDB array literal with 2-5 random email addresses |
| `json_arr(1, 5, 'email')` | JSON array of 1-5 random email addresses |
| `json_obj('source', 'web', 'version', 2, 'active', true)` | JSON metadata object for a JSONB column |

### Network

| Expression | Description |
|---|---|
| `inet('192.168.1.0/24')` | Random IP address within a CIDR block |

### Numeric distributions

| Expression | Description |
|---|---|
| `exp(0.5, 0, 100)` | Exponentially-distributed integer in [0, 100] |
| `exp_f(0.5, 0, 100, 2)` | Exponentially-distributed float in [0, 100] with 2 decimal places |
| `lognorm(1.0, 0.5, 1, 1000)` | Log-normally-distributed integer in [1, 1000] |
| `lognorm_f(1.0, 0.5, 1, 1000, 2)` | Log-normally-distributed float in [1, 1000] with 2 decimal places |
| `norm(4, 1, 1, 5)` | Normally-distributed integer review rating centred on 4, mostly 3-5 |
| `norm_f(50.0, 15.0, 1.0, 100.0, 2)` | Normally-distributed float price centred on 50.00, 2 decimal places |
| `norm_n(50.0, 10.0, 1, 100, 5, 10)` | 5-10 unique normally-distributed values as a comma-separated string |
| `nurand(1023, 1, customers / districts)` | Non-uniform random int using TPC-C NURand |
| `nurand_n(8191, 1, items, 5, 15)` | 5-15 unique NURand values as a comma-separated string |
| `uniform(0, 1)` | Uniform random float between 0 and 1 (e.g. for percentages) |
| `uniform_f(0.01, 999.99, 2)` | Random float between 0.01 and 999.99 with 2 decimal places |
| `zipf(2.0, 1.0, 999)` | Zipfian distribution: hot-key pattern where value 0 is most frequent |

### Random values

| Expression | Description |
|---|---|
| `bool()` | Random true or false |
| `gen('number:1,10')` | Random integer between 1 and 10 using gofakeit |
| `regex('#[0-9a-f]{6}')` | Random hex colour code |
| `regex('[A-Z]{2}[0-9]{2} [A-Z]{3}')` | Random license plate (e.g. "AB12 CDE") |
| `regex('[A-Z]{3}-[0-9]{4}')` | Product code matching a regex pattern |
| `regex('[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}')` | Random IPv4 address |
| `regex('[0-9a-f]{2}(:[0-9a-f]{2}){5}')` | Random MAC address |
| `regex('\\([0-9]{3}\\) [0-9]{3}-[0-9]{4}')` | Random US phone number |

### Reference data

| Expression | Description |
|---|---|
| `ref_diff('fetch_warehouses').w_id` | Unique row on each call within a query (no repeats) |
| `ref_each('SELECT id FROM warehouses ORDER BY id')` | Executes a SQL query; each row becomes a separate arg set |
| `ref_n('fetch_warehouses', 'id', 3, 8)` | Picks 3-8 unique random rows, returns comma-separated field values |
| `ref_perm('fetch_warehouses').w_id` | Random row pinned to this worker for its lifetime |
| `ref_rand('fetch_warehouses').w_id` | Random row from the dataset |
| `ref_same('fetch_warehouses').w_id` | Same random row for all ref_same calls within a single query execution |
| `weighted_sample_n('fetch_products', 'id', 'popularity', 3, 8)` | Pick 3-8 products weighted by their popularity column |

### Set distributions

| Expression | Description |
|---|---|
| `set_exp(['low', 'medium', 'high', 'critical'], 0.5)` | Exponential distribution; concentrates picks toward first item |
| `set_lognorm(['free', 'basic', 'pro', 'enterprise'], 0.5, 0.5)` | Log-normal distribution (right-skewed toward early indices) |
| `set_norm([1, 2, 3, 4, 5], 2, 0.8)` | Normal distribution; index 2 is most common |
| `set_rand(['1', '2', '3', '4', '5'], [5, 10, 20, 35, 30])` | Weighted random; skewed toward 4 and 5 stars |
| `set_rand(['credit_card', 'debit_card', 'paypal'], [])` | Uniform random payment method selection |
| `set_zipf(['electronics', 'clothing', 'books', 'food', 'toys'], 2.0, 1.0)` | Zipfian distribution; strong skew toward first items |

### Strings & formatting

| Expression | Description |
|---|---|
| `template('ORD-%05d', seq(1, 1))` | Formatted order number: "ORD-00001", "ORD-00002", ... |

### Vectors

| Expression | Description |
|---|---|
| `vector(32, 3, 0.3)` | 32-dimensional vector for testing; higher spread = more cluster overlap |
| `vector(384, 5, 0.1)` | pgvector-compatible 384-dimensional vector with 5 clusters and tight spread |
| `vector_norm(384, 5, 0.1, 2.0, 0.8)` | Normal centroid selection: cluster 2 is most common, bell curve falloff |
| `vector_zipf(384, 10, 0.1, 2.0, 1.0)` | Zipfian centroid selection: cluster 0 is the "hottest", realistic skew |

</details>

## Available gofakeit Patterns

These patterns can be used with `gen()`, `gen_batch()`, `json_arr()`, and `array()`. Patterns are case-insensitive. Parameters are separated from the function name by `:` and from each other by `,`.

```yaml
# No parameters
- gen('email')

# With parameters
- gen('number:1,100')

# In a batch
- gen_batch(1000, 100, 'email')

# In a JSON array
- json_arr(1, 5, 'firstname')

# In a PostgreSQL/CockroachDB array
- array(2, 4, 'word')
```

Patterns are validated at config load time. A typo like `gen('emial')` produces a clear error instead of silently returning the literal string `{emial}`.

<details>
<summary>All gofakeit patterns by category</summary>

### Address

| Pattern | Description | Example |
|---|---|---|
| `address` | Full address (street, city, state, zip, country) | `{street: 37802 Port Streetborough, city: Chesapeake, state: North Carolina, zip: 18508, country: Andorra}` |
| `city` | City name | `Mardarville` |
| `country` | Country name | `United States` |
| `countryabr` | 2-letter country code | `US` |
| `latitude` | Latitude coordinate | `41.7886` |
| `latituderange:0,90` | Latitude in range (default 0–90) | `52.31` |
| `longitude` | Longitude coordinate | `-112.0591` |
| `longituderange:0,180` | Longitude in range (default 0–180) | `73.45` |
| `state` | State name | `Idaho` |
| `stateabr` | 2-letter state abbreviation | `ID` |
| `street` | Full street (number + name + suffix) | `364 East Parkway` |
| `streetname` | Street name | `View` |
| `streetnumber` | Street number | `364` |
| `streetprefix` | Directional prefix (N, E, SW) | `East` |
| `streetsuffix` | Street type (Ave, St, Blvd) | `Parkway` |
| `timezone` | Timezone name | `America/New_York` |
| `timezoneabv` | 3-letter timezone abbreviation | `EST` |
| `timezonefull` | Full timezone name | `Eastern Standard Time` |
| `timezoneoffset` | UTC offset | `-5` |
| `timezoneregion` | Timezone region | `America` |
| `unit` | Building unit (apt, suite) | `Apt 204` |
| `zip` | Postal code | `83201` |

### Airline

| Pattern | Description | Example |
|---|---|---|
| `airlineaircrafttype` | Aircraft category | `Narrow-body` |
| `airlineairplane` | Aircraft model | `Boeing 737` |
| `airlineairport` | Airport name | `Heathrow Airport` |
| `airlineairportiata` | IATA airport code | `LHR` |
| `airlineflightnumber` | Flight number | `BA142` |
| `airlinerecordlocator` | Booking reference | `XJDF42` |
| `airlineseat` | Seat assignment | `14A` |

### Animals

| Pattern | Description | Example |
|---|---|---|
| `animal` | Animal name | `Lion` |
| `animaltype` | Animal type (mammal, bird, etc.) | `mammal` |
| `bird` | Bird species | `Eagle` |
| `cat` | Cat breed | `Siamese` |
| `dog` | Dog breed | `Labrador` |
| `farmanimal` | Farm animal | `Cow` |
| `petname` | Pet name | `Buddy` |

### Color

| Pattern | Description | Example |
|---|---|---|
| `color` | Color name | `MediumSlateBlue` |
| `hexcolor` | Hex color code | `#1a2b3c` |
| `nicecolors` | Curated color palette | `[#e8d5b7, #0e2430, ...]` |
| `rgbcolor` | RGB color values | `[52, 152, 219]` |
| `safecolor` | Web-safe color name | `fuchsia` |

### Company

| Pattern | Description | Example |
|---|---|---|
| `blurb` | Company description | `We provide scalable...` |
| `bs` | Business buzzword phrase | `synergize scalable mindshare` |
| `buzzword` | Business buzzword | `synergize` |
| `company` | Company name | `Acme Corp` |
| `companysuffix` | Company suffix (Inc., LLC) | `Inc.` |
| `job` | Job details | `{company: Google, title: Contractor, descriptor: District, level: Assurance}` |
| `jobdescriptor` | Job descriptor | `Senior` |
| `joblevel` | Job level | `Manager` |
| `jobtitle` | Job title | `Software Engineer` |
| `slogan` | Company slogan | `Think different.` |

### Contact

| Pattern | Description | Example |
|---|---|---|
| `email` | Email address | `markusmoen@pagac.net` |
| `phone` | Phone number | `6136459211` |
| `phoneformatted` | Formatted phone number | `(613) 645-9211` |
| `username` | Account username | `markus.moen` |

### Data Structures

| Pattern | Description | Example |
|---|---|---|
| `csv:,,10` | CSV rows (delimiter, row count) | `name,email\nAlice,...` |
| `fixed_width:10` | Fixed-width format | `Alice     ...` |
| `json:object,10` | JSON document (type, field count) | `{"name": "..."}` |
| `map` | Random key-value map | `{interest: 5418, only: 2991258, fly: {shall: 1188343}}` |
| `sql:,10` | SQL INSERT statements | `INSERT INTO ...` |
| `svg:500,500` | SVG image | `<svg>...</svg>` |
| `template:` | Template-driven text | *(from template)* |
| `xml:single,xml,record,10` | XML document | `<record>...</record>` |

### Date & Time

| Pattern | Description | Example |
|---|---|---|
| `date:RFC3339` | Date in specified format | `2023-07-15T14:32:07Z` |
| `daterange:2020-01-01,2025-12-31,yyyy-MM-dd` | Date in range with format | `2023-07-15` |
| `day` | Day of month | `15` |
| `futuredate` | Date in the future | `2027-03-21T10:00:00Z` |
| `hour` | Hour (0–23) | `14` |
| `minute` | Minute (0–59) | `32` |
| `month` | Month number (1–12) | `7` |
| `monthstring` | Month name | `July` |
| `nanosecond` | Nanosecond | `196519854` |
| `pastdate` | Date in the past | `2019-11-05T08:30:00Z` |
| `second` | Second (0–59) | `7` |
| `time:HH:mm:ss` | Time in format (default HH:mm:ss) | `14:32:07` |
| `timerange:08:00:00,17:00:00,HH:mm:ss` | Time in range with format | `12:45:23` |
| `weekday` | Weekday name | `Wednesday` |
| `year` | Year | `2023` |

### Emoji

| Pattern | Description | Example |
|---|---|---|
| `emoji` | Random emoji | `🎉` |
| `emojialias` | Emoji alias keyword | `:tada:` |
| `emojianimal` | Animal emoji | `🐕` |
| `emojicategory` | Emoji category | `Smileys & Emotion` |
| `emojiclothing` | Clothing emoji | `👗` |
| `emojicostume` | Costume/fantasy emoji | `🧛` |
| `emojielectronics` | Electronics emoji | `📱` |
| `emojiface` | Face/smiley emoji | `😊` |
| `emojiflag` | Flag emoji | `🇺🇸` |
| `emojifood` | Food emoji | `🍕` |
| `emojigame` | Game emoji | `🎮` |
| `emojigesture` | Gesture emoji | `🤷` |
| `emojihand` | Hand emoji | `👍` |
| `emojijob` | Job/role emoji | `👨‍🔬` |
| `emojilandmark` | Landmark emoji | `🗽` |
| `emojimusic` | Music emoji | `🎸` |
| `emojiperson` | Person emoji | `👩` |
| `emojiplant` | Plant emoji | `🌻` |
| `emojisentence:3` | Sentence with emojis (N emojis) | `I am 😊 and 🎉 today 🌟` |
| `emojisport` | Sport emoji | `⚽` |
| `emojitag` | Emoji tag | `happy` |
| `emojitools` | Tools emoji | `🔧` |
| `emojivehicle` | Vehicle emoji | `🚗` |
| `emojiweather` | Weather emoji | `☀️` |

### Entertainment

| Pattern | Description | Example |
|---|---|---|
| `book` | Book details | `{title: Sons and Lovers, author: James Joyce, genre: Saga}` |
| `bookauthor` | Book author | `F. Scott Fitzgerald` |
| `bookgenre` | Book genre | `Fiction` |
| `booktitle` | Book title | `The Great Gatsby` |
| `celebrityactor` | Celebrity actor | `Tom Hanks` |
| `celebritybusiness` | Business celebrity | `Elon Musk` |
| `celebritysport` | Sports celebrity | `Serena Williams` |
| `gamertag` | Gaming username | `xX_Slayer_Xx` |
| `hobby` | Hobby or pastime | `Photography` |
| `movie` | Movie details | `{name: Sherlock Jr., genre: Music}` |
| `moviegenre` | Movie genre | `Sci-Fi` |
| `moviename` | Movie title | `The Matrix` |
| `song` | Song details | `{name: Agora Hills, artist: Olivia Newton-John, genre: Country}` |
| `songartist` | Song artist | `Queen` |
| `songgenre` | Song genre | `Rock` |
| `songname` | Song title | `Bohemian Rhapsody` |

### Error Messages

| Pattern | Description | Example |
|---|---|---|
| `error` | Error message | `unexpected EOF` |
| `errordatabase` | Database error | `connection refused` |
| `errorgrpc` | gRPC error | `deadline exceeded` |
| `errorhttp` | HTTP error | `404 Not Found` |
| `errorhttpclient` | HTTP client error | `timeout awaiting...` |
| `errorhttpserver` | HTTP server error | `502 Bad Gateway` |
| `errorobject` | Error object | `{code: 500, message: service unavailable}` |
| `errorruntime` | Runtime error | `index out of bounds` |
| `errorvalidation` | Validation error | `field required` |

### Finance

| Pattern | Description | Example |
|---|---|---|
| `achaccount` | ACH bank account number | `586981958265` |
| `achrouting` | 9-digit ACH routing number | `071000013` |
| `bankname` | Bank name | `Chase` |
| `banktype` | Bank type | `Commercial` |
| `bitcoinaddress` | Bitcoin address | `1A1zP1eP5QGefi2D...` |
| `bitcoinprivatekey` | Bitcoin private key | `5HueCGU8rMjxEXx...` |
| `creditcard` | Full credit card details | `{type: UnionPay, number: 6376121963702920, exp: 10/29, cvv: 505}` |
| `creditcardcvv` | Credit card CVV | `513` |
| `creditcardexp` | Credit card expiry | `02/28` |
| `creditcardnumber` | Credit card number (default: any type) | `4111111111111111` |
| `creditcardtype` | Credit card type | `Visa` |
| `currency` | Currency details | `{short: ZAR, long: South Africa Rand}` |
| `currencylong` | Full currency name | `United States Dollar` |
| `currencyshort` | 3-letter currency code | `USD` |
| `cusip` | CUSIP security identifier | `38259P508` |
| `ein` | Employer Identification Number | `12-3456789` |
| `isin` | ISIN security identifier | `US38259P5081` |
| `price:0,1000` | Price in range (default 0–1000) | `42.99` |

### Food & Drink

| Pattern | Description | Example |
|---|---|---|
| `beeralcohol` | Beer alcohol content | `5.2%` |
| `beerblg` | Beer gravity (BLG) | `12.5` |
| `beerhop` | Beer hop variety | `Cascade` |
| `beeribu` | Beer bitterness (IBU) | `40` |
| `beermalt` | Beer malt type | `Pale Ale` |
| `beername` | Beer name | `Duvel` |
| `beerstyle` | Beer style | `IPA` |
| `beeryeast` | Beer yeast strain | `Safale US-05` |
| `breakfast` | Breakfast food | `Scrambled eggs` |
| `dessert` | Dessert item | `Chocolate cake` |
| `dinner` | Dinner food | `Grilled salmon` |
| `drink` | Drink | `Lemonade` |
| `fruit` | Fruit | `Apple` |
| `lunch` | Lunch food | `Caesar salad` |
| `snack` | Snack item | `Trail mix` |
| `vegetable` | Vegetable | `Broccoli` |

### Grammar

| Pattern | Description | Example |
|---|---|---|
| `adjective` | General adjective | `bright` |
| `adjectivedemonstrative` | Demonstrative adjective (this, that) | `this` |
| `adjectivedescriptive` | Descriptive adjective | `beautiful` |
| `adjectiveindefinite` | Indefinite adjective | `some` |
| `adjectiveinterrogative` | Interrogative adjective | `which` |
| `adjectivepossessive` | Possessive adjective | `our` |
| `adjectiveproper` | Proper adjective | `American` |
| `adjectivequantitative` | Quantitative adjective | `several` |
| `adverb` | General adverb | `quickly` |
| `adverbdegree` | Degree adverb | `very` |
| `adverbfrequencydefinite` | Definite frequency adverb | `daily` |
| `adverbfrequencyindefinite` | Indefinite frequency adverb | `often` |
| `adverbmanner` | Manner adverb | `carefully` |
| `adverbplace` | Place adverb | `here` |
| `adverbtimedefinite` | Definite time adverb | `yesterday` |
| `adverbtimeindefinite` | Indefinite time adverb | `soon` |
| `connective` | Connective word | `however` |
| `connectivecasual` | Causal connective | `because` |
| `connectivecomparative` | Comparative connective | `similarly` |
| `connectivecomplaint` | Complaint connective | `unfortunately` |
| `connectiveexamplify` | Example connective | `for instance` |
| `connectivelisting` | Listing connective | `firstly` |
| `connectivetime` | Time connective | `meanwhile` |
| `interjection` | Interjection | `wow` |
| `noun` | General noun | `table` |
| `nounabstract` | Abstract noun | `freedom` |
| `nouncollectiveanimal` | Animal collective noun | `flock` |
| `nouncollectivepeople` | People collective noun | `crowd` |
| `nouncollectivething` | Thing collective noun | `bundle` |
| `nouncommon` | Common noun | `book` |
| `nounconcrete` | Concrete noun | `chair` |
| `nouncountable` | Countable noun | `apple` |
| `noundeterminer` | Noun determiner | `the` |
| `nounproper` | Proper noun | `London` |
| `noununcountable` | Uncountable noun | `water` |
| `preposition` | General preposition | `with` |
| `prepositioncompound` | Compound preposition | `in front of` |
| `prepositiondouble` | Double preposition | `out of` |
| `prepositionsimple` | Simple preposition | `at` |
| `pronoun` | General pronoun | `she` |
| `pronoundemonstrative` | Demonstrative pronoun | `these` |
| `pronounindefinite` | Indefinite pronoun | `someone` |
| `pronouninterrogative` | Interrogative pronoun | `who` |
| `pronounobject` | Object pronoun | `him` |
| `pronounpersonal` | Personal pronoun | `I` |
| `pronounpossessive` | Possessive pronoun | `mine` |
| `pronounreflective` | Reflexive pronoun | `myself` |
| `pronounrelative` | Relative pronoun | `which` |
| `verb` | General verb | `run` |
| `verbaction` | Action verb | `jump` |
| `verbhelping` | Helping verb | `would` |
| `verbintransitive` | Intransitive verb | `sleep` |
| `verblinking` | Linking verb | `seem` |
| `verbtransitive` | Transitive verb | `carry` |

### Hacker

| Pattern | Description | Example |
|---|---|---|
| `hackerabbreviation` | Hacker abbreviation | `SQL` |
| `hackeradjective` | Hacker adjective | `back-end` |
| `hackeringverb` | Hacker -ing verb | `hacking` |
| `hackernoun` | Hacker noun | `firewall` |
| `hackerphrase` | Hacker phrase | `Use the neural TCP...` |
| `hackerverb` | Hacker verb | `parse` |

### Internet

| Pattern | Description | Example |
|---|---|---|
| `apiuseragent` | API client user agent | `curl/7.68.0` |
| `chromeuseragent` | Chrome user agent | `Mozilla/5.0 ... Chrome/...` |
| `domainname` | Domain name | `example.com` |
| `domainsuffix` | Domain suffix | `.com` |
| `firefoxuseragent` | Firefox user agent | `Mozilla/5.0 ... Firefox/...` |
| `httpmethod` | HTTP method | `GET` |
| `httpstatuscode` | HTTP status code | `404` |
| `httpstatuscodesimple` | Common HTTP status code | `200` |
| `httpversion` | HTTP version | `1.1` |
| `inputname` | HTML input element name | `first_name` |
| `ipv4address` | IPv4 address | `192.168.1.42` |
| `ipv6address` | IPv6 address | `2001:db8::1` |
| `macaddress` | MAC address | `00:1A:2B:3C:4D:5E` |
| `operauseragent` | Opera user agent | `Mozilla/5.0 ... OPR/...` |
| `safariuseragent` | Safari user agent | `Mozilla/5.0 ... Safari/...` |
| `url` | Web URL | `https://www.example.com/path` |
| `urlslug:3` | URL-safe slug (N words, default 3) | `modern-web-design` |
| `useragent` | Browser user agent string | `Mozilla/5.0 ...` |

### Language

| Pattern | Description | Example |
|---|---|---|
| `language` | Language name | `English` |
| `languageabbreviation` | Language abbreviation | `en` |
| `languagebcp` | BCP 47 language tag | `en-US` |
| `programminglanguage` | Programming language | `Go` |

### Minecraft

| Pattern | Description | Example |
|---|---|---|
| `minecraftanimal` | Minecraft animal | `Cow` |
| `minecraftarmorpart` | Armor piece | `Chestplate` |
| `minecraftarmortier` | Armor tier | `Diamond` |
| `minecraftbiome` | Minecraft biome | `Plains` |
| `minecraftdye` | Minecraft dye color | `Red` |
| `minecraftfood` | Minecraft food | `Bread` |
| `minecraftmobboss` | Boss mob | `Ender Dragon` |
| `minecraftmobhostile` | Hostile mob | `Creeper` |
| `minecraftmobneutral` | Neutral mob | `Enderman` |
| `minecraftmobpassive` | Passive mob | `Sheep` |
| `minecraftore` | Minecraft ore | `Diamond` |
| `minecrafttool` | Minecraft tool | `Pickaxe` |
| `minecraftvillagerjob` | Villager job | `Librarian` |
| `minecraftvillagerlevel` | Villager level | `Journeyman` |
| `minecraftvillagerstation` | Villager station | `Lectern` |
| `minecraftweapon` | Minecraft weapon | `Sword` |
| `minecraftweather` | Minecraft weather | `Rain` |
| `minecraftwood` | Minecraft wood type | `Oak` |

### Miscellaneous

| Pattern | Description | Example |
|---|---|---|
| `email_text` | Email message body | `Dear Sir/Madam...` |
| `imagejpeg:500,500` | Random JPEG image (W×H) | *(binary image data)* |
| `imagepng:500,500` | Random PNG image (W×H) | *(binary image data)* |
| `loglevel` | Log severity level | `error` |
| `password:true,true,true,true,false,12` | Password (lower, upper, numeric, special, space, length) | `aB3$kL9mPq2x` |
| `randomint:` | Random pick from int array | *(selected int)* |
| `randomstring:` | Random pick from string array | *(selected string)* |
| `randomuint:` | Random pick from uint array | *(selected uint)* |
| `shuffleints:` | Shuffle int array | *(shuffled array)* |
| `shufflestrings:` | Shuffle string array | *(shuffled array)* |
| `teams:,` | Split people into teams | *(team assignments)* |
| `weighted:,` | Weighted random selection | *(selected value)* |

### Numbers

| Pattern | Description | Example |
|---|---|---|
| `bool` | true or false | `true` |
| `dice:2,[6,6]` | Dice roll (count, sides per die) | `[4, 2]` |
| `digit` | Single digit (0–9) | `7` |
| `digitn:6` | String of N digits | `482910` |
| `flipacoin` | Coin toss | `Heads` |
| `float32` | 32-bit float | `3.14` |
| `float32range:1,10` | 32-bit float in range | `7.23` |
| `float64` | 64-bit float | `3.141592653` |
| `float64range:0,1` | 64-bit float in range | `0.7312` |
| `hexuint:8` | Hex unsigned integer (N hex chars) | `4a3f2b1c` |
| `int` | Random signed integer | `8294723` |
| `int8` | Signed 8-bit integer (−128 to 127) | `42` |
| `int16` | Signed 16-bit integer | `8294` |
| `int32` | Signed 32-bit integer | `829472389` |
| `int64` | Signed 64-bit integer | `8294723891234` |
| `intn:100` | Integer in [0, N) | `73` |
| `intrange:1,100` | Signed integer in range | `67` |
| `number:1,100` | Integer in range (default full int32 range) | `42` |
| `uint` | Unsigned integer | `4294967` |
| `uint8` | Unsigned 8-bit integer (0–255) | `200` |
| `uint16` | Unsigned 16-bit integer (0–65535) | `50000` |
| `uint32` | Unsigned 32-bit integer | `2948293` |
| `uint64` | Unsigned 64-bit integer | `394857239482` |
| `uintn:100` | Unsigned integer in [0, N) | `42` |
| `uintrange:0,1000` | Unsigned integer in range | `512` |

### Person

| Pattern | Description | Example |
|---|---|---|
| `age` | Age in years | `32` |
| `bio` | Random biography | `I'm a developer from NY...` |
| `ethnicity` | Cultural or ethnic background | `Caucasian` |
| `firstname` | Given name | `Markus` |
| `gender` | Gender classification | `male` |
| `lastname` | Family name | `Moen` |
| `middlename` | Middle name | `James` |
| `name` | Full name (first and last) | `Markus Moen` |
| `nameprefix` | Title or honorific (Mr., Mrs., Dr.) | `Mr.` |
| `namesuffix` | Suffix (Jr., Sr., III) | `Jr.` |
| `person` | Full personal details (name, contact, etc.) | `{first_name: Jessica, last_name: Hills, gender: female, age: 51, ssn: 961445393, ...}` |
| `ssn` | US Social Security Number | `296-28-1925` |

### Product

| Pattern | Description | Example |
|---|---|---|
| `product` | Product details | `{name: Water Dispenser, price: 91.59, material: cardboard, upc: 058601249007, ...}` |
| `productaudience` | Target audience | `Professionals` |
| `productbenefit` | Key product benefit | `Time-saving` |
| `productcategory` | Product category | `Electronics` |
| `productdescription` | Product description | `High-quality wireless...` |
| `productdimension` | Product dimensions | `10x5x3 inches` |
| `productfeature` | Product feature | `Waterproof` |
| `productisbn` | ISBN identifier | `978-3-16-148410-0` |
| `productmaterial` | Product material | `Stainless Steel` |
| `productname` | Product name | `Ergonomic Keyboard` |
| `productsuffix` | Product model suffix | `Pro` |
| `productupc` | UPC barcode | `012345678905` |
| `productusecase` | Product use case | `Office productivity` |

### School

| Pattern | Description | Example |
|---|---|---|
| `school` | School name | `Lincoln High School` |

### Social Media

| Pattern | Description | Example |
|---|---|---|
| `socialmedia` | Social media handle/URL | `@johndoe` |

### String Manipulation

| Pattern | Description | Example |
|---|---|---|
| `generate:{firstname} {lastname}` | Generate from template | `Alice Smith` |
| `id` | Short URL-safe base32 identifier | `01hzxq5v8k` |
| `lexify:???` | Replace `?` with random letters | `kqb` |
| `numerify:###` | Replace `#` with random digits | `482` |
| `regex:[A-Z]{3}-[0-9]{4}` | String matching regex | `ABK-7291` |
| `uuid` | RFC 4122 v4 UUID | `550e8400-e29b-41d4-a716-446655440000` |

### Text

| Pattern | Description | Example |
|---|---|---|
| `comment` | Comment or remark | `This is great work!` |
| `hipsterparagraph:2,5,1,\n` | Hipster paragraph | *(multi-sentence hipster text)* |
| `hipstersentence:5` | Hipster sentence (N words) | `Artisan cold-pressed vegan...` |
| `hipsterword` | Hipster vocabulary word | `artisan` |
| `letter` | Single ASCII letter | `g` |
| `lettern:8` | String of N letters | `abcqwzml` |
| `loremipsumparagraph:2,5,1,\n` | Lorem Ipsum paragraph | *(multi-sentence Lorem Ipsum)* |
| `loremipsumsentence:5` | Lorem Ipsum sentence (N words) | `Lorem ipsum dolor sit amet.` |
| `loremipsumword` | Lorem Ipsum word | `lorem` |
| `markdown` | Markdown-formatted text | *(formatted markdown)* |
| `paragraph:3,5,12,\n` | Paragraph (sentences, words, paragraphs, separator) | *(multi-sentence text)* |
| `phrase` | Short phrase | `a quiet afternoon` |
| `phraseadverb` | Adverb phrase | `very carefully` |
| `phrasenoun` | Noun phrase | `the old house` |
| `phrasepreposition` | Prepositional phrase | `in the garden` |
| `phraseverb` | Verb phrase | `runs quickly` |
| `question` | Question sentence | `Where did you go?` |
| `quote` | Quoted text | `"To be or not to be"` |
| `sentence:5` | Sentence with N words (default 5) | `The quick brown fox jumped.` |
| `vowel` | Single vowel | `e` |
| `word` | Random word | `themselves` |

### Vehicle

| Pattern | Description | Example |
|---|---|---|
| `car` | Car details | `{type: Passenger car heavy, fuel: Ethanol, transmission: Automatic, brand: Alfa Romeo, model: Lancer, year: 2014}` |
| `carfueltype` | Fuel type | `Electric` |
| `carmaker` | Car manufacturer | `Toyota` |
| `carmodel` | Car model | `Camry` |
| `cartransmissiontype` | Transmission type | `Automatic` |
| `cartype` | Car type | `Sedan` |

</details>
