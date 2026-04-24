---
title: Sync
weight: 4
---

# Sync

The `sync` command writes identical data to two databases and verifies consistency. It's designed for testing dual-write patterns, CDC pipelines, and cross-database replication.

Each database gets its own config file with driver-specific SQL, and the `--rng-seed` flag ensures both sides generate identical data.

## Subcommands

| Command | Description |
|---|---|
| `sync run` | Run `up` and `seed` on the source, and optionally the target |
| `sync verify` | Compare tables row-by-row across source and target |
| `sync down` | Run `deseed` and `down` on both databases |

## Flags

### Connection flags

These persistent flags apply to all sync subcommands:

<div class="cli-flags">

| Flag / Env Var | Default | Description |
|---|---|---|
| `--source-driver`<br>`EDG_SOURCE_DRIVER` | `pgx` | Source database driver |
| `--source-url`<br>`EDG_SOURCE_URL` | | Source database connection URL |
| `--source-config`<br>`EDG_SOURCE_CONFIG` | | Source edg config file |
| `--target-driver`<br>`EDG_TARGET_DRIVER` | `pgx` | Target database driver |
| `--target-url`<br>`EDG_TARGET_URL` | | Target database connection URL |
| `--target-config`<br>`EDG_TARGET_CONFIG` | | Target edg config file (omit for CDC mode) |

</div>

### Verify flags

These flags are specific to `sync verify`:

| Flag | Default | Description |
|---|---|---|
| `--tables` | | Comma-separated table names to verify (required) |
| `--order-by` | | Column for deterministic row ordering (required) |
| `--ignore-columns` | | Comma-separated columns to skip during comparison |
| `--wait` | `0` | Delay before verifying, to allow for replication lag |
| `--batch-size` | `10000` | Rows per verification batch |

## Dual-Write Mode example

This example writes 1,000 users and 5,000 orders to both CockroachDB and MySQL, then verifies the data matches.

### Setup

Start both databases and wait for them to be available:

```sh
docker compose -f _examples/compose_crdb.yml up -d
docker compose -f _examples/compose_mysql.yml up -d
```

### Seed both databases

```sh
edg sync run \
  --source-driver pgx \
  --source-url "postgres://root:password@localhost:26257/defaultdb?sslmode=disable" \
  --source-config _examples/sync/crdb.yaml \
  --target-driver mysql \
  --target-url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
  --target-config _examples/sync/mysql.yaml \
  --rng-seed 42
```

The PRNG is re-seeded before each side, so both databases receive identical generated values.

### Check the data

**CockroachDB:**

```sh
cockroach sql --insecure -e "SELECT id, name FROM users ORDER BY id LIMIT 5"
```

```
  id |      name
-----+-----------------
   1 | Shea Gibson
   2 | Alta Bayer
   3 | Reagan Powell
   4 | Aiden Elliott
   5 | Dock Marquardt
(5 rows)
```

**MySQL:**

```sh
mysql -u root -ppassword -h 127.0.0.1 defaultdb -e "SELECT id, name FROM users ORDER BY id LIMIT 5"
```

```
+----+----------------+
| id | name           |
+----+----------------+
|  1 | Shea Gibson    |
|  2 | Alta Bayer     |
|  3 | Reagan Powell  |
|  4 | Aiden Elliott  |
|  5 | Dock Marquardt |
+----+----------------+
```

**CockroachDB orders:**

```sh
cockroach sql --insecure -e "SELECT * FROM orders LIMIT 5"
```

```
   id  | user_id | amount |  status   |         created_at
-------+---------+--------+-----------+-----------------------------
  1001 |     620 | 199.20 | shipped   | 2026-04-24 14:51:56.250121
  1002 |     513 | 140.76 | pending   | 2026-04-24 14:51:56.250121
  1003 |     249 | 138.95 | delivered | 2026-04-24 14:51:56.250121
  1004 |     498 | 371.20 | pending   | 2026-04-24 14:51:56.250121
  1005 |     642 | 246.28 | pending   | 2026-04-24 14:51:56.250121
(5 rows)
```

