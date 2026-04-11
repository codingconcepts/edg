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
| `cond(pred, trueVal, falseVal)` | Ternary conditional |
| `nullable(expr, probability)` | NULL with given probability |
| `coalesce(v1, v2, ...)` | First non-nil value |
| `const(value)` | Literal constant |

### Batch
| Function | Description |
|---|---|
| `batch(n)` | Sequential indices [0, n) |
| `gen_batch(total, size, pattern)` | Batched gofakeit values |

### Other
| Function | Description |
|---|---|
| `bytes(n)` | Hex-encoded random bytes |
| `bit(n)` | Fixed-length bit string |
| `varbit(n)` | Variable-length bit string |
| `inet(cidr)` | Random IP in CIDR block |
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
args:
  - uniform_f(1.00, 99.99, 2)   # price
  - gen('number:1,10')           # quantity
  - arg(0) * float(arg(1))      # total = price * qty
```

### Full name from parts
```yaml
args:
  - gen('firstname')
  - gen('lastname')
  - arg(0) + ' ' + arg(1)       # "Alice Smith"
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

## Debugging Tips

- Use `edg repl` to test any expression interactively without a database
- Expressions are compiled at startup; syntax errors will be caught immediately
- `arg(index)` is zero-based and only works within the same query's args list
- `ref_same` resets between queries; `ref_perm` never resets
- Set distributions accept expr-lang array literals: `['a', 'b', 'c']`
- Weights in `set_rand` are relative, not percentages (`[40, 30, 20, 10]` and `[4, 3, 2, 1]` behave identically)
