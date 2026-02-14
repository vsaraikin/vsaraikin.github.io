/**
 * Lock-Free Stack Implementation in Go
 *
 * Uses atomic.Pointer and CAS to implement a thread-safe stack
 * without any locks.
 *
 * Run: go run lock_free_stack.go
 */

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// =============================================================================
// Lock-Free Stack
// =============================================================================

type Node struct {
	value int
	next  *Node
}

type LockFreeStack struct {
	head atomic.Pointer[Node]
	size atomic.Int64
}

func NewLockFreeStack() *LockFreeStack {
	return &LockFreeStack{}
}

func (s *LockFreeStack) Push(value int) {
	newNode := &Node{value: value}
	for {
		oldHead := s.head.Load()
		newNode.next = oldHead
		if s.head.CompareAndSwap(oldHead, newNode) {
			s.size.Add(1)
			return
		}
		// CAS failed, another goroutine modified head; retry
	}
}

func (s *LockFreeStack) Pop() (int, bool) {
	for {
		oldHead := s.head.Load()
		if oldHead == nil {
			return 0, false // Stack is empty
		}
		newHead := oldHead.next
		if s.head.CompareAndSwap(oldHead, newHead) {
			s.size.Add(-1)
			return oldHead.value, true
		}
		// CAS failed, another goroutine modified head; retry
	}
}

func (s *LockFreeStack) Size() int64 {
	return s.size.Load()
}

func (s *LockFreeStack) IsEmpty() bool {
	return s.head.Load() == nil
}

// =============================================================================
// Mutex-based Stack (for comparison)
// =============================================================================

type MutexStack struct {
	head *Node
	size int64
	mu   sync.Mutex
}

func NewMutexStack() *MutexStack {
	return &MutexStack{}
}

func (s *MutexStack) Push(value int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.head = &Node{value: value, next: s.head}
	s.size++
}

func (s *MutexStack) Pop() (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.head == nil {
		return 0, false
	}
	value := s.head.value
	s.head = s.head.next
	s.size--
	return value, true
}

// =============================================================================
// Demo
// =============================================================================

func main() {
	fmt.Println("=== Lock-Free Stack Demo ===\n")

	// Single-threaded test
	fmt.Println("Single-threaded test:")
	stack := NewLockFreeStack()

	for i := 1; i <= 5; i++ {
		stack.Push(i)
		fmt.Printf("  Pushed: %d (size: %d)\n", i, stack.Size())
	}

	fmt.Println()
	for !stack.IsEmpty() {
		if val, ok := stack.Pop(); ok {
			fmt.Printf("  Popped: %d (size: %d)\n", val, stack.Size())
		}
	}

	// Multi-threaded test
	fmt.Println("\n--- Multi-threaded test ---")

	const numGoroutines = 100
	const opsPerGoroutine = 1000

	stack = NewLockFreeStack()
	var wg sync.WaitGroup

	// Push phase
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				stack.Push(id*opsPerGoroutine + i)
			}
		}(g)
	}
	wg.Wait()

	fmt.Printf("After %d goroutines pushed %d items each:\n",
		numGoroutines, opsPerGoroutine)
	fmt.Printf("  Stack size: %d (expected: %d)\n",
		stack.Size(), numGoroutines*opsPerGoroutine)

	// Pop phase
	var popCount atomic.Int64
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				if _, ok := stack.Pop(); ok {
					popCount.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	fmt.Printf("After %d goroutines popped:\n", numGoroutines)
	fmt.Printf("  Items popped: %d\n", popCount.Load())
	fmt.Printf("  Stack size: %d\n", stack.Size())

	// Verify correctness
	if stack.Size() == 0 && popCount.Load() == numGoroutines*opsPerGoroutine {
		fmt.Println("\n✓ All items accounted for!")
	} else {
		fmt.Println("\n✗ Mismatch detected!")
	}
}
