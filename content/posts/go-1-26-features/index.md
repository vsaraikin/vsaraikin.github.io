---
title: "Go 1.26: Everything That Ships in February 2026"
date: 2026-02-01
draft: false
description: "Green Tea GC is now default, cgo is 30% faster, new(expr) works, SIMD lands as an experiment, and 20+ go fix modernizers can rewrite your code. The full breakdown."
tags: ["go"]
---

Go 1.26 ships more runtime and tooling changes than any release since 1.22. The Green Tea garbage collector is now the default, cgo overhead dropped 30%, small object allocation got specialized routines, and `go fix` was rebuilt from scratch with 20+ code modernizers.

Here's every change worth knowing about.

## Green Tea GC: Now the Default

The experimental garbage collector from Go 1.25 is now enabled for everyone. If you didn't opt into `GOEXPERIMENT=greenteagc` before, you get it automatically now.

The core change: instead of scanning individual objects scattered across the heap, Green Tea scans entire 8 KiB memory pages. This means contiguous memory access instead of pointer-chasing, which makes CPU prefetching actually work.

**Real-world impact:**

- 10-40% reduction in GC CPU overhead, depending on workload
- Additional ~10% improvement on CPUs with AVX-512 (Intel Ice Lake+, AMD Zen 4+), thanks to vectorized scanning
- Programs that spend 10% of time in GC can expect 1-4% total CPU savings

To opt out (this escape hatch disappears in Go 1.27):

```bash
GOEXPERIMENT=nogreenteagc go build ./...
```

If your service has significant GC pressure, this is the single biggest performance win in 1.26 — and you get it for free.

## Faster cgo: 30% Less Overhead

The runtime eliminated the `_Psyscall` processor state, which was an intermediate state that goroutines transitioned through during cgo calls. Removing it cuts the baseline overhead of every cgo call by about 30%.

```
BenchmarkCgoCall-8    28.55ns → 19.02ns    (-33%)
```

If you call into C libraries heavily — FFI-heavy crypto, database drivers, media processing — this adds up. If you don't use cgo, you won't notice.

## Faster Small Object Allocation

Go now generates calls to size-specialized memory allocation routines for objects between 1 and 512 bytes. Instead of routing every allocation through a single universal `mallocgc`, the runtime picks a specialized function tuned for the object's size class.

Up to 30% faster allocation for small objects. The Go team estimates ~1% total improvement for real programs with heavy allocation rates. Not dramatic, but free.

## `new(expr)`: Finally Useful

The `new` built-in used to only accept a type: `new(int)` returned a `*int` pointing to zero. Now it accepts expressions:

```go
p := new(42)       // *int pointing to 42
s := new([]int{1, 2, 3})  // *[]int

// Where it actually matters: optional fields in structs
type Config struct {
    Timeout *int `json:"timeout,omitempty"`
}

c := Config{
    Timeout: new(30),  // Before: func intPtr(i int) *int { return &i }
}
```

This eliminates the single most common helper function in Go codebases: the one-liner that takes a value and returns a pointer to it. Every project has `func ptr[T any](v T) *T { return &v }` somewhere. Now you don't need it.

## Self-Referential Generic Types

The restriction that a generic type couldn't reference itself in its type parameter list is gone:

```go
type Ordered[T Ordered[T]] interface {
    Less(T) bool
}

type Tree[T Ordered[T]] struct {
    nodes []T
}
```

This enables patterns like F-bounded polymorphism — common in Java and Scala, previously impossible in Go generics. Useful for builder patterns, comparable interfaces, and recursive data structures.

## `errors.AsType`: Type-Safe Error Handling

`errors.As` requires a pointer-to-pointer argument and uses reflection. The new generic `errors.AsType` is cleaner and ~3x faster:

```go
// Before
var appErr *AppError
if errors.As(err, &appErr) {
    fmt.Println(appErr.Code)
}

// After
if appErr, ok := errors.AsType[*AppError](err); ok {
    fmt.Println(appErr.Code)
}
```

No reflection, no pointer-to-pointer dance, compile-time type checking. This is what `errors.As` should have been from the start.

## `go fix`: Rebuilt From Scratch

The old `go fix` was a graveyard of obsolete refactoring rules from the Go 1.0 era. Nobody used it.

In 1.26, `go fix` is rebuilt on the same analysis framework as `go vet`, but with a different purpose: **safe, automatic code modernization**. Over 20 analyzers that update your code to use newer language features and stdlib APIs.

Example — replacing manual loops with `slices.Contains`:

```go
// Before go fix
func find(s []int, x int) bool {
    for _, v := range s {
        if x == v {
            return true
        }
    }
    return false
}

// After go fix
func find(s []int, x int) bool {
    return slices.Contains(s, x)
}
```

Run specific fixers or all of them:

```bash
go fix ./...                    # all modernizers
go fix -forvar ./...            # specific fixer
go fix -omitzero=false ./...    # disable specific fixer
```

The key difference from `go vet`: these fixes are safe to apply automatically. They modernize, they don't find bugs.

## `io.ReadAll`: 2x Faster

`io.ReadAll` now uses exponentially-sized intermediate buffers instead of growing linearly. Result: **~2x faster, ~50% less memory** for typical payloads.

```
ReadAll/65536-8    12,500ns → 6,250ns    (-50%)
allocs:            4,096B → 2,048B       (-50%)
```

If you read HTTP response bodies, file contents, or subprocess output — and nearly every Go program does — this helps everywhere with zero code changes.

