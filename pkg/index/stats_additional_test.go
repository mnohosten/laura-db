package index

import (
	"testing"
)

func TestIndexStats_EstimateRangeSelectivityEdgeCases(t *testing.T) {
	stats := NewIndexStats()

	// Set up statistics with min/max values
	stats.SetStats(100, 50, int64(0), int64(100))

	// Test range entirely below min - uses heuristic
	selectivity := stats.EstimateRangeSelectivity(int64(-50), int64(-10))
	if selectivity < 0.0 || selectivity > 1.0 {
		t.Errorf("Expected selectivity in [0,1], got %f", selectivity)
	}

	// Test range entirely above max - uses heuristic
	selectivity = stats.EstimateRangeSelectivity(int64(150), int64(200))
	if selectivity < 0.0 || selectivity > 1.0 {
		t.Errorf("Expected selectivity in [0,1], got %f", selectivity)
	}

	// Test range with end before start (invalid range)
	selectivity = stats.EstimateRangeSelectivity(int64(80), int64(20))
	if selectivity != 0.0 {
		t.Errorf("Expected selectivity 0.0 for invalid range, got %f", selectivity)
	}

	// Test with nil min/max (should handle gracefully)
	stats2 := NewIndexStats()
	stats2.SetStats(100, 50, nil, nil)
	selectivity = stats2.EstimateRangeSelectivity(int64(10), int64(50))
	// Should use default heuristic
	if selectivity < 0.0 || selectivity > 1.0 {
		t.Errorf("Expected selectivity in [0,1], got %f", selectivity)
	}

	// Test range covering entire dataset
	selectivity = stats.EstimateRangeSelectivity(int64(0), int64(100))
	if selectivity != 1.0 {
		t.Errorf("Expected selectivity 1.0 for range covering all data, got %f", selectivity)
	}

	// Test partial range
	selectivity = stats.EstimateRangeSelectivity(int64(25), int64(75))
	if selectivity <= 0.0 || selectivity >= 1.0 {
		t.Errorf("Expected selectivity between 0 and 1 for partial range, got %f", selectivity)
	}
}

func TestIndexStats_SelectivityMethod(t *testing.T) {
	stats := NewIndexStats()

	// High selectivity (many unique keys)
	stats.SetStats(1000, 900, int64(0), int64(1000))
	selectivity := stats.Selectivity()
	if selectivity < 0.8 || selectivity > 1.0 {
		t.Errorf("Expected high selectivity (~0.9), got %f", selectivity)
	}

	// Low selectivity (few unique keys)
	stats.SetStats(1000, 10, int64(0), int64(100))
	selectivity = stats.Selectivity()
	if selectivity < 0.0 || selectivity > 0.2 {
		t.Errorf("Expected low selectivity (~0.01), got %f", selectivity)
	}

	// Edge case: zero entries
	stats.SetStats(0, 0, int64(0), int64(0))
	selectivity = stats.Selectivity()
	if selectivity != 1.0 {
		t.Errorf("Expected selectivity 1.0 for empty index, got %f", selectivity)
	}
}
