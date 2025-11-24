package database

import (
	"os"
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
)

// TestSavepointBasic tests basic savepoint creation and rollback
func TestSavepointBasic(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("users")

	// Start a session
	session := db.StartSession()
	defer session.AbortTransaction()

	// Insert a document
	id1, err := session.InsertOne("users", map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Create a savepoint
	if err := session.CreateSavepoint("sp1"); err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Insert another document
	id2, err := session.InsertOne("users", map[string]interface{}{
		"name": "Bob",
		"age":  int64(25),
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Verify both documents are visible in the session
	doc1, err := session.FindOne("users", map[string]interface{}{"_id": id1})
	if err != nil || doc1 == nil {
		t.Fatalf("Expected to find document 1")
	}

	doc2, err := session.FindOne("users", map[string]interface{}{"_id": id2})
	if err != nil || doc2 == nil {
		t.Fatalf("Expected to find document 2")
	}

	// Rollback to savepoint
	if err := session.RollbackToSavepoint("sp1"); err != nil {
		t.Fatalf("Failed to rollback to savepoint: %v", err)
	}

	// Document 1 should still be visible
	doc1After, err := session.FindOne("users", map[string]interface{}{"_id": id1})
	if err != nil || doc1After == nil {
		t.Fatalf("Expected to find document 1 after rollback")
	}

	// Document 2 should NOT be visible (rolled back)
	// The insert operation was rolled back, so the document never existed
	// However, we need to commit first to see the actual state
	if err := session.CommitTransaction(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify in the collection
	objID1, err := document.ObjectIDFromHex(id1)
	if err != nil {
		t.Fatalf("Failed to parse ObjectID1: %v", err)
	}
	doc1Final, err := coll.FindOne(map[string]interface{}{"_id": objID1})
	if err != nil || doc1Final == nil {
		t.Errorf("Expected document 1 to exist after commit")
	}

	objID2, err := document.ObjectIDFromHex(id2)
	if err != nil {
		t.Fatalf("Failed to parse ObjectID2: %v", err)
	}
	doc2Final, err := coll.FindOne(map[string]interface{}{"_id": objID2})
	if err == nil && doc2Final != nil {
		t.Errorf("Expected document 2 to NOT exist after rollback")
	}
}

// TestSavepointMultiple tests multiple savepoints
func TestSavepointMultiple(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	session := db.StartSession()
	defer session.AbortTransaction()

	// Insert doc 1
	id1, _ := session.InsertOne("users", map[string]interface{}{"name": "Alice"})

	// Create savepoint 1
	if err := session.CreateSavepoint("sp1"); err != nil {
		t.Fatalf("Failed to create sp1: %v", err)
	}

	// Insert doc 2
	id2, _ := session.InsertOne("users", map[string]interface{}{"name": "Bob"})

	// Create savepoint 2
	if err := session.CreateSavepoint("sp2"); err != nil {
		t.Fatalf("Failed to create sp2: %v", err)
	}

	// Insert doc 3
	id3, _ := session.InsertOne("users", map[string]interface{}{"name": "Charlie"})

	// All three documents should be visible
	if doc, _ := session.FindOne("users", map[string]interface{}{"_id": id1}); doc == nil {
		t.Errorf("Expected to find doc 1")
	}
	if doc, _ := session.FindOne("users", map[string]interface{}{"_id": id2}); doc == nil {
		t.Errorf("Expected to find doc 2")
	}
	if doc, _ := session.FindOne("users", map[string]interface{}{"_id": id3}); doc == nil {
		t.Errorf("Expected to find doc 3")
	}

	// Rollback to sp2 (should remove doc 3)
	if err := session.RollbackToSavepoint("sp2"); err != nil {
		t.Fatalf("Failed to rollback to sp2: %v", err)
	}

	// sp2 should be removed after rollback (standard SQL behavior)
	savepoints := session.ListSavepoints()
	if len(savepoints) != 1 || savepoints[0] != "sp1" {
		t.Errorf("Expected only sp1 to remain, got: %v", savepoints)
	}

	// Rollback to sp1 (should remove doc 2)
	if err := session.RollbackToSavepoint("sp1"); err != nil {
		t.Fatalf("Failed to rollback to sp1: %v", err)
	}

	// Commit and verify only doc 1 exists
	if err := session.CommitTransaction(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	coll := db.Collection("users")
	objID1, _ := document.ObjectIDFromHex(id1)
	doc1, err := coll.FindOne(map[string]interface{}{"_id": objID1})
	if err != nil || doc1 == nil {
		t.Errorf("Expected doc 1 to exist")
	}

	objID2, _ := document.ObjectIDFromHex(id2)
	doc2, err := coll.FindOne(map[string]interface{}{"_id": objID2})
	if err == nil && doc2 != nil {
		t.Errorf("Expected doc 2 to NOT exist")
	}

	objID3, _ := document.ObjectIDFromHex(id3)
	doc3, err := coll.FindOne(map[string]interface{}{"_id": objID3})
	if err == nil && doc3 != nil {
		t.Errorf("Expected doc 3 to NOT exist")
	}
}

// TestSavepointUpdate tests savepoint with updates
func TestSavepointUpdate(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("users")

	// Insert initial document
	id, _ := coll.InsertOne(map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})

	session := db.StartSession()
	defer session.AbortTransaction()

	// Update the document
	if err := session.UpdateOne("users", map[string]interface{}{"_id": id}, map[string]interface{}{
		"$set": map[string]interface{}{"age": int64(31)},
	}); err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	// Create savepoint
	if err := session.CreateSavepoint("sp1"); err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Update again
	if err := session.UpdateOne("users", map[string]interface{}{"_id": id}, map[string]interface{}{
		"$set": map[string]interface{}{"age": int64(32)},
	}); err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	// Verify age is 32
	doc, _ := session.FindOne("users", map[string]interface{}{"_id": id})
	age, _ := doc.Get("age")
	if age != int64(32) {
		t.Errorf("Expected age 32, got %v", age)
	}

	// Rollback to savepoint
	if err := session.RollbackToSavepoint("sp1"); err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Commit and verify age is 31
	if err := session.CommitTransaction(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	objID, _ := document.ObjectIDFromHex(id)
	docFinal, err := coll.FindOne(map[string]interface{}{"_id": objID})
	if err != nil || docFinal == nil {
		t.Fatalf("Failed to find document after commit: %v", err)
	}
	ageFinal, _ := docFinal.Get("age")
	if ageFinal != int64(31) {
		t.Errorf("Expected age 31 after rollback, got %v", ageFinal)
	}
}

// TestSavepointDelete tests savepoint with deletes
func TestSavepointDelete(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("users")

	// Insert document
	id, _ := coll.InsertOne(map[string]interface{}{
		"name": "Alice",
	})

	session := db.StartSession()
	defer session.AbortTransaction()

	// Create savepoint before delete
	if err := session.CreateSavepoint("before_delete"); err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Delete the document
	if err := session.DeleteOne("users", map[string]interface{}{"_id": id}); err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Document should not be visible
	doc, err := session.FindOne("users", map[string]interface{}{"_id": id})
	if err == nil && doc != nil {
		t.Errorf("Expected document to be deleted in session")
	}

	// Rollback to savepoint
	if err := session.RollbackToSavepoint("before_delete"); err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Commit and verify document still exists
	if err := session.CommitTransaction(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	objID, _ := document.ObjectIDFromHex(id)
	docFinal, err := coll.FindOne(map[string]interface{}{"_id": objID})
	if err != nil || docFinal == nil {
		t.Errorf("Expected document to exist after rollback")
	}
}

// TestSavepointRelease tests releasing savepoints
func TestSavepointRelease(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	session := db.StartSession()
	defer session.AbortTransaction()

	// Create savepoints
	session.CreateSavepoint("sp1")
	session.CreateSavepoint("sp2")
	session.CreateSavepoint("sp3")

	// Verify all exist
	savepoints := session.ListSavepoints()
	if len(savepoints) != 3 {
		t.Errorf("Expected 3 savepoints, got %d", len(savepoints))
	}

	// Release sp2
	if err := session.ReleaseSavepoint("sp2"); err != nil {
		t.Fatalf("Failed to release savepoint: %v", err)
	}

	// Verify only sp1 and sp3 remain
	savepoints = session.ListSavepoints()
	if len(savepoints) != 2 {
		t.Errorf("Expected 2 savepoints after release, got %d", len(savepoints))
	}

	// Try to rollback to released savepoint (should fail)
	if err := session.RollbackToSavepoint("sp2"); err == nil {
		t.Errorf("Expected error when rolling back to released savepoint")
	}
}

// TestSavepointDuplicateName tests error on duplicate savepoint names
func TestSavepointDuplicateName(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	session := db.StartSession()
	defer session.AbortTransaction()

	// Create savepoint
	if err := session.CreateSavepoint("sp1"); err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Try to create again with same name
	if err := session.CreateSavepoint("sp1"); err == nil {
		t.Errorf("Expected error when creating duplicate savepoint")
	}
}

// TestSavepointInactiveTransaction tests savepoint operations on inactive transaction
func TestSavepointInactiveTransaction(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	session := db.StartSession()

	// Commit the transaction
	if err := session.CommitTransaction(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Try to create savepoint (should fail)
	if err := session.CreateSavepoint("sp1"); err == nil {
		t.Errorf("Expected error when creating savepoint on committed transaction")
	}

	// Start a new session and create savepoint
	session2 := db.StartSession()
	defer session2.AbortTransaction()
	session2.CreateSavepoint("sp1")

	// Abort the transaction
	session2.AbortTransaction()

	// Try to rollback (should fail)
	if err := session2.RollbackToSavepoint("sp1"); err == nil {
		t.Errorf("Expected error when rolling back on aborted transaction")
	}
}

// TestSavepointNonexistent tests operations on nonexistent savepoint
func TestSavepointNonexistent(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	session := db.StartSession()
	defer session.AbortTransaction()

	// Try to rollback to nonexistent savepoint
	if err := session.RollbackToSavepoint("nonexistent"); err == nil {
		t.Errorf("Expected error when rolling back to nonexistent savepoint")
	}

	// Try to release nonexistent savepoint
	if err := session.ReleaseSavepoint("nonexistent"); err == nil {
		t.Errorf("Expected error when releasing nonexistent savepoint")
	}
}

// TestSavepointComplexScenario tests a complex scenario with multiple operations
func TestSavepointComplexScenario(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("accounts")

	// Setup: Insert two accounts
	id1, _ := coll.InsertOne(map[string]interface{}{
		"name":    "Account A",
		"balance": int64(1000),
	})
	id2, _ := coll.InsertOne(map[string]interface{}{
		"name":    "Account B",
		"balance": int64(500),
	})

	session := db.StartSession()
	defer session.AbortTransaction()

	// Deduct from Account A
	session.UpdateOne("accounts", map[string]interface{}{"_id": id1}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(-100)},
	})

	// Create savepoint before crediting Account B
	session.CreateSavepoint("before_credit")

	// Credit to Account B
	session.UpdateOne("accounts", map[string]interface{}{"_id": id2}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(100)},
	})

	// Simulate error detection - rollback the credit
	session.RollbackToSavepoint("before_credit")

	// Create new savepoint
	session.CreateSavepoint("before_retry")

	// Credit to correct account (Account B again, but this time we keep it)
	session.UpdateOne("accounts", map[string]interface{}{"_id": id2}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(100)},
	})

	// Commit the transaction
	if err := session.CommitTransaction(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify final balances
	objID1, _ := document.ObjectIDFromHex(id1)
	acc1, _ := coll.FindOne(map[string]interface{}{"_id": objID1})
	balance1, _ := acc1.Get("balance")
	if balance1 != int64(900) { // 1000 - 100
		t.Errorf("Expected Account A balance 900, got %v", balance1)
	}

	objID2, _ := document.ObjectIDFromHex(id2)
	acc2, _ := coll.FindOne(map[string]interface{}{"_id": objID2})
	balance2, _ := acc2.Get("balance")
	if balance2 != int64(600) { // 500 + 100
		t.Errorf("Expected Account B balance 600, got %v", balance2)
	}
}

// TestSavepointWithSnapshotIsolation tests savepoint interaction with snapshot isolation
func TestSavepointWithSnapshotIsolation(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("users")

	// Insert initial document
	id, _ := coll.InsertOne(map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})

	session := db.StartSession()
	defer session.AbortTransaction()

	// Read the document (creates snapshot)
	doc1, _ := session.FindOne("users", map[string]interface{}{"_id": id})
	age1, _ := doc1.Get("age")
	if age1 != int64(30) {
		t.Errorf("Expected age 30, got %v", age1)
	}

	// Create savepoint
	session.CreateSavepoint("sp1")

	// Update the document in session
	session.UpdateOne("users", map[string]interface{}{"_id": id}, map[string]interface{}{
		"$set": map[string]interface{}{"age": int64(35)},
	})

	// Read again (should see update)
	doc2, _ := session.FindOne("users", map[string]interface{}{"_id": id})
	age2, _ := doc2.Get("age")
	if age2 != int64(35) {
		t.Errorf("Expected age 35, got %v", age2)
	}

	// Rollback to savepoint
	session.RollbackToSavepoint("sp1")

	// Read again (should see original value from snapshot)
	doc3, _ := session.FindOne("users", map[string]interface{}{"_id": id})
	age3, _ := doc3.Get("age")
	if age3 != int64(30) {
		t.Errorf("Expected age 30 after rollback, got %v", age3)
	}

	// Abort transaction
	session.AbortTransaction()
}
