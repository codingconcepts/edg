# edg

A database workload runner driven by YAML configuration. Define your schema, seed data, and transactional workloads in a single config file, then run them against any supported database with concurrent workers and real-time throughput reporting.

Query arguments are written as expressions compiled at startup, giving you access to global constants, random data generation, reference lookups, and TPC-C-compliant non-uniform random distributions.

## Table of Contents

- [Supported Databases](#supported-databases)
- [Installation](#installation)
- [Usage](#usage)
  - [Commands](#commands)
  - [Flags](#flags)
- [Configuration](#configuration)
  - [Globals](#globals)
  - [Sections](#sections)
  - [Query Types](#query-types)
  - [Run Weights](#run-weights)
- [Expressions](#expressions)
  - [Functions](#functions)
  - [User-Defined Expressions](#user-defined-expressions)
  - [Examples](#examples)
- [Example Workloads](#example-workloads)
- [Setup](#setup)

## Supported Databases

| Database | Driver | URL scheme |
|---|---|---|
| CockroachDB / PostgreSQL | `pgx` | `postgres://...` |
| Oracle | `oracle` | `oracle://...` |
| MySQL | `mysql` | `user:password@tcp(host:port)/database?parseTime=true` |

## Installation

```sh
go install github.com/codingconcepts/edg@latest
```

Or build from source:

```sh
git clone https://github.com/codingconcepts/edg
cd edg
go build -o edg .
```

## Usage

### Commands

| Command | Description |
|---|---|
| `up` | Create schema (tables, indexes) |
| `seed` | Populate tables with initial data |
| `run` | Execute the benchmark workload |
| `deseed` | Delete seeded data (truncate tables) |
| `down` | Tear down schema (drop tables) |

A typical workflow runs the commands in order: `up` -> `seed` -> `run` -> `deseed` -> `down`.

### Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--url` | | | Database connection URL (or set `URL` env var) |
| `--config` | | `_examples/tpcc/crdb.yaml` | Path to the workload YAML config file |
| `--driver` | | `pgx` | database/sql driver name (`pgx`, `oracle`, or `mysql`) |
| `--duration` | `-d` | `1m` | Benchmark duration (run command only) |
| `--workers` | `-w` | `1` | Number of concurrent workers (run command only) |
| `--print-interval` | | `1s` | Progress reporting interval (run command only) |

### Example

```sh
edg up \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg seed \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg run \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 100 \
-d 1m

edg deseed \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg down \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

## Configuration

Workloads are defined in a single YAML file with the following top-level keys:

```yaml
# Variables available in all expressions.
globals:

# User-defined expression functions.
expressions:

# Schema creation queries.
up:

# Data population queries.
seed:

# Data cleanup queries.
deseed:

# Schema teardown queries.
down:

# Per-worker initialisation queries (run before workload).
init:

# Weighted transaction mix (optional).
run_weights:

# Workload queries.
run:
```

### Globals

The `globals` section defines top-level variables available in all expressions:

```yaml
globals:
  warehouses: 1
  districts: 10
  customers: 30000
  items: 100000
```

These can be referenced directly in arg expressions, including in arithmetic:

```yaml
args:
  - customers / districts   # evaluates to 3000
  - warehouses * 10         # evaluates to 10
```

### Sections

Each section (`up`, `seed`, `deseed`, `down`, `init`, `run`) contains a list of named queries:

```yaml
up:
  - name: create_users
    query: |-
      CREATE TABLE IF NOT EXISTS users (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        email STRING NOT NULL
      )

seed:
  - name: populate_users
    args:
      - gen_batch(1000, 100, 'email')
    query: |-
      INSERT INTO users (email)
      SELECT unnest(string_to_array('$1', ','))
```

- **`up`** and **`down`** manage schema (CREATE/DROP).
- **`seed`** and **`deseed`** manage data (INSERT/TRUNCATE).
- **`init`** runs once per worker before the workload starts, typically to fetch reference data for use in `run` queries.
- **`run`** contains the transactional workload queries executed in a loop.

### Query Types

| Type | Description |
|---|---|
| `query` (default) | Executes the SQL and reads result rows. Results are stored in separate memory for each worker by query name, making them available to `ref_*` functions. |
| `exec` | Executes the SQL without reading results. Use for DDL, DML that returns no rows, or when results aren't needed. |

Queries can also specify a `wait` duration (e.g. `wait: 18s`) to introduce a keying/think-time delay after execution in the `run` section.

### Run Weights

The optional `run_weights` map controls the transaction mix during workload execution. Each key is a query name from the `run` section, and the value is a relative weight. On each iteration, a single transaction is chosen by weighted random selection:

```yaml
run_weights:
  new_order: 45
  payment: 43
  order_status: 4
  delivery: 4
  stock_level: 4
```

If `run_weights` is omitted, all `run` queries execute sequentially on each iteration.

## Expressions

Query arguments are written as expressions compiled at startup using [expr-lang/expr](https://github.com/expr-lang/expr). Each expression has access to the built-in functions, globals, and any user-defined expressions.

### Functions

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
| `batch(n)` | `[][]any` | Returns sequential integers `[0, n)` as batch arg sets, causing the parent query to run once per index. Use for driving batched execution without a SQL query. |
| `gen_batch(total, batchSize, pattern)` | `[][]any` | Generates `total` values using [gofakeit](https://github.com/brianvoe/gofakeit) `pattern`, grouped into batches of `batchSize`. Each batch arg is a comma-separated string of generated values. Combine with `unnest(string_to_array(...))` in SQL to expand into rows. |
| `ref_each(query)` | `[][]any` | Executes a SQL query and returns all rows. Each row becomes a separate arg set, causing the parent query to run once per row (batch mode). |
| `ref_n(name, field, min, max)` | `string` | Picks N unique random rows (N in [min, max]) from a named dataset, extracts `field` from each, and returns a comma-separated string (e.g. `"42,17,93"`). |
| `nurand(A, x, y)` | `int` | TPC-C Non-Uniform Random: `(((random(0,A) \| random(x,y)) + C) / (y-x+1)) + x`. The constant C is generated once per A value and persists for the worker's lifetime. |
| `nurand_n(A, x, y, min, max)` | `string` | Generates N unique NURand values (N in [min, max]) as a comma-separated string. |
| `set_rand(values, weights)` | `any` | Picks a random item from a set. If weights are provided, weighted random selection is used; otherwise uniform. Values and weights are separate arrays. |
| `set_normal(values, mean, stddev)` | `any` | Picks an item from a set using normal distribution. `mean` is the index selected most often; `stddev` controls spread (~68% of picks fall within `mean +/- stddev` indices, ~95% within `mean +/- 2*stddev`). A smaller stddev concentrates picks around the mean; a larger one spreads them more evenly. |
| `norm_rand(mean, stddev, min, max)` | `float64` | Normally-distributed random number in [min, max], rounded to 0 decimal places (whole number). |
| `norm_rand_f(mean, stddev, min, max, precision)` | `float64` | Normally-distributed random number in [min, max], rounded to `precision` decimal places. |
| `norm_rand_n(mean, stddev, min, max, minN, maxN)` | `string` | N unique normally-distributed values (N in [minN, maxN]) as a comma-separated string. |
| `uuid_v1()` | `string` | Generates a Version 1 UUID (timestamp + node ID). |
| `uuid_v4()` | `string` | Generates a Version 4 UUID (random). |
| `uuid_v6()` | `string` | Generates a Version 6 UUID (reordered timestamp). |
| `uuid_v7()` | `string` | Generates a Version 7 UUID (Unix timestamp + random, sortable). |
| `float_rand(min, max, precision)` | `float64` | Random float in [min, max] rounded to `precision` decimal places. |
| `uniform_rand(min, max)` | `float64` | Uniform random float in [min, max]. |
| `seq(start, step)` | `int` | Auto-incrementing sequence per worker. Returns `start + counter * step`, where counter increments on each call. |
| `zipf(s, v, max)` | `int` | Zipfian-distributed (power-law) random integer in [0, max]. `s` (> 1) controls skew, `v` (>= 1) offsets the distribution. Lower values are exponentially more frequent. |
| `cond(predicate, trueVal, falseVal)` | `any` | Returns `trueVal` if `predicate` is true, `falseVal` otherwise. |
| `coalesce(v1, v2, ...)` | `any` | Returns the first non-nil value from arguments. |
| `template(format, args...)` | `string` | Formats a string using Go's `fmt.Sprintf` syntax (e.g. `template('ORD-%05d', seq(1, 1))`). |
| `regex(pattern)` | `string` | Generates a random string matching the given regular expression. |
| `json_obj(k1, v1, k2, v2, ...)` | `string` | Builds a JSON object string from key-value pair arguments. |
| `json_arr(minN, maxN, pattern)` | `string` | Builds a JSON array of N random values (N in [minN, maxN]) generated by a gofakeit `pattern`. |
| `point(lat, lon, radiusKM)` | `map` | Generates a random geographic point within `radiusKM` of (`lat`, `lon`). Access fields with `.lat` and `.lon`. |
| `point_wkt(lat, lon, radiusKM)` | `string` | Generates a random geographic point within `radiusKM` of (`lat`, `lon`) as a WKT string: `POINT(lon lat)`. Use with `ST_GeomFromText` for native geometry columns. |
| `rand_timestamp(min, max)` | `string` | Random timestamp between `min` and `max` (both RFC3339 strings), returned as RFC3339. |
| `rand_duration(min, max)` | `string` | Random duration between `min` and `max` (Go duration strings like `"1h"`, `"30m"`). |
| `date_rand(format, min, max)` | `string` | Random timestamp between `min` and `max` (RFC3339), formatted using a Go time format string. |
| `date_offset(duration)` | `string` | Returns the current time offset by `duration` (e.g. `"-72h"`, `"30m"`), formatted as RFC3339. |
| `weighted_sample_n(name, field, weightField, minN, maxN)` | `string` | Picks N unique rows (N in [minN, maxN]) from a named dataset using weighted selection based on `weightField`, extracts `field`, and returns a comma-separated string. |

### User-Defined Expressions

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
| Conditionals | `args[0] > 10 ? 'yes' : 'no'`, `args[0] ?? 0` (nil coalescing) |
| Math functions | `min()`, `max()`, `abs()`, `ceil()`, `floor()`, `round()` |
| String functions | `upper()`, `lower()`, `trim()`, `split()`, `replace()` |
| Type conversion | `int()`, `float()`, `string()` |
| Pipe operator | `args[0] \| int` (equivalent to `int(args[0])`) |

### Examples

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
  - set_normal([1, 2, 3, 4, 5], 2, 0.8)

  # Generates a random UUID v4 (random) or v7 (time-ordered, sortable).
  - uuid_v4()
  - uuid_v7()

  # Random float between 0.01 and 999.99 with 2 decimal places (e.g. for prices).
  - float_rand(0.01, 999.99, 2)

  # Uniform random float between 0 and 1 (e.g. for percentages).
  - uniform_rand(0, 1)

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
  - norm_rand(4, 1, 1, 5)

  # Normally-distributed float price centred on 50.00, rounded to 2 decimal places.
  - norm_rand_f(50.0, 15.0, 1.0, 100.0, 2)

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
  # PostgreSQL/CockroachDB: ST_GeomFromText($1, 4326)
  # MySQL:                  ST_GeomFromText(?, 4326)
  # Oracle:                 SDO_UTIL.FROM_WKTGEOMETRY(:1)
  - point_wkt(51.5074, -0.1278, 10.0)

  # Random timestamp between two dates (RFC3339 format).
  - rand_timestamp('2020-01-01T00:00:00Z', '2025-01-01T00:00:00Z')

  # Random date formatted as YYYY-MM-DD.
  - date_rand('2006-01-02', '2020-01-01T00:00:00Z', '2025-01-01T00:00:00Z')

  # Timestamp 72 hours in the past (e.g. for TTL or expiry columns).
  - date_offset('-72h')

  # Random duration between 1 hour and 24 hours.
  - rand_duration('1h', '24h')

  # Pick 3-8 products weighted by their popularity column.
  - weighted_sample_n('fetch_products', 'id', 'popularity', 3, 8)
```

## Example Workloads

| Workload | Description |
|---|---|
| [TPC-C](_examples/tpcc/) | Full TPC-C benchmark with all 5 transaction profiles |
| [Bank](_examples/bank/) | Bank account operations for contention and correctness testing |
| [E-Commerce](_examples/ecommerce/) | E-commerce with categories, products, customers, and orders |
| [IoT](_examples/iot/) | IoT devices, sensors, and time-series readings |
| [Normal](_examples/normal/) | Product reviews with normal distribution ratings |
| [Pipeline](_examples/pipeline/) | Multi-table sequential reads and writes |
| [SaaS](_examples/saas/) | Multi-tenant SaaS with tenants, users, projects, and tasks |
| [Populate](_examples/populate/) | Billion-row data population benchmark |
| [Social](_examples/social/) | Social network with users, posts, follows, and tags |

