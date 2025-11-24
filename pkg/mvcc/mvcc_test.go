package mvcc

import (
	"testing"
)

func TestTransactionBeginCommit(t *testing.T) {
	txnMgr := NewTransactionManager()

	txn := txnMgr.Begin()
	if txn == nil {
		t.Fatal("Expected non-nil transaction")
	}

	if txn.State != TxnStateActive {
		t.Error("Expected transaction to be active")
	}

	err := txnMgr.Commit(txn)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	if txn.State != TxnStateCommitted {
		t.Error("Expected transaction to be committed")
	}
}

func TestTransactionReadWrite(t *testing.T) {
	txnMgr := NewTransactionManager()

	txn := txnMgr.Begin()

	// Write
	err := txnMgr.Write(txn, "key1", "value1")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read (should see own write)
	val, exists, err := txnMgr.Read(txn, "key1")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist")
	}
	if val.(string) != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}

	txnMgr.Commit(txn)
}

func TestSnapshotIsolation(t *testing.T) {
	txnMgr := NewTransactionManager()

	// T1: Write and commit
	t1 := txnMgr.Begin()
	txnMgr.Write(t1, "key1", "initial")
	txnMgr.Commit(t1)

	// T2: Start
	t2 := txnMgr.Begin()

	// T3: Modify key1
	t3 := txnMgr.Begin()
	txnMgr.Write(t3, "key1", "modified")
	txnMgr.Commit(t3)

	// T2 should still see "initial" (snapshot isolation)
	val, exists, _ := txnMgr.Read(t2, "key1")
	if !exists {
		t.Error("Expected key to exist in T2's snapshot")
	}
	if val.(string) != "initial" {
		t.Errorf("Expected 'initial', got %v (snapshot isolation violated)", val)
	}

	txnMgr.Commit(t2)

	// T4: Should see "modified"
	t4 := txnMgr.Begin()
	val, exists, _ = txnMgr.Read(t4, "key1")
	if !exists {
		t.Error("Expected key to exist")
	}
	if val.(string) != "modified" {
		t.Errorf("Expected 'modified', got %v", val)
	}
	txnMgr.Commit(t4)
}

func TestTransactionAbort(t *testing.T) {
	txnMgr := NewTransactionManager()

	txn := txnMgr.Begin()
	txnMgr.Write(txn, "key1", "value1")

	// Abort transaction
	err := txnMgr.Abort(txn)
	if err != nil {
		t.Fatalf("Abort failed: %v", err)
	}

	if txn.State != TxnStateAborted {
		t.Error("Expected transaction to be aborted")
	}

	// Read in new transaction - should not see aborted write
	txn2 := txnMgr.Begin()
	_, exists, _ := txnMgr.Read(txn2, "key1")
	if exists {
		t.Error("Expected key to not exist after abort")
	}
	txnMgr.Commit(txn2)
}

func TestConcurrentTransactions(t *testing.T) {
	txnMgr := NewTransactionManager()

	// Write initial value
	t1 := txnMgr.Begin()
	txnMgr.Write(t1, "counter", int64(0))
	txnMgr.Commit(t1)

	// Multiple concurrent transactions
	t2 := txnMgr.Begin()
	t3 := txnMgr.Begin()

	// Both read
	val2, _, _ := txnMgr.Read(t2, "counter")
	val3, _, _ := txnMgr.Read(t3, "counter")

	if val2.(int64) != 0 || val3.(int64) != 0 {
		t.Error("Both transactions should see initial value")
	}

	// Both write
	txnMgr.Write(t2, "counter", int64(1))
	txnMgr.Write(t3, "counter", int64(2))

	// Commit t2 (should succeed - first committer wins)
	err2 := txnMgr.Commit(t2)
	if err2 != nil {
		t.Errorf("t2 commit should succeed, got error: %v", err2)
	}

	// Commit t3 (should fail with conflict - first committer wins)
	err3 := txnMgr.Commit(t3)
	if err3 != ErrConflict {
		t.Errorf("t3 commit should fail with conflict, got: %v", err3)
	}

	// New transaction should see t2's committed value
	t4 := txnMgr.Begin()
	val, _, _ := txnMgr.Read(t4, "counter")
	if val.(int64) != 1 {
		t.Errorf("Expected 1 (from t2), got %v", val)
	}
	txnMgr.Commit(t4)
}

