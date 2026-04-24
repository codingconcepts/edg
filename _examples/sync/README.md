# Sync

Write identical data to PostgreSQL and MySQL, then verify both databases match. Tests dual-write consistency.

## Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
docker compose -f _examples/compose_mysql.yml up -d

until cockroach sql --insecure -e "SELECT 1" &>/dev/null; do sleep 1; done
until mysql -u root -ppassword -h 127.0.0.1 -e "SELECT 1" &>/dev/null 2>&1; do sleep 1; done
```

## Dual-write

Seed both databases with the same data (deterministic via `--rng-seed`), then verify:

```sh
go run ./cmd/edg sync run \
--source-driver pgx \
--source-url "postgres://root:password@localhost:26257/defaultdb?sslmode=disable" \
--source-config _examples/sync/crdb.yaml \
--target-driver mysql \
--target-url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
--target-config _examples/sync/mysql.yaml \
--rng-seed 42
```

Check data in both databases:

```sh
cockroach sql --insecure -e "SELECT id, name FROM users ORDER BY id LIMIT 5"
mysql -u root -ppassword -h 127.0.0.1 defaultdb -e "SELECT id, name FROM users ORDER BY id LIMIT 5"

cockroach sql --insecure -e "SELECT * FROM orders LIMIT 5"
mysql -u root -ppassword -h 127.0.0.1 defaultdb -e "SELECT * FROM orders LIMIT 5"
```

```sh
go run ./cmd/edg sync verify \
--source-driver pgx \
--source-url "postgres://root:password@localhost:26257/defaultdb?sslmode=disable" \
--target-driver mysql \
--target-url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
--tables users,orders \
--order-by id \
--ignore-columns created_at
```

## Replication (MOLT Fetch)

Seed the source only. Create schema on the target manually, then use an external tool like [MOLT Fetch](https://www.cockroachlabs.com/docs/molt/molt-fetch) to copy data across.

Create schema on both, seed source only:

```sh
go run ./cmd/edg up \
--driver pgx \
--url "postgres://root:password@localhost:26257/defaultdb?sslmode=disable" \
--config _examples/sync/crdb.yaml

go run ./cmd/edg up \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
--config _examples/sync/mysql.yaml

go run ./cmd/edg sync run \
--source-driver pgx \
--source-url "postgres://root:password@localhost:26257/defaultdb?sslmode=disable" \
--source-config _examples/sync/crdb.yaml \
--rng-seed 42
```

Copy data with MOLT Fetch (or similar):

```sh
molt fetch \
--source "postgres://root:password@localhost:26257/defaultdb?sslmode=disable" \
--target "mysql://root:password@localhost:3306/defaultdb"
```

Verify:

```sh
go run ./cmd/edg sync verify \
--source-driver pgx \
--source-url "postgres://root:password@localhost:26257/defaultdb?sslmode=disable" \
--target-driver mysql \
--target-url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
--tables users,orders \
--order-by id \
--ignore-columns created_at \
--wait 5s
```

## Teardown

```sh
go run ./cmd/edg sync down \
--source-driver pgx \
--source-url "postgres://root:password@localhost:26257/defaultdb?sslmode=disable" \
--source-config _examples/sync/crdb.yaml \
--target-driver mysql \
--target-url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
--target-config _examples/sync/mysql.yaml

docker compose -f _examples/compose_crdb.yml down
docker compose -f _examples/compose_mysql.yml down
```
