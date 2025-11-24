package index

import (
	"testing"
)

func TestBTree_CustomOrder(t *testing.T) {
	// Test creating BTree with custom order
	tree := NewBTree(5) // Order of 5

	// Insert some values
	tree.Insert(int64(10), "val10")
	tree.Insert(int64(20), "val20")
	tree.Insert(int64(5), "val5")
	tree.Insert(int64(15), "val15")

	// Search should work
	val, found := tree.Search(int64(10))
	if !found || val != "val10" {
		t.Error("Expected to find inserted value")
	}

	// Size should be correct
	if tree.Size() != 4 {
		t.Errorf("Expected size 4, got %d", tree.Size())
	}

	// Test with 0 order (should use default)
	tree2 := NewBTree(0)
	tree2.Insert("key", "value")
	val2, found2 := tree2.Search("key")
	if !found2 || val2 != "value" {
		t.Error("Expected to find value in tree with default order")
	}

	// Test with float64 keys to cover compare function
	tree3 := NewBTree(0)
	tree3.Insert(1.5, "val1")
	tree3.Insert(2.7, "val2")
	tree3.Insert(0.3, "val3")

	val3, found3 := tree3.Search(1.5)
	if !found3 || val3 != "val1" {
		t.Error("Expected to find float64 key 1.5")
	}

	// Range scan with floats
	keys, values := tree3.RangeScan(0.5, 2.0)
	if len(keys) != 1 || len(values) != 1 {
		t.Errorf("Expected 1 key in range [0.5, 2.0], got %d", len(keys))
	}
}

func TestCompositeKey_MatchesPrefix(t *testing.T) {
	// Test MatchesPrefix method
	ck1 := &CompositeKey{
		Values: []interface{}{"New York", int64(25), "active"},
	}

	// Matching prefix with 1 value
	prefix1 := &CompositeKey{
		Values: []interface{}{"New York"},
	}
	if !ck1.MatchesPrefix(prefix1) {
		t.Error("Expected key to match prefix with 1 value")
	}

	// Matching prefix with 2 values
	prefix2 := &CompositeKey{
		Values: []interface{}{"New York", int64(25)},
	}
	if !ck1.MatchesPrefix(prefix2) {
		t.Error("Expected key to match prefix with 2 values")
	}

	// Non-matching prefix
	prefix3 := &CompositeKey{
		Values: []interface{}{"Boston"},
	}
	if ck1.MatchesPrefix(prefix3) {
		t.Error("Expected key to not match different prefix")
	}

	// Prefix longer than key
	prefix4 := &CompositeKey{
		Values: []interface{}{"New York", int64(25), "active", "extra"},
	}
	if ck1.MatchesPrefix(prefix4) {
		t.Error("Expected key to not match longer prefix")
	}

	// Empty prefix (should match)
	prefix5 := &CompositeKey{
		Values: []interface{}{},
	}
	if !ck1.MatchesPrefix(prefix5) {
		t.Error("Expected key to match empty prefix")
	}
}

func TestCompositeKey_Compare(t *testing.T) {
	// Test Compare method
	ck1 := &CompositeKey{
		Values: []interface{}{"New York", int64(25)},
	}

	ck2 := &CompositeKey{
		Values: []interface{}{"New York", int64(30)},
	}

	// ck1 < ck2 (same first value, different second)
	if ck1.Compare(ck2) != -1 {
		t.Error("Expected ck1 < ck2")
	}

	if ck2.Compare(ck1) != 1 {
		t.Error("Expected ck2 > ck1")
	}

	// Equal keys
	ck3 := &CompositeKey{
		Values: []interface{}{"New York", int64(25)},
	}
	if ck1.Compare(ck3) != 0 {
		t.Error("Expected ck1 == ck3")
	}

	// Different first value
	ck4 := &CompositeKey{
		Values: []interface{}{"Boston", int64(25)},
	}
	if ck1.Compare(ck4) != 1 {
		t.Error("Expected ck1 > ck4 (New York > Boston)")
	}

	// Different lengths
	ck5 := &CompositeKey{
		Values: []interface{}{"New York"},
	}
	result := ck1.Compare(ck5)
	// Shorter key is less if all values match
	if result != 1 {
		t.Error("Expected ck1 > ck5 (longer key)")
	}
}
