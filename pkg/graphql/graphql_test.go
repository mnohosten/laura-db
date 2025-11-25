package graphql

import (
	"encoding/json"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/mnohosten/laura-db/pkg/database"
)

// TestGraphQLSchema tests the schema creation
func TestGraphQLSchema(t *testing.T) {
	// Create test database
	config := &database.Config{
		DataDir:        t.TempDir(),
		BufferPoolSize: 100,
	}
	db, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create schema
	schema, err := Schema(db)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Verify query type exists
	if schema.QueryType() == nil {
		t.Fatal("Query type is nil")
	}

	// Verify mutation type exists
	if schema.MutationType() == nil {
		t.Fatal("Mutation type is nil")
	}

	// Verify subscription type exists
	if schema.SubscriptionType() == nil {
		t.Fatal("Subscription type is nil")
	}
}

// TestGraphQLCreateCollection tests collection creation mutation
func TestGraphQLCreateCollection(t *testing.T) {
	config := &database.Config{
		DataDir:        t.TempDir(),
		BufferPoolSize: 100,
	}
	db, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	schema, err := Schema(db)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Execute mutation
	query := `
		mutation {
			createCollection(name: "users")
		}
	`

	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})

	if len(result.Errors) > 0 {
		t.Fatalf("GraphQL errors: %v", result.Errors)
	}

	// Verify result
	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Invalid result data type")
	}

	created, ok := data["createCollection"].(bool)
	if !ok || !created {
		t.Fatal("Collection not created")
	}

	// Verify collection exists
	if db.Collection("users") == nil {
		t.Fatal("Collection does not exist in database")
	}
}

// TestGraphQLInsertOne tests document insertion
func TestGraphQLInsertOne(t *testing.T) {
	config := &database.Config{
		DataDir:        t.TempDir(),
		BufferPoolSize: 100,
	}
	db, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection
	db.CreateCollection("users")

	schema, err := Schema(db)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Execute mutation
	query := `
		mutation {
			insertOne(
				collection: "users"
				document: {name: "Alice", age: 30}
			) {
				insertedId
			}
		}
	`

	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})

	if len(result.Errors) > 0 {
		t.Fatalf("GraphQL errors: %v", result.Errors)
	}

	// Verify result
	data := result.Data.(map[string]interface{})
	insertResult := data["insertOne"].(map[string]interface{})
	insertedId := insertResult["insertedId"].(string)

	if insertedId == "" {
		t.Fatal("No insertedId returned")
	}

	// Verify document exists in database
	coll := db.Collection("users")
	docs, err := coll.Find(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to find documents: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(docs))
	}
}

// TestGraphQLFind tests document queries
func TestGraphQLFind(t *testing.T) {
	config := &database.Config{
		DataDir:        t.TempDir(),
		BufferPoolSize: 100,
	}
	db, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection and insert documents
	coll, _ := db.CreateCollection("users")
	coll.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30)})
	coll.InsertOne(map[string]interface{}{"name": "Bob", "age": int64(25)})
	coll.InsertOne(map[string]interface{}{"name": "Charlie", "age": int64(35)})

	schema, err := Schema(db)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Execute query
	query := `
		{
			find(collection: "users", filter: {age: {$gte: 30}}) {
				_id
				data
			}
		}
	`

	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})

	if len(result.Errors) > 0 {
		t.Fatalf("GraphQL errors: %v", result.Errors)
	}

	// Verify result
	data := result.Data.(map[string]interface{})
	docs := data["find"].([]interface{})

	if len(docs) != 2 {
		t.Fatalf("Expected 2 documents, got %d", len(docs))
	}
}

// TestGraphQLCount tests document counting
func TestGraphQLCount(t *testing.T) {
	config := &database.Config{
		DataDir:        t.TempDir(),
		BufferPoolSize: 100,
	}
	db, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection and insert documents
	coll, _ := db.CreateCollection("users")
	coll.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30)})
	coll.InsertOne(map[string]interface{}{"name": "Bob", "age": int64(25)})
	coll.InsertOne(map[string]interface{}{"name": "Charlie", "age": int64(35)})

	schema, err := Schema(db)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Execute query
	query := `
		{
			count(collection: "users", filter: {age: {$gt: 26}})
		}
	`

	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})

	if len(result.Errors) > 0 {
		t.Fatalf("GraphQL errors: %v", result.Errors)
	}

	// Verify result
	data := result.Data.(map[string]interface{})
	count := data["count"].(int)

	if count != 2 {
		t.Fatalf("Expected count 2, got %d", count)
	}
}

