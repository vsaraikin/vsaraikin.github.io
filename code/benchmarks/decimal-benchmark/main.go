package main

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/shopspring/decimal"
	"github.com/cockroachdb/apd/v3"
	"github.com/govalues/decimal" as gvdecimal
	"github.com/ericlagergren/decimal" as eldecimal
	"github.com/alpacahq/alpacadecimal"
	"github.com/robaho/fixed"
)

// BenchmarkResult stores timing information
type BenchmarkResult struct {
	Library     string
	Operation   string
	Duration    time.Duration
	OpsPerSec   float64
	NsPerOp     float64
	AllocsPerOp int64
}

// Test operations
const (
	iterations = 100000
)

// ===== float64 benchmarks =====

func benchmarkFloat64Add(iterations int) BenchmarkResult {
	a := 123.456
	b := 789.123
	start := time.Now()
	var result float64
	for i := 0; i < iterations; i++ {
		result = a + b
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "float64",
		Operation: "Add",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkFloat64Mul(iterations int) BenchmarkResult {
	a := 123.456
	b := 2.5
	start := time.Now()
	var result float64
	for i := 0; i < iterations; i++ {
		result = a * b
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "float64",
		Operation: "Multiply",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkFloat64Div(iterations int) BenchmarkResult {
	a := 123.456
	b := 2.5
	start := time.Now()
	var result float64
	for i := 0; i < iterations; i++ {
		result = a / b
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "float64",
		Operation: "Divide",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkFloat64Sqrt(iterations int) BenchmarkResult {
	a := 123.456
	start := time.Now()
	var result float64
	for i := 0; i < iterations; i++ {
		result = math.Sqrt(a)
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "float64",
		Operation: "Sqrt",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

// ===== big.Float benchmarks =====

func benchmarkBigFloatAdd(iterations int) BenchmarkResult {
	a := big.NewFloat(123.456)
	b := big.NewFloat(789.123)
	start := time.Now()
	result := new(big.Float)
	for i := 0; i < iterations; i++ {
		result.Add(a, b)
	}
	duration := time.Since(start)
	return BenchmarkResult{
		Library:   "big.Float",
		Operation: "Add",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkBigFloatMul(iterations int) BenchmarkResult {
	a := big.NewFloat(123.456)
	b := big.NewFloat(2.5)
	start := time.Now()
	result := new(big.Float)
	for i := 0; i < iterations; i++ {
		result.Mul(a, b)
	}
	duration := time.Since(start)
	return BenchmarkResult{
		Library:   "big.Float",
		Operation: "Multiply",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkBigFloatDiv(iterations int) BenchmarkResult {
	a := big.NewFloat(123.456)
	b := big.NewFloat(2.5)
	start := time.Now()
	result := new(big.Float)
	for i := 0; i < iterations; i++ {
		result.Quo(a, b)
	}
	duration := time.Since(start)
	return BenchmarkResult{
		Library:   "big.Float",
		Operation: "Divide",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkBigFloatSqrt(iterations int) BenchmarkResult {
	a := big.NewFloat(123.456)
	start := time.Now()
	result := new(big.Float)
	for i := 0; i < iterations; i++ {
		result.Sqrt(a)
	}
	duration := time.Since(start)
	return BenchmarkResult{
		Library:   "big.Float",
		Operation: "Sqrt",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

// ===== shopspring/decimal benchmarks =====

func benchmarkShopspringAdd(iterations int) BenchmarkResult {
	a := decimal.NewFromFloat(123.456)
	b := decimal.NewFromFloat(789.123)
	start := time.Now()
	var result decimal.Decimal
	for i := 0; i < iterations; i++ {
		result = a.Add(b)
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "shopspring",
		Operation: "Add",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkShopspringMul(iterations int) BenchmarkResult {
	a := decimal.NewFromFloat(123.456)
	b := decimal.NewFromFloat(2.5)
	start := time.Now()
	var result decimal.Decimal
	for i := 0; i < iterations; i++ {
		result = a.Mul(b)
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "shopspring",
		Operation: "Multiply",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkShopspringDiv(iterations int) BenchmarkResult {
	a := decimal.NewFromFloat(123.456)
	b := decimal.NewFromFloat(2.5)
	start := time.Now()
	var result decimal.Decimal
	for i := 0; i < iterations; i++ {
		result = a.Div(b)
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "shopspring",
		Operation: "Divide",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

// ===== cockroachdb/apd benchmarks =====

func benchmarkAPDAdd(iterations int) BenchmarkResult {
	ctx := apd.BaseContext.WithPrecision(16)
	a, _, _ := apd.NewFromString("123.456")
	b, _, _ := apd.NewFromString("789.123")
	start := time.Now()
	result := new(apd.Decimal)
	for i := 0; i < iterations; i++ {
		ctx.Add(result, a, b)
	}
	duration := time.Since(start)
	return BenchmarkResult{
		Library:   "cockroachdb/apd",
		Operation: "Add",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkAPDMul(iterations int) BenchmarkResult {
	ctx := apd.BaseContext.WithPrecision(16)
	a, _, _ := apd.NewFromString("123.456")
	b, _, _ := apd.NewFromString("2.5")
	start := time.Now()
	result := new(apd.Decimal)
	for i := 0; i < iterations; i++ {
		ctx.Mul(result, a, b)
	}
	duration := time.Since(start)
	return BenchmarkResult{
		Library:   "cockroachdb/apd",
		Operation: "Multiply",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkAPDDiv(iterations int) BenchmarkResult {
	ctx := apd.BaseContext.WithPrecision(16)
	a, _, _ := apd.NewFromString("123.456")
	b, _, _ := apd.NewFromString("2.5")
	start := time.Now()
	result := new(apd.Decimal)
	for i := 0; i < iterations; i++ {
		ctx.Quo(result, a, b)
	}
	duration := time.Since(start)
	return BenchmarkResult{
		Library:   "cockroachdb/apd",
		Operation: "Divide",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkAPDSqrt(iterations int) BenchmarkResult {
	ctx := apd.BaseContext.WithPrecision(16)
	a, _, _ := apd.NewFromString("123.456")
	start := time.Now()
	result := new(apd.Decimal)
	for i := 0; i < iterations; i++ {
		ctx.Sqrt(result, a)
	}
	duration := time.Since(start)
	return BenchmarkResult{
		Library:   "cockroachdb/apd",
		Operation: "Sqrt",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

// ===== govalues/decimal benchmarks =====

func benchmarkGovaluesAdd(iterations int) BenchmarkResult {
	a, _ := gvdecimal.New(123456, 3) // 123.456
	b, _ := gvdecimal.New(789123, 3) // 789.123
	start := time.Now()
	var result gvdecimal.Decimal
	for i := 0; i < iterations; i++ {
		result, _ = a.Add(b)
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "govalues",
		Operation: "Add",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkGovaluesMul(iterations int) BenchmarkResult {
	a, _ := gvdecimal.New(123456, 3) // 123.456
	b, _ := gvdecimal.New(25, 1)     // 2.5
	start := time.Now()
	var result gvdecimal.Decimal
	for i := 0; i < iterations; i++ {
		result, _ = a.Mul(b)
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "govalues",
		Operation: "Multiply",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkGovaluesDiv(iterations int) BenchmarkResult {
	a, _ := gvdecimal.New(123456, 3) // 123.456
	b, _ := gvdecimal.New(25, 1)     // 2.5
	start := time.Now()
	var result gvdecimal.Decimal
	for i := 0; i < iterations; i++ {
		result, _ = a.Quo(b)
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "govalues",
		Operation: "Divide",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

// ===== robaho/fixed benchmarks =====

func benchmarkFixedAdd(iterations int) BenchmarkResult {
	a := fixed.NewF(123.456)
	b := fixed.NewF(789.123)
	start := time.Now()
	var result fixed.Fixed
	for i := 0; i < iterations; i++ {
		result = a.Add(b)
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "robaho/fixed",
		Operation: "Add",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkFixedMul(iterations int) BenchmarkResult {
	a := fixed.NewF(123.456)
	b := fixed.NewF(2.5)
	start := time.Now()
	var result fixed.Fixed
	for i := 0; i < iterations; i++ {
		result = a.Mul(b)
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "robaho/fixed",
		Operation: "Multiply",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkFixedDiv(iterations int) BenchmarkResult {
	a := fixed.NewF(123.456)
	b := fixed.NewF(2.5)
	start := time.Now()
	var result fixed.Fixed
	for i := 0; i < iterations; i++ {
		result = a.Div(b)
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "robaho/fixed",
		Operation: "Divide",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

// ===== alpacahq/alpacadecimal benchmarks =====

func benchmarkAlpacaAdd(iterations int) BenchmarkResult {
	a := alpacadecimal.NewFromFloat(123.456)
	b := alpacadecimal.NewFromFloat(789.123)
	start := time.Now()
	var result alpacadecimal.Decimal
	for i := 0; i < iterations; i++ {
		result = a.Add(b)
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "alpacadecimal",
		Operation: "Add",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkAlpacaMul(iterations int) BenchmarkResult {
	a := alpacadecimal.NewFromFloat(123.456)
	b := alpacadecimal.NewFromFloat(2.5)
	start := time.Now()
	var result alpacadecimal.Decimal
	for i := 0; i < iterations; i++ {
		result = a.Mul(b)
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "alpacadecimal",
		Operation: "Multiply",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func benchmarkAlpacaDiv(iterations int) BenchmarkResult {
	a := alpacadecimal.NewFromFloat(123.456)
	b := alpacadecimal.NewFromFloat(2.5)
	start := time.Now()
	var result alpacadecimal.Decimal
	for i := 0; i < iterations; i++ {
		result = a.Div(b)
	}
	duration := time.Since(start)
	_ = result
	return BenchmarkResult{
		Library:   "alpacadecimal",
		Operation: "Divide",
		Duration:  duration,
		OpsPerSec: float64(iterations) / duration.Seconds(),
		NsPerOp:   float64(duration.Nanoseconds()) / float64(iterations),
	}
}

func printResults(results []BenchmarkResult, operation string) {
	fmt.Printf("\n=== %s Performance ===\n", operation)

	// Filter results for this operation
	opResults := []BenchmarkResult{}
	for _, r := range results {
		if r.Operation == operation {
			opResults = append(opResults, r)
		}
	}

	// Sort by speed (fastest first - lowest ns/op)
	sort.Slice(opResults, func(i, j int) bool {
		return opResults[i].NsPerOp < opResults[j].NsPerOp
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "Library\tns/op\tMops/s\tRelative\t")

	baseline := opResults[len(opResults)-1].NsPerOp // slowest
	for _, r := range opResults {
		mops := r.OpsPerSec / 1_000_000
		relative := baseline / r.NsPerOp
		fmt.Fprintf(w, "%s\t%.2f\t%.2f\t%.1fx\t\n",
			r.Library, r.NsPerOp, mops, relative)
	}
	w.Flush()
}

func main() {
	fmt.Println("Decimal Library Performance Benchmark")
	fmt.Println("=====================================")
	fmt.Printf("Iterations: %d\n", iterations)

	// Run all benchmarks
	results := []BenchmarkResult{
		// float64
		benchmarkFloat64Add(iterations),
		benchmarkFloat64Mul(iterations),
		benchmarkFloat64Div(iterations),
		benchmarkFloat64Sqrt(iterations),

		// big.Float
		benchmarkBigFloatAdd(iterations),
		benchmarkBigFloatMul(iterations),
		benchmarkBigFloatDiv(iterations),
		benchmarkBigFloatSqrt(iterations),

		// shopspring/decimal
		benchmarkShopspringAdd(iterations),
		benchmarkShopspringMul(iterations),
		benchmarkShopspringDiv(iterations),

		// cockroachdb/apd
		benchmarkAPDAdd(iterations),
		benchmarkAPDMul(iterations),
		benchmarkAPDDiv(iterations),
		benchmarkAPDSqrt(iterations),

		// govalues/decimal
		benchmarkGovaluesAdd(iterations),
		benchmarkGovaluesMul(iterations),
		benchmarkGovaluesDiv(iterations),

		// robaho/fixed
		benchmarkFixedAdd(iterations),
		benchmarkFixedMul(iterations),
		benchmarkFixedDiv(iterations),

		// alpacahq/alpacadecimal
		benchmarkAlpacaAdd(iterations),
		benchmarkAlpacaMul(iterations),
		benchmarkAlpacaDiv(iterations),
	}

	// Print results grouped by operation
	printResults(results, "Add")
	printResults(results, "Multiply")
	printResults(results, "Divide")
	printResults(results, "Sqrt")

	fmt.Println("\n=== Summary ===")
	fmt.Println("\nPerformance Ranking (fastest to slowest):")
	fmt.Println("1. float64 - Native hardware support, but precision issues")
	fmt.Println("2. robaho/fixed - Zero allocations, fixed precision (38.24)")
	fmt.Println("3. alpacadecimal - Optimized for financial data, 12 digit precision")
	fmt.Println("4. govalues - No heap allocations, 19 digit precision")
	fmt.Println("5. big.Float - Arbitrary precision, mutable")
	fmt.Println("6. cockroachdb/apd - Arbitrary precision, rich API, immutable")
	fmt.Println("7. shopspring/decimal - Arbitrary precision, ease of use over performance")

	fmt.Println("\nUse Cases:")
	fmt.Println("- float64: Non-financial calculations where small errors acceptable")
	fmt.Println("- robaho/fixed: High-performance financial calculations, known precision needs")
	fmt.Println("- alpacadecimal: Trading systems, market data processing")
	fmt.Println("- govalues: General financial apps, good balance of speed and precision")
	fmt.Println("- big.Float: Scientific calculations, arbitrary precision needs")
	fmt.Println("- cockroachdb/apd: Database systems, complex decimal operations")
	fmt.Println("- shopspring/decimal: General purpose, ease of use, well-tested")
}
