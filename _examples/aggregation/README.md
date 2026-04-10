# Aggregation

Demonstrates the aggregation functions by computing summary statistics over an init dataset and recording snapshots into a table.

## Functions

| Function | Signature | Returns | Description |
|---|---|---|---|
| `sum` | `sum(name, field)` | `float64` | Sum of a numeric field across all rows |
| `avg` | `avg(name, field)` | `float64` | Average of a numeric field across all rows |
| `min` | `min(name, field)` | `float64` | Minimum value of a numeric field |
| `max` | `max(name, field)` | `float64` | Maximum value of a numeric field |
| `count` | `count(name)` | `int` | Number of rows in the dataset |
| `distinct` | `distinct(name, field)` | `int` | Number of distinct values for a field |

All aggregation functions operate on named datasets populated by `init` queries.

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
--config _examples/aggregation/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/aggregation/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/aggregation/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 4 \
-d 10s

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/aggregation/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/aggregation/crdb.yaml \
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
--config _examples/aggregation/mysql.yaml \
--url "root:password@tcp(localhost:3306)/aggregation?parseTime=true"

go run ./cmd/edg seed \
--driver mysql \
--config _examples/aggregation/mysql.yaml \
--url "root:password@tcp(localhost:3306)/aggregation?parseTime=true"

go run ./cmd/edg run \
--driver mysql \
--config _examples/aggregation/mysql.yaml \
--url "root:password@tcp(localhost:3306)/aggregation?parseTime=true" \
-w 4 \
-d 10s

go run ./cmd/edg deseed \
--driver mysql \
--config _examples/aggregation/mysql.yaml \
--url "root:password@tcp(localhost:3306)/aggregation?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/aggregation/mysql.yaml \
--url "root:password@tcp(localhost:3306)/aggregation?parseTime=true"
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
--config _examples/aggregation/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg seed \
--driver oracle \
--config _examples/aggregation/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg run \
--driver oracle \
--config _examples/aggregation/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 4 \
-d 10s

go run ./cmd/edg deseed \
--driver oracle \
--config _examples/aggregation/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg down \
--driver oracle \
--config _examples/aggregation/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```

## SQL Server

### Setup

```sh
docker compose -f _examples/compose_sqlserver.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver sqlserver \
--config _examples/aggregation/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=aggregation&encrypt=disable"

go run ./cmd/edg seed \
--driver sqlserver \
--config _examples/aggregation/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=aggregation&encrypt=disable"

go run ./cmd/edg run \
--driver sqlserver \
--config _examples/aggregation/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=aggregation&encrypt=disable" \
-w 4 \
-d 10s

go run ./cmd/edg deseed \
--driver sqlserver \
--config _examples/aggregation/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=aggregation&encrypt=disable"

go run ./cmd/edg down \
--driver sqlserver \
--config _examples/aggregation/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=aggregation&encrypt=disable"
```
