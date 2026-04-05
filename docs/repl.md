---
title: REPL
layout: default
nav_order: 8
---

# REPL

The `repl` command starts an interactive session where you can evaluate any expression from the function table. No database connection is required -- it's a quick way to explore functions, test distributions, and prototype argument expressions before adding them to a workload config.

```sh
edg repl
```

```
>> uniform(0, 100)
73.37

>> norm(50, 10, 0, 100)
53

>> set_norm([1, 2, 3, 4, 5], 2, 0.8)
3

>> uuid_v4()
a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d

>> template('ORD-%05d', seq(1, 1))
ORD-00001

>> zipf(2.0, 1.0, 999)
0

>> regex('[A-Z]{3}-[0-9]{4}')
QVM-8314

>> regex('#[0-9a-f]{6}')
#a3c2f1

>> regex('[A-Z]{2}[0-9]{2} [A-Z]{3}')
KD42 BXR

>> exp_f(0.5, 0, 100, 2)
12.74

>> lognorm_f(1.0, 0.5, 1, 1000, 2)
3.41

>> inet('192.168.1.0/24')
192.168.1.47

>> bytes(16)
\x4a7f2b9c01de38f56a8b3c4d5e6f7a8b

>> bit(8)
10110011

>> varbit(16)
101011

>> array(2, 5, 'email')
{john@example.com,anne@test.net,mike@domain.org}

>> time('08:00:00', '18:00:00')
14:23:07

>> timez('09:00:00', '17:00:00')
11:45:32+00:00

>> 1 + 2
3
```

## Loading Config

To load globals and user-defined expressions from a workload config, pass `--config`:

```sh
edg repl --config _examples/tpcc/crdb.yaml
```

```
edg repl - type expressions to evaluate
>> warehouses
1
>> warehouses * 10
10
>> nurand(1023, 1, 3000)
1842
```

If your config includes a `reference` section, those datasets are also available in the REPL when loaded with `--config`:

```sh
edg repl --config _examples/reference_data/crdb.yaml
```

```
>> ref_rand('regions').name
eu
>> ref_same('regions').cities
[d e f]
>> set_rand(ref_same('regions').cities, [])
e
```
