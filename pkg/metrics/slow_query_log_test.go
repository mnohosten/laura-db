package metrics

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestSlowQueryLog_LogQuery(t *testing.T) {
	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:  50 * time.Millisecond,
		MaxEntries: 100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	// Log a slow query (above threshold)
	sql.LogQuery(SlowQueryEntry{
		Duration:   100 * time.Millisecond,
		Operation:  "query",
		Collection: "users",
		Filter:     map[string]interface{}{"age": int64(25)},
	})

	// Log a fast query (below threshold)
	sql.LogQuery(SlowQueryEntry{
		Duration:   10 * time.Millisecond,
		Operation:  "query",
		Collection: "users",
		Filter:     map[string]interface{}{"name": "John"},
	})

	entries := sql.GetEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 slow query entry, got %d", len(entries))
	}

	if entries[0].Operation != "query" {
		t.Errorf("Expected operation 'query', got '%s'", entries[0].Operation)
	}
	if entries[0].Collection != "users" {
		t.Errorf("Expected collection 'users', got '%s'", entries[0].Collection)
	}
}

func TestSlowQueryLog_MaxEntries(t *testing.T) {
	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:  10 * time.Millisecond,
		MaxEntries: 5, // Small buffer
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	// Log 10 slow queries
	for i := 0; i < 10; i++ {
		sql.LogQuery(SlowQueryEntry{
			Duration:   20 * time.Millisecond,
			Operation:  "insert",
			Collection: "test",
		})
	}

	entries := sql.GetEntries()
	if len(entries) != 5 {
		t.Errorf("Expected 5 entries (max), got %d", len(entries))
	}
}

func TestSlowQueryLog_GetRecentEntries(t *testing.T) {
	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:  10 * time.Millisecond,
		MaxEntries: 100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	// Log 10 entries
	for i := 0; i < 10; i++ {
		sql.LogQuery(SlowQueryEntry{
			Duration:   20 * time.Millisecond,
			Operation:  "query",
			Collection: "test",
		})
	}

	recent := sql.GetRecentEntries(3)
	if len(recent) != 3 {
		t.Errorf("Expected 3 recent entries, got %d", len(recent))
	}
}

func TestSlowQueryLog_GetEntriesByCollection(t *testing.T) {
	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:  10 * time.Millisecond,
		MaxEntries: 100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	sql.LogQuery(SlowQueryEntry{
		Duration:   50 * time.Millisecond,
		Operation:  "query",
		Collection: "users",
	})

	sql.LogQuery(SlowQueryEntry{
		Duration:   60 * time.Millisecond,
		Operation:  "query",
		Collection: "products",
	})

	sql.LogQuery(SlowQueryEntry{
		Duration:   70 * time.Millisecond,
		Operation:  "insert",
		Collection: "users",
	})

	userEntries := sql.GetEntriesByCollection("users")
	if len(userEntries) != 2 {
		t.Errorf("Expected 2 entries for 'users', got %d", len(userEntries))
	}

	productEntries := sql.GetEntriesByCollection("products")
	if len(productEntries) != 1 {
		t.Errorf("Expected 1 entry for 'products', got %d", len(productEntries))
	}
}

func TestSlowQueryLog_GetEntriesByOperation(t *testing.T) {
	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:  10 * time.Millisecond,
		MaxEntries: 100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	sql.LogQuery(SlowQueryEntry{
		Duration:  50 * time.Millisecond,
		Operation: "query",
	})

	sql.LogQuery(SlowQueryEntry{
		Duration:  60 * time.Millisecond,
		Operation: "insert",
	})

	sql.LogQuery(SlowQueryEntry{
		Duration:  70 * time.Millisecond,
		Operation: "query",
	})

	queryEntries := sql.GetEntriesByOperation("query")
	if len(queryEntries) != 2 {
		t.Errorf("Expected 2 query entries, got %d", len(queryEntries))
	}

	insertEntries := sql.GetEntriesByOperation("insert")
	if len(insertEntries) != 1 {
		t.Errorf("Expected 1 insert entry, got %d", len(insertEntries))
	}
}

