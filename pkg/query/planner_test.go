package query

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/index"
)

func TestQueryPlannerExactMatch(t *testing.T) {
	// Create test index
	idx := index.NewIndex(&index.IndexConfig{
		Name:      "age_1",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	indexes := map[string]*index.Index{
		"age_1": idx,
	}

	// Create query planner
	planner := NewQueryPlanner(indexes)

	// Test query with exact match
	filter := map[string]interface{}{
		"age": 30,
	}
	q := NewQuery(filter)

	plan := planner.Plan(q)

	// Verify plan uses index
	if !plan.UseIndex {
		t.Error("Expected plan to use index")
	}

	if plan.IndexName != "age_1" {
		t.Errorf("Expected index name 'age_1', got '%s'", plan.IndexName)
	}

	if plan.ScanType != ScanTypeIndexExact {
		t.Error("Expected exact scan type")
	}

	if plan.ScanKey != 30 {
		t.Errorf("Expected scan key 30, got %v", plan.ScanKey)
	}

	if plan.EstimatedCost >= 50 {
		t.Errorf("Expected low cost for exact match, got %d", plan.EstimatedCost)
	}
}

func TestQueryPlannerRangeScan(t *testing.T) {
	// Create test index
	idx := index.NewIndex(&index.IndexConfig{
		Name:      "age_1",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	indexes := map[string]*index.Index{
		"age_1": idx,
	}

	planner := NewQueryPlanner(indexes)

	// Test query with range
	filter := map[string]interface{}{
		"age": map[string]interface{}{
			"$gte": 25,
			"$lte": 40,
		},
	}
	q := NewQuery(filter)

	plan := planner.Plan(q)

	// Verify plan uses index
	if !plan.UseIndex {
		t.Error("Expected plan to use index for range query")
	}

	if plan.ScanType != ScanTypeIndexRange {
		t.Error("Expected range scan type")
	}

	if plan.ScanStart != 25 {
		t.Errorf("Expected scan start 25, got %v", plan.ScanStart)
	}

	if plan.ScanEnd != 40 {
		t.Errorf("Expected scan end 40, got %v", plan.ScanEnd)
	}

	if plan.EstimatedCost >= 100 {
		t.Errorf("Expected low cost for range scan, got %d", plan.EstimatedCost)
	}
}

func TestQueryPlannerNoIndex(t *testing.T) {
	// Empty indexes
	indexes := map[string]*index.Index{}

	planner := NewQueryPlanner(indexes)

	// Query on field without index
	filter := map[string]interface{}{
		"name": "John",
	}
	q := NewQuery(filter)

	plan := planner.Plan(q)

	// Verify plan does NOT use index
	if plan.UseIndex {
		t.Error("Expected plan to NOT use index when no suitable index exists")
	}

	if plan.ScanType != ScanTypeCollection {
		t.Error("Expected collection scan type")
	}

	if plan.EstimatedCost < 1000 {
		t.Errorf("Expected high cost for collection scan, got %d", plan.EstimatedCost)
	}
}

func TestQueryPlannerWithMultipleIndexes(t *testing.T) {
	// Create multiple indexes
	ageIdx := index.NewIndex(&index.IndexConfig{
		Name:      "age_1",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	nameIdx := index.NewIndex(&index.IndexConfig{
		Name:      "name_1",
		FieldPath: "name",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	indexes := map[string]*index.Index{
		"age_1":  ageIdx,
		"name_1": nameIdx,
	}

	planner := NewQueryPlanner(indexes)

	// Query that could use either index
	filter := map[string]interface{}{
		"age":  30, // Exact match - cost 10
		"name": map[string]interface{}{"$gte": "A"}, // Range - cost 50
	}
	q := NewQuery(filter)

	plan := planner.Plan(q)

	// Should choose age index (exact match, lower cost)
	if !plan.UseIndex {
		t.Error("Expected plan to use index")
	}

	if plan.IndexName != "age_1" {
		t.Errorf("Expected planner to choose 'age_1' (lower cost), got '%s'", plan.IndexName)
	}

	// Should note that 'name' still needs filtering
	if len(plan.FilterSteps) == 0 {
		t.Error("Expected remaining filter steps for 'name' field")
	}
}

func TestQueryPlanExplain(t *testing.T) {
	idx := index.NewIndex(&index.IndexConfig{
		Name:      "age_1",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	indexes := map[string]*index.Index{
		"age_1": idx,
	}

	planner := NewQueryPlanner(indexes)

	filter := map[string]interface{}{
		"age": map[string]interface{}{
			"$gte": 20,
			"$lt":  30,
		},
	}
	q := NewQuery(filter)

	plan := planner.Plan(q)
	explain := plan.Explain()

	// Check explain output
	if explain["useIndex"] != true {
		t.Error("Explain should show useIndex=true")
	}

	if explain["scanType"] != "INDEX_RANGE" {
		t.Errorf("Expected scanType='INDEX_RANGE', got '%v'", explain["scanType"])
	}

	if explain["indexName"] != "age_1" {
		t.Error("Explain should include index name")
	}
}
