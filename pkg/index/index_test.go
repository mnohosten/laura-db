package index

import (
	"testing"
)

func createTestIndex(name string, fieldPaths []string, unique bool, filter map[string]interface{}) *Index {
	return NewIndex(&IndexConfig{
		Name:       name,
		FieldPaths: fieldPaths,
		Type:       IndexTypeBTree,
		Unique:     unique,
		Filter:     filter,
	})
}

func TestIndex_RangeScan(t *testing.T) {
	idx := createTestIndex("test_idx", []string{"age"}, false, nil)

	// Insert some values
	idx.Insert(int64(10), "doc1")
	idx.Insert(int64(20), "doc2")
	idx.Insert(int64(30), "doc3")
	idx.Insert(int64(40), "doc4")
	idx.Insert(int64(50), "doc5")

	// Range scan from 15 to 35
	keys, values := idx.RangeScan(int64(15), int64(35))

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys in range [15, 35], got %d", len(keys))
	}

	// Should contain 20 and 30
	expectedKeys := map[int64]bool{
		20: false,
		30: false,
	}

	for _, key := range keys {
		k := key.(int64)
		if _, ok := expectedKeys[k]; ok {
			expectedKeys[k] = true
		}
	}

	for k, found := range expectedKeys {
		if !found {
			t.Errorf("Expected to find key %d in range scan", k)
		}
	}

	// Verify corresponding values
	if len(values) != 2 {
		t.Errorf("Expected 2 values in range [15, 35], got %d", len(values))
	}
}

func TestIndex_Size(t *testing.T) {
	idx := createTestIndex("test_idx", []string{"score"}, false, nil)

	// Initially empty
	if idx.Size() != 0 {
		t.Errorf("Expected size 0 for empty index, got %d", idx.Size())
	}

	// Insert some entries
	idx.Insert(int64(100), "doc1")
	idx.Insert(int64(200), "doc2")
	idx.Insert(int64(300), "doc3")

	if idx.Size() != 3 {
		t.Errorf("Expected size 3 after 3 inserts, got %d", idx.Size())
	}

	// Delete one
	idx.Delete(int64(200))

	if idx.Size() != 2 {
		t.Errorf("Expected size 2 after delete, got %d", idx.Size())
	}
}

func TestIndex_FieldPath(t *testing.T) {
	// Single field
	idx1 := createTestIndex("idx1", []string{"name"}, false, nil)
	if idx1.FieldPath() != "name" {
		t.Errorf("Expected field path 'name', got '%s'", idx1.FieldPath())
	}

	// Compound index - should return first field
	idx2 := createTestIndex("idx2", []string{"city", "age"}, false, nil)
	if idx2.FieldPath() != "city" {
		t.Errorf("Expected field path 'city', got '%s'", idx2.FieldPath())
	}

	// Empty field paths (edge case) - will panic, so skip this test
}

func TestIndex_Filter(t *testing.T) {
	// Index without filter
	idx1 := createTestIndex("idx1", []string{"age"}, false, nil)
	if idx1.Filter() != nil {
		t.Error("Expected nil filter for non-partial index")
	}

	if idx1.IsPartial() {
		t.Error("Expected IsPartial to be false for non-partial index")
	}

	// Partial index with filter
	filter := map[string]interface{}{
		"status": "active",
	}
	idx2 := createTestIndex("idx2", []string{"email"}, false, filter)

	if !idx2.IsPartial() {
		t.Error("Expected IsPartial to be true for partial index")
	}

	returnedFilter := idx2.Filter()
	if returnedFilter == nil {
		t.Fatal("Expected filter to be returned")
	}

	if returnedFilter["status"] != "active" {
		t.Errorf("Expected filter status 'active', got %v", returnedFilter["status"])
	}
}

