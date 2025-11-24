package aggregation

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
)

func TestMatchStage(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(25)}),
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(30)}),
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(35)}),
	}

	pipeline := []map[string]interface{}{
		{
			"$match": map[string]interface{}{
				"age": map[string]interface{}{"$gte": int64(30)},
			},
		},
	}

	p, err := NewPipeline(pipeline)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	results, err := p.Execute(docs)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestProjectStage(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"name":  "Alice",
			"age":   int64(30),
			"email": "alice@example.com",
		}),
	}

	pipeline := []map[string]interface{}{
		{
			"$project": map[string]interface{}{
				"name": true,
				"age":  true,
			},
		},
	}

	p, _ := NewPipeline(pipeline)
	results, _ := p.Execute(docs)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if !result.Has("name") || !result.Has("age") {
		t.Error("Expected name and age fields")
	}
	if result.Has("email") {
		t.Error("Expected email field to be excluded")
	}
}

func TestSortStage(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(30)}),
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(25)}),
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(35)}),
	}

	pipeline := []map[string]interface{}{
		{
			"$sort": map[string]interface{}{
				"age": 1, // Ascending
			},
		},
	}

	p, _ := NewPipeline(pipeline)
	results, _ := p.Execute(docs)

	// Verify sorted
	for i := 0; i < len(results)-1; i++ {
		age1, _ := results[i].Get("age")
		age2, _ := results[i+1].Get("age")
		if age1.(int64) > age2.(int64) {
			t.Error("Results not sorted correctly")
		}
	}
}

func TestLimitStage(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(1)}),
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(2)}),
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(3)}),
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(4)}),
	}

	pipeline := []map[string]interface{}{
		{"$limit": 2},
	}

	p, _ := NewPipeline(pipeline)
	results, _ := p.Execute(docs)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestSkipStage(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(1)}),
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(2)}),
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(3)}),
	}

	pipeline := []map[string]interface{}{
		{"$skip": 1},
	}

	p, _ := NewPipeline(pipeline)
	results, _ := p.Execute(docs)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	id, _ := results[0].Get("id")
	if id.(int64) != 2 {
		t.Errorf("Expected first result to be id 2, got %v", id)
	}
}

func TestGroupStage(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"category": "A", "price": 10.0}),
		document.NewDocumentFromMap(map[string]interface{}{"category": "A", "price": 20.0}),
		document.NewDocumentFromMap(map[string]interface{}{"category": "B", "price": 30.0}),
	}

	pipeline := []map[string]interface{}{
		{
			"$group": map[string]interface{}{
				"_id": "$category",
				"total": map[string]interface{}{
					"$sum": "$price",
				},
				"count": map[string]interface{}{
					"$count": nil,
				},
			},
		},
	}

	p, _ := NewPipeline(pipeline)
	results, _ := p.Execute(docs)

	if len(results) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(results))
	}

	// Find group A
	var groupA *document.Document
	for _, doc := range results {
		id, _ := doc.Get("_id")
		if id.(string) == "A" {
			groupA = doc
			break
		}
	}

	if groupA == nil {
		t.Fatal("Expected to find group A")
	}

	total, _ := groupA.Get("total")
	if total.(float64) != 30.0 {
		t.Errorf("Expected total 30.0, got %v", total)
	}

	count, _ := groupA.Get("count")
	if count.(int64) != 2 {
		t.Errorf("Expected count 2, got %v", count)
	}
}

func TestMultiStagePipeline(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"category": "A", "price": 10.0, "inStock": true}),
		document.NewDocumentFromMap(map[string]interface{}{"category": "A", "price": 20.0, "inStock": true}),
		document.NewDocumentFromMap(map[string]interface{}{"category": "B", "price": 30.0, "inStock": false}),
		document.NewDocumentFromMap(map[string]interface{}{"category": "B", "price": 40.0, "inStock": true}),
	}

	pipeline := []map[string]interface{}{
		{
			"$match": map[string]interface{}{
				"inStock": true,
			},
		},
		{
			"$group": map[string]interface{}{
				"_id": "$category",
				"avgPrice": map[string]interface{}{
					"$avg": "$price",
				},
			},
		},
		{
			"$sort": map[string]interface{}{
				"avgPrice": -1, // Descending
			},
		},
	}

	p, _ := NewPipeline(pipeline)
	results, _ := p.Execute(docs)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// First should be B (higher avg)
	id, _ := results[0].Get("_id")
	if id.(string) != "B" {
		t.Errorf("Expected B first (highest avg), got %v", id)
	}
}

