# Correlated Totals

Demonstrates `distribute_sum()` and `distribute_weighted()` for generating parent-child data with correlated totals.

- **`distribute_sum(total, minN, maxN, precision)`** partitions `total` into a random number of parts (between `minN` and `maxN`), each rounded to `precision` decimal places. The parts always sum exactly to `total`.
- **`distribute_weighted(total, weights, noise, precision)`** splits `total` according to proportional weights with controlled randomness. `noise` (0-1) blends between exact proportions and fully random.

Each invoice has:
- 3-7 **line items** whose amounts sum to the total (`distribute_sum`)
- A **subtotal/tax/shipping** breakdown at roughly 85/10/5 proportions (`distribute_weighted`)

Two strategies are shown:

- **CockroachDB** uses a writable CTE to create each invoice and its line items atomically in a single statement.
- **MySQL** creates invoices with the breakdown first, then iterates them with `ref_diff` to insert correlated line items via `distribute_sum`.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/invoice_lines/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/invoice_lines/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

Confirm the data is as expected.

```sql
-- Line items sum to total, breakdown sums to total
SELECT
  i.id, i.total,
  SUM(li.amount) AS line_total,
  i.subtotal + i.tax + i.shipping AS breakdown_total,
  COUNT(*) AS lines,
  i.total = SUM(li.amount) AS lines_match,
  i.total = i.subtotal + i.tax + i.shipping AS breakdown_matches
FROM invoice i
JOIN line_item li ON li.invoice_id = i.id
GROUP BY i.id, i.total, i.subtotal, i.tax, i.shipping
ORDER BY i.id
LIMIT 10;

-- Should return zero rows
SELECT 'line_mismatch' AS issue, i.id, i.total
FROM invoice i
JOIN line_item li ON li.invoice_id = i.id
GROUP BY i.id, i.total
HAVING i.total != SUM(li.amount)
UNION ALL
SELECT 'breakdown_mismatch', id, total
FROM invoice
WHERE total != subtotal + tax + shipping;
```

```sh
go run ./cmd/edg deseed \
--driver pgx \
--config _examples/invoice_lines/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/invoice_lines/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

## MySQL

### Setup

```sh
docker compose -f _examples/compose_mysql.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver mysql \
--config _examples/invoice_lines/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg seed \
--driver mysql \
--config _examples/invoice_lines/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg deseed \
--driver mysql \
--config _examples/invoice_lines/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/invoice_lines/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"
```
