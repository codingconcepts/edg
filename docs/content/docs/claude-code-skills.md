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

**What it produces:**

- A full YAML config with `globals`, `up`, `seed`, `init`, `run`, `run_weights`, `deseed`, and `down` sections
- Appropriate expressions for data generation (`gen()`, `uuid_v7()`, distributions)
- Reference data setup via `init` queries for use in `run` with `ref_rand()`, `ref_same()`, etc.
- Driver-specific SQL patterns (batch expansion, DDL safety, upsert syntax)

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

- All ~60 built-in functions with signatures and descriptions
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

**What it translates:**

| Concern | What changes |
|---|---|
| Column types | `UUID` → `CHAR(36)`, `STRING` → `VARCHAR(n)`, etc. |
| Default values | `gen_random_uuid()` → `UUID()` / `NEWID()` / args-based |
| DDL safety | `IF NOT EXISTS` → `IF OBJECT_ID(...)` / PL/SQL exception blocks |
| Row generation | `generate_series` → recursive CTE / `CONNECT BY` |
| Batch expansion | `unnest(string_to_array(...))` → `JSON_TABLE` / `OPENJSON` / `XMLTABLE` |
| Upsert | `ON CONFLICT` → `ON DUPLICATE KEY` / `MERGE INTO` |
| Pagination | `LIMIT/OFFSET` → `FETCH NEXT ... ROWS ONLY` |
| Random ordering | `random()` → `RAND()` / `NEWID()` / `DBMS_RANDOM.VALUE` |
| Cleanup | `TRUNCATE CASCADE` → `DELETE FROM` / `CASCADE CONSTRAINTS PURGE` |

Expression args (`gen()`, `ref_rand()`, `zipf()`, etc.) are driver-agnostic and remain unchanged.

## Tips

- **Combine skills**: Generate a config with `/edg-config`, then validate it with `/edg-validate`, then port it to another driver with `/edg-migrate`.
- **Use the REPL**: All expression skills recommend `edg repl` for interactive testing; no database connection needed.
- **Validate after migration**: Always run `edg validate --driver <driver> --config <path>` after generating or migrating a config to catch issues before hitting the database.
