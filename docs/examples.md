---
title: Examples
layout: default
nav_order: 6
---

# Examples

## Argument Expressions

```yaml
args:
  # Always passes the integer 42.
  - const(42)

  # Evaluates the expression warehouses * 10 using globals; both forms are equivalent.
  - expr(warehouses * 10)
  - warehouses * 10

  # Generates a random integer between 1 and 10 using gofakeit.
  - gen('number:1,10')

  # Returns a random row from the 'fetch_warehouses' init query, pinned to this
  # worker for its lifetime, and extracts the w_id field.
  - ref_perm('fetch_warehouses').w_id

  # Non-uniform random int using TPC-C NURand.
  - nurand(1023, 1, customers / districts)

  # Generates between 5 and 15 unique NURand values as a comma-separated string.
  - nurand_n(8191, 1, items, 5, 15)

  # Drives batched execution: the parent query runs 10 times with $1 = 0..9.
  - batch(customers / batch_size)

  # Generates 100,000 unique emails via gofakeit, split into batches of 10,000.
  - gen_batch(customers, batch_size, 'email')

  # Picks a random payment method with uniform probability.
  - set_rand(['credit_card', 'debit_card', 'paypal'], [])

  # Picks a random rating weighted toward 4 and 5 stars.
  - set_rand(['1', '2', '3', '4', '5'], [5, 10, 20, 35, 30])

  # Picks a quantity using normal distribution.
  # mean=2 (value '3' at index 2 is most common), stddev=0.8 (~68% pick indices 1-3).
  - set_norm([1, 2, 3, 4, 5], 2, 0.8)

  # Picks a priority level using exponential distribution.
  # Higher rate concentrates picks toward the first item ('low').
  - set_exp(['low', 'medium', 'high', 'critical'], 0.5)

  # Picks a tier using log-normal distribution (right-skewed toward early indices).
  - set_lognorm(['free', 'basic', 'pro', 'enterprise'], 0.5, 0.5)

  # Picks a category using Zipfian distribution (strong skew toward first items).
  - set_zipf(['electronics', 'clothing', 'books', 'food', 'toys'], 2.0, 1.0)

  # Generates a random UUID v4 (random) or v7 (time-ordered, sortable).
  - uuid_v4()
  - uuid_v7()

  # Random float between 0.01 and 999.99 with 2 decimal places (e.g. for prices).
  - uniform_f(0.01, 999.99, 2)

  # Uniform random float between 0 and 1 (e.g. for percentages).
  - uniform(0, 1)

  # Auto-incrementing sequence: 1, 2, 3, ... (shared across calls for the worker).
  - seq(1, 1)

  # Auto-incrementing with custom start and step: 100, 110, 120, ...
  - seq(100, 10)

  # Formatted order number using template and seq: "ORD-00001", "ORD-00002", ...
  - template('ORD-%05d', seq(1, 1))

  # Zipfian distribution: hot-key pattern where value 0 is most frequent.
  # s=2.0 controls skew (higher = more skewed), v=1.0, max=999.
  - zipf(2.0, 1.0, 999)

  # Normally-distributed integer review rating centred on 4, mostly 3-5.
  - norm(4, 1, 1, 5)

  # Normally-distributed float price centred on 50.00, rounded to 2 decimal places.
  - norm_f(50.0, 15.0, 1.0, 100.0, 2)

  # Conditional value based on a random roll.
  - cond(gen('number:1,100') > 95, 'premium', 'standard')

  # First non-nil fallback value.
  - coalesce(ref_rand('optional_data').value, 'default')

  # Generate a product code matching a regex pattern.
  - regex('[A-Z]{3}-[0-9]{4}')

  # Build a JSON metadata object for a JSONB column.
  - json_obj('source', 'web', 'version', 2, 'active', true)

  # Build a JSON array of 1-5 random email addresses.
  - json_arr(1, 5, 'email')

  # Random geographic point within 10km of London, access lat/lon separately.
  - point(51.5074, -0.1278, 10.0).lat
  - point(51.5074, -0.1278, 10.0).lon

  # Random geographic point as WKT for native geometry columns.
  - point_wkt(51.5074, -0.1278, 10.0)

  # Random timestamp between two dates (RFC3339 format).
  - timestamp('2020-01-01T00:00:00Z', '2025-01-01T00:00:00Z')

  # Random date formatted as YYYY-MM-DD.
  - date('2006-01-02', '2020-01-01T00:00:00Z', '2025-01-01T00:00:00Z')

  # Timestamp 72 hours in the past (e.g. for TTL or expiry columns).
  - date_offset('-72h')

  # Random duration between 1 hour and 24 hours.
  - duration('1h', '24h')

  # Pick 3-8 products weighted by their popularity column.
  - weighted_sample_n('fetch_products', 'id', 'popularity', 3, 8)

  # Regex: generate a random US phone number.
  - regex('\\([0-9]{3}\\) [0-9]{3}-[0-9]{4}')

  # Regex: generate a random hex colour code.
  - regex('#[0-9a-f]{6}')

  # Regex: generate a random IPv4 address.
  - regex('[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}')

  # Regex: generate a random MAC address.
  - regex('[0-9a-f]{2}(:[0-9a-f]{2}){5}')

  # Regex: generate a random license plate (e.g. "AB12 CDE").
  - regex('[A-Z]{2}[0-9]{2} [A-Z]{3}')

  # Exponentially-distributed float in [0, 100] with 2 decimal places.
  - exp_f(0.5, 0, 100, 2)

  # Log-normally-distributed float in [1, 1000] with 2 decimal places.
  - lognorm_f(1.0, 0.5, 1, 1000, 2)

  # Random IP address within a CIDR block (e.g. for network simulation).
  - inet('192.168.1.0/24')

  # Random 16 bytes as a hex-encoded CockroachDB/PostgreSQL BYTES literal.
  - bytes(16)

  # Random fixed-length bit string of 8 bits (e.g. "10110011").
  - bit(8)

  # Random variable-length bit string of 1-16 bits.
  - varbit(16)

  # PostgreSQL/CockroachDB array literal with 2-5 random email addresses.
  - array(2, 5, 'email')

  # Random time of day between 08:00 and 18:00 (HH:MM:SS format).
  - time('08:00:00', '18:00:00')

  # Random time of day with timezone suffix (for TIMETZ columns).
  - timez('09:00:00', '17:00:00')

  # Sum of the 'price' field across all rows in the 'fetch_products' dataset.
  - sum('fetch_products', 'price')

  # Average price across all products.
  - avg('fetch_products', 'price')

  # Minimum and maximum price in the dataset.
  - min('fetch_products', 'price')
  - max('fetch_products', 'price')

  # Total number of rows in the dataset.
  - count('fetch_products')

  # Number of distinct category IDs across all products.
  - distinct('fetch_products', 'category_id')
```

