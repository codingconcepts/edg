---
title: Distributions
weight: 3
---

# Distributions

Several functions generate values using statistical distributions, giving you control over the shape of your random data.

## Numeric Distributions

| Function | Signature | Description |
|---|---|---|
| `exp_f` | `exp_f(rate, min, max, precision)` | Exponential decay from min |
| `lognorm_f` | `lognorm_f(mu, sigma, min, max, precision)` | Right-skewed with a long tail |
| `norm_f` | `norm_f(mean, stddev, min, max, precision)` | Bell curve centered on mean |
| `uniform` | `uniform(min, max)` | Flat distribution, every value equally likely |
| `zipf` | `zipf(s, v, max)` | Power-law skew, low values dominate |

## Set Distributions

Pick from a predefined set of values using a distribution to control which items are selected most often.

| Function | Signature | Description |
|---|---|---|
| `set_exp` | `set_exp(values, rate)` | Exponential distribution over indices; lower indices picked most often |
| `set_lognorm` | `set_lognorm(values, mu, sigma)` | Log-normal distribution over indices; right-skewed selection |
| `set_norm` | `set_norm(values, mean, stddev)` | Normal distribution over indices; `mean` index picked most often |
| `set_rand` | `set_rand(values, weights)` | Uniform or weighted random selection from a set |
| `set_zipf` | `set_zipf(values, s, v)` | Zipfian distribution over indices; strong power-law skew toward first items |
