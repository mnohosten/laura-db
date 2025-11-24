package query

import (
	"github.com/mnohosten/laura-db/pkg/index"
)

// QueryPlan represents an execution plan for a query
type QueryPlan struct {
	UseIndex      bool
	IndexName     string
	Index         *index.Index
	ScanType      ScanType
	ScanKey       interface{} // For exact match scans
	ScanStart     interface{} // For range scans
	ScanEnd       interface{} // For range scans
	PrefixKey     *index.CompositeKey // For compound index prefix matching
	EstimatedCost int
	FilterSteps   []string // Fields that still need filtering after index scan
	IsCovered     bool     // True if query can be satisfied entirely from index
	IndexedField  string   // The field covered by the index

	// Index intersection support
	UseIntersection bool                  // True if using multiple indexes
	IntersectPlans  []*IndexIntersectPlan // Plans for each index in intersection
}

// IndexIntersectPlan represents a single index scan in an intersection
type IndexIntersectPlan struct {
	IndexName string
	Index     *index.Index
	Field     string
	ScanType  ScanType
	ScanKey   interface{} // For exact match
	ScanStart interface{} // For range scans
	ScanEnd   interface{} // For range scans
}

// ScanType represents the type of index scan
type ScanType int

const (
	ScanTypeCollection ScanType = iota // Full collection scan
	ScanTypeIndexExact                 // Exact match on index
	ScanTypeIndexRange                 // Range scan on index
)

// QueryPlanner plans query execution
type QueryPlanner struct {
	indexes map[string]*index.Index
}

// NewQueryPlanner creates a new query planner
func NewQueryPlanner(indexes map[string]*index.Index) *QueryPlanner {
	return &QueryPlanner{
		indexes: indexes,
	}
}

// Plan creates an execution plan for a query
func (qp *QueryPlanner) Plan(q *Query) *QueryPlan {
	plan := &QueryPlan{
		UseIndex:      false,
		ScanType:      ScanTypeCollection,
		EstimatedCost: 1000000, // High cost for collection scan
		FilterSteps:   []string{},
	}

	if len(q.filter) == 0 {
		// Empty filter - must scan all documents
		return plan
	}

	// Analyze filter to find usable indexes
	// Use statistics to choose the most selective index
	bestPlan := plan
	for indexName, idx := range qp.indexes {
		indexPlan := qp.analyzeIndexForFilter(indexName, idx, q.filter)
		if indexPlan != nil {
			// Use statistics to refine cost estimate
			indexPlan.EstimatedCost = qp.estimateCostWithStats(indexPlan, idx)

			// Choose index based on cost, but prefer indexes that eliminate more filters
			// If costs are equal or close, prefer the one with fewer remaining filter steps
			if indexPlan.EstimatedCost < bestPlan.EstimatedCost {
				bestPlan = indexPlan
			} else if indexPlan.EstimatedCost == bestPlan.EstimatedCost {
				// Costs equal - prefer index that matches more fields
				if len(indexPlan.FilterSteps) < len(bestPlan.FilterSteps) {
					bestPlan = indexPlan
				}
			}
		}
	}

	// Try index intersection if we have multiple conditions
	intersectionPlan := qp.planIndexIntersection(q.filter)
	if intersectionPlan != nil && intersectionPlan.EstimatedCost < bestPlan.EstimatedCost {
		bestPlan = intersectionPlan
	}

	return bestPlan
}

// analyzeIndexForFilter analyzes if an index can be used for a filter
func (qp *QueryPlanner) analyzeIndexForFilter(indexName string, idx *index.Index, filter map[string]interface{}) *QueryPlan {
	// Handle compound indexes
	if idx.IsCompound() {
		return qp.analyzeCompoundIndexForFilter(indexName, idx, filter)
	}

	// Single-field index
	fieldPath := idx.FieldPath()

	// Check if this index's field is in the filter
	for filterField, filterValue := range filter {
		// Skip logical operators
		if filterField == "$and" || filterField == "$or" {
			continue
		}

		// Check if this field matches the index
		if filterField != fieldPath {
			continue
		}

		// Analyze the filter condition on this field
		return qp.analyzeSingleFieldFilter(indexName, idx, filterField, filterValue, filter)
	}

	return nil
}

