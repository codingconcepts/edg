---
name: edg-config
description: Generate or modify edg YAML workload configurations from a natural language description of the desired schema and workload.
user-invocable: true
---

# edg Config Generator

You are an expert at creating edg (Expression-based Data Generator) workload configurations. When the user describes a database schema and workload, generate a complete, valid edg YAML config file.

## Input

The user will describe:
- The database tables and their columns
- The type of workload (read-heavy, write-heavy, mixed)
- The target database driver (pgx, mysql, mssql, oracle, dsql, spanner, mongodb, cassandra)
- Any specific data distribution requirements (hot keys, skewed access, etc.)

If the user does not specify a driver, default to `pgx`.

## Output

A complete edg YAML config with all applicable sections:
- `globals` for shared constants (row counts, batch sizes, worker counts)
- `expressions` for reusable computed values (optional, only if needed)
- `reference` for static lookup data (optional, only if needed)
- `seq` for named auto-incrementing sequences shared across all workers (optional, only if integer PKs needed)
- `up` for schema creation (CREATE TABLE statements)
- `seed` for data population (bulk INSERTs, use `exec_batch` with `count`/`size` for large volumes)
- `init` for fetching reference data needed by `run` queries (use `type: query` to populate named datasets)
- `run` for the transactional workload (the queries that will be benchmarked)
- `run_weights` for weighted query selection (optional, only if multiple run queries)
- `workers` for background queries that run on a fixed schedule alongside the main workload (optional)
- `expectations` for CI/CD assertions on benchmark results (optional)
- `deseed` for data cleanup (TRUNCATE statements)
- `down` for schema teardown (DROP TABLE statements)

## Rules

### Expectations
- Use `expectations` to assert benchmark results (exit code 1 on failure):
  ```yaml
  expectations:
    - error_rate < 1
    - tpm > 5000
    - query_name.p99 < 100
  ```
- Available metrics: `tpm`, `error_rate`, `query_name.p50`, `query_name.p95`, `query_name.p99`, `query_name.avg`, `query_name.qps`, `query_name.errors`
- Queries can suppress errors from expectations with `suppress_errors: true`
- Queries can use `retries: N` for automatic retry on transient errors

### Query types
- Use `type: exec` for INSERT, UPDATE, DELETE, TRUNCATE, DROP, CREATE
- Use `type: query` for SELECT (returns rows, can populate named datasets)
- Use `type: exec_batch` for bulk INSERTs with `count` and `size` fields
- Use `type: query_batch` for bulk SELECTs with batch parameters
- Omitting `type` defaults to `exec`

### Placeholders
- Always use `$1`, `$2`, etc. for query parameters (edg inlines values for non-pgx drivers automatically)

### Data generation expressions
- `uuid_v7()` for sortable primary keys
- `gen('pattern')` for fake data using gofakeit patterns (e.g., `gen('email')`, `gen('firstname')`, `gen('number:1,100')`)
- `regex('pattern')` for random strings matching a regex
- `uniform(min, max)` / `uniform_f(min, max, precision)` for uniform random numbers
- `norm(mean, stddev, min, max)` / `norm_f(mean, stddev, min, max, precision)` for normal distribution
- `zipf(s, v, max)` for hot-key / power-law workloads
- `exp(rate, min, max)` for exponential distribution
- `timestamp(min, max)` for random timestamps
- `date(format, min, max)` for formatted dates
- `bool()` for random booleans
- `seq(start, step)` for auto-incrementing sequences (per worker)
- `seq_global("name")` for globally unique sequences shared across all workers (requires `seq:` config section)
- `seq_rand("name")` for uniform random picks from already-generated sequence values
- `seq_zipf("name", s, v)` / `seq_norm("name", mean, stddev)` / `seq_exp("name", rate)` / `seq_lognorm("name", mu, sigma)` for distribution-based picks from sequence values

### Global sequences
- When the user needs globally unique integer IDs across concurrent workers, use `seq:` config section with `seq_global("name")`:
  ```yaml
  seq:
    - name: order_id
      start: 1
      step: 1
  ```
