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
	IndexTypeText
	IndexType2D       // 2d planar geospatial index
	IndexType2DSphere // 2dsphere spherical geospatial index
)

// Index represents a database index
type Index struct {
	name       string
	fieldPaths []string               // Multiple fields for compound indexes
	indexType  IndexType
	isUnique   bool
	filter     map[string]interface{} // Partial index filter (nil = full index)
	btree      *BTree
	stats      *IndexStats
	mu         sync.RWMutex
}

// IndexConfig holds configuration for creating an index
type IndexConfig struct {
	Name       string
	FieldPath  string                 // Single field (deprecated, use FieldPaths)
	FieldPaths []string               // Multiple fields for compound indexes
	Type       IndexType
	Unique     bool
	Order      int                    // B-tree order
	Filter     map[string]interface{} // Partial index filter expression
}

// NewIndex creates a new index
func NewIndex(config *IndexConfig) *Index {
	// Handle backward compatibility: if FieldPath is set, use it
	fieldPaths := config.FieldPaths
	if len(fieldPaths) == 0 && config.FieldPath != "" {
		fieldPaths = []string{config.FieldPath}
	}

	// Ensure at least one field is specified
	if len(fieldPaths) == 0 {
		panic("index must have at least one field")
	}

	idx := &Index{
		name:       config.Name,
		fieldPaths: fieldPaths,
		indexType:  config.Type,
		isUnique:   config.Unique,
		filter:     config.Filter,
		stats:      NewIndexStats(),
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

	err := idx.btree.Insert(key, value)
	if err == nil {
		// Mark statistics as stale after successful insert
		idx.stats.Update()
	}
	return err
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

	err := idx.btree.Delete(key)
	if err == nil {
		// Mark statistics as stale after successful delete
		idx.stats.Update()
	}
	return err
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

// FieldPath returns the first field path (for backward compatibility)
func (idx *Index) FieldPath() string {
	if len(idx.fieldPaths) > 0 {
		return idx.fieldPaths[0]
	}
	return ""
}

// FieldPaths returns all field paths this index covers
func (idx *Index) FieldPaths() []string {
	return idx.fieldPaths
}

// IsCompound returns true if this is a compound index (multiple fields)
func (idx *Index) IsCompound() bool {
	return len(idx.fieldPaths) > 1
}

// IsUnique returns whether this is a unique index
func (idx *Index) IsUnique() bool {
	return idx.isUnique
}

// IsPartial returns true if this is a partial index (has a filter)
func (idx *Index) IsPartial() bool {
	return idx.filter != nil && len(idx.filter) > 0
}

// Filter returns the partial index filter expression (nil if not partial)
func (idx *Index) Filter() map[string]interface{} {
	return idx.filter
}

// Stats returns index statistics
func (idx *Index) Stats() map[string]interface{} {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	stats := map[string]interface{}{
		"name":        idx.name,
		"field_path":  idx.FieldPath(),      // First field for compatibility
		"field_paths": idx.fieldPaths,       // All fields
		"is_compound": idx.IsCompound(),     // Is this a compound index
		"is_partial":  idx.IsPartial(),      // Is this a partial index
		"type":        idx.indexType,
		"unique":      idx.isUnique,
		"size":        idx.btree.Size(),
		"height":      idx.btree.Height(),
	}

	if idx.IsPartial() {
		stats["filter"] = idx.filter
	}

	// Add index statistics
	for k, v := range idx.stats.ToMap() {
		stats[k] = v
	}

	return stats
}

// Analyze recalculates index statistics by scanning the entire index
func (idx *Index) Analyze() {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Collect all keys and values from the index
	keys, _ := idx.btree.RangeScan(nil, nil)

	if len(keys) == 0 {
		idx.stats.SetStats(0, 0, nil, nil)
		return
	}

	// Count unique keys
	uniqueKeys := make(map[interface{}]bool)
	var minValue, maxValue interface{}

	for i, key := range keys {
		uniqueKeys[key] = true

		// Track min/max
		if i == 0 {
			minValue = key
			maxValue = key
		} else {
			if idx.btree.compare(key, minValue) < 0 {
				minValue = key
			}
			if idx.btree.compare(key, maxValue) > 0 {
				maxValue = key
			}
		}
	}

	idx.stats.SetStats(len(keys), len(uniqueKeys), minValue, maxValue)
}

// GetStatistics returns the index statistics object
func (idx *Index) GetStatistics() *IndexStats {
	return idx.stats
}
