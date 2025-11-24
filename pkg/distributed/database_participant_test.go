package distributed

import (
	"context"
	"os"
	"testing"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/mvcc"
)

func setupTestDB(t *testing.T) (*database.Database, func()) {
	t.Helper()

	dataDir := t.TempDir()
	db, err := database.Open(database.DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(dataDir)
	}

	return db, cleanup
}

// TestDatabaseParticipantBasic tests basic participant creation
func TestDatabaseParticipantBasic(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	participant := NewDatabaseParticipant("db1", db)

	if participant.ID() != "db1" {
		t.Errorf("expected ID 'db1', got %s", participant.ID())
	}

	if participant.GetActiveSessionCount() != 0 {
		t.Errorf("expected 0 active sessions, got %d", participant.GetActiveSessionCount())
	}
}

// TestDatabaseParticipantStartTransaction tests starting a transaction
func TestDatabaseParticipantStartTransaction(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	participant := NewDatabaseParticipant("db1", db)

	txnID := mvcc.TxnID(1)
	session := participant.StartTransaction(txnID)

	if session == nil {
		t.Fatal("expected non-nil session")
	}

	if participant.GetActiveSessionCount() != 1 {
		t.Errorf("expected 1 active session, got %d", participant.GetActiveSessionCount())
	}

	// Get the session
	retrievedSession, err := participant.GetSession(txnID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if retrievedSession != session {
		t.Error("retrieved session does not match original session")
	}
}

// TestDatabaseParticipantPrepare tests the prepare phase
func TestDatabaseParticipantPrepare(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a collection
	coll := db.Collection("test")
	_, err := coll.InsertOne(map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})
	if err != nil {
		t.Fatalf("failed to insert document: %v", err)
	}

	participant := NewDatabaseParticipant("db1", db)
	txnID := mvcc.TxnID(2)
	session := participant.StartTransaction(txnID)

	// Perform some operations
	_, err = session.InsertOne("test", map[string]interface{}{
		"name": "Bob",
		"age":  int64(25),
	})
	if err != nil {
		t.Fatalf("failed to insert in session: %v", err)
	}

	// Prepare
	ctx := context.Background()
	vote, err := participant.Prepare(ctx, txnID)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}

	if !vote {
		t.Error("expected vote YES")
	}
}

// TestDatabaseParticipantCommit tests the commit phase
func TestDatabaseParticipantCommit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	coll := db.Collection("test")

	participant := NewDatabaseParticipant("db1", db)
	txnID := mvcc.TxnID(3)
	session := participant.StartTransaction(txnID)

	// Insert a document
	id, err := session.InsertOne("test", map[string]interface{}{
		"name": "Charlie",
		"age":  int64(35),
	})
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	// Prepare
	ctx := context.Background()
	vote, err := participant.Prepare(ctx, txnID)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	if !vote {
		t.Fatal("expected vote YES")
	}

	// Commit
	err = participant.Commit(ctx, txnID)
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// Verify the document exists in the collection
	// Convert the string ID back to ObjectID for querying
	objID, err := document.ObjectIDFromHex(id)
	if err != nil {
		t.Fatalf("failed to convert ID: %v", err)
	}
	doc, err := coll.FindOne(map[string]interface{}{"_id": objID})
	if err != nil {
		t.Fatalf("failed to find document: %v", err)
	}

	name, _ := doc.Get("name")
	if name != "Charlie" {
		t.Errorf("expected name 'Charlie', got %v", name)
	}

	// Verify session was removed
	if participant.GetActiveSessionCount() != 0 {
		t.Errorf("expected 0 active sessions after commit, got %d", participant.GetActiveSessionCount())
	}
}

// TestDatabaseParticipantAbort tests the abort phase
func TestDatabaseParticipantAbort(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	coll := db.Collection("test")

	participant := NewDatabaseParticipant("db1", db)
	txnID := mvcc.TxnID(4)
	session := participant.StartTransaction(txnID)

	// Insert a document
	id, err := session.InsertOne("test", map[string]interface{}{
		"name": "David",
		"age":  int64(40),
	})
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	// Prepare
	ctx := context.Background()
	participant.Prepare(ctx, txnID)

	// Abort
	err = participant.Abort(ctx, txnID)
	if err != nil {
		t.Fatalf("abort failed: %v", err)
	}

	// Verify the document does NOT exist in the collection
	_, err = coll.FindOne(map[string]interface{}{"_id": id})
	if err == nil {
		t.Error("expected error finding aborted document, but found it")
	}

	// Verify session was removed
	if participant.GetActiveSessionCount() != 0 {
		t.Errorf("expected 0 active sessions after abort, got %d", participant.GetActiveSessionCount())
	}
}

// TestDatabaseParticipantPrepareNonexistentSession tests prepare on nonexistent session
func TestDatabaseParticipantPrepareNonexistentSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	participant := NewDatabaseParticipant("db1", db)

	ctx := context.Background()
	_, err := participant.Prepare(ctx, mvcc.TxnID(999))
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

// TestDatabaseParticipantCommitNonexistentSession tests commit on nonexistent session
func TestDatabaseParticipantCommitNonexistentSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	participant := NewDatabaseParticipant("db1", db)

	ctx := context.Background()
	err := participant.Commit(ctx, mvcc.TxnID(999))
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

// TestDatabaseParticipantAbortNonexistentSession tests abort on nonexistent session
func TestDatabaseParticipantAbortNonexistentSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	participant := NewDatabaseParticipant("db1", db)

	ctx := context.Background()
	err := participant.Abort(ctx, mvcc.TxnID(999))
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

// TestDatabaseParticipantContextCancellation tests context cancellation
func TestDatabaseParticipantContextCancellation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	participant := NewDatabaseParticipant("db1", db)
	txnID := mvcc.TxnID(5)
	participant.StartTransaction(txnID)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Prepare should fail due to cancelled context
	_, err := participant.Prepare(ctx, txnID)
	if err == nil {
		t.Error("expected error due to cancelled context")
	}
}