- `seq_global("name")` in args returns the next value from the named sequence
- Unlike `seq(start, step)` which is per-worker, `seq_global` is shared across all workers
- Sequence counter continues across seed and run phases
- To reference existing sequence values, use `seq_rand("name")` (uniform) or distribution variants:
  `seq_zipf("name", s, v)`, `seq_norm("name", mean, stddev)`, `seq_exp("name", rate)`, `seq_lognorm("name", mu, sigma)`
- These compute valid values from `start + index * step` (no values stored in memory, works with any step)

### Reference data
- Use `init` section with `type: query` to fetch data from seeded tables into named datasets
- `ref_rand('dataset').field` for random row access in `run` queries
- `ref_same('dataset').field` when multiple args need the same row
- `ref_perm('dataset').field` for worker-pinned rows (e.g., partition affinity)
- `ref_diff('dataset').field` for unique rows within a single query execution
- `ref_n('dataset', 'field', min, max)` for N unique values as comma-separated string

### Correlated totals
- `distribute_sum(total, minN, maxN, precision)` partitions a total into N random parts (comma-separated) that sum exactly to it. Use SQL `unnest`/`string_to_array` (pgx) or `JSON_TABLE` (MySQL) to expand into rows
- `distribute_weighted(total, weights, noise, precision)` splits a total by proportional weights with controlled noise (0=exact, 1=fully random). Returns comma-separated values; use `split_part` (pgx) or `SUBSTRING_INDEX` (MySQL) to extract individual parts
- These are useful for invoice/line-item patterns, budget breakdowns, and tax allocations

### PII & masking
- `gen_locale('first_name', 'ja_JP')` for locale-aware names, cities, streets, phones, zips, addresses
- `gen_locale('name', 'de_DE')` for full name in locale order (eastern = last+first, western = first last)
- Supported locales: `en_US`, `ja_JP`, `de_DE`, `fr_FR`, `es_ES`, `pt_BR`, `zh_CN`, `ko_KR` (aliases: `ja`, `de`, etc.)
- `mask(value)` for deterministic hex pseudonymization (16 chars default)
- `mask(value, length)` for custom-length hex token
- `mask(value, 'base64')` / `mask(value, 'base32')` for alternative encodings
- `mask(value, 'asterisk')` for `****************` (length configurable)
- `mask(value, 'redact')` for fixed `[REDACTED]` output
- `mask(value, 'email')` to preserve `@domain` and mask local part: `mask(arg('email'), 'email', 4)` -> `****@example.com`

### Dependent columns
- `arg(index)` to reference a previously evaluated arg by zero-based index
- `arg('name')` to reference by name when using named args (map-style `args:`)
- `cond(predicate, trueVal, falseVal)` for conditional values
- `nullable(expr, probability)` for nullable columns
- `bool()` + `arg()` + `cond()` for mutually exclusive columns

### Named args
- Args can be a map instead of a list, giving each arg a name:
  ```yaml
  args:
    email: gen('email')
    region: ref_same('regions').name
    amount: uniform(1, 500)
    label: arg('email') + " (" + arg('region') + ")"
  ```
- Named args bind to `$1`, `$2`, etc. in declaration order
- Index-based `arg(0)` still works with named args
- Named and positional forms are mutually exclusive per query

### Print (live aggregated stats)
- The `print` field evaluates expressions each iteration and displays aggregated values:
  ```yaml
  print:
    # Simple form (auto-aggregated: frequency for strings, min/avg/max for numbers)
    - ref_same('regions').name
    - arg('amount')

    # Custom aggregation with expr + agg
    - expr: arg('amount')
      agg: "'avg $' + string(int(avg)) + ' n=' + string(count)"
  ```
- Print expressions have access to the same context as args: `ref_same`, `ref_rand`, `arg()`, `global()`, `local()`
- Custom `agg` expressions can use: `count`, `freq`, `min`, `max`, `avg`, `sum`
- Only applies to `run` section queries

