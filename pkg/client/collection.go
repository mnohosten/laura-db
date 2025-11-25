package client

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// Collection represents a database collection
type Collection struct {
	client *Client
	name   string
}

// Name returns the collection name
func (c *Collection) Name() string {
	return c.name
}

// InsertOne inserts a single document into the collection
func (c *Collection) InsertOne(doc map[string]interface{}) (string, error) {
	path := fmt.Sprintf("/%s/_doc", url.PathEscape(c.name))
	resp, err := c.client.doRequest("POST", path, doc)
	if err != nil {
		return "", err
	}

	var result struct {
		ID string `json:"_id"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("failed to parse insert response: %w", err)
	}

	return result.ID, nil
}

// InsertOneWithID inserts a document with a specific ID
func (c *Collection) InsertOneWithID(id string, doc map[string]interface{}) error {
	path := fmt.Sprintf("/%s/_doc/%s", url.PathEscape(c.name), url.PathEscape(id))
	_, err := c.client.doRequest("POST", path, doc)
	return err
}

// FindOne retrieves a single document by ID
func (c *Collection) FindOne(id string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/%s/_doc/%s", url.PathEscape(c.name), url.PathEscape(id))
	resp, err := c.client.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(resp.Result, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}

	return doc, nil
}

// UpdateOne updates a single document by ID
func (c *Collection) UpdateOne(id string, update map[string]interface{}) error {
	path := fmt.Sprintf("/%s/_doc/%s", url.PathEscape(c.name), url.PathEscape(id))
	_, err := c.client.doRequest("PUT", path, update)
	return err
}

// DeleteOne deletes a single document by ID
func (c *Collection) DeleteOne(id string) error {
	path := fmt.Sprintf("/%s/_doc/%s", url.PathEscape(c.name), url.PathEscape(id))
	_, err := c.client.doRequest("DELETE", path, nil)
	return err
}

// BulkOperation represents a bulk operation
type BulkOperation struct {
	Operation string                 `json:"operation"`
	ID        string                 `json:"_id,omitempty"`
	Document  map[string]interface{} `json:"document,omitempty"`
	Update    map[string]interface{} `json:"update,omitempty"`
}

// BulkResult represents the result of a bulk operation
type BulkResult struct {
	Inserted int      `json:"inserted"`
	Updated  int      `json:"updated"`
	Deleted  int      `json:"deleted"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
}

// Bulk performs multiple operations in a single request
func (c *Collection) Bulk(operations []BulkOperation) (*BulkResult, error) {
	path := fmt.Sprintf("/%s/_bulk", url.PathEscape(c.name))

	req := map[string]interface{}{
		"operations": operations,
	}

	resp, err := c.client.doRequest("POST", path, req)
	if err != nil {
		return nil, err
	}

	var result BulkResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse bulk result: %w", err)
	}

	return &result, nil
}

// SearchOptions represents options for search queries
type SearchOptions struct {
	// Filter is the query filter (e.g., {"age": {"$gt": 25}})
	Filter map[string]interface{} `json:"filter,omitempty"`
	// Projection specifies which fields to include/exclude
	Projection map[string]interface{} `json:"projection,omitempty"`
	// Sort specifies the sort order (e.g., {"age": 1, "name": -1})
	Sort map[string]interface{} `json:"sort,omitempty"`
	// Skip specifies the number of documents to skip
	Skip int `json:"skip,omitempty"`
	// Limit specifies the maximum number of documents to return
	Limit int `json:"limit,omitempty"`
}

// Search performs a query on the collection
func (c *Collection) Search(options *SearchOptions) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/%s/_search", url.PathEscape(c.name))

	if options == nil {
		options = &SearchOptions{}
	}

	resp, err := c.client.doRequest("POST", path, options)
	if err != nil {
		return nil, err
	}

	var docs []map[string]interface{}
	if err := json.Unmarshal(resp.Result, &docs); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	return docs, nil
}

// Find is a convenience method that searches with a filter
func (c *Collection) Find(filter map[string]interface{}) ([]map[string]interface{}, error) {
	return c.Search(&SearchOptions{Filter: filter})
}

// FindWithOptions searches with full options
func (c *Collection) FindWithOptions(filter, projection, sort map[string]interface{}, skip, limit int) ([]map[string]interface{}, error) {
	options := &SearchOptions{
		Filter:     filter,
		Projection: projection,
		Sort:       sort,
		Skip:       skip,
		Limit:      limit,
	}
	return c.Search(options)
}

// Count counts documents matching a filter
func (c *Collection) Count(filter map[string]interface{}) (int, error) {
	path := fmt.Sprintf("/%s/_count", url.PathEscape(c.name))

	var resp *Response
	var err error

	if filter != nil && len(filter) > 0 {
		// POST with filter
		resp, err = c.client.doRequest("POST", path, map[string]interface{}{"filter": filter})
	} else {
		// GET for total count
		resp, err = c.client.doRequest("GET", path, nil)
	}

	if err != nil {
		return 0, err
	}

	var result struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return 0, fmt.Errorf("failed to parse count result: %w", err)
	}

	return result.Count, nil
}

// Stats retrieves collection statistics
func (c *Collection) Stats() (*CollectionStats, error) {
	path := fmt.Sprintf("/%s/_stats", url.PathEscape(c.name))
	resp, err := c.client.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var stats CollectionStats
	if err := json.Unmarshal(resp.Result, &stats); err != nil {
		return nil, fmt.Errorf("failed to parse stats: %w", err)
	}

	return &stats, nil
}

// Drop drops the collection
func (c *Collection) Drop() error {
	return c.client.DropCollection(c.name)
}
