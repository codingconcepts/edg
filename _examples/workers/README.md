# Workers

An example of background worker queries running alongside the main workload.

The `workers` section defines queries that execute on a fixed schedule, independent of the main `run` loop. Each worker runs in its own goroutine with its own environment.

This example models a job queue where `run` workers claim and process jobs, while background `workers` handle lease expiry:

```yaml
workers:
  - name: reap_expired
    rate: 1/5s
    type: exec
    query: |-
      UPDATE job
      SET status = 'pending', worker_id = NULL, lease_expires_at = NULL
      WHERE status = 'running'
        AND lease_expires_at < now()
        AND attempts < max_attempts

  - name: fail_exhausted
    rate: 1/5s
    type: exec
    query: |-
      UPDATE job
      SET status = 'failed', completed_at = now()
      WHERE status = 'running'
        AND lease_expires_at < now()
        AND attempts >= max_attempts
```

The `rate` field uses the format `times/interval`. For example, `1/5s` means once every 5 seconds, and `3/1m` means 3 times per minute (once every 20 seconds).

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver pgx \
--config _examples/workers/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 30s
```
