# Normal Distribution

Demonstrates `norm` and `norm_n` by generating product reviews with realistic rating distributions. Most ratings cluster around 4 stars (mean=4, stddev=1, clamped to 1-5).

## Functions

| Function | Signature | Description |
|---|---|---|
| `norm` | `norm(mean, stddev, min, max)` | Single normally-distributed random integer |
| `norm_n` | `norm_n(mean, stddev, min, max, minN, maxN)` | N unique normally-distributed integers (comma-separated) |

## CockroachDB

### Setup

```sh
docker compose -f cmd/harness/compose/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/normal/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/normal/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/normal/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 30s
```

### Verify distribution

After running, check that ratings follow a bell curve around 4:

```sql
SELECT
  rating,
  count(*) AS total,
  repeat('█', (count(*) * 50 / max(count(*)) OVER ())::INT) AS histogram
FROM review
GROUP BY rating
ORDER BY rating;
```

Example output:

```
  rating | total |                     histogram
---------+-------+-----------------------------------------------------
       1 |   319 | █
       2 |  3398 | █████████
       3 | 12644 | ████████████████████████████████
       4 | 19940 | ██████████████████████████████████████████████████
       5 | 13271 | █████████████████████████████████
```

### Teardown

```sh
go run ./cmd/edg deseed \
--driver pgx \
--config _examples/normal/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/normal/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

## MySQL

### Setup

```sh
docker compose -f cmd/harness/compose/compose_mysql.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver mysql \
--config _examples/normal/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg seed \
--driver mysql \
--config _examples/normal/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg run \
--driver mysql \
--config _examples/normal/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
-w 10 \
-d 30s

go run ./cmd/edg deseed \
--driver mysql \
--config _examples/normal/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/normal/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"
```

## Oracle

### Setup

```sh
docker compose -f cmd/harness/compose/compose_oracle.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver oracle \
--config _examples/normal/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg seed \
--driver oracle \
--config _examples/normal/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg run \
--driver oracle \
--config _examples/normal/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 30s

go run ./cmd/edg deseed \
--driver oracle \
--config _examples/normal/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg down \
--driver oracle \
--config _examples/normal/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```

## MSSQL

### Setup

```sh
docker compose -f cmd/harness/compose/compose_mssql.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver mssql \
--config _examples/normal/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=normal&encrypt=disable"

go run ./cmd/edg seed \
--driver mssql \
--config _examples/normal/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=normal&encrypt=disable"

go run ./cmd/edg run \
--driver mssql \
--config _examples/normal/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=normal&encrypt=disable" \
-w 10 \
-d 30s
```

### Teardown

```sh
go run ./cmd/edg deseed \
--driver mssql \
--config _examples/normal/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=normal&encrypt=disable"

go run ./cmd/edg down \
--driver mssql \
--config _examples/normal/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=normal&encrypt=disable"
```
