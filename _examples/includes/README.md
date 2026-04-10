# Config Includes

This example demonstrates the `!include` directive for reusing reference data and expressions across workload files.

## Usage

Use the `!include` YAML tag to pull in content from another file:

```yaml
globals: !include shared/globals.yaml
up: !include shared/schema.yaml
down: !include shared/teardown.yaml
run: !include shared/run_queries.yaml
```

Paths are resolved relative to the file containing the `!include` directive.

## Structure

```
includes/
  crdb.yaml                          # CockroachDB workload
  mysql.yaml                         # MySQL workload
  oracle.yaml                        # Oracle workload
  shared/
    globals.yaml                     # Shared global variables (all databases)
    schema.yaml                      # CockroachDB schema
    schema_mysql.yaml                # MySQL schema
    schema_oracle.yaml               # Oracle schema
    teardown.yaml                    # CockroachDB teardown
    teardown_mysql.yaml              # MySQL teardown
    teardown_oracle.yaml             # Oracle teardown
    run_queries.yaml                 # CockroachDB run queries
    run_queries_mysql.yaml           # MySQL run queries
    run_queries_oracle.yaml          # Oracle run queries
```

## What can be included

An `!include` can appear anywhere a YAML value is expected:

- **Mapping value** - replace a key's value with the content of a file:
  ```yaml
  globals: !include shared/globals.yaml
  ```

- **Sequence value** - replace an entire list:
  ```yaml
  up: !include shared/schema.yaml
  ```

- **Sequence item** - splice items from an included file into a list:
  ```yaml
  run:
    - name: local_query
      query: SELECT 1
    - !include shared/extra_queries.yaml
  ```

Nested includes are supported (an included file can itself use `!include`). Circular includes are detected and produce an error.

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
--config _examples/includes/crdb.yaml \
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
--config _examples/includes/mysql.yaml \
--url "root:password@tcp(localhost:3306)/includes?parseTime=true"
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
--config _examples/includes/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```
