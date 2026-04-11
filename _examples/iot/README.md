# IoT

An IoT workload with devices, sensors, and time-series readings.

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
--config _examples/iot/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/iot/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/iot/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/iot/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/iot/crdb.yaml \
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
--config _examples/iot/mysql.yaml \
--url "root:password@tcp(localhost:3306)/iot?parseTime=true"

go run ./cmd/edg seed \
--driver mysql \
--config _examples/iot/mysql.yaml \
--url "root:password@tcp(localhost:3306)/iot?parseTime=true"

go run ./cmd/edg run \
--driver mysql \
--config _examples/iot/mysql.yaml \
--url "root:password@tcp(localhost:3306)/iot?parseTime=true" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver mysql \
--config _examples/iot/mysql.yaml \
--url "root:password@tcp(localhost:3306)/iot?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/iot/mysql.yaml \
--url "root:password@tcp(localhost:3306)/iot?parseTime=true"
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
--config _examples/iot/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg seed \
--driver oracle \
--config _examples/iot/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg run \
--driver oracle \
--config _examples/iot/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver oracle \
--config _examples/iot/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg down \
--driver oracle \
--config _examples/iot/oracle.yaml \
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
--config _examples/iot/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=iot&encrypt=disable"

go run ./cmd/edg seed \
--driver mssql \
--config _examples/iot/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=iot&encrypt=disable"

go run ./cmd/edg run \
--driver mssql \
--config _examples/iot/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=iot&encrypt=disable" \
-w 100 \
-d 1m

go run ./cmd/edg deseed \
--driver mssql \
--config _examples/iot/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=iot&encrypt=disable"

go run ./cmd/edg down \
--driver mssql \
--config _examples/iot/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=iot&encrypt=disable"
```
