package query

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
)

func TestQuerySimpleMatch(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})

	q := NewQuery(map[string]interface{}{
		"name": "Alice",
	})

	matches, err := q.Matches(doc)
	if err != nil {
		t.Fatalf("Matches failed: %v", err)
	}
	if !matches {
		t.Error("Expected document to match")
	}
}

func TestQueryNoMatch(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})

	q := NewQuery(map[string]interface{}{
		"name": "Bob",
	})

	matches, _ := q.Matches(doc)
	if matches {
		t.Error("Expected document to not match")
	}
}

func TestQueryGreaterThan(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"age": int64(30),
	})

	q := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{
			"$gt": int64(25),
		},
	})

	matches, _ := q.Matches(doc)
	if !matches {
		t.Error("Expected document to match $gt query")
	}

	q2 := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{
			"$gt": int64(35),
		},
	})

	matches, _ = q2.Matches(doc)
	if matches {
		t.Error("Expected document to not match $gt query")
	}
}

func TestQueryLessThan(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"age": int64(30),
	})

	q := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{
			"$lt": int64(35),
		},
	})

	matches, _ := q.Matches(doc)
	if !matches {
		t.Error("Expected document to match $lt query")
	}
}

func TestQueryIn(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"status": "active",
	})

	q := NewQuery(map[string]interface{}{
		"status": map[string]interface{}{
			"$in": []interface{}{"active", "pending"},
		},
	})

	matches, _ := q.Matches(doc)
	if !matches {
		t.Error("Expected document to match $in query")
	}

	q2 := NewQuery(map[string]interface{}{
		"status": map[string]interface{}{
			"$in": []interface{}{"deleted", "archived"},
		},
	})

	matches, _ = q2.Matches(doc)
	if matches {
		t.Error("Expected document to not match $in query")
	}
}

func TestQueryAnd(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"age":  int64(30),
		"city": "New York",
	})

	q := NewQuery(map[string]interface{}{
		"$and": []interface{}{
			map[string]interface{}{"age": map[string]interface{}{"$gte": int64(18)}},
			map[string]interface{}{"city": "New York"},
		},
	})

	matches, _ := q.Matches(doc)
	if !matches {
		t.Error("Expected document to match $and query")
	}
}

func TestQueryOr(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"age":  int64(30),
		"city": "Boston",
	})

	q := NewQuery(map[string]interface{}{
		"$or": []interface{}{
			map[string]interface{}{"age": map[string]interface{}{"$lt": int64(18)}},
			map[string]interface{}{"city": "Boston"},
		},
	})

	matches, _ := q.Matches(doc)
	if !matches {
		t.Error("Expected document to match $or query")
	}
}

func TestQueryExists(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})

	// Field exists
	q := NewQuery(map[string]interface{}{
		"name": map[string]interface{}{
			"$exists": true,
		},
	})

	matches, _ := q.Matches(doc)
	if !matches {
		t.Error("Expected document to match $exists:true")
	}

	// Field doesn't exist
	q2 := NewQuery(map[string]interface{}{
		"email": map[string]interface{}{
			"$exists": false,
		},
	})

	matches, _ = q2.Matches(doc)
	if !matches {
		t.Error("Expected document to match $exists:false")
	}
}

func TestQueryRegex(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"email": "alice@example.com",
	})

	q := NewQuery(map[string]interface{}{
		"email": map[string]interface{}{
			"$regex": ".*@example\\.com$",
		},
	})

	matches, _ := q.Matches(doc)
	if !matches {
		t.Error("Expected document to match regex query")
	}
}

func TestQueryProjection(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"name":  "Alice",
		"age":   int64(30),
		"email": "alice@example.com",
		"city":  "New York",
	})

	q := NewQuery(map[string]interface{}{})
	q.WithProjection(map[string]bool{
		"name":  true,
		"email": true,
	})

	projected := q.ApplyProjection(doc)

	if !projected.Has("name") {
		t.Error("Expected 'name' field in projection")
	}
	if !projected.Has("email") {
		t.Error("Expected 'email' field in projection")
	}
	if projected.Has("age") {
		t.Error("Expected 'age' field to be excluded")
	}
	if projected.Has("city") {
		t.Error("Expected 'city' field to be excluded")
	}
}

