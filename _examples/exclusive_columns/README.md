# Exclusive Columns

This example demonstrates how to populate a table where exactly one of two columns must be provided (XOR constraint). A common pattern for tables with a CHECK constraint enforcing mutual exclusivity.

## How it works

A coin flip is generated as an intermediate arg, then `cond()` and `arg()` are used to set one column to a value and the other to NULL:

```yaml
args:
  - gen('name')
  - bool()                           # coin flip
  - cond(arg(1), gen('email'), nil)  # email if true, NULL if false
  - cond(!arg(1), gen('phone'), nil) # phone if false, NULL if true
```

The coin flip arg is not bound to the query, only the name, email, and phone args are used as `$1`, `$2`, `$3`.

## Schema

The `contact` table enforces that exactly one of `email` or `phone` is provided:

```sql
CHECK (
  (email IS NOT NULL AND phone IS NULL) OR
  (email IS NULL AND phone IS NOT NULL)
)
```

## Running

```sh
go run ./cmd/edg all \
--driver pgx \
--config _examples/exclusive_columns/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" 1
--duration 5s
```
