package impex

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/document"
)

func TestCSVExporter_Export(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "1",
			"name": "Alice",
			"age":  int64(30),
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "2",
			"name": "Bob",
			"age":  int64(25),
		}),
	}

	exporter := NewCSVExporter([]string{"_id", "name", "age"})
	var buf bytes.Buffer

	err := exporter.Export(&buf, docs)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Verify header
	if lines[0] != "_id,name,age" {
		t.Errorf("Expected header '_id,name,age', got '%s'", lines[0])
	}

	// Verify first row
	if lines[1] != "1,Alice,30" {
		t.Errorf("Expected '1,Alice,30', got '%s'", lines[1])
	}

	// Verify second row
	if lines[2] != "2,Bob,25" {
		t.Errorf("Expected '2,Bob,25', got '%s'", lines[2])
	}
}

func TestCSVExporter_AutoDetectFields(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "1",
			"name": "Alice",
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id": "2",
			"name": "Bob",
			"age":  int64(25),
		}),
	}

	exporter := NewCSVExporter(nil) // Auto-detect fields
	var buf bytes.Buffer

	err := exporter.Export(&buf, docs)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Verify header contains all fields (_id should be first)
	header := lines[0]
	if !strings.HasPrefix(header, "_id") {
		t.Errorf("Expected header to start with '_id', got '%s'", header)
	}
	if !strings.Contains(header, "name") {
		t.Error("Header missing 'name'")
	}
	if !strings.Contains(header, "age") {
		t.Error("Header missing 'age'")
	}
}

func TestCSVExporter_ComplexTypes(t *testing.T) {
	now := time.Now()
	oid := document.NewObjectID()

	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":       oid,
			"timestamp": now,
			"tags":      []interface{}{"go", "database"},
			"metadata": map[string]interface{}{
				"version": int64(1),
			},
		}),
	}

	exporter := NewCSVExporter([]string{"_id", "timestamp", "tags", "metadata"})
	var buf bytes.Buffer

	err := exporter.Export(&buf, docs)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Verify ObjectID is hex
	if !strings.Contains(lines[1], oid.Hex()) {
		t.Error("ObjectID not converted to hex")
	}

	// Verify timestamp is RFC3339
	if !strings.Contains(lines[1], now.Format(time.RFC3339)) {
		t.Error("Timestamp not in RFC3339 format")
	}

	// Verify array is JSON encoded
	if !strings.Contains(lines[1], "[") || !strings.Contains(lines[1], "]") {
		t.Error("Array not JSON encoded")
	}
}

func TestCSVExporter_MissingFields(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "1",
			"name": "Alice",
			"age":  int64(30),
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "2",
			"name": "Bob",
			// Missing 'age' field
		}),
	}

	exporter := NewCSVExporter([]string{"_id", "name", "age"})
	var buf bytes.Buffer

	err := exporter.Export(&buf, docs)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Second row should have empty age field
	if lines[2] != "2,Bob," {
		t.Errorf("Expected '2,Bob,', got '%s'", lines[2])
	}
}

func TestCSVImporter_Import(t *testing.T) {
	csvData := `_id,name,age,active
1,Alice,30,true
2,Bob,25,false`

	importer := NewCSVImporter(nil)
	reader := strings.NewReader(csvData)

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
	active2, exists := docs[1].Get("active")
	if !exists || active2 != false {
		t.Errorf("Expected active false, got %v", active2)
	}
}

func TestCSVImporter_WithProvidedHeaders(t *testing.T) {
	csvData := `1,Alice,30
2,Bob,25`

	importer := NewCSVImporter([]string{"_id", "name", "age"})
	reader := strings.NewReader(csvData)

	docs, err := importer.Import(reader)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if len(docs) != 2 {
		t.Fatalf("Expected 2 documents, got %d", len(docs))
	}

	// Verify fields are correctly mapped
	name, exists := docs[0].Get("name")
	if !exists || name != "Alice" {
		t.Errorf("Expected name 'Alice', got %v", name)
	}
}

func TestCSVImporter_TypeParsing(t *testing.T) {
	csvData := `id,count,score,active,note
1,100,95.5,true,hello`

	importer := NewCSVImporter(nil)
	reader := strings.NewReader(csvData)

	docs, err := importer.Import(reader)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	doc := docs[0]

	// Verify int64
	count, _ := doc.Get("count")
	if count != int64(100) {
		t.Errorf("Expected int64(100), got %v (%T)", count, count)
	}

	// Verify float64
	score, _ := doc.Get("score")
	if score != 95.5 {
		t.Errorf("Expected 95.5, got %v (%T)", score, score)
	}

	// Verify bool
	active, _ := doc.Get("active")
	if active != true {
		t.Errorf("Expected true, got %v (%T)", active, active)
	}

	// Verify string
	note, _ := doc.Get("note")
	if note != "hello" {
		t.Errorf("Expected 'hello', got %v (%T)", note, note)
	}
}

