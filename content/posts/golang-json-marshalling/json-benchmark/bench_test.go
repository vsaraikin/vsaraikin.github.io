package jsonbench

import (
	"encoding/json"
	"testing"

	"github.com/bytedance/sonic"
	gojson "github.com/goccy/go-json"
	jsoniter "github.com/json-iterator/go"
	"github.com/minio/simdjson-go"
	segmentio "github.com/segmentio/encoding/json"
	"github.com/valyala/fastjson"
)

// Test data generators
var (
	simpleData  = generateSimpleData(100)
	nestedData  = generateNestedData(100)
	complexData = generateComplexData(50)

	simpleJSON, _  = json.Marshal(simpleData)
	nestedJSON, _  = json.Marshal(nestedData)
	complexJSON, _ = json.Marshal(complexData)
)

// ============================================================================
// Simple Struct Benchmarks - Marshal
// ============================================================================

func BenchmarkSimple_Marshal_Stdlib(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(simpleData)
	}
}

func BenchmarkSimple_Marshal_Sonic(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = sonic.Marshal(simpleData)
	}
}

func BenchmarkSimple_Marshal_GoJson(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = gojson.Marshal(simpleData)
	}
}

func BenchmarkSimple_Marshal_Jsoniter(b *testing.B) {
	ji := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ji.Marshal(simpleData)
	}
}

func BenchmarkSimple_Marshal_Segmentio(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = segmentio.Marshal(simpleData)
	}
}

func BenchmarkSimple_Marshal_EasyJson(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = simpleData[0].MarshalJSON()
	}
}

// ============================================================================
// Simple Struct Benchmarks - Unmarshal
// ============================================================================

func BenchmarkSimple_Unmarshal_Stdlib(b *testing.B) {
	var result []SimpleStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(simpleJSON, &result)
	}
}

func BenchmarkSimple_Unmarshal_Sonic(b *testing.B) {
	var result []SimpleStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sonic.Unmarshal(simpleJSON, &result)
	}
}

func BenchmarkSimple_Unmarshal_GoJson(b *testing.B) {
	var result []SimpleStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gojson.Unmarshal(simpleJSON, &result)
	}
}

func BenchmarkSimple_Unmarshal_Jsoniter(b *testing.B) {
	ji := jsoniter.ConfigCompatibleWithStandardLibrary
	var result []SimpleStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ji.Unmarshal(simpleJSON, &result)
	}
}

func BenchmarkSimple_Unmarshal_Segmentio(b *testing.B) {
	var result []SimpleStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = segmentio.Unmarshal(simpleJSON, &result)
	}
}

func BenchmarkSimple_Unmarshal_EasyJson(b *testing.B) {
	var result SimpleStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.UnmarshalJSON(simpleJSON)
	}
}

// ============================================================================
// Nested Struct Benchmarks - Marshal
// ============================================================================

func BenchmarkNested_Marshal_Stdlib(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(nestedData)
	}
}

func BenchmarkNested_Marshal_Sonic(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = sonic.Marshal(nestedData)
	}
}

func BenchmarkNested_Marshal_GoJson(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = gojson.Marshal(nestedData)
	}
}

func BenchmarkNested_Marshal_Jsoniter(b *testing.B) {
	ji := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ji.Marshal(nestedData)
	}
}

func BenchmarkNested_Marshal_Segmentio(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = segmentio.Marshal(nestedData)
	}
}

func BenchmarkNested_Marshal_EasyJson(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = nestedData[0].MarshalJSON()
	}
}

// ============================================================================
// Nested Struct Benchmarks - Unmarshal
// ============================================================================

func BenchmarkNested_Unmarshal_Stdlib(b *testing.B) {
	var result []NestedStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(nestedJSON, &result)
	}
}

func BenchmarkNested_Unmarshal_Sonic(b *testing.B) {
	var result []NestedStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sonic.Unmarshal(nestedJSON, &result)
	}
}

