package database

import (
	"fmt"
	"sync"
	"time"

	"github.com/mnohosten/laura-db/pkg/audit"
	"github.com/mnohosten/laura-db/pkg/mvcc"
	"github.com/mnohosten/laura-db/pkg/storage"
)

// Database represents a database instance
type Database struct {
	name          string
	collections   map[string]*Collection
	storage       *storage.StorageEngine
	txnMgr        *mvcc.TransactionManager
	auditLogger   *audit.AuditLogger // Audit logger for tracking operations
	cursorManager *CursorManager     // Cursor manager for server-side cursors
	mu            sync.RWMutex
	isOpen        bool
	ttlStopChan   chan struct{} // Channel to signal TTL cleanup goroutine to stop
	ttlWaitGroup  sync.WaitGroup
}

// Config holds database configuration
type Config struct {
	DataDir        string
	BufferPoolSize int
	AuditConfig    *audit.Config // Optional audit logging configuration
}

// DefaultConfig returns default configuration
func DefaultConfig(dataDir string) *Config {
	return &Config{
		DataDir:        dataDir,
		BufferPoolSize: 1000,
	}
}

// Open opens or creates a database
func Open(config *Config) (*Database, error) {
	// Create storage engine
	storageConfig := storage.DefaultConfig(config.DataDir)
	storageConfig.BufferPoolSize = config.BufferPoolSize

	storageEngine, err := storage.NewStorageEngine(storageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage engine: %w", err)
	}

	// Create transaction manager
	txnMgr := mvcc.NewTransactionManager()

	// Create audit logger if configured
	var auditLogger *audit.AuditLogger
	if config.AuditConfig != nil {
		var err error
		auditLogger, err = audit.NewAuditLogger(config.AuditConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create audit logger: %w", err)
		}
	}

	db := &Database{
		name:          "default",
		collections:   make(map[string]*Collection),
		storage:       storageEngine,
		txnMgr:        txnMgr,
		auditLogger:   auditLogger,
		cursorManager: NewCursorManager(),
		isOpen:        true,
		ttlStopChan:   make(chan struct{}),
	}

	// Start TTL cleanup goroutine
	db.startTTLCleanup()

	// Start cursor cleanup goroutine
	db.startCursorCleanup()

	return db, nil
}

// Collection returns a collection, creating it if it doesn't exist
func (db *Database) Collection(name string) *Collection {
	db.mu.Lock()
	defer db.mu.Unlock()

	if coll, exists := db.collections[name]; exists {
		return coll
	}

	// Create document store for this collection
	docStore := NewDocumentStore(db.storage.DiskManager(), 1000) // 1000 documents cache

	// Create new collection
	coll := NewCollection(name, db.txnMgr, docStore)
	coll.database = db.name
	coll.auditLogger = db.auditLogger
	db.collections[name] = coll
	return coll
}

// CreateCollection explicitly creates a collection
func (db *Database) CreateCollection(name string) (*Collection, error) {
	start := time.Now()
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.collections[name]; exists {
		if db.auditLogger != nil {
			db.auditLogger.LogOperation(audit.OperationCreateCollection, name, db.name, "", false, time.Since(start), fmt.Errorf("collection already exists"), nil)
		}
		return nil, fmt.Errorf("collection %s already exists", name)
	}

	// Create document store for this collection
	docStore := NewDocumentStore(db.storage.DiskManager(), 1000) // 1000 documents cache

	coll := NewCollection(name, db.txnMgr, docStore)
	coll.database = db.name
	coll.auditLogger = db.auditLogger
	db.collections[name] = coll

	// Log successful collection creation
	if db.auditLogger != nil {
		db.auditLogger.LogOperation(audit.OperationCreateCollection, name, db.name, "", true, time.Since(start), nil, nil)
	}

	return coll, nil
}

// DropCollection drops a collection
func (db *Database) DropCollection(name string) error {
	start := time.Now()
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.collections[name]; !exists {
		err := fmt.Errorf("collection %s does not exist", name)
		if db.auditLogger != nil {
			db.auditLogger.LogOperation(audit.OperationDropCollection, name, db.name, "", false, time.Since(start), err, nil)
		}
		return err
	}

	delete(db.collections, name)

	// Log successful collection drop
	if db.auditLogger != nil {
		db.auditLogger.LogOperation(audit.OperationDropCollection, name, db.name, "", true, time.Since(start), nil, nil)
	}

	return nil
}

