# Stage

The `go run ./cmd/edg stage` command generates data to files instead of executing against a database. It processes all config sections (up, seed, deseed, down) and writes the results in your chosen format. No database connection is required.

## Usage

```sh
go run ./cmd/edg stage --config <config.yaml> [--format <format>] [--output-dir <dir>]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `-f, --format` | `sql` | Output format: `sql`, `json`, `csv`, `parquet`, or `stdout` |
| `-o, --output-dir` | `.` | Directory for output files (created if it doesn't exist) |
| `--config` | | Workload YAML config file (required) |
| `--driver` | `pgx` | Controls SQL value formatting (quote style, hex literals) |
| `--rng-seed` | | PRNG seed for deterministic, reproducible output |

## Formats

### SQL

One file per config section. DDL statements (up/down) and DML statements (seed/deseed) are written as-is. Data-generating queries are expanded into individual `INSERT` statements with values inline.

```sh
go run ./cmd/edg stage --config _examples/output/config.yaml --format sql -o _examples/output/sql
```

**Files produced:**

| File | Contents |
|---|---|
| `up.sql` | `CREATE TABLE` statements |
| `seed.sql` | `INSERT INTO` statements (one per row) |
| `deseed.sql` | `DELETE` statements |
| `down.sql` | `DROP TABLE` statements |

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

**`seed.sql`** (10 customers + 30 orders = 40 statements)

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

### JSON

One file per config section, containing an object keyed by query name. Each key maps to an array of row objects. Only data-generating queries (those with args) are included; DDL/DML without args is skipped.

```sh
go run ./cmd/edg stage --config _examples/output/config.yaml --format json -o _examples/output/json
```

**Files produced:**

| File | Contents |
|---|---|
| `seed.json` | All seed query results grouped by query name |

> [!NOTE]
> Sections that only contain DDL or arg-less DML (up, deseed, down) produce no JSON file since there is no row data to write.

**`seed.json`**

```json
{
  "populate_customer": [
    {
      "email": "jonathonmarquardt@wilkinson.biz",
      "id": 1,
      "name": "Jessica Hills"
    },
    {
      "email": "maximelucas@bergstrom.com",
      "id": 2,
      "name": "Cedrick Saunders"
    },
    ...
  ],
  "populate_order": [
    {
      "amount": 199.19,
      "customer_id": 7,
      "id": 1,
      "status": "shipped"
    },
    {
      "amount": 140.75,
      "customer_id": 6,
      "id": 2,
      "status": "pending"
    },
    ...
  ]
}
```

### CSV

One file per data-generating query, named `{section}_{query}.csv`. Each file includes a header row followed by data rows. DDL/DML queries without args are skipped.

```sh
go run ./cmd/edg stage --config _examples/output/config.yaml --format csv -o _examples/output/csv
```

**Files produced:**

| File | Rows | Description |
|---|---|---|
| `seed_populate_customer.csv` | 10 | Customer data |
| `seed_populate_order.csv` | 30 | Order data with FK references |

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

### Parquet

One file per data-generating query, named `{section}_{query}.parquet`. All columns are stored as optional byte arrays (strings). DDL/DML queries without args are skipped.

```sh
go run ./cmd/edg stage --config _examples/output/config.yaml --format parquet -o _examples/output/parquet
```

**Files produced:**

| File | Rows | Description |
|---|---|---|
| `seed_populate_customer.parquet` | 10 | Customer data |
| `seed_populate_order.parquet` | 30 | Order data with FK references |

Parquet files can be inspected with tools like `parquet-tools`, DuckDB, or pandas:

```sh
duckdb -c "SELECT * FROM '_examples/output/parquet/seed_populate_customer.parquet' LIMIT 5"
```

### stdout

Streams SQL statements directly to standard output as they are generated, with no files written. Useful for piping into a database client or other tools.

```sh
go run ./cmd/edg stage --config _examples/output/config.yaml --format stdout
```

Output is identical to the SQL format but printed to the console instead of written to files. DDL statements (up/down) and DML statements (seed/deseed) are written as-is; data-generating queries are expanded into individual `INSERT` statements.

```sh
# Pipe directly into a database
go run ./cmd/edg stage --config _examples/output/config.yaml --format stdout | psql mydb

# Preview the first few statements
go run ./cmd/edg stage --config _examples/output/config.yaml --format stdout | head -20
```

> [!NOTE]
> The `--output-dir` flag is ignored when using stdout format.

## Column naming

Column names in the output are determined by the following priority:

1. **Named args** -- if the query uses named args (e.g. `id: seq_global("customer_id")`), column names come from the arg names
2. **INSERT column list** -- if the query SQL contains `INSERT INTO table (col1, col2, ...)`, columns are extracted from the parenthesized list
3. **Fallback** -- positional names `col_1`, `col_2`, etc.

## Referential integrity

The `stage` command preserves referential integrity across queries. Data generated by earlier queries is stored in memory and made available to subsequent queries via `ref_rand`, `ref_each`, `seq_rand`, and other reference functions.

For example, `populate_order` references `customer_id` values from the `customer_id` global sequence using `seq_rand("customer_id")`. The staged output will contain only valid customer IDs (1-10) in the order rows.

## Batch query expansion

Queries using `exec_batch` are expanded into individual rows in the output. The multi-row `VALUES` clause used for database execution (via `__values__`) is bypassed; instead, each row's expressions are evaluated independently, producing clean per-row data suitable for all output formats.

## Example config

The [config.yaml](config.yaml) in this directory demonstrates a typical setup:

```yaml
globals:
  customers: 10

seq:
  - name: customer_id
    start: 1
    step: 1
  - name: order_id
    start: 1
    step: 1

seed:
  - name: populate_customer
    type: exec_batch
    count: customers
    args:
      id: seq_global("customer_id")
      name: gen('name')
      email: gen('email')
    query: ...

  - name: populate_order
    type: exec_batch
    count: customers * 3
    args:
      id: seq_global("order_id")
      customer_id: seq_rand("customer_id")
      amount: uniform_f(9.99, 499.99, 2)
      status: set_rand(['pending', 'shipped', 'delivered'], [])
    query: ...
```
