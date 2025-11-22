package query

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/index"
)

func TestCoveredQueryDetection(t *testing.T) {
	// Create an index on "age" field
	idx := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	indexes := map[string]*index.Index{
		"age_idx": idx,
	}

	planner := NewQueryPlanner(indexes)

	tests := []struct {
		name       string
		filter     map[string]interface{}
		projection map[string]bool
		expectCovered bool
		reason     string
	}{
		{
			name:       "Covered - age and _id projection",
			filter:     map[string]interface{}{"age": map[string]interface{}{"$gte": 25}},
			projection: map[string]bool{"age": true, "_id": true},
			expectCovered: true,
			reason:     "Index provides both age and _id",
		},
		{
			name:       "Covered - age only projection",
			filter:     map[string]interface{}{"age": map[string]interface{}{"$gte": 25}},
			projection: map[string]bool{"age": true},
			expectCovered: true,
			reason:     "Index provides age",
		},
		{
			name:       "Covered - _id only projection",
			filter:     map[string]interface{}{"age": map[string]interface{}{"$gte": 25}},
			projection: map[string]bool{"_id": true},
			expectCovered: true,
			reason:     "Index provides _id",
		},
		{
			name:       "Not covered - no projection",
			filter:     map[string]interface{}{"age": map[string]interface{}{"$gte": 25}},
			projection: nil,
			expectCovered: false,
			reason:     "No projection means all fields needed",
		},
		{
			name:       "Not covered - additional field in projection",
			filter:     map[string]interface{}{"age": map[string]interface{}{"$gte": 25}},
			projection: map[string]bool{"age": true, "name": true},
			expectCovered: false,
			reason:     "Index doesn't provide 'name' field",
		},
		{
			name:       "Not covered - field not in index filter",
			filter:     map[string]interface{}{"name": "Alice"},
			projection: map[string]bool{"name": true},
			expectCovered: false,
			reason:     "No index on 'name' field",
		},
		{
			name:       "Not covered - exclusion projection",
			filter:     map[string]interface{}{"age": map[string]interface{}{"$gte": 25}},
			projection: map[string]bool{"age": false},
			expectCovered: false,
			reason:     "Exclusion projections not supported for covered queries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := NewQuery(tt.filter)
			if tt.projection != nil {
				query.WithProjection(tt.projection)
			}

			plan := planner.Plan(query)
			planner.DetectCoveredQuery(plan, query.GetProjection())

			if plan.IsCovered != tt.expectCovered {
				t.Errorf("Expected IsCovered=%v, got %v. Reason: %s",
					tt.expectCovered, plan.IsCovered, tt.reason)
			}
		})
	}
}