func TestSlowQueryLog_GetEntriesSince(t *testing.T) {
	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:  10 * time.Millisecond,
		MaxEntries: 100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	now := time.Now()

	// Log entry in the past
	sql.mu.Lock()
	sql.entries = append(sql.entries, SlowQueryEntry{
		Timestamp: now.Add(-10 * time.Minute),
		Duration:  50 * time.Millisecond,
		Operation: "query",
	})
	sql.mu.Unlock()

	// Log current entry
	sql.LogQuery(SlowQueryEntry{
		Duration:  60 * time.Millisecond,
		Operation: "insert",
	})

	// Get entries since 5 minutes ago
	recent := sql.GetEntriesSince(now.Add(-5 * time.Minute))
	if len(recent) != 1 {
		t.Errorf("Expected 1 recent entry, got %d", len(recent))
	}
}

func TestSlowQueryLog_GetStatistics(t *testing.T) {
	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:  10 * time.Millisecond,
		MaxEntries: 100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	sql.LogQuery(SlowQueryEntry{
		Duration:   50 * time.Millisecond,
		Operation:  "query",
		Collection: "users",
	})

	sql.LogQuery(SlowQueryEntry{
		Duration:   100 * time.Millisecond,
		Operation:  "insert",
		Collection: "products",
	})

	sql.LogQuery(SlowQueryEntry{
		Duration:   75 * time.Millisecond,
		Operation:  "query",
		Collection: "users",
	})

	stats := sql.GetStatistics()

	if stats["total_entries"].(int) != 3 {
		t.Errorf("Expected 3 total entries, got %v", stats["total_entries"])
	}

	avgDuration := stats["avg_duration_ms"].(float64)
	if avgDuration < 74.0 || avgDuration > 76.0 {
		t.Errorf("Expected avg duration ~75ms, got %.2fms", avgDuration)
	}

	byOp := stats["by_operation"].(map[string]int)
	if byOp["query"] != 2 {
		t.Errorf("Expected 2 queries, got %d", byOp["query"])
	}
	if byOp["insert"] != 1 {
		t.Errorf("Expected 1 insert, got %d", byOp["insert"])
	}

	byColl := stats["by_collection"].(map[string]int)
	if byColl["users"] != 2 {
		t.Errorf("Expected 2 entries for 'users', got %d", byColl["users"])
	}
}

func TestSlowQueryLog_Clear(t *testing.T) {
	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:  10 * time.Millisecond,
		MaxEntries: 100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	sql.LogQuery(SlowQueryEntry{
		Duration:  50 * time.Millisecond,
		Operation: "query",
	})

	if len(sql.GetEntries()) != 1 {
		t.Error("Expected 1 entry before clear")
	}

	sql.Clear()

	if len(sql.GetEntries()) != 0 {
		t.Error("Expected 0 entries after clear")
	}
}

func TestSlowQueryLog_ThresholdUpdate(t *testing.T) {
	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:  50 * time.Millisecond,
		MaxEntries: 100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	if sql.GetThreshold() != 50*time.Millisecond {
		t.Error("Expected initial threshold of 50ms")
	}

	sql.SetThreshold(100 * time.Millisecond)

	if sql.GetThreshold() != 100*time.Millisecond {
		t.Error("Expected updated threshold of 100ms")
	}
}

func TestSlowQueryLog_EnableDisable(t *testing.T) {
	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:  10 * time.Millisecond,
		MaxEntries: 100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	if !sql.IsEnabled() {
		t.Error("Expected log to be enabled")
	}

	sql.Disable()

	if sql.IsEnabled() {
		t.Error("Expected log to be disabled")
	}

	// Log should not record when disabled
	sql.LogQuery(SlowQueryEntry{
		Duration:  50 * time.Millisecond,
		Operation: "query",
	})

	if len(sql.GetEntries()) != 0 {
		t.Error("Expected no entries when disabled")
	}

	sql.Enable()

	if !sql.IsEnabled() {
		t.Error("Expected log to be enabled")
	}

	// Should record when enabled
	sql.LogQuery(SlowQueryEntry{
		Duration:  50 * time.Millisecond,
		Operation: "query",
	})

	if len(sql.GetEntries()) != 1 {
		t.Error("Expected 1 entry when enabled")
	}
}

func TestSlowQueryLog_ExportToJSON(t *testing.T) {
	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:  10 * time.Millisecond,
		MaxEntries: 100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	sql.LogQuery(SlowQueryEntry{
		Duration:   50 * time.Millisecond,
		Operation:  "query",
		Collection: "users",
	})

	var buf bytes.Buffer
	err = sql.ExportToJSON(&buf)
	if err != nil {
		t.Fatalf("Failed to export to JSON: %v", err)
	}

	// Verify JSON is valid
	var entries []SlowQueryEntry
	err = json.Unmarshal(buf.Bytes(), &entries)
	if err != nil {
		t.Fatalf("Failed to parse exported JSON: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry in JSON, got %d", len(entries))
	}
}

