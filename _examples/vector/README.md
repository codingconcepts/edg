# Vector Embeddings

Demonstrates `vector`, `vector_zipf`, and `vector_norm` for generating pgvector-compatible embeddings with realistic clustering for similarity search.

## Functions

| Function | Signature | Description |
|---|---|---|
| `vector` | `vector(dims, clusters, spread)` | Clustered unit-length vector literal with **uniform** centroid selection. |
| `vector_zipf` | `vector_zipf(dims, clusters, spread, s, v)` | **Zipfian** centroid selection - cluster 0 is the "hottest", realistic power-law skew. |
| `vector_norm` | `vector_norm(dims, clusters, spread, mean, stddev)` | **Normal** centroid selection - bell curve centered on a cluster index. |

## Schema

The example creates an `article` table with three embedding columns, one per distribution, so you can compare their clustering behaviour side by side:

| Column | Expression | Distribution |
|---|---|---|
| `embedding_uniform` | `vector(32, 5, 0.1)` | Equal-sized clusters |
| `embedding_zipf` | `vector_zipf(32, 5, 0.1, 2.0, 1.0)` | Cluster 0 dominates (power-law) |
| `embedding_norm` | `vector_norm(32, 5, 0.1, 2.0, 0.8)` | Cluster 2 most common (bell curve) |

## How clustering works

The first call lazily generates `clusters` random unit-vector centroids. Each subsequent call picks a centroid (uniform, Zipfian, or normal), adds Gaussian noise (sigma = `spread`), then normalizes to unit length. This produces groups of nearby vectors so that kNN search returns meaningful neighbours.

- **Low spread** (0.05-0.1): tight, well-separated clusters - ideal for testing index recall
- **High spread** (0.3-0.5): overlapping clusters - more realistic for production-like data
- **Zipfian** (`vector_zipf`): most vectors land in a few "hot" clusters, simulating real-world category skew
- **Normal** (`vector_norm`): a central cluster is most popular, with gradual falloff

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg up \
--driver pgx \
--config _examples/vector/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg seed \
--driver pgx \
--config _examples/vector/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg run \
--driver pgx \
--config _examples/vector/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 30s
```

### Verify similarity search

After seeding, compare nearest-neighbour results across the three distributions:

```sql
-- Uniform: roughly equal cluster sizes.
SELECT id, title, embedding_uniform <=> (SELECT embedding_uniform FROM article LIMIT 1) AS distance
FROM article
ORDER BY embedding_uniform <=> (SELECT embedding_uniform FROM article LIMIT 1)
LIMIT 5;

-- Zipfian: most results come from the dominant cluster.
SELECT id, title, embedding_zipf <=> (SELECT embedding_zipf FROM article LIMIT 1) AS distance
FROM article
ORDER BY embedding_zipf <=> (SELECT embedding_zipf FROM article LIMIT 1)
LIMIT 5;

-- Normal: results cluster around the center.
SELECT id, title, embedding_norm <=> (SELECT embedding_norm FROM article LIMIT 1) AS distance
FROM article
ORDER BY embedding_norm <=> (SELECT embedding_norm FROM article LIMIT 1)
LIMIT 5;
```

### Teardown

```sh
go run ./cmd/edg deseed \
--driver pgx \
--config _examples/vector/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/vector/crdb.yaml \
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
--config _examples/vector/mysql.yaml \
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
--config _examples/vector/oracle.yaml \
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
--config _examples/vector/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=vector&encrypt=disable"
```
