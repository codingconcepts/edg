# TATP

Telecom Application Transaction Processing benchmark. Models a mobile phone subscriber database with 80% reads and 20% writes. Simple schema, designed for maximum throughput testing.

Transaction profiles (per TATP spec):

- **get_subscriber_data** (35%) - Point lookup by subscriber ID
- **get_access_data** (35%) - Access info lookup
- **update_location** (14%) - Update subscriber location
- **get_new_destination** (10%) - Call forwarding lookup
- **update_subscriber_data** (2%) - Toggle a subscriber bit
- **insert_call_forwarding** (2%) - Add a forwarding rule
- **delete_call_forwarding** (2%) - Remove a forwarding rule

## CockroachDB

```sh
go run ./cmd/edg workload tatp all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 1m
```

## MySQL

```sh
go run ./cmd/edg workload tatp all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 1m
```

## Oracle

```sh
go run ./cmd/edg workload tatp all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 1m
```

## MSSQL

```sh
go run ./cmd/edg workload tatp all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=tatp&encrypt=disable" \
-w 10 \
-d 1m
```

## Cloud Spanner

```sh
SPANNER_EMULATOR_HOST=localhost:9010 \
go run ./cmd/edg workload tatp all \
--driver spanner \
--url "projects/test-project/instances/test-instance/databases/tatp" \
-w 10 \
-d 1m
```
