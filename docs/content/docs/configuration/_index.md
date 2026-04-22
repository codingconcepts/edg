---
title: Configuration
weight: 3
bookCollapseSection: true
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
