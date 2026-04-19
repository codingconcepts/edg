# Observability

Run a bank workload against CockroachDB with live Prometheus metrics and a pre-built Grafana dashboard. The workload includes read queries, writes, and a transfer transaction that rolls back on insufficient funds, so all metric types (QPS, latency, errors, commits, rollbacks) are visible in Grafana.

## Setup

Start CockroachDB, Prometheus, and Grafana:

```sh
docker compose -f _examples/observability/compose.yml up -d
```

## Run

```sh
go run ./cmd/edg all \
--driver pgx \
--config _examples/observability/workload.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
--metrics-addr :9090
```

The workload runs three query types with weighted random selection:

| Query | Weight | Description |
|---|---|---|
| `check_balance` | 30% | Read a random account balance |
| `credit_account` | 10% | Credit a random account |
| `lookup_customer` | 15% | Join lookup of a customer and their account |
| `top_balances` | 5% | Top 10 accounts by balance (heavier query) |
| `make_transfer` | 40% | Transfer between two accounts (rolls back if source has insufficient funds) |

## Observe

- **Grafana** [http://localhost:3000](http://localhost:3000) (no login required, the edg dashboard is pre-provisioned)
- **Prometheus** [http://localhost:9091](http://localhost:9091)
- **CockroachDB UI** [http://localhost:8080](http://localhost:8080)

## Teardown

```sh
docker compose -f _examples/observability/compose.yml down
```
