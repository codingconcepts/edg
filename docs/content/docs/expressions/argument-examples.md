---
title: Argument Examples
weight: 4
---

# Argument Expression Examples

These expressions are used in the `args:` list of a `run` query. Each entry in `args:` generates a value that is bound to a query parameter (`$1`, `$2`, etc.).

## Aggregation

| Expression | Description |
|---|---|
| `avg('fetch_products', 'price')` | Average price across all products |
| `count('fetch_products')` | Total number of rows in the dataset |
| `distinct('fetch_products', 'category_id')` | Number of distinct category IDs across all products |
| `max('fetch_products', 'price')` | Maximum price in the dataset |
| `min('fetch_products', 'price')` | Minimum price in the dataset |
| `sum('fetch_products', 'price')` | Sum of the price field across all rows |

## Batch operations

| Expression | Description |
|---|---|
| `batch(customers / batch_size)` | Drives batched execution: the parent query runs N times with $1 = 0..N-1 |
| `gen_batch(customers, batch_size, 'email')` | Generates unique emails via gofakeit, split into batches |
| `string_to_array('$1', __sep__)` | Splits a batch-expanded placeholder back into rows using the driver-aware separator |

> [!NOTE]
> Always use `__sep__` instead of a literal comma delimiter when splitting batch args. Generated values (names, addresses, etc.) can contain commas, which would silently split a single value into multiple rows and corrupt your data. `__sep__` uses the ASCII unit separator (char 31), which never appears in generated text.

## Binary

| Expression | Description |
|---|---|
| `bit(8)` | Random fixed-length bit string of 8 bits (e.g. `10110011`) |
| `blob(1024)` | Random 1KB blob as raw binary data (works across all databases) |
| `bytes(16)` | Random 16 bytes as a hex-encoded CockroachDB/PostgreSQL BYTES literal |
| `varbit(16)` | Random variable-length bit string of 1-16 bits |

## Conditionals & dependent columns

| Expression | Description |
|---|---|
| `arg(0) * float(arg(1))` | Compute a total from previously generated price and quantity |
| `arg(0) + " " + arg(1)` | Concatenate previously generated firstname and lastname |
| `arg('price') * float(arg('qty'))` | Same as above using [named args]({{< relref "configuration#named-args" >}}) |
| `coalesce(ref_rand('optional_data').value, 'default')` | First non-nil fallback value |
| `cond(arg(0), gen('email'), nil)` | Email if coin flip is true, NULL if false |
| `cond(gen('number:1,100') > 95, 'premium', 'standard')` | Conditional value based on a random roll |
| `{'fra': 'eu-central-1', 'sin': 'ap-southeast-1'}[env('FLY_REGION')] ?? fail('bad region')` | Map lookup with error on unknown value |
| `fail('unexpected value')` | Stop worker gracefully with an error message |
| `fatal('missing required config')` | Terminate entire process immediately |
| `nullable(gen('email'), 0.3)` | 30% chance of NULL, otherwise a random email |

## Constants & globals

| Expression | Description |
|---|---|
| `const(42)` | Always passes the integer 42 |
| `expr(warehouses * 10)` | Evaluates an arithmetic expression using globals |
| `global('warehouses')` | Looks up a global by name (equivalent to using the variable directly) |
| `int(coalesce(env_nil('CUSTOMERS'), 10000))` | Environment variable with default fallback, converted to int |
| `warehouses * 10` | Direct global reference in an expression (equivalent to `expr(...)`) |

## Dates & times

| Expression | Description |
|---|---|
| `date('2006-01-02', '2020-01-01T00:00:00Z', '2025-01-01T00:00:00Z')` | Random date formatted as YYYY-MM-DD |
| `date_offset('-72h')` | Timestamp 72 hours in the past (e.g. for TTL or expiry columns) |
| `duration('1h', '24h')` | Random duration between 1 hour and 24 hours |
| `time('08:00:00', '18:00:00')` | Random time of day between 08:00 and 18:00 (HH:MM:SS format) |
| `timestamp('2020-01-01T00:00:00Z', '2025-01-01T00:00:00Z')` | Random timestamp between two dates (RFC3339 format) |
| `timez('09:00:00', '17:00:00')` | Random time of day with timezone suffix (for TIMETZ columns) |

## Geographic

