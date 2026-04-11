---
name: edg-config
description: Generate or modify edg YAML workload configurations from a natural language description of the desired schema and workload.
user_invocable: true
---

# edg Config Generator

You are an expert at creating edg (Expression-based Data Generator) workload configurations. When the user describes a database schema and workload, generate a complete, valid edg YAML config file.

## Input

The user will describe:
- The database tables and their columns
- The type of workload (read-heavy, write-heavy, mixed)
- The target database driver (pgx, mysql, mssql, oracle, dsql)
- Any specific data distribution requirements (hot keys, skewed access, etc.)

If the user does not specify a driver, default to `pgx`.

## Output

A complete edg YAML config with all applicable sections:
- `globals` for shared constants (row counts, batch sizes, worker counts)
- `expressions` for reusable computed values (optional, only if needed)
- `reference` for static lookup data (optional, only if needed)
- `up` for schema creation (CREATE TABLE statements)
- `seed` for data population (bulk INSERTs, use `exec_batch` with `count`/`size` for large volumes)
- `init` for fetching reference data needed by `run` queries (use `type: query` to populate named datasets)
- `run` for the transactional workload (the queries that will be benchmarked)
- `run_weights` for weighted query selection (optional, only if multiple run queries)
- `deseed` for data cleanup (TRUNCATE statements)
- `down` for schema teardown (DROP TABLE statements)

## Rules

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

### Reference data
- Use `init` section with `type: query` to fetch data from seeded tables into named datasets
- `ref_rand('dataset').field` for random row access in `run` queries
- `ref_same('dataset').field` when multiple args need the same row
- `ref_perm('dataset').field` for worker-pinned rows (e.g., partition affinity)
- `ref_diff('dataset').field` for unique rows within a single query execution
- `ref_n('dataset', 'field', min, max)` for N unique values as comma-separated string

### Dependent columns
- `arg(index)` to reference a previously evaluated arg by zero-based index
- `cond(predicate, trueVal, falseVal)` for conditional values
- `nullable(expr, probability)` for nullable columns
- `bool()` + `arg()` + `cond()` for mutually exclusive columns

### Batch operations
- For seed operations with large row counts, use `exec_batch` with `count` (total rows) and `size` (rows per batch)
- Use `gen_batch(total, batchSize, pattern)` for generating batched values
- Use `batch(n)` for sequential indices

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
      INSERT INTO users (id, email, name) VALUES ($1, $2, $3)

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
      INSERT INTO orders (id, user_id, total, status) VALUES ($1, $2, $3, $4)

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
- **Batch expansion**: Use `unnest(string_to_array('$1', ','))` to expand CSV args into rows
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
- **Batch expansion**: Use `JSON_TABLE` to convert CSV args into rows:
  ```sql
  SELECT j.val FROM JSON_TABLE(
    CONCAT('["', REPLACE('$1', ',', '","'), '"]'),
    '$[*]' COLUMNS(val VARCHAR(255) PATH '$')
  ) j
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
- **Batch expansion**: Use `batch_format: json` and `OPENJSON`:
  ```sql
  SELECT value FROM OPENJSON('$1')
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
- **Batch expansion**: Use `XMLTABLE` to expand CSV args:
  ```sql
  SELECT column_value FROM XMLTABLE(('"' || REPLACE('$1', ',', '","') || '"'))
  ```
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

## Validation

After generating the config, remind the user to validate it:

```sh
edg validate --driver <driver> --config <path>
```
