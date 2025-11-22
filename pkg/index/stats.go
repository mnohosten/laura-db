package index

import (
	"sync"
	"time"
)

// IndexStats holds statistics about an index
type IndexStats struct {
	mu sync.RWMutex

	// Basic counts
	TotalEntries int // Total number of entries in the index
	UniqueKeys   int // Number of unique keys (cardinality)

	// Value distribution
	MinValue interface{} // Minimum value in the index
	MaxValue interface{} // Maximum value in the index

	// Metadata
	LastUpdated time.Time // When statistics were last updated
	IsStale     bool      // True if stats need to be recalculated
}

// NewIndexStats creates a new statistics tracker
func NewIndexStats() *IndexStats {
	return &IndexStats{
		TotalEntries: 0,
		UniqueKeys:   0,
		MinValue:     nil,
		MaxValue:     nil,
		LastUpdated:  time.Now(),
		IsStale:      true,
	}
}

// Update marks statistics as stale (needs recalculation)
func (s *IndexStats) Update() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsStale = true
}

// SetStats updates the statistics with new values
func (s *IndexStats) SetStats(totalEntries, uniqueKeys int, minVal, maxVal interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalEntries = totalEntries
	s.UniqueKeys = uniqueKeys
	s.MinValue = minVal
	s.MaxValue = maxVal
	s.LastUpdated = time.Now()
	s.IsStale = false
}

// GetStats returns a copy of the current statistics
func (s *IndexStats) GetStats() (totalEntries, uniqueKeys int, minVal, maxVal interface{}, isStale bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.TotalEntries, s.UniqueKeys, s.MinValue, s.MaxValue, s.IsStale
}

// Cardinality returns the number of unique keys
func (s *IndexStats) Cardinality() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.UniqueKeys
}

// Selectivity estimates how selective this index is (0.0 to 1.0)
// Lower values mean more selective (better for filtering)
func (s *IndexStats) Selectivity() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.TotalEntries == 0 {
		return 1.0 // No selectivity if empty
	}

	// Selectivity = unique_keys / total_entries
	// If every entry is unique, selectivity = 1.0
	// If all entries have same key, selectivity approaches 0
	return float64(s.UniqueKeys) / float64(s.TotalEntries)
}

// EstimateRangeSelectivity estimates selectivity for a range query
// This is a simplified estimation - could be improved with histograms
func (s *IndexStats) EstimateRangeSelectivity(start, end interface{}) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.MinValue == nil || s.MaxValue == nil || s.TotalEntries == 0 {
		return 0.5 // Default guess if no statistics
	}

	// For now, use a simple heuristic
	// TODO: Implement histogram-based estimation
	return 0.3 // Assume range queries are moderately selective
}

// ToMap converts statistics to a map for display
func (s *IndexStats) ToMap() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_entries": s.TotalEntries,
		"unique_keys":   s.UniqueKeys,
		"cardinality":   s.UniqueKeys,
		"selectivity":   s.Selectivity(),
		"min_value":     s.MinValue,
		"max_value":     s.MaxValue,
		"last_updated":  s.LastUpdated,
		"is_stale":      s.IsStale,
	}
}
