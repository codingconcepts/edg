---
name: edg-migrate
description: Convert an edg YAML config from one database driver to another (e.g., pgx to mysql, mssql to oracle, pgx to mongodb, pgx to cassandra).
user-invocable: true
---

# edg Config Migrator

You convert edg YAML workload configurations between database drivers. Given a config written for one driver and a target driver, you produce a working config for the target.

## Input

The user provides:
- A path to an existing edg config (or pastes the YAML)
- The source driver (infer from config if not stated)
- The target driver

## Migration Rules

Apply the following transformations based on the source and target driver. edg handles placeholder conversion automatically (`$1` works for all drivers), so focus on SQL dialect differences.

### Type Mappings

| Concept | pgx | mysql | mssql | oracle | mongodb | cassandra |
|---|---|---|---|---|---|---|
| UUID | `UUID` | `CHAR(36)` | `UNIQUEIDENTIFIER` | `VARCHAR2(36)` | *(string field)* | `UUID` |
| UUID default | `DEFAULT gen_random_uuid()` | `DEFAULT (UUID())` | `DEFAULT NEWID()` | *(generate in args)* | *(generate in args)* | *(generate in args)* |
| String | `STRING` or `VARCHAR(n)` | `VARCHAR(n)` | `NVARCHAR(n)` | `VARCHAR2(n)` | *(string field)* | `TEXT` |
| Unlimited string | `TEXT` | `TEXT` | `NVARCHAR(MAX)` | `CLOB` | *(string field)* | `TEXT` |
| Timestamp | `TIMESTAMP` | `TIMESTAMP` | `DATETIME2` | `TIMESTAMP` | *(ISODate field)* | `TIMESTAMP` |
| Timestamp default | `DEFAULT now()` | `DEFAULT CURRENT_TIMESTAMP` | `DEFAULT GETDATE()` | `DEFAULT SYSTIMESTAMP` | *(generate in args)* | *(generate in args)* |
| Boolean | `BOOL` | `TINYINT(1)` | `BIT` | `NUMBER(1)` | *(boolean field)* | `BOOLEAN` |
| Auto-increment | *(use UUID)* | `AUTO_INCREMENT` | `IDENTITY(1,1)` | `GENERATED ALWAYS AS IDENTITY` | *(not applicable)* | *(not applicable)* |
| Decimal | `DECIMAL(p,s)` | `DECIMAL(p,s)` | `DECIMAL(p,s)` | `NUMBER(p,s)` | *(number field)* | `DECIMAL` |
| Integer | `INT` | `INT` | `INT` | `NUMBER(10)` | *(number field)* | `INT` |
| Big integer | `BIGINT` | `BIGINT` | `BIGINT` | `NUMBER(19)` | *(number field)* | `BIGINT` |

### DDL Safety

| Driver | CREATE pattern | DROP pattern |
|---|---|---|
| pgx | `CREATE TABLE IF NOT EXISTS ...` | `DROP TABLE IF EXISTS ...` |
| mysql | `CREATE TABLE IF NOT EXISTS ...` | `DROP TABLE IF EXISTS ...` |
| mssql | `IF OBJECT_ID('t', 'U') IS NULL CREATE TABLE t (...)` | `IF OBJECT_ID('t', 'U') IS NOT NULL DROP TABLE t` |
| oracle | PL/SQL block with `EXCEPTION WHEN OTHERS THEN IF SQLCODE != -955 THEN RAISE; END IF; END;` | `DROP TABLE t CASCADE CONSTRAINTS PURGE` |
| mongodb | `{"create": "collection"}` | `{"drop": "collection"}` |
| cassandra | `CREATE TABLE IF NOT EXISTS ks.t (...)` | `DROP TABLE IF EXISTS ks.t` |

### Row Generation in Seed Queries

| Driver | Pattern |
|---|---|
| pgx | `generate_series(1, $1)` |
| mysql | `WITH RECURSIVE seq AS (SELECT 1 AS s UNION ALL SELECT s + 1 FROM seq WHERE s < $1) SELECT * FROM seq` |
| mssql | `WITH seq AS (SELECT 1 AS s UNION ALL SELECT s + 1 FROM seq WHERE s < $1) SELECT * FROM seq OPTION (MAXRECURSION 0)` |
| oracle | `SELECT LEVEL FROM DUAL CONNECT BY LEVEL <= $1` |

### Batch Expansion (expanding CSV/JSON args into rows)

| Driver | Pattern |
|---|---|
| pgx | `SELECT unnest(string_to_array('$1', __sep__))` |
| mysql | `SELECT j.val FROM JSON_TABLE(CONCAT('["', REPLACE('$1', __sep__, '","'), '"]'), '$[*]' COLUMNS(val VARCHAR(255) PATH '$')) j` |
| mssql | Use `batch_format: json` and `SELECT value FROM OPENJSON('$1')` |
| oracle | `SELECT column_value FROM XMLTABLE(('"' \|\| REPLACE('$1', __sep__, '","') \|\| '"'))` |
| mongodb | Not applicable; batch uses `exec_batch` with per-document `{"insert": ...}` commands |
| cassandra | Not applicable; batch uses `exec_batch` with CQL `INSERT` statements (unlogged batch internally) |

