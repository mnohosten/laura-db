package query

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
)

func TestElemMatchOperator(t *testing.T) {
	// Create test document with scores array
	doc := document.NewDocument()
	doc.Set("name", "Alice")
	doc.Set("scores", []interface{}{75, 82, 90, 78, 88})

	// Test $elemMatch - find score between 80 and 85
	query := NewQuery(map[string]interface{}{
		"scores": map[string]interface{}{
			"$elemMatch": map[string]interface{}{
				"$gte": 80,
				"$lt":  85,
			},
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Document should match: has score 82 which is between 80 and 85")
	}
}

func TestElemMatchNoMatch(t *testing.T) {
	// Create test document with scores array
	doc := document.NewDocument()
	doc.Set("name", "Bob")
	doc.Set("scores", []interface{}{75, 78, 76, 72})

	// Test $elemMatch - looking for score >= 80
	query := NewQuery(map[string]interface{}{
		"scores": map[string]interface{}{
			"$elemMatch": map[string]interface{}{
				"$gte": 80,
			},
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if matches {
		t.Error("Document should not match: no score >= 80")
	}
}

func TestElemMatchMultipleConditions(t *testing.T) {
	// Create test document with scores array
	doc := document.NewDocument()
	doc.Set("name", "Charlie")
	doc.Set("scores", []interface{}{85, 92, 88, 95, 78})

	// Test $elemMatch - score >= 90 AND < 95
	query := NewQuery(map[string]interface{}{
		"scores": map[string]interface{}{
			"$elemMatch": map[string]interface{}{
				"$gte": 90,
				"$lt":  95,
			},
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Document should match: has score 92 which is >= 90 and < 95")
	}
}

func TestElemMatchNonArray(t *testing.T) {
	// Create test document with non-array field
	doc := document.NewDocument()
	doc.Set("name", "Diana")
	doc.Set("score", 85) // Single value, not an array

	// Test $elemMatch on non-array field
	query := NewQuery(map[string]interface{}{
		"score": map[string]interface{}{
			"$elemMatch": map[string]interface{}{
				"$gte": 80,
			},
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if matches {
		t.Error("Document should not match: score is not an array")
	}
}

func TestSizeOperator(t *testing.T) {
	// Create test document with tags array
	doc := document.NewDocument()
	doc.Set("name", "Eve")
	doc.Set("tags", []interface{}{"go", "database", "nosql"})

	// Test $size - array with exactly 3 elements
	query := NewQuery(map[string]interface{}{
		"tags": map[string]interface{}{
			"$size": 3,
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Document should match: tags array has exactly 3 elements")
	}
}

func TestSizeOperatorNoMatch(t *testing.T) {
	// Create test document with tags array
	doc := document.NewDocument()
	doc.Set("name", "Frank")
	doc.Set("tags", []interface{}{"go", "database"})

	// Test $size - looking for exactly 3 elements
	query := NewQuery(map[string]interface{}{
		"tags": map[string]interface{}{
			"$size": 3,
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if matches {
		t.Error("Document should not match: tags array has 2 elements, not 3")
	}
}

func TestSizeOperatorEmptyArray(t *testing.T) {
	// Create test document with empty array
	doc := document.NewDocument()
	doc.Set("name", "Grace")
	doc.Set("tags", []interface{}{})

	// Test $size - array with 0 elements
	query := NewQuery(map[string]interface{}{
		"tags": map[string]interface{}{
			"$size": 0,
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Document should match: tags array is empty (size 0)")
	}
}

func TestSizeOperatorNonArray(t *testing.T) {
	// Create test document with non-array field
	doc := document.NewDocument()
	doc.Set("name", "Henry")
	doc.Set("tag", "go") // Single value, not an array

	// Test $size on non-array field
	query := NewQuery(map[string]interface{}{
		"tag": map[string]interface{}{
			"$size": 1,
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if matches {
		t.Error("Document should not match: tag is not an array")
	}
}

func TestCombinedArrayOperators(t *testing.T) {
	// Create test document
	doc := document.NewDocument()
	doc.Set("name", "Isabel")
	doc.Set("scores", []interface{}{75, 82, 88, 91, 85})
	doc.Set("tags", []interface{}{"go", "database", "nosql"})

	// Test combining $elemMatch and $size
	query := NewQuery(map[string]interface{}{
		"scores": map[string]interface{}{
			"$elemMatch": map[string]interface{}{
				"$gte": 90,
			},
		},
		"tags": map[string]interface{}{
			"$size": 3,
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Document should match: has score >= 90 and tags has size 3")
	}
}

func TestElemMatchWithStrings(t *testing.T) {
	// Create test document with string array
	doc := document.NewDocument()
	doc.Set("name", "Jack")
	doc.Set("tags", []interface{}{"mongodb", "database", "nosql", "go"})

	// Test $elemMatch with string comparison - won't work as expected since
	// we're using numeric operators, but let's test equality
	query := NewQuery(map[string]interface{}{
		"tags": map[string]interface{}{
			"$elemMatch": map[string]interface{}{
				"$eq": "go",
			},
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Document should match: tags contains 'go'")
	}
}

func TestSizeWithFloat(t *testing.T) {
	// Create test document
	doc := document.NewDocument()
	doc.Set("items", []interface{}{1, 2, 3, 4, 5})

	// Test $size with float64 (JSON unmarshaling produces float64)
	query := NewQuery(map[string]interface{}{
		"items": map[string]interface{}{
			"$size": float64(5),
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Document should match: items array has 5 elements")
	}
}
