---
title: Run Weights & Workers
weight: 2
---

# Run Weights & Workers

## Run Weights

The optional `run_weights` map controls the workload mix during execution. Each key is a run item name (either a standalone query name or a transaction name), and the value is a relative weight. On each iteration, a single item is chosen by weighted random selection:

```yaml
run_weights:
  check_balance: 50
  make_transfer: 50

run:
  - name: check_balance
    type: query
    args: [ref_rand('fetch_accounts').id]
    query: SELECT balance FROM account WHERE id = $1::UUID

  - transaction: make_transfer
    locals:
      amount: gen('number:1,100')
    queries:
      - name: read_balance
        type: query
        args: [ref_diff('fetch_accounts').id]
        query: SELECT id, balance FROM account WHERE id = $1::UUID
      - name: debit
        type: exec
        args:
          - ref_same('read_balance').id
          - local('amount')
        query: UPDATE account SET balance = balance - $2::FLOAT WHERE id = $1::UUID
```

In this example, each iteration picks either the standalone `check_balance` query (50% of the time) or the entire `make_transfer` transaction (50% of the time). When a transaction is selected, all its queries run inside a single `BEGIN`/`COMMIT` block.

If `run_weights` is omitted, all `run` items execute sequentially on each iteration.

## Workers

The `workers` section defines background queries that run independently on a fixed schedule alongside the main workload. Each worker is a regular query with an added `rate` field controlling execution frequency.

```yaml
workers:
  - name: reap_expired_leases
    rate: 1/5s
    type: exec
    query: |-
      UPDATE runs
      SET status = 'pending', worker_id = NULL
      WHERE status IN ('claimed', 'running')
        AND lease_expires_at < now()

  - name: refresh_stats
    rate: 3/1m
    type: query
    query: SELECT count(*) AS total FROM events
```

Workers are useful for background maintenance tasks that should run on a fixed cadence, independent of the main workload loop. For example: lease reapers, stats refreshers, cache warmers, or periodic cleanup jobs.

### Rate

The `rate` field specifies how many times the query executes per interval, using the format `times/interval`:

| Example | Meaning |
|---|---|
| `1/10s` | Once every 10 seconds |
| `3/1m` | 3 times every minute / once every 20 seconds |
| `5/1m30s` | 5 times every minute and a half / once every 18 seconds |
| `2/1s` / `1/500ms` | 2 times every second |

The interval uses Go duration syntax (`s`, `ms`, `m`, `h`). Executions are evenly spaced: `3/1m` fires every 20 seconds, not 3 times at the start of each minute.

> [!NOTE]
> The `rate` property can achieve the same interval in a number of ways (e.g. `2/1s` and `1/500ms` both result in a worker that fires twice every second), so use whichever expresses your intent the best and is easiest to read.

### Behaviour

- Each worker runs in its own goroutine with its own environment, so workers are safe to use with `ref_*` functions and prepared statements.
- Worker query results flow into the same stats and metrics pipeline as `run` queries, so they appear in progress output, the summary table, Prometheus metrics, and `expectations`.
- Workers support all the same fields as regular queries: `type`, `args`, `prepared`, `row`, etc.
- In staged mode, workers run for the entire duration across all stages (not restarted per stage).
- Workers respect context cancellation and stop when the workload finishes or is interrupted.

See [`_examples/workers/`](https://github.com/codingconcepts/edg/tree/main/_examples/workers) for a complete working example.
