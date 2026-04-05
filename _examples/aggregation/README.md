# Aggregation

Demonstrates the aggregation functions by computing summary statistics over an init dataset and recording snapshots into a table.

## Functions

| Function | Signature | Returns | Description |
|---|---|---|---|
| `sum` | `sum(name, field)` | `float64` | Sum of a numeric field across all rows |
| `avg` | `avg(name, field)` | `float64` | Average of a numeric field across all rows |
| `min` | `min(name, field)` | `float64` | Minimum value of a numeric field |
| `max` | `max(name, field)` | `float64` | Maximum value of a numeric field |
| `count` | `count(name)` | `int` | Number of rows in the dataset |
| `distinct` | `distinct(name, field)` | `int` | Number of distinct values for a field |

All aggregation functions operate on named datasets populated by `init` queries.

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
--config _examples/aggregation/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg seed \
--driver pgx \
--config _examples/aggregation/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg run \
--driver pgx \
--config _examples/aggregation/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 4 \
-d 10s
```

### Verify

```sql
SELECT * FROM agg_snapshot ORDER BY created_at DESC LIMIT 5;
```

```
                   id                  | total_products | total_value | avg_price | min_price | max_price | distinct_categories |          created_at
---------------------------------------+----------------+-------------+-----------+-----------+-----------+---------------------+-------------------------------
 a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d |            500 |   125432.50 |    250.87 |      1.23 |    499.87 |                   5 | 2026-04-05 12:00:01.000000+00
```

### Teardown

```sh
edg deseed \
--driver pgx \
--config _examples/aggregation/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg down \
--driver pgx \
--config _examples/aggregation/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
