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

	// Commit both
	txnMgr.Commit(t2)
	txnMgr.Commit(t3)

	// New transaction should see last committed value
	t4 := txnMgr.Begin()
	val, _, _ := txnMgr.Read(t4, "counter")
	if val.(int64) != 2 {
		t.Errorf("Expected 2, got %v", val)
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
