package index

import (
	"testing"
)

func TestCompositeKey(t *testing.T) {
	t.Run("Create composite key", func(t *testing.T) {
		key := NewCompositeKey("New York", int64(25), "Engineer")
		if len(key.Values) != 3 {
			t.Errorf("Expected 3 values, got %d", len(key.Values))
		}
	})

	t.Run("Compare equal keys", func(t *testing.T) {
		key1 := NewCompositeKey("NYC", int64(30))
		key2 := NewCompositeKey("NYC", int64(30))

		if key1.Compare(key2) != 0 {
			t.Error("Expected equal keys to compare as 0")
		}
	})

	t.Run("Compare less than", func(t *testing.T) {
		key1 := NewCompositeKey("Boston", int64(25))
		key2 := NewCompositeKey("NYC", int64(25))

		if key1.Compare(key2) != -1 {
			t.Error("Expected Boston < NYC")
		}
	})

	t.Run("Compare greater than", func(t *testing.T) {
		key1 := NewCompositeKey("NYC", int64(30))
		key2 := NewCompositeKey("NYC", int64(25))

		if key1.Compare(key2) != 1 {
			t.Error("Expected 30 > 25")
		}
	})

	t.Run("Compare different lengths", func(t *testing.T) {
		key1 := NewCompositeKey("NYC", int64(25))
		key2 := NewCompositeKey("NYC", int64(25), "Engineer")

		if key1.Compare(key2) != -1 {
			t.Error("Expected shorter key to be less than longer key")
		}
	})

	t.Run("Matches prefix", func(t *testing.T) {
		key := NewCompositeKey("NYC", int64(30), "Engineer")
		prefix := NewCompositeKey("NYC")

		if !key.MatchesPrefix(prefix) {
			t.Error("Expected key to match prefix")
		}
	})

	t.Run("Does not match prefix", func(t *testing.T) {
		key := NewCompositeKey("NYC", int64(30))
		prefix := NewCompositeKey("Boston")

		if key.MatchesPrefix(prefix) {
			t.Error("Expected key to not match different prefix")
		}
	})
}

func TestBTreeWithCompositeKeys(t *testing.T) {
	btree := NewBTree(4)

	t.Run("Insert composite keys", func(t *testing.T) {
		// Insert (city, age) pairs
		btree.Insert(NewCompositeKey("NYC", int64(25)), "id1")
		btree.Insert(NewCompositeKey("Boston", int64(30)), "id2")
		btree.Insert(NewCompositeKey("NYC", int64(30)), "id3")
		btree.Insert(NewCompositeKey("Boston", int64(25)), "id4")

		if btree.Size() != 4 {
			t.Errorf("Expected size 4, got %d", btree.Size())
		}
	})

	t.Run("Search composite keys", func(t *testing.T) {
		key := NewCompositeKey("NYC", int64(25))
		value, found := btree.Search(key)

		if !found {
			t.Error("Expected to find key")
		}
		if value != "id1" {
			t.Errorf("Expected id1, got %v", value)
		}
	})

	t.Run("Delete composite keys", func(t *testing.T) {
		key := NewCompositeKey("Boston", int64(30))
		err := btree.Delete(key)

		if err != nil {
			t.Errorf("Failed to delete: %v", err)
		}

		if btree.Size() != 3 {
			t.Errorf("Expected size 3 after delete, got %d", btree.Size())
		}

		_, found := btree.Search(key)
		if found {
			t.Error("Expected key to be deleted")
		}
	})
}

func TestCompoundIndexConfig(t *testing.T) {
	t.Run("Create compound index", func(t *testing.T) {
		config := &IndexConfig{
			Name:       "city_age_1",
			FieldPaths: []string{"city", "age"},
			Type:       IndexTypeBTree,
			Unique:     false,
			Order:      32,
		}

		idx := NewIndex(config)

		if !idx.IsCompound() {
			t.Error("Expected index to be compound")
		}

		if len(idx.FieldPaths()) != 2 {
			t.Errorf("Expected 2 field paths, got %d", len(idx.FieldPaths()))
		}

		if idx.FieldPath() != "city" {
			t.Errorf("Expected first field to be 'city', got %s", idx.FieldPath())
		}
	})

	t.Run("Backward compatibility with single field", func(t *testing.T) {
		config := &IndexConfig{
			Name:      "city_1",
			FieldPath: "city",
			Type:      IndexTypeBTree,
			Unique:    false,
			Order:     32,
		}

		idx := NewIndex(config)

		if idx.IsCompound() {
			t.Error("Expected index to be single-field")
		}

		if idx.FieldPath() != "city" {
			t.Errorf("Expected field to be 'city', got %s", idx.FieldPath())
		}
	})
}

func TestCompositeKeyOrdering(t *testing.T) {
	// Use unique composite keys (city, age, id) to avoid duplicates
	keys := []*CompositeKey{
		NewCompositeKey("NYC", int64(25), "id1"),
		NewCompositeKey("Boston", int64(30), "id2"),
		NewCompositeKey("NYC", int64(30), "id3"),
		NewCompositeKey("Boston", int64(25), "id4"),
		NewCompositeKey("Seattle", int64(28), "id5"),
	}

	// Expected order after sorting:
	// Boston/25/id4, Boston/30/id2, NYC/25/id1, NYC/30/id3, Seattle/28/id5

	btree := NewBTree(4)
	for i, key := range keys {
		err := btree.Insert(key, i)
		if err != nil {
			t.Logf("Skipped key %v (duplicate): %v", key.Values, err)
		} else {
			t.Logf("Inserted key %v", key.Values)
		}
	}

	// Range scan should return in sorted order
	allKeys, _ := btree.RangeScan(nil, nil)

	if len(allKeys) != 5 {
		t.Errorf("Expected 5 keys, got %d", len(allKeys))
	}

	// Verify first key is Boston/25/id4
	firstKey := allKeys[0].(*CompositeKey)
	if firstKey.Values[0] != "Boston" || firstKey.Values[1] != int64(25) {
		t.Errorf("Expected Boston/25 first, got %v/%v", firstKey.Values[0], firstKey.Values[1])
	}

	// Verify last key is Seattle/28/id5
	lastKey := allKeys[len(allKeys)-1].(*CompositeKey)
	if lastKey.Values[0] != "Seattle" || lastKey.Values[1] != int64(28) {
		t.Errorf("Expected Seattle/28 last, got %v/%v", lastKey.Values[0], lastKey.Values[1])
	}
}
