# Bank

A simpler workload modelling bank account operations (balance checks, credits, transfers). Useful for contention and correctness testing.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
docker exec -it node1 cockroach init --insecure
docker exec -it node1 cockroach sql --insecure
```

### Run

```sh
go run . all \
--driver pgx \
--config _examples/bank/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

# Or separately.
go run . up \
--driver pgx \
--config _examples/bank/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run . seed \
--driver pgx \
--config _examples/bank/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run . run \
--driver pgx \
--config _examples/bank/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 100 \
-d 1m

go run . deseed \
--driver pgx \
--config _examples/bank/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run . down \
--driver pgx \
--config _examples/bank/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

## Oracle

### Setup

```sh
docker run \
--name oracle \
-d \
-p 1521:1521 \
-p 5500:5500 \
-e ORACLE_PDB=defaultdb \
-e ORACLE_PWD=password \
container-registry.oracle.com/database/enterprise:19.19.0.0
```

### Run

```sh
go run . up \
--driver oracle \
--config _examples/bank/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run . seed \
--driver oracle \
--config _examples/bank/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run . run \
--driver oracle \
--config _examples/bank/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 100 \
-d 1m

go run . deseed \
--driver oracle \
--config _examples/bank/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run . down \
--driver oracle \
--config _examples/bank/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```
