package database

import (
	"fmt"
	"os"
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/mvcc"
)

func TestSessionBasicTransaction(t *testing.T) {
	// Create a temporary directory for the test
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	// Open database
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Start a session
	session := db.StartSession()

	// Insert a document within the transaction
	doc := map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	}

	id, err := session.InsertOne("users", doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Commit the transaction
	if err := session.CommitTransaction(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify the document was inserted (outside of transaction)
	coll := db.Collection("users")
	// Convert the hex string back to ObjectID for searching
	objID, err := document.ObjectIDFromHex(id)
	if err != nil {
		t.Fatalf("Failed to parse ObjectID: %v", err)
	}
	found, err := coll.FindOne(map[string]interface{}{"_id": objID})
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	name, _ := found.Get("name")
	if name != "Alice" {
		t.Errorf("Expected name 'Alice', got %v", name)
	}
}

func TestSessionAbortTransaction(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Start a session
	session := db.StartSession()

	// Insert a document within the transaction
	doc := map[string]interface{}{
		"name": "Bob",
		"age":  int64(25),
	}

	id, err := session.InsertOne("users", doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Abort the transaction
	if err := session.AbortTransaction(); err != nil {
		t.Fatalf("Failed to abort transaction: %v", err)
	}

	// Verify the document was NOT inserted
	coll := db.Collection("users")
	objID, _ := document.ObjectIDFromHex(id)
	_, err = coll.FindOne(map[string]interface{}{"_id": objID})
	if err == nil {
		t.Error("Expected document to not exist after abort, but it was found")
	}
}

func TestSessionMultipleOperations(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Insert initial documents outside of transaction
	coll := db.Collection("accounts")
	aliceID, _ := coll.InsertOne(map[string]interface{}{
		"name":    "Alice",
		"balance": int64(1000),
	})
	bobID, _ := coll.InsertOne(map[string]interface{}{
		"name":    "Bob",
		"balance": int64(500),
	})

	// Start a session for transfer
	session := db.StartSession()

	// Transfer $200 from Alice to Bob
	err = session.UpdateOne("accounts", map[string]interface{}{"_id": aliceID}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(-200)},
	})
	if err != nil {
		t.Fatalf("Failed to update Alice's account: %v", err)
	}

	err = session.UpdateOne("accounts", map[string]interface{}{"_id": bobID}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(200)},
	})
	if err != nil {
		t.Fatalf("Failed to update Bob's account: %v", err)
	}

	// Commit the transaction
	if err := session.CommitTransaction(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify balances
	aliceObjID, _ := document.ObjectIDFromHex(aliceID)
	bobObjID, _ := document.ObjectIDFromHex(bobID)
	alice, _ := coll.FindOne(map[string]interface{}{"_id": aliceObjID})
	bob, _ := coll.FindOne(map[string]interface{}{"_id": bobObjID})

	aliceBalance, _ := alice.Get("balance")
	bobBalance, _ := bob.Get("balance")

	if aliceBalance != int64(800) {
		t.Errorf("Expected Alice's balance to be 800, got %v", aliceBalance)
	}
	if bobBalance != int64(700) {
		t.Errorf("Expected Bob's balance to be 700, got %v", bobBalance)
	}
}

func TestSessionWriteConflictDetection(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Insert initial document outside of transaction
	coll := db.Collection("accounts")
	id, _ := coll.InsertOne(map[string]interface{}{
		"name":    "Charlie",
		"balance": int64(1000),
	})

	// Start two concurrent sessions
	session1 := db.StartSession()
	session2 := db.StartSession()

	// Both sessions read the same document
	objID, _ := document.ObjectIDFromHex(id)
	doc1, _ := session1.FindOne("accounts", map[string]interface{}{"_id": objID})
	doc2, _ := session2.FindOne("accounts", map[string]interface{}{"_id": objID})

	balance1, _ := doc1.Get("balance")
	balance2, _ := doc2.Get("balance")

	if balance1 != int64(1000) || balance2 != int64(1000) {
		t.Fatalf("Initial balance mismatch")
	}

	// Session 1 updates and commits
	session1.UpdateOne("accounts", map[string]interface{}{"_id": objID}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(500)},
	})
	if err := session1.CommitTransaction(); err != nil {
		t.Fatalf("Session 1 commit failed: %v", err)
	}

	// Session 2 tries to update the same document (should detect conflict)
	session2.UpdateOne("accounts", map[string]interface{}{"_id": objID}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(300)},
	})
	err = session2.CommitTransaction()
	if err != mvcc.ErrConflict {
		t.Errorf("Expected write conflict error, got: %v", err)
	}
}

