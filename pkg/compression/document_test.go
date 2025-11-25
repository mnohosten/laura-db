package compression

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
)

func TestCompressedDocumentEncodeDecode(t *testing.T) {
	compDoc, err := NewCompressedDocument(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressed document: %v", err)
	}
	defer compDoc.Close()

	// Create a test document
	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))
	doc.Set("email", "alice@example.com")
	doc.Set("active", true)

	// Encode and compress
	compressed, err := compDoc.Encode(doc)
	if err != nil {
		t.Fatalf("Failed to encode document: %v", err)
	}

	// Decode and decompress
	decoded, err := compDoc.Decode(compressed)
	if err != nil {
		t.Fatalf("Failed to decode document: %v", err)
	}

	// Verify fields
	if name, ok := decoded.Get("name"); !ok || name != "Alice" {
		t.Errorf("Name mismatch: got %v, want Alice", name)
	}

	if age, ok := decoded.Get("age"); !ok || age != int64(30) {
		t.Errorf("Age mismatch: got %v, want 30", age)
	}

	if email, ok := decoded.Get("email"); !ok || email != "alice@example.com" {
		t.Errorf("Email mismatch: got %v, want alice@example.com", email)
	}

	if active, ok := decoded.Get("active"); !ok || active != true {
		t.Errorf("Active mismatch: got %v, want true", active)
	}
}

func TestCompressedDocumentNestedData(t *testing.T) {
	compDoc, err := NewCompressedDocument(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressed document: %v", err)
	}
	defer compDoc.Close()

	// Create a document with nested data
	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())
	doc.Set("user", map[string]interface{}{
		"name": "Bob",
		"address": map[string]interface{}{
			"city":  "San Francisco",
			"state": "CA",
			"zip":   int64(94102),
		},
	})
	doc.Set("tags", []interface{}{"golang", "database", "nosql"})

	// Encode and compress
	compressed, err := compDoc.Encode(doc)
	if err != nil {
		t.Fatalf("Failed to encode document: %v", err)
	}

	// Decode and decompress
	decoded, err := compDoc.Decode(compressed)
	if err != nil {
		t.Fatalf("Failed to decode document: %v", err)
	}

	// Verify nested structure
	userVal, ok := decoded.Get("user")
	if !ok {
		t.Fatal("User field not found")
	}

	var user map[string]interface{}
	if userDoc, ok := userVal.(*document.Document); ok {
		user = userDoc.ToMap()
	} else if userMap, ok := userVal.(map[string]interface{}); ok {
		user = userMap
	} else {
		t.Fatalf("Unexpected user type: %T", userVal)
	}

	if user["name"] != "Bob" {
		t.Errorf("User name mismatch: got %v, want Bob", user["name"])
	}

	// Verify tags
	tagsVal, ok := decoded.Get("tags")
	if !ok {
		t.Fatal("Tags field not found")
	}

	tags, ok := tagsVal.([]interface{})
	if !ok {
		t.Fatalf("Tags is not an array: %T", tagsVal)
	}

	if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	}
}

func TestCompressedDocumentLargeDocument(t *testing.T) {
	compDoc, err := NewCompressedDocument(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressed document: %v", err)
	}
	defer compDoc.Close()

	// Create a large document with repetitive data
	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())

	// Add many fields with similar data
	for i := 0; i < 100; i++ {
		fieldName := "field_" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		doc.Set(fieldName, "This is a repeating value that should compress well")
	}

	// Encode and compress
	compressed, err := compDoc.Encode(doc)
	if err != nil {
		t.Fatalf("Failed to encode document: %v", err)
	}

	// Decode and decompress
	decoded, err := compDoc.Decode(compressed)
	if err != nil {
		t.Fatalf("Failed to decode document: %v", err)
	}

	if decoded.Len() != doc.Len() {
		t.Errorf("Document length mismatch: got %d, want %d", decoded.Len(), doc.Len())
	}
}

