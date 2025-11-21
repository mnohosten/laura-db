package index

import (
	"testing"
)

func TestBTreeInsertSearch(t *testing.T) {
	btree := NewBTree(3)

	// Insert
	err := btree.Insert(int64(10), "value10")
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Search
	val, found := btree.Search(int64(10))
	if !found {
		t.Error("Expected to find key 10")
	}
	if val.(string) != "value10" {
		t.Errorf("Expected 'value10', got %v", val)
	}

	// Search non-existent key
	_, found = btree.Search(int64(99))
	if found {
		t.Error("Expected to not find key 99")
	}
}

func TestBTreeMultipleInserts(t *testing.T) {
	btree := NewBTree(3)

	keys := []int64{50, 30, 70, 20, 40, 60, 80, 10, 90}
	for _, key := range keys {
		err := btree.Insert(key, key*10)
		if err != nil {
			t.Fatalf("Insert failed for key %d: %v", key, err)
		}
	}

	// Verify all keys
	for _, key := range keys {
		val, found := btree.Search(key)
		if !found {
			t.Errorf("Key %d not found", key)
		}
		if val.(int64) != key*10 {
			t.Errorf("Expected %d, got %v", key*10, val)
		}
	}

	// Verify size
	if btree.Size() != len(keys) {
		t.Errorf("Expected size %d, got %d", len(keys), btree.Size())
	}
}

func TestBTreeDuplicateInsert(t *testing.T) {
	btree := NewBTree(3)

	btree.Insert(int64(10), "value1")
	err := btree.Insert(int64(10), "value2")

	if err != ErrDuplicateKey {
		t.Error("Expected duplicate key error")
	}
}

func TestBTreeDelete(t *testing.T) {
	btree := NewBTree(3)

	btree.Insert(int64(10), "value10")
	btree.Insert(int64(20), "value20")
	btree.Insert(int64(30), "value30")

	// Delete
	err := btree.Delete(int64(20))
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, found := btree.Search(int64(20))
	if found {
		t.Error("Expected key 20 to be deleted")
	}

	// Verify others still exist
	_, found = btree.Search(int64(10))
	if !found {
		t.Error("Expected key 10 to still exist")
	}

	// Verify size
	if btree.Size() != 2 {
		t.Errorf("Expected size 2, got %d", btree.Size())
	}
}

func TestBTreeRangeScan(t *testing.T) {
	btree := NewBTree(3)

	// Insert keys 10, 20, 30, 40, 50
	for i := int64(10); i <= 50; i += 10 {
		btree.Insert(i, i*10)
	}

	// Range scan [20, 40]
	keys, values := btree.RangeScan(int64(20), int64(40))

	expectedKeys := []int64{20, 30, 40}
	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d", len(expectedKeys), len(keys))
	}

	for i, key := range keys {
		if key.(int64) != expectedKeys[i] {
			t.Errorf("Expected key %d, got %v", expectedKeys[i], key)
		}
		if values[i].(int64) != expectedKeys[i]*10 {
			t.Errorf("Expected value %d, got %v", expectedKeys[i]*10, values[i])
		}
	}
}

func TestBTreeStringKeys(t *testing.T) {
	btree := NewBTree(3)

	btree.Insert("apple", 1)
	btree.Insert("banana", 2)
	btree.Insert("cherry", 3)

	val, found := btree.Search("banana")
	if !found {
		t.Error("Expected to find 'banana'")
	}
	if val.(int) != 2 {
		t.Errorf("Expected 2, got %v", val)
	}

	// Range scan
	keys, _ := btree.RangeScan("apple", "cherry")
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys in range, got %d", len(keys))
	}
}

func TestIndexCreate(t *testing.T) {
	config := &IndexConfig{
		Name:      "test_idx",
		FieldPath: "email",
		Type:      IndexTypeBTree,
		Unique:    true,
		Order:     32,
	}

	idx := NewIndex(config)
	if idx.Name() != "test_idx" {
		t.Errorf("Expected name 'test_idx', got %s", idx.Name())
	}

	if !idx.IsUnique() {
		t.Error("Expected unique index")
	}
}

func TestIndexInsertSearch(t *testing.T) {
	config := &IndexConfig{
		Name:      "email_idx",
		FieldPath: "email",
		Unique:    true,
	}

	idx := NewIndex(config)

	// Insert
	err := idx.Insert("alice@example.com", "doc1")
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Search
	val, found := idx.Search("alice@example.com")
	if !found {
		t.Error("Expected to find key")
	}
	if val.(string) != "doc1" {
		t.Errorf("Expected 'doc1', got %v", val)
	}
}

func TestIndexUniquenessConstraint(t *testing.T) {
	config := &IndexConfig{
		Name:      "email_idx",
		FieldPath: "email",
		Unique:    true,
	}

	idx := NewIndex(config)

	idx.Insert("alice@example.com", "doc1")
	err := idx.Insert("alice@example.com", "doc2")

	if err == nil {
		t.Error("Expected error for duplicate key in unique index")
	}
}
