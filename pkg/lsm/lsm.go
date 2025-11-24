package lsm

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// LSMTree is a Log-Structured Merge tree
// Optimized for write-heavy workloads with sequential disk I/O
type LSMTree struct {
	dir         string
	memTable    *MemTable
	immutables  []*MemTable // Immutable memtables being flushed
	sstables    []*SSTable  // SSTables sorted newest to oldest
	mu          sync.RWMutex
	nextSSTableID int
	closed      bool

	// Configuration
	memTableSize  int64
	indexInterval int

	// Background workers
	flushChan     chan *MemTable
	compactChan   chan struct{}
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// Config holds LSM tree configuration
type Config struct {
	Dir           string
	MemTableSize  int64 // Max memtable size in bytes
	IndexInterval int   // Write index entry every N keys
}

// DefaultConfig returns default configuration
func DefaultConfig(dir string) *Config {
	return &Config{
		Dir:           dir,
		MemTableSize:  4 * 1024 * 1024, // 4MB
		IndexInterval: 100,               // Index every 100 keys
	}
}

// NewLSMTree creates a new LSM tree
func NewLSMTree(config *Config) (*LSMTree, error) {
	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	lsm := &LSMTree{
		dir:           config.Dir,
		memTable:      NewMemTable(config.MemTableSize),
		immutables:    make([]*MemTable, 0),
		sstables:      make([]*SSTable, 0),
		nextSSTableID: 0,
		memTableSize:  config.MemTableSize,
		indexInterval: config.IndexInterval,
		flushChan:     make(chan *MemTable, 10),
		compactChan:   make(chan struct{}, 1),
		stopChan:      make(chan struct{}),
		closed:        false,
	}

	// Load existing SSTables
	if err := lsm.loadSSTables(); err != nil {
		return nil, fmt.Errorf("failed to load sstables: %w", err)
	}

	// Start background workers
	lsm.wg.Add(2)
	go lsm.flushWorker()
	go lsm.compactionWorker()

	return lsm, nil
}

// loadSSTables loads existing SSTables from disk
func (lsm *LSMTree) loadSSTables() error {
	pattern := filepath.Join(lsm.dir, "sstable_*.sst")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	// Sort by ID (newest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i] > matches[j]
	})

	for _, path := range matches {
		sst, err := OpenSSTable(path)
		if err != nil {
			return fmt.Errorf("failed to open sstable %s: %w", path, err)
		}
		lsm.sstables = append(lsm.sstables, sst)

		// Update next ID
		var id int
		if _, err := fmt.Sscanf(filepath.Base(path), "sstable_%d.sst", &id); err == nil {
			if id >= lsm.nextSSTableID {
				lsm.nextSSTableID = id + 1
			}
		}
	}

	return nil
}

// Put inserts or updates a key-value pair
func (lsm *LSMTree) Put(key, value []byte) error {
	lsm.mu.Lock()
	defer lsm.mu.Unlock()

	if lsm.closed {
		return ErrClosed
	}

	timestamp := time.Now().UnixNano()

	// Insert into memtable
	if err := lsm.memTable.Put(key, value, timestamp); err != nil {
		return err
	}

	// Check if memtable is full
	var immutable *MemTable
	if lsm.memTable.IsFull() {
		// Make current memtable immutable
		lsm.immutables = append(lsm.immutables, lsm.memTable)
		immutable = lsm.memTable
		lsm.memTable = NewMemTable(lsm.memTableSize)
	}

	// Release lock before sending to channel to avoid deadlock
	lsm.mu.Unlock()

	// Trigger flush if needed
	if immutable != nil {
		lsm.flushChan <- immutable
	}

	// Re-acquire lock before defer unlock
	lsm.mu.Lock()
	return nil
}

// Get retrieves a value by key
func (lsm *LSMTree) Get(key []byte) ([]byte, bool, error) {
	lsm.mu.RLock()
	defer lsm.mu.RUnlock()

	if lsm.closed {
		return nil, false, ErrClosed
	}

	// Check memtable first
	if entry, found := lsm.memTable.Get(key); found {
		if entry.Deleted {
			return nil, false, nil // Tombstone
		}
		return entry.Value, true, nil
	}

	// Check immutable memtables (newest to oldest)
	for i := len(lsm.immutables) - 1; i >= 0; i-- {
		if entry, found := lsm.immutables[i].Get(key); found {
			if entry.Deleted {
				return nil, false, nil
			}
			return entry.Value, true, nil
		}
	}

	// Check SSTables (newest to oldest)
	for _, sst := range lsm.sstables {
		entry, found, err := sst.Get(key)
		if err != nil {
			return nil, false, err
		}
		if found {
			if entry.Deleted {
				return nil, false, nil
			}
			return entry.Value, true, nil
		}
	}

	return nil, false, nil
}

