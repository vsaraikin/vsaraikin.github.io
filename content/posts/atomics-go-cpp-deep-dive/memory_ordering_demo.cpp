/**
 * Memory Ordering Demo
 *
 * Demonstrates different memory orderings in C++ atomics:
 * - relaxed: no ordering guarantees
 * - acquire/release: synchronize between threads
 * - seq_cst: full sequential consistency
 *
 * Compile: g++ -std=c++17 -O2 -pthread memory_ordering_demo.cpp -o memory_demo
 * Run: ./memory_demo
 */

#include <atomic>
#include <chrono>
#include <iostream>
#include <thread>
#include <vector>

// =============================================================================
// Demo 1: Release-Acquire Synchronization
// =============================================================================

std::atomic<bool> ready{false};
int data = 0;

void demo_release_acquire() {
    std::cout << "=== Demo 1: Release-Acquire Synchronization ===\n\n";

    // Reset
    ready = false;
    data = 0;

    std::thread writer([]() {
        data = 42;  // Non-atomic write
        // Release: all writes before this are visible after acquire
        ready.store(true, std::memory_order_release);
        std::cout << "Writer: set data=42, ready=true\n";
    });

    std::thread reader([]() {
        // Acquire: see all writes before the release
        while (!ready.load(std::memory_order_acquire)) {
            // Spin
        }
        std::cout << "Reader: ready=true, data=" << data << "\n";

        if (data == 42) {
            std::cout << "✓ Correctly synchronized!\n";
        } else {
            std::cout << "✗ Data race detected! data=" << data << "\n";
        }
    });

    writer.join();
    reader.join();
    std::cout << "\n";
}

// =============================================================================
// Demo 2: Relaxed Ordering (Counter)
// =============================================================================

void demo_relaxed_counter() {
    std::cout << "=== Demo 2: Relaxed Ordering for Counters ===\n\n";

    std::atomic<int64_t> counter{0};
    constexpr int THREADS = 4;
    constexpr int OPS = 1'000'000;

    auto start = std::chrono::high_resolution_clock::now();

    std::vector<std::thread> threads;
    for (int t = 0; t < THREADS; ++t) {
        threads.emplace_back([&counter]() {
            for (int i = 0; i < OPS; ++i) {
                // Relaxed: just atomicity, no ordering
                counter.fetch_add(1, std::memory_order_relaxed);
            }
        });
    }

    for (auto& t : threads) {
        t.join();
    }

    auto elapsed = std::chrono::high_resolution_clock::now() - start;
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(elapsed).count();

    std::cout << "Threads: " << THREADS << "\n";
    std::cout << "Ops/thread: " << OPS << "\n";
    std::cout << "Final counter: " << counter << " (expected: " << THREADS * OPS << ")\n";
    std::cout << "Time: " << ms << " ms\n";

    if (counter == THREADS * OPS) {
        std::cout << "✓ Correct! Relaxed ordering is fine for counters.\n";
    }
    std::cout << "\n";
}

// =============================================================================
// Demo 3: Sequential Consistency
// =============================================================================

std::atomic<int> x{0}, y{0};
int r1 = 0, r2 = 0;

void demo_seq_cst() {
    std::cout << "=== Demo 3: Sequential Consistency ===\n\n";

    // This is the classic IRIW (Independent Reads of Independent Writes) test
    // With seq_cst, if thread 1 sees x=1, and thread 2 sees y=1,
    // then they must agree on the order of x and y becoming 1.

    constexpr int ITERATIONS = 100000;
    int anomalies = 0;

    for (int iter = 0; iter < ITERATIONS; ++iter) {
        x = 0;
        y = 0;
        r1 = 0;
        r2 = 0;

        std::thread t1([]() {
            x.store(1, std::memory_order_seq_cst);
        });

        std::thread t2([]() {
            y.store(1, std::memory_order_seq_cst);
        });

        std::thread t3([]() {
            while (x.load(std::memory_order_seq_cst) != 1) {}
            r1 = y.load(std::memory_order_seq_cst);
        });

        std::thread t4([]() {
            while (y.load(std::memory_order_seq_cst) != 1) {}
            r2 = x.load(std::memory_order_seq_cst);
        });

        t1.join();
        t2.join();
        t3.join();
        t4.join();

        // With seq_cst, r1=0 and r2=0 should be impossible
        // (it would mean x=1 happened before y=1 AND y=1 happened before x=1)
        if (r1 == 0 && r2 == 0) {
            anomalies++;
        }
    }

    std::cout << "Iterations: " << ITERATIONS << "\n";
    std::cout << "Anomalies (r1=0 && r2=0): " << anomalies << "\n";

    if (anomalies == 0) {
        std::cout << "✓ Sequential consistency maintained!\n";
    } else {
        std::cout << "✗ Anomalies detected (shouldn't happen with seq_cst)!\n";
    }
    std::cout << "\n";
}