// analyzeCompoundIndexForFilter analyzes if a compound index can be used for a filter
func (qp *QueryPlanner) analyzeCompoundIndexForFilter(indexName string, idx *index.Index, filter map[string]interface{}) *QueryPlan {
	fieldPaths := idx.FieldPaths()

	// For compound indexes, we need to match fields in order (prefix matching)
	// Example: index on [city, age] can be used for:
	//   - {city: "NYC"} (prefix match)
	//   - {city: "NYC", age: 30} (full match)
	// But NOT for:
	//   - {age: 30} (doesn't start with first field)

	// Find how many leading fields from the index are in the filter
	matchedFields := make([]string, 0)
	matchedValues := make([]interface{}, 0)

	for _, fieldPath := range fieldPaths {
		if filterValue, exists := filter[fieldPath]; exists {
			matchedFields = append(matchedFields, fieldPath)
			matchedValues = append(matchedValues, filterValue)
		} else {
			// Stop at first missing field (can't skip fields in compound index)
			break
		}
	}

	// Must match at least the first field to use compound index
	if len(matchedFields) == 0 {
		return nil
	}

	// Build query plan for compound index
	plan := &QueryPlan{
		UseIndex:     true,
		IndexName:    indexName,
		Index:        idx,
		FilterSteps:  qp.getRemainingFilters(matchedFields[0], filter), // Will refine this
		IndexedField: matchedFields[0], // Primary field
	}

	// Check if all matched fields have equality conditions
	allEquality := true
	compositeKeyValues := make([]interface{}, 0, len(matchedFields))

	for i := range matchedFields {
		value := matchedValues[i]

		// Check if it's an operator expression
		if operatorMap, ok := value.(map[string]interface{}); ok {
			// For compound indexes, only support $eq for non-final fields
			if eqValue, hasEq := operatorMap["$eq"]; hasEq {
				compositeKeyValues = append(compositeKeyValues, eqValue)
			} else if i == len(matchedFields)-1 {
				// For the last matched field, we can support range operators
				// But this requires partial composite key matching
				// For now, only support equality
				allEquality = false
				break
			} else {
				// Non-final field must be equality
				allEquality = false
				break
			}
		} else {
			// Direct value (implicit $eq)
			compositeKeyValues = append(compositeKeyValues, value)
		}
	}

	if allEquality && len(compositeKeyValues) == len(matchedFields) {
		compositeKey := index.NewCompositeKey(compositeKeyValues...)

		// Check if this is a prefix match or full match
		isPrefix := len(matchedFields) < len(fieldPaths)

		if isPrefix {
			// Prefix match - use range scan and filter by prefix
			plan.ScanType = ScanTypeIndexRange
			plan.ScanStart = nil // Scan from beginning
			plan.ScanEnd = nil   // Scan to end
			plan.PrefixKey = compositeKey
			plan.EstimatedCost = 20 // Low cost for prefix scan
		} else {
			// Full match - exact composite key lookup
			plan.ScanType = ScanTypeIndexExact
			plan.ScanKey = compositeKey
			plan.EstimatedCost = 10 // Very low cost for exact match
		}

		// Update remaining filters (exclude all matched fields)
		remaining := make([]string, 0)
		for field := range filter {
			if field == "$and" || field == "$or" {
				continue
			}
			isMatched := false
			for _, matched := range matchedFields {
				if field == matched {
					isMatched = true
					break
				}
			}
			if !isMatched {
				remaining = append(remaining, field)
			}
		}
		plan.FilterSteps = remaining

		return plan
	}

	// Partial match or range queries - for now, don't use compound index
	// (Future enhancement: support range on last field)
	return nil
}

