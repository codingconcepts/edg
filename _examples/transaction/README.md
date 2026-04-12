# Transaction

Demonstrates multi-statement transactions using the bank schema. The `make_transfer` workload reads two account balances and then debits one and credits the other, all inside an explicit `BEGIN`/`COMMIT` block. This ensures the read-then-write pattern sees consistent data and the debit/credit pair is atomic.

Compare with the [bank](_examples/bank) example, which achieves the same result using a single `UPDATE ... CASE` statement. The transaction approach is more realistic for applications that perform multiple round-trips within a transaction.

```yaml
run:
  - transaction: make_transfer
    queries:
      - name: read_source
        type: query
        args: [ref_diff('fetch_accounts').id]
        query: SELECT id, balance FROM account WHERE id = $1::UUID

      - name: debit_source
        type: exec
        args:
          - ref_same('read_source').id
          - gen('number:1,100')
        query: UPDATE account SET balance = balance - $2::FLOAT WHERE id = $1::UUID
```

Standalone queries and transactions can be mixed freely in the `run` section and selected via `run_weights`:

```yaml
run_weights:
  check_balance: 50
  make_transfer: 50
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
--config _examples/transaction/crdb.yaml \
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
--config _examples/transaction/mysql.yaml \
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
--config _examples/transaction/oracle.yaml \
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
--config _examples/transaction/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=transaction&encrypt=disable"
```
