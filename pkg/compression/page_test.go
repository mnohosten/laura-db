package compression

import (
	"bytes"
	"testing"

	"github.com/mnohosten/laura-db/pkg/storage"
)

func TestCompressedPageCompressDecompress(t *testing.T) {
	compPage, err := NewCompressedPage(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressed page: %v", err)
	}
	defer compPage.Close()

	// Create a test page
	page := storage.NewPage(123, storage.PageTypeData)
	page.LSN = 456
	// Fill page with some data
	copy(page.Data, []byte("This is test data for page compression"))

	// Compress
	compressed, err := compPage.CompressPage(page)
	if err != nil {
		t.Fatalf("Failed to compress page: %v", err)
	}

	// Decompress
	decompressed, err := compPage.DecompressPage(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress page: %v", err)
	}

	// Verify page metadata
	if decompressed.ID != page.ID {
		t.Errorf("Page ID mismatch: got %d, want %d", decompressed.ID, page.ID)
	}

	if decompressed.Type != page.Type {
		t.Errorf("Page Type mismatch: got %d, want %d", decompressed.Type, page.Type)
	}

	if decompressed.LSN != page.LSN {
		t.Errorf("Page LSN mismatch: got %d, want %d", decompressed.LSN, page.LSN)
	}

	// Verify data
	if !bytes.Equal(decompressed.Data, page.Data) {
		t.Errorf("Page data mismatch")
	}
}

func TestCompressedPageWithDifferentAlgorithms(t *testing.T) {
	algorithms := []struct {
		name   string
		config *Config
	}{
		{"Snappy", SnappyConfig()},
		{"Zstd", ZstdConfig(3)},
		{"Gzip", GzipConfig(6)},
		{"Zlib", &Config{Algorithm: AlgorithmZlib, Level: 6}},
	}

	// Create a test page with compressible data
	page := storage.NewPage(100, storage.PageTypeData)
	// Fill with repeating pattern
	pattern := []byte("ABCDEFGH")
	for i := 0; i < len(page.Data); i += len(pattern) {
		copy(page.Data[i:], pattern)
	}

	for _, algo := range algorithms {
		t.Run(algo.name, func(t *testing.T) {
			compPage, err := NewCompressedPage(algo.config)
			if err != nil {
				t.Fatalf("Failed to create compressed page: %v", err)
			}
			defer compPage.Close()

			compressed, err := compPage.CompressPage(page)
			if err != nil {
				t.Fatalf("Failed to compress page: %v", err)
			}

			t.Logf("%s: Original %d bytes -> Compressed %d bytes (%.2f%% ratio)",
				algo.name, storage.PageSize, len(compressed),
				float64(len(compressed))/float64(storage.PageSize)*100)

			decompressed, err := compPage.DecompressPage(compressed)
			if err != nil {
				t.Fatalf("Failed to decompress page: %v", err)
			}

			if !bytes.Equal(decompressed.Data, page.Data) {
				t.Errorf("Decompressed data doesn't match original")
			}
		})
	}
}

func TestCompressedPageFullPage(t *testing.T) {
	compPage, err := NewCompressedPage(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressed page: %v", err)
	}
	defer compPage.Close()

	// Create a page filled with data
	page := storage.NewPage(42, storage.PageTypeIndex)
	page.LSN = 999
	// Fill entire page data
	for i := range page.Data {
		page.Data[i] = byte(i % 256)
	}

	compressed, err := compPage.CompressPage(page)
	if err != nil {
		t.Fatalf("Failed to compress page: %v", err)
	}

	decompressed, err := compPage.DecompressPage(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress page: %v", err)
	}

	if !bytes.Equal(decompressed.Data, page.Data) {
		t.Errorf("Page data mismatch")
	}

	if decompressed.ID != page.ID || decompressed.Type != page.Type || decompressed.LSN != page.LSN {
		t.Errorf("Page metadata mismatch")
	}
}

