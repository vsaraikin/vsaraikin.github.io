---
title: "CPU Cache: What Every Programmer Should Know"
date: 2025-01-20
draft: true
description: "A deep dive into CPU caches, cache coherence, virtual memory, and NUMA. With benchmarks showing why your code might be 10x slower than it should be."
---

Your CPU runs at 4 GHz. That's 4 billion cycles per second. Each cycle takes 0.25 nanoseconds.

Reading from RAM takes 100 nanoseconds.

Do the math: while waiting for one memory read, your CPU could have executed 400 instructions. It just sits there. Waiting.

This is called the **Memory Wall**. And it's why CPU caches exist.

## The Memory Hierarchy

Modern CPUs don't just have "memory". They have layers:

```
┌─────────────────────────────────────────────────────────────┐
│                         CPU Core                            │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Registers (~0.25 ns)                    │   │
│  │                    64-256 bytes                      │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │               L1 Cache (~1 ns)                       │   │
│  │              32-64 KB per core                       │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │               L2 Cache (~3-4 ns)                     │   │
│  │             256 KB - 1 MB per core                   │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│                  L3 Cache (~10-20 ns)                       │
│                 8-64 MB shared across cores                 │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│                     RAM (~100 ns)                           │
│                      8-128 GB                               │
└─────────────────────────────────────────────────────────────┘
```

Each level is slower but bigger. L1 is tiny but blazing fast. RAM is huge but glacially slow (in CPU terms).

**The numbers that matter:**

| Level | Latency | Size | Analogy |
|-------|---------|------|---------|
| L1 | ~1 ns (4 cycles) | 32-64 KB | Your desk |
| L2 | ~4 ns (12 cycles) | 256 KB - 1 MB | Filing cabinet |
| L3 | ~12 ns (40 cycles) | 8-64 MB | Office storage room |
| RAM | ~100 ns (400 cycles) | 8-128 GB | Warehouse across town |

When you access memory, the CPU checks L1 first. Miss? Check L2. Miss? Check L3. Miss? Go to RAM and wait 400 cycles.

## Cache Lines: Memory Comes in Chunks

Here's something that surprises most developers: you can't read a single byte from memory.

When you access address `0x1000`, the CPU doesn't fetch just that byte. It fetches an entire **cache line** — 64 bytes on x86 (128 bytes on Apple M-series).

```
Memory Address:     0x1000  0x1001  0x1002  ...  0x103F
                    ├───────────────────────────────────┤
                              64-byte cache line
```

This is why sequential access is fast:

```cpp
// C++: Sequential vs Random Access
#include <vector>
#include <random>
#include <chrono>
#include <iostream>

int main() {
    constexpr size_t N = 10'000'000;
    std::vector<int64_t> arr(N);
    std::vector<size_t> indices(N);

    // Fill with data
    for (size_t i = 0; i < N; ++i) {
        arr[i] = i;
        indices[i] = i;
    }

    // Shuffle indices for random access
    std::random_device rd;
    std::mt19937 g(rd());
    std::shuffle(indices.begin(), indices.end(), g);

    // Sequential access
    int64_t sum = 0;
    auto start = std::chrono::high_resolution_clock::now();
    for (size_t i = 0; i < N; ++i) {
        sum += arr[i];  // Predictable, prefetcher-friendly
    }
    auto seq_time = std::chrono::high_resolution_clock::now() - start;

    // Random access
    sum = 0;
    start = std::chrono::high_resolution_clock::now();
    for (size_t i = 0; i < N; ++i) {
        sum += arr[indices[i]];  // Unpredictable, cache-hostile
    }
    auto rand_time = std::chrono::high_resolution_clock::now() - start;

    std::cout << "Sequential: "
              << std::chrono::duration_cast<std::chrono::milliseconds>(seq_time).count()
              << " ms\n";
    std::cout << "Random:     "
              << std::chrono::duration_cast<std::chrono::milliseconds>(rand_time).count()
              << " ms\n";
}
```

**Typical output:**

```
Sequential: 12 ms
Random:     620 ms
```

Random access is **50x slower**. Same data, same operations. Just different access pattern.

The equivalent in Go:

```go
// Go version
for i := 0; i < len(arr); i++ {
    sum += arr[i]           // Fast: sequential
}
for i := 0; i < len(arr); i++ {
    sum += arr[indices[i]]  // Slow: random
}
```

### The Prefetcher: Your Silent Ally

The CPU has hardware that watches your access patterns. If you're reading addresses 0, 8, 16, 24... it predicts you'll want 32 next and fetches it before you ask.

This is why:
- Arrays are fast (predictable pattern)
- Linked lists are slow (unpredictable jumps)
- Binary trees are slow (random-ish access)

```
Sequential traversal:
Address: 0 → 64 → 128 → 192 → 256
Prefetcher: "I see a pattern!" → Prefetches ahead
Result: ~1 ns per access

Linked list traversal:
Address: 0 → 4096 → 128 → 8192 → 512
Prefetcher: "???" → Can't predict
Result: ~50-100 ns per access
```

