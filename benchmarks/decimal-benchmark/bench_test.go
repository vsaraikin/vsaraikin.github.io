package decimalbench

import (
	"testing"

	"github.com/cockroachdb/apd/v3"
	govalues "github.com/govalues/decimal"
	shopspring "github.com/shopspring/decimal"
)

// Benchmark simple addition: price + tax
func BenchmarkAddition_Float64(b *testing.B) {
	price := 99.99
	tax := 8.50
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = price + tax
	}
}

func BenchmarkAddition_Shopspring(b *testing.B) {
	price := shopspring.NewFromFloat(99.99)
	tax := shopspring.NewFromFloat(8.50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = price.Add(tax)
	}
}

func BenchmarkAddition_Govalues(b *testing.B) {
	price, _ := govalues.Parse("99.99")
	tax, _ := govalues.Parse("8.50")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = price.Add(tax)
	}
}

func BenchmarkAddition_Cockroach(b *testing.B) {
	price, _, _ := apd.NewFromString("99.99")
	tax, _, _ := apd.NewFromString("8.50")
	result := apd.New(0, 0)
	ctx := apd.BaseContext
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ctx.Add(result, price, tax)
	}
}

// Benchmark multiplication: price * quantity
func BenchmarkMultiply_Float64(b *testing.B) {
	price := 19.99
	quantity := 1000.0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = price * quantity
	}
}

func BenchmarkMultiply_Shopspring(b *testing.B) {
	price := shopspring.NewFromFloat(19.99)
	quantity := shopspring.NewFromInt(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = price.Mul(quantity)
	}
}

func BenchmarkMultiply_Govalues(b *testing.B) {
	price, _ := govalues.Parse("19.99")
	quantity, _ := govalues.Parse("1000")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = price.Mul(quantity)
	}
}

func BenchmarkMultiply_Cockroach(b *testing.B) {
	price, _, _ := apd.NewFromString("19.99")
	quantity, _, _ := apd.NewFromString("1000")
	result := apd.New(0, 0)
	ctx := apd.BaseContext
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ctx.Mul(result, price, quantity)
	}
}

// Benchmark division: total / count
func BenchmarkDivide_Float64(b *testing.B) {
	total := 12345.67
	count := 123.0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = total / count
	}
}

func BenchmarkDivide_Shopspring(b *testing.B) {
	total := shopspring.NewFromFloat(12345.67)
	count := shopspring.NewFromInt(123)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = total.Div(count)
	}
}

func BenchmarkDivide_Govalues(b *testing.B) {
	total, _ := govalues.Parse("12345.67")
	count, _ := govalues.Parse("123")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = total.Quo(count)
	}
}

func BenchmarkDivide_Cockroach(b *testing.B) {
	total, _, _ := apd.NewFromString("12345.67")
	count, _, _ := apd.NewFromString("123")
	result := apd.New(0, 0)
	ctx := apd.BaseContext
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ctx.Quo(result, total, count)
	}
}

// Benchmark parsing from string
func BenchmarkParse_Float64(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = 123.45
	}
}

func BenchmarkParse_Shopspring(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = shopspring.NewFromString("123.45")
	}
}

func BenchmarkParse_Govalues(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = govalues.Parse("123.45")
	}
}

func BenchmarkParse_Cockroach(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = apd.NewFromString("123.45")
	}
}

// Benchmark complex calculation: (price * qty) + tax - discount
func BenchmarkComplex_Float64(b *testing.B) {
	price := 29.99
	qty := 15.0
	tax := 0.08
	discount := 5.00
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subtotal := price * qty
		taxAmount := subtotal * tax
		_ = subtotal + taxAmount - discount
	}
}

func BenchmarkComplex_Shopspring(b *testing.B) {
	price := shopspring.NewFromFloat(29.99)
	qty := shopspring.NewFromInt(15)
	tax := shopspring.NewFromFloat(0.08)
	discount := shopspring.NewFromFloat(5.00)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subtotal := price.Mul(qty)
		taxAmount := subtotal.Mul(tax)
		_ = subtotal.Add(taxAmount).Sub(discount)
	}
}

func BenchmarkComplex_Govalues(b *testing.B) {
	price, _ := govalues.Parse("29.99")
	qty, _ := govalues.Parse("15")
	tax, _ := govalues.Parse("0.08")
	discount, _ := govalues.Parse("5.00")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subtotal, _ := price.Mul(qty)
		taxAmount, _ := subtotal.Mul(tax)
		withTax, _ := subtotal.Add(taxAmount)
		_, _ = withTax.Sub(discount)
	}
}

func BenchmarkComplex_Cockroach(b *testing.B) {
	price, _, _ := apd.NewFromString("29.99")
	qty, _, _ := apd.NewFromString("15")
	tax, _, _ := apd.NewFromString("0.08")
	discount, _, _ := apd.NewFromString("5.00")
	ctx := apd.BaseContext
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subtotal := apd.New(0, 0)
		taxAmount := apd.New(0, 0)
		withTax := apd.New(0, 0)
		result := apd.New(0, 0)
		_, _ = ctx.Mul(subtotal, price, qty)
		_, _ = ctx.Mul(taxAmount, subtotal, tax)
		_, _ = ctx.Add(withTax, subtotal, taxAmount)
		_, _ = ctx.Sub(result, withTax, discount)
	}
}
