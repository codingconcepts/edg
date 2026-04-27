---
title: Queries
weight: 3
---

# Queries

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
    type: exec_batch
    count: 1000
    size: 100
    args:
      - gen('email')
    query: |-
      INSERT INTO users (email)
      __values__
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
| `batch_format` | Controls how batch values are serialized when using driver-specific batch expansion. Default uses the ASCII unit separator (char 31, `\x1f`). Set to `json` for JSON arrays (used with MSSQL `OPENJSON`). Not needed when using `__values__`. |

Each arg expression is evaluated once per row. For example, with `count: 1000` and `size: 100`, you get 10 batches of 100 rows each.

#### Multi-Row VALUES (`__values__`) - Recommended

The `__values__` token generates a standard multi-row `VALUES` clause. This is the recommended approach for batch inserts. It produces a single INSERT statement per batch and works the same way across all SQL drivers that support multi-row VALUES (pgx, mysql, mssql, spanner, dsql).

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
      __values__
```

Which resolves to the following query automatically by edg:

```sql
INSERT INTO contact (name, email)
VALUES ('Alice', 'alice@example.com'), ('Bob', 'bob@example.com'), ...
```

Each batch of `size` rows produces exactly one INSERT statement. With `count: 1000` and `size: 100`, edg sends 10 INSERT statements, each containing 100 value tuples.

`__values__` also works with upsert and update patterns:

**Upsert (PostgreSQL / CockroachDB):**
```yaml
query: |-
  INSERT INTO product (name, price)
  __values__
  ON CONFLICT (name) DO UPDATE SET price = EXCLUDED.price
```

**Upsert (MySQL):**
```yaml
query: |-
  INSERT INTO product (name, price)
  __values__
  ON DUPLICATE KEY UPDATE price = VALUES(price)
```

**Upsert (MSSQL):**
```yaml
query: |-
  MERGE INTO product AS target
  USING (
    __values__
  ) AS source(name, price)
  ON target.name = source.name
  WHEN MATCHED THEN UPDATE SET price = source.price
  WHEN NOT MATCHED THEN INSERT (name, price) VALUES (source.name, source.price);
```

**Update via CTE (PostgreSQL / CockroachDB):**
```yaml
query: |-
  UPDATE product
  SET price = v.price
  FROM (
    __values__
  ) AS v(id, price)
  WHERE product.id = v.id::UUID
```

> [!NOTE]
> `__values__` works with pgx, mysql, mssql, spanner, and dsql. SQL Server limits multi-row VALUES to 1000 rows per INSERT; set `size` accordingly. For MongoDB and Cassandra, use their native batch patterns described below.

#### Oracle (`INSERT ALL`)

Oracle does not support multi-row `VALUES`. Instead, use the parameterized form `__values__(table(col1, col2))` to generate an `INSERT ALL ... SELECT 1 FROM DUAL` statement:

```yaml
seed:
  - name: insert_products
    type: exec_batch
    count: 100
    size: 10
    args:
      - gen('productname')
      - uniform_f(1.00, 100.00, 2)
    query: |-
      INSERT ALL __values__(product(name, price))
```

This produces:

```sql
INSERT ALL
INTO product (name, price) VALUES ('Widget', 9.99)
INTO product (name, price) VALUES ('Gadget', 24.50)
...
SELECT 1 FROM DUAL
```

The table name and column list are extracted from the token parameters. Non-Oracle drivers ignore the parameters and use standard multi-row `VALUES` syntax.

See [`_examples/multi_row_dml/`](https://github.com/codingconcepts/edg/tree/main/_examples/multi_row_dml) for complete working examples across multiple databases.

#### Driver-Specific Batch Expansion

As an alternative to `__values__`, edg supports driver-specific batch expansion patterns using `__sep__` (the ASCII unit separator). These are more verbose but give you full control over the SQL shape (useful for non-INSERT contexts or advanced patterns).

##### Postgres / CockroachDB

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

Which resolves to:

```sql
INSERT INTO users (email)
SELECT unnest(string_to_array(
  'a@x.com\x1fb@y.com\x1f...\x1fz@x.com', chr(31)
))
```

##### MySQL

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

Which resolves to:

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

##### Oracle

Oracle batch values are unpacked using `xmltable` with `tokenize`. Multiple columns are correlated by joining on `rowid`:

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

Which resolves to:

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

##### MSSQL

When `batch_format` is set to `json`, each arg position is serialized as a properly escaped JSON array (e.g. `["val1","val2","val3"]`). Multiple `OPENJSON` calls are correlated using `[key]`, which is the zero-based array index. NULL values appear as JSON `null` and can be handled with `NULLIF(j.value, 'null')` if the target column is nullable:

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

Which resolves to:

```sql
INSERT INTO contact (name, email)
SELECT j1.value, j2.value
FROM OPENJSON('["Alice","Bob",...,"Zara"]') j1
JOIN OPENJSON('["a@x.com","b@y.com",...,"z@x.com"]') j2
  ON j1.[key] = j2.[key]
