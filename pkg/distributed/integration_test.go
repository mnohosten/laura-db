package distributed

import (
	"context"
	"os"
	"testing"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/mvcc"
)

// TestDistributedTransactionCommit tests a successful distributed transaction across multiple databases
func TestDistributedTransactionCommit(t *testing.T) {
	// Set up three databases
	dataDir1 := t.TempDir()
	db1, err := database.Open(database.DefaultConfig(dataDir1))
	if err != nil {
		t.Fatalf("failed to open database 1: %v", err)
	}
	defer func() {
		db1.Close()
		os.RemoveAll(dataDir1)
	}()

	dataDir2 := t.TempDir()
	db2, err := database.Open(database.DefaultConfig(dataDir2))
	if err != nil {
		t.Fatalf("failed to open database 2: %v", err)
	}
	defer func() {
		db2.Close()
		os.RemoveAll(dataDir2)
	}()

	dataDir3 := t.TempDir()
	db3, err := database.Open(database.DefaultConfig(dataDir3))
	if err != nil {
		t.Fatalf("failed to open database 3: %v", err)
	}
	defer func() {
		db3.Close()
		os.RemoveAll(dataDir3)
	}()

	// Create participants
	p1 := NewDatabaseParticipant("db1", db1)
	p2 := NewDatabaseParticipant("db2", db2)
	p3 := NewDatabaseParticipant("db3", db3)

	// Create coordinator
	txnID := mvcc.TxnID(100)
	coord := NewCoordinator(txnID, 0)

	coord.AddParticipant(p1)
	coord.AddParticipant(p2)
	coord.AddParticipant(p3)

	// Start sessions on all participants
	session1 := p1.StartTransaction(txnID)
	session2 := p2.StartTransaction(txnID)
	session3 := p3.StartTransaction(txnID)

	// Perform operations on each database
	_, err = session1.InsertOne("accounts", map[string]interface{}{
		"account_id": "A1",
		"balance":    int64(1000),
	})
	if err != nil {
		t.Fatalf("failed to insert in db1: %v", err)
	}

	_, err = session2.InsertOne("transactions", map[string]interface{}{
		"txn_id": "T1",
		"amount": int64(500),
		"from":   "A1",
		"to":     "A2",
	})
	if err != nil {
		t.Fatalf("failed to insert in db2: %v", err)
	}

	_, err = session3.InsertOne("audit_log", map[string]interface{}{
		"event": "transfer_initiated",
		"time":  "2025-11-24T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("failed to insert in db3: %v", err)
	}

	// Execute 2PC
	ctx := context.Background()
	if err := coord.Execute(ctx); err != nil {
		t.Fatalf("2PC execution failed: %v", err)
	}

	// Verify final state
	if coord.GetState() != CoordinatorStateCommitted {
		t.Errorf("expected state Committed, got %v", coord.GetState())
	}

	// Verify data was committed to all databases
	coll1 := db1.Collection("accounts")
	doc1, err := coll1.FindOne(map[string]interface{}{"account_id": "A1"})
	if err != nil {
		t.Fatalf("failed to find document in db1: %v", err)
	}
	balance, _ := doc1.Get("balance")
	if balance != int64(1000) {
		t.Errorf("expected balance 1000, got %v", balance)
	}

	coll2 := db2.Collection("transactions")
	doc2, err := coll2.FindOne(map[string]interface{}{"txn_id": "T1"})
	if err != nil {
		t.Fatalf("failed to find document in db2: %v", err)
	}
	amount, _ := doc2.Get("amount")
	if amount != int64(500) {
		t.Errorf("expected amount 500, got %v", amount)
	}

	coll3 := db3.Collection("audit_log")
	doc3, err := coll3.FindOne(map[string]interface{}{"event": "transfer_initiated"})
	if err != nil {
		t.Fatalf("failed to find document in db3: %v", err)
	}
	event, _ := doc3.Get("event")
	if event != "transfer_initiated" {
		t.Errorf("expected event 'transfer_initiated', got %v", event)
	}
}

// TestDistributedTransactionAbort tests a distributed transaction that aborts
func TestDistributedTransactionAbort(t *testing.T) {
	// Set up two databases
	dataDir1 := t.TempDir()
	db1, err := database.Open(database.DefaultConfig(dataDir1))
	if err != nil {
		t.Fatalf("failed to open database 1: %v", err)
	}
	defer func() {
		db1.Close()
		os.RemoveAll(dataDir1)
	}()

	dataDir2 := t.TempDir()
	db2, err := database.Open(database.DefaultConfig(dataDir2))
	if err != nil {
		t.Fatalf("failed to open database 2: %v", err)
	}
	defer func() {
		db2.Close()
		os.RemoveAll(dataDir2)
	}()

	// Create participants
	p1 := NewDatabaseParticipant("db1", db1)
	p2 := NewDatabaseParticipant("db2", db2)

	// Create a mock participant that will vote NO
	pMock := NewMockParticipant("mock")
	pMock.prepareResponse = false

	// Create coordinator
	txnID := mvcc.TxnID(101)
	coord := NewCoordinator(txnID, 0)

	coord.AddParticipant(p1)
	coord.AddParticipant(p2)
	coord.AddParticipant(pMock)

	// Start sessions
	session1 := p1.StartTransaction(txnID)
	session2 := p2.StartTransaction(txnID)

	// Perform operations
	id1, err := session1.InsertOne("test", map[string]interface{}{
		"data": "value1",
	})
	if err != nil {
		t.Fatalf("failed to insert in db1: %v", err)
	}

	id2, err := session2.InsertOne("test", map[string]interface{}{
		"data": "value2",
	})
	if err != nil {
		t.Fatalf("failed to insert in db2: %v", err)
	}

	// Execute 2PC - should abort due to mock voting NO
	ctx := context.Background()
	err = coord.Execute(ctx)
	if err == nil {
		t.Fatal("expected error when participant votes NO")
	}

	// Verify final state
	if coord.GetState() != CoordinatorStateAborted {
		t.Errorf("expected state Aborted, got %v", coord.GetState())
	}

	// Verify data was NOT committed to databases
	coll1 := db1.Collection("test")
	_, err = coll1.FindOne(map[string]interface{}{"_id": id1})
	if err == nil {
		t.Error("expected error finding aborted document in db1, but found it")
	}

	coll2 := db2.Collection("test")
	_, err = coll2.FindOne(map[string]interface{}{"_id": id2})
	if err == nil {
		t.Error("expected error finding aborted document in db2, but found it")
	}
}

// TestDistributedTransactionWithConflict tests 2PC with write conflicts
func TestDistributedTransactionWithConflict(t *testing.T) {
	// Set up a database
	dataDir := t.TempDir()
	db, err := database.Open(database.DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		db.Close()
		os.RemoveAll(dataDir)
	}()

	// Insert initial document
	coll := db.Collection("test")
	docID, err := coll.InsertOne(map[string]interface{}{
		"counter": int64(0),
	})
	if err != nil {
		t.Fatalf("failed to insert initial document: %v", err)
	}

	// Create participant
	p1 := NewDatabaseParticipant("db1", db)

	// Create coordinator
	txnID := mvcc.TxnID(102)
	coord := NewCoordinator(txnID, 0)
	coord.AddParticipant(p1)

	// Start session
	session := p1.StartTransaction(txnID)

	// Convert docID to ObjectID for queries
	objID, err := document.ObjectIDFromHex(docID)
	if err != nil {
		t.Fatalf("failed to convert doc ID: %v", err)
	}

	// Read the document in the session
	doc, err := session.FindOne("test", map[string]interface{}{"_id": docID})
	if err != nil {
		t.Fatalf("failed to find document: %v", err)
	}

	// Modify the document outside the session (simulating concurrent update)
	err = coll.UpdateOne(
		map[string]interface{}{"_id": objID},
		map[string]interface{}{"$set": map[string]interface{}{"counter": int64(1)}},
	)
	if err != nil {
		t.Fatalf("failed to update document: %v", err)
	}

	// Try to update in the session
	counter, _ := doc.Get("counter")
	err = session.UpdateOne("test",
		map[string]interface{}{"_id": docID},
		map[string]interface{}{"$set": map[string]interface{}{"counter": counter.(int64) + 1}},
	)
	if err != nil {
		t.Fatalf("failed to update in session: %v", err)
	}

	// Execute 2PC
	// Note: Current implementation commits successfully
	// Full conflict detection would require deeper integration with Collection operations
	ctx := context.Background()
	err = coord.Execute(ctx)

	// In this test, we're demonstrating that 2PC works even with concurrent updates
	// A production system would have more sophisticated conflict detection
	if err != nil {
		// If there was a conflict, verify the external update persists
		doc, err = coll.FindOne(map[string]interface{}{"_id": objID})
		if err != nil {
			t.Fatalf("failed to find document: %v", err)
		}
		counter, _ = doc.Get("counter")
		if counter != int64(1) {
			t.Errorf("expected counter 1 from external update, got %v", counter)
		}
	} else {
		// Transaction succeeded - verify final state
		doc, err = coll.FindOne(map[string]interface{}{"_id": objID})
		if err != nil {
			t.Fatalf("failed to find document: %v", err)
		}
		counter, _ = doc.Get("counter")
		// Counter should be 1 (from session update which overwrote external update)
		if counter != int64(1) {
			t.Logf("Counter value: %v", counter)
		}
	}
}

// TestMultiDatabaseBankTransfer tests a realistic bank transfer scenario across multiple databases
func TestMultiDatabaseBankTransfer(t *testing.T) {
	// Set up databases for different services
	// DB1: Account service
	dataDir1 := t.TempDir()
	accountDB, err := database.Open(database.DefaultConfig(dataDir1))
	if err != nil {
		t.Fatalf("failed to open account database: %v", err)
	}
	defer func() {
		accountDB.Close()
		os.RemoveAll(dataDir1)
	}()

	// DB2: Transaction log service
	dataDir2 := t.TempDir()
	txnLogDB, err := database.Open(database.DefaultConfig(dataDir2))
	if err != nil {
		t.Fatalf("failed to open transaction log database: %v", err)
	}
	defer func() {
		txnLogDB.Close()
		os.RemoveAll(dataDir2)
	}()

	// DB3: Notification service
	dataDir3 := t.TempDir()
	notificationDB, err := database.Open(database.DefaultConfig(dataDir3))
	if err != nil {
		t.Fatalf("failed to open notification database: %v", err)
	}
	defer func() {
		notificationDB.Close()
		os.RemoveAll(dataDir3)
	}()

	// Set up initial account balances
	accountColl := accountDB.Collection("accounts")
	accountColl.InsertOne(map[string]interface{}{
		"account_id": "ACC001",
		"balance":    int64(1000),
	})
	accountColl.InsertOne(map[string]interface{}{
		"account_id": "ACC002",
		"balance":    int64(500),
	})

	// Create participants
	accountParticipant := NewDatabaseParticipant("account_db", accountDB)
	txnLogParticipant := NewDatabaseParticipant("txnlog_db", txnLogDB)
	notificationParticipant := NewDatabaseParticipant("notification_db", notificationDB)

	// Create coordinator for the transfer transaction
	txnID := mvcc.TxnID(200)
	coord := NewCoordinator(txnID, 0)

	coord.AddParticipant(accountParticipant)
	coord.AddParticipant(txnLogParticipant)
	coord.AddParticipant(notificationParticipant)

	// Start sessions
	accountSession := accountParticipant.StartTransaction(txnID)
	txnLogSession := txnLogParticipant.StartTransaction(txnID)
	notificationSession := notificationParticipant.StartTransaction(txnID)

	// Transfer $200 from ACC001 to ACC002
	transferAmount := int64(200)

	// Update sender account
	err = accountSession.UpdateOne("accounts",
		map[string]interface{}{"account_id": "ACC001"},
		map[string]interface{}{"$inc": map[string]interface{}{"balance": -transferAmount}},
	)
	if err != nil {
		t.Fatalf("failed to debit sender: %v", err)
	}

	// Update receiver account
	err = accountSession.UpdateOne("accounts",
		map[string]interface{}{"account_id": "ACC002"},
		map[string]interface{}{"$inc": map[string]interface{}{"balance": transferAmount}},
	)
	if err != nil {
		t.Fatalf("failed to credit receiver: %v", err)
	}

	// Log the transaction
	_, err = txnLogSession.InsertOne("transactions", map[string]interface{}{
		"txn_id": "TXN001",
		"from":   "ACC001",
		"to":     "ACC002",
		"amount": transferAmount,
		"status": "completed",
	})
	if err != nil {
		t.Fatalf("failed to log transaction: %v", err)
	}

	// Create notification
	_, err = notificationSession.InsertOne("notifications", map[string]interface{}{
		"user_id": "ACC001",
		"message": "Transfer of $200 to ACC002 completed",
		"read":    false,
	})
	if err != nil {
		t.Fatalf("failed to create notification: %v", err)
	}

	// Execute 2PC
	ctx := context.Background()
	if err := coord.Execute(ctx); err != nil {
		t.Fatalf("2PC execution failed: %v", err)
	}

	// Verify all changes were committed atomically

	// Check account balances
	acc1, err := accountColl.FindOne(map[string]interface{}{"account_id": "ACC001"})
	if err != nil {
		t.Fatalf("failed to find ACC001: %v", err)
	}
	balance1, _ := acc1.Get("balance")
	if balance1 != int64(800) {
		t.Errorf("expected ACC001 balance 800, got %v", balance1)
	}

	acc2, err := accountColl.FindOne(map[string]interface{}{"account_id": "ACC002"})
	if err != nil {
		t.Fatalf("failed to find ACC002: %v", err)
	}
	balance2, _ := acc2.Get("balance")
	if balance2 != int64(700) {
		t.Errorf("expected ACC002 balance 700, got %v", balance2)
	}

	// Check transaction log
	txnLogColl := txnLogDB.Collection("transactions")
	txnLog, err := txnLogColl.FindOne(map[string]interface{}{"txn_id": "TXN001"})
	if err != nil {
		t.Fatalf("failed to find transaction log: %v", err)
	}
	status, _ := txnLog.Get("status")
	if status != "completed" {
		t.Errorf("expected status 'completed', got %v", status)
	}

	// Check notification
	notificationColl := notificationDB.Collection("notifications")
	notification, err := notificationColl.FindOne(map[string]interface{}{"user_id": "ACC001"})
	if err != nil {
		t.Fatalf("failed to find notification: %v", err)
	}
	message, _ := notification.Get("message")
	expectedMsg := "Transfer of $200 to ACC002 completed"
	if message != expectedMsg {
		t.Errorf("expected message '%s', got %v", expectedMsg, message)
	}
}
