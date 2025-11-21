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