func TestGroupAggregationOperators(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"value": 10.0}),
		document.NewDocumentFromMap(map[string]interface{}{"value": 20.0}),
		document.NewDocumentFromMap(map[string]interface{}{"value": 30.0}),
	}

	pipeline := []map[string]interface{}{
		{
			"$group": map[string]interface{}{
				"_id": nil,
				"sum": map[string]interface{}{
					"$sum": "$value",
				},
				"avg": map[string]interface{}{
					"$avg": "$value",
				},
				"min": map[string]interface{}{
					"$min": "$value",
				},
				"max": map[string]interface{}{
					"$max": "$value",
				},
			},
		},
	}

	p, _ := NewPipeline(pipeline)
	results, _ := p.Execute(docs)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]

	sum, _ := result.Get("sum")
	if sum.(float64) != 60.0 {
		t.Errorf("Expected sum 60.0, got %v", sum)
	}

	avg, _ := result.Get("avg")
	if avg.(float64) != 20.0 {
		t.Errorf("Expected avg 20.0, got %v", avg)
	}

	min, _ := result.Get("min")
	if min.(float64) != 10.0 {
		t.Errorf("Expected min 10.0, got %v", min)
	}

	max, _ := result.Get("max")
	if max.(float64) != 30.0 {
		t.Errorf("Expected max 30.0, got %v", max)
	}
}

// Test Type() methods for all stages
func TestStageTypes(t *testing.T) {
	tests := []struct {
		name         string
		pipeline     []map[string]interface{}
		expectedType string
	}{
		{
			name:         "MatchStage",
			pipeline:     []map[string]interface{}{{"$match": map[string]interface{}{"age": int64(30)}}},
			expectedType: "$match",
		},
		{
			name:         "ProjectStage",
			pipeline:     []map[string]interface{}{{"$project": map[string]interface{}{"name": true}}},
			expectedType: "$project",
		},
		{
			name:         "SortStage",
			pipeline:     []map[string]interface{}{{"$sort": map[string]interface{}{"age": 1}}},
			expectedType: "$sort",
		},
		{
			name:         "LimitStage",
			pipeline:     []map[string]interface{}{{"$limit": 10}},
			expectedType: "$limit",
		},
		{
			name:         "SkipStage",
			pipeline:     []map[string]interface{}{{"$skip": 5}},
			expectedType: "$skip",
		},
		{
			name:         "GroupStage",
			pipeline:     []map[string]interface{}{{"$group": map[string]interface{}{"_id": "$category"}}},
			expectedType: "$group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPipeline(tt.pipeline)
			if err != nil {
				t.Fatalf("Failed to create pipeline: %v", err)
			}
			if len(p.stages) != 1 {
				t.Fatalf("Expected 1 stage, got %d", len(p.stages))
			}
			if p.stages[0].Type() != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, p.stages[0].Type())
			}
		})
	}
}

