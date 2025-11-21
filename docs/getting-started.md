# Getting Started

## Installation

```bash
# Clone or create your project
go mod init your-project

# Import the database
import "github.com/krizos/document-database/pkg/database"
```

## Quick Start

### 1. Open a Database

```go
package main

import (
    "log"
    "github.com/krizos/document-database/pkg/database"
)

func main() {
    // Open database (creates if doesn't exist)
    config := database.DefaultConfig("./mydata")
    db, err := database.Open(config)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Database is ready!
}
```

### 2. Insert Documents

```go
// Get or create collection
users := db.Collection("users")

// Insert one document
id, err := users.InsertOne(map[string]interface{}{
    "name":  "Alice",
    "email": "alice@example.com",
    "age":   int64(30),
    "tags":  []interface{}{"admin", "developer"},
})

// Insert many documents
ids, err := users.InsertMany([]map[string]interface{}{
    {"name": "Bob", "age": int64(25)},
    {"name": "Charlie", "age": int64(35)},
})
```

### 3. Query Documents

```go
// Find all documents
allUsers, _ := users.Find(map[string]interface{}{})

// Find with filter
adults, _ := users.Find(map[string]interface{}{
    "age": map[string]interface{}{"$gte": int64(18)},
})

// Find one document
alice, _ := users.FindOne(map[string]interface{}{
    "name": "Alice",
})

// Complex query
results, _ := users.Find(map[string]interface{}{
    "$or": []interface{}{
        map[string]interface{}{"age": map[string]interface{}{"$lt": int64(25)}},
        map[string]interface{}{"tags": map[string]interface{}{"$in": []interface{}{"admin"}}},
    },
})
```

### 4. Update Documents

```go
// Update one
err = users.UpdateOne(
    map[string]interface{}{"name": "Alice"},
    map[string]interface{}{
        "$set": map[string]interface{}{
            "age":  int64(31),
            "city": "New York",
        },
    },
)

// Update many
count, err := users.UpdateMany(
    map[string]interface{}{"age": map[string]interface{}{"$lt": int64(30)}},
    map[string]interface{}{
        "$inc": map[string]interface{}{
            "age": int64(1),
        },
    },
)

// Unset fields
users.UpdateOne(
    map[string]interface{}{"name": "Bob"},
    map[string]interface{}{
        "$unset": map[string]interface{}{
            "temporaryField": true,
        },
    },
)
```

### 5. Delete Documents

```go
// Delete one
err = users.DeleteOne(map[string]interface{}{
    "name": "Bob",
})

// Delete many
count, err := users.DeleteMany(map[string]interface{}{
    "age": map[string]interface{}{"$lt": int64(18)},
})
```

### 6. Create Indexes

```go
// Create unique index on email
err = users.CreateIndex("email", true)

// Create non-unique index on city
err = users.CreateIndex("city", false)

// List indexes
indexes := users.ListIndexes()
for _, idx := range indexes {
    fmt.Printf("Index: %s on %s\n", idx["name"], idx["field_path"])
}
```

### 7. Aggregation Pipeline

```go
// Group and analyze data
results, _ := users.Aggregate([]map[string]interface{}{
    {
        "$match": map[string]interface{}{
            "age": map[string]interface{}{"$gte": int64(18)},
        },
    },
    {
        "$group": map[string]interface{}{
            "_id": "$city",
            "avgAge": map[string]interface{}{
                "$avg": "$age",
            },
            "count": map[string]interface{}{
                "$count": nil,
            },
        },
    },
    {
        "$sort": map[string]interface{}{
            "count": -1,
        },
    },
})

// Print results
for _, doc := range results {
    city, _ := doc.Get("_id")
    avgAge, _ := doc.Get("avgAge")
    count, _ := doc.Get("count")
    fmt.Printf("%s: %v users, avg age %.1f\n", city, count, avgAge)
}
```

## Common Patterns

### Pagination

