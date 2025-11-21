package index

import (
	"fmt"
	"sync"
)

// IndexType represents the type of index
type IndexType int

const (
	IndexTypeBTree IndexType = iota
	IndexTypeHash
)

// Index represents a database index
type Index struct {
	name       string
	fieldPath  string
	indexType  IndexType
	isUnique   bool
	btree      *BTree
	mu         sync.RWMutex
}

// IndexConfig holds configuration for creating an index
type IndexConfig struct {
	Name      string
	FieldPath string
	Type      IndexType
	Unique    bool
	Order     int // B-tree order
}

// NewIndex creates a new index
func NewIndex(config *IndexConfig) *Index {
	idx := &Index{
		name:      config.Name,
		fieldPath: config.FieldPath,
		indexType: config.Type,
		isUnique:  config.Unique,
	}

	switch config.Type {
	case IndexTypeBTree:
		order := config.Order
		if order == 0 {
			order = 32 // Default order
		}
		idx.btree = NewBTree(order)
	default:
		idx.btree = NewBTree(32)
	}

	return idx
}

// Insert inserts a key-value pair into the index
func (idx *Index) Insert(key interface{}, value interface{}) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.isUnique {
		// Check if key already exists
		if _, exists := idx.btree.Search(key); exists {
			return fmt.Errorf("duplicate key in unique index: %v", key)
		}
	}

	return idx.btree.Insert(key, value)
}

// Search finds values by key
func (idx *Index) Search(key interface{}) (interface{}, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.btree.Search(key)
}

// Delete removes a key from the index
func (idx *Index) Delete(key interface{}) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	return idx.btree.Delete(key)
}

// RangeScan performs a range query
func (idx *Index) RangeScan(start, end interface{}) ([]interface{}, []interface{}) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.btree.RangeScan(start, end)
}

// Size returns the number of entries in the index
func (idx *Index) Size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.btree.Size()
}

// Name returns the index name
func (idx *Index) Name() string {
	return idx.name
}

// FieldPath returns the field path this index covers
func (idx *Index) FieldPath() string {
	return idx.fieldPath
}

// IsUnique returns whether this is a unique index
func (idx *Index) IsUnique() bool {
	return idx.isUnique
}

// Stats returns index statistics
func (idx *Index) Stats() map[string]interface{} {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return map[string]interface{}{
		"name":       idx.name,
		"field_path": idx.fieldPath,
		"type":       idx.indexType,
		"unique":     idx.isUnique,
		"size":       idx.btree.Size(),
		"height":     idx.btree.Height(),
	}
}
