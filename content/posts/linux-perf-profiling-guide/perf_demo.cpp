/**
 * perf Profiling Demo
 *
 * This program demonstrates various performance patterns that can be
 * analyzed with Linux perf tool:
 *
 * 1. CPU-bound hot function
 * 2. Cache-friendly vs cache-hostile access patterns
 * 3. Branch prediction (sorted vs unsorted data)
 * 4. Memory allocation overhead
 *
 * Compile:
 *   g++ -O2 -g -fno-omit-frame-pointer perf_demo.cpp -o perf_demo
 *
 * Profile:
 *   perf stat ./perf_demo                    # Basic stats
 *   perf stat -e cache-misses ./perf_demo    # Cache analysis
 *   perf record -g ./perf_demo               # CPU profiling
 *   perf report                              # View results
 *
 * Generate flame graph:
 *   perf record -F 99 -g ./perf_demo
 *   perf script > out.perf
 *   ./stackcollapse-perf.pl out.perf | ./flamegraph.pl > flamegraph.svg
 */

#include <algorithm>
#include <chrono>
#include <cmath>
#include <iostream>
#include <numeric>
#include <random>
#include <string>
#include <vector>

// Prevent compiler from optimizing away results
template<typename T>
void do_not_optimize(T&& value) {
    asm volatile("" : : "r,m"(value) : "memory");
}

// =============================================================================
// Demo 1: Hot Function (CPU-bound)
// This function will show up prominently in perf report
// =============================================================================

// Intentionally slow hash function
uint64_t slow_hash(const std::string& str) {
    uint64_t hash = 5381;
    for (char c : str) {
        // Lots of CPU work per character
        hash = ((hash << 5) + hash) ^ c;
        hash = hash * 31 + c;
        hash ^= (hash >> 17);
        hash *= 0x85ebca6b;
        hash ^= (hash >> 13);
        hash *= 0xc2b2ae35;
        hash ^= (hash >> 16);
    }
    return hash;
}

void demo_hot_function() {
    std::cout << "Demo 1: Hot Function (CPU-bound)\n";
    std::cout << "================================\n";

    std::vector<std::string> data;
    for (int i = 0; i < 100000; ++i) {
        data.push_back("string_number_" + std::to_string(i) + "_with_extra_data");
    }

    auto start = std::chrono::high_resolution_clock::now();

    uint64_t total = 0;
    // This loop will be the "hot spot"
    for (int iter = 0; iter < 10; ++iter) {
        for (const auto& s : data) {
            total += slow_hash(s);  // <- This will dominate CPU time
        }
    }

    auto elapsed = std::chrono::high_resolution_clock::now() - start;
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(elapsed).count();

    std::cout << "Result: " << total << "\n";
    std::cout << "Time: " << ms << " ms\n\n";
    do_not_optimize(total);
}

// =============================================================================
// Demo 2: Cache Access Patterns
// Compare sequential vs random access - perf stat -e cache-misses
// =============================================================================

void demo_cache_access() {
    std::cout << "Demo 2: Cache Access Patterns\n";
    std::cout << "=============================\n";

    constexpr size_t N = 10'000'000;
    std::vector<int64_t> data(N);
    std::vector<size_t> indices(N);

    // Initialize
    std::iota(data.begin(), data.end(), 0);
    std::iota(indices.begin(), indices.end(), 0);

    // Shuffle for random access
    std::mt19937 rng(42);
    std::shuffle(indices.begin(), indices.end(), rng);

    int64_t sum = 0;

    // Sequential access (cache-friendly)
    auto start = std::chrono::high_resolution_clock::now();
    for (size_t i = 0; i < N; ++i) {
        sum += data[i];  // Prefetcher works perfectly
    }
    auto seq_time = std::chrono::high_resolution_clock::now() - start;
    do_not_optimize(sum);

    // Random access (cache-hostile)
    sum = 0;
    start = std::chrono::high_resolution_clock::now();
    for (size_t i = 0; i < N; ++i) {
        sum += data[indices[i]];  // Cache misses on almost every access
    }
    auto rand_time = std::chrono::high_resolution_clock::now() - start;
    do_not_optimize(sum);

    auto seq_ms = std::chrono::duration_cast<std::chrono::milliseconds>(seq_time).count();
    auto rand_ms = std::chrono::duration_cast<std::chrono::milliseconds>(rand_time).count();

    std::cout << "Sequential access: " << seq_ms << " ms\n";
    std::cout << "Random access:     " << rand_ms << " ms\n";
    std::cout << "Slowdown:          " << (double)rand_ms / seq_ms << "x\n\n";
}

