---
title: Quick Start
weight: 2
---

# Quick Start

This guide gets you from zero to running a workload in under a minute.

## Try expressions (no database needed)

After [installing]({{< relref "installation" >}}) edg, you can evaluate expressions directly from the command line:

```sh
edg email
# naomiroberts@robinson.net

edg "uuid_v4()"
# c8952841-6f5b-4743-a6de-2200415c2f03

edg "regex('[A-Z]{3}-[0-9]{4}')"
# QVM-8314
```

Or start an interactive REPL session:

```sh
edg repl
```

```
>> uniform(0, 100)
73.37

>> set_rand(['credit_card', 'debit_card', 'paypal'], [])
debit_card

>> template('ORD-%05d', seq(1, 1))
ORD-00001
```

## Run a workload

Create a file called `workload.yaml`:

```yaml
globals:
  users: 1000
  batch_size: 100

up:
  - name: create_users
    query: |-
      CREATE TABLE IF NOT EXISTS users (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        email STRING NOT NULL
      )

seed:
  - name: populate_users
    args:
      - gen_batch(users, batch_size, 'email')
    query: |-
      INSERT INTO users (email)
      SELECT unnest(string_to_array('$1', ','))

init:
  - name: fetch_users
    query: SELECT id, email FROM users ORDER BY random()

run:
  - name: get_user
    args:
      - ref_rand('fetch_users').id
    query: |-
      SELECT * FROM users WHERE id = $1::UUID

deseed:
  - name: truncate_users
    type: exec
    query: TRUNCATE TABLE users CASCADE

down:
  - name: drop_users
    type: exec
    query: DROP TABLE IF EXISTS users
```

Run the full lifecycle with a single command:

```sh
edg all \
  --driver pgx \
  --config workload.yaml \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  -w 10 \
  -d 30s
```

This creates the table, seeds 1,000 users, runs random lookups with 10 concurrent workers for 30 seconds, then cleans up.

## What next?

- [CLI Reference]({{< relref "cli-reference" >}}) -- all commands and flags
- [Configuration]({{< relref "configuration" >}}) -- full YAML config reference
- [Expressions]({{< relref "expressions" >}}) -- every built-in function and the expr-lang feature set
- [Example Workloads]({{< relref "example-workloads" >}}) -- TPC-C, e-commerce, IoT, and more
- [Integration Testing]({{< relref "integration-testing" >}}) -- using edg to seed databases for integration tests
