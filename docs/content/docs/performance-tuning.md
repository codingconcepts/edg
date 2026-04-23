---
title: Performance Tuning
weight: 12
---

# Performance Tuning

Tips for maximising throughput and getting clean benchmark results.

## Batch Size

For seed operations using `exec_batch`, the `size` field controls how many rows are grouped into each SQL statement. This has a significant impact on insert throughput:

| size | Effect |
|---|---|
| Too small (1–10) | Excessive round trips; dominated by network latency |
| Sweet spot (100–5000) | Good throughput with manageable transaction size |
| Too large (10000+) | Large transactions; risk of lock contention and OOM on the database |

Start with `size: 1000` and adjust based on your row width and database. Wide rows (many columns, large strings) benefit from smaller batches.

```yaml
seed:
  - name: populate_users
    type: exec_batch
    count: 1000000
    size: 5000
    args:
      - gen('email')
    query: |-
      INSERT INTO users (email)
      SELECT unnest(string_to_array('$1', __sep__))
```

## Workers vs Pool Size

The `--workers` flag controls concurrency (goroutines), while `--pool-size` controls the number of database connections. The relationship between them affects performance:

| Configuration | Behaviour |
|---|---|
| `workers == pool-size` | Each worker gets a dedicated connection. No contention. |
| `workers > pool-size` | Workers share connections. Some block waiting. Useful for simulating connection-limited environments. |
| `workers < pool-size` | Extra connections sit idle. No benefit over matching. |
| `pool-size = 0` (default) | Driver default (usually unlimited). Each worker gets its own connection. |

For benchmarks measuring peak throughput, match `pool-size` to `workers` or leave it at the default. For benchmarks simulating production conditions, set `pool-size` to match your application's connection pool.

## Prepared Statements

Setting `prepared: true` on `run` queries reduces server-side parse overhead by caching the query plan per worker:

```yaml
run:
  - name: lookup_product
    type: query
    prepared: true
    args:
      - ref_rand('fetch_products').id
    query: SELECT id, name, price FROM product WHERE id = $1
```

The benefit scales with query complexity. Simple point lookups see minimal improvement, but multi-table joins with aggregations can see 20–30% latency reduction.

Prepared statements are not compatible with batch types (`query_batch`, `exec_batch`) or queries inside transactions.

## Warmup Period

Database caches, JIT compilation, and connection establishment all affect early query latencies. Use `--warmup-duration` to run the workload for a period before collecting metrics:

```sh
edg run \
  --warmup-duration 30s \
  --duration 5m \
  -w 10 \
  ...
```

Workers run during warmup but results are discarded. This produces more representative p50/p95/p99 numbers by excluding cold-start outliers.

## Deterministic Seeding

Use `--rng-seed` to make generated data deterministic. This eliminates variability between runs, making it easier to compare benchmark results:

```sh
edg all --rng-seed 42 --driver pgx --config workload.yaml --url ${DATABASE_URL} -w 10 -d 5m
```

Two runs with the same seed produce identical seed data and identical random selections during the run phase.

## Distribution Selection

Choosing the right distribution for your workload matters more than raw throughput numbers:

| Distribution | When to use |
|---|---|
| `uniform` / `ref_rand` | Even access across all rows. Good baseline. |
| `zipf` / `set_zipf` | Hot-key patterns. Realistic for most OLTP workloads where some rows are accessed far more often. |
| `norm` / `set_norm` | Bell curve access. Good for time-based or range queries where most activity clusters around a centre. |
| `exp` / `set_exp` | Heavy bias toward low values. Good for recency-biased access (recent orders, new users). |

Zipfian distributions with `s=1.1` to `s=2.0` are the most common for realistic OLTP benchmarks. Higher `s` values create more contention.

## Run Weights

Use `run_weights` to control the read/write mix rather than duplicating queries:

```yaml
run_weights:
  read_order: 80
  update_status: 15
  insert_order: 5
```

This produces 80% reads, 15% updates, and 5% inserts, a realistic OLTP mix. Without `run_weights`, all queries execute sequentially on every iteration.

## Monitoring During Runs

Use `--metrics-addr` to expose Prometheus metrics for real-time monitoring:

```sh
edg run --metrics-addr :9090 --driver pgx --config workload.yaml --url ${DATABASE_URL} -w 10 -d 5m
```

This lets you correlate edg metrics with database-side metrics in Grafana. See [Observability]({{< relref "observability" >}}) for dashboard setup.

## Stages for Load Profiling

Use stages to vary worker counts over time and observe how the database responds to changing load:

```yaml
stages:
  - name: ramp
    workers: 1
    duration: 30s
  - name: low
    workers: 10
    duration: 2m
  - name: peak
    workers: 100
    duration: 5m
  - name: cooldown
    workers: 10
    duration: 2m
```

Combine with Prometheus metrics to produce load-vs-latency curves that reveal database scaling characteristics.
