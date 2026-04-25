# each_cartesian

This minimal example shows how to reference environment variables and access them from your edg config files.

## CockroachDB

### Setup

```sh
docker compose -f cmd/harness/compose/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/each_cartesian/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/each_cartesian/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

# Check the data.
cockroach sql --insecure -e "SELECT COUNT(*) FROM c"

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/each_cartesian/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/each_cartesian/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
