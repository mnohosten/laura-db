package document

import (
	"testing"
)

func TestBSONEncodeDecode(t *testing.T) {
	doc := NewDocument()
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))
	doc.Set("active", true)

	// Encode
	encoder := NewEncoder()
	data, err := encoder.Encode(doc)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty encoded data")
	}

	// Decode
	decoder := NewDecoder(data)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify fields
	val, exists := decoded.Get("name")
	if !exists || val.(string) != "Alice" {
		t.Error("Name field not correctly decoded")
	}

	val, exists = decoded.Get("age")
	if !exists || val.(int64) != 30 {
		t.Error("Age field not correctly decoded")
	}

	val, exists = decoded.Get("active")
	if !exists || val.(bool) != true {
		t.Error("Active field not correctly decoded")
	}
}

func TestBSONEncodeDecodeAllTypes(t *testing.T) {
	doc := NewDocument()
	doc.Set("null", nil)
	doc.Set("bool", true)
	doc.Set("int32", int32(42))
	doc.Set("int64", int64(100))
	doc.Set("float", 3.14)
	doc.Set("string", "hello")
	doc.Set("binary", []byte{0x01, 0x02, 0x03})

	encoder := NewEncoder()
	data, err := encoder.Encode(doc)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoder := NewDecoder(data)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify each type
	val, exists := decoded.Get("bool")
	if !exists || val.(bool) != true {
		t.Error("Bool not correctly encoded/decoded")
	}

	val, exists = decoded.Get("int32")
	if !exists || val.(int32) != 42 {
		t.Error("Int32 not correctly encoded/decoded")
	}

	val, exists = decoded.Get("string")
	if !exists || val.(string) != "hello" {
		t.Error("String not correctly encoded/decoded")
	}
}

func TestBSONEncodeDecodeNested(t *testing.T) {
	doc := NewDocument()

	nested := NewDocument()
	nested.Set("city", "New York")
	nested.Set("zip", "10001")

	doc.Set("name", "Alice")
	doc.Set("address", nested)

	encoder := NewEncoder()
	data, err := encoder.Encode(doc)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoder := NewDecoder(data)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	val, exists := decoded.Get("address")
	if !exists {
		t.Fatal("Address field not found")
	}

	addr := val.(*Document)
	city, exists := addr.Get("city")
	if !exists || city.(string) != "New York" {
		t.Error("Nested document not correctly encoded/decoded")
	}
}

func TestBSONEncodeDecodeArray(t *testing.T) {
	doc := NewDocument()
	doc.Set("tags", []interface{}{"admin", "user", "developer"})

	encoder := NewEncoder()
	data, err := encoder.Encode(doc)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoder := NewDecoder(data)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	val, exists := decoded.Get("tags")
	if !exists {
		t.Fatal("Tags field not found")
	}

	tags := val.([]interface{})
	if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	}

	if tags[0].(string) != "admin" {
		t.Error("Array element not correctly decoded")
	}
}

// Test BSON encoding with ObjectID
func TestBSONEncodeDecodeObjectID(t *testing.T) {
	doc := NewDocument()
	id := NewObjectID()
	doc.Set("_id", id)
	doc.Set("name", "Test")

	encoder := NewEncoder()
	data, err := encoder.Encode(doc)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoder := NewDecoder(data)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	val, exists := decoded.Get("_id")
	if !exists {
		t.Fatal("_id field not found")
	}

	decodedID := val.(ObjectID)
	if decodedID.String() != id.String() {
		t.Error("ObjectID not correctly encoded/decoded")
	}
}

// Test BSON with mixed array types
func TestBSONEncodeDecodeMixedArray(t *testing.T) {
	doc := NewDocument()
	doc.Set("mixed", []interface{}{
		int64(42),
		"string",
		true,
		3.14,
		[]byte{0x01, 0x02},
	})

	encoder := NewEncoder()
	data, err := encoder.Encode(doc)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoder := NewDecoder(data)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	val, exists := decoded.Get("mixed")
	if !exists {
		t.Fatal("Mixed array not found")
	}

	arr := val.([]interface{})
	if len(arr) != 5 {
		t.Errorf("Expected 5 elements, got %d", len(arr))
	}

	// Verify types
	if arr[0].(int64) != 42 {
		t.Error("First element (int64) incorrect")
	}
	if arr[1].(string) != "string" {
		t.Error("Second element (string) incorrect")
	}
	if arr[2].(bool) != true {
		t.Error("Third element (bool) incorrect")
	}
}

// Test BSON with nested arrays
func TestBSONEncodeDecodeNestedArray(t *testing.T) {
	doc := NewDocument()
	doc.Set("nested", []interface{}{
		[]interface{}{"a", "b"},
		[]interface{}{int64(1), int64(2)},
	})

	encoder := NewEncoder()
	data, err := encoder.Encode(doc)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoder := NewDecoder(data)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	val, exists := decoded.Get("nested")
	if !exists {
		t.Fatal("Nested array not found")
	}

	outer := val.([]interface{})
	if len(outer) != 2 {
		t.Errorf("Expected 2 outer elements, got %d", len(outer))
	}

	// Check first nested array
	inner1 := outer[0].([]interface{})
	if len(inner1) != 2 || inner1[0].(string) != "a" {
		t.Error("First nested array incorrect")
	}
}

// Test BSON with deeply nested documents
func TestBSONEncodeDeeplyNested(t *testing.T) {
	doc := NewDocument()

	level1 := NewDocument()
	level2 := NewDocument()
	level3 := NewDocument()

	level3.Set("value", "deep")
	level2.Set("level3", level3)
	level1.Set("level2", level2)
	doc.Set("level1", level1)

	encoder := NewEncoder()
	data, err := encoder.Encode(doc)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoder := NewDecoder(data)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Navigate through nested documents
	l1, _ := decoded.Get("level1")
	l2, _ := l1.(*Document).Get("level2")
	l3, _ := l2.(*Document).Get("level3")
	val, _ := l3.(*Document).Get("value")

	if val.(string) != "deep" {
		t.Error("Deeply nested value not correctly encoded/decoded")
	}
}

// Test BSON empty document
func TestBSONEncodeDecodeEmpty(t *testing.T) {
	doc := NewDocument()

	encoder := NewEncoder()
	data, err := encoder.Encode(doc)
	if err != nil {
		t.Fatalf("Encode empty document failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty data for empty document")
	}

	decoder := NewDecoder(data)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode empty document failed: %v", err)
	}

	if decoded.Len() != 0 {
		t.Errorf("Expected empty document, got %d fields", decoded.Len())
	}
}
