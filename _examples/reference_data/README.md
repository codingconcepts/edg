# Reference Data

Demonstrates the `reference` config section, which loads static datasets into memory without a database query. Reference data is available to all `ref_*` functions (`ref_rand`, `ref_same`, `ref_diff`, etc.) just like `init` query results.

This example defines a **categories** reference dataset with names and markup multipliers, then uses `ref_same` to seed products with a consistent category name and markup from the same row.

## CockroachDB

### Setup

```sh
cockroach demo --insecure --no-example-database
```

### Run

```sh
go run . up \
--driver pgx \
--config _examples/reference_data/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run . seed \
--driver pgx \
--config _examples/reference_data/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

Check data

```sql
SELECT
  region,
  COUNT(*)
FROM customer
GROUP BY region;

SELECT
  c.region,
  c.city
FROM customer c
ORDER BY random()
LIMIT 10;

SELECT
  c.region,
  array_agg(DISTINCT c.city) AS cities
FROM customer c
GROUP BY c.region
ORDER BY c.region;
```

Teardown

```sh
go run . deseed \
--driver pgx \
--config _examples/reference_data/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run . down \
--driver pgx \
--config _examples/reference_data/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