func TestIndex_BuildStateMethods(t *testing.T) {
	idx := createTestIndex("test_idx", []string{"field"}, false, nil)

	// Initially should be ready (no build in progress)
	if !idx.IsReady() {
		t.Error("Expected index to be ready initially")
	}

	if idx.IsBuilding() {
		t.Error("Expected index to not be building initially")
	}

	// Start build
	idx.StartBuild(100)

	if idx.IsReady() {
		t.Error("Expected index to not be ready when building")
	}

	if !idx.IsBuilding() {
		t.Error("Expected index to be building after StartBuild")
	}

	// Update progress
	idx.UpdateBuildProgress(50)

	progress := idx.GetBuildProgress()
	if progress["processed"] != 50 {
		t.Errorf("Expected processed 50, got %v", progress["processed"])
	}

	// Increment progress
	idx.IncrementBuildProgress()
	idx.IncrementBuildProgress()

	progress = idx.GetBuildProgress()
	if progress["processed"] != 52 {
		t.Errorf("Expected processed 52 after 2 increments, got %v", progress["processed"])
	}

	// Complete build
	idx.CompleteBuild()

	if !idx.IsReady() {
		t.Error("Expected index to be ready after CompleteBuild")
	}

	if idx.IsBuilding() {
		t.Error("Expected index to not be building after CompleteBuild")
	}

	state := idx.GetBuildState()
	if state != IndexStateReady {
		t.Errorf("Expected state Ready, got %v", state)
	}
}

func TestIndex_BuildStateFail(t *testing.T) {
	idx := createTestIndex("test_idx", []string{"field"}, false, nil)

	// Start build
	idx.StartBuild(100)
	idx.UpdateBuildProgress(30)

	// Fail the build
	errorMsg := "disk error"
	idx.FailBuild(errorMsg)

	if idx.IsReady() {
		t.Error("Expected index to not be ready after FailBuild")
	}

	if idx.IsBuilding() {
		t.Error("Expected index to not be building after FailBuild")
	}

	state := idx.GetBuildState()
	if state != IndexStateFailed {
		t.Errorf("Expected state Failed, got %v", state)
	}

	progress := idx.GetBuildProgress()
	if progress["error"] != errorMsg {
		t.Errorf("Expected error message '%s', got %v", errorMsg, progress["error"])
	}
}

func TestIndex_StatsWithBuildProgress(t *testing.T) {
	idx := createTestIndex("test_idx", []string{"field"}, false, nil)

	// Stats when ready (not building) - build_state should be "ready"
	stats1 := idx.Stats()
	if stats1["build_state"] != "ready" {
		t.Errorf("Expected build_state 'ready', got %v", stats1["build_state"])
	}

	// Start build
	idx.StartBuild(100)

	// Stats with build progress
	stats2 := idx.Stats()
	if stats2["build_state"] != "building" {
		t.Errorf("Expected build_state 'building', got %v", stats2["build_state"])
	}

	if _, ok := stats2["build_progress"]; !ok {
		t.Error("Expected build_progress in stats when building")
	}

	// Complete build
	idx.CompleteBuild()

	// Stats after completion
	stats3 := idx.Stats()
	if stats3["build_state"] != "ready" {
		t.Errorf("Expected build_state 'ready', got %v", stats3["build_state"])
	}

	// build_progress should not be present when ready
	if _, ok := stats3["build_progress"]; ok {
		t.Error("Expected no build_progress in stats when ready")
	}
}

func TestIndex_Cardinality(t *testing.T) {
	idx := createTestIndex("test_idx", []string{"category"}, false, nil)

	// Insert some values
	idx.Insert("electronics", "doc1")
	idx.Insert("books", "doc2")
	idx.Insert("electronics", "doc3") // Duplicate key
	idx.Insert("clothing", "doc4")

	// Analyze to update statistics
	idx.Analyze()

	stats := idx.Stats()
	cardinality, ok := stats["cardinality"].(int)
	if !ok {
		t.Fatal("Expected cardinality in stats")
	}

	// Should have 3 unique keys: electronics, books, clothing
	if cardinality != 3 {
		t.Errorf("Expected cardinality 3, got %d", cardinality)
	}
}

func TestIndex_StatsWithPartialFilter(t *testing.T) {
	filter := map[string]interface{}{
		"age": map[string]interface{}{
			"$gte": int64(18),
		},
	}

	idx := createTestIndex("partial_idx", []string{"email"}, false, filter)

	stats := idx.Stats()

	// Verify partial index flag
	if stats["is_partial"] != true {
		t.Error("Expected is_partial to be true")
	}

	// Verify filter is included
	filterInStats, ok := stats["filter"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected filter in stats for partial index")
	}

	if filterInStats["age"] == nil {
		t.Error("Expected age field in filter")
	}
}