func TestGetCompressionStats(t *testing.T) {
	algorithms := []struct {
		name   string
		config *Config
	}{
		{"Snappy", SnappyConfig()},
		{"Zstd", ZstdConfig(3)},
		{"Gzip", GzipConfig(6)},
	}

	// Create a test document
	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())
	doc.Set("name", "Compression Test")
	doc.Set("description", "This is a test document to measure compression performance")
	doc.Set("data", []interface{}{
		"item1", "item2", "item3", "item4", "item5",
		"item1", "item2", "item3", "item4", "item5", // Repetition for better compression
	})

	for _, algo := range algorithms {
		t.Run(algo.name, func(t *testing.T) {
			compDoc, err := NewCompressedDocument(algo.config)
			if err != nil {
				t.Fatalf("Failed to create compressed document: %v", err)
			}
			defer compDoc.Close()

			stats, err := compDoc.GetCompressionStats(doc)
			if err != nil {
				t.Fatalf("Failed to get compression stats: %v", err)
			}

			t.Logf("Algorithm: %s", stats.Algorithm)
			t.Logf("Original Size: %d bytes", stats.OriginalSize)
			t.Logf("Compressed Size: %d bytes", stats.CompressedSize)
			t.Logf("Compression Ratio: %.2f%%", stats.Ratio*100)
			t.Logf("Space Savings: %.2f%%", stats.SpaceSavings)

			if stats.OriginalSize <= 0 {
				t.Error("Original size should be positive")
			}

			if stats.CompressedSize <= 0 {
				t.Error("Compressed size should be positive")
			}

			if stats.Algorithm != algo.config.Algorithm.String() {
				t.Errorf("Algorithm mismatch: got %s, want %s",
					stats.Algorithm, algo.config.Algorithm.String())
			}
		})
	}
}

func TestCompressedDocumentEmptyDocument(t *testing.T) {
	compDoc, err := NewCompressedDocument(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressed document: %v", err)
	}
	defer compDoc.Close()

	// Create an empty document
	doc := document.NewDocument()

	// Encode and compress
	compressed, err := compDoc.Encode(doc)
	if err != nil {
		t.Fatalf("Failed to encode empty document: %v", err)
	}

	// Decode and decompress
	decoded, err := compDoc.Decode(compressed)
	if err != nil {
		t.Fatalf("Failed to decode empty document: %v", err)
	}

	if decoded.Len() != 0 {
		t.Errorf("Expected empty document, got %d fields", decoded.Len())
	}
}

func TestCompressedDocumentAllDataTypes(t *testing.T) {
	compDoc, err := NewCompressedDocument(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressed document: %v", err)
	}
	defer compDoc.Close()

	// Create a document with all data types
	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())
	doc.Set("string", "text")
	doc.Set("int64", int64(42))
	doc.Set("float64", 3.14159)
	doc.Set("bool", true)
	doc.Set("null", nil)
	doc.Set("array", []interface{}{int64(1), int64(2), int64(3)})
	doc.Set("nested", map[string]interface{}{
		"field": "value",
	})
	doc.Set("binary", []byte{0x01, 0x02, 0x03, 0x04})

	// Encode and compress
	compressed, err := compDoc.Encode(doc)
	if err != nil {
		t.Fatalf("Failed to encode document: %v", err)
	}

	// Decode and decompress
	decoded, err := compDoc.Decode(compressed)
	if err != nil {
		t.Fatalf("Failed to decode document: %v", err)
	}

	// Verify all fields
	if str, ok := decoded.Get("string"); !ok || str != "text" {
		t.Errorf("String field mismatch")
	}

	if num, ok := decoded.Get("int64"); !ok || num != int64(42) {
		t.Errorf("Int64 field mismatch")
	}

	if f, ok := decoded.Get("float64"); !ok || f != 3.14159 {
		t.Errorf("Float64 field mismatch")
	}

	if b, ok := decoded.Get("bool"); !ok || b != true {
		t.Errorf("Bool field mismatch")
	}

	if n, ok := decoded.Get("null"); !ok || n != nil {
		t.Errorf("Null field mismatch")
	}

	if arr, ok := decoded.Get("array"); !ok {
		t.Errorf("Array field missing")
	} else {
		arrVal, ok := arr.([]interface{})
		if !ok || len(arrVal) != 3 {
			t.Errorf("Array field mismatch")
		}
	}

	if _, ok := decoded.Get("nested"); !ok {
		t.Errorf("Nested field missing")
	}

	if bin, ok := decoded.Get("binary"); !ok {
		t.Errorf("Binary field missing")
	} else {
		binVal, ok := bin.([]byte)
		if !ok || len(binVal) != 4 {
			t.Errorf("Binary field mismatch")
		}
	}
}

