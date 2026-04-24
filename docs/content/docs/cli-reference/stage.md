---
title: Stage
weight: 4
---

# Stage

The `stage` command generates data to files instead of executing against a database. It processes all config sections (`up`, `seed`, `deseed`, `down`) and writes the results in your chosen format. No database connection or `--url` is required.

```sh
edg stage \
--config _examples/output/config.yaml \
--format sql \
--output-dir ./out
```

## Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--format` | `-f` | `sql` | Output format: `sql`, `json`, `csv`, `parquet`, or `stdout` |
| `--output-dir` | `-o` | `.` | Directory for output files (created if it doesn't exist) |

Global flags `--config`, `--driver`, and `--rng-seed` also apply. The `--driver` flag controls SQL value formatting (quote style, hex literals, etc.) even though no database connection is made. Use `--rng-seed` to produce deterministic, reproducible output across runs.

## Formats

### SQL

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

### JSON

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

### CSV

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

### Parquet

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

### stdout

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

## Column naming

Column names in the output are determined by the following priority:

1. **Named args** - if the query uses named args (`id: seq_global("customer_id")`), column names come from the arg names
2. **INSERT column list** - if the query SQL contains `INSERT INTO table (col1, col2, ...)`, columns are extracted from the parenthesized list
3. **Fallback** - positional names `col_1`, `col_2`, etc.

Named args are recommended for the clearest output. See [Configuration > Named Args]({{< relref "configuration" >}}#named-args) for details.

## Referential integrity

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

## Batch query expansion

Queries using `exec_batch` or `query_batch` are expanded into individual rows. The batch CSV-joining logic used for database execution is bypassed; instead, each row's expressions are evaluated independently, producing clean per-row data in all output formats. The `count` and `size` fields control how many rows are generated.

## Example

A complete working example is available in [`_examples/output/`](https://github.com/codingconcepts/edg/tree/main/_examples/output), including a config file and pre-generated output in all formats. To regenerate:

```sh
edg stage --config _examples/output/config.yaml -f sql -o _examples/output/sql
edg stage --config _examples/output/config.yaml -f json -o _examples/output/json
edg stage --config _examples/output/config.yaml -f csv -o _examples/output/csv
edg stage --config _examples/output/config.yaml -f parquet -o _examples/output/parquet
edg stage --config _examples/output/config.yaml -f stdout
```
