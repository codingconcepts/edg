# Expectations

An example of post-run assertions using the `expectations` section. Expectations are boolean expressions evaluated against run metrics after the workload finishes. If any expectation fails, edg exits with a non-zero status code.

This is useful for CI/CD pipelines where you want to gate deployments on performance characteristics like error rate, latency percentiles, or throughput.

```yaml
expectations:
  - error_rate < 1
  - check_balance.p99 < 50
  - credit_account.p99 < 59
  - tpm > 1000
```

## Referencing globals

Expectations can reference any variable defined in the `globals` section. This avoids hardcoding values that already exist in your config:

```yaml
globals:
  accounts: 10000
  max_error_pct: 5

expectations:
  - error_rate < max_error_pct
  - query: SELECT COUNT(*) AS cnt FROM account
    expr: cnt == accounts
```

Global names must not collide with built-in metrics (`error_rate`, `success_count`, `total_errors`, `tpm`). edg will reject the config at startup if they do.

## Database-backed expectations

An expectation can also be an object with `query` and `expr` fields. The SQL query runs after the workload finishes and its first-row columns are available in the expression. Globals are available in these expressions too:

```yaml
globals:
  accounts: 10000

expectations:
  - error_rate < 1
  - query: SELECT COUNT(*) AS cnt FROM account
    expr: cnt == accounts
  - query: SELECT SUM(balance) AS total FROM account
    expr: total > 0
```

After the run summary, expectations are printed with a PASS/FAIL status:

```
expectations
  PASS error_rate < 1
  PASS check_balance.p99 < 100
  PASS credit_account.p99 < 100
  PASS account_count.cnt > 0
  FAIL tpm > 1000
```

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver pgx \
--config _examples/expectations/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

## MySQL

### Setup

```sh
docker compose -f _examples/compose_mysql.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver mysql \
--config _examples/expectations/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"
```

## Oracle

### Setup

```sh
docker compose -f _examples/compose_oracle.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver oracle \
--config _examples/expectations/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```

## MSSQL

### Setup

```sh
docker compose -f _examples/compose_mssql.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver mssql \
--config _examples/expectations/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=expectations&encrypt=disable"
```