**MySQL orders:**

```sh
mysql -u root -ppassword -h 127.0.0.1 defaultdb -e "SELECT * FROM orders LIMIT 5"
```

```
+------+---------+--------+-----------+---------------------+
| id   | user_id | amount | status    | created_at          |
+------+---------+--------+-----------+---------------------+
| 1001 |     620 | 199.20 | shipped   | 2026-04-24 14:51:56 |
| 1002 |     513 | 140.76 | pending   | 2026-04-24 14:51:56 |
| 1003 |     249 | 138.95 | delivered | 2026-04-24 14:51:56 |
| 1004 |     498 | 371.20 | pending   | 2026-04-24 14:51:56 |
| 1005 |     642 | 246.28 | pending   | 2026-04-24 14:51:56 |
+------+---------+--------+-----------+---------------------+
```

IDs, names, amounts, and statuses match across both databases.

### Verify consistency

```sh
edg sync verify \
  --source-driver pgx \
  --source-url "postgres://root:password@localhost:26257/defaultdb?sslmode=disable" \
  --target-driver mysql \
  --target-url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
  --tables users,orders \
  --order-by id \
  --ignore-columns created_at
```

```
INFO table verified table=users rows=1000
INFO table verified table=orders rows=5000
INFO all tables verified
```

The `--ignore-columns created_at` flag skips the timestamp column since CockroachDB and MySQL store timestamps at different precisions.

Verification uses batched keyset pagination (default 10,000 rows per batch), so it works efficiently on tables with millions of rows without loading the entire table into memory.

### Teardown

```sh
edg sync down \
  --source-driver pgx \
  --source-url "postgres://root:password@localhost:26257/defaultdb?sslmode=disable" \
  --source-config _examples/sync/crdb.yaml \
  --target-driver mysql \
  --target-url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
  --target-config _examples/sync/mysql.yaml

docker compose -f _examples/compose_crdb.yml down
docker compose -f _examples/compose_mysql.yml down
```

## External Replication mode

When `--target-config` is omitted, `sync run` only writes to the source. The target is expected to receive data through an external replication mechanism (e.g. MOLT Fetch/Replicator, Debezium, logical replication).

```sh
edg sync run \
  --source-driver pgx \
  --source-url "postgres://root:password@localhost:26257/defaultdb?sslmode=disable" \
  --source-config _examples/sync/crdb.yaml \
  --rng-seed 42
```

After replication completes, use `sync verify` with `--wait` to allow for lag:

```sh
edg sync verify \
  --source-driver pgx \
  --source-url "postgres://root:password@localhost:26257/defaultdb?sslmode=disable" \
  --target-driver mysql \
  --target-url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
  --tables users,orders \
  --order-by id \
  --ignore-columns created_at \
  --wait 5s
```

## How verification works

The `sync verify` command compares tables using a merge-join over batched keyset pagination:

1. Both databases are queried in batches: `SELECT * FROM table WHERE order_col > last_value ORDER BY order_col LIMIT batch_size`
2. Rows are compared by the `--order-by` column using a sorted merge. Rows present in one side but not the other are reported as `MISSING` or `EXTRA`
3. For matching rows, all columns (except `--ignore-columns`) are compared as strings for driver-agnostic type handling
4. Mismatches are reported immediately as they're found
5. Memory usage is O(batch_size), not O(total_rows)

If any mismatches are found, `sync verify` exits with status code 1.

### Mismatch output

```
MISMATCH table=users id=42 column=email source="old@example.com" target="new@example.com"
MISSING  table=orders id=99 side=target
EXTRA    table=orders id=100 side=target
```

| Type | Meaning |
|---|---|
| `MISMATCH` | Row exists in both but a column value differs |
| `MISSING` | Row exists in source but not in target |
| `EXTRA` | Row exists in target but not in source |
