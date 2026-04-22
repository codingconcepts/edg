---
title: Configuration
weight: 4
---

# Configuration

Workloads are defined in a single YAML file with the following top-level keys:

```yaml
# Variables available in all expressions.
globals:

# User-defined expression functions.
expressions:

# Reusable arg templates for queries.
rows:

# Static datasets available to ref_* functions without a database query.
reference:

# Named auto-incrementing sequences shared across all workers.
seq:

# Staged workload execution (overrides -w and -d flags).
stages:

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

# Workload queries (standalone or grouped into transactions).
run:

# Background queries that run independently on a fixed schedule.
workers:

# Post-run assertions for CI/CD (exit non-zero on failure).
expectations:
```

## Globals

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
  - customers / districts # evaluates to 3000
  - warehouses * 10       # evaluates to 10
```

### Expression-valued globals

Global values can be expressions, including references to other globals and environment variables. String values are compiled as expressions; if compilation fails (e.g. a plain string like `"new york"`), the value is kept as a literal.

Globals are evaluated in YAML document order, so later globals can reference earlier ones:

```yaml
globals:
  warehouses: 1
  districts: warehouses * 10  # evaluates to 10
  customers: districts * 3000 # evaluates to 30000
```

### Globals from environment variables

Use `env()` to read a required environment variable, or `env_nil()` for an optional one. Combine `env_nil()` with `coalesce()` to provide a default:

```yaml
globals:
  # Required (fails at startup if DB_BATCH_SIZE is not set).
  batch_size: env('DB_BATCH_SIZE')

  # Optional (falls back to 10000 if CUSTOMERS is not set).
  customers: int(coalesce(env_nil('CUSTOMERS'), 10000))
```

> [!WARNING]
> `env()` returns a string. Wrap with `int()` or `float()` if arithmetic is needed downstream.

## Rows

The `rows` section defines reusable arg templates. Each key is a row name, and the value is a list of arg expressions. Queries can reference a row template using the `row` field instead of `args`:

```yaml
rows:
  customer:
    - gen('email')
    - gen('name')
    - timestamp('2020-01-01T00:00:00Z', '2024-01-01T00:00:00Z')

seed:
  - name: seed_customers
    type: exec_batch
    count: 50000
    size: 5000
    row: customer
    query: |-
      INSERT INTO customer (email, name, created_at)
      SELECT e, n, t
      FROM unnest(
        string_to_array('$1', __sep__),
        string_to_array('$2', __sep__),
        string_to_array('$3', __sep__)
      ) AS t(e, n, t)

run:
  - name: insert_customer
    type: exec
    row: customer
    query: |-
      INSERT INTO customer (email, name, created_at)
      VALUES ($1, $2, $3)
```

The `row` field and `args` field are mutually exclusive, a query must use one or the other, not both. Row names must be defined in the `rows` section; referencing an unknown row name is a validation error.

Row templates support all the same expressions as `args`. They are expanded before compilation, so from the query's perspective there is no difference between `row: customer` and provide arguments via `args`.

## Reference

The `reference` section defines static datasets that are loaded into the environment at startup, making them available to `ref_rand`, `ref_same`, `ref_perm`, and `ref_diff` functions without needing an `init` query or database connection. Each key is a dataset name, and the value is a list of row objects:

```yaml
reference:
  regions:
    - {name: us, cities: [a, b, c]}
    - {name: eu, cities: [d, e, f]}
    - {name: ap, cities: [g, h, i]}
```

Reference datasets work exactly like datasets populated by `init` queries. You can use them in any arg expression:

```yaml
args:
  # Random region row, access the 'name' field.
  - ref_rand('regions').name

  # Same row reused across all ref_same calls in this query execution.
  - ref_same('regions').name
  - set_rand(ref_same('regions').cities, [])
```

This is useful when your lookup data is small and known ahead of time, avoiding the need for a database round-trip.

## Seq

The `seq` section defines named auto-incrementing sequences that are shared across all workers. Unlike `seq(start, step)` which maintains a separate counter per worker, `seq_global("name")` returns globally unique values using atomic counters.

```yaml
seq:
  - name: order_id
    start: 1
    step: 1
  - name: line_item_id
    start: 1000
    step: 5
```

| Field | Description |
|---|---|
| `name` | Sequence identifier, referenced in `seq_global("name")` calls. Must be unique. |
| `start` | Initial value returned by the first call. |
| `step` | Increment between consecutive values. |

Use `seq_global("name")` in any arg expression:

```yaml
seed:
  - name: seed_orders
    type: exec
    count: 1000
    args:
      - seq_global("order_id")
      - gen('firstname')
    query: INSERT INTO orders (id, name) VALUES ($1, $2)
```

Sequences work in all sections (`seed`, `run`, `workers`) and produce gap-free, globally unique values across concurrent workers. The same sequence can be used in both seed and run; the counter continues from where seeding left off.

To reference a random value that has already been generated by a sequence, use the distribution functions:

| Function | Description |
|---|---|
| `seq_rand("name")` | Uniform random pick from generated values |
| `seq_zipf("name", s, v)` | Zipfian-distributed pick (hot early values) |
| `seq_norm("name", mean, stddev)` | Normal-distributed pick |
| `seq_exp("name", rate)` | Exponential-distributed pick |
| `seq_lognorm("name", mu, sigma)` | Log-normal-distributed pick |

These functions compute valid values algebraically from the sequence's start, step, and current counter without generating and storing each sequence in memory. This is useful for foreign key references where the referenced column was populated by `seq_global`.

See [`_examples/global_sequences/`](https://github.com/codingconcepts/edg/tree/main/_examples/global_sequences) for a complete working example but here's a quick introduction.

Create two tables, one to store 1,000 orders and one to store references to them (using each of the sequence generators):

```yaml
globals:
  orders: 1000
  samples: 10000
  batch_size: 100

