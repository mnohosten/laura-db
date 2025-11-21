package database

import (
	"fmt"
	"sync"

	"github.com/mnohosten/laura-db/pkg/aggregation"
	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/index"
	"github.com/mnohosten/laura-db/pkg/mvcc"
	"github.com/mnohosten/laura-db/pkg/query"
)

// Collection represents a collection of documents
type Collection struct {
	name      string
	documents map[string]*document.Document // _id -> document
	indexes   map[string]*index.Index       // index name -> index
	txnMgr    *mvcc.TransactionManager
	mu        sync.RWMutex
}

// NewCollection creates a new collection
func NewCollection(name string, txnMgr *mvcc.TransactionManager) *Collection {
	coll := &Collection{
		name:      name,
		documents: make(map[string]*document.Document),
		indexes:   make(map[string]*index.Index),
		txnMgr:    txnMgr,
	}

	// Create default index on _id
	idIndex := index.NewIndex(&index.IndexConfig{
		Name:      "_id_",
		FieldPath: "_id",
		Type:      index.IndexTypeBTree,
		Unique:    true,
		Order:     32,
	})
	coll.indexes["_id_"] = idIndex

	return coll
}

// InsertOne inserts a single document
func (c *Collection) InsertOne(doc map[string]interface{}) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create document
	d := document.NewDocumentFromMap(doc)

	// Generate _id if not provided
	var id string
	if idVal, exists := d.Get("_id"); exists {
		id = fmt.Sprintf("%v", idVal)
	} else {
		objectID := document.NewObjectID()
		d.Set("_id", objectID)
		id = objectID.Hex()
	}

	// Check if document already exists
	if _, exists := c.documents[id]; exists {
		return "", fmt.Errorf("document with _id %s already exists", id)
	}

	// Insert into indexes
	for _, idx := range c.indexes {
		fieldValue, exists := d.Get(idx.FieldPath())
		if !exists && idx.IsUnique() {
			continue // Skip missing fields for unique indexes
		}
		if exists {
			if err := idx.Insert(fieldValue, id); err != nil {
				return "", fmt.Errorf("failed to insert into index %s: %w", idx.Name(), err)
			}
		}
	}

	// Store document
	c.documents[id] = d

	return id, nil
}

