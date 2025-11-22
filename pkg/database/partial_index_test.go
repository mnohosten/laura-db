package database

import (
	"os"
	"testing"
)

func TestCreatePartialIndex(t *testing.T) {
	dir := "./test_partial_create"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("users")

	// Insert test documents
	coll.InsertOne(map[string]interface{}{
		"name":   "Alice",
		"age":    25,
		"active": true,
	})
	coll.InsertOne(map[string]interface{}{
		"name":   "Bob",
		"age":    35,
		"active": false,
	})
	coll.InsertOne(map[string]interface{}{
		"name":   "Charlie",
		"age":    45,
		"active": true,
	})

	// Create partial index: only index active users
	filter := map[string]interface{}{
		"active": true,
	}
	err = coll.CreatePartialIndex("name", filter, false)
	if err != nil {
		t.Fatalf("Failed to create partial index: %v", err)
	}

	// Verify index exists and shows as partial
	indexes := coll.ListIndexes()
	found := false
	for _, idx := range indexes {
		if idx["name"] == "name_partial" {
			found = true
			if !idx["is_partial"].(bool) {
				t.Error("Expected index to be marked as partial")
			}
			// Check index size - should only have 2 docs (active users)
			if idx["size"].(int) != 2 {
				t.Errorf("Expected 2 docs in partial index, got %d", idx["size"])
			}
			break
		}
	}

	if !found {
		t.Error("Partial index not found in index list")
	}
}

func TestPartialIndexInsert(t *testing.T) {
	dir := "./test_partial_insert"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("products")

	// Create partial index: only index products in stock
	filter := map[string]interface{}{
		"inStock": true,
	}
	err := coll.CreatePartialIndex("sku", filter, false)
	if err != nil {
		t.Fatalf("Failed to create partial index: %v", err)
	}

	// Insert document that matches filter - should be indexed
	coll.InsertOne(map[string]interface{}{
		"sku":     "ABC123",
		"name":    "Widget",
		"inStock": true,
	})

	// Insert document that doesn't match filter - should NOT be indexed
	coll.InsertOne(map[string]interface{}{
		"sku":     "DEF456",
		"name":    "Gadget",
		"inStock": false,
	})

	// Check index size
	indexes := coll.ListIndexes()
	for _, idx := range indexes {
		if idx["name"] == "sku_partial" {
			if idx["size"].(int) != 1 {
				t.Errorf("Expected 1 doc in partial index, got %d", idx["size"])
			}
			break
		}
	}
}

func TestPartialIndexUpdate(t *testing.T) {
	dir := "./test_partial_update"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("tasks")

	// Create partial index: only index incomplete tasks
	filter := map[string]interface{}{
		"completed": false,
	}
	coll.CreatePartialIndex("priority", filter, false)

	// Insert incomplete task
	coll.InsertOne(map[string]interface{}{
		"title":     "Task 1",
		"priority":  1,
		"completed": false,
	})

	// Verify it's in the index
	indexes := coll.ListIndexes()
	for _, idx := range indexes {
		if idx["name"] == "priority_partial" {
			if idx["size"].(int) != 1 {
				t.Errorf("Expected 1 doc before update, got %d", idx["size"])
			}
		}
	}

	// Mark task as completed - should remove from partial index
	coll.UpdateOne(
		map[string]interface{}{"title": "Task 1"},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"completed": true,
			},
		},
	)

	// Verify it's removed from index
	indexes = coll.ListIndexes()
	for _, idx := range indexes {
		if idx["name"] == "priority_partial" {
			if idx["size"].(int) != 0 {
				t.Errorf("Expected 0 docs after update, got %d", idx["size"])
			}
		}
	}
}

func TestPartialIndexWithComplexFilter(t *testing.T) {
	dir := "./test_partial_complex"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("orders")

	// Create partial index with complex filter: high-value pending orders
	filter := map[string]interface{}{
		"status": "pending",
		"amount": map[string]interface{}{
			"$gt": 100,
		},
	}
	coll.CreatePartialIndex("orderNumber", filter, false)

	// Insert various orders
	coll.InsertOne(map[string]interface{}{
		"orderNumber": "ORD001",
		"status":      "pending",
		"amount":      150, // Matches filter
	})

	coll.InsertOne(map[string]interface{}{
		"orderNumber": "ORD002",
		"status":      "pending",
		"amount":      50, // Doesn't match (amount too low)
	})

	coll.InsertOne(map[string]interface{}{
		"orderNumber": "ORD003",
		"status":      "completed",
		"amount":      200, // Doesn't match (wrong status)
	})

	coll.InsertOne(map[string]interface{}{
		"orderNumber": "ORD004",
		"status":      "pending",
		"amount":      500, // Matches filter
	})

	// Check index size - should have 2 docs (ORD001 and ORD004)
	indexes := coll.ListIndexes()
	for _, idx := range indexes {
		if idx["name"] == "orderNumber_partial" {
			if idx["size"].(int) != 2 {
				t.Errorf("Expected 2 docs in partial index, got %d", idx["size"])
			}
		}
	}
}