func TestCSVImporter_ComplexTypes(t *testing.T) {
	timestamp := time.Now().Format(time.RFC3339)
	csvData := `_id,timestamp,tags
507f1f77bcf86cd799439011,` + timestamp + `,"[""go"",""database""]"`

	importer := NewCSVImporter(nil)
	reader := strings.NewReader(csvData)

	docs, err := importer.Import(reader)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
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

	// Verify array (JSON parsed)
	tags, exists := doc.Get("tags")
	if !exists {
		t.Error("Missing tags field")
	}
	if _, ok := tags.([]interface{}); !ok {
		t.Errorf("Expected array type, got %T", tags)
	}
}

func TestCSVImporter_EmptyFields(t *testing.T) {
	csvData := `_id,name,age
1,Alice,30
2,Bob,
3,,25`

	importer := NewCSVImporter(nil)
	reader := strings.NewReader(csvData)

	docs, err := importer.Import(reader)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Second document should not have 'age' field
	if _, exists := docs[1].Get("age"); exists {
		t.Error("Expected age field to be missing for Bob")
	}

	// Third document should not have 'name' field
	if _, exists := docs[2].Get("name"); exists {
		t.Error("Expected name field to be missing for document 3")
	}
}

func TestCSVRoundTrip(t *testing.T) {
	// Create original documents
	originalDocs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":    "1",
			"name":   "Alice",
			"age":    int64(30),
			"active": true,
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":   "2",
			"name":  "Bob",
			"age":   int64(25),
			"score": 95.5,
		}),
	}

	fields := []string{"_id", "name", "age", "score", "active"}

	// Export to CSV
	exporter := NewCSVExporter(fields)
	var buf bytes.Buffer
	err := exporter.Export(&buf, originalDocs)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import from CSV
	importer := NewCSVImporter(nil)
	importedDocs, err := importer.Import(&buf)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify document count
	if len(importedDocs) != len(originalDocs) {
		t.Fatalf("Expected %d documents, got %d", len(originalDocs), len(importedDocs))
	}

	// Verify key fields are preserved
	name1, _ := importedDocs[0].Get("name")
	if name1 != "Alice" {
		t.Errorf("Expected name 'Alice', got %v", name1)
	}

	age1, _ := importedDocs[0].Get("age")
	if age1 != int64(30) {
		t.Errorf("Expected age 30, got %v", age1)
	}

	active1, _ := importedDocs[0].Get("active")
	if active1 != true {
		t.Errorf("Expected active true, got %v", active1)
	}
}

func TestCSVExporter_EmptyDocuments(t *testing.T) {
	exporter := NewCSVExporter([]string{"_id", "name"})
	var buf bytes.Buffer

	err := exporter.Export(&buf, []*document.Document{})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Should produce no output
	output := buf.String()
	if output != "" {
		t.Errorf("Expected empty output, got: %s", output)
	}
}

func TestCSVExporter_SpecialCharacters(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "1",
			"text": "Hello, World!",
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "2",
			"text": `Line 1
Line 2`,
		}),
	}

	exporter := NewCSVExporter([]string{"_id", "text"})
	var buf bytes.Buffer

	err := exporter.Export(&buf, docs)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// CSV library should handle special characters (commas, newlines, quotes)
	output := buf.String()
	if !strings.Contains(output, "Hello, World!") {
		t.Error("Missing text with comma")
	}
}

func TestCSVImporter_ConvertJSONValue(t *testing.T) {
	tests := []struct {
		name     string
		csvData  string
		field    string
		expected interface{}
		checkType func(interface{}) bool
	}{
		{
			name:    "JSONArray",
			csvData: `data
"[1,2,3]"`,
			field: "data",
			checkType: func(v interface{}) bool {
				arr, ok := v.([]interface{})
				if !ok {
					return false
				}
				// Check that array elements are converted to int64
				for _, elem := range arr {
					if _, ok := elem.(int64); !ok {
						return false
					}
				}
				return true
			},
		},
		{
			name:    "JSONObject",
			csvData: `data
"{""key"":42}"`,
			field: "data",
			checkType: func(v interface{}) bool {
				obj, ok := v.(map[string]interface{})
				if !ok {
					return false
				}
				// Check that nested number is converted to int64
				if val, ok := obj["key"]; ok {
					if _, ok := val.(int64); ok {
						return true
					}
				}
				return false
			},
		},
		{
			name:    "JSONNestedArray",
			csvData: `data
"[[1,2],[3,4]]"`,
			field: "data",
			checkType: func(v interface{}) bool {
				arr, ok := v.([]interface{})
				if !ok {
					return false
				}
				// Check nested arrays
				for _, elem := range arr {
					if _, ok := elem.([]interface{}); !ok {
						return false
					}
				}
				return true
			},
		},
		{
			name:    "JSONNestedObject",
			csvData: `data
"{""nested"":{""value"":10.5}}"`,
			field: "data",
			checkType: func(v interface{}) bool {
				obj, ok := v.(map[string]interface{})
				if !ok {
					return false
				}
				nested, ok := obj["nested"].(map[string]interface{})
				if !ok {
					return false
				}
				// Float value should remain float64 (not whole number)
				if val, ok := nested["value"].(float64); ok {
					return val == 10.5
				}
				return false
			},
		},
		{
			name:    "JSONWholeNumberAsInt64",
			csvData: `data
"[42.0,43.0]"`,
			field: "data",
			checkType: func(v interface{}) bool {
				arr, ok := v.([]interface{})
				if !ok {
					return false
				}
				// Whole numbers should be converted to int64
				for _, elem := range arr {
					if _, ok := elem.(int64); !ok {
						return false
					}
				}
				return true
			},
		},
		{
			name:    "JSONFloatRemainsfloat64",
			csvData: `data
"[42.5,43.5]"`,
			field: "data",
			checkType: func(v interface{}) bool {
				arr, ok := v.([]interface{})
				if !ok {
					return false
				}
				// Non-whole numbers should remain float64
				for _, elem := range arr {
					if _, ok := elem.(float64); !ok {
						return false
					}
				}
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importer := NewCSVImporter(nil)
			docs, err := importer.Import(strings.NewReader(tt.csvData))
			if err != nil {
				t.Fatalf("Import failed: %v", err)
			}
			if len(docs) != 1 {
				t.Fatalf("Expected 1 document, got %d", len(docs))
			}

			value, exists := docs[0].Get(tt.field)
			if !exists {
				t.Fatalf("Field %s not found", tt.field)
			}

			if !tt.checkType(value) {
				t.Errorf("Type check failed for value: %v (%T)", value, value)
			}
		})
	}
}

