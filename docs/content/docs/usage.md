---
title: Usage
weight: 2
---

# Usage

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

Running `edg` with an expression (no subcommand) evaluates it and prints the result. Bare words are treated as [gofakeit](https://github.com/brianvoe/gofakeit) patterns, so `edg email` is equivalent to `edg "gen('email')"`. For expressions with parentheses or special characters, quote the argument.

```
edg email
naomiroberts@robinson.net

edg firstname
Laura

edg lastname
Thompson

edg "uuid_v4()"
c8952841-6f5b-4743-a6de-2200415c2f03

edg "regex('[A-Z]{3}-[0-9]{4}')"
QVM-8314

edg "set_rand(['a','b','c'], [])"
b
```

A typical workflow runs the commands in order: `up` -> `seed` -> `run` -> `deseed` -> `down`. The `all` command runs this entire sequence in a single invocation.

## Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--url` | | | Database connection URL (or set `URL` env var) |
| `--config` | | `_examples/tpcc/crdb.yaml` | Path to the workload YAML config file |
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
