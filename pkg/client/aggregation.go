package client

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// AggregationPipeline represents an aggregation pipeline
type AggregationPipeline []map[string]interface{}

// Aggregate executes an aggregation pipeline
func (c *Collection) Aggregate(pipeline AggregationPipeline) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/%s/_aggregate", url.PathEscape(c.name))

	req := map[string]interface{}{
		"pipeline": pipeline,
	}

	resp, err := c.client.doRequest("POST", path, req)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(resp.Result, &results); err != nil {
		return nil, fmt.Errorf("failed to parse aggregation results: %w", err)
	}

	return results, nil
}

// NewPipeline creates a new aggregation pipeline builder
func NewPipeline() *PipelineBuilder {
	return &PipelineBuilder{
		stages: make([]map[string]interface{}, 0),
	}
}

// PipelineBuilder helps build aggregation pipelines
type PipelineBuilder struct {
	stages []map[string]interface{}
}

// Match adds a $match stage to filter documents
func (pb *PipelineBuilder) Match(filter map[string]interface{}) *PipelineBuilder {
	pb.stages = append(pb.stages, map[string]interface{}{
		"$match": filter,
	})
	return pb
}

// Group adds a $group stage to group documents
func (pb *PipelineBuilder) Group(id interface{}, accumulators map[string]interface{}) *PipelineBuilder {
	groupStage := map[string]interface{}{
		"_id": id,
	}
	for k, v := range accumulators {
		groupStage[k] = v
	}
	pb.stages = append(pb.stages, map[string]interface{}{
		"$group": groupStage,
	})
	return pb
}

// Project adds a $project stage to shape documents
func (pb *PipelineBuilder) Project(projection map[string]interface{}) *PipelineBuilder {
	pb.stages = append(pb.stages, map[string]interface{}{
		"$project": projection,
	})
	return pb
}

// Sort adds a $sort stage to order documents
func (pb *PipelineBuilder) Sort(sort map[string]interface{}) *PipelineBuilder {
	pb.stages = append(pb.stages, map[string]interface{}{
		"$sort": sort,
	})
	return pb
}

// Limit adds a $limit stage to limit the number of documents
func (pb *PipelineBuilder) Limit(limit int) *PipelineBuilder {
	pb.stages = append(pb.stages, map[string]interface{}{
		"$limit": limit,
	})
	return pb
}

// Skip adds a $skip stage to skip documents
func (pb *PipelineBuilder) Skip(skip int) *PipelineBuilder {
	pb.stages = append(pb.stages, map[string]interface{}{
		"$skip": skip,
	})
	return pb
}

// Build returns the completed pipeline
func (pb *PipelineBuilder) Build() AggregationPipeline {
	return pb.stages
}

// Execute runs the pipeline on the given collection
func (pb *PipelineBuilder) Execute(coll *Collection) ([]map[string]interface{}, error) {
	return coll.Aggregate(pb.Build())
}

// Aggregation helper functions for common operations

// Sum creates a $sum accumulator
func Sum(field string) map[string]interface{} {
	return map[string]interface{}{
		"$sum": "$" + field,
	}
}

// SumValue creates a $sum accumulator with a constant value
func SumValue(value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"$sum": value,
	}
}

// Avg creates an $avg accumulator
func Avg(field string) map[string]interface{} {
	return map[string]interface{}{
		"$avg": "$" + field,
	}
}

// Min creates a $min accumulator
func Min(field string) map[string]interface{} {
	return map[string]interface{}{
		"$min": "$" + field,
	}
}

// Max creates a $max accumulator
func Max(field string) map[string]interface{} {
	return map[string]interface{}{
		"$max": "$" + field,
	}
}

// Push creates a $push accumulator
func Push(field string) map[string]interface{} {
	return map[string]interface{}{
		"$push": "$" + field,
	}
}

// Count creates a count accumulator (sum of 1)
func Count() map[string]interface{} {
	return map[string]interface{}{
		"$sum": 1,
	}
}
