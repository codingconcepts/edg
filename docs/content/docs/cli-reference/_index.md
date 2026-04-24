---
title: CLI Reference
weight: 5
---

# CLI Reference

## Commands

| Command | Description |
|---|---|
| `edg <expression>` | Evaluate a single expression and print the result |
| `up` | Create schema (tables, indexes) |
| `seed` | Populate tables with initial data |
| `run` | Execute the benchmark workload |
| `deseed` | Delete seeded data (truncate tables) |
| `down` | Tear down schema (drop tables) |
| `all` | Run up, seed, run, deseed, and down in sequence |
| `stage` | Generate data to files instead of a database |
| `init` | Generate a starter config from an existing database schema |
| `repl` | Interactive expression evaluator |
| `workload <name> <command>` | Run a built-in workload without a config file |
| `validate config` | Validate a config file without connecting to a database |
| `validate license` | Validate a license key and print its details |

Running `edg` with an expression (no subcommand) evaluates it and prints the result. Bare words are treated as [gofakeit](https://github.com/brianvoe/gofakeit) patterns, so `edg email` is equivalent to `edg "gen('email')"`. For expressions with parentheses or special characters, quote the argument.

A typical workflow runs the commands in order: `up` -> `seed` -> `run` -> `deseed` -> `down`. The `all` command runs this entire sequence in a single invocation.

## Flags

<div class="cli-flags">

| Flag / Env Var | Short | Default | Description |
|---|---|---|---|
| `--url`<br>`EDG_URL` | | | Database connection URL |
| `--config`<br>`EDG_CONFIG` | | | Path to the workload YAML config file (required for database commands, optional for `repl`) |
| `--driver`<br>`EDG_DRIVER` | | `pgx` | database/sql driver name (`pgx`, `dsql`, `oracle`, `mysql`, `mssql`, or `spanner`) |
| `--rng-seed`<br>`EDG_RNG_SEED` | | | PRNG seed for deterministic output (useful for regression testing) |
| `--duration` | `-d` | `1m` | Benchmark duration (run and all commands) |
| `--workers` | `-w` | `1` | Number of concurrent workers (run and all commands) |
| `--license`<br>`EDG_LICENSE` | | | License key for enterprise drivers (see [Licensing](/docs/licensing/)) |
| `--retries`<br>`EDG_RETRIES` | | `0` | Number of transaction retry attempts on error (run and all commands). Uses exponential backoff (1ms, 2ms, 4ms, ...). Only applies to transactions, not standalone queries. See [Retries](#retries) for details. |
| `--errors`<br>`EDG_ERRORS` | | `false` | Print worker errors to stderr (run and all commands). See [Error Output](#error-output) for details. |
| `--print-interval` | | `1s` | Progress reporting interval (run and all commands) |
| `--metrics-addr`<br>`EDG_METRICS_ADDR` | | | Address for Prometheus metrics endpoint (e.g. `:9090`). See [Observability]({{< relref "observability" >}}) for details. |
| `--pool-size`<br>`EDG_POOL_SIZE` | | `0` | Maximum number of open database connections. `0` uses the driver default (typically unlimited). |
| `--warmup-duration` | | `0` | Warmup period before collecting metrics (e.g. `10s`). Workers run during warmup but results are discarded. See [Warmup](#warmup) for details. |

</div>

Every flag with an env var listed above can be set via the environment instead of the command line. Flags take precedence over environment variables, which take precedence over defaults.

```sh
export EDG_URL="postgres://root@localhost:26257?sslmode=disable"
export EDG_DRIVER="pgx"
export EDG_CONFIG="workload.yaml"

# No need to pass --url, --driver, or --config:
edg run -w 10 -d 5m

# Flags override env vars when both are set:
edg run -w 10 -d 5m --driver mysql --url "user:pass@tcp(localhost:3306)/db"
```

## Example

### Database

Run each lifecycle command individually against a database, or use `all` to run the entire sequence in one invocation.

```sh
edg up \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg seed \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg run \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 100 \
-d 1m

edg deseed \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg down \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

Or use `all` to run the entire workflow in one command:

```sh
edg all \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 100 \
-d 5m
```

### Aurora DSQL

The `dsql` driver uses AWS IAM authentication instead of a username and password. Pass the cluster endpoint as the `--url` value:

```sh
edg all \
--driver dsql \
--config workload.yaml \
--url "clusterid.dsql.us-east-1.on.aws" \
-w 10 \
-d 5m
```

AWS credentials are resolved from the standard chain (environment variables, `~/.aws/credentials`, IAM role, etc.). The region is parsed from the cluster endpoint automatically. Auth tokens are refreshed on every new connection, so long-running workloads work without interruption.

DSQL uses PostgreSQL-compatible SQL, so use `$1`, `$2` placeholders in your queries.

## Licensing

The `pgx` and `mysql` drivers are free to use. Enterprise drivers (`oracle`, `mssql`, `dsql`, `spanner`) require a license key passed via `--license` or `EDG_LICENSE`. The license is validated before connecting to the database. See the [Licensing](/docs/licensing/) page for full details.

## Validating Config

The `validate config` command parses a config file and checks it for errors without connecting to a database. It catches YAML syntax errors, invalid expressions, unknown function calls, duplicate query names, shadowed built-ins, and invalid query types.

```sh
edg validate config --config _examples/tpcc/crdb.yaml
```

```
config is valid
```

This is useful for catching mistakes before deploying a workload or as a CI check.

## Validating a License

The `validate license` command checks whether a license key is valid for a given driver and prints the license details.

```sh
edg validate license --driver oracle --license "your-license-key"
```

```
License info:
  ID:         acme-corp
  Email:      admin@acme.com
  Drivers:    [oracle mssql]
  Issued at:  2025-01-15
  Expires at: 2026-01-15
License is valid for driver "oracle".
```

If the driver doesn't require a license, the output tells you:

```sh
edg validate license --driver pgx --license "your-license-key"
```

```
License info:
  ID:         acme-corp
  Email:      admin@acme.com
  Drivers:    [oracle mssql]
  Issued at:  2025-01-15
  Expires at: 2026-01-15
Driver "pgx" does not require a license.
```

If the license is expired or doesn't cover the requested driver, you'll see an error:

```sh
edg validate license --driver dsql --license "your-license-key"
```

```
License info:
  ID:         acme-corp
  Email:      admin@acme.com
  Drivers:    [oracle mssql]
  Issued at:  2025-01-15
  Expires at: 2026-01-15
Error: license does not include driver "dsql" (licensed: [oracle mssql])
```

The `EDG_LICENSE` environment variable is also accepted:

```sh
export EDG_LICENSE="your-license-key"
edg validate license --driver oracle
```

## Retries

The `--retries` flag controls how many times a failed transaction is retried before the error is recorded. The default is `0` (no retries). Retries only apply to transactions (queries wrapped in a `transaction:` block), not standalone queries.

When a transaction fails, edg waits with exponential backoff before retrying:

| Attempt | Backoff |
|---|---|
| 1st retry | 2ms |
| 2nd retry | 4ms |
| 3rd retry | 8ms |
| 4th retry | 16ms |
| *n*th retry | 2^*n* ms |

If all retry attempts fail, the last error is recorded in the stats and the worker continues to the next iteration. Context cancellation (e.g. `Ctrl+C` or duration expiry) stops retries immediately.

```sh
edg run \
  --driver pgx \
  --config workload.yaml \
  --url ${DATABASE_URL} \
  --retries 3 \
  -w 10 \
  -d 5m
```

## Error Output

By default, individual query errors during the `run` phase are counted but not printed. The `--errors` flag prints each error to stderr as it occurs, which is useful for debugging:

```sh
edg run \
  --driver pgx \
  --config workload.yaml \
  --url ${DATABASE_URL} \
  --errors \
  -w 10 \
  -d 5m
```

```
2025/04/23 14:32:07 ERROR run error worker=3 error="running run query debit_source: pq: insufficient funds"
2025/04/23 14:32:07 ERROR run error worker=7 error="running run query debit_source: pq: insufficient funds"
```

Without `--errors`, the same failures still appear in the summary table's ERRORS column and count toward `error_rate` in expectations.

## Connection Pool

The `--pool-size` flag sets the maximum number of open database connections (`SetMaxOpenConns` and `SetMaxIdleConns`). The default `0` uses the driver's default, which is typically unlimited.

Setting pool size is useful for:

- **Simulating constrained environments** where the application has a fixed connection budget.
- **Preventing connection exhaustion** when running with many workers against a database with connection limits.
- **Isolating connection overhead** from query performance in benchmarks.

```sh
edg run \
  --driver pgx \
  --config workload.yaml \
  --url ${DATABASE_URL} \
  --pool-size 20 \
  -w 50 \
  -d 5m
```

In this example, 50 workers share 20 connections. Workers that can't acquire a connection will block until one becomes available.

## Warmup

The `--warmup-duration` flag runs workers for a specified period before collecting metrics. During warmup, query results are discarded. They don't appear in progress output, the summary, Prometheus metrics, or expectations.

This produces cleaner benchmark results by allowing the database to warm its caches, JIT-compile query plans, and reach a steady state before measurement begins.

```sh
edg run \
  --driver pgx \
  --config workload.yaml \
  --url ${DATABASE_URL} \
  --warmup-duration 30s \
  -w 10 \
  -d 5m
```

In this example, workers run for 30 seconds of warmup (discarded), then 5 minutes of measured execution. The total wall-clock time is 5m30s.

When using stages, warmup applies before the first stage begins collecting metrics.

## Run Behaviour

### Workers and Initialisation

Each worker gets its own isolated environment. The `init` section runs once, and its results are cloned to each worker so that functions like `ref_rand` and `ref_diff` don't interfere across workers. Per-worker state includes sequence counters (`seq`), permanent row picks (`ref_perm`), and NURand constants.

### Stages

When a config file includes a `stages` section, the `-w` and `-d` flags are ignored. Instead, each stage defines its own worker count and duration, and stages run sequentially. See [Configuration > Stages]({{< relref "configuration#stages" >}}) for details.

```sh
edg run \
--driver pgx \
--config _examples/stages/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

### Error Handling

Query errors during `run` are **non-fatal**. The worker logs the error and increments an error counter but continues to the next iteration. This lets you observe error rates without aborting the benchmark. Errors in other sections (`up`, `seed`, `deseed`, `down`, `init`) are fatal and stop execution immediately.

### Interrupting with Ctrl+C

Pressing `Ctrl+C` during `run` or `all` cancels the workload gracefully. Workers finish their current iteration and stop. When using `all`, the cleanup phases (`deseed` and `down`) still run after interruption, using a fresh context.

### Output

During the run, progress is printed at the `--print-interval` (default: every second):

```
59s / 1m0s
QUERY          COUNT  ERRORS  AVG      p50      p95      p99      QPS
check_balance  3674   0       2.631ms  2.367ms  4.154ms  6.252ms  62.3
credit_target  3769   0       1.68ms   1.495ms  2.624ms  3.911ms  63.9
debit_source   3769   0       2.376ms  2.13ms   3.722ms  5.288ms  63.9
read_source    3770   0       2.047ms  1.803ms  3.254ms  5.052ms  63.9
read_target    3769   0       2.839ms  2.579ms  4.486ms  6.446ms  63.9

TRANSACTION    COMMITS  ROLLBACKS  ERRORS  AVG       p50       p95       p99       TPS
make_transfer  3769     0          0       13.053ms  12.424ms  18.498ms  26.074ms  63.9
```

After all workers complete, a final summary is printed:

```
summary
Duration:  1m0.004s
Workers:   1

QUERY          COUNT  ERRORS  AVG      p50      p95      p99      QPS
check_balance  3749   0       2.628ms  2.362ms  4.14ms   6.249ms  62.5
credit_target  3828   0       1.681ms  1.497ms  2.624ms  3.911ms  63.8
debit_source   3828   1       2.381ms  2.13ms   3.724ms  5.338ms  63.8
read_source    3829   0       2.046ms  1.802ms  3.25ms   5.052ms  63.8
read_target    3829   0       2.843ms  2.583ms  4.485ms  6.446ms  63.8

TRANSACTION    COMMITS  ROLLBACKS  ERRORS  AVG       p50       p95       p99       TPS
make_transfer  3828     0          1       13.063ms  12.438ms  18.498ms  26.652ms  63.8

Transactions:  19063
Errors:        1
tpm:           19061.6
```

| Metric | Description |
|---|---|
| COUNT | Total successful query executions |
| ERRORS | Total failed query executions |
| AVG | Mean execution time per query |
| p50 | Median latency (50th percentile) |
| p95 | 95th percentile latency |
| p99 | 99th percentile latency |
| QPS | Queries per second (count / elapsed seconds) |
| tpm | Transactions per minute across all queries |

### Expectations

When the config file includes an `expectations` section, results are printed after the summary and the exit code reflects whether all expectations passed:

```
expectations
  PASS  error_rate < 1
  PASS  check_balance.p99 < 100
  FAIL  tpm > 5000

1 expectation(s) failed
```

If any expectation fails, edg exits with status code 1. When using `all`, teardown (`deseed` and `down`) still runs before the non-zero exit.

See [Configuration > Expectations]({{< relref "../configuration/expectations" >}}) for the full list of available metrics and expression syntax.
