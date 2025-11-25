# LauraDB API Reference

Complete API reference for LauraDB - an educational MongoDB-like document database written in Go.

## Table of Contents

1. [Database Operations](#database-operations)
2. [Collection Operations](#collection-operations)
3. [Session API (Transactions)](#session-api-transactions)
4. [Query Building](#query-building)
5. [Index Management](#index-management)
6. [Aggregation Pipeline](#aggregation-pipeline)
7. [HTTP Server](#http-server)
8. [Update Operators](#update-operators)
9. [Data Types](#data-types)
10. [Error Handling](#error-handling)

---

## Database Operations

### Opening a Database

```go
import "github.com/mnohosten/laura-db/pkg/database"

// Create configuration
config := database.DefaultConfig("/path/to/data")

// Customize configuration (optional)
config.BufferPoolSize = 2000

// Open database
db, err := database.Open(config)
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

### Database Methods

#### `Open(config *Config) (*Database, error)`
Opens or creates a database instance with the specified configuration.

**Parameters:**
- `config`: Database configuration (data directory, buffer pool size)

**Returns:**
- `*Database`: Database instance
- `error`: Error if database cannot be opened

**Example:**
```go
config := database.DefaultConfig("./data")
db, err := database.Open(config)
```

---

#### `Close() error`
Closes the database and flushes all pending data to disk.

**Returns:**
- `error`: Error if close fails

**Example:**
```go
err := db.Close()
```

---

#### `Collection(name string) *Collection`
Gets or creates a collection. Creates the collection if it doesn't exist.

**Parameters:**
- `name`: Collection name

**Returns:**
- `*Collection`: Collection instance

**Example:**
```go
users := db.Collection("users")
```

---

#### `CreateCollection(name string) (*Collection, error)`
Explicitly creates a new collection.

**Parameters:**
- `name`: Collection name

**Returns:**
- `*Collection`: Created collection
- `error`: Error if collection already exists

**Example:**
```go
coll, err := db.CreateCollection("products")
```

---

#### `DropCollection(name string) error`
Drops an entire collection and all its indexes.

**Parameters:**
- `name`: Collection name

**Returns:**
- `error`: Error if collection doesn't exist or drop fails

**Example:**
```go
err := db.DropCollection("old_data")
```

---

#### `ListCollections() []string`
Returns the names of all collections in the database.

**Returns:**
- `[]string`: List of collection names

**Example:**
```go
collections := db.ListCollections()
for _, name := range collections {
    fmt.Println(name)
}
```

---

#### `Stats() map[string]interface{}`
Returns database-level statistics.

**Returns:**
- `map[string]interface{}`: Statistics including collection count, transaction count, storage info

**Example:**
```go
stats := db.Stats()
fmt.Printf("Collections: %v\n", stats["collections"])
```

---

### Configuration

#### `Config` Struct
```go
type Config struct {
    DataDir        string        // Path to data directory
    BufferPoolSize int           // Number of pages in buffer pool (default: 1000)
    AuditConfig    *audit.Config // Optional audit logging configuration
}
```

**Configuration Fields:**

- **`DataDir`** (string, required)
  - Path to directory where all database files are stored
  - LauraDB creates: `data.db` (main data), `wal.log` (write-ahead log), `collections/` (metadata)
  - Default: `.laura-db` in current directory
  - Data persists across server restarts

- **`BufferPoolSize`** (int, default: 1000)
  - Number of 4KB pages to cache in memory
  - Default 1000 pages = ~4MB of page cache
  - Higher values = more memory usage but better read performance
  - Recommended: 2000-5000 for production workloads
  - Each collection also maintains a separate document cache (LRU)

- **`AuditConfig`** (*audit.Config, optional)
  - Configuration for audit logging
  - Set to `nil` to disable audit logging
  - See [Audit Logging Documentation](audit-logging.md) for details

#### `DefaultConfig(dataDir string) *Config`
Returns a configuration with sensible defaults.

**Parameters:**
- `dataDir`: Path to data directory

**Returns:**
- `*Config`: Configuration with defaults (BufferPoolSize: 1000)

**Example:**
```go
config := database.DefaultConfig("./mydata")

// Customize for production workload
config.BufferPoolSize = 5000  // 20MB buffer pool
```

---

## Collection Operations

### CRUD Operations

#### `InsertOne(doc map[string]interface{}) (string, error)`
Inserts a single document into the collection.

**Parameters:**
- `doc`: Document as a map (field name → value)

**Returns:**
- `string`: Generated or provided `_id` value
- `error`: Error if insert fails

**Example:**
```go
doc := map[string]interface{}{
    "name": "Alice",
    "age": int64(30),
    "email": "alice@example.com",
}
id, err := users.InsertOne(doc)
```

**Note:** If `_id` is not provided, an ObjectID is automatically generated.

---

#### `InsertMany(docs []map[string]interface{}) ([]string, error)`
Inserts multiple documents in a single operation.

**Parameters:**
- `docs`: Slice of documents

**Returns:**
- `[]string`: List of inserted document IDs
- `error`: Error if any insert fails (entire operation is atomic)

**Example:**
```go
docs := []map[string]interface{}{
    {"name": "Bob", "age": int64(25)},
    {"name": "Carol", "age": int64(35)},
}
ids, err := users.InsertMany(docs)
```

---

#### `FindOne(filter map[string]interface{}) (*document.Document, error)`
Finds the first document matching the filter.

**Parameters:**
- `filter`: Query filter (see [Query Operators](#query-operators))

**Returns:**
- `*document.Document`: Found document
- `error`: `ErrDocumentNotFound` if no match, or other error

**Example:**
```go
doc, err := users.FindOne(map[string]interface{}{
    "email": "alice@example.com",
})
if err == database.ErrDocumentNotFound {
    fmt.Println("User not found")
}
```

---

#### `Find(filter map[string]interface{}) ([]*document.Document, error)`
Finds all documents matching the filter.

**Parameters:**
- `filter`: Query filter (empty map `{}` matches all documents)

**Returns:**
- `[]*document.Document`: Slice of matching documents
- `error`: Error if query fails

**Example:**
```go
// Find all users over 25
docs, err := users.Find(map[string]interface{}{
    "age": map[string]interface{}{"$gt": int64(25)},
})
```

---

#### `FindWithOptions(filter map[string]interface{}, options *QueryOptions) ([]*document.Document, error)`
Finds documents with advanced options (projection, sort, pagination).

**Parameters:**
- `filter`: Query filter
- `options`: Query options (projection, sort, limit, skip)

**Returns:**
- `[]*document.Document`: Matching documents
- `error`: Error if query fails

**Example:**
```go
options := &database.QueryOptions{
    Projection: map[string]bool{"name": true, "email": true},
    Sort: []query.SortField{
        {Field: "age", Ascending: false},
    },
    Limit: 10,
    Skip: 20,
}
docs, err := users.FindWithOptions(map[string]interface{}{}, options)
```

---

#### `UpdateOne(filter map[string]interface{}, update map[string]interface{}) error`
Updates the first document matching the filter.

**Parameters:**
- `filter`: Query filter
- `update`: Update operations (see [Update Operators](#update-operators))

**Returns:**
- `error`: Error if update fails or no document matches

**Example:**
```go
err := users.UpdateOne(
    map[string]interface{}{"name": "Alice"},
    map[string]interface{}{
        "$set": map[string]interface{}{"age": int64(31)},
        "$inc": map[string]interface{}{"loginCount": int64(1)},
    },
)
```

---

#### `UpdateMany(filter map[string]interface{}, update map[string]interface{}) (int, error)`
Updates all documents matching the filter.

**Parameters:**
- `filter`: Query filter
- `update`: Update operations

**Returns:**
- `int`: Number of documents updated
- `error`: Error if update fails

**Example:**
```go
count, err := users.UpdateMany(
    map[string]interface{}{"status": "inactive"},
    map[string]interface{}{
        "$set": map[string]interface{}{"archived": true},
    },
)
fmt.Printf("Archived %d users\n", count)
```

---

#### `DeleteOne(filter map[string]interface{}) error`
Deletes the first document matching the filter.

**Parameters:**
- `filter`: Query filter

**Returns:**
- `error`: Error if delete fails or no document matches

**Example:**
```go
err := users.DeleteOne(map[string]interface{}{
    "_id": "507f1f77bcf86cd799439011",
})
```

---

#### `DeleteMany(filter map[string]interface{}) (int, error)`
Deletes all documents matching the filter.

**Parameters:**
- `filter`: Query filter

**Returns:**
- `int`: Number of documents deleted
- `error`: Error if delete fails

**Example:**
```go
count, err := users.DeleteMany(map[string]interface{}{
    "lastLogin": map[string]interface{}{
        "$lt": time.Now().AddDate(0, -6, 0),
    },
})
```

---

#### `Count(filter map[string]interface{}) (int, error)`
Counts documents matching the filter.

**Parameters:**
- `filter`: Query filter (empty map counts all documents)

**Returns:**
- `int`: Number of matching documents
- `error`: Error if count fails

**Example:**
```go
count, err := users.Count(map[string]interface{}{
    "active": true,
})
```

---

### Specialized Queries

#### `TextSearch(searchText string, options *QueryOptions) ([]*document.Document, error)`
Performs full-text search using text indexes with BM25 relevance scoring.

**Parameters:**
- `searchText`: Search query text
- `options`: Query options (projection, limit, skip)

**Returns:**
- `[]*document.Document`: Documents sorted by relevance score
- `error`: Error if no text index exists or search fails

**Example:**
```go
// Create text index first
err := articles.CreateTextIndex([]string{"title", "body"})

// Search
results, err := articles.TextSearch("golang database", &database.QueryOptions{
    Limit: 10,
})
```

---

#### `Near(fieldPath string, center *geo.Point, maxDistance float64, limit int, options *QueryOptions) ([]*document.Document, error)`
Finds documents near a geographic point.

**Parameters:**
- `fieldPath`: Field containing geospatial data
- `center`: Center point for proximity search
- `maxDistance`: Maximum distance (meters for 2dsphere, units for 2d)
- `limit`: Maximum number of results
- `options`: Additional query options

**Returns:**
- `[]*document.Document`: Documents sorted by distance
- `error`: Error if no geo index exists

**Example:**
```go
import "github.com/mnohosten/laura-db/pkg/geo"

// Create 2dsphere index first
err := locations.Create2DSphereIndex("coordinates")

// Find nearby locations
center := geo.NewPoint(-122.4194, 37.7749) // San Francisco
results, err := locations.Near("coordinates", center, 5000, 10, nil)
```

---

#### `GeoWithin(fieldPath string, polygon *geo.Polygon, options *QueryOptions) ([]*document.Document, error)`
Finds documents within a polygon.

**Parameters:**
- `fieldPath`: Field containing geospatial data
- `polygon`: Polygon boundary
- `options`: Query options

**Returns:**
- `[]*document.Document`: Documents inside polygon
- `error`: Error if query fails

**Example:**
```go
polygon := geo.NewPolygon([]*geo.Point{
    geo.NewPoint(-122.5, 37.8),
    geo.NewPoint(-122.3, 37.8),
    geo.NewPoint(-122.3, 37.7),
    geo.NewPoint(-122.5, 37.7),
    geo.NewPoint(-122.5, 37.8), // Close polygon
})
results, err := locations.GeoWithin("coordinates", polygon, nil)
```

---

#### `GeoIntersects(fieldPath string, box *geo.BoundingBox, options *QueryOptions) ([]*document.Document, error)`
Finds documents intersecting a bounding box.

**Parameters:**
- `fieldPath`: Field containing geospatial data
- `box`: Bounding box
- `options`: Query options

**Returns:**
- `[]*document.Document`: Documents intersecting box
- `error`: Error if query fails

**Example:**
```go
box := geo.NewBoundingBox(-122.5, 37.7, -122.3, 37.8)
results, err := locations.GeoIntersects("coordinates", box, nil)
```

---

### Aggregation

#### `Aggregate(pipeline []map[string]interface{}) ([]*document.Document, error)`
Executes an aggregation pipeline.

**Parameters:**
- `pipeline`: Slice of aggregation stages (see [Aggregation Pipeline](#aggregation-pipeline))

**Returns:**
- `[]*document.Document`: Aggregation results
- `error`: Error if pipeline execution fails

**Example:**
```go
pipeline := []map[string]interface{}{
    {"$match": map[string]interface{}{"status": "active"}},
    {"$group": map[string]interface{}{
        "_id": "$department",
        "avgSalary": map[string]interface{}{"$avg": "$salary"},
        "count": map[string]interface{}{"$sum": int64(1)},
    }},
    {"$sort": map[string]interface{}{"avgSalary": int64(-1)}},
}
results, err := employees.Aggregate(pipeline)
```

---

### Index Management

#### `CreateIndex(fieldPath string, unique bool) error`
Creates a B+ tree index on a single field.

**Parameters:**
- `fieldPath`: Field to index (use dot notation for nested fields)
- `unique`: Whether to enforce uniqueness

**Returns:**
- `error`: Error if index creation fails

**Example:**
```go
err := users.CreateIndex("email", true) // Unique index
err = users.CreateIndex("age", false)   // Non-unique index
```

---

#### `CreateIndexWithBackground(fieldPath string, unique bool, background bool) error`
Creates an index with optional background building (non-blocking).

**Parameters:**
- `fieldPath`: Field to index
- `unique`: Uniqueness constraint
- `background`: If true, index builds in background

**Returns:**
- `error`: Error if index creation fails

**Example:**
```go
// Create index in background (doesn't block)
err := users.CreateIndexWithBackground("username", true, true)

// Monitor progress
progress, _ := users.GetIndexBuildProgress("username_1")
fmt.Printf("Build progress: %v%%\n", progress["percentComplete"])
```

---

#### `CreateCompoundIndex(fieldPaths []string, unique bool) error`
Creates a compound index on multiple fields.

**Parameters:**
- `fieldPaths`: Fields to include in compound index (order matters)
- `unique`: Uniqueness constraint on combination

**Returns:**
- `error`: Error if index creation fails

**Example:**
```go
// Compound index on city and age (in that order)
err := users.CreateCompoundIndex([]string{"city", "age"}, false)

// Queries can use index prefix:
// - {"city": "NYC"} uses index
// - {"city": "NYC", "age": 30} uses full index
// - {"age": 30} does NOT use index (missing prefix)
```

---

#### `CreatePartialIndex(fieldPath string, filter map[string]interface{}, unique bool) error`
Creates a partial index (indexes only documents matching filter).

**Parameters:**
- `fieldPath`: Field to index
- `filter`: Filter expression (only matching documents are indexed)
- `unique`: Uniqueness constraint

**Returns:**
- `error`: Error if index creation fails

**Example:**
```go
// Index only active users
err := users.CreatePartialIndex(
    "email",
    map[string]interface{}{"status": "active"},
    true,
)
```

---

#### `CreateTextIndex(fieldPaths []string) error`
Creates a text index for full-text search.

**Parameters:**
- `fieldPaths`: Fields to include in text index

**Returns:**
- `error`: Error if index creation fails

**Example:**
```go
err := articles.CreateTextIndex([]string{"title", "body", "tags"})
```

---

#### `Create2DIndex(fieldPath string) error`
Creates a 2D planar geospatial index (Euclidean distance).

**Parameters:**
- `fieldPath`: Field containing `[longitude, latitude]` or GeoJSON point

**Returns:**
- `error`: Error if index creation fails

**Example:**
```go
err := locations.Create2DIndex("coordinates")
```

---

#### `Create2DSphereIndex(fieldPath string) error`
Creates a 2dsphere index for spherical (Earth) coordinates.

**Parameters:**
- `fieldPath`: Field containing `[longitude, latitude]` or GeoJSON point

**Returns:**
- `error`: Error if index creation fails

**Example:**
```go
err := locations.Create2DSphereIndex("coordinates")
```

---

#### `CreateTTLIndex(fieldPath string, ttlSeconds int64) error`
Creates a TTL (time-to-live) index for automatic document expiration.

**Parameters:**
- `fieldPath`: Field containing timestamp (time.Time or RFC3339 string)
- `ttlSeconds`: Documents expire after this many seconds

**Returns:**
- `error`: Error if index creation fails

**Example:**
```go
// Delete sessions after 1 hour (3600 seconds)
err := sessions.CreateTTLIndex("createdAt", 3600)
```

**Note:** Cleanup runs every 60 seconds in the background.

---

#### `DropIndex(indexName string) error`
Drops an index.

**Parameters:**
- `indexName`: Name of index to drop

**Returns:**
- `error`: Error if index doesn't exist

**Example:**
```go
err := users.DropIndex("age_1")
```

---

#### `ListIndexes() []map[string]interface{}`
Lists all indexes with their statistics.

**Returns:**
- `[]map[string]interface{}`: Index information (name, type, fields, stats)

**Example:**
```go
indexes := users.ListIndexes()
for _, idx := range indexes {
    fmt.Printf("Index: %s, Type: %s\n", idx["name"], idx["type"])
}
```

---

#### `GetIndexBuildProgress(indexName string) (map[string]interface{}, error)`
Gets the build progress for a background index.

**Parameters:**
- `indexName`: Name of index

**Returns:**
- `map[string]interface{}`: Progress info (state, itemsProcessed, totalItems, percentComplete)
- `error`: Error if index doesn't exist

**Example:**
```go
progress, err := users.GetIndexBuildProgress("username_1")
if progress["state"] == "building" {
    fmt.Printf("Progress: %.1f%%\n", progress["percentComplete"])
}
```

---

### Analysis & Diagnostics

#### `Explain(filter map[string]interface{}) map[string]interface{}`
Returns the query execution plan.

**Parameters:**
- `filter`: Query filter

**Returns:**
- `map[string]interface{}`: Execution plan (index selection, scan type, cost estimate)

**Example:**
```go
plan := users.Explain(map[string]interface{}{"age": int64(30)})
fmt.Printf("Using index: %v\n", plan["indexName"])
fmt.Printf("Scan type: %v\n", plan["scanType"])
```

---

#### `Analyze()`
Recalculates index statistics for better query optimization.

**Example:**
```go
users.Analyze()
```

**Note:** Called automatically on significant collection changes.

---

#### `Stats() map[string]interface{}`
Returns collection statistics.

**Returns:**
- `map[string]interface{}`: Stats (document count, index info, size estimates)

**Example:**
```go
stats := users.Stats()
fmt.Printf("Documents: %v\n", stats["count"])
```

---

#### `CleanupExpiredDocuments() int`
Manually triggers TTL cleanup (normally runs automatically).

**Returns:**
- `int`: Number of documents deleted

**Example:**
```go
deleted := sessions.CleanupExpiredDocuments()
```

---

## Session API (Transactions)

Sessions provide multi-document ACID transactions with snapshot isolation.

### Starting a Session

```go
session := db.StartSession()
defer session.AbortTransaction() // Safety net

// ... perform operations ...

err := session.CommitTransaction()
```

---

### Session Methods

#### `CommitTransaction() error`
Commits all pending operations in the session.

**Returns:**
- `error`: Error if commit fails (e.g., write conflict)

**Example:**
```go
err := session.CommitTransaction()
if err == mvcc.ErrConflict {
    // Handle write conflict
}
```

---

#### `AbortTransaction() error`
Aborts the transaction without applying changes.

**Returns:**
- `error`: Error if abort fails

**Example:**
```go
err := session.AbortTransaction()
```

---

#### `InsertOne(collName string, doc map[string]interface{}) (string, error)`
Inserts a document within the transaction.

**Parameters:**
- `collName`: Collection name
- `doc`: Document to insert

**Returns:**
- `string`: Document ID
- `error`: Error if insert fails

**Example:**
```go
id, err := session.InsertOne("users", map[string]interface{}{
    "name": "Alice",
})
```

---

#### `FindOne(collName string, filter map[string]interface{}) (*document.Document, error)`
Finds a document within the transaction (sees own writes).

**Parameters:**
- `collName`: Collection name
- `filter`: Query filter

**Returns:**
- `*document.Document`: Found document
- `error`: Error if not found

**Example:**
```go
doc, err := session.FindOne("users", map[string]interface{}{
    "_id": id,
})
```

---

#### `UpdateOne(collName string, filter map[string]interface{}, update map[string]interface{}) error`
Updates a document within the transaction.

**Parameters:**
- `collName`: Collection name
- `filter`: Query filter
- `update`: Update operations

**Returns:**
- `error`: Error if update fails

**Example:**
```go
err := session.UpdateOne("users",
    map[string]interface{}{"_id": id},
    map[string]interface{}{"$inc": map[string]interface{}{"balance": int64(-100)}},
)
```

---

#### `DeleteOne(collName string, filter map[string]interface{}) error`
Deletes a document within the transaction.

**Parameters:**
- `collName`: Collection name
- `filter`: Query filter

**Returns:**
- `error`: Error if delete fails

**Example:**
```go
err := session.DeleteOne("logs", map[string]interface{}{"_id": logID})
```

---

### Savepoints

Savepoints allow partial rollback within a transaction.

#### `CreateSavepoint(name string) error`
Creates a named savepoint.

**Parameters:**
- `name`: Savepoint name

**Returns:**
- `error`: Error if savepoint creation fails

**Example:**
```go
err := session.CreateSavepoint("before_transfer")
```

---

#### `RollbackToSavepoint(name string) error`
Rolls back to a savepoint without aborting the entire transaction.

**Parameters:**
- `name`: Savepoint name

**Returns:**
- `error`: Error if savepoint doesn't exist

**Example:**
```go
err := session.RollbackToSavepoint("before_transfer")
```

---

#### `ReleaseSavepoint(name string) error`
Releases a savepoint (frees resources).

**Parameters:**
- `name`: Savepoint name

**Returns:**
- `error`: Error if savepoint doesn't exist

**Example:**
```go
err := session.ReleaseSavepoint("before_transfer")
```

---

#### `ListSavepoints() []string`
Lists all active savepoint names.

**Returns:**
- `[]string`: Savepoint names

**Example:**
```go
savepoints := session.ListSavepoints()
```

---

### Transaction Helper

#### `WithTransaction(fn func(session *Session) error) error`
Executes a function within a transaction with automatic commit/abort.

**Parameters:**
- `fn`: Function to execute (receives session)

**Returns:**
- `error`: Error from function or commit

**Example:**
```go
err := db.WithTransaction(func(session *database.Session) error {
    // Transfer money between accounts
    _, err := session.UpdateOne("accounts",
        map[string]interface{}{"_id": fromID},
        map[string]interface{}{"$inc": map[string]interface{}{"balance": int64(-100)}},
    )
    if err != nil {
        return err
    }

    _, err = session.UpdateOne("accounts",
        map[string]interface{}{"_id": toID},
        map[string]interface{}{"$inc": map[string]interface{}{"balance": int64(100)}},
    )
    return err
})
```

---

## Query Building

### Query Operators

#### Comparison Operators

```go
// Equal
{"age": int64(30)}
{"age": map[string]interface{}{"$eq": int64(30)}}

// Not equal
{"status": map[string]interface{}{"$ne": "inactive"}}

// Greater than
{"age": map[string]interface{}{"$gt": int64(25)}}

// Greater than or equal
{"age": map[string]interface{}{"$gte": int64(25)}}

// Less than
{"age": map[string]interface{}{"$lt": int64(50)}}

// Less than or equal
{"age": map[string]interface{}{"$lte": int64(50)}}

// In array
{"status": map[string]interface{}{"$in": []interface{}{"active", "pending"}}}

// Not in array
{"status": map[string]interface{}{"$nin": []interface{}{"deleted", "banned"}}}
```

---

#### Logical Operators

```go
// AND (implicit)
{"age": int64(30), "status": "active"}

// AND (explicit)
{"$and": []interface{}{
    map[string]interface{}{"age": map[string]interface{}{"$gte": int64(25)}},
    map[string]interface{}{"age": map[string]interface{}{"$lte": int64(50)}},
}}

// OR
{"$or": []interface{}{
    map[string]interface{}{"status": "active"},
    map[string]interface{}{"priority": "high"},
}}

// NOT
{"age": map[string]interface{}{
    "$not": map[string]interface{}{"$lt": int64(18)},
}}
```

---

#### Element Operators

```go
// Field exists
{"email": map[string]interface{}{"$exists": true}}

// Field doesn't exist
{"deletedAt": map[string]interface{}{"$exists": false}}

// Type check
{"age": map[string]interface{}{"$type": "number"}}
```

---

#### Array Operators

```go
// All elements match
{"tags": map[string]interface{}{"$all": []interface{}{"go", "database"}}}

// Array element matches sub-query
{"scores": map[string]interface{}{
    "$elemMatch": map[string]interface{}{
        "$gte": int64(80),
        "$lt": int64(90),
    },
}}

// Array size
{"tags": map[string]interface{}{"$size": int64(3)}}
```

---

#### Evaluation Operators

```go
// Regular expression
{"email": map[string]interface{}{
    "$regex": ".*@example\\.com$",
}}

// Modulo
{"qty": map[string]interface{}{
    "$mod": []interface{}{int64(5), int64(0)}, // divisible by 5
}}
```

---

### QueryOptions

```go
type QueryOptions struct {
    Projection map[string]bool    // Field inclusion/exclusion
    Sort       []query.SortField  // Sort order
    Limit      int                // Maximum results
    Skip       int                // Skip initial results
}
```

#### Projection

```go
// Include only specific fields
options := &database.QueryOptions{
    Projection: map[string]bool{
        "name": true,
        "email": true,
    },
}

// Exclude specific fields
options := &database.QueryOptions{
    Projection: map[string]bool{
        "password": false,
        "ssn": false,
    },
}
```

**Note:** Cannot mix inclusion and exclusion (except for `_id`).

---

#### Sorting

```go
import "github.com/mnohosten/laura-db/pkg/query"

options := &database.QueryOptions{
    Sort: []query.SortField{
        {Field: "age", Ascending: false},     // Descending by age
        {Field: "name", Ascending: true},     // Then ascending by name
    },
}
```

---

#### Pagination

```go
// Page 3, 20 items per page
options := &database.QueryOptions{
    Skip: 40,   // Skip first 40 results
    Limit: 20,  // Return next 20 results
}
```

---

## Index Management

### Index Types

LauraDB supports multiple index types for different use cases:

1. **B+ Tree Index** - General-purpose, supports range queries
2. **Compound Index** - Multiple fields, prefix matching
3. **Text Index** - Full-text search with BM25 scoring
4. **Geospatial Index** - 2D planar or 2dsphere for geographic data
5. **TTL Index** - Automatic document expiration
6. **Partial Index** - Filtered index for subset of documents

### Index Naming

Indexes are automatically named using the pattern:
- Single field: `fieldname_1`
- Compound: `field1_1_field2_1_field3_1`
- Text: `text_field1_field2`
- Geo: `geo_fieldname` or `geo2dsphere_fieldname`
- TTL: `ttl_fieldname`

### Index Statistics

```go
indexes := collection.ListIndexes()
for _, idx := range indexes {
    fmt.Printf("Name: %s\n", idx["name"])
    fmt.Printf("Type: %s\n", idx["type"])
    fmt.Printf("Fields: %v\n", idx["fields"])
    fmt.Printf("Unique: %v\n", idx["unique"])
    fmt.Printf("Size: %v\n", idx["size"])

    if stats, ok := idx["stats"].(map[string]interface{}); ok {
        fmt.Printf("Cardinality: %v\n", stats["cardinality"])
        fmt.Printf("Selectivity: %v\n", stats["selectivity"])
    }
}
```

---

## Aggregation Pipeline

### Pipeline Structure

An aggregation pipeline is a sequence of stages that process documents:

```go
pipeline := []map[string]interface{}{
    // Stage 1
    {"$match": map[string]interface{}{ /* filter */ }},
    // Stage 2
    {"$group": map[string]interface{}{ /* grouping */ }},
    // Stage 3
    {"$sort": map[string]interface{}{ /* sort order */ }},
}

results, err := collection.Aggregate(pipeline)
```

---

### Aggregation Stages

#### `$match` - Filter Documents

```go
{"$match": map[string]interface{}{
    "status": "active",
    "age": map[string]interface{}{"$gte": int64(18)},
}}
```

Filters documents using query operators. Should be early in pipeline for performance.

---

#### `$group` - Group and Aggregate

```go
{"$group": map[string]interface{}{
    "_id": "$department",  // Group by field
    "totalSalary": map[string]interface{}{"$sum": "$salary"},
    "avgAge": map[string]interface{}{"$avg": "$age"},
    "maxSalary": map[string]interface{}{"$max": "$salary"},
    "minSalary": map[string]interface{}{"$min": "$salary"},
    "count": map[string]interface{}{"$sum": int64(1)},
    "employees": map[string]interface{}{"$push": "$name"},
}}
```

**Group Key:**
- String (e.g., `"$department"`): Group by field value
- `nil` or `null`: Group all documents together

**Accumulators:**
- `$sum`: Sum of values
- `$avg`: Average of values
- `$min`: Minimum value
- `$max`: Maximum value
- `$count`: Count documents
- `$push`: Array of values

---

#### `$project` - Select and Transform Fields

```go
{"$project": map[string]interface{}{
    "name": int64(1),           // Include field
    "email": int64(1),          // Include field
    "password": int64(0),       // Exclude field
    "fullName": map[string]interface{}{  // Computed field (not yet supported)
        "$concat": []interface{}{"$firstName", " ", "$lastName"},
    },
}}
```

**Note:** Currently supports inclusion/exclusion. Field transformations planned for future release.

---

#### `$sort` - Sort Results

```go
{"$sort": map[string]interface{}{
    "age": int64(-1),    // Descending
    "name": int64(1),    // Ascending
}}
```

---

#### `$limit` - Limit Results

```go
{"$limit": int64(10)}
```

---

#### `$skip` - Skip Initial Results

```go
{"$skip": int64(20)}
```

---

### Complete Example

```go
// Calculate average order value by customer, top 10
pipeline := []map[string]interface{}{
    // Filter completed orders
    {"$match": map[string]interface{}{
        "status": "completed",
    }},

    // Group by customer
    {"$group": map[string]interface{}{
        "_id": "$customerId",
        "totalOrders": map[string]interface{}{"$sum": int64(1)},
        "totalSpent": map[string]interface{}{"$sum": "$amount"},
        "avgOrderValue": map[string]interface{}{"$avg": "$amount"},
    }},

    // Sort by total spent
    {"$sort": map[string]interface{}{
        "totalSpent": int64(-1),
    }},

    // Top 10
    {"$limit": int64(10)},
}

results, err := orders.Aggregate(pipeline)
```

---

## HTTP Server

LauraDB includes a REST API server for remote access.

### Starting the Server

```go
import "github.com/mnohosten/laura-db/pkg/server"

config := server.DefaultConfig()
config.Port = 8080
config.DataDir = "./data"

srv, err := server.New(config)
if err != nil {
    log.Fatal(err)
}

// Start server (blocking)
err = srv.Start()

// Or start with graceful shutdown
go srv.Start()

// Shutdown
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(ctx)
```

---

### Server Configuration

```go
type Config struct {
    Host           string        // Bind address (default: "localhost")
    Port           int           // Port (default: 8080)
    DataDir        string        // Data directory
    BufferSize     int           // Buffer pool size
    ReadTimeout    time.Duration // Read timeout (default: 30s)
    WriteTimeout   time.Duration // Write timeout (default: 30s)
    IdleTimeout    time.Duration // Idle timeout (default: 120s)
    MaxRequestSize int64         // Max request size (default: 10MB)
    EnableCORS     bool          // Enable CORS (default: true)
    AllowedOrigins []string      // CORS origins (default: ["*"])
    AllowedMethods []string      // CORS methods
    AllowedHeaders []string      // CORS headers
    EnableLogging  bool          // Enable request logging (default: true)
    LogFormat      string        // Log format (default: "json")
}
```

---

### REST API Endpoints

All endpoints use JSON for request/response bodies.

#### Health & Stats

```
GET /_health
```
Returns server health status.

**Response:**
```json
{"status": "ok"}
```

---

```
GET /_stats
```
Returns database statistics.

**Response:**
```json
{
  "collections": 5,
  "transactions": {"active": 2, "committed": 150, "aborted": 3},
  "storage": {"bufferPoolSize": 1000, "pagesInMemory": 234}
}
```

---

```
GET /_collections
```
Lists all collections.

**Response:**
```json
{
  "collections": ["users", "products", "orders"]
}
```

---

#### Collection Management

```
PUT /{collection}/
```
Creates a collection.

**Response:**
```json
{"message": "Collection created", "collection": "users"}
```

---

```
DELETE /{collection}/
```
Drops a collection.

**Response:**
```json
{"message": "Collection dropped", "collection": "users"}
```

---

```
GET /{collection}/_stats
```
Returns collection statistics.

**Response:**
```json
{
  "name": "users",
  "count": 1000,
  "indexes": [...]
}
```

---

#### Document Operations

```
POST /{collection}/_doc
```
Inserts a document (auto-generated ID).

**Request:**
```json
{"name": "Alice", "age": 30}
```

**Response:**
```json
{"id": "507f1f77bcf86cd799439011"}
```

---

```
POST /{collection}/_doc/{id}
```
Inserts a document with specific ID.

**Request:**
```json
{"name": "Alice", "age": 30}
```

**Response:**
```json
{"id": "alice123"}
```

---

```
GET /{collection}/_doc/{id}
```
Retrieves a document by ID.

**Response:**
```json
{
  "_id": "507f1f77bcf86cd799439011",
  "name": "Alice",
  "age": 30
}
```

---

```
PUT /{collection}/_doc/{id}
```
Updates a document.

**Request:**
```json
{
  "$set": {"age": 31},
  "$inc": {"loginCount": 1}
}
```

**Response:**
```json
{"message": "Document updated", "id": "507f1f77bcf86cd799439011"}
```

---

```
DELETE /{collection}/_doc/{id}
```
Deletes a document.

**Response:**
```json
{"message": "Document deleted", "id": "507f1f77bcf86cd799439011"}
```

---

#### Bulk Operations

```
POST /{collection}/_bulk
```
Inserts multiple documents.

**Request:**
```json
{
  "documents": [
    {"name": "Alice", "age": 30},
    {"name": "Bob", "age": 25}
  ]
}
```

**Response:**
```json
{
  "inserted": 2,
  "ids": ["507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012"]
}
```

---

#### Query Operations

```
POST /{collection}/_search
```
Searches documents.

**Request:**
```json
{
  "filter": {"age": {"$gte": 25}},
  "projection": {"name": true, "age": true},
  "sort": [{"field": "age", "ascending": false}],
  "limit": 10,
  "skip": 0
}
```

**Response:**
```json
{
  "documents": [
    {"_id": "...", "name": "Alice", "age": 30},
    {"_id": "...", "name": "Bob", "age": 25}
  ],
  "count": 2
}
```

---

```
GET /{collection}/_count
POST /{collection}/_count
```
Counts documents.

**POST Request:**
```json
{"filter": {"status": "active"}}
```

**Response:**
```json
{"count": 150}
```

---

#### Aggregation

```
POST /{collection}/_aggregate
```
Executes aggregation pipeline.

**Request:**
```json
{
  "pipeline": [
    {"$match": {"status": "active"}},
    {"$group": {
      "_id": "$department",
      "count": {"$sum": 1}
    }}
  ]
}
```

**Response:**
```json
{
  "results": [
    {"_id": "Engineering", "count": 50},
    {"_id": "Sales", "count": 30}
  ]
}
```

---

#### Index Management

```
POST /{collection}/_index
```
Creates an index.

**Request:**
```json
{
  "field": "email",
  "unique": true,
  "background": false
}
```

**Response:**
```json
{"message": "Index created", "name": "email_1"}
```

---

```
GET /{collection}/_index
```
Lists indexes.

**Response:**
```json
{
  "indexes": [
    {
      "name": "email_1",
      "type": "btree",
      "fields": ["email"],
      "unique": true,
      "size": 1000
    }
  ]
}
```

---

```
DELETE /{collection}/_index/{name}
```
Drops an index.

**Response:**
```json
{"message": "Index dropped", "name": "email_1"}
```

---

## Update Operators

Update operators modify documents during update operations.

### Field Operators

#### `$set` - Set Field Values

```go
{"$set": map[string]interface{}{
    "status": "active",
    "lastModified": time.Now(),
}}
```

Sets field values, creating fields if they don't exist.

---

#### `$unset` - Remove Fields

```go
{"$unset": map[string]interface{}{
    "temporaryField": "",
    "deprecated": "",
}}
```

Removes fields from documents (value is ignored).

---

#### `$rename` - Rename Fields

```go
{"$rename": map[string]interface{}{
    "oldName": "newName",
    "email": "emailAddress",
}}
```

Renames fields.

---

#### `$currentDate` - Set to Current Date

```go
{"$currentDate": map[string]interface{}{
    "lastModified": true,
    "updatedAt": true,
}}
```

Sets field to current timestamp.

---

### Numeric Operators

#### `$inc` - Increment

```go
{"$inc": map[string]interface{}{
    "views": int64(1),
    "balance": int64(-100),  // Decrement
}}
```

Increments numeric fields (use negative to decrement).

---

#### `$mul` - Multiply

```go
{"$mul": map[string]interface{}{
    "price": 1.1,  // Increase by 10%
}}
```

Multiplies numeric field by value.

---

#### `$min` - Update if Less Than Current

```go
{"$min": map[string]interface{}{
    "lowestPrice": int64(99),
}}
```

Updates field only if new value is less than current value.

---

#### `$max` - Update if Greater Than Current

```go
{"$max": map[string]interface{}{
    "highScore": int64(9500),
}}
```

Updates field only if new value is greater than current value.

---

### Array Operators

#### `$push` - Add to Array

```go
// Simple push
{"$push": map[string]interface{}{
    "tags": "new-tag",
}}

// Push multiple with $each
{"$push": map[string]interface{}{
    "tags": map[string]interface{}{
        "$each": []interface{}{"tag1", "tag2", "tag3"},
    },
}}
```

Appends element(s) to array.

---

#### `$addToSet` - Add Unique to Array

```go
// Simple add
{"$addToSet": map[string]interface{}{
    "tags": "unique-tag",
}}

// Add multiple with $each
{"$addToSet": map[string]interface{}{
    "tags": map[string]interface{}{
        "$each": []interface{}{"tag1", "tag2"},
    },
}}
```

Adds to array only if element doesn't already exist.

---

#### `$pull` - Remove from Array

```go
{"$pull": map[string]interface{}{
    "tags": "old-tag",
    "scores": map[string]interface{}{"$lt": int64(50)},  // Remove matching condition
}}
```

Removes all matching elements from array.

---

#### `$pullAll` - Remove Multiple from Array

```go
{"$pullAll": map[string]interface{}{
    "tags": []interface{}{"tag1", "tag2", "tag3"},
}}
```

Removes multiple specific values from array.

---

#### `$pop` - Remove First or Last Element

```go
// Remove last element
{"$pop": map[string]interface{}{
    "items": int64(1),
}}

// Remove first element
{"$pop": map[string]interface{}{
    "items": int64(-1),
}}
```

Removes first (`-1`) or last (`1`) array element.

---

### Bitwise Operators

#### `$bit` - Bitwise Operations

```go
{"$bit": map[string]interface{}{
    "flags": map[string]interface{}{
        "and": int64(0xFF00),  // Bitwise AND
    },
}}

{"$bit": map[string]interface{}{
    "flags": map[string]interface{}{
        "or": int64(0x0001),   // Bitwise OR
    },
}}

{"$bit": map[string]interface{}{
    "flags": map[string]interface{}{
        "xor": int64(0x000F),  // Bitwise XOR
    },
}}
```

Performs bitwise operations on integer fields.

---

### Combining Operators

Multiple operators can be used in a single update:

```go
update := map[string]interface{}{
    "$set": map[string]interface{}{
        "status": "active",
    },
    "$inc": map[string]interface{}{
        "loginCount": int64(1),
    },
    "$push": map[string]interface{}{
        "loginHistory": time.Now(),
    },
    "$currentDate": map[string]interface{}{
        "lastLogin": true,
    },
}

err := users.UpdateOne(filter, update)
```

---

## Data Types

### Document

Documents are represented as `map[string]interface{}` in Go.

**Important:** All numeric values must be `int64` to match BSON encoding:

```go
doc := map[string]interface{}{
    "name": "Alice",
    "age": int64(30),        // Correct
    "score": int64(95),      // Correct
    // "count": 42,          // Wrong - will cause type errors
}
```

---

### ObjectID

MongoDB-compatible 12-byte unique identifier.

#### Creating ObjectIDs

```go
import "github.com/mnohosten/laura-db/pkg/document"

// Generate new ObjectID
id := document.NewObjectID()

// Parse from hex string
id, err := document.ObjectIDFromHex("507f1f77bcf86cd799439011")
```

#### ObjectID Methods

```go
id := document.NewObjectID()

hex := id.Hex()                    // Hex string representation
timestamp := id.Timestamp()        // Creation timestamp
machineID := id.MachineID()       // Machine identifier
processID := id.ProcessID()       // Process identifier
counter := id.Counter()           // Counter value
```

---

### Geospatial Types

#### Point

```go
import "github.com/mnohosten/laura-db/pkg/geo"

// Create point
point := geo.NewPoint(-122.4194, 37.7749)  // longitude, latitude

// Parse GeoJSON
point, err := geo.ParseGeoJSONPoint(map[string]interface{}{
    "type": "Point",
    "coordinates": []interface{}{-122.4194, 37.7749},
})

// Access coordinates
lng := point.Longitude
lat := point.Latitude
```

---

#### Polygon

```go
// Create polygon from points
polygon := geo.NewPolygon([]*geo.Point{
    geo.NewPoint(-122.5, 37.8),
    geo.NewPoint(-122.3, 37.8),
    geo.NewPoint(-122.3, 37.7),
    geo.NewPoint(-122.5, 37.7),
    geo.NewPoint(-122.5, 37.8),  // Close polygon
})
```

---

#### BoundingBox

```go
// Create bounding box
box := geo.NewBoundingBox(
    -122.5, 37.7,  // minLng, minLat
    -122.3, 37.8,  // maxLng, maxLat
)
```

---

### Time Types

LauraDB supports multiple time representations:

```go
// time.Time
doc := map[string]interface{}{
    "createdAt": time.Now(),
}

// RFC3339 string (parsed automatically)
doc := map[string]interface{}{
    "createdAt": "2024-01-15T10:30:00Z",
}

// Unix timestamp (int64)
doc := map[string]interface{}{
    "createdAt": time.Now().Unix(),
}
```

---

## Error Handling

### Common Errors

```go
import "github.com/mnohosten/laura-db/pkg/database"

// Document not found
doc, err := users.FindOne(filter)
if err == database.ErrDocumentNotFound {
    // Handle not found
}

// Collection not found
if err == database.ErrCollectionNotFound {
    // Handle missing collection
}

// Database closed
if err == database.ErrDatabaseClosed {
    // Handle closed database
}

// Duplicate key (unique constraint violation)
_, err = users.InsertOne(doc)
if err == database.ErrDuplicateKey {
    // Handle duplicate
}
```

---

### Transaction Errors

```go
import "github.com/mnohosten/laura-db/pkg/mvcc"

// Write conflict
err := session.CommitTransaction()
if err == mvcc.ErrConflict {
    // Retry transaction
}

// Transaction aborted
if err == mvcc.ErrAborted {
    // Transaction was aborted
}

// Inactive transaction
if err == mvcc.ErrInactive {
    // Transaction not active
}
```

---

### Error Checking Pattern

```go
// Always check errors
result, err := collection.Find(filter)
if err != nil {
    log.Printf("Query failed: %v", err)
    return err
}

// Use errors.Is for wrapped errors
if errors.Is(err, database.ErrDocumentNotFound) {
    // Handle specific error
}
```

---

## Best Practices

### Numeric Values

Always use `int64` for numbers:

```go
// Correct
doc := map[string]interface{}{
    "age": int64(30),
    "count": int64(100),
}

// Incorrect
doc := map[string]interface{}{
    "age": 30,      // Type int, will cause errors
    "count": 100,   // Type int, will cause errors
}
```

---

### Index Usage

1. **Create indexes on frequently queried fields**
2. **Use compound indexes for multi-field queries**
3. **Use partial indexes to save space**
4. **Background index creation for large collections**
5. **Call `Analyze()` after bulk operations**

```go
// Good: Create index before querying
users.CreateIndex("email", true)

// Good: Compound index for common query pattern
users.CreateCompoundIndex([]string{"city", "age"}, false)

// Good: Partial index for subset
users.CreatePartialIndex("email",
    map[string]interface{}{"status": "active"},
    true,
)
```

---

### Transaction Usage

1. **Keep transactions short**
2. **Use `WithTransaction` for automatic cleanup**
3. **Handle write conflicts with retry logic**
4. **Use savepoints for complex transactions**

```go
// Good: Short transaction with auto-cleanup
err := db.WithTransaction(func(s *database.Session) error {
    // Quick operations
    return nil
})

// Good: Retry on conflict
for retries := 0; retries < 3; retries++ {
    err := db.WithTransaction(func(s *database.Session) error {
        // Transaction logic
        return nil
    })
    if err != mvcc.ErrConflict {
        break
    }
    time.Sleep(time.Millisecond * 10)
}
```

---

### Query Performance

1. **Use indexes for better query performance**
2. **Use `Explain()` to verify index usage**
3. **Limit result sets with `Limit` and `Skip`**
4. **Use projections to reduce data transfer**
5. **Put `$match` early in aggregation pipelines**

```go
// Good: Check execution plan
plan := users.Explain(filter)
if plan["indexName"] == nil {
    log.Println("Warning: Query not using index")
}

// Good: Paginate results
options := &database.QueryOptions{
    Limit: 20,
    Skip: page * 20,
}
```

---

## Examples

### Complete CRUD Application

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/mnohosten/laura-db/pkg/database"
    "github.com/mnohosten/laura-db/pkg/document"
)

func main() {
    // Open database
    config := database.DefaultConfig("./data")
    db, err := database.Open(config)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Get collection
    users := db.Collection("users")

    // Create index
    users.CreateIndex("email", true)

    // Insert
    id, err := users.InsertOne(map[string]interface{}{
        "name": "Alice",
        "email": "alice@example.com",
        "age": int64(30),
        "createdAt": time.Now(),
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Inserted ID: %s\n", id)

    // Find
    doc, err := users.FindOne(map[string]interface{}{
        "email": "alice@example.com",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found: %+v\n", doc)

    // Update
    err = users.UpdateOne(
        map[string]interface{}{"_id": id},
        map[string]interface{}{
            "$set": map[string]interface{}{"age": int64(31)},
            "$currentDate": map[string]interface{}{"lastModified": true},
        },
    )
    if err != nil {
        log.Fatal(err)
    }

    // Delete
    err = users.DeleteOne(map[string]interface{}{"_id": id})
    if err != nil {
        log.Fatal(err)
    }
}
```

---

### Transaction Example

```go
// Transfer money between accounts
err := db.WithTransaction(func(session *database.Session) error {
    // Debit from account
    err := session.UpdateOne("accounts",
        map[string]interface{}{"_id": fromID},
        map[string]interface{}{
            "$inc": map[string]interface{}{"balance": int64(-amount)},
        },
    )
    if err != nil {
        return err
    }

    // Credit to account
    err = session.UpdateOne("accounts",
        map[string]interface{}{"_id": toID},
        map[string]interface{}{
            "$inc": map[string]interface{}{"balance": int64(amount)},
        },
    )
    return err
})

if err != nil {
    log.Printf("Transaction failed: %v", err)
}
```

---

### Aggregation Example

```go
// Calculate sales by category
pipeline := []map[string]interface{}{
    // Filter recent sales
    {"$match": map[string]interface{}{
        "date": map[string]interface{}{
            "$gte": time.Now().AddDate(0, -1, 0),
        },
    }},

    // Group by category
    {"$group": map[string]interface{}{
        "_id": "$category",
        "totalSales": map[string]interface{}{"$sum": "$amount"},
        "avgSale": map[string]interface{}{"$avg": "$amount"},
        "orderCount": map[string]interface{}{"$sum": int64(1)},
    }},

    // Sort by total sales
    {"$sort": map[string]interface{}{
        "totalSales": int64(-1),
    }},

    // Top 10
    {"$limit": int64(10)},
}

results, err := orders.Aggregate(pipeline)
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("Category: %v, Total: %v, Avg: %v, Orders: %v\n",
        result.Get("_id"),
        result.Get("totalSales"),
        result.Get("avgSale"),
        result.Get("orderCount"),
    )
}
```

---

## Performance Characteristics

LauraDB uses disk-based storage with multiple layers of caching for optimal performance.

### Storage Architecture

- **Disk Persistence**: All documents stored on disk using slotted page structure (4KB pages)
- **Write-Ahead Log (WAL)**: Sequential writes ensure durability before applying changes
- **Buffer Pool**: LRU cache for frequently accessed pages (configurable via `BufferPoolSize`)
- **Document Cache**: Per-collection LRU cache for frequently accessed documents
- **Query Cache**: Per-collection cache for query results (5-minute TTL)

### Expected Performance

Performance varies based on caching efficiency and workload:

| Operation | Cached | Disk (Cold) | Notes |
|-----------|--------|-------------|-------|
| InsertOne | ~100-200µs | ~500-1000µs | Includes WAL write |
| FindOne by _id | ~50-100µs | ~500-1000µs | With document cache |
| Find (indexed) | ~100-200µs | ~1-2ms | Per matching document |
| Find (full scan) | ~200-500µs | ~2-5ms | Per page scanned |
| UpdateOne | ~150-300µs | ~1-2ms | Includes index updates |
| DeleteOne | ~100-200µs | ~1-2ms | Includes index updates |
| Index lookup | ~50-100µs | ~500-1000µs | B+ tree traversal |
| Range scan | ~200-500µs | ~1-5ms | Per page, benefits from prefetching |

**Notes:**
- "Cached" means data is in buffer pool or document cache
- "Disk (Cold)" means data must be read from disk
- Actual performance depends on: hardware (SSD vs HDD), workload patterns, cache hit rates, and document sizes
- SSDs provide 10-100x faster random access than HDDs
- Sequential operations (WAL writes, range scans) perform well on both SSDs and HDDs

### Performance Tuning Tips

1. **Buffer Pool Sizing**
   ```go
   config := database.DefaultConfig("./data")
   config.BufferPoolSize = 5000  // Increase for larger datasets (20MB cache)
   ```

2. **Use Indexes for Queries**
   ```go
   // Create index on frequently queried fields
   coll.CreateIndex("email", true)  // unique index
   coll.CreateIndex("age", false)   // non-unique index
   ```

3. **Batch Operations**
   ```go
   // Better: Insert multiple documents in one call
   coll.InsertMany(docs)

   // Avoid: Multiple individual inserts
   for _, doc := range docs {
       coll.InsertOne(doc)  // Each insert has overhead
   }
   ```

4. **Use Covered Queries**
   - Queries that can be satisfied entirely from index data (no document fetch)
   - Create compound indexes matching your query patterns
   - Use projections to request only indexed fields

5. **Limit Result Sets**
   ```go
   options := &database.QueryOptions{
       Limit: 100,  // Prevent loading too many documents
   }
   results, _ := coll.Find(filter, options)
   ```

6. **Monitor Query Performance**
   ```go
   // Use Explain to verify index usage
   plan := coll.Explain(filter)
   if plan["indexName"] == nil {
       log.Println("Warning: Query not using index - consider creating one")
   }
   ```

### Scaling Considerations

- **Dataset Size**: LauraDB can handle datasets larger than available RAM through disk storage and caching
- **Memory Usage**: `(BufferPoolSize × 4KB) + (per-collection document cache) + (query cache)`
- **Disk Space**: Approximately 1.2-1.5x your actual data size (includes indexes, WAL, and overhead)
- **Concurrency**: MVCC provides non-blocking reads; multiple readers never block writers

---

## Additional Resources

- [Getting Started Guide](getting-started.md)
- [Storage Engine Documentation](storage-engine.md)
- [MVCC Transactions](mvcc.md)
- [Query Optimization](statistics-optimization.md)
- [Indexing Guide](indexing.md)
- [HTTP API Documentation](http-api.md)
- [Performance Tuning Guide](performance-tuning.md)

---

## Version

This documentation is for LauraDB v0.1.0.

Last updated: 2025-01-15 (disk storage implementation)
