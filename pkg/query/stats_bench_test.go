package query

import (
	"fmt"
	"testing"

	"github.com/mnohosten/laura-db/pkg/index"
)

// BenchmarkPlannerWithStatistics benchmarks query planning with statistics
func BenchmarkPlannerWithStatistics(b *testing.B) {
	// Create two indexes with different cardinality
	highCardIdx := index.NewIndex(&index.IndexConfig{
		Name:      "high_card_idx",
		FieldPath: "unique_id",
		Type:      index.IndexTypeBTree,
		Unique:    true,
		Order:     32,
	})

	// Insert 10,000 unique values
	for i := 0; i < 10000; i++ {
		highCardIdx.Insert(int64(i), fmt.Sprintf("id_%d", i))
	}
	highCardIdx.Analyze()

	lowCardIdx := index.NewIndex(&index.IndexConfig{
		Name:      "low_card_idx",
		FieldPath: "category",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Insert only 10 unique values
	for i := 0; i < 10; i++ {
		lowCardIdx.Insert(int64(i), fmt.Sprintf("id_%d", i))
	}
	lowCardIdx.Analyze()

	indexes := map[string]*index.Index{
		"high_card_idx": highCardIdx,
		"low_card_idx":  lowCardIdx,
	}

	planner := NewQueryPlanner(indexes)

	// Query that can use either index
	query := NewQuery(map[string]interface{}{
		"unique_id": int64(5000),
		"category":  int64(5),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plan := planner.Plan(query)
		_ = plan
	}
}

// BenchmarkPlannerWithoutStatistics benchmarks query planning without statistics (stale stats)
func BenchmarkPlannerWithoutStatistics(b *testing.B) {
	// Create two indexes
	highCardIdx := index.NewIndex(&index.IndexConfig{
		Name:      "high_card_idx",
		FieldPath: "unique_id",
		Type:      index.IndexTypeBTree,
		Unique:    true,
		Order:     32,
	})

	for i := 0; i < 10000; i++ {
		highCardIdx.Insert(int64(i), fmt.Sprintf("id_%d", i))
	}
	// Don't analyze - stats will be stale

	lowCardIdx := index.NewIndex(&index.IndexConfig{
		Name:      "low_card_idx",
		FieldPath: "category",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	for i := 0; i < 10; i++ {
		lowCardIdx.Insert(int64(i), fmt.Sprintf("id_%d", i))
	}
	// Don't analyze - stats will be stale

	indexes := map[string]*index.Index{
		"high_card_idx": highCardIdx,
		"low_card_idx":  lowCardIdx,
	}

	planner := NewQueryPlanner(indexes)

	query := NewQuery(map[string]interface{}{
		"unique_id": int64(5000),
		"category":  int64(5),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plan := planner.Plan(query)
		_ = plan
	}
}

// BenchmarkIndexAnalyze benchmarks the cost of analyzing an index
func BenchmarkIndexAnalyze(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			idx := index.NewIndex(&index.IndexConfig{
				Name:      "bench_idx",
				FieldPath: "field",
				Type:      index.IndexTypeBTree,
				Unique:    false,
				Order:     32,
			})

			// Insert data
			for i := 0; i < size; i++ {
				idx.Insert(int64(i), fmt.Sprintf("id_%d", i))
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idx.Analyze()
			}
		})
	}
}

// BenchmarkCostEstimation benchmarks cost estimation with different index sizes
func BenchmarkCostEstimation(b *testing.B) {
	// Create indexes with different cardinalities
	testCases := []struct {
		name        string
		cardinality int
	}{
		{"card_10", 10},
		{"card_100", 100},
		{"card_1000", 1000},
		{"card_10000", 10000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			idx := index.NewIndex(&index.IndexConfig{
				Name:      "bench_idx",
				FieldPath: "field",
				Type:      index.IndexTypeBTree,
				Unique:    false,
				Order:     32,
			})

			for i := 0; i < tc.cardinality; i++ {
				idx.Insert(int64(i), fmt.Sprintf("id_%d", i))
			}
			idx.Analyze()

			indexes := map[string]*index.Index{
				"bench_idx": idx,
			}

			planner := NewQueryPlanner(indexes)

			query := NewQuery(map[string]interface{}{
				"field": int64(50),
			})

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				plan := planner.Plan(query)
				_ = plan
			}
		})
	}
}

// BenchmarkMultiIndexSelection benchmarks planning with many indexes
func BenchmarkMultiIndexSelection(b *testing.B) {
	numIndexes := []int{2, 5, 10, 20}

	for _, n := range numIndexes {
		b.Run(fmt.Sprintf("indexes_%d", n), func(b *testing.B) {
			indexes := make(map[string]*index.Index)

			// Create n indexes with varying cardinality
			for i := 0; i < n; i++ {
				idx := index.NewIndex(&index.IndexConfig{
					Name:      fmt.Sprintf("idx_%d", i),
					FieldPath: fmt.Sprintf("field_%d", i),
					Type:      index.IndexTypeBTree,
					Unique:    false,
					Order:     32,
				})

				// Varying cardinality: 10, 100, 1000, etc.
				cardinality := 10 * (i + 1) * (i + 1)
				for j := 0; j < cardinality; j++ {
					idx.Insert(int64(j), fmt.Sprintf("id_%d", j))
				}
				idx.Analyze()

				indexes[fmt.Sprintf("idx_%d", i)] = idx
			}

			planner := NewQueryPlanner(indexes)

			// Query that matches all indexes
			filter := make(map[string]interface{})
			for i := 0; i < n; i++ {
				filter[fmt.Sprintf("field_%d", i)] = int64(50)
			}
			query := NewQuery(filter)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				plan := planner.Plan(query)
				_ = plan
			}
		})
	}
}

// BenchmarkRangeQueryCostEstimation benchmarks range query cost estimation
func BenchmarkRangeQueryCostEstimation(b *testing.B) {
	idx := index.NewIndex(&index.IndexConfig{
		Name:      "range_idx",
		FieldPath: "score",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Insert 100,000 entries
	for i := 0; i < 100000; i++ {
		idx.Insert(int64(i), fmt.Sprintf("id_%d", i))
	}
	idx.Analyze()

	indexes := map[string]*index.Index{
		"range_idx": idx,
	}

	planner := NewQueryPlanner(indexes)

	// Range query
	query := NewQuery(map[string]interface{}{
		"score": map[string]interface{}{"$gte": int64(50000)},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plan := planner.Plan(query)
		_ = plan
	}
}