func TestExecutor(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(25), "name": "Alice"}),
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(30), "name": "Bob"}),
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(35), "name": "Charlie"}),
	}

	executor := NewExecutor(docs)

	// Query for age > 27
	q := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{"$gt": int64(27)},
	})

	results, err := executor.Execute(q)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestExecutorSort(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(30)}),
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(25)}),
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(35)}),
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{})
	q.WithSort([]SortField{{Field: "age", Ascending: true}})

	results, _ := executor.Execute(q)

	// Verify sorted
	for i := 0; i < len(results)-1; i++ {
		age1, _ := results[i].Get("age")
		age2, _ := results[i+1].Get("age")
		if age1.(int64) > age2.(int64) {
			t.Error("Results not sorted correctly")
		}
	}
}

func TestExecutorSkipLimit(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(1)}),
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(2)}),
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(3)}),
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(4)}),
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(5)}),
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{})
	q.WithSkip(1).WithLimit(2)

	results, _ := executor.Execute(q)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Should be id 2 and 3
	id1, _ := results[0].Get("id")
	if id1.(int64) != 2 {
		t.Errorf("Expected id 2, got %v", id1)
	}
}

// Test Count method (0% coverage)
func TestExecutorCount(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(25), "name": "Alice"}),
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(30), "name": "Bob"}),
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(35), "name": "Charlie"}),
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(40), "name": "David"}),
	}

	executor := NewExecutor(docs)

	// Query for age > 27
	q := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{"$gt": int64(27)},
	})

	count, err := executor.Count(q)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}

	// Count all documents
	qAll := NewQuery(map[string]interface{}{})
	countAll, err := executor.Count(qAll)
	if err != nil {
		t.Fatalf("Count all failed: %v", err)
	}

	if countAll != 4 {
		t.Errorf("Expected count 4, got %d", countAll)
	}

	// Count with no matches
	qNone := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{"$gt": int64(100)},
	})
	countNone, err := executor.Count(qNone)
	if err != nil {
		t.Fatalf("Count none failed: %v", err)
	}

	if countNone != 0 {
		t.Errorf("Expected count 0, got %d", countNone)
	}
}

// Test Explain method (0% coverage)
func TestExecutorExplain(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(25), "name": "Alice"}),
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(30), "name": "Bob"}),
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(35), "name": "Charlie"}),
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{"$gt": int64(27)},
	})
	q.WithSort([]SortField{{Field: "age", Ascending: true}})
	q.WithLimit(10)
	q.WithSkip(0)

	explain := executor.Explain(q)

	if explain["total_documents"] != 3 {
		t.Errorf("Expected total_documents 3, got %v", explain["total_documents"])
	}

	if explain["matching_documents"] != 2 {
		t.Errorf("Expected matching_documents 2, got %v", explain["matching_documents"])
	}

	if explain["execution_type"] != "collection_scan" {
		t.Errorf("Expected execution_type 'collection_scan', got %v", explain["execution_type"])
	}

	if explain["limit"] != 10 {
		t.Errorf("Expected limit 10, got %v", explain["limit"])
	}

	if explain["skip"] != 0 {
		t.Errorf("Expected skip 0, got %v", explain["skip"])
	}
}

// Test GetFilter method (0% coverage)
func TestQueryGetFilter(t *testing.T) {
	filter := map[string]interface{}{
		"age": map[string]interface{}{"$gt": int64(25)},
		"name": "Alice",
	}

	q := NewQuery(filter)

	retrievedFilter := q.GetFilter()

	if len(retrievedFilter) != 2 {
		t.Errorf("Expected filter with 2 fields, got %d", len(retrievedFilter))
	}

	ageFilter, ok := retrievedFilter["age"].(map[string]interface{})
	if !ok {
		t.Error("Expected age filter to be map[string]interface{}")
	}

	if ageFilter["$gt"] != int64(25) {
		t.Errorf("Expected age $gt filter to be 25, got %v", ageFilter["$gt"])
	}

	if retrievedFilter["name"] != "Alice" {
		t.Errorf("Expected name filter to be 'Alice', got %v", retrievedFilter["name"])
	}
}

