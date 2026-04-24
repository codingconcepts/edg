---
title: expr-lang Syntax
weight: 3
---

# expr-lang Syntax

Expressions support the full expr-lang feature set, including:

| Category | Examples |
|---|---|
| Arithmetic | `+`, `-`, `*`, `/`, `%`, `**` |
| Array functions | `filter()`, `map()`, `reduce()`, `sort()`, `sortBy()`, `reverse()`, `first()`, `last()`, `take()`, `flatten()`, `uniq()`, `concat()`, `join()`, `find()`, `findIndex()`, `findLast()`, `findLastIndex()`, `all()`, `any()`, `one()`, `none()`, `groupBy()` |
| Bitwise | `bitand()`, `bitor()`, `bitxor()`, `bitnand()`, `bitnot()`, `bitshl()`, `bitshr()`, `bitushr()` |
| Comparison & logic | `==`, `!=`, `<`, `>`, `and`, `or`, `not` |
| Conditionals | `args[0] > 10 ? 'yes' : 'no'`, `args[0] ?? 0` (nil coalescing), `if`/`else` |
| Language | `let` bindings, `#` predicates (current element in closures), `len()`, `get()` |
| Map functions | `keys()`, `values()` |
| Math functions | `abs()`, `ceil()`, `floor()`, `round()`, `mean()`, `median()` |
| Operators | `\|` (pipe), `in` (membership), `..` (range), `[:]` (slice), `?.` (optional chaining) |
| String functions | `upper()`, `lower()`, `trim()`, `trimPrefix()`, `trimSuffix()`, `split()`, `splitAfter()`, `replace()`, `repeat()`, `indexOf()`, `lastIndexOf()`, `hasPrefix()`, `hasSuffix()` |
| String operators | `contains`, `startsWith`, `endsWith`, `matches` (regex) |
| Type conversion | `int()`, `float()`, `string()`, `type()`, `toJSON()`, `fromJSON()`, `toBase64()`, `fromBase64()`, `toPairs()`, `fromPairs()` |

These examples show expr-lang expressions used with edg's reference data. Each expression can appear anywhere an edg expression is accepted (e.g. in `args:`, `expressions:`, or inline globals). Visit the [Expressions example](https://github.com/codingconcepts/edg/tree/main/_examples/expression/) for a complete runnable demonstration.

The examples below assume the following in-memory reference data:

```yaml
reference:
  products:
    - {name: Widget, category: electronics, price: 29.99, stock: 150, active: true}
    - {name: Gadget, category: electronics, price: 49.99, stock: 80, active: true}
    - {name: Notebook, category: stationery, price: 4.99, stock: 500, active: true}
    - {name: Pen, category: stationery, price: 1.99, stock: 1000, active: true}
    - {name: Cable, category: electronics, price: 9.99, stock: 0, active: false}
  regions:
    - {code: us-east, zone: us, cities: [new_york, boston, miami]}
    - {code: eu-west, zone: eu, cities: [london, paris, dublin]}
    - {code: ap-south, zone: ap, cities: [mumbai, singapore, tokyo]}
```

