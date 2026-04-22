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
| `stage` | Generate data to files instead of a database |
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
| `--retries` | | `0` | Number of transaction retry attempts on error (run and all commands) |
| `--errors` | | `false` | Print worker errors to stderr (run and all commands) |
| `--print-interval` | | `1s` | Progress reporting interval (run and all commands) |

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

### Stage

The `stage` command generates data to files instead of executing against a database. It processes all config sections (`up`, `seed`, `deseed`, `down`) and writes the results in your chosen format. No database connection or `--url` is required.

```sh
edg stage \
--config _examples/output/config.yaml \
--format sql \
--output-dir ./out
```

#### Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--format` | `-f` | `sql` | Output format: `sql`, `json`, `csv`, `parquet`, or `stdout` |
| `--output-dir` | `-o` | `.` | Directory for output files (created if it doesn't exist) |

Global flags `--config`, `--driver`, and `--rng-seed` also apply. The `--driver` flag controls SQL value formatting (quote style, hex literals, etc.) even though no database connection is made. Use `--rng-seed` to produce deterministic, reproducible output across runs.

#### Formats

##### SQL

One file per config section containing executable SQL statements.

```sh
edg stage --config config.yaml --format sql -o ./out
```

| File | Contents |
|---|---|
| `up.sql` | DDL statements (`CREATE TABLE`, etc.) |
| `seed.sql` | One `INSERT` statement per generated row |
| `deseed.sql` | DML cleanup statements (`DELETE`, `TRUNCATE`) |
| `down.sql` | DDL teardown statements (`DROP TABLE`, etc.) |

**`up.sql`**

```sql
CREATE TABLE IF NOT EXISTS customer (
  id INT PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS purchase_order (
  id INT PRIMARY KEY,
  customer_id INT NOT NULL,
  amount DECIMAL(10,2) NOT NULL,
  status TEXT NOT NULL
);
```

**`seed.sql`** - query placeholders (`$1`, `$2`, etc.) are resolved with generated values inline:

```sql
INSERT INTO customer (id, name, email) VALUES (1, 'Jessica Hills', 'jonathonmarquardt@wilkinson.biz');
INSERT INTO customer (id, name, email) VALUES (2, 'Cedrick Saunders', 'maximelucas@bergstrom.com');
INSERT INTO customer (id, name, email) VALUES (3, 'Jose Watkins', 'brockhoward@walters.net');
...
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (1, 7, 199.19, 'shipped');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (2, 6, 140.75, 'pending');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (3, 3, 138.94, 'delivered');
...
```

**`deseed.sql`**

```sql
DELETE FROM purchase_order;
DELETE FROM customer;
```

**`down.sql`**

```sql
DROP TABLE IF EXISTS purchase_order;
DROP TABLE IF EXISTS customer;
```

##### JSON

One file per config section, containing an object keyed by query name. Each key maps to an array of row objects with column names as keys.

```sh
edg stage --config config.yaml --format json -o ./out
```

| File | Contents |
|---|---|
| `seed.json` | All seed query results grouped by query name |

> [!NOTE]
> Sections that only contain DDL or arg-less DML (up, deseed, down) produce no JSON file since there is no row data to write.

```json
{
  "populate_customer": [
    {
      "id": 1,
      "name": "Jessica Hills",
      "email": "jonathonmarquardt@wilkinson.biz"
    },
    {
      "id": 2,
      "name": "Cedrick Saunders",
      "email": "maximelucas@bergstrom.com"
    },
    ...
  ],
  "populate_order": [
    {
      "id": 1,
      "customer_id": 7,
      "amount": 199.19,
      "status": "shipped"
    },
    {
      "id": 2,
      "customer_id": 6,
      "amount": 140.75,
      "status": "pending"
    },
    ...
  ]
}
```

##### CSV

One file per data-generating query, named `{section}_{query}.csv`. Each file includes a header row derived from column names, followed by data rows.

```sh
edg stage --config config.yaml --format csv -o ./out
```

| File | Contents |
|---|---|
| `seed_populate_customer.csv` | Header + one row per customer |
| `seed_populate_order.csv` | Header + one row per order |

> [!NOTE]
> Queries without args (DDL, plain DML) are skipped since there is no row data to write.

**`seed_populate_customer.csv`**

```csv
id,name,email
1,Jessica Hills,jonathonmarquardt@wilkinson.biz
2,Cedrick Saunders,maximelucas@bergstrom.com
3,Jose Watkins,brockhoward@walters.net
4,Reginald Larson,clarissahart@baker.biz
...
```

**`seed_populate_order.csv`**

```csv
id,customer_id,amount,status
1,7,199.19,shipped
2,6,140.75,pending
3,3,138.94,delivered
4,5,371.19,pending
...
```

##### Parquet

One file per data-generating query, named `{section}_{query}.parquet`. All columns are stored as optional byte arrays (strings), making the files compatible with any Parquet reader.

```sh
edg stage --config config.yaml --format parquet -o ./out
```

| File | Contents |
|---|---|
| `seed_populate_customer.parquet` | Customer data (10 rows) |
| `seed_populate_order.parquet` | Order data (30 rows) |

Parquet files can be inspected with tools like DuckDB, `parquet-tools`, or pandas:

```sh
# DuckDB
duckdb -c "SELECT * FROM './out/seed_populate_customer.parquet' LIMIT 5"

# Python / pandas
python3 -c "import pandas; print(pandas.read_parquet('./out/seed_populate_customer.parquet').head())"
```

##### stdout

Streams SQL statements directly to standard output as they are generated, with no files written. Log output is suppressed so only SQL reaches stdout, making it safe for piping.

```sh
edg stage --config config.yaml --format stdout
```

Output is identical to the SQL format but printed to the console instead of written to files. DDL statements (up/down) and DML statements (seed/deseed) are written as-is; data-generating queries have their placeholders resolved with generated values inline.

```sh
# Pipe directly into a database
edg stage --config config.yaml --format stdout | psql mydb

# Preview the first few statements
edg stage --config config.yaml --format stdout | head -20
```

> [!NOTE]
> The `--output-dir` flag is ignored when using stdout format.

#### Column naming

Column names in the output are determined by the following priority:

1. **Named args** - if the query uses named args (`id: seq_global("customer_id")`), column names come from the arg names
2. **INSERT column list** - if the query SQL contains `INSERT INTO table (col1, col2, ...)`, columns are extracted from the parenthesized list
3. **Fallback** - positional names `col_1`, `col_2`, etc.

Named args are recommended for the clearest output. See [Configuration > Named Args]({{< relref "configuration" >}}#named-args) for details.

#### Referential integrity

The `stage` command preserves referential integrity across queries. Data generated by earlier queries is stored in memory and made available to subsequent queries via `ref_rand`, `ref_each`, `seq_rand`, and other reference functions; exactly as it would be during normal database execution.

For example, given a config with two seed queries:

```yaml
seq:
  - name: customer_id
    start: 1
    step: 1

seed:
  - name: populate_customer
    type: exec_batch
    count: 10
    args:
      id: seq_global("customer_id")
      name: gen('name')
    query: INSERT INTO customer (id, name) ...

  - name: populate_order
    type: exec_batch
    count: 30
    args:
      id: seq(1, 1)
      customer_id: seq_rand("customer_id")
    query: INSERT INTO purchase_order (id, customer_id) ...
```

The `customer_id` column in `populate_order` will only contain values from the `customer_id` global sequence (1–10), preserving the foreign key relationship in all output formats.

#### Batch query expansion

Queries using `exec_batch` or `query_batch` are expanded into individual rows. The batch CSV-joining logic used for database execution is bypassed; instead, each row's expressions are evaluated independently, producing clean per-row data in all output formats. The `count` and `size` fields control how many rows are generated.

#### Example

A complete working example is available in [`_examples/output/`](https://github.com/codingconcepts/edg/tree/main/_examples/output), including a config file and pre-generated output in all formats. To regenerate:

```sh
edg stage --config _examples/output/config.yaml -f sql -o _examples/output/sql
edg stage --config _examples/output/config.yaml -f json -o _examples/output/json
edg stage --config _examples/output/config.yaml -f csv -o _examples/output/csv
edg stage --config _examples/output/config.yaml -f parquet -o _examples/output/parquet
edg stage --config _examples/output/config.yaml -f stdout
```

### Workload

The `workload` command runs a built-in workload without needing a config file. Fifteen benchmarks are embedded in the binary. Each workload supports all six lifecycle commands (`up`, `seed`, `run`, `deseed`, `down`, `all`) and selects the correct config for the `--driver` automatically.

Built-in workloads:

| Workload | Description |
|---|---|
| `bank` | Bank account operations for contention and correctness testing |
| `ch-benchmark` | CH-benCHmark mixed OLTP+OLAP (TPC-C transactions + TPC-H analytical queries) |
| `kv` | Simple key-value read/write benchmark |
| `movr` | Vehicle-sharing application with rides, vehicles, and users |
| `seats` | Airline reservation system with flight booking contention |
| `sysbench-insert` | Sysbench `oltp_insert` pure insert micro-benchmark |
| `sysbench-point-select` | Sysbench `oltp_point_select` pure read micro-benchmark |
| `sysbench-read-write` | Sysbench `oltp_read_write` mixed read-write micro-benchmark |
| `sysbench-update-index` | Sysbench `oltp_update_index` indexed column update micro-benchmark |
| `tatp` | Telecom Application Transaction Processing (80% reads, 20% writes) |
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
| `spanner` | Google Cloud Spanner variant (GoogleSQL) |

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

- **`up`** `CREATE TABLE` statements derived from the database's own DDL (CockroachDB's `SHOW CREATE TABLE`, MySQL's `SHOW CREATE TABLE`, Oracle's `DBMS_METADATA.GET_DDL`, reconstructed from `sys` catalog views for MSSQL, or reconstructed from `INFORMATION_SCHEMA` for Spanner).
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

**Google Cloud Spanner (GoogleSQL)**
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
