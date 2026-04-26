# Print

Demonstrates the `print` field on queries, which evaluates expressions each iteration and displays aggregated values in the progress and summary output.

Print entries can be a simple string (auto-aggregated) or a map with `expr` and `agg` fields for custom aggregation. Print expressions have access to the same context as query args: `ref_same`, `ref_rand`, `arg()`, `global()`, `local()`, and all built-in functions.

## Simple form (auto-aggregate)

A plain string auto-detects the value type: categorical values show frequency distributions, numeric values show min/avg/max.

```yaml
print:
  - ref_same('regions').name
```

## Custom aggregation

Use the map form to write an `agg` expression that controls the output. The `agg` expression has access to these variables:

| Variable | Type | Description |
|---|---|---|
| `count` | int | Total observations |
| `freq` | map[string]int | Value frequency distribution |
| `min` | float | Minimum numeric value |
| `max` | float | Maximum numeric value |
| `avg` | float | Mean of numeric values |
| `sum` | float | Sum of numeric values |

All [expr-lang](https://expr-lang.org/docs/language-definition) functions are available in `agg` expressions (`toPairs`, `sortBy`, `join`, `map`, `string`, `int`, etc.).

### Examples

**Frequency distribution** - top 5 values sorted by count:

```yaml
- expr: ref_same('regions').name
  agg: "join(map(sortBy(toPairs(freq), -#[1])[:5], #[0] + '=' + string(#[1])), ' ')"
# us=340 eu=330 ap=330
```

**Range with average** - formatted numeric summary:

```yaml
- expr: arg(3)
  agg: "'avg $' + string(int(avg)) + ' n=' + string(count)"
# avg $250 n=1000
```

**Total count** - simple observation counter:

```yaml
- expr: arg(0)
  agg: "string(count)"
# 1000
```

**Sum** - running total:

```yaml
- expr: arg(3)
  agg: "'total=$' + string(int(sum))"
# total=$250317
```

**Min/max** - value bounds:

```yaml
- expr: arg(3)
  agg: "string(int(min)) + ' - ' + string(int(max))"
# 1 - 499
```

**Unique count** - number of distinct values:

```yaml
- expr: ref_same('regions').name
  agg: "string(len(freq)) + ' unique'"
# 3 unique
```

## Example output

```
PRINT         VALUES
insert_order  us=340 eu=330 ap=330
insert_order  chicago=120 tokyo=115 london=110 dallas=108 paris=105
insert_order  avg $250 n=1000
read_order    us=340 eu=330 ap=330
read_order    1000
```

> [!NOTE]
> Print expressions using `ref_same` see the same row selected for the query args in that iteration, so `ref_same('regions').name` in `print` matches `ref_same('regions').name` in `args`.

## CockroachDB

### Setup

```sh
docker compose -f cmd/harness/compose/compose_crdb.yml up -d
```

### Run

```sh
go run ./cmd/edg all \
--driver pgx \
--config _examples/print/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 10 \
-d 30s
```
