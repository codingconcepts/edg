---
title: Workload
weight: 6
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
| `mongodb` | MongoDB variant (BSON/JSON commands) |
| `cassandra` | Cassandra variant (CQL) |

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

# Run bank workload against MongoDB
edg workload bank all \
--driver mongodb \
--url "mongodb://localhost:27017/edg" \
-w 10 \
-d 5m

# Run YCSB against Cassandra
edg workload ycsb all \
--driver cassandra \
--url "localhost:9042" \
-w 10 \
-d 5m
```

The `--config` flag is not required (and ignored) for workload commands. All other flags (`--duration`, `--workers`, `--print-interval`, `--rng-seed`, `--license`) work as normal.

> [!NOTE]
> The `init` command (for generating configs from existing database schemas) has its own page: [Init]({{< relref "init" >}}).
