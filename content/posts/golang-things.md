---
title: "False Sharing in Go: When Your Goroutines Fight Over Cache Lines"
date: 2025-10-04T20:08:40+02:00
draft: true
description: "About CPU cache architecture, coherence protocols, false sharing."
---

## The Problem You Didn't Know You Had

You've written beautifully concurrent Go code. Your goroutines work on completely separate data. No shared mutexes, no data races. The code is clean, the tests pass, and your profiler shows high CPU utilization across all cores.

Yet somehow, when you scale from 2 to 8 cores, performance barely improves. In some cases, it even gets *worse*.

What's happening? Your goroutines are fighting over something invisible: cache lines.

This is false sharing, and it's one of the most frustrating performance problems in concurrent programming because your code is logically correct—the compiler won't warn you, the race detector stays silent, and yet you're leaving massive performance on the table.

This post will take you through the hardware fundamentals (what cache lines are, how coherence protocols work), show you exactly how false sharing manifests in Go, and give you concrete techniques to eliminate it. Along the way, we'll build benchmarks to measure the impact—because seeing is believing.

## Understanding Cache Lines: The Atom of Memory Transfer

When your CPU needs data from RAM, it doesn't fetch a single byte or even a single variable. It grabs a whole chunk—typically 64 bytes—called a **cache line**. Think of it like ordering books from a warehouse: even if you only need one book, they ship you an entire box of books from the same shelf.

Why 64 bytes? It's a sweet spot that hardware engineers discovered: small enough to fit many lines in the limited cache space, large enough to exploit spatial locality (if your program reads byte `X`, it'll probably soon read byte `X+1`).

This matters because:

```go
struct {
    a int64  // 8 bytes at offset 0
    b int64  // 8 bytes at offset 8
    c int64  // 8 bytes at offset 16
    ...
}
```

If your goroutine reads field `a`, the CPU automatically loads `b`, `c`, and the next 40 bytes into the cache line, anticipating you might need them soon. Usually that's brilliant. Sometimes it's a disaster.

### The Cache Hierarchy: A Multi-Level Memory System

Modern CPUs don't have just one cache—they have a hierarchy optimized for different access patterns:

#### L1 Cache (32–64 KB per core)

- Closest to the CPU core
- Access time: ~4 cycles (~1 nanosecond)
- Split into L1i (instruction) and L1d (data)
- Private to each core

#### L2 Cache (256 KB–1 MB per core)

- Slightly larger, slightly slower
- Access time: ~12 cycles (~3–4 nanoseconds)
- Still private to each core
- Filters requests before hitting the shared L3

#### L3 Cache (8–64 MB shared)

- Shared across all cores on the chip
- Access time: ~40–50 cycles (~12–15 nanoseconds)
- Acts as a victim cache and reduces main memory traffic
- This is where cores can "see" each other's data

#### Main Memory (DRAM, several GB)

- Access time: ~200 cycles (~60–80 nanoseconds)
- Multiple orders of magnitude slower than L1

The key insight: data flows through this hierarchy in **cache line** sized chunks (64 bytes). When core 0 modifies a variable, the entire containing cache line must be synchronized across all caches that hold it.

## Cache Coherence: Keeping Everyone's Story Straight

Here's the challenge: you have 8 CPU cores, each with its own private L1 and L2 cache. All of them can read and modify the same memory addresses. How do you ensure they don't see stale data?

If core 0 modifies `x`, and core 1 has an old copy of `x` in its cache, core 1's reads will return the wrong value. This is called the **cache coherence problem**, and modern CPUs solve it with sophisticated protocols.

### The MESI Protocol: A Finite State Machine for Cache Lines

The most common solution is the **MESI protocol** (named after its four states: Modified, Exclusive, Shared, Invalid). Every cache line in every core's cache is tagged with one of these states:

**Modified (M)** — "I own this, and I've changed it"
- This core holds the *only* valid copy
- The copy is *dirty* (differs from main memory)
- Writes are free—no need to notify other cores since they don't have it
- Eventually this line will be written back to memory

