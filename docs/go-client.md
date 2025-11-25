# LauraDB Go Client Library

The LauraDB Go client library provides a native Go interface for interacting with LauraDB servers over HTTP. It offers a clean, idiomatic API that abstracts the underlying REST API calls.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Client Configuration](#client-configuration)
- [Connection Management](#connection-management)
- [Database Operations](#database-operations)
- [Collection Operations](#collection-operations)
- [Document Operations](#document-operations)
- [Query Operations](#query-operations)
- [Aggregation Pipeline](#aggregation-pipeline)
- [Index Management](#index-management)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Installation

```bash
go get github.com/mnohosten/laura-db/pkg/client
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/mnohosten/laura-db/pkg/client"
)

func main() {
    // Create a client
    c := client.NewDefaultClient()
    defer c.Close()

    // Get a collection handle
    users := c.Collection("users")

    // Insert a document
    doc := map[string]interface{}{
        "name": "Alice",
        "age":  int64(30),
        "city": "New York",
    }
    id, err := users.InsertOne(doc)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Inserted document with ID: %s\n", id)

    // Find a document
    found, err := users.FindOne(id)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found: %v\n", found)
}
```

## Client Configuration

### Default Configuration

```go
// Creates a client with default settings:
// - Host: localhost
// - Port: 8080
// - Timeout: 30s
// - MaxIdleConns: 10
// - MaxConnsPerHost: 10
c := client.NewDefaultClient()
defer c.Close()
```

### Custom Configuration

```go
config := &client.Config{
    Host:            "example.com",
    Port:            9090,
    Timeout:         10 * time.Second,
    MaxIdleConns:    20,
    MaxConnsPerHost: 20,
}

c := client.NewClient(config)
defer c.Close()
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Host` | string | "localhost" | Server hostname or IP address |
| `Port` | int | 8080 | Server port |
| `Timeout` | time.Duration | 30s | HTTP request timeout |
| `MaxIdleConns` | int | 10 | Maximum idle connections |
| `MaxConnsPerHost` | int | 10 | Maximum connections per host |

## Connection Management

### Creating a Client

```go
// Default configuration
c := client.NewDefaultClient()

// Custom configuration
config := &client.Config{
    Host: "db.example.com",
    Port: 8080,
}
c := client.NewClient(config)
```

### Closing a Client

```go
// Always close the client when done
defer c.Close()

// Or explicitly
err := c.Close()
if err != nil {
    log.Printf("Error closing client: %v", err)
}
```

### Health Checks

```go
health, err := c.Health()
if err != nil {
    log.Fatalf("Health check failed: %v", err)
}

fmt.Printf("Status: %s\n", health.Status)
fmt.Printf("Uptime: %s\n", health.Uptime)
fmt.Printf("Time: %v\n", health.Time)
```

## Database Operations

### List Collections

```go
collections, err := c.ListCollections()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Collections: %v\n", collections)
```

### Create Collection

```go
err := c.CreateCollection("users")
if err != nil {
    log.Fatal(err)
}
```

### Drop Collection

```go
err := c.DropCollection("users")
if err != nil {
    log.Fatal(err)
}
```

### Database Statistics

```go
stats, err := c.Stats()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Database: %s\n", stats.Name)
fmt.Printf("Collections: %d\n", stats.Collections)
fmt.Printf("Active Transactions: %d\n", stats.ActiveTransactions)

// Buffer pool stats
bp := stats.StorageStats.BufferPool
fmt.Printf("Buffer Pool Hit Rate: %.2f%%\n", bp.HitRate*100)
fmt.Printf("Buffer Pool Size: %d/%d\n", bp.Size, bp.Capacity)

// Disk stats
disk := stats.StorageStats.Disk
fmt.Printf("Total Reads: %d\n", disk.TotalReads)
fmt.Printf("Total Writes: %d\n", disk.TotalWrites)
```

## Collection Operations

### Get Collection Handle

```go
users := c.Collection("users")
```

### Collection Statistics

```go
stats, err := users.Stats()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Collection: %s\n", stats.Name)
fmt.Printf("Document Count: %d\n", stats.Count)
fmt.Printf("Index Count: %d\n", stats.Indexes)

// Index details
for _, idx := range stats.IndexDetails {
    fmt.Printf("  Index: %s (type: %s, unique: %v)\n",
        idx.Name, idx.Type, idx.Unique)
}
```

### Drop Collection

```go
err := users.Drop()
if err != nil {
    log.Fatal(err)
}
```

## Document Operations

### Insert One

```go
doc := map[string]interface{}{
    "name": "Alice",
    "age":  int64(30),
    "city": "New York",
}

id, err := users.InsertOne(doc)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Inserted with ID: %s\n", id)
```

### Insert One with Custom ID

```go
doc := map[string]interface{}{
    "name": "Bob",
    "age":  int64(25),
}

err := users.InsertOneWithID("custom-id-123", doc)
if err != nil {
    log.Fatal(err)
}
```

### Find One by ID

```go
doc, err := users.FindOne("507f1f77bcf86cd799439011")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found: %v\n", doc)
```

### Update One

```go
update := map[string]interface{}{
    "$set": map[string]interface{}{
        "age": int64(31),
        "city": "San Francisco",
    },
}

err := users.UpdateOne("507f1f77bcf86cd799439011", update)
if err != nil {
    log.Fatal(err)
}
```

**Update Operators:**
- `$set`: Set field values
- `$unset`: Remove fields
- `$inc`: Increment numeric values
- `$mul`: Multiply numeric values
- `$min`: Update if less than current
- `$max`: Update if greater than current
- `$push`: Add to array
- `$pull`: Remove from array
- `$addToSet`: Add unique to array
- `$pop`: Remove first/last from array
- `$rename`: Rename fields

### Delete One

```go
err := users.DeleteOne("507f1f77bcf86cd799439011")
if err != nil {
    log.Fatal(err)
}
```

### Bulk Operations

```go
operations := []client.BulkOperation{
    {
        Operation: "insert",
        Document: map[string]interface{}{
            "name": "Alice",
            "age":  int64(30),
        },
    },
    {
        Operation: "update",
        ID:        "507f1f77bcf86cd799439011",
        Update: map[string]interface{}{
            "$set": map[string]interface{}{"age": int64(31)},
        },
    },
    {
        Operation: "delete",
        ID:        "507f1f77bcf86cd799439012",
    },
}

result, err := users.Bulk(operations)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Inserted: %d\n", result.Inserted)
fmt.Printf("Updated: %d\n", result.Updated)
fmt.Printf("Deleted: %d\n", result.Deleted)
fmt.Printf("Failed: %d\n", result.Failed)
```

## Query Operations

### Find with Filter

```go
// Simple equality filter
filter := map[string]interface{}{
    "city": "New York",
}
docs, err := users.Find(filter)

// Comparison operators
filter = map[string]interface{}{
    "age": map[string]interface{}{
        "$gt": int64(25),
        "$lt": int64(40),
    },
}
docs, err = users.Find(filter)

// Logical operators
filter = map[string]interface{}{
    "$and": []interface{}{
        map[string]interface{}{"age": map[string]interface{}{"$gt": int64(25)}},
        map[string]interface{}{"city": "New York"},
    },
}
docs, err = users.Find(filter)
```

**Query Operators:**

**Comparison:**
- `$eq`: Equal to
- `$ne`: Not equal to
- `$gt`: Greater than
- `$gte`: Greater than or equal
- `$lt`: Less than
- `$lte`: Less than or equal
- `$in`: In array
- `$nin`: Not in array

**Logical:**
- `$and`: Logical AND
- `$or`: Logical OR
- `$not`: Logical NOT

**Element:**
- `$exists`: Field exists
- `$type`: Field type

**Array:**
- `$all`: All elements match
- `$elemMatch`: At least one element matches
- `$size`: Array size

**Evaluation:**
- `$regex`: Regular expression match

### Search with Options

```go
options := &client.SearchOptions{
    Filter: map[string]interface{}{
        "age": map[string]interface{}{"$gt": int64(25)},
    },
    Projection: map[string]interface{}{
        "name": 1,
        "age":  1,
    },
    Sort: map[string]interface{}{
        "age": -1, // Descending
    },
    Skip:  10,
    Limit: 20,
}

docs, err := users.Search(options)
```

### Find with Options

```go
filter := map[string]interface{}{
    "city": "New York",
}
projection := map[string]interface{}{
    "name": 1,
    "age":  1,
}
sort := map[string]interface{}{
    "age": -1,
}

docs, err := users.FindWithOptions(filter, projection, sort, 0, 10)
```

### Count Documents

```go
// Count all documents
count, err := users.Count(nil)

// Count with filter
filter := map[string]interface{}{
    "age": map[string]interface{}{"$gt": int64(25)},
}
count, err = users.Count(filter)
```

## Aggregation Pipeline

### Using Pipeline Builder

```go
pipeline := client.NewPipeline().
    Match(map[string]interface{}{
        "status": "active",
    }).
    Group("$city", map[string]interface{}{
        "count":     client.Count(),
        "avgAge":    client.Avg("age"),
        "totalAge":  client.Sum("age"),
        "minAge":    client.Min("age"),
        "maxAge":    client.Max("age"),
        "names":     client.Push("name"),
    }).
    Sort(map[string]interface{}{
        "count": -1,
    }).
    Limit(10).
    Build()

results, err := users.Aggregate(pipeline)
```

### Direct Aggregation

```go
pipeline := client.AggregationPipeline{
    {
        "$match": map[string]interface{}{
            "age": map[string]interface{}{"$gt": int64(25)},
        },
    },
    {
        "$group": map[string]interface{}{
            "_id":   "$city",
            "count": map[string]interface{}{"$sum": 1},
        },
    },
    {
        "$sort": map[string]interface{}{
            "count": -1,
        },
    },
}

results, err := users.Aggregate(pipeline)
```

### Pipeline Stages

- `$match`: Filter documents
- `$group`: Group documents and accumulate
- `$project`: Select and transform fields
- `$sort`: Sort documents
- `$limit`: Limit number of documents
- `$skip`: Skip documents

### Aggregation Accumulators

```go
client.Sum("field")         // Sum of field values
client.SumValue(1)          // Sum constant (count)
client.Avg("field")         // Average of field values
client.Min("field")         // Minimum field value
client.Max("field")         // Maximum field value
client.Push("field")        // Array of field values
client.Count()              // Count of documents
```

### Execute Pipeline

```go
// Using the builder's Execute method
results, err := client.NewPipeline().
    Match(filter).
    Group("$status", map[string]interface{}{
        "count": client.Count(),
    }).
    Execute(users)
```

## Index Management

### Create B-Tree Index

```go
// Simple B-tree index
err := users.CreateBTreeIndex("age_idx", "age", false)

// Unique B-tree index
err := users.CreateBTreeIndex("email_idx", "email", true)
```

### Create Compound Index

```go
fields := map[string]int{
    "city": 1,   // Ascending
    "age":  -1,  // Descending
}

err := users.CreateCompoundIndex("city_age_idx", fields, false)
```

### Create Text Index

```go
err := users.CreateTextIndex("desc_text", "description")
```

### Create Geospatial Indexes

```go
// 2D planar index
err := users.CreateGeo2DIndex("location_2d", "location")

// 2dsphere spherical index
err := users.CreateGeo2DSphereIndex("location_sphere", "coordinates")
```

### Create TTL Index

```go
// Documents expire 24 hours after createdAt timestamp
err := users.CreateTTLIndex("created_ttl", "createdAt", "24h")

// Other TTL durations: "1h", "30m", "7d", etc.
```

### Create Partial Index

```go
// Index only active users
filter := map[string]interface{}{
    "status": "active",
}

err := users.CreatePartialIndex("active_email_idx", "email", filter, true)
```

### Create Index (Advanced)

```go
options := client.IndexOptions{
    Name:   "complex_idx",
    Type:   client.IndexTypeBTree,
    Field:  "score",
    Unique: false,
    Sparse: true,
    PartialFilter: map[string]interface{}{
        "score": map[string]interface{}{
            "$gt": int64(90),
        },
    },
}

err := users.CreateIndex(options)
```

### List Indexes

```go
indexes, err := users.ListIndexes()
if err != nil {
    log.Fatal(err)
}

for _, idx := range indexes {
    fmt.Printf("Name: %s\n", idx.Name)
    fmt.Printf("Type: %s\n", idx.Type)
    fmt.Printf("Fields: %v\n", idx.Fields)
    fmt.Printf("Unique: %v\n", idx.Unique)
    fmt.Printf("Sparse: %v\n", idx.Sparse)
}
```

### Drop Index

```go
err := users.DropIndex("age_idx")
if err != nil {
    log.Fatal(err)
}
```

## Error Handling

The client returns errors that can be checked for specific types:

```go
doc, err := users.FindOne("invalid-id")
if err != nil {
    // Check error message for API errors
    if strings.Contains(err.Error(), "API error") {
        fmt.Println("Server returned an error")
    } else {
        fmt.Println("Network or client error")
    }
    return
}
```

**Common Errors:**

- Network errors: Connection refused, timeout
- API errors: Document not found, collection not found, duplicate key
- Client errors: Invalid JSON, encoding errors

## Best Practices

### 1. Always Close Clients

```go
c := client.NewDefaultClient()
defer c.Close()
```

### 2. Use Numeric Type Conversions

LauraDB stores numbers as `int64`. Always use explicit type conversions:

```go
doc := map[string]interface{}{
    "age":   int64(30),    // Correct
    "count": int64(100),   // Correct
}

// Avoid untyped integers
doc := map[string]interface{}{
    "age": 30,  // May cause issues
}
```

### 3. Reuse Collection Handles

```go
// Good: Reuse the collection handle
users := c.Collection("users")
for i := 0; i < 100; i++ {
    users.InsertOne(doc)
}

// Avoid: Creating new handles repeatedly
for i := 0; i < 100; i++ {
    c.Collection("users").InsertOne(doc)
}
```

### 4. Use Bulk Operations

```go
// Efficient: Single bulk request
operations := make([]client.BulkOperation, 100)
for i := 0; i < 100; i++ {
    operations[i] = client.BulkOperation{
        Operation: "insert",
        Document:  createDoc(i),
    }
}
result, _ := users.Bulk(operations)

// Inefficient: Multiple individual requests
for i := 0; i < 100; i++ {
    users.InsertOne(createDoc(i))
}
```

### 5. Configure Connection Pooling

```go
config := &client.Config{
    Host:            "db.example.com",
    Port:            8080,
    Timeout:         30 * time.Second,
    MaxIdleConns:    50,  // Increase for high load
    MaxConnsPerHost: 50,  // Match MaxIdleConns
}
c := client.NewClient(config)
```

### 6. Use Indexes for Queries

```go
// Create index for frequently queried fields
users.CreateBTreeIndex("city_idx", "city", false)

// Queries on indexed fields are much faster
docs, _ := users.Find(map[string]interface{}{"city": "New York"})
```

### 7. Handle Errors Appropriately

```go
id, err := users.InsertOne(doc)
if err != nil {
    if strings.Contains(err.Error(), "duplicate key") {
        // Handle duplicate key error
        log.Printf("Document already exists")
    } else {
        // Handle other errors
        log.Printf("Insert failed: %v", err)
    }
    return
}
```

## Examples

### Complete CRUD Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/mnohosten/laura-db/pkg/client"
)

func main() {
    // Connect
    c := client.NewDefaultClient()
    defer c.Close()

    // Get collection
    users := c.Collection("users")

    // Create
    doc := map[string]interface{}{
        "name":  "Alice",
        "email": "alice@example.com",
        "age":   int64(30),
    }
    id, err := users.InsertOne(doc)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created: %s\n", id)

    // Read
    found, err := users.FindOne(id)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Read: %v\n", found)

    // Update
    update := map[string]interface{}{
        "$set": map[string]interface{}{
            "age": int64(31),
        },
    }
    err = users.UpdateOne(id, update)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Updated")

    // Delete
    err = users.DeleteOne(id)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Deleted")
}
```

### Aggregation Example

```go
// Group users by city and calculate statistics
results, err := client.NewPipeline().
    Match(map[string]interface{}{
        "status": "active",
    }).
    Group("$city", map[string]interface{}{
        "totalUsers": client.Count(),
        "avgAge":     client.Avg("age"),
        "minAge":     client.Min("age"),
        "maxAge":     client.Max("age"),
    }).
    Sort(map[string]interface{}{
        "totalUsers": -1,
    }).
    Limit(10).
    Execute(users)

if err != nil {
    log.Fatal(err)
}

for _, r := range results {
    fmt.Printf("City: %s, Users: %.0f, Avg Age: %.1f\n",
        r["_id"], r["totalUsers"], r["avgAge"])
}
```

### Index Management Example

```go
// Create various indexes
users.CreateBTreeIndex("age_idx", "age", false)
users.CreateBTreeIndex("email_idx", "email", true)

fields := map[string]int{
    "city": 1,
    "age":  -1,
}
users.CreateCompoundIndex("city_age_idx", fields, false)

users.CreateTextIndex("bio_text", "biography")

// List all indexes
indexes, _ := users.ListIndexes()
for _, idx := range indexes {
    fmt.Printf("%s: %s\n", idx.Name, idx.Type)
}

// Drop an index
users.DropIndex("age_idx")
```

## Performance Tips

1. **Use connection pooling**: Configure `MaxIdleConns` and `MaxConnsPerHost` appropriately
2. **Leverage indexes**: Create indexes for frequently queried fields
3. **Use bulk operations**: Insert/update/delete multiple documents in one request
4. **Use projections**: Only retrieve fields you need
5. **Set timeouts**: Configure appropriate `Timeout` values for your use case
6. **Reuse clients**: Don't create new clients for every request
7. **Use aggregation pipelines**: More efficient than client-side data processing

## API Reference

For a complete API reference, see the [GoDoc documentation](https://pkg.go.dev/github.com/mnohosten/laura-db/pkg/client).

## See Also

- [HTTP API Reference](http-api.md)
- [Query Engine Documentation](query-engine.md)
- [Aggregation Pipeline](aggregation.md)
- [Index Types](indexing.md)
