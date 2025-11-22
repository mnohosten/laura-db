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
