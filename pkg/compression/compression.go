package compression

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"

	"github.com/klauspost/compress/snappy"
	"github.com/klauspost/compress/zstd"
)

// Algorithm represents a compression algorithm
type Algorithm int

const (
	// AlgorithmNone indicates no compression
	AlgorithmNone Algorithm = iota
	// AlgorithmSnappy is fast compression with moderate ratio (default for hot data)
	AlgorithmSnappy
	// AlgorithmZstd is balanced compression with good speed and ratio (recommended)
	AlgorithmZstd
	// AlgorithmGzip is standard compression with good ratio
	AlgorithmGzip
	// AlgorithmZlib is similar to gzip
	AlgorithmZlib
)

// String returns the string representation of the algorithm
func (a Algorithm) String() string {
	switch a {
	case AlgorithmNone:
		return "none"
	case AlgorithmSnappy:
		return "snappy"
	case AlgorithmZstd:
		return "zstd"
	case AlgorithmGzip:
		return "gzip"
	case AlgorithmZlib:
		return "zlib"
	default:
		return "unknown"
	}
}

// Config holds compression configuration
type Config struct {
	Algorithm Algorithm
	Level     int // Compression level (meaning varies by algorithm)
}

// DefaultConfig returns the default compression configuration (Zstd with default level)
func DefaultConfig() *Config {
	return &Config{
		Algorithm: AlgorithmZstd,
		Level:     3, // Default Zstd level (balanced)
	}
}

// SnappyConfig returns configuration for Snappy (fast compression)
func SnappyConfig() *Config {
	return &Config{
		Algorithm: AlgorithmSnappy,
		Level:     0, // Snappy doesn't use levels
	}
}

// GzipConfig returns configuration for Gzip
func GzipConfig(level int) *Config {
	if level < gzip.NoCompression || level > gzip.BestCompression {
		level = gzip.DefaultCompression
	}
	return &Config{
		Algorithm: AlgorithmGzip,
		Level:     level,
	}
}

// ZstdConfig returns configuration for Zstd
func ZstdConfig(level int) *Config {
	// Zstd levels typically range from 1 (fastest) to 19 (best compression)
	if level < 1 || level > 19 {
		level = 3 // Default level
	}
	return &Config{
		Algorithm: AlgorithmZstd,
		Level:     level,
	}
}

// Compressor handles data compression
type Compressor struct {
	config     *Config
	zstdEnc    *zstd.Encoder
	zstdDec    *zstd.Decoder
	bufferPool *bytes.Buffer
}

// NewCompressor creates a new compressor with the given configuration
func NewCompressor(config *Config) (*Compressor, error) {
	if config == nil {
		config = DefaultConfig()
	}

	c := &Compressor{
		config:     config,
		bufferPool: new(bytes.Buffer),
	}

	// Pre-create zstd encoder/decoder if using zstd
	if config.Algorithm == AlgorithmZstd {
		var err error
		encLevel := zstd.EncoderLevelFromZstd(config.Level)
		c.zstdEnc, err = zstd.NewWriter(nil, zstd.WithEncoderLevel(encLevel))
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
		}

		c.zstdDec, err = zstd.NewReader(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
		}
	}

	return c, nil
}

// Compress compresses the input data
func (c *Compressor) Compress(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	switch c.config.Algorithm {
	case AlgorithmNone:
		return data, nil

	case AlgorithmSnappy:
		return snappy.Encode(nil, data), nil

	case AlgorithmZstd:
		return c.zstdEnc.EncodeAll(data, nil), nil

	case AlgorithmGzip:
		c.bufferPool.Reset()
		writer, err := gzip.NewWriterLevel(c.bufferPool, c.config.Level)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip writer: %w", err)
		}
		if _, err := writer.Write(data); err != nil {
			return nil, fmt.Errorf("failed to write gzip data: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("failed to close gzip writer: %w", err)
		}
		return c.bufferPool.Bytes(), nil

	case AlgorithmZlib:
		c.bufferPool.Reset()
		writer, err := zlib.NewWriterLevel(c.bufferPool, c.config.Level)
		if err != nil {
			return nil, fmt.Errorf("failed to create zlib writer: %w", err)
		}
		if _, err := writer.Write(data); err != nil {
			return nil, fmt.Errorf("failed to write zlib data: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("failed to close zlib writer: %w", err)
		}
		return c.bufferPool.Bytes(), nil

	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %v", c.config.Algorithm)
	}
}

// Decompress decompresses the input data
func (c *Compressor) Decompress(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	switch c.config.Algorithm {
	case AlgorithmNone:
		return data, nil

	case AlgorithmSnappy:
		decoded, err := snappy.Decode(nil, data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode snappy: %w", err)
		}
		return decoded, nil

	case AlgorithmZstd:
		decoded, err := c.zstdDec.DecodeAll(data, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to decode zstd: %w", err)
		}
		return decoded, nil

	case AlgorithmGzip:
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.Close()

		c.bufferPool.Reset()
		if _, err := io.Copy(c.bufferPool, reader); err != nil {
			return nil, fmt.Errorf("failed to read gzip data: %w", err)
		}
		return c.bufferPool.Bytes(), nil

	case AlgorithmZlib:
		reader, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("failed to create zlib reader: %w", err)
		}
		defer reader.Close()

		c.bufferPool.Reset()
		if _, err := io.Copy(c.bufferPool, reader); err != nil {
			return nil, fmt.Errorf("failed to read zlib data: %w", err)
		}
		return c.bufferPool.Bytes(), nil

	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %v", c.config.Algorithm)
	}
}

// Close closes the compressor and releases resources
func (c *Compressor) Close() error {
	if c.zstdEnc != nil {
		c.zstdEnc.Close()
	}
	if c.zstdDec != nil {
		c.zstdDec.Close()
	}
	return nil
}

// CompressionRatio calculates the compression ratio
func CompressionRatio(originalSize, compressedSize int) float64 {
	if originalSize == 0 {
		return 0
	}
	return float64(compressedSize) / float64(originalSize)
}

// SpaceSavings calculates the space savings percentage
func SpaceSavings(originalSize, compressedSize int) float64 {
	if originalSize == 0 {
		return 0
	}
	return (1.0 - CompressionRatio(originalSize, compressedSize)) * 100
}