**Exclusive (E)** — "I own this, but haven't changed it yet"
- This core holds the only copy, and it's *clean* (matches main memory)
- No other core has cached this line
- The first write silently transitions to Modified (no bus traffic needed)

**Shared (S)** — "Others might have this too"
- Multiple cores hold a copy, all clean
- Reads are free
- Writes require broadcasting an invalidation to all other cores

**Invalid (I)** — "My copy is stale"
- This core's cached copy is no longer valid
- Any access requires fetching the line fresh

### State Transitions: The Dance of Invalidations

Let's walk through what happens when two cores access the same cache line:

**Scenario: Core 0 and Core 1 both want variable `x` (initially in state I on both)**

1. **Core 0 reads `x`**
   - Cache miss → fetch from memory
   - State: `I` → `E` (exclusive, since no one else has it)

2. **Core 1 reads `x`**
   - Core 0 snoops the bus, sees the request
   - Core 0 transitions: `E` → `S`
   - Core 1 loads: `I` → `S`
   - Both cores now hold clean shared copies

3. **Core 0 writes to `x`**
   - Core 0 broadcasts an "invalidate" message
   - Core 1 transitions: `S` → `I`
   - Core 0 transitions: `S` → `M`
   - Now only core 0 has a valid (dirty) copy

4. **Core 1 reads `x` again**
   - Core 1 issues a read request
   - Core 0 snoops it, sees it owns the line in M state
   - Core 0 must write back to memory *or* provide the data directly
   - Core 0: `M` → `S`, Core 1: `I` → `S`

Every write to a Shared line forces invalidations across the interconnect. This is expensive—it involves:
- Broadcasting on the cache coherence bus
- Waiting for acknowledgments from all other cores
- Potentially stalling the pipeline until the invalidations complete

### MOESI and Beyond

Some processors (like AMD) extend MESI with an **Owned (O)** state:

**Owned (O)** — "I have the dirty copy, but others have clean copies"
- This core is responsible for writing back to memory eventually
- Other cores can have Shared copies
- Reduces write-back traffic when multiple cores read a modified line

Intel uses **MESIF**, adding a **Forward (F)** state to designate which core should respond to requests for a Shared line, reducing contention on the L3 cache.

The details vary by microarchitecture, but the fundamental principle is the same: **cache coherence operates at cache line granularity**. When you write one byte, the entire 64-byte line participates in the coherence protocol.

## False Sharing: The Invisible Performance Killer

Now we can finally understand the problem. Imagine you have this perfectly reasonable Go code:

```go
type Stats struct {
    requestsA uint64  // updated by goroutine A
    requestsB uint64  // updated by goroutine B
}

var stats Stats

// goroutine A (running on core 0)
for {
    atomic.AddUint64(&stats.requestsA, 1)
}

// goroutine B (running on core 1)
for {
    atomic.AddUint64(&stats.requestsB, 1)
}
```

These goroutines access *completely separate variables*. No data race. The Go race detector is silent. The code is logically correct.

But here's what's actually happening at the hardware level:

**Memory Layout:**

```text
Address 0x1000:  [requestsA: 8 bytes][requestsB: 8 bytes][padding: 48 bytes]
                 <---------------- 64-byte cache line ---------------->
```

Both `requestsA` and `requestsB` live in the same cache line.

**The Ping-Pong Effect:**

1. **Goroutine A (core 0) increments `requestsA`**
   - Core 0 needs exclusive access to the cache line
   - Broadcasts invalidation → cache line state on core 1: `S` → `I`
   - Core 0: `S` → `M`

2. **Goroutine B (core 1) increments `requestsB`**
   - Core 1 needs the cache line but it's Invalid
   - Fetches from core 0, invalidating core 0's copy
   - Core 0: `M` → `I`, Core 1: `I` → `M`

3. **Goroutine A increments again**
   - Cache line is Invalid on core 0 now
   - Fetch from core 1, invalidate core 1
   - The line bounces back...

4. **Repeat millions of times per second**

The variables are logically independent, but the cache line is the unit of coherence. Every write to `requestsA` invalidates core 1's cache, even though core 1 never reads `requestsA`. Every write to `requestsB` invalidates core 0's cache, even though core 0 never reads `requestsB`.

