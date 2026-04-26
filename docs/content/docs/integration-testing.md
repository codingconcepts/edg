---
title: Integration Testing
weight: 7
---

# Integration Testing

edg is uniquely suited to work as a self-contained integration testing tool for databases. A single YAML config expresses a full test, all the way from schema creation, data population, query execution, assertions, and tear down. The `edg all` command runs this entire lifecycle and fails if any query errors or expectation is not met, making it a drop-in CI gate.

## How it works

The `all` command runs five phases in order:

```
up  ->  seed  ->  run  ->  deseed  ->  down
```

1. **up** - creates tables and indexes
2. **seed** - populates them with realistic data
3. **run** - executes your query workload with concurrent workers, collecting latency and error metrics
4. **deseed** - truncates tables
5. **down** - drops tables

After `run`, any [expectations]({{< relref "configuration" >}}#expectations) are evaluated against the collected metrics. If an expectation fails, edg still runs teardown (`deseed` and `down`) before exiting with a failure code.

## Writing an integration test

A complete integration test in a single config:

```yaml
globals:
  customers: 10000
  initial_balance: 10000
  batch_size: 5000

up:
  - name: create_customer
    query: |-
      CREATE TABLE IF NOT EXISTS customer (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        email STRING NOT NULL
      )

  - name: create_account
    query: |-
      CREATE TABLE IF NOT EXISTS account (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        balance FLOAT NOT NULL,
        customer_id UUID NOT NULL REFERENCES customer(id)
      )

seed:
  - name: populate_customer
    args:
      - gen_batch(customers, batch_size, 'email')
    query: |-
      INSERT INTO customer (email)
      SELECT unnest(string_to_array('$1', sep))

  - name: populate_account
    args:
      - batch(customers / batch_size)
      - initial_balance
      - batch_size
    query: |-
      INSERT INTO account (balance, customer_id)
      SELECT $2::FLOAT, c.id
      FROM customer c
      ORDER BY c.id
      OFFSET $1::INT * $3::INT
      LIMIT $3::INT

init:
  - name: fetch_accounts
    type: query
    query: SELECT id FROM account ORDER BY random()

run_weights:
  check_balance: 50
  credit_account: 50

run:
  - name: check_balance
    type: query
    args:
      - ref_rand('fetch_accounts').id
    query: |-
      SELECT balance FROM account WHERE id = $1::UUID

  - name: credit_account
    type: exec
    args:
      - ref_rand('fetch_accounts').id
      - gen('number:1,1000')
    query: |-
      UPDATE account SET balance = balance + $2::FLOAT
      WHERE id = $1::UUID

expectations:
  - error_rate < 1
  - check_balance.p99 < 50
  - credit_account.p99 < 50
  - tpm > 1000

deseed:
  - name: truncate_account
    type: exec
    query: TRUNCATE TABLE account CASCADE

  - name: truncate_customer
    type: exec
    query: TRUNCATE TABLE customer CASCADE

down:
  - name: drop_account
    type: exec
    query: DROP TABLE IF EXISTS account

  - name: drop_customer
    type: exec
    query: DROP TABLE IF EXISTS customer
```

Run it:

```sh
edg all \
  --driver pgx \
  --config integration-test.yaml \
  --url ${DATABASE_URL} \
  -w 10 \
  -d 30s
```

edg creates the schema, seeds 10,000 customers with accounts, runs balance checks and credits with 10 workers for 30 seconds, then asserts that:

- The overall error rate stays below 1%
- Both queries stay under 50ms at p99
- Throughput exceeds 1,000 transactions per minute

If any assertion fails, the test is deemed to have failed. The database is cleaned up either way.

## What to test

The `expectations` section supports [global and per-query metrics]({{< relref "configuration" >}}#available-metrics):

**Correctness** - queries should not error:

```yaml
expectations:
  - error_rate == 0
  - make_transfer.error_count == 0
```

**Latency** - queries should respond within bounds:

```yaml
expectations:
  - get_user.p99 < 25
  - checkout.p99 < 200
```

**Throughput** - the system should sustain a minimum transaction rate:

```yaml
expectations:
  - tpm > 5000
```

**Combined** - multiple conditions in a single expression:

```yaml
expectations:
  - error_rate < 0.5 && tpm > 10000
```

**Referencing globals** - use variables from the `globals` section to avoid hardcoding values:

```yaml
globals:
  accounts: 10000
  max_error_pct: 5

expectations:
  - error_rate < max_error_pct
  - query: SELECT COUNT(*) AS cnt FROM account
    expr: cnt == accounts
```

## Multiple test scenarios

Use separate config files for different test scenarios, or use [!includes]({{< relref "configuration" >}}#includes) to share schema definitions while varying seed data and workloads:

```
test-fixtures/
  shared/
    schema.yaml    # up + down (shared across scenarios)
    teardown.yaml  # deseed (shared across scenarios)
  happy-path.yaml  # seed for standard flow
  edge-cases.yaml  # seed for boundary conditions
  empty-state.yaml # no seed, just schema
```

```yaml
# happy-path.yaml
globals:
  users: 500
  batch_size: 100

up: !include shared/schema.yaml
seed:
  - name: populate_users
    args:
      - gen_batch(users, batch_size, 'email')
    query: |-
      INSERT INTO users (email)
      SELECT unnest(string_to_array('$1', sep))
deseed: !include shared/teardown.yaml
down: !include shared/schema.yaml
```

Run the scenario you need:

```sh
edg all \
  --driver pgx \
  --config test-fixtures/happy-path.yaml \
  --url ${DATABASE_URL} \
  -w 5 \
  -d 30s
```

## Deterministic seeding

The `--rng-seed` flag makes expression output deterministic. Two runs with the same seed produce identical generated values:

```sh
edg all \
  --driver pgx \
  --config integration-test.yaml \
  --url ${DATABASE_URL} \
  --rng-seed 42 \
  -w 10 \
  -d 30s
```

Functions like `gen()`, `uniform()`, `set_rand()`, and other random expressions return the same sequence each time. This makes test runs reproducible and debugging easier because the data is predictable across runs.

## CI pipeline example

### GitHub Actions

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: testdb
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U test"
          --health-interval 5s
          --health-timeout 5s
          --health-retries 5

    env:
      DATABASE_URL: "postgres://test:test@localhost:5432/testdb?sslmode=disable"

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: stable

      - name: Install edg
        run: go install github.com/codingconcepts/edg@latest

      - name: Run integration tests
        run: |
          edg all \
            --driver pgx \
            --config integration-test.yaml \
            --url ${DATABASE_URL} \
            --rng-seed 42 \
            -w 10 \
            -d 30s
```

Because `edg all` handles setup, execution, assertions, and teardown in a single command, the pipeline step is just one invocation. A non-zero exit fails the build.

### Docker Compose

For local development, pair edg with a `docker-compose.yaml` that spins up the database:

```yaml
services:
  db:
    image: postgres:16
    environment:
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
      POSTGRES_DB: testdb
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "test"]
      interval: 2s
      timeout: 5s
      retries: 5
```

Then run tests as follows:

```sh
docker compose up -d --wait

edg all \
  --driver pgx \
  --config integration-test.yaml \
  --url ${DATABASE_URL} \
  --rng-seed 42 \
  -w 10 \
  -d 30s

docker compose down
```

## Running phases separately

For debugging or when you want to inspect the database between phases, run each step individually:

```sh
edg up --driver pgx --config integration-test.yaml --url ${DATABASE_URL}
edg seed --driver pgx --config integration-test.yaml --url ${DATABASE_URL}

# Inspect the database, run ad-hoc queries, etc.

edg run --driver pgx --config integration-test.yaml --url ${DATABASE_URL} -w 10 -d 30s
edg deseed --driver pgx --config integration-test.yaml --url ${DATABASE_URL}
edg down --driver pgx --config integration-test.yaml --url ${DATABASE_URL}
```

## Driver-specific considerations

The examples above use PostgreSQL/CockroachDB (`pgx`) syntax. When targeting other drivers, adjust the SQL accordingly. Key differences for Spanner (GoogleSQL):

| Concern | pgx | Spanner |
|---|---|---|
| UUID column | `UUID DEFAULT gen_random_uuid()` | `STRING(36) DEFAULT (GENERATE_UUID())` |
| Primary key | inline `PRIMARY KEY` | `PRIMARY KEY (col)` at table level |
| Batch expansion | `unnest(string_to_array('$1', sep))` | `UNNEST(SPLIT('$1', CODE_POINTS_TO_STRING([31])))` |
| Type cast | `$1::UUID`, `$2::FLOAT` | `CAST($1 AS STRING)`, `CAST($2 AS FLOAT64)` |
| Random ordering | `ORDER BY random()` | `TABLESAMPLE RESERVOIR (N ROWS)` |
| Cleanup | `TRUNCATE TABLE t CASCADE` | `DELETE FROM t WHERE TRUE` |
| Bind params | `$1`, `$2` | `@p1`, `@p2` |

For complete Spanner examples, see the [built-in workloads]({{< relref "cli-reference" >}}#workload) which include Spanner variants for every benchmark.

## Tips

- **Use `--rng-seed` in CI.** Deterministic data eliminates an entire class of flaky tests caused by random values.
- **Keep seed data small.** Integration tests should be fast. Thousands of rows are usually enough -- save millions for load testing.
- **Validate configs in CI.** Add `edg validate --config integration-test.yaml` as an earlier pipeline step to catch YAML errors before they hit the database.
- **Use `stages` for ramp-up tests.** The [stages]({{< relref "configuration" >}}#stages) section lets you vary worker counts and durations across phases, useful for testing how your database handles increasing load.
