package database

import (
	"fmt"
	"sync"
	"time"

	"github.com/mnohosten/laura-db/pkg/aggregation"
	"github.com/mnohosten/laura-db/pkg/cache"
	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/index"
	"github.com/mnohosten/laura-db/pkg/mvcc"
	"github.com/mnohosten/laura-db/pkg/query"
)

// Collection represents a collection of documents
type Collection struct {
	name       string
	documents  map[string]*document.Document // _id -> document
	indexes    map[string]*index.Index       // index name -> index
	txnMgr     *mvcc.TransactionManager
	queryCache *cache.LRUCache               // Query result cache
	mu         sync.RWMutex
}

// NewCollection creates a new collection
func NewCollection(name string, txnMgr *mvcc.TransactionManager) *Collection {
	coll := &Collection{
		name:       name,
		documents:  make(map[string]*document.Document),
		indexes:    make(map[string]*index.Index),
		txnMgr:     txnMgr,
		queryCache: cache.NewLRUCache(1000, 5*time.Minute), // 1000 queries, 5min TTL
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

	// Invalidate query cache on write
	c.queryCache.Clear()

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

	// Generate cache key from query parameters
	var sort []interface{}
	var projection map[string]bool
	skip := 0
	limit := 0

	if options.Sort != nil {
		// Convert []query.SortField to []interface{}
		sort = make([]interface{}, len(options.Sort))
		for i, sf := range options.Sort {
			sort[i] = map[string]interface{}{
				"field":     sf.Field,
				"ascending": sf.Ascending,
			}
		}
	}
	if options.Projection != nil {
		projection = options.Projection
	}
	if options.Skip > 0 {
		skip = options.Skip
	}
	if options.Limit > 0 {
		limit = options.Limit
	}

	cacheKey := cache.GenerateKey(filter, sort, skip, limit, projection)

	// Check cache
	if cached, found := c.queryCache.Get(cacheKey); found {
		// Cache hit - return cached results
		if results, ok := cached.([]*document.Document); ok {
			return results, nil
		}
	}

	// Cache miss - execute query
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

	results, err := c.executeQuery(q)
	if err != nil {
		return nil, err
	}

	// Store in cache
	c.queryCache.Put(cacheKey, results)

	return results, nil
}

// executeQuery executes a query with query planning and index optimization
func (c *Collection) executeQuery(q *query.Query) ([]*document.Document, error) {
	// Create query planner
	planner := query.NewQueryPlanner(c.indexes)

	// Generate execution plan
	plan := planner.Plan(q)

	// Detect if query can be covered by index
	planner.DetectCoveredQuery(plan, q.GetProjection())

	// Create executor with document map for efficient lookups
	executor := query.NewExecutorWithMap(c.documents)

	// Execute with plan (will use index if beneficial)
	return executor.ExecuteWithPlan(q, plan)
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
	if err := c.applyUpdate(doc, update); err != nil {
		return err
	}

	// Invalidate query cache on write
	c.queryCache.Clear()

	return nil
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

	// Invalidate query cache on write
	c.queryCache.Clear()

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
		} else if key == "$mul" {
			// $mul operator - multiply field by value
			if mulMap, ok := value.(map[string]interface{}); ok {
				for field, mulVal := range mulMap {
					if currentVal, exists := doc.Get(field); exists {
						if currentNum, ok := toFloat64(currentVal); ok {
							if mulNum, ok := toFloat64(mulVal); ok {
								doc.Set(field, currentNum*mulNum)
							}
						}
					} else {
						// Field doesn't exist, set to 0 (MongoDB behavior)
						doc.Set(field, 0)
					}
				}
			}
		} else if key == "$min" {
			// $min operator - update field if value is less than current
			if minMap, ok := value.(map[string]interface{}); ok {
				for field, minVal := range minMap {
					if currentVal, exists := doc.Get(field); exists {
						if currentNum, ok := toFloat64(currentVal); ok {
							if minNum, ok := toFloat64(minVal); ok {
								if minNum < currentNum {
									doc.Set(field, minNum)
								}
							}
						}
					} else {
						// Field doesn't exist, set to minVal (MongoDB behavior)
						doc.Set(field, minVal)
					}
				}
			}
		} else if key == "$max" {
			// $max operator - update field if value is greater than current
			if maxMap, ok := value.(map[string]interface{}); ok {
				for field, maxVal := range maxMap {
					if currentVal, exists := doc.Get(field); exists {
						if currentNum, ok := toFloat64(currentVal); ok {
							if maxNum, ok := toFloat64(maxVal); ok {
								if maxNum > currentNum {
									doc.Set(field, maxNum)
								}
							}
						}
					} else {
						// Field doesn't exist, set to maxVal (MongoDB behavior)
						doc.Set(field, maxVal)
					}
				}
			}
		} else if key == "$push" {
			// $push operator - add element(s) to array
			if pushMap, ok := value.(map[string]interface{}); ok {
				for field, pushVal := range pushMap {
					// Check if using $each modifier for bulk push
					var valuesToPush []interface{}
					if modifierMap, ok := pushVal.(map[string]interface{}); ok {
						if eachValues, hasEach := modifierMap["$each"]; hasEach {
							// $each modifier - push multiple values
							if eachArray, ok := eachValues.([]interface{}); ok {
								valuesToPush = eachArray
							}
						}
					}

					// If no $each modifier, push single value
					if valuesToPush == nil {
						valuesToPush = []interface{}{pushVal}
					}

					if currentVal, exists := doc.Get(field); exists {
						// Field exists, append to array
						if arr, ok := currentVal.([]interface{}); ok {
							arr = append(arr, valuesToPush...)
							doc.Set(field, arr)
						}
					} else {
						// Field doesn't exist, create new array
						doc.Set(field, valuesToPush)
					}
				}
			}
		} else if key == "$pull" {
			// $pull operator - remove elements matching value
			if pullMap, ok := value.(map[string]interface{}); ok {
				for field, pullVal := range pullMap {
					if currentVal, exists := doc.Get(field); exists {
						if arr, ok := currentVal.([]interface{}); ok {
							newArr := make([]interface{}, 0)
							for _, elem := range arr {
								if !compareValues2(elem, pullVal) {
									newArr = append(newArr, elem)
								}
							}
							doc.Set(field, newArr)
						}
					}
				}
			}
		} else if key == "$addToSet" {
			// $addToSet operator - add element(s) only if not already in array
			if addMap, ok := value.(map[string]interface{}); ok {
				for field, addVal := range addMap {
					// Check if using $each modifier for bulk addToSet
					var valuesToAdd []interface{}
					if modifierMap, ok := addVal.(map[string]interface{}); ok {
						if eachValues, hasEach := modifierMap["$each"]; hasEach {
							// $each modifier - add multiple unique values
							if eachArray, ok := eachValues.([]interface{}); ok {
								valuesToAdd = eachArray
							}
						}
					}

					// If no $each modifier, add single value
					if valuesToAdd == nil {
						valuesToAdd = []interface{}{addVal}
					}

					if currentVal, exists := doc.Get(field); exists {
						// Field exists, check each value and add if not present
						if arr, ok := currentVal.([]interface{}); ok {
							for _, val := range valuesToAdd {
								found := false
								for _, elem := range arr {
									if compareValues2(elem, val) {
										found = true
										break
									}
								}
								if !found {
									arr = append(arr, val)
								}
							}
							doc.Set(field, arr)
						}
					} else {
						// Field doesn't exist, create new array with unique values
						uniqueVals := make([]interface{}, 0)
						for _, val := range valuesToAdd {
							found := false
							for _, existing := range uniqueVals {
								if compareValues2(existing, val) {
									found = true
									break
								}
							}
							if !found {
								uniqueVals = append(uniqueVals, val)
							}
						}
						doc.Set(field, uniqueVals)
					}
				}
			}
		} else if key == "$pop" {
			// $pop operator - remove first (-1) or last (1) element from array
			if popMap, ok := value.(map[string]interface{}); ok {
				for field, popVal := range popMap {
					if currentVal, exists := doc.Get(field); exists {
						if arr, ok := currentVal.([]interface{}); ok {
							if len(arr) > 0 {
								if popInt, ok := toFloat64(popVal); ok {
									if popInt < 0 {
										// Remove first element
										doc.Set(field, arr[1:])
									} else {
										// Remove last element
										doc.Set(field, arr[:len(arr)-1])
									}
								}
							}
						}
					}
				}
			}
		} else if key == "$rename" {
			// $rename operator - rename a field
			if renameMap, ok := value.(map[string]interface{}); ok {
				for oldField, newFieldVal := range renameMap {
					if newField, ok := newFieldVal.(string); ok {
						// Get value from old field
						if val, exists := doc.Get(oldField); exists {
							// Set new field
							doc.Set(newField, val)
							// Delete old field
							doc.Delete(oldField)
						}
					}
				}
			}
		} else if key == "$currentDate" {
			// $currentDate operator - set field to current date/time
			if dateMap, ok := value.(map[string]interface{}); ok {
				for field, typeSpec := range dateMap {
					// Check if user wants timestamp or date (default is date)
					useTimestamp := false
					if specMap, ok := typeSpec.(map[string]interface{}); ok {
						if typeVal, ok := specMap["$type"]; ok {
							if typeStr, ok := typeVal.(string); ok {
								useTimestamp = (typeStr == "timestamp")
							}
						}
					}

					// Set current time
					if useTimestamp {
						doc.Set(field, time.Now().Unix())
					} else {
						doc.Set(field, time.Now())
					}
				}
			}
		} else if key == "$pullAll" {
			// $pullAll operator - remove all instances of multiple values from array
			if pullAllMap, ok := value.(map[string]interface{}); ok {
				for field, pullValues := range pullAllMap {
					if currentVal, exists := doc.Get(field); exists {
						if arr, ok := currentVal.([]interface{}); ok {
							if valuesToRemove, ok := pullValues.([]interface{}); ok {
								// Create a map for O(1) lookup
								removeMap := make(map[interface{}]bool)
								for _, v := range valuesToRemove {
									removeMap[v] = true
								}

								// Filter array
								newArr := make([]interface{}, 0)
								for _, elem := range arr {
									// Check if element should be removed
									shouldRemove := false
									for removeVal := range removeMap {
										if compareValues2(elem, removeVal) {
											shouldRemove = true
											break
										}
									}
									if !shouldRemove {
										newArr = append(newArr, elem)
									}
								}
								doc.Set(field, newArr)
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

	// Invalidate query cache on write
	c.queryCache.Clear()

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

	// Invalidate query cache on write
	c.queryCache.Clear()

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

// Explain returns the execution plan for a query
func (c *Collection) Explain(filter map[string]interface{}) map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create query
	q := query.NewQuery(filter)

	// Create query planner
	planner := query.NewQueryPlanner(c.indexes)

	// Generate execution plan
	plan := planner.Plan(q)

	// Get plan explanation
	explanation := plan.Explain()

	// Add collection info
	explanation["collection"] = c.name
	explanation["totalDocuments"] = len(c.documents)
	explanation["availableIndexes"] = make([]string, 0, len(c.indexes))
	for indexName := range c.indexes {
		explanation["availableIndexes"] = append(explanation["availableIndexes"].([]string), indexName)
	}

	return explanation
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

// compareValues2 compares two values for equality (for array operations)
func compareValues2(a, b interface{}) bool {
	// Try numeric comparison
	aVal, aOk := toFloat64(a)
	bVal, bOk := toFloat64(b)
	if aOk && bOk {
		return aVal == bVal
	}

	// String comparison
	aStr, aOk := a.(string)
	bStr, bOk := b.(string)
	if aOk && bOk {
		return aStr == bStr
	}

	// Boolean comparison
	aBool, aOk := a.(bool)
	bBool, bOk := b.(bool)
	if aOk && bOk {
		return aBool == bBool
	}

	// Direct comparison for other types
	return a == b
}