// TestCompressedDocumentDecodeInvalidData tests decoding corrupted compressed data
func TestCompressedDocumentDecodeInvalidData(t *testing.T) {
	compDoc, err := NewCompressedDocument(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressed document: %v", err)
	}
	defer compDoc.Close()

	// Test with invalid compressed data
	invalidData := []byte("this is not valid compressed data")
	_, err = compDoc.Decode(invalidData)
	if err == nil {
		t.Error("Expected error when decoding invalid compressed data")
	}
}

// TestCompressedDocumentNilConfig tests NewCompressedDocument with nil config
func TestCompressedDocumentNilConfig(t *testing.T) {
	compDoc, err := NewCompressedDocument(nil)
	if err != nil {
		t.Fatalf("NewCompressedDocument(nil) should use default config, got error: %v", err)
	}
	defer compDoc.Close()

	// Should work with default config
	doc := document.NewDocument()
	doc.Set("test", "value")

	compressed, err := compDoc.Encode(doc)
	if err != nil {
		t.Fatalf("Failed to encode with default config: %v", err)
	}

	decoded, err := compDoc.Decode(compressed)
	if err != nil {
		t.Fatalf("Failed to decode with default config: %v", err)
	}

	if val, ok := decoded.Get("test"); !ok || val != "value" {
		t.Errorf("Value mismatch after encode/decode with default config")
	}
}

// TestGetCompressionStatsEmptyDoc tests GetCompressionStats with empty document
func TestGetCompressionStatsEmptyDoc(t *testing.T) {
	compDoc, err := NewCompressedDocument(ZstdConfig(3))
	if err != nil {
		t.Fatalf("Failed to create compressed document: %v", err)
	}
	defer compDoc.Close()

	// Test with empty document
	doc := document.NewDocument()
	stats, err := compDoc.GetCompressionStats(doc)
	if err != nil {
		t.Fatalf("Failed to get compression stats for empty document: %v", err)
	}

	if stats.OriginalSize <= 0 {
		t.Error("Original size should be positive even for empty document")
	}
}

// TestCompressedDocumentMultipleAlgorithms tests document compression with different algorithms
func TestCompressedDocumentMultipleAlgorithms(t *testing.T) {
	algorithms := []struct {
		name   string
		config *Config
	}{
		{"Snappy", SnappyConfig()},
		{"Zstd", ZstdConfig(3)},
		{"Gzip", GzipConfig(6)},
		{"Zlib", &Config{Algorithm: AlgorithmZlib, Level: 6}},
		{"None", &Config{Algorithm: AlgorithmNone}},
	}

	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())
	doc.Set("data", "test data that should compress")

	for _, algo := range algorithms {
		t.Run(algo.name, func(t *testing.T) {
			compDoc, err := NewCompressedDocument(algo.config)
			if err != nil {
				t.Fatalf("Failed to create compressed document with %s: %v", algo.name, err)
			}
			defer compDoc.Close()

			compressed, err := compDoc.Encode(doc)
			if err != nil {
				t.Fatalf("Failed to encode with %s: %v", algo.name, err)
			}

			decoded, err := compDoc.Decode(compressed)
			if err != nil {
				t.Fatalf("Failed to decode with %s: %v", algo.name, err)
			}

			if val, ok := decoded.Get("data"); !ok || val != "test data that should compress" {
				t.Errorf("Data mismatch with %s algorithm", algo.name)
			}
		})
	}
}
