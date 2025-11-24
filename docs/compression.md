# Compression in LauraDB

LauraDB implements comprehensive compression support for documents, indexes, and storage pages to reduce disk usage and improve I/O performance.

## Overview

The compression package (`pkg/compression`) provides:

- **Multiple algorithms**: Snappy, Zstd, Gzip, Zlib
- **Document compression**: Compress BSON-encoded documents
- **Page compression**: Compress storage pages (4KB blocks)
- **Configurable levels**: Trade-off between speed and compression ratio
- **Statistics**: Track compression ratios and space savings

## Compression Algorithms

### Snappy (Fast Compression)
- **Speed**: Very fast (2.2μs compression, 563ns decompression)
- **Ratio**: Moderate (~5-10% for typical data)
- **Use case**: Hot data, frequently accessed documents
- **Memory**: Low overhead (~4KB per operation)

```go
config := compression.SnappyConfig()
compressor, _ := compression.NewCompressor(config)
defer compressor.Close()
```

### Zstd (Recommended - Balanced)
- **Speed**: Fast (2.3μs compression, 2.2μs decompression)
- **Ratio**: Excellent (~1-2% for repetitive data, 70% for typical documents)
- **Use case**: Default choice for most workloads
- **Levels**: 1-19 (1=fastest, 19=best compression, default=3)

```go
config := compression.ZstdConfig(3)  // Default level
compressor, _ := compression.NewCompressor(config)
defer compressor.Close()
```

### Gzip (Standard Compression)
- **Speed**: Slower (47μs compression, 4.2μs decompression)
- **Ratio**: Good (~1-2% for repetitive data)
- **Use case**: Cold storage, archival data
- **Levels**: 1-9 (1=fastest, 9=best compression, default=6)

```go
config := compression.GzipConfig(6)  // Default level
compressor, _ := compression.NewCompressor(config)
defer compressor.Close()
```

### Zlib
- **Speed**: Similar to Gzip
- **Ratio**: Similar to Gzip (~1-3% for repetitive data)
- **Use case**: Compatibility with zlib-based systems

```go
config := &compression.Config{
    Algorithm: compression.AlgorithmZlib,
    Level:     6,
}
compressor, _ := compression.NewCompressor(config)
defer compressor.Close()
```

## Usage Examples

### Compressing Documents

```go
package main

import (
    "fmt"
    "github.com/mnohosten/laura-db/pkg/compression"
    "github.com/mnohosten/laura-db/pkg/document"
)

func main() {
    // Create compressed document handler
    compDoc, err := compression.NewCompressedDocument(compression.ZstdConfig(3))
    if err != nil {
        panic(err)
    }
    defer compDoc.Close()

    // Create a document
    doc := document.NewDocument()
    doc.Set("_id", document.NewObjectID())
    doc.Set("name", "Alice")
    doc.Set("email", "alice@example.com")
    doc.Set("age", int64(30))

    // Compress
    compressed, err := compDoc.Encode(doc)
    if err != nil {
        panic(err)
    }

    // Decompress
    decoded, err := compDoc.Decode(compressed)
    if err != nil {
        panic(err)
    }

    // Get compression statistics
    stats, _ := compDoc.GetCompressionStats(doc)
    fmt.Printf("Original: %d bytes\n", stats.OriginalSize)
    fmt.Printf("Compressed: %d bytes\n", stats.CompressedSize)
    fmt.Printf("Space Savings: %.2f%%\n", stats.SpaceSavings)
}
```

### Compressing Storage Pages