The result: **cache line ping-pong**. Your cores spend more time fighting over cache line ownership than doing actual work.

### The Performance Cost

Each invalidation + refill involves:

- Broadcasting on the coherence interconnect (~40 cycles)
- Waiting for acknowledgments from other cores
- Fetching the line from the remote cache or L3 (~50+ cycles)
- Stalling the pipeline while waiting

On a modern 3 GHz CPU, that's 30+ nanoseconds per bounce. If your goroutines update these counters millions of times per second, you're burning orders of magnitude more cycles on cache coherence traffic than on the actual arithmetic.

### Where False Sharing Hides in Go

#### Adjacent struct fields updated by different goroutines

```go
type Metrics struct {
    count1 uint64  // goroutine 1
    count2 uint64  // goroutine 2
    count3 uint64  // goroutine 3
    // ... all in the same cache line
}
```

#### Array of structs with hot fields

```go
type Worker struct {
    id       int
    counter  uint64  // frequently updated
}

workers := make([]Worker, numCPUs)
// workers[0].counter and workers[1].counter likely share a cache line
```

#### Per-shard counters placed naively

```go
type Counter struct {
    shards [8]uint64  // all 64 bytes = one cache line!
}
```

Each shard is meant to be independent, but they all fight over the same cache line.

## Seeing is Believing: A Minimal Reproducer

Let's build the simplest possible case to demonstrate false sharing. We'll create two struct types:

1. `pair` — fields are adjacent (false sharing expected)
2. `paddedPair` — fields are separated by 56 bytes of padding (no false sharing)

```go
package falsesharing

const cacheLineSize = 64

// Unpadded: a and b will share a cache line
type pair struct {
    a uint64  // 8 bytes at offset 0
    b uint64  // 8 bytes at offset 8
    // Total size: 16 bytes → fits in one 64-byte cache line
}

// Padded: a and b live on separate cache lines
type paddedPair struct {
    a uint64                  // 8 bytes
    _ [cacheLineSize - 8]byte // 56 bytes of padding
    b uint64                  // 8 bytes (starts at offset 64)
    _ [cacheLineSize - 8]byte // 56 bytes of padding
    // Total size: 128 bytes → a and b are on different cache lines
}
```

The `_` field is an unnamed padding field that Go ignores but the compiler allocates. By padding each `uint64` to 64 bytes, we guarantee `a` and `b` start on separate cache line boundaries.

## Measuring the Impact: Benchmarks

Now let's measure the actual performance difference. We'll create benchmarks where two goroutines increment different fields of the same struct—one with false sharing, one without.

**Important:** These benchmarks use two separate approaches:

1. **Non-atomic increments** — Safe because each goroutine writes to a different field (no data race)
2. **Atomic increments** — Demonstrates that `sync/atomic` ensures correctness but doesn't prevent false sharing

Save this as `false_sharing_bench_test.go` and run:

```bash
go test -run=^$ -bench=. -benchtime=2s -cpu=1,2,4 -count=3
```