func TestTransactionDelete(t *testing.T) {
	txnMgr := NewTransactionManager()

	// Write
	t1 := txnMgr.Begin()
	txnMgr.Write(t1, "key1", "value1")
	txnMgr.Commit(t1)

	// Delete
	t2 := txnMgr.Begin()
	err := txnMgr.Delete(t2, "key1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	txnMgr.Commit(t2)

	// Read - should not exist
	t3 := txnMgr.Begin()
	_, exists, _ := txnMgr.Read(t3, "key1")
	if exists {
		t.Error("Expected key to not exist after delete")
	}
	txnMgr.Commit(t3)
}

// Test GetActiveTransactions
func TestGetActiveTransactions(t *testing.T) {
	txnMgr := NewTransactionManager()

	// Initially should be 0
	if count := txnMgr.GetActiveTransactions(); count != 0 {
		t.Errorf("Expected 0 active transactions, got %d", count)
	}

	// Start a transaction
	t1 := txnMgr.Begin()
	if count := txnMgr.GetActiveTransactions(); count != 1 {
		t.Errorf("Expected 1 active transaction, got %d", count)
	}

	// Start another transaction
	t2 := txnMgr.Begin()
	if count := txnMgr.GetActiveTransactions(); count != 2 {
		t.Errorf("Expected 2 active transactions, got %d", count)
	}

	// Commit one
	txnMgr.Commit(t1)
	if count := txnMgr.GetActiveTransactions(); count != 1 {
		t.Errorf("Expected 1 active transaction after commit, got %d", count)
	}

	// Abort the other
	txnMgr.Abort(t2)
	if count := txnMgr.GetActiveTransactions(); count != 0 {
		t.Errorf("Expected 0 active transactions after abort, got %d", count)
	}
}

// Test GetWriteSet and SetWriteSet
func TestGetAndSetWriteSet(t *testing.T) {
	txnMgr := NewTransactionManager()

	txn := txnMgr.Begin()

	// Write some data
	txnMgr.Write(txn, "key1", "value1")
	txnMgr.Write(txn, "key2", "value2")

	// Get write set
	writeSet := txn.GetWriteSet()
	if len(writeSet) != 2 {
		t.Errorf("Expected write set size 2, got %d", len(writeSet))
	}

	if writeSet["key1"].Value != "value1" {
		t.Errorf("Expected key1='value1', got %v", writeSet["key1"].Value)
	}

	// Modify write set copy (should not affect original)
	writeSet["key3"] = &VersionedValue{Value: "value3", Version: uint64(txn.ID), CreatedBy: txn.ID}

	// Original should still have 2 entries
	if len(txn.WriteSet) != 2 {
		t.Errorf("Expected original write set still has 2 entries, got %d", len(txn.WriteSet))
	}

	// Test SetWriteSet
	newWriteSet := make(map[string]*VersionedValue)
	newWriteSet["keyA"] = &VersionedValue{Value: "valueA", Version: uint64(txn.ID), CreatedBy: txn.ID}
	newWriteSet["keyB"] = &VersionedValue{Value: "valueB", Version: uint64(txn.ID), CreatedBy: txn.ID}

	txn.SetWriteSet(newWriteSet)

	// Check that write set was replaced
	if len(txn.WriteSet) != 2 {
		t.Errorf("Expected write set size 2 after SetWriteSet, got %d", len(txn.WriteSet))
	}

	if _, exists := txn.WriteSet["keyA"]; !exists {
		t.Error("Expected keyA to exist in write set")
	}

	if _, exists := txn.WriteSet["key1"]; exists {
		t.Error("Expected key1 to not exist in write set after replacement")
	}

	txnMgr.Commit(txn)
}

// Test GetReadSet and SetReadSet
func TestGetAndSetReadSet(t *testing.T) {
	txnMgr := NewTransactionManager()

	// Write initial data
	t1 := txnMgr.Begin()
	txnMgr.Write(t1, "key1", "value1")
	txnMgr.Write(t1, "key2", "value2")
	txnMgr.Commit(t1)

	// Read data in new transaction
	t2 := txnMgr.Begin()
	txnMgr.Read(t2, "key1")
	txnMgr.Read(t2, "key2")

	// Get read set
	readSet := t2.GetReadSet()
	if len(readSet) != 2 {
		t.Errorf("Expected read set size 2, got %d", len(readSet))
	}

	// Modify copy (should not affect original)
	readSet["key3"] = uint64(999)

	// Original should still have 2 entries
	if len(t2.ReadSet) != 2 {
		t.Errorf("Expected original read set still has 2 entries, got %d", len(t2.ReadSet))
	}

	// Test SetReadSet
	newReadSet := make(map[string]uint64)
	newReadSet["keyX"] = uint64(100)
	newReadSet["keyY"] = uint64(200)

	t2.SetReadSet(newReadSet)

	// Check that read set was replaced
	if len(t2.ReadSet) != 2 {
		t.Errorf("Expected read set size 2 after SetReadSet, got %d", len(t2.ReadSet))
	}

	if _, exists := t2.ReadSet["keyX"]; !exists {
		t.Error("Expected keyX to exist in read set")
	}

	if _, exists := t2.ReadSet["key1"]; exists {
		t.Error("Expected key1 to not exist in read set after replacement")
	}

	txnMgr.Commit(t2)
}

// Test VersionStore GetLatest
func TestVersionStoreGetLatest(t *testing.T) {
	txnMgr := NewTransactionManager()

	// Test non-existent key
	val, exists := txnMgr.versionStore.GetLatest("nonexistent")
	if exists {
		t.Error("Expected key to not exist")
	}
	if val != nil {
		t.Errorf("Expected nil value, got %v", val)
	}

	// Write a value
	t1 := txnMgr.Begin()
	txnMgr.Write(t1, "key1", "value1")
	txnMgr.Commit(t1)

	// Get latest
	val, exists = txnMgr.versionStore.GetLatest("key1")
	if !exists {
		t.Error("Expected key to exist")
	}
	if val.(string) != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}

	// Update the value
	t2 := txnMgr.Begin()
	txnMgr.Write(t2, "key1", "value2")
	txnMgr.Commit(t2)

	// Get latest again (should be updated value)
	val, exists = txnMgr.versionStore.GetLatest("key1")
	if !exists {
		t.Error("Expected key to exist")
	}
	if val.(string) != "value2" {
		t.Errorf("Expected 'value2', got %v", val)
	}

	// Delete the key
	t3 := txnMgr.Begin()
	txnMgr.Delete(t3, "key1")
	txnMgr.Commit(t3)

	// Get latest (should not exist after delete)
	val, exists = txnMgr.versionStore.GetLatest("key1")
	if exists {
		t.Error("Expected key to not exist after delete")
	}
}

