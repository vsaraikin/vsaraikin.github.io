/**
 * Atomic vs Mutex Benchmark
 *
 * Demonstrates the performance difference between atomic operations
 * and mutex-protected increments.
 *
 * Compile: g++ -std=c++17 -O2 -pthread atomic_vs_mutex_bench.cpp -o atomic_bench
 * Run: ./atomic_bench
 */

#include <atomic>
#include <chrono>
#include <iomanip>
#include <iostream>
#include <mutex>
#include <thread>
#include <vector>

// =============================================================================
// Configuration
// =============================================================================

constexpr int ITERATIONS = 100'000'000;
constexpr int THREADS = 4;

// =============================================================================
// Atomic Benchmark
// =============================================================================

std::atomic<int64_t> atomic_counter{0};

void bench_atomic() {
    atomic_counter = 0;
    auto start = std::chrono::high_resolution_clock::now();

    std::vector<std::thread> threads;
    for (int t = 0; t < THREADS; ++t) {
        threads.emplace_back([]() {
            for (int i = 0; i < ITERATIONS / THREADS; ++i) {
                // Relaxed ordering: just need atomicity, not ordering
                atomic_counter.fetch_add(1, std::memory_order_relaxed);
            }
        });
    }

    for (auto& t : threads) {
        t.join();
    }

    auto elapsed = std::chrono::high_resolution_clock::now() - start;
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(elapsed).count();
    auto ns_per_op = std::chrono::duration_cast<std::chrono::nanoseconds>(elapsed).count() / ITERATIONS;

    std::cout << "Atomic (relaxed):     " << std::setw(6) << ms << " ms  ("
              << ns_per_op << " ns/op)\n";
}

void bench_atomic_seq_cst() {
    atomic_counter = 0;
    auto start = std::chrono::high_resolution_clock::now();

    std::vector<std::thread> threads;
    for (int t = 0; t < THREADS; ++t) {
        threads.emplace_back([]() {
            for (int i = 0; i < ITERATIONS / THREADS; ++i) {
                // Sequential consistency: strongest ordering
                atomic_counter.fetch_add(1, std::memory_order_seq_cst);
            }
        });
    }

    for (auto& t : threads) {
        t.join();
    }

    auto elapsed = std::chrono::high_resolution_clock::now() - start;
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(elapsed).count();
    auto ns_per_op = std::chrono::duration_cast<std::chrono::nanoseconds>(elapsed).count() / ITERATIONS;

    std::cout << "Atomic (seq_cst):     " << std::setw(6) << ms << " ms  ("
              << ns_per_op << " ns/op)\n";
}

// =============================================================================
// Mutex Benchmark
// =============================================================================

int64_t mutex_counter = 0;
std::mutex mtx;

void bench_mutex() {
    mutex_counter = 0;
    auto start = std::chrono::high_resolution_clock::now();

    std::vector<std::thread> threads;
    for (int t = 0; t < THREADS; ++t) {
        threads.emplace_back([]() {
            for (int i = 0; i < ITERATIONS / THREADS; ++i) {
                std::lock_guard<std::mutex> lock(mtx);
                mutex_counter++;
            }
        });
    }

    for (auto& t : threads) {
        t.join();
    }

    auto elapsed = std::chrono::high_resolution_clock::now() - start;
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(elapsed).count();
    auto ns_per_op = std::chrono::duration_cast<std::chrono::nanoseconds>(elapsed).count() / ITERATIONS;

    std::cout << "Mutex:                " << std::setw(6) << ms << " ms  ("
              << ns_per_op << " ns/op)\n";
}

// =============================================================================
// Spinlock Benchmark (for comparison)
// =============================================================================

class Spinlock {
    std::atomic_flag flag = ATOMIC_FLAG_INIT;
public:
    void lock() {
        while (flag.test_and_set(std::memory_order_acquire)) {
            // Spin
        }
    }
    void unlock() {
        flag.clear(std::memory_order_release);
    }
};

int64_t spinlock_counter = 0;
Spinlock spinlock;

void bench_spinlock() {
    spinlock_counter = 0;
    auto start = std::chrono::high_resolution_clock::now();

    std::vector<std::thread> threads;
    for (int t = 0; t < THREADS; ++t) {
        threads.emplace_back([]() {
            for (int i = 0; i < ITERATIONS / THREADS; ++i) {
                spinlock.lock();
                spinlock_counter++;
                spinlock.unlock();
            }
        });
    }

    for (auto& t : threads) {
        t.join();
    }

    auto elapsed = std::chrono::high_resolution_clock::now() - start;
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(elapsed).count();
    auto ns_per_op = std::chrono::duration_cast<std::chrono::nanoseconds>(elapsed).count() / ITERATIONS;

    std::cout << "Spinlock:             " << std::setw(6) << ms << " ms  ("
              << ns_per_op << " ns/op)\n";
}

// =============================================================================
// CAS-based increment (manual implementation)
// =============================================================================

std::atomic<int64_t> cas_counter{0};

void bench_cas() {
    cas_counter = 0;
    auto start = std::chrono::high_resolution_clock::now();

    std::vector<std::thread> threads;
    for (int t = 0; t < THREADS; ++t) {
        threads.emplace_back([]() {
            for (int i = 0; i < ITERATIONS / THREADS; ++i) {
                int64_t expected = cas_counter.load(std::memory_order_relaxed);
                while (!cas_counter.compare_exchange_weak(
                    expected, expected + 1,
                    std::memory_order_relaxed,
                    std::memory_order_relaxed)) {
                    // Retry on failure
                }
            }
        });
    }

    for (auto& t : threads) {
        t.join();
    }

    auto elapsed = std::chrono::high_resolution_clock::now() - start;
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(elapsed).count();
    auto ns_per_op = std::chrono::duration_cast<std::chrono::nanoseconds>(elapsed).count() / ITERATIONS;

    std::cout << "CAS loop:             " << std::setw(6) << ms << " ms  ("
              << ns_per_op << " ns/op)\n";
}

// =============================================================================
// Main
// =============================================================================

int main() {
    std::cout << "=== Atomic vs Mutex Benchmark ===\n";
    std::cout << "Threads: " << THREADS << "\n";
    std::cout << "Iterations: " << ITERATIONS << " total\n\n";

    bench_atomic();
    bench_atomic_seq_cst();
    bench_cas();
    bench_spinlock();
    bench_mutex();

    std::cout << "\n=== Results Verification ===\n";
    std::cout << "Atomic counter:   " << atomic_counter << "\n";
    std::cout << "CAS counter:      " << cas_counter << "\n";
    std::cout << "Spinlock counter: " << spinlock_counter << "\n";
    std::cout << "Mutex counter:    " << mutex_counter << "\n";

    return 0;
}