```go
package falsesharing

import (
    "runtime"
    "sync"
    "sync/atomic"
    "testing"
)

const cacheLineSize = 64

type pair struct {
    a uint64
    b uint64
}

type paddedPair struct {
    a uint64
    _ [cacheLineSize - 8]byte
    b uint64
    _ [cacheLineSize - 8]byte
}

var sink uint64 // Prevents compiler from optimizing away our work

// BenchmarkPair_FalseSharing_NoAtomic demonstrates false sharing with non-atomic writes.
// Since each goroutine writes to a distinct field, there's no data race.
// But both fields share a cache line, causing performance degradation.
func BenchmarkPair_FalseSharing_NoAtomic(b *testing.B) {
    runtime.GOMAXPROCS(2) // Pin to 2 cores
    var p pair
    var wg sync.WaitGroup
    b.ReportAllocs()
    b.ResetTimer()

    wg.Add(2)
    // Goroutine 1: increment field 'a'
    go func() {
        for i := 0; i < b.N; i++ {
            p.a++ // Write to offset 0 in the cache line
        }
        wg.Done()
    }()
    // Goroutine 2: increment field 'b'
    go func() {
        for i := 0; i < b.N; i++ {
            p.b++ // Write to offset 8 in the same cache line
        }
        wg.Done()
    }()
    wg.Wait()
    sink = p.a + p.b // Ensure values are used
}

// BenchmarkPair_Padded_NoAtomic is the control: same logic, but a and b
// are on separate cache lines. No false sharing.
func BenchmarkPair_Padded_NoAtomic(b *testing.B) {
    runtime.GOMAXPROCS(2)
    var p paddedPair
    var wg sync.WaitGroup
    b.ReportAllocs()
    b.ResetTimer()

    wg.Add(2)
    go func() {
        for i := 0; i < b.N; i++ {
            p.a++ // Write to offset 0 (first cache line)
        }
        wg.Done()
    }()
    go func() {
        for i := 0; i < b.N; i++ {
            p.b++ // Write to offset 64 (second cache line)
        }
        wg.Done()
    }()
    wg.Wait()
    sink = p.a + p.b
}

// BenchmarkPair_FalseSharing_Atomic shows that atomic operations don't fix false sharing.
// Atomics ensure correctness (no torn reads/writes), but the cache line still bounces.
func BenchmarkPair_FalseSharing_Atomic(b *testing.B) {
    runtime.GOMAXPROCS(2)
    var p pair
    var wg sync.WaitGroup
    b.ReportAllocs()
    b.ResetTimer()

    wg.Add(2)
    go func() {
        for i := 0; i < b.N; i++ {
            atomic.AddUint64(&p.a, 1) // Atomic, but still shares cache line
        }
        wg.Done()
    }()
    go func() {
        for i := 0; i < b.N; i++ {
            atomic.AddUint64(&p.b, 1)
        }
        wg.Done()
    }()
    wg.Wait()
    sink = atomic.LoadUint64(&p.a) + atomic.LoadUint64(&p.b)
}

// BenchmarkPair_Padded_Atomic: atomic operations with proper padding.
// This is both correct AND fast.
func BenchmarkPair_Padded_Atomic(b *testing.B) {
    runtime.GOMAXPROCS(2)
    var p paddedPair
    var wg sync.WaitGroup
    b.ReportAllocs()
    b.ResetTimer()

    wg.Add(2)
    go func() {
        for i := 0; i < b.N; i++ {
            atomic.AddUint64(&p.a, 1)
        }
        wg.Done()
    }()
    go func() {
        for i := 0; i < b.N; i++ {
            atomic.AddUint64(&p.b, 1)
        }
        wg.Done()
    }()
    wg.Wait()
    sink = atomic.LoadUint64(&p.a) + atomic.LoadUint64(&p.b)
}
```

### What to Expect

Run the benchmarks and you'll see something like this (actual numbers vary by CPU):

```text
BenchmarkPair_FalseSharing_NoAtomic-2     50000000    42.3 ns/op
BenchmarkPair_Padded_NoAtomic-2          200000000     6.8 ns/op

BenchmarkPair_FalseSharing_Atomic-2       40000000    55.1 ns/op
BenchmarkPair_Padded_Atomic-2            150000000     8.2 ns/op
```

**Key observations:**

- **6–7x slowdown** from false sharing in the non-atomic case
- **Atomics don't fix it**: the atomic version with false sharing is *slower* than the non-atomic version with padding
- **Single core (`-cpu=1`)**: the gap shrinks dramatically because there's no inter-core traffic
- **More cores (`-cpu=4`)**: false sharing gets even worse as more cores contend

## How to Prevent False Sharing in Go

Now that you understand the problem, here are practical techniques to avoid it.

### 1. Explicit Padding

The most direct solution: add padding between frequently-written fields.

**Before (false sharing):**

```go
type Metrics struct {
    reads  uint64
    writes uint64
}
```

**After (no false sharing):**

```go
type Metrics struct {
    reads  uint64
    _      [56]byte  // pad to 64 bytes
    writes uint64
    _      [56]byte  // pad to 64 bytes
}
```

