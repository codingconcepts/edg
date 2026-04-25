# E-Commerce

An e-commerce workload with categories, products, customers, and orders.

## CockroachDB

### Setup

```sh
docker compose -f cmd/harness/compose/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/ecommerce/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/ecommerce/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/ecommerce/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/ecommerce/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/ecommerce/crdb.yaml \
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
--config _examples/ecommerce/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg seed \
--driver mysql \
--config _examples/ecommerce/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg run \
--driver mysql \
--config _examples/ecommerce/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver mysql \
--config _examples/ecommerce/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/ecommerce/mysql.yaml \
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
--config _examples/ecommerce/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg seed \
--driver oracle \
--config _examples/ecommerce/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg run \
--driver oracle \
--config _examples/ecommerce/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver oracle \
--config _examples/ecommerce/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg down \
--driver oracle \
--config _examples/ecommerce/oracle.yaml \
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
--config _examples/ecommerce/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ecommerce&encrypt=disable"

go run ./cmd/edg seed \
--driver mssql \
--config _examples/ecommerce/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ecommerce&encrypt=disable"

go run ./cmd/edg run \
--driver mssql \
--config _examples/ecommerce/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ecommerce&encrypt=disable" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver mssql \
--config _examples/ecommerce/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ecommerce&encrypt=disable"

go run ./cmd/edg down \
--driver mssql \
--config _examples/ecommerce/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ecommerce&encrypt=disable"
```