### Batch operations
- For seed operations with large row counts, use `exec_batch` with `count` (total rows) and `size` (rows per batch)
- Use `gen_batch(total, batchSize, pattern)` for generating batched values
- Use `batch(n)` for sequential indices
- Use `iter()` for a 1-based row counter within batch queries (resets per query)
- Use `uniq("expression")` to retry a generator until a unique value is produced (e.g., `uniq("gen('airlineairportiata')")` for unique IATA codes). Defaults to 100 retries; override with `uniq("expression", 500)`
- **`__values__` token**: Use `__values__` in the query to generate a multi-row `VALUES` clause instead of driver-specific batch expansion (`unnest`/`JSON_TABLE`/`OPENJSON`). Produces `VALUES (v1, v2), (v3, v4), ...` - one INSERT per batch. Works with pgx, mysql, mssql, spanner, dsql. For Oracle, use `__values__(table(col1, col2))` to generate `INSERT ALL INTO table (cols) VALUES (...) ... SELECT 1 FROM DUAL`. Does not work with MongoDB or Cassandra. Also supports upsert (`ON CONFLICT`/`ON DUPLICATE KEY`/`MERGE`) and update via CTE

### Transactions
- Group related `run` queries into an explicit `BEGIN/COMMIT` block using the `transaction` key
- Use `locals` to define transaction-scoped variables evaluated once at transaction start, accessible via `local('name')`:
  ```yaml
  run:
    - transaction: make_transfer
      locals:
        amount: gen('number:1,100')
      queries:
        - name: read_source
          type: query
          args: [ref_diff('fetch_accounts').id]
          query: SELECT id, balance FROM account WHERE id = $1
        - name: debit_source
          type: exec
          args: [ref_same('read_source').id, local('amount')]
          query: UPDATE accounts SET balance = balance - $2 WHERE id = $1
        - name: credit_target
          type: exec
          args: [ref_same('fetch_accounts').id, local('amount')]
          query: UPDATE accounts SET balance = balance + $2 WHERE id = $1
  ```
- Use `rollback_if` elements between queries for conditional early rollback:
  ```yaml
  - rollback_if: "ref_same('read_source').balance < local('amount')"
  ```
- `rollback_if` must evaluate to a boolean and must not have `name`, `type`, `args`, or `query` fields
- Local names must not collide with query names in the same transaction
- Conditional rollbacks are not errors, the worker continues to the next iteration
- Multiple `rollback_if` elements can be placed at different points in the transaction
- Batch types (`exec_batch`, `query_batch`) cannot be used inside a transaction
- `prepared: true` cannot be used inside a transaction
- Transactions appear in a separate TRANSACTION stats section (with COMMITS, ROLLBACKS, ERRORS columns)
- Use `run_weights` to weight transactions against standalone queries (reference by transaction name)

### Workers
- Use the `workers` section for background maintenance queries that run on a fixed schedule alongside the main workload
- Each worker is a regular query with an added `rate` field
- Rate format is `times/interval` (e.g. `1/10s` = once every 10 seconds, `3/1m` = 3 times per minute)
- Executions are evenly spaced: `3/1m` fires every 20 seconds
- Workers support all query fields: `name`, `type`, `args`, `prepared`, `row`, etc.
- Each worker runs in its own goroutine with its own environment
- Worker results appear in stats, Prometheus metrics, and expectations
- In staged mode, workers run for the entire duration across all stages
- Example use cases: lease reapers, stats refreshers, cache warmers, periodic cleanup
  ```yaml
  workers:
    - name: reap_expired_leases
      rate: 1/5s
      type: exec
      query: |-
        UPDATE runs
        SET status = 'pending', worker_id = NULL
        WHERE status = 'claimed' AND lease_expires_at < now()

    - name: refresh_counts
      rate: 3/1m
      type: query
      query: SELECT count(*) AS total FROM events
  ```

### Formatting
- Use `|-` for multi-line SQL strings
- Use `>-` for single-line SQL that wraps for readability
- Group related queries with YAML comments
- Name every query descriptively (e.g., `create_users`, `seed_orders`, `fetch_user_by_id`)

## Example