### Row-Major vs Column-Major: The Matrix Trap

C and C++ store 2D arrays in **row-major** order. Row 0 is contiguous, then row 1, etc.

```
Logical view:           Memory layout:
┌───┬───┬───┐
│ 0 │ 1 │ 2 │  Row 0    [0][1][2][3][4][5][6][7][8]
├───┼───┼───┤            ─────── ─────── ───────
│ 3 │ 4 │ 5 │  Row 1     Row 0   Row 1   Row 2
├───┼───┼───┤
│ 6 │ 7 │ 8 │  Row 2
└───┴───┴───┘
```

Iterating by rows is cache-friendly. Iterating by columns is not:

```cpp
// matrix_traversal.cpp
#include <chrono>
#include <iostream>
#include <vector>

constexpr size_t SIZE = 10000;

int main() {
    // 10000 x 10000 matrix = 800 MB for int64_t
    std::vector<std::vector<int64_t>> matrix(SIZE, std::vector<int64_t>(SIZE, 1));

    int64_t sum = 0;

    // Row-major: sequential in memory
    auto start = std::chrono::high_resolution_clock::now();
    for (size_t i = 0; i < SIZE; ++i) {
        for (size_t j = 0; j < SIZE; ++j) {
            sum += matrix[i][j];  // Access: [0][0], [0][1], [0][2]...
        }
    }
    auto row_time = std::chrono::high_resolution_clock::now() - start;

    // Column-major: jumping around memory
    sum = 0;
    start = std::chrono::high_resolution_clock::now();
    for (size_t j = 0; j < SIZE; ++j) {
        for (size_t i = 0; i < SIZE; ++i) {
            sum += matrix[i][j];  // Access: [0][0], [1][0], [2][0]...
        }
    }
    auto col_time = std::chrono::high_resolution_clock::now() - start;

    std::cout << "Row-major:    "
              << std::chrono::duration_cast<std::chrono::milliseconds>(row_time).count()
              << " ms\n";
    std::cout << "Column-major: "
              << std::chrono::duration_cast<std::chrono::milliseconds>(col_time).count()
              << " ms\n";
}
```

**Output:**

```
Row-major:    245 ms
Column-major: 1842 ms
```

Column-major is **7.5x slower**. Same matrix, same operation, different loop order.

Why? Each column access jumps 10,000 elements (80 KB). Every single access is a cache miss.

### Linked List vs Vector: The Data Structure Choice

Linked lists are taught in every CS course. They're also terrible for performance.

```cpp
// list_vs_vector.cpp
#include <chrono>
#include <iostream>
#include <list>
#include <numeric>
#include <vector>

constexpr size_t N = 10'000'000;

int main() {
    // Vector: contiguous memory
    std::vector<int64_t> vec(N);
    std::iota(vec.begin(), vec.end(), 0);

    // List: scattered across heap
    std::list<int64_t> lst(vec.begin(), vec.end());

    int64_t sum = 0;

    // Sum vector
    auto start = std::chrono::high_resolution_clock::now();
    for (auto& val : vec) {
        sum += val;
    }
    auto vec_time = std::chrono::high_resolution_clock::now() - start;

    // Sum list
    sum = 0;
    start = std::chrono::high_resolution_clock::now();
    for (auto& val : lst) {
        sum += val;
    }
    auto lst_time = std::chrono::high_resolution_clock::now() - start;

    std::cout << "Vector: "
              << std::chrono::duration_cast<std::chrono::milliseconds>(vec_time).count()
              << " ms\n";
    std::cout << "List:   "
              << std::chrono::duration_cast<std::chrono::milliseconds>(lst_time).count()
              << " ms\n";
}
```

**Output:**

```
Vector: 8 ms
List:   87 ms
```

**10x slower** for linked list. Each node is a pointer chase. Each pointer chase is a cache miss.

This is why Bjarne Stroustrup says: *"Vector is almost always the best choice."*

## False Sharing: The Hidden Performance Killer

This is where it gets interesting. And painful.

Consider two goroutines, each incrementing its own counter:

```go
type Counters struct {
    a int64  // goroutine 1 uses this
    b int64  // goroutine 2 uses this
}

var counters Counters

// Goroutine 1
for i := 0; i < 1_000_000; i++ {
    counters.a++
}

// Goroutine 2
for i := 0; i < 1_000_000; i++ {
    counters.b++
}
```

They're accessing **different** variables. No data race. No shared state.

But it's slow. Really slow.

Why? Both `a` and `b` fit in the same 64-byte cache line:

```
Cache Line (64 bytes):
┌────────┬────────┬──────────────────────────────────────────┐
│   a    │   b    │              padding...                  │
│ 8 bytes│ 8 bytes│              48 bytes                    │
└────────┴────────┴──────────────────────────────────────────┘
    ↑         ↑
  Core 1    Core 2
  writes    writes
```