func TestWithTransactionSuccess(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Use WithTransaction helper
	err = db.WithTransaction(func(session *Session) error {
		// Insert multiple documents
		_, err := session.InsertOne("users", map[string]interface{}{
			"name": "Alice",
			"age":  int64(30),
		})
		if err != nil {
			return err
		}

		_, err = session.InsertOne("users", map[string]interface{}{
			"name": "Bob",
			"age":  int64(25),
		})
		return err
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Verify both documents were inserted
	coll := db.Collection("users")
	results, _ := coll.Find(map[string]interface{}{})
	if len(results) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(results))
	}
}

func TestWithTransactionFailure(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Use WithTransaction helper with intentional error
	err = db.WithTransaction(func(session *Session) error {
		// Insert a document
		_, err := session.InsertOne("users", map[string]interface{}{
			"name": "Charlie",
			"age":  int64(35),
		})
		if err != nil {
			return err
		}

		// Return an error to trigger rollback
		return fmt.Errorf("intentional error")
	})

	if err == nil {
		t.Fatal("Expected transaction to fail")
	}

	// Verify no documents were inserted
	coll := db.Collection("users")
	results, _ := coll.Find(map[string]interface{}{})
	if len(results) != 0 {
		t.Errorf("Expected 0 documents after rollback, got %d", len(results))
	}
}

func TestSessionMultiCollectionTransaction(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Use multi-collection transaction
	err = db.WithTransaction(func(session *Session) error {
		// Insert into users collection
		userID, err := session.InsertOne("users", map[string]interface{}{
			"name":  "David",
			"email": "david@example.com",
		})
		if err != nil {
			return err
		}

		// Insert into orders collection with reference to user
		_, err = session.InsertOne("orders", map[string]interface{}{
			"user_id": userID,
			"product": "Laptop",
			"amount":  int64(1200),
		})
		return err
	})

	if err != nil {
		t.Fatalf("Multi-collection transaction failed: %v", err)
	}

	// Verify documents in both collections
	users := db.Collection("users")
	orders := db.Collection("orders")

	userDocs, _ := users.Find(map[string]interface{}{})
	orderDocs, _ := orders.Find(map[string]interface{}{})

	if len(userDocs) != 1 {
		t.Errorf("Expected 1 user, got %d", len(userDocs))
	}
	if len(orderDocs) != 1 {
		t.Errorf("Expected 1 order, got %d", len(orderDocs))
	}
}

func TestSessionReadYourOwnWrites(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Start a session
	session := db.StartSession()

	// Insert a document
	id, err := session.InsertOne("users", map[string]interface{}{
		"name": "Eve",
		"age":  int64(28),
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Read the same document within the same transaction
	objID, _ := document.ObjectIDFromHex(id)
	doc, err := session.FindOne("users", map[string]interface{}{"_id": objID})
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	name, _ := doc.Get("name")
	if name != "Eve" {
		t.Errorf("Expected to read own write, got name: %v", name)
	}

	// Commit
	session.CommitTransaction()
}

func TestSessionSnapshotIsolation(t *testing.T) {
	dataDir := t.TempDir()
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Insert initial document
	coll := db.Collection("counters")
	id, _ := coll.InsertOne(map[string]interface{}{
		"name":  "counter",
		"value": int64(0),
	})

	// Start session 1
	objID, _ := document.ObjectIDFromHex(id)
	session1 := db.StartSession()
	doc1, _ := session1.FindOne("counters", map[string]interface{}{"_id": objID})
	val1, _ := doc1.Get("value")
	initialValue1 := val1.(int64)

	// Another session updates the counter and commits
	session2 := db.StartSession()
	session2.UpdateOne("counters", map[string]interface{}{"_id": objID}, map[string]interface{}{
		"$inc": map[string]interface{}{"value": int64(10)},
	})
	session2.CommitTransaction()

	// Session 1 reads again - should still see the same snapshot
	doc1Again, _ := session1.FindOne("counters", map[string]interface{}{"_id": objID})
	val1Again, _ := doc1Again.Get("value")
	value1Again := val1Again.(int64)

	if value1Again != initialValue1 {
		t.Errorf("Snapshot isolation violated: expected %d, got %d", initialValue1, value1Again)
	}

	// Session 1 commits
	session1.CommitTransaction()

	// Now a new read should see the updated value
	doc, _ := coll.FindOne(map[string]interface{}{"_id": objID})
	docVal, _ := doc.Get("value")
	if docVal != int64(10) {
		t.Errorf("Expected value 10 after commit, got %v", docVal)
	}
}

// TestSession_Transaction tests the Transaction() method
func TestSession_Transaction(t *testing.T) {
	dataDir := t.TempDir()
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	session := db.StartSession()

	// Get the underlying transaction
	txn := session.Transaction()
	if txn == nil {
		t.Fatal("Expected non-nil transaction")
	}

	// Verify it's the same transaction object
	if session.txn != txn {
		t.Error("Transaction() should return the session's transaction")
	}

	session.AbortTransaction()
}
