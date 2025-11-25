package database

import (
	"os"
	"testing"
)

func TestBulkWrite_Insert(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "lauradb-bulk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Open database
	db, err := Open(DefaultConfig(tempDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get collection
	coll := db.Collection("users")

	// Create bulk operations with multiple inserts
	operations := []BulkOperation{
		{
			Type: "insert",
			Document: map[string]interface{}{
				"name": "Alice",
				"age":  int64(30),
			},
		},
		{
			Type: "insert",
			Document: map[string]interface{}{
				"name": "Bob",
				"age":  int64(25),
			},
		},
		{
			Type: "insert",
			Document: map[string]interface{}{
				"name": "Charlie",
				"age":  int64(35),
			},
		},
	}

	// Execute bulk write
	result, err := coll.BulkWrite(operations, true)
	if err != nil {
		t.Fatalf("BulkWrite failed: %v", err)
	}

	// Verify results
	if result.InsertedCount != 3 {
		t.Errorf("Expected 3 inserts, got %d", result.InsertedCount)
	}
	if len(result.InsertedIds) != 3 {
		t.Errorf("Expected 3 inserted IDs, got %d", len(result.InsertedIds))
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(result.Errors), result.Errors)
	}

	// Verify documents were inserted
	count, err := coll.Count(nil)
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 documents in collection, got %d", count)
	}
}

func TestBulkWrite_Update(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "lauradb-bulk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Open database
	db, err := Open(DefaultConfig(tempDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get collection
	coll := db.Collection("users")

	// Insert test documents
	coll.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30)})
	coll.InsertOne(map[string]interface{}{"name": "Bob", "age": int64(25)})
	coll.InsertOne(map[string]interface{}{"name": "Charlie", "age": int64(35)})

	// Create bulk operations with updates
	operations := []BulkOperation{
		{
			Type:   "update",
			Filter: map[string]interface{}{"name": "Alice"},
			Update: map[string]interface{}{"$set": map[string]interface{}{"age": int64(31)}},
		},
		{
			Type:   "update",
			Filter: map[string]interface{}{"name": "Bob"},
			Update: map[string]interface{}{"$set": map[string]interface{}{"age": int64(26)}},
		},
	}

	// Execute bulk write
	result, err := coll.BulkWrite(operations, true)
	if err != nil {
		t.Fatalf("BulkWrite failed: %v", err)
	}

	// Verify results
	if result.ModifiedCount != 2 {
		t.Errorf("Expected 2 updates, got %d", result.ModifiedCount)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(result.Errors), result.Errors)
	}

	// Verify documents were updated
	alice, err := coll.FindOne(map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("Failed to find Alice: %v", err)
	}
	if age, _ := alice.Get("age"); age != int64(31) {
		t.Errorf("Expected Alice's age to be 31, got %v", age)
	}

	bob, err := coll.FindOne(map[string]interface{}{"name": "Bob"})
	if err != nil {
		t.Fatalf("Failed to find Bob: %v", err)
	}
	if age, _ := bob.Get("age"); age != int64(26) {
		t.Errorf("Expected Bob's age to be 26, got %v", age)
	}
}

func TestBulkWrite_Delete(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "lauradb-bulk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Open database
	db, err := Open(DefaultConfig(tempDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get collection
	coll := db.Collection("users")

	// Insert test documents
	coll.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30)})
	coll.InsertOne(map[string]interface{}{"name": "Bob", "age": int64(25)})
	coll.InsertOne(map[string]interface{}{"name": "Charlie", "age": int64(35)})

	// Create bulk operations with deletes
	operations := []BulkOperation{
		{
			Type:   "delete",
			Filter: map[string]interface{}{"name": "Alice"},
		},
		{
			Type:   "delete",
			Filter: map[string]interface{}{"name": "Bob"},
		},
	}

	// Execute bulk write
	result, err := coll.BulkWrite(operations, true)
	if err != nil {
		t.Fatalf("BulkWrite failed: %v", err)
	}

	// Verify results
	if result.DeletedCount != 2 {
		t.Errorf("Expected 2 deletes, got %d", result.DeletedCount)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(result.Errors), result.Errors)
	}

	// Verify documents were deleted
	count, err := coll.Count(nil)
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 document remaining, got %d", count)
	}
}

