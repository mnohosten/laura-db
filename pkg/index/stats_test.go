package index

import (
	"testing"
	"time"
)

func TestIndexStats(t *testing.T) {
	stats := NewIndexStats()

	// Initial state
	if !stats.IsStale {
		t.Error("Expected new stats to be stale")
	}

	// Set statistics
	stats.SetStats(1000, 50, int64(1), int64(100))

	total, unique, min, max, stale := stats.GetStats()
	if total != 1000 {
		t.Errorf("Expected 1000 total entries, got %d", total)
	}
	if unique != 50 {
		t.Errorf("Expected 50 unique keys, got %d", unique)
	}
	if min.(int64) != 1 {
		t.Errorf("Expected min=1, got %v", min)
	}
	if max.(int64) != 100 {
		t.Errorf("Expected max=100, got %v", max)
	}
	if stale {
		t.Error("Expected stats to not be stale after setting")
	}
}

func TestSelectivity(t *testing.T) {
	stats := NewIndexStats()

	tests := []struct {
		name              string
		totalEntries      int
		uniqueKeys        int
		expectedSelectivity float64
	}{
		{
			name:              "All unique",
			totalEntries:      100,
			uniqueKeys:        100,
			expectedSelectivity: 1.0,
		},
		{
			name:              "Half unique",
			totalEntries:      100,
			uniqueKeys:        50,
			expectedSelectivity: 0.5,
		},
		{
			name:              "Low selectivity",
			totalEntries:      1000,
			uniqueKeys:        10,
			expectedSelectivity: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats.SetStats(tt.totalEntries, tt.uniqueKeys, nil, nil)
			selectivity := stats.Selectivity()

			if selectivity != tt.expectedSelectivity {
				t.Errorf("Expected selectivity %.2f, got %.2f",
					tt.expectedSelectivity, selectivity)
			}
		})
	}
}

func TestStatsUpdate(t *testing.T) {
	stats := NewIndexStats()
	stats.SetStats(100, 50, int64(1), int64(100))

	// Verify not stale
	if stats.IsStale {
		t.Error("Stats should not be stale after setting")
	}

	// Mark as stale
	stats.Update()

	// Verify stale
	if !stats.IsStale {
		t.Error("Stats should be stale after update")
	}
}

