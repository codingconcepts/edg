---
title: Home
type: docs
---

{{< logo >}}

# edg

An Expression-based Data Generator driven by YAML configuration.

## The reason for edg

edg is the result of two wonderful decades spent developing software that talks to databases.

Back in 2014 when I connected to my first database (Oracle), I - rather diligently - added approximately 25 rows of immaculate data to ensure that all of my code paths were covered. Fast forward to the day of the first production release and everything ground to a halt. The problem? I hadn't _really_ tested my database at all.

Sure, I'd added a _variety_ of data to the test database but I hadn't understood what would happen when production amounts of traffic (or production amounts of data) would be thrown at my database or application.

edg was built to help you do just that. Connect to one of the [supported databases](#supported-databases), create database objects, seed realistic amounts of data quickly, and define transactional workloads; all from YAML. Concurrent workers and real-time throughput reporting will take of the rest.

Query arguments are written as expressions compiled at startup, giving you access to global constants, random data generation, reference lookups, and a bunch of [random distributions](/docs/expressions/#numeric-distributions).

## Supported databases

| Database | Driver | URL (example) |
|---|---|---|
| Aurora DSQL | `dsql` | `clusterid.dsql.us-east-1.on.aws` |
| Cassandra | `cassandra` | `localhost:9042` |
| CockroachDB / PostgreSQL | `pgx` | `postgres://root@localhost:26257/db?sslmode=disable` |
| Google Cloud Spanner | `spanner` | `projects/PROJECT/instances/INSTANCE/databases/DATABASE` |
| MongoDB | `mongodb` | `mongodb://localhost:27017/db` |
| MSSQL | `mssql` | `sqlserver://user:password@host:port?database=db&encrypt=disable` |
| MySQL | `mysql` | `user:password@tcp(host:port)/db?parseTime=true` |
| Oracle | `oracle` | `oracle://system:password@localhost:1521/db` |

## Supported features

| Feature | pgx | mysql | mongodb | cassandra | mssql | oracle | dsql | spanner |
|---|---|---|---|---|---|---|---|---|
| up / seed / run / deseed / down | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ |
| sync run / down | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ |
| sync verify | ☑️ | ☑️ | | | ☑️ | ☑️ | ☑️ | ☑️ |
| init (schema introspection) | ☑️ | ☑️ | | | ☑️ | ☑️ | ☑️ | ☑️ |
| Batch operations | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ |
| Expectations | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ |
| Prepared statements | ☑️ | ☑️ | | | ☑️ | ☑️ | ☑️ | ☑️ |
| Stages | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ |
| Transactions | ☑️ | ☑️ | | | ☑️ | ☑️ | ☑️ | ☑️ |
| Workers | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ | ☑️ |

## Quick start

Install with the Go toolchain.

```sh
go install github.com/codingconcepts/edg@latest
```

Run all of the configured config steps.

```sh
edg all \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 100 \
-d 5m
```