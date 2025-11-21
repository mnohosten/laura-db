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
