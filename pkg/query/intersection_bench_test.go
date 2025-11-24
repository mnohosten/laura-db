package query

import (
	"fmt"
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/index"
)

// BenchmarkIndexIntersectionTwoFields benchmarks intersection with two indexes
func BenchmarkIndexIntersectionTwoFields(b *testing.B) {
	// Create 10,000 documents
	docs := make([]*document.Document, 10000)
	for i := 0; i < 10000; i++ {
		docs[i] = document.NewDocumentFromMap(map[string]interface{}{
			"_id":  document.NewObjectID().Hex(),
			"age":  int64(20 + (i % 50)),        // Ages 20-69
			"city": "City" + fmt.Sprintf("%d", i%20), // 20 different cities
		})
	}

	// Create indexes
	ageIndex := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})
	cityIndex := index.NewIndex(&index.IndexConfig{
		Name:      "city_idx",
		FieldPath: "city",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	for _, doc := range docs {
		if age, exists := doc.Get("age"); exists {
			if id, exists := doc.Get("_id"); exists {
				ageIndex.Insert(age, id.(string))
			}
		}
		if city, exists := doc.Get("city"); exists {
			if id, exists := doc.Get("_id"); exists {
				cityIndex.Insert(city, id.(string))
			}
		}
	}

	ageIndex.Analyze()
	cityIndex.Analyze()

	indexes := map[string]*index.Index{
		"age_idx":  ageIndex,
		"city_idx": cityIndex,
	}
	planner := NewQueryPlanner(indexes)

	query := NewQuery(map[string]interface{}{
		"age":  int64(25),
		"city": "City5",
	})

	plan := planner.Plan(query)
	executor := NewExecutor(docs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.ExecuteWithPlan(query, plan)
		if err != nil {
			b.Fatalf("Query execution failed: %v", err)
		}
	}
}

// BenchmarkSingleIndexTwoFields benchmarks using a single index for same query
func BenchmarkSingleIndexTwoFields(b *testing.B) {
	// Create 10,000 documents
	docs := make([]*document.Document, 10000)
	for i := 0; i < 10000; i++ {
		docs[i] = document.NewDocumentFromMap(map[string]interface{}{
			"_id":  document.NewObjectID().Hex(),
			"age":  int64(20 + (i % 50)),
			"city": "City" + fmt.Sprintf("%d", i%20),
		})
	}

	// Create only age index (force single index usage)
	ageIndex := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	for _, doc := range docs {
		if age, exists := doc.Get("age"); exists {
			if id, exists := doc.Get("_id"); exists {
				ageIndex.Insert(age, id.(string))
			}
		}
	}

	ageIndex.Analyze()

	indexes := map[string]*index.Index{
		"age_idx": ageIndex,
	}
	planner := NewQueryPlanner(indexes)

	query := NewQuery(map[string]interface{}{
		"age":  int64(25),
		"city": "City5",
	})

	plan := planner.Plan(query)
	executor := NewExecutor(docs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.ExecuteWithPlan(query, plan)
		if err != nil {
			b.Fatalf("Query execution failed: %v", err)
		}
	}
}

// BenchmarkIndexIntersectionThreeFields benchmarks intersection with three indexes
func BenchmarkIndexIntersectionThreeFields(b *testing.B) {
	docs := make([]*document.Document, 10000)
	for i := 0; i < 10000; i++ {
		docs[i] = document.NewDocumentFromMap(map[string]interface{}{
			"_id":    document.NewObjectID().Hex(),
			"age":    int64(20 + (i % 50)),
			"city":   "City" + fmt.Sprintf("%d", i%20),
			"status": []string{"active", "inactive"}[i%2],
		})
	}

	ageIndex := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})
	cityIndex := index.NewIndex(&index.IndexConfig{
		Name:      "city_idx",
		FieldPath: "city",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})
	statusIndex := index.NewIndex(&index.IndexConfig{
		Name:      "status_idx",
		FieldPath: "status",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	for _, doc := range docs {
		if age, exists := doc.Get("age"); exists {
			if id, exists := doc.Get("_id"); exists {
				ageIndex.Insert(age, id.(string))
			}
		}
		if city, exists := doc.Get("city"); exists {
			if id, exists := doc.Get("_id"); exists {
				cityIndex.Insert(city, id.(string))
			}
		}
		if status, exists := doc.Get("status"); exists {
			if id, exists := doc.Get("_id"); exists {
				statusIndex.Insert(status, id.(string))
			}
		}
	}

	ageIndex.Analyze()
	cityIndex.Analyze()
	statusIndex.Analyze()

	indexes := map[string]*index.Index{
		"age_idx":    ageIndex,
		"city_idx":   cityIndex,
		"status_idx": statusIndex,
	}
	planner := NewQueryPlanner(indexes)

	query := NewQuery(map[string]interface{}{
		"age":    int64(25),
		"city":   "City5",
		"status": "active",
	})

	plan := planner.Plan(query)
	executor := NewExecutor(docs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.ExecuteWithPlan(query, plan)
		if err != nil {
			b.Fatalf("Query execution failed: %v", err)
		}
	}
}

// BenchmarkCollectionScanTwoFields benchmarks collection scan without indexes
func BenchmarkCollectionScanTwoFields(b *testing.B) {
	docs := make([]*document.Document, 10000)
	for i := 0; i < 10000; i++ {
		docs[i] = document.NewDocumentFromMap(map[string]interface{}{
			"_id":  document.NewObjectID().Hex(),
			"age":  int64(20 + (i % 50)),
			"city": "City" + fmt.Sprintf("%d", i%20),
		})
	}

	// No indexes
	indexes := map[string]*index.Index{}
	planner := NewQueryPlanner(indexes)

	query := NewQuery(map[string]interface{}{
		"age":  int64(25),
		"city": "City5",
	})

	plan := planner.Plan(query)
	executor := NewExecutor(docs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.ExecuteWithPlan(query, plan)
		if err != nil {
			b.Fatalf("Query execution failed: %v", err)
		}
	}
}

// BenchmarkIndexIntersectionScaling benchmarks how intersection scales with data size
func BenchmarkIndexIntersectionScaling(b *testing.B) {
	sizes := []int{1000, 5000, 10000, 50000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			docs := make([]*document.Document, size)
			for i := 0; i < size; i++ {
				docs[i] = document.NewDocumentFromMap(map[string]interface{}{
					"_id":  document.NewObjectID().Hex(),
					"age":  int64(20 + (i % 50)),
					"city": "City" + fmt.Sprintf("%d", i%20),
				})
			}

			ageIndex := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})
			cityIndex := index.NewIndex(&index.IndexConfig{
		Name:      "city_idx",
		FieldPath: "city",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

			for _, doc := range docs {
				if age, exists := doc.Get("age"); exists {
					if id, exists := doc.Get("_id"); exists {
						ageIndex.Insert(age, id.(string))
					}
				}
				if city, exists := doc.Get("city"); exists {
					if id, exists := doc.Get("_id"); exists {
						cityIndex.Insert(city, id.(string))
					}
				}
			}

			ageIndex.Analyze()
			cityIndex.Analyze()

			indexes := map[string]*index.Index{
				"age_idx":  ageIndex,
				"city_idx": cityIndex,
			}
			planner := NewQueryPlanner(indexes)

			query := NewQuery(map[string]interface{}{
				"age":  int64(25),
				"city": "City5",
			})

			plan := planner.Plan(query)
			executor := NewExecutor(docs)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := executor.ExecuteWithPlan(query, plan)
				if err != nil {
					b.Fatalf("Query execution failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkIndexIntersectionWithRanges benchmarks intersection with range queries
func BenchmarkIndexIntersectionWithRanges(b *testing.B) {
	docs := make([]*document.Document, 10000)
	for i := 0; i < 10000; i++ {
		docs[i] = document.NewDocumentFromMap(map[string]interface{}{
			"_id":    document.NewObjectID().Hex(),
			"age":    int64(20 + (i % 50)),
			"salary": int64(30000 + (i % 100000)),
		})
	}

	ageIndex := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})
	salaryIndex := index.NewIndex(&index.IndexConfig{
		Name:      "salary_idx",
		FieldPath: "salary",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	for _, doc := range docs {
		if age, exists := doc.Get("age"); exists {
			if id, exists := doc.Get("_id"); exists {
				ageIndex.Insert(age, id.(string))
			}
		}
		if salary, exists := doc.Get("salary"); exists {
			if id, exists := doc.Get("_id"); exists {
				salaryIndex.Insert(salary, id.(string))
			}
		}
	}

	ageIndex.Analyze()
	salaryIndex.Analyze()

	indexes := map[string]*index.Index{
		"age_idx":    ageIndex,
		"salary_idx": salaryIndex,
	}
	planner := NewQueryPlanner(indexes)

	query := NewQuery(map[string]interface{}{
		"age":    map[string]interface{}{"$gte": int64(30)},
		"salary": map[string]interface{}{"$lte": int64(70000)},
	})

	plan := planner.Plan(query)
	executor := NewExecutor(docs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.ExecuteWithPlan(query, plan)
		if err != nil {
			b.Fatalf("Query execution failed: %v", err)
		}
	}
}

// BenchmarkIndexIntersectionSelectivity benchmarks intersection with varying selectivity
func BenchmarkIndexIntersectionSelectivity(b *testing.B) {
	// High selectivity (few matches per index)
	b.Run("high_selectivity", func(b *testing.B) {
		docs := make([]*document.Document, 10000)
		for i := 0; i < 10000; i++ {
			docs[i] = document.NewDocumentFromMap(map[string]interface{}{
				"_id":  document.NewObjectID().Hex(),
				"age":  int64(20 + (i % 100)), // 100 different ages
				"city": "City" + fmt.Sprintf("%d", i%100), // 100 different cities
			})
		}

		ageIndex := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})
		cityIndex := index.NewIndex(&index.IndexConfig{
		Name:      "city_idx",
		FieldPath: "city",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

		for _, doc := range docs {
			if age, exists := doc.Get("age"); exists {
				if id, exists := doc.Get("_id"); exists {
					ageIndex.Insert(age, id.(string))
				}
			}
			if city, exists := doc.Get("city"); exists {
				if id, exists := doc.Get("_id"); exists {
					cityIndex.Insert(city, id.(string))
				}
			}
		}

		ageIndex.Analyze()
		cityIndex.Analyze()

		indexes := map[string]*index.Index{
			"age_idx":  ageIndex,
			"city_idx": cityIndex,
		}
		planner := NewQueryPlanner(indexes)

		query := NewQuery(map[string]interface{}{
			"age":  int64(25),
			"city": "City25",
		})

		plan := planner.Plan(query)
		executor := NewExecutor(docs)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := executor.ExecuteWithPlan(query, plan)
			if err != nil {
				b.Fatalf("Query execution failed: %v", err)
			}
		}
	})

	// Low selectivity (many matches per index)
	b.Run("low_selectivity", func(b *testing.B) {
		docs := make([]*document.Document, 10000)
		for i := 0; i < 10000; i++ {
			docs[i] = document.NewDocumentFromMap(map[string]interface{}{
				"_id":  document.NewObjectID().Hex(),
				"age":  int64(20 + (i % 5)), // Only 5 different ages
				"city": "City" + fmt.Sprintf("%d", i%5), // Only 5 different cities
			})
		}

		ageIndex := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})
		cityIndex := index.NewIndex(&index.IndexConfig{
		Name:      "city_idx",
		FieldPath: "city",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

		for _, doc := range docs {
			if age, exists := doc.Get("age"); exists {
				if id, exists := doc.Get("_id"); exists {
					ageIndex.Insert(age, id.(string))
				}
			}
			if city, exists := doc.Get("city"); exists {
				if id, exists := doc.Get("_id"); exists {
					cityIndex.Insert(city, id.(string))
				}
			}
		}

		ageIndex.Analyze()
		cityIndex.Analyze()

		indexes := map[string]*index.Index{
			"age_idx":  ageIndex,
			"city_idx": cityIndex,
		}
		planner := NewQueryPlanner(indexes)

		query := NewQuery(map[string]interface{}{
			"age":  int64(22),
			"city": "City2",
		})

		plan := planner.Plan(query)
		executor := NewExecutor(docs)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := executor.ExecuteWithPlan(query, plan)
			if err != nil {
				b.Fatalf("Query execution failed: %v", err)
			}
		}
	})
}
