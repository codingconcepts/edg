---
title: Home
type: docs
---

{{< logo >}}

# edg

An Expression-based Data Generator driven by YAML configuration. Define your schema, seed data, and transactional workloads in a single config file, then run them against any supported database with concurrent workers and real-time throughput reporting.

Query arguments are written as expressions compiled at startup, giving you access to global constants, random data generation, reference lookups, and TPC-C-compliant non-uniform random distributions.

## Supported Databases

| Database | Driver | URL (example) |
|---|---|---|
| CockroachDB / PostgreSQL | `pgx` | `postgres://root@localhost:26257/db?sslmode=disable` |
| Aurora DSQL | `dsql` | `clusterid.dsql.us-east-1.on.aws` |
| Oracle | `oracle` | `oracle://system:password@localhost:1521/db` |
| MySQL | `mysql` | `user:password@tcp(host:port)/db?parseTime=true` |
| MSSQL | `mssql` | `sqlserver://user:password@host:port?database=db&encrypt=disable` |

## Quick Start

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