// InsertMany inserts multiple documents
func (c *Collection) InsertMany(docs []map[string]interface{}) ([]string, error) {
	ids := make([]string, 0, len(docs))

	for _, doc := range docs {
		id, err := c.InsertOne(doc)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// FindOne finds a single document matching the filter
func (c *Collection) FindOne(filter map[string]interface{}) (*document.Document, error) {
	results, err := c.Find(filter)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, ErrDocumentNotFound
	}

	return results[0], nil
}

// Find finds all documents matching the filter
func (c *Collection) Find(filter map[string]interface{}) ([]*document.Document, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	q := query.NewQuery(filter)
	return c.executeQuery(q)
}

// FindWithOptions finds documents with query options
func (c *Collection) FindWithOptions(filter map[string]interface{}, options *QueryOptions) ([]*document.Document, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	q := query.NewQuery(filter)

	if options.Projection != nil {
		q.WithProjection(options.Projection)
	}
	if options.Sort != nil {
		q.WithSort(options.Sort)
	}
	if options.Limit > 0 {
		q.WithLimit(options.Limit)
	}
	if options.Skip > 0 {
		q.WithSkip(options.Skip)
	}

	return c.executeQuery(q)
}

// executeQuery executes a query
func (c *Collection) executeQuery(q *query.Query) ([]*document.Document, error) {
	// Get all documents
	docs := make([]*document.Document, 0, len(c.documents))
	for _, doc := range c.documents {
		docs = append(docs, doc)
	}

	// Execute query
	executor := query.NewExecutor(docs)
	return executor.Execute(q)
}

// UpdateOne updates a single document matching the filter
func (c *Collection) UpdateOne(filter map[string]interface{}, update map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find document
	doc, err := c.findOneInternal(filter)
	if err != nil {
		return err
	}

	// Apply update
	return c.applyUpdate(doc, update)
}

// UpdateMany updates all documents matching the filter
func (c *Collection) UpdateMany(filter map[string]interface{}, update map[string]interface{}) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find documents
	docs, err := c.findInternal(filter)
	if err != nil {
		return 0, err
	}

	// Update each document
	count := 0
	for _, doc := range docs {
		if err := c.applyUpdate(doc, update); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

// applyUpdate applies an update to a document
func (c *Collection) applyUpdate(doc *document.Document, update map[string]interface{}) error {
	for key, value := range update {
		if key == "$set" {
			// $set operator
			if setMap, ok := value.(map[string]interface{}); ok {
				for field, val := range setMap {
					doc.Set(field, val)
				}
			}
		} else if key == "$unset" {
			// $unset operator
			if unsetMap, ok := value.(map[string]interface{}); ok {
				for field := range unsetMap {
					doc.Delete(field)
				}
			}
		} else if key == "$inc" {
			// $inc operator
			if incMap, ok := value.(map[string]interface{}); ok {
				for field, incVal := range incMap {
					if currentVal, exists := doc.Get(field); exists {
						if currentNum, ok := toFloat64(currentVal); ok {
							if incNum, ok := toFloat64(incVal); ok {
								doc.Set(field, currentNum+incNum)
							}
						}
					}
				}
			}
		} else {
			// Direct field update
			doc.Set(key, value)
		}
	}

	return nil
}

// DeleteOne deletes a single document matching the filter
func (c *Collection) DeleteOne(filter map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	doc, err := c.findOneInternal(filter)
	if err != nil {
		return err
	}

	idVal, _ := doc.Get("_id")
	id := fmt.Sprintf("%v", idVal)

	// Remove from indexes
	for _, idx := range c.indexes {
		if fieldValue, exists := doc.Get(idx.FieldPath()); exists {
			idx.Delete(fieldValue)
		}
	}

	// Delete document
	delete(c.documents, id)

	return nil
}

// DeleteMany deletes all documents matching the filter
func (c *Collection) DeleteMany(filter map[string]interface{}) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	docs, err := c.findInternal(filter)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, doc := range docs {
		idVal, _ := doc.Get("_id")
		id := fmt.Sprintf("%v", idVal)

		// Remove from indexes
		for _, idx := range c.indexes {
			if fieldValue, exists := doc.Get(idx.FieldPath()); exists {
				idx.Delete(fieldValue)
			}
		}

		delete(c.documents, id)
		count++
	}

	return count, nil
}

// Count returns the number of documents matching the filter
func (c *Collection) Count(filter map[string]interface{}) (int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	docs, err := c.findInternal(filter)
	return len(docs), err
}

// CreateIndex creates an index on a field
func (c *Collection) CreateIndex(fieldPath string, unique bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	indexName := fieldPath + "_1"
	if _, exists := c.indexes[indexName]; exists {
		return fmt.Errorf("index %s already exists", indexName)
	}

	idx := index.NewIndex(&index.IndexConfig{
		Name:      indexName,
		FieldPath: fieldPath,
		Type:      index.IndexTypeBTree,
		Unique:    unique,
		Order:     32,
	})

	// Build index from existing documents
	for id, doc := range c.documents {
		if fieldValue, exists := doc.Get(fieldPath); exists {
			if err := idx.Insert(fieldValue, id); err != nil {
				return fmt.Errorf("failed to build index: %w", err)
			}
		}
	}

	c.indexes[indexName] = idx
	return nil
}

// DropIndex drops an index
func (c *Collection) DropIndex(indexName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if indexName == "_id_" {
		return fmt.Errorf("cannot drop _id index")
	}

	if _, exists := c.indexes[indexName]; !exists {
		return fmt.Errorf("index %s does not exist", indexName)
	}

	delete(c.indexes, indexName)
	return nil
}

// ListIndexes returns all indexes
func (c *Collection) ListIndexes() []map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	indexes := make([]map[string]interface{}, 0, len(c.indexes))
	for _, idx := range c.indexes {
		indexes = append(indexes, idx.Stats())
	}
	return indexes
}

// Aggregate executes an aggregation pipeline
func (c *Collection) Aggregate(pipeline []map[string]interface{}) ([]*document.Document, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Get all documents
	docs := make([]*document.Document, 0, len(c.documents))
	for _, doc := range c.documents {
		docs = append(docs, doc)
	}

	// Create and execute pipeline
	aggPipeline, err := aggregation.NewPipeline(pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	return aggPipeline.Execute(docs)
}

// Name returns the collection name
func (c *Collection) Name() string {
	return c.name
}

// Stats returns collection statistics
func (c *Collection) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"name":          c.name,
		"count":         len(c.documents),
		"indexes":       len(c.indexes),
		"index_details": c.ListIndexes(),
	}
}

// findOneInternal finds one document (caller must hold lock)
func (c *Collection) findOneInternal(filter map[string]interface{}) (*document.Document, error) {
	docs, err := c.findInternal(filter)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, ErrDocumentNotFound
	}
	return docs[0], nil
}

// findInternal finds documents (caller must hold lock)
func (c *Collection) findInternal(filter map[string]interface{}) ([]*document.Document, error) {
	q := query.NewQuery(filter)

	docs := make([]*document.Document, 0, len(c.documents))
	for _, doc := range c.documents {
		docs = append(docs, doc)
	}

	executor := query.NewExecutor(docs)
	return executor.Execute(q)
}

// toFloat64 converts to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}
