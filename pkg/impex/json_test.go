package impex

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/document"
)

func TestJSONExporter_Export(t *testing.T) {
	// Create test documents
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":    document.NewObjectID(),
			"name":   "Alice",
			"age":    int64(30),
			"active": true,
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":   document.NewObjectID(),
			"name":  "Bob",
			"age":   int64(25),
			"score": 95.5,
		}),
	}

	tests := []struct {
		name   string
		pretty bool
	}{
		{"compact JSON", false},
		{"pretty JSON", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := NewJSONExporter(tt.pretty)
			var buf bytes.Buffer

			err := exporter.Export(&buf, docs)
			if err != nil {
				t.Fatalf("Export failed: %v", err)
			}

			output := buf.String()
			if output == "" {
				t.Error("Export produced empty output")
			}

			// Verify JSON contains expected data
			if !strings.Contains(output, "Alice") {
				t.Error("Export missing 'Alice'")
			}
			if !strings.Contains(output, "Bob") {
				t.Error("Export missing 'Bob'")
			}
		})
	}
}

func TestJSONExporter_ExportComplexTypes(t *testing.T) {
	now := time.Now()
	oid := document.NewObjectID()

	doc := document.NewDocumentFromMap(map[string]interface{}{
		"_id":       oid,
		"timestamp": now,
		"tags":      []interface{}{"go", "database", "nosql"},
		"metadata": map[string]interface{}{
			"version": int64(1),
			"author":  "tester",
		},
	})

	exporter := NewJSONExporter(true)
	var buf bytes.Buffer

	err := exporter.Export(&buf, []*document.Document{doc})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()

	// Verify ObjectID is converted to hex
	if !strings.Contains(output, oid.Hex()) {
		t.Error("ObjectID not converted to hex")
	}

	// Verify timestamp is converted to RFC3339
	if !strings.Contains(output, now.Format(time.RFC3339)) {
		t.Error("Timestamp not converted to RFC3339")
	}

	// Verify array is present
	if !strings.Contains(output, "go") || !strings.Contains(output, "database") {
		t.Error("Array elements missing")
	}

	// Verify nested object is present
	if !strings.Contains(output, "version") || !strings.Contains(output, "author") {
		t.Error("Nested object fields missing")
	}
}

func TestJSONImporter_Import(t *testing.T) {
	jsonData := `[
		{
			"_id": "507f1f77bcf86cd799439011",
			"name": "Alice",
			"age": 30,
			"active": true
		},
		{
			"_id": "507f1f77bcf86cd799439012",
			"name": "Bob",
			"age": 25,
			"score": 95.5
		}
	]`

	importer := NewJSONImporter()
	reader := strings.NewReader(jsonData)

	docs, err := importer.Import(reader)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if len(docs) != 2 {
		t.Fatalf("Expected 2 documents, got %d", len(docs))
	}

	// Verify first document
	name, exists := docs[0].Get("name")
	if !exists || name != "Alice" {
		t.Errorf("Expected name 'Alice', got %v", name)
	}

	age, exists := docs[0].Get("age")
	if !exists || age != int64(30) {
		t.Errorf("Expected age 30, got %v", age)
	}

	active, exists := docs[0].Get("active")
	if !exists || active != true {
		t.Errorf("Expected active true, got %v", active)
	}

	// Verify second document
	score, exists := docs[1].Get("score")
	if !exists || score != 95.5 {
		t.Errorf("Expected score 95.5, got %v", score)
	}
}

