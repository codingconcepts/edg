# TPC-C

A TPC-C benchmark implementation with all 5 transaction profiles (New-Order, Payment, Order-Status, Delivery, Stock-Level) using writable CTEs for atomic execution.

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
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
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
--config _examples/tpcc/mysql.yaml \
--url "root:password@tcp(localhost:3306)/tpcc?parseTime=true"

go run ./cmd/edg seed \
--driver mysql \
--config _examples/tpcc/mysql.yaml \
--url "root:password@tcp(localhost:3306)/tpcc?parseTime=true"

go run ./cmd/edg run \
--driver mysql \
--config _examples/tpcc/mysql.yaml \
--url "root:password@tcp(localhost:3306)/tpcc?parseTime=true" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver mysql \
--config _examples/tpcc/mysql.yaml \
--url "root:password@tcp(localhost:3306)/tpcc?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/tpcc/mysql.yaml \
--url "root:password@tcp(localhost:3306)/tpcc?parseTime=true"
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
--config _examples/tpcc/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg seed \
--driver oracle \
--config _examples/tpcc/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg run \
--driver oracle \
--config _examples/tpcc/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver oracle \
--config _examples/tpcc/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg down \
--driver oracle \
--config _examples/tpcc/oracle.yaml \
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
--config _examples/tpcc/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=tpcc&encrypt=disable"

go run ./cmd/edg seed \
--driver sqlserver \
--config _examples/tpcc/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=tpcc&encrypt=disable"

go run ./cmd/edg run \
--driver sqlserver \
--config _examples/tpcc/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=tpcc&encrypt=disable" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver sqlserver \
--config _examples/tpcc/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=tpcc&encrypt=disable"

go run ./cmd/edg down \
--driver sqlserver \
--config _examples/tpcc/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=tpcc&encrypt=disable"
```
