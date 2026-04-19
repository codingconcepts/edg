# Workload

Built-in workloads (bank, kv, movr, tpcc, tpch, ttlbench, ttllogger, ycsb) that run without a config file. The `--driver` flag selects the correct embedded config automatically.

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
-d 5s

go run ./cmd/edg workload kv all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5s

go run ./cmd/edg workload movr all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5s

go run ./cmd/edg workload tpcc all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5s

go run ./cmd/edg workload tpch all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 4 \
-d 5s

go run ./cmd/edg workload ttlbench all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5s

go run ./cmd/edg workload ttllogger all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5s

go run ./cmd/edg workload ycsb all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5s
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
-d 5s

go run ./cmd/edg workload kv all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 5s

go run ./cmd/edg workload movr all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 5s

go run ./cmd/edg workload tpcc all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 5s

go run ./cmd/edg workload tpch all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 4 \
-d 5s

go run ./cmd/edg workload ttlbench all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 5s

go run ./cmd/edg workload ttllogger all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 5s

go run ./cmd/edg workload ycsb all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 5s
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
-d 5s

go run ./cmd/edg workload kv all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 5s

go run ./cmd/edg workload movr all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 5s

go run ./cmd/edg workload tpcc all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 5s

go run ./cmd/edg workload tpch all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 4 \
-d 5s

go run ./cmd/edg workload ttlbench all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 5s

go run ./cmd/edg workload ttllogger all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 5s

go run ./cmd/edg workload ycsb all \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 5s
```

## Cloud Spanner

### Setup

```sh
docker compose -f _examples/compose_spanner.yml up -d

# Create instance and databases.
curl -s -X POST 'http://localhost:9020/v1/projects/test-project/instances' \
--json '{"instanceId":"test-instance","instance":{"config":"projects/test-project/instanceConfigs/emulator-config","displayName":"Test","nodeCount":1}}'

for db in bank kv movr tpcc tpch ttlbench ttllogger ycsb; do
  curl -s -X POST "http://localhost:9020/v1/projects/test-project/instances/test-instance/databases" \
    --json "{\"createStatement\":\"CREATE DATABASE \`${db}\`\"}"
done
```

### Run

```sh
SPANNER_EMULATOR_HOST=localhost:9010 \
go run ./cmd/edg workload bank all \
--driver spanner \
--url "projects/test-project/instances/test-instance/databases/bank" \
-w 10 \
-d 5s

SPANNER_EMULATOR_HOST=localhost:9010 \
go run ./cmd/edg workload kv all \
--driver spanner \
--url "projects/test-project/instances/test-instance/databases/kv" \
-w 10 \
-d 5s

SPANNER_EMULATOR_HOST=localhost:9010 \
go run ./cmd/edg workload movr all \
--driver spanner \
--url "projects/test-project/instances/test-instance/databases/movr" \
-w 10 \
-d 5s

CUSTOMERS="1000" \
ORDERS="1000" \
STOCK="1000" \
ITEMS="1000" \
SPANNER_EMULATOR_HOST=localhost:9010 \
go run ./cmd/edg workload tpcc all \
--driver spanner \
--url "projects/test-project/instances/test-instance/databases/tpcc" \
-w 10 \
-d 5s

SPANNER_EMULATOR_HOST=localhost:9010 \
go run ./cmd/edg workload tpch all \
--driver spanner \
--url "projects/test-project/instances/test-instance/databases/tpch" \
-w 4 \
-d 5s

SPANNER_EMULATOR_HOST=localhost:9010 \
go run ./cmd/edg workload ttlbench all \
--driver spanner \
--url "projects/test-project/instances/test-instance/databases/ttlbench" \
-w 10 \
-d 5s

SPANNER_EMULATOR_HOST=localhost:9010 \
go run ./cmd/edg workload ttllogger all \
--driver spanner \
--url "projects/test-project/instances/test-instance/databases/ttllogger" \
-w 10 \
-d 5s

SPANNER_EMULATOR_HOST=localhost:9010 \
go run ./cmd/edg workload ycsb all \
--driver spanner \
--url "projects/test-project/instances/test-instance/databases/ycsb" \
-w 10 \
-d 5s
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
-d 5s

go run ./cmd/edg workload kv all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=kv&encrypt=disable" \
-w 10 \
-d 5s

go run ./cmd/edg workload movr all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=movr&encrypt=disable" \
-w 10 \
-d 5s

go run ./cmd/edg workload tpcc all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=tpcc&encrypt=disable" \
-w 10 \
-d 5s

go run ./cmd/edg workload tpch all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=tpch&encrypt=disable" \
-w 4 \
-d 5s

go run ./cmd/edg workload ttlbench all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ttlbench&encrypt=disable" \
-w 10 \
-d 5s

go run ./cmd/edg workload ttllogger all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ttllogger&encrypt=disable" \
-w 10 \
-d 5s

go run ./cmd/edg workload ycsb all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ycsb&encrypt=disable" \
-w 10 \
-d 5s
```