When Core 1 writes to `a`, the cache coherence protocol invalidates Core 2's copy of the entire cache line. Core 2 must re-fetch. Then Core 2 writes to `b`, invalidating Core 1's copy. Ping-pong. Forever.

### The Fix: Padding

```go
type CountersPadded struct {
    a   int64
    _   [56]byte  // padding to fill 64-byte cache line
    b   int64
    _   [56]byte
}
```

Now `a` and `b` are on different cache lines. No more ping-pong.

**Benchmark results:**

```
BenchmarkFalseSharing-8        1000    45,000 ns/op
BenchmarkPadded-8              1000     7,000 ns/op
```

**6.4x faster** by adding padding. Same logic. Just different memory layout.

### C++ Full Benchmark

```cpp
// false_sharing.cpp
// Compile: g++ -std=c++17 -O2 -pthread false_sharing.cpp -o false_sharing

#include <atomic>
#include <chrono>
#include <iostream>
#include <thread>
#include <new>  // for hardware_destructive_interference_size

constexpr size_t N = 100'000'000;

// Bad: both counters likely share a cache line
struct CountersBad {
    std::atomic<int64_t> a{0};
    std::atomic<int64_t> b{0};
};

// Good: each counter on its own cache line
struct CountersGood {
    alignas(64) std::atomic<int64_t> a{0};
    alignas(64) std::atomic<int64_t> b{0};
};

// Even better: use the standard constant (C++17)
// Note: not all compilers define this yet
#ifdef __cpp_lib_hardware_interference_size
struct CountersBest {
    alignas(std::hardware_destructive_interference_size)
        std::atomic<int64_t> a{0};
    alignas(std::hardware_destructive_interference_size)
        std::atomic<int64_t> b{0};
};
#endif

template<typename T>
auto benchmark() {
    T counters;

    auto start = std::chrono::high_resolution_clock::now();

    std::thread t1([&]() {
        for (size_t i = 0; i < N; ++i) {
            counters.a.fetch_add(1, std::memory_order_relaxed);
        }
    });

    std::thread t2([&]() {
        for (size_t i = 0; i < N; ++i) {
            counters.b.fetch_add(1, std::memory_order_relaxed);
        }
    });

    t1.join();
    t2.join();

    return std::chrono::high_resolution_clock::now() - start;
}

int main() {
    std::cout << "sizeof(CountersBad):  " << sizeof(CountersBad) << " bytes\n";
    std::cout << "sizeof(CountersGood): " << sizeof(CountersGood) << " bytes\n\n";

    auto bad_time = benchmark<CountersBad>();
    auto good_time = benchmark<CountersGood>();

    std::cout << "False sharing:  "
              << std::chrono::duration_cast<std::chrono::milliseconds>(bad_time).count()
              << " ms\n";
    std::cout << "With padding:   "
              << std::chrono::duration_cast<std::chrono::milliseconds>(good_time).count()
              << " ms\n";
    std::cout << "Speedup:        "
              << std::chrono::duration<double>(bad_time).count() /
                 std::chrono::duration<double>(good_time).count()
              << "x\n";
}
```

**Typical output:**

```
sizeof(CountersBad):  16 bytes
sizeof(CountersGood): 128 bytes

False sharing:  2847 ms
With padding:   498 ms
Speedup:        5.7x
```

The padded version uses 8x more memory (128 vs 16 bytes) but runs 5-6x faster. That's often a good trade.

### Alternative: Manual Padding

If `alignas` isn't available or you need more control:

```cpp
struct CountersManualPad {
    std::atomic<int64_t> a;
    char padding1[64 - sizeof(std::atomic<int64_t>)];  // Fill to 64 bytes
    std::atomic<int64_t> b;
    char padding2[64 - sizeof(std::atomic<int64_t>)];
};

static_assert(sizeof(CountersManualPad) == 128, "Check your padding");
```

## Cache Coherence: The MESI Protocol

How do multiple cores keep their caches consistent?

When Core 1 writes to address X, Core 2's cached copy of X becomes invalid. The hardware handles this automatically through the **MESI protocol**.

MESI stands for the four states a cache line can be in:

```
┌─────────────────────────────────────────────────────────────┐
│                      MESI States                            │
├──────────┬──────────────────────────────────────────────────┤
│ Modified │ I have the only copy. It's dirty (not in RAM).  │
│          │ I must write it back before anyone else reads.  │
├──────────┼──────────────────────────────────────────────────┤
│ Exclusive│ I have the only copy. It matches RAM.           │
│          │ I can modify it without telling anyone.         │
├──────────┼──────────────────────────────────────────────────┤
│ Shared   │ Multiple cores have this line. All match RAM.   │
│          │ Must notify others before modifying.            │
├──────────┼──────────────────────────────────────────────────┤
│ Invalid  │ This cache line is garbage. Don't use it.       │
│          │ Must fetch from another core or RAM.            │
└──────────┴──────────────────────────────────────────────────┘
```

