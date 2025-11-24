package compression

import (
	"encoding/binary"
	"fmt"

	"github.com/mnohosten/laura-db/pkg/storage"
)

const (
	// CompressedPageHeaderSize is the size of the compressed page header
	// [1-byte algorithm][4-byte original size][4-byte compressed size]
	CompressedPageHeaderSize = 9
)

// CompressedPage wraps storage pages with compression
type CompressedPage struct {
	compressor *Compressor
}

// NewCompressedPage creates a new compressed page handler
func NewCompressedPage(config *Config) (*CompressedPage, error) {
	compressor, err := NewCompressor(config)
	if err != nil {
		return nil, err
	}

	return &CompressedPage{
		compressor: compressor,
	}, nil
}

// CompressPage compresses a storage page's data
// Returns: [header][compressed data] where header contains metadata
func (cp *CompressedPage) CompressPage(page *storage.Page) ([]byte, error) {
	// Serialize the page
	pageData := page.Serialize()

	// Compress the page data
	compressed, err := cp.compressor.Compress(pageData)
	if err != nil {
		return nil, fmt.Errorf("failed to compress page: %w", err)
	}

	// Build result with header
	// Header: [1-byte algorithm][4-byte original size][4-byte compressed size]
	result := make([]byte, CompressedPageHeaderSize+len(compressed))
	result[0] = byte(cp.compressor.config.Algorithm)
	binary.LittleEndian.PutUint32(result[1:5], uint32(len(pageData)))
	binary.LittleEndian.PutUint32(result[5:9], uint32(len(compressed)))
	copy(result[CompressedPageHeaderSize:], compressed)

	return result, nil
}

// DecompressPage decompresses page data and reconstructs the page
func (cp *CompressedPage) DecompressPage(data []byte) (*storage.Page, error) {
	if len(data) < CompressedPageHeaderSize {
		return nil, fmt.Errorf("invalid compressed page data: too short")
	}

	// Read header
	algorithm := Algorithm(data[0])
	originalSize := binary.LittleEndian.Uint32(data[1:5])
	compressedSize := binary.LittleEndian.Uint32(data[5:9])

	// Validate algorithm matches
	if algorithm != cp.compressor.config.Algorithm {
		return nil, fmt.Errorf("algorithm mismatch: expected %v, got %v",
			cp.compressor.config.Algorithm, algorithm)
	}

	// Validate compressed size
	if len(data)-CompressedPageHeaderSize != int(compressedSize) {
		return nil, fmt.Errorf("compressed size mismatch: expected %d, got %d",
			compressedSize, len(data)-CompressedPageHeaderSize)
	}

	// Decompress
	compressedData := data[CompressedPageHeaderSize:]
	decompressed, err := cp.compressor.Decompress(compressedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress page: %w", err)
	}

	// Validate original size
	if len(decompressed) != int(originalSize) {
		return nil, fmt.Errorf("decompressed size mismatch: expected %d, got %d",
			originalSize, len(decompressed))
	}

	// Reconstruct page
	page := storage.NewPage(0, storage.PageTypeData)
	if err := page.Deserialize(decompressed); err != nil {
		return nil, fmt.Errorf("failed to deserialize page: %w", err)
	}

	return page, nil
}

// Close closes the compressed page handler
func (cp *CompressedPage) Close() error {
	return cp.compressor.Close()
}

// PageCompressionStats holds statistics about page compression
type PageCompressionStats struct {
	PageID         storage.PageID
	OriginalSize   int
	CompressedSize int
	Ratio          float64
	SpaceSavings   float64
	Algorithm      string
}

// GetPageCompressionStats returns compression statistics for a page
func (cp *CompressedPage) GetPageCompressionStats(page *storage.Page) (*PageCompressionStats, error) {
	// Serialize page
	pageData := page.Serialize()

	// Compress
	compressed, err := cp.compressor.Compress(pageData)
	if err != nil {
		return nil, fmt.Errorf("failed to compress page: %w", err)
	}

	originalSize := len(pageData)
	compressedSize := len(compressed)

	return &PageCompressionStats{
		PageID:         page.ID,
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		Ratio:          CompressionRatio(originalSize, compressedSize),
		SpaceSavings:   SpaceSavings(originalSize, compressedSize),
		Algorithm:      cp.compressor.config.Algorithm.String(),
	}, nil
}
