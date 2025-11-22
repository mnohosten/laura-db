package query

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/index"
)

func TestPlannerWithStatistics(t *testing.T) {
	// Create two indexes with different cardinality
	// Index 1: age - lower cardinality (5 unique values out of 5 entries)
	ageIdx := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Insert only 5 unique ages
	for i := 0; i < 5; i++ {
		ageIdx.Insert(int64(20+i), "id_"+string(rune(i)))
	}
	ageIdx.Analyze()

	// Index 2: email - higher cardinality (100 unique values out of 100 entries)
	emailIdx := index.NewIndex(&index.IndexConfig{
		Name:      "email_idx",
		FieldPath: "email",
		Type:      index.IndexTypeBTree,
		Unique:    true,
		Order:     32,
	})

	// Insert 100 unique emails
	for i := 0; i < 100; i++ {
		emailIdx.Insert("email_"+string(rune(i)), "id_"+string(rune(i)))
	}
	emailIdx.Analyze()

	indexes := map[string]*index.Index{
		"age_idx":   ageIdx,
		"email_idx": emailIdx,
	}

	planner := NewQueryPlanner(indexes)

	t.Run("Choose high selectivity index", func(t *testing.T) {
		// Query that can use either index
		query := NewQuery(map[string]interface{}{
			"age":   20,
			"email": "test@example.com",
		})

		plan := planner.Plan(query)

		// Should choose email_idx because it has higher selectivity
		if plan.IndexName != "email_idx" {
			t.Errorf("Expected to use email_idx (high selectivity), got %s", plan.IndexName)
		}

		// Verify cardinality is higher for email index
		ageStats := ageIdx.GetStatistics()
		emailStats := emailIdx.GetStatistics()

		if ageStats.Cardinality() >= emailStats.Cardinality() {
			t.Errorf("Expected email index to have higher cardinality: age=%d, email=%d",
				ageStats.Cardinality(), emailStats.Cardinality())
		}
	})

	t.Run("Verify cost estimation", func(t *testing.T) {
		// Create plans for each index
		agePlan := planner.analyzeIndexForFilter("age_idx", ageIdx,
			map[string]interface{}{"age": 20})
		emailPlan := planner.analyzeIndexForFilter("email_idx", emailIdx,
			map[string]interface{}{"email": "test@example.com"})

		// Estimate costs with statistics
		ageCost := planner.estimateCostWithStats(agePlan, ageIdx)
		emailCost := planner.estimateCostWithStats(emailPlan, emailIdx)

		// Email index should have lower cost (more selective)
		if emailCost >= ageCost {
			t.Errorf("Expected email_idx to have lower cost: email=%d, age=%d",
				emailCost, ageCost)
		}
	})
}

func TestPlannerWithoutStatistics(t *testing.T) {
	// Create index without running Analyze()
	idx := index.NewIndex(&index.IndexConfig{
		Name:      "test_idx",
		FieldPath: "field",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Insert data but don't analyze
	for i := 0; i < 10; i++ {
		idx.Insert(int64(i), string(rune(i)))
	}

	indexes := map[string]*index.Index{
		"test_idx": idx,
	}

	planner := NewQueryPlanner(indexes)

	// Query should still work, but use default cost estimation
	query := NewQuery(map[string]interface{}{
		"field": int64(5),
	})

	plan := planner.Plan(query)

	if !plan.UseIndex {
		t.Error("Expected to use index even without statistics")
	}

	// Cost should be default (not stats-based)
	// Since stats are stale, estimateCostWithStats should return original cost
	if plan.EstimatedCost <= 0 {
		t.Error("Expected positive estimated cost")
	}
}

func TestCostEstimationForRangeQueries(t *testing.T) {
	idx := index.NewIndex(&index.IndexConfig{
		Name:      "range_idx",
		FieldPath: "score",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Insert 1000 entries
	for i := 0; i < 1000; i++ {
		idx.Insert(int64(i), string(rune(i)))
	}
	idx.Analyze()

	indexes := map[string]*index.Index{
		"range_idx": idx,
	}

	planner := NewQueryPlanner(indexes)

	// Range query
	query := NewQuery(map[string]interface{}{
		"score": map[string]interface{}{"$gte": int64(500)},
	})

	plan := planner.Plan(query)

	if !plan.UseIndex {
		t.Error("Expected to use index for range query")
	}

	if plan.ScanType != ScanTypeIndexRange {
		t.Errorf("Expected INDEX_RANGE scan type, got %v", plan.ScanType)
	}

	// Cost should be reasonable (not too high, not too low)
	if plan.EstimatedCost < 20 || plan.EstimatedCost > 500 {
		t.Errorf("Expected cost between 20-500 for range query, got %d", plan.EstimatedCost)
	}
}

func TestSelectivityInfluencesPlanChoice(t *testing.T) {
	// Create indexes with extreme cardinality differences
	// High cardinality index (many unique values)
	uniqueIdx := index.NewIndex(&index.IndexConfig{
		Name:      "unique_idx",
		FieldPath: "unique_field",
		Type:      index.IndexTypeBTree,
		Unique:    true,
		Order:     32,
	})

	for i := 0; i < 1000; i++ {
		uniqueIdx.Insert(int64(i), string(rune(i)))
	}
	uniqueIdx.Analyze()

	// Low cardinality index (few unique values)
	nonSelectiveIdx := index.NewIndex(&index.IndexConfig{
		Name:      "non_selective_idx",
		FieldPath: "category",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Only 3 unique values (B-tree replaces duplicates, so we only get 3 entries)
	for i := 0; i < 3; i++ {
		nonSelectiveIdx.Insert(int64(i), string(rune(i)))
	}
	nonSelectiveIdx.Analyze()

	indexes := map[string]*index.Index{
		"unique_idx":         uniqueIdx,
		"non_selective_idx": nonSelectiveIdx,
	}

	planner := NewQueryPlanner(indexes)

	// Both indexes match the query
	query := NewQuery(map[string]interface{}{
		"unique_field": int64(500),
		"category":     int64(1),
	})

	plan := planner.Plan(query)

	// Should choose the unique index (higher cardinality)
	if plan.IndexName != "unique_idx" {
		t.Errorf("Expected to choose unique_idx, got %s", plan.IndexName)
	}

	// Verify cardinality difference
	uniqueStats := uniqueIdx.GetStatistics()
	nonSelectiveStats := nonSelectiveIdx.GetStatistics()

	uniqueCard := uniqueStats.Cardinality()
	nonSelectiveCard := nonSelectiveStats.Cardinality()

	if uniqueCard <= nonSelectiveCard {
		t.Errorf("Expected unique index to have higher cardinality: unique=%d, non_selective=%d",
			uniqueCard, nonSelectiveCard)
	}

	t.Logf("Cardinality comparison: unique=%d, non_selective=%d", uniqueCard, nonSelectiveCard)
	t.Logf("Chosen index: %s with cost %d", plan.IndexName, plan.EstimatedCost)
}