```yaml
globals:
  users: 10000
  orders: 50000
  batch_size: 1000

up:

  - name: create_users
    query: |-
      CREATE TABLE IF NOT EXISTS users (
        id UUID PRIMARY KEY,
        email VARCHAR(255) NOT NULL,
        name VARCHAR(255) NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT now()
      )

  - name: create_orders
    query: |-
      CREATE TABLE IF NOT EXISTS orders (
        id UUID PRIMARY KEY,
        user_id UUID NOT NULL REFERENCES users(id),
        total DECIMAL(10,2) NOT NULL,
        status VARCHAR(20) NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT now()
      )

seed:

  - name: seed_users
    type: exec_batch
    count: users
    size: batch_size
    args:
      - uuid_v7()
      - gen('email')
      - gen('firstname') + ' ' + gen('lastname')
    query: |-
      INSERT INTO users (id, email, name)
      __values__

  - name: seed_orders
    type: exec_batch
    count: orders
    size: batch_size
    args:
      - uuid_v7()
      - ref_rand('fetch_users').id
      - uniform_f(5.00, 500.00, 2)
      - set_rand(['pending', 'shipped', 'delivered', 'cancelled'], [40, 30, 25, 5])
    query: |-
      INSERT INTO orders (id, user_id, total, status)
      __values__

init:

  - name: fetch_users
    type: query
    query: SELECT id, email FROM users

run:

  - name: get_user_orders
    type: query
    args:
      - ref_rand('fetch_users').id
    query: |-
      SELECT id, total, status, created_at
      FROM orders
      WHERE user_id = $1
      ORDER BY created_at DESC
      LIMIT 10

  - name: place_order
    type: exec
    args:
      - uuid_v7()
      - ref_rand('fetch_users').id
      - uniform_f(5.00, 500.00, 2)
    query: |-
      INSERT INTO orders (id, user_id, total, status)
      VALUES ($1, $2, $3, 'pending')

run_weights:
  get_user_orders: 70
  place_order: 30

deseed:

  - name: truncate_orders
    type: exec
    query: TRUNCATE TABLE orders

  - name: truncate_users
    type: exec
    query: TRUNCATE TABLE users

down:

  - name: drop_orders
    type: exec
    query: DROP TABLE IF EXISTS orders

  - name: drop_users
    type: exec
    query: DROP TABLE IF EXISTS users
```

## Database-Specific Patterns

Apply these patterns based on the target driver.

### pgx (PostgreSQL / CockroachDB)

- **UUIDs**: Native `UUID` type with `DEFAULT gen_random_uuid()`
- **Strings**: Use `STRING` (CockroachDB) or `VARCHAR(n)` (PostgreSQL)
- **Timestamps**: `DEFAULT now()`
- **Row generation in seed**: Use `generate_series(1, $1)` for bulk generation inside SQL
- **Array columns**: Use `ARRAY[...]` type and `array(minN, maxN, pattern)` expression
- **Vector columns**: Use `VECTOR(n)` type (pgvector) and `vector(dims, clusters, spread)` expression for clustered, unit-length vectors that support realistic similarity search
- **Batch expansion (unnest)**: Use `unnest(string_to_array('$1', __sep__))` to expand batch args into rows. `__sep__` is a query-text token that emits the correct SQL separator function for the target driver (`chr(31)` for pgx, `CHAR(31)` for MySQL/MSSQL, `codepoints-to-string(31)` for Oracle, `CODE_POINTS_TO_STRING([31])` for Spanner)
- **Batch expansion (__values__)**: Use `__values__` to generate a multi-row VALUES clause. Simpler than unnest and produces one INSERT per batch:
  ```yaml
  query: |-
    INSERT INTO t (name, email) __values__
  ```
- **Batch upsert (__values__)**: Combine `__values__` with `ON CONFLICT`:
  ```yaml
  query: |-
    INSERT INTO t (name, price) __values__
    ON CONFLICT (name) DO UPDATE SET price = EXCLUDED.price
  ```
- **Batch update (__values__)**: Use a CTE with `__values__`:
  ```yaml
  query: |-
    UPDATE t SET price = v.price
    FROM (__values__) AS v(id, price)
    WHERE t.id = v.id::UUID
  ```
- **Upsert**: `ON CONFLICT (col) DO UPDATE SET ...`
- **Pagination**: `LIMIT $1 OFFSET $2`
- **Random ordering**: `ORDER BY random()`
- **Cleanup**: `TRUNCATE TABLE ... CASCADE`
- **DDL safety**: `CREATE TABLE IF NOT EXISTS`, `DROP TABLE IF EXISTS`

