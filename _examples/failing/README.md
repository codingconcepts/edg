# Failing

Demonstrates map lookup with `fail()` to validate environment variables at runtime. If `FLY_REGION` isn't a recognized key in the map, `fail('bad region')` gracefully stops the worker with an error.

## Functions

| Function | Signature | Description |
|---|---|---|
| `env` | `env('VAR')` | Read an environment variable |
| `fail` | `fail('message')` | Stop the worker gracefully with an error message |
| `??` | `expr ?? fallback` | Nil-coalesce: use fallback when expr is nil |

## CockroachDB

### Setup

```sh
docker compose -f cmd/harness/compose/compose_crdb.yml up -d
```

### Run (success)

With a valid region, events are inserted with the mapped AWS region:

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/failing/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

FLY_REGION=fra \
go run ./cmd/edg run \
--driver pgx \
--config _examples/failing/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 1 \
-d 5s

# Check the data.
cockroach sql --insecure -e "SELECT region, payload FROM event LIMIT 5"
```

### Run (failure)

With an unrecognized region, `fail()` stops the worker:

```sh
FLY_REGION=unknown \
go run ./cmd/edg run \
--driver pgx \
--config _examples/failing/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5s
```

### Teardown

```sh
go run ./cmd/edg deseed \
--driver pgx \
--config _examples/failing/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/failing/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
