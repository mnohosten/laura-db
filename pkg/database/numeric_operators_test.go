package database

import (
	"os"
	"testing"
)

func TestNumericMulOperator(t *testing.T) {
	// Create test database
	testDir := "./test_numeric_mul"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document with numeric field
	_, err = coll.InsertOne(map[string]interface{}{
		"name":  "Product A",
		"price": 100.0,
		"qty":   5,
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $mul - multiply price by 1.5
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Product A"},
		map[string]interface{}{
			"$mul": map[string]interface{}{
				"price": 1.5,
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to multiply: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Product A"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	price, exists := doc.Get("price")
	if !exists {
		t.Fatal("Price field not found")
	}

	priceVal, _ := toFloat64(price)
	if priceVal != 150.0 {
		t.Errorf("Expected price to be 150.0, got %v", priceVal)
	}

	// Test $mul on non-existent field - should set to 0
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Product A"},
		map[string]interface{}{
			"$mul": map[string]interface{}{
				"discount": 0.9,
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to multiply non-existent field: %v", err)
	}

	doc, err = coll.FindOne(map[string]interface{}{"name": "Product A"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	discount, exists := doc.Get("discount")
	if !exists {
		t.Fatal("Discount field should exist after $mul")
	}

	discountVal, _ := toFloat64(discount)
	if discountVal != 0 {
		t.Errorf("Expected discount to be 0, got %v", discountVal)
	}
}

func TestNumericMinOperator(t *testing.T) {
	// Create test database
	testDir := "./test_numeric_min"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document with numeric fields
	_, err = coll.InsertOne(map[string]interface{}{
		"name":     "Player A",
		"highScore": 100,
		"lowScore":  50,
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $min - should update lowScore to 30 (30 < 50)
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Player A"},
		map[string]interface{}{
			"$min": map[string]interface{}{
				"lowScore": 30,
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to apply $min: %v", err)
	}

	// Verify lowScore was updated
	doc, err := coll.FindOne(map[string]interface{}{"name": "Player A"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	lowScore, _ := doc.Get("lowScore")
	lowScoreVal, _ := toFloat64(lowScore)
	if lowScoreVal != 30 {
		t.Errorf("Expected lowScore to be 30, got %v", lowScoreVal)
	}

	// Test $min - should NOT update lowScore (40 > 30)
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Player A"},
		map[string]interface{}{
			"$min": map[string]interface{}{
				"lowScore": 40,
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to apply $min: %v", err)
	}

	doc, err = coll.FindOne(map[string]interface{}{"name": "Player A"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	lowScore, _ = doc.Get("lowScore")
	lowScoreVal, _ = toFloat64(lowScore)
	if lowScoreVal != 30 {
		t.Errorf("Expected lowScore to remain 30, got %v", lowScoreVal)
	}

	// Test $min on non-existent field - should set to value
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Player A"},
		map[string]interface{}{
			"$min": map[string]interface{}{
				"minTime": 42,
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to apply $min on non-existent field: %v", err)
	}

	doc, err = coll.FindOne(map[string]interface{}{"name": "Player A"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	minTime, exists := doc.Get("minTime")
	if !exists {
		t.Fatal("minTime field should exist after $min")
	}

	minTimeVal, _ := toFloat64(minTime)
	if minTimeVal != 42 {
		t.Errorf("Expected minTime to be 42, got %v", minTimeVal)
	}
}

func TestNumericMaxOperator(t *testing.T) {
	// Create test database
	testDir := "./test_numeric_max"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document with numeric fields
	_, err = coll.InsertOne(map[string]interface{}{
		"name":     "Player B",
		"highScore": 100,
		"lowScore":  50,
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $max - should update highScore to 150 (150 > 100)
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Player B"},
		map[string]interface{}{
			"$max": map[string]interface{}{
				"highScore": 150,
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to apply $max: %v", err)
	}

	// Verify highScore was updated
	doc, err := coll.FindOne(map[string]interface{}{"name": "Player B"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	highScore, _ := doc.Get("highScore")
	highScoreVal, _ := toFloat64(highScore)
	if highScoreVal != 150 {
		t.Errorf("Expected highScore to be 150, got %v", highScoreVal)
	}

	// Test $max - should NOT update highScore (120 < 150)
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Player B"},
		map[string]interface{}{
			"$max": map[string]interface{}{
				"highScore": 120,
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to apply $max: %v", err)
	}

	doc, err = coll.FindOne(map[string]interface{}{"name": "Player B"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	highScore, _ = doc.Get("highScore")
	highScoreVal, _ = toFloat64(highScore)
	if highScoreVal != 150 {
		t.Errorf("Expected highScore to remain 150, got %v", highScoreVal)
	}

	// Test $max on non-existent field - should set to value
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Player B"},
		map[string]interface{}{
			"$max": map[string]interface{}{
				"maxLevel": 10,
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to apply $max on non-existent field: %v", err)
	}

	doc, err = coll.FindOne(map[string]interface{}{"name": "Player B"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	maxLevel, exists := doc.Get("maxLevel")
	if !exists {
		t.Fatal("maxLevel field should exist after $max")
	}

	maxLevelVal, _ := toFloat64(maxLevel)
	if maxLevelVal != 10 {
		t.Errorf("Expected maxLevel to be 10, got %v", maxLevelVal)
	}
}

func TestNumericOperatorsCombined(t *testing.T) {
	// Create test database
	testDir := "./test_numeric_combined"
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
		"name":   "Stats",
		"count":  10,
		"total":  100,
		"min":    5,
		"max":    50,
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test multiple operators in one update
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Stats"},
		map[string]interface{}{
			"$inc": map[string]interface{}{
				"count": 1,
			},
			"$mul": map[string]interface{}{
				"total": 2,
			},
			"$min": map[string]interface{}{
				"min": 3,
			},
			"$max": map[string]interface{}{
				"max": 60,
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to apply combined operators: %v", err)
	}

	// Verify all changes
	doc, err := coll.FindOne(map[string]interface{}{"name": "Stats"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	count, _ := doc.Get("count")
	countVal, _ := toFloat64(count)
	if countVal != 11 {
		t.Errorf("Expected count to be 11, got %v", countVal)
	}

	total, _ := doc.Get("total")
	totalVal, _ := toFloat64(total)
	if totalVal != 200 {
		t.Errorf("Expected total to be 200, got %v", totalVal)
	}

	min, _ := doc.Get("min")
	minVal, _ := toFloat64(min)
	if minVal != 3 {
		t.Errorf("Expected min to be 3, got %v", minVal)
	}

	max, _ := doc.Get("max")
	maxVal, _ := toFloat64(max)
	if maxVal != 60 {
		t.Errorf("Expected max to be 60, got %v", maxVal)
	}
}