### Upsert / Merge

| Driver | Pattern |
|---|---|
| pgx | `ON CONFLICT (col) DO UPDATE SET ...` |
| mysql | `ON DUPLICATE KEY UPDATE col = VALUES(col)` |
| mssql | `MERGE INTO t USING (SELECT @p1 AS c1) src ON t.c1 = src.c1 WHEN MATCHED THEN UPDATE SET ... WHEN NOT MATCHED THEN INSERT ...;` |
| oracle | `MERGE INTO t USING (SELECT :1 AS c1 FROM DUAL) src ON (t.c1 = src.c1) WHEN MATCHED THEN UPDATE SET ... WHEN NOT MATCHED THEN INSERT ...` |

### Pagination

| Driver | Pattern |
|---|---|
| pgx | `LIMIT $1 OFFSET $2` |
| mysql | `LIMIT $1 OFFSET $2` |
| mssql | `OFFSET $1 ROWS FETCH NEXT $2 ROWS ONLY` |
| oracle | `OFFSET $1 ROWS FETCH FIRST $2 ROWS ONLY` |

### Random Ordering

| Driver | Pattern |
|---|---|
| pgx | `ORDER BY random()` |
| mysql | `ORDER BY RAND()` |
| mssql | `ORDER BY NEWID()` |
| oracle | `ORDER BY DBMS_RANDOM.VALUE` |

### Categorical Selection (in SQL)

| Driver | Pattern |
|---|---|
| pgx | `(ARRAY['a','b','c'])[index]` |
| mysql | `ELT(index, 'a', 'b', 'c')` |
| mssql | `CASE WHEN ... THEN ... END` |
| oracle | `DECODE(index, 1, 'a', 2, 'b', 3, 'c')` |

### Cleanup

| Driver | Deseed | Drop |
|---|---|---|
| pgx | `TRUNCATE TABLE t CASCADE` | `DROP TABLE IF EXISTS t` |
| mysql | `DELETE FROM t` | `DROP TABLE IF EXISTS t` |
| mssql | `DELETE FROM t` | `IF OBJECT_ID('t', 'U') IS NOT NULL DROP TABLE t` |
| oracle | `TRUNCATE TABLE t` | `DROP TABLE t CASCADE CONSTRAINTS PURGE` |
| mongodb | `{"delete": "t", "deletes": [{"q": {}, "limit": 0}]}` | `{"drop": "t"}` |
| cassandra | `TRUNCATE ks.t` | `DROP TABLE IF EXISTS ks.t` |

### MongoDB-Specific Notes

When migrating SQL configs to MongoDB:
- Replace `CREATE TABLE` with `{"create": "collection"}`
- Replace `INSERT INTO t (cols) VALUES (...)` with `{"insert": "t", "documents": [{"field": $1, ...}]}`
- Replace `SELECT ... FROM t WHERE ...` with `{"find": "t", "filter": {"field": $1}}`
- Replace `UPDATE t SET ... WHERE ...` with `{"update": "t", "updates": [{"q": {"_id": $1}, "u": {"$set": {"field": $2}}}]}`
- Replace `DELETE FROM t` with `{"delete": "t", "deletes": [{"q": {}, "limit": 0}]}`
- Replace `DROP TABLE` with `{"drop": "t"}`
- MongoDB is schemaless - no column types, no constraints, no foreign keys
- All placeholders are inlined into JSON command text

### Cassandra-Specific Notes

When migrating SQL configs to Cassandra:
- Add a `CREATE KEYSPACE` query before any `CREATE TABLE` queries
- Prefix all table names with keyspace: `ks.table`
- Replace `VARCHAR(n)` / `STRING` with `TEXT`
- Replace `DECIMAL(p,s)` with `DECIMAL` or `DOUBLE`
- Replace `BOOL` with `BOOLEAN`
- Remove `DEFAULT` clauses - generate all values in args
- Remove foreign key constraints (`REFERENCES`)
- Remove `CASCADE` from `TRUNCATE`
- Add `DROP KEYSPACE IF EXISTS ks` at end of `down` section
- No transactions - convert transaction blocks to standalone queries

## Process

1. Read the source config
2. Identify all SQL patterns that need driver-specific translation
3. Apply the mappings above
4. Preserve all edg expression args unchanged (they are driver-agnostic)
5. Preserve globals, expressions, reference, run_weights, and other non-SQL sections unchanged
6. If the source uses `batch_format`, adjust for the target driver
7. Remind the user to validate: `edg validate config --driver <target> --config <path>`
8. Suggest staging to preview the migrated output without a database:
   ```sh
   edg stage --config <path> --format sql -o ./preview
   ```
   This generates data to files, letting the user inspect SQL syntax, value formatting, and data distributions for the target driver before connecting to a real database.
