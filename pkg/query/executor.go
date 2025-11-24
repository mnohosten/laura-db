package query

import (
	"fmt"
	"sort"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/index"
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
	// Check if this is a covered query (can be satisfied entirely from index)
	if plan.IsCovered {
		return e.executeCoveredQuery(query, plan)
	}

	var candidates []*document.Document

	if plan.UseIntersection {
		// Use index intersection
		var err error
		candidates, err = e.executeIndexIntersection(plan)
		if err != nil {
			// Fall back to collection scan if intersection fails
			candidates = e.documents
		}
	} else if plan.UseIndex && plan.Index != nil {
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

// executeCoveredQuery executes a query entirely from index data without fetching documents
func (e *Executor) executeCoveredQuery(query *Query, plan *QueryPlan) ([]*document.Document, error) {
	var keys []interface{}
	var values []interface{}

	switch plan.ScanType {
	case ScanTypeIndexExact:
		// Exact match scan
		searchKey := plan.ScanKey
		// Convert int to int64 (documents store numbers as int64)
		if v, ok := searchKey.(int); ok {
			searchKey = int64(v)
		}

		value, exists := plan.Index.Search(searchKey)
		if exists {
			keys = []interface{}{searchKey}
			values = []interface{}{value}
		}

	case ScanTypeIndexRange:
		// Range scan - gets both keys and values
		// Handle nil start/end (for unbounded ranges)
		start := plan.ScanStart
		end := plan.ScanEnd

		// Convert int to int64 (documents store numbers as int64)
		if v, ok := start.(int); ok {
			start = int64(v)
		}
		if v, ok := end.(int); ok {
			end = int64(v)
		}

		// Determine type from start value to ensure consistent types
		if end == nil && start != nil {
			// Use same type as start for the upper bound
			switch start.(type) {
			case int:
				end = int(2147483647) // max int32
			case int32:
				end = int32(2147483647)
			case int64:
				end = int64(9223372036854775807)
			case float64:
				end = float64(1.7976931348623157e+308)
			default:
				end = int64(9223372036854775807)
			}
		}
		if start == nil && end != nil {
			// Use same type as end for the lower bound
			switch end.(type) {
			case int:
				start = int(-2147483648)
			case int32:
				start = int32(-2147483648)
			case int64:
				start = int64(-9223372036854775808)
			case float64:
				start = float64(-1.7976931348623157e+308)
			default:
				start = int64(-9223372036854775808)
			}
		}

		allKeys, allValues := plan.Index.RangeScan(start, end)

		// If this is a prefix match on compound index, filter by prefix
		if plan.PrefixKey != nil {
			keys = make([]interface{}, 0)
			values = make([]interface{}, 0)
			for i, key := range allKeys {
				if compositeKey, ok := key.(*index.CompositeKey); ok {
					if compositeKey.MatchesPrefix(plan.PrefixKey) {
						keys = append(keys, key)
						values = append(values, allValues[i])
					}
				}
			}
		} else {
			keys = allKeys
			values = allValues
		}

	default:
		// Should not happen for covered queries
		return nil, fmt.Errorf("invalid scan type for covered query")
	}

	// Build documents from index data
	results := make([]*document.Document, 0, len(keys))
	projection := query.GetProjection()

	for i := 0; i < len(keys) && i < len(values); i++ {
		doc := document.NewDocument()

		// Add _id if requested in projection (or if no specific projection given)
		if projection == nil || projection["_id"] {
			if idStr, ok := values[i].(string); ok {
				doc.Set("_id", idStr)
			}
		}

		// Add indexed field if requested in projection (or if no specific projection given)
		if projection == nil || projection[plan.IndexedField] {
			doc.Set(plan.IndexedField, keys[i])
		}

		results = append(results, doc)
	}

	// Sort results (using the same sorting logic)
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

	// Note: No need to apply projection since we only built the requested fields

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
		keys, values := plan.Index.RangeScan(plan.ScanStart, plan.ScanEnd)
		docIDs = make([]string, 0, len(values))

		// If this is a prefix match on compound index, filter by prefix
		if plan.PrefixKey != nil {
			for i, key := range keys {
				if compositeKey, ok := key.(*index.CompositeKey); ok {
					if compositeKey.MatchesPrefix(plan.PrefixKey) {
						if idStr, ok := values[i].(string); ok {
							docIDs = append(docIDs, idStr)
						}
					}
				}
			}
		} else {
			// Normal range scan without prefix filtering
			for _, v := range values {
				if idStr, ok := v.(string); ok {
					docIDs = append(docIDs, idStr)
				}
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

// executeIndexIntersection executes an index intersection plan
func (e *Executor) executeIndexIntersection(plan *QueryPlan) ([]*document.Document, error) {
	if len(plan.IntersectPlans) == 0 {
		return e.documents, fmt.Errorf("no intersection plans provided")
	}

	// Execute each index scan and collect document IDs
	idSets := make([]map[string]bool, len(plan.IntersectPlans))

	for i, intersectPlan := range plan.IntersectPlans {
		idSet := make(map[string]bool)

		// Execute the index scan for this plan
		docIDs, err := e.executeIntersectIndexScan(intersectPlan)
		if err != nil {
			return nil, err
		}

		// Add all IDs to the set
		for _, id := range docIDs {
			idSet[id] = true
		}

		idSets[i] = idSet
	}

	// Intersect all sets (find common document IDs)
	resultIDs := e.intersectSets(idSets)

	// Convert document IDs to documents
	docs := make([]*document.Document, 0, len(resultIDs))
	for id := range resultIDs {
		if doc, exists := e.documentsMap[id]; exists {
			docs = append(docs, doc)
		}
	}

	return docs, nil
}

// executeIntersectIndexScan executes a single index scan for intersection
func (e *Executor) executeIntersectIndexScan(plan *IndexIntersectPlan) ([]string, error) {
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
		// Should not happen
		return nil, fmt.Errorf("invalid scan type for intersection")
	}

	return docIDs, nil
}

// intersectSets performs set intersection on multiple sets
func (e *Executor) intersectSets(sets []map[string]bool) map[string]bool {
	if len(sets) == 0 {
		return make(map[string]bool)
	}

	if len(sets) == 1 {
		return sets[0]
	}

	// Start with the smallest set for efficiency
	smallestIdx := 0
	smallestSize := len(sets[0])
	for i := 1; i < len(sets); i++ {
		if len(sets[i]) < smallestSize {
			smallestIdx = i
			smallestSize = len(sets[i])
		}
	}

	// Result starts with the smallest set
	result := make(map[string]bool)
	for id := range sets[smallestIdx] {
		// Check if this ID exists in all other sets
		inAll := true
		for i, set := range sets {
			if i == smallestIdx {
				continue
			}
			if !set[id] {
				inAll = false
				break
			}
		}
		if inAll {
			result[id] = true
		}
	}

	return result
}
