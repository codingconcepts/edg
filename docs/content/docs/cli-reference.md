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
| `init` | Generate a starter config from an existing database schema |
| `repl` | Interactive expression evaluator |
| `workload <name> <command>` | Run a built-in workload without a config file |
| `validate config` | Validate a config file without connecting to a database |
| `validate license` | Validate a license key and print its details |

Running `edg` with an expression (no subcommand) evaluates it and prints the result. Bare words are treated as [gofakeit](https://github.com/brianvoe/gofakeit) patterns, so `edg email` is equivalent to `edg "gen('email')"`. For expressions with parentheses or special characters, quote the argument.

A typical workflow runs the commands in order: `up` -> `seed` -> `run` -> `deseed` -> `down`. The `all` command runs this entire sequence in a single invocation.

## Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--url` | | | Database connection URL (or set `URL` env var) |
| `--config` | | | Path to the workload YAML config file (required for database commands, optional for `repl`) |
| `--driver` | | `pgx` | database/sql driver name (`pgx`, `dsql`, `oracle`, `mysql`, `mssql`, or `spanner`) |
| `--rng-seed` | | | PRNG seed for deterministic output (useful for regression testing) |
| `--duration` | `-d` | `1m` | Benchmark duration (run and all commands) |
| `--workers` | `-w` | `1` | Number of concurrent workers (run and all commands) |
| `--license` | | | License key for enterprise drivers, or set `EDG_LICENSE` env var (see [Licensing](/docs/licensing/)) |
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

### Workload

The `workload` command runs a built-in workload without needing a config file. Eight benchmarks are embedded in the binary. Each workload supports all six lifecycle commands (`up`, `seed`, `run`, `deseed`, `down`, `all`) and selects the correct config for the `--driver` automatically.

Built-in workloads:

| Workload | Description |
|---|---|
| `bank` | Bank account operations for contention and correctness testing |
| `kv` | Simple key-value read/write benchmark |
| `movr` | Vehicle-sharing application with rides, vehicles, and users |
| `tpcc` | Full TPC-C benchmark with all 5 transaction profiles |
| `tpch` | TPC-H decision-support benchmark with analytical queries |
| `ttlbench` | Insert throughput under row-level TTL garbage collection pressure |
| `ttllogger` | Structured log ingestion with TTL-based expiry |
| `ycsb` | Yahoo! Cloud Serving Benchmark with configurable workload profiles |

Supported drivers:

| Driver | Config used |
|---|---|
| `pgx`, `dsql` | CockroachDB/PostgreSQL variant |
| `mysql` | MySQL variant |
| `oracle` | Oracle variant |
| `mssql` | SQL Server variant |
| `spanner` | Google Cloud Spanner variant |

```sh
# Create the bank schema
edg workload bank up \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable"

# Seed and run in one step
edg workload bank all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5m

# Run KV benchmark
edg workload kv all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 20 \
-d 5m

# Run MovR ride-sharing workload
edg workload movr all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5m

# Run TPC-C against MySQL
edg workload tpcc all \
--driver mysql \
--url "root:password@tcp(localhost:3306)/tpcc?parseTime=true" \
-w 50 \
-d 10m

# Run TPC-H analytical queries
edg workload tpch all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 4 \
-d 5m

# TTL benchmarks (CockroachDB native TTL)
edg workload ttlbench all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5m

edg workload ttllogger all \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 5m

# Run YCSB against MSSQL
edg workload ycsb all \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=ycsb&encrypt=disable" \
-w 20 \
-d 5m

# Run KV benchmark against Spanner
edg workload kv all \
--driver spanner \
--url "projects/my-project/instances/my-instance/databases/my-db" \
--license "$EDG_LICENSE" \
-w 10 \
-d 5m
```

The `--config` flag is not required (and ignored) for workload commands. All other flags (`--duration`, `--workers`, `--print-interval`, `--rng-seed`, `--license`) work as normal.

### Init

The `init` command connects to an existing database, inspects its schema, and prints a complete config to stdout. The output includes `globals`, `up`, `seed`, `deseed`, and `down` sections ready to use with the other commands.

| Flag | Required | Description |
|---|---|---|
| `--schema` | Yes (or `--database`) | Schema or database name to introspect (e.g. `public`, `defaultdb`, `dbo`, `SYSTEM`) |
| `--database` | Yes (or `--schema`) | Alias for `--schema` |
| `--driver` | Yes | Database driver (`pgx`, `mysql`, `mssql`, `oracle`, `dsql`, `spanner`) |
| `--url` | Yes | Connection URL for the source database |

```sh
edg init \
--driver pgx \
--url "postgres://root@localhost:26257/defaultdb?sslmode=disable" \
--schema public > workload.yaml
```

The generated config:

- **`up`** `CREATE TABLE` statements derived from the database's own DDL (CockroachDB's `SHOW CREATE TABLE`, MySQL's `SHOW CREATE TABLE`, Oracle's `DBMS_METADATA.GET_DDL`, or reconstructed from `sys` catalog views for MSSQL).
- **`seed`** One `INSERT` per table with an expression for each non-generated column. Columns with auto-increment, identity, or default functions like `gen_random_uuid()` and `now()` are skipped. Expressions are chosen by data type (e.g. `uuid_v4()` for UUID, `uniform(1, 1000)` for INT, `gen('sentence:3')` for VARCHAR). `CHECK BETWEEN` constraints are detected and used to narrow the range.
- **`deseed`** `TRUNCATE` (pgx, oracle) or `DELETE FROM` (mysql, mssql, spanner) in reverse dependency order.
- **`down`** `DROP TABLE` in reverse dependency order.
- Tables are topologically sorted so parent tables are created before children.

> [!WARNING]
> The output is a starting point. You'll typically want to refine the seed expressions to produce more realistic data, add a `run` section, and adjust `globals.rows`.

#### Examples by driver

**CockroachDB / PostgreSQL (pgx)**
```sh
edg init \
--driver pgx \
--url "postgres://root@localhost:26257/dbname?sslmode=disable" \
--schema public > workload.yaml
```

**Aurora DSQL**
```sh
edg init \
--driver dsql \
--url "clusterid.dsql.us-east-1.on.aws" \
--schema public > workload.yaml
```

**MySQL**
```sh
edg init \
--driver mysql \
--url "root:password@tcp(localhost:3306)/dbname?parseTime=true" \
--database dbname > workload.yaml
```

**MSSQL**
```sh
edg init \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=master&encrypt=disable" \
--schema dbo > workload.yaml
```

**Oracle**
```sh
edg init \
--driver oracle \
--url "oracle://system:password@localhost:1521/dbname" \
--schema SYSTEM > workload.yaml
```

**Google Cloud Spanner**
```sh
edg init \
--driver spanner \
--url "projects/my-project/instances/my-instance/databases/my-db" \
--license "$EDG_LICENSE" \
--schema "" > workload.yaml
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

The `pgx` and `mysql` drivers are free to use. Enterprise drivers (`oracle`, `mssql`, `dsql`, `spanner`) require a license key passed via `--license` or the `EDG_LICENSE` environment variable. The license is validated before connecting to the database. See the [Licensing](/docs/licensing/) page for full details.

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