### mysql

- **UUIDs**: Use `CHAR(36)` with `DEFAULT (UUID())`
- **Strings**: `VARCHAR(n)` - always specify length
- **Timestamps**: `DEFAULT CURRENT_TIMESTAMP`
- **Row generation in seed**: Use a recursive CTE:
  ```sql
  WITH RECURSIVE seq AS (
    SELECT 1 AS s UNION ALL SELECT s + 1 FROM seq WHERE s < $1
  ) SELECT * FROM seq
  ```
- **Batch expansion (JSON_TABLE)**: Use `JSON_TABLE` to convert batch args into rows. `__sep__` emits the driver-aware separator:
  ```sql
  SELECT j.val FROM JSON_TABLE(
    CONCAT('["', REPLACE('$1', __sep__, '","'), '"]'),
    '$[*]' COLUMNS(val VARCHAR(255) PATH '$')
  ) j
  ```
- **Batch expansion (__values__)**: Use `__values__` for simpler multi-row VALUES:
  ```yaml
  query: |-
    INSERT INTO t (name, email) __values__
  ```
- **Upsert**: `ON DUPLICATE KEY UPDATE col = VALUES(col)`
- **Categorical selection**: Use `ELT(index, 'val1', 'val2', ...)` instead of array indexing
- **Random ordering**: `ORDER BY RAND()`
- **Cleanup**: `DELETE FROM table` (preferred over TRUNCATE for FK-safe cleanup)

### mssql (SQL Server)

- **UUIDs**: `UNIQUEIDENTIFIER` with `DEFAULT NEWID()`
- **Strings**: `NVARCHAR(n)` for Unicode support, `NVARCHAR(MAX)` for unlimited
- **Timestamps**: `DATETIME2` with `DEFAULT GETDATE()`
- **Row generation in seed**: Recursive CTE with `OPTION (MAXRECURSION 0)`:
  ```sql
  WITH seq AS (
    SELECT 1 AS s UNION ALL SELECT s + 1 FROM seq WHERE s < $1
  ) SELECT * FROM seq OPTION (MAXRECURSION 0)
  ```
- **Batch expansion (OPENJSON)**: Use `batch_format: json` and `OPENJSON`:
  ```sql
  SELECT value FROM OPENJSON('$1')
  ```
- **Batch expansion (__values__)**: Use `__values__` for simpler multi-row VALUES (max 1000 rows per INSERT):
  ```yaml
  query: |-
    INSERT INTO t (name, email) __values__
  ```
- **Upsert**: Use `MERGE INTO ... USING ... ON ... WHEN MATCHED THEN UPDATE ... WHEN NOT MATCHED THEN INSERT ...`
- **DDL safety**: Wrap in existence check:
  ```sql
  IF OBJECT_ID('table_name', 'U') IS NULL CREATE TABLE table_name (...)
  ```
- **Pagination**: `OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY`
- **Random ordering**: `ORDER BY NEWID()`
- **Cleanup**: `DELETE FROM table` (preferred)

### oracle

- **Identifiers**: `NUMBER GENERATED ALWAYS AS IDENTITY` for auto-increment, or explicit `NUMBER` type (UUID is uncommon)
- **Strings**: `VARCHAR2(n)` - Oracle-specific type
- **Timestamps**: `DEFAULT SYSTIMESTAMP`
- **Row generation in seed**: Use `CONNECT BY`:
  ```sql
  SELECT LEVEL FROM DUAL CONNECT BY LEVEL <= $1
  ```
- **Batch expansion (XMLTABLE)**: Use `XMLTABLE` to expand batch args. `__sep__` emits the driver-aware separator:
  ```sql
  SELECT column_value FROM XMLTABLE(('"' || REPLACE('$1', __sep__, '","') || '"'))
  ```
- **Batch expansion (__values__)**: Use `__values__(table(col1, col2))` for Oracle `INSERT ALL`:
  ```yaml
  query: INSERT ALL __values__(product(name, price))
  ```
  Generates: `INTO product (name, price) VALUES (...)\nINTO product ... \nSELECT 1 FROM DUAL`
