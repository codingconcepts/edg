# LTREE

Hierarchical org chart using PostgreSQL's `ltree` extension. Each management level builds its path by appending to its parent's path via `ltree()`:

- 1 CEO → `Jane_Doe`
- 5 VPs → `Jane_Doe.Alice_Smith`
- 15 Directors → `Jane_Doe.Alice_Smith.Bob_Jones`
- 50 Managers → `Jane_Doe.Alice_Smith.Bob_Jones.Carol_White`
- 200 ICs → `Jane_Doe.Alice_Smith.Bob_Jones.Carol_White.Dave_Brown`

The `ltree()` function joins its arguments with dots and sanitizes labels to valid ltree characters (`[A-Za-z0-9_]`).

## CockroachDB

### Setup

```sh
docker compose -f cmd/harness/compose/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/ltree/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/ltree/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

### Check

Connect

```sh
cockroach sql --insecure
```

Run queries

```sql
-- Show full tree
SELECT id, name, title, path FROM employees ORDER BY path LIMIT 30;

-- Find all reports under a VP (pick a real VP path)
SELECT * FROM employees
WHERE path <@ (SELECT path FROM employees WHERE title = 'VP' LIMIT 1);

-- Find direct reports of CEO
SELECT * FROM employees WHERE path <@ (SELECT path FROM employees WHERE title = 'CEO') AND nlevel(path) = 2;

-- Count employees at each depth
SELECT nlevel(path) AS depth, count(*) FROM employees GROUP BY depth ORDER BY depth;
```

### Teardown

```sh
go run ./cmd/edg deseed \
--driver pgx \
--config _examples/ltree/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/ltree/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
