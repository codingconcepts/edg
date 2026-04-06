<p align="center">
  <img src="assets/logo.png" alt="drawing" width="350"/>
</p>

# edg

A database workload runner driven by YAML configuration. Define your schema, seed data, and transactional workloads in a single config file, then run them against any supported database with concurrent workers and real-time throughput reporting.

Query arguments are written as expressions compiled at startup, giving you access to global constants, random data generation, reference lookups, and TPC-C-compliant non-uniform random distributions.

## Supported Databases

| Database | Driver | URL scheme |
|---|---|---|
| CockroachDB / PostgreSQL | `pgx` | `postgres://...` |
| Oracle | `oracle` | `oracle://...` |
| MySQL | `mysql` | `user:password@tcp(host:port)/database?parseTime=true` |

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

View the docs [here](https://codingconcepts.github.io/edg).

## Todos

Column-name lowercasing should be removed. The column should always match the name in the database.

Why do seed and up work differently to run? Can't they all work the same?

Add docs for run_weights and count/size for batch queries in the most appropriate place. For the batch queries, I think a decent amount of explanation and examples are required.

Update the docs for all of the medium impact things.

