package compression

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompressorNone(t *testing.T) {
	config := &Config{Algorithm: AlgorithmNone}
	compressor, err := NewCompressor(config)
	if err != nil {
		t.Fatalf("Failed to create compressor: %v", err)
	}
	defer compressor.Close()

	data := []byte("hello world")
	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	if !bytes.Equal(compressed, data) {
		t.Errorf("Expected no compression, got different data")
	}

	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Decompressed data doesn't match original")
	}
}

func TestCompressorSnappy(t *testing.T) {
	config := SnappyConfig()
	compressor, err := NewCompressor(config)
	if err != nil {
		t.Fatalf("Failed to create compressor: %v", err)
	}
	defer compressor.Close()

	// Create compressible data
	data := []byte(strings.Repeat("hello world ", 100))

	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	if len(compressed) >= len(data) {
		t.Logf("Warning: Compressed size (%d) >= original size (%d)", len(compressed), len(data))
	}

	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Decompressed data doesn't match original")
	}
}

func TestCompressorZstd(t *testing.T) {
	config := ZstdConfig(3)
	compressor, err := NewCompressor(config)
	if err != nil {
		t.Fatalf("Failed to create compressor: %v", err)
	}
	defer compressor.Close()

	// Create compressible data
	data := []byte(strings.Repeat("the quick brown fox jumps over the lazy dog ", 100))

	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	t.Logf("Original: %d bytes, Compressed: %d bytes, Ratio: %.2f%%",
		len(data), len(compressed), CompressionRatio(len(data), len(compressed))*100)

	if len(compressed) >= len(data) {
		t.Errorf("Zstd should compress repeating data efficiently")
	}

	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Decompressed data doesn't match original")
	}
}

func TestCompressorGzip(t *testing.T) {
	config := GzipConfig(6)
	compressor, err := NewCompressor(config)
	if err != nil {
		t.Fatalf("Failed to create compressor: %v", err)
	}
	defer compressor.Close()

	data := []byte(strings.Repeat("compression test data ", 100))

	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Decompressed data doesn't match original")
	}
}

func TestCompressorZlib(t *testing.T) {
	config := &Config{Algorithm: AlgorithmZlib, Level: 6}
	compressor, err := NewCompressor(config)
	if err != nil {
		t.Fatalf("Failed to create compressor: %v", err)
	}
	defer compressor.Close()

	data := []byte(strings.Repeat("zlib compression test ", 100))

	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Decompressed data doesn't match original")
	}
}

func TestCompressionRatios(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		dataSize int
	}{
		{"Snappy", SnappyConfig(), 10000},
		{"Zstd-Fast", ZstdConfig(1), 10000},
		{"Zstd-Default", ZstdConfig(3), 10000},
		{"Zstd-High", ZstdConfig(9), 10000},
		{"Gzip-Fast", GzipConfig(1), 10000},
		{"Gzip-Default", GzipConfig(6), 10000},
		{"Gzip-Best", GzipConfig(9), 10000},
	}

	// Create realistic data (JSON-like structure with repetition)
	createData := func(size int) []byte {
		pattern := `{"name":"John Doe","age":30,"email":"john@example.com","active":true}`
		var buf bytes.Buffer
		for buf.Len() < size {
			buf.WriteString(pattern)
		}
		return buf.Bytes()[:size]
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressor, err := NewCompressor(tt.config)
			if err != nil {
				t.Fatalf("Failed to create compressor: %v", err)
			}
			defer compressor.Close()

			data := createData(tt.dataSize)
			compressed, err := compressor.Compress(data)
			if err != nil {
				t.Fatalf("Failed to compress: %v", err)
			}

			ratio := CompressionRatio(len(data), len(compressed))
			savings := SpaceSavings(len(data), len(compressed))

			t.Logf("Algorithm: %s, Original: %d, Compressed: %d, Ratio: %.2f%%, Savings: %.2f%%",
				tt.config.Algorithm, len(data), len(compressed), ratio*100, savings)

			// Verify decompression
			decompressed, err := compressor.Decompress(compressed)
			if err != nil {
				t.Fatalf("Failed to decompress: %v", err)
			}

			if !bytes.Equal(decompressed, data) {
				t.Errorf("Decompressed data doesn't match original")
			}
		})
	}
}

func TestEmptyData(t *testing.T) {
	compressor, err := NewCompressor(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create compressor: %v", err)
	}
	defer compressor.Close()

	data := []byte{}
	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("Failed to compress empty data: %v", err)
	}

	if len(compressed) != 0 {
		t.Errorf("Expected empty compressed data, got %d bytes", len(compressed))
	}

	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress empty data: %v", err)
	}

	if len(decompressed) != 0 {
		t.Errorf("Expected empty decompressed data, got %d bytes", len(decompressed))
	}
}

func TestRandomData(t *testing.T) {
	// Random data shouldn't compress well
	compressor, err := NewCompressor(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressor: %v", err)
	}
	defer compressor.Close()

	// Simulate random data (not truly random, but varied enough)
	data := make([]byte, 1000)
	for i := range data {
		data[i] = byte(i % 256)
	}

	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	// Random data might not compress well, but should still decompress correctly
	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Decompressed data doesn't match original")
	}

	t.Logf("Random data: Original %d, Compressed %d, Ratio: %.2f%%",
		len(data), len(compressed), CompressionRatio(len(data), len(compressed))*100)
}

func TestCompressionRatioCalculation(t *testing.T) {
	tests := []struct {
		original   int
		compressed int
		wantRatio  float64
		wantSaving float64
	}{
		{1000, 500, 0.5, 50.0},
		{1000, 250, 0.25, 75.0},
		{1000, 1000, 1.0, 0.0},
		{0, 0, 0.0, 0.0},
	}

	for _, tt := range tests {
		ratio := CompressionRatio(tt.original, tt.compressed)
		savings := SpaceSavings(tt.original, tt.compressed)

		if ratio != tt.wantRatio {
			t.Errorf("CompressionRatio(%d, %d) = %f, want %f",
				tt.original, tt.compressed, ratio, tt.wantRatio)
		}

		if savings != tt.wantSaving {
			t.Errorf("SpaceSavings(%d, %d) = %f, want %f",
				tt.original, tt.compressed, savings, tt.wantSaving)
		}
	}
}

func TestAlgorithmString(t *testing.T) {
	tests := []struct {
		algo Algorithm
		want string
	}{
		{AlgorithmNone, "none"},
		{AlgorithmSnappy, "snappy"},
		{AlgorithmZstd, "zstd"},
		{AlgorithmGzip, "gzip"},
		{AlgorithmZlib, "zlib"},
		{Algorithm(999), "unknown"},
	}

	for _, tt := range tests {
		got := tt.algo.String()
		if got != tt.want {
			t.Errorf("Algorithm(%d).String() = %s, want %s", tt.algo, got, tt.want)
		}
	}
}
