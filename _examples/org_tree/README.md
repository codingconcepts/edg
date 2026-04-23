# Org Tree

Hierarchical org chart using seed capture for self-referential generation. Each level references the previous level's captured dataset via `ref_rand()`, producing a realistic tree structure:

- 1 CEO
- 5 VPs (report to CEO)
- 15 Directors (report to VPs)
- 50 Managers (report to Directors)
- 200 ICs (report to Managers)

Seed queries with named args automatically store their generated rows as datasets. Later seed queries reference them with `ref_rand('query_name')` to build parent-child relationships without needing a database round-trip.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/org_tree/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/org_tree/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

### Check

Connect

```sh
cockroach sql --insecure
```

Run query

```sql
WITH RECURSIVE tree AS (
  SELECT id, name, title, manager_id, 0 AS depth,
         name::STRING AS path
  FROM employees
  WHERE manager_id IS NULL

  UNION ALL

  SELECT e.id, e.name, e.title, e.manager_id, t.depth + 1,
         t.path || '/' || e.name
  FROM employees e
  JOIN tree t ON e.manager_id = t.id
)
SELECT
  repeat('  ', depth) || title || ': ' || name AS org_chart
FROM tree
ORDER BY path
LIMIT 30;
```

### Teardown

```sh
go run ./cmd/edg deseed \
--driver pgx \
--config _examples/org_tree/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/org_tree/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
