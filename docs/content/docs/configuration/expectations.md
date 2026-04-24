---
title: Expectations
weight: 1
---

# Expectations

The `expectations` section defines assertions that are evaluated after the workload finishes. Each expectation is a boolean expression checked against the collected run metrics. If any expectation fails, edg prints the results and exits with a non-zero status code, making it suitable for CI/CD pipelines.

```yaml
expectations:
  - error_rate < 1
  - check_balance.p99 < 100
  - tpm > 5000
```

Expressions use the same [expr](https://expr-lang.org/) engine as arg expressions and must evaluate to a boolean.

## Referencing globals

Expectations can reference any variable defined in the `globals` section. This avoids hardcoding values that already exist in your config:

```yaml
globals:
  accounts: 10000
  max_error_pct: 5

expectations:
  - error_rate < max_error_pct
  - query: SELECT COUNT(*) AS cnt FROM account
    expr: cnt == accounts
```

Global names must not collide with built-in metrics (`error_rate`, `success_count`, `total_errors`, `tpm`). edg will reject the config at startup if they do.

## Database-backed expectations

An expectation can be an object with `query` and `expr` fields. The SQL query runs after the workload finishes and its first-row columns are available in the expression alongside globals and metrics:

```yaml
expectations:
  - query: SELECT COUNT(*) AS cnt FROM account
    expr: cnt == accounts
  - query: SELECT SUM(balance) AS total FROM account
    expr: total > 0
```

## Available Metrics

Expectations have access to globals, global metrics, and per-query metrics. All latency values are in milliseconds and all error rates are percentages (0–100).

### Global metrics

| Metric | Type | Description |
|---|---|---|
| `error_rate` | float | Overall error rate as a percentage between 0 and 100. Calculated as `error_count / (error_count + success_count) * 100`. |
| `success_count` | int | Total successful operations across all queries. |
| `total_errors` | int | Total failed operations across all queries. |
| `tpm` | float | Transactions per minute (success_count / elapsed minutes). |

### Per-query metrics

Per-query metrics are accessed using dot notation with the query name, e.g. `check_balance.p99`. The query name must match the `name` field in your `run` section.

| Metric | Type | Description |
|---|---|---|
| `<query>.success_count` | int | Successful operations for this query. |
| `<query>.error_count` | int | Failed operations for this query. |
| `<query>.error_rate` | float | Error rate as a percentage. |
| `<query>.avg` | float | Average latency in ms. |
| `<query>.p50` | float | 50th percentile latency in ms. |
| `<query>.p95` | float | 95th percentile latency in ms. |
| `<query>.p99` | float | 99th percentile latency in ms. |
| `<query>.qps` | float | Queries per second. |

## Examples

Ensure the overall error rate stays below 1%:

```yaml
expectations:
  - error_rate < 1
```

Ensure a specific query's p99 latency stays under 100ms and its error rate is zero:

```yaml
expectations:
  - check_balance.p99 < 100
  - check_balance.errors == 0
```

Combine multiple conditions in a single expression:

```yaml
expectations:
  - error_rate < 0.5 && tpm > 10000
```

## Output

After the run summary, expectations are printed with a PASS/FAIL status:

```
expectations
  PASS  error_rate < 1
  PASS  check_balance.p99 < 100
  FAIL  tpm > 5000
```

If any expectation fails, edg exits with status code 1 and reports the number of failures. When using the `all` command, teardown (`deseed` and `down`) still runs before the non-zero exit, so your database is left clean regardless of expectation results.

## CI/CD Usage

A typical CI pipeline step runs the workload and relies on the exit code to gate the build:

```sh
edg all \
  --driver pgx \
  --config workload.yaml \
  --url "$DATABASE_URL" \
  -w 50 \
  -d 5m
```

If any expectation defined in `workload.yaml` fails, the command exits with code 1, failing the pipeline step.

For a complete guide to using edg as an integration testing tool, see [Integration Testing]({{< relref "integration-testing" >}}).
