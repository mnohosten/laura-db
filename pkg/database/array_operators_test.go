package database

import (
	"os"
	"testing"
)

func TestArrayPushOperator(t *testing.T) {
	// Create test database
	testDir := "./test_array_push"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document with array
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Alice",
		"tags": []interface{}{"go", "database"},
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $push (use name field for filter since _id is complex)
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Alice"},
		map[string]interface{}{
			"$push": map[string]interface{}{
				"tags": "performance",
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to push: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	tags, exists := doc.Get("tags")
	if !exists {
		t.Fatal("Tags field not found")
	}

	tagsArr := tags.([]interface{})
	if len(tagsArr) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tagsArr))
	}
	if tagsArr[2] != "performance" {
		t.Errorf("Expected last tag to be 'performance', got '%v'", tagsArr[2])
	}
}

func TestArrayPullOperator(t *testing.T) {
	// Create test database
	testDir := "./test_array_pull"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document with array
	_, err = coll.InsertOne(map[string]interface{}{
		"name":   "Bob",
		"scores": []interface{}{10, 20, 30, 20, 40},
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $pull - remove all instances of 20
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Bob"},
		map[string]interface{}{
			"$pull": map[string]interface{}{
				"scores": 20,
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to pull: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Bob"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	scores, exists := doc.Get("scores")
	if !exists {
		t.Fatal("Scores field not found")
	}

	scoresArr := scores.([]interface{})
	if len(scoresArr) != 3 {
		t.Errorf("Expected 3 scores after pull, got %d", len(scoresArr))
	}

	// Verify no 20s remain
	for _, score := range scoresArr {
		if score == 20 || score == float64(20) {
			t.Error("Found 20 in array after $pull")
		}
	}
}

func TestArrayAddToSetOperator(t *testing.T) {
	// Create test database
	testDir := "./test_array_addtoset"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document with array
	_, err = coll.InsertOne(map[string]interface{}{
		"name":  "Charlie",
		"roles": []interface{}{"admin", "user"},
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $addToSet with existing value - should not add
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Charlie"},
		map[string]interface{}{
			"$addToSet": map[string]interface{}{
				"roles": "admin",
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to addToSet: %v", err)
	}

	// Verify - should still have 2 roles
	doc, err := coll.FindOne(map[string]interface{}{"name": "Charlie"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	roles, _ := doc.Get("roles")
	rolesArr := roles.([]interface{})
	if len(rolesArr) != 2 {
		t.Errorf("Expected 2 roles (duplicate not added), got %d", len(rolesArr))
	}

	// Test $addToSet with new value - should add
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Charlie"},
		map[string]interface{}{
			"$addToSet": map[string]interface{}{
				"roles": "moderator",
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to addToSet: %v", err)
	}

	// Verify - should now have 3 roles
	doc, err = coll.FindOne(map[string]interface{}{"name": "Charlie"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	roles, _ = doc.Get("roles")
	rolesArr = roles.([]interface{})
	if len(rolesArr) != 3 {
		t.Errorf("Expected 3 roles, got %d", len(rolesArr))
	}
}

func TestArrayPopOperator(t *testing.T) {
	// Create test database
	testDir := "./test_array_pop"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document with array
	_, err = coll.InsertOne(map[string]interface{}{
		"name":    "David",
		"numbers": []interface{}{1, 2, 3, 4, 5},
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $pop with 1 (remove last)
	err = coll.UpdateOne(
		map[string]interface{}{"name": "David"},
		map[string]interface{}{
			"$pop": map[string]interface{}{
				"numbers": 1,
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to pop last: %v", err)
	}

	// Verify - 5 should be removed
	doc, err := coll.FindOne(map[string]interface{}{"name": "David"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	numbers, _ := doc.Get("numbers")
	numbersArr := numbers.([]interface{})
	if len(numbersArr) != 4 {
		t.Errorf("Expected 4 numbers after pop, got %d", len(numbersArr))
	}

	// Test $pop with -1 (remove first)
	err = coll.UpdateOne(
		map[string]interface{}{"name": "David"},
		map[string]interface{}{
			"$pop": map[string]interface{}{
				"numbers": -1,
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to pop first: %v", err)
	}

	// Verify - 1 should be removed
	doc, err = coll.FindOne(map[string]interface{}{"name": "David"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	numbers, _ = doc.Get("numbers")
	numbersArr = numbers.([]interface{})
	if len(numbersArr) != 3 {
		t.Errorf("Expected 3 numbers after pop, got %d", len(numbersArr))
	}

	// Should be [2, 3, 4]
	firstVal, _ := toFloat64(numbersArr[0])
	lastVal, _ := toFloat64(numbersArr[2])
	if firstVal != 2 {
		t.Errorf("Expected first element to be 2, got %v", numbersArr[0])
	}
	if lastVal != 4 {
		t.Errorf("Expected last element to be 4, got %v", numbersArr[2])
	}
}
