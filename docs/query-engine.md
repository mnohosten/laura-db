# Query Engine

## Overview

The query engine evaluates MongoDB-like queries against documents, supporting comparison operators, logical operators, projections, sorting, and pagination.

## Query Structure

A query consists of:
- **Filter**: Which documents to match
- **Projection**: Which fields to return
- **Sort**: How to order results
- **Skip/Limit**: Pagination controls

```go
results, _ := collection.FindWithOptions(
    // Filter
    map[string]interface{}{
        "age": map[string]interface{}{"$gte": 18},
        "city": "New York",
    },
    // Options
    &QueryOptions{
        Projection: map[string]bool{"name": true, "email": true},
        Sort: []SortField{{Field: "age", Ascending: false}},
        Limit: 10,
        Skip: 0,
    },
)
```

## Operators

### Comparison Operators

#### $eq - Equal

```go
// Explicit
{"age": {"$eq": 30}}

// Implicit (shorthand)
{"age": 30}
```

Matches documents where field equals value.

#### $ne - Not Equal

```go
{"status": {"$ne": "deleted"}}
```

Matches documents where field doesn't equal value.

#### $gt - Greater Than

```go
{"age": {"$gt": 18}}
```

Matches documents where field > value.

#### $gte - Greater Than or Equal

```go
{"price": {"$gte": 100}}
```

Matches documents where field >= value.

#### $lt - Less Than

```go
{"stock": {"$lt": 10}}
```

Matches documents where field < value.

#### $lte - Less Than or Equal

```go
{"age": {"$lte": 65}}
```

Matches documents where field <= value.

#### $in - In Array

```go
{"status": {"$in": []interface{}{"active", "pending"}}}
```

Matches documents where field value is in array.

#### $nin - Not In Array

```go
{"role": {"$nin": []interface{}{"admin", "superuser"}}}
```

Matches documents where field value is NOT in array.

### Logical Operators

#### $and - Logical AND

```go
{
    "$and": []interface{}{
        map[string]interface{}{"age": map[string]interface{}{"$gte": 18}},
        map[string]interface{}{"city": "New York"},
    },
}
```

All conditions must be true.

**Note**: Implicit AND for multiple fields:
```go
// These are equivalent:
{"age": 30, "city": "New York"}

{"$and": []interface{}{
    map[string]interface{}{"age": 30},
    map[string]interface{}{"city": "New York"},
}}
```

#### $or - Logical OR

```go
{
    "$or": []interface{}{
        map[string]interface{}{"age": map[string]interface{}{"$lt": 18}},
        map[string]interface{}{"age": map[string]interface{}{"$gt": 65}},
    },
}
```

At least one condition must be true.

### Element Operators

#### $exists - Field Exists

```go
{"phone": {"$exists": true}}   // Has phone field
{"email": {"$exists": false}}  // Doesn't have email field
```

Checks if field exists in document.

### Evaluation Operators

#### $regex - Regular Expression

```go
{"name": {"$regex": "^A.*"}}  // Names starting with 'A'
{"email": {"$regex": ".*@gmail\\.com$"}}  // Gmail addresses
```

Matches field against regex pattern.

#### $size - Array Size

```go
{"tags": {"$size": 3}}  // Arrays with exactly 3 elements
```

Matches arrays with specific length.

## Query Execution

### Execution Flow

```
Query → Filter → Sort → Skip → Limit → Project → Results
```

1. **Filter**: Match documents against query criteria
2. **Sort**: Order results by specified fields
3. **Skip**: Skip first N results (pagination)
4. **Limit**: Return at most N results
5. **Project**: Select/exclude fields

### Example Execution

```go
// Query: Find active users in NY, sorted by age, page 2
filter := map[string]interface{}{
    "status": "active",
    "city": "New York",
}
options := &QueryOptions{
    Sort: []SortField{{Field: "age", Ascending: true}},
    Skip: 10,   // Skip first 10 (page 1)
    Limit: 10,  // Return next 10 (page 2)
    Projection: map[string]bool{"name": true, "age": true},
}
results, _ := coll.FindWithOptions(filter, options)

// Execution:
// 1. Scan all documents
// 2. Filter: status="active" AND city="New York" → 50 matches
// 3. Sort: Order by age → [18, 19, 20, ..., 65]
// 4. Skip: Skip first 10 → Start at 11th
// 5. Limit: Return 10 → Documents 11-20
// 6. Project: Return only {name, age} fields
```

## Operator Evaluation

### Numeric Comparisons

Supports type coercion across numeric types:

```go
// All these match if field value is 42:
{"count": 42}              // int
{"count": int64(42)}       // int64
{"count": float64(42.0)}   // float64
{"count": int32(42)}       // int32
```

### String Comparisons

Lexicographic ordering:

```go
{"name": {"$gt": "Alice"}}
// Matches: "Bob", "Charlie", "Diana"
// Doesn't match: "Alice", "Aaron"
```

### Type Handling

Comparisons only work between compatible types:
- Numbers compare across int/float types
- Strings compare with strings
- Different types don't compare (treated as not equal)

## Projections

Control which fields are returned.

### Inclusion Projection