seq:
  - name: order_id
    start: 1
    step: 1

up:

  - name: create_orders
    query: |-
      CREATE TABLE IF NOT EXISTS orders (
        id INT PRIMARY KEY,
        customer STRING NOT NULL
      )

  - name: create_samples
    query: |-
      CREATE TABLE IF NOT EXISTS samples (
        id INT PRIMARY KEY DEFAULT unique_rowid(),
        uniform_val INT NOT NULL,
        zipf_val INT NOT NULL,
        norm_val INT NOT NULL,
        exp_val INT NOT NULL,
        lognorm_val INT NOT NULL
      )

seed:

  - name: seed_orders
    type: exec_batch
    count: orders
    size: batch_size
    args:
      - seq_global("order_id")
      - gen('firstname') + ' ' + gen('lastname')
    query: |-
      INSERT INTO orders (id, customer)
      SELECT
        unnest(string_to_array('$1', chr(31))::INT[]),
        unnest(string_to_array('$2', chr(31)))

  - name: seed_samples
    type: exec_batch
    count: samples
    size: batch_size
    args:
      - seq_rand("order_id")
      - seq_zipf("order_id", 1.1, 1.0)
      - seq_norm("order_id", 500, 150)
      - seq_exp("order_id", 0.01)
      - seq_lognorm("order_id", 5.5, 0.5)
    query: |-
      INSERT INTO samples (uniform_val, zipf_val, norm_val, exp_val, lognorm_val)
      SELECT
        unnest(string_to_array('$1', chr(31))::INT[]),
        unnest(string_to_array('$2', chr(31))::INT[]),
        unnest(string_to_array('$3', chr(31))::INT[]),
        unnest(string_to_array('$4', chr(31))::INT[]),
        unnest(string_to_array('$5', chr(31))::INT[])
```

After seeding, the samples table will show various distributions of order ids (all with complete referential integrity to the orders table) and can be queried to show their distribution as follows:

```sql
-- Uniform distribution.
SELECT
  div(uniform_val - 1, 50) * 50 + 1 AS bucket,
  count(*) AS total,
  repeat('█', (count(*) * 50 / max(count(*)) OVER ())::INT) AS histogram
FROM samples
GROUP BY 1
ORDER BY 1;

  bucket | total |                     histogram
---------+-------+-----------------------------------------------------
       1 |   481 | █████████████████████████████████████████████
      51 |   484 | █████████████████████████████████████████████
     101 |   496 | ███████████████████████████████████████████████
     151 |   465 | ████████████████████████████████████████████
     201 |   492 | ██████████████████████████████████████████████
     251 |   510 | ████████████████████████████████████████████████
     301 |   516 | ████████████████████████████████████████████████
     351 |   513 | ████████████████████████████████████████████████
     401 |   471 | ████████████████████████████████████████████
     451 |   533 | ██████████████████████████████████████████████████
     501 |   519 | █████████████████████████████████████████████████
     551 |   498 | ███████████████████████████████████████████████
     601 |   522 | █████████████████████████████████████████████████
     651 |   518 | █████████████████████████████████████████████████
     701 |   492 | ██████████████████████████████████████████████
     751 |   473 | ████████████████████████████████████████████
     801 |   522 | █████████████████████████████████████████████████
     851 |   515 | ████████████████████████████████████████████████
     901 |   472 | ████████████████████████████████████████████
     951 |   508 | ████████████████████████████████████████████████

-- Normal distribution.
SELECT
  div(norm_val - 1, 50) * 50 + 1 AS bucket,
  count(*) AS total,
  repeat('█', (count(*) * 50 / max(count(*)) OVER ())::INT) AS histogram
FROM samples
GROUP BY 1
ORDER BY 1;

  bucket | total |                     histogram
---------+-------+-----------------------------------------------------
       1 |    10 |
      51 |    23 | █
     101 |    59 | ██
     151 |   142 | █████
     201 |   254 | █████████
     251 |   390 | ██████████████
     301 |   667 | ████████████████████████
     351 |   905 | █████████████████████████████████
     401 |  1159 | ██████████████████████████████████████████
     451 |  1380 | ██████████████████████████████████████████████████
     501 |  1362 | █████████████████████████████████████████████████
     551 |  1162 | ██████████████████████████████████████████
     601 |   917 | █████████████████████████████████
     651 |   652 | ████████████████████████
     701 |   440 | ████████████████
     751 |   263 | ██████████
     801 |   128 | █████
     851 |    57 | ██
     901 |    27 | █
     951 |     3 |

-- Exponential distribution.
SELECT
  div(exp_val - 1, 50) * 50 + 1 AS bucket,
  count(*) AS total,
  repeat('█', (count(*) * 50 / max(count(*)) OVER ())::INT) AS histogram
FROM samples
GROUP BY 1
ORDER BY 1;

  bucket | total |                     histogram
---------+-------+-----------------------------------------------------
       1 |  3961 | ██████████████████████████████████████████████████
      51 |  2436 | ███████████████████████████████
     101 |  1475 | ███████████████████
     151 |   782 | ██████████
     201 |   532 | ███████
     251 |   329 | ████
     301 |   207 | ███
     351 |   111 | █
     401 |    66 | █
     451 |    40 | █
     501 |    26 |
     551 |    19 |
     601 |     7 |
     651 |     1 |
     701 |     1 |
     751 |     5 |
     851 |     1 |
     901 |     1 |

