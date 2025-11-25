# Migrating from MongoDB to LauraDB

This guide helps MongoDB users transition to LauraDB by highlighting similarities, differences, and providing practical migration examples.

## Table of Contents

- [Overview](#overview)
- [Key Similarities](#key-similarities)
- [Key Differences](#key-differences)
- [Syntax Comparison](#syntax-comparison)
- [Migration Strategies](#migration-strategies)
- [Code Migration Examples](#code-migration-examples)
- [Feature Mapping](#feature-mapping)
- [Performance Considerations](#performance-considerations)
- [Common Pitfalls](#common-pitfalls)
- [Migration Checklist](#migration-checklist)

## Overview

LauraDB is designed as an educational MongoDB-like document database with similar concepts and APIs. If you're familiar with MongoDB, you'll feel at home with LauraDB. However, there are some important differences to be aware of.

### Why Migrate to LauraDB?

- **Embedded Mode**: Use LauraDB as a library directly in your Go application (no separate server process)
- **Educational Focus**: Understand database internals with clean, documented code
- **Lightweight**: Minimal dependencies, small footprint
- **MVCC Transactions**: Snapshot isolation for consistent reads without blocking
- **Go-Native**: Built in Go, for Go applications

### When to Use LauraDB vs MongoDB

**Use LauraDB when:**
- Building Go applications that need an embedded database
- Learning database internals and architecture
- Prototyping document-based applications
- You need a lightweight alternative to MongoDB
- Single-node deployment is sufficient

**Use MongoDB when:**
- You need production-grade distributed systems
- Horizontal scaling and sharding are required
- You need advanced features like replica sets, GridFS, or transactions across shards
- Multi-language driver support is essential
- You require enterprise features and support

## Key Similarities

### Document Model
Both MongoDB and LauraDB use:
- JSON-like BSON documents
- Dynamic schemas with flexible field types
- Nested documents and arrays
- ObjectID for unique identifiers

### Query Language
LauraDB supports most MongoDB query operators:
- Comparison: `$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`, `$in`, `$nin`
- Logical: `$and`, `$or`, `$not`
- Element: `$exists`, `$type`
- Array: `$all`, `$elemMatch`, `$size`
- Evaluation: `$regex`

### Indexing
Both support:
- Single-field indexes
- Compound indexes (multiple fields)
- Unique indexes
- Text indexes (full-text search)
- Geospatial indexes (2d, 2dsphere)
- TTL indexes (automatic expiration)
- Partial indexes (filtered)

### Aggregation
Both provide pipeline-based aggregation with stages:
- `$match`, `$group`, `$project`, `$sort`, `$limit`, `$skip`
- Aggregation operators: `$sum`, `$avg`, `$min`, `$max`, `$count`, `$push`

## Key Differences

### 1. Deployment Model

**MongoDB:**
```bash
# Separate server process
mongod --dbpath /data/db --port 27017

# Connect with client
mongo mongodb://localhost:27017
```

**LauraDB:**
```go
// Embedded in your application
db, err := database.Open(database.DefaultConfig("./data"))
defer db.Close()

// Or use HTTP server mode
./laura-server -port 8080 -data-dir ./data
```

### 2. Language Integration

**MongoDB:**
- Separate server process
- Connect via drivers (Go, Python, Java, Node.js, etc.)
- Network communication overhead

**LauraDB:**
- Native Go library (embedded mode)
- Direct function calls (no network overhead)
- HTTP server mode available for multi-language access

### 3. Storage Engine

**MongoDB:**
- WiredTiger storage engine (production default)
- Supports compression, encryption at rest
- Memory-mapped files or B-trees

**LauraDB:**
- Custom page-based storage with Write-Ahead Log (WAL)
- In-memory document storage with persistent WAL
- Buffer pool with LRU eviction
- Optional memory-mapped files for performance

### 4. Transaction Model

**MongoDB:**
- ACID transactions (multi-document since 4.0)
- Replica set or sharded cluster required for multi-document transactions

**LauraDB:**
- MVCC (Multi-Version Concurrency Control) with snapshot isolation
- Single-node ACID transactions
- Non-blocking reads (readers don't block writers)

### 5. Type System

**MongoDB:**
```javascript
// Untyped integers default to appropriate type
db.collection.insertOne({ age: 30 })
```

**LauraDB:**
```go
// Must explicitly use int64 for consistency
collection.InsertOne(map[string]interface{}{
    "age": int64(30),  // Required
})
```

**Important:** LauraDB requires explicit `int64` for numeric values due to Go's type system and BSON encoding requirements.

### 6. Supported Features

| Feature | MongoDB | LauraDB |
|---------|---------|---------|
| Document CRUD | ✅ | ✅ |
| Query Operators | ✅ Full | ✅ Most common |
| Indexes (B+tree, text, geo) | ✅ | ✅ |
| Aggregation Pipeline | ✅ Full | ✅ Core stages |
| ACID Transactions | ✅ Multi-doc | ✅ Single-node |
| Replication | ✅ | ❌ |
| Sharding | ✅ | ❌ |
| GridFS | ✅ | ❌ |
| Change Streams | ✅ | ✅ (via pkg/changestream) |
| Authentication | ✅ | ✅ (via pkg/auth) |
| TLS/SSL | ✅ | ✅ (via docs/tls-ssl.md) |
| Horizontal Scaling | ✅ | ❌ |

## Syntax Comparison

### Insert Operations

**MongoDB (JavaScript):**
```javascript
db.users.insertOne({
  name: "Alice",
  email: "alice@example.com",
  age: 30
})

db.users.insertMany([
  { name: "Bob", age: 25 },
  { name: "Charlie", age: 35 }
])
```

**LauraDB (Go):**
```go
users := db.Collection("users")

users.InsertOne(map[string]interface{}{
    "name":  "Alice",
    "email": "alice@example.com",
    "age":   int64(30),  // Note: int64 required
})

users.InsertMany([]map[string]interface{}{
    {"name": "Bob", "age": int64(25)},
    {"name": "Charlie", "age": int64(35)},
})
```

### Find Operations

**MongoDB:**
```javascript
// Find all
db.users.find({})

// Find with filter
db.users.find({ age: { $gte: 18 } })

// Find one
db.users.findOne({ name: "Alice" })

// Find with projection and sort
db.users.find(
  { age: { $gte: 18 } },
  { name: 1, email: 1 }
).sort({ age: -1 }).limit(10)
```

**LauraDB:**
```go
// Find all
users.Find(map[string]interface{}{})

// Find with filter
users.Find(map[string]interface{}{
    "age": map[string]interface{}{"$gte": int64(18)},
})

// Find one
users.FindOne(map[string]interface{}{
    "name": "Alice",
})

// Find with options
users.FindWithOptions(
    map[string]interface{}{
        "age": map[string]interface{}{"$gte": int64(18)},
    },
    &database.FindOptions{
        Projection: map[string]int{"name": 1, "email": 1},
        Sort:       map[string]int{"age": -1},
        Limit:      10,
    },
)
```

### Update Operations

**MongoDB:**
```javascript
// Update one
db.users.updateOne(
  { name: "Alice" },
  { $set: { age: 31 }, $push: { tags: "verified" } }
)

// Update many
db.users.updateMany(
  { age: { $lt: 18 } },
  { $set: { status: "minor" } }
)
```

**LauraDB:**
```go
// Update one
users.UpdateOne(
    map[string]interface{}{"name": "Alice"},
    map[string]interface{}{
        "$set":  map[string]interface{}{"age": int64(31)},
        "$push": map[string]interface{}{"tags": "verified"},
    },
)

// Update many
users.UpdateMany(
    map[string]interface{}{
        "age": map[string]interface{}{"$lt": int64(18)},
    },
    map[string]interface{}{
        "$set": map[string]interface{}{"status": "minor"},
    },
)
```

### Delete Operations

**MongoDB:**
```javascript
// Delete one
db.users.deleteOne({ name: "Alice" })

// Delete many
db.users.deleteMany({ age: { $lt: 18 } })
```

**LauraDB:**
```go
// Delete one
users.DeleteOne(map[string]interface{}{"name": "Alice"})

// Delete many
users.DeleteMany(map[string]interface{}{
    "age": map[string]interface{}{"$lt": int64(18)},
})
```

### Index Management

**MongoDB:**
```javascript
// Create single-field index
db.users.createIndex({ email: 1 }, { unique: true })

// Create compound index
db.users.createIndex({ city: 1, age: 1 })

// Create text index
db.users.createIndex({ bio: "text" })

// List indexes
db.users.getIndexes()

// Drop index
db.users.dropIndex("email_1")
```

**LauraDB:**
```go
// Create single-field index
users.CreateIndex("email", true)  // unique: true

// Create compound index
users.CreateCompoundIndex([]string{"city", "age"}, false)

// Create text index
users.CreateTextIndex([]string{"bio"}, "english")

// List indexes
indexes := users.ListIndexes()

// Drop index
users.DropIndex("email")
```

### Aggregation Pipeline

**MongoDB:**
```javascript
db.orders.aggregate([
  { $match: { status: "completed" } },
  { $group: {
      _id: "$customerId",
      total: { $sum: "$amount" },
      count: { $count: {} }
  }},
  { $sort: { total: -1 } },
  { $limit: 10 }
])
```

**LauraDB:**
```go
orders := db.Collection("orders")

pipeline := []map[string]interface{}{
    {"$match": map[string]interface{}{"status": "completed"}},
    {"$group": map[string]interface{}{
        "_id":   "$customerId",
        "total": map[string]interface{}{"$sum": "$amount"},
        "count": map[string]interface{}{"$count": map[string]interface{}{}},
    }},
    {"$sort": map[string]interface{}{"total": int64(-1)}},
    {"$limit": int64(10)},
}

results, _ := orders.Aggregate(pipeline)
```

## Migration Strategies

### Strategy 1: Embedded Mode Migration (Recommended for Go Apps)

Best for Go applications that can embed LauraDB directly.

**Before (MongoDB):**
```go
// MongoDB client
client, _ := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
database := client.Database("myapp")
users := database.Collection("users")

// Insert
users.InsertOne(ctx, bson.M{"name": "Alice"})
```

**After (LauraDB):**
```go
// LauraDB embedded
db, _ := database.Open(database.DefaultConfig("./data"))
defer db.Close()
users := db.Collection("users")

// Insert (no context needed for single-node)
users.InsertOne(map[string]interface{}{"name": "Alice"})
```

### Strategy 2: HTTP Server Mode Migration

Best for polyglot environments or when you can't embed the database.

**Step 1: Start LauraDB HTTP Server**
```bash
./laura-server -port 8080 -data-dir ./data
```

**Step 2: Use REST API**
```bash
# Insert document
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "age": 30}'

# Find documents
curl "http://localhost:8080/api/users?filter={\"age\":{\"$gte\":18}}"

# Update document
curl -X PUT http://localhost:8080/api/users/<id> \
  -H "Content-Type: application/json" \
  -d '{"$set": {"age": 31}}'
```

### Strategy 3: Gradual Migration

Migrate one collection at a time while running both databases in parallel.

**Phase 1: Dual Write**
```go
// Write to both MongoDB and LauraDB
func createUser(user User) error {
    // Write to MongoDB
    _, err1 := mongoUsers.InsertOne(ctx, user)

    // Write to LauraDB
    _, err2 := lauraUsers.InsertOne(userToMap(user))

    if err1 != nil || err2 != nil {
        return errors.New("dual write failed")
    }
    return nil
}
```

**Phase 2: Read from LauraDB, Fallback to MongoDB**
```go
func getUser(id string) (*User, error) {
    // Try LauraDB first
    doc, err := lauraUsers.FindOne(map[string]interface{}{"_id": id})
    if err == nil {
        return mapToUser(doc), nil
    }

    // Fallback to MongoDB
    var user User
    err = mongoUsers.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
    return &user, err
}
```

**Phase 3: Full Cutover**
```go
// Remove MongoDB code entirely
func getUser(id string) (*User, error) {
    doc, err := lauraUsers.FindOne(map[string]interface{}{"_id": id})
    if err != nil {
        return nil, err
    }
    return mapToUser(doc), nil
}
```

## Code Migration Examples

### Example 1: User Management Service

**MongoDB Version:**
```go
package main

import (
    "context"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/bson"
)

type UserService struct {
    collection *mongo.Collection
}

func NewUserService() (*UserService, error) {
    client, err := mongo.Connect(context.TODO(),
        options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        return nil, err
    }

    return &UserService{
        collection: client.Database("myapp").Collection("users"),
    }, nil
}

func (s *UserService) CreateUser(name, email string, age int) error {
    _, err := s.collection.InsertOne(context.TODO(), bson.M{
        "name":  name,
        "email": email,
        "age":   age,
    })
    return err
}

func (s *UserService) FindUsersByAge(minAge int) ([]bson.M, error) {
    cursor, err := s.collection.Find(context.TODO(), bson.M{
        "age": bson.M{"$gte": minAge},
    })
    if err != nil {
        return nil, err
    }

    var results []bson.M
    if err = cursor.All(context.TODO(), &results); err != nil {
        return nil, err
    }
    return results, nil
}
```

**LauraDB Version:**
```go
package main

import (
    "github.com/mnohosten/laura-db/pkg/database"
)

type UserService struct {
    collection *database.Collection
}

func NewUserService() (*UserService, error) {
    db, err := database.Open(database.DefaultConfig("./data"))
    if err != nil {
        return nil, err
    }

    return &UserService{
        collection: db.Collection("users"),
    }, nil
}

func (s *UserService) CreateUser(name, email string, age int) error {
    _, err := s.collection.InsertOne(map[string]interface{}{
        "name":  name,
        "email": email,
        "age":   int64(age),  // Convert to int64
    })
    return err
}

func (s *UserService) FindUsersByAge(minAge int) ([]map[string]interface{}, error) {
    return s.collection.Find(map[string]interface{}{
        "age": map[string]interface{}{"$gte": int64(minAge)},
    })
}
```

**Key Changes:**
1. Replace `mongo.Connect()` with `database.Open()`
2. Remove `context.TODO()` calls (not needed for single-node)
3. Replace `bson.M` with `map[string]interface{}`
4. Convert integer values to `int64`

### Example 2: E-commerce Order Processing

**MongoDB Version:**
```go
func ProcessOrder(orderID string) error {
    // Start transaction
    session, err := client.StartSession()
    if err != nil {
        return err
    }
    defer session.EndSession(context.TODO())

    callback := func(sc mongo.SessionContext) (interface{}, error) {
        // Decrement inventory
        _, err := inventory.UpdateOne(sc,
            bson.M{"productID": orderID},
            bson.M{"$inc": bson.M{"quantity": -1}},
        )
        if err != nil {
            return nil, err
        }

        // Update order status
        _, err = orders.UpdateOne(sc,
            bson.M{"_id": orderID},
            bson.M{"$set": bson.M{"status": "processed"}},
        )
        return nil, err
    }

    _, err = session.WithTransaction(context.TODO(), callback)
    return err
}
```

**LauraDB Version:**
```go
func ProcessOrder(db *database.Database, orderID string) error {
    // Start transaction
    txn := db.BeginTransaction()
    defer func() {
        if txn.IsActive() {
            txn.Abort()
        }
    }()

    inventory := db.Collection("inventory")
    orders := db.Collection("orders")

    // Decrement inventory
    _, err := inventory.UpdateOne(
        map[string]interface{}{"productID": orderID},
        map[string]interface{}{
            "$inc": map[string]interface{}{"quantity": int64(-1)},
        },
    )
    if err != nil {
        return err
    }

    // Update order status
    _, err = orders.UpdateOne(
        map[string]interface{}{"_id": orderID},
        map[string]interface{}{
            "$set": map[string]interface{}{"status": "processed"},
        },
    )
    if err != nil {
        return err
    }

    return txn.Commit()
}
```

**Key Changes:**
1. Replace `session.WithTransaction()` with `db.BeginTransaction()`
2. Use defer for automatic abort on error
3. Explicit `Commit()` call
4. Simpler API without session context

## Feature Mapping

### Query Operators

| MongoDB | LauraDB | Status | Notes |
|---------|---------|--------|-------|
| `$eq` | `$eq` | ✅ | Equal |
| `$ne` | `$ne` | ✅ | Not equal |
| `$gt` | `$gt` | ✅ | Greater than |
| `$gte` | `$gte` | ✅ | Greater than or equal |
| `$lt` | `$lt` | ✅ | Less than |
| `$lte` | `$lte` | ✅ | Less than or equal |
| `$in` | `$in` | ✅ | In array |
| `$nin` | `$nin` | ✅ | Not in array |
| `$and` | `$and` | ✅ | Logical AND |
| `$or` | `$or` | ✅ | Logical OR |
| `$not` | `$not` | ✅ | Logical NOT |
| `$nor` | - | ❌ | Not supported |
| `$exists` | `$exists` | ✅ | Field exists |
| `$type` | `$type` | ✅ | BSON type check |
| `$regex` | `$regex` | ✅ | Pattern matching |
| `$all` | `$all` | ✅ | Array contains all |
| `$elemMatch` | `$elemMatch` | ✅ | Array element match |
| `$size` | `$size` | ✅ | Array size |
| `$mod` | - | ❌ | Use application logic |
| `$where` | - | ❌ | Not supported |

### Update Operators

| MongoDB | LauraDB | Status |
|---------|---------|--------|
| `$set` | `$set` | ✅ |
| `$unset` | `$unset` | ✅ |
| `$inc` | `$inc` | ✅ |
| `$mul` | `$mul` | ✅ |
| `$min` | `$min` | ✅ |
| `$max` | `$max` | ✅ |
| `$push` | `$push` | ✅ |
| `$pull` | `$pull` | ✅ |
| `$pullAll` | `$pullAll` | ✅ |
| `$pop` | `$pop` | ✅ |
| `$addToSet` | `$addToSet` | ✅ |
| `$rename` | `$rename` | ✅ |
| `$currentDate` | `$currentDate` | ✅ |
| `$bit` | `$bit` | ✅ |

### Aggregation Stages

| MongoDB | LauraDB | Status |
|---------|---------|--------|
| `$match` | `$match` | ✅ |
| `$group` | `$group` | ✅ |
| `$project` | `$project` | ✅ |
| `$sort` | `$sort` | ✅ |
| `$limit` | `$limit` | ✅ |
| `$skip` | `$skip` | ✅ |
| `$unwind` | - | ❌ |
| `$lookup` | - | ❌ |
| `$graphLookup` | - | ❌ |
| `$facet` | - | ❌ |

## Performance Considerations

### LauraDB Advantages

1. **No Network Overhead (Embedded Mode)**
   - Direct function calls vs network round trips
   - 50-100x faster for simple operations
   - Zero serialization/deserialization overhead

2. **Efficient Memory Usage**
   - In-memory documents with persistent WAL
   - No separate server process memory overhead
   - Configurable buffer pool size

3. **Query Cache**
   - Built-in LRU cache for frequent queries
   - 96x faster for cached queries
   - Automatic invalidation on writes

4. **Covered Queries**
   - 2.2x faster when query satisfied by index alone
   - No document fetch required
   - Automatic detection and optimization

### MongoDB Advantages

1. **Horizontal Scaling**
   - Sharding for large datasets (100+ GB)
   - Replica sets for high availability
   - Read replicas for scaling reads

2. **Production Features**
   - Compression algorithms (Snappy, Zstd)
   - Advanced caching (WiredTiger cache)
   - Memory-mapped file optimizations

3. **Mature Ecosystem**
   - Extensive tooling (Compass, Ops Manager)
   - Cloud hosting (Atlas)
   - Enterprise support

### Migration Performance Tips

1. **Index Strategy**
   ```go
   // Create indexes BEFORE migrating large datasets
   collection.CreateIndex("email", true)
   collection.CreateCompoundIndex([]string{"city", "age"}, false)

   // Then bulk insert
   collection.InsertMany(documents)
   ```

2. **Batch Operations**
   ```go
   // Batch inserts are more efficient
   batch := make([]map[string]interface{}, 0, 1000)
   for _, doc := range allDocs {
       batch = append(batch, doc)
       if len(batch) >= 1000 {
           collection.InsertMany(batch)
           batch = batch[:0]
       }
   }
   ```

3. **Use Projections**
   ```go
   // Only fetch fields you need
   users.FindWithOptions(
       filter,
       &database.FindOptions{
           Projection: map[string]int{"name": 1, "email": 1},
       },
   )
   ```

4. **Parallel Query Execution**
   ```go
   // Enable for large datasets (automatic threshold detection)
   // Benefits kick in at 1000+ documents
   results, _ := collection.Find(filter)
   ```

## Common Pitfalls

### 1. Type Mismatches

**Problem:**
```go
// This will panic with type assertion error
collection.InsertOne(map[string]interface{}{
    "age": 30,  // Wrong: untyped int
})

doc, _ := collection.FindOne(map[string]interface{}{"_id": id})
age := doc["age"].(int64)  // Panic: interface {} is int, not int64
```

**Solution:**
```go
// Always use int64 for numbers
collection.InsertOne(map[string]interface{}{
    "age": int64(30),  // Correct
})

doc, _ := collection.FindOne(map[string]interface{}{"_id": id})
age := doc["age"].(int64)  // OK
```

### 2. Missing Context Removal

**Problem:**
```go
// MongoDB code with contexts
users.InsertOne(ctx, document)
users.Find(ctx, filter)
```

**Solution:**
```go
// LauraDB doesn't need context for single-node operations
users.InsertOne(document)
users.Find(filter)
```

### 3. BSON vs map[string]interface{}

**Problem:**
```go
// MongoDB uses bson.M
filter := bson.M{"age": bson.M{"$gte": 18}}
```

**Solution:**
```go
// LauraDB uses native Go maps
filter := map[string]interface{}{
    "age": map[string]interface{}{"$gte": int64(18)},
}
```

### 4. Transaction API Differences

**Problem:**
```go
// MongoDB session-based transactions
session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
    // operations
})
```

**Solution:**
```go
// LauraDB transaction API
txn := db.BeginTransaction()
defer func() {
    if txn.IsActive() {
        txn.Abort()
    }
}()

// operations

txn.Commit()
```

### 5. Connection String Migration

**Problem:**
```go
// MongoDB connection string
"mongodb://user:pass@host:27017/database?replicaSet=rs0"
```

**Solution:**
```go
// LauraDB embedded mode (no connection string)
db, _ := database.Open(database.DefaultConfig("./data"))

// Or HTTP mode
"http://localhost:8080"
```

## Migration Checklist

### Pre-Migration

- [ ] **Analyze Current Usage**
  - Identify all collections and schemas
  - Document query patterns and indexes
  - Note any MongoDB-specific features used
  - Measure current performance metrics

- [ ] **Verify Feature Compatibility**
  - Check query operators against feature mapping table
  - Identify unsupported features ($nor, $where, $lookup, etc.)
  - Plan workarounds for missing features

- [ ] **Test Environment Setup**
  - Install LauraDB in test environment
  - Create sample data migration script
  - Run performance benchmarks

### During Migration

- [ ] **Data Migration**
  - Export MongoDB data (use mongoexport or custom scripts)
  - Convert BSON types to Go maps with int64 for numbers
  - Import data using InsertMany in batches
  - Verify document counts match

- [ ] **Index Migration**
  - Recreate all indexes in LauraDB
  - Test query performance with indexes
  - Use background index building for large collections

- [ ] **Code Migration**
  - Replace MongoDB driver imports
  - Update connection/initialization code
  - Convert bson.M to map[string]interface{}
  - Add int64 type conversions
  - Update transaction code
  - Remove context parameters

- [ ] **Testing**
  - Run unit tests against LauraDB
  - Perform integration testing
  - Load testing with production-like data
  - Compare performance metrics

### Post-Migration

- [ ] **Monitoring**
  - Set up performance monitoring
  - Track query performance
  - Monitor memory usage
  - Check disk usage and WAL growth

- [ ] **Optimization**
  - Tune buffer pool size
  - Optimize index strategy
  - Enable query cache
  - Use covered queries where possible

- [ ] **Documentation**
  - Update team documentation
  - Document any workarounds for missing features
  - Share performance tuning configurations

## Additional Resources

### LauraDB Documentation

- [Getting Started Guide](getting-started.md)
- [API Reference](api-reference.md)
- [Query Engine](query-engine.md)
- [Indexing Guide](indexing.md)
- [Aggregation Pipeline](aggregation.md)
- [Performance Tuning](performance-tuning.md)
- [Architecture Overview](architecture.md)

### Tools

- [Import/Export Tools](import-export.md)
- [Migration Tools](migration-tools.md)
- [Backup and Restore](docs/backup-restore.md)

### Client Libraries

- [Go Client](go-client.md)
- [Python Client](python-client.md)
- [Node.js Client](nodejs-client.md)
- [Java Client](java-client.md)

### Support

- GitHub Issues: [https://github.com/mnohosten/laura-db/issues](https://github.com/mnohosten/laura-db/issues)
- Documentation: [https://github.com/mnohosten/laura-db/tree/main/docs](https://github.com/mnohosten/laura-db/tree/main/docs)

## Conclusion

Migrating from MongoDB to LauraDB is straightforward for most use cases, especially for Go applications that can benefit from embedded mode. The similar document model and query language make the transition smooth, while key differences (type system, transaction API) are well-documented and easy to handle.

For applications that don't require horizontal scaling, replication, or MongoDB-specific advanced features, LauraDB provides a lightweight, efficient, and easy-to-understand alternative with excellent performance characteristics.

Remember:
- Always use `int64` for numeric values
- Take advantage of embedded mode for Go applications
- Migrate indexes before bulk data import
- Test thoroughly in a staging environment
- Monitor performance and optimize as needed

Happy migrating! 🚀
