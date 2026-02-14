/**
 * Atomic vs Mutex Benchmark (Go)
 *
 * Run: go test -bench=. -benchmem
 */

package atomics

import (
	"sync"
	"sync/atomic"
	"testing"
)

// =============================================================================
// Global state for benchmarks
// =============================================================================

var (
	atomicCounter  int64
	atomic64       atomic.Int64
	mutexCounter   int64
	rwMutexCounter int64
	mu             sync.Mutex
	rwMu           sync.RWMutex
)

// =============================================================================
// Single-threaded benchmarks
// =============================================================================

func BenchmarkAtomicAdd_Single(b *testing.B) {
	for i := 0; i < b.N; i++ {
		atomic.AddInt64(&atomicCounter, 1)
	}
}

func BenchmarkAtomicInt64_Single(b *testing.B) {
	for i := 0; i < b.N; i++ {
		atomic64.Add(1)
	}
}

func BenchmarkMutex_Single(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mu.Lock()
		mutexCounter++
		mu.Unlock()
	}
}

func BenchmarkRWMutexWrite_Single(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rwMu.Lock()
		rwMutexCounter++
		rwMu.Unlock()
	}
}

// =============================================================================
// Multi-threaded benchmarks (parallel)
// =============================================================================

func BenchmarkAtomicAdd_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			atomic.AddInt64(&atomicCounter, 1)
		}
	})
}

func BenchmarkAtomicInt64_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			atomic64.Add(1)
		}
	})
}

func BenchmarkMutex_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			mutexCounter++
			mu.Unlock()
		}
	})
}

func BenchmarkRWMutexWrite_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rwMu.Lock()
			rwMutexCounter++
			rwMu.Unlock()
		}
	})
}

// =============================================================================
// CAS benchmarks
// =============================================================================

var casCounter int64

func BenchmarkCAS_Single(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for {
			old := atomic.LoadInt64(&casCounter)
			if atomic.CompareAndSwapInt64(&casCounter, old, old+1) {
				break
			}
		}
	}
}

func BenchmarkCAS_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for {
				old := atomic.LoadInt64(&casCounter)
				if atomic.CompareAndSwapInt64(&casCounter, old, old+1) {
					break
				}
			}
		}
	})
}

// =============================================================================
// Read-heavy benchmarks (atomic.Value vs RWMutex)
// =============================================================================

type Config struct {
	Value int
}

var (
	atomicConfig atomic.Value
	mutexConfig  Config
	configMu     sync.RWMutex
)

func init() {
	atomicConfig.Store(Config{Value: 42})
	mutexConfig = Config{Value: 42}
}

func BenchmarkAtomicValue_Read(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = atomicConfig.Load().(Config)
	}
}

func BenchmarkRWMutex_Read(b *testing.B) {
	for i := 0; i < b.N; i++ {
		configMu.RLock()
		_ = mutexConfig
		configMu.RUnlock()
	}
}

func BenchmarkAtomicValue_ReadParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = atomicConfig.Load().(Config)
		}
	})
}

func BenchmarkRWMutex_ReadParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			configMu.RLock()
			_ = mutexConfig
			configMu.RUnlock()
		}
	})
}

// =============================================================================
// False sharing demonstration
// =============================================================================

type CountersWithFalseSharing struct {
	a int64
	b int64
}

type CountersPadded struct {
	a int64
	_ [56]byte
	b int64
	_ [56]byte
}

func BenchmarkFalseSharing(b *testing.B) {
	var c CountersWithFalseSharing
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			atomic.AddInt64(&c.a, 1)
			atomic.AddInt64(&c.b, 1)
		}
	})
}

func BenchmarkNoPadding_Separate(b *testing.B) {
	var c CountersWithFalseSharing
	b.SetParallelism(2)
	b.RunParallel(func(pb *testing.PB) {
		// Half threads increment a, half increment b
		id := atomic.AddInt64(&c.a, 0) % 2
		for pb.Next() {
			if id == 0 {
				atomic.AddInt64(&c.a, 1)
			} else {
				atomic.AddInt64(&c.b, 1)
			}
		}
	})
}

func BenchmarkWithPadding_Separate(b *testing.B) {
	var c CountersPadded
	var id int64
	b.RunParallel(func(pb *testing.PB) {
		myId := atomic.AddInt64(&id, 1) % 2
		for pb.Next() {
			if myId == 0 {
				atomic.AddInt64(&c.a, 1)
			} else {
				atomic.AddInt64(&c.b, 1)
			}
		}
	})
}
