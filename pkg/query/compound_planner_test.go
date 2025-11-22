package query

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/index"
)

func TestCompoundIndexPlanning(t *testing.T) {
	t.Run("Full match on compound index", func(t *testing.T) {
		// Create compound index on city and age
		idx := index.NewIndex(&index.IndexConfig{
			Name:       "city_age_1",
			FieldPaths: []string{"city", "age"},
			Type:       index.IndexTypeBTree,
			Unique:     false,
			Order:      32,
		})

		// Insert some data
		idx.Insert(index.NewCompositeKey("NYC", int64(25)), "id1")
		idx.Insert(index.NewCompositeKey("NYC", int64(30)), "id2")
		idx.Insert(index.NewCompositeKey("Boston", int64(25)), "id3")
		idx.Analyze()

		indexes := map[string]*index.Index{
			"city_age_1": idx,
		}

		planner := NewQueryPlanner(indexes)

		// Query matching both fields
		query := NewQuery(map[string]interface{}{
			"city": "NYC",
			"age":  int64(30),
		})

		plan := planner.Plan(query)

		if !plan.UseIndex {
			t.Error("Expected to use compound index")
		}

		if plan.IndexName != "city_age_1" {
			t.Errorf("Expected to use city_age_1, got %s", plan.IndexName)
		}

		if plan.ScanType != ScanTypeIndexExact {
			t.Errorf("Expected exact scan, got %v", plan.ScanType)
		}

		// Verify composite key
		compositeKey, ok := plan.ScanKey.(*index.CompositeKey)
		if !ok {
			t.Fatal("Expected CompositeKey as ScanKey")
		}

		if len(compositeKey.Values) != 2 {
			t.Errorf("Expected 2 values in composite key, got %d", len(compositeKey.Values))
		}

		if compositeKey.Values[0] != "NYC" || compositeKey.Values[1] != int64(30) {
			t.Errorf("Expected [NYC, 30], got %v", compositeKey.Values)
		}
	})

	t.Run("Prefix match on compound index", func(t *testing.T) {
		// Create compound index on city, age, salary
		idx := index.NewIndex(&index.IndexConfig{
			Name:       "city_age_salary_1",
			FieldPaths: []string{"city", "age", "salary"},
			Type:       index.IndexTypeBTree,
			Unique:     false,
			Order:      32,
		})

		idx.Insert(index.NewCompositeKey("NYC", int64(25), int64(50000)), "id1")
		idx.Analyze()

		indexes := map[string]*index.Index{
			"city_age_salary_1": idx,
		}

		planner := NewQueryPlanner(indexes)

		// Query matching only first field (prefix)
		query := NewQuery(map[string]interface{}{
			"city": "NYC",
		})

		plan := planner.Plan(query)

		if !plan.UseIndex {
			t.Error("Expected to use compound index with prefix match")
		}

		if plan.IndexName != "city_age_salary_1" {
			t.Errorf("Expected to use city_age_salary_1, got %s", plan.IndexName)
		}

		// Prefix match should use range scan with PrefixKey
		if plan.ScanType != ScanTypeIndexRange {
			t.Errorf("Expected ScanTypeIndexRange for prefix match, got %v", plan.ScanType)
		}

		if plan.PrefixKey == nil {
			t.Fatal("Expected PrefixKey to be set for prefix match")
		}

		// Verify prefix key has only 1 value
		if len(plan.PrefixKey.Values) != 1 {
			t.Errorf("Expected 1 value in prefix key, got %d", len(plan.PrefixKey.Values))
		}

		if plan.PrefixKey.Values[0] != "NYC" {
			t.Errorf("Expected [NYC], got %v", plan.PrefixKey.Values)
		}
	})

	t.Run("Cannot use compound index without first field", func(t *testing.T) {
		// Create compound index on city and age
		idx := index.NewIndex(&index.IndexConfig{
			Name:       "city_age_1",
			FieldPaths: []string{"city", "age"},
			Type:       index.IndexTypeBTree,
			Unique:     false,
			Order:      32,
		})

		idx.Insert(index.NewCompositeKey("NYC", int64(25)), "id1")
		idx.Analyze()

		indexes := map[string]*index.Index{
			"city_age_1": idx,
		}

		planner := NewQueryPlanner(indexes)

		// Query matching only second field (not a valid prefix)
		query := NewQuery(map[string]interface{}{
			"age": int64(25),
		})

		plan := planner.Plan(query)

		// Should fall back to collection scan since compound index can't be used
		if plan.UseIndex {
			t.Error("Should not use compound index when first field is missing")
		}

		if plan.ScanType != ScanTypeCollection {
			t.Errorf("Expected collection scan, got %v", plan.ScanType)
		}
	})

	t.Run("Choose compound index over single-field index", func(t *testing.T) {
		// Create both compound and single-field indexes
		compoundIdx := index.NewIndex(&index.IndexConfig{
			Name:       "city_age_1",
			FieldPaths: []string{"city", "age"},
			Type:       index.IndexTypeBTree,
			Unique:     false,
			Order:      32,
		})

		singleIdx := index.NewIndex(&index.IndexConfig{
			Name:      "city_1",
			FieldPath: "city",
			Type:      index.IndexTypeBTree,
			Unique:    false,
			Order:     32,
		})

		// Insert same data into both
		compoundIdx.Insert(index.NewCompositeKey("NYC", int64(25)), "id1")
		compoundIdx.Insert(index.NewCompositeKey("NYC", int64(30)), "id2")
		compoundIdx.Insert(index.NewCompositeKey("Boston", int64(25)), "id3")
		compoundIdx.Analyze()

		singleIdx.Insert("NYC", "id1")
		singleIdx.Insert("NYC", "id2")
		singleIdx.Insert("Boston", "id3")
		singleIdx.Analyze()

		indexes := map[string]*index.Index{
			"city_age_1": compoundIdx,
			"city_1":     singleIdx,
		}

		planner := NewQueryPlanner(indexes)

		// Query matching both fields - should prefer compound index
		query := NewQuery(map[string]interface{}{
			"city": "NYC",
			"age":  int64(30),
		})

		plan := planner.Plan(query)

		if !plan.UseIndex {
			t.Error("Expected to use an index")
		}

		// Compound index should be chosen because it matches more fields
		if plan.IndexName != "city_age_1" {
			t.Errorf("Expected compound index city_age_1, got %s", plan.IndexName)
		}

		// Should have no remaining filter steps (both fields covered by index)
		if len(plan.FilterSteps) != 0 {
			t.Errorf("Expected no remaining filters, got %v", plan.FilterSteps)
		}
	})

	t.Run("Compound index with three fields", func(t *testing.T) {
		idx := index.NewIndex(&index.IndexConfig{
			Name:       "city_age_salary_1",
			FieldPaths: []string{"city", "age", "salary"},
			Type:       index.IndexTypeBTree,
			Unique:     false,
			Order:      32,
		})

		idx.Insert(index.NewCompositeKey("NYC", int64(25), int64(50000)), "id1")
		idx.Insert(index.NewCompositeKey("NYC", int64(30), int64(60000)), "id2")
		idx.Analyze()

		indexes := map[string]*index.Index{
			"city_age_salary_1": idx,
		}

		planner := NewQueryPlanner(indexes)

		// Test matching first two fields (prefix match)
		query := NewQuery(map[string]interface{}{
			"city": "NYC",
			"age":  int64(25),
		})

		plan := planner.Plan(query)

		if !plan.UseIndex {
			t.Error("Expected to use compound index")
		}

		// This is a prefix match (2 out of 3 fields), should use PrefixKey
		if plan.PrefixKey == nil {
			t.Fatal("Expected PrefixKey to be set for 2-field prefix match")
		}

		if len(plan.PrefixKey.Values) != 2 {
			t.Errorf("Expected 2-field prefix, got %d", len(plan.PrefixKey.Values))
		}
	})
}

