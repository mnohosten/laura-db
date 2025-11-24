package database

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mnohosten/laura-db/pkg/aggregation"
	"github.com/mnohosten/laura-db/pkg/cache"
	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/geo"
	"github.com/mnohosten/laura-db/pkg/index"
	"github.com/mnohosten/laura-db/pkg/mvcc"
	"github.com/mnohosten/laura-db/pkg/query"
)

// Collection represents a collection of documents
type Collection struct {
	name        string
	documents   map[string]*document.Document // _id -> document
	indexes     map[string]*index.Index       // index name -> index
	textIndexes map[string]*index.TextIndex   // text index name -> text index
	geoIndexes  map[string]*index.GeoIndex    // geo index name -> geo index
	ttlIndexes  map[string]*index.TTLIndex    // ttl index name -> ttl index
	txnMgr      *mvcc.TransactionManager
	queryCache  *cache.LRUCache // Query result cache
	mu          sync.RWMutex
}

// NewCollection creates a new collection
func NewCollection(name string, txnMgr *mvcc.TransactionManager) *Collection {
	coll := &Collection{
		name:        name,
		documents:   make(map[string]*document.Document),
		indexes:     make(map[string]*index.Index),
		textIndexes: make(map[string]*index.TextIndex),
		geoIndexes:  make(map[string]*index.GeoIndex),
		ttlIndexes:  make(map[string]*index.TTLIndex),
		txnMgr:      txnMgr,
		queryCache:  cache.NewLRUCache(1000, 5*time.Minute), // 1000 queries, 5min TTL
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
		// Check if document matches partial index filter
		if !c.matchesPartialIndexFilter(d, idx) {
			continue // Skip this index if document doesn't match filter
		}

		if idx.IsCompound() {
			// Compound index
			if compositeKey, allFieldsExist := c.extractCompositeKey(d, idx.FieldPaths()); allFieldsExist {
				if err := idx.Insert(compositeKey, id); err != nil {
					return "", fmt.Errorf("failed to insert into compound index %s: %w", idx.Name(), err)
				}
			} else if !idx.IsUnique() {
				// For non-unique indexes, we could still insert partial keys,
				// but for now we skip documents with missing fields
				continue
			}
		} else {
			// Single-field index
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
	}

	// Insert into text indexes
	for _, textIdx := range c.textIndexes {
		texts := make([]string, 0, len(textIdx.FieldPaths()))
		for _, fieldPath := range textIdx.FieldPaths() {
			if fieldValue, exists := d.Get(fieldPath); exists {
				if str, ok := fieldValue.(string); ok {
					texts = append(texts, str)
				}
			}
		}
		if len(texts) > 0 {
			textIdx.Index(id, texts)
		}
	}

	// Insert into geo indexes
	for _, geoIdx := range c.geoIndexes {
		if fieldValue, exists := d.Get(geoIdx.FieldPath()); exists {
			if pointMap, ok := fieldValue.(map[string]interface{}); ok {
				if point, err := geo.ParseGeoJSONPoint(pointMap); err == nil {
					geoIdx.Index(id, point)
				}
			}
		}
	}

	// Insert into TTL indexes
	for _, ttlIdx := range c.ttlIndexes {
		if fieldValue, exists := d.Get(ttlIdx.FieldPath()); exists {
			var timestamp time.Time
			switch v := fieldValue.(type) {
			case time.Time:
				timestamp = v
			case string:
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					timestamp = t
				}
			case int64:
				timestamp = time.Unix(v, 0)
			}
			if !timestamp.IsZero() {
				ttlIdx.Index(id, timestamp)
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

	// Get document ID for index updates
	idVal, _ := doc.Get("_id")
	id := fmt.Sprintf("%v", idVal)

	// Remove old index entries before update
	for _, idx := range c.indexes {
		if idx.IsCompound() {
			// Compound index
			if compositeKey, allFieldsExist := c.extractCompositeKey(doc, idx.FieldPaths()); allFieldsExist {
				idx.Delete(compositeKey)
			}
		} else {
			// Single-field index
			if fieldValue, exists := doc.Get(idx.FieldPath()); exists {
				idx.Delete(fieldValue)
			}
		}
	}

	// Remove from text indexes before update
	for _, textIdx := range c.textIndexes {
		textIdx.Remove(id)
	}

	// Remove from geo indexes before update
	for _, geoIdx := range c.geoIndexes {
		geoIdx.Remove(id)
	}

	// Remove from TTL indexes before update
	for _, ttlIdx := range c.ttlIndexes {
		ttlIdx.Remove(id)
	}

	// Apply update
	if err := c.applyUpdate(doc, update); err != nil {
		return err
	}

	// Add new index entries after update
	for _, idx := range c.indexes {
		// Check if document matches partial index filter
		if !c.matchesPartialIndexFilter(doc, idx) {
			continue // Skip this index if document doesn't match filter
		}

		if idx.IsCompound() {
			// Compound index
			if compositeKey, allFieldsExist := c.extractCompositeKey(doc, idx.FieldPaths()); allFieldsExist {
				if err := idx.Insert(compositeKey, id); err != nil {
					// Log error but continue - index might have duplicate
					continue
				}
			}
		} else {
			// Single-field index
			if fieldValue, exists := doc.Get(idx.FieldPath()); exists {
				if err := idx.Insert(fieldValue, id); err != nil {
					// Log error but continue - index might have duplicate
					continue
				}
			}
		}
	}

	// Re-index in text indexes after update
	for _, textIdx := range c.textIndexes {
		texts := make([]string, 0, len(textIdx.FieldPaths()))
		for _, fieldPath := range textIdx.FieldPaths() {
			if fieldValue, exists := doc.Get(fieldPath); exists {
				if str, ok := fieldValue.(string); ok {
					texts = append(texts, str)
				}
			}
		}
		if len(texts) > 0 {
			textIdx.Index(id, texts)
		}
	}

	// Re-index in geo indexes after update
	for _, geoIdx := range c.geoIndexes {
		if fieldValue, exists := doc.Get(geoIdx.FieldPath()); exists {
			if pointMap, ok := fieldValue.(map[string]interface{}); ok {
				if point, err := geo.ParseGeoJSONPoint(pointMap); err == nil {
					geoIdx.Index(id, point)
				}
			}
		}
	}

	// Re-index in TTL indexes after update
	for _, ttlIdx := range c.ttlIndexes {
		if fieldValue, exists := doc.Get(ttlIdx.FieldPath()); exists {
			var timestamp time.Time
			switch v := fieldValue.(type) {
			case time.Time:
				timestamp = v
			case string:
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					timestamp = t
				}
			case int64:
				timestamp = time.Unix(v, 0)
			}
			if !timestamp.IsZero() {
				ttlIdx.Index(id, timestamp)
			}
		}
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
		// Get document ID for index updates
		idVal, _ := doc.Get("_id")
		id := fmt.Sprintf("%v", idVal)

		// Remove old index entries before update
		for _, idx := range c.indexes {
			if idx.IsCompound() {
				// Compound index
				if compositeKey, allFieldsExist := c.extractCompositeKey(doc, idx.FieldPaths()); allFieldsExist {
					idx.Delete(compositeKey)
				}
			} else {
				// Single-field index
				if fieldValue, exists := doc.Get(idx.FieldPath()); exists {
					idx.Delete(fieldValue)
				}
			}
		}

		// Remove from text indexes before update
		for _, textIdx := range c.textIndexes {
			textIdx.Remove(id)
		}

		// Remove from geo indexes before update
		for _, geoIdx := range c.geoIndexes {
			geoIdx.Remove(id)
		}

		// Remove from TTL indexes before update
		for _, ttlIdx := range c.ttlIndexes {
			ttlIdx.Remove(id)
		}

		// Apply update
		if err := c.applyUpdate(doc, update); err != nil {
			return count, err
		}

		// Add new index entries after update
		for _, idx := range c.indexes {
			// Check if document matches partial index filter
			if !c.matchesPartialIndexFilter(doc, idx) {
				continue // Skip this index if document doesn't match filter
			}

			if idx.IsCompound() {
				// Compound index
				if compositeKey, allFieldsExist := c.extractCompositeKey(doc, idx.FieldPaths()); allFieldsExist {
					if err := idx.Insert(compositeKey, id); err != nil {
						// Log error but continue - index might have duplicate
						continue
					}
				}
			} else {
				// Single-field index
				if fieldValue, exists := doc.Get(idx.FieldPath()); exists {
					if err := idx.Insert(fieldValue, id); err != nil {
						// Log error but continue - index might have duplicate
						continue
					}
				}
			}
		}

		// Re-index in text indexes after update
		for _, textIdx := range c.textIndexes {
			texts := make([]string, 0, len(textIdx.FieldPaths()))
			for _, fieldPath := range textIdx.FieldPaths() {
				if fieldValue, exists := doc.Get(fieldPath); exists {
					if str, ok := fieldValue.(string); ok {
						texts = append(texts, str)
					}
				}
			}
			if len(texts) > 0 {
				textIdx.Index(id, texts)
			}
		}

		// Re-index in geo indexes after update
		for _, geoIdx := range c.geoIndexes {
			if fieldValue, exists := doc.Get(geoIdx.FieldPath()); exists {
				if pointMap, ok := fieldValue.(map[string]interface{}); ok {
					if point, err := geo.ParseGeoJSONPoint(pointMap); err == nil {
						geoIdx.Index(id, point)
					}
				}
			}
		}

		// Re-index in TTL indexes after update
		for _, ttlIdx := range c.ttlIndexes {
			if fieldValue, exists := doc.Get(ttlIdx.FieldPath()); exists {
				var timestamp time.Time
				switch v := fieldValue.(type) {
				case time.Time:
					timestamp = v
				case string:
					if t, err := time.Parse(time.RFC3339, v); err == nil {
						timestamp = t
					}
				case int64:
					timestamp = time.Unix(v, 0)
				}
				if !timestamp.IsZero() {
					ttlIdx.Index(id, timestamp)
				}
			}
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
		} else if key == "$bit" {
			// $bit operator - perform bitwise operations (and, or, xor)
			if bitMap, ok := value.(map[string]interface{}); ok {
				for field, operations := range bitMap {
					if opMap, ok := operations.(map[string]interface{}); ok {
						if currentVal, exists := doc.Get(field); exists {
							if currentInt, ok := toInt64(currentVal); ok {
								result := currentInt

								// Apply bitwise AND
								if andVal, hasAnd := opMap["and"]; hasAnd {
									if andInt, ok := toInt64(andVal); ok {
										result = result & andInt
									}
								}

								// Apply bitwise OR
								if orVal, hasOr := opMap["or"]; hasOr {
									if orInt, ok := toInt64(orVal); ok {
										result = result | orInt
									}
								}

								// Apply bitwise XOR
								if xorVal, hasXor := opMap["xor"]; hasXor {
									if xorInt, ok := toInt64(xorVal); ok {
										result = result ^ xorInt
									}
								}

								doc.Set(field, result)
							}
						} else {
							// Field doesn't exist, initialize to 0 and apply operations
							result := int64(0)

							// Apply bitwise AND
							if andVal, hasAnd := opMap["and"]; hasAnd {
								if andInt, ok := toInt64(andVal); ok {
									result = result & andInt
								}
							}

							// Apply bitwise OR
							if orVal, hasOr := opMap["or"]; hasOr {
								if orInt, ok := toInt64(orVal); ok {
									result = result | orInt
								}
							}

							// Apply bitwise XOR
							if xorVal, hasXor := opMap["xor"]; hasXor {
								if xorInt, ok := toInt64(xorVal); ok {
									result = result ^ xorInt
								}
							}

							doc.Set(field, result)
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
		if idx.IsCompound() {
			// Compound index
			if compositeKey, allFieldsExist := c.extractCompositeKey(doc, idx.FieldPaths()); allFieldsExist {
				idx.Delete(compositeKey)
			}
		} else {
			// Single-field index
			if fieldValue, exists := doc.Get(idx.FieldPath()); exists {
				idx.Delete(fieldValue)
			}
		}
	}

	// Remove from text indexes
	for _, textIdx := range c.textIndexes {
		textIdx.Remove(id)
	}

	// Remove from geo indexes
	for _, geoIdx := range c.geoIndexes {
		geoIdx.Remove(id)
	}

	// Remove from TTL indexes
	for _, ttlIdx := range c.ttlIndexes {
		ttlIdx.Remove(id)
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
			if idx.IsCompound() {
				// Compound index
				if compositeKey, allFieldsExist := c.extractCompositeKey(doc, idx.FieldPaths()); allFieldsExist {
					idx.Delete(compositeKey)
				}
			} else {
				// Single-field index
				if fieldValue, exists := doc.Get(idx.FieldPath()); exists {
					idx.Delete(fieldValue)
				}
			}
		}

		// Remove from text indexes
		for _, textIdx := range c.textIndexes {
			textIdx.Remove(id)
		}

		// Remove from geo indexes
		for _, geoIdx := range c.geoIndexes {
			geoIdx.Remove(id)
		}

		// Remove from TTL indexes
		for _, ttlIdx := range c.ttlIndexes {
			ttlIdx.Remove(id)
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

// extractCompositeKey extracts values for a compound index from a document
// Returns the composite key and a boolean indicating if all fields were present
func (c *Collection) extractCompositeKey(d *document.Document, fieldPaths []string) (*index.CompositeKey, bool) {
	values := make([]interface{}, 0, len(fieldPaths))
	for _, fieldPath := range fieldPaths {
		if fieldValue, exists := d.Get(fieldPath); exists {
			values = append(values, fieldValue)
		} else {
			return nil, false
		}
	}
	return index.NewCompositeKey(values...), true
}

// matchesPartialIndexFilter checks if a document matches a partial index filter
// Returns true if index has no filter (full index) or document matches the filter
func (c *Collection) matchesPartialIndexFilter(d *document.Document, idx *index.Index) bool {
	if !idx.IsPartial() {
		return true // Full index - all documents match
	}

	// Create query from filter and check if document matches
	q := query.NewQuery(idx.Filter())
	matches, err := q.Matches(d)
	if err != nil {
		// On error, don't index the document
		return false
	}
	return matches
}

// CreateIndex creates an index on a field
func (c *Collection) CreateIndex(fieldPath string, unique bool) error {
	return c.CreateIndexWithBackground(fieldPath, unique, false)
}

// CreateIndexWithBackground creates an index on a field with optional background building
func (c *Collection) CreateIndexWithBackground(fieldPath string, unique bool, background bool) error {
	c.mu.Lock()

	indexName := fieldPath + "_1"
	if _, exists := c.indexes[indexName]; exists {
		c.mu.Unlock()
		return fmt.Errorf("index %s already exists", indexName)
	}

	idx := index.NewIndex(&index.IndexConfig{
		Name:       indexName,
		FieldPath:  fieldPath,
		Type:       index.IndexTypeBTree,
		Unique:     unique,
		Order:      32,
		Background: background,
	})

	// Add index to collection immediately (even if building in background)
	c.indexes[indexName] = idx

	if background {
		// Capture snapshot of documents while holding lock
		snapshots := c.captureSingleFieldSnapshot(idx, fieldPath)
		c.mu.Unlock()
		// Build index in background (after releasing lock)
		c.buildSingleFieldIndexInBackgroundWithSnapshot(idx, snapshots)
	} else {
		// Build index synchronously from existing documents
		for id, doc := range c.documents {
			if fieldValue, exists := doc.Get(fieldPath); exists {
				if err := idx.Insert(fieldValue, id); err != nil {
					c.mu.Unlock()
					return fmt.Errorf("failed to build index: %w", err)
				}
			}
		}
		c.mu.Unlock()
	}

	return nil
}

// CreateCompoundIndex creates a compound index on multiple fields
func (c *Collection) CreateCompoundIndex(fieldPaths []string, unique bool) error {
	return c.CreateCompoundIndexWithBackground(fieldPaths, unique, false)
}

// CreateCompoundIndexWithBackground creates a compound index on multiple fields with optional background building
func (c *Collection) CreateCompoundIndexWithBackground(fieldPaths []string, unique bool, background bool) error {
	c.mu.Lock()

	if len(fieldPaths) == 0 {
		c.mu.Unlock()
		return fmt.Errorf("compound index must have at least one field")
	}

	// Generate index name: field1_field2_..._1
	indexName := ""
	for i, field := range fieldPaths {
		if i > 0 {
			indexName += "_"
		}
		indexName += field
	}
	indexName += "_1"

	if _, exists := c.indexes[indexName]; exists {
		c.mu.Unlock()
		return fmt.Errorf("index %s already exists", indexName)
	}

	idx := index.NewIndex(&index.IndexConfig{
		Name:       indexName,
		FieldPaths: fieldPaths,
		Type:       index.IndexTypeBTree,
		Unique:     unique,
		Order:      32,
		Background: background,
	})

	// Add index to collection immediately (even if building in background)
	c.indexes[indexName] = idx

	if background {
		// Capture snapshot of documents while holding lock
		snapshots := c.captureCompoundIndexSnapshot(idx, fieldPaths)
		c.mu.Unlock()
		// Build index in background (after releasing lock)
		c.buildCompoundIndexInBackgroundWithSnapshot(idx, snapshots)
	} else {
		// Build index synchronously from existing documents
		for id, doc := range c.documents {
			// Extract all field values for the composite key
			values := make([]interface{}, 0, len(fieldPaths))
			allFieldsExist := true

			for _, fieldPath := range fieldPaths {
				if fieldValue, exists := doc.Get(fieldPath); exists {
					values = append(values, fieldValue)
				} else {
					// If any field is missing, skip this document
					allFieldsExist = false
					break
				}
			}

			if allFieldsExist {
				compositeKey := index.NewCompositeKey(values...)
				if err := idx.Insert(compositeKey, id); err != nil {
					c.mu.Unlock()
					return fmt.Errorf("failed to build compound index: %w", err)
				}
			}
		}
		c.mu.Unlock()
	}

	return nil
}

// CreatePartialIndex creates a partial index that only indexes documents matching a filter
func (c *Collection) CreatePartialIndex(fieldPath string, filter map[string]interface{}, unique bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(filter) == 0 {
		return fmt.Errorf("partial index must have a filter expression")
	}

	// Generate index name: fieldPath_partial
	indexName := fieldPath + "_partial"

	if _, exists := c.indexes[indexName]; exists {
		return fmt.Errorf("index %s already exists", indexName)
	}

	idx := index.NewIndex(&index.IndexConfig{
		Name:      indexName,
		FieldPath: fieldPath,
		Type:      index.IndexTypeBTree,
		Unique:    unique,
		Order:     32,
		Filter:    filter,
	})

	// Build index from existing documents that match the filter
	for id, doc := range c.documents {
		// Check if document matches the filter
		if c.matchesPartialIndexFilter(doc, idx) {
			// Get field value and insert into index
			if fieldValue, exists := doc.Get(fieldPath); exists {
				if err := idx.Insert(fieldValue, id); err != nil {
					return fmt.Errorf("failed to build partial index: %w", err)
				}
			}
		}
	}

	c.indexes[indexName] = idx
	return nil
}

// CreateTextIndex creates a text search index on one or more text fields
func (c *Collection) CreateTextIndex(fieldPaths []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(fieldPaths) == 0 {
		return fmt.Errorf("text index must have at least one field")
	}

	// Generate index name: field1_field2_..._text
	indexName := ""
	for i, field := range fieldPaths {
		if i > 0 {
			indexName += "_"
		}
		indexName += field
	}
	indexName += "_text"

	if _, exists := c.textIndexes[indexName]; exists {
		return fmt.Errorf("text index %s already exists", indexName)
	}

	// Create text index
	textIdx := index.NewTextIndex(indexName, fieldPaths)

	// Build index from existing documents
	for id, doc := range c.documents {
		// Extract all text field values
		texts := make([]string, 0, len(fieldPaths))

		for _, fieldPath := range fieldPaths {
			if fieldValue, exists := doc.Get(fieldPath); exists {
				// Convert value to string
				if str, ok := fieldValue.(string); ok {
					texts = append(texts, str)
				}
			}
		}

		// Only index if we found at least one text field
		if len(texts) > 0 {
			textIdx.Index(id, texts)
		}
	}

	c.textIndexes[indexName] = textIdx
	return nil
}

// Create2DIndex creates a 2d planar geospatial index on a field
func (c *Collection) Create2DIndex(fieldPath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	indexName := fieldPath + "_2d"

	if _, exists := c.geoIndexes[indexName]; exists {
		return fmt.Errorf("2d index %s already exists", indexName)
	}

	// Create 2d index
	geoIdx := index.NewGeoIndex(indexName, fieldPath, index.IndexType2D)

	// Build index from existing documents
	for id, doc := range c.documents {
		if fieldValue, exists := doc.Get(fieldPath); exists {
			// Try to parse as GeoJSON point
			if pointMap, ok := fieldValue.(map[string]interface{}); ok {
				if point, err := geo.ParseGeoJSONPoint(pointMap); err == nil {
					geoIdx.Index(id, point)
				}
			}
		}
	}

	c.geoIndexes[indexName] = geoIdx
	return nil
}

// Create2DSphereIndex creates a 2dsphere spherical geospatial index on a field
func (c *Collection) Create2DSphereIndex(fieldPath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	indexName := fieldPath + "_2dsphere"

	if _, exists := c.geoIndexes[indexName]; exists {
		return fmt.Errorf("2dsphere index %s already exists", indexName)
	}

	// Create 2dsphere index
	geoIdx := index.NewGeoIndex(indexName, fieldPath, index.IndexType2DSphere)

	// Build index from existing documents
	for id, doc := range c.documents {
		if fieldValue, exists := doc.Get(fieldPath); exists {
			// Try to parse as GeoJSON point
			if pointMap, ok := fieldValue.(map[string]interface{}); ok {
				if point, err := geo.ParseGeoJSONPoint(pointMap); err == nil {
					geoIdx.Index(id, point)
				}
			}
		}
	}

	c.geoIndexes[indexName] = geoIdx
	return nil
}

// CreateTTLIndex creates a TTL (time-to-live) index on a date field
// Documents will be automatically deleted ttlSeconds after the timestamp in the field
func (c *Collection) CreateTTLIndex(fieldPath string, ttlSeconds int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	indexName := fieldPath + "_ttl"

	if _, exists := c.ttlIndexes[indexName]; exists {
		return fmt.Errorf("ttl index %s already exists", indexName)
	}

	// Create TTL index
	ttlIdx := index.NewTTLIndex(indexName, fieldPath, ttlSeconds)

	// Build index from existing documents
	for id, doc := range c.documents {
		if fieldValue, exists := doc.Get(fieldPath); exists {
			// Try to convert to time.Time
			var timestamp time.Time
			switch v := fieldValue.(type) {
			case time.Time:
				timestamp = v
			case string:
				// Try to parse as RFC3339 string
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					timestamp = t
				}
			case int64:
				// Unix timestamp in seconds
				timestamp = time.Unix(v, 0)
			default:
				continue // Skip non-time values
			}

			ttlIdx.Index(id, timestamp)
		}
	}

	c.ttlIndexes[indexName] = ttlIdx
	return nil
}

// DropIndex drops an index (B+ tree, compound, text, geo, or ttl)
func (c *Collection) DropIndex(indexName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if indexName == "_id_" {
		return fmt.Errorf("cannot drop _id index")
	}

	// Check if it's a regular index
	if _, exists := c.indexes[indexName]; exists {
		delete(c.indexes, indexName)
		return nil
	}

	// Check if it's a text index
	if _, exists := c.textIndexes[indexName]; exists {
		delete(c.textIndexes, indexName)
		return nil
	}

	// Check if it's a geo index
	if _, exists := c.geoIndexes[indexName]; exists {
		delete(c.geoIndexes, indexName)
		return nil
	}

	// Check if it's a TTL index
	if _, exists := c.ttlIndexes[indexName]; exists {
		delete(c.ttlIndexes, indexName)
		return nil
	}

	return fmt.Errorf("index %s does not exist", indexName)
}

// ListIndexes returns all indexes (B+ tree, compound, text, geo, and ttl)
func (c *Collection) ListIndexes() []map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	indexes := make([]map[string]interface{}, 0, len(c.indexes)+len(c.textIndexes)+len(c.geoIndexes)+len(c.ttlIndexes))

	// Add regular indexes
	for _, idx := range c.indexes {
		indexes = append(indexes, idx.Stats())
	}

	// Add text indexes
	for _, textIdx := range c.textIndexes {
		indexes = append(indexes, textIdx.Stats())
	}

	// Add geo indexes
	for _, geoIdx := range c.geoIndexes {
		indexes = append(indexes, geoIdx.Stats())
	}

	// Add TTL indexes
	for _, ttlIdx := range c.ttlIndexes {
		indexes = append(indexes, map[string]interface{}{
			"name":       ttlIdx.Name(),
			"field":      ttlIdx.FieldPath(),
			"type":       "ttl",
			"ttlSeconds": ttlIdx.TTLSeconds(),
			"count":      ttlIdx.Count(),
		})
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
		"name":             c.name,
		"count":            len(c.documents),
		"indexes":          len(c.indexes),
		"index_count":      len(c.indexes),
		"text_index_count": len(c.textIndexes),
		"geo_index_count":  len(c.geoIndexes),
		"ttl_index_count":  len(c.ttlIndexes),
		"index_details":    c.ListIndexes(),
	}
}

// Analyze recalculates statistics for all indexes in the collection
func (c *Collection) Analyze() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, idx := range c.indexes {
		idx.Analyze()
	}

	for _, textIdx := range c.textIndexes {
		textIdx.Analyze()
	}
}

// TextSearch performs a text search using text indexes
// Returns documents sorted by relevance score (highest first)
func (c *Collection) TextSearch(searchText string, options *QueryOptions) ([]*document.Document, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.textIndexes) == 0 {
		return nil, fmt.Errorf("no text index available for text search")
	}

	// Use the first text index (in practice, we'd select based on fields)
	var textIdx *index.TextIndex
	for _, idx := range c.textIndexes {
		textIdx = idx
		break
	}

	// Search the text index
	results := textIdx.Search(searchText)

	// Convert results to documents
	docs := make([]*document.Document, 0, len(results))
	for _, result := range results {
		if doc, exists := c.documents[result.DocID]; exists {
			// Add relevance score as a metadata field
			docCopy := document.NewDocumentFromMap(doc.ToMap())
			docCopy.Set("_textScore", result.Score)
			docs = append(docs, docCopy)
		}
	}

	// Apply projection if specified
	if options != nil && options.Projection != nil {
		for i, doc := range docs {
			projected := document.NewDocument()

			// Always include _id unless explicitly excluded
			excludeId := false
			if val, exists := options.Projection["_id"]; exists && !val {
				excludeId = true
			}

			if !excludeId {
				if id, exists := doc.Get("_id"); exists {
					projected.Set("_id", id)
				}
			}

			// Include requested fields
			for field, include := range options.Projection {
				if field == "_id" {
					continue // Already handled above
				}
				if include {
					if val, exists := doc.Get(field); exists {
						projected.Set(field, val)
					}
				}
			}

			docs[i] = projected
		}
	}

	// Apply skip
	if options != nil && options.Skip > 0 {
		if options.Skip >= len(docs) {
			return []*document.Document{}, nil
		}
		docs = docs[options.Skip:]
	}

	// Apply limit
	if options != nil && options.Limit > 0 && options.Limit < len(docs) {
		docs = docs[:options.Limit]
	}

	return docs, nil
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

func toInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case int:
		return int64(val), true
	case float64:
		return int64(val), true
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

// Near finds documents near a geographic point
// For 2d indexes: maxDistance is in coordinate units
// For 2dsphere indexes: maxDistance is in meters
func (c *Collection) Near(fieldPath string, center *geo.Point, maxDistance float64, limit int, options *QueryOptions) ([]*document.Document, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Find the geospatial index for this field
	var geoIdx *index.GeoIndex
	for _, idx := range c.geoIndexes {
		if idx.FieldPath() == fieldPath {
			geoIdx = idx
			break
		}
	}

	if geoIdx == nil {
		return nil, fmt.Errorf("no geospatial index found for field %s", fieldPath)
	}

	// Perform near search
	results := geoIdx.Near(center, maxDistance, limit)

	// Convert results to documents
	docs := make([]*document.Document, 0, len(results))
	for _, result := range results {
		if doc, exists := c.documents[result.DocID]; exists {
			// Add distance as metadata field
			docCopy := document.NewDocumentFromMap(doc.ToMap())
			docCopy.Set("_distance", result.Distance)
			docs = append(docs, docCopy)
		}
	}

	// Apply projection if specified
	if options != nil && options.Projection != nil {
		for i, doc := range docs {
			projected := document.NewDocument()

			// Always include _id unless explicitly excluded
			excludeId := false
			if val, exists := options.Projection["_id"]; exists && !val {
				excludeId = true
			}

			if !excludeId {
				if id, exists := doc.Get("_id"); exists {
					projected.Set("_id", id)
				}
			}

			// Preserve _distance metadata
			if distance, exists := doc.Get("_distance"); exists {
				projected.Set("_distance", distance)
			}

			// Include requested fields
			for field, include := range options.Projection {
				if field == "_id" {
					continue // Already handled above
				}
				if include {
					if val, exists := doc.Get(field); exists {
						projected.Set(field, val)
					}
				}
			}

			docs[i] = projected
		}
	}

	return docs, nil
}

// GeoWithin finds documents within a polygon
func (c *Collection) GeoWithin(fieldPath string, polygon *geo.Polygon, options *QueryOptions) ([]*document.Document, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Find the geospatial index for this field
	var geoIdx *index.GeoIndex
	for _, idx := range c.geoIndexes {
		if idx.FieldPath() == fieldPath {
			geoIdx = idx
			break
		}
	}

	if geoIdx == nil {
		return nil, fmt.Errorf("no geospatial index found for field %s", fieldPath)
	}

	// Perform within search
	docIDs := geoIdx.Within(polygon)

	// Convert IDs to documents
	docs := make([]*document.Document, 0, len(docIDs))
	for _, id := range docIDs {
		if doc, exists := c.documents[id]; exists {
			docs = append(docs, doc)
		}
	}

	// Apply projection if specified
	if options != nil && options.Projection != nil {
		for i, doc := range docs {
			projected := document.NewDocument()

			// Always include _id unless explicitly excluded
			excludeId := false
			if val, exists := options.Projection["_id"]; exists && !val {
				excludeId = true
			}

			if !excludeId {
				if id, exists := doc.Get("_id"); exists {
					projected.Set("_id", id)
				}
			}

			// Include requested fields
			for field, include := range options.Projection {
				if field == "_id" {
					continue // Already handled above
				}
				if include {
					if val, exists := doc.Get(field); exists {
						projected.Set(field, val)
					}
				}
			}

			docs[i] = projected
		}
	}

	// Apply skip and limit
	if options != nil {
		if options.Skip > 0 && options.Skip < len(docs) {
			docs = docs[options.Skip:]
		} else if options.Skip >= len(docs) {
			docs = []*document.Document{}
		}

		if options.Limit > 0 && options.Limit < len(docs) {
			docs = docs[:options.Limit]
		}
	}

	return docs, nil
}

// GeoIntersects finds documents whose geometry intersects with a bounding box
func (c *Collection) GeoIntersects(fieldPath string, box *geo.BoundingBox, options *QueryOptions) ([]*document.Document, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Find the geospatial index for this field
	var geoIdx *index.GeoIndex
	for _, idx := range c.geoIndexes {
		if idx.FieldPath() == fieldPath {
			geoIdx = idx
			break
		}
	}

	if geoIdx == nil {
		return nil, fmt.Errorf("no geospatial index found for field %s", fieldPath)
	}

	// Perform intersects search (bounding box query)
	docIDs := geoIdx.InBox(box)

	// Convert IDs to documents
	docs := make([]*document.Document, 0, len(docIDs))
	for _, id := range docIDs {
		if doc, exists := c.documents[id]; exists {
			docs = append(docs, doc)
		}
	}

	// Apply projection if specified
	if options != nil && options.Projection != nil {
		for i, doc := range docs {
			projected := document.NewDocument()

			// Always include _id unless explicitly excluded
			excludeId := false
			if val, exists := options.Projection["_id"]; exists && !val {
				excludeId = true
			}

			if !excludeId {
				if id, exists := doc.Get("_id"); exists {
					projected.Set("_id", id)
				}
			}

			// Include requested fields
			for field, include := range options.Projection {
				if field == "_id" {
					continue // Already handled above
				}
				if include {
					if val, exists := doc.Get(field); exists {
						projected.Set(field, val)
					}
				}
			}

			docs[i] = projected
		}
	}

	// Apply skip and limit
	if options != nil {
		if options.Skip > 0 && options.Skip < len(docs) {
			docs = docs[options.Skip:]
		} else if options.Skip >= len(docs) {
			docs = []*document.Document{}
		}

		if options.Limit > 0 && options.Limit < len(docs) {
			docs = docs[:options.Limit]
		}
	}

	return docs, nil
}

// CleanupExpiredDocuments removes documents that have expired according to TTL indexes
// Returns the number of documents deleted
func (c *Collection) CleanupExpiredDocuments() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.ttlIndexes) == 0 {
		return 0
	}

	currentTime := time.Now()
	deletedCount := 0
	toDelete := make(map[string]bool) // Use map to avoid duplicates

	// Collect all expired document IDs from all TTL indexes
	for _, ttlIdx := range c.ttlIndexes {
		expiredDocs := ttlIdx.GetExpiredDocuments(currentTime)
		for _, docID := range expiredDocs {
			toDelete[docID] = true
		}
	}

	// Delete each expired document
	for docID := range toDelete {
		doc, exists := c.documents[docID]
		if !exists {
			continue
		}

		// Remove from regular indexes
		for _, idx := range c.indexes {
			if idx.IsCompound() {
				if compositeKey, allFieldsExist := c.extractCompositeKey(doc, idx.FieldPaths()); allFieldsExist {
					idx.Delete(compositeKey)
				}
			} else {
				if fieldValue, exists := doc.Get(idx.FieldPath()); exists {
					idx.Delete(fieldValue)
				}
			}
		}

		// Remove from text indexes
		for _, textIdx := range c.textIndexes {
			textIdx.Remove(docID)
		}

		// Remove from geo indexes
		for _, geoIdx := range c.geoIndexes {
			geoIdx.Remove(docID)
		}

		// Remove from TTL indexes
		for _, ttlIdx := range c.ttlIndexes {
			ttlIdx.Remove(docID)
		}

		// Delete the document
		delete(c.documents, docID)
		deletedCount++
	}

	// Invalidate query cache if documents were deleted
	if deletedCount > 0 {
		c.queryCache.Clear()
	}

	return deletedCount
}

// docSnapshot represents a snapshot of a document for background index building
type docSnapshot struct {
	id            string
	fieldValue    interface{}
	compositeKey  *index.CompositeKey
	allFieldsExist bool
	matchesFilter bool
}

// captureSingleFieldSnapshot captures a snapshot of documents for single-field index building
// Must be called while holding c.mu lock
func (c *Collection) captureSingleFieldSnapshot(idx *index.Index, fieldPath string) []docSnapshot {
	snapshots := make([]docSnapshot, 0, len(c.documents))
	for id, doc := range c.documents {
		snapshot := docSnapshot{id: id}

		// Check if document matches partial index filter (if applicable)
		if idx.IsPartial() {
			snapshot.matchesFilter = c.matchesPartialIndexFilter(doc, idx)
			if !snapshot.matchesFilter {
				snapshots = append(snapshots, snapshot)
				continue
			}
		} else {
			snapshot.matchesFilter = true
		}

		// Extract field value at snapshot time
		if fieldValue, exists := doc.Get(fieldPath); exists {
			snapshot.fieldValue = fieldValue
		}

		snapshots = append(snapshots, snapshot)
	}
	return snapshots
}

// buildSingleFieldIndexInBackgroundWithSnapshot builds index from a snapshot in the background
func (c *Collection) buildSingleFieldIndexInBackgroundWithSnapshot(idx *index.Index, snapshots []docSnapshot) {
	// Start build in goroutine
	go func() {
		// Mark index as building
		idx.StartBuild(len(snapshots))

		// Process documents
		for _, snapshot := range snapshots {
			// Skip documents that don't match filter
			if !snapshot.matchesFilter {
				idx.IncrementBuildProgress()
				continue
			}

			// Skip documents without the indexed field
			if snapshot.fieldValue == nil {
				idx.IncrementBuildProgress()
				continue
			}

			// Insert into index (may fail if already exists due to concurrent write)
			if err := idx.Insert(snapshot.fieldValue, snapshot.id); err != nil {
				// Skip duplicate key errors - document was likely added by concurrent write
				// This can happen when a document from the snapshot gets updated/inserted
				// after the snapshot but before the background builder processes it
				errMsg := err.Error()
				if strings.Contains(errMsg, "duplicate") {
					// Skip duplicate (concurrent insert already added it)
					idx.IncrementBuildProgress()
					continue
				}
				// Real error - mark build as failed
				idx.FailBuild(fmt.Sprintf("failed to build index: %v", err))
				return
			}

			idx.IncrementBuildProgress()
		}

		// Mark index as ready
		idx.CompleteBuild()
	}()
}

// captureCompoundIndexSnapshot captures a snapshot of documents for compound index building
// Must be called while holding c.mu lock
func (c *Collection) captureCompoundIndexSnapshot(idx *index.Index, fieldPaths []string) []docSnapshot {
	snapshots := make([]docSnapshot, 0, len(c.documents))
	for id, doc := range c.documents {
		snapshot := docSnapshot{id: id}

		// Check if document matches partial index filter (if applicable)
		if idx.IsPartial() {
			snapshot.matchesFilter = c.matchesPartialIndexFilter(doc, idx)
			if !snapshot.matchesFilter {
				snapshots = append(snapshots, snapshot)
				continue
			}
		} else {
			snapshot.matchesFilter = true
		}

		// Extract all field values for the composite key at snapshot time
		values := make([]interface{}, 0, len(fieldPaths))
		snapshot.allFieldsExist = true

		for _, fieldPath := range fieldPaths {
			if fieldValue, exists := doc.Get(fieldPath); exists {
				values = append(values, fieldValue)
			} else {
				// If any field is missing, skip this document
				snapshot.allFieldsExist = false
				break
			}
		}

		if snapshot.allFieldsExist {
			snapshot.compositeKey = index.NewCompositeKey(values...)
		}

		snapshots = append(snapshots, snapshot)
	}
	return snapshots
}

// buildCompoundIndexInBackgroundWithSnapshot builds compound index from a snapshot in the background
func (c *Collection) buildCompoundIndexInBackgroundWithSnapshot(idx *index.Index, snapshots []docSnapshot) {
	// Start build in goroutine
	go func() {
		// Mark index as building
		idx.StartBuild(len(snapshots))

		// Process documents
		for _, snapshot := range snapshots {
			// Skip documents that don't match filter
			if !snapshot.matchesFilter {
				idx.IncrementBuildProgress()
				continue
			}

			// Skip documents without all required fields
			if !snapshot.allFieldsExist || snapshot.compositeKey == nil {
				idx.IncrementBuildProgress()
				continue
			}

			// Insert into index (may fail if already exists due to concurrent write)
			if err := idx.Insert(snapshot.compositeKey, snapshot.id); err != nil {
				// Skip duplicate key errors - document was likely added by concurrent write
				errMsg := err.Error()
				if strings.Contains(errMsg, "duplicate") {
					// Skip duplicate (concurrent insert already added it)
					idx.IncrementBuildProgress()
					continue
				}
				// Real error - mark build as failed
				idx.FailBuild(fmt.Sprintf("failed to build compound index: %v", err))
				return
			}

			idx.IncrementBuildProgress()
		}

		// Mark index as ready
		idx.CompleteBuild()
	}()
}

// GetIndexBuildProgress returns the build progress for a named index
func (c *Collection) GetIndexBuildProgress(indexName string) (map[string]interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	idx, exists := c.indexes[indexName]
	if !exists {
		return nil, fmt.Errorf("index %s not found", indexName)
	}

	return idx.GetBuildProgress(), nil
}
