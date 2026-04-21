# Global Sequences

Demonstrates globally unique auto-incrementing sequences shared across all workers, and the five distribution functions for referencing existing sequence values.

The `seq` section defines named sequences with a start value and step. `seq_global("name")` returns globally unique values using atomic counters. The distribution functions (`seq_rand`, `seq_zipf`, `seq_norm`, `seq_exp`, `seq_lognorm`) compute valid values algebraically from start/step/counter without storing values in memory.

## Functions

| Function | Description |
|---|---|
| `seq_global("name")` | Next globally unique value |
| `seq_rand("name")` | Uniform random pick |
| `seq_zipf("name", s, v)` | Zipfian (hot early values) |
| `seq_norm("name", mean, stddev)` | Normal distribution |
| `seq_exp("name", rate)` | Exponential distribution |
| `seq_lognorm("name", mu, sigma)` | Log-normal distribution |

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/global_sequences/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/global_sequences/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

### Verify distributions

After seeding, run the following queries to visualise each distribution as a histogram. Each query buckets the 1-1000 value 

```sql
-- Uniform (flat).
SELECT
  div(uniform_val - 1, 50) * 50 + 1 AS bucket,
  count(*) AS total,
  repeat('█', (count(*) * 50 / max(count(*)) OVER ())::INT) AS histogram
FROM samples
GROUP BY 1
ORDER BY 1;

-- Zipfian (hot early values).
SELECT
  div(zipf_val - 1, 50) * 50 + 1 AS bucket,
  count(*) AS total,
  repeat('█', (count(*) * 50 / max(count(*)) OVER ())::INT) AS histogram
FROM samples
GROUP BY 1
ORDER BY 1;

-- Normal (bell curve).
SELECT
  div(norm_val - 1, 50) * 50 + 1 AS bucket,
  count(*) AS total,
  repeat('█', (count(*) * 50 / max(count(*)) OVER ())::INT) AS histogram
FROM samples
GROUP BY 1
ORDER BY 1;

-- Exponential (decay).
SELECT
  div(exp_val - 1, 50) * 50 + 1 AS bucket,
  count(*) AS total,
  repeat('█', (count(*) * 50 / max(count(*)) OVER ())::INT) AS histogram
FROM samples
GROUP BY 1
ORDER BY 1;

-- Log-normal (right-skewed).
SELECT
  div(lognorm_val - 1, 50) * 50 + 1 AS bucket,
  count(*) AS total,
  repeat('█', (count(*) * 50 / max(count(*)) OVER ())::INT) AS histogram
FROM samples
GROUP BY 1
ORDER BY 1;
```

### Teardown

```sh
go run ./cmd/edg deseed \
--driver pgx \
--config _examples/global_sequences/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/global_sequences/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
