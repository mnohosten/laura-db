package database

import (
	"os"
	"testing"
	"time"
)

func TestRenameOperator(t *testing.T) {
	// Create test database
	testDir := "./test_rename"
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
		"name":     "Alice",
		"old_name": "OldValue",
		"age":      25,
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $rename - rename old_name to new_name
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Alice"},
		map[string]interface{}{
			"$rename": map[string]interface{}{
				"old_name": "new_name",
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to rename: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	// Old field should not exist
	if _, exists := doc.Get("old_name"); exists {
		t.Error("Old field 'old_name' should not exist after rename")
	}

	// New field should exist with the old value
	newVal, exists := doc.Get("new_name")
	if !exists {
		t.Fatal("New field 'new_name' should exist after rename")
	}
	if newVal != "OldValue" {
		t.Errorf("Expected new_name='OldValue', got %v", newVal)
	}

	// Other fields should be unchanged
	age, _ := doc.Get("age")
	ageNum, _ := toFloat64(age)
	if ageNum != 25 {
		t.Errorf("Age should be 25, got %v", ageNum)
	}
}

func TestRenameNonExistentField(t *testing.T) {
	// Create test database
	testDir := "./test_rename_nonexistent"
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
		"name": "Bob",
		"age":  30,
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Try to rename non-existent field
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Bob"},
		map[string]interface{}{
			"$rename": map[string]interface{}{
				"nonexistent": "new_field",
			},
		},
	)
	if err != nil {
		t.Fatalf("Rename should not fail for non-existent field: %v", err)
	}

	// Verify document unchanged
	doc, err := coll.FindOne(map[string]interface{}{"name": "Bob"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	// new_field should not exist
	if _, exists := doc.Get("new_field"); exists {
		t.Error("new_field should not exist when renaming non-existent field")
	}
}

func TestCurrentDateOperator(t *testing.T) {
	// Create test database
	testDir := "./test_current_date"
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
		"name": "Charlie",
		"age":  35,
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Record time before update
	beforeUpdate := time.Now()

	// Test $currentDate with timestamp
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Charlie"},
		map[string]interface{}{
			"$currentDate": map[string]interface{}{
				"lastModified": map[string]interface{}{
					"$type": "timestamp",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to set current date: %v", err)
	}

	// Record time after update
	afterUpdate := time.Now()

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Charlie"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	lastModified, exists := doc.Get("lastModified")
	if !exists {
		t.Fatal("lastModified field should exist")
	}

	// Check that it's a timestamp (Unix time)
	var timestamp int64
	switch v := lastModified.(type) {
	case int64:
		timestamp = v
	case int:
		timestamp = int64(v)
	case float64:
		timestamp = int64(v)
	default:
		t.Fatalf("lastModified should be a numeric timestamp, got %T", lastModified)
	}

	// Verify timestamp is within reasonable range
	if timestamp < beforeUpdate.Unix() || timestamp > afterUpdate.Unix() {
		t.Errorf("Timestamp %d not within expected range [%d, %d]",
			timestamp, beforeUpdate.Unix(), afterUpdate.Unix())
	}
}

func TestCurrentDateWithDate(t *testing.T) {
	// Create test database
	testDir := "./test_current_date_obj"
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
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	beforeUpdate := time.Now()

	// Test $currentDate with date object (default or $type: "date")
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Diana"},
		map[string]interface{}{
			"$currentDate": map[string]interface{}{
				"createdAt": true, // Simple true means use date object
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to set current date: %v", err)
	}

	afterUpdate := time.Now()

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Diana"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	createdAt, exists := doc.Get("createdAt")
	if !exists {
		t.Fatal("createdAt field should exist")
	}

	// Check that it's a time.Time object
	timeVal, ok := createdAt.(time.Time)
	if !ok {
		t.Fatalf("createdAt should be a time.Time, got %T", createdAt)
	}

	// Verify time is within reasonable range
	if timeVal.Before(beforeUpdate) || timeVal.After(afterUpdate) {
		t.Errorf("Time %v not within expected range [%v, %v]",
			timeVal, beforeUpdate, afterUpdate)
	}
}

func TestPullAllOperator(t *testing.T) {
	// Create test database
	testDir := "./test_pullall"
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
		"name":   "Eve",
		"scores": []interface{}{10, 20, 30, 20, 40, 50, 30},
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $pullAll - remove multiple values (20 and 30)
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Eve"},
		map[string]interface{}{
			"$pullAll": map[string]interface{}{
				"scores": []interface{}{20, 30},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to pullAll: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Eve"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	scores, exists := doc.Get("scores")
	if !exists {
		t.Fatal("Scores field not found")
	}

	scoresArr := scores.([]interface{})

	// Should only have [10, 40, 50] remaining
	if len(scoresArr) != 3 {
		t.Errorf("Expected 3 scores after pullAll, got %d", len(scoresArr))
	}

	// Verify no 20s or 30s remain
	for _, score := range scoresArr {
		scoreNum, _ := toFloat64(score)
		if scoreNum == 20 || scoreNum == 30 {
			t.Errorf("Found %v in array after $pullAll", scoreNum)
		}
	}

	// Verify remaining values
	expectedScores := []float64{10, 40, 50}
	for i, expected := range expectedScores {
		actual, _ := toFloat64(scoresArr[i])
		if actual != expected {
			t.Errorf("At index %d: expected %v, got %v", i, expected, actual)
		}
	}
}

func TestPullAllWithStrings(t *testing.T) {
	// Create test database
	testDir := "./test_pullall_strings"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document with string array
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Frank",
		"tags": []interface{}{"go", "database", "mongodb", "go", "nosql", "database"},
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $pullAll - remove "go" and "database"
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Frank"},
		map[string]interface{}{
			"$pullAll": map[string]interface{}{
				"tags": []interface{}{"go", "database"},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to pullAll: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Frank"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	tags, _ := doc.Get("tags")
	tagsArr := tags.([]interface{})

	// Should only have ["mongodb", "nosql"]
	if len(tagsArr) != 2 {
		t.Errorf("Expected 2 tags after pullAll, got %d", len(tagsArr))
	}

	// Verify content
	if tagsArr[0] != "mongodb" || tagsArr[1] != "nosql" {
		t.Errorf("Expected [mongodb, nosql], got %v", tagsArr)
	}
}

func TestCombinedOperators(t *testing.T) {
	// Test using multiple operators together
	testDir := "./test_combined_ops"
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
		"name":      "Grace",
		"old_field": "value",
		"score":     100,
		"tags":      []interface{}{"a", "b", "c", "d"},
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Apply multiple operators in one update
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Grace"},
		map[string]interface{}{
			"$rename": map[string]interface{}{
				"old_field": "new_field",
			},
			"$inc": map[string]interface{}{
				"score": 50,
			},
			"$pullAll": map[string]interface{}{
				"tags": []interface{}{"a", "c"},
			},
			"$currentDate": map[string]interface{}{
				"updated": map[string]interface{}{
					"$type": "timestamp",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to apply combined operators: %v", err)
	}

	// Verify all changes
	doc, err := coll.FindOne(map[string]interface{}{"name": "Grace"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	// Check renamed field
	if _, exists := doc.Get("old_field"); exists {
		t.Error("old_field should not exist")
	}
	if newVal, exists := doc.Get("new_field"); !exists || newVal != "value" {
		t.Error("new_field should exist with value 'value'")
	}

	// Check incremented score
	score, _ := doc.Get("score")
	scoreNum, _ := toFloat64(score)
	if scoreNum != 150 {
		t.Errorf("Expected score=150, got %v", scoreNum)
	}

	// Check pullAll on tags
	tags, _ := doc.Get("tags")
	tagsArr := tags.([]interface{})
	if len(tagsArr) != 2 || tagsArr[0] != "b" || tagsArr[1] != "d" {
		t.Errorf("Expected tags=[b, d], got %v", tagsArr)
	}

	// Check currentDate set updated field
	if _, exists := doc.Get("updated"); !exists {
		t.Error("updated field should exist")
	}
}