**Why 56 bytes?** Each `uint64` is 8 bytes. To reach the next 64-byte boundary, we need 64 - 8 = 56 bytes of padding.

### 2. Struct-of-Arrays (SoA) vs Array-of-Structs (AoS)

If you have many workers each updating a hot field, consider splitting the layout.

**Array-of-Structs (AoS) — false sharing likely:**

```go
type Worker struct {
    id      int
    counter uint64  // hot field
    name    string
}

workers := make([]Worker, 100)
// workers[0].counter and workers[1].counter are close in memory
```

**Struct-of-Arrays (SoA) — better cache behavior:**

```go
type Workers struct {
    ids      []int
    counters []uint64  // all hot fields in one slice
    names    []string
}

workers := Workers{
    ids:      make([]int, 100),
    counters: make([]uint64, 100),
    names:    make([]string, 100),
}
// Still need padding if counters are updated by different goroutines!
```

But even with SoA, you may need padding if adjacent elements are updated concurrently. Better yet: shard.

### 3. Sharded Counters

Instead of one contended counter, use multiple shards—one per core (or per goroutine).

**Before (single counter, heavy contention):**

```go
type Counter struct {
    value uint64
}

func (c *Counter) Inc() {
    atomic.AddUint64(&c.value, 1)
}
```

**After (sharded, no contention):**

```go
const cacheLineSize = 64

// Each shard is padded to occupy its own cache line
type shard struct {
    v uint64                  // 8 bytes
    _ [cacheLineSize - 8]byte // 56 bytes padding → total 64 bytes
}

type Counter struct {
    shards []shard
}

func NewCounter() *Counter {
    n := runtime.GOMAXPROCS(0) // One shard per logical CPU
    return &Counter{shards: make([]shard, n)}
}

// Add increments the counter for a specific shard (typically based on goroutine ID or hash)
func (c *Counter) Add(delta uint64, shardID int) {
    atomic.AddUint64(&c.shards[shardID%len(c.shards)].v, delta)
}

// Sum returns the total across all shards (call infrequently)
func (c *Counter) Sum() uint64 {
    var total uint64
    for i := range c.shards {
        total += atomic.LoadUint64(&c.shards[i].v)
    }
    return total
}
```

**How it works:**

- Each goroutine increments its own shard (no contention)
- Each shard occupies a full 64-byte cache line (no false sharing)
- Reading the total requires summing all shards (done infrequently)

This pattern is used internally in Go's `sync` package and in high-performance libraries like `uber-go/atomic`.

### 4. Separate Hot and Cold Fields

Keep frequently-written fields away from rarely-changed fields.

**Before:**

```go
type Server struct {
    config     Config   // rarely changes
    requestCount uint64 // updated constantly
    lastRestart  time.Time
}
```

**After:**

```go
type ServerConfig struct {
    config      Config
    lastRestart time.Time
}

type ServerMetrics struct {
    requestCount uint64
    _            [56]byte
}

type Server struct {
    cfg     *ServerConfig  // read-mostly, shared
    metrics *ServerMetrics // write-heavy, possibly per-core
}
```

### 5. Trust but Verify

The Go compiler *may* reorder fields for alignment, but it won't add cache-line padding automatically. If performance is critical:

- **Measure** with `go test -bench`
- **Profile** with `perf` on Linux to see cache miss rates
- **Pad explicitly** if you detect false sharing

### Important Notes

#### Atomics ≠ No False Sharing

`sync/atomic` ensures correctness (no torn reads, sequentially consistent operations), but it does *not* prevent cache line bouncing. You still need proper layout.

#### Cache Line Size

This post assumes 64-byte cache lines (common on x86-64, ARM64). Some systems differ:

- Most modern Intel/AMD: 64 bytes
- Apple M1/M2: **128 bytes** (!)
- Some embedded CPUs: 32 bytes

For portable code:

```go
const cacheLineSize = 128 // Conservative: covers Apple Silicon
```

Or detect at runtime (advanced).

## Debugging False Sharing: A Checklist

When you suspect false sharing, ask yourself:

**1. Are multiple goroutines writing to adjacent memory?**

