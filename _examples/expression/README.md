# Expressions

Demonstrates the built-in [expr-lang](https://expr-lang.org/docs/language-definition) features available in any query argument or user-defined expression. These complement edg's custom functions (gen, ref_rand, distributions, etc.) and can be freely combined with them.

## Features

| Category | Functions / Operators |
|---|---|
| Array predicates | `all`, `any`, `one`, `none` |
| Array search | `find`, `findIndex`, `findLast`, `findLastIndex` |
| Array transform | `filter`, `map`, `sort`, `sortBy`, `reverse`, `uniq`, `concat`, `flatten` |
| Array aggregate | `reduce`, `mean`, `median`, `first`, `last`, `take` |
| Map | `keys`, `values`, `groupBy` |
| String operators | `contains`, `startsWith`, `endsWith`, `matches`, `in` |
| String functions | `trimPrefix`, `trimSuffix`, `splitAfter`, `repeat`, `indexOf`, `lastIndexOf`, `hasPrefix`, `hasSuffix` |
| Type conversion | `type`, `toJSON`, `fromJSON`, `toBase64`, `fromBase64`, `toPairs`, `fromPairs` |
| Operators | `..` (range), `[:]` (slice), `?.` (optional chaining), `??` (nil coalescing), `if`/`else` |
| Bitwise | `bitand`, `bitor`, `bitxor`, `bitnand`, `bitnot`, `bitshl`, `bitshr`, `bitushr` |
| Language | `let` bindings, `#` predicates, closures |
| Misc | `len`, `get` |

> **Note:** edg's custom `sum`, `min`, `max`, `count`, `date`, and `duration` functions shadow the expr built-ins of the same name. Use `reduce()` for array totals and the edg-specific functions for dates and distributions.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d
docker exec -it node1 cockroach init --insecure
docker exec -it node1 cockroach sql --insecure
```

### Run

```sh
edg up \
--driver pgx \
--config _examples/expression/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg run \
--driver pgx \
--config _examples/expression/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable" \
-w 1 \
-d 5s
```

### Verify

```sql
SELECT label, result FROM expr_demo ORDER BY label;
```

```
        label        |                  result
---------------------+-------------------------------------------
 all(# > 0)          | true
 any(# > 25)         | true
 bitand(0xFF, 0x0F)  | 15
 contains            | true
 filter+map          | 19.99,29.99,14.99
 find(# > 20)        | 29.99
 first               | Alice
 if/else             | B
 keys                | a,b,c
 len(array)          | 5
 let bindings        | 25
 mean                | 15.992
 range 1..5          | 15
 reduce(#acc + #)    | 79.95
 reverse             | Eve,Diana,Charlie,Bob,Alice
 slice [1:3]         | Bob,Charlie
 sort                | 4.99,9.99,14.99,19.99,29.99
 toBase64            | aGVsbG8=
 toJSON              | {"name":"test","value":42}
 trimPrefix          | value
 type                | int
 ...
```

### Teardown

```sh
edg deseed \
--driver pgx \
--config _examples/expression/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"

edg down \
--driver pgx \
--config _examples/expression/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```
