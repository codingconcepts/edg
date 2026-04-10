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

After the run summary, expectations are printed with a PASS/FAIL status:

```
expectations
  PASS error_rate < 1
  PASS check_balance.p99 < 100
  PASS credit_account.p99 < 100
  FAIL tpm > 1000
```

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
docker exec -it node1 cockroach init --insecure
docker exec -it node1 cockroach sql --insecure
```

### Run

```sh
go run ./cmd/edg all \
--driver pgx \
--config _examples/expectations/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
