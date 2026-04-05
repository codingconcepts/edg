---
title: Home
layout: home
nav_order: 1
---

<p align="center">
  <img src="{{ '/assets/logo.png' | relative_url }}" alt="edg logo" width="350"/>
</p>

# edg

A database workload runner driven by YAML configuration. Define your schema, seed data, and transactional workloads in a single config file, then run them against any supported database with concurrent workers and real-time throughput reporting.

Query arguments are written as expressions compiled at startup, giving you access to global constants, random data generation, reference lookups, and TPC-C-compliant non-uniform random distributions.

## Supported Databases

| Database | Driver | URL scheme |
|---|---|---|
| CockroachDB / PostgreSQL | `pgx` | `postgres://...` |
| Oracle | `oracle` | `oracle://...` |
| MySQL | `mysql` | `user:password@tcp(host:port)/database?parseTime=true` |