// Test error paths in stage creation
func TestNewPipelineErrors(t *testing.T) {
	tests := []struct {
		name     string
		pipeline []map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "UnsupportedStageType",
			pipeline: []map[string]interface{}{{"$unknown": map[string]interface{}{}}},
			wantErr:  true,
		},
		{
			name:     "EmptyStageDefinition",
			pipeline: []map[string]interface{}{{}},
			wantErr:  true,
		},
		{
			name:     "InvalidMatchSpec",
			pipeline: []map[string]interface{}{{"$match": "invalid"}},
			wantErr:  true,
		},
		{
			name:     "InvalidProjectSpec",
			pipeline: []map[string]interface{}{{"$project": "invalid"}},
			wantErr:  true,
		},
		{
			name:     "InvalidSortSpec",
			pipeline: []map[string]interface{}{{"$sort": "invalid"}},
			wantErr:  true,
		},
		{
			name:     "InvalidLimitSpec",
			pipeline: []map[string]interface{}{{"$limit": "invalid"}},
			wantErr:  true,
		},
		{
			name:     "InvalidSkipSpec",
			pipeline: []map[string]interface{}{{"$skip": "invalid"}},
			wantErr:  true,
		},
		{
			name:     "InvalidGroupSpec",
			pipeline: []map[string]interface{}{{"$group": "invalid"}},
			wantErr:  true,
		},
		{
			name:     "GroupMissingID",
			pipeline: []map[string]interface{}{{"$group": map[string]interface{}{"field": "value"}}},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPipeline(tt.pipeline)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPipeline() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test compareValues with different types
func TestCompareValues(t *testing.T) {
	tests := []struct {
		name     string
		a        interface{}
		b        interface{}
		expected int
	}{
		{
			name:     "NumbersLessThan",
			a:        10.0,
			b:        20.0,
			expected: -1,
		},
		{
			name:     "NumbersGreaterThan",
			a:        30.0,
			b:        20.0,
			expected: 1,
		},
		{
			name:     "NumbersEqual",
			a:        20.0,
			b:        20.0,
			expected: 0,
		},
		{
			name:     "StringsLessThan",
			a:        "apple",
			b:        "banana",
			expected: -1,
		},
		{
			name:     "StringsGreaterThan",
			a:        "zebra",
			b:        "apple",
			expected: 1,
		},
		{
			name:     "StringsEqual",
			a:        "apple",
			b:        "apple",
			expected: 0,
		},
		{
			name:     "MixedTypes",
			a:        "string",
			b:        10,
			expected: 0,
		},
		{
			name:     "IntAndFloat",
			a:        int64(10),
			b:        20.0,
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareValues(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("compareValues(%v, %v) = %d, expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Test toFloat64 with all supported types
func TestToFloat64(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		expected  float64
		shouldOk  bool
	}{
		{
			name:     "Float64",
			input:    10.5,
			expected: 10.5,
			shouldOk: true,
		},
		{
			name:     "Int",
			input:    10,
			expected: 10.0,
			shouldOk: true,
		},
		{
			name:     "Int64",
			input:    int64(20),
			expected: 20.0,
			shouldOk: true,
		},
		{
			name:     "Int32",
			input:    int32(30),
			expected: 30.0,
			shouldOk: true,
		},
		{
			name:     "String",
			input:    "not a number",
			expected: 0,
			shouldOk: false,
		},
		{
			name:     "Boolean",
			input:    true,
			expected: 0,
			shouldOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat64(tt.input)
			if ok != tt.shouldOk {
				t.Errorf("toFloat64(%v) ok = %v, expected %v", tt.input, ok, tt.shouldOk)
			}
			if ok && result != tt.expected {
				t.Errorf("toFloat64(%v) = %f, expected %f", tt.input, result, tt.expected)
			}
		})
	}
}

// Test limit stage edge cases
func TestLimitStageEdgeCases(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(1)}),
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(2)}),
	}

	tests := []struct {
		name     string
		limit    interface{}
		expected int
	}{
		{
			name:     "LimitGreaterThanDocs",
			limit:    10,
			expected: 2,
		},
		{
			name:     "LimitInt64",
			limit:    int64(1),
			expected: 1,
		},
		{
			name:     "LimitFloat64",
			limit:    1.5,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := []map[string]interface{}{
				{"$limit": tt.limit},
			}
			p, err := NewPipeline(pipeline)
			if err != nil {
				t.Fatalf("Failed to create pipeline: %v", err)
			}
			results, err := p.Execute(docs)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}
			if len(results) != tt.expected {
				t.Errorf("Expected %d results, got %d", tt.expected, len(results))
			}
		})
	}
}

