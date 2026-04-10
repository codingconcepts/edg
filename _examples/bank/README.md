# Bank

A simpler workload modelling bank account operations (balance checks, credits, transfers). Useful for contention and correctness testing.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
docker exec -it node1 cockroach init --insecure
docker exec -it node1 cockroach sql --insecure
```

### Run

```sh
go run ./cmd/edg all \
--driver pgx \
--config _examples/bank/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

# Or separately.
go run ./cmd/edg up \
--driver pgx \
--config _examples/bank/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/bank/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/bank/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/bank/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/bank/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

## MySQL

### Setup

```sh
docker compose -f _examples/compose_mysql.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver mysql \
--config _examples/bank/mysql.yaml \
--url "root:password@tcp(localhost:3306)/bank?parseTime=true"
```

## Oracle

### Setup

```sh
docker compose -f _examples/compose_oracle.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver oracle \
--config _examples/bank/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```

## SQL Server

### Setup

```sh
docker compose -f _examples/compose_sqlserver.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver sqlserver \
--config _examples/bank/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=bank&encrypt=disable"

# Or separately.
go run ./cmd/edg up \
--driver sqlserver \
--config _examples/bank/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=bank&encrypt=disable"

go run ./cmd/edg seed \
--driver sqlserver \
--config _examples/bank/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=bank&encrypt=disable"

go run ./cmd/edg run \
--driver sqlserver \
--config _examples/bank/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=bank&encrypt=disable" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver sqlserver \
--config _examples/bank/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=bank&encrypt=disable"

go run ./cmd/edg down \
--driver sqlserver \
--config _examples/bank/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=bank&encrypt=disable"
```
