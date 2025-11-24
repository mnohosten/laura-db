package main

import (
	"fmt"
	"log"

	"github.com/mnohosten/laura-db/pkg/compression"
	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/storage"
)

func main() {
	fmt.Println("=== LauraDB Compression Demo ===")

	// Demo 1: Document Compression
	fmt.Println("1. Document Compression")
	fmt.Println("------------------------")
	demoDocumentCompression()

	fmt.Println()

	// Demo 2: Page Compression
	fmt.Println("2. Page Compression")
	fmt.Println("-------------------")
	demoPageCompression()

	fmt.Println()

	// Demo 3: Algorithm Comparison
	fmt.Println("3. Algorithm Comparison")
	fmt.Println("-----------------------")
	demoAlgorithmComparison()

	fmt.Println()

	// Demo 4: Large Document Compression
	fmt.Println("4. Large Document Compression")
	fmt.Println("------------------------------")
	demoLargeDocument()
}

func demoDocumentCompression() {
	// Create a document with realistic data
	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())
	doc.Set("name", "Alice Johnson")
	doc.Set("email", "alice.johnson@example.com")
	doc.Set("age", int64(28))
	doc.Set("active", true)
	doc.Set("address", map[string]interface{}{
		"street":  "123 Main St",
		"city":    "San Francisco",
		"state":   "CA",
		"zipcode": int64(94102),
	})
	doc.Set("tags", []interface{}{"engineer", "golang", "databases", "distributed-systems"})
	doc.Set("metadata", map[string]interface{}{
		"created_at": "2025-01-15T10:30:00Z",
		"updated_at": "2025-01-20T14:45:00Z",
		"version":    int64(3),
	})

	// Try different compression algorithms
	algorithms := []struct {
		name   string
		config *compression.Config
	}{
		{"Snappy", compression.SnappyConfig()},
		{"Zstd (Level 1)", compression.ZstdConfig(1)},
		{"Zstd (Level 3)", compression.ZstdConfig(3)},
		{"Zstd (Level 9)", compression.ZstdConfig(9)},
		{"Gzip (Level 6)", compression.GzipConfig(6)},
	}

	for _, algo := range algorithms {
		compDoc, err := compression.NewCompressedDocument(algo.config)
		if err != nil {
			log.Printf("Failed to create compressor: %v\n", err)
			continue
		}
		defer compDoc.Close()

		stats, err := compDoc.GetCompressionStats(doc)
		if err != nil {
			log.Printf("Failed to get stats: %v\n", err)
			continue
		}

		fmt.Printf("%-18s | Original: %4d bytes | Compressed: %4d bytes | Ratio: %5.2f%% | Savings: %5.2f%%\n",
			algo.name, stats.OriginalSize, stats.CompressedSize,
			stats.Ratio*100, stats.SpaceSavings)
	}

	// Verify round-trip encoding/decoding
	compDoc, _ := compression.NewCompressedDocument(compression.ZstdConfig(3))
	defer compDoc.Close()

	compressed, _ := compDoc.Encode(doc)
	decoded, err := compDoc.Decode(compressed)
	if err != nil {
		log.Fatalf("Failed to decode: %v\n", err)
	}

	name, _ := decoded.Get("name")
	fmt.Printf("\nRound-trip verification: name = %s ✓\n", name)
}

func demoPageCompression() {
	// Create a storage page with realistic data
	page := storage.NewPage(42, storage.PageTypeData)
	page.LSN = 12345

	// Fill page with semi-realistic data (simulating serialized documents)
	pattern := `{"id":"abc123","name":"Record","value":42,"data":"information"}`
	for i := 0; i+len(pattern) < len(page.Data); i += len(pattern) {
		copy(page.Data[i:], pattern)
	}

	// Try different compression algorithms
	algorithms := []struct {
		name   string
		config *compression.Config
	}{
		{"Snappy", compression.SnappyConfig()},
		{"Zstd (Level 3)", compression.ZstdConfig(3)},
		{"Gzip (Level 6)", compression.GzipConfig(6)},
	}

	for _, algo := range algorithms {
		compPage, err := compression.NewCompressedPage(algo.config)
		if err != nil {
			log.Printf("Failed to create compressor: %v\n", err)
			continue
		}
		defer compPage.Close()

		stats, err := compPage.GetPageCompressionStats(page)
		if err != nil {
			log.Printf("Failed to get stats: %v\n", err)
			continue
		}

		fmt.Printf("%-18s | Original: %4d bytes | Compressed: %4d bytes | Ratio: %5.2f%% | Savings: %5.2f%%\n",
			algo.name, stats.OriginalSize, stats.CompressedSize,
			stats.Ratio*100, stats.SpaceSavings)
	}

	// Verify round-trip compression/decompression
	compPage, _ := compression.NewCompressedPage(compression.ZstdConfig(3))
	defer compPage.Close()

	compressed, _ := compPage.CompressPage(page)
	decompressed, err := compPage.DecompressPage(compressed)
	if err != nil {
		log.Fatalf("Failed to decompress page: %v\n", err)
	}

	fmt.Printf("\nRound-trip verification: Page ID = %d, LSN = %d ✓\n",
		decompressed.ID, decompressed.LSN)
}

