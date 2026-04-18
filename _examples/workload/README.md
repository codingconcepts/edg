# Workload

Built-in workloads (bank, tpcc, ycsb) that run without a config file. The `--driver` flag selects the correct embedded config automatically.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg workload bank all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 1m

# Or separately.
go run ./cmd/edg workload bank up \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg workload bank seed \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg workload bank run \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 1m

go run ./cmd/edg workload bank deseed \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg workload bank down \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable"
```

### Other workloads

```sh
go run ./cmd/edg workload tpcc all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 1m

go run ./cmd/edg workload ycsb all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 1m
```

## MySQL

### Setup

```sh
docker compose -f _examples/compose_mysql.yml up -d
```

### Run

```sh
go run ./cmd/edg workload bank all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 1m

go run ./cmd/edg workload tpcc all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 1m

go run ./cmd/edg workload ycsb all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 1m
```

## Oracle

### Setup

```sh
docker compose -f _examples/compose_oracle.yml up -d
```

### Run

```sh
go run ./cmd/edg workload bank all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 1m

go run ./cmd/edg workload tpcc all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 1m

go run ./cmd/edg workload ycsb all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 1m
```

## MSSQL

### Setup

```sh
docker compose -f _examples/compose_mssql.yml up -d
```

### Run

```sh
go run ./cmd/edg workload bank all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=bank&encrypt=disable" \
-w 10 \
-d 1m

go run ./cmd/edg workload tpcc all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=tpcc&encrypt=disable" \
-w 10 \
-d 1m

go run ./cmd/edg workload ycsb all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ycsb&encrypt=disable" \
-w 10 \
-d 1m
```