// analyzeSingleFieldFilter analyzes a single field filter
func (qp *QueryPlanner) analyzeSingleFieldFilter(indexName string, idx *index.Index, field string, value interface{}, fullFilter map[string]interface{}) *QueryPlan {
	plan := &QueryPlan{
		UseIndex:     true,
		IndexName:    indexName,
		Index:        idx,
		FilterSteps:  qp.getRemainingFilters(field, fullFilter),
		IndexedField: idx.FieldPath(),
	}

	// Check if it's an operator expression
	if operatorMap, ok := value.(map[string]interface{}); ok {
		// Handle different operators
		hasGt := false
		hasGte := false
		hasLt := false
		hasLte := false
		var gtValue, gteValue, ltValue, lteValue interface{}

		for opStr, opValue := range operatorMap {
			switch opStr {
			case "$eq":
				// Exact match - best case for index
				plan.ScanType = ScanTypeIndexExact
				plan.ScanKey = opValue
				plan.EstimatedCost = 10 // Very low cost
				return plan

			case "$gt":
				hasGt = true
				gtValue = opValue

			case "$gte":
				hasGte = true
				gteValue = opValue

			case "$lt":
				hasLt = true
				ltValue = opValue

			case "$lte":
				hasLte = true
				lteValue = opValue

			case "$in":
				// Could use index for each value, but for now treat as medium cost
				plan.ScanType = ScanTypeCollection
				plan.EstimatedCost = 500 // Medium cost - could be optimized
				return plan

			default:
				// Other operators can't use index effectively
				return nil
			}
		}

		// Check if we can do a range scan
		if hasGt || hasGte || hasLt || hasLte {
			plan.ScanType = ScanTypeIndexRange
			plan.EstimatedCost = 50 // Low cost for range scan

			// Determine range boundaries
			if hasGte {
				plan.ScanStart = gteValue
			} else if hasGt {
				plan.ScanStart = gtValue
			}

			if hasLte {
				plan.ScanEnd = lteValue
			} else if hasLt {
				plan.ScanEnd = ltValue
			}

			return plan
		}

		// Can't use index for this operator
		return nil
	}

	// Direct value comparison (implicit $eq)
	plan.ScanType = ScanTypeIndexExact
	plan.ScanKey = value
	plan.EstimatedCost = 10 // Very low cost
	return plan
}

// getRemainingFilters returns fields that need filtering after index scan
func (qp *QueryPlanner) getRemainingFilters(indexedField string, filter map[string]interface{}) []string {
	remaining := []string{}
	for field := range filter {
		if field != indexedField && field != "$and" && field != "$or" {
			remaining = append(remaining, field)
		}
	}
	return remaining
}

// estimateCostWithStats estimates query cost using index statistics
func (qp *QueryPlanner) estimateCostWithStats(plan *QueryPlan, idx *index.Index) int {
	stats := idx.GetStatistics()

	// If statistics are stale or missing, use default cost
	totalEntries, uniqueKeys, _, _, isStale := stats.GetStats()
	if isStale {
		return plan.EstimatedCost
	}

	switch plan.ScanType {
	case ScanTypeIndexExact:
		// Exact match - cost depends on cardinality
		// Higher cardinality (more unique values) = lower cost per lookup
		// This favors indexes with more distinct values

		if uniqueKeys > 1000 {
			// Very high cardinality - excellent selectivity
			return 5
		} else if uniqueKeys > 100 {
			// High cardinality
			return 8
		} else if uniqueKeys > 10 {
			// Medium cardinality
			return 12
		} else {
			// Low cardinality
			return 20
		}

	case ScanTypeIndexRange:
		// Range scan - estimate based on total entries
		// Assume range queries cover 30% of data on average
		rangeSelectivity := 0.3
		estimatedRows := int(float64(totalEntries) * rangeSelectivity)

		// Cost is proportional to estimated rows
		cost := estimatedRows
		if cost < 20 {
			cost = 20 // Minimum cost for range scan
		}
		if cost > 500 {
			cost = 500 // Cap cost
		}
		return cost

	default:
		return plan.EstimatedCost
	}
}