- **Upsert**: Use `MERGE INTO ... USING (SELECT :1 AS col FROM DUAL) src ON ... WHEN MATCHED THEN UPDATE ... WHEN NOT MATCHED THEN INSERT ...`
- **DDL safety**: Wrap in PL/SQL with exception handling:
  ```sql
  BEGIN
    EXECUTE IMMEDIATE 'CREATE TABLE ...';
  EXCEPTION WHEN OTHERS THEN
    IF SQLCODE != -955 THEN RAISE; END IF;
  END;
  ```
- **Drop safety**: `DROP TABLE ... CASCADE CONSTRAINTS PURGE`
- **Categorical selection**: Use `DECODE(index, 1, 'val1', 2, 'val2', ...)` instead of array indexing
- **Random functions**: `DBMS_RANDOM.VALUE()` for floats, `DBMS_RANDOM.STRING()` for strings
- **Pagination**: `FETCH FIRST :1 ROWS ONLY`
- **Random ordering**: `ORDER BY DBMS_RANDOM.VALUE`

### dsql (Aurora DSQL)

- Follows the same patterns as `pgx` (uses PostgreSQL wire protocol)
- Note: Some CockroachDB-specific SQL (e.g., `STRING` type) may not be available; prefer standard PostgreSQL types

### mongodb

MongoDB uses BSON/JSON command syntax instead of SQL. Queries are JSON objects specifying the command and its parameters.

- **Collections (not tables)**: Use `{"create": "name"}` to create, `{"drop": "name"}` to drop
- **Inserts**: `{"insert": "collection", "documents": [{"_id": $1, "field": $2}]}`
- **Reads**: `{"find": "collection", "filter": {}}` or `{"find": "collection", "filter": {"field": $1}}`
- **Deletes**: `{"delete": "collection", "deletes": [{"q": {}, "limit": 0}]}`
- **Updates**: `{"update": "collection", "updates": [{"q": {"_id": $1}, "u": {"$set": {"field": $2}}}]}`
- **Placeholders**: `$1`, `$2`, etc. are inlined directly into the JSON command text
- **No DDL types**: MongoDB is schemaless; `up` creates collections, `down` drops them
- **Batch inserts**: Use `exec_batch` with `count`/`size`; each batch inserts one document per execution
- **Reference data**: Use `{"find": "collection", "filter": {}}` in `seed` or `init` with `type: query` to populate datasets
- **Transactions**: edg supports `transaction:` blocks for MongoDB using multi-document sessions. Commands run within a session context and are committed or rolled back atomically. Use the same `transaction:` / `locals` / `rollback_if` syntax as SQL drivers

Example:
```yaml
up:
  - name: create_users
    type: exec
    query: |-
      {"create": "users"}

seed:
  - name: insert_users
    type: exec_batch
    count: 1000
    args:
      - gen('uuid')
      - gen('email')
    query: |-
      {"insert": "users", "documents": [{"_id": $1, "email": $2}]}

  - name: fetch_users
    query: |-
      {"find": "users", "filter": {}}

init:
  - name: load_users
    query: |-
      {"find": "users", "filter": {}}

run:
  - name: get_user
    args:
      - ref_rand('load_users')._id
    query: |-
      {"find": "users", "filter": {"_id": $1}}

deseed:
  - name: delete_users
    type: exec
    query: |-
      {"delete": "users", "deletes": [{"q": {}, "limit": 0}]}

down:
  - name: drop_users
    type: exec
    query: |-
      {"drop": "users"}
```

### cassandra

Cassandra uses CQL (Cassandra Query Language). Tables must live inside a keyspace.