// RenameCollection renames a collection
func (db *Database) RenameCollection(oldName, newName string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if !db.isOpen {
		return fmt.Errorf("database is closed")
	}

	// Check if old collection exists
	coll, exists := db.collections[oldName]
	if !exists {
		return fmt.Errorf("collection %s does not exist", oldName)
	}

	// Check if new collection name already exists
	if _, exists := db.collections[newName]; exists {
		return fmt.Errorf("collection %s already exists", newName)
	}

	// Rename the collection
	coll.name = newName
	db.collections[newName] = coll
	delete(db.collections, oldName)

	return nil
}

// ListCollections returns all collection names
func (db *Database) ListCollections() []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	names := make([]string, 0, len(db.collections))
	for name := range db.collections {
		names = append(names, name)
	}
	return names
}

// BeginTransaction starts a new transaction
func (db *Database) BeginTransaction() *mvcc.Transaction {
	return db.txnMgr.Begin()
}

// CommitTransaction commits a transaction
func (db *Database) CommitTransaction(txn *mvcc.Transaction) error {
	return db.txnMgr.Commit(txn)
}

// AbortTransaction aborts a transaction
func (db *Database) AbortTransaction(txn *mvcc.Transaction) error {
	return db.txnMgr.Abort(txn)
}

// Close closes the database
func (db *Database) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if !db.isOpen {
		return nil
	}

	// Stop TTL cleanup goroutine
	close(db.ttlStopChan)
	db.ttlWaitGroup.Wait()

	// Flush all data
	if err := db.storage.FlushAll(); err != nil {
		return fmt.Errorf("failed to flush data: %w", err)
	}

	// Checkpoint
	if err := db.storage.Checkpoint(); err != nil {
		return fmt.Errorf("failed to checkpoint: %w", err)
	}

	// Close storage engine
	if err := db.storage.Close(); err != nil {
		return fmt.Errorf("failed to close storage: %w", err)
	}

	db.isOpen = false
	return nil
}

// Stats returns database statistics
func (db *Database) Stats() map[string]interface{} {
	db.mu.RLock()
	defer db.mu.RUnlock()

	collectionStats := make(map[string]interface{})
	for name, coll := range db.collections {
		collectionStats[name] = coll.Stats()
	}

	return map[string]interface{}{
		"name":                  db.name,
		"collections":           len(db.collections),
		"collection_stats":      collectionStats,
		"active_transactions":   db.txnMgr.GetActiveTransactions(),
		"storage_stats":         db.storage.Stats(),
	}
}

// startTTLCleanup starts a background goroutine that periodically cleans up expired documents
func (db *Database) startTTLCleanup() {
	db.ttlWaitGroup.Add(1)
	go db.ttlCleanupLoop()
}

// ttlCleanupLoop runs the TTL cleanup process every 60 seconds
func (db *Database) ttlCleanupLoop() {
	defer db.ttlWaitGroup.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			db.cleanupExpiredDocuments()
		case <-db.ttlStopChan:
			return
		}
	}
}

// cleanupExpiredDocuments runs cleanup on all collections with TTL indexes
func (db *Database) cleanupExpiredDocuments() {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, coll := range db.collections {
		coll.CleanupExpiredDocuments()
	}
}

// startCursorCleanup starts the cursor cleanup goroutine
func (db *Database) startCursorCleanup() {
	db.ttlWaitGroup.Add(1)
	go db.cursorCleanupLoop()
}

// cursorCleanupLoop runs the cursor cleanup process every 60 seconds
func (db *Database) cursorCleanupLoop() {
	defer db.ttlWaitGroup.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			db.cursorManager.CleanupTimedOutCursors()
		case <-db.ttlStopChan:
			return
		}
	}
}

// CursorManager returns the database's cursor manager
func (db *Database) CursorManager() *CursorManager {
	return db.cursorManager
}
