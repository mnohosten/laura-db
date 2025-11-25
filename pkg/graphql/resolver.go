package graphql

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/document"
)

// Resolver handles GraphQL query and mutation resolution
type Resolver struct {
	db *database.Database
}

// NewResolver creates a new Resolver instance
func NewResolver(db *database.Database) *Resolver {
	return &Resolver{db: db}
}

// FindOne resolves the findOne query
func (r *Resolver) FindOne(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return nil, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Parse filter
	var filter map[string]interface{}
	if filterArg, ok := p.Args["filter"]; ok && filterArg != nil {
		filter, ok = filterArg.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid filter format")
		}
	} else {
		filter = map[string]interface{}{}
	}

	// Execute find
	docs, err := coll.Find(filter)
	if err != nil {
		return nil, fmt.Errorf("find failed: %w", err)
	}

	if len(docs) == 0 {
		return nil, nil
	}

	// Return first document
	doc := docs[0]
	idVal, _ := doc.Get("_id")
	return map[string]interface{}{
		"_id":  fmt.Sprintf("%v", idVal),
		"data": doc.ToMap(),
	}, nil
}

// Find resolves the find query
func (r *Resolver) Find(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return nil, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Parse filter
	var filter map[string]interface{}
	if filterArg, ok := p.Args["filter"]; ok && filterArg != nil {
		filter, ok = filterArg.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid filter format")
		}
	} else {
		filter = map[string]interface{}{}
	}

	// Create query options
	opts := &database.QueryOptions{}

	// Parse limit
	if limitArg, ok := p.Args["limit"]; ok {
		if limit, ok := limitArg.(int); ok {
			opts.Limit = limit
		}
	}

	// Parse skip
	if skipArg, ok := p.Args["skip"]; ok {
		if skip, ok := skipArg.(int); ok {
			opts.Skip = skip
		}
	}

	// Execute find with options
	docs, err := coll.FindWithOptions(filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find failed: %w", err)
	}

	// Convert documents to GraphQL format
	results := make([]map[string]interface{}, len(docs))
	for i, doc := range docs {
		idVal, _ := doc.Get("_id")
		results[i] = map[string]interface{}{
			"_id":  fmt.Sprintf("%v", idVal),
			"data": doc.ToMap(),
		}
	}

	return results, nil
}

// Count resolves the count query
func (r *Resolver) Count(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return nil, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Parse filter
	var filter map[string]interface{}
	if filterArg, ok := p.Args["filter"]; ok && filterArg != nil {
		filter, ok = filterArg.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid filter format")
		}
	} else {
		filter = map[string]interface{}{}
	}

	// Execute count
	count, err := coll.Count(filter)
	if err != nil {
		return nil, fmt.Errorf("count failed: %w", err)
	}

	return count, nil
}

// ListCollections resolves the listCollections query
func (r *Resolver) ListCollections(p graphql.ResolveParams) (interface{}, error) {
	collections := r.db.ListCollections()
	return collections, nil
}

// CollectionStats resolves the collectionStats query
func (r *Resolver) CollectionStats(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return nil, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Get stats
	count, err := coll.Count(map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to get count: %w", err)
	}

	indexes := coll.ListIndexes()

	return map[string]interface{}{
		"name":          collectionName,
		"documentCount": count,
		"indexCount":    len(indexes),
	}, nil
}

// ListIndexes resolves the listIndexes query
func (r *Resolver) ListIndexes(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return nil, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Get indexes
	indexes := coll.ListIndexes()

	// Convert to GraphQL format (indexes are already []map[string]interface{})
	results := make([]map[string]interface{}, len(indexes))
	for i, idx := range indexes {
		// Extract fields with default values if missing
		name, _ := idx["name"].(string)
		field, _ := idx["field"].(string)
		unique, _ := idx["unique"].(bool)
		idxType, _ := idx["type"].(string)

		results[i] = map[string]interface{}{
			"name":   name,
			"field":  field,
			"unique": unique,
			"type":   idxType,
		}
	}

	return results, nil
}

// Aggregate resolves the aggregate query
func (r *Resolver) Aggregate(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return nil, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Parse pipeline
	var pipeline []map[string]interface{}
	if pipelineArg, ok := p.Args["pipeline"]; ok && pipelineArg != nil {
		pipelineSlice, ok := pipelineArg.([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid pipeline format")
		}

		pipeline = make([]map[string]interface{}, len(pipelineSlice))
		for i, stage := range pipelineSlice {
			stageMap, ok := stage.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid pipeline stage format")
			}
			pipeline[i] = stageMap
		}
	} else {
		pipeline = []map[string]interface{}{}
	}

	// Execute aggregation
	results, err := coll.Aggregate(pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation failed: %w", err)
	}

	return map[string]interface{}{
		"results": results,
	}, nil
}

// CreateCollection resolves the createCollection mutation
func (r *Resolver) CreateCollection(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	name, ok := p.Args["name"].(string)
	if !ok {
		return false, fmt.Errorf("collection name is required")
	}

	// Create collection
	_, err := r.db.CreateCollection(name)
	if err != nil {
		return false, fmt.Errorf("failed to create collection: %w", err)
	}

	return true, nil
}

// DropCollection resolves the dropCollection mutation
func (r *Resolver) DropCollection(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	name, ok := p.Args["name"].(string)
	if !ok {
		return false, fmt.Errorf("collection name is required")
	}

	// Drop collection
	err := r.db.DropCollection(name)
	if err != nil {
		return false, fmt.Errorf("failed to drop collection: %w", err)
	}

	return true, nil
}

