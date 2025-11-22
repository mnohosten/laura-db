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
	EstimatedCost int
	FilterSteps   []string // Fields that still need filtering after index scan
	IsCovered     bool     // True if query can be satisfied entirely from index
	IndexedField  string   // The field covered by the index
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
	bestPlan := plan
	for indexName, idx := range qp.indexes {
		indexPlan := qp.analyzeIndexForFilter(indexName, idx, q.filter)
		if indexPlan != nil && indexPlan.EstimatedCost < bestPlan.EstimatedCost {
			bestPlan = indexPlan
		}
	}

	return bestPlan
}

// analyzeIndexForFilter analyzes if an index can be used for a filter
func (qp *QueryPlanner) analyzeIndexForFilter(indexName string, idx *index.Index, filter map[string]interface{}) *QueryPlan {
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

	if plan.UseIndex {
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