- **Keyspaces**: `CREATE KEYSPACE IF NOT EXISTS ks WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}`
- **Tables**: `CREATE TABLE IF NOT EXISTS ks.table (id UUID PRIMARY KEY, ...)`
- **Column types**: `UUID`, `TEXT`, `INT`, `DOUBLE`, `TIMESTAMP`, `BOOLEAN`, `BLOB`
- **No `DEFAULT` values**: Generate all values in args (e.g., `gen('uuid')` for UUIDs)
- **Inserts**: Standard `INSERT INTO ks.table (col1, col2) VALUES ($1, $2)`
- **Reads**: `SELECT col1, col2 FROM ks.table` or `SELECT ... WHERE partition_key = $1`
- **Batch inserts**: Use `exec_batch` with `count`/`size`; edg uses Cassandra's unlogged batch internally
- **Cleanup**: `TRUNCATE ks.table` for deseed
- **Teardown**: `DROP TABLE IF EXISTS ks.table` then `DROP KEYSPACE IF EXISTS ks`
- **Placeholders**: Use `$1`, `$2`, etc.; edg converts to `?` automatically
- **Transactions**: edg supports `transaction:` blocks for Cassandra using logged batches. Reads execute immediately; writes are buffered and committed atomically. Use the same `transaction:` / `locals` / `rollback_if` syntax as SQL drivers

Example:
```yaml
up:
  - name: create_keyspace
    type: exec
    query: |-
      CREATE KEYSPACE IF NOT EXISTS edg
      WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}

  - name: create_users
    type: exec
    query: |-
      CREATE TABLE IF NOT EXISTS edg.users (
        id UUID PRIMARY KEY,
        email TEXT
      )

seed:
  - name: insert_users
    type: exec_batch
    count: 1000
    args:
      - gen('uuid')
      - gen('email')
    query: |-
      INSERT INTO edg.users (id, email) VALUES ($1, $2)

  - name: fetch_users
    query: |-
      SELECT id, email FROM edg.users

init:
  - name: load_users
    query: |-
      SELECT id FROM edg.users

run:
  - name: get_user
    args:
      - ref_rand('load_users').id
    query: |-
      SELECT id, email FROM edg.users WHERE id = $1

deseed:
  - name: truncate_users
    type: exec
    query: TRUNCATE edg.users

down:
  - name: drop_users
    type: exec
    query: DROP TABLE IF EXISTS edg.users

  - name: drop_keyspace
    type: exec
    query: DROP KEYSPACE IF EXISTS edg
```

## Generating from an existing database

If the user already has a database with tables, suggest `edg init` to generate a starting config:

```sh
# PostgreSQL / CockroachDB
edg init --driver pgx --url "postgres://..." --schema public > workload.yaml

# MySQL (use --database for the database name)
edg init --driver mysql --url "root:pass@tcp(localhost:3306)/dbname?parseTime=true" --database dbname > workload.yaml

# MSSQL
edg init --driver mssql --url "sqlserver://..." --schema dbo > workload.yaml

# Oracle
edg init --driver oracle --url "oracle://..." --schema SYSTEM > workload.yaml

# Aurora DSQL
edg init --driver dsql --url "clusterid.dsql.us-east-1.on.aws" --schema public > workload.yaml
```

The `--schema` and `--database` flags are interchangeable. Use `--schema` for drivers where the value is a schema name (pgx, dsql, mssql, oracle) and `--database` for drivers where it's a database name (mysql).

The generated config is a starting point, seed expressions will match column types and constraints but won't produce realistic data. The user should refine the config after generation.

## Validation

After generating the config, remind the user to validate it:

```sh
edg validate config --config <path>
```

## Staging (file output without a database)

The `edg stage` command generates data to files instead of executing against a database. No `--url` or database connection is required. This is useful for previewing generated data, loading into external tools, or generating migration scripts.

```sh
edg stage --config <path> --format <format> --output-dir <dir>
```

