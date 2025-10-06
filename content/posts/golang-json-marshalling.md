---
title: "Go JSON Performance Showdown: Benchmarking the Fastest Libraries"
date: 2025-10-05
draft: false
description: "A comprehensive benchmark comparison of 8 high-performance JSON libraries for Go—from standard library to SIMD-accelerated parsers."
---

JSON serialization is everywhere in Go: REST APIs, config files, data pipelines, message queues. For most applications, `encoding/json` from the standard library works fine. But when you're processing millions of requests per second or dealing with large payloads, JSON becomes a bottleneck.

This post benchmarks **8 of the fastest JSON libraries** for Go, explains *why* they're fast, and helps you choose the right one for your use case.

## The Contenders

| Library           | Type             | Key Feature                                   |
| ----------------- | ---------------- | --------------------------------------------- |
| **encoding/json** | Standard Library | Baseline, most compatible                     |
| **sonic**         | Reflection + JIT | JIT compilation + SIMD instructions           |
| **go-json**       | Reflection       | Optimized reflection with minimal allocations |
| **jsoniter**      | Reflection       | Drop-in replacement, configurable modes       |
| **segmentio**     | Reflection       | Clean API, high performance                   |
| **easyjson**      | Code Generation  | Pre-generated marshaling code                 |
| **fastjson**      | Parse-only       | Zero-allocation parser, no structs            |
| **simdjson-go**   | Parse-only       | SIMD-accelerated parsing (port of simdjson)   |

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
SIMD:       Process 16-32 bytes simultaneously

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

I ran benchmarks on three struct types:

1. **Simple** — 3 fields (int, string, int)
2. **Nested** — 8 fields including slices, maps, timestamps
3. **Complex** — Deeply nested with relationships

Apple M1, macOS, Go 1.25
Iterations: 10,000 per test

### Marshal Performance (MB/s)

```
Simple Struct (100 objects):
Library         MB/s    Relative
sonic           892     4.2x
go-json         624     2.9x
segmentio       601     2.8x
jsoniter        580     2.7x
easyjson        712     3.4x  (requires codegen)
encoding/json   213     1.0x (baseline)

Nested Struct (100 objects):
Library         MB/s    Relative
sonic           654     3.8x
easyjson        587     3.4x
go-json         421     2.4x
segmentio       408     2.4x
jsoniter        395     2.3x
encoding/json   172     1.0x

Complex Struct (50 objects):
Library         MB/s    Relative
sonic           521     3.6x
easyjson        489     3.4x
go-json         338     2.3x
jsoniter        312     2.2x
segmentio       305     2.1x
encoding/json   145     1.0x
```

### Unmarshal Performance (MB/s)

```
Simple Struct:
Library         MB/s    Relative
sonic           1124    5.1x
easyjson        987     4.5x
go-json         682     3.1x
jsoniter        651     3.0x
segmentio       624     2.8x
encoding/json   220     1.0x

Nested Struct:
Library         MB/s    Relative
sonic           876     4.8x
easyjson        824     4.5x
go-json         534     2.9x
jsoniter        502     2.7x
segmentio       485     2.6x
encoding/json   183     1.0x

Complex Struct:
Library         MB/s    Relative
sonic           712     4.6x
easyjson        678     4.4x
go-json         421     2.7x
jsoniter        398     2.6x
segmentio       382     2.5x
encoding/json   155     1.0x
```

### Parse-Only Performance

```
Library         MB/s    Relative
simdjson-go     3421    15.6x
fastjson        2687    12.3x
encoding/json   219     1.0x
```

## Visualization

Here's how the libraries stack up across different operations:

Marshal Performance (Relative to stdlib):

```
sonic       ████████████████ 4.2x
easyjson    ██████████████ 3.4x
go-json     ███████████ 2.9x
segmentio   ███████████ 2.8x
jsoniter    ███████████ 2.7x
stdlib      ████ 1.0x
```

Unmarshal Performance (Relative to stdlib):

```
sonic       █████████████████████ 5.1x
easyjson    ██████████████████ 4.5x
go-json     ████████████ 3.1x
jsoniter    ████████████ 3.0x
segmentio   ███████████ 2.8x
stdlib      ████ 1.0x
```

## Running the Benchmark Yourself

The full benchmark code is available in the [repository](github.com/vsaraikin/vsaraikin.github.io):

```bash
cd benchmarks/json-benchmark
go mod download
go run main.go
```

For easyjson support:

```bash
go install github.com/mailru/easyjson/...@latest
easyjson -all main.go
```

>Performance varies by workload. Always benchmark with your actual data structures.

Me personally, I use `easyjson` because I like the balance of speed and type safety.
