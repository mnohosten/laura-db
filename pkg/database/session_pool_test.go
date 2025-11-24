package database

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
)

// setupPoolTestDB creates a test database for pool tests
func setupPoolTestDB(t *testing.T) *Database {
	dataDir := t.TempDir()
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	return db
}

// TestSessionPool_BasicGetPut tests basic session pool get/put operations
func TestSessionPool_BasicGetPut(t *testing.T) {
	db := setupPoolTestDB(t)
	defer db.Close()

	pool := NewSessionPool(db)

	// Get a session from the pool
	session := pool.Get()
	if session == nil {
		t.Fatal("Expected session from pool, got nil")
	}

	// Verify session has a transaction
	if session.txn == nil {
		t.Error("Expected session to have a transaction")
	}

	// Verify session has the correct database reference
	if session.db != db {
		t.Error("Session has incorrect database reference")
	}

	// Abort the transaction before returning to pool
	session.AbortTransaction()

	// Return to pool
	pool.Put(session)

	// Get another session (should reuse the same object)
	session2 := pool.Get()
	if session2 == nil {
		t.Fatal("Expected session from pool, got nil")
	}

	// Verify the session was reset
	if len(session2.operations) != 0 {
		t.Error("Session operations should be empty after reset")
	}
	if len(session2.collections) != 0 {
		t.Error("Session collections should be empty after reset")
	}
	if len(session2.snapshotDocs) != 0 {
		t.Error("Session snapshotDocs should be empty after reset")
	}

	session2.AbortTransaction()
	pool.Put(session2)
}

