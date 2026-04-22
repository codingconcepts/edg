---
title: REPL
weight: 2
---

# REPL

The `repl` command starts an interactive session where you can evaluate any expression from the function table. No database connection or config file is required; it's a quick way to explore functions, test distributions, and prototype argument expressions before adding them to a workload config.

```sh
edg repl
```

```
>> 1 + 2
3

>> array(2, 5, 'email')
{john@example.com,anne@test.net,mike@domain.org}

>> bit(8)
10110011

>> bytes(16)
\x4a7f2b9c01de38f56a8b3c4d5e6f7a8b

>> exp_f(0.5, 0, 100, 2)
12.74

>> inet('192.168.1.0/24')
192.168.1.47

>> lognorm_f(1.0, 0.5, 1, 1000, 2)
3.41

>> norm(50, 10, 0, 100)
53

>> regex('[A-Z]{3}-[0-9]{4}')
QVM-8314

>> regex('#[0-9a-f]{6}')
#a3c2f1

>> regex('[A-Z]{2}[0-9]{2} [A-Z]{3}')
KD42 BXR

>> set_norm([1, 2, 3, 4, 5], 2, 0.8)
3

>> template('ORD-%05d', seq(1, 1))
ORD-00001

>> time('08:00:00', '18:00:00')
14:23:07

>> timez('09:00:00', '17:00:00')
11:45:32+00:00

>> uniform(0, 100)
73.37

>> uuid_v4()
a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d

>> varbit(16)
101011

>> zipf(2.0, 1.0, 999)
0
```

## Loading Config

To load globals and user-defined expressions from a workload config, pass `--config`:

```sh
edg repl --config _examples/tpcc/crdb.yaml
```

```
>> warehouses
1

>> warehouses * 10
10
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