func demoAlgorithmComparison() {
	// Create multiple documents with varying characteristics
	testCases := []struct {
		name string
		doc  *document.Document
	}{
		{
			name: "Small Document",
			doc: func() *document.Document {
				d := document.NewDocument()
				d.Set("_id", document.NewObjectID())
				d.Set("name", "Bob")
				d.Set("age", int64(25))
				return d
			}(),
		},
		{
			name: "Medium Document",
			doc: func() *document.Document {
				d := document.NewDocument()
				d.Set("_id", document.NewObjectID())
				d.Set("title", "Engineering Best Practices")
				d.Set("content", "This document describes best practices for software engineering including code review, testing, documentation, and deployment strategies.")
				d.Set("tags", []interface{}{"engineering", "best-practices", "software", "documentation"})
				return d
			}(),
		},
		{
			name: "Repetitive Document",
			doc: func() *document.Document {
				d := document.NewDocument()
				d.Set("_id", document.NewObjectID())
				for i := 0; i < 20; i++ {
					key := fmt.Sprintf("field_%d", i)
					d.Set(key, "This is a repeating value")
				}
				return d
			}(),
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n%s:\n", tc.name)

		// Test with Zstd (recommended)
		compDoc, _ := compression.NewCompressedDocument(compression.ZstdConfig(3))
		defer compDoc.Close()

		stats, _ := compDoc.GetCompressionStats(tc.doc)
		fmt.Printf("  Zstd-3: %d bytes → %d bytes (%.1f%% savings)\n",
			stats.OriginalSize, stats.CompressedSize, stats.SpaceSavings)
	}
}

func demoLargeDocument() {
	// Create a large document with 500 fields
	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())
	doc.Set("type", "large_record")
	doc.Set("created", "2025-01-15T10:00:00Z")

	// Add many fields with realistic data
	for i := 0; i < 500; i++ {
		fieldName := fmt.Sprintf("metric_%d", i)
		doc.Set(fieldName, map[string]interface{}{
			"value":     int64(i * 10),
			"timestamp": "2025-01-15T10:00:00Z",
			"status":    "active",
			"metadata": map[string]interface{}{
				"source": "sensor",
				"unit":   "ms",
			},
		})
	}

	// Get original size
	encoder := document.NewEncoder()
	bsonData, _ := encoder.Encode(doc)
	originalSize := len(bsonData)

	// Compress with different algorithms
	algorithms := []struct {
		name   string
		config *compression.Config
	}{
		{"Snappy", compression.SnappyConfig()},
		{"Zstd-1", compression.ZstdConfig(1)},
		{"Zstd-3", compression.ZstdConfig(3)},
		{"Zstd-9", compression.ZstdConfig(9)},
	}

	fmt.Printf("Large document with 500 nested fields\n")
	fmt.Printf("Original BSON size: %d bytes (%.1f KB)\n\n", originalSize, float64(originalSize)/1024)

	for _, algo := range algorithms {
		compDoc, _ := compression.NewCompressedDocument(algo.config)
		defer compDoc.Close()

		stats, _ := compDoc.GetCompressionStats(doc)
		fmt.Printf("%-10s: %6d bytes (%.1f KB) | Ratio: %5.2f%% | Savings: %5.2f%%\n",
			algo.name, stats.CompressedSize, float64(stats.CompressedSize)/1024,
			stats.Ratio*100, stats.SpaceSavings)
	}

	// Demonstrate space savings
	compDoc, _ := compression.NewCompressedDocument(compression.ZstdConfig(3))
	defer compDoc.Close()
	stats, _ := compDoc.GetCompressionStats(doc)

	fmt.Printf("\nWith Zstd-3 compression:\n")
	fmt.Printf("  Disk space saved: %d bytes (%.1f KB)\n",
		originalSize-stats.CompressedSize,
		float64(originalSize-stats.CompressedSize)/1024)
	fmt.Printf("  For 1000 documents: %.1f MB saved\n",
		float64(originalSize-stats.CompressedSize)*1000/1024/1024)
}
