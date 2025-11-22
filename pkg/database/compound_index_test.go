package database

import (
	"os"
	"testing"
)

func TestCollectionCompoundIndex(t *testing.T) {
	dir := "./test_compound_idx"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("users")

	t.Run("Create compound index", func(t *testing.T) {
		err := coll.CreateCompoundIndex([]string{"city", "age"}, false)
		if err != nil {
			t.Fatalf("Failed to create compound index: %v", err)
		}

		indexes := coll.ListIndexes()
		found := false
		for _, idx := range indexes {
			if idx["name"] == "city_age_1" {
				found = true
				if !idx["is_compound"].(bool) {
					t.Error("Expected index to be marked as compound")
				}
				fieldPaths := idx["field_paths"].([]string)
				if len(fieldPaths) != 2 {
					t.Errorf("Expected 2 field paths, got %d", len(fieldPaths))
				}
				if fieldPaths[0] != "city" || fieldPaths[1] != "age" {
					t.Errorf("Expected [city, age], got %v", fieldPaths)
				}
			}
		}
		if !found {
			t.Error("Compound index not found in index list")
		}
	})

	t.Run("Insert documents with compound index", func(t *testing.T) {
		docs := []map[string]interface{}{
			{"name": "Alice", "city": "NYC", "age": int64(25)},
			{"name": "Bob", "city": "NYC", "age": int64(30)},
			{"name": "Charlie", "city": "Boston", "age": int64(25)},
			{"name": "Diana", "city": "Boston", "age": int64(30)},
			{"name": "Eve", "city": "Seattle", "age": int64(28)},
		}

		for _, doc := range docs {
			_, err := coll.InsertOne(doc)
			if err != nil {
				t.Fatalf("Failed to insert document: %v", err)
			}
		}

		count, _ := coll.Count(map[string]interface{}{})
		if count != 5 {
			t.Errorf("Expected 5 documents, got %d", count)
		}
	})

	t.Run("Query using full compound index", func(t *testing.T) {
		// Query matching both city and age
		results, err := coll.Find(map[string]interface{}{
			"city": "NYC",
			"age":  int64(30),
		})

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if len(results) > 0 {
			name, _ := results[0].Get("name")
			if name != "Bob" {
				t.Errorf("Expected Bob, got %v", name)
			}
		}
	})

	t.Run("Query using compound index prefix", func(t *testing.T) {
		// Query matching only city (prefix of compound index)
		results, err := coll.Find(map[string]interface{}{
			"city": "NYC",
		})

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results (Alice and Bob), got %d", len(results))
		}
	})

	t.Run("Query not using compound index", func(t *testing.T) {
		// Query matching only age (not a prefix)
		results, err := coll.Find(map[string]interface{}{
			"age": int64(25),
		})

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		// Should find Alice and Charlie
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("Update document with compound index", func(t *testing.T) {
		// Update Alice's age
		err := coll.UpdateOne(
			map[string]interface{}{"name": "Alice"},
			map[string]interface{}{
				"$set": map[string]interface{}{"age": int64(26)},
			},
		)

		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		// Verify update worked
		results, _ := coll.Find(map[string]interface{}{
			"name": "Alice",
		})

		if len(results) > 0 {
			age, _ := results[0].Get("age")
			if age != int64(26) {
				t.Errorf("Expected age 26, got %v", age)
			}
		}

		// Verify compound index still works
		results, _ = coll.Find(map[string]interface{}{
			"city": "NYC",
			"age":  int64(26),
		})

		if len(results) != 1 {
			t.Errorf("Expected to find updated Alice, got %d results", len(results))
		}
	})

	t.Run("Delete document with compound index", func(t *testing.T) {
		err := coll.DeleteOne(map[string]interface{}{"name": "Eve"})
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify delete
		count, _ := coll.Count(map[string]interface{}{})
		if count != 4 {
			t.Errorf("Expected 4 documents after delete, got %d", count)
		}

		// Verify compound index still works
		results, _ := coll.Find(map[string]interface{}{
			"city": "Seattle",
			"age":  int64(28),
		})

		if len(results) != 0 {
			t.Error("Expected no results for deleted document")
		}
	})
}

func TestCompoundIndexWithThreeFields(t *testing.T) {
	dir := "./test_compound_3field"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("employees")

	// Create 3-field compound index
	err = coll.CreateCompoundIndex([]string{"department", "level", "salary"}, false)
	if err != nil {
		t.Fatalf("Failed to create 3-field compound index: %v", err)
	}

	// Insert test data
	docs := []map[string]interface{}{
		{"name": "Alice", "department": "Engineering", "level": "Senior", "salary": int64(120000)},
		{"name": "Bob", "department": "Engineering", "level": "Junior", "salary": int64(80000)},
		{"name": "Charlie", "department": "Sales", "level": "Senior", "salary": int64(100000)},
	}

	for _, doc := range docs {
		coll.InsertOne(doc)
	}

	t.Run("Full 3-field match", func(t *testing.T) {
		results, err := coll.Find(map[string]interface{}{
			"department": "Engineering",
			"level":      "Senior",
			"salary":     int64(120000),
		})

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if len(results) > 0 {
			name, _ := results[0].Get("name")
			if name != "Alice" {
				t.Errorf("Expected Alice, got %v", name)
			}
		}
	})

	t.Run("2-field prefix match", func(t *testing.T) {
		results, err := coll.Find(map[string]interface{}{
			"department": "Engineering",
			"level":      "Senior",
		})

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("1-field prefix match", func(t *testing.T) {
		results, err := coll.Find(map[string]interface{}{
			"department": "Engineering",
		})

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})
}

func TestCompoundIndexUniqueness(t *testing.T) {
	dir := "./test_compound_unique"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("users")

	// Create unique compound index
	err = coll.CreateCompoundIndex([]string{"email", "username"}, true)
	if err != nil {
		t.Fatalf("Failed to create unique compound index: %v", err)
	}

	// Insert first document
	_, err = coll.InsertOne(map[string]interface{}{
		"email":    "alice@example.com",
		"username": "alice123",
		"name":     "Alice",
	})

	if err != nil {
		t.Fatalf("Failed to insert first document: %v", err)
	}

	// Try to insert duplicate (same email and username)
	_, err = coll.InsertOne(map[string]interface{}{
		"email":    "alice@example.com",
		"username": "alice123",
		"name":     "Alice Duplicate",
	})

	if err == nil {
		t.Error("Expected duplicate key error for unique compound index")
	}

	// Insert with different combination (should work)
	_, err = coll.InsertOne(map[string]interface{}{
		"email":    "alice@example.com",
		"username": "alice456", // Different username
		"name":     "Alice Alt",
	})

	if err != nil {
		t.Errorf("Should allow different compound key: %v", err)
	}
}
