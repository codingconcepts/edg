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
  - customers / districts   # evaluates to 3000
  - warehouses * 10         # evaluates to 10
```

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
        string_to_array('$1', ','),
        string_to_array('$2', ','),
        string_to_array('$3', ',')
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
      SELECT unnest(string_to_array('$1', ','))
```

- **`up`** and **`down`** manage schema (CREATE/DROP).
- **`seed`** and **`deseed`** manage data (INSERT/TRUNCATE).
- **`init`** runs once per worker before the workload starts, typically to fetch reference data for use in `run` queries.
- **`run`** contains the transactional workload queries executed in a loop.

## Query Types

| Type | Description |
|---|---|
| `query` (default) | Executes the SQL and reads result rows. Results are stored in separate memory for each worker by query name, making them available to `ref_*` functions. |
| `exec` | Executes the SQL without reading results. Use for DDL, DML that returns no rows, or when results aren't needed. |
| `query_batch` | Like `query`, but evaluates args repeatedly (controlled by `count` and `size`) and collects values into comma-separated strings per arg position. Each batch becomes a separate query execution whose results are stored. |
| `exec_batch` | Like `exec`, but evaluates args repeatedly (controlled by `count` and `size`) and collects values into comma-separated strings per arg position. Each batch becomes a separate exec. |

### Batch Fields

The `query_batch` and `exec_batch` types use two additional fields to control how args are generated and grouped:

| Field | Description |
|---|---|
| `count` | Total number of rows to generate. Evaluated as an expression, so it can reference globals. |
| `size` | Number of rows per batch. If omitted or zero, defaults to `count` (single batch). Also evaluated as an expression. |

Each arg expression is evaluated once per row. The results are collected into comma-separated strings per arg position. For example, with `count: 1000` and `size: 100`, you get 10 batches, each containing a comma-separated string of 100 generated values.

```yaml
seed:
  - name: populate_users
    type: exec_batch
    count: customers          # expression: uses the 'customers' global
    size: batch_size          # expression: uses the 'batch_size' global
    args:
      - gen('email')
    query: |-
      INSERT INTO users (email)
      SELECT unnest(string_to_array('$1', ','))
```

### Wait

Queries can specify a `wait` duration (e.g. `wait: 18s`) to introduce a keying/think-time delay after execution. This only applies to queries in the `run` section and is ignored in other sections.

### Placeholders

Arg placeholders (`$1`, `$2`, etc.) are passed to the database in one of two ways: **inlined** or as **bind params**.

#### Inlining

Inlining means edg performs a text replacement on the SQL string _before_ sending it to the database. Every `$N` in the query is replaced with the literal arg value. For example, given:

```yaml
args:
  - gen_batch(1000, 100, 'email')
query: |-
  INSERT INTO users (email)
  SELECT unnest(string_to_array('$1', ','))
```

If `$1` evaluates to `alice@x.com,bob@y.com,...`, the SQL sent to the database becomes:

```sql
INSERT INTO users (email)
SELECT unnest(string_to_array('alice@x.com,bob@y.com,...', ','))
```

The database never sees `$1`, it receives a fully formed query with the values baked in. This is used for:

- **`query_batch` / `exec_batch`** types (always inlined).
- **Batch-expanded queries** using `gen_batch`, `batch`, or `ref_each` (in any section).

Inlining lets you use `$N` as a universal placeholder syntax across all drivers (pgx, MySQL, Oracle) without worrying about driver-specific bind param formats. It also avoids a pgx-stdlib issue where numeric values are sent as DECIMAL, which CockroachDB can't mix with INT in arithmetic.

Because the value is embedded in the SQL text, quoted placeholders like `'$1'` are common in batch patterns, the quotes become part of the final SQL string (e.g. `string_to_array('alice@x.com,...', ',')`).

#### Bind params

All other queries use native driver bind parameters. The placeholder stays in the SQL and the values are sent separately, allowing the database to cache query plans and avoid re-parsing.

Each driver has its own placeholder format:

| Driver | Placeholder format |
|---|---|
| `pgx` (PostgreSQL / CockroachDB) | `$1`, `$2`, `$3` |
| `mysql` | `?` (positional) |
| `oracle` | `:1`, `:2`, `:3` |

Since `run` queries always use bind params, their SQL must use the correct format for the target driver.

### Column Name Normalisation

When a `query` or `query_batch` result is stored, all column names are lowercased before being added to the environment. This means a SQL column `W_ID` becomes accessible as `ref_rand('fetch_warehouses').w_id`, not `.W_ID`.

## Run Weights

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
