package query

import (
	"sort"

	"github.com/mnohosten/laura-db/pkg/document"
)

// Executor executes queries against a collection of documents
type Executor struct {
	documents []*document.Document
	indexes   map[string]interface{} // Field name -> index
}

// NewExecutor creates a new query executor
func NewExecutor(documents []*document.Document) *Executor {
	return &Executor{
		documents: documents,
		indexes:   make(map[string]interface{}),
	}
}

// Execute executes a query and returns matching documents
func (e *Executor) Execute(query *Query) ([]*document.Document, error) {
	results := make([]*document.Document, 0)

	// Filter documents
	for _, doc := range e.documents {
		matches, err := query.Matches(doc)
		if err != nil {
			return nil, err
		}
		if matches {
			results = append(results, doc)
		}
	}

	// Sort results
	if len(query.GetSort()) > 0 {
		e.sortDocuments(results, query.GetSort())
	}

	// Apply skip
	if query.GetSkip() > 0 {
		if query.GetSkip() >= len(results) {
			results = []*document.Document{}
		} else {
			results = results[query.GetSkip():]
		}
	}

	// Apply limit
	if query.GetLimit() > 0 && query.GetLimit() < len(results) {
		results = results[:query.GetLimit()]
	}

	// Apply projection
	for i, doc := range results {
		results[i] = query.ApplyProjection(doc)
	}

	return results, nil
}

// sortDocuments sorts documents based on sort fields
func (e *Executor) sortDocuments(docs []*document.Document, sortFields []SortField) {
	sort.Slice(docs, func(i, j int) bool {
		for _, field := range sortFields {
			vi, existsI := docs[i].Get(field.Field)
			vj, existsJ := docs[j].Get(field.Field)

			// Handle missing fields
			if !existsI && !existsJ {
				continue
			}
			if !existsI {
				return !field.Ascending
			}
			if !existsJ {
				return field.Ascending
			}

			// Compare values
			cmp := compareValues(vi, vj)
			if cmp != 0 {
				if field.Ascending {
					return cmp < 0
				}
				return cmp > 0
			}
		}
		return false
	})
}

// compareValues compares two values
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func compareValues(a, b interface{}) int {
	// Try numeric comparison
	aVal, aOk := toFloat64(a)
	bVal, bOk := toFloat64(b)
	if aOk && bOk {
		if aVal < bVal {
			return -1
		} else if aVal > bVal {
			return 1
		}
		return 0
	}

	// String comparison
	aStr, aOk := a.(string)
	bStr, bOk := b.(string)
	if aOk && bOk {
		if aStr < bStr {
			return -1
		} else if aStr > bStr {
			return 1
		}
		return 0
	}

	// Default: equal
	return 0
}

// Count returns the number of documents matching the query
func (e *Executor) Count(query *Query) (int, error) {
	count := 0
	for _, doc := range e.documents {
		matches, err := query.Matches(doc)
		if err != nil {
			return 0, err
		}
		if matches {
			count++
		}
	}
	return count, nil
}

// Explain returns query execution statistics
func (e *Executor) Explain(query *Query) map[string]interface{} {
	matchCount := 0
	totalDocs := len(e.documents)

	for _, doc := range e.documents {
		if matches, _ := query.Matches(doc); matches {
			matchCount++
		}
	}

	return map[string]interface{}{
		"total_documents":    totalDocs,
		"matching_documents": matchCount,
		"execution_type":     "collection_scan", // Would be "index_scan" if using index
		"filter":             query.GetFilter(),
		"sort":               query.GetSort(),
		"limit":              query.GetLimit(),
		"skip":               query.GetSkip(),
	}
}
