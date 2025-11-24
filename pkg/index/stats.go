package index

import (
	"sync"
	"time"
)

// HistogramBucket represents a single bucket in a histogram
type HistogramBucket struct {
	LowerBound interface{} // Lower bound of the bucket (inclusive)
	UpperBound interface{} // Upper bound of the bucket (exclusive)
	Count      int         // Number of values in this bucket
	Frequency  float64     // Normalized frequency (count / total)
}

// Histogram tracks the distribution of values for better range estimation
type Histogram struct {
	Buckets    []*HistogramBucket
	NumBuckets int
}

// IndexStats holds statistics about an index
type IndexStats struct {
	mu sync.RWMutex

	// Basic counts
	TotalEntries int // Total number of entries in the index
	UniqueKeys   int // Number of unique keys (cardinality)

	// Value distribution
	MinValue interface{} // Minimum value in the index
	MaxValue interface{} // Maximum value in the index

	// Histogram for better range query estimation
	Histogram *Histogram

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

// SetHistogram updates the histogram
func (s *IndexStats) SetHistogram(histogram *Histogram) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Histogram = histogram
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
// Uses histogram-based estimation if available, otherwise falls back to simple heuristic
func (s *IndexStats) EstimateRangeSelectivity(start, end interface{}) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.MinValue == nil || s.MaxValue == nil || s.TotalEntries == 0 {
		return 0.5 // Default guess if no statistics
	}

	// Use histogram-based estimation if available
	if s.Histogram != nil && len(s.Histogram.Buckets) > 0 {
		return s.estimateWithHistogram(start, end)
	}

	// Fallback to simple range estimation based on min/max
	return s.estimateWithMinMax(start, end)
}

// estimateWithHistogram uses histogram buckets for accurate range estimation
func (s *IndexStats) estimateWithHistogram(start, end interface{}) float64 {
	// Convert start and end to comparable values (int64 or float64)
	startNum, startOk := toComparable(start)
	endNum, endOk := toComparable(end)

	if !startOk || !endOk {
		return 0.3 // Default for non-numeric types
	}

	// Sum up frequencies for buckets that overlap with the range
	totalFrequency := 0.0

	for _, bucket := range s.Histogram.Buckets {
		bucketStart, bucketStartOk := toComparable(bucket.LowerBound)
		bucketEnd, bucketEndOk := toComparable(bucket.UpperBound)

		if !bucketStartOk || !bucketEndOk {
			continue
		}

		// Check if bucket overlaps with query range
		if bucketEnd <= startNum || bucketStart >= endNum {
			// No overlap
			continue
		}

		// Calculate overlap percentage
		overlap := 1.0
		if bucketStart < startNum {
			// Partial overlap at start
			bucketWidth := bucketEnd - bucketStart
			if bucketWidth > 0 {
				overlap = (bucketEnd - startNum) / bucketWidth
			}
		}
		if bucketEnd > endNum {
			// Partial overlap at end
			bucketWidth := bucketEnd - bucketStart
			if bucketWidth > 0 {
				overlap *= (endNum - bucketStart) / bucketWidth
			}
		}

		// Add weighted frequency
		totalFrequency += bucket.Frequency * overlap
	}

	// Ensure selectivity is in [0, 1] range
	if totalFrequency > 1.0 {
		totalFrequency = 1.0
	}
	if totalFrequency < 0.0 {
		totalFrequency = 0.0
	}

	return totalFrequency
}

// estimateWithMinMax provides simple range estimation using min/max values
func (s *IndexStats) estimateWithMinMax(start, end interface{}) float64 {
	minNum, minOk := toComparable(s.MinValue)
	maxNum, maxOk := toComparable(s.MaxValue)
	startNum, startOk := toComparable(start)
	endNum, endOk := toComparable(end)

	if !minOk || !maxOk || !startOk || !endOk {
		return 0.3 // Default for non-numeric types
	}

	// Calculate range width
	totalRange := maxNum - minNum
	if totalRange == 0 {
		return 1.0 // All values are the same
	}

	queryRange := endNum - startNum
	if queryRange < 0 {
		return 0.0 // Invalid range
	}

	// Estimate selectivity as ratio of query range to total range
	selectivity := queryRange / totalRange

	// Clamp to [0, 1]
	if selectivity > 1.0 {
		selectivity = 1.0
	}
	if selectivity < 0.0 {
		selectivity = 0.0
	}

	return selectivity
}

// toComparable converts a value to a comparable numeric type (float64)
func toComparable(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float64:
		return val, true
	case int32:
		return float64(val), true
	default:
		return 0, false
	}
}

// BuildHistogram creates a histogram from a set of values
// numBuckets specifies how many buckets to create (default: 10)
func BuildHistogram(values []interface{}, numBuckets int) *Histogram {
	if len(values) == 0 || numBuckets <= 0 {
		return nil
	}

	// Default to 10 buckets
	if numBuckets == 0 {
		numBuckets = 10
	}

	// Convert values to comparable numbers
	nums := make([]float64, 0, len(values))
	for _, v := range values {
		if num, ok := toComparable(v); ok {
			nums = append(nums, num)
		}
	}

	if len(nums) == 0 {
		return nil
	}

	// Find min and max
	minVal := nums[0]
	maxVal := nums[0]
	for _, n := range nums {
		if n < minVal {
			minVal = n
		}
		if n > maxVal {
			maxVal = n
		}
	}

	// Create buckets
	buckets := make([]*HistogramBucket, numBuckets)
	bucketWidth := (maxVal - minVal) / float64(numBuckets)

	// Handle edge case where all values are the same
	if bucketWidth == 0 {
		buckets[0] = &HistogramBucket{
			LowerBound: minVal,
			UpperBound: minVal + 1,
			Count:      len(nums),
			Frequency:  1.0,
		}
		return &Histogram{
			Buckets:    buckets[:1],
			NumBuckets: 1,
		}
	}

	// Initialize buckets
	for i := 0; i < numBuckets; i++ {
		lowerBound := minVal + float64(i)*bucketWidth
		upperBound := minVal + float64(i+1)*bucketWidth

		// Make sure last bucket includes maxVal
		if i == numBuckets-1 {
			upperBound = maxVal + 0.0001 // Slightly larger to include max
		}

		buckets[i] = &HistogramBucket{
			LowerBound: lowerBound,
			UpperBound: upperBound,
			Count:      0,
			Frequency:  0.0,
		}
	}

	// Count values in each bucket
	for _, num := range nums {
		bucketIdx := int((num - minVal) / bucketWidth)
		// Clamp to valid range
		if bucketIdx < 0 {
			bucketIdx = 0
		}
		if bucketIdx >= numBuckets {
			bucketIdx = numBuckets - 1
		}
		buckets[bucketIdx].Count++
	}

	// Calculate frequencies
	totalCount := float64(len(nums))
	for _, bucket := range buckets {
		bucket.Frequency = float64(bucket.Count) / totalCount
	}

	return &Histogram{
		Buckets:    buckets,
		NumBuckets: numBuckets,
	}
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
