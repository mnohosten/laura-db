package encryption

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/mnohosten/laura-db/pkg/storage"
)

// EncryptedStorageEngine wraps a storage engine with encryption
type EncryptedStorageEngine struct {
	diskMgr    *EncryptedDiskManager
	bufferPool *storage.BufferPool
	wal        *EncryptedWAL
	mu         sync.RWMutex
	dataDir    string
	isOpen     bool
	config     *StorageConfig
}

// StorageConfig holds encrypted storage engine configuration
type StorageConfig struct {
	DataDir          string
	BufferPoolSize   int
	EncryptionConfig *Config
}

// DefaultStorageConfig returns default configuration (no encryption)
func DefaultStorageConfig(dataDir string) *StorageConfig {
	return &StorageConfig{
		DataDir:          dataDir,
		BufferPoolSize:   1000,
		EncryptionConfig: DefaultConfig(),
	}
}

// NewEncryptedStorageEngine creates a new encrypted storage engine
func NewEncryptedStorageEngine(config *StorageConfig) (*EncryptedStorageEngine, error) {
	// Create data directory if it doesn't exist
	if err := ensureDir(config.DataDir); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open encrypted disk manager
	dataPath := filepath.Join(config.DataDir, "data.db")
	diskMgr, err := NewEncryptedDiskManager(dataPath, config.EncryptionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create encrypted disk manager: %w", err)
	}

	// Open encrypted WAL
	walPath := filepath.Join(config.DataDir, "wal.log")
	wal, err := NewEncryptedWAL(walPath, config.EncryptionConfig)
	if err != nil {
		diskMgr.Close()
		return nil, fmt.Errorf("failed to create encrypted WAL: %w", err)
	}

	// Create buffer pool (uses the underlying DiskManager interface)
	// We need to create a wrapper that implements DiskManager interface
	bufferPool := storage.NewBufferPool(config.BufferPoolSize, &diskManagerAdapter{diskMgr})

	engine := &EncryptedStorageEngine{
		diskMgr:    diskMgr,
		bufferPool: bufferPool,
		wal:        wal,
		dataDir:    config.DataDir,
		isOpen:     true,
		config:     config,
	}

	// Perform recovery if needed
	if err := engine.recover(); err != nil {
		engine.Close()
		return nil, fmt.Errorf("failed to recover: %w", err)
	}

	return engine, nil
}

// diskManagerAdapter adapts EncryptedDiskManager to storage.DiskManager interface
type diskManagerAdapter struct {
	*EncryptedDiskManager
}

// recover performs crash recovery by replaying the encrypted WAL
func (ese *EncryptedStorageEngine) recover() error {
	records, err := ese.wal.Replay()
	if err != nil {
		return fmt.Errorf("failed to replay WAL: %w", err)
	}

	if len(records) == 0 {
		return nil // Nothing to recover
	}

	// Replay log records (same logic as regular storage engine)
	for _, record := range records {
		switch record.Type {
		case storage.LogRecordInsert, storage.LogRecordUpdate:
			// Fetch page and update LSN
			page, err := ese.bufferPool.FetchPage(record.PageID)
			if err != nil {
				return fmt.Errorf("failed to fetch page during recovery: %w", err)
			}

			// Update page LSN to match log
			page.LSN = record.LSN
			page.MarkDirty()

			ese.bufferPool.UnpinPage(record.PageID, true)

		case storage.LogRecordCheckpoint:
			// Checkpoint - can skip earlier records in production
			continue

		case storage.LogRecordCommit, storage.LogRecordAbort:
			// Transaction management - handled by MVCC layer
			continue
		}
	}

	// Flush all recovered pages
	if err := ese.bufferPool.FlushAllPages(); err != nil {
		return fmt.Errorf("failed to flush pages after recovery: %w", err)
	}

	return nil
}

// AllocatePage allocates a new page
func (ese *EncryptedStorageEngine) AllocatePage() (*storage.Page, error) {
	if !ese.isOpen {
		return nil, fmt.Errorf("storage engine is closed")
	}

	return ese.bufferPool.NewPage()
}

// FetchPage retrieves a page by ID
func (ese *EncryptedStorageEngine) FetchPage(pageID storage.PageID) (*storage.Page, error) {
	if !ese.isOpen {
		return nil, fmt.Errorf("storage engine is closed")
	}

	return ese.bufferPool.FetchPage(pageID)
}

// UnpinPage unpins a page (allows it to be evicted)
func (ese *EncryptedStorageEngine) UnpinPage(pageID storage.PageID, isDirty bool) error {
	return ese.bufferPool.UnpinPage(pageID, isDirty)
}

// FlushPage writes a specific page to disk
func (ese *EncryptedStorageEngine) FlushPage(pageID storage.PageID) error {
	return ese.bufferPool.FlushPage(pageID)
}

// FlushAll writes all dirty pages to disk
func (ese *EncryptedStorageEngine) FlushAll() error {
	if err := ese.bufferPool.FlushAllPages(); err != nil {
		return err
	}
	return ese.wal.Flush()
}

// LogOperation writes an operation to the encrypted WAL
func (ese *EncryptedStorageEngine) LogOperation(record *storage.LogRecord) (uint64, error) {
	return ese.wal.Append(record)
}

// Checkpoint creates a checkpoint in the WAL
func (ese *EncryptedStorageEngine) Checkpoint() error {
	// Flush all dirty pages
	if err := ese.bufferPool.FlushAllPages(); err != nil {
		return fmt.Errorf("failed to flush pages: %w", err)
	}

	// Write checkpoint record
	if err := ese.wal.Checkpoint(); err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	// Sync disk
	if err := ese.diskMgr.Sync(); err != nil {
		return fmt.Errorf("failed to sync disk: %w", err)
	}

	return nil
}

// Close closes the storage engine
func (ese *EncryptedStorageEngine) Close() error {
	ese.mu.Lock()
	defer ese.mu.Unlock()

	if !ese.isOpen {
		return nil
	}

	// Flush all dirty pages
	if err := ese.bufferPool.FlushAllPages(); err != nil {
		return fmt.Errorf("failed to flush pages on close: %w", err)
	}

	// Close WAL
	if err := ese.wal.Close(); err != nil {
		return fmt.Errorf("failed to close WAL: %w", err)
	}

	// Close disk manager
	if err := ese.diskMgr.Close(); err != nil {
		return fmt.Errorf("failed to close disk manager: %w", err)
	}

	ese.isOpen = false
	return nil
}

// Stats returns storage engine statistics including encryption info
func (ese *EncryptedStorageEngine) Stats() map[string]interface{} {
	stats := map[string]interface{}{
		"buffer_pool": ese.bufferPool.Stats(),
		"disk":        ese.diskMgr.Stats(),
		"encryption": map[string]interface{}{
			"algorithm": ese.config.EncryptionConfig.Algorithm.String(),
			"enabled":   ese.config.EncryptionConfig.Algorithm != AlgorithmNone,
		},
	}
	return stats
}

// GetEncryptionConfig returns the encryption configuration
func (ese *EncryptedStorageEngine) GetEncryptionConfig() *Config {
	return ese.config.EncryptionConfig
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}