// TestGraphQLUpdateOne tests document updates
func TestGraphQLUpdateOne(t *testing.T) {
	config := &database.Config{
		DataDir:        t.TempDir(),
		BufferPoolSize: 100,
	}
	db, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection and insert document
	coll, _ := db.CreateCollection("users")
	coll.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30)})

	schema, err := Schema(db)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Execute mutation
	query := `
		mutation {
			updateOne(
				collection: "users"
				filter: {name: "Alice"}
				update: {$set: {age: 31}}
			) {
				modifiedCount
			}
		}
	`

	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})

	if len(result.Errors) > 0 {
		t.Fatalf("GraphQL errors: %v", result.Errors)
	}

	// Verify result
	data := result.Data.(map[string]interface{})
	updateResult := data["updateOne"].(map[string]interface{})
	modifiedCount := updateResult["modifiedCount"].(int)

	if modifiedCount != 1 {
		t.Fatalf("Expected modifiedCount 1, got %d", modifiedCount)
	}

	// Verify update in database
	docs, _ := coll.Find(map[string]interface{}{"name": "Alice"})
	if len(docs) != 1 {
		t.Fatal("Document not found after update")
	}

	age, _ := docs[0].Get("age")
	if age != int64(31) {
		t.Fatalf("Expected age 31, got %v", age)
	}
}

// TestGraphQLDeleteOne tests document deletion
func TestGraphQLDeleteOne(t *testing.T) {
	config := &database.Config{
		DataDir:        t.TempDir(),
		BufferPoolSize: 100,
	}
	db, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection and insert documents
	coll, _ := db.CreateCollection("users")
	coll.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30)})
	coll.InsertOne(map[string]interface{}{"name": "Bob", "age": int64(25)})

	schema, err := Schema(db)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Execute mutation
	query := `
		mutation {
			deleteOne(
				collection: "users"
				filter: {name: "Alice"}
			) {
				deletedCount
			}
		}
	`

	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})

	if len(result.Errors) > 0 {
		t.Fatalf("GraphQL errors: %v", result.Errors)
	}

	// Verify result
	data := result.Data.(map[string]interface{})
	deleteResult := data["deleteOne"].(map[string]interface{})
	deletedCount := deleteResult["deletedCount"].(int)

	if deletedCount != 1 {
		t.Fatalf("Expected deletedCount 1, got %d", deletedCount)
	}

	// Verify deletion in database
	count, _ := coll.Count(map[string]interface{}{})
	if count != 1 {
		t.Fatalf("Expected 1 document remaining, got %d", count)
	}
}

// TestGraphQLListCollections tests listing collections
func TestGraphQLListCollections(t *testing.T) {
	config := &database.Config{
		DataDir:        t.TempDir(),
		BufferPoolSize: 100,
	}
	db, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collections
	db.CreateCollection("users")
	db.CreateCollection("products")
	db.CreateCollection("orders")

	schema, err := Schema(db)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Execute query
	query := `
		{
			listCollections
		}
	`

	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})

	if len(result.Errors) > 0 {
		t.Fatalf("GraphQL errors: %v", result.Errors)
	}

	// Verify result
	data := result.Data.(map[string]interface{})
	collections := data["listCollections"].([]interface{})

	if len(collections) != 3 {
		t.Fatalf("Expected 3 collections, got %d", len(collections))
	}
}

