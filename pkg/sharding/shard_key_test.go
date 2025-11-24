package sharding

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
)

func TestNewShardKey(t *testing.T) {
	// Test range shard key
	sk := NewRangeShardKey("user_id")
	if sk.Type != ShardKeyTypeRange {
		t.Errorf("expected range type, got %v", sk.Type)
	}
	if len(sk.Fields) != 1 || sk.Fields[0] != "user_id" {
		t.Errorf("unexpected fields: %v", sk.Fields)
	}

	// Test hash shard key
	sk = NewHashShardKey("session_id")
	if sk.Type != ShardKeyTypeHash {
		t.Errorf("expected hash type, got %v", sk.Type)
	}
}

func TestShardKeyExtractValue(t *testing.T) {
	sk := NewRangeShardKey("user_id")

	// Test single field extraction
	doc := map[string]interface{}{
		"user_id": int64(123),
		"name":    "Alice",
	}

	value, err := sk.ExtractShardKeyValue(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != int64(123) {
		t.Errorf("expected 123, got %v", value)
	}

	// Test missing field
	doc = map[string]interface{}{
		"name": "Bob",
	}
	_, err = sk.ExtractShardKeyValue(doc)
	if err == nil {
		t.Error("expected error for missing field")
	}
}

func TestShardKeyExtractCompoundValue(t *testing.T) {
	sk := NewRangeShardKey("country", "user_id")

	doc := map[string]interface{}{
		"country": "US",
		"user_id": int64(456),
		"name":    "Charlie",
	}

	value, err := sk.ExtractShardKeyValue(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	compValue, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected compound value, got %T", value)
	}

	if compValue["country"] != "US" {
		t.Errorf("unexpected country: %v", compValue["country"])
	}
	if compValue["user_id"] != int64(456) {
		t.Errorf("unexpected user_id: %v", compValue["user_id"])
	}
}

func TestShardKeyHashValue(t *testing.T) {
	sk := NewHashShardKey("user_id")

	// Test string hash
	hash1 := sk.HashValue("user123")
	hash2 := sk.HashValue("user123")
	if hash1 != hash2 {
		t.Error("same value should produce same hash")
	}

	hash3 := sk.HashValue("user456")
	if hash1 == hash3 {
		t.Error("different values should produce different hashes")
	}

	// Test int64 hash
	hash4 := sk.HashValue(int64(123))
	hash5 := sk.HashValue(int64(123))
	if hash4 != hash5 {
		t.Error("same int64 should produce same hash")
	}

	// Test ObjectID hash
	id1 := document.NewObjectID()
	hash6 := sk.HashValue(id1)
	hash7 := sk.HashValue(id1)
	if hash6 != hash7 {
		t.Error("same ObjectID should produce same hash")
	}
}

func TestShardKeyHashValueMD5(t *testing.T) {
	sk := NewHashShardKey("user_id")

	hash1 := sk.HashValueMD5("user123")
	hash2 := sk.HashValueMD5("user123")
	if hash1 != hash2 {
		t.Error("same value should produce same MD5 hash")
	}

	hash3 := sk.HashValueMD5("user456")
	if hash1 == hash3 {
		t.Error("different values should produce different MD5 hashes")
	}
}

func TestShardKeyCompareValues(t *testing.T) {
	sk := NewRangeShardKey("user_id")

	// Test string comparison
	cmp := sk.CompareValues("apple", "banana")
	if cmp >= 0 {
		t.Error("apple should be less than banana")
	}

	cmp = sk.CompareValues("banana", "apple")
	if cmp <= 0 {
		t.Error("banana should be greater than apple")
	}

	cmp = sk.CompareValues("apple", "apple")
	if cmp != 0 {
		t.Error("apple should equal apple")
	}

	// Test int64 comparison
	cmp = sk.CompareValues(int64(10), int64(20))
	if cmp >= 0 {
		t.Error("10 should be less than 20")
	}

	cmp = sk.CompareValues(int64(20), int64(10))
	if cmp <= 0 {
		t.Error("20 should be greater than 10")
	}

	// Test float64 comparison
	cmp = sk.CompareValues(10.5, 20.5)
	if cmp >= 0 {
		t.Error("10.5 should be less than 20.5")
	}

	// Test nil handling
	cmp = sk.CompareValues(nil, int64(10))
	if cmp >= 0 {
		t.Error("nil should be less than any value")
	}

	cmp = sk.CompareValues(int64(10), nil)
	if cmp <= 0 {
		t.Error("any value should be greater than nil")
	}
}

func TestShardKeyCompareCompoundValues(t *testing.T) {
	sk := NewRangeShardKey("country", "user_id")

	a := map[string]interface{}{
		"country": "US",
		"user_id": int64(100),
	}

	b := map[string]interface{}{
		"country": "US",
		"user_id": int64(200),
	}

	cmp := sk.CompareValues(a, b)
	if cmp >= 0 {
		t.Error("a should be less than b (same country, different user_id)")
	}

	c := map[string]interface{}{
		"country": "UK",
		"user_id": int64(50),
	}

	cmp = sk.CompareValues(b, c)
	if cmp <= 0 {
		t.Error("b should be greater than c (US > UK)")
	}
}

func TestShardKeyValidate(t *testing.T) {
	// Test valid shard key
	sk := NewRangeShardKey("user_id")
	if err := sk.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test empty fields
	sk = &ShardKey{
		Fields: []string{},
		Type:   ShardKeyTypeRange,
	}
	if err := sk.Validate(); err == nil {
		t.Error("expected error for empty fields")
	}

	// Test duplicate fields
	sk = &ShardKey{
		Fields: []string{"user_id", "user_id"},
		Type:   ShardKeyTypeRange,
	}
	if err := sk.Validate(); err == nil {
		t.Error("expected error for duplicate fields")
	}
}

func TestShardKeyString(t *testing.T) {
	// Single field
	sk := NewRangeShardKey("user_id")
	str := sk.String()
	if str != "{user_id: range}" {
		t.Errorf("unexpected string: %s", str)
	}

	// Compound key
	sk = NewRangeShardKey("country", "user_id")
	str = sk.String()
	if str != "{[country user_id]: range}" {
		t.Errorf("unexpected string: %s", str)
	}
}

func TestShardKeyTypeString(t *testing.T) {
	if ShardKeyTypeRange.String() != "range" {
		t.Errorf("unexpected range string: %s", ShardKeyTypeRange.String())
	}

	if ShardKeyTypeHash.String() != "hash" {
		t.Errorf("unexpected hash string: %s", ShardKeyTypeHash.String())
	}
}

func TestShardKeyHashDistribution(t *testing.T) {
	sk := NewHashShardKey("user_id")

	// Generate hashes for many values and check distribution
	buckets := make([]int, 10)
	for i := 0; i < 10000; i++ {
		hash := sk.HashValue(int64(i))
		bucket := hash % 10
		buckets[bucket]++
	}

	// Each bucket should have roughly 1000 entries (10% variance allowed)
	for i, count := range buckets {
		if count < 900 || count > 1100 {
			t.Errorf("bucket %d has poor distribution: %d entries", i, count)
		}
	}
}

func TestShardKeyCompareObjectID(t *testing.T) {
	sk := NewRangeShardKey("_id")

	id1 := document.NewObjectID()
	id2 := document.NewObjectID()

	// IDs generated sequentially should be comparable
	cmp := sk.CompareValues(id1, id2)
	// id1 should be less than id2 (generated earlier)
	if cmp >= 0 {
		t.Error("earlier ObjectID should be less than later ObjectID")
	}

	// Same ID should be equal
	cmp = sk.CompareValues(id1, id1)
	if cmp != 0 {
		t.Error("same ObjectID should be equal")
	}
}

func TestShardKeyHashValueEdgeCases(t *testing.T) {
	sk := NewHashShardKey("field")

	tests := []struct {
		name  string
		value interface{}
	}{
		{"nil value", nil},
		{"bool true", true},
		{"bool false", false},
		{"float64", float64(123.456)},
		{"float32", float32(78.9)},
		{"map", map[string]interface{}{"key": "value"}},
		{"slice", []interface{}{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := sk.HashValue(tt.value)
			// Just verify it returns a value without panicking
			if hash == 0 && tt.value != nil && tt.value != false {
				// Most values should produce non-zero hash
				t.Logf("Hash for %v is zero", tt.name)
			}
		})
	}
}

func TestShardKeyHashValueMD5EdgeCases(t *testing.T) {
	sk := NewHashShardKey("field")

	tests := []struct {
		name  string
		value interface{}
	}{
		{"time.Time", document.ObjectID{}}, // ObjectID contains timestamp
		{"complex map", map[string]interface{}{"a": 1, "b": "test", "c": true}},
		{"nested slice", []interface{}{[]interface{}{1, 2}, []interface{}{3, 4}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := sk.HashValueMD5(tt.value)
			// Verify it returns a value
			if hash == 0 {
				t.Logf("MD5 hash for %v is zero", tt.name)
			}
		})
	}
}

func TestShardKeyCompareFloats(t *testing.T) {
	sk := NewRangeShardKey("field")

	tests := []struct {
		name     string
		a, b     interface{}
		expected int
	}{
		{"equal floats", float64(1.5), float64(1.5), 0},
		{"a less than b", float64(1.0), float64(2.0), -1},
		{"a greater than b", float64(3.0), float64(1.0), 1},
		{"float32 comparison", float32(1.5), float32(2.5), -1},
		{"mixed float types", float64(2.0), float32(2.0), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sk.CompareValues(tt.a, tt.b)
			if (result < 0 && tt.expected >= 0) ||
				(result > 0 && tt.expected <= 0) ||
				(result == 0 && tt.expected != 0) {
				t.Errorf("Expected comparison result %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestShardKeyCompareCompoundKeysEdgeCases(t *testing.T) {
	sk := &ShardKey{
		Fields: []string{"region", "user_id"},
		Type:   ShardKeyTypeRange,
	}

	tests := []struct {
		name     string
		a, b     interface{}
		expected int
	}{
		{
			"first field different",
			[]interface{}{"US", int64(100)},
			[]interface{}{"UK", int64(100)},
			1, // "US" > "UK"
		},
		{
			"second field different",
			[]interface{}{"US", int64(100)},
			[]interface{}{"US", int64(200)},
			-1, // 100 < 200
		},
		{
			"all equal",
			[]interface{}{"US", int64(100)},
			[]interface{}{"US", int64(100)},
			0,
		},
		{
			"different lengths",
			[]interface{}{"US"},
			[]interface{}{"US", int64(100)},
			1, // longer is greater (based on actual implementation)
		},
		{
			"mixed types in compound",
			[]interface{}{"US", float64(1.5)},
			[]interface{}{"US", int64(2)},
			-1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sk.CompareValues(tt.a, tt.b)
			if (result < 0 && tt.expected >= 0) ||
				(result > 0 && tt.expected <= 0) ||
				(result == 0 && tt.expected != 0) {
				t.Errorf("Expected comparison result sign %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestShardKeyTypeStringUnknown(t *testing.T) {
	// Test unknown type
	var unknownType ShardKeyType = 99
	result := unknownType.String()
	if result != "unknown" {
		t.Errorf("Expected 'unknown', got %s", result)
	}
}

func TestShardKeyCompareValuesTypeMismatches(t *testing.T) {
	sk := NewRangeShardKey("field")

	tests := []struct {
		name string
		a, b interface{}
	}{
		{"string vs int", "abc", int64(123)},
		{"int vs string", int(42), "hello"},
		{"float vs string", float64(3.14), "pi"},
		{"int64 vs float", int64(100), float64(200.5)},
		{"ObjectID vs string", document.NewObjectID(), "str"},
		{"map vs string", map[string]interface{}{"key": "value"}, "map"},
		{"slice vs int", []interface{}{1, 2, 3}, int64(5)},
		{"bool vs int", true, int64(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic and returns a result
			result := sk.CompareValues(tt.a, tt.b)
			_ = result // We expect some comparison result
		})
	}
}

func TestShardKeyCompareValuesNilHandling(t *testing.T) {
	sk := NewRangeShardKey("field")

	// Both nil
	if sk.CompareValues(nil, nil) != 0 {
		t.Error("nil should equal nil")
	}

	// a nil, b non-nil
	if sk.CompareValues(nil, int64(5)) >= 0 {
		t.Error("nil should be less than non-nil")
	}

	// a non-nil, b nil
	if sk.CompareValues(int64(5), nil) <= 0 {
		t.Error("non-nil should be greater than nil")
	}
}

func TestShardKeyCompareCompoundKeysErrors(t *testing.T) {
	sk := &ShardKey{
		Fields: []string{"region", "user_id"},
		Type:   ShardKeyTypeRange,
	}

	tests := []struct {
		name string
		a, b map[string]interface{}
	}{
		{
			"missing field in a",
			map[string]interface{}{"user_id": int64(100)},
			map[string]interface{}{"region": "US", "user_id": int64(100)},
		},
		{
			"missing field in b",
			map[string]interface{}{"region": "US", "user_id": int64(100)},
			map[string]interface{}{"user_id": int64(200)},
		},
		{
			"both missing same field",
			map[string]interface{}{"user_id": int64(100)},
			map[string]interface{}{"user_id": int64(200)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic, handles missing fields gracefully
			result := sk.CompareValues(tt.a, tt.b)
			_ = result
		})
	}
}

func TestShardKeyHashValueMD5MoreTypes(t *testing.T) {
	sk := NewHashShardKey("field")

	// Test various types that use MD5
	tests := []struct {
		name  string
		value interface{}
	}{
		{"string", "test-string"},
		{"int", int(12345)},
		{"int64", int64(9876543210)},
		{"float64", float64(3.14159)},
		{"bool true", true},
		{"bool false", false},
		{"slice of strings", []interface{}{"a", "b", "c"}},
		{"slice of numbers", []interface{}{int64(1), int64(2), int64(3)}},
		{"nested map", map[string]interface{}{
			"outer": map[string]interface{}{
				"inner": "value",
			},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := sk.HashValueMD5(tt.value)
			// Verify MD5 produces consistent results
			hash2 := sk.HashValueMD5(tt.value)
			if hash != hash2 {
				t.Error("MD5 hash should be consistent for same input")
			}
		})
	}
}
