# Distributions

Demonstrates all five distribution functions by writing values into a single table with a `dist_type` label, making it easy to compare histograms side by side.

## Functions

### Numeric distributions

| Function | Signature | Description |
|---|---|---|
| `uniform` | `uniform(min, max)` | Flat distribution, every value equally likely |
| `zipf` | `zipf(s, v, max)` | Power-law skew, low values dominate |
| `norm_f` | `norm_f(mean, stddev, min, max, precision)` | Bell curve centered on mean |
| `exp_f` | `exp_f(rate, min, max, precision)` | Exponential decay from min |
| `lognorm_f` | `lognorm_f(mu, sigma, min, max, precision)` | Right-skewed with a long tail |

### Set distributions

Pick from a predefined set of values using a distribution to control which items are selected most often.

| Function | Signature | Description |
|---|---|---|
| `set_rand` | `set_rand(values, weights)` | Uniform or weighted random selection from a set |
| `set_norm` | `set_norm(values, mean, stddev)` | Normal distribution over indices; `mean` index picked most often |
| `set_exp` | `set_exp(values, rate)` | Exponential distribution over indices; lower indices picked most often |
| `set_lognorm` | `set_lognorm(values, mu, sigma)` | Log-normal distribution over indices; right-skewed selection |
| `set_zipf` | `set_zipf(values, s, v)` | Zipfian distribution over indices; strong power-law skew toward first items |

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
--config _examples/distributions/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/distributions/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 30s

go run ./cmd/edg deseed \
--driver pgx \
--config _examples/distributions/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/distributions/crdb.yaml \
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
--config _examples/distributions/mysql.yaml \
--url "root:password@tcp(localhost:3306)/distributions?parseTime=true"

go run ./cmd/edg run \
--driver mysql \
--config _examples/distributions/mysql.yaml \
--url "root:password@tcp(localhost:3306)/distributions?parseTime=true" \
-w 10 \
-d 30s

go run ./cmd/edg deseed \
--driver mysql \
--config _examples/distributions/mysql.yaml \
--url "root:password@tcp(localhost:3306)/distributions?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/distributions/mysql.yaml \
--url "root:password@tcp(localhost:3306)/distributions?parseTime=true"
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
--config _examples/distributions/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg run \
--driver oracle \
--config _examples/distributions/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 10 \
-d 30s

go run ./cmd/edg deseed \
--driver oracle \
--config _examples/distributions/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg down \
--driver oracle \
--config _examples/distributions/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```

## SQL Server

### Setup

```sh
docker compose -f _examples/compose_sqlserver.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver sqlserver \
--config _examples/distributions/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=distributions&encrypt=disable"

go run ./cmd/edg run \
--driver sqlserver \
--config _examples/distributions/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=distributions&encrypt=disable" \
-w 10 \
-d 30s

go run ./cmd/edg deseed \
--driver sqlserver \
--config _examples/distributions/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=distributions&encrypt=disable"

go run ./cmd/edg down \
--driver sqlserver \
--config _examples/distributions/sqlserver.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=distributions&encrypt=disable"
```
