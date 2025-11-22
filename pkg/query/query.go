package query

import (
	"fmt"

	"github.com/mnohosten/laura-db/pkg/document"
)

// Query represents a database query
type Query struct {
	filter     map[string]interface{}
	projection map[string]bool
	sort       []SortField
	limit      int
	skip       int
}

// SortField represents a field to sort by
type SortField struct {
	Field     string
	Ascending bool
}

// NewQuery creates a new query
func NewQuery(filter map[string]interface{}) *Query {
	return &Query{
		filter:     filter,
		projection: nil,
		sort:       nil,
		limit:      0,
		skip:       0,
	}
}

// WithProjection sets the projection
func (q *Query) WithProjection(projection map[string]bool) *Query {
	q.projection = projection
	return q
}

// WithSort sets the sort order
func (q *Query) WithSort(fields []SortField) *Query {
	q.sort = fields
	return q
}

// WithLimit sets the limit
func (q *Query) WithLimit(limit int) *Query {
	q.limit = limit
	return q
}

// WithSkip sets the skip
func (q *Query) WithSkip(skip int) *Query {
	q.skip = skip
	return q
}

// Matches checks if a document matches the query filter
func (q *Query) Matches(doc *document.Document) (bool, error) {
	if len(q.filter) == 0 {
		return true, nil // Empty filter matches all
	}

	return q.evaluateFilter(doc, q.filter)
}

// evaluateFilter evaluates a filter against a document
func (q *Query) evaluateFilter(doc *document.Document, filter map[string]interface{}) (bool, error) {
	for key, value := range filter {
		// Check for logical operators
		if key == string(OpAnd) {
			result, err := q.evaluateAnd(doc, value)
			if err != nil || !result {
				return false, err
			}
			continue
		}

		if key == string(OpOr) {
			result, err := q.evaluateOr(doc, value)
			if err != nil || !result {
				return false, err
			}
			continue
		}

		// Field comparison
		fieldValue, exists := doc.Get(key)

		// Handle operator expressions
		if operatorMap, ok := value.(map[string]interface{}); ok {
			for opStr, opValue := range operatorMap {
				op := Operator(opStr)

				// Special case for $exists
				if op == OpExists {
					result, err := EvaluateOperator(op, fieldValue, opValue)
					if err != nil {
						return false, err
					}
					if !result {
						return false, nil
					}
					continue
				}

				// For other operators, field must exist
				if !exists {
					return false, nil
				}

				result, err := EvaluateOperator(op, fieldValue, opValue)
				if err != nil {
					return false, err
				}
				if !result {
					return false, nil
				}
			}
		} else {
			// Direct equality comparison
			if !exists {
				return false, nil
			}
			if !evaluateEqual(fieldValue, value) {
				return false, nil
			}
		}
	}

	return true, nil
}

// evaluateAnd evaluates $and operator
func (q *Query) evaluateAnd(doc *document.Document, value interface{}) (bool, error) {
	conditions, ok := value.([]interface{})
	if !ok {
		return false, fmt.Errorf("$and requires an array of conditions")
	}

	for _, condition := range conditions {
		condMap, ok := condition.(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("invalid condition in $and")
		}

		result, err := q.evaluateFilter(doc, condMap)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}

	return true, nil
}

// evaluateOr evaluates $or operator
func (q *Query) evaluateOr(doc *document.Document, value interface{}) (bool, error) {
	conditions, ok := value.([]interface{})
	if !ok {
		return false, fmt.Errorf("$or requires an array of conditions")
	}

	for _, condition := range conditions {
		condMap, ok := condition.(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("invalid condition in $or")
		}

		result, err := q.evaluateFilter(doc, condMap)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}

	return false, nil
}

// ApplyProjection applies the projection to a document
func (q *Query) ApplyProjection(doc *document.Document) *document.Document {
	if q.projection == nil || len(q.projection) == 0 {
		return doc
	}

	result := document.NewDocument()

	// Determine if this is an inclusion or exclusion projection
	isInclusion := false
	for _, include := range q.projection {
		if include {
			isInclusion = true
			break
		}
	}

	if isInclusion {
		// Include only specified fields
		for field, include := range q.projection {
			if include {
				if value, exists := doc.Get(field); exists {
					result.Set(field, value)
				}
			}
		}
	} else {
		// Exclude specified fields
		for _, key := range doc.Keys() {
			if exclude, exists := q.projection[key]; !exists || !exclude {
				if value, exists := doc.Get(key); exists {
					result.Set(key, value)
				}
			}
		}
	}

	return result
}

// GetFilter returns the filter
func (q *Query) GetFilter() map[string]interface{} {
	return q.filter
}

// GetLimit returns the limit
func (q *Query) GetLimit() int {
	return q.limit
}

// GetSkip returns the skip
func (q *Query) GetSkip() int {
	return q.skip
}

// GetSort returns the sort fields
func (q *Query) GetSort() []SortField {
	return q.sort
}

// GetProjection returns the projection
func (q *Query) GetProjection() map[string]bool {
	return q.projection
}
