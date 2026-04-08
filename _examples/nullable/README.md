# Nullable Columns

This example demonstrates the `nullable(expr, probability)` function for injecting NULL values into nullable columns with controlled frequency.

## Usage

Wrap any expression with `nullable` and provide a probability between 0.0 and 1.0:

```yaml
args:
  - nullable(gen('email'), 0.3)      # 30% NULL, 70% random email
  - nullable(gen('sentence:5'), 0.5) # 50% NULL, 50% random sentence
  - nullable(uuid_v4(), 0.8)         # 80% NULL, 20% random UUID
```

## Schema

The example creates a `user` table with several nullable columns, each with a different NULL probability:

| Column | Expression | NULL % |
|---|---|---|
| `email` | `gen('email')` | 0% (NOT NULL) |
| `phone` | `nullable(gen('phone'), 0.3)` | 30% |
| `bio` | `nullable(gen('sentence:5'), 0.5)` | 50% |
| `referred_by` | `nullable(uuid_v4(), 0.8)` | 80% |

## Running

```sh
go run . all \
--driver pgx \
--config _examples/nullable/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
