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
| `--driver` | | `pgx` | database/sql driver name (`pgx`, `oracle`, or `mysql`) |
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

### Error Handling

Query errors during `run` are **non-fatal**. The worker logs the error and increments an error counter but continues to the next iteration. This lets you observe error rates without aborting the benchmark. Errors in other sections (`up`, `seed`, `deseed`, `down`, `init`) are fatal and stop execution immediately.

### Interrupting with Ctrl+C

Pressing `Ctrl+C` during `run` or `all` cancels the workload gracefully. Workers finish their current iteration and stop. When using `all`, the cleanup phases (`deseed` and `down`) still run after interruption, using a fresh context.

### Output

During the run, progress is printed at the `--print-interval` (default: every second):

```
5s elapsed
QUERY       COUNT  ERRORS  AVG        p50        p95        p99        QPS
get_user    4820   0       1.037ms    0.998ms    1.842ms    2.451ms    964.0
```

After all workers complete, a final summary is printed:

```
summary
Duration:  1m0.001s
Workers:   10

QUERY       COUNT   ERRORS  AVG        p50        p95        p99        QPS
get_user    58241   0       1.028ms    0.991ms    1.804ms    2.389ms    970.7

Transactions:  58241
Errors:        0
tpm:           3494460.0
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