```go
package main

import (
    "fmt"
    "github.com/mnohosten/laura-db/pkg/compression"
    "github.com/mnohosten/laura-db/pkg/storage"
)

func main() {
    // Create compressed page handler
    compPage, err := compression.NewCompressedPage(compression.ZstdConfig(3))
    if err != nil {
        panic(err)
    }
    defer compPage.Close()

    // Create a storage page
    page := storage.NewPage(1, storage.PageTypeData)
    copy(page.Data, []byte("Important data..."))

    // Compress the page
    compressed, err := compPage.CompressPage(page)
    if err != nil {
        panic(err)
    }

    // Decompress the page
    decompressed, err := compPage.DecompressPage(compressed)
    if err != nil {
        panic(err)
    }

    // Get compression statistics
    stats, _ := compPage.GetPageCompressionStats(page)
    fmt.Printf("Page %d: %.2f%% space savings\n", stats.PageID, stats.SpaceSavings)
}
```

### Raw Data Compression

```go
package main

import (
    "fmt"
    "github.com/mnohosten/laura-db/pkg/compression"
)

func main() {
    // Create compressor
    compressor, err := compression.NewCompressor(compression.ZstdConfig(3))
    if err != nil {
        panic(err)
    }
    defer compressor.Close()

    // Compress data
    data := []byte("Hello, World!")
    compressed, err := compressor.Compress(data)
    if err != nil {
        panic(err)
    }

    // Decompress data
    decompressed, err := compressor.Decompress(compressed)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Original: %s\n", string(decompressed))
}
```

## Performance Characteristics

### Compression Speed (Apple M4 Max)

| Algorithm | Compression | Decompression | Memory/Op |
|-----------|-------------|---------------|-----------|
| Snappy    | 2.2μs       | 563ns         | 4KB       |
| Zstd-1    | 2.1μs       | 2.2μs         | 4KB       |
| Zstd-3    | 2.3μs       | 2.2μs         | 4KB       |
| Zstd-9    | 2.6μs       | 2.2μs         | 7KB       |
| Gzip-1    | 63μs        | 4.2μs         | 1.2MB     |
| Gzip-6    | 50μs        | 4.2μs         | 800KB     |
| Gzip-9    | 48μs        | 4.2μs         | 800KB     |

### Document Compression

| Operation | Time | Memory | Allocations |
|-----------|------|--------|-------------|
| Compress Document | 5.2μs | 2KB | 38 |
| Decompress Document | 1.2μs | 2.6KB | 69 |
| Compress Large Doc (100 fields) | 10μs | 23KB | 111 |
| Decompress Large Doc | 17μs | 39KB | 728 |

### Page Compression

| Operation | Time | Memory |
|-----------|------|--------|
| Compress Page (4KB) | 4.8μs | 8.4KB |
| Decompress Page | 2.6μs | 13KB |

### Compression Ratios (Real-World Data)

| Data Type | Algorithm | Original | Compressed | Ratio | Savings |
|-----------|-----------|----------|------------|-------|---------|
| Repetitive JSON | Snappy | 10KB | 569B | 5.69% | 94.31% |
| Repetitive JSON | Zstd-3 | 10KB | 87B | 0.87% | 99.13% |
| Repetitive JSON | Gzip-6 | 10KB | 142B | 1.42% | 98.58% |
| Empty Page | Zstd-3 | 4KB | 24B | 0.59% | 99.41% |
| Repetitive Page | Zstd-3 | 4KB | 77B | 1.88% | 98.12% |
| Typical Document | Zstd-3 | 266B | 185B | 69.55% | 30.45% |

## Choosing a Compression Algorithm

### Use Snappy when:
- **Speed is critical**: Minimal CPU overhead
- **Hot data**: Frequently read/written documents
- **Low latency**: Real-time applications
- **Compression ratio is secondary**: Can tolerate larger compressed size

### Use Zstd when:
- **Balanced performance**: Good speed + excellent compression
- **General purpose**: Default choice for most workloads
- **Repetitive data**: JSON documents, logs, time-series
- **Moderate CPU available**: Acceptable 2-3μs overhead

### Use Gzip when:
- **Cold storage**: Infrequently accessed data
- **Maximum compression**: Space is more important than CPU
- **Archival**: Long-term storage
- **Compatibility**: Integration with gzip-based systems

