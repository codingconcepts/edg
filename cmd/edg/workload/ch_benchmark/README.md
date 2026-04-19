# CH-benCHmark

Mixed OLTP+OLAP workload combining TPC-C transactions with TPC-H-style analytical queries running concurrently on the same schema. Tests HTAP (Hybrid Transactional/Analytical Processing) capability.

Uses the full TPC-C schema with nation, region, and supplier tables added per the CH-benCHmark specification.

Workload mix:

- **OLTP (80%)** - TPC-C transactions (new_order, payment, order_status, delivery, stock_level)
- **OLAP (20%)** - Analytical queries adapted to TPC-C schema (pricing summary, revenue forecast, important stock, shipping modes)

## CockroachDB

```sh
go run ./cmd/edg workload ch-benchmark all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 1m
```

## MySQL

```sh
go run ./cmd/edg workload ch-benchmark all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 1m
```

## Oracle

```sh
go run ./cmd/edg workload ch-benchmark all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 1m
```

## MSSQL

```sh
go run ./cmd/edg workload ch-benchmark all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ch&encrypt=disable" \
-w 10 \
-d 1m
```

## Cloud Spanner

```sh
SPANNER_EMULATOR_HOST=localhost:9010 \
go run ./cmd/edg workload ch-benchmark all \
--driver spanner \
--url "projects/test-project/instances/test-instance/databases/ch" \
-w 10 \
-d 1m
```
