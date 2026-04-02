# Social

A social network workload with users, posts, follows, and tags.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
docker exec -it node1 cockroach init --insecure
docker exec -it node1 cockroach sql --insecure
```

### Run

```sh
edg up \
--driver pgx \
--config _examples/social/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg seed \
--driver pgx \
--config _examples/social/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg run \
--driver pgx \
--config _examples/social/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 100 \
-d 1m

edg deseed \
--driver pgx \
--config _examples/social/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg down \
--driver pgx \
--config _examples/social/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
