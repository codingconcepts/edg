---
title: Expressions
layout: default
nav_order: 5
---

# Expressions

Query arguments are written as expressions compiled at startup using [expr-lang/expr](https://github.com/expr-lang/expr). Each expression has access to the built-in functions, globals, and any user-defined expressions.

## Functions

| Function | Returns | Description |
|---|---|---|
| `const(value)` | `any` | Returns the value as-is. Useful for literal constants. |
| `expr(expression)` | `any` | Evaluates an arithmetic expression. Alias for `const`, the expr engine handles the arithmetic. |
| `gen(pattern)` | `string` | Generates a random value using [gofakeit](https://github.com/brianvoe/gofakeit) patterns (e.g. `gen('number:1,100')`). |
| `global(name)` | `any` | Looks up a value from the `globals` section by name. Globals are also available directly as variables, so `global('warehouses')` and `warehouses` are equivalent. |
| `ref_rand(name)` | `map` | Returns a random row from a named dataset (populated by an `init` query). Access fields with dot notation: `ref_rand('fetch_warehouses').w_id`. |
| `ref_same(name)` | `map` | Returns a random row, but the same row is reused across all `ref_same` calls within a single query execution. Cleared between iterations. |
| `ref_perm(name)` | `map` | Returns a random row on first call, then the same row for the entire lifetime of the worker. |
| `ref_diff(name)` | `map` | Returns unique rows across multiple calls within the same query execution. Uses a swap-based index to avoid repeats. |
| `batch(n)` | `[][]any` | Returns sequential integers `[0, n)` as batch arg sets, |
| `gen_batch(total, batchSize, pattern)` | `[][]any` | Generates `total` values using [gofakeit](https://github.com/brianvoe/gofakeit) `pattern`, grouped into batches of `batchSize`. Each batch arg is a comma-separated string of generated values. |
| `ref_each(query)` | `[][]any` | Executes a SQL query and returns all rows. Each row becomes a separate arg set. |
| `ref_n(name, field, min, max)` | `string` | Picks N unique random rows (N in [min, max]) from a named dataset, extracts `field` from each, and returns a comma-separated string. |
| `nurand(A, x, y)` | `int` | TPC-C Non-Uniform Random: `(((random(0,A) \| random(x,y)) + C) / (y-x+1)) + x`. |
| `nurand_n(A, x, y, min, max)` | `string` | Generates N unique NURand values (N in [min, max]) as a comma-separated string. |
| `set_rand(values, weights)` | `any` | Picks a random item from a set. If weights are provided, weighted random selection is used; otherwise uniform. |
| `set_norm(values, mean, stddev)` | `any` | Picks an item from a set using normal distribution. |
| `set_exp(values, rate)` | `any` | Picks an item from a set using exponential distribution. |
| `set_lognorm(values, mu, sigma)` | `any` | Picks an item from a set using log-normal distribution. |
| `set_zipf(values, s, v)` | `any` | Picks an item from a set using Zipfian distribution. |
| `exp(rate, min, max)` | `float64` | Exponentially-distributed random number in [min, max], rounded to 0 decimal places. |
| `exp_f(rate, min, max, precision)` | `float64` | Exponentially-distributed random number in [min, max], rounded to `precision` decimal places. |
| `lognorm(mu, sigma, min, max)` | `float64` | Log-normally-distributed random number in [min, max], rounded to 0 decimal places. |
| `lognorm_f(mu, sigma, min, max, precision)` | `float64` | Log-normally-distributed random number in [min, max], rounded to `precision` decimal places. |
| `norm(mean, stddev, min, max)` | `float64` | Normally-distributed random number in [min, max], rounded to 0 decimal places. |
| `norm_f(mean, stddev, min, max, precision)` | `float64` | Normally-distributed random number in [min, max], rounded to `precision` decimal places. |
| `norm_n(mean, stddev, min, max, minN, maxN)` | `string` | N unique normally-distributed values (N in [minN, maxN]) as a comma-separated string. |
| `uuid_v1()` | `string` | Generates a Version 1 UUID (timestamp + node ID). |
| `uuid_v4()` | `string` | Generates a Version 4 UUID (random). |
| `uuid_v6()` | `string` | Generates a Version 6 UUID (reordered timestamp). |
| `uuid_v7()` | `string` | Generates a Version 7 UUID (Unix timestamp + random, sortable). |
| `uniform_f(min, max, precision)` | `float64` | Uniform random float in [min, max] rounded to `precision` decimal places. |
| `uniform(min, max)` | `float64` | Uniform random float in [min, max]. |
| `seq(start, step)` | `int` | Auto-incrementing sequence per worker. Returns `start + counter * step`. |
| `zipf(s, v, max)` | `int` | Zipfian-distributed random integer in [0, max]. |
| `cond(predicate, trueVal, falseVal)` | `any` | Returns `trueVal` if `predicate` is true, `falseVal` otherwise. |
| `coalesce(v1, v2, ...)` | `any` | Returns the first non-nil value from arguments. |
| `template(format, args...)` | `string` | Formats a string using Go's `fmt.Sprintf` syntax. |
| `regex(pattern)` | `string` | Generates a random string matching the given regular expression. |
| `json_obj(k1, v1, k2, v2, ...)` | `string` | Builds a JSON object string from key-value pair arguments. |
| `json_arr(minN, maxN, pattern)` | `string` | Builds a JSON array of N random values (N in [minN, maxN]) generated by a gofakeit `pattern`. |
| `bytes(n)` | `string` | Random `n` bytes as a hex-encoded string with `\x` prefix. |
| `bit(n)` | `string` | Random fixed-length bit string of exactly `n` bits. |
| `varbit(n)` | `string` | Random variable-length bit string of 1 to `n` bits. |
| `inet(cidr)` | `string` | Random IP address within the given CIDR block. |
| `array(minN, maxN, pattern)` | `string` | PostgreSQL/CockroachDB array literal with a random number of elements. |
| `time(min, max)` | `string` | Random time of day between `min` and `max` (HH:MM:SS format). |
| `timez(min, max)` | `string` | Random time of day with `+00:00` timezone suffix. |
| `point(lat, lon, radiusKM)` | `map` | Generates a random geographic point within `radiusKM` of (`lat`, `lon`). Access fields with `.lat` and `.lon`. |
| `point_wkt(lat, lon, radiusKM)` | `string` | Generates a random geographic point as a WKT string: `POINT(lon lat)`. |
| `timestamp(min, max)` | `string` | Random timestamp between `min` and `max` (RFC3339). |
| `duration(min, max)` | `string` | Random duration between `min` and `max` (Go duration strings). |
| `date(format, min, max)` | `string` | Random timestamp formatted using a Go time format string. |
| `date_offset(duration)` | `string` | Returns the current time offset by `duration`, formatted as RFC3339. |
| `weighted_sample_n(name, field, weightField, minN, maxN)` | `string` | Picks N unique rows using weighted selection, returns a comma-separated string. |
| `sum(name, field)` | `float64` | Sum of a numeric field across all rows in a named dataset. |
| `avg(name, field)` | `float64` | Average of a numeric field across all rows in a named dataset. |
| `min(name, field)` | `float64` | Minimum value of a numeric field in a named dataset. |
| `max(name, field)` | `float64` | Maximum value of a numeric field in a named dataset. |
| `count(name)` | `int` | Number of rows in a named dataset. |
| `distinct(name, field)` | `int` | Number of distinct values for a field in a named dataset. |

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

Expressions support the full expr-lang feature set, including:

| Category | Examples |
|---|---|
| Arithmetic | `+`, `-`, `*`, `/`, `%`, `**` |
| Comparison & logic | `==`, `!=`, `<`, `>`, `and`, `or`, `not` |
| Conditionals | `args[0] > 10 ? 'yes' : 'no'`, `args[0] ?? 0` (nil coalescing), `if`/`else` |
| Math functions | `abs()`, `ceil()`, `floor()`, `round()`, `mean()`, `median()` |
| String functions | `upper()`, `lower()`, `trim()`, `trimPrefix()`, `trimSuffix()`, `split()`, `splitAfter()`, `replace()`, `repeat()`, `indexOf()`, `lastIndexOf()`, `hasPrefix()`, `hasSuffix()` |
| String operators | `contains`, `startsWith`, `endsWith`, `matches` (regex) |
| Array functions | `filter()`, `map()`, `reduce()`, `sort()`, `sortBy()`, `reverse()`, `first()`, `last()`, `take()`, `flatten()`, `uniq()`, `concat()`, `join()`, `find()`, `findIndex()`, `findLast()`, `findLastIndex()`, `all()`, `any()`, `one()`, `none()`, `groupBy()` |
| Map functions | `keys()`, `values()` |
| Type conversion | `int()`, `float()`, `string()`, `type()`, `toJSON()`, `fromJSON()`, `toBase64()`, `fromBase64()`, `toPairs()`, `fromPairs()` |
| Bitwise | `bitand()`, `bitor()`, `bitxor()`, `bitnand()`, `bitnot()`, `bitshl()`, `bitshr()`, `bitushr()` |
| Operators | `\|` (pipe), `in` (membership), `..` (range), `[:]` (slice), `?.` (optional chaining) |
| Language | `let` bindings, `#` predicates (current element in closures), `len()`, `get()` |

See the [Expressions example](https://github.com/codingconcepts/edg/tree/main/_examples/expression/) for a complete demonstration of every category.
