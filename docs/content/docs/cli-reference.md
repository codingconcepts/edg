---
title: CLI Reference
weight: 3
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
| `repl` | Interactive expression evaluator |
| `validate` | Validate a config file without connecting to a database |

Running `edg` with an expression (no subcommand) evaluates it and prints the result. Bare words are treated as [gofakeit](https://github.com/brianvoe/gofakeit) patterns, so `edg email` is equivalent to `edg "gen('email')"`. For expressions with parentheses or special characters, quote the argument.

A typical workflow runs the commands in order: `up` -> `seed` -> `run` -> `deseed` -> `down`. The `all` command runs this entire sequence in a single invocation.

## Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--url` | | | Database connection URL (or set `URL` env var) |
| `--config` | | | Path to the workload YAML config file (required for database commands, optional for `repl`) |
| `--driver` | | `pgx` | database/sql driver name (`pgx`, `dsql`, `oracle`, `mysql`, or `mssql`) |
| `--rng-seed` | | | PRNG seed for deterministic output (useful for regression testing) |
| `--duration` | `-d` | `1m` | Benchmark duration (run and all commands) |
| `--workers` | `-w` | `1` | Number of concurrent workers (run and all commands) |
| `--print-interval` | | `1s` | Progress reporting interval (run and all commands) |

## Example

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

## Validating Config

The `validate` command parses a config file and checks it for errors without connecting to a database. It catches YAML syntax errors, invalid expressions, unknown function calls, duplicate query names, shadowed built-ins, and invalid query types.

```sh
edg validate --config _examples/tpcc/crdb.yaml
```

```
config is valid
```

This is useful for catching mistakes before deploying a workload or as a CI check.

## Run Behaviour

### Workers and Initialisation

Each worker gets its own isolated environment. The `init` section runs once, and its results are cloned to each worker so that functions like `ref_rand` and `ref_diff` don't interfere across workers. Per-worker state includes sequence counters (`seq`), permanent row picks (`ref_perm`), and NURand constants.

### Stages

When a config file includes a `stages` section, the `-w` and `-d` flags are ignored. Instead, each stage defines its own worker count and duration, and stages run sequentially. See [Configuration > Stages](/docs/configuration/#stages) for details.

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

See [Configuration > Expectations]({{< relref "configuration" >}}#expectations) for the full list of available metrics and expression syntax.