- Fields in the same struct updated by different goroutines?
- Array elements accessed by index, where `workers[i]` and `workers[i+1]` are hot-written by different goroutines?

**2. Is your performance scaling poorly with cores?**

- Run benchmarks with `-cpu=1,2,4,8`
- If throughput barely improves (or degrades) as cores increase, suspect false sharing

**3. Are you seeing high cache miss rates?**

On Linux, use `perf`:

```bash
perf stat -e cache-references,cache-misses,LLC-loads,LLC-load-misses go test -bench=.
```

Look for high `LLC-load-misses` (Last Level Cache misses) — a smoking gun for cache line bouncing.

**4. Did you verify the fix?**

- Add padding to suspect structs
- Re-run benchmarks with `-cpu>=2`
- If throughput improves significantly, you've found it

## Real-World Examples in the Wild

False sharing isn't just a theoretical curiosity—it shows up in production Go code:

**Go's `sync.Pool`**

The `sync.Pool` implementation uses per-P (processor) local caches with explicit padding to avoid false sharing between goroutines running on different cores.

**High-performance networking libraries**

Libraries like `fasthttp` use sharded counters and padded structs to maximize throughput on multi-core systems.

**Database connection pools**

Per-connection statistics (queries executed, errors) must be carefully laid out to avoid false sharing when many goroutines use the pool concurrently.

## When Not to Worry

Don't prematurely optimize. False sharing only matters when:

- You have high-frequency writes (millions per second)
- Multiple goroutines are writing concurrently
- You're running on multiple cores
- Profiling shows you're bottlenecked on memory/cache performance

For most CRUD apps, HTTP servers with moderate load, or infrequent updates, false sharing is invisible. Optimize only when measurements show it's a problem.

## Further Reading and References

**Papers and Articles:**

- [Ulrich Drepper, "What Every Programmer Should Know About Memory"](https://people.freebsd.org/~lstewart/articles/cpumemory.pdf) — The definitive deep dive into CPU cache architecture
- [Intel 64 and IA-32 Architectures Optimization Reference Manual](https://www.intel.com/content/www/us/en/developer/articles/technical/intel-sdm.html) — Official optimization guide from Intel
- [Martin Thompson, "False Sharing"](https://mechanical-sympathy.blogspot.com/2011/07/false-sharing.html) — Excellent explanation from the Mechanical Sympathy blog

**Go-Specific Resources:**

- [Go `sync` package source](https://github.com/golang/go/tree/master/src/sync) — See how Go's standard library uses cache-line padding internally
- [Brad Fitzpatrick, "Go: What's New in March 2024"](https://go.dev/blog/) — Performance improvements often involve cache-aware data structures

**Hardware Documentation:**

- [AMD Optimization Guide](https://www.amd.com/en/support/tech-docs)
- [ARM Cortex-A Series Programmer's Guide](https://developer.arm.com/documentation/)

**Tools:**

- `perf` (Linux) — Hardware performance counters
- `go test -bench` — Benchmarking
- Intel VTune — Advanced profiling (shows cache misses visually)

---

## Wrapping Up

False sharing is one of those problems that sits at the intersection of hardware and software. Your code can be logically perfect—no races, clean concurrency patterns—and yet performance suffers because of invisible battles at the cache line level.

The good news: once you understand the hardware model (cache lines, coherence protocols, the MESI dance), the solutions are straightforward. Pad your hot fields, shard your counters, separate hot from cold data. Measure before and after. The performance gains can be dramatic.

The better news: you don't need to worry about this for every struct. False sharing is a high-frequency, multi-core problem. If your profiler doesn't show cache contention, don't add padding "just in case." But when you do hit it—when your 8-core server performs like a dual-core—now you know where to look.

Go gives you the tools: `sync/atomic`, explicit padding with `_` fields, and a benchmark framework to measure it all. The hardware gives you the constraints: 64-byte cache lines, MESI invalidations, and the speed of light. Your job is to design data structures that work *with* the hardware, not against it.

Now go forth and stop those cache lines from ping-ponging.


Reference:

- https://people.freebsd.org/~lstewart/articles/cpumemory.pdf