func TestPartialIndexUnique(t *testing.T) {
	dir := "./test_partial_unique"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("employees")

	// Create unique partial index: only enforce unique email for active employees
	filter := map[string]interface{}{
		"status": "active",
	}
	coll.CreatePartialIndex("email", filter, true)

	// Insert active employee
	_, err := coll.InsertOne(map[string]interface{}{
		"name":   "Alice",
		"email":  "alice@example.com",
		"status": "active",
	})
	if err != nil {
		t.Fatalf("Failed to insert first active employee: %v", err)
	}

	// Try to insert another active employee with same email - should fail
	_, err = coll.InsertOne(map[string]interface{}{
		"name":   "Alice Clone",
		"email":  "alice@example.com",
		"status": "active",
	})
	if err == nil {
		t.Error("Expected unique constraint violation for active employees")
	}

	// Insert inactive employee with same email - should succeed (not in partial index)
	_, err = coll.InsertOne(map[string]interface{}{
		"name":   "Alice Inactive",
		"email":  "alice@example.com",
		"status": "inactive",
	})
	if err != nil {
		t.Errorf("Should allow duplicate email for inactive employee: %v", err)
	}
}

func TestPartialIndexDelete(t *testing.T) {
	dir := "./test_partial_delete"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("items")

	// Create partial index
	filter := map[string]interface{}{
		"visible": true,
	}
	coll.CreatePartialIndex("code", filter, false)

	// Insert matching document
	coll.InsertOne(map[string]interface{}{
		"code":    "ITEM001",
		"visible": true,
	})

	// Verify it's in index
	indexes := coll.ListIndexes()
	for _, idx := range indexes {
		if idx["name"] == "code_partial" {
			if idx["size"].(int) != 1 {
				t.Fatalf("Expected 1 doc before delete, got %d", idx["size"])
			}
		}
	}

	// Delete the document
	coll.DeleteOne(map[string]interface{}{"code": "ITEM001"})

	// Verify it's removed from index
	indexes = coll.ListIndexes()
	for _, idx := range indexes {
		if idx["name"] == "code_partial" {
			if idx["size"].(int) != 0 {
				t.Errorf("Expected 0 docs after delete, got %d", idx["size"])
			}
		}
	}
}

func TestPartialIndexWithLogicalOperators(t *testing.T) {
	dir := "./test_partial_logical"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("accounts")

	// Create partial index with $or operator
	filter := map[string]interface{}{
		"$or": []interface{}{
			map[string]interface{}{"type": "premium"},
			map[string]interface{}{"verified": true},
		},
	}
	coll.CreatePartialIndex("accountId", filter, false)

	// Insert various accounts
	coll.InsertOne(map[string]interface{}{
		"accountId": "ACC001",
		"type":      "premium",
		"verified":  false, // Matches (premium)
	})

	coll.InsertOne(map[string]interface{}{
		"accountId": "ACC002",
		"type":      "basic",
		"verified":  true, // Matches (verified)
	})

	coll.InsertOne(map[string]interface{}{
		"accountId": "ACC003",
		"type":      "basic",
		"verified":  false, // Doesn't match
	})

	// Check index size - should have 2 docs
	indexes := coll.ListIndexes()
	for _, idx := range indexes {
		if idx["name"] == "accountId_partial" {
			if idx["size"].(int) != 2 {
				t.Errorf("Expected 2 docs in partial index, got %d", idx["size"])
			}
		}
	}
}

func TestPartialIndexEmptyFilter(t *testing.T) {
	dir := "./test_partial_empty"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("test")

	// Try to create partial index with empty filter - should fail
	err := coll.CreatePartialIndex("field", map[string]interface{}{}, false)
	if err == nil {
		t.Error("Expected error when creating partial index with empty filter")
	}
}

func TestDropPartialIndex(t *testing.T) {
	dir := "./test_partial_drop"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("test")

	// Create partial index
	filter := map[string]interface{}{"active": true}
	coll.CreatePartialIndex("name", filter, false)

	// Verify it exists
	indexes := coll.ListIndexes()
	found := false
	for _, idx := range indexes {
		if idx["name"] == "name_partial" {
			found = true
		}
	}
	if !found {
		t.Fatal("Partial index not created")
	}

	// Drop the index
	err := coll.DropIndex("name_partial")
	if err != nil {
		t.Fatalf("Failed to drop partial index: %v", err)
	}

	// Verify it's gone
	indexes = coll.ListIndexes()
	for _, idx := range indexes {
		if idx["name"] == "name_partial" {
			t.Error("Partial index still exists after drop")
		}
	}
}

func TestPartialIndexUpdateMany(t *testing.T) {
	dir := "./test_partial_update_many"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("documents")

	// Create partial index: only published documents
	filter := map[string]interface{}{"published": true}
	coll.CreatePartialIndex("title", filter, false)

	// Insert documents
	coll.InsertOne(map[string]interface{}{
		"title":     "Doc 1",
		"published": true,
	})
	coll.InsertOne(map[string]interface{}{
		"title":     "Doc 2",
		"published": true,
	})
	coll.InsertOne(map[string]interface{}{
		"title":     "Doc 3",
		"published": false,
	})

	// Verify 2 docs in index
	indexes := coll.ListIndexes()
	for _, idx := range indexes {
		if idx["name"] == "title_partial" {
			if idx["size"].(int) != 2 {
				t.Fatalf("Expected 2 docs initially, got %d", idx["size"])
			}
		}
	}

	// Unpublish all documents
	coll.UpdateMany(
		map[string]interface{}{},
		map[string]interface{}{
			"$set": map[string]interface{}{"published": false},
		},
	)

	// Verify index is now empty
	indexes = coll.ListIndexes()
	for _, idx := range indexes {
		if idx["name"] == "title_partial" {
			if idx["size"].(int) != 0 {
				t.Errorf("Expected 0 docs after update, got %d", idx["size"])
			}
		}
	}
}
