# Populate

A bulk data population example demonstrating `gen_batch` and `batch` functions to insert large volumes of data efficiently. Creates 1 million customers and their accounts using batched inserts of 10K rows at a time.

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
--config _examples/populate/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/populate/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/populate/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/populate/crdb.yaml \
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
--config _examples/populate/mysql.yaml \
--url "root:password@tcp(localhost:3306)/populate?parseTime=true"

go run ./cmd/edg seed \
--driver mysql \
--config _examples/populate/mysql.yaml \
--url "root:password@tcp(localhost:3306)/populate?parseTime=true"

go run ./cmd/edg deseed \
--driver mysql \
--config _examples/populate/mysql.yaml \
--url "root:password@tcp(localhost:3306)/populate?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/populate/mysql.yaml \
--url "root:password@tcp(localhost:3306)/populate?parseTime=true"
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
--config _examples/populate/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg seed \
--driver oracle \
--config _examples/populate/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg deseed \
--driver oracle \
--config _examples/populate/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg down \
--driver oracle \
--config _examples/populate/oracle.yaml \
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
--config _examples/populate/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=populate&encrypt=disable"

go run ./cmd/edg seed \
--driver mssql \
--config _examples/populate/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=populate&encrypt=disable"

go run ./cmd/edg deseed \
--driver mssql \
--config _examples/populate/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=populate&encrypt=disable"

go run ./cmd/edg down \
--driver mssql \
--config _examples/populate/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=populate&encrypt=disable"
```