-- Log-normal distribution.
SELECT
  div(lognorm_val - 1, 50) * 50 + 1 AS bucket,
  count(*) AS total,
  repeat('█', (count(*) * 50 / max(count(*)) OVER ())::INT) AS histogram
FROM samples
GROUP BY 1
ORDER BY 1;

  bucket | total |                     histogram
---------+-------+-----------------------------------------------------
       1 |    10 |
      51 |   342 | █████████
     101 |  1293 | ███████████████████████████████████
     151 |  1836 | ██████████████████████████████████████████████████
     201 |  1699 | ██████████████████████████████████████████████
     251 |  1425 | ███████████████████████████████████████
     301 |  1024 | ████████████████████████████
     351 |   767 | █████████████████████
     401 |   516 | ██████████████
     451 |   335 | █████████
     501 |   238 | ██████
     551 |   147 | ████
     601 |   121 | ███
     651 |    81 | ██
     701 |    71 | ██
     751 |    38 | █
     801 |    25 | █
     851 |    15 |
     901 |    13 |
     951 |     4 |

-- Zipfian distribution.
SELECT
  div(zipf_val - 1, 50) * 50 + 1 AS bucket,
  count(*) AS total,
  repeat('█', (count(*) * 50 / max(count(*)) OVER ())::INT) AS histogram
FROM samples
GROUP BY 1
ORDER BY 1;

  bucket | total |                     histogram
---------+-------+-----------------------------------------------------
       1 |  6904 | ██████████████████████████████████████████████████
      51 |   778 | ██████
     101 |   458 | ███
     151 |   312 | ██
     201 |   255 | ██
     251 |   177 | █
     301 |   146 | █
     351 |   112 | █
     401 |   106 | █
     451 |    77 | █
     501 |   104 | █
     551 |    76 | █
     601 |    79 | █
     651 |    87 | █
     701 |    56 |
     751 |    64 |
     801 |    66 |
     851 |    41 |
     901 |    54 |
     951 |    48 |
```

## Stages

The `stages` section defines a sequence of workload phases, each with its own worker count and duration. When stages are present, the `-w` and `-d` CLI flags are ignored.

```yaml
stages:
  - name: ramp
    workers: 1
    duration: 10s
  - name: steady
    workers: 10
    duration: 30s
  - name: cooldown
    workers: 2
    duration: 10s
```

Each stage runs sequentially. When a stage completes (its duration expires), the next stage starts immediately with a new set of workers. The `init` section runs once before the first stage, and its results are shared across all stages.

| Field | Description |
|---|---|
| `name` | Stage identifier, logged when the stage starts. |
| `workers` | Number of concurrent workers for this stage. Defaults to 1 if omitted. |
| `duration` | How long this stage runs (e.g. `10s`, `5m`, `1h`). |

This is useful for simulating ramp-up patterns, sustained load tests, or multi-phase benchmarks without running separate commands.

See [`_examples/stages/`](https://github.com/codingconcepts/edg/tree/main/_examples/stages) for a complete working example.

## Sections

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
      SELECT unnest(string_to_array('$1', __sep__))
```

