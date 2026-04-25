# Geo Spatial

A dating app workload with user profiles, location-based discovery, swipes, matches, and messaging.

## CockroachDB

### Setup

```sh
docker compose -f cmd/harness/compose/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/geo_spatial/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/geo_spatial/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/geo_spatial/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 30s
```

Check data

```sh
cockroach sql --insecure \
-e "SELECT direction, COUNT(*) FROM swipe GROUP BY direction"
```

```sh
go run ./cmd/edg deseed \
--driver pgx \
--config _examples/geo_spatial/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/geo_spatial/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