// Delete marks a key as deleted
func (lsm *LSMTree) Delete(key []byte) error {
	lsm.mu.Lock()
	defer lsm.mu.Unlock()

	if lsm.closed {
		return ErrClosed
	}

	timestamp := time.Now().UnixNano()
	return lsm.memTable.Delete(key, timestamp)
}

// flushWorker handles background memtable flushing
func (lsm *LSMTree) flushWorker() {
	defer lsm.wg.Done()

	for {
		select {
		case memTable := <-lsm.flushChan:
			if err := lsm.flushMemTable(memTable); err != nil {
				// Log error in production, skip for now
				fmt.Printf("flush error: %v\n", err)
			}
		case <-lsm.stopChan:
			return
		}
	}
}

// flushMemTable flushes a memtable to an SSTable
func (lsm *LSMTree) flushMemTable(memTable *MemTable) error {
	lsm.mu.Lock()
	id := lsm.nextSSTableID
	lsm.nextSSTableID++
	lsm.mu.Unlock()

	// Create SSTable writer
	writer, err := NewSSTableWriter(lsm.dir, id, lsm.indexInterval)
	if err != nil {
		return fmt.Errorf("failed to create sstable writer: %w", err)
	}

	// Write all entries from memtable
	iter := memTable.Iterator()
	for iter.Next() {
		entry := iter.Entry()
		if err := writer.Write(entry); err != nil {
			return fmt.Errorf("failed to write entry: %w", err)
		}
	}

	// Finalize SSTable
	sst, err := writer.Finalize()
	if err != nil {
		return fmt.Errorf("failed to finalize sstable: %w", err)
	}

	// Update LSM tree
	lsm.mu.Lock()
	defer lsm.mu.Unlock()

	// Add SSTable to list (at beginning - newest first)
	lsm.sstables = append([]*SSTable{sst}, lsm.sstables...)

	// Remove from immutables
	for i, imm := range lsm.immutables {
		if imm == memTable {
			lsm.immutables = append(lsm.immutables[:i], lsm.immutables[i+1:]...)
			break
		}
	}

	// Trigger compaction if needed
	if len(lsm.sstables) > 4 {
		select {
		case lsm.compactChan <- struct{}{}:
		default:
		}
	}

	return nil
}

// compactionWorker handles background compaction
func (lsm *LSMTree) compactionWorker() {
	defer lsm.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-lsm.compactChan:
			if err := lsm.compact(); err != nil {
				fmt.Printf("compaction error: %v\n", err)
			}
		case <-ticker.C:
			// Periodic compaction check
			lsm.mu.RLock()
			needsCompaction := len(lsm.sstables) > 4
			lsm.mu.RUnlock()

			if needsCompaction {
				if err := lsm.compact(); err != nil {
					fmt.Printf("compaction error: %v\n", err)
				}
			}
		case <-lsm.stopChan:
			return
		}
	}
}

