package compression

import (
	"strings"
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/storage"
)

// BenchmarkCompression benchmarks different compression algorithms
func BenchmarkCompressionSnappy(b *testing.B) {
	data := []byte(strings.Repeat("benchmark data for compression testing ", 100))
	compressor, _ := NewCompressor(SnappyConfig())
	defer compressor.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.Compress(data)
	}
}

func BenchmarkCompressionZstd(b *testing.B) {
	data := []byte(strings.Repeat("benchmark data for compression testing ", 100))
	compressor, _ := NewCompressor(ZstdConfig(3))
	defer compressor.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.Compress(data)
	}
}

func BenchmarkCompressionGzip(b *testing.B) {
	data := []byte(strings.Repeat("benchmark data for compression testing ", 100))
	compressor, _ := NewCompressor(GzipConfig(6))
	defer compressor.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.Compress(data)
	}
}

// BenchmarkDecompression benchmarks different decompression algorithms
func BenchmarkDecompressionSnappy(b *testing.B) {
	data := []byte(strings.Repeat("benchmark data for decompression testing ", 100))
	compressor, _ := NewCompressor(SnappyConfig())
	defer compressor.Close()
	compressed, _ := compressor.Compress(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.Decompress(compressed)
	}
}

func BenchmarkDecompressionZstd(b *testing.B) {
	data := []byte(strings.Repeat("benchmark data for decompression testing ", 100))
	compressor, _ := NewCompressor(ZstdConfig(3))
	defer compressor.Close()
	compressed, _ := compressor.Compress(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.Decompress(compressed)
	}
}

func BenchmarkDecompressionGzip(b *testing.B) {
	data := []byte(strings.Repeat("benchmark data for decompression testing ", 100))
	compressor, _ := NewCompressor(GzipConfig(6))
	defer compressor.Close()
	compressed, _ := compressor.Compress(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.Decompress(compressed)
	}
}

// BenchmarkDocumentCompression benchmarks document compression
func BenchmarkDocumentCompression(b *testing.B) {
	compDoc, _ := NewCompressedDocument(ZstdConfig(3))
	defer compDoc.Close()

	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())
	doc.Set("name", "Performance Test")
	doc.Set("email", "perf@example.com")
	doc.Set("age", int64(30))
	doc.Set("active", true)
	doc.Set("tags", []interface{}{"golang", "database", "nosql"})
	doc.Set("data", map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
		"field3": "value3",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compDoc.Encode(doc)
	}
}

func BenchmarkDocumentDecompression(b *testing.B) {
	compDoc, _ := NewCompressedDocument(ZstdConfig(3))
	defer compDoc.Close()

	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())
	doc.Set("name", "Performance Test")
	doc.Set("email", "perf@example.com")
	doc.Set("age", int64(30))
	doc.Set("active", true)
	doc.Set("tags", []interface{}{"golang", "database", "nosql"})

	compressed, _ := compDoc.Encode(doc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compDoc.Decode(compressed)
	}
}

// BenchmarkPageCompression benchmarks page compression
func BenchmarkPageCompression(b *testing.B) {
	compPage, _ := NewCompressedPage(ZstdConfig(3))
	defer compPage.Close()

	page := storage.NewPage(1, storage.PageTypeData)
	// Fill with realistic data
	pattern := "This is realistic page data with some repetition. "
	for i := 0; i+len(pattern) < len(page.Data); i += len(pattern) {
		copy(page.Data[i:], pattern)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compPage.CompressPage(page)
	}
}

func BenchmarkPageDecompression(b *testing.B) {
	compPage, _ := NewCompressedPage(ZstdConfig(3))
	defer compPage.Close()

	page := storage.NewPage(1, storage.PageTypeData)
	pattern := "This is realistic page data with some repetition. "
	for i := 0; i+len(pattern) < len(page.Data); i += len(pattern) {
		copy(page.Data[i:], pattern)
	}

	compressed, _ := compPage.CompressPage(page)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compPage.DecompressPage(compressed)
	}
}

// BenchmarkCompressionLevels benchmarks different compression levels
func BenchmarkZstdLevel1(b *testing.B) {
	data := []byte(strings.Repeat("compression level benchmark ", 200))
	compressor, _ := NewCompressor(ZstdConfig(1))
	defer compressor.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.Compress(data)
	}
}

func BenchmarkZstdLevel3(b *testing.B) {
	data := []byte(strings.Repeat("compression level benchmark ", 200))
	compressor, _ := NewCompressor(ZstdConfig(3))
	defer compressor.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.Compress(data)
	}
}

func BenchmarkZstdLevel9(b *testing.B) {
	data := []byte(strings.Repeat("compression level benchmark ", 200))
	compressor, _ := NewCompressor(ZstdConfig(9))
	defer compressor.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.Compress(data)
	}
}

// BenchmarkLargeDocument benchmarks compression of large documents
func BenchmarkLargeDocumentCompression(b *testing.B) {
	compDoc, _ := NewCompressedDocument(ZstdConfig(3))
	defer compDoc.Close()

	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())

	// Create a large document
	for i := 0; i < 100; i++ {
		fieldName := "field_" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		doc.Set(fieldName, "value with some repeating data for compression")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compDoc.Encode(doc)
	}
}

func BenchmarkLargeDocumentDecompression(b *testing.B) {
	compDoc, _ := NewCompressedDocument(ZstdConfig(3))
	defer compDoc.Close()

	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())

	for i := 0; i < 100; i++ {
		fieldName := "field_" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		doc.Set(fieldName, "value with some repeating data for compression")
	}

	compressed, _ := compDoc.Encode(doc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compDoc.Decode(compressed)
	}
}

// BenchmarkCompareAlgorithms benchmarks all algorithms for comparison
func BenchmarkCompareAlgorithmsCompress(b *testing.B) {
	data := []byte(strings.Repeat("algorithm comparison benchmark data ", 100))

	benchmarks := []struct {
		name   string
		config *Config
	}{
		{"Snappy", SnappyConfig()},
		{"Zstd-1", ZstdConfig(1)},
		{"Zstd-3", ZstdConfig(3)},
		{"Zstd-9", ZstdConfig(9)},
		{"Gzip-1", GzipConfig(1)},
		{"Gzip-6", GzipConfig(6)},
		{"Gzip-9", GzipConfig(9)},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			compressor, _ := NewCompressor(bm.config)
			defer compressor.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = compressor.Compress(data)
			}
		})
	}
}