func TestCoveredQueryExecution(t *testing.T) {
	// Create test documents
	docs := []*document.Document{
		createTestDoc("1", "Alice", 25),
		createTestDoc("2", "Bob", 30),
		createTestDoc("3", "Charlie", 35),
		createTestDoc("4", "Diana", 28),
		createTestDoc("5", "Eve", 32),
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

	t.Run("Covered query - age >= 30", func(t *testing.T) {
		query := NewQuery(map[string]interface{}{
			"age": map[string]interface{}{"$gte": 30},
		})
		query.WithProjection(map[string]bool{"age": true, "_id": true})

		plan := planner.Plan(query)
		planner.DetectCoveredQuery(plan, query.GetProjection())

		if !plan.IsCovered {
			t.Fatal("Expected covered query, got not covered")
		}

		results, err := executor.ExecuteWithPlan(query, plan)
		if err != nil {
			t.Fatalf("Query execution failed: %v", err)
		}

		// Should return 3 documents (Bob:30, Eve:32, Charlie:35)
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		// Verify documents contain only age and _id
		for _, doc := range results {
			fields := doc.ToMap()
			for field := range fields {
				if field != "age" && field != "_id" {
					t.Errorf("Unexpected field '%s' in covered query result", field)
				}
			}

			// Verify age values
			if ageVal, exists := doc.Get("age"); exists {
				var age int64
				switch v := ageVal.(type) {
				case int64:
					age = v
				case int:
					age = int64(v)
				case float64:
					age = int64(v)
				default:
					t.Errorf("Unexpected age type: %T", ageVal)
					continue
				}
				if age < 30 {
					t.Errorf("Result contains age %d, expected >= 30", age)
				}
			} else {
				t.Error("Result missing 'age' field")
			}
		}
	})

	t.Run("Covered query - exact match age=28", func(t *testing.T) {
		query := NewQuery(map[string]interface{}{
			"age": 28,
		})
		query.WithProjection(map[string]bool{"age": true, "_id": true})

		plan := planner.Plan(query)
		planner.DetectCoveredQuery(plan, query.GetProjection())

		if !plan.IsCovered {
			t.Fatal("Expected covered query, got not covered")
		}

		results, err := executor.ExecuteWithPlan(query, plan)
		if err != nil {
			t.Fatalf("Query execution failed: %v", err)
		}

		// Should return 1 document (Diana:28)
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if len(results) > 0 {
			ageVal, _ := results[0].Get("age")
			var age int64
			switch v := ageVal.(type) {
			case int64:
				age = v
			case int:
				age = int64(v)
			case float64:
				age = int64(v)
			}
			if age != 28 {
				t.Errorf("Expected age=28, got %v (type %T)", ageVal, ageVal)
			}
		}
	})

	t.Run("Covered query - only _id projection", func(t *testing.T) {
		query := NewQuery(map[string]interface{}{
			"age": map[string]interface{}{"$lte": 28},
		})
		query.WithProjection(map[string]bool{"_id": true})

		plan := planner.Plan(query)
		planner.DetectCoveredQuery(plan, query.GetProjection())

		if !plan.IsCovered {
			t.Fatal("Expected covered query, got not covered")
		}

		results, err := executor.ExecuteWithPlan(query, plan)
		if err != nil {
			t.Fatalf("Query execution failed: %v", err)
		}

		// Should return 2 documents (Alice:25, Diana:28)
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		// Verify documents contain only _id
		for _, doc := range results {
			fields := doc.ToMap()
			if len(fields) > 1 {
				t.Errorf("Expected only _id field, got fields: %v", fields)
			}
			if _, exists := doc.Get("_id"); !exists {
				t.Error("Result missing '_id' field")
			}
		}
	})

	t.Run("Covered query - with limit", func(t *testing.T) {
		query := NewQuery(map[string]interface{}{
			"age": map[string]interface{}{"$gte": 25},
		})
		query.WithProjection(map[string]bool{"age": true})
		query.WithLimit(2)

		plan := planner.Plan(query)
		planner.DetectCoveredQuery(plan, query.GetProjection())

		if !plan.IsCovered {
			t.Fatal("Expected covered query, got not covered")
		}

		results, err := executor.ExecuteWithPlan(query, plan)
		if err != nil {
			t.Fatalf("Query execution failed: %v", err)
		}

		// Should return only 2 documents due to limit
		if len(results) != 2 {
			t.Errorf("Expected 2 results (limit), got %d", len(results))
		}
	})
}

func TestCoveredQueryExplain(t *testing.T) {
	idx := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	indexes := map[string]*index.Index{
		"age_idx": idx,
	}

	planner := NewQueryPlanner(indexes)

	query := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{"$gte": 30},
	})
	query.WithProjection(map[string]bool{"age": true, "_id": true})

	plan := planner.Plan(query)
	planner.DetectCoveredQuery(plan, query.GetProjection())

	explanation := plan.Explain()

	// Verify explanation includes covered query info
	if isCovered, exists := explanation["isCovered"]; !exists || !isCovered.(bool) {
		t.Error("Explanation should indicate covered query")
	}

	if note, exists := explanation["note"]; !exists {
		t.Error("Explanation should include note about covered query")
	} else if noteStr, ok := note.(string); !ok || noteStr == "" {
		t.Error("Covered query note should be non-empty string")
	}

	if indexedField, exists := explanation["indexedField"]; !exists || indexedField != "age" {
		t.Errorf("Explanation should show indexed field 'age', got %v", indexedField)
	}
}

// Helper function to create test documents
func createTestDoc(id, name string, age int) *document.Document {
	doc := document.NewDocument()
	doc.Set("_id", id)
	doc.Set("name", name)
	doc.Set("age", age)
	return doc
}
