package aggregation

import (
	"fmt"
	"sort"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/query"
)

// Pipeline represents an aggregation pipeline
type Pipeline struct {
	stages []Stage
}

// Stage represents a single stage in the pipeline
type Stage interface {
	Execute(docs []*document.Document) ([]*document.Document, error)
	Type() string
}

// NewPipeline creates a new aggregation pipeline
func NewPipeline(stages []map[string]interface{}) (*Pipeline, error) {
	pipeline := &Pipeline{
		stages: make([]Stage, 0, len(stages)),
	}

	for _, stageDef := range stages {
		stage, err := createStage(stageDef)
		if err != nil {
			return nil, err
		}
		pipeline.stages = append(pipeline.stages, stage)
	}

	return pipeline, nil
}

// Execute executes the pipeline
func (p *Pipeline) Execute(docs []*document.Document) ([]*document.Document, error) {
	result := docs

	for _, stage := range p.stages {
		var err error
		result, err = stage.Execute(result)
		if err != nil {
			return nil, fmt.Errorf("stage %s failed: %w", stage.Type(), err)
		}
	}

	return result, nil
}

// createStage creates a stage from a definition
func createStage(stageDef map[string]interface{}) (Stage, error) {
	for stageType, stageSpec := range stageDef {
		switch stageType {
		case "$match":
			return newMatchStage(stageSpec)
		case "$project":
			return newProjectStage(stageSpec)
		case "$sort":
			return newSortStage(stageSpec)
		case "$limit":
			return newLimitStage(stageSpec)
		case "$skip":
			return newSkipStage(stageSpec)
		case "$group":
			return newGroupStage(stageSpec)
		default:
			return nil, fmt.Errorf("unsupported stage type: %s", stageType)
		}
	}
	return nil, fmt.Errorf("empty stage definition")
}

// MatchStage filters documents
type MatchStage struct {
	query *query.Query
}

func newMatchStage(spec interface{}) (*MatchStage, error) {
	filter, ok := spec.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("$match requires a filter object")
	}

	return &MatchStage{
		query: query.NewQuery(filter),
	}, nil
}

func (s *MatchStage) Execute(docs []*document.Document) ([]*document.Document, error) {
	result := make([]*document.Document, 0)
	for _, doc := range docs {
		matches, err := s.query.Matches(doc)
		if err != nil {
			return nil, err
		}
		if matches {
			result = append(result, doc)
		}
	}
	return result, nil
}

func (s *MatchStage) Type() string {
	return "$match"
}

// ProjectStage selects fields
type ProjectStage struct {
	projection map[string]interface{}
}

func newProjectStage(spec interface{}) (*ProjectStage, error) {
	projection, ok := spec.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("$project requires a projection object")
	}

	return &ProjectStage{
		projection: projection,
	}, nil
}

func (s *ProjectStage) Execute(docs []*document.Document) ([]*document.Document, error) {
	result := make([]*document.Document, 0, len(docs))

	for _, doc := range docs {
		projected := document.NewDocument()

		for field, spec := range s.projection {
			if include, ok := spec.(bool); ok && include {
				// Include field
				if value, exists := doc.Get(field); exists {
					projected.Set(field, value)
				}
			} else if include, ok := spec.(int); ok && include == 1 {
				// Include field (MongoDB style)
				if value, exists := doc.Get(field); exists {
					projected.Set(field, value)
				}
			} else {
				// Computed field (simplified - just copy for now)
				if value, exists := doc.Get(field); exists {
					projected.Set(field, value)
				}
			}
		}

		result = append(result, projected)
	}

	return result, nil
}

func (s *ProjectStage) Type() string {
	return "$project"
}

// SortStage sorts documents
type SortStage struct {
	sortFields []query.SortField
}

func newSortStage(spec interface{}) (*SortStage, error) {
	sortSpec, ok := spec.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("$sort requires a sort specification")
	}

	sortFields := make([]query.SortField, 0)
	for field, order := range sortSpec {
		ascending := true
		if orderInt, ok := order.(int); ok {
			ascending = orderInt >= 0
		} else if orderInt, ok := order.(int64); ok {
			ascending = orderInt >= 0
		}

		sortFields = append(sortFields, query.SortField{
			Field:     field,
			Ascending: ascending,
		})
	}

	return &SortStage{
		sortFields: sortFields,
	}, nil
}

