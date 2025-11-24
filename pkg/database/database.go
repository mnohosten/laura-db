package database

import (
	"fmt"
	"sync"
	"time"

	"github.com/mnohosten/laura-db/pkg/mvcc"
	"github.com/mnohosten/laura-db/pkg/storage"
)

// Database represents a database instance
type Database struct {
	name         string
	collections  map[string]*Collection
	storage      *storage.StorageEngine
	txnMgr       *mvcc.TransactionManager
	mu           sync.RWMutex
	isOpen       bool
	ttlStopChan  chan struct{} // Channel to signal TTL cleanup goroutine to stop
	ttlWaitGroup sync.WaitGroup
}

// Config holds database configuration
type Config struct {
	DataDir        string
	BufferPoolSize int
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

	db := &Database{
		name:        "default",
		collections: make(map[string]*Collection),
		storage:     storageEngine,
		txnMgr:      txnMgr,
		isOpen:      true,
		ttlStopChan: make(chan struct{}),
	}

	// Start TTL cleanup goroutine
	db.startTTLCleanup()

	return db, nil
}

// Collection returns a collection, creating it if it doesn't exist
func (db *Database) Collection(name string) *Collection {
	db.mu.Lock()
	defer db.mu.Unlock()

	if coll, exists := db.collections[name]; exists {
		return coll
	}

	// Create new collection
	coll := NewCollection(name, db.txnMgr)
	db.collections[name] = coll
	return coll
}

// CreateCollection explicitly creates a collection
func (db *Database) CreateCollection(name string) (*Collection, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.collections[name]; exists {
		return nil, fmt.Errorf("collection %s already exists", name)
	}

	coll := NewCollection(name, db.txnMgr)
	db.collections[name] = coll
	return coll, nil
}

// DropCollection drops a collection
func (db *Database) DropCollection(name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.collections[name]; !exists {
		return fmt.Errorf("collection %s does not exist", name)
	}

	delete(db.collections, name)
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
