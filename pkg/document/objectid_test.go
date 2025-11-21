package document

import (
	"testing"
	"time"
)

func TestNewObjectID(t *testing.T) {
	id := NewObjectID()

	if id.IsZero() {
		t.Error("Expected non-zero ObjectID")
	}

	// Check hex representation
	hex := id.Hex()
	if len(hex) != 24 {
		t.Errorf("Expected 24 character hex string, got %d", len(hex))
	}
}

func TestObjectIDUniqueness(t *testing.T) {
	id1 := NewObjectID()
	id2 := NewObjectID()

	if id1 == id2 {
		t.Error("Expected unique ObjectIDs")
	}

	if id1.Hex() == id2.Hex() {
		t.Error("Expected unique hex representations")
	}
}

func TestObjectIDFromHex(t *testing.T) {
	original := NewObjectID()
	hex := original.Hex()

	parsed, err := ObjectIDFromHex(hex)
	if err != nil {
		t.Fatalf("Failed to parse hex: %v", err)
	}

	if parsed != original {
		t.Error("Parsed ObjectID doesn't match original")
	}
}

func TestObjectIDFromHexInvalid(t *testing.T) {
	// Too short
	_, err := ObjectIDFromHex("abc")
	if err == nil {
		t.Error("Expected error for short hex string")
	}

	// Invalid characters
	_, err = ObjectIDFromHex("zzzzzzzzzzzzzzzzzzzzzzzz")
	if err == nil {
		t.Error("Expected error for invalid hex characters")
	}
}

func TestObjectIDTimestamp(t *testing.T) {
	before := time.Now()
	id := NewObjectID()
	after := time.Now()

	timestamp := id.Timestamp()

	if timestamp.Before(before.Add(-time.Second)) || timestamp.After(after.Add(time.Second)) {
		t.Error("ObjectID timestamp is not within expected range")
	}
}

func TestObjectIDIsZero(t *testing.T) {
	var zeroID ObjectID
	if !zeroID.IsZero() {
		t.Error("Expected zero ObjectID to return true for IsZero()")
	}

	id := NewObjectID()
	if id.IsZero() {
		t.Error("Expected non-zero ObjectID to return false for IsZero()")
	}
}

func TestObjectIDString(t *testing.T) {
	id := NewObjectID()
	str := id.String()
	hex := id.Hex()

	if str != hex {
		t.Error("String() and Hex() should return the same value")
	}
}