| Category | Expression | Example output |
|---|---|---|
| Arithmetic | `10 % 3`<br><br>`ref_rand('products').stock % 7` | `1` -> `3` |
| Arithmetic | `2 ** 8`<br><br>`len(ref_same('regions').cities) ** 3` | `256` -> `27` |
| Arithmetic | `3 + 4 * 2`<br><br>`ref_rand('products').price + ref_rand('products').stock * 2` | `11` -> `329.99` |
| Array functions | `all(ref_same('regions').cities, {len(#) > 3})` | `true` |
| Array functions | `any(ref_same('regions').cities, {# == 'miami'})` | `true` |
| Array functions | `concat(ref_same('regions').cities, ['london', 'paris'])` | `[new_york, boston, miami, london, paris]` |
| Array functions | `filter(ref_same('regions').cities, {# startsWith 'b'})` | `[boston]` |
| Array functions | `find(ref_same('regions').cities, {# startsWith 'b'})` | `boston` |
| Array functions | `findIndex(ref_same('regions').cities, {# startsWith 'b'})` | `1` |
| Array functions | `findLast(ref_same('regions').cities, {# endsWith 'i'})` | `miami` |
| Array functions | `findLastIndex(ref_same('regions').cities, {# endsWith 'i'})` | `2` |
| Array functions | `first(ref_same('regions').cities)` | `new_york` |
| Array functions | `flatten([['new_york', 'boston'], ['london', 'paris']])`<br><br>`flatten([ref_same('regions').cities, ['sydney', 'tokyo']])` | `[new_york, boston, london, paris]` -> `[new_york, boston, miami, sydney, tokyo]` |
| Array functions | `groupBy(ref_same('regions').cities, {len(#) > 5})` | `{false: [miami, boston], true: [new_york]}` |
| Array functions | `join(ref_same('regions').cities, ', ')` | `new_york, boston, miami` |
| Array functions | `last(ref_same('regions').cities)` | `miami` |
| Array functions | `map(ref_same('regions').cities, {upper(#)})` | `[NEW_YORK, BOSTON, MIAMI]` |
| Array functions | `none(ref_same('regions').cities, {# == 'tokyo'})` | `true` |
| Array functions | `one(ref_same('regions').cities, {# == 'miami'})` | `true` |
| Array functions | `reduce([29.99, 49.99, 4.99, 1.99, 9.99], {#acc + #}, 0)`<br><br>`reduce(ref_same('regions').cities, {#acc + len(#)}, 0)` | `96.95` -> `19` |
| Array functions | `reverse(ref_same('regions').cities)` | `[miami, boston, new_york]` |
| Array functions | `sort(ref_same('regions').cities)` | `[boston, miami, new_york]` |
| Array functions | `sortBy(['Pen', 'Widget', 'Cable'], {len(#)})`<br><br>`sortBy(ref_same('regions').cities, {len(#)})` | `[Pen, Cable, Widget]` -> `[miami, boston, new_york]` |
| Array functions | `take(ref_same('regions').cities, 2)` | `[new_york, boston]` |
| Array functions | `uniq(['electronics', 'stationery', 'electronics'])`<br><br>`uniq(concat(ref_same('regions').cities, ref_same('regions').cities))` | `[electronics, stationery]` -> `[new_york, boston, miami]` |
| Bitwise | `bitand(0b1100, 0b1010)`<br><br>`bitand(ref_rand('products').stock, 0xFF)` | `8` -> `150` |
| Bitwise | `bitnot(0b1100)`<br><br>`bitnot(ref_rand('products').stock)` | `-13` -> `-151` |
| Bitwise | `bitor(0b1100, 0b1010)`<br><br>`bitor(ref_rand('products').stock, 1)` | `14` -> `151` |
| Bitwise | `bitshl(1, 4)`<br><br>`bitshl(1, len(ref_same('regions').cities))` | `16` -> `8` |
| Bitwise | `bitshr(16, 4)`<br><br>`bitshr(ref_rand('products').stock, 1)` | `1` -> `75` |
| Bitwise | `bitxor(0b1100, 0b1010)`<br><br>`bitxor(ref_rand('products').stock, 0xFF)` | `6` -> `105` |
| Comparison & logic | `not (ref_rand('products').category == 'stationery')` | `true` |
| Comparison & logic | `ref_rand('products').stock > 0 and ref_rand('products').active` | `true` |
| Conditionals | `nil ?? 'unknown'`<br><br>`ref_rand('products')?.description ?? 'none'` | `unknown` -> `none` |
| Conditionals | `ref_rand('products').stock > 0 ? 'in_stock' : 'sold_out'` | `in_stock` |
| Language | `all(ref_same('regions').cities, {len(#) > 0})` | `true` |
| Language | `get(ref_rand('products'), 'name')` | `Widget` |
| Language | `len(ref_same('regions').cities)` | `3` |
| Language | `let p = ref_rand('products'); p.price + 10` | `39.99` |
| Map functions | `keys(ref_rand('products'))` | `[name, category, price, stock, active]` |
| Map functions | `values(ref_rand('products'))` | `[Widget, electronics, 29.99, 150, true]` |
| Math functions | `abs(-7)`<br><br>`abs(ref_rand('products').stock - 200)` | `7` -> `50` |
| Math functions | `ceil(3.2)`<br><br>`ceil(ref_rand('products').price)` | `4` -> `30` |
| Math functions | `floor(ref_rand('products').price)` | `29` |
| Math functions | `mean([29.99, 49.99, 4.99, 1.99, 9.99])`<br><br>`mean(map(ref_same('regions').cities, {len(#)}))` | `19.39` -> `6.33` |
| Math functions | `median([1.99, 4.99, 9.99, 29.99, 49.99])`<br><br>`median(map(ref_same('regions').cities, {len(#)}))` | `9.99` -> `6` |
| Math functions | `round(ref_rand('products').price)` | `30` |
| Operators | `1..5`<br><br>`1..len(ref_same('regions').cities) + 1` | `[1, 2, 3, 4]` -> `[1, 2, 3]` |
| Operators | `ref_rand('products')?.name` | `Widget` |
| Operators | `ref_rand('products').price \| int` | `29` |
| Operators | `ref_same('regions').cities[0:2]` | `[new_york, boston]` |
| Operators | `ref_same('regions').zone in ['us', 'eu', 'ap']` | `true` |
| String functions | `hasPrefix(ref_same('regions').code, 'us')` | `true` |
| String functions | `hasSuffix(ref_same('regions').code, 'east')` | `true` |
| String functions | `indexOf('london', 'on')`<br><br>`indexOf(ref_rand('products').category, 'c')` | `1` -> `3` |
| String functions | `lastIndexOf('london', 'on')`<br><br>`lastIndexOf(ref_rand('products').category, 'c')` | `4` -> `9` |
| String functions | `lower(ref_rand('products').category)` | `electronics` |
| String functions | `repeat('*', 5)`<br><br>`repeat(ref_same('regions').zone, 3)` | `*****` -> `ususus` |
| String functions | `replace(ref_same('regions').code, '-', '_')` | `us_east` |
| String functions | `split('new_york,boston,miami', ',')`<br><br>`split(ref_same('regions').code, '-')` | `[new_york, boston, miami]` -> `[us, east]` |
| String functions | `splitAfter('us,eu,ap', ',')`<br><br>`splitAfter(ref_same('regions').code, '-')` | `[us,, eu,, ap]` -> `[us-, east]` |
| String functions | `trim('  gadget  ')`<br><br>`trim(ref_rand('products').name)` | `gadget` -> `Widget` |
| String functions | `trimPrefix(ref_same('regions').code, ref_same('regions').zone + '-')` | `east` |
| String functions | `trimSuffix(ref_same('regions').code, '-' + trimPrefix(ref_same('regions').code, ref_same('regions').zone + '-'))` | `us` |
| String functions | `upper(ref_rand('products').name)` | `WIDGET` |
| String operators | `ref_rand('products').category contains 'electron'` | `true` |
| String operators | `ref_same('regions').code endsWith 'east'` | `true` |
| String operators | `ref_same('regions').code matches '[a-z]+-[a-z]+'` | `true` |
| String operators | `ref_same('regions').code startsWith 'us'` | `true` |
| Type conversion | `float(ref_rand('products').stock)` | `150.0` |
| Type conversion | `fromBase64('V2lkZ2V0')`<br><br>`fromBase64(toBase64(ref_rand('products').name))` | `Widget` -> `Widget` |
| Type conversion | `fromJSON('{"code":"us-east","zone":"us"}')`<br><br>`fromJSON('{"id":' + string(ref_rand('products').stock) + '}')` | `{code: us-east, zone: us}` -> `{id: 150}` |
| Type conversion | `fromPairs([['name', 'Widget'], ['price', 29.99]])`<br><br>`fromPairs([['product', ref_rand('products').name], ['zone', ref_same('regions').zone]])` | `{name: Widget, price: 29.99}` -> `{product: Widget, zone: us}` |
| Type conversion | `int('42')`<br><br>`int(ref_rand('products').price)` | `42` -> `29` |
| Type conversion | `string(ref_rand('products').price)` | `29.99` |
| Type conversion | `toBase64(ref_rand('products').name)` | `V2lkZ2V0` |
| Type conversion | `toJSON(ref_rand('products'))` | `{"active":true,"category":"electronics","name":"Widget","price":29.99,"stock":150}` |
| Type conversion | `toPairs({name: 'Widget', price: 29.99})`<br><br>`toPairs(ref_rand('products'))` | `[[name, Widget], [price, 29.99]]` -> `[[active, true], [category, electronics], [name, Widget], [price, 29.99], [stock, 150]]` |
| Type conversion | `type(ref_rand('products').price)` | `float` |

