# Named Args

Demonstrates the named args syntax, where `args` is a map instead of a list. Named args let you reference values by name with `arg('name')` instead of by index with `arg(0)`.

## Positional vs named

Positional (existing):

```yaml
args:
  - gen('email')
  - ref_same('regions').name
  - set_rand(ref_same('regions').cities, [])
  - uniform(1, 500)
print:
  - arg(3)
```

Named (equivalent):

```yaml
args:
  email: gen('email')
  region: ref_same('regions').name
  city: set_rand(ref_same('regions').cities, [])
  amount: uniform(1, 500)
print:
  - arg('amount')
```

Named args bind to `$1`, `$2`, etc. in declaration order, so query placeholders work identically. The only difference is how you reference previously computed args within expressions.

> [!NOTE]
> Index-based `arg(0)` still works with named args. Named and positional references can be mixed within expressions.

## CockroachDB

### Setup

```sh
docker compose -f cmd/harness/compose/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/named_args/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/named_args/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/named_args/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 10s

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/named_args/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/named_args/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