### State Transitions

```
                    ┌──────────────┐
         ┌─────────→│   Invalid    │←─────────┐
         │          └──────────────┘          │
         │                 │                  │
         │           Read  │                  │ Other core
         │                 ↓                  │ writes
         │          ┌──────────────┐          │
         │     ┌───→│   Shared     │←───┐     │
         │     │    └──────────────┘    │     │
         │     │           │            │     │
   Other │     │     Write │            │     │
   core  │     │           ↓            │     │
   reads │     │    ┌──────────────┐    │     │
         │     │    │  Modified    │────┘     │
         │     │    └──────────────┘          │
         │     │           │                  │
         │     │   Writeback               │
         │     │           ↓                  │
         │     │    ┌──────────────┐          │
         │     └────│  Exclusive   │──────────┘
         │          └──────────────┘
         │                 │
         └─────────────────┘
                 Other core reads
```

### Why This Matters

Every time a core modifies data that another core has cached, it must:

1. Send an "I'm about to write" message
2. Wait for all other cores to invalidate their copies
3. Wait for acknowledgment
4. Only then, perform the write

This takes time. On a modern CPU, transitioning a cache line from Shared to Modified can take 40-100 cycles — roughly the same as going to RAM.

**This is why lock-free doesn't mean wait-free.** Even without locks, concurrent writes to nearby memory cause coherence traffic.

## Virtual Memory: Addresses Are Lies

Every address your program uses is fake.

When your Go program accesses address `0x7fff5678`, that's a **virtual address**. The CPU translates it to a **physical address** — the actual location in RAM.

Why? Three reasons:

1. **Isolation**: Process A can't access Process B's memory
2. **Flexibility**: Physical memory can be fragmented; virtual memory looks contiguous
3. **Overcommit**: You can allocate more virtual memory than physical RAM exists

### Page Tables

Memory is divided into **pages** (typically 4 KB). The OS maintains a **page table** that maps virtual pages to physical frames:

```
Virtual Address: 0x7fff5678
                 ├────────────┼──────────┤
                 │ Page Number│  Offset  │
                 │  (0x7fff5) │  (0x678) │
                 └────────────┴──────────┘
                       │
                       ↓
                 ┌─────────────────────┐
                 │    Page Table       │
                 ├─────────────────────┤
                 │ VPN 0x7fff5 → PFN X │
                 └─────────────────────┘
                       │
                       ↓
Physical Address: (X * 4096) + 0x678
```

But here's the problem: page tables are in memory. To translate one address, you need to read from memory. That's slow.

### Multi-Level Page Tables

Modern systems use hierarchical page tables (4 levels on x86-64):

```
Virtual Address (48 bits used):
┌─────────┬─────────┬─────────┬─────────┬──────────────┐
│  PML4   │  PDPT   │   PD    │   PT    │    Offset    │
│ 9 bits  │ 9 bits  │ 9 bits  │ 9 bits  │   12 bits    │
└─────────┴─────────┴─────────┴─────────┴──────────────┘
     │         │         │         │
     ↓         ↓         ↓         ↓
   Level 4 → Level 3 → Level 2 → Level 1 → Physical Page
```

Four memory accesses just to translate one address. Four trips to RAM. 400 cycles each. That's 1,600 cycles to read one byte.

Unacceptable.

## TLB: The Translation Cache

The **Translation Lookaside Buffer** (TLB) caches recent virtual-to-physical translations.

```
┌─────────────────────────────────────────────────────────────┐
│                          CPU                                │
│  ┌───────────────────────────────────────────────────────┐ │
│  │                        TLB                             │ │
│  │  ┌──────────────────┬──────────────────┐              │ │
│  │  │ Virtual Page     │ Physical Frame   │              │ │
│  │  ├──────────────────┼──────────────────┤              │ │
│  │  │ 0x7fff5          │ 0x1a3b7          │              │ │
│  │  │ 0x7fff4          │ 0x0042f          │              │ │
│  │  │ 0x00401          │ 0x89abc          │              │ │
│  │  │ ...              │ ...              │              │ │
│  │  └──────────────────┴──────────────────┘              │ │
│  │             32-1024 entries                           │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

**TLB hit**: Translation in ~1 cycle
**TLB miss**: Page walk through 4 levels, ~100-1000 cycles

The TLB is tiny (typically 64-1024 entries) but has a hit rate above 99% for most workloads. Why? Locality. Programs tend to access the same pages repeatedly.

### TLB Misses Kill Performance

Each TLB entry covers one page (4 KB). With 64 entries, you can cover 256 KB of memory with instant translations.

If your working set is larger, or you're jumping around memory randomly, TLB misses add up:

```go
// TLB-friendly: sequential access within pages
for i := 0; i < len(arr); i++ {
    sum += arr[i]  // Same page for many iterations
}