// =============================================================================
// Demo 4: Compare-And-Swap (CAS)
// =============================================================================

void demo_cas() {
    std::cout << "=== Demo 4: Compare-And-Swap ===\n\n";

    std::atomic<int> value{0};

    // Simulate multiple threads trying to initialize a value
    std::vector<std::thread> threads;
    std::atomic<int> winners{0};

    for (int t = 0; t < 10; ++t) {
        threads.emplace_back([&value, &winners, t]() {
            int expected = 0;
            // Try to be the first to set value from 0 to (t+1)*10
            if (value.compare_exchange_strong(expected, (t + 1) * 10)) {
                winners++;
                std::cout << "Thread " << t << " won! Set value to " << (t + 1) * 10 << "\n";
            } else {
                std::cout << "Thread " << t << " lost. Value was already " << expected << "\n";
            }
        });
    }

    for (auto& t : threads) {
        t.join();
    }

    std::cout << "\nFinal value: " << value << "\n";
    std::cout << "Winners: " << winners << " (should be 1)\n";

    if (winners == 1) {
        std::cout << "✓ Only one thread succeeded (as expected)!\n";
    }
    std::cout << "\n";
}

// =============================================================================
// Demo 5: Spinlock using atomic_flag
// =============================================================================

class Spinlock {
    std::atomic_flag flag = ATOMIC_FLAG_INIT;

public:
    void lock() {
        while (flag.test_and_set(std::memory_order_acquire)) {
            // Spin - could add pause/yield here
        }
    }

    void unlock() {
        flag.clear(std::memory_order_release);
    }
};

void demo_spinlock() {
    std::cout << "=== Demo 5: Spinlock Implementation ===\n\n";

    Spinlock spinlock;
    int shared_counter = 0;
    constexpr int THREADS = 4;
    constexpr int OPS = 100000;

    auto start = std::chrono::high_resolution_clock::now();

    std::vector<std::thread> threads;
    for (int t = 0; t < THREADS; ++t) {
        threads.emplace_back([&]() {
            for (int i = 0; i < OPS; ++i) {
                spinlock.lock();
                shared_counter++;
                spinlock.unlock();
            }
        });
    }

    for (auto& t : threads) {
        t.join();
    }

    auto elapsed = std::chrono::high_resolution_clock::now() - start;
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(elapsed).count();

    std::cout << "Threads: " << THREADS << "\n";
    std::cout << "Ops/thread: " << OPS << "\n";
    std::cout << "Final counter: " << shared_counter << " (expected: " << THREADS * OPS << ")\n";
    std::cout << "Time: " << ms << " ms\n";

    if (shared_counter == THREADS * OPS) {
        std::cout << "✓ Spinlock works correctly!\n";
    }
    std::cout << "\n";
}

// =============================================================================
// Main
// =============================================================================

int main() {
    std::cout << "========================================\n";
    std::cout << "   Memory Ordering Demonstration\n";
    std::cout << "========================================\n\n";

    demo_release_acquire();
    demo_relaxed_counter();
    demo_seq_cst();
    demo_cas();
    demo_spinlock();

    std::cout << "=== All demos complete ===\n";
    return 0;
}
