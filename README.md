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
| Google Cloud Spanner | `spanner` | `projects/PROJECT/instances/INSTANCE/databases/DATABASE` |
| MongoDB | `mongodb` | `mongodb://localhost:27017/db` |
| Cassandra | `cassandra` | `cassandra://localhost:9042/keyspace` |

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
docker compose -f cmd/harness/compose/compose_crdb.yml up -d
make integration_test_crdb
docker compose -f cmd/harness/compose/compose_crdb.yml down

# MySQL
docker compose -f cmd/harness/compose/compose_mysql.yml up -d
make integration_test_mysql
docker compose -f cmd/harness/compose/compose_mysql.yml down

# MSSQL
docker compose -f cmd/harness/compose/compose_mssql.yml up -d
make integration_test_mssql
docker compose -f cmd/harness/compose/compose_mssql.yml down

# Oracle
docker compose -f cmd/harness/compose/compose_oracle.yml up -d
make integration_test_oracle
docker compose -f cmd/harness/compose/compose_oracle.yml down
```

**Schema tests** (`pkg/schema`):

```sh
# CockroachDB
docker compose -f cmd/harness/compose/compose_crdb.yml up -d
make integration_test_schema_crdb
docker compose -f cmd/harness/compose/compose_crdb.yml down

# MySQL
docker compose -f cmd/harness/compose/compose_mysql.yml up -d
make integration_test_schema_mysql
docker compose -f cmd/harness/compose/compose_mysql.yml down

# MSSQL
docker compose -f cmd/harness/compose/compose_mssql.yml up -d
make integration_test_schema_mssql
docker compose -f cmd/harness/compose/compose_mssql.yml down

# Oracle
docker compose -f cmd/harness/compose/compose_oracle.yml up -d
make integration_test_schema_oracle
docker compose -f cmd/harness/compose/compose_oracle.yml down
```

## Todos

* Add expectations for all edg workload configs
* Rethink iter() - perhaps its better to use seq with a start pos
* Ensure MongoDB uses object ids
* Ensure all batch queries are inserting at least 10 rows (some cassandra batches are 1 at a time)
* Unique constraint awareness (e.g. uniq(gen('email')) will repeated call gen until a new unique value is found. For uniq(gen('email'), 10), it will try 10 times, then fail)
* MongoDB and Cassandra sync verify support
* Log levels
* Better error output
* Comparison mode (run the same workload against databases or different configurations of the same) and produce side-by-side differences