// InsertOne resolves the insertOne mutation
func (r *Resolver) InsertOne(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return nil, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Parse document
	docData, ok := p.Args["document"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("document is required")
	}

	// Insert document
	id, err := coll.InsertOne(docData)
	if err != nil {
		return nil, fmt.Errorf("insert failed: %w", err)
	}

	return map[string]interface{}{
		"insertedId": id, // InsertOne returns a string
	}, nil
}

// InsertMany resolves the insertMany mutation
func (r *Resolver) InsertMany(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return nil, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Parse documents
	docsArg, ok := p.Args["documents"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("documents array is required")
	}

	docs := make([]map[string]interface{}, len(docsArg))
	for i, docArg := range docsArg {
		doc, ok := docArg.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid document format at index %d", i)
		}
		docs[i] = doc
	}

	// Insert documents
	ids, err := coll.InsertMany(docs)
	if err != nil {
		return nil, fmt.Errorf("insert failed: %w", err)
	}

	// InsertMany already returns []string
	return map[string]interface{}{
		"insertedIds":   ids,
		"insertedCount": len(ids),
	}, nil
}

// UpdateOne resolves the updateOne mutation
func (r *Resolver) UpdateOne(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return nil, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Parse filter
	filter, ok := p.Args["filter"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("filter is required")
	}

	// Parse update
	update, ok := p.Args["update"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("update is required")
	}

	// Execute update
	err := coll.UpdateOne(filter, update)
	if err != nil {
		return nil, fmt.Errorf("update failed: %w", err)
	}

	// UpdateOne returns error if no match, nil otherwise
	return map[string]interface{}{
		"matchedCount":  1,
		"modifiedCount": 1,
	}, nil
}

// UpdateMany resolves the updateMany mutation
func (r *Resolver) UpdateMany(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return nil, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Parse filter
	filter, ok := p.Args["filter"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("filter is required")
	}

	// Parse update
	update, ok := p.Args["update"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("update is required")
	}

	// Execute update
	count, err := coll.UpdateMany(filter, update)
	if err != nil {
		return nil, fmt.Errorf("update failed: %w", err)
	}

	return map[string]interface{}{
		"matchedCount":  count,
		"modifiedCount": count,
	}, nil
}

// DeleteOne resolves the deleteOne mutation
func (r *Resolver) DeleteOne(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return nil, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Parse filter
	filter, ok := p.Args["filter"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("filter is required")
	}

	// Execute delete
	err := coll.DeleteOne(filter)
	if err != nil {
		return nil, fmt.Errorf("delete failed: %w", err)
	}

	// DeleteOne returns error if no match, nil otherwise
	return map[string]interface{}{
		"deletedCount": 1,
	}, nil
}

// DeleteMany resolves the deleteMany mutation
func (r *Resolver) DeleteMany(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return nil, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Parse filter
	filter, ok := p.Args["filter"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("filter is required")
	}

	// Execute delete
	count, err := coll.DeleteMany(filter)
	if err != nil {
		return nil, fmt.Errorf("delete failed: %w", err)
	}

	return map[string]interface{}{
		"deletedCount": count,
	}, nil
}

// CreateIndex resolves the createIndex mutation
func (r *Resolver) CreateIndex(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return false, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return false, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Parse field
	field, ok := p.Args["field"].(string)
	if !ok {
		return false, fmt.Errorf("field is required")
	}

	// Parse unique flag
	unique := false
	if uniqueArg, ok := p.Args["unique"]; ok {
		unique, _ = uniqueArg.(bool)
	}

	// Parse optional name
	var name string
	if nameArg, ok := p.Args["name"]; ok && nameArg != nil {
		name, _ = nameArg.(string)
	}
	if name == "" {
		name = field + "_1"
	}

	// Create index
	err := coll.CreateIndex(field, unique)
	if err != nil {
		return false, fmt.Errorf("failed to create index: %w", err)
	}

	return true, nil
}

// DropIndex resolves the dropIndex mutation
func (r *Resolver) DropIndex(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return false, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return false, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Parse name
	name, ok := p.Args["name"].(string)
	if !ok {
		return false, fmt.Errorf("index name is required")
	}

	// Drop index
	err := coll.DropIndex(name)
	if err != nil {
		return false, fmt.Errorf("failed to drop index: %w", err)
	}

	return true, nil
}

// WatchCollection resolves the watchCollection subscription
func (r *Resolver) WatchCollection(p graphql.ResolveParams) (interface{}, error) {
	// Get collection name
	collectionName, ok := p.Args["collection"].(string)
	if !ok {
		return nil, fmt.Errorf("collection name is required")
	}

	// Get collection
	coll := r.db.Collection(collectionName)
	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Parse optional filter
	var _ map[string]interface{}
	if filterArg, ok := p.Args["filter"]; ok && filterArg != nil {
		_, ok = filterArg.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid filter format")
		}
	}

	// Create a channel for subscription
	changes := make(chan *document.Document)

	// Start watching collection
	// Note: This is a simplified implementation
	// In a production system, you would use the change stream functionality
	go func() {
		// This would integrate with LauraDB's change stream functionality
		// For now, this is a placeholder
		defer close(changes)
	}()

	return changes, nil
}
