# Batch Types

Demonstrates `query_batch` and `exec_batch` query types. A `query_batch` inserts products and captures the returned rows, then an `exec_batch` references those rows to insert reviews against them.

- **`query_batch`** evaluates args per row (controlled by `count` and `size`), collects values into comma-separated strings, and stores the query results for use by `ref_*` functions.
- **`exec_batch`** does the same arg generation but executes without reading results.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
docker exec -it node1 cockroach init --insecure
docker exec -it node1 cockroach sql --insecure
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/batch/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/batch/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

Check data

```sql
SELECT
  p.name,
  COUNT(r.id) AS review_count,
  AVG(r.rating)::DECIMAL(3,1) AS avg_rating,
  array_agg(r.rating) AS ratings
FROM product p
JOIN review r ON r.product_id = p.id
GROUP BY p.name
ORDER BY review_count DESC;
```

### Teardown

```sh
go run ./cmd/edg deseed \
--driver pgx \
--config _examples/batch/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/batch/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
