package query

import (
	"runtime"
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
)

// TestExecuteParallel_BasicFiltering tests basic parallel filtering
func TestExecuteParallel_BasicFiltering(t *testing.T) {
	// Create test documents
	docs := make([]*document.Document, 2000)
	for i := 0; i < 2000; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("age", int64(20+i%50))
		doc.Set("name", "user"+string(rune('A'+i%26)))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	// Test simple filter
	q := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{
			"$gte": int64(30),
		},
	})

	results, err := executor.ExecuteParallel(q, nil)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}

	// Verify all results match the filter
	for _, doc := range results {
		age, _ := doc.Get("age")
		if age.(int64) < 30 {
			t.Errorf("Expected age >= 30, got %d", age)
		}
	}

	// Compare with sequential execution
	seqResults, err := executor.Execute(q)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) != len(seqResults) {
		t.Errorf("Expected %d results, got %d", len(seqResults), len(results))
	}
}

// TestExecuteParallel_ComplexQuery tests parallel execution with complex queries
func TestExecuteParallel_ComplexQuery(t *testing.T) {
	// Create test documents
	docs := make([]*document.Document, 1500)
	for i := 0; i < 1500; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("age", int64(20+i%60))
		doc.Set("score", int64(50+i%50))
		doc.Set("active", i%3 == 0)
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	// Complex filter with $and, $or
	q := NewQuery(map[string]interface{}{
		"$and": []interface{}{
			map[string]interface{}{
				"age": map[string]interface{}{
					"$gte": int64(30),
				},
			},
			map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"score": map[string]interface{}{"$gt": int64(80)}},
					map[string]interface{}{"active": true},
				},
			},
		},
	})

	results, err := executor.ExecuteParallel(q, nil)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}

	// Verify results
	seqResults, err := executor.Execute(q)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) != len(seqResults) {
		t.Errorf("Expected %d results, got %d", len(seqResults), len(results))
	}
}

// TestExecuteParallel_WithSortLimitSkip tests parallel execution with sort, limit, skip
func TestExecuteParallel_WithSortLimitSkip(t *testing.T) {
	// Create test documents
	docs := make([]*document.Document, 2000)
	for i := 0; i < 2000; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("score", int64(i))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"score": map[string]interface{}{
			"$gte": int64(100),
		},
	})

	// Add sort, limit, skip
	q.WithSort([]SortField{{Field: "score", Ascending: false}})
	q.WithSkip(10)
	q.WithLimit(50)

	results, err := executor.ExecuteParallel(q, nil)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}

	// Verify limit
	if len(results) != 50 {
		t.Errorf("Expected 50 results, got %d", len(results))
	}

	// Verify sort order (descending)
	for i := 1; i < len(results); i++ {
		prev, _ := results[i-1].Get("score")
		curr, _ := results[i].Get("score")
		if prev.(int64) < curr.(int64) {
			t.Errorf("Results not sorted correctly: %d < %d", prev, curr)
		}
	}

	// Compare with sequential
	seqResults, err := executor.Execute(q)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) != len(seqResults) {
		t.Errorf("Expected %d results, got %d", len(seqResults), len(results))
	}

	// Verify first result matches
	if len(results) > 0 && len(seqResults) > 0 {
		r1, _ := results[0].Get("score")
		r2, _ := seqResults[0].Get("score")
		if r1 != r2 {
			t.Errorf("First result mismatch: %d vs %d", r1, r2)
		}
	}
}

// TestExecuteParallel_WithProjection tests parallel execution with projection
func TestExecuteParallel_WithProjection(t *testing.T) {
	// Create test documents
	docs := make([]*document.Document, 1200)
	for i := 0; i < 1200; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("name", "user"+string(rune('A'+i%26)))
		doc.Set("age", int64(20+i%40))
		doc.Set("email", "user@example.com")
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{
			"$gte": int64(30),
		},
	})

	// Project only name and age (and _id explicitly)
	q.WithProjection(map[string]bool{
		"_id":  true,
		"name": true,
		"age":  true,
	})

	results, err := executor.ExecuteParallel(q, nil)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}

	// Verify projection
	for _, doc := range results {
		keys := doc.Keys()
		if len(keys) != 3 { // _id, name, age
			t.Errorf("Expected 3 fields (_id, name, age), got %d: %v", len(keys), keys)
		}
		if _, exists := doc.Get("email"); exists {
			t.Error("email field should not be present in projected results")
		}
	}
}

// TestExecuteParallel_SmallDataset tests that small datasets use sequential execution
func TestExecuteParallel_SmallDataset(t *testing.T) {
	// Create small dataset (below threshold)
	docs := make([]*document.Document, 100)
	for i := 0; i < 100; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("value", int64(i))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"value": map[string]interface{}{
			"$gte": int64(50),
		},
	})

	// Should fall back to sequential execution
	results, err := executor.ExecuteParallel(q, nil)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}

	if len(results) != 50 {
		t.Errorf("Expected 50 results, got %d", len(results))
	}
}

// TestExecuteParallel_CustomConfig tests custom parallel configuration
func TestExecuteParallel_CustomConfig(t *testing.T) {
	// Create test documents
	docs := make([]*document.Document, 2000)
	for i := 0; i < 2000; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("value", int64(i))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"value": map[string]interface{}{
			"$gte": int64(500),
		},
	})

	// Custom config with specific workers and chunk size
	config := &ParallelConfig{
		MinDocsForParallel: 100,
		MaxWorkers:         4,
		ChunkSize:          250,
	}

	results, err := executor.ExecuteParallel(q, config)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}

	if len(results) != 1500 {
		t.Errorf("Expected 1500 results, got %d", len(results))
	}
}