// DetectCoveredQuery checks if the query can be satisfied entirely from the index
func (qp *QueryPlanner) DetectCoveredQuery(plan *QueryPlan, projection map[string]bool) {
	// Query must use an index to be covered
	if !plan.UseIndex {
		plan.IsCovered = false
		return
	}

	// If no projection specified, we need all fields (not covered)
	if projection == nil || len(projection) == 0 {
		plan.IsCovered = false
		return
	}

	// Check if all requested fields are available from index
	// Index provides: the indexed field and _id (from the index value)
	for field, include := range projection {
		if !include {
			// Exclusion projection - not supported for covered queries yet
			plan.IsCovered = false
			return
		}

		// Check if field is available from index
		if field != plan.IndexedField && field != "_id" {
			// Field not available from index
			plan.IsCovered = false
			return
		}
	}

	// All projected fields are available from index!
	plan.IsCovered = true
}

// Explain returns a human-readable explanation of the query plan
func (plan *QueryPlan) Explain() map[string]interface{} {
	result := map[string]interface{}{
		"estimatedCost": plan.EstimatedCost,
		"useIndex":      plan.UseIndex,
		"isCovered":     plan.IsCovered,
	}

	if plan.UseIntersection {
		result["scanType"] = "INDEX_INTERSECTION"
		indexNames := make([]string, len(plan.IntersectPlans))
		fields := make([]string, len(plan.IntersectPlans))
		for i, ip := range plan.IntersectPlans {
			indexNames[i] = ip.IndexName
			fields[i] = ip.Field
		}
		result["indexes"] = indexNames
		result["fields"] = fields
		result["note"] = "Using multiple indexes with set intersection"

		if len(plan.FilterSteps) > 0 {
			result["additionalFilters"] = plan.FilterSteps
		}
	} else if plan.UseIndex {
		result["indexName"] = plan.IndexName
		result["indexedField"] = plan.IndexedField

		switch plan.ScanType {
		case ScanTypeIndexExact:
			result["scanType"] = "INDEX_EXACT"
			result["scanKey"] = plan.ScanKey
		case ScanTypeIndexRange:
			result["scanType"] = "INDEX_RANGE"
			if plan.ScanStart != nil {
				result["scanStart"] = plan.ScanStart
			}
			if plan.ScanEnd != nil {
				result["scanEnd"] = plan.ScanEnd
			}
		default:
			result["scanType"] = "COLLECTION_SCAN"
		}

		if len(plan.FilterSteps) > 0 {
			result["additionalFilters"] = plan.FilterSteps
		}

		if plan.IsCovered {
			result["note"] = "Query can be satisfied entirely from index (covered query)"
		}
	} else {
		result["scanType"] = "COLLECTION_SCAN"
		result["reason"] = "No suitable index found"
	}

	return result
}

// planIndexIntersection attempts to create a plan using multiple indexes
func (qp *QueryPlanner) planIndexIntersection(filter map[string]interface{}) *QueryPlan {
	// Extract simple field conditions (no $and, $or for now)
	fieldConditions := make(map[string]interface{})
	for field, value := range filter {
		if field != "$and" && field != "$or" && field != "$not" {
			fieldConditions[field] = value
		}
	}

	// Need at least 2 fields to consider intersection
	if len(fieldConditions) < 2 {
		return nil
	}

	// Find indexes that can be used for each field
	intersectPlans := make([]*IndexIntersectPlan, 0)
	usedFields := make([]string, 0)

	for field, value := range fieldConditions {
		// Try to find an index for this field
		for indexName, idx := range qp.indexes {
			// Skip compound indexes for now (simpler to use single-field indexes)
			if idx.IsCompound() {
				continue
			}

			// Check if this index matches the field
			if idx.FieldPath() != field {
				continue
			}

			// Analyze the condition
			intersectPlan := qp.createIntersectPlan(indexName, idx, field, value)
			if intersectPlan != nil {
				intersectPlans = append(intersectPlans, intersectPlan)
				usedFields = append(usedFields, field)
				break // Found an index for this field
			}
		}
	}

	// Need at least 2 indexes to do intersection
	if len(intersectPlans) < 2 {
		return nil
	}

	// Estimate cost of intersection
	// Cost = sum of index scans + intersection overhead
	estimatedCost := qp.estimateIntersectionCost(intersectPlans)

	// Find remaining filters (fields not covered by any index)
	remainingFilters := make([]string, 0)
	for field := range fieldConditions {
		covered := false
		for _, usedField := range usedFields {
			if field == usedField {
				covered = true
				break
			}
		}
		if !covered {
			remainingFilters = append(remainingFilters, field)
		}
	}

	return &QueryPlan{
		UseIndex:        true,
		UseIntersection: true,
		IntersectPlans:  intersectPlans,
		ScanType:        ScanTypeCollection, // Not used for intersection
		EstimatedCost:   estimatedCost,
		FilterSteps:     remainingFilters,
		IsCovered:       false, // Intersection requires document fetch
	}
}