| Expression | Description |
|---|---|
| `point(51.5074, -0.1278, 10.0).lat` | Random geographic point within 10km of London, latitude |
| `point(51.5074, -0.1278, 10.0).lon` | Random geographic point within 10km of London, longitude |
| `point_wkt(51.5074, -0.1278, 10.0)` | Random geographic point as WKT for native geometry columns |

## Hierarchical (ltree)

| Expression | Description |
|---|---|
| `ltree('Top', 'Science', 'Astronomy')` | PostgreSQL/CockroachDB ltree path: `Top.Science.Astronomy` |
| `ltree(arg('name'))` | Single-label root path from a previously generated name |
| `ltree(ref_rand('parent').path, arg('name'))` | Append a new label to a parent's path for hierarchical data |
| `ltree(gen('word'), gen('word'), gen('word'))` | Random 3-level path from generated words |

> [!NOTE]
> Invalid ltree characters (hyphens, spaces, etc.) are automatically replaced with underscores. Nil and empty parts are skipped.

## Identifiers

| Expression | Description |
|---|---|
| `seq(1, 1)` | Auto-incrementing sequence: 1, 2, 3, ... (per worker) |
| `seq(100, 10)` | Auto-incrementing with custom start and step: 100, 110, 120, ... |
| `uuid_v1()` | Random UUID v1 (timestamp + node ID) |
| `uuid_v4()` | Random UUID v4 (random) |
| `uuid_v6()` | Random UUID v6 (reordered timestamp) |
| `uuid_v7()` | Random UUID v7 (time-ordered, sortable) |

## JSON & arrays

| Expression | Description |
|---|---|
| `array(2, 5, 'email')` | PostgreSQL/CockroachDB array literal with 2-5 random email addresses |
| `json_arr(1, 5, 'email')` | JSON array of 1-5 random email addresses |
| `json_obj('source', 'web', 'version', 2, 'active', true)` | JSON metadata object for a JSONB column |

## Network

| Expression | Description |
|---|---|
| `inet('192.168.1.0/24')` | Random IP address within a CIDR block |

## Correlated totals

| Expression | Description |
|---|---|
| `distribute_sum(100.00, 3, 7, 2)` | 3-7 random amounts that sum exactly to 100.00, each with 2 decimal places |
| `distribute_sum(arg(1), 3, 7, 2)` | Partition a previously computed total across 3-7 child values |
| `distribute_sum(ref_same('invoices').total, 3, 7, 2)` | Partition an invoice's total into line item amounts |
| `distribute_weighted(1000, [50, 30, 20], 0, 2)` | Exact 50/30/20 split: `500.00,300.00,200.00` |
| `distribute_weighted(1000, [50, 30, 20], 0.3, 2)` | Approximate 50/30/20 split with 30% noise |
| `distribute_weighted(arg(1), [7, 2, 1], 0.1, 2)` | Split a parent value roughly 70/20/10 |

## Numeric distributions

| Expression | Description |
|---|---|
| `exp(0.5, 0, 100)` | Exponentially-distributed integer in [0, 100] |
| `exp_f(0.5, 0, 100, 2)` | Exponentially-distributed float in [0, 100] with 2 decimal places |
| `lognorm(1.0, 0.5, 1, 1000)` | Log-normally-distributed integer in [1, 1000] |
| `lognorm_f(1.0, 0.5, 1, 1000, 2)` | Log-normally-distributed float in [1, 1000] with 2 decimal places |
| `norm(4, 1, 1, 5)` | Normally-distributed integer review rating centred on 4, mostly 3-5 |
| `norm_f(50.0, 15.0, 1.0, 100.0, 2)` | Normally-distributed float price centred on 50.00, 2 decimal places |
| `norm_n(50.0, 10.0, 1, 100, 5, 10)` | 5-10 unique normally-distributed values as a comma-separated string |
| `nurand(1023, 1, customers / districts)` | Non-uniform random int using TPC-C NURand |
| `nurand_n(8191, 1, items, 5, 15)` | 5-15 unique NURand values as a comma-separated string |
| `uniform(0, 1)` | Uniform random float between 0 and 1 (e.g. for percentages) |
| `uniform_f(0.01, 999.99, 2)` | Random float between 0.01 and 999.99 with 2 decimal places |
| `zipf(2.0, 1.0, 999)` | Zipfian distribution: hot-key pattern where value 0 is most frequent |

## PII & locale

