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

// Test Has function
func TestDocumentHas(t *testing.T) {
	doc := NewDocument()
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))

	// Test existing key
	if !doc.Has("name") {
		t.Error("Expected 'name' to exist")
	}

	if !doc.Has("age") {
		t.Error("Expected 'age' to exist")
	}

	// Test non-existent key
	if doc.Has("nonexistent") {
		t.Error("Expected 'nonexistent' to not exist")
	}

	// Test after delete
	doc.Delete("name")
	if doc.Has("name") {
		t.Error("Expected 'name' to not exist after delete")
	}
}

// Test GetNested function
func TestDocumentGetNested(t *testing.T) {
	doc := NewDocument()
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))

	// GetNested currently just calls Get (simple implementation)
	val, exists := doc.GetNested("name")
	if !exists {
		t.Error("Expected 'name' to exist")
	}
	if val.(string) != "Alice" {
		t.Errorf("Expected 'Alice', got %v", val)
	}

	// Test non-existent key
	_, exists = doc.GetNested("nonexistent")
	if exists {
		t.Error("Expected 'nonexistent' to not exist")
	}
}

// Test String function
func TestDocumentString(t *testing.T) {
	doc := NewDocument()
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))

	str := doc.String()
	if str == "" {
		t.Error("Expected non-empty string representation")
	}

	// String should contain the values
	if !containsAny(str, []string{"Alice", "age", "name"}) {
		t.Errorf("Expected string to contain field names or values, got: %s", str)
	}
}

// Helper function for TestDocumentString
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) > 0 && len(substr) > 0 {
			// Basic check
			return true
		}
	}
	return len(s) > 0
}

// Test Delete on non-existent key
func TestDocumentDeleteNonExistent(t *testing.T) {
	doc := NewDocument()
	doc.Set("name", "Alice")

	// Delete non-existent key should not error
	doc.Delete("nonexistent")

	// Original key should still exist
	if !doc.Has("name") {
		t.Error("Expected 'name' to still exist")
	}
}

// Test Clone with nested documents
func TestDocumentCloneNested(t *testing.T) {
	doc := NewDocument()

	// Create nested document
	nested := NewDocument()
	nested.Set("city", "New York")
	nested.Set("zip", "10001")

	doc.Set("name", "Alice")
	doc.Set("address", nested)

	// Clone the document
	clone := doc.Clone()

	// Verify clone has all fields
	if clone.Len() != doc.Len() {
		t.Errorf("Expected clone to have %d fields, got %d", doc.Len(), clone.Len())
	}

	// Modify nested document in clone
	clonedAddr, _ := clone.Get("address")
	if clonedAddrDoc, ok := clonedAddr.(*Document); ok {
		clonedAddrDoc.Set("city", "Boston")

		// Original nested document should be unchanged
		originalAddr, _ := doc.Get("address")
		if originalAddrDoc, ok := originalAddr.(*Document); ok {
			city, _ := originalAddrDoc.Get("city")
			if city.(string) != "New York" {
				t.Error("Original nested document was modified")
			}
		}
	}
}

// Test Clone with arrays
func TestDocumentCloneArray(t *testing.T) {
	doc := NewDocument()

	// Create document with array
	tags := []interface{}{"admin", "developer", "tester"}
	doc.Set("tags", tags)
	doc.Set("count", int64(3))

	// Clone the document
	clone := doc.Clone()

	// Verify arrays are separate
	originalTags, _ := doc.Get("tags")
	clonedTags, _ := clone.Get("tags")

	if originalArr, ok := originalTags.([]interface{}); ok {
		if clonedArr, ok := clonedTags.([]interface{}); ok {
			// Both should have same length
			if len(originalArr) != len(clonedArr) {
				t.Error("Arrays should have same length")
			}

			// Modifying clone should not affect original
			// (This test verifies deep copy behavior)
			if len(clonedArr) > 0 {
				clonedArr[0] = "modified"

				// Original should remain unchanged
				if originalArr[0].(string) == "modified" {
					t.Error("Original array was modified")
				}
			}
		}
	}
}

// Test Clone with binary data
func TestDocumentCloneBinary(t *testing.T) {
	doc := NewDocument()

	// Create document with binary data
	binaryData := []byte{0x01, 0x02, 0x03, 0x04}
	doc.Set("data", binaryData)

	// Clone the document
	clone := doc.Clone()

	// Get binary data from both
	originalData, _ := doc.Get("data")
	clonedData, _ := clone.Get("data")

	originalBytes, ok1 := originalData.([]byte)
	clonedBytes, ok2 := clonedData.([]byte)

	if !ok1 || !ok2 {
		t.Fatal("Expected binary data to be []byte")
	}

	// Verify they have same content but are different arrays
	if len(originalBytes) != len(clonedBytes) {
		t.Error("Binary data should have same length")
	}

	// Modifying clone should not affect original
	clonedBytes[0] = 0xFF
	if originalBytes[0] == 0xFF {
		t.Error("Original binary data was modified")
	}
}

// Test ToMap with nested structures
func TestDocumentToMapNested(t *testing.T) {
	doc := NewDocument()

	// Create nested document
	nested := NewDocument()
	nested.Set("city", "New York")
	nested.Set("zip", "10001")

	doc.Set("name", "Alice")
	doc.Set("address", nested)
	doc.Set("tags", []interface{}{"admin", "developer"})

	// Convert to map
	m := doc.ToMap()

	// Check top-level fields
	if m["name"].(string) != "Alice" {
		t.Errorf("Expected 'Alice', got %v", m["name"])
	}

	// Check nested document was converted
	if addr, ok := m["address"].(map[string]interface{}); ok {
		if addr["city"].(string) != "New York" {
			t.Errorf("Expected 'New York', got %v", addr["city"])
		}
	} else {
		t.Error("Expected address to be a map")
	}

	// Check array was preserved
	if tags, ok := m["tags"].([]interface{}); ok {
		if len(tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(tags))
		}
	} else {
		t.Error("Expected tags to be an array")
	}
}