// TLB-hostile: jumping between many pages
for i := 0; i < 1000; i++ {
    for j := 0; j < 1000; j++ {
        sum += matrix[j][i]  // Different page each inner iteration
    }
}
```

### Huge Pages

One solution: use **huge pages** (2 MB or 1 GB instead of 4 KB).

- 4 KB page → TLB covers 256 KB with 64 entries
- 2 MB page → TLB covers 128 MB with 64 entries

**C++: Allocating Huge Pages (Linux)**

```cpp
// huge_pages.cpp
#include <sys/mman.h>
#include <cstdlib>
#include <iostream>

int main() {
    constexpr size_t SIZE = 256 * 1024 * 1024;  // 256 MB

    // Request huge pages with mmap
    void* ptr = mmap(
        nullptr,
        SIZE,
        PROT_READ | PROT_WRITE,
        MAP_PRIVATE | MAP_ANONYMOUS | MAP_HUGETLB,
        -1,
        0
    );

    if (ptr == MAP_FAILED) {
        // Fallback: try transparent huge pages
        ptr = mmap(
            nullptr,
            SIZE,
            PROT_READ | PROT_WRITE,
            MAP_PRIVATE | MAP_ANONYMOUS,
            -1,
            0
        );
        // Advise kernel to use huge pages
        madvise(ptr, SIZE, MADV_HUGEPAGE);
        std::cout << "Using transparent huge pages\n";
    } else {
        std::cout << "Using explicit huge pages\n";
    }

    // Use memory...
    int64_t* arr = static_cast<int64_t*>(ptr);
    for (size_t i = 0; i < SIZE / sizeof(int64_t); ++i) {
        arr[i] = i;
    }

    munmap(ptr, SIZE);
}
```

Enable transparent huge pages system-wide:

```bash
# Check current setting
cat /sys/kernel/mm/transparent_hugepage/enabled

# Enable
echo always | sudo tee /sys/kernel/mm/transparent_hugepage/enabled
```

In Go, you can't directly control this. But the OS will automatically use huge pages for large allocations if transparent huge pages are enabled.

## NUMA: When Memory Isn't Equal

On multi-socket servers, there's another layer of complexity.

**NUMA** = Non-Uniform Memory Access

```
┌─────────────────────────────────────────────────────────────┐
│                        Socket 0                             │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐          │
│  │ Core 0  │ │ Core 1  │ │ Core 2  │ │ Core 3  │          │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘          │
│       └──────────┬┴──────────┬┴──────────┘                │
│                  │  L3 Cache  │                            │
│                  └──────┬─────┘                            │
│                         │                                  │
│                  ┌──────┴──────┐                           │
│                  │  Memory     │ ← Local: ~100 ns          │
│                  │  Controller │                           │
│                  └──────┬──────┘                           │
└─────────────────────────┼───────────────────────────────────┘
                          │
                    Interconnect (QPI/UPI)
                          │
┌─────────────────────────┼───────────────────────────────────┐
│                  ┌──────┴──────┐                           │
│                  │  Memory     │ ← Remote: ~150 ns (+50%)  │
│                  │  Controller │                           │
│                  └──────┬──────┘                           │
│                         │                                  │
│                  ┌──────┴─────┐                            │
│                  │  L3 Cache  │                            │
│       ┌──────────┴┬──────────┴┬──────────┐                │
│  ┌────┴────┐ ┌────┴────┐ ┌────┴────┐ ┌────┴────┐          │
│  │ Core 4  │ │ Core 5  │ │ Core 6  │ │ Core 7  │          │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘          │
│                        Socket 1                             │
└─────────────────────────────────────────────────────────────┘
```

Each CPU socket has its own memory. Accessing local memory: ~100 ns. Accessing the other socket's memory: ~150 ns.

### The Problem NUMA Solves

Old approach (SMP - Symmetric Multi-Processing): All CPUs share one memory bus.

```
       CPU0    CPU1    CPU2    CPU3
         │       │       │       │
         └───────┴───┬───┴───────┘
                     │
              Shared Memory Bus ← Bottleneck!
                     │
                 ┌───┴───┐
                 │ Memory│
                 └───────┘
```

With many cores, the bus becomes saturated. Adding more CPUs doesn't help — they all fight for the same bus.

NUMA gives each socket its own memory controller. More sockets = more bandwidth. But now memory access is non-uniform.

### NUMA-Aware Programming

```go
// Bad: Thread on Socket 1 keeps accessing Socket 0's memory
data := make([]int, 1_000_000)  // Allocated on Socket 0
// ... thread migrates to Socket 1 ...
for i := range data {           // Every access crosses interconnect
    data[i]++
}

// Better: Keep data and threads together
// Use OS APIs to pin threads and allocate memory on same node
```

In Linux, you can use `numactl` or `libnuma` to control placement:

```bash
# Run process with memory and threads on node 0
numactl --cpunodebind=0 --membind=0 ./myprogram

# Check NUMA topology
numactl --hardware

