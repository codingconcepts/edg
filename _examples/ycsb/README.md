# YCSB

A Yahoo! Cloud Serving Benchmark implementation with all 6 workload profiles (A-F) using Zipfian key distribution. Switch between profiles by changing `run_weights` in the config.

Workload profiles:

- **A** - Update heavy: 50% read, 50% update
- **B** - Read mostly: 95% read, 5% update
- **C** - Read only: 100% read
- **D** - Read latest: 95% read, 5% insert
- **E** - Short ranges: 95% scan, 5% insert
- **F** - Read-modify-write: 50% read, 50% read-modify-write

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/ycsb/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/ycsb/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/ycsb/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/ycsb/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/ycsb/crdb.yaml \
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
--config _examples/ycsb/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg seed \
--driver mysql \
--config _examples/ycsb/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg run \
--driver mysql \
--config _examples/ycsb/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver mysql \
--config _examples/ycsb/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/ycsb/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"
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
--config _examples/ycsb/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg seed \
--driver oracle \
--config _examples/ycsb/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg run \
--driver oracle \
--config _examples/ycsb/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver oracle \
--config _examples/ycsb/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg down \
--driver oracle \
--config _examples/ycsb/oracle.yaml \
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
--config _examples/ycsb/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ycsb&encrypt=disable"

go run ./cmd/edg seed \
--driver mssql \
--config _examples/ycsb/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ycsb&encrypt=disable"

go run ./cmd/edg run \
--driver mssql \
--config _examples/ycsb/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ycsb&encrypt=disable" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver mssql \
--config _examples/ycsb/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ycsb&encrypt=disable"

go run ./cmd/edg down \
--driver mssql \
--config _examples/ycsb/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ycsb&encrypt=disable"
```
