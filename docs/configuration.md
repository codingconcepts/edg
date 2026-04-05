---
title: Configuration
layout: default
nav_order: 4
---

# Configuration

Workloads are defined in a single YAML file with the following top-level keys:

```yaml
# Variables available in all expressions.
globals:

# User-defined expression functions.
expressions:

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

Queries can also specify a `wait` duration (e.g. `wait: 18s`) to introduce a keying/think-time delay after execution in the `run` section.

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