# Monitor NUMA statistics
numastat -c
```

### C++: NUMA-Aware Memory Allocation

```cpp
// numa_aware.cpp
// Compile: g++ -O2 -lnuma numa_aware.cpp -o numa_aware

#include <numa.h>
#include <sched.h>
#include <chrono>
#include <iostream>
#include <thread>
#include <vector>

constexpr size_t SIZE = 100'000'000;

void benchmark_local() {
    // Pin thread to node 0
    numa_run_on_node(0);

    // Allocate memory on node 0
    int64_t* arr = static_cast<int64_t*>(
        numa_alloc_onnode(SIZE * sizeof(int64_t), 0)
    );

    // Access (same node)
    auto start = std::chrono::high_resolution_clock::now();
    int64_t sum = 0;
    for (size_t i = 0; i < SIZE; ++i) {
        sum += arr[i];
    }
    auto elapsed = std::chrono::high_resolution_clock::now() - start;

    std::cout << "Local access:  "
              << std::chrono::duration_cast<std::chrono::milliseconds>(elapsed).count()
              << " ms (sum=" << sum << ")\n";

    numa_free(arr, SIZE * sizeof(int64_t));
}

void benchmark_remote() {
    // Pin thread to node 0
    numa_run_on_node(0);

    // Allocate memory on node 1 (remote)
    int64_t* arr = static_cast<int64_t*>(
        numa_alloc_onnode(SIZE * sizeof(int64_t), 1)
    );

    // Access (cross-node)
    auto start = std::chrono::high_resolution_clock::now();
    int64_t sum = 0;
    for (size_t i = 0; i < SIZE; ++i) {
        sum += arr[i];
    }
    auto elapsed = std::chrono::high_resolution_clock::now() - start;

    std::cout << "Remote access: "
              << std::chrono::duration_cast<std::chrono::milliseconds>(elapsed).count()
              << " ms (sum=" << sum << ")\n";

    numa_free(arr, SIZE * sizeof(int64_t));
}

int main() {
    if (numa_available() < 0) {
        std::cerr << "NUMA not available\n";
        return 1;
    }

    std::cout << "NUMA nodes: " << numa_max_node() + 1 << "\n\n";

    benchmark_local();
    benchmark_remote();
}
```

Output on a 2-socket system:

```
NUMA nodes: 2

Local access:  89 ms
Remote access: 134 ms
```

Remote memory access is **50% slower**. For memory-bound workloads, this adds up fast.

### C++: Thread-Local Data Pattern

A common pattern for NUMA-aware code: each thread owns its data.

```cpp
// Per-thread data, allocated on thread's local node
struct alignas(64) ThreadData {
    std::vector<int64_t> local_data;
    int64_t result{0};
    // Padding to avoid false sharing between ThreadData instances
};

void worker(ThreadData& data, int node) {
    // Pin to specific NUMA node
    numa_run_on_node(node);

    // Allocate on local node
    data.local_data.resize(1'000'000);

    // Work on local data
    for (auto& val : data.local_data) {
        data.result += val;
    }
}
```

In Go, you have less control. The runtime moves goroutines between threads. But you can:

1. Use `GOMAXPROCS` to limit to one NUMA node's cores
2. Structure data to minimize cross-node access
3. Use worker pools where each worker owns its data

## Putting It All Together

When your program reads address X:

```
1. CPU generates virtual address X

2. TLB lookup
   ├─ Hit? → Got physical address (1 cycle)
   └─ Miss? → Page walk (100-1000 cycles)
              └─ Walk through 4 page table levels
              └─ Each level may miss cache → RAM access

3. L1 cache lookup
   ├─ Hit? → Got data (4 cycles)
   └─ Miss? → Check L2

4. L2 cache lookup
   ├─ Hit? → Got data (12 cycles)
   └─ Miss? → Check L3

5. L3 cache lookup
   ├─ Hit? → Got data (40 cycles)
   └─ Miss? → Go to RAM

6. RAM access
   ├─ Local NUMA node? → 100 ns
   └─ Remote NUMA node? → 150 ns

7. Data arrives (64-byte cache line)
   └─ Stored in L1, L2, L3
   └─ TLB entry created
   └─ Next access to same line: 4 cycles
```

Best case: 4 cycles.
Worst case: 2000+ cycles.

That's a 500x difference for the same instruction.

## Struct Layout: Field Ordering Matters

Compilers add padding to align fields. Bad ordering wastes memory and cache space.

```cpp
// struct_padding.cpp
#include <cstddef>
#include <iostream>

// Bad: 24 bytes due to padding
struct BadLayout {
    char a;      // 1 byte
                 // 7 bytes padding (align next field to 8)
    double b;    // 8 bytes
    char c;      // 1 byte
                 // 7 bytes padding (align struct to 8)
};

// Good: 16 bytes, no wasted space
struct GoodLayout {
    double b;    // 8 bytes
    char a;      // 1 byte
    char c;      // 1 byte
                 // 6 bytes padding (unavoidable for alignment)
};

