# Distributions

Demonstrates all five distribution functions by writing values into a single table with a `dist_type` label, making it easy to compare histograms side by side.

## Functions

### Numeric distributions

| Function | Signature | Description |
|---|---|---|
| `uniform` | `uniform(min, max)` | Flat distribution, every value equally likely |
| `zipf` | `zipf(s, v, max)` | Power-law skew, low values dominate |
| `norm_f` | `norm_f(mean, stddev, min, max, precision)` | Bell curve centered on mean |
| `exp_f` | `exp_f(rate, min, max, precision)` | Exponential decay from min |
| `lognorm_f` | `lognorm_f(mu, sigma, min, max, precision)` | Right-skewed with a long tail |

### Set distributions

Pick from a predefined set of values using a distribution to control which items are selected most often.

| Function | Signature | Description |
|---|---|---|
| `set_rand` | `set_rand(values, weights)` | Uniform or weighted random selection from a set |
| `set_norm` | `set_norm(values, mean, stddev)` | Normal distribution over indices; `mean` index picked most often |
| `set_exp` | `set_exp(values, rate)` | Exponential distribution over indices; lower indices picked most often |
| `set_lognorm` | `set_lognorm(values, mu, sigma)` | Log-normal distribution over indices; right-skewed selection |
| `set_zipf` | `set_zipf(values, s, v)` | Zipfian distribution over indices; strong power-law skew toward first items |

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
--config _examples/distributions/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/distributions/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 30s
```

### Verify distributions

After running, query each distribution as a histogram. The `value` column is bucketed into ranges of 10, and the bar length is scaled relative to the most common bucket within that distribution.

#### All distributions at a glance

```sql
SELECT
  dist_type,
  (floor(value / 10) * 10)::INT AS bucket,
  count(*) AS total,
  repeat('#', (count(*) * 40 / max(count(*)) OVER (PARTITION BY dist_type))::INT) AS histogram
FROM distributions
GROUP BY dist_type, bucket
ORDER BY dist_type, bucket;
```

#### Individual distribution

Replace `'normal'` with any of: `uniform`, `zipfian`, `normal`, `exponential`, `lognormal`.

```sql
SELECT
  (floor(value / 10) * 10)::INT AS bucket,
  count(*) AS total,
  repeat('#', (count(*) * 50 / max(count(*)) OVER ())::INT) AS histogram
FROM distributions
WHERE dist_type = 'normal'
GROUP BY bucket
ORDER BY bucket;
```

Example output (normal):

```
  bucket | total |                     histogram
---------+-------+-----------------------------------------------------
       0 |    21 | #
      10 |   122 | ####
      20 |   476 | ##############
      30 |  1083 | #################################
      40 |  1599 | ################################################
      50 |  1661 | ##################################################
      60 |  1087 | #################################
      70 |   472 | ##############
      80 |   132 | ####
      90 |    25 | #
```

### Teardown

```sh
go run ./cmd/edg deseed \
--driver pgx \
--config _examples/distributions/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/distributions/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