func TestIndexAnalyze(t *testing.T) {
	idx := NewIndex(&IndexConfig{
		Name:      "test_idx",
		FieldPath: "age",
		Type:      IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Insert some data (unique keys since B-tree replaces duplicates)
	idx.Insert(int64(25), "id1")
	idx.Insert(int64(30), "id2")
	idx.Insert(int64(35), "id3")

	// Analyze the index
	idx.Analyze()

	stats := idx.GetStatistics()
	total, unique, min, max, stale := stats.GetStats()

	// Verify statistics
	if total != 3 {
		t.Errorf("Expected 3 total entries, got %d", total)
	}
	if unique != 3 {
		t.Errorf("Expected 3 unique keys (25, 30, 35), got %d", unique)
	}
	if min.(int64) != 25 {
		t.Errorf("Expected min=25, got %v", min)
	}
	if max.(int64) != 35 {
		t.Errorf("Expected max=35, got %v", max)
	}
	if stale {
		t.Error("Stats should not be stale after analyze")
	}

	// Verify selectivity (all unique = 1.0)
	selectivity := stats.Selectivity()
	expectedSelectivity := 1.0
	if selectivity != expectedSelectivity {
		t.Errorf("Expected selectivity %.2f, got %.2f",
			expectedSelectivity, selectivity)
	}
}

func TestIndexStatsStaleOnInsert(t *testing.T) {
	idx := NewIndex(&IndexConfig{
		Name:      "test_idx",
		FieldPath: "field",
		Type:      IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Insert and analyze
	idx.Insert(int64(1), "id1")
	idx.Analyze()

	stats := idx.GetStatistics()
	if stats.IsStale {
		t.Error("Stats should not be stale after analyze")
	}

	// Insert more data
	idx.Insert(int64(2), "id2")

	// Stats should now be stale
	if !stats.IsStale {
		t.Error("Stats should be stale after insert")
	}
}

func TestIndexStatsStaleOnDelete(t *testing.T) {
	idx := NewIndex(&IndexConfig{
		Name:      "test_idx",
		FieldPath: "field",
		Type:      IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Insert and analyze
	idx.Insert(int64(1), "id1")
	idx.Insert(int64(2), "id2")
	idx.Analyze()

	stats := idx.GetStatistics()
	if stats.IsStale {
		t.Error("Stats should not be stale after analyze")
	}

	// Delete data
	idx.Delete(int64(1))

	// Stats should now be stale
	if !stats.IsStale {
		t.Error("Stats should be stale after delete")
	}
}

func TestIndexStatsToMap(t *testing.T) {
	stats := NewIndexStats()
	stats.SetStats(100, 75, int64(1), int64(100))

	statsMap := stats.ToMap()

	// Verify all expected fields are present
	expectedFields := []string{
		"total_entries", "unique_keys", "cardinality",
		"selectivity", "min_value", "max_value",
		"last_updated", "is_stale",
	}

	for _, field := range expectedFields {
		if _, exists := statsMap[field]; !exists {
			t.Errorf("Expected field %s in stats map", field)
		}
	}

	// Verify values
	if statsMap["total_entries"].(int) != 100 {
		t.Errorf("Expected total_entries=100, got %v", statsMap["total_entries"])
	}
	if statsMap["unique_keys"].(int) != 75 {
		t.Errorf("Expected unique_keys=75, got %v", statsMap["unique_keys"])
	}
	if statsMap["cardinality"].(int) != 75 {
		t.Errorf("Expected cardinality=75, got %v", statsMap["cardinality"])
	}
	if statsMap["selectivity"].(float64) != 0.75 {
		t.Errorf("Expected selectivity=0.75, got %v", statsMap["selectivity"])
	}
}

func TestIndexStatsInStats(t *testing.T) {
	idx := NewIndex(&IndexConfig{
		Name:      "test_idx",
		FieldPath: "age",
		Type:      IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Insert data (unique keys)
	for i := 0; i < 10; i++ {
		idx.Insert(int64(i), string(rune(i)))
	}

	// Analyze
	idx.Analyze()

	// Get stats map
	stats := idx.Stats()

	// Verify index stats are included
	if _, exists := stats["total_entries"]; !exists {
		t.Error("Expected total_entries in Stats() output")
	}
	if _, exists := stats["cardinality"]; !exists {
		t.Error("Expected cardinality in Stats() output")
	}
	if _, exists := stats["selectivity"]; !exists {
		t.Error("Expected selectivity in Stats() output")
	}

	// Verify values
	if stats["total_entries"].(int) != 10 {
		t.Errorf("Expected 10 entries, got %v", stats["total_entries"])
	}
	if stats["cardinality"].(int) != 10 {
		t.Errorf("Expected cardinality=10, got %v", stats["cardinality"])
	}
}

func TestStatsLastUpdated(t *testing.T) {
	stats := NewIndexStats()

	before := time.Now()
	time.Sleep(10 * time.Millisecond)
	stats.SetStats(100, 50, nil, nil)
	time.Sleep(10 * time.Millisecond)
	after := time.Now()

	if stats.LastUpdated.Before(before) {
		t.Error("LastUpdated should be after 'before' timestamp")
	}
	if stats.LastUpdated.After(after) {
		t.Error("LastUpdated should be before 'after' timestamp")
	}
}

func TestCompoundIndexStats(t *testing.T) {
	idx := NewIndex(&IndexConfig{
		Name:       "city_age_1",
		FieldPaths: []string{"city", "age"},
		Type:       IndexTypeBTree,
		Unique:     false,
		Order:      32,
	})

	// Insert compound index data
	idx.Insert(NewCompositeKey("NYC", int64(25)), "id1")
	idx.Insert(NewCompositeKey("NYC", int64(30)), "id2")
	idx.Insert(NewCompositeKey("Boston", int64(25)), "id3")
	idx.Insert(NewCompositeKey("Boston", int64(30)), "id4")
	idx.Insert(NewCompositeKey("Seattle", int64(28)), "id5")

	// Analyze the index
	idx.Analyze()

	stats := idx.GetStatistics()
	total, unique, min, max, stale := stats.GetStats()

	// Verify statistics
	if total != 5 {
		t.Errorf("Expected 5 total entries, got %d", total)
	}
	if unique != 5 {
		t.Errorf("Expected 5 unique composite keys, got %d", unique)
	}
	if stale {
		t.Error("Stats should not be stale after analyze")
	}

	// Verify min/max are CompositeKeys
	if _, ok := min.(*CompositeKey); !ok {
		t.Error("Expected min to be CompositeKey")
	}
	if _, ok := max.(*CompositeKey); !ok {
		t.Error("Expected max to be CompositeKey")
	}

	// Verify selectivity (all unique = 1.0)
	selectivity := stats.Selectivity()
	if selectivity != 1.0 {
		t.Errorf("Expected selectivity 1.0, got %.2f", selectivity)
	}

	// Test Stats() map output
	statsMap := idx.Stats()
	if !statsMap["is_compound"].(bool) {
		t.Error("Expected is_compound to be true")
	}

	fieldPaths := statsMap["field_paths"].([]string)
	if len(fieldPaths) != 2 || fieldPaths[0] != "city" || fieldPaths[1] != "age" {
		t.Errorf("Expected field_paths [city, age], got %v", fieldPaths)
	}
}

// Histogram tests

func TestBuildHistogram(t *testing.T) {
	// Test with integer values
	values := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		values[i] = int64(i)
	}

	histogram := BuildHistogram(values, 10)
	if histogram == nil {
		t.Fatal("Expected histogram to be created")
	}

	if histogram.NumBuckets != 10 {
		t.Errorf("Expected 10 buckets, got %d", histogram.NumBuckets)
	}

	// Verify total frequency sums to 1.0
	totalFreq := 0.0
	for _, bucket := range histogram.Buckets {
		totalFreq += bucket.Frequency
	}

	if totalFreq < 0.99 || totalFreq > 1.01 {
		t.Errorf("Expected total frequency ~1.0, got %f", totalFreq)
	}
}

func TestBuildHistogramUniformDistribution(t *testing.T) {
	// Test with uniform distribution
	values := make([]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		values[i] = int64(i)
	}

	histogram := BuildHistogram(values, 10)

	// With uniform distribution, each bucket should have ~0.1 frequency
	for i, bucket := range histogram.Buckets {
		// Allow some tolerance for rounding
		if bucket.Frequency < 0.09 || bucket.Frequency > 0.11 {
			t.Errorf("Bucket %d: expected frequency ~0.1, got %f", i, bucket.Frequency)
		}
	}
}

func TestBuildHistogramSkewedDistribution(t *testing.T) {
	// Test with skewed distribution (most values near 0)
	values := make([]interface{}, 1000)
	for i := 0; i < 900; i++ {
		values[i] = int64(i % 10) // 0-9
	}
	for i := 900; i < 1000; i++ {
		values[i] = int64(50 + (i % 50)) // 50-99
	}

	histogram := BuildHistogram(values, 10)

	// First bucket should have much higher frequency
	if histogram.Buckets[0].Frequency < 0.5 {
		t.Errorf("First bucket should have high frequency for skewed distribution, got %f",
			histogram.Buckets[0].Frequency)
	}
}

func TestEstimateRangeSelectivityWithHistogram(t *testing.T) {
	stats := NewIndexStats()

	// Create data: values from 0 to 99
	values := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		values[i] = int64(i)
	}

	// Build histogram
	histogram := BuildHistogram(values, 10)
	stats.SetHistogram(histogram)
	stats.SetStats(100, 100, int64(0), int64(99))

	// Test range [0, 10) - should be ~10% selective
	selectivity := stats.EstimateRangeSelectivity(int64(0), int64(10))
	if selectivity < 0.08 || selectivity > 0.12 {
		t.Errorf("Expected selectivity ~0.1 for range [0, 10), got %f", selectivity)
	}

	// Test range [0, 50) - should be ~50% selective
	selectivity = stats.EstimateRangeSelectivity(int64(0), int64(50))
	if selectivity < 0.48 || selectivity > 0.52 {
		t.Errorf("Expected selectivity ~0.5 for range [0, 50), got %f", selectivity)
	}

	// Test range [0, 100) - should be ~100% selective
	selectivity = stats.EstimateRangeSelectivity(int64(0), int64(100))
	if selectivity < 0.98 || selectivity > 1.0 {
		t.Errorf("Expected selectivity ~1.0 for range [0, 100), got %f", selectivity)
	}
}

func TestEstimateRangeSelectivityWithoutHistogram(t *testing.T) {
	stats := NewIndexStats()
	stats.SetStats(100, 100, int64(0), int64(99))

	// Without histogram, should use min/max estimation
	// Range [0, 50) is 50/99 ~= 0.505
	selectivity := stats.EstimateRangeSelectivity(int64(0), int64(50))
	if selectivity < 0.48 || selectivity > 0.52 {
		t.Errorf("Expected selectivity ~0.5 for range [0, 50), got %f", selectivity)
	}
}

func TestEstimateRangeSelectivityPartialBucketOverlap(t *testing.T) {
	stats := NewIndexStats()

	// Create data from 0 to 99
	values := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		values[i] = int64(i)
	}

	histogram := BuildHistogram(values, 10)
	stats.SetHistogram(histogram)
	stats.SetStats(100, 100, int64(0), int64(99))

	// Test range [5, 15) - overlaps two buckets partially
	selectivity := stats.EstimateRangeSelectivity(int64(5), int64(15))

	// Should be approximately 10/100 = 0.1
	if selectivity < 0.08 || selectivity > 0.12 {
		t.Errorf("Expected selectivity ~0.1 for range [5, 15), got %f", selectivity)
	}
}

func TestBuildHistogramSameValues(t *testing.T) {
	// All values are the same
	values := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		values[i] = int64(42)
	}

	histogram := BuildHistogram(values, 10)

	// Should create 1 bucket with all values
	if histogram.NumBuckets != 1 {
		t.Errorf("Expected 1 bucket for same values, got %d", histogram.NumBuckets)
	}

	if histogram.Buckets[0].Frequency != 1.0 {
		t.Errorf("Expected frequency 1.0 for single bucket, got %f",
			histogram.Buckets[0].Frequency)
	}
}

func TestBuildHistogramEmptyValues(t *testing.T) {
	histogram := BuildHistogram([]interface{}{}, 10)
	if histogram != nil {
		t.Error("Expected nil histogram for empty values")
	}
}

func TestBuildHistogramNonNumericValues(t *testing.T) {
	values := []interface{}{"a", "b", "c"}
	histogram := BuildHistogram(values, 10)
	if histogram != nil {
		t.Error("Expected nil histogram for non-numeric values")
	}
}

func TestEstimateRangeSelectivityInvalidRange(t *testing.T) {
	stats := NewIndexStats()
	stats.SetStats(100, 100, int64(0), int64(99))

	// Invalid range (end < start)
	selectivity := stats.EstimateRangeSelectivity(int64(50), int64(10))
	if selectivity != 0.0 {
		t.Errorf("Expected selectivity 0.0 for invalid range, got %f", selectivity)
	}
}

func TestToComparable(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected float64
		ok       bool
	}{
		{int(42), 42.0, true},
		{int64(42), 42.0, true},
		{float64(42.5), 42.5, true},
		{int32(42), 42.0, true},
		{"string", 0, false},
		{true, 0, false},
	}

	for _, test := range tests {
		result, ok := toComparable(test.input)
		if ok != test.ok {
			t.Errorf("toComparable(%v): expected ok=%v, got ok=%v",
				test.input, test.ok, ok)
		}
		if ok && result != test.expected {
			t.Errorf("toComparable(%v): expected %v, got %v",
				test.input, test.expected, result)
		}
	}
}

// Benchmark histogram-based estimation
func BenchmarkEstimateRangeSelectivityWithHistogram(b *testing.B) {
	stats := NewIndexStats()

	// Create uniformly distributed data
	values := make([]interface{}, 10000)
	for i := 0; i < 10000; i++ {
		values[i] = int64(i)
	}

	histogram := BuildHistogram(values, 100)
	stats.SetHistogram(histogram)
	stats.SetStats(10000, 10000, int64(0), int64(9999))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats.EstimateRangeSelectivity(int64(1000), int64(5000))
	}
}

// Benchmark min/max-based estimation
func BenchmarkEstimateRangeSelectivityWithoutHistogram(b *testing.B) {
	stats := NewIndexStats()
	stats.SetStats(10000, 10000, int64(0), int64(9999))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats.EstimateRangeSelectivity(int64(1000), int64(5000))
	}
}

// Benchmark histogram building
func BenchmarkBuildHistogram(b *testing.B) {
	values := make([]interface{}, 10000)
	for i := 0; i < 10000; i++ {
		values[i] = int64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildHistogram(values, 100)
	}
}
