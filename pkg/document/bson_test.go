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
