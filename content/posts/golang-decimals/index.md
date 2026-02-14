---
title: "How Do Go Decimal Packages Work Under the Hood?"
date: 2025-11-19
draft: false
description: "Inside shopspring/decimal: what type Decimal hides, why .Add allocates 6 times, and how three libraries trade precision for speed."
tags: ["go", "performance"]
---

## The 0.1 + 0.2 Problem

Try this in any language:

```go
fmt.Println(0.1 + 0.2) // 0.30000000000000004
```

Not a bug. IEEE 754 working as designed.

Computers store numbers in binary (powers of 2). Most decimal fractions can't be represented exactly. The closest float64 can get to 0.1 is:

```text
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

```text
Library      ns/op    B/op    allocs   vs float64
float64      0.32     0       0        1.0x
govalues     5.70     0       0        17.8x
cockroach    26.53    0       0        83.3x
shopspring   99.11    176     6        311x
```

### Multiplication

```text
Library      ns/op    B/op    allocs   vs float64
float64      0.31     0       0        1.0x
govalues     5.67     0       0        18.3x
cockroach    21.60    0       0        69.8x
shopspring   38.20    80      2        123x
```

### Division

```text
Library      ns/op    B/op    allocs   vs float64
float64      0.32     0       0        1.0x
cockroach    18.54    16      1        58.4x
shopspring   280.7    368     12       884x
govalues     289.4    0       0        911x
```

Division kills govalues because it needs more precision than 19 digits can provide.

### Complex ((price × qty) + tax - discount)

```text
Library      ns/op    B/op    allocs   vs float64
float64      0.32     0       0        1.0x
govalues     27.55    0       0        86.5x
cockroach    120.0    0       0        377x
shopspring   387.9    616     19       1,218x
```

## What's Inside `type Decimal struct`

The most popular library, shopspring/decimal, stores every number as two fields:

```go
type Decimal struct {
    value *big.Int
    exp   int32
}
```

The number is `value × 10^exp`. So `12.345` is stored as `value=12345, exp=-3`. The value field is a pointer to `math/big.Int` — a variable-length integer that lives on the heap.

This is the root of both the library's power (unlimited precision) and its performance cost (every number involves a heap allocation).

## How `.Add` Actually Works

```go
func (d Decimal) Add(d2 Decimal) Decimal {
    rd, rd2 := RescalePair(d, d2)
    d3Value := new(big.Int).Add(rd.value, rd2.value)
    return Decimal{
        value: d3Value,
        exp:   rd.exp,
    }
}
```

Three steps:

**1. Rescale to a common exponent.** You can't add `120 × 10⁻²` and `5 × 10⁻³` directly — the exponents differ. `RescalePair` picks the smaller exponent (more precise) and scales the other operand up:

```go
func RescalePair(d1 Decimal, d2 Decimal) (Decimal, Decimal) {
    if d1.exp < d2.exp {
        return d1, d2.rescale(d1.exp)
    } else if d1.exp > d2.exp {
        return d1.rescale(d2.exp), d2
    }
    return d1, d2
}
```

`rescale` computes `10^diff` and multiplies the value. To rescale `1.20` (value=120, exp=-2) to exp=-3: compute `10^1 = 10`, multiply `120 × 10 = 1200`. Now both operands have exp=-3.

**2. Add the big.Int values.** `new(big.Int).Add(1200, 5)` = 1205. This allocates a fresh `big.Int` on the heap.

**3. Return a new Decimal.** `Decimal{value: 1205, exp: -3}` = 1.205.

## Why 6 Allocations for a Single Add

Look at the benchmark: `shopspring 99.11 ns/op, 176 B/op, 6 allocs`. Where do 6 allocations come from?

The `rescale` method alone allocates three `big.Int` values — a copy of the original value, the power of 10, and the scaled result. When both operands need rescaling, that's six. Add the final result allocation, and a single `.Add()` call can trigger 4-7 heap allocations depending on whether exponents match.

This is a deliberate design choice. shopspring chose immutability — every method returns a new Decimal, never mutates the receiver. Safe API. Garbage for the GC.

## The Other Two Approaches

**cockroachdb/apd** is mutable (like `math/big`):

```go
result := apd.New(0, 0)
ctx.Add(result, a, b) // Reuses result
ctx.Mul(result, result, c) // Still reusing
```

Same `big.Int` under the hood, but `z.Add(x, y)` writes into `z` instead of allocating. Faster. More error-prone — you can accidentally mutate inputs.

**govalues/decimal** avoids `big.Int` entirely. The entire decimal fits in 16 bytes — two uint64s. Stack-allocated, zero heap allocations. The tradeoff: 19 digits of precision, and division can overflow.

You're paying 18-1218x performance penalty for correctness. Where on that spectrum depends on which tradeoff you pick: unlimited precision with heap pressure, mutable reuse with footguns, or fixed precision with overflow risk.