// Test VersionStore GetAllKeys
func TestVersionStoreGetAllKeys(t *testing.T) {
	txnMgr := NewTransactionManager()

	// Initially no keys
	keys := txnMgr.versionStore.GetAllKeys()
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys, got %d", len(keys))
	}

	// Write some values
	t1 := txnMgr.Begin()
	txnMgr.Write(t1, "key1", "value1")
	txnMgr.Write(t1, "key2", "value2")
	txnMgr.Write(t1, "key3", "value3")
	txnMgr.Commit(t1)

	// Get all keys
	keys = txnMgr.versionStore.GetAllKeys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Check that all keys are present
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	expectedKeys := []string{"key1", "key2", "key3"}
	for _, expectedKey := range expectedKeys {
		if !keyMap[expectedKey] {
			t.Errorf("Expected key %s to be present", expectedKey)
		}
	}
}

// Test VersionStore GetVersionCount
func TestVersionStoreGetVersionCount(t *testing.T) {
	txnMgr := NewTransactionManager()

	// Non-existent key
	count := txnMgr.versionStore.GetVersionCount("nonexistent")
	if count != 0 {
		t.Errorf("Expected count 0 for non-existent key, got %d", count)
	}

	// Write initial version
	t1 := txnMgr.Begin()
	txnMgr.Write(t1, "key1", "value1")
	txnMgr.Commit(t1)

	count = txnMgr.versionStore.GetVersionCount("key1")
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Update (creates new version)
	t2 := txnMgr.Begin()
	txnMgr.Write(t2, "key1", "value2")
	txnMgr.Commit(t2)

	count = txnMgr.versionStore.GetVersionCount("key1")
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}

	// Update again
	t3 := txnMgr.Begin()
	txnMgr.Write(t3, "key1", "value3")
	txnMgr.Commit(t3)

	count = txnMgr.versionStore.GetVersionCount("key1")
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

// Test error conditions
func TestTransactionErrors(t *testing.T) {
	txnMgr := NewTransactionManager()

	// Test read on non-active transaction
	txn := txnMgr.Begin()
	txnMgr.Commit(txn)

	_, _, err := txnMgr.Read(txn, "key1")
	if err != ErrTransactionNotActive {
		t.Errorf("Expected ErrTransactionNotActive, got %v", err)
	}

	// Test write on non-active transaction
	err = txnMgr.Write(txn, "key1", "value1")
	if err != ErrTransactionNotActive {
		t.Errorf("Expected ErrTransactionNotActive, got %v", err)
	}

	// Test delete on non-active transaction
	err = txnMgr.Delete(txn, "key1")
	if err != ErrTransactionNotActive {
		t.Errorf("Expected ErrTransactionNotActive, got %v", err)
	}

	// Test commit on already committed transaction
	err = txnMgr.Commit(txn)
	if err != ErrTransactionNotActive {
		t.Errorf("Expected ErrTransactionNotActive, got %v", err)
	}

	// Test abort on already committed transaction
	err = txnMgr.Abort(txn)
	if err != ErrTransactionNotActive {
		t.Errorf("Expected ErrTransactionNotActive, got %v", err)
	}
}
