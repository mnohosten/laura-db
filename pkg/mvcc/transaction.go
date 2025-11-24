package mvcc

import (
	"sync"
	"sync/atomic"
	"time"
)

// TxnID is a unique transaction identifier
type TxnID uint64

// TxnState represents the state of a transaction
type TxnState int

const (
	TxnStateActive TxnState = iota
	TxnStateCommitted
	TxnStateAborted
)

// Transaction represents a database transaction
type Transaction struct {
	ID          TxnID
	StartTime   time.Time
	CommitTime  time.Time
	State       TxnState
	ReadVersion uint64                     // Snapshot version for reads
	WriteSet    map[string]*VersionedValue // Local changes
	ReadSet     map[string]uint64          // Tracks versions of keys read (for conflict detection)
	mu          sync.RWMutex
}

// VersionedValue represents a value with version information
type VersionedValue struct {
	Value       interface{}
	Version     uint64
	CreatedBy   TxnID
	DeletedBy   TxnID // 0 if not deleted
	CommitTime  time.Time
}

// TransactionManager manages all active transactions
type TransactionManager struct {
	nextTxnID      uint64
	nextVersion    uint64
	activeTxns     map[TxnID]*Transaction
	committedTxns  map[TxnID]*Transaction
	mu             sync.RWMutex
	versionStore   *VersionStore
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager() *TransactionManager {
	return &TransactionManager{
		nextTxnID:     1,
		nextVersion:   1,
		activeTxns:    make(map[TxnID]*Transaction),
		committedTxns: make(map[TxnID]*Transaction),
		versionStore:  NewVersionStore(),
	}
}

// Begin starts a new transaction
func (tm *TransactionManager) Begin() *Transaction {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	txnID := TxnID(atomic.AddUint64(&tm.nextTxnID, 1))
	readVersion := atomic.LoadUint64(&tm.nextVersion)

	txn := &Transaction{
		ID:          txnID,
		StartTime:   time.Now(),
		State:       TxnStateActive,
		ReadVersion: readVersion,
		WriteSet:    make(map[string]*VersionedValue),
		ReadSet:     make(map[string]uint64),
	}

	tm.activeTxns[txnID] = txn
	return txn
}

// Commit commits a transaction
func (tm *TransactionManager) Commit(txn *Transaction) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	txn.mu.Lock()
	defer txn.mu.Unlock()

	if txn.State != TxnStateActive {
		return ErrTransactionNotActive
	}

	// Write conflict detection (First-Committer-Wins / Optimistic Concurrency Control)
	// Check if any keys that were read have been modified since this transaction started
	for key, readVersion := range txn.ReadSet {
		currentVersion := tm.versionStore.GetLatestVersion(key)

		// If the key is in the write set, we're updating it, so check for conflicts
		if _, isWritten := txn.WriteSet[key]; isWritten {
			// Check if another transaction modified this key after we read it
			if currentVersion > readVersion && currentVersion > txn.ReadVersion {
				// Write-write conflict detected
				return ErrConflict
			}
		}
		// For read-only keys in read set, we use snapshot isolation
		// (no conflict detection needed as readers don't block writers)
	}

	// Also check keys that are written but not explicitly read
	// to detect write-write conflicts
	for key := range txn.WriteSet {
		if _, wasRead := txn.ReadSet[key]; !wasRead {
			// This key was written without being explicitly read
			// Check if it was modified after this transaction started
			currentVersion := tm.versionStore.GetLatestVersion(key)
			if currentVersion > txn.ReadVersion {
				// Write-write conflict detected
				return ErrConflict
			}
		}
	}

	// Assign commit version
	commitVersion := atomic.AddUint64(&tm.nextVersion, 1)
	txn.CommitTime = time.Now()

	// Apply write set to version store
	for key, versionedValue := range txn.WriteSet {
		versionedValue.Version = commitVersion
		versionedValue.CommitTime = txn.CommitTime
		tm.versionStore.Put(key, versionedValue)
	}

	// Update transaction state
	txn.State = TxnStateCommitted

	// Move from active to committed
	delete(tm.activeTxns, txn.ID)
	tm.committedTxns[txn.ID] = txn

	// Trigger garbage collection if needed
	go tm.maybeGarbageCollect()

	return nil
}

// Abort aborts a transaction
func (tm *TransactionManager) Abort(txn *Transaction) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	txn.mu.Lock()
	defer txn.mu.Unlock()

	if txn.State != TxnStateActive {
		return ErrTransactionNotActive
	}

	// Discard write set
	txn.WriteSet = nil
	txn.State = TxnStateAborted

	// Remove from active transactions
	delete(tm.activeTxns, txn.ID)

	return nil
}

