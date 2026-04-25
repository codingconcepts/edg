# Pipeline

A minimal example demonstrating chained run queries where each step's results feed the next. Three tables (`a`, `b`, `c`) with foreign key relationships show how `ref_rand` and `ref_same` pass data between sequential run queries.

## CockroachDB

### Setup

```sh
docker compose -f cmd/harness/compose/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/pipeline/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/pipeline/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/pipeline/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 1m

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/pipeline/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/pipeline/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

## MySQL

### Setup

```sh
docker compose -f cmd/harness/compose/compose_mysql.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver mysql \
--config _examples/pipeline/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg seed \
--driver mysql \
--config _examples/pipeline/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg run \
--driver mysql \
--config _examples/pipeline/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 1m

go run ./cmd/edg deseed \
--driver mysql \
--config _examples/pipeline/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/pipeline/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"
```

## Oracle

### Setup

```sh
docker compose -f cmd/harness/compose/compose_oracle.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver oracle \
--config _examples/pipeline/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg seed \
--driver oracle \
--config _examples/pipeline/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg run \
--driver oracle \
--config _examples/pipeline/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 1m

go run ./cmd/edg deseed \
--driver oracle \
--config _examples/pipeline/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg down \
--driver oracle \
--config _examples/pipeline/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```

## MSSQL

### Setup

```sh
docker compose -f cmd/harness/compose/compose_mssql.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver mssql \
--config _examples/pipeline/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=pipeline&encrypt=disable"

go run ./cmd/edg seed \
--driver mssql \
--config _examples/pipeline/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=pipeline&encrypt=disable"

go run ./cmd/edg run \
--driver mssql \
--config _examples/pipeline/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=pipeline&encrypt=disable" \
-w 10 \
-d 1m

go run ./cmd/edg deseed \
--driver mssql \
--config _examples/pipeline/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=pipeline&encrypt=disable"

go run ./cmd/edg down \
--driver mssql \
--config _examples/pipeline/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=pipeline&encrypt=disable"
```