// compact performs compaction of SSTables
// Simple strategy: merge oldest N SSTables
func (lsm *LSMTree) compact() error {
	lsm.mu.Lock()

	if len(lsm.sstables) <= 4 {
		lsm.mu.Unlock()
		return nil
	}

	// Select oldest 4 SSTables for compaction
	numToCompact := 4
	if numToCompact > len(lsm.sstables) {
		numToCompact = len(lsm.sstables)
	}

	toCompact := lsm.sstables[len(lsm.sstables)-numToCompact:]

	// Create a copy to avoid holding references
	toCompactCopy := make([]*SSTable, len(toCompact))
	copy(toCompactCopy, toCompact)

	id := lsm.nextSSTableID
	lsm.nextSSTableID++

	lsm.mu.Unlock()

	// Merge SSTables
	merged, err := lsm.mergeSSTables(toCompactCopy, id)
	if err != nil {
		return fmt.Errorf("failed to merge sstables: %w", err)
	}

	// Update SSTable list
	lsm.mu.Lock()
	defer lsm.mu.Unlock()

	// Remove the compacted SSTables from the list and append the merged one
	// We need to re-filter because new SSTables might have been added
	newList := make([]*SSTable, 0, len(lsm.sstables))
	for _, sst := range lsm.sstables {
		shouldRemove := false
		for _, compacted := range toCompactCopy {
			if sst.path == compacted.path {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			newList = append(newList, sst)
		}
	}
	lsm.sstables = append(newList, merged)

	// Delete old SSTable files
	for _, sst := range toCompactCopy {
		os.Remove(sst.path)
	}

	return nil
}

// mergeSSTables merges multiple SSTables into one
func (lsm *LSMTree) mergeSSTables(sstables []*SSTable, newID int) (*SSTable, error) {
	writer, err := NewSSTableWriter(lsm.dir, newID, lsm.indexInterval)
	if err != nil {
		return nil, err
	}

	// Create iterators for all SSTables
	type iterEntry struct {
		iter  *SSTableIterator
		entry *MemTableEntry
		valid bool
	}

	iters := make([]*iterEntry, len(sstables))
	for i, sst := range sstables {
		iter, err := sst.Iterator()
		if err != nil {
			return nil, err
		}
		iters[i] = &iterEntry{iter: iter, valid: iter.Next()}
		if iters[i].valid {
			iters[i].entry = iter.Entry()
		}
	}

	// Merge entries in sorted order
	var lastKey []byte
	for {
		// Find minimum key among all iterators
		minIdx := -1
		var minEntry *MemTableEntry

		for i, it := range iters {
			if !it.valid {
				continue
			}
			if minIdx == -1 || compareBytes(it.entry.Key, minEntry.Key) < 0 {
				minIdx = i
				minEntry = it.entry
			}
		}

		if minIdx == -1 {
			break // All iterators exhausted
		}

		// Write entry if key is different (deduplicate)
		if lastKey == nil || compareBytes(minEntry.Key, lastKey) != 0 {
			// Skip tombstones during compaction
			if !minEntry.Deleted {
				if err := writer.Write(minEntry); err != nil {
					return nil, err
				}
			}
			// Make a copy of the key to avoid aliasing issues
			lastKey = make([]byte, len(minEntry.Key))
			copy(lastKey, minEntry.Key)
		}

		// Advance iterator
		iters[minIdx].valid = iters[minIdx].iter.Next()
		if iters[minIdx].valid {
			iters[minIdx].entry = iters[minIdx].iter.Entry()
		}
	}

	// Close all iterators
	for _, it := range iters {
		it.iter.Close()
	}

	return writer.Finalize()
}

// compareBytes compares two byte slices
func compareBytes(a, b []byte) int {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	for i := 0; i < minLen; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}

	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

// Flush waits for all pending flushes to complete
func (lsm *LSMTree) Flush() error {
	// Wait until all immutables are flushed
	for {
		lsm.mu.RLock()
		numImmutables := len(lsm.immutables)
		lsm.mu.RUnlock()

		if numImmutables == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// Close closes the LSM tree
func (lsm *LSMTree) Close() error {
	lsm.mu.Lock()
	if lsm.closed {
		lsm.mu.Unlock()
		return nil
	}
	lsm.closed = true

	// Capture current memtable and immutables before releasing lock
	currentMemTable := lsm.memTable
	immutables := make([]*MemTable, len(lsm.immutables))
	copy(immutables, lsm.immutables)

	lsm.mu.Unlock()

	// Stop background workers
	close(lsm.stopChan)
	lsm.wg.Wait()

	// Flush current memtable if it has any data
	if currentMemTable != nil && currentMemTable.Size() > 0 {
		if err := lsm.flushMemTable(currentMemTable); err != nil {
			return err
		}
	}

	// Flush any remaining immutable memtables
	for _, memTable := range immutables {
		if err := lsm.flushMemTable(memTable); err != nil {
			return err
		}
	}

	return nil
}

// Stats returns LSM tree statistics
func (lsm *LSMTree) Stats() map[string]interface{} {
	lsm.mu.RLock()
	defer lsm.mu.RUnlock()

	totalEntries := 0
	for _, sst := range lsm.sstables {
		totalEntries += sst.numEntries
	}

	return map[string]interface{}{
		"memtable_size":     lsm.memTable.Size(),
		"num_immutables":    len(lsm.immutables),
		"num_sstables":      len(lsm.sstables),
		"total_entries":     totalEntries,
		"next_sstable_id":   lsm.nextSSTableID,
	}
}
