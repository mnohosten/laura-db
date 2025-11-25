package client

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// IndexType represents the type of index
type IndexType string

const (
	IndexTypeBTree    IndexType = "btree"
	IndexTypeCompound IndexType = "compound"
	IndexTypeText     IndexType = "text"
	IndexTypeGeo2D    IndexType = "2d"
	IndexTypeGeo2DSphere IndexType = "2dsphere"
	IndexTypeTTL      IndexType = "ttl"
)

// IndexOptions represents options for creating an index
type IndexOptions struct {
	// Name is the index name (required)
	Name string `json:"name"`
	// Type is the index type (required)
	Type IndexType `json:"type"`
	// Fields specifies the indexed fields with sort order
	// For single-field: {"field": 1} or {"field": -1}
	// For compound: {"field1": 1, "field2": -1}
	Fields map[string]int `json:"fields,omitempty"`
	// Field is a single field name (for non-compound indexes)
	Field string `json:"field,omitempty"`
	// Unique specifies if the index enforces uniqueness
	Unique bool `json:"unique,omitempty"`
	// Sparse specifies if the index is sparse
	Sparse bool `json:"sparse,omitempty"`
	// TTL specifies the time-to-live duration (for TTL indexes)
	// Format: "24h", "1h30m", etc.
	TTL string `json:"ttl,omitempty"`
	// PartialFilter specifies a filter for partial indexes
	PartialFilter map[string]interface{} `json:"partial_filter,omitempty"`
}

// CreateIndex creates an index on the collection
func (c *Collection) CreateIndex(options IndexOptions) error {
	path := fmt.Sprintf("/%s/_index", url.PathEscape(c.name))
	_, err := c.client.doRequest("POST", path, options)
	return err
}

// ListIndexes lists all indexes on the collection
func (c *Collection) ListIndexes() ([]IndexInfo, error) {
	path := fmt.Sprintf("/%s/_index", url.PathEscape(c.name))
	resp, err := c.client.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Indexes []IndexInfo `json:"indexes"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse indexes: %w", err)
	}

	return result.Indexes, nil
}

// DropIndex drops an index by name
func (c *Collection) DropIndex(name string) error {
	path := fmt.Sprintf("/%s/_index/%s", url.PathEscape(c.name), url.PathEscape(name))
	_, err := c.client.doRequest("DELETE", path, nil)
	return err
}

// Index builder helpers for common index types

// CreateBTreeIndex creates a single-field B-tree index
func (c *Collection) CreateBTreeIndex(name, field string, unique bool) error {
	return c.CreateIndex(IndexOptions{
		Name:   name,
		Type:   IndexTypeBTree,
		Field:  field,
		Unique: unique,
	})
}

// CreateCompoundIndex creates a multi-field compound index
func (c *Collection) CreateCompoundIndex(name string, fields map[string]int, unique bool) error {
	return c.CreateIndex(IndexOptions{
		Name:   name,
		Type:   IndexTypeCompound,
		Fields: fields,
		Unique: unique,
	})
}

// CreateTextIndex creates a text search index
func (c *Collection) CreateTextIndex(name, field string) error {
	return c.CreateIndex(IndexOptions{
		Name:  name,
		Type:  IndexTypeText,
		Field: field,
	})
}

// CreateGeo2DIndex creates a 2D geospatial index
func (c *Collection) CreateGeo2DIndex(name, field string) error {
	return c.CreateIndex(IndexOptions{
		Name:  name,
		Type:  IndexTypeGeo2D,
		Field: field,
	})
}

// CreateGeo2DSphereIndex creates a 2dsphere geospatial index
func (c *Collection) CreateGeo2DSphereIndex(name, field string) error {
	return c.CreateIndex(IndexOptions{
		Name:  name,
		Type:  IndexTypeGeo2DSphere,
		Field: field,
	})
}

// CreateTTLIndex creates a TTL (time-to-live) index
func (c *Collection) CreateTTLIndex(name, field string, ttl string) error {
	return c.CreateIndex(IndexOptions{
		Name:  name,
		Type:  IndexTypeTTL,
		Field: field,
		TTL:   ttl,
	})
}

// CreatePartialIndex creates a partial index with a filter
func (c *Collection) CreatePartialIndex(name, field string, filter map[string]interface{}, unique bool) error {
	return c.CreateIndex(IndexOptions{
		Name:          name,
		Type:          IndexTypeBTree,
		Field:         field,
		Unique:        unique,
		PartialFilter: filter,
	})
}
