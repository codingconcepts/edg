# Reference Data

Demonstrates the `reference` config section, which loads static datasets into memory without a database query. Reference data is available to all `ref_*` functions (`ref_rand`, `ref_same`, `ref_diff`, etc.) just like `init` query results.

This example defines a **regions** reference dataset with names and cities, then uses `ref_same` to seed customers with a consistent region and city from the same row.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
docker exec -it node1 cockroach init --insecure
docker exec -it node1 cockroach sql --insecure
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/reference_data/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/reference_data/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/reference_data/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/reference_data/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

## MySQL

### Setup

```sh
docker compose -f _examples/compose_mysql.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver mysql \
--config _examples/reference_data/mysql.yaml \
--url "root:password@tcp(localhost:3306)/reference_data?parseTime=true"

go run ./cmd/edg seed \
--driver mysql \
--config _examples/reference_data/mysql.yaml \
--url "root:password@tcp(localhost:3306)/reference_data?parseTime=true"

go run ./cmd/edg deseed \
--driver mysql \
--config _examples/reference_data/mysql.yaml \
--url "root:password@tcp(localhost:3306)/reference_data?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/reference_data/mysql.yaml \
--url "root:password@tcp(localhost:3306)/reference_data?parseTime=true"
```

## Oracle

### Setup

```sh
docker compose -f _examples/compose_oracle.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver oracle \
--config _examples/reference_data/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg seed \
--driver oracle \
--config _examples/reference_data/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg deseed \
--driver oracle \
--config _examples/reference_data/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg down \
--driver oracle \
--config _examples/reference_data/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```