// =============================================================================
// Demo 3: Branch Prediction
// Compare sorted vs unsorted data - perf stat -e branch-misses
// =============================================================================

void demo_branch_prediction() {
    std::cout << "Demo 3: Branch Prediction\n";
    std::cout << "=========================\n";

    constexpr size_t N = 10'000'000;
    std::vector<int> data(N);

    std::mt19937 rng(42);
    std::uniform_int_distribution<int> dist(0, 255);
    for (auto& x : data) {
        x = dist(rng);
    }

    // Make a sorted copy
    std::vector<int> sorted_data = data;
    std::sort(sorted_data.begin(), sorted_data.end());

    int64_t sum = 0;

    // Unsorted data: unpredictable branches
    auto start = std::chrono::high_resolution_clock::now();
    for (int iter = 0; iter < 10; ++iter) {
        for (int x : data) {
            if (x >= 128) {  // Branch is ~50% taken, random pattern
                sum += x;
            }
        }
    }
    auto unsorted_time = std::chrono::high_resolution_clock::now() - start;
    do_not_optimize(sum);

    // Sorted data: predictable branches
    sum = 0;
    start = std::chrono::high_resolution_clock::now();
    for (int iter = 0; iter < 10; ++iter) {
        for (int x : sorted_data) {
            if (x >= 128) {  // First half: never taken. Second half: always taken.
                sum += x;
            }
        }
    }
    auto sorted_time = std::chrono::high_resolution_clock::now() - start;
    do_not_optimize(sum);

    auto unsorted_ms = std::chrono::duration_cast<std::chrono::milliseconds>(unsorted_time).count();
    auto sorted_ms = std::chrono::duration_cast<std::chrono::milliseconds>(sorted_time).count();

    std::cout << "Unsorted (unpredictable): " << unsorted_ms << " ms\n";
    std::cout << "Sorted (predictable):     " << sorted_ms << " ms\n";
    std::cout << "Speedup:                  " << (double)unsorted_ms / sorted_ms << "x\n\n";
}

// =============================================================================
// Demo 4: Memory Allocation Overhead
// Shows malloc/free overhead in perf report
// =============================================================================

struct SmallObject {
    int data[4];
    SmallObject() { std::fill(std::begin(data), std::end(data), 0); }
};

void demo_allocation_overhead() {
    std::cout << "Demo 4: Memory Allocation Overhead\n";
    std::cout << "===================================\n";

    constexpr size_t N = 1'000'000;

    // Many small allocations (malloc/free heavy)
    auto start = std::chrono::high_resolution_clock::now();
    for (size_t iter = 0; iter < 10; ++iter) {
        std::vector<SmallObject*> objects;
        objects.reserve(N);

        for (size_t i = 0; i < N; ++i) {
            objects.push_back(new SmallObject());  // malloc called
        }
        for (auto* obj : objects) {
            delete obj;  // free called
        }
    }
    auto many_alloc_time = std::chrono::high_resolution_clock::now() - start;

    // Single allocation (efficient)
    start = std::chrono::high_resolution_clock::now();
    for (size_t iter = 0; iter < 10; ++iter) {
        std::vector<SmallObject> objects(N);  // One allocation for all
        do_not_optimize(objects.data());
    }
    auto single_alloc_time = std::chrono::high_resolution_clock::now() - start;

    auto many_ms = std::chrono::duration_cast<std::chrono::milliseconds>(many_alloc_time).count();
    auto single_ms = std::chrono::duration_cast<std::chrono::milliseconds>(single_alloc_time).count();

    std::cout << "Many small allocations: " << many_ms << " ms\n";
    std::cout << "Single allocation:      " << single_ms << " ms\n";
    std::cout << "Speedup:                " << (double)many_ms / single_ms << "x\n\n";
}