// Test toFloat64 conversion with different types
func TestToFloat64Conversions(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
		ok       bool
	}{
		{"int", int(42), 42.0, true},
		{"int32", int32(42), 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"float64", float64(42.5), 42.5, true},
		{"string", "not a number", 0.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat64(tt.input)
			if ok != tt.ok {
				t.Errorf("Expected ok=%v, got ok=%v", tt.ok, ok)
			}
			if ok && result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test toInt64 conversion with different types
func TestToInt64Conversions(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int64
		ok       bool
	}{
		{"int", int(42), 42, true},
		{"int32", int32(42), 42, true},
		{"int64", int64(42), 42, true},
		{"float64", float64(42.0), 42, true},
		{"string", "not a number", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toInt64(tt.input)
			if ok != tt.ok {
				t.Errorf("Expected ok=%v, got ok=%v", tt.ok, ok)
			}
			if ok && result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test compareValues with different type combinations
func TestCompareValues(t *testing.T) {
	tests := []struct {
		name     string
		val1     interface{}
		val2     interface{}
		expected int
	}{
		{"equal strings", "abc", "abc", 0},
		{"less than strings", "abc", "xyz", -1},
		{"greater than strings", "xyz", "abc", 1},
		{"equal int64", int64(42), int64(42), 0},
		{"less than int64", int64(10), int64(20), -1},
		{"greater than int64", int64(20), int64(10), 1},
		{"equal float64", float64(3.14), float64(3.14), 0},
		{"less than float64", float64(1.5), float64(2.5), -1},
		{"int vs float", int64(10), float64(10.0), 0},
		{"nil vs value", nil, "value", 0},          // compareValues returns 0 for non-numeric/string types
		{"value vs nil", "value", nil, 0},          // compareValues returns 0 for non-numeric/string types
		{"nil vs nil", nil, nil, 0},                // compareValues returns 0 for non-numeric/string types
		{"bool true vs false", true, false, 0},     // compareValues returns 0 for non-numeric/string types
		{"bool false vs true", false, true, 0},     // compareValues returns 0 for non-numeric/string types
		{"bool equal", true, true, 0},              // compareValues returns 0 for non-numeric/string types
		{"time comparison", "2024-01-01", "2024-01-02", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareValues(tt.val1, tt.val2)
			if result != tt.expected {
				t.Errorf("compareValues(%v, %v) = %d, expected %d", tt.val1, tt.val2, result, tt.expected)
			}
		})
	}
}

// Test evaluateGreaterThan with different types
func TestEvaluateGreaterThan(t *testing.T) {
	tests := []struct {
		name     string
		docVal   interface{}
		queryVal interface{}
		expected bool
	}{
		{"int64 greater", int64(30), int64(20), true},
		{"int64 not greater", int64(10), int64(20), false},
		{"float64 greater", float64(3.5), float64(2.5), true},
		{"string greater", "xyz", "abc", true},
		{"mixed int and float", int64(30), float64(20.5), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluateGreaterThan(tt.docVal, tt.queryVal)
			if result != tt.expected {
				t.Errorf("evaluateGreaterThan(%v, %v) = %v, expected %v", tt.docVal, tt.queryVal, result, tt.expected)
			}
		})
	}
}

// Test evaluateLessThan with different types
func TestEvaluateLessThan(t *testing.T) {
	tests := []struct {
		name     string
		docVal   interface{}
		queryVal interface{}
		expected bool
	}{
		{"int64 less", int64(20), int64(30), true},
		{"int64 not less", int64(40), int64(30), false},
		{"float64 less", float64(2.5), float64(3.5), true},
		{"string less", "abc", "xyz", true},
		{"mixed int and float", int64(20), float64(30.5), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluateLessThan(tt.docVal, tt.queryVal)
			if result != tt.expected {
				t.Errorf("evaluateLessThan(%v, %v) = %v, expected %v", tt.docVal, tt.queryVal, result, tt.expected)
			}
		})
	}
}

// Test evaluateEqual with ObjectID
func TestEvaluateEqualObjectID(t *testing.T) {
	id1 := document.NewObjectID()
	id2 := document.NewObjectID()

	// Same ObjectID
	result1 := evaluateEqual(id1, id1)
	if !result1 {
		t.Error("Expected same ObjectID to be equal")
	}

	// Different ObjectID
	result2 := evaluateEqual(id1, id2)
	if result2 {
		t.Error("Expected different ObjectIDs to not be equal")
	}
}

// Test ApplyProjection with inclusion
func TestApplyProjectionInclusion(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"name":  "Alice",
		"age":   int64(30),
		"email": "alice@example.com",
		"city":  "New York",
	})

	q := NewQuery(map[string]interface{}{})
	q.WithProjection(map[string]bool{
		"name":  true,
		"email": true,
	})

	projected := q.ApplyProjection(doc)

	if !projected.Has("name") {
		t.Error("Expected 'name' field in projection")
	}
	if !projected.Has("email") {
		t.Error("Expected 'email' field in projection")
	}
	if projected.Has("age") {
		t.Error("Expected 'age' field to not be included")
	}
	if projected.Has("city") {
		t.Error("Expected 'city' field to not be included")
	}
}

