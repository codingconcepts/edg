---
title: Workload
weight: 3
---

# Workload

The `workload` command runs a built-in workload without needing a config file. Fifteen benchmarks are embedded in the binary. Each workload supports all six lifecycle commands (`up`, `seed`, `run`, `deseed`, `down`, `all`) and selects the correct config for the `--driver` automatically.

## Built-in workloads

| Workload | Description |
|---|---|
| `bank` | Bank account operations for contention and correctness testing |
| `ch-benchmark` | CH-benCHmark mixed OLTP+OLAP (TPC-C transactions + TPC-H analytical queries) |
| `kv` | Simple key-value read/write benchmark |
| `movr` | Vehicle-sharing application with rides, vehicles, and users |
| `seats` | Airline reservation system with flight booking contention |
| `sysbench-insert` | Sysbench `oltp_insert` pure insert micro-benchmark |
| `sysbench-point-select` | Sysbench `oltp_point_select` pure read micro-benchmark |
| `sysbench-read-write` | Sysbench `oltp_read_write` mixed read-write micro-benchmark |
| `sysbench-update-index` | Sysbench `oltp_update_index` indexed column update micro-benchmark |
| `tatp` | Telecom Application Transaction Processing (80% reads, 20% writes) |
| `tpcc` | Full TPC-C benchmark with all 5 transaction profiles |
| `tpch` | TPC-H decision-support benchmark with analytical queries |
| `ttlbench` | Insert throughput under row-level TTL garbage collection pressure |
| `ttllogger` | Structured log ingestion with TTL-based expiry |
| `ycsb` | Yahoo! Cloud Serving Benchmark with configurable workload profiles |

## Supported drivers

| Driver | Config used |
|---|---|
| `pgx`, `dsql` | CockroachDB/PostgreSQL variant |
| `mysql` | MySQL variant |
| `oracle` | Oracle variant |
| `mssql` | SQL Server variant |
| `spanner` | Google Cloud Spanner variant (GoogleSQL) |

## Examples

```sh
# Create the bank schema
edg workload bank up \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable"

# Seed and run in one step
edg workload bank all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5m

# Run KV benchmark
edg workload kv all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 20 \
-d 5m

# Run MovR ride-sharing workload
edg workload movr all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5m

# Run TPC-C against MySQL
edg workload tpcc all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/tpcc?parseTime=true" \
-w 50 \
-d 10m

# Run TPC-H analytical queries
edg workload tpch all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 4 \
-d 5m

# TTL benchmarks (CockroachDB native TTL)
edg workload ttlbench all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5m

edg workload ttllogger all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5m

# Run YCSB against MSSQL
edg workload ycsb all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ycsb&encrypt=disable" \
-w 20 \
-d 5m

# Run KV benchmark against Spanner
edg workload kv all \
--driver spanner \
--url "projects/my-project/instances/my-instance/databases/my-db" \
--license "$EDG_LICENSE" \
-w 10 \
-d 5m
```

The `--config` flag is not required (and ignored) for workload commands. All other flags (`--duration`, `--workers`, `--print-interval`, `--rng-seed`, `--license`) work as normal.

## Init

The `init` command connects to an existing database, inspects its schema, and prints a complete config to stdout. The output includes `globals`, `up`, `seed`, `deseed`, and `down` sections ready to use with the other commands.

| Flag | Required | Description |
|---|---|---|
| `--schema` | Yes (or `--database`) | Schema or database name to introspect (e.g. `public`, `defaultdb`, `dbo`, `SYSTEM`) |
| `--database` | Yes (or `--schema`) | Alias for `--schema` |
| `--driver` | Yes | Database driver (`pgx`, `mysql`, `mssql`, `oracle`, `dsql`, `spanner`) |
| `--url` | Yes | Connection URL for the source database |

```sh
edg init \
--driver pgx \
--url "postgres://root@localhost:26257/defaultdb?sslmode=disable" \
--schema public > workload.yaml
```

The generated config:

- **`up`** `CREATE TABLE` statements derived from the database's own DDL (CockroachDB's `SHOW CREATE TABLE`, MySQL's `SHOW CREATE TABLE`, Oracle's `DBMS_METADATA.GET_DDL`, reconstructed from `sys` catalog views for MSSQL, or reconstructed from `INFORMATION_SCHEMA` for Spanner).
- **`seed`** One `INSERT` per table with an expression for each non-generated column. Columns with auto-increment, identity, or default functions like `gen_random_uuid()` and `now()` are skipped. Expressions are chosen by data type (e.g. `uuid_v4()` for UUID, `uniform(1, 1000)` for INT, `gen('sentence:3')` for VARCHAR). `CHECK BETWEEN` constraints are detected and used to narrow the range.
- **`deseed`** `TRUNCATE` (pgx, oracle) or `DELETE FROM` (mysql, mssql, spanner) in reverse dependency order.
- **`down`** `DROP TABLE` in reverse dependency order.
- Tables are topologically sorted so parent tables are created before children.

> [!WARNING]
> The output is a starting point. You'll typically want to refine the seed expressions to produce more realistic data, add a `run` section, and adjust `globals.rows`.

### Examples by driver

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
