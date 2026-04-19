# SEATS

Airline reservation system benchmark. Models flights, customers, and seat reservations with contention on seat availability. Transactions include finding flights, checking open seats, booking, updating, and cancelling reservations.

## CockroachDB

```sh
go run ./cmd/edg workload seats all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 1m
```

## MySQL

```sh
go run ./cmd/edg workload seats all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 1m
```

## Oracle

```sh
go run ./cmd/edg workload seats all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 1m
```

## MSSQL

```sh
go run ./cmd/edg workload seats all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=seats&encrypt=disable" \
-w 10 \
-d 1m
```

## Cloud Spanner

```sh
SPANNER_EMULATOR_HOST=localhost:9010 \
go run ./cmd/edg workload seats all \
--driver spanner \
--url "projects/test-project/instances/test-instance/databases/seats" \
-w 10 \
-d 1m
```
