---
title: "Go's unsafe Package: Power, Peril, and Practical Patterns"
date: 2025-10-29
draft: false
description: "A deep dive into Go's unsafe package: what it is, how it works, when to use it, and why you probably shouldn't."
---

Go is a safe language. The compiler catches type mismatches. The runtime prevents buffer overflows. The garbage collector manages memory.

Then there's `unsafe`.

The `unsafe` package lets you bypass all of Go's safety guarantees. It gives you raw memory access, pointer arithmetic, and the ability to reinterpret any bytes as any type.

This article explains what `unsafe` actually does, how it works under the hood, and when it's worth the risk.

## What Is unsafe?

The [`unsafe`](https://pkg.go.dev/unsafe) package provides three things:

1. **`unsafe.Pointer`** - A pointer that can point to any type
2. **Type functions** - `Sizeof`, `Alignof`, `Offsetof`
3. **Memory functions** - `Add`, `Slice`, `String`, `SliceData`, `StringData`

That's it. No magic. Just raw memory access.

```go
import "unsafe"

// Convert any pointer to unsafe.Pointer
var x int = 42
ptr := unsafe.Pointer(&x)

// Convert unsafe.Pointer to any pointer type
y := *(*float64)(ptr) // Reinterpret int bits as float64
```

The package exists because sometimes you need to:

- Talk to C code via cgo
- Implement low-level data structures
- Squeeze out every last nanosecond of performance

## The Types

### unsafe.Pointer

`unsafe.Pointer` is the escape hatch from Go's type system. Four conversions are allowed:

```go
// 1. Any pointer → unsafe.Pointer
var x int = 42
p := unsafe.Pointer(&x)

// 2. unsafe.Pointer → any pointer
f := (*float64)(p)

// 3. unsafe.Pointer → uintptr (for arithmetic)
addr := uintptr(p)

// 4. uintptr → unsafe.Pointer (dangerous!)
p2 := unsafe.Pointer(addr)
```

The first two are useful. The last two are dangerous.

### uintptr: The Trap

`uintptr` is just an integer. It holds a memory address, but the garbage collector doesn't know that.

```go
// DANGEROUS: Don't do this
ptr := unsafe.Pointer(&x)
addr := uintptr(ptr)  // Now it's just a number
// GC can move or free x here!
ptr2 := unsafe.Pointer(addr)  // May point to garbage
```

The garbage collector can move objects at any time. If you store an address in a `uintptr`, the GC won't update it. Your pointer becomes invalid.

**Rule**: Never store `uintptr` in a variable. Convert back to `unsafe.Pointer` in the same expression.

```go
// CORRECT: Same expression
nextField := unsafe.Pointer(uintptr(ptr) + offset)

// WRONG: Stored in variable
addr := uintptr(ptr)  // GC can invalidate this
// ... any code here ...
nextField := unsafe.Pointer(addr)  // Undefined behavior
```

## The Functions

### Sizeof, Alignof, Offsetof

These are compile-time constants. No runtime cost.

```go
type User struct {
    ID   int64
    Name string
    Age  int32
}

unsafe.Sizeof(User{})      // 32 (on 64-bit)
unsafe.Alignof(User{})     // 8
unsafe.Offsetof(User{}.Age) // 24
```

`Offsetof` is particularly useful for accessing struct fields by offset:

```go
func getAge(u *User) int32 {
    ptr := unsafe.Pointer(u)
    agePtr := unsafe.Add(ptr, unsafe.Offsetof(u.Age))
    return *(*int32)(agePtr)
}
```

### unsafe.Add

Pointer arithmetic without the `uintptr` dance:

```go
next := unsafe.Add(ptr, 8)
```

Cleaner and safer than the old `unsafe.Pointer(uintptr(ptr) + 8)` pattern. The compiler can verify it.

### unsafe.Slice

Create a slice from a pointer and length:

```go
slice := unsafe.Slice((*byte)(ptr), length)
```

### unsafe.String, unsafe.StringData

Create strings from bytes and vice versa:

```go
// Bytes → String (zero-copy)
b := []byte("hello")
s := unsafe.String(&b[0], len(b))

// String → underlying bytes pointer
ptr := unsafe.StringData(s)
```

**Warning**: Since strings are immutable, you must not modify the bytes after creating a string from them.

### unsafe.SliceData (Go 1.20+)

Get the underlying array pointer from a slice:

```go
s := []int{1, 2, 3}
ptr := unsafe.SliceData(s)  // *int pointing to first element
```

## The Six Valid Patterns

The [Go documentation specifies exactly six valid patterns](https://pkg.go.dev/unsafe#Pointer) for `unsafe.Pointer`. Everything else is undefined behavior.

### Pattern 1: Type Conversion

Convert between pointer types with compatible memory layouts:

```go
// int64 and float64 have same size
var i int64 = 0x4059000000000000
f := *(*float64)(unsafe.Pointer(&i))
fmt.Println(f)  // 100.0
```

**Requirement**: T2 must be no larger than T1.

### Pattern 2: unsafe.Pointer → uintptr → unsafe.Pointer (Same Expression)

```go
// Advance pointer by offset
p := unsafe.Pointer(uintptr(ptr) + offset)
```

Both conversions must be in the same expression. No storing in variables.

### Pattern 3: Calling reflect.Value.Pointer() or UnsafeAddr()

```go
// Must convert immediately in same expression
p := (*int)(unsafe.Pointer(reflect.ValueOf(&x).Pointer()))
```

### Pattern 4: Syscall Arguments

The compiler has special handling for syscalls:

```go
// Compiler keeps the object alive during syscall
syscall.Syscall(SYS_READ, fd, uintptr(unsafe.Pointer(&buf[0])), len(buf))
```

**Warning**: You cannot store the `uintptr` in a variable first.

### Pattern 5: unsafe.Add, unsafe.Slice

```go
next := unsafe.Add(ptr, 8)
slice := unsafe.Slice((*byte)(ptr), length)
```

## Zero-Copy String Conversion

The most common use of `unsafe`: converting between `string` and `[]byte` without allocation.

### The Problem

Standard conversion copies data:

```go
s := "hello"
b := []byte(s)  // Allocates new []byte, copies 5 bytes

b2 := []byte("hello")
s2 := string(b2)  // Allocates new string, copies 5 bytes
```

For large strings in hot paths, this hurts performance.

### The Solution

```go
// String → []byte (zero-copy, READ-ONLY!)
func StringToBytes(s string) []byte {
    return unsafe.Slice(unsafe.StringData(s), len(s))
}

// []byte → String (zero-copy)
func BytesToString(b []byte) string {
    return unsafe.String(unsafe.SliceData(b), len(b))
}
```

### Compiler Optimization

The compiler can [optimize standard conversions](https://go.dev/doc/go1.22#compiler) in some cases:

```go
// The compiler may optimize this to zero-copy
// if b doesn't escape and isn't modified
s := string(b)
```

The compiler detects when the result doesn't escape to the heap and isn't modified, then skips the copy. Check if your code benefits before reaching for `unsafe`.

### Critical Warning

**Never modify a byte slice obtained from a string:**

```go
s := "hello"
b := StringToBytes(s)
b[0] = 'H'  // UNDEFINED BEHAVIOR!
```

Strings are immutable. The bytes might live in read-only memory. Modifying them can crash your program or corrupt memory silently.

## Real-World Usage: How Projects Use unsafe

### fasthttp

[fasthttp](https://github.com/valyala/fasthttp) processes 200K requests/second using zero-allocation patterns:

```go
// From fasthttp: zero-copy string conversion
func b2s(b []byte) string {
    return *(*string)(unsafe.Pointer(&b))
}

func s2b(s string) []byte {
    return unsafe.Slice(unsafe.StringData(s), len(s))
}
```

They use `unsafe` throughout for:

- Converting request/response bodies without copying
- Reusing buffers across requests
- Avoiding allocations in hot paths

### Standard Library

Even Go's standard library uses `unsafe`:

- **reflect**: Implements type introspection
- **sync**: Atomic operations on complex types
- **runtime**: Obviously needs raw memory access
- **strings.Builder**: Uses `unsafe` internally for efficiency

```go
// From strings.Builder
func (b *Builder) String() string {
    return unsafe.String(unsafe.SliceData(b.buf), len(b.buf))
}
```

### Protocol Buffers

Libraries like [molecule](https://github.com/richardartoul/molecule) use `unsafe` for zero-allocation parsing:

```go
// Returns unsafe view over underlying bytes
func (v Value) AsBytesUnsafe() []byte {
    return unsafe.Slice((*byte)(v.ptr), v.len)
}
```

## Performance: Is It Worth It?

Benchmark: string ↔ []byte conversion on a 1KB string.

```text
BenchmarkStandardConversion-8    5000000    234 ns/op    1024 B/op    1 allocs/op
BenchmarkUnsafeConversion-8     500000000   2.3 ns/op       0 B/op    0 allocs/op
```

100x faster. Zero allocations.

But here's the thing: 234ns is nothing if you're doing I/O. Network calls take milliseconds. Database queries take milliseconds. That 234ns is noise.

**Use unsafe for:**

- HTTP servers handling 100K+ req/s
- Protocol parsers processing GB/s
- Hot loops in data processing pipelines

**Don't use unsafe for:**

- Normal application code
- Code that does I/O
- Anything where safety matters more than speed

## The Dangers

### 1. Memory Corruption

```go
s := "hello"
b := unsafe.Slice(unsafe.StringData(s), len(s))
b[0] = 'H'  // May corrupt memory or crash
```

### 2. Race Conditions

```go
// Shared byte slice used as string
var shared []byte

go func() {
    s := BytesToString(shared)  // Zero-copy
    fmt.Println(s)
}()

go func() {
    shared[0] = 'X'  // Race! Other goroutine sees corruption
}()
```

### 3. GC Hazards

```go
func bad() *int {
    x := 42
    addr := uintptr(unsafe.Pointer(&x))
    // x goes out of scope, GC can reclaim it
    return (*int)(unsafe.Pointer(addr))  // Dangling pointer
}
```

### 4. Platform Dependence

```go
// Assumes 64-bit pointers
type Header struct {
    ptr uintptr  // 8 bytes on 64-bit, 4 bytes on 32-bit
}
```

Your code may break on different architectures.

## Tools for Safety

### go vet

Catches common `unsafe` misuses:

```bash
go vet ./...
```

### Race Detector

Finds data races involving `unsafe`:

```bash
go test -race ./...
go run -race main.go
```

### [checkptr](https://pkg.go.dev/cmd/compile#hdr-Compiler_Directives)

Runtime checking for `unsafe.Pointer` rules:

```bash
go build -gcflags=all=-d=checkptr ./...
```

Enabled by default with `-race` or `-msan`.

`unsafe` gives you raw memory access and zero-copy conversions at the cost of type safety, memory safety, and portability. Profile first, isolate the unsafe code behind a clean API, and always test with `-race`. For 99% of Go code, you don't need it.

## References

- [unsafe package documentation](https://pkg.go.dev/unsafe)
- [Go 101: Type-Unsafe Pointers](https://go101.org/article/unsafe.html)
- [Safe Use of unsafe.Pointer - Gopher Academy](https://blog.gopheracademy.com/advent-2019/safe-use-of-unsafe-pointer/)
- [Exploring unsafe Features in Go 1.20](https://medium.com/@bradford_hamilton/exploring-unsafe-features-in-go-1-20-a-hands-on-demo-7149ba82e6e1)
- [fasthttp - Zero-allocation HTTP](https://github.com/valyala/fasthttp)
- [unsafe.String and unsafe.StringData](https://boldlygo.tech/archive/2025-01-28-unsafe.string-and-unsafe.stringdata/)
