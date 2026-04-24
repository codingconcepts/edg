# Cassandra

A basic Cassandra example demonstrating keyspace and table creation, inserts with generated UUIDs and emails, and cross-table references using CQL.

## Setup

```sh
docker compose -f _examples/compose_cassandra.yml up -d
```

Cassandra takes ~30 seconds to become ready:

```sh
until docker exec cassandra cqlsh -e "SELECT now() FROM system.local" > /dev/null 2>&1; do
  echo "waiting for cassandra..."; sleep 2
done && echo "cassandra ready"
```

## Run

```sh
go run ./cmd/edg up \
--driver cassandra \
--config _examples/cassandra/cassandra.yaml \
--url "cassandra://localhost:9042"

go run ./cmd/edg seed \
--driver cassandra \
--config _examples/cassandra/cassandra.yaml \
--url "cassandra://localhost:9042"
```

Check data:

```sh
docker exec cassandra cqlsh -e "SELECT COUNT(*) FROM edg.customer; SELECT COUNT(*) FROM edg.account;"
docker exec cassandra cqlsh -e "SELECT * FROM edg.customer LIMIT 5;"
docker exec cassandra cqlsh -e "SELECT * FROM edg.account LIMIT 5;"
```

```sh
go run ./cmd/edg deseed \
--driver cassandra \
--config _examples/cassandra/cassandra.yaml \
--url "cassandra://localhost:9042"

go run ./cmd/edg down \
--driver cassandra \
--config _examples/cassandra/cassandra.yaml \
--url "cassandra://localhost:9042"
```
