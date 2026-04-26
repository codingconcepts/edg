---
title: Troubleshooting
weight: 11
---

# Troubleshooting

Common issues and how to resolve them.

## Config Errors

### "unknown function" or expression compile error

An arg expression references a function that doesn't exist. Common causes:

- **Typo in function name**: `gen('eamil')` instead of `gen('email')`. gofakeit patterns are validated at load time, so a typo produces a clear error.
- **Missing quotes**: `ref_rand(fetch_users).id` instead of `ref_rand('fetch_users').id`. Dataset names must be quoted strings.
- **Missing init query**: Using `ref_rand('fetch_users')` in `run` without a corresponding `init` query named `fetch_users`.

Run `edg validate config --config workload.yaml` to catch these before hitting the database.

### "duplicate query name"

Two queries in the same config have the same `name`. Names must be unique across all sections because they're used as dataset keys for `ref_*` functions and as metric labels.

### "row and args are mutually exclusive"

A query specifies both `row: template_name` and `args: [...]`. Use one or the other.

## Connection Errors

### "connecting to database: dial tcp ... connection refused"

The database isn't running or isn't reachable at the specified URL. Verify:

1. The database is running: `docker ps` or equivalent.
2. The URL is correct: check host, port, and protocol.
3. Network access: firewalls, Docker networking (`host.docker.internal` vs `localhost`).

### "connecting to database: pq: password authentication failed"

Credentials in the `--url` are wrong. For local development, check if your database requires a password (CockroachDB insecure mode doesn't; PostgreSQL usually does).

### "driver X requires a license"

You're using an enterprise driver (`oracle`, `mssql`, `dsql`, `spanner`) without a license. Set `--license` or export `EDG_LICENSE`. See [Licensing]({{< relref "licensing" >}}).

## Runtime Errors

### High error rate during run

Check the specific errors with `--errors`:

```sh
edg run --errors --driver pgx --config workload.yaml --url ${DATABASE_URL}
```

Common causes:

- **Foreign key violations**: Seed data doesn't cover the referenced values. Ensure `init` queries fetch the right dataset.
- **Unique constraint violations**: `gen()` can produce duplicates. Use `uuid_v4()` or `seq_global()` for unique columns.
- **Type mismatches**: A string value passed where the database expects an integer. Use explicit casts in SQL (e.g. `$1::INT`) or wrap expressions with `int()`.

### Seed is slow

Batch insert performance depends on `size` (rows per batch). Too small = too many round trips. Too large = transaction too big.

- Start with `size: 1000` for most databases.
- Use `exec_batch` (not `exec`) for bulk inserts.
- See [Performance Tuning]({{< relref "performance-tuning" >}}) for detailed guidance.

### "context deadline exceeded" during seed

The seed phase is taking longer than expected. Unlike the `run` phase, seed has no timeout by default. This error usually means the database itself is timing out (e.g. a statement timeout configured on the server).

- Reduce batch `size` to commit smaller transactions.
- Check database-side statement or transaction timeout settings.

### Workers not reaching expected QPS

- **Connection pool exhaustion**: Workers may be blocking on connection acquisition. Try increasing `--pool-size` or reducing `--workers`.
- **Database bottleneck**: Check database-side metrics (CPU, IOPS, lock contention).
- **Cold cache**: Use `--warmup-duration` to let the database warm up before measuring.

## Placeholder Errors

### "pq: could not determine data type of parameter $1"

This happens when PostgreSQL/CockroachDB can't infer the type of a bind parameter. Add an explicit cast:

```yaml
args:
  - ref_rand('fetch_accounts').id
query: SELECT * FROM account WHERE id = $1::UUID
```

### Placeholders not working with MySQL / Oracle / MSSQL

`run` queries use native bind parameters, which differ by driver:

| Driver | Placeholder |
|---|---|
| `pgx`, `dsql` | `$1`, `$2` |
| `mysql` | `?` |
| `oracle` | `:1`, `:2` |
| `mssql`, `spanner` | `@p1`, `@p2` |

Batch queries (`exec_batch`, `query_batch`) and batch-expanded queries (using `gen_batch`, `batch`, `ref_each`) inline values into the SQL text, so they always use `$1`, `$2` regardless of driver.

## Batch Separator Issues

### Data corruption with comma separators

**Never use commas as batch separators.** Generated values (names, addresses) can contain commas, silently splitting one value into multiple rows.

The simplest solution is to use `__values__`, which avoids batch separators entirely by generating a standard multi-row `VALUES` clause:

```yaml
# Recommended - no separator issues.
type: exec_batch
count: 1000
size: 100
args:
  - gen('name')
query: |-
  INSERT INTO users (name)
  __values__
```

If you're using the driver-specific batch expansion patterns (`unnest`, `JSON_TABLE`, etc.), always use `__sep__` instead of a literal comma:

```yaml
# Wrong - will corrupt data containing commas.
query: SELECT unnest(string_to_array('$1', ','))

# Correct - uses ASCII unit separator (char 31).
query: SELECT unnest(string_to_array('$1', __sep__))
```

## Expression Debugging

Use the REPL to test expressions interactively:

```sh
edg repl
>> gen('email')
markusmoen@pagac.net

>> uniform_f(0.01, 999.99, 2)
347.82
```

Load a config to test expressions with globals and reference data:

```sh
edg repl --config workload.yaml
>> warehouses * 10
10

>> ref_rand('regions').name
eu
```
