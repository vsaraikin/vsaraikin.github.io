---
title: "Float64 Math is Broken. Comparing Go Decimal Libraries"
date: 2025-10-08
draft: true
description: "The classic 0.1 + 0.2 problem, three decimal libraries, and benchmarks on why correctness costs 1000x performance."
---

## The 0.1 + 0.2 Problem

Try this in any language:

```go
fmt.Println(0.1 + 0.2) // 0.30000000000000004
```

Not a bug. IEEE 754 working as designed.

Computers store numbers in binary (powers of 2). Most decimal fractions can't be represented exactly. The closest float64 can get to 0.1 is:

```
0.1000000000000000055511151231257827021181583404541015625
```

For calculating distance in Google Maps to the nearest cafe, who cares. For money, this kills you.

## Decimal Libraries

Decimal libraries solve the problem by storing numbers in base 10. No rounding errors.

Three options in Go. All slower than float64.

| Library                                                         | Performance | Precision | Memory                      | Notes                    |
| --------------------------------------------------------------- | ----------- | --------- | --------------------------- | ------------------------ |
| **[shopspring/decimal](https://github.com/shopspring/decimal)** | Slowest     | Unlimited | Allocates heavily           | Most popular, safest API |
| **[cockroachdb/apd](https://github.com/cockroachdb/apd)**       | Middle      | Unlimited | Mutable, reuses memory      | Used in CockroachDB SQL  |
| **[govalues/decimal](https://github.com/govalues/decimal)**     | Fastest     | 19 digits | Stack-allocated, zero alloc | Blows up on overflow     |

## Benchmarks

**Test environment:** Apple M1, Go 1.25.1

### Addition

```
Library      ns/op    B/op    allocs   vs float64
float64      0.32     0       0        1.0x
govalues     5.70     0       0        17.8x
cockroach    26.53    0       0        83.3x
shopspring   99.11    176     6        311x
```

### Multiplication

```
Library      ns/op    B/op    allocs   vs float64
float64      0.31     0       0        1.0x
govalues     5.67     0       0        18.3x
cockroach    21.60    0       0        69.8x
shopspring   38.20    80      2        123x
```

### Division

```
Library      ns/op    B/op    allocs   vs float64
float64      0.32     0       0        1.0x
cockroach    18.54    16      1        58.4x
shopspring   280.7    368     12       884x
govalues     289.4    0       0        911x
```

Division kills govalues because it needs more precision than 19 digits can provide.

### Complex ((price × qty) + tax - discount)

```
Library      ns/op    B/op    allocs   vs float64
float64      0.32     0       0        1.0x
govalues     27.55    0       0        86.5x
cockroach    120.0    0       0        377x
shopspring   387.9    616     19       1,218x
```

## Why So Slow?

**shopspring/decimal** is immutable. Every operation allocates a new object:

```go
a := decimal.NewFromFloat(1.5)
b := decimal.NewFromFloat(2.3)
c := a.Add(b) // New allocation
d := c.Mul(a) // Another allocation
```

Safe API. Garbage for the GC.

**cockroachdb/apd** is mutable (like `math/big`):

```go
result := apd.New(0, 0)
ctx.Add(result, a, b) // Reuses result
ctx.Mul(result, result, c) // Still reusing
```

Faster. More error-prone. You can accidentally mutate inputs.

**govalues/decimal** fits in 16 bytes. Stack-allocated:

```go
a, _ := decimal.Parse("1.5")
b, _ := decimal.Parse("2.3")
c, _ := a.Add(b) // No heap allocation
```

Fast. Dangerous if you overflow 19 digits. No warning, just wrong results.

## Run the benchmarks

```bash
cd benchmarks/decimal-benchmark
./run.sh
```

So, you're paying 18-1218x performance penalty for correctness using decimal libraries.

>Results will vary by CPU architecture.
