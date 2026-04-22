# Composite Types

Demonstrates CockroachDB composite types (user-defined record types) used as column types. Creates `amount` and `dimensions` composite types, then uses them in product and order tables.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver pgx \
--config _examples/composite_types/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 10s

# Or separately.
go run ./cmd/edg up \
--driver pgx \
--config _examples/composite_types/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/composite_types/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/composite_types/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 1m

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/composite_types/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/composite_types/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