```go
func GetPage(coll *database.Collection, page, pageSize int) ([]*document.Document, error) {
    return coll.FindWithOptions(
        map[string]interface{}{},
        &database.QueryOptions{
            Sort: []query.SortField{
                {Field: "_id", Ascending: true},
            },
            Skip:  page * pageSize,
            Limit: pageSize,
        },
    )
}

// Usage
page1, _ := GetPage(users, 0, 10)  // First 10
page2, _ := GetPage(users, 1, 10)  // Next 10
```

### Search with Projections

```go
// Return only specific fields
results, _ := users.FindWithOptions(
    map[string]interface{}{
        "city": "New York",
    },
    &database.QueryOptions{
        Projection: map[string]bool{
            "name":  true,
            "email": true,
            "_id":   false,
        },
    },
)
```

### Top N Query

```go
// Top 10 highest rated products
topProducts, _ := products.FindWithOptions(
    map[string]interface{}{},
    &database.QueryOptions{
        Sort: []query.SortField{
            {Field: "rating", Ascending: false},
        },
        Limit: 10,
    },
)
```

### Existence Check

```go
// Find documents with optional field
withPhone, _ := users.Find(map[string]interface{}{
    "phone": map[string]interface{}{"$exists": true},
})

// Find documents missing field
withoutPhone, _ := users.Find(map[string]interface{}{
    "phone": map[string]interface{}{"$exists": false},
})
```

### Regex Search

```go
// Find Gmail users
gmailUsers, _ := users.Find(map[string]interface{}{
    "email": map[string]interface{}{
        "$regex": ".*@gmail\\.com$",
    },
})

// Find names starting with 'A'
aNames, _ := users.Find(map[string]interface{}{
    "name": map[string]interface{}{
        "$regex": "^A",
    },
})
```

### Count Documents

```go
// Count all
total, _ := users.Count(map[string]interface{}{})

// Count matching filter
adults, _ := users.Count(map[string]interface{}{
    "age": map[string]interface{}{"$gte": int64(18)},
})
```

## Configuration

### Database Configuration

```go
config := &database.Config{
    DataDir:        "./data",
    BufferPoolSize: 1000,  // Number of pages to cache
}
db, _ := database.Open(config)
```

### Index Configuration

```go
// Default B-tree index (order 32)
users.CreateIndex("field", false)

// Custom index (future - not yet exposed)
// idx := index.NewIndex(&index.IndexConfig{
//     Name:      "custom_idx",
//     FieldPath: "field",
//     Type:      index.IndexTypeBTree,
//     Unique:    false,
//     Order:     64,  // Larger order = shallower tree
// })
```

## Error Handling

```go
// Document not found
doc, err := users.FindOne(filter)
if err == database.ErrDocumentNotFound {
    fmt.Println("No matching document")
}

// Duplicate key (unique index violation)
_, err = users.InsertOne(map[string]interface{}{
    "email": "duplicate@example.com",
})
if err != nil {
    fmt.Printf("Insert failed: %v\n", err)
}

// Database closed
err = db.Close()
if err != nil {
    fmt.Printf("Close failed: %v\n", err)
}
```

## Data Types

### Supported Types

```go
doc := map[string]interface{}{
    "string":    "hello",
    "int64":     int64(42),
    "int32":     int32(42),
    "float64":   3.14,
    "bool":      true,
    "null":      nil,
    "array":     []interface{}{1, 2, 3},
    "nested":    map[string]interface{}{"key": "value"},
    "binary":    []byte{0x01, 0x02},
    "objectid":  document.NewObjectID(),
}
```

### Type Conversions

```go
// Always use int64 for integers
age := int64(30)  // ✓ Correct
age := 30         // ✗ May cause issues

// Arrays must be []interface{}
tags := []interface{}{"a", "b", "c"}  // ✓ Correct
tags := []string{"a", "b", "c"}       // ✗ Won't work
```

## Collections

### Create Collection

```go
// Implicitly created on first use
users := db.Collection("users")

// Explicitly create
users, err := db.CreateCollection("users")
```

### Drop Collection

```go
err := db.DropCollection("users")
```

### List Collections

