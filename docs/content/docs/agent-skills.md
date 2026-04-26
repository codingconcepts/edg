---
title: Agent Skills
weight: 9
---

# Agent Skills

edg ships with a set of [Agent Skills](https://claude.ai/claude-code); slash commands that help you generate, validate, debug, and migrate workload configs without leaving your terminal.

## Setup

The skills live in `.claude/skills/` in the edg repository. Each skill is a directory containing a `SKILL.md` file. If you've cloned edg and are running Claude Code from the repo root, they're available automatically.

To use them in a separate project, copy the `.claude/skills/` directory into your project:

```sh
cp -r /path/to/edg/.claude/skills/ .claude/skills/
```

## Available Skills

### `/edg-config` - Generate a workload config

Generates a complete edg YAML config from a natural language description of your schema and workload. Includes database-specific patterns for all supported drivers.

**Example prompts:**

```
/edg-config

I have a users table and an orders table on CockroachDB.
Users have an email, name, and created_at timestamp.
Orders reference a user and have a total, status, and created_at.
I want a 70/30 read/write mix benchmarking order lookups and new order inserts.
Seed with 10k users and 50k orders.
```

```
/edg-config

MySQL e-commerce schema: products (name, category, price, stock),
customers (email, name), and a cart_items join table.
Write-heavy workload simulating add-to-cart with Zipfian product selection
(some products are much more popular). Use batch inserts for seeding.
```

```
/edg-config

Generate a sync config pair for CockroachDB (pgx) and MySQL.
Tables: users (id, email, name) and orders (id, user_id, amount, status).
1000 users and 5000 orders, batched 100 at a time.
I'll use edg sync run with --rng-seed for deterministic dual-write testing.
```

```
/edg-config

MongoDB workload: customers collection with email and name fields,
and an accounts collection referencing customers with a balance field.
Seed 500 customers and 500 accounts, then run random balance lookups.
```

```
/edg-config

Cassandra IoT schema: create a keyspace with a devices table (id UUID, name TEXT)
and a readings table (device_id UUID, ts TIMESTAMP, value DOUBLE).
Seed 100 devices and 10k readings, then run time-range queries.
```

**What it produces:**

- A full YAML config with `globals`, `up`, `seed`, `init`, `run`, `run_weights`, `workers`, `deseed`, and `down` sections
- Appropriate expressions for data generation (`gen()`, `uuid_v7()`, distributions)
- Reference data setup via `init` queries for use in `run` with `ref_rand()`, `ref_same()`, etc.
- Driver-specific SQL patterns (batch expansion, DDL safety, upsert syntax)
- **Sync config pairs** for cross-database consistency testing - matched schemas with explicit IDs, identical seed args, and driver-specific batch SQL for use with `edg sync run`

---

### `/edg-validate` - Validate and fix a config

Runs `edg validate` on a config file, interprets errors, and suggests fixes.

**Example prompts:**

```
/edg-validate

Check my config at workloads/bench.yaml for the pgx driver.
```

```
/edg-validate

Validate _examples/ecommerce/mysql.yaml and fix any issues you find.
```

**What it does:**

1. Detects the driver from the config content (or uses one you specify)
2. Runs `edg validate --driver <driver> --config <path>`
3. Explains each error in plain language
4. Suggests (or applies) fixes for common issues:
   - Unknown functions or typos in expressions
   - Missing `init` datasets referenced by `ref_*` calls
   - Wrong query type (`exec` vs `query`)
   - Missing `count`/`size` on batch operations

---

### `/edg-expression` - Compose and debug expressions

Helps you find the right edg expression for a use case, explains how functions work, and debugs syntax errors.

**Example prompts:**

```
/edg-expression

I need to generate a price between $1 and $500 with a log-normal distribution
skewed toward cheaper items, rounded to 2 decimal places.
```

```
/edg-expression

How do I make two columns mutually exclusive (either an email OR a phone
number, but never both?)
```

```
/edg-expression

What's the difference between ref_rand, ref_same, and ref_perm?
When would I use each one?
```

**What it covers:**

- All 60+ built-in functions with signatures and descriptions
- Distribution selection guidance (uniform vs normal vs zipf vs exponential)
- Reference data patterns (`ref_rand` vs `ref_same` vs `ref_perm` vs `ref_diff`)
- Dependent column patterns with `arg()`, `cond()`, and `bool()`
- The full expr-lang feature set (array functions, string ops, math, conditionals)
- Tips for debugging with `edg repl`

---

### `/edg-migrate` - Convert a config between drivers

Converts an edg config written for one database driver to another, handling all SQL dialect differences.

**Example prompts:**

```
/edg-migrate

Convert _examples/ecommerce/crdb.yaml from pgx to mysql.
```

```
/edg-migrate

I have a workload config for PostgreSQL and need it to work on SQL Server (mssql).
The config is at workloads/bench.yaml.
```

```
/edg-migrate

Convert my PostgreSQL workload at workloads/users.yaml to MongoDB.
```

**What it translates:**

| Concern | What changes |
|---|---|
| Batch expansion | `unnest(string_to_array(...))` -> `JSON_TABLE` / `OPENJSON` / `XMLTABLE` / `UNNEST(SPLIT(...))` (Spanner) |
| Cleanup | `TRUNCATE CASCADE` -> `DELETE FROM` / `CASCADE CONSTRAINTS PURGE` / `DELETE FROM ... WHERE TRUE` (Spanner) / `TRUNCATE` (Cassandra) / `{"delete": ...}` (MongoDB) |
| Column types | `UUID` -> `CHAR(36)` / `STRING(36)` / `TEXT` (Cassandra), `STRING` -> `VARCHAR(n)`, etc. |
| DDL safety | `IF NOT EXISTS` -> `IF OBJECT_ID(...)` / PL/SQL exception blocks / `CREATE TABLE IF NOT EXISTS` (Spanner) |
| Default values | `gen_random_uuid()` -> `UUID()` / `NEWID()` / `GENERATE_UUID()` / args-based |
| Pagination | `LIMIT/OFFSET` -> `FETCH NEXT ... ROWS ONLY` |
| Placeholders | `$1` -> `?` (MySQL, Cassandra) / `:1` (Oracle) / `@p1` (MSSQL, Spanner) / inlined JSON (MongoDB) |
| Primary key | inline `PRIMARY KEY` -> table-level `PRIMARY KEY (col)` (Spanner) |
| Query format | SQL -> BSON/JSON commands (MongoDB) / CQL (Cassandra) |
| Random ordering | `random()` -> `RAND()` / `NEWID()` / `DBMS_RANDOM.VALUE` / `TABLESAMPLE RESERVOIR` (Spanner) |
| Row generation | `generate_series` -> recursive CTE / `CONNECT BY` / `GENERATE_ARRAY` + `UNNEST` (Spanner) |
| Schema creation | `CREATE TABLE` -> `{"create": "collection"}` (MongoDB) / `CREATE KEYSPACE` + `CREATE TABLE` (Cassandra) |
| Upsert | `ON CONFLICT` -> `ON DUPLICATE KEY` / `MERGE INTO` / `INSERT OR UPDATE` (Spanner) |

Expression args (`gen()`, `ref_rand()`, `zipf()`, etc.) are driver-agnostic and remain unchanged.

## Tips

- **Combine skills**: Generate a config with `/edg-config`, then validate it with `/edg-validate`, then port it to another driver with `/edg-migrate`.
- **Use the REPL**: All expression skills recommend `edg repl` for interactive testing; no database connection needed.
- **Validate after migration**: Always run `edg validate --driver <driver> --config <path>` after generating or migrating a config to catch issues before hitting the database.