func TestCompoundIndexExplain(t *testing.T) {
	idx := index.NewIndex(&index.IndexConfig{
		Name:       "city_age_1",
		FieldPaths: []string{"city", "age"},
		Type:       index.IndexTypeBTree,
		Unique:     false,
		Order:      32,
	})

	idx.Insert(index.NewCompositeKey("NYC", int64(25)), "id1")
	idx.Analyze()

	indexes := map[string]*index.Index{
		"city_age_1": idx,
	}

	planner := NewQueryPlanner(indexes)

	query := NewQuery(map[string]interface{}{
		"city": "NYC",
		"age":  int64(25),
	})

	plan := planner.Plan(query)
	explanation := plan.Explain()

	t.Logf("Compound index query plan:")
	t.Logf("  Use Index: %v", explanation["useIndex"])
	t.Logf("  Index Name: %v", explanation["indexName"])
	t.Logf("  Scan Type: %v", explanation["scanType"])
	t.Logf("  Scan Key: %v", explanation["scanKey"])
	t.Logf("  Estimated Cost: %v", explanation["estimatedCost"])

	if explanation["useIndex"] != true {
		t.Error("Expected useIndex to be true")
	}

	if explanation["indexName"] != "city_age_1" {
		t.Errorf("Expected city_age_1, got %v", explanation["indexName"])
	}

	if explanation["scanType"] != "INDEX_EXACT" {
		t.Errorf("Expected INDEX_EXACT, got %v", explanation["scanType"])
	}
}
