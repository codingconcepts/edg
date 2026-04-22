# Anomalies

Generates traffic patterns that provoke classic read/write anomalies under weak isolation levels. Under SERIALIZABLE (CockroachDB default), these patterns are prevented by transaction aborts. Increase workers to amplify contention and observe retry behaviour.

## Anomalies modelled

**Lost update** - Two transactions read the same counter, each computes `value + 1`, and writes it back. Under READ COMMITTED both commits succeed but one increment is silently lost. Under SERIALIZABLE the conflict is detected and one transaction is aborted.

```yaml
- transaction: lost_update
  queries:
    - name: read_counter
      type: query
      args: [ref_rand('fetch_counters').id]
      query: SELECT id, value FROM counter WHERE id = $1::UUID

    - name: write_counter
      type: exec
      args:
        - ref_same('read_counter').id
        - ref_same('read_counter').value
      query: UPDATE counter SET value = $2::INT + 1 WHERE id = $1::UUID
```

**Stale read** - A transaction reads an account balance and then debits it. Concurrent `credit_account` writes may change the balance between the read and the debit. Under READ COMMITTED the debit acts on a stale value; under SERIALIZABLE the snapshot is consistent.

**Write skew** - Two doctors on the same shift each check that at least two doctors are on duty, then go off duty. Under READ COMMITTED both see the same count and both commit, leaving zero coverage. Under SERIALIZABLE one transaction is aborted to preserve the invariant.

```yaml
- transaction: write_skew
  queries:
    - name: check_coverage
      type: query
      args: [ref_same('read_doctor').shift_date]
      query: |-
        SELECT COUNT(*) AS cnt
        FROM on_call
        WHERE shift_date = $1::DATE AND on_duty = true

    - rollback_if: "ref_same('check_coverage').cnt <= 1"

    - name: go_off_duty
      type: exec
      args: [ref_same('read_doctor').id]
      query: UPDATE on_call SET on_duty = false WHERE id = $1::UUID
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
--config _examples/anomalies/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 1 \
-d 10s

go run ./cmd/edg all \
--driver pgx \
--config _examples/anomalies/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 2 \
-d 10s

go run ./cmd/edg all \
--driver pgx \
--config _examples/anomalies/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 4 \
-d 10s

go run ./cmd/edg all \
--driver pgx \
--config _examples/anomalies/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 8 \
-d 10s \
--retries 3
```
