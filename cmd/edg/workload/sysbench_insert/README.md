# Sysbench Insert

Pure insert micro-benchmark matching sysbench's `oltp_insert` profile. Every operation inserts a new row with a unique key. Useful for measuring ingestion throughput, index build cost, and storage engine write path.

## CockroachDB

```sh
go run ./cmd/edg workload sysbench-insert all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 1m
```

## MySQL

```sh
go run ./cmd/edg workload sysbench-insert all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 1m
```

## Oracle

```sh
go run ./cmd/edg workload sysbench-insert all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 1m
```

## MSSQL

```sh
go run ./cmd/edg workload sysbench-insert all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=sysbench&encrypt=disable" \
-w 10 \
-d 1m
```

## Cloud Spanner

```sh
SPANNER_EMULATOR_HOST=localhost:9010 \
go run ./cmd/edg workload sysbench-insert all \
--driver spanner \
--url "projects/test-project/instances/test-instance/databases/sysbench" \
-w 10 \
-d 1m
```
