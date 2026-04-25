# Blob

A binary data workload demonstrating `blob()` and `bytes()` for inserting and reading raw binary data.

| Function | Type | Databases |
|---|---|---|
| `blob(n)` | `[]byte` | All (pgx, mysql, oracle, mssql) |
| `bytes(n)` | `string` | PostgreSQL/CockroachDB only |

## CockroachDB

### Setup

```sh
docker compose -f cmd/harness/compose/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver pgx \
--config _examples/blob/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 30s
```

## MySQL

### Setup

```sh
docker compose -f cmd/harness/compose/compose_mysql.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver mysql \
--config _examples/blob/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 30s
```

## Oracle

### Setup

```sh
docker compose -f cmd/harness/compose/compose_oracle.yml up -d
```

### Run

```sh
go run ./cmd/edg run \
--driver oracle \
--config _examples/blob/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 30s
```

## MSSQL

### Setup

```sh
docker compose -f cmd/harness/compose/compose_mssql.yml up -d
```

### Run

```sh
go run ./cmd/edg run \
--driver mssql \
--config _examples/blob/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=blob&encrypt=disable" \
-w 10 \
-d 30s
```