func TestCSVExporter_FormatValueAllTypes(t *testing.T) {
	now := time.Now()
	oid := document.NewObjectID()

	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"nil_val":    nil,
			"string_val": "test",
			"int_val":    int64(42),
			"float_val":  3.14,
			"bool_val":   true,
			"oid_val":    oid,
			"time_val":   now,
			"array_val":  []interface{}{1, 2, 3},
			"map_val": map[string]interface{}{
				"key": "value",
			},
		}),
	}

	exporter := NewCSVExporter([]string{
		"nil_val", "string_val", "int_val", "float_val", "bool_val",
		"oid_val", "time_val", "array_val", "map_val",
	})
	var buf bytes.Buffer

	err := exporter.Export(&buf, docs)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}

	// Verify header
	expectedHeader := "nil_val,string_val,int_val,float_val,bool_val,oid_val,time_val,array_val,map_val"
	if lines[0] != expectedHeader {
		t.Errorf("Header mismatch.\nExpected: %s\nGot:      %s", expectedHeader, lines[0])
	}

	// Verify data row contains expected values
	dataRow := lines[1]
	if !strings.Contains(dataRow, "test") {
		t.Error("Missing string value")
	}
	if !strings.Contains(dataRow, "42") {
		t.Error("Missing int value")
	}
	if !strings.Contains(dataRow, "3.14") {
		t.Error("Missing float value")
	}
	if !strings.Contains(dataRow, "true") {
		t.Error("Missing bool value")
	}
	if !strings.Contains(dataRow, oid.Hex()) {
		t.Error("Missing ObjectID hex value")
	}
	if !strings.Contains(dataRow, now.Format(time.RFC3339)) {
		t.Error("Missing timestamp value")
	}
}

func TestCSVImporter_ParseRowEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		csvData     string
		expectError bool
		checkFunc   func([]*document.Document) error
	}{
		{
			name:        "InvalidObjectID",
			csvData:     "id\ninvalid_oid",
			expectError: false,
			checkFunc: func(docs []*document.Document) error {
				val, _ := docs[0].Get("id")
				// Should remain as string if invalid ObjectID
				if _, ok := val.(string); !ok {
					return bytes.ErrTooLarge // Using a random error for test
				}
				return nil
			},
		},
		{
			name:        "InvalidTimestamp",
			csvData:     "timestamp\nnot_a_timestamp",
			expectError: false,
			checkFunc: func(docs []*document.Document) error {
				val, _ := docs[0].Get("timestamp")
				// Should remain as string if invalid timestamp
				if _, ok := val.(string); !ok {
					return bytes.ErrTooLarge
				}
				return nil
			},
		},
		{
			name:        "InvalidJSON",
			csvData:     "data\n\"{invalid json}\"",
			expectError: false,
			checkFunc: func(docs []*document.Document) error {
				val, _ := docs[0].Get("data")
				// Should remain as string if invalid JSON
				if _, ok := val.(string); !ok {
					return bytes.ErrTooLarge
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importer := NewCSVImporter(nil)
			docs, err := importer.Import(strings.NewReader(tt.csvData))
			if (err != nil) != tt.expectError {
				t.Fatalf("Expected error: %v, got: %v", tt.expectError, err)
			}
			if !tt.expectError && tt.checkFunc != nil {
				if err := tt.checkFunc(docs); err != nil {
					t.Error("Check function failed")
				}
			}
		})
	}
}

