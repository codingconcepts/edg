<p align="center">
  <img src="docs/static/assets/logo.png" alt="drawing" width="350"/>
</p>

# edg

A database workload runner driven by YAML configuration. Define your schema, seed data, and transactional workloads in a single config file, then run them against any supported database with concurrent workers and real-time throughput reporting.

Query arguments are written as expressions compiled at startup time, giving you access to global constants, random data generators, inter-table referencing, and a variety of random distribution generators (normal, exp, Zipfian, etc.)

## Supported Databases

| Database | Driver | URL (example) |
|---|---|---|
| CockroachDB / PostgreSQL | `pgx` | `postgres://root@localhost:26257/db?sslmode=disable` |
| Aurora DSQL | `dsql` | `clusterid.dsql.us-east-1.on.aws` |
| Oracle | `oracle` | `oracle://system:password@localhost:1521/db` |
| MySQL | `mysql` | `user:password@tcp(host:port)/db?parseTime=true` |
| MSSQL | `mssql` | `sqlserver://user:password@host:port?database=db&encrypt=disable` |

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

## Local development

### Running integration tests

Start the database with docker compose, then run the corresponding test target.

**Workload tests** (`pkg/env`):

```sh
# CockroachDB
docker compose -f _examples/compose_crdb.yml up -d
make integration_test_crdb
docker compose -f _examples/compose_crdb.yml down

# MySQL
docker compose -f _examples/compose_mysql.yml up -d
make integration_test_mysql
docker compose -f _examples/compose_mysql.yml down

# MSSQL
docker compose -f _examples/compose_mssql.yml up -d
make integration_test_mssql
docker compose -f _examples/compose_mssql.yml down

# Oracle
docker compose -f _examples/compose_oracle.yml up -d
make integration_test_oracle
docker compose -f _examples/compose_oracle.yml down
```

**Schema tests** (`pkg/schema`):

```sh
# CockroachDB
docker compose -f _examples/compose_crdb.yml up -d
make integration_test_schema_crdb
docker compose -f _examples/compose_crdb.yml down

# MySQL
docker compose -f _examples/compose_mysql.yml up -d
make integration_test_schema_mysql
docker compose -f _examples/compose_mysql.yml down

# MSSQL
docker compose -f _examples/compose_mssql.yml up -d
make integration_test_schema_mssql
docker compose -f _examples/compose_mssql.yml down

# Oracle
docker compose -f _examples/compose_oracle.yml up -d
make integration_test_schema_oracle
docker compose -f _examples/compose_oracle.yml down
```

## Todos

* Pre-baked workloads for quick starts
  * Add existing cockroach workload examples
  * Add all relevant _examples
* Spanner support
* Better error output
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