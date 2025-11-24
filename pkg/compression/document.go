package compression

import (
	"fmt"

	"github.com/mnohosten/laura-db/pkg/document"
)

// CompressedDocument wraps a document with compression capabilities
type CompressedDocument struct {
	compressor *Compressor
}

// NewCompressedDocument creates a new compressed document handler
func NewCompressedDocument(config *Config) (*CompressedDocument, error) {
	compressor, err := NewCompressor(config)
	if err != nil {
		return nil, err
	}

	return &CompressedDocument{
		compressor: compressor,
	}, nil
}

// Encode encodes and compresses a document
func (cd *CompressedDocument) Encode(doc *document.Document) ([]byte, error) {
	// First encode to BSON
	encoder := document.NewEncoder()
	bsonData, err := encoder.Encode(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to encode document: %w", err)
	}

	// Then compress the BSON data
	compressed, err := cd.compressor.Compress(bsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to compress document: %w", err)
	}

	return compressed, nil
}

// Decode decompresses and decodes a document
func (cd *CompressedDocument) Decode(data []byte) (*document.Document, error) {
	// First decompress
	decompressed, err := cd.compressor.Decompress(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress document: %w", err)
	}

	// Then decode from BSON
	decoder := document.NewDecoder(decompressed)
	doc, err := decoder.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode document: %w", err)
	}

	return doc, nil
}

// Close closes the compressed document handler
func (cd *CompressedDocument) Close() error {
	return cd.compressor.Close()
}

// CompressionStats holds statistics about document compression
type CompressionStats struct {
	OriginalSize   int
	CompressedSize int
	Ratio          float64
	SpaceSavings   float64
	Algorithm      string
}

// GetCompressionStats returns compression statistics for a document
func (cd *CompressedDocument) GetCompressionStats(doc *document.Document) (*CompressionStats, error) {
	// Encode to BSON
	encoder := document.NewEncoder()
	bsonData, err := encoder.Encode(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to encode document: %w", err)
	}

	// Compress
	compressed, err := cd.compressor.Compress(bsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to compress document: %w", err)
	}

	originalSize := len(bsonData)
	compressedSize := len(compressed)

	return &CompressionStats{
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		Ratio:          CompressionRatio(originalSize, compressedSize),
		SpaceSavings:   SpaceSavings(originalSize, compressedSize),
		Algorithm:      cd.compressor.config.Algorithm.String(),
	}, nil
}
