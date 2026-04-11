---
name: edg-migrate
description: Convert an edg YAML config from one database driver to another (e.g., pgx to mysql, mssql to oracle).
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

| Concept | pgx | mysql | mssql | oracle |
|---|---|---|---|---|
| UUID | `UUID` | `CHAR(36)` | `UNIQUEIDENTIFIER` | `VARCHAR2(36)` |
| UUID default | `DEFAULT gen_random_uuid()` | `DEFAULT (UUID())` | `DEFAULT NEWID()` | *(generate in args)* |
| String | `STRING` or `VARCHAR(n)` | `VARCHAR(n)` | `NVARCHAR(n)` | `VARCHAR2(n)` |
| Unlimited string | `TEXT` | `TEXT` | `NVARCHAR(MAX)` | `CLOB` |
| Timestamp | `TIMESTAMP` | `TIMESTAMP` | `DATETIME2` | `TIMESTAMP` |
| Timestamp default | `DEFAULT now()` | `DEFAULT CURRENT_TIMESTAMP` | `DEFAULT GETDATE()` | `DEFAULT SYSTIMESTAMP` |
| Boolean | `BOOL` | `TINYINT(1)` | `BIT` | `NUMBER(1)` |
| Auto-increment | *(use UUID)* | `AUTO_INCREMENT` | `IDENTITY(1,1)` | `GENERATED ALWAYS AS IDENTITY` |
| Decimal | `DECIMAL(p,s)` | `DECIMAL(p,s)` | `DECIMAL(p,s)` | `NUMBER(p,s)` |
| Integer | `INT` | `INT` | `INT` | `NUMBER(10)` |
| Big integer | `BIGINT` | `BIGINT` | `BIGINT` | `NUMBER(19)` |

### DDL Safety

| Driver | CREATE pattern | DROP pattern |
|---|---|---|
| pgx | `CREATE TABLE IF NOT EXISTS ...` | `DROP TABLE IF EXISTS ...` |
| mysql | `CREATE TABLE IF NOT EXISTS ...` | `DROP TABLE IF EXISTS ...` |
| mssql | `IF OBJECT_ID('t', 'U') IS NULL CREATE TABLE t (...)` | `IF OBJECT_ID('t', 'U') IS NOT NULL DROP TABLE t` |
| oracle | PL/SQL block with `EXCEPTION WHEN OTHERS THEN IF SQLCODE != -955 THEN RAISE; END IF; END;` | `DROP TABLE t CASCADE CONSTRAINTS PURGE` |

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
| pgx | `SELECT unnest(string_to_array('$1', ','))` |
| mysql | `SELECT j.val FROM JSON_TABLE(CONCAT('["', REPLACE('$1', ',', '","'), '"]'), '$[*]' COLUMNS(val VARCHAR(255) PATH '$')) j` |
| mssql | Use `batch_format: json` and `SELECT value FROM OPENJSON('$1')` |
| oracle | `SELECT column_value FROM XMLTABLE(('"' \|\| REPLACE('$1', ',', '","') \|\| '"'))` |

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

## Process

1. Read the source config
2. Identify all SQL patterns that need driver-specific translation
3. Apply the mappings above
4. Preserve all edg expression args unchanged (they are driver-agnostic)
5. Preserve globals, expressions, reference, run_weights, and other non-SQL sections unchanged
6. If the source uses `batch_format`, adjust for the target driver
7. Remind the user to validate: `edg validate --driver <target> --config <path>`
