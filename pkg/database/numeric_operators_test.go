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

func TestBitOperatorAnd(t *testing.T) {
	// Create test database
	testDir := "./test_bit_and"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document with integer field
	// Binary: 13 = 1101, 9 = 1001, 13 & 9 = 1001 = 9
	_, err = coll.InsertOne(map[string]interface{}{
		"name":  "Bitwise Test",
		"flags": int64(13),
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $bit with AND operation
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Bitwise Test"},
		map[string]interface{}{
			"$bit": map[string]interface{}{
				"flags": map[string]interface{}{
					"and": int64(9),
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to apply $bit and: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Bitwise Test"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	flags, exists := doc.Get("flags")
	if !exists {
		t.Fatal("Flags field not found")
	}

	flagsVal, _ := toInt64(flags)
	if flagsVal != 9 {
		t.Errorf("Expected flags to be 9 (13 & 9), got %v", flagsVal)
	}
}

func TestBitOperatorOr(t *testing.T) {
	// Create test database
	testDir := "./test_bit_or"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document with integer field
	// Binary: 12 = 1100, 10 = 1010, 12 | 10 = 1110 = 14
	_, err = coll.InsertOne(map[string]interface{}{
		"name":  "Bitwise Test",
		"flags": int64(12),
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $bit with OR operation
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Bitwise Test"},
		map[string]interface{}{
			"$bit": map[string]interface{}{
				"flags": map[string]interface{}{
					"or": int64(10),
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to apply $bit or: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Bitwise Test"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	flags, exists := doc.Get("flags")
	if !exists {
		t.Fatal("Flags field not found")
	}

	flagsVal, _ := toInt64(flags)
	if flagsVal != 14 {
		t.Errorf("Expected flags to be 14 (12 | 10), got %v", flagsVal)
	}
}

func TestBitOperatorXor(t *testing.T) {
	// Create test database
	testDir := "./test_bit_xor"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document with integer field
	// Binary: 15 = 1111, 9 = 1001, 15 ^ 9 = 0110 = 6
	_, err = coll.InsertOne(map[string]interface{}{
		"name":  "Bitwise Test",
		"flags": int64(15),
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $bit with XOR operation
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Bitwise Test"},
		map[string]interface{}{
			"$bit": map[string]interface{}{
				"flags": map[string]interface{}{
					"xor": int64(9),
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to apply $bit xor: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Bitwise Test"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	flags, exists := doc.Get("flags")
	if !exists {
		t.Fatal("Flags field not found")
	}

	flagsVal, _ := toInt64(flags)
	if flagsVal != 6 {
		t.Errorf("Expected flags to be 6 (15 ^ 9), got %v", flagsVal)
	}
}

func TestBitOperatorCombined(t *testing.T) {
	// Create test database
	testDir := "./test_bit_combined"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document with integer field
	// Start with 255 = 11111111
	_, err = coll.InsertOne(map[string]interface{}{
		"name":  "Bitwise Test",
		"flags": int64(255),
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $bit with multiple operations in sequence (and, or, xor)
	// 255 & 127 = 127 (01111111)
	// 127 | 64 = 127 (01111111)
	// 127 ^ 15 = 112 (01110000)
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Bitwise Test"},
		map[string]interface{}{
			"$bit": map[string]interface{}{
				"flags": map[string]interface{}{
					"and": int64(127),
					"or":  int64(64),
					"xor": int64(15),
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to apply $bit combined: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Bitwise Test"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	flags, exists := doc.Get("flags")
	if !exists {
		t.Fatal("Flags field not found")
	}

	flagsVal, _ := toInt64(flags)
	expected := int64(112) // ((255 & 127) | 64) ^ 15 = 112
	if flagsVal != expected {
		t.Errorf("Expected flags to be %v, got %v", expected, flagsVal)
	}
}

func TestBitOperatorNonExistentField(t *testing.T) {
	// Create test database
	testDir := "./test_bit_nonexistent"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Insert document without flags field
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Bitwise Test",
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test $bit with OR on non-existent field (should initialize to 0, then OR)
	// 0 | 15 = 15
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Bitwise Test"},
		map[string]interface{}{
			"$bit": map[string]interface{}{
				"flags": map[string]interface{}{
					"or": int64(15),
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to apply $bit on non-existent field: %v", err)
	}

	// Verify
	doc, err := coll.FindOne(map[string]interface{}{"name": "Bitwise Test"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	flags, exists := doc.Get("flags")
	if !exists {
		t.Fatal("Flags field should exist after $bit")
	}

	flagsVal, _ := toInt64(flags)
	if flagsVal != 15 {
		t.Errorf("Expected flags to be 15 (0 | 15), got %v", flagsVal)
	}
}

func TestBitOperatorPermissions(t *testing.T) {
	// Create test database
	testDir := "./test_bit_permissions"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Practical example: file permissions using bitwise operations
	// Read = 4, Write = 2, Execute = 1
	const (
		READ    = int64(4)
		WRITE   = int64(2)
		EXECUTE = int64(1)
	)

	// Insert user with read+write permissions (6)
	_, err = coll.InsertOne(map[string]interface{}{
		"username":    "alice",
		"permissions": READ | WRITE, // 6
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Add execute permission
	err = coll.UpdateOne(
		map[string]interface{}{"username": "alice"},
		map[string]interface{}{
			"$bit": map[string]interface{}{
				"permissions": map[string]interface{}{
					"or": EXECUTE,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to add execute permission: %v", err)
	}

	// Verify permissions = 7 (read+write+execute)
	doc, err := coll.FindOne(map[string]interface{}{"username": "alice"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	perms, _ := doc.Get("permissions")
	permsVal, _ := toInt64(perms)
	if permsVal != 7 {
		t.Errorf("Expected permissions to be 7, got %v", permsVal)
	}

	// Remove write permission using AND with NOT pattern
	// To remove write (2), we AND with ~2 = -3 in two's complement
	// But for cleaner approach, use AND with (READ | EXECUTE) = 5
	err = coll.UpdateOne(
		map[string]interface{}{"username": "alice"},
		map[string]interface{}{
			"$bit": map[string]interface{}{
				"permissions": map[string]interface{}{
					"and": READ | EXECUTE, // 5
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to remove write permission: %v", err)
	}

	// Verify permissions = 5 (read+execute)
	doc, err = coll.FindOne(map[string]interface{}{"username": "alice"})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	perms, _ = doc.Get("permissions")
	permsVal, _ = toInt64(perms)
	if permsVal != 5 {
		t.Errorf("Expected permissions to be 5, got %v", permsVal)
	}
}

// Benchmarks for $bit operator

func BenchmarkBitOperatorAnd(b *testing.B) {
	testDir := "./bench_bit_and"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("bench")

	// Insert test document
	_, err = coll.InsertOne(map[string]interface{}{
		"id":    1,
		"flags": int64(255),
	})
	if err != nil {
		b.Fatalf("Failed to insert: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := coll.UpdateOne(
			map[string]interface{}{"id": 1},
			map[string]interface{}{
				"$bit": map[string]interface{}{
					"flags": map[string]interface{}{
						"and": int64(127),
					},
				},
			},
		)
		if err != nil {
			b.Fatalf("Failed to update: %v", err)
		}
	}
}

func BenchmarkBitOperatorOr(b *testing.B) {
	testDir := "./bench_bit_or"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("bench")

	// Insert test document
	_, err = coll.InsertOne(map[string]interface{}{
		"id":    1,
		"flags": int64(128),
	})
	if err != nil {
		b.Fatalf("Failed to insert: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := coll.UpdateOne(
			map[string]interface{}{"id": 1},
			map[string]interface{}{
				"$bit": map[string]interface{}{
					"flags": map[string]interface{}{
						"or": int64(15),
					},
				},
			},
		)
		if err != nil {
			b.Fatalf("Failed to update: %v", err)
		}
	}
}

func BenchmarkBitOperatorXor(b *testing.B) {
	testDir := "./bench_bit_xor"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("bench")

	// Insert test document
	_, err = coll.InsertOne(map[string]interface{}{
		"id":    1,
		"flags": int64(100),
	})
	if err != nil {
		b.Fatalf("Failed to insert: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := coll.UpdateOne(
			map[string]interface{}{"id": 1},
			map[string]interface{}{
				"$bit": map[string]interface{}{
					"flags": map[string]interface{}{
						"xor": int64(50),
					},
				},
			},
		)
		if err != nil {
			b.Fatalf("Failed to update: %v", err)
		}
	}
}

func BenchmarkBitOperatorCombined(b *testing.B) {
	testDir := "./bench_bit_combined"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("bench")

	// Insert test document
	_, err = coll.InsertOne(map[string]interface{}{
		"id":    1,
		"flags": int64(255),
	})
	if err != nil {
		b.Fatalf("Failed to insert: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := coll.UpdateOne(
			map[string]interface{}{"id": 1},
			map[string]interface{}{
				"$bit": map[string]interface{}{
					"flags": map[string]interface{}{
						"and": int64(127),
						"or":  int64(64),
						"xor": int64(15),
					},
				},
			},
		)
		if err != nil {
			b.Fatalf("Failed to update: %v", err)
		}
	}
}