## Example Workloads

| Workload | Description |
|---|---|
| [TPC-C](https://github.com/codingconcepts/edg/tree/main/_examples/tpcc/) | Full TPC-C benchmark with all 5 transaction profiles |
| [Bank](https://github.com/codingconcepts/edg/tree/main/_examples/bank/) | Bank account operations for contention and correctness testing |
| [E-Commerce](https://github.com/codingconcepts/edg/tree/main/_examples/ecommerce/) | E-commerce with categories, products, customers, and orders |
| [IoT](https://github.com/codingconcepts/edg/tree/main/_examples/iot/) | IoT devices, sensors, and time-series readings |
| [Normal](https://github.com/codingconcepts/edg/tree/main/_examples/normal/) | Product reviews with normal distribution ratings |
| [Pipeline](https://github.com/codingconcepts/edg/tree/main/_examples/pipeline/) | Multi-table sequential reads and writes |
| [SaaS](https://github.com/codingconcepts/edg/tree/main/_examples/saas/) | Multi-tenant SaaS with tenants, users, projects, and tasks |
| [Populate](https://github.com/codingconcepts/edg/tree/main/_examples/populate/) | Billion-row data population benchmark |
| [Social](https://github.com/codingconcepts/edg/tree/main/_examples/social/) | Social network with users, posts, follows, and tags |
| [Aggregation](https://github.com/codingconcepts/edg/tree/main/_examples/aggregation/) | Demonstrates aggregation functions (sum, avg, min, max, count, distinct) |
| [Reference Data](https://github.com/codingconcepts/edg/tree/main/_examples/reference_data/) | Static reference datasets without database queries |
| [Expressions](https://github.com/codingconcepts/edg/tree/main/_examples/expression/) | Demonstrates expr-lang built-in features (array, map, string, bitwise, etc.) |