func TestGetPageCompressionStats(t *testing.T) {
	compPage, err := NewCompressedPage(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressed page: %v", err)
	}
	defer compPage.Close()

	// Create a page with compressible data
	page := storage.NewPage(1, storage.PageTypeData)
	// Repetitive data compresses well
	pattern := "This is a repeating pattern for testing compression. "
	for i := 0; i+len(pattern) < len(page.Data); i += len(pattern) {
		copy(page.Data[i:], pattern)
	}

	stats, err := compPage.GetPageCompressionStats(page)
	if err != nil {
		t.Fatalf("Failed to get compression stats: %v", err)
	}

	t.Logf("Page ID: %d", stats.PageID)
	t.Logf("Original Size: %d bytes", stats.OriginalSize)
	t.Logf("Compressed Size: %d bytes", stats.CompressedSize)
	t.Logf("Compression Ratio: %.2f%%", stats.Ratio*100)
	t.Logf("Space Savings: %.2f%%", stats.SpaceSavings)
	t.Logf("Algorithm: %s", stats.Algorithm)

	if stats.PageID != page.ID {
		t.Errorf("Page ID mismatch in stats")
	}

	if stats.OriginalSize != storage.PageSize {
		t.Errorf("Original size should be PageSize (%d), got %d", storage.PageSize, stats.OriginalSize)
	}

	if stats.CompressedSize <= 0 {
		t.Error("Compressed size should be positive")
	}

	if stats.Algorithm != "zstd" {
		t.Errorf("Algorithm mismatch: got %s, want zstd", stats.Algorithm)
	}

	// Repetitive data should compress well
	if stats.SpaceSavings < 50 {
		t.Logf("Warning: Expected >50%% savings for repetitive data, got %.2f%%", stats.SpaceSavings)
	}
}

func TestCompressedPageEmptyData(t *testing.T) {
	compPage, err := NewCompressedPage(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressed page: %v", err)
	}
	defer compPage.Close()

	// Create a page with empty data (all zeros)
	page := storage.NewPage(0, storage.PageTypeData)
	// Data is already all zeros from initialization

	compressed, err := compPage.CompressPage(page)
	if err != nil {
		t.Fatalf("Failed to compress page: %v", err)
	}

	// All zeros should compress very well
	t.Logf("Empty page: %d bytes -> %d bytes (%.2f%% ratio)",
		storage.PageSize, len(compressed),
		float64(len(compressed))/float64(storage.PageSize)*100)

	decompressed, err := compPage.DecompressPage(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress page: %v", err)
	}

	if !bytes.Equal(decompressed.Data, page.Data) {
		t.Errorf("Decompressed data doesn't match original")
	}
}

func TestCompressedPageInvalidData(t *testing.T) {
	compPage, err := NewCompressedPage(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressed page: %v", err)
	}
	defer compPage.Close()

	// Test with too short data
	_, err = compPage.DecompressPage([]byte{1, 2, 3})
	if err == nil {
		t.Error("Expected error for too short data")
	}

	// Test with invalid header
	invalidData := make([]byte, CompressedPageHeaderSize+10)
	invalidData[0] = byte(AlgorithmZstd)
	// Set invalid sizes
	_, err = compPage.DecompressPage(invalidData)
	if err == nil {
		t.Error("Expected error for invalid compressed data")
	}
}

func TestCompressedPageAlgorithmMismatch(t *testing.T) {
	// Compress with Zstd
	compPageZstd, err := NewCompressedPage(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create zstd compressor: %v", err)
	}
	defer compPageZstd.Close()

	page := storage.NewPage(1, storage.PageTypeData)
	copy(page.Data, []byte("test data"))

	compressed, err := compPageZstd.CompressPage(page)
	if err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	// Try to decompress with Snappy
	compPageSnappy, err := NewCompressedPage(SnappyConfig())
	if err != nil {
		t.Fatalf("Failed to create snappy compressor: %v", err)
	}
	defer compPageSnappy.Close()

	_, err = compPageSnappy.DecompressPage(compressed)
	if err == nil {
		t.Error("Expected error for algorithm mismatch")
	}
}

func TestCompressedPageDifferentTypes(t *testing.T) {
	compPage, err := NewCompressedPage(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressed page: %v", err)
	}
	defer compPage.Close()

	pageTypes := []storage.PageType{
		storage.PageTypeData,
		storage.PageTypeIndex,
		storage.PageTypeFreeList,
		storage.PageTypeOverflow,
	}

	for _, pageType := range pageTypes {
		t.Run(pageType.String(), func(t *testing.T) {
			page := storage.NewPage(1, pageType)
			copy(page.Data, []byte("test data for different page types"))

			compressed, err := compPage.CompressPage(page)
			if err != nil {
				t.Fatalf("Failed to compress %v page: %v", pageType, err)
			}

			decompressed, err := compPage.DecompressPage(compressed)
			if err != nil {
				t.Fatalf("Failed to decompress %v page: %v", pageType, err)
			}

			if decompressed.Type != pageType {
				t.Errorf("Page type mismatch: got %v, want %v", decompressed.Type, pageType)
			}
		})
	}
}