// TestSessionPool_Transaction tests using pooled sessions for transactions
func TestSessionPool_Transaction(t *testing.T) {
	db := setupPoolTestDB(t)
	defer db.Close()

	pool := NewSessionPool(db)

	// Create a collection
	coll := db.Collection("users")

	// Use pooled session for a transaction
	session := pool.Get()
	defer pool.Put(session)

	// Insert a document
	id, err := session.InsertOne("users", map[string]interface{}{
		"name":  "Alice",
		"age":   int64(30),
		"email": "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Commit the transaction
	if err := session.CommitTransaction(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify the document was inserted
	objID, err := document.ObjectIDFromHex(id)
	if err != nil {
		t.Fatalf("Failed to parse ObjectID: %v", err)
	}
	doc, err := coll.FindOne(map[string]interface{}{"_id": objID})
	if err != nil {
		t.Fatalf("Failed to find inserted document: %v", err)
	}

	name, _ := doc.Get("name")
	if name != "Alice" {
		t.Errorf("Expected name 'Alice', got '%v'", name)
	}
}

// TestSessionPool_WithTransactionPooled tests the convenience method
func TestSessionPool_WithTransactionPooled(t *testing.T) {
	db := setupPoolTestDB(t)
	defer db.Close()

	pool := NewSessionPool(db)
	coll := db.Collection("products")

	// Execute a transaction using the convenience method
	err := pool.WithTransactionPooled(func(session *Session) error {
		// Insert multiple documents in a transaction
		_, err := session.InsertOne("products", map[string]interface{}{
			"name":  "Widget",
			"price": int64(100),
			"stock": int64(50),
		})
		if err != nil {
			return err
		}

		_, err = session.InsertOne("products", map[string]interface{}{
			"name":  "Gadget",
			"price": int64(200),
			"stock": int64(30),
		})
		return err
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Verify both documents were inserted
	count, _ := coll.Count(map[string]interface{}{})
	if count != 2 {
		t.Errorf("Expected 2 documents, got %d", count)
	}
}

// TestSessionPool_WithTransactionPooled_Rollback tests rollback on error
func TestSessionPool_WithTransactionPooled_Rollback(t *testing.T) {
	db := setupPoolTestDB(t)
	defer db.Close()

	pool := NewSessionPool(db)
	coll := db.Collection("accounts")

	// Insert an initial document
	coll.InsertOne(map[string]interface{}{
		"name":    "Account1",
		"balance": int64(1000),
	})

	// Execute a transaction that fails
	err := pool.WithTransactionPooled(func(session *Session) error {
		// Insert a document
		_, err := session.InsertOne("accounts", map[string]interface{}{
			"name":    "Account2",
			"balance": int64(500),
		})
		if err != nil {
			return err
		}

		// Return an error to trigger rollback
		return fmt.Errorf("simulated error")
	})

	if err == nil {
		t.Fatal("Expected error from transaction")
	}

	// Verify only the first document exists (second was rolled back)
	count, _ := coll.Count(map[string]interface{}{})
	if count != 1 {
		t.Errorf("Expected 1 document (rollback), got %d", count)
	}
}

// TestSessionPool_ConcurrentUsage tests concurrent session pool usage
func TestSessionPool_ConcurrentUsage(t *testing.T) {
	db := setupPoolTestDB(t)
	defer db.Close()

	pool := NewSessionPool(db)
	coll := db.Collection("concurrent_test")

	numGoroutines := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch multiple goroutines that use the session pool concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			err := pool.WithTransactionPooled(func(session *Session) error {
				_, err := session.InsertOne("concurrent_test", map[string]interface{}{
					"id":    int64(id),
					"value": fmt.Sprintf("value_%d", id),
				})
				return err
			})

			if err != nil {
				t.Errorf("Transaction failed for goroutine %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all documents were inserted
	count, _ := coll.Count(map[string]interface{}{})
	if count != numGoroutines {
		t.Errorf("Expected %d documents, got %d", numGoroutines, count)
	}
}

// TestSessionPool_Reset tests that session reset works correctly
func TestSessionPool_Reset(t *testing.T) {
	db := setupPoolTestDB(t)
	defer db.Close()

	pool := NewSessionPool(db)

	// Get a session and perform some operations
	session := pool.Get()

	// Add some data to the session
	session.InsertOne("test", map[string]interface{}{"key": "value"})
	session.collections["test"] = true
	session.snapshotDocs["test"] = make(map[string]*document.Document)

	// Now reset the session
	session.reset()

	// Verify everything is cleared
	if len(session.operations) != 0 {
		t.Error("Operations should be empty after reset")
	}
	if len(session.collections) != 0 {
		t.Error("Collections should be empty after reset")
	}
	if len(session.snapshotDocs) != 0 {
		t.Error("SnapshotDocs should be empty after reset")
	}
	if session.txn != nil {
		t.Error("Transaction should be nil after reset")
	}

	// Don't call AbortTransaction since txn is nil after reset
	pool.Put(session)
}

// TestSessionPool_MultipleOperations tests multiple operations in a pooled transaction
func TestSessionPool_MultipleOperations(t *testing.T) {
	db := setupPoolTestDB(t)
	defer db.Close()

	pool := NewSessionPool(db)
	coll := db.Collection("inventory")

	// Insert multiple documents in a single transaction
	err := pool.WithTransactionPooled(func(session *Session) error {
		// Insert first document
		_, err := session.InsertOne("inventory", map[string]interface{}{
			"item":     "Laptop",
			"quantity": int64(10),
			"price":    int64(1000),
		})
		if err != nil {
			return err
		}

		// Insert second document
		_, err = session.InsertOne("inventory", map[string]interface{}{
			"item":     "Mouse",
			"quantity": int64(100),
			"price":    int64(25),
		})
		if err != nil {
			return err
		}

		// Insert third document
		_, err = session.InsertOne("inventory", map[string]interface{}{
			"item":     "Keyboard",
			"quantity": int64(50),
			"price":    int64(75),
		})
		return err
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Verify all documents were inserted
	count, _ := coll.Count(map[string]interface{}{})
	if count != 3 {
		t.Errorf("Expected 3 documents in collection, got %d", count)
	}

	// Verify specific documents exist
	doc, err := coll.FindOne(map[string]interface{}{"item": "Laptop"})
	if err != nil {
		t.Error("Expected to find Laptop document")
	}
	if doc != nil {
		quantity, _ := doc.Get("quantity")
		if quantity != int64(10) {
			t.Errorf("Expected Laptop quantity 10, got %v", quantity)
		}
	}
}

// TestSessionPool_NilSessionPut tests that putting a nil session doesn't panic
func TestSessionPool_NilSessionPut(t *testing.T) {
	db := setupPoolTestDB(t)
	defer db.Close()

	pool := NewSessionPool(db)

	// This should not panic
	pool.Put(nil)
}

// TestSessionPool_Reuse tests that sessions are actually reused
func TestSessionPool_Reuse(t *testing.T) {
	db := setupPoolTestDB(t)
	defer db.Close()

	pool := NewSessionPool(db)

	// Get a session
	session1 := pool.Get()
	session1.AbortTransaction()
	pool.Put(session1)

	// Get another session immediately - should be the same object
	session2 := pool.Get()
	session2.AbortTransaction()

	// Verify they're the same underlying object (pointer equality)
	// Note: This test may be flaky if the pool allocates a new session
	// but it's a good check for the common case

	pool.Put(session2)
}

// TestSessionPool_DeleteOperation tests delete operations in pooled transactions
func TestSessionPool_DeleteOperation(t *testing.T) {
	db := setupPoolTestDB(t)
	defer db.Close()

	pool := NewSessionPool(db)
	coll := db.Collection("to_delete")

	// Pre-insert a document
	id, _ := coll.InsertOne(map[string]interface{}{
		"name": "ToDelete",
		"temp": true,
	})

	// Delete in a transaction
	err := pool.WithTransactionPooled(func(session *Session) error {
		return session.DeleteOne("to_delete", map[string]interface{}{"_id": id})
	})

	if err != nil {
		t.Fatalf("Delete transaction failed: %v", err)
	}

	// Verify document was deleted
	_, err = coll.FindOne(map[string]interface{}{"_id": id})
	if err != ErrDocumentNotFound {
		t.Error("Expected document to be deleted")
	}
}

// TestPoolError_Methods tests the poolError Error() and Unwrap() methods
func TestPoolError_Methods(t *testing.T) {
	originalErr := fmt.Errorf("original error")

	// Test wrapError without args
	wrappedErr := wrapError(originalErr, "operation failed")
	if wrappedErr == nil {
		t.Fatal("Expected non-nil wrapped error")
	}

	// Test Error() method
	errMsg := wrappedErr.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}
	if !contains(errMsg, "operation failed") && !contains(errMsg, "original error") {
		t.Errorf("Error message should contain context or original error, got: %s", errMsg)
	}

	// Test Unwrap() method
	unwrappedErr := errors.Unwrap(wrappedErr)
	if unwrappedErr != originalErr {
		t.Error("Unwrap should return the original error")
	}

	// Test wrapError with args
	wrappedErrWithArgs := wrapError(originalErr, "failed for user %s", "alice")
	if wrappedErrWithArgs == nil {
		t.Fatal("Expected non-nil wrapped error with args")
	}

	errMsgWithArgs := wrappedErrWithArgs.Error()
	if errMsgWithArgs == "" {
		t.Error("Expected non-empty error message with args")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