func TestJSONImporter_ImportComplexTypes(t *testing.T) {
	timestamp := time.Now().Format(time.RFC3339)
	jsonData := `[
		{
			"_id": "507f1f77bcf86cd799439011",
			"timestamp": "` + timestamp + `",
			"tags": ["go", "database", "nosql"],
			"metadata": {
				"version": 1,
				"author": "tester"
			}
		}
	]`

	importer := NewJSONImporter()
	reader := strings.NewReader(jsonData)

	docs, err := importer.Import(reader)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(docs))
	}

	doc := docs[0]

	// Verify ObjectID
	id, exists := doc.Get("_id")
	if !exists {
		t.Error("Missing _id field")
	}
	if _, ok := id.(document.ObjectID); !ok {
		t.Errorf("Expected ObjectID type, got %T", id)
	}

	// Verify timestamp
	ts, exists := doc.Get("timestamp")
	if !exists {
		t.Error("Missing timestamp field")
	}
	if _, ok := ts.(time.Time); !ok {
		t.Errorf("Expected time.Time type, got %T", ts)
	}

	// Verify array
	tags, exists := doc.Get("tags")
	if !exists {
		t.Error("Missing tags field")
	}
	tagsArr, ok := tags.([]interface{})
	if !ok || len(tagsArr) != 3 {
		t.Errorf("Expected array of 3 elements, got %v", tags)
	}

	// Verify nested object
	metadata, exists := doc.Get("metadata")
	if !exists {
		t.Error("Missing metadata field")
	}
	metaMap, ok := metadata.(map[string]interface{})
	if !ok {
		t.Errorf("Expected map type, got %T", metadata)
	}
	if metaMap["version"] != int64(1) {
		t.Errorf("Expected version 1, got %v", metaMap["version"])
	}
}

func TestJSONRoundTrip(t *testing.T) {
	// Create original documents
	originalDocs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":    document.NewObjectID(),
			"name":   "Alice",
			"age":    int64(30),
			"active": true,
			"tags":   []interface{}{"admin", "user"},
			"profile": map[string]interface{}{
				"bio":      "Software Engineer",
				"location": "NYC",
			},
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":   document.NewObjectID(),
			"name":  "Bob",
			"age":   int64(25),
			"score": 95.5,
		}),
	}

	// Export to JSON
	exporter := NewJSONExporter(false)
	var buf bytes.Buffer
	err := exporter.Export(&buf, originalDocs)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import from JSON
	importer := NewJSONImporter()
	importedDocs, err := importer.Import(&buf)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify document count
	if len(importedDocs) != len(originalDocs) {
		t.Fatalf("Expected %d documents, got %d", len(originalDocs), len(importedDocs))
	}

	// Verify first document fields
	for i, original := range originalDocs {
		imported := importedDocs[i]

		// Check each field
		origMap := original.ToMap()
		for key := range origMap {
			origVal, _ := original.Get(key)
			impVal, exists := imported.Get(key)

			if !exists {
				t.Errorf("Document %d missing field %s", i, key)
				continue
			}

			// Special handling for ObjectID (hex string comparison)
			if oid, ok := origVal.(document.ObjectID); ok {
				if impOid, ok := impVal.(document.ObjectID); ok {
					if oid.Hex() != impOid.Hex() {
						t.Errorf("Document %d field %s: ObjectID mismatch", i, key)
					}
				} else {
					t.Errorf("Document %d field %s: expected ObjectID, got %T", i, key, impVal)
				}
				continue
			}

			// Note: Deep comparison of complex types would require more sophisticated logic
			// For now, we verify the types match
			if origVal != nil && impVal != nil {
				// Type checking for complex types
				switch origVal.(type) {
				case []interface{}:
					if _, ok := impVal.([]interface{}); !ok {
						t.Errorf("Document %d field %s: type mismatch", i, key)
					}
				case map[string]interface{}:
					if _, ok := impVal.(map[string]interface{}); !ok {
						t.Errorf("Document %d field %s: type mismatch", i, key)
					}
				}
			}
		}
	}
}

func TestJSONExporter_EmptyDocuments(t *testing.T) {
	exporter := NewJSONExporter(false)
	var buf bytes.Buffer

	err := exporter.Export(&buf, []*document.Document{})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Should produce an empty array
	output := strings.TrimSpace(buf.String())
	if output != "[]" {
		t.Errorf("Expected empty array, got: %s", output)
	}
}

func TestJSONImporter_InvalidJSON(t *testing.T) {
	invalidJSON := `{ this is not valid JSON }`

	importer := NewJSONImporter()
	reader := strings.NewReader(invalidJSON)

	_, err := importer.Import(reader)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}
