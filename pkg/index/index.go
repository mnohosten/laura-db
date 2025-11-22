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
	name          string
	fieldPaths    []string               // Multiple fields for compound indexes
	indexType     IndexType
	isUnique      bool
	filter        map[string]interface{} // Partial index filter (nil = full index)
	btree         *BTree
	stats         *IndexStats
	buildProgress *IndexBuildProgress // Track background index build progress
	mu            sync.RWMutex
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
	Background bool                   // Build index in background (non-blocking)
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
		name:          config.Name,
		fieldPaths:    fieldPaths,
		indexType:     config.Type,
		isUnique:      config.Unique,
		filter:        config.Filter,
		stats:         NewIndexStats(),
		buildProgress: NewIndexBuildProgress(),
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

	// Add build progress if index is building
	if idx.buildProgress != nil {
		buildState := idx.buildProgress.GetState()
		stats["build_state"] = buildState.String()
		if buildState == IndexStateBuilding {
			stats["build_progress"] = idx.buildProgress.GetProgress()
		}
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

// GetBuildState returns the current index build state
func (idx *Index) GetBuildState() IndexBuildState {
	if idx.buildProgress == nil {
		return IndexStateReady
	}
	return idx.buildProgress.GetState()
}

// GetBuildProgress returns the build progress information
func (idx *Index) GetBuildProgress() map[string]interface{} {
	if idx.buildProgress == nil {
		return map[string]interface{}{
			"state": "ready",
		}
	}
	return idx.buildProgress.GetProgress()
}

// IsReady returns true if the index is ready to use (not building)
func (idx *Index) IsReady() bool {
	return idx.GetBuildState() == IndexStateReady
}

// IsBuilding returns true if the index is currently being built
func (idx *Index) IsBuilding() bool {
	return idx.GetBuildState() == IndexStateBuilding
}

// StartBuild marks the index build as started
func (idx *Index) StartBuild(totalDocs int) {
	if idx.buildProgress != nil {
		idx.buildProgress.Start(totalDocs)
	}
}

// UpdateBuildProgress updates the number of processed documents
func (idx *Index) UpdateBuildProgress(processed int) {
	if idx.buildProgress != nil {
		idx.buildProgress.Update(processed)
	}
}

// IncrementBuildProgress increments the processed document count by one
func (idx *Index) IncrementBuildProgress() {
	if idx.buildProgress != nil {
		idx.buildProgress.Increment()
	}
}

// CompleteBuild marks the index build as completed successfully
func (idx *Index) CompleteBuild() {
	if idx.buildProgress != nil {
		idx.buildProgress.Complete()
	}
}

// FailBuild marks the index build as failed
func (idx *Index) FailBuild(err string) {
	if idx.buildProgress != nil {
		idx.buildProgress.Fail(err)
	}
}
