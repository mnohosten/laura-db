package query

import (
	"fmt"
	"sort"

	"github.com/mnohosten/laura-db/pkg/document"
)

// Executor executes queries against a collection of documents
type Executor struct {
	documents   []*document.Document
	documentsMap map[string]*document.Document // _id -> document for index lookups
	indexes     map[string]interface{}          // Field name -> index
}

// NewExecutor creates a new query executor
func NewExecutor(documents []*document.Document) *Executor {
	// Build document map for O(1) lookups by ID
	docMap := make(map[string]*document.Document)
	for _, doc := range documents {
		if idVal, exists := doc.Get("_id"); exists {
			idStr := documentIDToString(idVal)
			docMap[idStr] = doc
		}
	}

	return &Executor{
		documents:    documents,
		documentsMap: docMap,
		indexes:      make(map[string]interface{}),
	}
}

// NewExecutorWithMap creates an executor with pre-built document map
func NewExecutorWithMap(documentsMap map[string]*document.Document) *Executor {
	documents := make([]*document.Document, 0, len(documentsMap))
	for _, doc := range documentsMap {
		documents = append(documents, doc)
	}

	return &Executor{
		documents:    documents,
		documentsMap: documentsMap,
		indexes:      make(map[string]interface{}),
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

// ExecuteWithPlan executes a query using a query plan (potentially using indexes)
func (e *Executor) ExecuteWithPlan(query *Query, plan *QueryPlan) ([]*document.Document, error) {
	var candidates []*document.Document

	if plan.UseIndex && plan.Index != nil {
		// Use index to get candidate documents
		var err error
		candidates, err = e.executeIndexScan(plan)
		if err != nil {
			// Fall back to collection scan if index scan fails
			candidates = e.documents
		}
	} else {
		// Full collection scan
		candidates = e.documents
	}

	// Filter candidates (apply remaining filters after index scan)
	results := make([]*document.Document, 0)
	for _, doc := range candidates {
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

// executeIndexScan retrieves documents using the index
func (e *Executor) executeIndexScan(plan *QueryPlan) ([]*document.Document, error) {
	var docIDs []string

	switch plan.ScanType {
	case ScanTypeIndexExact:
		// Exact match scan
		value, exists := plan.Index.Search(plan.ScanKey)
		if exists {
			if idStr, ok := value.(string); ok {
				docIDs = []string{idStr}
			}
		}

	case ScanTypeIndexRange:
		// Range scan
		_, values := plan.Index.RangeScan(plan.ScanStart, plan.ScanEnd)
		docIDs = make([]string, 0, len(values))
		for _, v := range values {
			if idStr, ok := v.(string); ok {
				docIDs = append(docIDs, idStr)
			}
		}

	default:
		// Should not happen, but fall back to all documents
		return e.documents, nil
	}

	// Convert document IDs to documents
	docs := make([]*document.Document, 0, len(docIDs))
	for _, id := range docIDs {
		if doc, exists := e.documentsMap[id]; exists {
			docs = append(docs, doc)
		}
	}

	return docs, nil
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

// documentIDToString converts a document _id to string, handling ObjectID type
func documentIDToString(id interface{}) string {
	switch v := id.(type) {
	case string:
		return v
	case document.ObjectID:
		return v.Hex()
	default:
		return fmt.Sprintf("%v", v)
	}
}
