package database

import (
	"os"
	"testing"
)

func TestPushWithEach(t *testing.T) {
	// Create test database
	testDir := "./test_push_each"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Alice",
		"tags": []interface{}{"go"},
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $push with $each - add multiple tags at once
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Alice"},
		map[string]interface{}{
			"$push": map[string]interface{}{
				"tags": map[string]interface{}{
					"$each": []interface{}{"database", "nosql", "mongodb"},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to push with $each: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	tags, _ := doc.Get("tags")
	tagsArr := tags.([]interface{})

	// Should have all 4 tags: "go", "database", "nosql", "mongodb"
	expectedTags := []string{"go", "database", "nosql", "mongodb"}
	if len(tagsArr) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(tagsArr))
	}

	for i, expected := range expectedTags {
		if tagsArr[i] != expected {
			t.Errorf("At index %d: expected %v, got %v", i, expected, tagsArr[i])
		}
	}
}

func TestPushWithEachOnNewField(t *testing.T) {
	// Create test database
	testDir := "./test_push_each_new"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document without tags field
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Bob",
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $push with $each on non-existent field
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Bob"},
		map[string]interface{}{
			"$push": map[string]interface{}{
				"tags": map[string]interface{}{
					"$each": []interface{}{"tag1", "tag2", "tag3"},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to push with $each: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Bob"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	tags, exists := doc.Get("tags")
	if !exists {
		t.Fatal("Tags field should exist")
	}

	tagsArr := tags.([]interface{})
	if len(tagsArr) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tagsArr))
	}
}

func TestAddToSetWithEach(t *testing.T) {
	// Create test database
	testDir := "./test_addtoset_each"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document
	_, err = coll.InsertOne(map[string]interface{}{
		"name":  "Charlie",
		"tags":  []interface{}{"go", "database"},
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $addToSet with $each - add multiple tags, some duplicates
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Charlie"},
		map[string]interface{}{
			"$addToSet": map[string]interface{}{
				"tags": map[string]interface{}{
					"$each": []interface{}{"database", "nosql", "mongodb", "go"},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to addToSet with $each: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Charlie"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	tags, _ := doc.Get("tags")
	tagsArr := tags.([]interface{})

	// Should only have unique tags: "go", "database", "nosql", "mongodb"
	if len(tagsArr) != 4 {
		t.Errorf("Expected 4 unique tags, got %d", len(tagsArr))
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, tag := range tagsArr {
		tagStr := tag.(string)
		if seen[tagStr] {
			t.Errorf("Duplicate tag found: %s", tagStr)
		}
		seen[tagStr] = true
	}
}

func TestAddToSetWithEachAllDuplicates(t *testing.T) {
	// Create test database
	testDir := "./test_addtoset_each_dups"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Diana",
		"tags": []interface{}{"tag1", "tag2", "tag3"},
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $addToSet with $each - all values already exist
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Diana"},
		map[string]interface{}{
			"$addToSet": map[string]interface{}{
				"tags": map[string]interface{}{
					"$each": []interface{}{"tag1", "tag2", "tag3"},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to addToSet with $each: %v", err)
	}

	// Verify - should still have only 3 tags
	doc, err := coll.FindOne(map[string]interface{}{"name": "Diana"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	tags, _ := doc.Get("tags")
	tagsArr := tags.([]interface{})

	if len(tagsArr) != 3 {
		t.Errorf("Expected 3 tags (no duplicates), got %d", len(tagsArr))
	}
}

func TestAddToSetWithEachOnNewField(t *testing.T) {
	// Create test database
	testDir := "./test_addtoset_each_new"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document without tags field
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Eve",
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $addToSet with $each on non-existent field with duplicates in input
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Eve"},
		map[string]interface{}{
			"$addToSet": map[string]interface{}{
				"tags": map[string]interface{}{
					"$each": []interface{}{"tag1", "tag2", "tag1", "tag3", "tag2"},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to addToSet with $each: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Eve"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	tags, exists := doc.Get("tags")
	if !exists {
		t.Fatal("Tags field should exist")
	}

	tagsArr := tags.([]interface{})
	// Should only have 3 unique tags even though 5 were provided
	if len(tagsArr) != 3 {
		t.Errorf("Expected 3 unique tags, got %d", len(tagsArr))
	}
}

func TestPushWithoutEachStillWorks(t *testing.T) {
	// Ensure backward compatibility - $push without $each should still work
	testDir := "./test_push_backward"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Frank",
		"tags": []interface{}{"tag1"},
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test regular $push without $each
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Frank"},
		map[string]interface{}{
			"$push": map[string]interface{}{
				"tags": "tag2",
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to push: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Frank"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	tags, _ := doc.Get("tags")
	tagsArr := tags.([]interface{})

	if len(tagsArr) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tagsArr))
	}
	if tagsArr[0] != "tag1" || tagsArr[1] != "tag2" {
		t.Errorf("Expected [tag1, tag2], got %v", tagsArr)
	}
}

func TestAddToSetWithoutEachStillWorks(t *testing.T) {
	// Ensure backward compatibility - $addToSet without $each should still work
	testDir := "./test_addtoset_backward"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Grace",
		"tags": []interface{}{"tag1"},
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test regular $addToSet without $each
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Grace"},
		map[string]interface{}{
			"$addToSet": map[string]interface{}{
				"tags": "tag2",
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to addToSet: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Grace"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	tags, _ := doc.Get("tags")
	tagsArr := tags.([]interface{})

	if len(tagsArr) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tagsArr))
	}
}

func TestPushEachWithNumbers(t *testing.T) {
	// Test $push with $each using numeric values
	testDir := "./test_push_each_numbers"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document
	_, err = coll.InsertOne(map[string]interface{}{
		"name":   "Henry",
		"scores": []interface{}{75},
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $push with $each for numbers
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Henry"},
		map[string]interface{}{
			"$push": map[string]interface{}{
				"scores": map[string]interface{}{
					"$each": []interface{}{82, 88, 90, 85},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to push with $each: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Henry"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	scores, _ := doc.Get("scores")
	scoresArr := scores.([]interface{})

	if len(scoresArr) != 5 {
		t.Errorf("Expected 5 scores, got %d", len(scoresArr))
	}
}