// Check with offsetof
int main() {
    std::cout << "BadLayout:\n";
    std::cout << "  sizeof: " << sizeof(BadLayout) << "\n";
    std::cout << "  offsetof(a): " << offsetof(BadLayout, a) << "\n";
    std::cout << "  offsetof(b): " << offsetof(BadLayout, b) << "\n";
    std::cout << "  offsetof(c): " << offsetof(BadLayout, c) << "\n\n";

    std::cout << "GoodLayout:\n";
    std::cout << "  sizeof: " << sizeof(GoodLayout) << "\n";
    std::cout << "  offsetof(b): " << offsetof(GoodLayout, b) << "\n";
    std::cout << "  offsetof(a): " << offsetof(GoodLayout, a) << "\n";
    std::cout << "  offsetof(c): " << offsetof(GoodLayout, c) << "\n";
}
```

Output:

```
BadLayout:
  sizeof: 24
  offsetof(a): 0
  offsetof(b): 8
  offsetof(c): 16

GoodLayout:
  sizeof: 16
  offsetof(b): 0
  offsetof(a): 8
  offsetof(c): 9
```

**Rule of thumb**: order fields from largest to smallest.

```cpp
struct OptimalOrder {
    // 8-byte types first
    int64_t id;
    double value;
    void* ptr;

    // 4-byte types
    int32_t count;
    float ratio;

    // 2-byte types
    int16_t flags;

    // 1-byte types last
    char status;
    bool active;
};
```

## Practical Checklist

### Data Layout
- [ ] Keep related data together (same cache line)
- [ ] Order struct fields by size (largest first) to minimize padding
- [ ] Use arrays instead of pointer-heavy structures when possible
- [ ] Pad concurrent data to avoid false sharing

### Access Patterns
- [ ] Prefer sequential access over random
- [ ] Process data in cache-line-sized chunks
- [ ] Iterate row-major (how C/Go store arrays)

### Concurrency
- [ ] Separate read-heavy and write-heavy data
- [ ] Pad counters and locks used by different cores
- [ ] Consider per-core data structures

### NUMA (Multi-Socket Servers)
- [ ] Keep threads and their data on the same node
- [ ] Use `numactl` for pinning
- [ ] Monitor with `numastat`

## Benchmark: Everything Together

### C++ Comprehensive Benchmark

```cpp
// cache_benchmark.cpp
// Compile: g++ -std=c++17 -O2 -pthread cache_benchmark.cpp -o cache_benchmark

#include <atomic>
#include <chrono>
#include <iostream>
#include <random>
#include <thread>
#include <vector>

constexpr size_t N = 10'000'000;

// ============== False Sharing ==============
struct CountersBad {
    std::atomic<int64_t> a{0};
    std::atomic<int64_t> b{0};
};

struct CountersGood {
    alignas(64) std::atomic<int64_t> a{0};
    alignas(64) std::atomic<int64_t> b{0};
};

template<typename T>
auto bench_false_sharing() {
    T c;
    auto start = std::chrono::high_resolution_clock::now();

    std::thread t1([&]() {
        for (size_t i = 0; i < N; ++i)
            c.a.fetch_add(1, std::memory_order_relaxed);
    });
    std::thread t2([&]() {
        for (size_t i = 0; i < N; ++i)
            c.b.fetch_add(1, std::memory_order_relaxed);
    });

    t1.join();
    t2.join();
    return std::chrono::high_resolution_clock::now() - start;
}

// ============== Sequential vs Random ==============
auto bench_access_pattern() {
    std::vector<int64_t> arr(N);
    std::vector<size_t> indices(N);

    for (size_t i = 0; i < N; ++i) {
        arr[i] = i;
        indices[i] = i;
    }

    std::mt19937 rng(42);
    std::shuffle(indices.begin(), indices.end(), rng);

    int64_t sum = 0;

    // Sequential
    auto start = std::chrono::high_resolution_clock::now();
    for (size_t i = 0; i < N; ++i) sum += arr[i];
    auto seq_time = std::chrono::high_resolution_clock::now() - start;

    // Random
    sum = 0;
    start = std::chrono::high_resolution_clock::now();
    for (size_t i = 0; i < N; ++i) sum += arr[indices[i]];
    auto rand_time = std::chrono::high_resolution_clock::now() - start;

    return std::make_pair(seq_time, rand_time);
}

// ============== Row vs Column Major ==============
auto bench_matrix_traversal() {
    constexpr size_t SIZE = 4000;
    std::vector<std::vector<int64_t>> matrix(SIZE, std::vector<int64_t>(SIZE, 1));

    int64_t sum = 0;

    // Row-major
    auto start = std::chrono::high_resolution_clock::now();
    for (size_t i = 0; i < SIZE; ++i)
        for (size_t j = 0; j < SIZE; ++j)
            sum += matrix[i][j];
    auto row_time = std::chrono::high_resolution_clock::now() - start;

    // Column-major
    sum = 0;
    start = std::chrono::high_resolution_clock::now();
    for (size_t j = 0; j < SIZE; ++j)
        for (size_t i = 0; i < SIZE; ++i)
            sum += matrix[i][j];
    auto col_time = std::chrono::high_resolution_clock::now() - start;

    return std::make_pair(row_time, col_time);
}

