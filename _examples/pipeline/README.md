# Pipeline

A minimal example demonstrating chained run queries where each step's results feed the next. Three tables (`a`, `b`, `c`) with foreign key relationships show how `ref_rand` and `ref_same` pass data between sequential run queries.

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
docker compose -f _examples/compose_mysql.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver mysql \
--config _examples/pipeline/mysql.yaml \
--url "root:password@tcp(localhost:3306)/pipeline?parseTime=true"

go run ./cmd/edg seed \
--driver mysql \
--config _examples/pipeline/mysql.yaml \
--url "root:password@tcp(localhost:3306)/pipeline?parseTime=true"

go run ./cmd/edg run \
--driver mysql \
--config _examples/pipeline/mysql.yaml \
--url "root:password@tcp(localhost:3306)/pipeline?parseTime=true" \
-w 10 \
-d 1m

go run ./cmd/edg deseed \
--driver mysql \
--config _examples/pipeline/mysql.yaml \
--url "root:password@tcp(localhost:3306)/pipeline?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/pipeline/mysql.yaml \
--url "root:password@tcp(localhost:3306)/pipeline?parseTime=true"
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

## SQL Server

### Setup

```sh
docker compose -f _examples/compose_sqlserver.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver sqlserver \
--config _examples/pipeline/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=pipeline&encrypt=disable"

go run ./cmd/edg seed \
--driver sqlserver \
--config _examples/pipeline/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=pipeline&encrypt=disable"

go run ./cmd/edg run \
--driver sqlserver \
--config _examples/pipeline/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=pipeline&encrypt=disable" \
-w 10 \
-d 1m

go run ./cmd/edg deseed \
--driver sqlserver \
--config _examples/pipeline/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=pipeline&encrypt=disable"

go run ./cmd/edg down \
--driver sqlserver \
--config _examples/pipeline/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=pipeline&encrypt=disable"
```