func TestBulkWrite_Mixed(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "lauradb-bulk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Open database
	db, err := Open(DefaultConfig(tempDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get collection
	coll := db.Collection("users")

	// Insert initial document
	coll.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30)})

	// Create bulk operations with mixed operations
	operations := []BulkOperation{
		{
			Type: "insert",
			Document: map[string]interface{}{
				"name": "Bob",
				"age":  int64(25),
			},
		},
		{
			Type:   "update",
			Filter: map[string]interface{}{"name": "Alice"},
			Update: map[string]interface{}{"$set": map[string]interface{}{"age": int64(31)}},
		},
		{
			Type: "insert",
			Document: map[string]interface{}{
				"name": "Charlie",
				"age":  int64(35),
			},
		},
		{
			Type:   "delete",
			Filter: map[string]interface{}{"name": "Bob"},
		},
	}

	// Execute bulk write
	result, err := coll.BulkWrite(operations, true)
	if err != nil {
		t.Fatalf("BulkWrite failed: %v", err)
	}

	// Verify results
	if result.InsertedCount != 2 {
		t.Errorf("Expected 2 inserts, got %d", result.InsertedCount)
	}
	if result.ModifiedCount != 1 {
		t.Errorf("Expected 1 update, got %d", result.ModifiedCount)
	}
	if result.DeletedCount != 1 {
		t.Errorf("Expected 1 delete, got %d", result.DeletedCount)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(result.Errors), result.Errors)
	}

	// Verify final state
	count, err := coll.Count(nil)
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 documents remaining, got %d", count)
	}
}

func TestBulkWrite_OrderedStopsOnError(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "lauradb-bulk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Open database
	db, err := Open(DefaultConfig(tempDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get collection
	coll := db.Collection("users")

	// Create bulk operations with an invalid operation in the middle
	operations := []BulkOperation{
		{
			Type: "insert",
			Document: map[string]interface{}{
				"name": "Alice",
				"age":  int64(30),
			},
		},
		{
			Type: "insert",
			Document: map[string]interface{}{
				"name": "Bob",
				"age":  int64(25),
			},
		},
		{
			Type: "update",
			// Missing Filter - this will cause an error
			Update: map[string]interface{}{"$set": map[string]interface{}{"age": int64(35)}},
		},
		{
			Type: "insert",
			Document: map[string]interface{}{
				"name": "David",
				"age":  int64(40),
			},
		},
	}

	// Execute bulk write in ordered mode
	result, err := coll.BulkWrite(operations, true)
	if err == nil {
		t.Fatal("Expected error for invalid update operation, got nil")
	}

	// Should have inserted 2 documents (Alice, Bob) before hitting the error
	if result.InsertedCount != 2 {
		t.Errorf("Expected 2 inserts before error, got %d", result.InsertedCount)
	}

	// Should have 1 error
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	// Fourth operation (David) should not have been executed due to ordered mode
	count, err := coll.Count(nil)
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 documents (ordered mode stops on error), got %d", count)
	}
}

func TestBulkWrite_UnorderedContinuesOnError(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "lauradb-bulk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Open database
	db, err := Open(DefaultConfig(tempDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get collection
	coll := db.Collection("users")

	// Create bulk operations with an invalid operation in the middle
	operations := []BulkOperation{
		{
			Type: "insert",
			Document: map[string]interface{}{
				"name": "Alice",
				"age":  int64(30),
			},
		},
		{
			Type: "insert",
			Document: map[string]interface{}{
				"name": "Bob",
				"age":  int64(25),
			},
		},
		{
			Type: "update",
			// Missing Filter - this will cause an error
			Update: map[string]interface{}{"$set": map[string]interface{}{"age": int64(35)}},
		},
		{
			Type: "insert",
			Document: map[string]interface{}{
				"name": "David",
				"age":  int64(40),
			},
		},
	}

	// Execute bulk write in unordered mode
	result, err := coll.BulkWrite(operations, false)
	if err == nil {
		t.Fatal("Expected error for invalid update operation, got nil")
	}

	// In unordered mode, should continue after error
	// Should have inserted 3 documents (Alice, Bob, David) - all valid operations
	if result.InsertedCount != 3 {
		t.Errorf("Expected 3 inserts (unordered continues on error), got %d", result.InsertedCount)
	}

	// Should have 1 error
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	// All valid operations should have been executed
	count, err := coll.Count(nil)
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 documents (unordered mode continues on error), got %d", count)
	}
}

func TestBulkWrite_InvalidOperation(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "lauradb-bulk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Open database
	db, err := Open(DefaultConfig(tempDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get collection
	coll := db.Collection("users")

	// Create bulk operations with invalid operation type
	operations := []BulkOperation{
		{
			Type: "invalid",
		},
	}

	// Execute bulk write
	result, err := coll.BulkWrite(operations, true)
	if err == nil {
		t.Fatal("Expected error for invalid operation type, got nil")
	}

	// Should have 1 error
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestBulkWrite_EmptyOperations(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "lauradb-bulk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Open database
	db, err := Open(DefaultConfig(tempDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get collection
	coll := db.Collection("users")

	// Create empty bulk operations
	operations := []BulkOperation{}

	// Execute bulk write
	result, err := coll.BulkWrite(operations, true)
	if err != nil {
		t.Fatalf("BulkWrite failed: %v", err)
	}

	// Verify all counts are zero
	if result.InsertedCount != 0 || result.ModifiedCount != 0 || result.DeletedCount != 0 {
		t.Errorf("Expected all counts to be 0, got: inserted=%d, modified=%d, deleted=%d",
			result.InsertedCount, result.ModifiedCount, result.DeletedCount)
	}
}
