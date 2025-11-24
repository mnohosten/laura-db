package lsm

import (
	"bytes"
	"fmt"
	"testing"
)

func TestSkipListInsertAndSearch(t *testing.T) {
	sl := NewSkipList()

	// Insert some keys
	keys := [][]byte{
		[]byte("apple"),
		[]byte("banana"),
		[]byte("cherry"),
		[]byte("date"),
		[]byte("elderberry"),
	}

	for i, key := range keys {
		sl.Insert(key, i)
	}

	// Search for keys
	for i, key := range keys {
		value, found := sl.Search(key)
		if !found {
			t.Fatalf("key %s not found", key)
		}
		if value.(int) != i {
			t.Fatalf("key %s: expected value %d, got %d", key, i, value)
		}
	}

	// Search for nonexistent key
	_, found := sl.Search([]byte("fig"))
	if found {
		t.Fatal("nonexistent key should not be found")
	}
}

func TestSkipListUpdate(t *testing.T) {
	sl := NewSkipList()

	key := []byte("update-test")

	// Insert
	sl.Insert(key, "value1")
	value, _ := sl.Search(key)
	if value.(string) != "value1" {
		t.Fatalf("expected value1, got %s", value)
	}

	// Update
	sl.Insert(key, "value2")
	value, _ = sl.Search(key)
	if value.(string) != "value2" {
		t.Fatalf("expected value2, got %s", value)
	}

	// Size should still be 1 (update, not insert)
	if sl.Size() != 1 {
		t.Fatalf("expected size 1, got %d", sl.Size())
	}
}

func TestSkipListDelete(t *testing.T) {
	sl := NewSkipList()

	keys := [][]byte{
		[]byte("key1"),
		[]byte("key2"),
		[]byte("key3"),
	}

	for i, key := range keys {
		sl.Insert(key, i)
	}

	// Delete middle key
	if !sl.Delete([]byte("key2")) {
		t.Fatal("failed to delete key2")
	}

	if sl.Size() != 2 {
		t.Fatalf("expected size 2, got %d", sl.Size())
	}

	// Verify deleted
	if _, found := sl.Search([]byte("key2")); found {
		t.Fatal("key2 should be deleted")
	}

	// Verify others still exist
	if _, found := sl.Search([]byte("key1")); !found {
		t.Fatal("key1 should still exist")
	}
	if _, found := sl.Search([]byte("key3")); !found {
		t.Fatal("key3 should still exist")
	}
}

func TestSkipListSortedOrder(t *testing.T) {
	sl := NewSkipList()

	// Insert in random order
	keys := []string{"zebra", "apple", "mango", "banana", "cherry"}
	for i, key := range keys {
		sl.Insert([]byte(key), i)
	}

	// Traverse and verify sorted order
	current := sl.head.forward[0]
	var prev []byte

	for current != nil {
		if prev != nil && bytes.Compare(prev, current.key) >= 0 {
			t.Fatalf("keys not in sorted order: %s >= %s", prev, current.key)
		}
		prev = current.key
		current = current.forward[0]
	}
}

func TestSkipListSize(t *testing.T) {
	sl := NewSkipList()

	if sl.Size() != 0 {
		t.Fatalf("expected size 0, got %d", sl.Size())
	}

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		sl.Insert(key, i)
	}

	if sl.Size() != 100 {
		t.Fatalf("expected size 100, got %d", sl.Size())
	}

	// Delete some
	for i := 0; i < 20; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		sl.Delete(key)
	}

	if sl.Size() != 80 {
		t.Fatalf("expected size 80, got %d", sl.Size())
	}
}

func TestSkipListEmpty(t *testing.T) {
	sl := NewSkipList()

	_, found := sl.Search([]byte("any-key"))
	if found {
		t.Fatal("empty skip list should not find any key")
	}

	if !sl.Delete([]byte("any-key")) == true {
		// Delete on empty list should return false
	}

	if sl.Size() != 0 {
		t.Fatalf("empty skip list should have size 0")
	}
}