| Flag | Short | Default | Description |
|---|---|---|---|
| `--format` | `-f` | `sql` | Output format: `sql`, `json`, `csv`, `parquet`, or `stdout` |
| `--output-dir` | `-o` | `.` | Directory for output files (created if it doesn't exist) |

### Output formats

| Format | File naming | Description |
|---|---|---|
| `sql` | `{section}.sql` | Executable SQL statements (DDL + one resolved statement per generated row) |
| `json` | `{section}.json` | Objects keyed by query name (data-generating queries only) |
| `csv` | `{section}_{query}.csv` | CSV with headers per data-generating query |
| `parquet` | `{section}_{query}.parquet` | Apache Parquet per data-generating query (all columns as optional byte arrays) |
| `stdout` | *(none)* | Streams resolved SQL to stdout (no files written, log output suppressed) |

### Key behaviours

- Batch queries (`exec_batch`) are expanded into individual rows. The batch CSV-joining logic is bypassed
- Referential integrity is preserved: generated data from earlier queries is stored in memory for `ref_rand`, `ref_each`, `seq_rand`, etc.
- The `--driver` flag still controls SQL value formatting (quote style, hex literals)
- The `--rng-seed` flag produces deterministic, reproducible output
- Column names come from: named args > INSERT column list > fallback `col_1`, `col_2`, etc.
- DDL/DML without args (CREATE, DROP, DELETE, TRUNCATE) is included in SQL output but skipped in JSON/CSV/Parquet

After generating the config, suggest staging to preview data:

```sh
edg stage --config workload.yaml --format csv -o ./preview
```

## Sync Configs (Cross-Database Consistency)

When the user wants to test dual-write consistency, CDC pipelines, or cross-database replication, generate **paired configs** - one per database driver - for use with `edg sync run`.

### Requirements for sync-compatible configs

- **Explicit IDs**: Use `INT PRIMARY KEY` with `seq(1, 1)` instead of auto-generated IDs (`SERIAL`, `AUTO_INCREMENT`, `UUID`). Both databases must produce identical IDs.
- **Matching schemas**: Same table names, column names, and logical types. SQL dialect differs (e.g., `DEFAULT NOW()` vs `DEFAULT CURRENT_TIMESTAMP`).
- **Matching seed args**: Both configs must use identical `args` expressions (same `gen()`, `ref_rand()`, `uniform()`, `set_rand()`, etc.). The `--rng-seed` flag + PRNG re-seeding ensures identical values.
- **Use `exec_batch`**: Sync configs should use `type: exec_batch` with `count` and `size` for efficient bulk inserts.
- **Driver-specific batch SQL**: Use `unnest(string_to_array('$N', __sep__))` for pgx and `JSON_TABLE(CONCAT(...))` for MySQL, or use `__values__` for a cross-driver multi-row VALUES clause (works with pgx, mysql, mssql, spanner, dsql, and oracle via `__values__(table(cols))`).
- **No `run` section**: Sync configs only need `up`, `seed`, `deseed`, and `down`. The benchmark workload is separate.
- **Shared globals**: Both configs should use the same `globals` values (row counts, batch sizes).

### Example

For a CockroachDB + MySQL sync pair, generate two files:

**crdb.yaml:**
```yaml
globals:
  user_count: 1000
  batch_size: 100

up:
  - name: create_users
    query: |-
      CREATE TABLE IF NOT EXISTS users (
        id INT PRIMARY KEY,
        email VARCHAR(255) NOT NULL,
        name VARCHAR(255) NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT NOW()
      )

seed:
  - name: seed_users
    type: exec_batch
    count: user_count
    size: batch_size
    args:
      - seq(1, 1)
      - gen('email')
      - gen('name')
    query: |-
      INSERT INTO users (id, email, name)
      __values__
```

**mysql.yaml:**
```yaml
globals:
  user_count: 1000
  batch_size: 100

up:
  - name: create_users
    query: |-
      CREATE TABLE IF NOT EXISTS users (
        id INT PRIMARY KEY,
        email VARCHAR(255) NOT NULL,
        name VARCHAR(255) NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
      )

seed:
  - name: seed_users
    type: exec_batch
    count: user_count
    size: batch_size
    args:
      - seq(1, 1)
      - gen('email')
      - gen('name')
    query: |-
      INSERT INTO users (id, email, name)
      __values__
```

### Usage

After generating paired configs, show the user how to run them:

```sh
edg sync run \
  --source-driver pgx --source-url "postgres://..." --source-config crdb.yaml \
  --target-driver mysql --target-url "root:pass@tcp(...)/?parseTime=true" --target-config mysql.yaml \
  --rng-seed 42

edg sync verify \
  --source-driver pgx --source-url "postgres://..." \
  --target-driver mysql --target-url "root:pass@tcp(...)/?parseTime=true" \
  --tables users --order-by id --ignore-columns created_at
```

For CDC mode (source only, replication handles target), omit `--target-config` from `sync run`.
