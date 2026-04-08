---
title: Expressions
weight: 5
---

# Expressions

Query arguments are written as expressions compiled at startup using [expr-lang/expr](https://github.com/expr-lang/expr). Each expression has access to the built-in functions, globals, and any user-defined expressions.

> **Tip:** Use `edg repl` to try any expression interactively without a database connection. See [REPL]({{< relref "repl" >}}) for details.

## Functions

| Function | Returns | Description |
|---|---|---|
| `arg(index)` | `any` | Returns the value of a previously evaluated arg by its zero-based index. Enables dependent columns where later args reference earlier ones. |
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

## Function Lifecycle

Several functions maintain state. Understanding when that state resets is important for getting correct results:

| Function | Scope | Resets |
|---|---|---|
| `arg(index)` | Per-query | Returns the value of arg at `index` from the current query execution. Cleared before the next query. In batch queries, resets per row. |
| `ref_rand(name)` | None | Fresh random row on every call |
| `ref_same(name)` | Per-query | Picks a row on first call within a query; all subsequent `ref_same` calls for the same dataset within that query return the same row. **Cleared before the next query.** |
| `ref_perm(name)` | Per-worker | Picks a row on first call and returns that same row for the entire lifetime of the worker. Never resets. |
| `ref_diff(name)` | Per-query | Returns a unique row on each call within a query (no repeats). Index resets before the next query. |
| `seq(start, step)` | Per-worker | Counter starts at 0 for each worker and increments on every call. Two workers both calling `seq(1, 1)` will produce the same sequence independently -- values are **not globally unique**. |
| `nurand(A, x, y)` | Per-worker | The TPC-C constant C is generated once per worker per A value and stays fixed for the worker's lifetime. |

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

## expr examples

Here are some example expressions and their outputs but visit the [Expressions example](https://github.com/codingconcepts/edg/tree/main/_examples/expression/) for a complete demonstration of every category.

Assume that the following is available to the functions that would benefit from having in-memory reference data:

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
| Arithmetic | `3 + 4 * 2`<br><br>`ref_rand('products').price + ref_rand('products').stock * 2` | `11`<br><br>`329.99` |
| Arithmetic | `10 % 3`<br><br>`ref_rand('products').stock % 7` | `1`<br><br>`3` |
| Arithmetic | `2 ** 8`<br><br>`len(ref_same('regions').cities) ** 3` | `256`<br><br>`27` |
| Comparison & logic | `ref_rand('products').stock > 0 and ref_rand('products').active` | `true` |
| Comparison & logic | `not (ref_rand('products').category == 'stationery')` | `true` |
| Conditionals | `ref_rand('products').stock > 0 ? 'in_stock' : 'sold_out'` | `in_stock` |
| Conditionals | `nil ?? 'unknown'`<br><br>`ref_rand('products')?.description ?? 'none'` | `unknown`<br><br>`none` |
| Math functions | `abs(-7)`<br><br>`abs(ref_rand('products').stock - 200)` | `7`<br><br>`50` |
| Math functions | `ceil(3.2)`<br><br>`ceil(ref_rand('products').price)` | `4`<br><br>`30` |
| Math functions | `floor(ref_rand('products').price)` | `29` |
| Math functions | `round(ref_rand('products').price)` | `30` |
| Math functions | `mean([29.99, 49.99, 4.99, 1.99, 9.99])`<br><br>`mean(map(ref_same('regions').cities, {len(#)}))` | `19.39`<br><br>`6.33` |
| Math functions | `median([1.99, 4.99, 9.99, 29.99, 49.99])`<br><br>`median(map(ref_same('regions').cities, {len(#)}))` | `9.99`<br><br>`6` |
| String functions | `upper(ref_rand('products').name)` | `WIDGET` |
| String functions | `lower(ref_rand('products').category)` | `electronics` |
| String functions | `trim('  gadget  ')`<br><br>`trim(ref_rand('products').name)` | `gadget`<br><br>`Widget` |
| String functions | `trimPrefix(ref_same('regions').code, ref_same('regions').zone + '-')` | `east` |
| String functions | `trimSuffix(ref_same('regions').code, '-' + trimPrefix(ref_same('regions').code, ref_same('regions').zone + '-'))` | `us` |
| String functions | `split('new_york,boston,miami', ',')`<br><br>`split(ref_same('regions').code, '-')` | `[new_york, boston, miami]`<br><br>`[us, east]` |
| String functions | `splitAfter('us,eu,ap', ',')`<br><br>`splitAfter(ref_same('regions').code, '-')` | `[us,, eu,, ap]`<br><br>`[us-, east]` |
| String functions | `replace(ref_same('regions').code, '-', '_')` | `us_east` |
| String functions | `repeat('*', 5)`<br><br>`repeat(ref_same('regions').zone, 3)` | `*****`<br><br>`ususus` |
| String functions | `indexOf('london', 'on')`<br><br>`indexOf(ref_rand('products').category, 'c')` | `1`<br><br>`3` |
| String functions | `lastIndexOf('london', 'on')`<br><br>`lastIndexOf(ref_rand('products').category, 'c')` | `4`<br><br>`9` |
| String functions | `hasPrefix(ref_same('regions').code, 'us')` | `true` |
| String functions | `hasSuffix(ref_same('regions').code, 'east')` | `true` |
| String operators | `ref_rand('products').category contains 'electron'` | `true` |
| String operators | `ref_same('regions').code startsWith 'us'` | `true` |
| String operators | `ref_same('regions').code endsWith 'east'` | `true` |
| String operators | `ref_same('regions').code matches '[a-z]+-[a-z]+'` | `true` |
| Array functions | `filter(ref_same('regions').cities, {# startsWith 'b'})` | `[boston]` |
| Array functions | `map(ref_same('regions').cities, {upper(#)})` | `[NEW_YORK, BOSTON, MIAMI]` |
| Array functions | `reduce([29.99, 49.99, 4.99, 1.99, 9.99], {#acc + #}, 0)`<br><br>`reduce(ref_same('regions').cities, {#acc + len(#)}, 0)` | `96.95`<br><br>`19` |
| Array functions | `sort(ref_same('regions').cities)` | `[boston, miami, new_york]` |
| Array functions | `sortBy(['Pen', 'Widget', 'Cable'], {len(#)})`<br><br>`sortBy(ref_same('regions').cities, {len(#)})` | `[Pen, Cable, Widget]`<br><br>`[miami, boston, new_york]` |
| Array functions | `reverse(ref_same('regions').cities)` | `[miami, boston, new_york]` |
| Array functions | `first(ref_same('regions').cities)` | `new_york` |
| Array functions | `last(ref_same('regions').cities)` | `miami` |
| Array functions | `take(ref_same('regions').cities, 2)` | `[new_york, boston]` |
| Array functions | `flatten([['new_york', 'boston'], ['london', 'paris']])`<br><br>`flatten([ref_same('regions').cities, ['sydney', 'tokyo']])` | `[new_york, boston, london, paris]`<br><br>`[new_york, boston, miami, sydney, tokyo]` |
| Array functions | `uniq(['electronics', 'stationery', 'electronics'])`<br><br>`uniq(concat(ref_same('regions').cities, ref_same('regions').cities))` | `[electronics, stationery]`<br><br>`[new_york, boston, miami]` |
| Array functions | `concat(ref_same('regions').cities, ['london', 'paris'])` | `[new_york, boston, miami, london, paris]` |
| Array functions | `join(ref_same('regions').cities, ', ')` | `new_york, boston, miami` |
| Array functions | `find(ref_same('regions').cities, {# startsWith 'b'})` | `boston` |
| Array functions | `findIndex(ref_same('regions').cities, {# startsWith 'b'})` | `1` |
| Array functions | `findLast(ref_same('regions').cities, {# endsWith 'i'})` | `miami` |
| Array functions | `findLastIndex(ref_same('regions').cities, {# endsWith 'i'})` | `2` |
| Array functions | `all(ref_same('regions').cities, {len(#) > 3})` | `true` |
| Array functions | `any(ref_same('regions').cities, {# == 'miami'})` | `true` |
| Array functions | `one(ref_same('regions').cities, {# == 'miami'})` | `true` |
| Array functions | `none(ref_same('regions').cities, {# == 'tokyo'})` | `true` |
| Array functions | `groupBy(ref_same('regions').cities, {len(#) > 5})` | `{false: [miami, boston], true: [new_york]}` |
| Map functions | `keys(ref_rand('products'))` | `[name, category, price, stock, active]` |
| Map functions | `values(ref_rand('products'))` | `[Widget, electronics, 29.99, 150, true]` |
| Type conversion | `int('42')`<br><br>`int(ref_rand('products').price)` | `42`<br><br>`29` |
| Type conversion | `float(ref_rand('products').stock)` | `150.0` |
| Type conversion | `string(ref_rand('products').price)` | `29.99` |
| Type conversion | `type(ref_rand('products').price)` | `float` |
| Type conversion | `toJSON(ref_rand('products'))` | `{"active":true,"category":"electronics","name":"Widget","price":29.99,"stock":150}` |
| Type conversion | `fromJSON('{"code":"us-east","zone":"us"}')`<br><br>`fromJSON('{"id":' + string(ref_rand('products').stock) + '}')` | `{code: us-east, zone: us}`<br><br>`{id: 150}` |
| Type conversion | `toBase64(ref_rand('products').name)` | `V2lkZ2V0` |
| Type conversion | `fromBase64('V2lkZ2V0')`<br><br>`fromBase64(toBase64(ref_rand('products').name))` | `Widget`<br><br>`Widget` |
| Type conversion | `toPairs({name: 'Widget', price: 29.99})`<br><br>`toPairs(ref_rand('products'))` | `[[name, Widget], [price, 29.99]]`<br><br>`[[active, true], [category, electronics], [name, Widget], [price, 29.99], [stock, 150]]` |
| Type conversion | `fromPairs([['name', 'Widget'], ['price', 29.99]])`<br><br>`fromPairs([['product', ref_rand('products').name], ['zone', ref_same('regions').zone]])` | `{name: Widget, price: 29.99}`<br><br>`{product: Widget, zone: us}` |
| Bitwise | `bitand(0b1100, 0b1010)`<br><br>`bitand(ref_rand('products').stock, 0xFF)` | `8`<br><br>`150` |
| Bitwise | `bitor(0b1100, 0b1010)`<br><br>`bitor(ref_rand('products').stock, 1)` | `14`<br><br>`151` |
| Bitwise | `bitxor(0b1100, 0b1010)`<br><br>`bitxor(ref_rand('products').stock, 0xFF)` | `6`<br><br>`105` |
| Bitwise | `bitnot(0b1100)`<br><br>`bitnot(ref_rand('products').stock)` | `-13`<br><br>`-151` |
| Bitwise | `bitshl(1, 4)`<br><br>`bitshl(1, len(ref_same('regions').cities))` | `16`<br><br>`8` |
| Bitwise | `bitshr(16, 4)`<br><br>`bitshr(ref_rand('products').stock, 1)` | `1`<br><br>`75` |
| Operators | `ref_rand('products').price \| int` | `29` |
| Operators | `ref_same('regions').zone in ['us', 'eu', 'ap']` | `true` |
| Operators | `1..5`<br><br>`1..len(ref_same('regions').cities) + 1` | `[1, 2, 3, 4]`<br><br>`[1, 2, 3]` |
| Operators | `ref_same('regions').cities[0:2]` | `[new_york, boston]` |
| Operators | `ref_rand('products')?.name` | `Widget` |
| Language | `let p = ref_rand('products'); p.price + 10` | `39.99` |
| Language | `all(ref_same('regions').cities, {len(#) > 0})` | `true` |
| Language | `len(ref_same('regions').cities)` | `3` |
| Language | `get(ref_rand('products'), 'name')` | `Widget` |

### Advanced expr expressions

These examples combine multiple functions and reference lookups to show more advanced usage patterns.

| Description | Expression | Example output |
|---|---|---|
| Conditional formatting | `let p = ref_rand('products'); p.stock > 100 ? upper(p.name) : lower(p.name)` | `WIDGET` |
| Discount pricing | `let p = ref_rand('products'); p.active ? round(p.price * 0.9) : 0` | `27` |
| Multi-condition classification | `let p = ref_rand('products'); p.price > 10 and p.stock > 0 ? 'premium' : (p.active ? 'basic' : 'discontinued')` | `premium` |
| Derived metric | `let p = ref_rand('products'); int(ceil(p.price * float(p.stock) / 100))` | `45` |
| Derived slug | `let p = ref_rand('products'); replace(lower(p.name + '_' + p.category), ' ', '_')` | `widget_electronics` |
| Conditional string ops | `let r = ref_same('regions'); hasPrefix(r.code, 'us') ? upper(first(r.cities)) : lower(last(r.cities))` | `NEW_YORK` |
| Chained array ops | `join(take(sort(map(ref_same('regions').cities, {upper(#)})), 2), ', ')` | `BOSTON, MIAMI` |
| Reduce mapped values | `reduce(map(ref_same('regions').cities, {len(#)}), {#acc + #}, 0)` | `19` |
| Filtered count | `len(filter(ref_same('regions').cities, {len(#) > 5}))` | `2` |
| JSON from refs | `toJSON(fromPairs([['product', ref_rand('products').name], ['zone', ref_same('regions').zone]]))` | `{"product":"Widget","zone":"us"}` |

## Distributions

Several functions generate values using statistical distributions, giving you control over the shape of your random data.

### Numeric Distributions

| Function | Signature | Description |
|---|---|---|
| `uniform` | `uniform(min, max)` | Flat distribution, every value equally likely |
| `zipf` | `zipf(s, v, max)` | Power-law skew, low values dominate |
| `norm_f` | `norm_f(mean, stddev, min, max, precision)` | Bell curve centered on mean |
| `exp_f` | `exp_f(rate, min, max, precision)` | Exponential decay from min |
| `lognorm_f` | `lognorm_f(mu, sigma, min, max, precision)` | Right-skewed with a long tail |

### Set Distributions

Pick from a predefined set of values using a distribution to control which items are selected most often.

| Function | Signature | Description |
|---|---|---|
| `set_rand` | `set_rand(values, weights)` | Uniform or weighted random selection from a set |
| `set_norm` | `set_norm(values, mean, stddev)` | Normal distribution over indices; `mean` index picked most often |
| `set_exp` | `set_exp(values, rate)` | Exponential distribution over indices; lower indices picked most often |
| `set_lognorm` | `set_lognorm(values, mu, sigma)` | Log-normal distribution over indices; right-skewed selection |
| `set_zipf` | `set_zipf(values, s, v)` | Zipfian distribution over indices; strong power-law skew toward first items |

## Argument Expression Examples

```yaml
args:
  # Always passes the integer 42.
  - const(42)

  # Evaluates the expression warehouses * 10 using globals; both forms are equivalent.
  - expr(warehouses * 10)
  - warehouses * 10

  # Generates a random integer between 1 and 10 using gofakeit.
  - gen('number:1,10')

  # Returns a random row from the 'fetch_warehouses' init query, pinned to this
  # worker for its lifetime, and extracts the w_id field.
  - ref_perm('fetch_warehouses').w_id

  # Non-uniform random int using TPC-C NURand.
  - nurand(1023, 1, customers / districts)

  # Generates between 5 and 15 unique NURand values as a comma-separated string.
  - nurand_n(8191, 1, items, 5, 15)

  # Drives batched execution: the parent query runs 10 times with $1 = 0..9.
  - batch(customers / batch_size)

  # Generates 100,000 unique emails via gofakeit, split into batches of 10,000.
  - gen_batch(customers, batch_size, 'email')

  # Picks a random payment method with uniform probability.
  - set_rand(['credit_card', 'debit_card', 'paypal'], [])

  # Picks a random rating weighted toward 4 and 5 stars.
  - set_rand(['1', '2', '3', '4', '5'], [5, 10, 20, 35, 30])

  # Picks a quantity using normal distribution.
  # mean=2 (value '3' at index 2 is most common), stddev=0.8 (~68% pick indices 1-3).
  - set_norm([1, 2, 3, 4, 5], 2, 0.8)

  # Picks a priority level using exponential distribution.
  # Higher rate concentrates picks toward the first item ('low').
  - set_exp(['low', 'medium', 'high', 'critical'], 0.5)

  # Picks a tier using log-normal distribution (right-skewed toward early indices).
  - set_lognorm(['free', 'basic', 'pro', 'enterprise'], 0.5, 0.5)

  # Picks a category using Zipfian distribution (strong skew toward first items).
  - set_zipf(['electronics', 'clothing', 'books', 'food', 'toys'], 2.0, 1.0)

  # Generates a random UUID v4 (random) or v7 (time-ordered, sortable).
  - uuid_v4()
  - uuid_v7()

  # Random float between 0.01 and 999.99 with 2 decimal places (e.g. for prices).
  - uniform_f(0.01, 999.99, 2)

  # Uniform random float between 0 and 1 (e.g. for percentages).
  - uniform(0, 1)

  # Auto-incrementing sequence: 1, 2, 3, ... (shared across calls for the worker).
  - seq(1, 1)

  # Auto-incrementing with custom start and step: 100, 110, 120, ...
  - seq(100, 10)

  # Formatted order number using template and seq: "ORD-00001", "ORD-00002", ...
  - template('ORD-%05d', seq(1, 1))

  # Zipfian distribution: hot-key pattern where value 0 is most frequent.
  # s=2.0 controls skew (higher = more skewed), v=1.0, max=999.
  - zipf(2.0, 1.0, 999)

  # Normally-distributed integer review rating centred on 4, mostly 3-5.
  - norm(4, 1, 1, 5)

  # Normally-distributed float price centred on 50.00, rounded to 2 decimal places.
  - norm_f(50.0, 15.0, 1.0, 100.0, 2)

  # Conditional value based on a random roll.
  - cond(gen('number:1,100') > 95, 'premium', 'standard')

  # First non-nil fallback value.
  - coalesce(ref_rand('optional_data').value, 'default')

  # Generate a product code matching a regex pattern.
  - regex('[A-Z]{3}-[0-9]{4}')

  # Build a JSON metadata object for a JSONB column.
  - json_obj('source', 'web', 'version', 2, 'active', true)

  # Build a JSON array of 1-5 random email addresses.
  - json_arr(1, 5, 'email')

  # Random geographic point within 10km of London, access lat/lon separately.
  - point(51.5074, -0.1278, 10.0).lat
  - point(51.5074, -0.1278, 10.0).lon

  # Random geographic point as WKT for native geometry columns.
  - point_wkt(51.5074, -0.1278, 10.0)

  # Random timestamp between two dates (RFC3339 format).
  - timestamp('2020-01-01T00:00:00Z', '2025-01-01T00:00:00Z')

  # Random date formatted as YYYY-MM-DD.
  - date('2006-01-02', '2020-01-01T00:00:00Z', '2025-01-01T00:00:00Z')

  # Timestamp 72 hours in the past (e.g. for TTL or expiry columns).
  - date_offset('-72h')

  # Random duration between 1 hour and 24 hours.
  - duration('1h', '24h')

  # Pick 3-8 products weighted by their popularity column.
  - weighted_sample_n('fetch_products', 'id', 'popularity', 3, 8)

  # Regex: generate a random US phone number.
  - regex('\\([0-9]{3}\\) [0-9]{3}-[0-9]{4}')

  # Regex: generate a random hex colour code.
  - regex('#[0-9a-f]{6}')

  # Regex: generate a random IPv4 address.
  - regex('[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}')

  # Regex: generate a random MAC address.
  - regex('[0-9a-f]{2}(:[0-9a-f]{2}){5}')

  # Regex: generate a random license plate (e.g. "AB12 CDE").
  - regex('[A-Z]{2}[0-9]{2} [A-Z]{3}')

  # Exponentially-distributed float in [0, 100] with 2 decimal places.
  - exp_f(0.5, 0, 100, 2)

  # Log-normally-distributed float in [1, 1000] with 2 decimal places.
  - lognorm_f(1.0, 0.5, 1, 1000, 2)

  # Random IP address within a CIDR block (e.g. for network simulation).
  - inet('192.168.1.0/24')

  # Random 16 bytes as a hex-encoded CockroachDB/PostgreSQL BYTES literal.
  - bytes(16)

  # Random fixed-length bit string of 8 bits (e.g. "10110011").
  - bit(8)

  # Random variable-length bit string of 1-16 bits.
  - varbit(16)

  # PostgreSQL/CockroachDB array literal with 2-5 random email addresses.
  - array(2, 5, 'email')

  # Random time of day between 08:00 and 18:00 (HH:MM:SS format).
  - time('08:00:00', '18:00:00')

  # Random time of day with timezone suffix (for TIMETZ columns).
  - timez('09:00:00', '17:00:00')

  # Sum of the 'price' field across all rows in the 'fetch_products' dataset.
  - sum('fetch_products', 'price')

  # Average price across all products.
  - avg('fetch_products', 'price')

  # Minimum and maximum price in the dataset.
  - min('fetch_products', 'price')
  - max('fetch_products', 'price')

  # Total number of rows in the dataset.
  - count('fetch_products')

  # Number of distinct category IDs across all products.
  - distinct('fetch_products', 'category_id')

  # Dependent columns: later args can reference earlier ones by index.
  # arg(0) is the first arg, arg(1) is the second, etc.
  - gen('firstname')      # $1 = "Alice"
  - gen('lastname')       # $2 = "Smith"
  - arg(0) + " " + arg(1) # $3 = "Alice Smith"

  # Compute a total from previously generated values.
  - uniform_f(1.00, 99.99, 2) # $1 = price (e.g. 29.99)
  - gen('number:1,10')        # $2 = quantity (e.g. 3)
  - arg(0) * float(arg(1))    # $3 = total (e.g. 89.97)
```