Include only specified fields:

```go
projection := map[string]bool{
    "name": true,
    "email": true,
}
```

### Exclusion Projection

Include all fields except specified:

```go
projection := map[string]bool{
    "password": false,
    "ssn": false,
}
```

**Note**: Can't mix inclusion and exclusion (except for _id).

### _id Field

The _id field is always included by default unless explicitly excluded:

```go
// Include _id
projection := map[string]bool{"name": true}

// Exclude _id
projection := map[string]bool{"name": true, "_id": false}
```

## Sorting

Order results by one or more fields.

### Single Field Sort

```go
// Ascending
sort := []SortField{{Field: "age", Ascending: true}}

// Descending
sort := []SortField{{Field: "price", Ascending: false}}
```

### Multi-Field Sort

```go
sort := []SortField{
    {Field: "city", Ascending: true},    // First by city
    {Field: "age", Ascending: false},    // Then by age (desc)
}
```

Sort order: First field is primary, second field breaks ties, etc.

### Missing Fields

Documents missing sort fields are placed at the end:

```go
// Sort by age
// Document A: {age: 30}
// Document B: {age: 25}
// Document C: {} (no age)
//
// Result: B, A, C
```

## Pagination

Implement pagination using skip and limit.

### Basic Pagination

```go
pageSize := 10
pageNum := 2  // 0-indexed

results, _ := coll.FindWithOptions(
    filter,
    &QueryOptions{
        Skip: pageNum * pageSize,  // Skip 20
        Limit: pageSize,            // Return 10
    },
)
```

### Pagination Pattern

```go
func GetPage(coll *Collection, filter map[string]interface{}, page, pageSize int) ([]*document.Document, error) {
    return coll.FindWithOptions(filter, &QueryOptions{
        Skip: page * pageSize,
        Limit: pageSize,
        Sort: []SortField{{Field: "_id", Ascending: true}}, // Stable ordering
    })
}

// Usage
page1, _ := GetPage(coll, filter, 0, 10)  // First 10
page2, _ := GetPage(coll, filter, 1, 10)  // Next 10
```

**Best Practice**: Always sort when paginating for consistent results.

## Complex Query Examples

### Example 1: Range Query

Find products priced between $10 and $100:

```go
results, _ := coll.Find(map[string]interface{}{
    "price": map[string]interface{}{
        "$gte": 10.0,
        "$lte": 100.0,
    },
})
```

### Example 2: Multiple Conditions

Find adults in New York or Boston:

```go
results, _ := coll.Find(map[string]interface{}{
    "age": map[string]interface{}{"$gte": 18},
    "$or": []interface{}{
        map[string]interface{}{"city": "New York"},
        map[string]interface{}{"city": "Boston"},
    },
})
```

### Example 3: Regex Search

Find users with Gmail addresses:

```go
results, _ := coll.Find(map[string]interface{}{
    "email": map[string]interface{}{
        "$regex": ".*@gmail\\.com$",
    },
})
```

### Example 4: Array Membership

Find documents with specific tags:

```go
results, _ := coll.Find(map[string]interface{}{
    "tags": map[string]interface{}{
        "$in": []interface{}{"featured", "popular"},
    },
})
```

### Example 5: Existence Check

Find documents with optional field:

```go
// Has phone number
hasPhone, _ := coll.Find(map[string]interface{}{
    "phone": map[string]interface{}{"$exists": true},
})

// Missing phone number
noPhone, _ := coll.Find(map[string]interface{}{
    "phone": map[string]interface{}{"$exists": false},
})
```

### Example 6: Complex Filter with Projection

Find expensive electronics, return name and price only:

```go
results, _ := coll.FindWithOptions(
    map[string]interface{}{
        "category": "Electronics",
        "price": map[string]interface{}{"$gte": 500.0},
        "inStock": true,
    },
    &QueryOptions{
        Projection: map[string]bool{
            "name": true,
            "price": true,
            "_id": false,
        },
        Sort: []SortField{{Field: "price", Ascending: false}},
        Limit: 5,
    },
)
```

## Performance Considerations

### Query Planning with Statistics

The query planner uses index statistics to choose the most efficient execution strategy:

1. **Statistics Collection**: Indexes track cardinality, selectivity, and value distribution
2. **Cost Estimation**: Each potential index is assigned an estimated query cost
3. **Plan Selection**: The planner chooses the index with the lowest estimated cost
4. **Execution**: Query executes using the selected index (or collection scan if no suitable index)

**Time complexity**:
- With index: O(log n + k) where k = matching documents
- Without index: O(n) where n = total documents

### Statistics-Based Optimization

```go
// Create indexes
coll.CreateIndex("email", true)   // High cardinality
coll.CreateIndex("status", false) // Low cardinality

// Analyze to collect statistics
coll.Analyze()

// Query that could use either index
results, _ := coll.Find(map[string]interface{}{
    "email": "alice@example.com",
    "status": "active",
})

// Planner automatically chooses email index
// (higher cardinality = more selective = lower cost)
```

**Explain Query Plans:**