// =============================================================================
// Demo 5: Matrix Traversal (Row-major vs Column-major)
// Shows L1 cache miss patterns
// =============================================================================

void demo_matrix_traversal() {
    std::cout << "Demo 5: Matrix Traversal (Cache Lines)\n";
    std::cout << "======================================\n";

    constexpr size_t SIZE = 4000;
    std::vector<std::vector<int64_t>> matrix(SIZE, std::vector<int64_t>(SIZE, 1));

    int64_t sum = 0;

    // Row-major (cache-friendly)
    auto start = std::chrono::high_resolution_clock::now();
    for (size_t i = 0; i < SIZE; ++i) {
        for (size_t j = 0; j < SIZE; ++j) {
            sum += matrix[i][j];
        }
    }
    auto row_time = std::chrono::high_resolution_clock::now() - start;
    do_not_optimize(sum);

    // Column-major (cache-hostile)
    sum = 0;
    start = std::chrono::high_resolution_clock::now();
    for (size_t j = 0; j < SIZE; ++j) {
        for (size_t i = 0; i < SIZE; ++i) {
            sum += matrix[i][j];
        }
    }
    auto col_time = std::chrono::high_resolution_clock::now() - start;
    do_not_optimize(sum);

    auto row_ms = std::chrono::duration_cast<std::chrono::milliseconds>(row_time).count();
    auto col_ms = std::chrono::duration_cast<std::chrono::milliseconds>(col_time).count();

    std::cout << "Row-major:    " << row_ms << " ms\n";
    std::cout << "Column-major: " << col_ms << " ms\n";
    std::cout << "Slowdown:     " << (double)col_ms / row_ms << "x\n\n";
}

// =============================================================================
// Main
// =============================================================================

void print_usage() {
    std::cout << "perf Profiling Demo\n";
    std::cout << "==================\n\n";
    std::cout << "Usage: ./perf_demo [demo_number]\n\n";
    std::cout << "Demos:\n";
    std::cout << "  1  Hot function (CPU-bound)\n";
    std::cout << "  2  Cache access patterns\n";
    std::cout << "  3  Branch prediction\n";
    std::cout << "  4  Memory allocation overhead\n";
    std::cout << "  5  Matrix traversal\n";
    std::cout << "  all  Run all demos (default)\n\n";
    std::cout << "Profiling commands:\n";
    std::cout << "  perf stat ./perf_demo                         # Basic stats\n";
    std::cout << "  perf stat -e cache-misses ./perf_demo 2       # Cache analysis\n";
    std::cout << "  perf stat -e branch-misses ./perf_demo 3      # Branch analysis\n";
    std::cout << "  perf record -g ./perf_demo && perf report     # CPU profile\n";
    std::cout << "\n";
}

int main(int argc, char* argv[]) {
    std::string demo = "all";
    if (argc > 1) {
        demo = argv[1];
    }

    if (demo == "-h" || demo == "--help") {
        print_usage();
        return 0;
    }

    std::cout << "=== perf Profiling Demo ===\n\n";

    if (demo == "1" || demo == "all") {
        demo_hot_function();
    }
    if (demo == "2" || demo == "all") {
        demo_cache_access();
    }
    if (demo == "3" || demo == "all") {
        demo_branch_prediction();
    }
    if (demo == "4" || demo == "all") {
        demo_allocation_overhead();
    }
    if (demo == "5" || demo == "all") {
        demo_matrix_traversal();
    }

    std::cout << "=== Done ===\n";
    return 0;
}
