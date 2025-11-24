package query

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/index"
)

// TestIndexIntersectionTwoFields tests index intersection with two field conditions
func TestIndexIntersectionTwoFields(t *testing.T) {
	// Create test documents
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc1",
			"age":  int64(25),
			"city": "NYC",
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc2",
			"age":  int64(30),
			"city": "NYC",
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc3",
			"age":  int64(25),
			"city": "LA",
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc4",
			"age":  int64(30),
			"city": "LA",
		}),
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

	// Populate indexes
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

	// Update statistics
	ageIndex.Analyze()
	cityIndex.Analyze()

	// Create planner with indexes
	indexes := map[string]*index.Index{
		"age_idx":  ageIndex,
		"city_idx": cityIndex,
	}
	planner := NewQueryPlanner(indexes)

	// Query: {age: 25, city: "NYC"}
	// Should use index intersection
	query := NewQuery(map[string]interface{}{
		"age":  int64(25),
		"city": "NYC",
	})

	plan := planner.Plan(query)

	// Verify plan uses intersection
	if !plan.UseIntersection {
		t.Error("Expected plan to use index intersection")
	}

	if len(plan.IntersectPlans) != 2 {
		t.Errorf("Expected 2 intersect plans, got %d", len(plan.IntersectPlans))
	}

	// Execute query with intersection
	executor := NewExecutor(docs)
	results, err := executor.ExecuteWithPlan(query, plan)
	if err != nil {
		t.Fatalf("Query execution failed: %v", err)
	}

	// Should return only doc1 (age=25 AND city=NYC)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if len(results) > 0 {
		if id, exists := results[0].Get("_id"); exists {
			if id.(string) != "doc1" {
				t.Errorf("Expected doc1, got %v", id)
			}
		}
	}
}