func TestCompositeKey_String(t *testing.T) {
	ck := &CompositeKey{
		Values: []interface{}{"New York", int64(25)},
	}

	str := ck.String()

	// Should contain both values in string representation
	if str == "" {
		t.Error("Expected non-empty string representation")
	}

	// Check that it's formatted properly (implementation detail)
	// Just verify it doesn't panic and returns something
	if len(str) < 5 {
		t.Error("Expected meaningful string representation")
	}
}

func TestStats_CardinalityMethod(t *testing.T) {
	idx := createTestIndex("test_idx", []string{"status"}, false, nil)

	// Insert some values
	idx.Insert("active", "doc1")
	idx.Insert("inactive", "doc2")
	idx.Insert("active", "doc3")    // Duplicate
	idx.Insert("pending", "doc4")
	idx.Insert("active", "doc5")    // Another duplicate
	idx.Insert("completed", "doc6")

	// Need to call Analyze to calculate cardinality
	idx.Analyze()

	// Cardinality method from stats - should have 4 unique keys
	card := idx.GetStatistics().Cardinality()
	if card != 4 {
		t.Errorf("Expected cardinality 4 (active, inactive, pending, completed), got %d", card)
	}
}

func TestCompareValues_AllTypes(t *testing.T) {
	// Test string comparison
	if compareValues("apple", "banana") != -1 {
		t.Error("Expected 'apple' < 'banana'")
	}

	if compareValues("zebra", "apple") != 1 {
		t.Error("Expected 'zebra' > 'apple'")
	}

	if compareValues("test", "test") != 0 {
		t.Error("Expected 'test' == 'test'")
	}

	// Test numeric comparison
	if compareValues(int64(10), int64(20)) != -1 {
		t.Error("Expected 10 < 20")
	}

	if compareValues(int64(50), int64(30)) != 1 {
		t.Error("Expected 50 > 30")
	}

	if compareValues(int64(42), int64(42)) != 0 {
		t.Error("Expected 42 == 42")
	}

	// Test float comparison
	if compareValues(3.14, 2.71) != 1 {
		t.Error("Expected 3.14 > 2.71")
	}

	// Test boolean comparison
	if compareValues(false, true) != -1 {
		t.Error("Expected false < true")
	}

	if compareValues(true, false) != 1 {
		t.Error("Expected true > false")
	}

	if compareValues(true, true) != 0 {
		t.Error("Expected true == true")
	}

	// Test nil comparison
	if compareValues(nil, "something") != -1 {
		t.Error("Expected nil < non-nil")
	}

	if compareValues("something", nil) != 1 {
		t.Error("Expected non-nil > nil")
	}

	if compareValues(nil, nil) != 0 {
		t.Error("Expected nil == nil")
	}

	// Test mixed types - returns 0 (equal) for unsupported type combinations
	result := compareValues(int64(10), "string")
	if result != 0 {
		t.Error("Expected mixed types to be treated as equal (return 0)")
	}

	// Test int32 comparison
	if compareValues(int32(5), int32(10)) != -1 {
		t.Error("Expected int32(5) < int32(10)")
	}

	if compareValues(int32(20), int32(10)) != 1 {
		t.Error("Expected int32(20) > int32(10)")
	}

	// Test []byte comparison
	if compareValues([]byte("abc"), []byte("def")) != -1 {
		t.Error("Expected []byte('abc') < []byte('def')")
	}

	if compareValues([]byte("xyz"), []byte("abc")) != 1 {
		t.Error("Expected []byte('xyz') > []byte('abc')")
	}

	// Test int64 vs int comparison (cross-type numeric)
	if compareValues(int64(5), int(10)) != -1 {
		t.Error("Expected int64(5) < int(10)")
	}

	if compareValues(int64(20), int(10)) != 1 {
		t.Error("Expected int64(20) > int(10)")
	}

	// Test int comparison
	if compareValues(int(5), int(10)) != -1 {
		t.Error("Expected int(5) < int(10)")
	}

	if compareValues(int(20), int(10)) != 1 {
		t.Error("Expected int(20) > int(10)")
	}
}
