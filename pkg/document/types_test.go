package document

import (
	"testing"
	"time"
)

// Test Type.String() method
func TestTypeString(t *testing.T) {
	tests := []struct {
		typ      Type
		expected string
	}{
		{TypeNull, "null"},
		{TypeBoolean, "boolean"},
		{TypeInt32, "int32"},
		{TypeInt64, "int64"},
		{TypeFloat64, "float64"},
		{TypeString, "string"},
		{TypeBinary, "binary"},
		{TypeObjectID, "objectid"},
		{TypeArray, "array"},
		{TypeDocument, "document"},
		{TypeTimestamp, "timestamp"},
		{Type(0xFF), "unknown"}, // Unknown type
	}

	for _, tt := range tests {
		result := tt.typ.String()
		if result != tt.expected {
			t.Errorf("Type(%d).String() = %s, expected %s", tt.typ, result, tt.expected)
		}
	}
}

// Test NewValue with various types
func TestNewValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected Type
	}{
		{"nil", nil, TypeNull},
		{"boolean true", true, TypeBoolean},
		{"boolean false", false, TypeBoolean},
		{"int32", int32(42), TypeInt32},
		{"int64", int64(42), TypeInt64},
		{"int", int(42), TypeInt64}, // int is converted to int64
		{"float64", float64(3.14), TypeFloat64},
		{"string", "hello", TypeString},
		{"binary", []byte{0x01, 0x02}, TypeBinary},
		{"objectid", NewObjectID(), TypeObjectID},
		{"timestamp", time.Now(), TypeTimestamp},
		{"array", []interface{}{1, 2, 3}, TypeArray},
		{"map", map[string]interface{}{"key": "value"}, TypeDocument},
		{"document pointer", &Document{}, TypeDocument},
		{"document value", Document{}, TypeDocument},
		{"unknown type", struct{}{}, TypeNull}, // Unknown types become null
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValue(tt.input)
			if v == nil {
				t.Fatal("NewValue returned nil")
			}
			if v.Type != tt.expected {
				t.Errorf("NewValue(%v) Type = %s, expected %s", tt.input, v.Type.String(), tt.expected.String())
			}
		})
	}
}

// Test NewValue int conversion
func TestNewValueIntConversion(t *testing.T) {
	// Test that int is converted to int64
	v := NewValue(int(42))
	if v.Type != TypeInt64 {
		t.Errorf("Expected TypeInt64, got %s", v.Type.String())
	}

	// Check the data was converted
	if data, ok := v.Data.(int64); !ok {
		t.Error("Expected data to be int64")
	} else if data != 42 {
		t.Errorf("Expected data to be 42, got %d", data)
	}
}

// Test NewValue with null types
func TestNewValueNull(t *testing.T) {
	// Test explicit nil
	v := NewValue(nil)
	if v.Type != TypeNull {
		t.Errorf("Expected TypeNull for nil, got %s", v.Type.String())
	}

	// Test unknown type becomes null
	v = NewValue(struct{ unexported int }{42})
	if v.Type != TypeNull {
		t.Errorf("Expected TypeNull for unknown type, got %s", v.Type.String())
	}
	if v.Data != nil {
		t.Error("Expected Data to be nil for unknown type")
	}
}

// Test NewValue preserves data correctly
func TestNewValueDataPreservation(t *testing.T) {
	// String
	v := NewValue("test string")
	if v.Data.(string) != "test string" {
		t.Error("String data not preserved")
	}

	// Int64
	v = NewValue(int64(12345))
	if v.Data.(int64) != 12345 {
		t.Error("Int64 data not preserved")
	}

	// Float64
	v = NewValue(float64(3.14159))
	if v.Data.(float64) != 3.14159 {
		t.Error("Float64 data not preserved")
	}

	// Boolean
	v = NewValue(true)
	if v.Data.(bool) != true {
		t.Error("Boolean data not preserved")
	}

	// Binary
	binary := []byte{0x01, 0x02, 0x03}
	v = NewValue(binary)
	if data, ok := v.Data.([]byte); !ok {
		t.Error("Binary data not preserved")
	} else if len(data) != 3 {
		t.Error("Binary data length mismatch")
	}

	// Array
	arr := []interface{}{1, "two", 3.0}
	v = NewValue(arr)
	if data, ok := v.Data.([]interface{}); !ok {
		t.Error("Array data not preserved")
	} else if len(data) != 3 {
		t.Error("Array data length mismatch")
	}

	// Map
	m := map[string]interface{}{"key": "value"}
	v = NewValue(m)
	if data, ok := v.Data.(map[string]interface{}); !ok {
		t.Error("Map data not preserved")
	} else if data["key"] != "value" {
		t.Error("Map data content mismatch")
	}
}