```go
collections := db.ListCollections()
for _, name := range collections {
    fmt.Println(name)
}
```

## Statistics

### Database Stats

```go
stats := db.Stats()
fmt.Printf("Collections: %v\n", stats["collections"])
fmt.Printf("Active transactions: %v\n", stats["active_transactions"])
fmt.Printf("Storage stats: %v\n", stats["storage_stats"])
```

### Collection Stats

```go
stats := users.Stats()
fmt.Printf("Documents: %v\n", stats["count"])
fmt.Printf("Indexes: %v\n", stats["indexes"])
```

## Complete Example

```go
package main

import (
    "fmt"
    "log"
    "github.com/krizos/document-database/pkg/database"
)

func main() {
    // Open database
    db, err := database.Open(database.DefaultConfig("./data"))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Get collection
    products := db.Collection("products")

    // Insert products
    products.InsertMany([]map[string]interface{}{
        {"name": "Laptop", "price": 999.99, "category": "Electronics"},
        {"name": "Mouse", "price": 29.99, "category": "Electronics"},
        {"name": "Desk", "price": 399.99, "category": "Furniture"},
    })

    // Create index
    products.CreateIndex("category", false)

    // Query
    electronics, _ := products.Find(map[string]interface{}{
        "category": "Electronics",
        "price":    map[string]interface{}{"$lt": 1000.0},
    })

    fmt.Printf("Found %d electronics\n", len(electronics))

    // Aggregate
    results, _ := products.Aggregate([]map[string]interface{}{
        {
            "$group": map[string]interface{}{
                "_id": "$category",
                "avgPrice": map[string]interface{}{
                    "$avg": "$price",
                },
                "count": map[string]interface{}{
                    "$count": nil,
                },
            },
        },
    })

    fmt.Println("Category summary:")
    for _, doc := range results {
        category, _ := doc.Get("_id")
        avgPrice, _ := doc.Get("avgPrice")
        count, _ := doc.Get("count")
        fmt.Printf("  %s: %v items, avg $%.2f\n", category, count, avgPrice)
    }
}
```

## Next Steps

- Read [Architecture](../README.md) for system overview
- Explore [Query Engine](query-engine.md) for advanced queries
- Learn [Aggregation](aggregation.md) for data analysis
- Understand [Indexing](indexing.md) for performance
- Study [Storage Engine](storage-engine.md) for internals

## Examples

Check the `examples/` directory for more demos:
- `examples/basic/` - Basic operations
- `examples/full_demo/` - Comprehensive demo
- `examples/aggregation_demo/` - Aggregation examples

Run examples:
```bash
cd examples/full_demo
go run main.go
```

## Troubleshooting

### Database Won't Open

```go
db, err := database.Open(config)
if err != nil {
    // Check:
    // - Directory permissions
    // - Disk space
    // - Existing locks
    log.Printf("Failed to open database: %v", err)
}
```

### Slow Queries

1. Create indexes on queried fields
2. Use projections to reduce data
3. Add limits to cap results
4. Filter early in pipelines

### Memory Usage

Adjust buffer pool size:
```go
config := &database.Config{
    DataDir:        "./data",
    BufferPoolSize: 500,  // Reduce if memory constrained
}
```

## Best Practices

1. **Always close database**: Use `defer db.Close()`
2. **Use int64 for integers**: Avoid plain `int`
3. **Create indexes**: For frequently-queried fields
4. **Use projections**: Return only needed fields
5. **Add limits**: Cap result sizes
6. **Handle errors**: Check all error returns
7. **Batch inserts**: Use `InsertMany` for multiple documents
8. **Filter early**: In queries and pipelines

## Performance Tips

1. Create indexes on filter fields
2. Use projections to reduce data transfer
3. Add limits to queries
4. Place $match early in pipelines
5. Use $project to remove unneeded fields
6. Batch operations when possible
7. Monitor buffer pool hit rate

## Getting Help

- **Documentation**: See `/docs` directory
- **Examples**: See `/examples` directory
- **Issues**: Educational project - experiment and learn!

Happy coding! 🚀