func TestSlowQueryLog_FileLogging(t *testing.T) {
	tmpFile := "/tmp/slow_query_test.log"
	defer os.Remove(tmpFile)

	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:   10 * time.Millisecond,
		MaxEntries:  100,
		LogFilePath: tmpFile,
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}
	defer sql.Close()

	sql.LogQuery(SlowQueryEntry{
		Duration:   50 * time.Millisecond,
		Operation:  "query",
		Collection: "users",
	})

	// Close to flush
	sql.Close()

	// Verify file exists and has content
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected log file to have content")
	}

	// Verify it's valid JSON
	var entry SlowQueryEntry
	err = json.Unmarshal(data, &entry)
	if err != nil {
		t.Fatalf("Failed to parse log file JSON: %v", err)
	}
}

func TestSlowQueryLog_GetTopSlowest(t *testing.T) {
	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:  10 * time.Millisecond,
		MaxEntries: 100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	durations := []time.Duration{
		50 * time.Millisecond,
		200 * time.Millisecond,
		30 * time.Millisecond,
		100 * time.Millisecond,
		150 * time.Millisecond,
	}

	for _, d := range durations {
		sql.LogQuery(SlowQueryEntry{
			Duration:  d,
			Operation: "query",
		})
	}

	top3 := sql.GetTopSlowest(3)
	if len(top3) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(top3))
	}

	// Verify they're sorted by duration (descending)
	if top3[0].Duration != 200*time.Millisecond {
		t.Errorf("Expected slowest to be 200ms, got %v", top3[0].Duration)
	}
	if top3[1].Duration != 150*time.Millisecond {
		t.Errorf("Expected second slowest to be 150ms, got %v", top3[1].Duration)
	}
	if top3[2].Duration != 100*time.Millisecond {
		t.Errorf("Expected third slowest to be 100ms, got %v", top3[2].Duration)
	}
}

func TestSlowQueryLog_GetSlowestByCollection(t *testing.T) {
	sql, err := NewSlowQueryLog(&SlowQueryLogConfig{
		Threshold:  10 * time.Millisecond,
		MaxEntries: 100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	sql.LogQuery(SlowQueryEntry{
		Duration:   50 * time.Millisecond,
		Operation:  "query",
		Collection: "users",
	})

	sql.LogQuery(SlowQueryEntry{
		Duration:   100 * time.Millisecond,
		Operation:  "query",
		Collection: "users",
	})

	sql.LogQuery(SlowQueryEntry{
		Duration:   75 * time.Millisecond,
		Operation:  "query",
		Collection: "products",
	})

	slowest := sql.GetSlowestByCollection()

	if len(slowest) != 2 {
		t.Errorf("Expected 2 collections, got %d", len(slowest))
	}

	if slowest["users"].Duration != 100*time.Millisecond {
		t.Errorf("Expected slowest users query to be 100ms, got %v", slowest["users"].Duration)
	}

	if slowest["products"].Duration != 75*time.Millisecond {
		t.Errorf("Expected slowest products query to be 75ms, got %v", slowest["products"].Duration)
	}
}

func TestSlowQueryLog_DefaultConfig(t *testing.T) {
	config := DefaultSlowQueryLogConfig()

	if config.Threshold != 100*time.Millisecond {
		t.Errorf("Expected default threshold 100ms, got %v", config.Threshold)
	}
	if config.MaxEntries != 1000 {
		t.Errorf("Expected default max entries 1000, got %d", config.MaxEntries)
	}
	if !config.Enabled {
		t.Error("Expected default enabled to be true")
	}
	if !config.IncludeProfile {
		t.Error("Expected default include profile to be true")
	}
}

func TestSlowQueryLog_EmptyStatistics(t *testing.T) {
	sql, err := NewSlowQueryLog(DefaultSlowQueryLogConfig())
	if err != nil {
		t.Fatalf("Failed to create slow query log: %v", err)
	}

	stats := sql.GetStatistics()

	if stats["total_entries"].(int) != 0 {
		t.Errorf("Expected 0 entries, got %v", stats["total_entries"])
	}
}
