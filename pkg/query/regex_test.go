package query

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
)

func TestRegexOperator(t *testing.T) {
	// Create test document
	doc := document.NewDocument()
	doc.Set("name", "Alice Johnson")
	doc.Set("email", "alice@example.com")
	doc.Set("city", "New York")

	// Test $regex - starts with "Alice"
	query := NewQuery(map[string]interface{}{
		"name": map[string]interface{}{
			"$regex": "^Alice",
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Document should match: name starts with 'Alice'")
	}
}

func TestRegexCaseInsensitive(t *testing.T) {
	// Create test document
	doc := document.NewDocument()
	doc.Set("name", "Alice Johnson")

	// Test $regex - case insensitive match
	query := NewQuery(map[string]interface{}{
		"name": map[string]interface{}{
			"$regex": "(?i)^alice",
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Document should match: case insensitive regex for 'alice'")
	}
}

func TestRegexEndsWith(t *testing.T) {
	// Create test document
	doc := document.NewDocument()
	doc.Set("email", "alice@example.com")

	// Test $regex - ends with .com
	query := NewQuery(map[string]interface{}{
		"email": map[string]interface{}{
			"$regex": "\\.com$",
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Document should match: email ends with '.com'")
	}
}

func TestRegexContains(t *testing.T) {
	// Create test document
	doc := document.NewDocument()
	doc.Set("description", "LauraDB is a MongoDB-like database")

	// Test $regex - contains "MongoDB"
	query := NewQuery(map[string]interface{}{
		"description": map[string]interface{}{
			"$regex": "MongoDB",
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Document should match: description contains 'MongoDB'")
	}
}

func TestRegexNoMatch(t *testing.T) {
	// Create test document
	doc := document.NewDocument()
	doc.Set("name", "Bob Smith")

	// Test $regex - looking for "Alice"
	query := NewQuery(map[string]interface{}{
		"name": map[string]interface{}{
			"$regex": "^Alice",
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if matches {
		t.Error("Document should not match: name does not start with 'Alice'")
	}
}

func TestRegexEmailValidation(t *testing.T) {
	// Create test documents
	validDoc := document.NewDocument()
	validDoc.Set("email", "user@example.com")

	invalidDoc := document.NewDocument()
	invalidDoc.Set("email", "not-an-email")

	// Email validation regex (simple version)
	emailPattern := "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"

	query := NewQuery(map[string]interface{}{
		"email": map[string]interface{}{
			"$regex": emailPattern,
		},
	})

	// Valid email should match
	matches, err := query.Matches(validDoc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !matches {
		t.Error("Valid email should match regex pattern")
	}

	// Invalid email should not match
	matches, err = query.Matches(invalidDoc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if matches {
		t.Error("Invalid email should not match regex pattern")
	}
}

func TestRegexPhoneNumber(t *testing.T) {
	// Create test document
	doc := document.NewDocument()
	doc.Set("phone", "555-1234")

	// Test $regex - phone number pattern
	query := NewQuery(map[string]interface{}{
		"phone": map[string]interface{}{
			"$regex": "^\\d{3}-\\d{4}$",
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Document should match: phone matches pattern XXX-XXXX")
	}
}

func TestRegexNonStringField(t *testing.T) {
	// Create test document with non-string field
	doc := document.NewDocument()
	doc.Set("age", 25)

	// Test $regex on non-string field
	query := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{
			"$regex": "^25",
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if matches {
		t.Error("Document should not match: age is not a string")
	}
}

func TestRegexInvalidPattern(t *testing.T) {
	// Create test document
	doc := document.NewDocument()
	doc.Set("name", "Alice")

	// Test $regex with invalid pattern
	query := NewQuery(map[string]interface{}{
		"name": map[string]interface{}{
			"$regex": "[invalid(",
		},
	})

	_, err := query.Matches(doc)
	if err == nil {
		t.Error("Query should fail with invalid regex pattern")
	}
}

func TestRegexMultiline(t *testing.T) {
	// Create test document with multiline text
	doc := document.NewDocument()
	doc.Set("text", "Line 1\nLine 2\nLine 3")

	// Test $regex - match across lines
	query := NewQuery(map[string]interface{}{
		"text": map[string]interface{}{
			"$regex": "Line 1.*Line 2",
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// By default, . doesn't match newlines in Go regex
	if matches {
		t.Error("Should not match: . doesn't match newlines by default")
	}
}

func TestRegexWithDotAll(t *testing.T) {
	// Create test document with multiline text
	doc := document.NewDocument()
	doc.Set("text", "Line 1\nLine 2\nLine 3")

	// Test $regex - match across lines using (?s) flag
	query := NewQuery(map[string]interface{}{
		"text": map[string]interface{}{
			"$regex": "(?s)Line 1.*Line 2",
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Should match: (?s) flag makes . match newlines")
	}
}

func TestRegexWordBoundary(t *testing.T) {
	// Create test document
	doc := document.NewDocument()
	doc.Set("text", "The database is fast")

	// Test $regex - word boundary
	query := NewQuery(map[string]interface{}{
		"text": map[string]interface{}{
			"$regex": "\\bdata\\w+\\b",
		},
	})

	matches, err := query.Matches(doc)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !matches {
		t.Error("Should match: 'database' starts with 'data'")
	}
}

func TestRegexAlternation(t *testing.T) {
	// Create test documents
	doc1 := document.NewDocument()
	doc1.Set("status", "active")

	doc2 := document.NewDocument()
	doc2.Set("status", "pending")

	doc3 := document.NewDocument()
	doc3.Set("status", "inactive")

	// Test $regex - alternation (active OR pending)
	query := NewQuery(map[string]interface{}{
		"status": map[string]interface{}{
			"$regex": "^(active|pending)$",
		},
	})

	// Should match active
	matches, err := query.Matches(doc1)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !matches {
		t.Error("Should match: status is 'active'")
	}

	// Should match pending
	matches, err = query.Matches(doc2)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !matches {
		t.Error("Should match: status is 'pending'")
	}

	// Should not match inactive
	matches, err = query.Matches(doc3)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if matches {
		t.Error("Should not match: status is 'inactive'")
	}
}