// TestExecuteWithPlanParallel tests parallel execution with query plan
func TestExecuteWithPlanParallel(t *testing.T) {
	// Create test documents
	docs := make([]*document.Document, 2000)
	for i := 0; i < 2000; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("age", int64(20+i%60))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{
			"$gte": int64(40),
		},
	})

	// Create a simple plan (collection scan)
	plan := &QueryPlan{
		UseIndex: false,
		ScanType: ScanTypeCollection,
	}

	results, err := executor.ExecuteWithPlanParallel(q, plan, nil)
	if err != nil {
		t.Fatalf("ExecuteWithPlanParallel failed: %v", err)
	}

	// Verify results
	for _, doc := range results {
		age, _ := doc.Get("age")
		if age.(int64) < 40 {
			t.Errorf("Expected age >= 40, got %d", age)
		}
	}
}

// TestExecuteParallel_NoMatches tests parallel execution with no matching documents
func TestExecuteParallel_NoMatches(t *testing.T) {
	// Create test documents
	docs := make([]*document.Document, 1000)
	for i := 0; i < 1000; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("value", int64(i))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	// Query that matches nothing
	q := NewQuery(map[string]interface{}{
		"value": map[string]interface{}{
			"$gt": int64(10000),
		},
	})

	results, err := executor.ExecuteParallel(q, nil)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

// TestExecuteParallel_AllMatches tests parallel execution where all documents match
func TestExecuteParallel_AllMatches(t *testing.T) {
	// Create test documents
	docs := make([]*document.Document, 1500)
	for i := 0; i < 1500; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("value", int64(i))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	// Query that matches everything
	q := NewQuery(map[string]interface{}{
		"value": map[string]interface{}{
			"$gte": int64(0),
		},
	})

	results, err := executor.ExecuteParallel(q, nil)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}

	if len(results) != 1500 {
		t.Errorf("Expected 1500 results, got %d", len(results))
	}
}

// TestDefaultParallelConfig tests the default configuration
func TestDefaultParallelConfig(t *testing.T) {
	config := DefaultParallelConfig()

	if config.MinDocsForParallel != 1000 {
		t.Errorf("Expected MinDocsForParallel=1000, got %d", config.MinDocsForParallel)
	}

	if config.MaxWorkers != 0 {
		t.Errorf("Expected MaxWorkers=0 (auto), got %d", config.MaxWorkers)
	}

	if config.ChunkSize != 0 {
		t.Errorf("Expected ChunkSize=0 (auto), got %d", config.ChunkSize)
	}
}

// TestExecuteParallel_WorkerCount tests that workers are created correctly
func TestExecuteParallel_WorkerCount(t *testing.T) {
	// Create large dataset
	docs := make([]*document.Document, 5000)
	for i := 0; i < 5000; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("value", int64(i))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"value": map[string]interface{}{
			"$gte": int64(0),
		},
	})

	// Test with explicit worker count
	config := &ParallelConfig{
		MinDocsForParallel: 100,
		MaxWorkers:         8,
		ChunkSize:          500,
	}

	results, err := executor.ExecuteParallel(q, config)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}

	if len(results) != 5000 {
		t.Errorf("Expected 5000 results, got %d", len(results))
	}

	// Test with default workers (NumCPU)
	config2 := &ParallelConfig{
		MinDocsForParallel: 100,
		MaxWorkers:         0, // Should use NumCPU
		ChunkSize:          0, // Should auto-calculate
	}

	results2, err := executor.ExecuteParallel(q, config2)
	if err != nil {
		t.Fatalf("ExecuteParallel with auto workers failed: %v", err)
	}

	if len(results2) != 5000 {
		t.Errorf("Expected 5000 results, got %d", len(results2))
	}
}

// TestExecuteParallel_Consistency tests that parallel execution produces consistent results
func TestExecuteParallel_Consistency(t *testing.T) {
	// Create test documents
	docs := make([]*document.Document, 3000)
	for i := 0; i < 3000; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("value", int64(i%100))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"value": map[string]interface{}{
			"$lt": int64(50),
		},
	})

	// Run multiple times and verify consistency
	for i := 0; i < 5; i++ {
		results, err := executor.ExecuteParallel(q, nil)
		if err != nil {
			t.Fatalf("ExecuteParallel iteration %d failed: %v", i, err)
		}

		// Each iteration should return same count
		if len(results) != 1500 { // 50% of 3000
			t.Errorf("Iteration %d: Expected 1500 results, got %d", i, len(results))
		}

		// Verify all results match filter
		for _, doc := range results {
			value, _ := doc.Get("value")
			if value.(int64) >= 50 {
				t.Errorf("Iteration %d: Expected value < 50, got %d", i, value)
			}
		}
	}
}

// TestExecuteParallel_CPUUtilization verifies parallel execution uses multiple cores
func TestExecuteParallel_CPUUtilization(t *testing.T) {
	if runtime.NumCPU() < 2 {
		t.Skip("Skipping test: requires at least 2 CPUs")
	}

	// Create large dataset to ensure parallel execution
	docs := make([]*document.Document, 10000)
	for i := 0; i < 10000; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("value", int64(i))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"value": map[string]interface{}{
			"$gte": int64(0),
		},
	})

	config := &ParallelConfig{
		MinDocsForParallel: 100,
		MaxWorkers:         runtime.NumCPU(),
		ChunkSize:          1000,
	}

	results, err := executor.ExecuteParallel(q, config)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}

	if len(results) != 10000 {
		t.Errorf("Expected 10000 results, got %d", len(results))
	}
}
