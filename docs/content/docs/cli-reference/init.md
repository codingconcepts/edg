---
title: Init
weight: 2
---

# Init

The `init` command connects to an existing database, inspects its schema, and prints a complete config to stdout. The output includes `globals`, `up`, `seed`, `deseed`, and `down` sections ready to use with the other commands.

## Flags

| Flag | Required | Description |
|---|---|---|
| <nobr>`--schema`</nobr> | <nobr>Yes (or `--database`)</nobr> | Schema or database name to introspect (e.g. `public`, `defaultdb`, `dbo`, `SYSTEM`) |
| <nobr>`--database`</nobr> | <nobr>Yes (or `--schema`)</nobr> | Alias for `--schema` |
| <nobr>`--driver`</nobr> | Yes | Database driver (`pgx`, `mysql`, `mssql`, `oracle`, `dsql`, `spanner`) |
| <nobr>`--url`</nobr> | Yes | Connection URL for the source database |

```sh
edg init \
--driver pgx \
--url "postgres://root@localhost:26257/defaultdb?sslmode=disable" \
--schema public > workload.yaml
```

## Generated Config

The generated config includes:

- **`up`** `CREATE TABLE` statements derived from the database's own DDL (CockroachDB's `SHOW CREATE TABLE`, MySQL's `SHOW CREATE TABLE`, Oracle's `DBMS_METADATA.GET_DDL`, reconstructed from `sys` catalog views for MSSQL, or reconstructed from `INFORMATION_SCHEMA` for Spanner).
- **`seed`** One `INSERT` per table with an expression for each non-generated column. Columns with auto-increment, identity, or default functions like `gen_random_uuid()` and `now()` are skipped. Expressions are chosen by data type (e.g. `uuid_v4()` for UUID, `uniform(1, 1000)` for INT, `gen('sentence:3')` for VARCHAR). `CHECK BETWEEN` constraints are detected and used to narrow the range.
- **`deseed`** `TRUNCATE` (pgx, oracle) or `DELETE FROM` (mysql, mssql, spanner) in reverse dependency order.
- **`down`** `DROP TABLE` in reverse dependency order.
- Tables are topologically sorted so parent tables are created before children.

> [!WARNING]
> The output is a starting point. You'll typically want to refine the seed expressions to produce more realistic data, add a `run` section, and adjust `globals.rows`.

## Examples by driver

**CockroachDB / PostgreSQL (pgx)**
```sh
edg init \
--driver pgx \
--url "postgres://root@localhost:26257/dbname?sslmode=disable" \
--schema public > workload.yaml
```

**Aurora DSQL**
```sh
edg init \
--driver dsql \
--url "clusterid.dsql.us-east-1.on.aws" \
--schema public > workload.yaml
```

**MySQL**
```sh
edg init \
--driver mysql \
--url "root:password@tcp(localhost:3306)/dbname?parseTime=true" \
--database dbname > workload.yaml
```

**MSSQL**
```sh
edg init \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=master&encrypt=disable" \
--schema dbo > workload.yaml
```

**Oracle**
```sh
edg init \
--driver oracle \
--url "oracle://system:password@localhost:1521/dbname" \
--schema SYSTEM > workload.yaml
```

**Google Cloud Spanner (GoogleSQL)**
```sh
edg init \
--driver spanner \
--url "projects/my-project/instances/my-instance/databases/my-db" \
--license "$EDG_LICENSE" \
--schema "" > workload.yaml
```