func (s *SortStage) Execute(docs []*document.Document) ([]*document.Document, error) {
	// Create a copy to avoid modifying the original slice
	result := make([]*document.Document, len(docs))
	copy(result, docs)

	sort.Slice(result, func(i, j int) bool {
		for _, field := range s.sortFields {
			vi, existsI := result[i].Get(field.Field)
			vj, existsJ := result[j].Get(field.Field)

			if !existsI && !existsJ {
				continue
			}
			if !existsI {
				return !field.Ascending
			}
			if !existsJ {
				return field.Ascending
			}

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

	return result, nil
}

func (s *SortStage) Type() string {
	return "$sort"
}

// LimitStage limits the number of documents
type LimitStage struct {
	limit int
}

func newLimitStage(spec interface{}) (*LimitStage, error) {
	var limit int

	switch v := spec.(type) {
	case int:
		limit = v
	case int64:
		limit = int(v)
	case float64:
		limit = int(v)
	default:
		return nil, fmt.Errorf("$limit requires a number")
	}

	return &LimitStage{limit: limit}, nil
}

func (s *LimitStage) Execute(docs []*document.Document) ([]*document.Document, error) {
	if s.limit >= len(docs) {
		return docs, nil
	}
	return docs[:s.limit], nil
}

func (s *LimitStage) Type() string {
	return "$limit"
}

// SkipStage skips documents
type SkipStage struct {
	skip int
}

func newSkipStage(spec interface{}) (*SkipStage, error) {
	var skip int

	switch v := spec.(type) {
	case int:
		skip = v
	case int64:
		skip = int(v)
	case float64:
		skip = int(v)
	default:
		return nil, fmt.Errorf("$skip requires a number")
	}

	return &SkipStage{skip: skip}, nil
}

func (s *SkipStage) Execute(docs []*document.Document) ([]*document.Document, error) {
	if s.skip >= len(docs) {
		return []*document.Document{}, nil
	}
	return docs[s.skip:], nil
}

func (s *SkipStage) Type() string {
	return "$skip"
}

// GroupStage groups documents
type GroupStage struct {
	id     interface{}
	fields map[string]interface{}
}

func newGroupStage(spec interface{}) (*GroupStage, error) {
	groupSpec, ok := spec.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("$group requires a group specification")
	}

	id, exists := groupSpec["_id"]
	if !exists {
		return nil, fmt.Errorf("$group requires an _id field")
	}

	fields := make(map[string]interface{})
	for k, v := range groupSpec {
		if k != "_id" {
			fields[k] = v
		}
	}

	return &GroupStage{
		id:     id,
		fields: fields,
	}, nil
}

func (s *GroupStage) Execute(docs []*document.Document) ([]*document.Document, error) {
	// Group documents by _id field
	groups := make(map[interface{}][]*document.Document)

	for _, doc := range docs {
		groupKey := s.extractGroupKey(doc)
		groups[groupKey] = append(groups[groupKey], doc)
	}

	// Create result documents
	result := make([]*document.Document, 0, len(groups))
	for groupKey, groupDocs := range groups {
		groupDoc := document.NewDocument()
		groupDoc.Set("_id", groupKey)

		// Compute aggregations
		for fieldName, aggSpec := range s.fields {
			value, err := s.computeAggregation(aggSpec, groupDocs)
			if err != nil {
				return nil, err
			}
			groupDoc.Set(fieldName, value)
		}

		result = append(result, groupDoc)
	}

	return result, nil
}

func (s *GroupStage) extractGroupKey(doc *document.Document) interface{} {
	if idStr, ok := s.id.(string); ok {
		if len(idStr) > 0 && idStr[0] == '$' {
			// Field reference
			fieldName := idStr[1:]
			if value, exists := doc.Get(fieldName); exists {
				return value
			}
		}
		return idStr
	}
	return s.id
}

func (s *GroupStage) computeAggregation(aggSpec interface{}, docs []*document.Document) (interface{}, error) {
	if aggMap, ok := aggSpec.(map[string]interface{}); ok {
		for op, fieldRef := range aggMap {
			switch op {
			case "$sum":
				return s.computeSum(fieldRef, docs), nil
			case "$avg":
				return s.computeAvg(fieldRef, docs), nil
			case "$min":
				return s.computeMin(fieldRef, docs), nil
			case "$max":
				return s.computeMax(fieldRef, docs), nil
			case "$count":
				return int64(len(docs)), nil
			}
		}
	}
	return nil, fmt.Errorf("unsupported aggregation operator")
}

func (s *GroupStage) computeSum(fieldRef interface{}, docs []*document.Document) float64 {
	sum := 0.0

	if fieldStr, ok := fieldRef.(string); ok && len(fieldStr) > 0 && fieldStr[0] == '$' {
		fieldName := fieldStr[1:]
		for _, doc := range docs {
			if value, exists := doc.Get(fieldName); exists {
				if num, ok := toFloat64(value); ok {
					sum += num
				}
			}
		}
	} else if num, ok := toFloat64(fieldRef); ok {
		sum = num * float64(len(docs))
	}

	return sum
}

func (s *GroupStage) computeAvg(fieldRef interface{}, docs []*document.Document) float64 {
	if len(docs) == 0 {
		return 0
	}
	return s.computeSum(fieldRef, docs) / float64(len(docs))
}

func (s *GroupStage) computeMin(fieldRef interface{}, docs []*document.Document) interface{} {
	if fieldStr, ok := fieldRef.(string); ok && len(fieldStr) > 0 && fieldStr[0] == '$' {
		fieldName := fieldStr[1:]
		var min interface{}

		for _, doc := range docs {
			if value, exists := doc.Get(fieldName); exists {
				if min == nil || compareValues(value, min) < 0 {
					min = value
				}
			}
		}
		return min
	}
	return nil
}

func (s *GroupStage) computeMax(fieldRef interface{}, docs []*document.Document) interface{} {
	if fieldStr, ok := fieldRef.(string); ok && len(fieldStr) > 0 && fieldStr[0] == '$' {
		fieldName := fieldStr[1:]
		var max interface{}

		for _, doc := range docs {
			if value, exists := doc.Get(fieldName); exists {
				if max == nil || compareValues(value, max) > 0 {
					max = value
				}
			}
		}
		return max
	}
	return nil
}

func (s *GroupStage) Type() string {
	return "$group"
}

// Helper functions

func compareValues(a, b interface{}) int {
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

	return 0
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	default:
		return 0, false
	}
}