// TestGraphQLCreateIndex tests index creation
func TestGraphQLCreateIndex(t *testing.T) {
	config := &database.Config{
		DataDir:        t.TempDir(),
		BufferPoolSize: 100,
	}
	db, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection
	db.CreateCollection("users")

	schema, err := Schema(db)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Execute mutation
	query := `
		mutation {
			createIndex(
				collection: "users"
				field: "email"
				unique: true
			)
		}
	`

	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})

	if len(result.Errors) > 0 {
		t.Fatalf("GraphQL errors: %v", result.Errors)
	}

	// Verify result
	data := result.Data.(map[string]interface{})
	created := data["createIndex"].(bool)

	if !created {
		t.Fatal("Index not created")
	}

	// Verify index exists in database
	coll := db.Collection("users")
	indexes := coll.ListIndexes()

	// Should have _id index + email index
	if len(indexes) < 2 {
		t.Fatalf("Expected at least 2 indexes, got %d", len(indexes))
	}
}

// TestGraphQLAggregate tests aggregation pipeline
func TestGraphQLAggregate(t *testing.T) {
	config := &database.Config{
		DataDir:        t.TempDir(),
		BufferPoolSize: 100,
	}
	db, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection and insert documents
	coll, _ := db.CreateCollection("orders")
	coll.InsertOne(map[string]interface{}{"product": "A", "quantity": int64(10), "price": int64(100)})
	coll.InsertOne(map[string]interface{}{"product": "B", "quantity": int64(5), "price": int64(200)})
	coll.InsertOne(map[string]interface{}{"product": "A", "quantity": int64(15), "price": int64(100)})

	schema, err := Schema(db)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Execute query with aggregation
	query := `
		{
			aggregate(
				collection: "orders"
				pipeline: [
					{$group: {_id: "$product", totalQuantity: {$sum: "$quantity"}}}
				]
			) {
				results
			}
		}
	`

	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})

	if len(result.Errors) > 0 {
		t.Fatalf("GraphQL errors: %v", result.Errors)
	}

	// Verify result
	data := result.Data.(map[string]interface{})
	aggResult := data["aggregate"].(map[string]interface{})
	results := aggResult["results"].([]interface{})

	if len(results) != 2 {
		t.Fatalf("Expected 2 aggregation results, got %d", len(results))
	}
}

// TestGraphQLVariables tests query with variables
func TestGraphQLVariables(t *testing.T) {
	config := &database.Config{
		DataDir:        t.TempDir(),
		BufferPoolSize: 100,
	}
	db, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection and insert documents
	coll, _ := db.CreateCollection("users")
	coll.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30)})
	coll.InsertOne(map[string]interface{}{"name": "Bob", "age": int64(25)})

	schema, err := Schema(db)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Execute query with variables
	query := `
		query FindUsers($collection: String!, $filter: JSON) {
			find(collection: $collection, filter: $filter) {
				_id
				data
			}
		}
	`

	variables := map[string]interface{}{
		"collection": "users",
		"filter": map[string]interface{}{
			"name": "Alice",
		},
	}

	result := graphql.Do(graphql.Params{
		Schema:         schema,
		RequestString:  query,
		VariableValues: variables,
	})

	if len(result.Errors) > 0 {
		t.Fatalf("GraphQL errors: %v", result.Errors)
	}

	// Verify result
	data := result.Data.(map[string]interface{})
	docs := data["find"].([]interface{})

	if len(docs) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(docs))
	}
}

// TestJSONScalar tests the JSON scalar type
func TestJSONScalar(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "map",
			input:    map[string]interface{}{"key": "value"},
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "slice",
			input:    []interface{}{1, 2, 3},
			expected: []interface{}{1, 2, 3},
		},
		{
			name:     "string",
			input:    `{"key": "value"}`,
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "number",
			input:    42,
			expected: 42,
		},
		{
			name:     "nil",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JSONScalar.ParseValue(tt.input)

			if tt.input == nil {
				if result != nil {
					t.Fatalf("Expected nil, got %v", result)
				}
				return
			}

			// For complex types, compare JSON representations
			expectedJSON, _ := json.Marshal(tt.expected)
			resultJSON, _ := json.Marshal(result)

			if string(expectedJSON) != string(resultJSON) {
				t.Fatalf("Expected %s, got %s", expectedJSON, resultJSON)
			}
		})
	}
}
