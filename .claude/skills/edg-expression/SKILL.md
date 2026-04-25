---
name: edg-expression
description: Help compose edg expressions. Explain functions, debug syntax, and generate the right incantation for a given use case.
user-invocable: true
---

# edg Expression Helper

You help users compose, debug, and understand edg expressions. edg uses [expr-lang/expr](https://github.com/expr-lang/expr) as its expression engine, extended with ~60 built-in functions for data generation, distribution sampling, and reference data access.

## What you do

- Explain what a specific function does and when to use it
- Compose expressions for a described use case
- Debug expression syntax errors
- Suggest the REPL for interactive testing: `edg repl`

## Quick Reference

### Identifiers
| Function | Description |
|---|---|
| `uuid_v7()` | Sortable UUID (preferred for PKs) |
| `uuid_v4()` | Random UUID |
| `seq(start, step)` | Auto-incrementing counter per worker |
| `seq_global(name)` | Shared auto-incrementing counter across all workers (requires `seq:` config section) |
| `seq_rand(name)` | Uniform random from generated sequence values |
| `seq_zipf(name, s, v)` | Zipfian-distributed pick from sequence (hot early values) |
| `seq_norm(name, mean, stddev)` | Normal-distributed pick from sequence |
| `seq_exp(name, rate)` | Exponential-distributed pick from sequence |
| `seq_lognorm(name, mu, sigma)` | Log-normal-distributed pick from sequence |

### Random Data (gofakeit)
| Function | Description |
|---|---|
| `gen('email')` | Random email |
| `gen('firstname')` | Random first name |
| `gen('lastname')` | Random last name |
| `gen('number:1,100')` | Random int in range |
| `gen('sentence:5')` | Random sentence |
| `regex('[A-Z]{3}-[0-9]{4}')` | String matching regex |
| `bool()` | Random true/false |

### Numeric Distributions
| Function | Description |
|---|---|
| `uniform(min, max)` | Flat/even distribution |
| `uniform_f(min, max, precision)` | Uniform float with decimal places |
| `norm(mean, stddev, min, max)` | Bell curve |
| `norm_f(mean, stddev, min, max, precision)` | Bell curve float |
| `exp(rate, min, max)` | Exponential decay |
| `lognorm(mu, sigma, min, max)` | Right-skewed long tail |
| `zipf(s, v, max)` | Power-law hot keys |
| `nurand(A, x, y)` | TPC-C non-uniform random |

### Set Distributions
| Function | Description |
|---|---|
| `set_rand(values, weights)` | Uniform or weighted pick from a set |
| `set_norm(values, mean, stddev)` | Normal distribution over set indices |
| `set_exp(values, rate)` | Exponential over set indices |
| `set_zipf(values, s, v)` | Zipfian over set indices |
| `set_lognorm(values, mu, sigma)` | Log-normal over set indices |

### Dates & Times
| Function | Description |
|---|---|
| `timestamp(min, max)` | Random RFC3339 timestamp |
| `date(format, min, max)` | Random formatted date |
| `date_offset(duration)` | Now +/- duration |
| `duration(min, max)` | Random Go duration string |
| `time(min, max)` | Random HH:MM:SS |

### Strings & Formatting
| Function | Description |
|---|---|
| `template(format, args...)` | Go fmt.Sprintf |
| `json_obj(k1, v1, ...)` | Build JSON object |
| `json_arr(minN, maxN, pattern)` | Build JSON array |
| `array(minN, maxN, pattern)` | PostgreSQL array literal |
| `vector(dims, clusters, spread)` | pgvector literal with clustered unit vectors (uniform) |
| `vector_zipf(dims, clusters, spread, s, v)` | pgvector literal with Zipfian centroid selection |
| `vector_norm(dims, clusters, spread, mean, stddev)` | pgvector literal with normal centroid selection |

### Reference Data
| Function | Scope | Description |
|---|---|---|
| `ref_rand(name)` | None | Fresh random row every call |
| `ref_same(name)` | Per-query | Same row within one query execution |
| `ref_diff(name)` | Per-query | Unique row per call within one query |
| `ref_perm(name)` | Per-worker | Fixed row for worker lifetime |
| `ref_each(query)` | Expansion | One execution per returned row |
| `ref_n(name, field, min, max)` | None | N unique values as CSV string |

### Aggregation (over named datasets)
| Function | Description |
|---|---|
| `count(name)` | Row count |
| `sum(name, field)` | Sum of field |
| `avg(name, field)` | Average of field |
| `min(name, field)` | Minimum of field |
| `max(name, field)` | Maximum of field |
| `distinct(name, field)` | Count of distinct values |

### Conditionals & Dependencies
| Function | Description |
|---|---|
| `arg(index)` | Reference earlier arg by zero-based index |
| `arg('name')` | Reference earlier arg by name (named args only) |
| `cond(pred, trueVal, falseVal)` | Ternary conditional |
| `nullable(expr, probability)` | NULL with given probability |
| `coalesce(v1, v2, ...)` | First non-nil value |
| `const(value)` | Literal constant |
| `fail(message)` | Stop current worker gracefully with error |
| `fatal(message)` | Terminate entire process immediately |

### Correlated Totals
| Function | Description |
|---|---|
| `distribute_sum(total, minN, maxN, precision)` | N random parts summing exactly to total (comma-separated) |
| `distribute_weighted(total, weights, noise, precision)` | Split total by proportional weights with controlled noise (0=exact, 1=random) |

### Multi-Value
| Function | Description |
|---|---|
| `nurand_n(A, x, y, minN, maxN)` | N unique NURand values (comma-separated) |
| `norm_n(mean, stddev, min, max, minN, maxN)` | N unique normally-distributed values (comma-separated) |
| `weighted_sample_n(name, field, weightField, minN, maxN)` | N weighted unique rows from a dataset (comma-separated) |

### Batch
| Function | Description |
|---|---|
| `batch(n)` | Sequential indices [0, n) |
| `gen_batch(total, size, pattern)` | Batched gofakeit values |
| `iter()` | 1-based row counter for batch queries (resets per query) |

### Uniqueness
| Function | Description |
|---|---|
| `uniq(expression)` | Retry expression until unique value found (100 attempts default) |
| `uniq(expression, maxRetries)` | Same, with custom retry limit |

### PII & Masking
| Function | Description |
|---|---|
| `gen_locale('first_name', 'ja_JP')` | Locale-aware name/address/phone generation |
| `gen_locale('name', 'de_DE')` | Full name in locale order (eastern = last+first, western = first last) |
| `mask(value)` | Deterministic 16-char hex token (same input → same output) |
| `mask(value, length)` | Hex token truncated to `length` chars |
| `mask(value, 'base64')` | Base64-encoded token |
| `mask(value, 'base32')` | Base32-encoded token |
| `mask(value, 'asterisk')` | Repeated `*` characters (default 16) |
| `mask(value, 'asterisk', 4)` | 4 asterisks |
| `mask(value, 'redact')` | Fixed string `[REDACTED]`, length ignored |
| `mask(value, 'email')` | Preserves `@domain`, masks local part with `*` |
| `mask(value, 'email', 4)` | `****@domain.com` |

Supported locales: `en_US`, `ja_JP`, `de_DE`, `fr_FR`, `es_ES`, `pt_BR`, `zh_CN`, `ko_KR` (aliases: `ja`, `de`, `fr`, etc.)

### Other
| Function | Description |
|---|---|
| `blob(n)` | Random n bytes as raw binary (cross-database) |
| `bytes(n)` | Hex-encoded random bytes (pgx only) |
| `bit(n)` | Fixed-length bit string |
| `varbit(n)` | Variable-length bit string |
| `inet(cidr)` | Random IP in CIDR block |
| `ltree(parts...)` | PostgreSQL ltree path from dot-joined parts |
| `point(lat, lon, radiusKM)` | Random geographic point (map) |
| `point_wkt(lat, lon, radiusKM)` | Random point as WKT string |

## Common Patterns

### Mutually exclusive columns
```yaml
args:
  - bool()                            # coin flip
  - cond(arg(0), gen('email'), nil)   # email if true
  - cond(!arg(0), gen('phone'), nil)  # phone if false
```

### Computed total from earlier args
```yaml
# Positional
args:
  - uniform_f(1.00, 99.99, 2)   # price
  - gen('number:1,10')           # quantity
  - arg(0) * float(arg(1))      # total = price * qty

# Named (equivalent)
args:
  price: uniform_f(1.00, 99.99, 2)
  quantity: gen('number:1,10')
  total: arg('price') * float(arg('quantity'))
```

### Full name from parts
```yaml
# Positional
args:
  - gen('firstname')
  - gen('lastname')
  - arg(0) + ' ' + arg(1)       # "Alice Smith"

# Named (equivalent)
args:
  first: gen('firstname')
  last: gen('lastname')
  full: arg('first') + ' ' + arg('last')
```

### Weighted category selection
```yaml
args:
  - set_rand(['electronics', 'clothing', 'books', 'food'], [40, 30, 20, 10])
```

### Hot-key access (Zipfian)
```yaml
args:
  - zipf(2.0, 1.0, 999)  # value 0 is most frequent
```

### Worker-pinned partition
```yaml
args:
  - ref_perm('fetch_warehouses').w_id  # same warehouse for entire worker lifetime
```

### Invoice line items (distribute_sum)
```yaml
args:
  - uuid_v4()                           # invoice id
  - uniform_f(100, 10000, 2)            # total
  - distribute_sum(arg(1), 3, 7, 2)     # 3-7 amounts summing to total
# SQL: unnest(string_to_array('$3', ',')::NUMERIC[])
```

### Subtotal/tax/shipping breakdown (distribute_weighted)
```yaml
args:
  - uniform_f(100, 10000, 2)                           # total
  - distribute_weighted(arg(0), [85, 10, 5], 0.1, 2)   # ~85/10/5 split with 10% noise
# SQL: split_part('$2', ',', 1)::NUMERIC  -- subtotal
#      split_part('$2', ',', 2)::NUMERIC  -- tax
#      split_part('$2', ',', 3)::NUMERIC  -- shipping
```

### Masking PII
```yaml
args:
  first_name: gen_locale('first_name', 'ja_JP')
  last_name: gen_locale('last_name', 'ja_JP')
  full_name: arg('last_name') + arg('first_name')
  email: gen('email')
  masked_name: mask(arg('full_name'))                # hex token
  masked_email: mask(arg('email'), 'email', 4)       # ****@example.com
  redacted_phone: mask(arg('phone'), 'redact')       # [REDACTED]
```

### Unique codes in batch inserts
```yaml
# 200 airports each with a unique 3-char IATA code
- name: populate_airport
  type: exec_batch
  count: 200
  size: 100
  args:
    - iter()
    - uniq("gen('airlineairportiata')")
  query: |-
    INSERT INTO airport (id, code) VALUES ($1::INT, $2)
```

### Error handling with fail/fatal
```yaml
args:
  # Stop worker if map lookup misses
  - {'us': 'us-east-1', 'eu': 'eu-west-1'}[env('REGION')] ?? fail('unknown REGION')

  # Kill entire process on critical misconfiguration
  - fatal('missing required config')
```

`fail()` stops only the current worker; `fatal()` terminates the entire process.

## Debugging Tips

- Use `edg repl` to test any expression interactively without a database
- Expressions are compiled at startup; syntax errors will be caught immediately
- `arg(index)` is zero-based and only works within the same query's args list; `arg('name')` works with named args (map-style `args:`)
- Named and positional arg forms are mutually exclusive per query
- `ref_same` resets between queries; `ref_perm` never resets
- Set distributions accept expr-lang array literals: `['a', 'b', 'c']`
- Weights in `set_rand` are relative, not percentages (`[40, 30, 20, 10]` and `[4, 3, 2, 1]` behave identically)