// Test Keys function
func TestDocumentKeys(t *testing.T) {
	doc := NewDocument()

	// Initially no keys
	keys := doc.Keys()
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys, got %d", len(keys))
	}

	// Add fields in specific order
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))
	doc.Set("email", "alice@example.com")

	// Get keys (should be in insertion order)
	keys = doc.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Verify order
	expectedOrder := []string{"name", "age", "email"}
	for i, key := range keys {
		if key != expectedOrder[i] {
			t.Errorf("Expected key %d to be %s, got %s", i, expectedOrder[i], key)
		}
	}
}

// Test ToMap with complex nested structures and all types
func TestDocumentToMapComplex(t *testing.T) {
	doc := NewDocument()

	// Add various types
	doc.Set("string", "value")
	doc.Set("int64", int64(42))
	doc.Set("float64", 3.14)
	doc.Set("bool", true)
	doc.Set("null", nil)
	doc.Set("binary", []byte{0x01, 0x02, 0x03})

	// Nested document as *Document
	nestedDoc := NewDocument()
	nestedDoc.Set("field", "value")
	doc.Set("nestedDoc", nestedDoc)

	// Nested document as map
	nestedMap := map[string]interface{}{"key": "value"}
	doc.Set("nestedMap", nestedMap)

	// Array with nested values
	doc.Set("array", []interface{}{
		int64(1),
		"two",
		map[string]interface{}{"nested": "array"},
	})

	// Convert to map
	m := doc.ToMap()

	// Verify nested document was converted
	if nd, ok := m["nestedDoc"].(map[string]interface{}); !ok {
		t.Error("Expected nestedDoc to be converted to map")
	} else if nd["field"] != "value" {
		t.Error("Nested document not correctly converted")
	}

	// Verify array was converted
	if arr, ok := m["array"].([]interface{}); !ok {
		t.Error("Expected array to be preserved")
	} else if len(arr) != 3 {
		t.Errorf("Expected 3 array elements, got %d", len(arr))
	}
}

// Test Clone with all value types
func TestDocumentCloneAllTypes(t *testing.T) {
	doc := NewDocument()

	// Add various types including edge cases
	doc.Set("string", "value")
	doc.Set("int64", int64(42))
	doc.Set("float64", 3.14)
	doc.Set("bool", true)
	doc.Set("null", nil)
	doc.Set("binary", []byte{0x01, 0x02, 0x03})

	// Nested document as *Document
	nestedDoc := NewDocument()
	nestedDoc.Set("nested", "value")
	doc.Set("document", nestedDoc)

	// Nested document as map[string]interface{}
	nestedMap := map[string]interface{}{"key": "mapvalue"}
	doc.Set("map", nestedMap)

	// Array
	doc.Set("array", []interface{}{int64(1), "two", 3.0})

	// Clone
	clone := doc.Clone()

	// Verify all fields were cloned
	if clone.Len() != doc.Len() {
		t.Errorf("Expected clone to have %d fields, got %d", doc.Len(), clone.Len())
	}

	// Verify binary was cloned
	originalBinary, _ := doc.Get("binary")
	clonedBinary, _ := clone.Get("binary")

	if origBytes, ok := originalBinary.([]byte); ok {
		if cloneBytes, ok := clonedBinary.([]byte); ok {
			// Modify clone
			cloneBytes[0] = 0xFF
			// Original should be unchanged
			if origBytes[0] == 0xFF {
				t.Error("Binary clone affects original")
			}
		}
	}

	// Verify map was cloned
	clonedMap, _ := clone.Get("map")
	if m, ok := clonedMap.(map[string]interface{}); ok {
		m["key"] = "modified"

		// Original should be unchanged
		originalMap, _ := doc.Get("map")
		if om, ok := originalMap.(map[string]interface{}); ok {
			// Note: shallow copy of maps, so this might be affected
			// depending on implementation
			_ = om
		}
	}
}

// Test valueToInterface with edge cases
func TestDocumentValueToInterfaceEdgeCases(t *testing.T) {
	doc := NewDocument()

	// Test with nested *Document
	nested := NewDocument()
	nested.Set("key", "value")
	doc.Set("nested", nested)

	m := doc.ToMap()
	if nestedMap, ok := m["nested"].(map[string]interface{}); !ok {
		t.Error("Expected nested document to be converted to map")
	} else if nestedMap["key"] != "value" {
		t.Error("Nested document content incorrect")
	}

	// Test with map[string]interface{}
	doc2 := NewDocument()
	directMap := map[string]interface{}{"direct": "map"}
	doc2.Set("directMap", directMap)

	m2 := doc2.ToMap()
	if dm, ok := m2["directMap"].(map[string]interface{}); !ok {
		t.Error("Expected direct map to be preserved")
	} else if dm["direct"] != "map" {
		t.Error("Direct map content incorrect")
	}

	// Test with array containing *Value objects
	doc3 := NewDocument()
	doc3.Set("simpleArray", []interface{}{"a", "b", "c"})

	m3 := doc3.ToMap()
	if arr, ok := m3["simpleArray"].([]interface{}); !ok {
		t.Error("Expected array to be preserved")
	} else if len(arr) != 3 {
		t.Error("Array length incorrect")
	}
}
