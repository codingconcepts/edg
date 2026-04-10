<p align="center">
  <img src="docs/static/assets/logo.png" alt="drawing" width="350"/>
</p>

# edg

A database workload runner driven by YAML configuration. Define your schema, seed data, and transactional workloads in a single config file, then run them against any supported database with concurrent workers and real-time throughput reporting.

Query arguments are written as expressions compiled at startup, giving you access to global constants, random data generation, reference lookups, and TPC-C-compliant non-uniform random distributions.

## Supported Databases

| Database | Driver | URL (example) |
|---|---|---|
| CockroachDB / PostgreSQL | `pgx` | `postgres://root@localhost:26257/db?sslmode=disable` |
| Aurora DSQL | `dsql` | `clusterid.dsql.us-east-1.on.aws` |
| Oracle | `oracle` | `oracle://system:password@localhost:1521/db` |
| MySQL | `mysql` | `user:password@tcp(host:port)/db?parseTime=true` |
| SQL Server | `sqlserver` | `sqlserver://user:password@host:port?database=db&encrypt=disable` |

## Quick Start

```sh
go install github.com/codingconcepts/edg@latest
```

```sh
edg all \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 100 \
-d 5m
```

## Documentation

View the docs [here](https://edg.run/docs).

## Todos

* Progress indication
* Comparison mode (run the same workload against databases or different configurations of the same) and produce side-by-side differences
* Global sequencies (`seq_global(name, start, step)`) for all workers to share in a thread safe way
* Prometheus metrics endpoint
* Dry run mode
* Output to file for testing and feeding other tools
  * CSV
  * JSON
  * Parquet
  * SQL
* Unique constraint awareness