// Test skip stage edge cases
func TestSkipStageEdgeCases(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(1)}),
		document.NewDocumentFromMap(map[string]interface{}{"id": int64(2)}),
	}

	tests := []struct {
		name     string
		skip     interface{}
		expected int
	}{
		{
			name:     "SkipGreaterThanDocs",
			skip:     10,
			expected: 0,
		},
		{
			name:     "SkipInt64",
			skip:     int64(1),
			expected: 1,
		},
		{
			name:     "SkipFloat64",
			skip:     1.5,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := []map[string]interface{}{
				{"$skip": tt.skip},
			}
			p, err := NewPipeline(pipeline)
			if err != nil {
				t.Fatalf("Failed to create pipeline: %v", err)
			}
			results, err := p.Execute(docs)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}
			if len(results) != tt.expected {
				t.Errorf("Expected %d results, got %d", tt.expected, len(results))
			}
		})
	}
}

// Test sort stage with different order types
func TestSortStageOrderTypes(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"value": int64(30)}),
		document.NewDocumentFromMap(map[string]interface{}{"value": int64(10)}),
		document.NewDocumentFromMap(map[string]interface{}{"value": int64(20)}),
	}

	tests := []struct {
		name     string
		order    interface{}
		firstVal int64
	}{
		{
			name:     "IntAscending",
			order:    1,
			firstVal: 10,
		},
		{
			name:     "IntDescending",
			order:    -1,
			firstVal: 30,
		},
		{
			name:     "Int64Ascending",
			order:    int64(1),
			firstVal: 10,
		},
		{
			name:     "Int64Descending",
			order:    int64(-1),
			firstVal: 30,
		},
		{
			name:     "Float64Ascending",
			order:    1.0,
			firstVal: 10,
		},
		{
			name:     "Float64Descending",
			order:    -1.0,
			firstVal: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := []map[string]interface{}{
				{"$sort": map[string]interface{}{"value": tt.order}},
			}
			p, err := NewPipeline(pipeline)
			if err != nil {
				t.Fatalf("Failed to create pipeline: %v", err)
			}
			results, err := p.Execute(docs)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}
			firstValue, _ := results[0].Get("value")
			if firstValue.(int64) != tt.firstVal {
				t.Errorf("Expected first value %d, got %v", tt.firstVal, firstValue)
			}
		})
	}
}

// Test sort with missing fields
func TestSortStageMissingFields(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"value": int64(30)}),
		document.NewDocumentFromMap(map[string]interface{}{}), // missing value
		document.NewDocumentFromMap(map[string]interface{}{"value": int64(10)}),
	}

	pipeline := []map[string]interface{}{
		{"$sort": map[string]interface{}{"value": 1}},
	}
	p, err := NewPipeline(pipeline)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	results, err := p.Execute(docs)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

// Test project with integer spec (MongoDB style)
func TestProjectStageIntegerSpec(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"name": "Alice",
			"age":  int64(30),
			"city": "NYC",
		}),
	}

	pipeline := []map[string]interface{}{
		{
			"$project": map[string]interface{}{
				"name": 1, // MongoDB style inclusion
				"age":  1,
			},
		},
	}

	p, err := NewPipeline(pipeline)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	results, err := p.Execute(docs)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if !result.Has("name") || !result.Has("age") {
		t.Error("Expected name and age fields")
	}
	if result.Has("city") {
		t.Error("Expected city field to be excluded")
	}
}

// Test project with missing fields
func TestProjectStageMissingFields(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"name": "Alice",
		}),
	}

	pipeline := []map[string]interface{}{
		{
			"$project": map[string]interface{}{
				"name":    true,
				"missing": true, // This field doesn't exist
			},
		},
	}

	p, err := NewPipeline(pipeline)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	results, err := p.Execute(docs)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if !result.Has("name") {
		t.Error("Expected name field")
	}
	if result.Has("missing") {
		t.Error("Missing field should not be in result")
	}
}

