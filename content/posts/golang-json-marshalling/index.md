---
title: "Go JSON Performance Showdown: Benchmarking the Fastest Libraries"
date: 2025-10-08
draft: false
description: "A comprehensive benchmark comparison of 8 high-performance JSON libraries for Go: from standard library to SIMD-accelerated parsers."
tags: ["go", "performance", "benchmarks"]
---

JSON serialization is everywhere in Go: REST APIs, config files, data pipelines, message queues. For most applications, `encoding/json` from the standard library works fine. But when you're processing millions of requests per second or dealing with large payloads, JSON becomes a bottleneck.

This post benchmarks **8 of the fastest JSON libraries** for Go, explains _why_ they're fast, and helps you choose the right one for your use case.

## The Contenders

| Library                                                 | Type             | Key Feature                                   |
| ------------------------------------------------------- | ---------------- | --------------------------------------------- |
| **encoding/json**                                       | Standard Library | Baseline, most compatible                     |
| **[sonic](https://github.com/bytedance/sonic)**         | Reflection + JIT | JIT compilation + SIMD instructions           |
| **[go-json](https://github.com/goccy/go-json)**         | Reflection       | Optimized reflection with minimal allocations |
| **[jsoniter](https://github.com/json-iterator/go)**     | Reflection       | Drop-in replacement, configurable modes       |
| **[segmentio](https://github.com/segmentio/encoding)**  | Reflection       | Clean API, high performance                   |
| **[easyjson](https://github.com/mailru/easyjson)**      | Code Generation  | Pre-generated marshaling code                 |
| **[fastjson](https://github.com/valyala/fastjson)**     | Parse-only       | Zero-allocation parser, no structs            |
| **[simdjson-go](https://github.com/minio/simdjson-go)** | Parse-only       | SIMD-accelerated parsing (port of simdjson)   |

## Why Are They Fast?

### 1. Code Generation (easyjson)

Instead of using reflection at runtime, `easyjson` generates specialized `MarshalJSON` and `UnmarshalJSON` methods for your structs at compile time.

**Standard library approach:**

```go
// encoding/json uses reflection to discover struct fields at runtime
json.Marshal(user) // inspects User type every time
```

**easyjson approach:**

```go
// Pre-generated code, no reflection
func (u *User) MarshalJSON() ([]byte, error) {
    // Hand-optimized code specific to User struct
}
```

### 2. JIT Compilation + SIMD (sonic)

ByteDance's `sonic` uses **just-in-time compilation** to generate native machine code specialized for your Go types. It also leverages **SIMD** (Single Instruction, Multiple Data) instructions to process multiple bytes in parallel.

By default: Process 1 byte at a time
SIMD: Process 16-32 bytes simultaneously

When scanning for escape characters (`"`, `\`, etc.) or validating UTF-8, SIMD does 16 comparisons in one CPU instruction.

### 3. Optimized Reflection (go-json, jsoniter, segmentio)

These libraries still use reflection but optimize the common paths:

- **Type caching**: Cache reflection metadata per type
- **Inline fast paths**: Special cases for primitive types (int, string, bool)
- **Reduced allocations**: Reuse buffers, avoid intermediate copies
- **Optimized string escaping**: Hand-tuned assembly for common operations

**go-json** (goccy):

```go
// Caches encoder/decoder per type
encoder := gojson.NewEncoder(w)
encoder.Encode(user) // fast path for known types
```

**jsoniter**:

```go
// Configurable performance modes
var json = jsoniter.ConfigFastest  // skip some validation
var json = jsoniter.ConfigCompatibleWithStandardLibrary // safer
```

### 4. Parse-Only Libraries (fastjson, simdjson-go)

If you only need to **read** JSON (not marshal Go structs), parse-only libraries are blazing fast.

**fastjson**:

- Zero allocations for parsing
- No reflection, no struct binding
- Access fields via getter API

```go
var p fastjson.Parser
v, _ := p.Parse(`{"name":"Alice","age":30}`)
name := v.GetStringBytes("name")  // []byte("Alice"), no allocation
age := v.GetInt("age")            // 30
```

**simdjson-go**:

- Port of the C++ simdjson library
- Uses SIMD for parallel byte scanning
- Parses at **gigabytes per second**

Best reading performance, but no struct marshaling.

## Benchmark Results

Benchmarks on three struct types:

1. **Simple** — 3 fields (int, string, int)
2. **Nested** — 8 fields including slices, maps, timestamps
3. **Complex** — Deeply nested with relationships

**Test environment:** Apple M1, Go 1.25.1, `-benchtime=2s`

### Marshal Performance

```
Simple Struct (100 objects):
Library         ns/op    B/op    allocs    Relative
easyjson        75       128     1         363x
jsoniter        8,949    16,927  102       3.1x
go-json         19,413   16,928  102       1.4x
sonic           19,256   17,249  103       1.4x
segmentio       25,327   20,128  202       1.1x
stdlib          27,325   16,933  102       1.0x

Nested Struct (100 objects):
Library         ns/op    B/op     allocs   Relative
easyjson        658      960      8        280x
jsoniter        76,442   115,907  802      2.4x
sonic           106,749  119,282  803      1.7x
go-json         116,918  115,898  802      1.6x
segmentio       152,342  127,108  902      1.2x
stdlib          183,988  117,974  803      1.0x

Complex Struct (50 objects):
Library         ns/op    B/op     allocs   Relative
easyjson        2,450    2,204    15       158x
jsoniter        117,440  168,021  753      3.3x
sonic           183,264  172,705  754      2.1x
go-json         222,197  168,003  753      1.7x
segmentio       312,088  177,607  803      1.2x
stdlib          387,108  173,381  753      1.0x
```

### Unmarshal Performance

```
Simple Struct:
Library         ns/op     B/op    allocs   Relative
easyjson        100       80      3        448x
fastjson        6,049     0       0        7.4x
go-json         21,966    9,961   201      2.0x
segmentio       23,734    1,056   100      1.9x
sonic           28,976    10,574  203      1.6x
jsoniter        32,681    12,565  737      1.4x
stdlib          44,933    1,224   103      1.0x

Nested Struct:
Library         ns/op     B/op     allocs   Relative
easyjson        134       208      4        1,678x
fastjson        20,845    3        0        10.8x
go-json         105,703   96,039   1,401    2.1x
segmentio       109,770   55,426   1,300    2.1x
sonic           118,399   97,770   1,403    1.9x
jsoniter        134,685   86,228   2,674    1.7x
stdlib          225,339   55,636   1,304    1.0x

Complex Struct:
Library         ns/op     B/op     allocs   Relative
easyjson        143       208      4        3,681x
fastjson        55,149    28       0        9.6x
go-json         257,463   191,653  2,251    2.0x
sonic           276,056   194,849  2,253    1.9x
segmentio       277,387   76,494   2,200    1.9x
jsoniter        355,029   164,246  5,818    1.5x
stdlib          527,415   76,778   2,205    1.0x
```

## Running the Benchmark Yourself

```bash
cd benchmarks/json-benchmark
./run.sh
```

For easyjson support:

```bash
go install github.com/mailru/easyjson/...@latest
easyjson -all main.go
```

> Performance varies by workload. Always benchmark with your actual data structures.

Me personally, I use `easyjson` because I like the balance of speed and type safety.
