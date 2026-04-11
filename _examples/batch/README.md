# Batch Types

Demonstrates `query_batch` and `exec_batch` query types. A `query_batch` inserts products and captures the returned rows, then an `exec_batch` references those rows to insert reviews against them.

- **`query_batch`** evaluates args per row (controlled by `count` and `size`), collects values into comma-separated strings, and stores the query results for use by `ref_*` functions.
- **`exec_batch`** does the same arg generation but executes without reading results.

> **Note:** MySQL and Oracle do not support `INSERT...RETURNING`, so the MySQL and Oracle configs use `exec_batch` + a follow-up `query` to fetch inserted rows instead of `query_batch`.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
docker exec -it node1 cockroach init --insecure
docker exec -it node1 cockroach sql --insecure
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/batch/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/batch/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/batch/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/batch/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

## MySQL

### Setup

```sh
docker compose -f _examples/compose_mysql.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver mysql \
--config _examples/batch/mysql.yaml \
--url "root:password@tcp(localhost:3306)/batch?parseTime=true"

go run ./cmd/edg seed \
--driver mysql \
--config _examples/batch/mysql.yaml \
--url "root:password@tcp(localhost:3306)/batch?parseTime=true"

go run ./cmd/edg deseed \
--driver mysql \
--config _examples/batch/mysql.yaml \
--url "root:password@tcp(localhost:3306)/batch?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/batch/mysql.yaml \
--url "root:password@tcp(localhost:3306)/batch?parseTime=true"
```

## Oracle

### Setup

```sh
docker compose -f _examples/compose_oracle.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver oracle \
--config _examples/batch/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg seed \
--driver oracle \
--config _examples/batch/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg deseed \
--driver oracle \
--config _examples/batch/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg down \
--driver oracle \
--config _examples/batch/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```

## MSSQL

### Setup

```sh
docker compose -f _examples/compose_mssql.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver mssql \
--config _examples/batch/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=batch&encrypt=disable"

go run ./cmd/edg seed \
--driver mssql \
--config _examples/batch/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=batch&encrypt=disable"

go run ./cmd/edg deseed \
--driver mssql \
--config _examples/batch/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=batch&encrypt=disable"

go run ./cmd/edg down \
--driver mssql \
--config _examples/batch/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=batch&encrypt=disable"
```