// Read reads a value within a transaction using snapshot isolation
func (tm *TransactionManager) Read(txn *Transaction, key string) (interface{}, bool, error) {
	txn.mu.Lock()
	defer txn.mu.Unlock()

	if txn.State != TxnStateActive {
		return nil, false, ErrTransactionNotActive
	}

	// First check write set (read your own writes)
	if versionedValue, ok := txn.WriteSet[key]; ok {
		if versionedValue.DeletedBy != 0 {
			return nil, false, nil // Deleted in this transaction
		}
		return versionedValue.Value, true, nil
	}

	// Read from version store using snapshot isolation
	value, exists, err := tm.versionStore.GetVersion(key, txn.ReadVersion)

	// Track the version read for conflict detection
	// Record the version that was visible at read time
	if exists {
		// Get the actual version number of what we read
		latestVersion := tm.versionStore.GetLatestVersion(key)
		if latestVersion <= txn.ReadVersion {
			txn.ReadSet[key] = latestVersion
		} else {
			txn.ReadSet[key] = txn.ReadVersion
		}
	} else {
		// Key didn't exist, record version 0
		txn.ReadSet[key] = 0
	}

	return value, exists, err
}

// Write writes a value within a transaction
func (tm *TransactionManager) Write(txn *Transaction, key string, value interface{}) error {
	txn.mu.Lock()
	defer txn.mu.Unlock()

	if txn.State != TxnStateActive {
		return ErrTransactionNotActive
	}

	// Add to write set
	txn.WriteSet[key] = &VersionedValue{
		Value:      value,
		CreatedBy:  txn.ID,
		DeletedBy:  0,
	}

	return nil
}

// Delete deletes a value within a transaction
func (tm *TransactionManager) Delete(txn *Transaction, key string) error {
	txn.mu.Lock()
	defer txn.mu.Unlock()

	if txn.State != TxnStateActive {
		return ErrTransactionNotActive
	}

	// Mark as deleted in write set
	txn.WriteSet[key] = &VersionedValue{
		Value:      nil,
		CreatedBy:  txn.ID,
		DeletedBy:  txn.ID,
	}

	return nil
}

// maybeGarbageCollect triggers garbage collection if needed
func (tm *TransactionManager) maybeGarbageCollect() {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Find minimum read version among active transactions
	minReadVersion := atomic.LoadUint64(&tm.nextVersion)
	for _, txn := range tm.activeTxns {
		if txn.ReadVersion < minReadVersion {
			minReadVersion = txn.ReadVersion
		}
	}

	// Remove versions older than minReadVersion
	tm.versionStore.GarbageCollect(minReadVersion)
}

// GetActiveTransactions returns the number of active transactions
func (tm *TransactionManager) GetActiveTransactions() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.activeTxns)
}

// GetWriteSet returns a copy of the transaction's write set
// This is used for savepoint functionality
func (txn *Transaction) GetWriteSet() map[string]*VersionedValue {
	txn.mu.RLock()
	defer txn.mu.RUnlock()

	writeSetCopy := make(map[string]*VersionedValue)
	for key, val := range txn.WriteSet {
		writeSetCopy[key] = &VersionedValue{
			Value:      val.Value,
			Version:    val.Version,
			CreatedBy:  val.CreatedBy,
			DeletedBy:  val.DeletedBy,
			CommitTime: val.CommitTime,
		}
	}
	return writeSetCopy
}

// GetReadSet returns a copy of the transaction's read set
// This is used for savepoint functionality
func (txn *Transaction) GetReadSet() map[string]uint64 {
	txn.mu.RLock()
	defer txn.mu.RUnlock()

	readSetCopy := make(map[string]uint64)
	for key, version := range txn.ReadSet {
		readSetCopy[key] = version
	}
	return readSetCopy
}

// SetWriteSet replaces the transaction's write set
// This is used for savepoint rollback functionality
func (txn *Transaction) SetWriteSet(writeSet map[string]*VersionedValue) {
	txn.mu.Lock()
	defer txn.mu.Unlock()

	txn.WriteSet = make(map[string]*VersionedValue)
	for key, val := range writeSet {
		txn.WriteSet[key] = &VersionedValue{
			Value:      val.Value,
			Version:    val.Version,
			CreatedBy:  val.CreatedBy,
			DeletedBy:  val.DeletedBy,
			CommitTime: val.CommitTime,
		}
	}
}

// SetReadSet replaces the transaction's read set
// This is used for savepoint rollback functionality
func (txn *Transaction) SetReadSet(readSet map[string]uint64) {
	txn.mu.Lock()
	defer txn.mu.Unlock()

	txn.ReadSet = make(map[string]uint64)
	for key, version := range readSet {
		txn.ReadSet[key] = version
	}
}