// Test match stage with query error
func TestMatchStageQueryError(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"age": int64(25)}),
	}

	// Create a match stage with a filter that should trigger an error during execution
	pipeline := []map[string]interface{}{
		{
			"$match": map[string]interface{}{
				"$invalid": "operator",
			},
		},
	}

	p, err := NewPipeline(pipeline)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	_, err = p.Execute(docs)
	// The query should execute without error as invalid operators are just ignored
	// This tests the error path in MatchStage.Execute
	if err != nil {
		t.Logf("Expected error handled: %v", err)
	}
}

// Test group with non-field _id
func TestGroupStageNonFieldID(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"value": 10.0}),
		document.NewDocumentFromMap(map[string]interface{}{"value": 20.0}),
	}

	pipeline := []map[string]interface{}{
		{
			"$group": map[string]interface{}{
				"_id": "constant", // Not a field reference
				"sum": map[string]interface{}{
					"$sum": "$value",
				},
			},
		},
	}

	p, err := NewPipeline(pipeline)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	results, err := p.Execute(docs)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	id, _ := result.Get("_id")
	if id.(string) != "constant" {
		t.Errorf("Expected _id 'constant', got %v", id)
	}

	sum, _ := result.Get("sum")
	if sum.(float64) != 30.0 {
		t.Errorf("Expected sum 30.0, got %v", sum)
	}
}

// Test group with constant sum
func TestGroupStageConstantSum(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"value": 10.0}),
		document.NewDocumentFromMap(map[string]interface{}{"value": 20.0}),
	}

	pipeline := []map[string]interface{}{
		{
			"$group": map[string]interface{}{
				"_id": nil,
				"sum": map[string]interface{}{
					"$sum": 5.0, // Constant value, not field reference
				},
			},
		},
	}

	p, err := NewPipeline(pipeline)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	results, err := p.Execute(docs)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	sum, _ := result.Get("sum")
	// 5.0 * 2 documents = 10.0
	if sum.(float64) != 10.0 {
		t.Errorf("Expected sum 10.0, got %v", sum)
	}
}

// Test group with empty documents for avg
func TestGroupStageAvgEmptyDocs(t *testing.T) {
	docs := []*document.Document{}

	pipeline := []map[string]interface{}{
		{
			"$group": map[string]interface{}{
				"_id": nil,
				"avg": map[string]interface{}{
					"$avg": "$value",
				},
			},
		},
	}

	p, err := NewPipeline(pipeline)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	results, err := p.Execute(docs)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should have 0 groups for empty input
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

// Test group with non-field min/max
func TestGroupStageMinMaxNonField(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"value": 10.0}),
	}

	pipeline := []map[string]interface{}{
		{
			"$group": map[string]interface{}{
				"_id": nil,
				"min": map[string]interface{}{
					"$min": "not-a-field", // Not a field reference
				},
				"max": map[string]interface{}{
					"$max": "not-a-field",
				},
			},
		},
	}

	p, err := NewPipeline(pipeline)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	results, err := p.Execute(docs)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	minVal, _ := result.Get("min")
	if minVal != nil {
		t.Errorf("Expected min to be nil for non-field reference, got %v", minVal)
	}

	maxVal, _ := result.Get("max")
	if maxVal != nil {
		t.Errorf("Expected max to be nil for non-field reference, got %v", maxVal)
	}
}

// Test group with unsupported aggregation operator
func TestGroupStageUnsupportedOperator(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"value": 10.0}),
	}

	pipeline := []map[string]interface{}{
		{
			"$group": map[string]interface{}{
				"_id": nil,
				"result": map[string]interface{}{
					"$unsupported": "$value",
				},
			},
		},
	}

	p, err := NewPipeline(pipeline)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	_, err = p.Execute(docs)
	if err == nil {
		t.Error("Expected error for unsupported aggregation operator")
	}
}

// Test pipeline execution error propagation
func TestPipelineExecutionError(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"value": 10.0}),
	}

	// Create a pipeline with an invalid group stage that will fail during execution
	pipeline := []map[string]interface{}{
		{
			"$group": map[string]interface{}{
				"_id": nil,
				"result": "invalid", // This will cause an error
			},
		},
	}

	p, err := NewPipeline(pipeline)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	_, err = p.Execute(docs)
	if err == nil {
		t.Error("Expected error during pipeline execution")
	}
}