int main() {
    using ms = std::chrono::milliseconds;

    std::cout << "=== Cache Performance Benchmarks ===\n\n";

    // False sharing
    auto bad = bench_false_sharing<CountersBad>();
    auto good = bench_false_sharing<CountersGood>();
    std::cout << "1. False Sharing (2 threads, 10M increments each)\n";
    std::cout << "   Without padding: " << std::chrono::duration_cast<ms>(bad).count() << " ms\n";
    std::cout << "   With padding:    " << std::chrono::duration_cast<ms>(good).count() << " ms\n";
    std::cout << "   Speedup:         " << (double)bad.count() / good.count() << "x\n\n";

    // Access pattern
    auto [seq, rnd] = bench_access_pattern();
    std::cout << "2. Access Pattern (10M elements)\n";
    std::cout << "   Sequential: " << std::chrono::duration_cast<ms>(seq).count() << " ms\n";
    std::cout << "   Random:     " << std::chrono::duration_cast<ms>(rnd).count() << " ms\n";
    std::cout << "   Slowdown:   " << (double)rnd.count() / seq.count() << "x\n\n";

    // Matrix traversal
    auto [row, col] = bench_matrix_traversal();
    std::cout << "3. Matrix Traversal (4000x4000)\n";
    std::cout << "   Row-major:    " << std::chrono::duration_cast<ms>(row).count() << " ms\n";
    std::cout << "   Column-major: " << std::chrono::duration_cast<ms>(col).count() << " ms\n";
    std::cout << "   Slowdown:     " << (double)col.count() / row.count() << "x\n";
}
```

Typical output:

```
=== Cache Performance Benchmarks ===

1. False Sharing (2 threads, 10M increments each)
   Without padding: 287 ms
   With padding:    48 ms
   Speedup:         5.98x

2. Access Pattern (10M elements)
   Sequential: 12 ms
   Random:     614 ms
   Slowdown:   51.2x

3. Matrix Traversal (4000x4000)
   Row-major:    38 ms
   Column-major: 298 ms
   Slowdown:     7.8x
```

### Go Version

```go
package main

import (
    "fmt"
    "math/rand"
    "sync"
    "sync/atomic"
    "time"
)

const N = 10_000_000

type CountersBad struct {
    a int64
    b int64
}

type CountersGood struct {
    a int64
    _ [56]byte
    b int64
}

func main() {
    // False sharing
    var bad CountersBad
    var good CountersGood
    var wg sync.WaitGroup

    start := time.Now()
    wg.Add(2)
    go func() { defer wg.Done(); for i := 0; i < N; i++ { atomic.AddInt64(&bad.a, 1) } }()
    go func() { defer wg.Done(); for i := 0; i < N; i++ { atomic.AddInt64(&bad.b, 1) } }()
    wg.Wait()
    badTime := time.Since(start)

    start = time.Now()
    wg.Add(2)
    go func() { defer wg.Done(); for i := 0; i < N; i++ { atomic.AddInt64(&good.a, 1) } }()
    go func() { defer wg.Done(); for i := 0; i < N; i++ { atomic.AddInt64(&good.b, 1) } }()
    wg.Wait()
    goodTime := time.Since(start)

    fmt.Printf("False sharing: %v\n", badTime)
    fmt.Printf("With padding:  %v\n", goodTime)
    fmt.Printf("Speedup:       %.2fx\n", float64(badTime)/float64(goodTime))
}
```

Same patterns, same speedups. The hardware doesn't care what language you use.

## References

- [What Every Programmer Should Know About Memory - Ulrich Drepper](https://people.freebsd.org/~lstewart/articles/cpumemory.pdf)
- [CPU Caches and Why You Care - Scott Meyers](https://www.aristeia.com/TalkNotes/ACCU2011_CPUCaches.pdf)
- [Gallery of Processor Cache Effects](http://igoro.com/archive/gallery-of-processor-cache-effects/)
- [False Sharing - Wikipedia](https://en.wikipedia.org/wiki/False_sharing)
- [MESI Protocol - Wikipedia](https://en.wikipedia.org/wiki/MESI_protocol)
- [NUMA Overview - ACM Queue](https://queue.acm.org/detail.cfm?id=2513149)
- [TLB in Paging - GeeksforGeeks](https://www.geeksforgeeks.org/translation-lookaside-buffer-tlb-in-paging/)
- [Go 101: Cache Lines](https://g4s8.wtf/posts/go-cashlines/)
- [100 Go Mistakes: False Sharing](https://100go.co/92-false-sharing/)