```

##### Spanner (GoogleSQL)

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

Which resolves to:

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
> **Spanner limitations:**
> - No `TRUNCATE` - use `DELETE FROM table WHERE TRUE` for deseed operations.
> - `INSERT OR UPDATE` for upserts instead of `ON CONFLICT`.
> - **Drop indexes before tables** - Spanner requires all indexes on a table to be dropped before the table itself. Add `DROP INDEX IF EXISTS idx_name` entries in the `down` section before the corresponding `DROP TABLE`.
> - **No `RAND()`** - use `MOD(ABS(FARM_FINGERPRINT(GENERATE_UUID())), N)` for random integers in range `[0, N)`.
> - **No `CHR()`** - use `CODE_POINTS_TO_STRING([code_point])` instead.
> - **No `UNNEST(...) AS v(col1, col2, ...)`** - Spanner does not support column aliasing on UNNEST. Use `__values__` for batch inserts, or `UNNEST(...) AS val WITH OFFSET` for single-column expansion.
> - **Strict typing with bind params** - `gen('number:...')` returns float64, which Spanner rejects for INT64 columns when using bind params (`@pN`). Wrap in `int()`: `int(gen('number:1,100'))`.
> - **String bind params** - if a value needs to be STRING for Spanner but the expression returns a number, use inlined `$1`/`'$1'` placeholders instead of `@pN` bind params.

#### MongoDB

MongoDB queries use BSON/JSON command syntax instead of SQL. Placeholders (`$1`, `$2`, etc.) are inlined into the JSON command text. Batch operations insert multiple documents in a single command:

```yaml
seed:
  - name: insert_customers
    type: exec_batch
    count: 1000
    args:
      - gen('uuid')
      - gen('email')
    query: |-
      {"insert": "customer", "documents": [{"_id": $1, "email": $2}]}
```

Schema operations use JSON commands:

```yaml
up:
  - name: create_customer
    type: exec
    query: |-
      {"create": "customer"}

down:
  - name: drop_customer
    type: exec
    query: |-
      {"drop": "customer"}
```

Read queries use the `find` command:

```yaml
init:
  - name: fetch_customers
    query: |-
      {"find": "customer", "filter": {}}
```

#### Cassandra

Cassandra uses CQL (Cassandra Query Language). Placeholders (`$1`, `$2`, etc.) are converted to `?` automatically. Batch operations use Cassandra's unlogged batch internally:

```yaml
seed:
  - name: insert_customers
    type: exec_batch
    count: 1000
    args:
      - gen('uuid')
      - gen('email')
    query: |-
      INSERT INTO edg.customer (id, email) VALUES ($1, $2)
```

Cassandra requires a keyspace to be created before tables:

```yaml
up:
  - name: create_keyspace
    type: exec
    query: |-
      CREATE KEYSPACE IF NOT EXISTS edg
      WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}

  - name: create_customer
    type: exec
    query: |-
      CREATE TABLE IF NOT EXISTS edg.customer (
        id UUID PRIMARY KEY,
        email TEXT
      )
```

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

#### Cassandra transactions

Cassandra does not support ACID transactions. When you use a `transaction` block with the Cassandra driver, edg uses a **logged batch** internally. This has important implications:

- **Writes are atomic**: All `exec` queries in the transaction are accumulated into a single batch and sent to the database on commit. Either all writes succeed or none do.
- **Reads bypass the batch**: Any `query` type inside the transaction executes immediately against the session, outside the batch. Reads do not see uncommitted writes from the same transaction.
- **Rollback is a no-op**: If a `rollback_if` condition triggers or a query fails, the batch is simply discarded. There is no undo of already-executed reads.

This makes Cassandra transactions suitable for grouping related writes that must succeed together, but not for read-then-write patterns that require isolation. If your workload needs to read a value and then write based on it within the same transaction, use lightweight transactions (LWT) with `IF` conditions in individual queries instead.

#### MongoDB transactions

MongoDB transactions use sessions and provide full ACID guarantees. Both reads and writes participate in the transaction and see each other's results, matching the behaviour of SQL drivers.

> [!NOTE]
> MongoDB transactions require a **replica set** or **sharded cluster**. Standalone `mongod` instances do not support multi-document transactions. If you are testing locally, use `mongod --replSet rs0` or a Docker Compose setup with replica set initialisation.

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

Inlining means edg performs a text replacement on the SQL string _before_ sending it to the database. Every `$N` in the query (or `__values__` token) is replaced with the literal arg values. For example, with `__values__`:

```yaml
type: exec_batch
count: 3
args:
  - gen('email')
query: |-
  INSERT INTO users (email)
  __values__
```

The SQL sent to the database becomes:

```sql
INSERT INTO users (email)
VALUES ('alice@x.com'), ('bob@y.com'), ('carol@z.com')
```

With driver-specific batch expansion, `$N` placeholders are inlined similarly:

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

The database never sees `$1` or `__values__`, it receives a fully formed query with the values baked in. This is used for:

- **`__values__`** token replacement (recommended).
- **`query_batch` / `exec_batch`** types with `$N` placeholders (always inlined).
- **Batch-expanded queries** using `gen_batch`, `batch`, or `ref_each` (in any section).

Inlining lets you use `$N` as a universal placeholder syntax across all drivers (pgx, MySQL, Oracle, SQL Server) without worrying about driver-specific bind param formats. It also avoids a pgx-stdlib issue where numeric values are sent as DECIMAL, which CockroachDB can't mix with INT in arithmetic.

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
| `mongodb` | `$1`, `$2`, `$3` (inlined into JSON commands) |
| `cassandra` | `?` (positional) |

Since `run` queries always use bind params, their SQL must use the correct format for the target driver.

### Column Name Normalisation

When a `query` or `query_batch` result is stored, all column names are lowercased before being added to the environment. This means a SQL column `W_ID` becomes accessible as `ref_rand('fetch_warehouses').w_id`, not `.W_ID`.
