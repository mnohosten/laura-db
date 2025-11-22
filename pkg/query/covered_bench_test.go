package query

import (
	"fmt"
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/index"
)

// BenchmarkCoveredQuery benchmarks a covered query (index-only)
func BenchmarkCoveredQuery(b *testing.B) {
	// Create test documents
	docs := make([]*document.Document, 1000)
	for i := 0; i < 1000; i++ {
		doc := document.NewDocument()
		doc.Set("_id", fmt.Sprintf("id_%d", i))
		doc.Set("age", int64(20+(i%50)))
		doc.Set("name", fmt.Sprintf("User%d", i))
		doc.Set("email", fmt.Sprintf("user%d@example.com", i))
		doc.Set("status", "active")
		docs[i] = doc
	}

	// Create index on age
	idx := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Populate index
	for _, doc := range docs {
		if ageVal, exists := doc.Get("age"); exists {
			if idVal, exists := doc.Get("_id"); exists {
				idx.Insert(ageVal, idVal)
			}
		}
	}

	indexes := map[string]*index.Index{
		"age_idx": idx,
	}

	executor := NewExecutor(docs)
	planner := NewQueryPlanner(indexes)

	// Query with projection that can be satisfied from index
	query := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{"$gte": int64(30)},
	})
	query.WithProjection(map[string]bool{"age": true, "_id": true})

	plan := planner.Plan(query)
	planner.DetectCoveredQuery(plan, query.GetProjection())

	if !plan.IsCovered {
		b.Fatal("Expected covered query")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.ExecuteWithPlan(query, plan)
	}
}

// BenchmarkNonCoveredQuery benchmarks a non-covered query (requires document fetch)
func BenchmarkNonCoveredQuery(b *testing.B) {
	// Create test documents
	docs := make([]*document.Document, 1000)
	for i := 0; i < 1000; i++ {
		doc := document.NewDocument()
		doc.Set("_id", fmt.Sprintf("id_%d", i))
		doc.Set("age", int64(20+(i%50)))
		doc.Set("name", fmt.Sprintf("User%d", i))
		doc.Set("email", fmt.Sprintf("user%d@example.com", i))
		doc.Set("status", "active")
		docs[i] = doc
	}

	// Create index on age
	idx := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Populate index
	for _, doc := range docs {
		if ageVal, exists := doc.Get("age"); exists {
			if idVal, exists := doc.Get("_id"); exists {
				idx.Insert(ageVal, idVal)
			}
		}
	}

	indexes := map[string]*index.Index{
		"age_idx": idx,
	}

	executor := NewExecutor(docs)
	planner := NewQueryPlanner(indexes)

	// Query with projection that requires additional fields (not covered)
	query := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{"$gte": int64(30)},
	})
	query.WithProjection(map[string]bool{"age": true, "_id": true, "name": true})

	plan := planner.Plan(query)
	planner.DetectCoveredQuery(plan, query.GetProjection())

	if plan.IsCovered {
		b.Fatal("Expected non-covered query")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.ExecuteWithPlan(query, plan)
	}
}

// BenchmarkCoveredQueryLarge benchmarks covered query with larger dataset
func BenchmarkCoveredQueryLarge(b *testing.B) {
	// Create test documents
	docs := make([]*document.Document, 10000)
	for i := 0; i < 10000; i++ {
		doc := document.NewDocument()
		doc.Set("_id", fmt.Sprintf("id_%d", i))
		doc.Set("age", int64(20+(i%80)))
		doc.Set("name", fmt.Sprintf("User%d", i))
		doc.Set("email", fmt.Sprintf("user%d@example.com", i))
		doc.Set("status", "active")
		doc.Set("city", []string{"NYC", "LA", "SF", "CHI", "BOS"}[i%5])
		docs[i] = doc
	}

	// Create index on age
	idx := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Populate index
	for _, doc := range docs {
		if ageVal, exists := doc.Get("age"); exists {
			if idVal, exists := doc.Get("_id"); exists {
				idx.Insert(ageVal, idVal)
			}
		}
	}

	indexes := map[string]*index.Index{
		"age_idx": idx,
	}

	executor := NewExecutor(docs)
	planner := NewQueryPlanner(indexes)

	// Covered query
	query := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{"$gte": int64(50)},
	})
	query.WithProjection(map[string]bool{"age": true})

	plan := planner.Plan(query)
	planner.DetectCoveredQuery(plan, query.GetProjection())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.ExecuteWithPlan(query, plan)
	}
}

// BenchmarkNonCoveredQueryLarge benchmarks non-covered query with larger dataset
func BenchmarkNonCoveredQueryLarge(b *testing.B) {
	// Create test documents
	docs := make([]*document.Document, 10000)
	for i := 0; i < 10000; i++ {
		doc := document.NewDocument()
		doc.Set("_id", fmt.Sprintf("id_%d", i))
		doc.Set("age", int64(20+(i%80)))
		doc.Set("name", fmt.Sprintf("User%d", i))
		doc.Set("email", fmt.Sprintf("user%d@example.com", i))
		doc.Set("status", "active")
		doc.Set("city", []string{"NYC", "LA", "SF", "CHI", "BOS"}[i%5])
		docs[i] = doc
	}

	// Create index on age
	idx := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Populate index
	for _, doc := range docs {
		if ageVal, exists := doc.Get("age"); exists {
			if idVal, exists := doc.Get("_id"); exists {
				idx.Insert(ageVal, idVal)
			}
		}
	}

	indexes := map[string]*index.Index{
		"age_idx": idx,
	}

	executor := NewExecutor(docs)
	planner := NewQueryPlanner(indexes)

	// Non-covered query (needs all fields)
	query := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{"$gte": int64(50)},
	})
	// No projection = need all fields

	plan := planner.Plan(query)
	planner.DetectCoveredQuery(plan, query.GetProjection())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.ExecuteWithPlan(query, plan)
	}
}