### Advanced expressions

These combine multiple functions and reference lookups for more complex use cases.

| Description | Expression | Example output |
|---|---|---|
| Chained array ops | `join(take(sort(map(ref_same('regions').cities, {upper(#)})), 2), ', ')` | `BOSTON, MIAMI` |
| Conditional formatting | `let p = ref_rand('products'); p.stock > 100 ? upper(p.name) : lower(p.name)` | `WIDGET` |
| Conditional string ops | `let r = ref_same('regions'); hasPrefix(r.code, 'us') ? upper(first(r.cities)) : lower(last(r.cities))` | `NEW_YORK` |
| Derived metric | `let p = ref_rand('products'); int(ceil(p.price * float(p.stock) / 100))` | `45` |
| Derived slug | `let p = ref_rand('products'); replace(lower(p.name + '_' + p.category), ' ', '_')` | `widget_electronics` |
| Discount pricing | `let p = ref_rand('products'); p.active ? round(p.price * 0.9) : 0` | `27` |
| Filtered count | `len(filter(ref_same('regions').cities, {len(#) > 5}))` | `2` |
| JSON from refs | `toJSON(fromPairs([['product', ref_rand('products').name], ['zone', ref_same('regions').zone]]))` | `{"product":"Widget","zone":"us"}` |
| Multi-condition classification | `let p = ref_rand('products'); p.price > 10 and p.stock > 0 ? 'premium' : (p.active ? 'basic' : 'discontinued')` | `premium` |
| Reduce mapped values | `reduce(map(ref_same('regions').cities, {len(#)}), {#acc + #}, 0)` | `19` |
| Switch on env var | `{'fra': 'eu-central-1', 'sin': 'ap-southeast-1', 'iad': 'us-east-1'}[env('FLY_REGION')] ?? fail('unknown FLY_REGION')` | `eu-central-1` |
