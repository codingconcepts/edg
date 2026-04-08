<p align="center">
  <img src="assets/logo.png" alt="drawing" width="350"/>
</p>

# edg

A database workload runner driven by YAML configuration. Define your schema, seed data, and transactional workloads in a single config file, then run them against any supported database with concurrent workers and real-time throughput reporting.

Query arguments are written as expressions compiled at startup, giving you access to global constants, random data generation, reference lookups, and TPC-C-compliant non-uniform random distributions.

## Supported Databases

| Database | Driver | URL (example) |
|---|---|---|
| CockroachDB / PostgreSQL | `pgx` | `postgres://root@localhost:26257/db?sslmode=disable` |
| Oracle | `oracle` | `oracle://system:password@localhost:1521/db` |
| MySQL | `mysql` | `user:password@tcp(host:port)/db?parseTime=true` |

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

* Additional driver support
  * Support DSQL with custom driver
  * SQLite
  * SQL Server
* Ability to separate config parts into separate files
* Config includes / composition (!include shared/references.yaml) to reuse reference data and expressions across workload files
* Progress indication
* Ability to provide expectations (e.g. error rate < 1%) which will be good for CI/CD usage
* Comparison mode (run the same workload against databases or different configurations of the same) and produce side-by-side differences
* Ramp-up for workers
* Staged workloads (e.g. start with 10 qps for 1 hour...)
* Global sequencies (`seq_global(name, start, step)`) for all workers to share in a thread safe way
* Prometheus metrics endpoint
* Dry run mode
* Output to file for testing and feeding other tools
  * CSV
  * JSON
  * Parquet
  * SQL
* Unique constraint awareness
* Weighted NULL injection for nullable columns - nullable(expr, probability)