```go
plan := planner.Plan(query)
explanation := plan.Explain()

fmt.Printf("Using Index: %v\n", explanation["useIndex"])
fmt.Printf("Index Name: %v\n", explanation["indexName"])
fmt.Printf("Scan Type: %v\n", explanation["scanType"])
fmt.Printf("Estimated Cost: %v\n", explanation["estimatedCost"])
fmt.Printf("Is Covered: %v\n", explanation["isCovered"])
```

### Optimization Tips

**Best practices**:
- Run `Analyze()` periodically to keep statistics fresh
- Create indexes on high-cardinality fields (many unique values)
- Use projections to reduce data transfer
- Use limits to cap result size
- Check query plans with `Explain()` to verify index usage

**Future optimizations**:
- Index intersection (use multiple indexes)
- Query plan caching
- Histogram-based selectivity estimation

## Query API

### Find Methods

```go
// Find all matching documents
docs, err := coll.Find(filter)

// Find one document
doc, err := coll.FindOne(filter)

// Find with options
docs, err := coll.FindWithOptions(filter, options)

// Count matching documents
count, err := coll.Count(filter)
```

### Query Options

```go
type QueryOptions struct {
    Projection map[string]bool      // Which fields to return
    Sort       []query.SortField    // Sort order
    Limit      int                  // Max results
    Skip       int                  // Skip results (pagination)
}
```

## Error Handling

### Common Errors

```go
// Document not found
doc, err := coll.FindOne(filter)
if err == database.ErrDocumentNotFound {
    // Handle not found
}

// Invalid regex
_, err := coll.Find(map[string]interface{}{
    "name": map[string]interface{}{
        "$regex": "[invalid",  // Unclosed bracket
    },
})
// Returns regex compilation error
```

## Testing Queries

### Verify Query Results

```go
// Insert test data
coll.InsertOne(map[string]interface{}{"age": 25, "name": "Alice"})
coll.InsertOne(map[string]interface{}{"age": 30, "name": "Bob"})
coll.InsertOne(map[string]interface{}{"age": 35, "name": "Charlie"})

// Test query
results, _ := coll.Find(map[string]interface{}{
    "age": map[string]interface{}{"$gt": 27},
})

// Verify
if len(results) != 2 {
    log.Printf("Expected 2 results, got %d", len(results))
}
```

### Explain Query

```go
// Get execution stats
executor := query.NewExecutor(documents)
stats := executor.Explain(query)

fmt.Printf("Execution type: %s\n", stats["execution_type"])
fmt.Printf("Documents scanned: %d\n", stats["total_documents"])
fmt.Printf("Documents matched: %d\n", stats["matching_documents"])
```

## Comparison with MongoDB

### Supported

- ✓ Comparison operators ($eq, $ne, $gt, $gte, $lt, $lte, $in, $nin)
- ✓ Logical operators ($and, $or)
- ✓ Element operators ($exists)
- ✓ Regex matching ($regex)
- ✓ Projections
- ✓ Sorting
- ✓ Skip/Limit

### Not Yet Implemented

- Array operators ($all, $elemMatch)
- Type checking ($type)
- Nested field queries (dot notation)
- $not operator
- Array element access
- Update operators ($set, $inc, $push, etc.) - Available in UpdateOne/UpdateMany

### Differences

- Simpler type coercion
- No query optimization (yet)
- No index integration in query execution (yet)
- No geospatial queries
- No text search

## Best Practices

### 1. Use Specific Queries

```go
// Bad: Too broad
coll.Find(map[string]interface{}{})

// Good: Specific filter
coll.Find(map[string]interface{}{
    "status": "active",
    "category": "electronics",
})
```

### 2. Use Projections

```go
// Bad: Return all fields
coll.Find(filter)

// Good: Return only needed fields
coll.FindWithOptions(filter, &QueryOptions{
    Projection: map[string]bool{"name": true, "price": true},
})
```

### 3. Use Limits

```go
// Bad: Return all matches
coll.Find(filter)

// Good: Cap result size
coll.FindWithOptions(filter, &QueryOptions{Limit: 100})
```

### 4. Sort for Pagination

```go
// Bad: Inconsistent ordering
coll.FindWithOptions(filter, &QueryOptions{Skip: 10, Limit: 10})

// Good: Stable ordering
coll.FindWithOptions(filter, &QueryOptions{
    Sort: []SortField{{Field: "_id", Ascending: true}},
    Skip: 10,
    Limit: 10,
})
```

## Future Enhancements

1. **Nested field queries**: Support dot notation (user.address.city)
2. **More array operators**: $all (already have $elemMatch)
3. **Query plan caching**: Reuse execution plans for identical queries
4. **Parallel query execution**: Multi-threaded filtering for large collections
5. **Query hints**: Let users specify which index to use
6. **Histogram-based selectivity**: More accurate cardinality estimation for range queries

## Summary

The query engine provides:
- **MongoDB-like syntax**: Familiar query language
- **Rich operators**: Comparison, logical, element, evaluation
- **Flexible projections**: Control returned fields
- **Sorting and pagination**: Order and paginate results
- **Type coercion**: Works across numeric types

The query engine is the primary interface for retrieving data, making it essential to understand for effective database use.