### Use None when:
- **Already compressed**: Images, videos, already-compressed data
- **Random data**: Won't compress well anyway
- **Extreme low latency**: Cannot tolerate any overhead

## Configuration Recommendations

### Development/Testing
```go
// Fast compression for quick iteration
config := compression.SnappyConfig()
```

### Production - Hot Data
```go
// Balance of speed and compression
config := compression.ZstdConfig(1)  // Fastest Zstd
```

### Production - Warm Data
```go
// Default balanced configuration
config := compression.ZstdConfig(3)  // Recommended
```

### Production - Cold Data
```go
// Maximum compression
config := compression.ZstdConfig(9)  // Or Gzip-9
```

## Integration with Database

### Document Storage

When storing documents, compression can be applied at the BSON encoding layer:

```go
// Without compression
encoder := document.NewEncoder()
bsonData, _ := encoder.Encode(doc)

// With compression
compDoc, _ := compression.NewCompressedDocument(compression.ZstdConfig(3))
compressedData, _ := compDoc.Encode(doc)

// Space savings: ~30-70% for typical documents
```

### Page Storage

Storage pages can be compressed before writing to disk:

```go
// Without compression
pageData := page.Serialize()  // 4096 bytes

// With compression
compPage, _ := compression.NewCompressedPage(compression.ZstdConfig(3))
compressedPage, _ := compPage.CompressPage(page)  // ~77-200 bytes

// Space savings: ~95-98% for repetitive data
```

### Index Compression

B+ tree nodes can be compressed when flushing to disk:

```go
// Serialize B+ tree node to bytes
nodeData := serializeNode(node)

// Compress before writing to page
compressor, _ := compression.NewCompressor(compression.ZstdConfig(3))
compressed, _ := compressor.Compress(nodeData)

// Space savings: Significant for string keys
```

## Best Practices

### 1. Reuse Compressor Instances
```go
// Good: Reuse compressor
compressor, _ := compression.NewCompressor(config)
defer compressor.Close()
for _, data := range batch {
    compressed, _ := compressor.Compress(data)
    // Process compressed data
}

// Bad: Create new compressor each time
for _, data := range batch {
    compressor, _ := compression.NewCompressor(config)
    compressed, _ := compressor.Compress(data)
    compressor.Close()  // Wasteful!
}
```

### 2. Consider Data Characteristics
```go
// High entropy (random) data - don't compress
if isRandom(data) {
    config := &compression.Config{Algorithm: compression.AlgorithmNone}
}

// Repetitive data - use high compression
if isRepetitive(data) {
    config := compression.ZstdConfig(9)
}
```

### 3. Monitor Compression Ratios
```go
stats, _ := compDoc.GetCompressionStats(doc)
if stats.SpaceSavings < 10.0 {
    // Compression not effective, consider disabling
    log.Printf("Poor compression: %.2f%% savings", stats.SpaceSavings)
}
```

### 4. Close Compressors
```go
compressor, _ := compression.NewCompressor(config)
defer compressor.Close()  // Important: releases Zstd resources
```

## Limitations

- **CPU overhead**: Compression adds 2-50μs per operation depending on algorithm
- **Memory usage**: Each compressor uses 4KB-1MB depending on algorithm
- **No streaming**: Entire data must fit in memory (acceptable for 4KB pages)
- **Not thread-safe**: Each goroutine needs its own compressor instance

## Future Enhancements

- [ ] LZ4 compression algorithm (even faster than Snappy)
- [ ] Adaptive compression (auto-select algorithm based on data)
- [ ] Compression at collection level (enable/disable per collection)
- [ ] Dictionary compression for similar documents
- [ ] Streaming compression for large documents
- [ ] Background compression/decompression workers
- [ ] Compression statistics per collection

## References

- [Zstandard Documentation](https://github.com/facebook/zstd)
- [Snappy Documentation](https://github.com/google/snappy)
- [klauspost/compress](https://github.com/klauspost/compress) - High-performance Go compression library
