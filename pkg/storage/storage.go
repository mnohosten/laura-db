package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// StorageEngine manages data persistence with WAL and buffer pool
type StorageEngine struct {
	diskMgr    *DiskManager
	bufferPool *BufferPool
	wal        *WAL
	mu         sync.RWMutex
	dataDir    string
	isOpen     bool
}

// Config holds storage engine configuration
type Config struct {
	DataDir        string
	BufferPoolSize int // Number of pages to cache
}

// DefaultConfig returns default configuration
func DefaultConfig(dataDir string) *Config {
	return &Config{
		DataDir:        dataDir,
		BufferPoolSize: 1000, // Cache 1000 pages (~4MB)
	}
}

// NewStorageEngine creates a new storage engine
func NewStorageEngine(config *Config) (*StorageEngine, error) {
	// Create data directory if it doesn't exist
	if err := ensureDir(config.DataDir); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open disk manager
	dataPath := filepath.Join(config.DataDir, "data.db")
	diskMgr, err := NewDiskManager(dataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create disk manager: %w", err)
	}

	// Open WAL
	walPath := filepath.Join(config.DataDir, "wal.log")
	wal, err := NewWAL(walPath)
	if err != nil {
		diskMgr.Close()
		return nil, fmt.Errorf("failed to create WAL: %w", err)
	}

	// Create buffer pool
	bufferPool := NewBufferPool(config.BufferPoolSize, diskMgr)

	engine := &StorageEngine{
		diskMgr:    diskMgr,
		bufferPool: bufferPool,
		wal:        wal,
		dataDir:    config.DataDir,
		isOpen:     true,
	}

	// Perform recovery if needed
	if err := engine.recover(); err != nil {
		engine.Close()
		return nil, fmt.Errorf("failed to recover: %w", err)
	}

	return engine, nil
}

// recover performs crash recovery by replaying the WAL
func (se *StorageEngine) recover() error {
	records, err := se.wal.Replay()
	if err != nil {
		return fmt.Errorf("failed to replay WAL: %w", err)
	}

	if len(records) == 0 {
		return nil // Nothing to recover
	}

	// Replay log records
	for _, record := range records {
		switch record.Type {
		case LogRecordInsert, LogRecordUpdate:
			// Fetch page and update LSN
			page, err := se.bufferPool.FetchPage(record.PageID)
			if err != nil {
				return fmt.Errorf("failed to fetch page during recovery: %w", err)
			}

			// Update page LSN to match log
			page.LSN = record.LSN
			page.MarkDirty()

			se.bufferPool.UnpinPage(record.PageID, true)

		case LogRecordCheckpoint:
			// Checkpoint - can skip earlier records in production
			continue

		case LogRecordCommit, LogRecordAbort:
			// Transaction management - handled by MVCC layer
			continue
		}
	}

	// Flush all recovered pages
	if err := se.bufferPool.FlushAllPages(); err != nil {
		return fmt.Errorf("failed to flush pages after recovery: %w", err)
	}

	return nil
}

// AllocatePage allocates a new page
func (se *StorageEngine) AllocatePage() (*Page, error) {
	if !se.isOpen {
		return nil, fmt.Errorf("storage engine is closed")
	}

	return se.bufferPool.NewPage()
}

// FetchPage retrieves a page by ID
func (se *StorageEngine) FetchPage(pageID PageID) (*Page, error) {
	if !se.isOpen {
		return nil, fmt.Errorf("storage engine is closed")
	}

	return se.bufferPool.FetchPage(pageID)
}

// UnpinPage unpins a page (allows it to be evicted)
func (se *StorageEngine) UnpinPage(pageID PageID, isDirty bool) error {
	return se.bufferPool.UnpinPage(pageID, isDirty)
}

// FlushPage writes a specific page to disk
func (se *StorageEngine) FlushPage(pageID PageID) error {
	return se.bufferPool.FlushPage(pageID)
}

// FlushAll writes all dirty pages to disk
func (se *StorageEngine) FlushAll() error {
	if err := se.bufferPool.FlushAllPages(); err != nil {
		return err
	}
	return se.wal.Flush()
}

// LogOperation writes an operation to the WAL
func (se *StorageEngine) LogOperation(record *LogRecord) (uint64, error) {
	return se.wal.Append(record)
}

// Checkpoint creates a checkpoint in the WAL
func (se *StorageEngine) Checkpoint() error {
	// Flush all dirty pages
	if err := se.bufferPool.FlushAllPages(); err != nil {
		return fmt.Errorf("failed to flush pages: %w", err)
	}

	// Write checkpoint record
	if err := se.wal.Checkpoint(); err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	// Sync disk
	if err := se.diskMgr.Sync(); err != nil {
		return fmt.Errorf("failed to sync disk: %w", err)
	}

	return nil
}

// Close closes the storage engine
func (se *StorageEngine) Close() error {
	se.mu.Lock()
	defer se.mu.Unlock()

	if !se.isOpen {
		return nil
	}

	// Flush all dirty pages
	if err := se.bufferPool.FlushAllPages(); err != nil {
		return fmt.Errorf("failed to flush pages on close: %w", err)
	}

	// Close WAL
	if err := se.wal.Close(); err != nil {
		return fmt.Errorf("failed to close WAL: %w", err)
	}

	// Close disk manager
	if err := se.diskMgr.Close(); err != nil {
		return fmt.Errorf("failed to close disk manager: %w", err)
	}

	se.isOpen = false
	return nil
}

// Stats returns storage engine statistics
func (se *StorageEngine) Stats() map[string]interface{} {
	return map[string]interface{}{
		"buffer_pool": se.bufferPool.Stats(),
		"disk":        se.diskMgr.Stats(),
	}
}

// DiskManager returns the disk manager
func (se *StorageEngine) DiskManager() *DiskManager {
	return se.diskMgr
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}