- **`up`** and **`down`** manage schema (CREATE/DROP).
- **`seed`** and **`deseed`** manage data (INSERT/TRUNCATE).
- **`init`** runs once per worker before the workload starts, typically to fetch reference data for use in `run` queries.
- **`run`** contains the workload queries executed in a loop. Queries can be standalone or grouped into [transactions](#transactions).

## Args

The `args` field provides values for query placeholders (`$1`, `$2`, etc.). Each expression is evaluated at runtime using the full expression environment (globals, reference data, `ref_*` functions, generators, and more).

Args can be written in two forms: positional (list) or named (map). Both bind to `$1`, `$2`, `$3`, etc. in declaration order. The difference is how you reference previously computed args within expressions.

### Positional args

The default form is a list. Reference earlier args by zero-based index with `arg(0)`, `arg(1)`, etc.:

```yaml
args:
  - gen('email')
  - ref_same('regions').name
  - set_rand(ref_same('regions').cities, [])
  - uniform(1, 500)
  - arg(0) + " (" + arg(1) + ")" # depends on args 0 and 1
```

### Named args

The map form gives each arg a name. Reference earlier args by name with `arg('name')`:

```yaml
args:
  email: gen('email')
  region: ref_same('regions').name
  city: set_rand(ref_same('regions').cities, [])
  amount: uniform(1, 500)
  label_named: arg('email') + " (" + arg('region') + ")"
  label_pos: arg(0) + " (" + arg(1) + ")" # Produces the same as label_named.
```

Named args bind to placeholders in declaration order (`email` -> `$1`, `region` -> `$2`, etc.), so query SQL is identical to the positional form. Index-based access still works (`arg(0)` and `arg('email')` return the same value).

> [!NOTE]
> Named and positional forms are mutually exclusive per query. Use one or the other.

See [`_examples/named_args/`](https://github.com/codingconcepts/edg/tree/main/_examples/named_args) for a complete working example.

## Query Types

| Type | Description |
|---|---|
| `query` (default) | Executes the SQL and reads result rows. Results are stored in separate memory for each worker by query name, making them available to `ref_*` functions. |
| `exec` | Executes the SQL without reading results. Use for DDL, DML that returns no rows, or when results aren't needed. |
| `query_batch` | Like `query`, but evaluates args repeatedly (controlled by `count` and `size`) and collects values into unit-separator-delimited (ASCII 31) strings per arg position. Each batch becomes a separate query execution whose results are stored. |
| `exec_batch` | Like `exec`, but evaluates args repeatedly (controlled by `count` and `size`) and collects values into unit-separator-delimited (ASCII 31) strings per arg position. Each batch becomes a separate exec. |

### Batch Fields

The `query_batch` and `exec_batch` types use two additional fields to control how args are generated and grouped:

| Field | Description |
|---|---|
| `count` | Total number of rows to generate. Evaluated as an expression, so it can reference globals. |
| `size` | Number of rows per batch. If omitted or zero, defaults to `count` (single batch). Also evaluated as an expression. |
| `batch_format` | Controls how batch values are serialized. Default uses the ASCII unit separator (char 31, `\x1f`) as a delimiter. Set to `json` to produce JSON arrays. |

Each arg expression is evaluated once per row. The results are collected into strings per arg position, delimited by the ASCII unit separator (char 31, `\x1f`). For example, with `count: 1000` and `size: 100`, you get 10 batches, each containing 100 generated values.

#### Postgres / CockroachDB

By default, batch values are joined with the ASCII unit separator (char 31). This works well with PostgreSQL/CockroachDB `string_to_array` and MySQL `JSON_TABLE`:

```yaml
seed:
  - name: populate_users
    type: exec_batch
    count: 1000
    size: 100
    args:
      - gen('email')
    query: |-
      INSERT INTO users (email)
      SELECT unnest(string_to_array('$1', __sep__))
```

Which resolves to the following query automatically by edg:

```sql
INSERT INTO users (email)
SELECT unnest(string_to_array(
  'a@x.com\x1fb@y.com\x1f...\x1fz@x.com', chr(31)
))
```

#### MySQL

MySQL uses `JSON_TABLE` to unpack batch values. The unit-separator-delimited string is converted to a JSON array using `CONCAT` and `REPLACE`, then `JSON_TABLE` extracts each element as a row. Multiple columns are correlated using `FOR ORDINALITY`:

```yaml
seed:
  - name: populate_users
    type: exec_batch
    count: 1000
    size: 100
    args:
      - gen('email')
    query: |-
      INSERT INTO users (email)
      SELECT j.val
      FROM JSON_TABLE(
        CONCAT('["', REPLACE('$1', __sep__, '","'), '"]'),
        '$[*]' COLUMNS(val VARCHAR(255) PATH '$')
      ) j
```

Which resolves to the following query automatically by edg:

```sql
INSERT INTO users (email)
SELECT j.val
FROM JSON_TABLE(
  CONCAT('["',
    REPLACE(
      'a@x.com\x1fb@y.com\x1f...\x1fz@x.com',
      CHAR(31), '","'
    ),
  '"]'),
  '$[*]' COLUMNS(val VARCHAR(255) PATH '$')
) j
```

#### Oracle

For Oracle, batch values are joined with the unit separator and unpacked using `xmltable` with `tokenize`. Multiple columns are correlated by joining on `rowid`:

```yaml
seed:
  - name: populate_users
    type: exec_batch
    count: 3
    size: 3
    args:
      - gen('name')
      - gen('email')
    query: |-
      INSERT INTO users (name, email)
      SELECT x1.value, x2.value
      FROM xmltable(
             'for $s in tokenize($v, __sep__) return <r>{$s}</r>'
             PASSING '$1' AS "v"
             COLUMNS value VARCHAR2(255) PATH '.'
           ) x1
      JOIN xmltable(
             'for $s in tokenize($v, __sep__) return <r>{$s}</r>'
             PASSING '$2' AS "v"
             COLUMNS value VARCHAR2(255) PATH '.'
           ) x2 ON x1.rowid = x2.rowid
```

Which resolves to the following query automatically by edg:

```sql
INSERT INTO users (name, email)
SELECT x1.value, x2.value
FROM xmltable(
  'for $s in tokenize($v, codepoints-to-string(31)) return <r>{$s}</r>'
  PASSING 'Alice\x1fBob\x1fCharlie' AS "v"
  COLUMNS value VARCHAR2(255) PATH '.'
) x1
JOIN xmltable(
  'for $s in tokenize($v, codepoints-to-string(31)) return <r>{$s}</r>'
  PASSING 'a@x.com\x1fb@y.com\x1fc@z.com' AS "v"
  COLUMNS value VARCHAR2(255) PATH '.'
) x2 ON x1.rowid = x2.rowid
```

#### MSSQL

When `batch_format` is set to `json`, each arg position is serialized as a properly escaped JSON array (e.g. `["val1","val2","val3"]`). This is recommended for MSSQL, where `OPENJSON` expects JSON input. Multiple `OPENJSON` calls are correlated using `[key]`, which is the zero-based array index. NULL values appear as JSON `null` and can be handled with `NULLIF(j.value, 'null')` if the target column is nullable:

```yaml
seed:
  - name: populate_contacts
    type: exec_batch
    batch_format: json
    count: 1000
    size: 100
    args:
      - gen('name')
      - gen('email')
    query: |-
      INSERT INTO contact (name, email)
      SELECT j1.value, j2.value
      FROM OPENJSON('$1') j1
      JOIN OPENJSON('$2') j2 ON j1.[key] = j2.[key]
```

Which resolves to the following query automatically by edg:

```sql
INSERT INTO contact (name, email)
SELECT j1.value, j2.value
FROM OPENJSON('["Alice","Bob",...,"Zara"]') j1
JOIN OPENJSON('["a@x.com","b@y.com",...,"z@x.com"]') j2
  ON j1.[key] = j2.[key]
```

#### Spanner (GoogleSQL)

Spanner uses GoogleSQL syntax. Batch values are unpacked with `UNNEST(SPLIT(..., __sep__))`. The `__sep__` token resolves to `CODE_POINTS_TO_STRING([31])` for Spanner, and `UNNEST` converts the resulting array into rows. Multiple columns are correlated using `WITH OFFSET`:

```yaml
seed:
  - name: populate_users
    type: exec_batch
    count: 1000
    size: 100
    args:
      - gen('email')
    query: |-
      INSERT INTO users (email)
      SELECT val
      FROM UNNEST(SPLIT('$1', __sep__)) AS val
```

Which resolves to the following query automatically by edg:

```sql
INSERT INTO users (email)
SELECT val
FROM UNNEST(SPLIT(
  'a@x.com\x1fb@y.com\x1f...\x1fz@x.com',
  CODE_POINTS_TO_STRING([31])
)) AS val
```

For multiple columns, use separate `UNNEST` calls joined with `WITH OFFSET`:

```yaml
seed:
  - name: populate_contacts
    type: exec_batch
    count: 1000
    size: 100
    args:
      - gen('name')
      - gen('email')
    query: |-
      INSERT INTO contact (name, email)
      SELECT n, e
      FROM UNNEST(SPLIT('$1', __sep__)) AS n WITH OFFSET o1
      JOIN UNNEST(SPLIT('$2', __sep__)) AS e WITH OFFSET o2
        ON o1 = o2
```

> [!NOTE]
> Spanner does not support `TRUNCATE`. Use `DELETE FROM table WHERE TRUE` for deseed operations. Spanner also uses `INSERT OR UPDATE` for upserts instead of `ON CONFLICT`.

### Prepared Statements

Setting `prepared: true` on a `run` query causes the SQL statement to be prepared once per worker and reused across iterations. This reduces server-side parse overhead for high-throughput workloads by allowing the database to cache the query plan.

```yaml
run:
  - name: lookup_product
    type: query
    prepared: true
    args:
      - ref_rand('fetch_products').id
    query: |-
      SELECT id, name, price FROM product WHERE id = $1

  - name: update_price
    type: exec
    prepared: true
    args:
      - ref_rand('fetch_products').id
      - uniform_f(1, 100, 2)
    query: |-
      UPDATE product SET price = $2 WHERE id = $1
```

Prepared statements work with both `query` and `exec` types. They are **not** used for batch types (`query_batch`, `exec_batch`) or queries that undergo batch expansion (via `gen_batch`, `batch`, or `ref_each`), since the SQL changes on each execution in those cases.

Each worker maintains its own statement cache, so prepared statements are safe to use with any number of concurrent workers. Statements are prepared lazily on first use and automatically closed when the worker finishes.

Prepared queries always use `$1`, `$2`, ... placeholders regardless of the target driver. edg automatically translates them to the driver's native format (`?` for mysql, `:N` for oracle, `@pN` for mssql/spanner) at prepare time.

The benefit scales with query complexity. Simple point lookups show minimal improvement, but multi-table joins and aggregations can see significant gains. For example, a 4-table join with GROUP BY, HAVING, and multiple aggregates against CockroachDB:

```
QUERY                        AVG      p50      p95      p99
category_revenue             5.671ms  5.362ms  7.351ms  11.393ms
category_revenue_no_prepare  7.493ms  7.099ms  9.505ms  14.5ms
order_details                3.353ms  3.151ms  4.453ms  7.839ms
order_details_no_prepare     4.377ms  4.258ms  6.37ms   8.354ms
```

See [`_examples/prepared/`](https://github.com/codingconcepts/edg/tree/main/_examples/prepared) for complete working examples across all supported databases.

### Transactions

The `run` section supports grouping multiple queries into an explicit database transaction. Queries inside a transaction block are wrapped in `BEGIN`/`COMMIT` and execute against the same database connection, so reads and writes within a transaction see each other's results.

```yaml
run:
  - transaction: make_transfer
    locals:
      amount: gen('number:1,100')
    queries:
      - name: read_source
        type: query
        args:
          - ref_diff('fetch_accounts').id
        query: SELECT id, balance FROM account WHERE id = $1::UUID

      - name: read_target
        type: query
        args:
          - ref_diff('fetch_accounts').id
        query: SELECT id, balance FROM account WHERE id = $1::UUID

      - name: debit_source
        type: exec
        args:
          - ref_same('read_source').id
          - local('amount')
        query: UPDATE account SET balance = balance - $2::FLOAT WHERE id = $1::UUID

      - name: credit_target
        type: exec
        args:
          - ref_same('read_target').id
          - local('amount')
        query: UPDATE account SET balance = balance + $2::FLOAT WHERE id = $1::UUID
```

The `locals` map defines transaction-scoped variables. Each expression is evaluated once when the transaction begins, and the result is available to all queries via `local('name')`. This ensures the same value is used consistently across multiple queries. For example, the same transfer amount for both the debit and credit.

Local names must not collide with query names in the same transaction.

Transactions and standalone queries can coexist in the same `run` section:

```yaml
run:

  - name: check_balance
    type: query
    args:
      - ref_rand('fetch_accounts').id
    query: SELECT balance FROM account WHERE id = $1::UUID

  - transaction: make_transfer
    locals:
      amount: gen('number:1,100')
    queries:
      - name: read_source
        type: query
        args:
          - ref_diff('fetch_accounts').id
        query: SELECT id, balance FROM account WHERE id = $1::UUID

      - name: read_target
        type: query
        args:
          - ref_diff('fetch_accounts').id
        query: SELECT id, balance FROM account WHERE id = $1::UUID

      - name: debit_source
        type: exec
        args:
          - ref_same('read_source').id
          - local('amount')
        query: UPDATE account SET balance = balance - $2::FLOAT WHERE id = $1::UUID

      - name: credit_target
        type: exec
        args:
          - ref_same('read_target').id
          - local('amount')
        query: UPDATE account SET balance = balance + $2::FLOAT WHERE id = $1::UUID
```

#### When to use transactions

Use transactions when your workload needs **read-then-write patterns** or **multi-statement atomicity**. For example:

- Read an account balance, then debit it (the read and write must see consistent data).
- Insert a row into two related tables that must either both succeed or both roll back.
- Simulate realistic application behaviour where multiple queries happen inside a single database transaction.

Use standalone queries when each operation is independent and doesn't need transactional guarantees.

#### Conditional rollback

A `rollback_if` element can be placed between queries in a transaction. When reached, its expression is evaluated; if it returns `true`, the transaction is rolled back immediately. This is not treated as an error, the worker continues to the next iteration.

```yaml
run:

  - name: check_balance
    type: query
    args:
      - ref_rand('fetch_accounts').id
    query: SELECT balance FROM account WHERE id = $1::UUID

  - transaction: make_transfer
    locals:
      amount: gen('number:1,100')
    queries:
      - name: read_source
        type: query
        args: [ref_diff('fetch_accounts').id]
        query: SELECT id, balance FROM account WHERE id = $1::UUID

      - rollback_if: "ref_same('read_source').balance < local('amount')"

      - name: debit_source
        type: exec
        args:
          - ref_same('read_source').id
          - local('amount')
        query: UPDATE account SET balance = balance - $2::FLOAT WHERE id = $1::UUID
```

In this example, the transfer `amount` is generated once as a local. After `read_source` runs, the condition checks whether the account balance can cover the transfer amount. If not, the transaction rolls back before `debit_source` executes. Multiple `rollback_if` elements can be placed at different points in the transaction to check different conditions.

The expression has access to all data in the environment, including results from queries that have already run within the transaction. A `rollback_if` element must not have `name`, `type`, `args`, or `query` fields.

#### Constraints

- **Batch types not allowed**: `query_batch` and `exec_batch` cannot be used inside a transaction.
- **Prepared statements not allowed**: Queries inside a transaction cannot set `prepared: true`.
- A transaction must contain at least one query.
- Transaction names and standalone query names share the same namespace for `run_weights`.
- The `rollback_if` expression must evaluate to a boolean.

#### Error handling

If any query inside a transaction fails, the transaction is rolled back. During the `run` phase, the error is logged and the worker continues to the next iteration (same as standalone query errors). Conditional rollbacks (via `rollback_if`) are not errors and do not appear in the error rate.

### Wait

Queries can specify a `wait` duration (e.g. `wait: 18s`) to introduce a keying/think-time delay after execution. This only applies to queries in the `run` section and is ignored in other sections.

### Print

The `print` field accepts a list of expressions that are evaluated each iteration and aggregated across all workers for display in the progress and summary output.

#### Simple form

A plain string entry auto-detects the value type. String values show frequency distributions (top 10 by count) and numeric values show min/avg/max.

```yaml
run:
  - name: insert_order
    type: exec
    args:
      - gen('email')
      - ref_same('regions').name
      - set_rand(ref_same('regions').cities, [])
      - uniform(1, 500)
    print:
      - ref_same('regions').name
      - arg(3)
    query: |-
      INSERT INTO print_order (email, region, city, amount)
      VALUES ($1, $2, $3, $4::DECIMAL)
```

With [named args](#named-args), the same example becomes more readable:

```yaml
run:
  - name: insert_order
    type: exec
    args:
      email: gen('email')
      region: ref_same('regions').name
      city: set_rand(ref_same('regions').cities, [])
      amount: uniform(1, 500)
    print:
      - ref_same('regions').name
      - arg('amount')
    query: |-
      INSERT INTO print_order (email, region, city, amount)
      VALUES ($1, $2, $3, $4::DECIMAL)
```

Print expressions have access to the same context as query args: `ref_same`, `ref_rand`, `arg()`, `global()`, `local()`, and all built-in functions. Expressions using `ref_same` see the same row selected for the query args in that iteration.

#### Custom aggregation

Use the map form with `expr` and `agg` fields for full control over how values are aggregated and displayed. The `agg` field is an [expr](https://expr-lang.org/docs/language-definition) expression evaluated against the accumulated state:

| Variable | Type | Description |
|---|---|---|
| `count` | int | Total observations |
| `freq` | map[string]int | Value frequency distribution |
| `min` | float | Minimum numeric value |
| `max` | float | Maximum numeric value |
| `avg` | float | Mean of numeric values |
| `sum` | float | Sum of numeric values |

These variables can be combined in any [expr-lang](https://expr-lang.org) expression to produce custom summary output:

```yaml
    print:
      - ref_same('regions').name

      - expr: set_rand(ref_same('regions').cities, [])
        agg: "join(map(sortBy(toPairs(freq), -#[1])[:5], #[0] + '=' + string(#[1])), ' ')"

      - expr: arg(3)
        agg: "'$' + string(int(min)) + ' - $' + string(int(max)) + ' (avg $' + string(int(avg)) + ', n=' + string(count) + ')'"
```

#### Output

```
PRINT         VALUES
insert_order  ref_same('regions').name                   us=340 eu=330 ap=330
insert_order  set_rand(ref_same('regions').cities, [])   chicago=120 tokyo=115 london=110 dallas=108 paris=105
insert_order  arg(3)                                     $1 - $499 (avg $250, n=1000)
```

See [`_examples/print/`](https://github.com/codingconcepts/edg/tree/main/_examples/print) for a complete working example.

### Placeholders

Arg placeholders (`$1`, `$2`, etc.) are passed to the database in one of two ways: **inlined** or as **bind params**.

#### Inlining

Inlining means edg performs a text replacement on the SQL string _before_ sending it to the database. Every `$N` in the query is replaced with the literal arg value. For example, given:

```yaml
args:
  - gen_batch(1000, 100, 'email')
query: |-
  INSERT INTO users (email)
  SELECT unnest(string_to_array('$1', __sep__))
```

If `$1` evaluates to `alice@x.com\x1fbob@y.com\x1f...`, the SQL sent to the database becomes:

```sql
INSERT INTO users (email)
SELECT unnest(string_to_array('alice@x.com\x1fbob@y.com\x1f...', chr(31)))
```

The database never sees `$1`, it receives a fully formed query with the values baked in. This is used for:

- **`query_batch` / `exec_batch`** types (always inlined).
- **Batch-expanded queries** using `gen_batch`, `batch`, or `ref_each` (in any section).

Inlining lets you use `$N` as a universal placeholder syntax across all drivers (pgx, MySQL, Oracle, SQL Server) without worrying about driver-specific bind param formats. It also avoids a pgx-stdlib issue where numeric values are sent as DECIMAL, which CockroachDB can't mix with INT in arithmetic.

Because the value is embedded in the SQL text, quoted placeholders like `'$1'` are common in batch patterns, the quotes become part of the final SQL string (e.g. `string_to_array('alice@x.com,...', ',')`).

#### Bind params

All other queries use native driver bind parameters. The placeholder stays in the SQL and the values are sent separately, allowing the database to cache query plans and avoid re-parsing.

Each driver has its own placeholder format:

| Driver | Placeholder format |
|---|---|
| `pgx` (PostgreSQL / CockroachDB) | `$1`, `$2`, `$3` |
| `dsql` (Aurora DSQL) | `$1`, `$2`, `$3` |
| `mysql` | `?` (positional) |
| `oracle` | `:1`, `:2`, `:3` |
| `mssql` | `@p1`, `@p2`, `@p3` |
| `spanner` | `@p1`, `@p2`, `@p3` |

Since `run` queries always use bind params, their SQL must use the correct format for the target driver.

### Column Name Normalisation

When a `query` or `query_batch` result is stored, all column names are lowercased before being added to the environment. This means a SQL column `W_ID` becomes accessible as `ref_rand('fetch_warehouses').w_id`, not `.W_ID`.

## Run Weights

The optional `run_weights` map controls the workload mix during execution. Each key is a run item name (either a standalone query name or a transaction name), and the value is a relative weight. On each iteration, a single item is chosen by weighted random selection:

```yaml
run_weights:
  check_balance: 50
  make_transfer: 50

run:
  - name: check_balance
    type: query
    args: [ref_rand('fetch_accounts').id]
    query: SELECT balance FROM account WHERE id = $1::UUID

  - transaction: make_transfer
    locals:
      amount: gen('number:1,100')
    queries:
      - name: read_balance
        type: query
        args: [ref_diff('fetch_accounts').id]
        query: SELECT id, balance FROM account WHERE id = $1::UUID
      - name: debit
        type: exec
        args:
          - ref_same('read_balance').id
          - local('amount')
        query: UPDATE account SET balance = balance - $2::FLOAT WHERE id = $1::UUID
```

In this example, each iteration picks either the standalone `check_balance` query (50% of the time) or the entire `make_transfer` transaction (50% of the time). When a transaction is selected, all its queries run inside a single `BEGIN`/`COMMIT` block.

If `run_weights` is omitted, all `run` items execute sequentially on each iteration.

## Workers

The `workers` section defines background queries that run independently on a fixed schedule alongside the main workload. Each worker is a regular query with an added `rate` field controlling execution frequency.

```yaml
workers:
  - name: reap_expired_leases
    rate: 1/5s
    type: exec
    query: |-
      UPDATE runs
      SET status = 'pending', worker_id = NULL
      WHERE status IN ('claimed', 'running')
        AND lease_expires_at < now()

  - name: refresh_stats
    rate: 3/1m
    type: query
    query: SELECT count(*) AS total FROM events
```

Workers are useful for background maintenance tasks that should run on a fixed cadence, independent of the main workload loop. For example: lease reapers, stats refreshers, cache warmers, or periodic cleanup jobs.

### Rate

The `rate` field specifies how many times the query executes per interval, using the format `times/interval`:

| Example | Meaning |
|---|---|
| `1/10s` | Once every 10 seconds |
| `3/1m` | 3 times every minute / once every 20 seconds |
| `5/1m30s` | 5 times every minute and a half / once every 18 seconds |
| `2/1s` / `1/500ms` | 2 times every second |

The interval uses Go duration syntax (`s`, `ms`, `m`, `h`). Executions are evenly spaced: `3/1m` fires every 20 seconds, not 3 times at the start of each minute.

> [!NOTE]
> The `rate` property can achieve the same interval in a number of ways (e.g. `2/1s` and `1/500ms` both result in a worker that fires twice every second), so use whichever expresses your intent the best and is easiest to read.

### Behaviour

- Each worker runs in its own goroutine with its own environment, so workers are safe to use with `ref_*` functions and prepared statements.
- Worker query results flow into the same stats and metrics pipeline as `run` queries, so they appear in progress output, the summary table, Prometheus metrics, and `expectations`.
- Workers support all the same fields as regular queries: `type`, `args`, `prepared`, `row`, etc.
- In staged mode, workers run for the entire duration across all stages (not restarted per stage).
- Workers respect context cancellation and stop when the workload finishes or is interrupted.

See [`_examples/workers/`](https://github.com/codingconcepts/edg/tree/main/_examples/workers) for a complete working example.

## Expectations

The `expectations` section defines assertions that are evaluated after the workload finishes. Each expectation is a boolean expression checked against the collected run metrics. If any expectation fails, edg prints the results and exits with a non-zero status code, making it suitable for CI/CD pipelines.

```yaml
expectations:
  - error_rate < 1
  - check_balance.p99 < 100
  - tpm > 5000
```

Expressions use the same [expr](https://expr-lang.org/) engine as arg expressions and must evaluate to a boolean.

### Referencing globals

Expectations can reference any variable defined in the `globals` section. This avoids hardcoding values that already exist in your config:

```yaml
globals:
  accounts: 10000
  max_error_pct: 5

expectations:
  - error_rate < max_error_pct
  - query: SELECT COUNT(*) AS cnt FROM account
    expr: cnt == accounts
```

Global names must not collide with built-in metrics (`error_rate`, `success_count`, `total_errors`, `tpm`). edg will reject the config at startup if they do.

### Database-backed expectations

An expectation can be an object with `query` and `expr` fields. The SQL query runs after the workload finishes and its first-row columns are available in the expression alongside globals and metrics:

```yaml
expectations:
  - query: SELECT COUNT(*) AS cnt FROM account
    expr: cnt == accounts
  - query: SELECT SUM(balance) AS total FROM account
    expr: total > 0
```

### Available Metrics

Expectations have access to globals, global metrics, and per-query metrics. All latency values are in milliseconds and all error rates are percentages (0–100).

#### Global metrics

| Metric | Type | Description |
|---|---|---|
| `error_rate` | float | Overall error rate as a percentage between 0 and 100. Calculated as `error_count / (error_count + success_count) * 100`. |
| `success_count` | int | Total successful operations across all queries. |
| `total_errors` | int | Total failed operations across all queries. |
| `tpm` | float | Transactions per minute (success_count / elapsed minutes). |

#### Per-query metrics

Per-query metrics are accessed using dot notation with the query name, e.g. `check_balance.p99`. The query name must match the `name` field in your `run` section.

| Metric | Type | Description |
|---|---|---|
| `<query>.success_count` | int | Successful operations for this query. |
| `<query>.error_count` | int | Failed operations for this query. |
| `<query>.error_rate` | float | Error rate as a percentage. |
| `<query>.avg` | float | Average latency in ms. |
| `<query>.p50` | float | 50th percentile latency in ms. |
| `<query>.p95` | float | 95th percentile latency in ms. |
| `<query>.p99` | float | 99th percentile latency in ms. |
| `<query>.qps` | float | Queries per second. |

### Examples

Ensure the overall error rate stays below 1%:

```yaml
expectations:
  - error_rate < 1
```

Ensure a specific query's p99 latency stays under 100ms and its error rate is zero:

```yaml
expectations:
  - check_balance.p99 < 100
  - check_balance.errors == 0
```

Combine multiple conditions in a single expression:

```yaml
expectations:
  - error_rate < 0.5 && tpm > 10000
```

### Output

After the run summary, expectations are printed with a PASS/FAIL status:

```
expectations
  PASS  error_rate < 1
  PASS  check_balance.p99 < 100
  FAIL  tpm > 5000
```

If any expectation fails, edg exits with status code 1 and reports the number of failures. When using the `all` command, teardown (`deseed` and `down`) still runs before the non-zero exit, so your database is left clean regardless of expectation results.

### CI/CD Usage

A typical CI pipeline step runs the workload and relies on the exit code to gate the build:

```sh
edg all \
  --driver pgx \
  --config workload.yaml \
  --url "$DATABASE_URL" \
  -w 50 \
  -d 5m
```

If any expectation defined in `workload.yaml` fails, the command exits with code 1, failing the pipeline step.

For a complete guide to using edg as an integration testing tool, see [Integration Testing]({{< relref "integration-testing" >}}).

## Includes

Use the `!include` YAML tag to split workload configs into reusable fragments. This is useful when multiple workloads share the same schema, reference data, or expressions.

```yaml
globals: !include shared/globals.yaml
up: !include shared/schema.yaml
down: !include shared/teardown.yaml
run: !include shared/run_queries.yaml
```

Paths are resolved relative to the file containing the `!include` directive.

### Mapping value

Replace a key's value with the content of a file:

```yaml
globals: !include shared/globals.yaml
```

Where `shared/globals.yaml` contains:

```yaml
batch_size: 10000
customers: 100000
```

### Sequence value

Replace an entire list:

```yaml
up: !include shared/schema.yaml
```

Where `shared/schema.yaml` contains:

```yaml
- name: create_users
  query: CREATE TABLE users (id UUID PRIMARY KEY, email STRING NOT NULL)
- name: create_orders
  query: CREATE TABLE orders (id UUID PRIMARY KEY, user_id UUID REFERENCES users(id))
```

### Sequence item

Splice items from an included file into a list alongside local entries:

```yaml
run:
  - name: local_query
    type: query
    query: SELECT 1
  - !include shared/extra_queries.yaml
```

Items from the included file are merged into the parent sequence rather than nested.

### Nested includes

Included files can themselves use `!include`. Circular includes are detected and produce an error.

See [`_examples/includes/`](https://github.com/codingconcepts/edg/tree/main/_examples/includes) for a complete working example.
