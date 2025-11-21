package document

import (
	"testing"
)

func TestNewDocument(t *testing.T) {
	doc := NewDocument()
	if doc == nil {
		t.Fatal("NewDocument returned nil")
	}
	if doc.Len() != 0 {
		t.Errorf("Expected empty document, got length %d", doc.Len())
	}
}

func TestDocumentSetGet(t *testing.T) {
	doc := NewDocument()

	// Test string
	doc.Set("name", "Alice")
	val, exists := doc.Get("name")
	if !exists {
		t.Error("Expected name field to exist")
	}
	if val.(string) != "Alice" {
		t.Errorf("Expected 'Alice', got %v", val)
	}

	// Test int64
	doc.Set("age", int64(30))
	val, exists = doc.Get("age")
	if !exists {
		t.Error("Expected age field to exist")
	}
	if val.(int64) != 30 {
		t.Errorf("Expected 30, got %v", val)
	}

	// Test boolean
	doc.Set("active", true)
	val, exists = doc.Get("active")
	if !exists {
		t.Error("Expected active field to exist")
	}
	if val.(bool) != true {
		t.Errorf("Expected true, got %v", val)
	}
}

func TestDocumentDelete(t *testing.T) {
	doc := NewDocument()
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))

	if doc.Len() != 2 {
		t.Errorf("Expected 2 fields, got %d", doc.Len())
	}

	doc.Delete("name")
	if doc.Len() != 1 {
		t.Errorf("Expected 1 field after delete, got %d", doc.Len())
	}

	_, exists := doc.Get("name")
	if exists {
		t.Error("Expected name field to be deleted")
	}
}

func TestDocumentClone(t *testing.T) {
	doc := NewDocument()
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))

	clone := doc.Clone()
	if clone.Len() != doc.Len() {
		t.Errorf("Expected clone to have %d fields, got %d", doc.Len(), clone.Len())
	}

	// Modify clone
	clone.Set("age", int64(31))

	// Original should be unchanged
	val, _ := doc.Get("age")
	if val.(int64) != 30 {
		t.Error("Original document was modified")
	}

	// Clone should be changed
	val, _ = clone.Get("age")
	if val.(int64) != 31 {
		t.Error("Clone was not modified")
	}
}

func TestDocumentToMap(t *testing.T) {
	doc := NewDocument()
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))

	m := doc.ToMap()
	if len(m) != 2 {
		t.Errorf("Expected map with 2 entries, got %d", len(m))
	}

	if m["name"].(string) != "Alice" {
		t.Errorf("Expected 'Alice', got %v", m["name"])
	}
}

func TestNewDocumentFromMap(t *testing.T) {
	m := map[string]interface{}{
		"name":  "Bob",
		"age":   int64(25),
		"email": "bob@example.com",
	}

	doc := NewDocumentFromMap(m)
	if doc.Len() != 3 {
		t.Errorf("Expected 3 fields, got %d", doc.Len())
	}

	val, exists := doc.Get("name")
	if !exists || val.(string) != "Bob" {
		t.Error("Expected name field with value 'Bob'")
	}
}

func TestDocumentNestedStructures(t *testing.T) {
	doc := NewDocument()

	// Nested document
	nested := map[string]interface{}{
		"city":    "New York",
		"country": "USA",
	}
	doc.Set("address", nested)

	val, exists := doc.Get("address")
	if !exists {
		t.Fatal("Expected address field to exist")
	}

	addr := val.(map[string]interface{})
	if addr["city"].(string) != "New York" {
		t.Error("Expected nested city to be 'New York'")
	}

	// Array
	tags := []interface{}{"admin", "developer"}
	doc.Set("tags", tags)

	val, exists = doc.Get("tags")
	if !exists {
		t.Fatal("Expected tags field to exist")
	}

	tagArray := val.([]interface{})
	if len(tagArray) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tagArray))
	}
}
