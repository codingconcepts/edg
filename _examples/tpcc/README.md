# TPC-C

A TPC-C benchmark implementation with all 5 transaction profiles (New-Order, Payment, Order-Status, Delivery, Stock-Level) using writable CTEs for atomic execution.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
docker exec -it node1 cockroach init --insecure
docker exec -it node1 cockroach sql --insecure
```

### Run

```sh
edg up \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg seed \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg run \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 100 \
-d 1m

edg deseed \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg down \
--driver pgx \
--config _examples/tpcc/crdb.yaml \
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
edg up \
--driver oracle \
--config _examples/tpcc/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

edg seed \
--driver oracle \
--config _examples/tpcc/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

edg run \
--driver oracle \
--config _examples/tpcc/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb" \
-w 100 \
-d 1m

edg deseed \
--driver oracle \
--config _examples/tpcc/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

edg down \
--driver oracle \
--config _examples/tpcc/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```
