# Sysbench Update Index

Pure indexed-column update micro-benchmark matching sysbench's `oltp_update_index` profile. Every operation increments column `k` (which has a secondary index), causing both primary and index writes. Useful for measuring write amplification and secondary index maintenance overhead.

## CockroachDB

```sh
go run ./cmd/edg workload sysbench-update-index all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 1m
```

## MySQL

```sh
go run ./cmd/edg workload sysbench-update-index all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 1m
```

## Oracle

```sh
go run ./cmd/edg workload sysbench-update-index all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 1m
```

## MSSQL

```sh
go run ./cmd/edg workload sysbench-update-index all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=sysbench&encrypt=disable" \
-w 10 \
-d 1m
```

## Cloud Spanner

```sh
SPANNER_EMULATOR_HOST=localhost:9010 \
go run ./cmd/edg workload sysbench-update-index all \
--driver spanner \
--url "projects/test-project/instances/test-instance/databases/sysbench" \
-w 10 \
-d 1m
```
