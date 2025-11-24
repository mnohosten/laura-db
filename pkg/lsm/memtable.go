package lsm

import (
	"sync"
)

// MemTable is an in-memory sorted data structure for LSM tree
// Uses a skip list for O(log n) insert/search with better cache locality than tree structures
type MemTable struct {
	skipList *SkipList
	size     int64 // Size in bytes
	maxSize  int64 // Maximum size before flush
	mu       sync.RWMutex
}

// MemTableEntry represents a key-value entry with metadata
type MemTableEntry struct {
	Key       []byte
	Value     []byte
	Timestamp int64 // For MVCC support
	Deleted   bool  // Tombstone for deletions
}

// NewMemTable creates a new MemTable
func NewMemTable(maxSize int64) *MemTable {
	return &MemTable{
		skipList: NewSkipList(),
		size:     0,
		maxSize:  maxSize,
	}
}

// Put inserts or updates a key-value pair
func (mt *MemTable) Put(key, value []byte, timestamp int64) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	entry := &MemTableEntry{
		Key:       key,
		Value:     value,
		Timestamp: timestamp,
		Deleted:   false,
	}

	// Insert into skip list
	oldSize := mt.skipList.Size()
	mt.skipList.Insert(key, entry)
	newSize := mt.skipList.Size()

	// Update size (approximate)
	entrySize := int64(len(key) + len(value) + 16) // 16 bytes for metadata
	if newSize > oldSize {
		// New entry
		mt.size += entrySize
	} else {
		// Update existing - size might change
		mt.size += entrySize // Simplified - could track old value size
	}

	return nil
}

// Get retrieves a value by key
func (mt *MemTable) Get(key []byte) (*MemTableEntry, bool) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	value, found := mt.skipList.Search(key)
	if !found {
		return nil, false
	}

	entry, ok := value.(*MemTableEntry)
	if !ok {
		return nil, false
	}

	return entry, true
}

// Delete marks a key as deleted (tombstone)
func (mt *MemTable) Delete(key []byte, timestamp int64) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	entry := &MemTableEntry{
		Key:       key,
		Value:     nil,
		Timestamp: timestamp,
		Deleted:   true,
	}

	mt.skipList.Insert(key, entry)
	mt.size += int64(len(key) + 16) // Tombstone overhead

	return nil
}

// Size returns the current size in bytes
func (mt *MemTable) Size() int64 {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.size
}

// IsFull returns true if memtable should be flushed
func (mt *MemTable) IsFull() bool {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.size >= mt.maxSize
}

// Iterator returns an iterator over all entries in sorted order
func (mt *MemTable) Iterator() *MemTableIterator {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	return &MemTableIterator{
		skipList: mt.skipList,
		current:  mt.skipList.head, // Start at head, not first node
	}
}

// MemTableIterator iterates over entries in sorted order
type MemTableIterator struct {
	skipList *SkipList
	current  *SkipListNode
}

// Next advances the iterator
func (it *MemTableIterator) Next() bool {
	if it.current == nil {
		return false
	}
	it.current = it.current.forward[0]
	return it.current != nil
}

// Entry returns the current entry
func (it *MemTableIterator) Entry() *MemTableEntry {
	if it.current == nil {
		return nil
	}
	entry, _ := it.current.value.(*MemTableEntry)
	return entry
}

// Reset resets the iterator to the beginning
func (it *MemTableIterator) Reset() {
	it.current = it.skipList.head
}
