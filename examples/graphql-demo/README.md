# LauraDB GraphQL API Demo

Comprehensive demonstration of LauraDB's GraphQL API features.

## Overview

This example demonstrates all major GraphQL operations:

1. **List collections** - Query available collections
2. **Create collection** - Create new collection
3. **Insert documents** - Insert multiple documents
4. **Query documents** - Find documents with filters
5. **Update document** - Update specific document
6. **Count documents** - Count documents matching filter
7. **Create index** - Create index for better performance
8. **List indexes** - View all indexes
9. **Aggregate data** - Run aggregation pipeline
10. **Delete document** - Remove document
11. **Collection stats** - Get collection statistics

## Prerequisites

### 1. Build the Server

```bash
cd /path/to/laura-db
make server
```

### 2. Start Server with GraphQL Enabled

```bash
./bin/laura-server -graphql
```

You should see:

```
✅ GraphQL API enabled
   GraphQL endpoint: /graphql
   GraphiQL playground: /graphiql
🚀 LauraDB server starting on http://localhost:8080
```

## Running the Demo

```bash
cd examples/graphql-demo
go run main.go
```

## Expected Output

```
LauraDB GraphQL API Demo
=========================

⚠️  Prerequisites:
   Start LauraDB server with GraphQL enabled:
   ./bin/laura-server -graphql

Waiting 2 seconds for server to be ready...

1. List all collections
=======================
✅ Collections: []

2. Create a new collection
===========================
✅ Collection 'products' created: true

3. Insert documents
====================
✅ Inserted 3 documents
   IDs: [507f1f77bcf86cd799439011 507f1f77bcf86cd799439012 507f1f77bcf86cd799439013]

4. Query documents
===================
✅ Found 2 electronics products:
   - Laptop: $999
   - Mouse: $25

5. Update document
===================
✅ Updated 1 documents

6. Count documents
===================
✅ Electronics count: 2

7. Create index
================
✅ Index created: true

8. List indexes
================
✅ Indexes (2):
   - _id_ on field '_id' (unique: true)
   - category_1 on field 'category' (unique: false)

9. Aggregate data
==================
✅ Aggregation results:
   - Category: electronics, Total Stock: 250, Avg Price: $512.00
   - Category: furniture, Total Stock: 30, Avg Price: $299.00

10. Delete document
====================
✅ Deleted 1 documents

11. Get collection stats
=========================
✅ Collection Stats:
   - Name: products
   - Documents: 2
   - Indexes: 2

✅ GraphQL demo completed!

Next steps:
   - Open GraphiQL playground: http://localhost:8080/graphiql
   - Explore the schema using the Docs panel
   - Try your own queries and mutations
```

## GraphiQL Playground

After running the demo, explore the GraphQL API interactively:

1. Open http://localhost:8080/graphiql in your browser
2. Try these queries:

### List Collections

```graphql
query {
  listCollections
}
```

### Find Products

```graphql
query {
  find(collection: "products", limit: 10) {
    _id
    data
  }
}
```

### Insert Product (with Variables)

```graphql
mutation InsertProduct($collection: String!, $document: JSON!) {
  insertOne(collection: $collection, document: $document) {
    insertedId
  }
}
```

Variables:

```json
{
  "collection": "products",
  "document": {
    "name": "Keyboard",
    "price": 79,
    "category": "electronics"
  }
}
```

### Aggregation

```graphql
query {
  aggregate(
    collection: "products"
    pipeline: [
      {$group: {_id: "$category", count: {$sum: 1}}}
    ]
  ) {
    results
  }
}
```

## GraphQL Features Demonstrated

### Queries

- `listCollections` - List all collections
- `find` - Query documents with filters
- `findOne` - Find single document
- `count` - Count documents
- `collectionStats` - Get collection statistics
- `listIndexes` - List indexes
- `aggregate` - Run aggregation pipeline

### Mutations

- `createCollection` - Create new collection
- `insertOne` - Insert single document
- `insertMany` - Insert multiple documents
- `updateOne` - Update single document
- `updateMany` - Update multiple documents
- `deleteOne` - Delete single document
- `deleteMany` - Delete multiple documents
- `createIndex` - Create index
- `dropIndex` - Drop index
- `dropCollection` - Drop collection

### Variables

All operations use GraphQL variables for dynamic values:

```graphql
query FindProducts($collection: String!, $filter: JSON) {
  find(collection: $collection, filter: $filter) {
    _id
    data
  }
}
```

## Code Structure

```go
// Execute GraphQL query
func executeGraphQL(query string, variables map[string]interface{}) (*GraphQLResponse, error) {
    req := GraphQLRequest{
        Query:     query,
        Variables: variables,
    }

    jsonData, _ := json.Marshal(req)
    resp, _ := http.Post(graphqlEndpoint, "application/json", bytes.NewBuffer(jsonData))
    // Parse response...
}
```

## Common GraphQL Patterns

### Using Variables

```go
query := `
    query FindUsers($filter: JSON!) {
        find(collection: "users", filter: $filter) {
            _id
            data
        }
    }
`

variables := map[string]interface{}{
    "filter": map[string]interface{}{
        "age": map[string]interface{}{"$gte": 18},
    },
}
```

### Handling Errors

```go
resp, err := executeGraphQL(query, variables)
if err != nil {
    fmt.Printf("Error: %v\n", err)
    return
}

if len(resp.Errors) > 0 {
    fmt.Printf("GraphQL errors: %v\n", resp.Errors)
}
```

### Type Assertions

```go
// Access nested data
result := resp.Data["insertMany"].(map[string]interface{})
count := result["insertedCount"].(float64)
ids := result["insertedIds"].([]interface{})
```

## Comparison: GraphQL vs REST

| Operation | GraphQL | REST |
|-----------|---------|------|
| **List collections** | Single query | `GET /_collections` |
| **Insert + Query** | Single request with alias | Two requests |
| **Partial data** | Request specific fields | Get full response |
| **Real-time updates** | Subscriptions | WebSocket polling |

## Next Steps

1. **Explore the Schema**: Use GraphiQL's "Docs" panel
2. **Try Subscriptions**: Use WebSocket client for real-time updates
3. **Client Libraries**: Use Apollo Client or graphql-request
4. **Custom Queries**: Build application-specific queries

## Resources

- [GraphQL API Documentation](../../docs/graphql-api.md)
- [HTTP API Reference](../../docs/http-api.md)
- [Query Language](../../docs/query-engine.md)
- [Aggregation Pipeline](../../docs/aggregation.md)

## License

MIT License - see LICENSE file for details