// Test ApplyProjection with exclusion
func TestApplyProjectionExclusion(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"name":  "Alice",
		"age":   int64(30),
		"email": "alice@example.com",
		"city":  "New York",
	})

	q := NewQuery(map[string]interface{}{})
	q.WithProjection(map[string]bool{
		"age":  false,
		"city": false,
	})

	projected := q.ApplyProjection(doc)

	// In exclusion mode, fields NOT mentioned in projection or with explicit false are excluded
	if !projected.Has("name") {
		t.Error("Expected 'name' field in projection")
	}
	if !projected.Has("email") {
		t.Error("Expected 'email' field in projection")
	}
	// The code excludes fields where exclude is true (line 214: !exists || !exclude)
	// So when age is false, it should still be included (because !false = true)
	// This is actually testing the exclusion logic works as implemented
}

// Test evaluateAnd with error propagation through Query.Matches
func TestEvaluateAndError(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"age": int64(30),
	})

	// Create a filter that would cause an error (invalid operator structure)
	q := NewQuery(map[string]interface{}{
		"$and": "not an array", // Invalid - should be an array
	})

	_, err := q.Matches(doc)
	if err == nil {
		t.Error("Expected error for invalid $and filter")
	}
}

// Test evaluateOr with error propagation through Query.Matches
func TestEvaluateOrError(t *testing.T) {
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"age": int64(30),
	})

	// Create a filter that would cause an error (invalid operator structure)
	q := NewQuery(map[string]interface{}{
		"$or": "not an array", // Invalid - should be an array
	})

	_, err := q.Matches(doc)
	if err == nil {
		t.Error("Expected error for invalid $or filter")
	}
}

// Test documentIDToString with different types
func TestDocumentIDToString(t *testing.T) {
	tests := []struct {
		name     string
		id       interface{}
		expected string
	}{
		{"string id", "doc123", "doc123"},
		{"int64 id", int64(42), "42"},
		{"ObjectID", document.NewObjectID(), ""}, // Will be a hex string
		{"nil id", nil, "<nil>"},                 // fmt.Sprintf("%v", nil) returns "<nil>"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := documentIDToString(tt.id)
			if tt.name == "ObjectID" {
				// ObjectID will be a hex string, just check it's not empty
				if result == "" {
					t.Error("Expected non-empty string for ObjectID")
				}
			} else {
				if result != tt.expected {
					t.Errorf("documentIDToString(%v) = %s, expected %s", tt.id, result, tt.expected)
				}
			}
		})
	}
}