func BenchmarkNested_Unmarshal_GoJson(b *testing.B) {
	var result []NestedStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gojson.Unmarshal(nestedJSON, &result)
	}
}

func BenchmarkNested_Unmarshal_Jsoniter(b *testing.B) {
	ji := jsoniter.ConfigCompatibleWithStandardLibrary
	var result []NestedStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ji.Unmarshal(nestedJSON, &result)
	}
}

func BenchmarkNested_Unmarshal_Segmentio(b *testing.B) {
	var result []NestedStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = segmentio.Unmarshal(nestedJSON, &result)
	}
}

func BenchmarkNested_Unmarshal_EasyJson(b *testing.B) {
	var result NestedStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.UnmarshalJSON(nestedJSON)
	}
}

// ============================================================================
// Complex Struct Benchmarks - Marshal
// ============================================================================

func BenchmarkComplex_Marshal_Stdlib(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(complexData)
	}
}

func BenchmarkComplex_Marshal_Sonic(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = sonic.Marshal(complexData)
	}
}

func BenchmarkComplex_Marshal_GoJson(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = gojson.Marshal(complexData)
	}
}

func BenchmarkComplex_Marshal_Jsoniter(b *testing.B) {
	ji := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ji.Marshal(complexData)
	}
}

func BenchmarkComplex_Marshal_Segmentio(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = segmentio.Marshal(complexData)
	}
}

func BenchmarkComplex_Marshal_EasyJson(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = complexData[0].MarshalJSON()
	}
}

// ============================================================================
// Complex Struct Benchmarks - Unmarshal
// ============================================================================

func BenchmarkComplex_Unmarshal_Stdlib(b *testing.B) {
	var result []ComplexStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(complexJSON, &result)
	}
}

func BenchmarkComplex_Unmarshal_Sonic(b *testing.B) {
	var result []ComplexStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sonic.Unmarshal(complexJSON, &result)
	}
}

func BenchmarkComplex_Unmarshal_GoJson(b *testing.B) {
	var result []ComplexStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gojson.Unmarshal(complexJSON, &result)
	}
}

func BenchmarkComplex_Unmarshal_Jsoniter(b *testing.B) {
	ji := jsoniter.ConfigCompatibleWithStandardLibrary
	var result []ComplexStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ji.Unmarshal(complexJSON, &result)
	}
}

func BenchmarkComplex_Unmarshal_Segmentio(b *testing.B) {
	var result []ComplexStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = segmentio.Unmarshal(complexJSON, &result)
	}
}

func BenchmarkComplex_Unmarshal_EasyJson(b *testing.B) {
	var result ComplexStruct
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.UnmarshalJSON(complexJSON)
	}
}

// ============================================================================
// Parse-only benchmarks
// ============================================================================

func BenchmarkSimple_Parse_FastJson(b *testing.B) {
	var p fastjson.Parser
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.ParseBytes(simpleJSON)
	}
}

var simdjsonSink *simdjson.ParsedJson

func BenchmarkSimple_Parse_SimdJson(b *testing.B) {
	var pj *simdjson.ParsedJson
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pj, _ = simdjson.Parse(simpleJSON, nil)
	}
	simdjsonSink = pj
}

func BenchmarkNested_Parse_FastJson(b *testing.B) {
	var p fastjson.Parser
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.ParseBytes(nestedJSON)
	}
}

func BenchmarkNested_Parse_SimdJson(b *testing.B) {
	var pj *simdjson.ParsedJson
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pj, _ = simdjson.Parse(nestedJSON, nil)
	}
	simdjsonSink = pj
}

func BenchmarkComplex_Parse_FastJson(b *testing.B) {
	var p fastjson.Parser
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.ParseBytes(complexJSON)
	}
}

func BenchmarkComplex_Parse_SimdJson(b *testing.B) {
	var pj *simdjson.ParsedJson
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pj, _ = simdjson.Parse(complexJSON, nil)
	}
	simdjsonSink = pj
}