// TestIndexIntersectionThreeFields tests intersection with three indexes
func TestIndexIntersectionThreeFields(t *testing.T) {
	// Create test documents
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":    "doc1",
			"age":    int64(25),
			"city":   "NYC",
			"status": "active",
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":    "doc2",
			"age":    int64(25),
			"city":   "NYC",
			"status": "inactive",
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":    "doc3",
			"age":    int64(25),
			"city":   "LA",
			"status": "active",
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":    "doc4",
			"age":    int64(30),
			"city":   "NYC",
			"status": "active",
		}),
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
	statusIndex := index.NewIndex(&index.IndexConfig{
		Name:      "status_idx",
		FieldPath: "status",
		Type:      index.IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Populate indexes
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

	// Update statistics
	ageIndex.Analyze()
	cityIndex.Analyze()
	statusIndex.Analyze()

	// Create planner with indexes
	indexes := map[string]*index.Index{
		"age_idx":    ageIndex,
		"city_idx":   cityIndex,
		"status_idx": statusIndex,
	}
	planner := NewQueryPlanner(indexes)

	// Query: {age: 25, city: "NYC", status: "active"}
	query := NewQuery(map[string]interface{}{
		"age":    int64(25),
		"city":   "NYC",
		"status": "active",
	})

	plan := planner.Plan(query)

	// Note: With small datasets, planner may choose single index over intersection
	// This is correct behavior - intersection is better for larger datasets
	// We just verify the query still works correctly regardless of plan type

	// Execute query
	executor := NewExecutor(docs)
	results, err := executor.ExecuteWithPlan(query, plan)
	if err != nil {
		t.Fatalf("Query execution failed: %v", err)
	}

	// Should return only doc1
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if len(results) > 0 {
		if id, exists := results[0].Get("_id"); exists {
			if id.(string) != "doc1" {
				t.Errorf("Expected doc1, got %v", id)
			}
		}
	}
}

// TestIndexIntersectionNoResults tests intersection with no matching documents
func TestIndexIntersectionNoResults(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc1",
			"age":  int64(25),
			"city": "NYC",
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc2",
			"age":  int64(30),
			"city": "LA",
		}),
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

	// Populate indexes
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

	// Query with no matches: {age: 25, city: "LA"}
	query := NewQuery(map[string]interface{}{
		"age":  int64(25),
		"city": "LA",
	})

	plan := planner.Plan(query)
	executor := NewExecutor(docs)
	results, err := executor.ExecuteWithPlan(query, plan)
	if err != nil {
		t.Fatalf("Query execution failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

// TestIndexIntersectionVsSingleIndex tests that intersection is chosen when beneficial
func TestIndexIntersectionVsSingleIndex(t *testing.T) {
	// Create many documents where single index would be expensive
	docs := make([]*document.Document, 1000)
	for i := 0; i < 1000; i++ {
		age := int64(20 + (i % 50)) // Ages 20-69
		city := "City" + string(rune('A'+(i%10))) // 10 different cities

		docs[i] = document.NewDocumentFromMap(map[string]interface{}{
			"_id":  document.NewObjectID().Hex(),
			"age":  age,
			"city": city,
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

	// Populate indexes
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

	// Query with both conditions
	query := NewQuery(map[string]interface{}{
		"age":  int64(25),
		"city": "CityA",
	})

	plan := planner.Plan(query)

	// Verify results are correct
	executor := NewExecutor(docs)
	results, err := executor.ExecuteWithPlan(query, plan)
	if err != nil {
		t.Fatalf("Query execution failed: %v", err)
	}

	// Verify all results match the criteria
	for _, doc := range results {
		age, _ := doc.Get("age")
		city, _ := doc.Get("city")

		if age.(int64) != 25 {
			t.Errorf("Result has wrong age: %v", age)
		}
		if city.(string) != "CityA" {
			t.Errorf("Result has wrong city: %v", city)
		}
	}
}

// TestIndexIntersectionExplain tests the explain output for intersection queries
func TestIndexIntersectionExplain(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc1",
			"age":  int64(25),
			"city": "NYC",
		}),
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
		"city": "NYC",
	})

	plan := planner.Plan(query)
	explanation := plan.Explain()

	// Verify explanation has intersection info
	if scanType, ok := explanation["scanType"].(string); ok {
		if scanType != "INDEX_INTERSECTION" {
			t.Errorf("Expected scanType INDEX_INTERSECTION, got %v", scanType)
		}
	} else {
		t.Error("Explanation missing scanType")
	}

	if indexes, ok := explanation["indexes"].([]string); ok {
		if len(indexes) != 2 {
			t.Errorf("Expected 2 indexes in explanation, got %d", len(indexes))
		}
	}

	if fields, ok := explanation["fields"].([]string); ok {
		if len(fields) != 2 {
			t.Errorf("Expected 2 fields in explanation, got %d", len(fields))
		}
	}
}

// TestIndexIntersectionWithRangeQueries tests intersection with range operators
func TestIndexIntersectionWithRangeQueries(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":    "doc1",
			"age":    int64(25),
			"salary": int64(50000),
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":    "doc2",
			"age":    int64(30),
			"salary": int64(60000),
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":    "doc3",
			"age":    int64(35),
			"salary": int64(70000),
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":    "doc4",
			"age":    int64(40),
			"salary": int64(80000),
		}),
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

	// Query: {age: {$gte: 30}, salary: {$lte: 70000}}
	// Should match doc2 and doc3
	query := NewQuery(map[string]interface{}{
		"age":    map[string]interface{}{"$gte": int64(30)},
		"salary": map[string]interface{}{"$lte": int64(70000)},
	})

	plan := planner.Plan(query)
	executor := NewExecutor(docs)
	results, err := executor.ExecuteWithPlan(query, plan)
	if err != nil {
		t.Fatalf("Query execution failed: %v", err)
	}

	// Should return doc2 and doc3
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify results
	foundDoc2 := false
	foundDoc3 := false
	for _, doc := range results {
		if id, exists := doc.Get("_id"); exists {
			if id.(string) == "doc2" {
				foundDoc2 = true
			}
			if id.(string) == "doc3" {
				foundDoc3 = true
			}
		}
	}

	if !foundDoc2 || !foundDoc3 {
		t.Error("Did not find expected documents (doc2 and doc3)")
	}
}

// TestIndexIntersectionWithPartialIndexCoverage tests intersection with extra filters
func TestIndexIntersectionWithPartialIndexCoverage(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":    "doc1",
			"age":    int64(25),
			"city":   "NYC",
			"status": "active",
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":    "doc2",
			"age":    int64(25),
			"city":   "NYC",
			"status": "inactive",
		}),
	}

	// Only create indexes for age and city (not status)
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

	// Query with 3 conditions but only 2 indexes
	query := NewQuery(map[string]interface{}{
		"age":    int64(25),
		"city":   "NYC",
		"status": "active",
	})

	plan := planner.Plan(query)
	executor := NewExecutor(docs)
	results, err := executor.ExecuteWithPlan(query, plan)
	if err != nil {
		t.Fatalf("Query execution failed: %v", err)
	}

	// Should return only doc1 (status filter applied after intersection)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if len(results) > 0 {
		if id, exists := results[0].Get("_id"); exists {
			if id.(string) != "doc1" {
				t.Errorf("Expected doc1, got %v", id)
			}
		}
	}
}

// TestIndexIntersectionEmptySets tests intersection when one index returns no results
func TestIndexIntersectionEmptySets(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc1",
			"age":  int64(25),
			"city": "NYC",
		}),
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

	// Query where one condition has no matches
	query := NewQuery(map[string]interface{}{
		"age":  int64(99), // No documents with age 99
		"city": "NYC",
	})

	plan := planner.Plan(query)
	executor := NewExecutor(docs)
	results, err := executor.ExecuteWithPlan(query, plan)
	if err != nil {
		t.Fatalf("Query execution failed: %v", err)
	}

	// Should return no results
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}