## `slog.NewMultiHandler`: Log to Multiple Destinations

Write logs to multiple places without third-party libraries:

```go
stdoutHandler := slog.NewTextHandler(os.Stdout, nil)

file, _ := os.OpenFile("/tmp/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
fileHandler := slog.NewJSONHandler(file, nil)

logger := slog.New(slog.NewMultiHandler(stdoutHandler, fileHandler))
logger.Info("login", slog.String("user", "alice"), slog.Int("id", 42))
// Writes to both stdout (text) and file (JSON)
```

`Enabled()` returns true if _any_ handler is enabled at that level.

## Reflect Iterators

Range over struct fields, methods, and function parameters:

```go
typ := reflect.TypeFor[http.Client]()

for f := range typ.Fields() {
    fmt.Println(f.Name, f.Type)
}

for m := range typ.Methods() {
    fmt.Println(m.Name, m.Type)
}
```

Cleaner than indexing with `Field(i)` in a `for i := 0; i < typ.NumField(); i++` loop.

## `bytes.Buffer.Peek`

Read from a buffer without advancing the position:

```go
buf := bytes.NewBufferString("Hello World")
sample, _ := buf.Peek(5)  // "Hello" — position unchanged
```

Useful for protocol parsing where you need to inspect the next bytes before deciding how to handle them.

## Experimental: SIMD Instructions

`simd/archsimd` gives direct access to architecture-specific vector operations. Currently AMD64 only, with 128-bit, 256-bit, and 512-bit types.

```go
//go:build goexperiment.simd

import "simd/archsimd"

if archsimd.X86.AVX512() {
    va := archsimd.LoadFloat32x16Slice(a[i : i+16])
    vb := archsimd.LoadFloat32x16Slice(b[i : i+16])
    vSum := va.Add(vb)
    vSum.StoreFloat32x16Slice(result[i : i+16])
}
```

Enable with `GOEXPERIMENT=simd`. This is the first time Go provides explicit SIMD access without CGo or assembly stubs. Early days, but the direction is significant.

## Experimental: Goroutine Leak Detection

A new pprof profile type that detects goroutines blocked indefinitely on unreachable synchronization objects:

```go
// This leaks goroutines — now detectable
func processItems(items []Item) ([]Result, error) {
    ch := make(chan result)
    for _, item := range items {
        go func() {
            res, err := process(item)
            ch <- result{res, err}
        }()
    }
    for range len(items) {
        r := <-ch
        if r.err != nil {
            return nil, r.err  // Early return — remaining goroutines leak
        }
    }
    // ...
}
```

Enable with `GOEXPERIMENT=goroutineleakprofile`. Access via `/debug/pprof/goroutineleak`. Expected to become default in Go 1.27.

## Experimental: `runtime/secret`

Secure erasure of sensitive data after use — zeroes registers, stack, and new heap allocations:

```go
//go:build goexperiment.runtimesecret

import "runtime/secret"

secret.Do(func() {
    privKey, _ := ecdh.P256().GenerateKey(rand.Reader)
    shared, _ := privKey.ECDH(peerPubKey)
    sessionKey = deriveKey(shared)
    // privKey and shared are securely erased after Do() returns
})
```

AMD64 and ARM64 on Linux only. For cryptographic applications where key material must not linger in memory.

## Post-Quantum Crypto Enabled by Default

TLS connections now use hybrid post-quantum key exchanges (`SecP256r1MLKEM768`, `SecP384r1MLKEM1024`) by default. No code changes needed — `crypto/tls` negotiates them automatically.

Also new: `crypto/hpke` package implementing Hybrid Public Key Encryption (RFC 9180).

Disable post-quantum if needed: `GODEBUG=tlssecpmlkem=0`.

## Crypto APIs Drop `io.Reader` Parameters

Cryptographic functions that accepted an `io.Reader` for randomness now ignore it and always use the system CSPRNG:

```go
// The nil is fine — the Reader parameter is ignored anyway
key, _ := ecdsa.GenerateKey(elliptic.P256(), nil)
```

For deterministic testing, use the new `testing/cryptotest.SetGlobalRandom(t, seed)`.

## Everything Else

**`fmt.Errorf` optimization** — unformatted calls (`fmt.Errorf("failed")`) now match `errors.New` performance. ~92% faster for the common case.

**Heap address randomization** — on 64-bit platforms, the runtime randomizes the heap base address at startup. Security hardening against address prediction attacks in cgo programs.

**`os.Process.WithHandle`** — access the underlying process handle (pidfd on Linux 5.4+, handle on Windows) for reliable process management.

**`signal.NotifyContext` cause** — the signal that triggered cancellation is now available via `context.Cause(ctx)`.

**`net.Dialer` typed methods** — `DialTCP`, `DialUDP`, `DialIP`, `DialUnix` with context support.

**`netip.Prefix.Compare`** — sort IP prefixes with `slices.SortFunc(prefixes, netip.Prefix.Compare)`.

**`testing.T.ArtifactDir()`** — dedicated directory for test output files, accessible via `-artifacts` flag.

**`image/jpeg` rewrite** — new encoder/decoder that's faster and more accurate. Output may differ bit-for-bit from previous versions.

**pprof flame graphs** — the web UI (`-http` flag) now defaults to flame graph view instead of the old graph.

**Platform notes:**

- Last release supporting macOS 12 Monterey (1.27 requires Ventura)
- 32-bit `windows/arm` removed
- `linux/riscv64` now supports the race detector
- WebAssembly heap management uses smaller increments — significant memory savings for small heaps