| Expression | Description |
|---|---|
| `gen_locale('first_name', 'ja_JP')` | Japanese first name (e.g. 太郎, 花子) |
| `gen_locale('last_name', 'de_DE')` | German last name (e.g. Müller, Schmidt) |
| `gen_locale('name', 'ja_JP')` | Full name in locale order (東 = 佐藤太郎, 西 = Hans Müller) |
| `gen_locale('city', 'fr_FR')` | French city name (e.g. Paris, Lyon) |
| `gen_locale('street', 'es_ES')` | Spanish street name (e.g. Gran Vía) |
| `gen_locale('phone', 'ko_KR')` | Korean phone number (e.g. 010-1234-5678) |
| `gen_locale('zip', 'ja_JP')` | Japanese postal code (e.g. 123-4567) |
| `gen_locale('address', 'de_DE')` | Full German address with street number, city, and zip |
| `mask('john@example.com')` | Deterministic 16-char hex token (same input → same output) |
| `mask(arg('email'), 8)` | 8-char masked token of a previously generated email |

Supported locales: `en_US`, `ja_JP`, `de_DE`, `fr_FR`, `es_ES`, `pt_BR`, `zh_CN`, `ko_KR`. Aliases like `ja`, `de`, `fr` also work.

## Random values

| Expression | Description |
|---|---|
| `bool()` | Random true or false |
| `gen('number:1,10')` | Random integer between 1 and 10 using gofakeit |
| `regex('#[0-9a-f]{6}')` | Random hex colour code |
| `regex('[A-Z]{2}[0-9]{2} [A-Z]{3}')` | Random license plate (e.g. "AB12 CDE") |
| `regex('[A-Z]{3}-[0-9]{4}')` | Product code matching a regex pattern |
| `regex('[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}')` | Random IPv4 address |
| `regex('[0-9a-f]{2}(:[0-9a-f]{2}){5}')` | Random MAC address |
| `regex('\\([0-9]{3}\\) [0-9]{3}-[0-9]{4}')` | Random US phone number |

## Reference data

| Expression | Description |
|---|---|
| `ref_diff('fetch_warehouses').w_id` | Unique row on each call within a query (no repeats) |
| `ref_each('SELECT id FROM warehouses ORDER BY id')` | Executes a SQL query; each row becomes a separate arg set |
| `ref_n('fetch_warehouses', 'id', 3, 8)` | Picks 3-8 unique random rows, returns comma-separated field values |
| `ref_perm('fetch_warehouses').w_id` | Random row pinned to this worker for its lifetime |
| `ref_rand('fetch_warehouses').w_id` | Random row from the dataset |
| `ref_same('fetch_warehouses').w_id` | Same random row for all ref_same calls within a single query execution |
| `weighted_sample_n('fetch_products', 'id', 'popularity', 3, 8)` | Pick 3-8 products weighted by their popularity column |

## Set distributions

| Expression | Description |
|---|---|
| `set_exp(['low', 'medium', 'high', 'critical'], 0.5)` | Exponential distribution; concentrates picks toward first item |
| `set_lognorm(['free', 'basic', 'pro', 'enterprise'], 0.5, 0.5)` | Log-normal distribution (right-skewed toward early indices) |
| `set_norm([1, 2, 3, 4, 5], 2, 0.8)` | Normal distribution; index 2 is most common |
| `set_rand(['1', '2', '3', '4', '5'], [5, 10, 20, 35, 30])` | Weighted random; skewed toward 4 and 5 stars |
| `set_rand(['credit_card', 'debit_card', 'paypal'], [])` | Uniform random payment method selection |
| `set_zipf(['electronics', 'clothing', 'books', 'food', 'toys'], 2.0, 1.0)` | Zipfian distribution; strong skew toward first items |

## Strings & formatting

| Expression | Description |
|---|---|
| `template('ORD-%05d', seq(1, 1))` | Formatted order number: "ORD-00001", "ORD-00002", ... |

## Vectors

| Expression | Description |
|---|---|
| `vector(32, 3, 0.3)` | 32-dimensional vector for testing; higher spread = more cluster overlap |
| `vector(384, 5, 0.1)` | pgvector-compatible 384-dimensional vector with 5 clusters and tight spread |
| `vector_norm(384, 5, 0.1, 2.0, 0.8)` | Normal centroid selection: cluster 2 is most common, bell curve falloff |
| `vector_zipf(384, 10, 0.1, 2.0, 1.0)` | Zipfian centroid selection: cluster 0 is the "hottest", realistic skew |