// createIntersectPlan creates a plan for a single index in an intersection
func (qp *QueryPlanner) createIntersectPlan(indexName string, idx *index.Index, field string, value interface{}) *IndexIntersectPlan {
	plan := &IndexIntersectPlan{
		IndexName: indexName,
		Index:     idx,
		Field:     field,
	}

	// Check if it's an operator expression
	if operatorMap, ok := value.(map[string]interface{}); ok {
		// Handle different operators
		for opStr, opValue := range operatorMap {
			switch opStr {
			case "$eq":
				plan.ScanType = ScanTypeIndexExact
				plan.ScanKey = opValue
				return plan

			case "$gt", "$gte":
				plan.ScanType = ScanTypeIndexRange
				if opStr == "$gte" {
					plan.ScanStart = opValue
				} else {
					plan.ScanStart = opValue
				}
				return plan

			case "$lt", "$lte":
				plan.ScanType = ScanTypeIndexRange
				if opStr == "$lte" {
					plan.ScanEnd = opValue
				} else {
					plan.ScanEnd = opValue
				}
				return plan

			default:
				// Unsupported operator for intersection
				return nil
			}
		}
	}

	// Direct value comparison (implicit $eq)
	plan.ScanType = ScanTypeIndexExact
	plan.ScanKey = value
	return plan
}

// estimateIntersectionCost estimates the cost of using index intersection
func (qp *QueryPlanner) estimateIntersectionCost(plans []*IndexIntersectPlan) int {
	if len(plans) == 0 {
		return 1000000
	}

	// Get statistics for each index
	costs := make([]int, len(plans))
	cardinalities := make([]int, len(plans))

	for i, plan := range plans {
		stats := plan.Index.GetStatistics()
		totalEntries, uniqueKeys, _, _, isStale := stats.GetStats()

		if isStale {
			// Use default estimates if stats are stale
			costs[i] = 100
			cardinalities[i] = 100
			continue
		}

		// Estimate cardinality (number of results) from this index scan
		switch plan.ScanType {
		case ScanTypeIndexExact:
			// Exact match - estimate based on uniqueness
			if uniqueKeys > 0 {
				cardinalities[i] = totalEntries / uniqueKeys
			} else {
				cardinalities[i] = totalEntries
			}
			costs[i] = 5 // Low cost for exact lookup

		case ScanTypeIndexRange:
			// Range scan - estimate 30% of total entries
			cardinalities[i] = int(float64(totalEntries) * 0.3)
			costs[i] = cardinalities[i] / 10 // Cost proportional to results
			if costs[i] < 10 {
				costs[i] = 10
			}

		default:
			cardinalities[i] = totalEntries
			costs[i] = 100
		}
	}

	// Total cost = sum of index scans + intersection overhead
	totalScanCost := 0
	for _, cost := range costs {
		totalScanCost += cost
	}

	// Intersection overhead = size of smallest result set (we intersect sets)
	minCardinality := cardinalities[0]
	for _, card := range cardinalities {
		if card < minCardinality {
			minCardinality = card
		}
	}

	// Add overhead for set operations
	intersectionOverhead := minCardinality / 2
	if intersectionOverhead < 5 {
		intersectionOverhead = 5
	}

	return totalScanCost + intersectionOverhead
}
