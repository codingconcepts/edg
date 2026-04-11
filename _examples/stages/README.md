# Stages

An example of staged execution, with stages of different duration, with different numbers of workers.

When `stages` is defined in the config, the `-w` and `-d` CLI flags are ignored. Each stage runs sequentially with its own worker count and duration:

```yaml
stages:
  - name: ramp
    workers: 1
    duration: 10s
  - name: steady
    workers: 10
    duration: 30s
```

This runs the workload with 1 worker for 10 seconds, then 10 workers for 30 seconds.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
docker exec -it node1 cockroach init --insecure
docker exec -it node1 cockroach sql --insecure
```

### Run

```sh
go run ./cmd/edg all \
--driver pgx \
--config _examples/stages/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

## MySQL

### Setup

```sh
docker compose -f _examples/compose_mysql.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver mysql \
--config _examples/stages/mysql.yaml \
--url "root:password@tcp(localhost:3306)/stages?parseTime=true"
```

## Oracle

### Setup

```sh
docker compose -f _examples/compose_oracle.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver oracle \
--config _examples/stages/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```

## MSSQL

### Setup

```sh
docker compose -f _examples/compose_mssql.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver mssql \
--config _examples/stages/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=stages&encrypt=disable"
```
