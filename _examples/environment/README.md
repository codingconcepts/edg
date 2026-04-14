# environment

This minimal example shows how to reference environment variables and access them from your edg config files.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/environment/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

ABC="the quick brown fox" DEF="jumps over the lazy dog" \
go run ./cmd/edg seed \
--driver pgx \
--config _examples/environment/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

# Check the data.
cockroach sql --insecure -e "SELECT key, val FROM e ORDER BY key"

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/environment/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/environment/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
