---
title: Example Workloads
weight: 6
---

# Example Workloads

Complete workload configs you can run directly or use as starting points for your own.

| Workload | Description |
|---|---|
| [Aggregation](https://github.com/codingconcepts/edg/tree/main/_examples/aggregation/) | Demonstrates aggregation functions (sum, avg, min, max, count, distinct) |
| [Bank](https://github.com/codingconcepts/edg/tree/main/_examples/bank/) | Bank account operations for contention and correctness testing |
| [Blob](https://github.com/codingconcepts/edg/tree/main/_examples/blob/) | Binary data with `blob()` (all databases) and `bytes()` (PostgreSQL/CockroachDB) |
| [Batch](https://github.com/codingconcepts/edg/tree/main/_examples/batch/) | Demonstrates `query_batch` and `exec_batch` query types for batch inserts and updates |
| [Distributions](https://github.com/codingconcepts/edg/tree/main/_examples/distributions/) | All five distribution functions (uniform, zipf, norm_f, exp_f, lognorm_f) |
| [Each Cartesian](https://github.com/codingconcepts/edg/tree/main/_examples/each_cartesian/) | Cartesian product seeding with `ref_each` across multiple tables |
| [E-Commerce](https://github.com/codingconcepts/edg/tree/main/_examples/ecommerce/) | E-commerce with categories, products, customers, and orders |
| [Environment](https://github.com/codingconcepts/edg/tree/main/_examples/environment/) | Demonstrates fetching and using environment variables |
| [Exclusive Columns](https://github.com/codingconcepts/edg/tree/main/_examples/exclusive_columns/) | Mutually exclusive columns - either col_a or col_b, never both |
| [Expectations](https://github.com/codingconcepts/edg/tree/main/_examples/expectations/) | Post-run assertions for CI/CD gating on error rate, latency, and throughput |
| [Expressions](https://github.com/codingconcepts/edg/tree/main/_examples/expression/) | Demonstrates expr-lang built-in features (array, map, string, bitwise, etc.) |
| [Includes](https://github.com/codingconcepts/edg/tree/main/_examples/includes/) | Splitting and reusing config fragments with the `!include` directive |
| [IoT](https://github.com/codingconcepts/edg/tree/main/_examples/iot/) | IoT devices, sensors, and time-series readings |
| [Normal](https://github.com/codingconcepts/edg/tree/main/_examples/normal/) | Product reviews with normal distribution ratings |
| [Nullable](https://github.com/codingconcepts/edg/tree/main/_examples/nullable/) | Demonstrates `nullable(expr, probability)` for injecting NULLs with controlled frequency |
| [Pipeline](https://github.com/codingconcepts/edg/tree/main/_examples/pipeline/) | Multi-table sequential reads and writes |
| [Populate](https://github.com/codingconcepts/edg/tree/main/_examples/populate/) | Billion-row data population benchmark |
| [Prepared](https://github.com/codingconcepts/edg/tree/main/_examples/prepared/) | Prepared statements for reduced parse overhead in high-throughput workloads |
| [Reference Data](https://github.com/codingconcepts/edg/tree/main/_examples/reference_data/) | Static reference datasets without database queries |
| [SaaS](https://github.com/codingconcepts/edg/tree/main/_examples/saas/) | Multi-tenant SaaS with tenants, users, projects, and tasks |
| [Social](https://github.com/codingconcepts/edg/tree/main/_examples/social/) | Social network with users, posts, follows, and tags |
| [Stages](https://github.com/codingconcepts/edg/tree/main/_examples/stages/) | Staged execution with different worker counts and durations per phase |
| [Transaction](https://github.com/codingconcepts/edg/tree/main/_examples/transaction/) | Multi-statement transactions with read-then-write patterns inside BEGIN/COMMIT |
| [TPC-C](https://github.com/codingconcepts/edg/tree/main/_examples/tpcc/) | Full TPC-C benchmark with all 5 transaction profiles |
| [Vector](https://github.com/codingconcepts/edg/tree/main/_examples/vector/) | pgvector-compatible embeddings with clustered vectors for similarity search |
| [Workload](https://github.com/codingconcepts/edg/tree/main/_examples/workload/) | Built-in workloads (bank, tpcc, ycsb) without a config file |
| [YCSB](https://github.com/codingconcepts/edg/tree/main/_examples/ycsb/) | Yahoo! Cloud Serving Benchmark with a single usertable and configurable workload profiles |
