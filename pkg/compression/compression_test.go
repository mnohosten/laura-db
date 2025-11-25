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

// TestGzipConfigInvalidLevel tests GzipConfig with invalid levels
func TestGzipConfigInvalidLevel(t *testing.T) {
	tests := []struct {
		name  string
		level int
	}{
		{"Too Low", -2},
		{"Too High", 10},
		{"Way Too High", 100},
		{"Negative", -999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := GzipConfig(tt.level)
			if config.Level != -1 { // DefaultCompression value
				t.Errorf("GzipConfig(%d) should default to DefaultCompression, got level %d", tt.level, config.Level)
			}
			if config.Algorithm != AlgorithmGzip {
				t.Errorf("Expected AlgorithmGzip, got %v", config.Algorithm)
			}
		})
	}
}

// TestGzipConfigValidLevels tests GzipConfig with valid levels
func TestGzipConfigValidLevels(t *testing.T) {
	tests := []int{0, 1, 5, 9} // NoCompression, BestSpeed, DefaultCompression, BestCompression

	for _, level := range tests {
		t.Run(string(rune('0'+level)), func(t *testing.T) {
			config := GzipConfig(level)
			if config.Level != level {
				t.Errorf("GzipConfig(%d) should preserve level, got %d", level, config.Level)
			}
		})
	}
}

// TestZstdConfigInvalidLevel tests ZstdConfig with invalid levels
func TestZstdConfigInvalidLevel(t *testing.T) {
	tests := []struct {
		name  string
		level int
	}{
		{"Zero", 0},
		{"Negative", -5},
		{"Too High", 20},
		{"Way Too High", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ZstdConfig(tt.level)
			if config.Level != 3 { // Default level
				t.Errorf("ZstdConfig(%d) should default to level 3, got %d", tt.level, config.Level)
			}
			if config.Algorithm != AlgorithmZstd {
				t.Errorf("Expected AlgorithmZstd, got %v", config.Algorithm)
			}
		})
	}
}

// TestZstdConfigValidLevels tests ZstdConfig with valid levels
func TestZstdConfigValidLevels(t *testing.T) {
	tests := []int{1, 3, 9, 15, 19}

	for _, level := range tests {
		t.Run(string(rune('0'+level)), func(t *testing.T) {
			config := ZstdConfig(level)
			if config.Level != level {
				t.Errorf("ZstdConfig(%d) should preserve level, got %d", level, config.Level)
			}
		})
	}
}

// TestNewCompressorNilConfig tests NewCompressor with nil config
func TestNewCompressorNilConfig(t *testing.T) {
	compressor, err := NewCompressor(nil)
	if err != nil {
		t.Fatalf("NewCompressor(nil) should use default config, got error: %v", err)
	}
	defer compressor.Close()

	if compressor.config.Algorithm != AlgorithmZstd {
		t.Errorf("Expected default algorithm Zstd, got %v", compressor.config.Algorithm)
	}
	if compressor.config.Level != 3 {
		t.Errorf("Expected default level 3, got %d", compressor.config.Level)
	}
}

// TestCompressorInvalidAlgorithm tests compression with unsupported algorithm
func TestCompressorInvalidAlgorithm(t *testing.T) {
	config := &Config{Algorithm: Algorithm(999), Level: 0}
	compressor, err := NewCompressor(config)
	if err != nil {
		t.Fatalf("Failed to create compressor: %v", err)
	}
	defer compressor.Close()

	data := []byte("test data")

	// Test compress with invalid algorithm
	_, err = compressor.Compress(data)
	if err == nil {
		t.Error("Compress with invalid algorithm should return error")
	}

	// Test decompress with invalid algorithm
	_, err = compressor.Decompress(data)
	if err == nil {
		t.Error("Decompress with invalid algorithm should return error")
	}
}

// TestDecompressInvalidData tests decompression of corrupted data
func TestDecompressInvalidData(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		data   []byte
	}{
		{"Snappy Invalid", SnappyConfig(), []byte("invalid snappy data")},
		{"Zstd Invalid", ZstdConfig(3), []byte("invalid zstd data")},
		{"Gzip Invalid", GzipConfig(6), []byte("invalid gzip data")},
		{"Zlib Invalid", &Config{Algorithm: AlgorithmZlib, Level: 6}, []byte("invalid zlib data")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressor, err := NewCompressor(tt.config)
			if err != nil {
				t.Fatalf("Failed to create compressor: %v", err)
			}
			defer compressor.Close()

			_, err = compressor.Decompress(tt.data)
			if err == nil {
				t.Error("Decompressing invalid data should return error")
			}
		})
	}
}

// TestCompressDecompressRoundTrip tests all algorithms with various data sizes
func TestCompressDecompressRoundTrip(t *testing.T) {
	algorithms := []struct {
		name   string
		config *Config
	}{
		{"None", &Config{Algorithm: AlgorithmNone}},
		{"Snappy", SnappyConfig()},
		{"Zstd-1", ZstdConfig(1)},
		{"Zstd-19", ZstdConfig(19)},
		{"Gzip-0", GzipConfig(0)},
		{"Gzip-9", GzipConfig(9)},
		{"Zlib-1", &Config{Algorithm: AlgorithmZlib, Level: 1}},
		{"Zlib-9", &Config{Algorithm: AlgorithmZlib, Level: 9}},
	}

	dataSizes := []int{0, 1, 10, 100, 1000, 10000}

	for _, algo := range algorithms {
		t.Run(algo.name, func(t *testing.T) {
			compressor, err := NewCompressor(algo.config)
			if err != nil {
				t.Fatalf("Failed to create compressor: %v", err)
			}
			defer compressor.Close()

			for _, size := range dataSizes {
				data := make([]byte, size)
				for i := range data {
					data[i] = byte(i % 256)
				}

				compressed, err := compressor.Compress(data)
				if err != nil {
					t.Errorf("Compress failed for size %d: %v", size, err)
					continue
				}

				// Make a copy of compressed data since buffer pool may reuse it
				compressedCopy := make([]byte, len(compressed))
				copy(compressedCopy, compressed)

				decompressed, err := compressor.Decompress(compressedCopy)
				if err != nil {
					t.Errorf("Decompress failed for size %d: %v", size, err)
					continue
				}

				if !bytes.Equal(decompressed, data) {
					t.Errorf("Round trip failed for size %d", size)
				}
			}
		})
	}
}
