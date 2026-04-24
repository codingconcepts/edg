---
title: Example Workloads
weight: 6
bookToc: false
---

# Example Workloads

Complete workload configs you can run directly or use as starting points for your own.

| Workload | Description |
|---|---|
| [Aggregation](https://github.com/codingconcepts/edg/tree/main/_examples/aggregation/) | Demonstrates aggregation functions (sum, avg, min, max, count, distinct) |
| [Bank](https://github.com/codingconcepts/edg/tree/main/_examples/bank/) | Bank account operations for contention and correctness testing |
| [CH-benCHmark](https://github.com/codingconcepts/edg/tree/main/cmd/edg/workload/ch_benchmark/) | Mixed OLTP+OLAP workload combining TPC-C transactions with TPC-H-style analytical queries |
| [Batch](https://github.com/codingconcepts/edg/tree/main/_examples/batch/) | Demonstrates `query_batch` and `exec_batch` query types for batch inserts and updates |
| [Blob](https://github.com/codingconcepts/edg/tree/main/_examples/blob/) | Binary data with `blob()` (all databases) and `bytes()` (PostgreSQL/CockroachDB) |
| [Composite Types](https://github.com/codingconcepts/edg/tree/main/_examples/composite_types/) | PostgreSQL/CockroachDB composite types (`CREATE TYPE ... AS (...)`) as column types with `ROW(...)::type` construction and `(col).field` access |
| [Distributions](https://github.com/codingconcepts/edg/tree/main/_examples/distributions/) | All five distribution functions (uniform, zipf, norm_f, exp_f, lognorm_f) |
| [E-Commerce](https://github.com/codingconcepts/edg/tree/main/_examples/ecommerce/) | E-commerce with categories, products, customers, and orders |
| [Each Cartesian](https://github.com/codingconcepts/edg/tree/main/_examples/each_cartesian/) | Cartesian product seeding with `ref_each` across multiple tables |
| [Environment](https://github.com/codingconcepts/edg/tree/main/_examples/environment/) | Demonstrates fetching and using environment variables |
| [Failing](https://github.com/codingconcepts/edg/tree/main/_examples/failing/) | Map lookup with `fail()` to validate environment variables and stop workers on unknown values |
| [Exclusive Columns](https://github.com/codingconcepts/edg/tree/main/_examples/exclusive_columns/) | Mutually exclusive columns - either col_a or col_b, never both |
| [Expectations](https://github.com/codingconcepts/edg/tree/main/_examples/expectations/) | Post-run assertions for CI/CD gating on error rate, latency, and throughput |
| [Expressions](https://github.com/codingconcepts/edg/tree/main/_examples/expression/) | Demonstrates expr-lang built-in features (array, map, string, bitwise, etc.) |
| [Global Sequences](https://github.com/codingconcepts/edg/tree/main/_examples/global_sequences/) | Globally unique auto-incrementing sequences shared across all workers with `seq_global` |
| [Includes](https://github.com/codingconcepts/edg/tree/main/_examples/includes/) | Splitting and reusing config fragments with the `!include` directive |
| [IoT](https://github.com/codingconcepts/edg/tree/main/_examples/iot/) | IoT devices, sensors, and time-series readings |
| [LTREE](https://github.com/codingconcepts/edg/tree/main/_examples/ltree/) | Hierarchical org chart using PostgreSQL's `ltree` extension with `ltree()` path builder |
| [Locale](https://github.com/codingconcepts/edg/tree/main/_examples/locale/) | Locale-aware PII generation (`gen_locale`) with deterministic masking (`mask`) across JP and DE regions |
| [Normal](https://github.com/codingconcepts/edg/tree/main/_examples/normal/) | Product reviews with normal distribution ratings |
| [Named Args](https://github.com/codingconcepts/edg/tree/main/_examples/named_args/) | Map-style args with `arg('name')` instead of `arg(0)` |
| [Nullable](https://github.com/codingconcepts/edg/tree/main/_examples/nullable/) | Demonstrates `nullable(expr, probability)` for injecting NULLs with controlled frequency |
| [Observability](https://github.com/codingconcepts/edg/tree/main/_examples/observability/) | Prometheus metrics and Grafana dashboard with queries, writes, and rollback-prone transactions |
| [Org Tree](https://github.com/codingconcepts/edg/tree/main/_examples/org_tree/) | Hierarchical org chart (CEO -> VPs -> Directors -> Managers -> ICs) using seed capture for self-referential generation |
| [Pipeline](https://github.com/codingconcepts/edg/tree/main/_examples/pipeline/) | Multi-table sequential reads and writes |
| [Populate](https://github.com/codingconcepts/edg/tree/main/_examples/populate/) | Billion-row data population benchmark |
| [Prepared](https://github.com/codingconcepts/edg/tree/main/_examples/prepared/) | Prepared statements for reduced parse overhead in high-throughput workloads |
| [Print](https://github.com/codingconcepts/edg/tree/main/_examples/print/) | Live aggregated stats with `print` expressions (frequency, min/avg/max, custom agg) |
| [Reference Data](https://github.com/codingconcepts/edg/tree/main/_examples/reference_data/) | Static reference datasets without database queries |
| [SaaS](https://github.com/codingconcepts/edg/tree/main/_examples/saas/) | Multi-tenant SaaS with tenants, users, projects, and tasks |
| [SEATS](https://github.com/codingconcepts/edg/tree/main/cmd/edg/workload/seats/) | Airline reservation system benchmark with flight booking contention |
| [Social](https://github.com/codingconcepts/edg/tree/main/_examples/social/) | Social network with users, posts, follows, and tags |
| [Stages](https://github.com/codingconcepts/edg/tree/main/_examples/stages/) | Staged execution with different worker counts and durations per phase |
| [Sync](https://github.com/codingconcepts/edg/tree/main/_examples/sync/) | Dual-write consistency testing across databases (CockroachDB + MySQL) with batched verification |
| [Sysbench Insert](https://github.com/codingconcepts/edg/tree/main/cmd/edg/workload/sysbench_insert/) | Pure insert micro-benchmark (`oltp_insert`) for ingestion throughput |
| [Sysbench Point Select](https://github.com/codingconcepts/edg/tree/main/cmd/edg/workload/sysbench_point_select/) | Pure point-select micro-benchmark (`oltp_point_select`) for read latency |
| [Sysbench Read Write](https://github.com/codingconcepts/edg/tree/main/cmd/edg/workload/sysbench_read_write/) | Mixed read-write micro-benchmark (`oltp_read_write`) with scans, updates, and deletes |
| [Sysbench Update Index](https://github.com/codingconcepts/edg/tree/main/cmd/edg/workload/sysbench_update_index/) | Pure indexed-column update micro-benchmark (`oltp_update_index`) |
| [TATP](https://github.com/codingconcepts/edg/tree/main/cmd/edg/workload/tatp/) | Telecom Application Transaction Processing benchmark (80% reads, 20% writes) |
| [TPC-C](https://github.com/codingconcepts/edg/tree/main/_examples/tpcc/) | Full TPC-C benchmark with all 5 transaction profiles |
| [Transaction](https://github.com/codingconcepts/edg/tree/main/_examples/transaction/) | Multi-statement transactions with read-then-write patterns inside BEGIN/COMMIT |
| [Vector](https://github.com/codingconcepts/edg/tree/main/_examples/vector/) | pgvector-compatible embeddings with clustered vectors for similarity search |
| [Workers](https://github.com/codingconcepts/edg/tree/main/_examples/workers/) | Background worker queries on a fixed schedule alongside the main run loop (job queue with lease expiry) |
| [Workload](https://github.com/codingconcepts/edg/tree/main/_examples/workload/) | Built-in workloads (bank, ch, kv, movr, seats, sysbench-insert, sysbench-point-select, sysbench-read-write, sysbench-update-index, tatp, tpcc, tpch, ttlbench, ttllogger, ycsb) without a config file |
| [YCSB](https://github.com/codingconcepts/edg/tree/main/_examples/ycsb/) | Yahoo! Cloud Serving Benchmark with a single usertable and configurable workload profiles |
