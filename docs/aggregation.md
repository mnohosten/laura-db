# Aggregation Pipeline

## Overview

The aggregation pipeline processes documents through a sequence of stages, transforming and analyzing data. Each stage performs an operation (filter, group, sort, etc.) and passes results to the next stage.

## Pipeline Concept

Think of it as a Unix pipeline for documents:

```bash
# Unix pipeline
cat data.txt | grep "active" | sort | head -10

# Aggregation pipeline
documents | $match | $sort | $limit
```

Each stage transforms the data and passes it along.

## Pipeline Structure

```go
results, _ := collection.Aggregate([]map[string]interface{}{
    {"$match": ...},     // Stage 1: Filter
    {"$group": ...},     // Stage 2: Group
    {"$sort": ...},      // Stage 3: Sort
    {"$limit": ...},     // Stage 4: Limit
})
```

**Execution**: Documents flow through stages sequentially.

## Stages

### $match - Filter Documents

Filters documents like a query.

```go
{"$match": map[string]interface{}{
    "age": map[string]interface{}{"$gte": 18},
    "status": "active",
}}
```

**Purpose**: Reduce dataset early in pipeline for efficiency.

**When to use**: Always place $match as early as possible.

### $project - Select/Transform Fields

Selects which fields to include/exclude.

```go
{"$project": map[string]interface{}{
    "name": true,
    "email": true,
    "_id": false,
}}
```

**Purpose**: Shape output documents, reduce data size.

**Field values**:
- `true` or `1`: Include field
- `false` or `0`: Exclude field

### $group - Group and Aggregate

Groups documents by an expression and computes aggregations.

```go
{"$group": map[string]interface{}{
    "_id": "$category",                         // Group by category
    "totalSales": map[string]interface{}{
        "$sum": "$price",                       // Sum all prices
    },
    "avgPrice": map[string]interface{}{
        "$avg": "$price",                       // Average price
    },
    "count": map[string]interface{}{
        "$count": nil,                          // Count documents
    },
}}
```

**Field references**: `$fieldName` refers to field in input documents.

**Aggregation operators**:
- `$sum`: Sum values
- `$avg`: Average values
- `$min`: Minimum value
- `$max`: Maximum value
- `$count`: Count documents

### $sort - Sort Documents

Sorts documents by fields.

```go
{"$sort": map[string]interface{}{
    "age": 1,        // Ascending
    "name": -1,      // Descending
}}
```

**Sort order**:
- `1`: Ascending
- `-1`: Descending

**Multi-field sort**: Earlier fields have priority.

### $limit - Limit Results

Returns only first N documents.

```go
{"$limit": 10}
```

**Purpose**: Cap result size, implement pagination.

**Best practice**: Use after $sort for "top N" queries.

### $skip - Skip Documents

Skips first N documents.

```go
{"$skip": 20}
```

**Purpose**: Implement pagination.

**Pattern**: Use $skip and $limit together:
```go
{"$skip": pageNum * pageSize},
{"$limit": pageSize},
```

## Aggregation Operators

### $sum - Sum

Sum numeric values.

```go
// Sum a field
{"$sum": "$quantity"}

// Sum constant (count)
{"$sum": 1}

// Sum expression
{"$sum": "$price"}
```

### $avg - Average

Average of numeric values.

```go
{"$avg": "$price"}
```

Returns 0 if group is empty.

### $min - Minimum

Minimum value.

```go
{"$min": "$price"}
```

Works with numbers and strings.

### $max - Maximum

Maximum value.

```go
{"$max": "$price"}
```

Works with numbers and strings.

### $count - Count

Count documents in group.

```go
{"$count": nil}
```

Equivalent to `{"$sum": 1}`.

## Examples

### Example 1: Simple Filter and Sort

Find electronics, sort by price:

```go
results, _ := coll.Aggregate([]map[string]interface{}{
    {
        "$match": map[string]interface{}{
            "category": "Electronics",
        },
    },
    {
        "$sort": map[string]interface{}{
            "price": -1,  // Highest price first
        },
    },
    {
        "$limit": 5,  // Top 5
    },
})
```

### Example 2: Group by Category

Total sales per category:

```go
results, _ := coll.Aggregate([]map[string]interface{}{
    {
        "$group": map[string]interface{}{
            "_id": "$category",
            "totalRevenue": map[string]interface{}{
                "$sum": "$price",
            },
            "productCount": map[string]interface{}{
                "$count": nil,
            },
        },
    },
    {
        "$sort": map[string]interface{}{
            "totalRevenue": -1,
        },
    },
})

// Results:
// [
//   {_id: "Electronics", totalRevenue: 5000, productCount: 15},
//   {_id: "Furniture", totalRevenue: 3000, productCount: 8},
// ]
```

### Example 3: Average by Group

Average price per region:

```go
results, _ := coll.Aggregate([]map[string]interface{}{
    {
        "$group": map[string]interface{}{
            "_id": "$region",
            "avgPrice": map[string]interface{}{
                "$avg": "$price",
            },
            "minPrice": map[string]interface{}{
                "$min": "$price",
            },
            "maxPrice": map[string]interface{}{
                "$max": "$price",
            },
        },
    },
})
```

### Example 4: Filter After Grouping

Categories with average price > $200:

```go
results, _ := coll.Aggregate([]map[string]interface{}{
    {
        "$group": map[string]interface{}{
            "_id": "$category",
            "avgPrice": map[string]interface{}{
                "$avg": "$price",
            },
        },
    },
    {
        "$match": map[string]interface{}{
            "avgPrice": map[string]interface{}{
                "$gt": 200.0,
            },
        },
    },
})
```

**Note**: $match after $group filters grouped results, not original documents.

### Example 5: Multi-Stage Pipeline

Complex analysis:

```go
results, _ := coll.Aggregate([]map[string]interface{}{
    // Stage 1: Filter recent orders
    {
        "$match": map[string]interface{}{
            "status": "completed",
            "date": map[string]interface{}{
                "$gte": "2024-01-01",
            },
        },
    },
    // Stage 2: Group by customer
    {
        "$group": map[string]interface{}{
            "_id": "$customerId",
            "totalSpent": map[string]interface{}{
                "$sum": "$amount",
            },
            "orderCount": map[string]interface{}{
                "$count": nil,
            },
        },
    },
    // Stage 3: Filter high-value customers
    {
        "$match": map[string]interface{}{
            "totalSpent": map[string]interface{}{
                "$gte": 1000.0,
            },
        },
    },
    // Stage 4: Sort by spending
    {
        "$sort": map[string]interface{}{
            "totalSpent": -1,
        },
    },
    // Stage 5: Top 10
    {
        "$limit": 10,
    },
})
```

### Example 6: Pagination with Aggregation

```go
page := 2
pageSize := 10

results, _ := coll.Aggregate([]map[string]interface{}{
    {
        "$match": map[string]interface{}{
            "status": "active",
        },
    },
    {
        "$sort": map[string]interface{}{
            "createdAt": -1,
        },
    },
    {
        "$skip": page * pageSize,
    },
    {
        "$limit": pageSize,
    },
})
```

## Pipeline Optimization

### Early Filtering

**Bad**: Filter late
```go
{{"$sort": ...}, {"$limit": ...}, {"$match": ...}}
// Sorts all documents before filtering
```

**Good**: Filter early
```go
{{"$match": ...}, {"$sort": ...}, {"$limit": ...}}
// Filters first, then sorts fewer documents
```

### Projection After Grouping

**Bad**: Project before grouping
```go
{{"$project": ...}, {"$group": ...}}
// Group might need fields you projected out
```

**Good**: Project after grouping
```go
{{"$group": ...}, {"$project": ...}}
// Shape final output
```

### Limit Early When Possible

```go
// If you only need top 10 per category
{
    {"$sort": ...},
    {"$limit": 10},  // Reduce dataset
    {"$group": ...},
}
```

## Use Cases

### Analytics

```go
// Sales summary
{"$group": {
    "_id": "$productId",
    "totalSold": {"$sum": "$quantity"},
    "revenue": {"$sum": "$price"},
}}
```

### Reporting

```go
// Monthly report
{"$group": {
    "_id": "$month",
    "users": {"$count": nil},
    "avgAge": {"$avg": "$age"},
}}
```

### Data Transformation

```go
// Reshape data
{"$project": {
    "fullName": "$name",
    "contact": "$email",
}}
```

### Top-N Queries

```go
// Top 10 products
{{"$sort": {"sales": -1}}, {"$limit": 10}}
```

### Pagination

```go
// Page 3
{{"$skip": 20}, {"$limit": 10}}
```

## Performance Considerations

### Pipeline Stages Cost

| Stage | Cost | Notes |
|-------|------|-------|
| $match | O(n) | Early $match reduces n for later stages |
| $group | O(n) | Memory intensive for large groups |
| $sort | O(n log n) | Most expensive stage |
| $project | O(n) | Cheap, reduces data size |
| $limit | O(1) | Very cheap |
| $skip | O(n) | Must process skipped documents |

### Memory Usage

Grouping stages ($group) hold data in memory:
- One entry per unique group
- Can be large if many unique values
- Consider filtering before grouping

### Optimization Tips

1. **Filter early**: Use $match as first stage
2. **Sort late**: Sort smaller datasets
3. **Project wisely**: Remove unneeded fields early
4. **Limit results**: Use $limit to cap output
5. **Index-friendly**: Use $match conditions that can use indexes (future)

## Comparison: Query vs Aggregation

### When to Use Find/Query

- Simple filtering
- No grouping needed
- Straightforward sorting
- Direct field access

```go
// Simple query is easier
coll.Find(map[string]interface{}{
    "age": map[string]interface{}{"$gte": 18},
})
```

### When to Use Aggregation

- Grouping and aggregation
- Multi-stage transformations
- Complex calculations
- Data analysis

```go
// Aggregation for analytics
coll.Aggregate([]map[string]interface{}{
    {"$group": {"_id": "$category", "total": {"$sum": "$price"}}},
})
```

## Comparison with MongoDB

### Supported

- ✓ $match
- ✓ $project
- ✓ $group
- ✓ $sort
- ✓ $limit
- ✓ $skip
- ✓ Basic aggregation operators ($sum, $avg, $min, $max, $count)

### Not Yet Implemented

- $lookup (joins)
- $unwind (array expansion)
- $addFields (computed fields)
- $facet (multiple pipelines)
- $bucket (histograms)
- $graphLookup (recursive queries)
- Expression operators ($concat, $substr, etc.)
- Date operators
- Conditional operators ($cond, $ifNull)

### Differences

- Simpler field references (no complex expressions yet)
- No index integration (yet)
- Limited aggregation operators
- No nested pipeline stages

## Error Handling

```go
results, err := coll.Aggregate(pipeline)
if err != nil {
    // Handle errors:
    // - Invalid stage type
    // - Invalid operator
    // - Missing required fields
    // - Type mismatches
    log.Printf("Aggregation error: %v", err)
}
```

## Testing Aggregations

```go
// Insert test data
coll.InsertMany([]map[string]interface{}{
    {"category": "A", "price": 10},
    {"category": "A", "price": 20},
    {"category": "B", "price": 30},
})

// Test aggregation
results, _ := coll.Aggregate([]map[string]interface{}{
    {"$group": {
        "_id": "$category",
        "total": {"$sum": "$price"},
    }},
})

// Verify
// Should return 2 groups:
// {_id: "A", total: 30}
// {_id: "B", total: 30}
```

## Best Practices

### 1. Start with $match

```go
// Good
{{"$match": {"status": "active"}}, {"$group": ...}}

// Bad
{{"$group": ...}, {"$match": {"status": "active"}}}
```

### 2. Use Meaningful Group IDs

```go
// Good: Clear grouping
{"$group": {"_id": "$category", ...}}

// Bad: Unclear
{"$group": {"_id": "$c", ...}}
```

### 3. Name Computed Fields Clearly

```go
// Good
{"totalRevenue": {"$sum": "$price"}}
{"averageAge": {"$avg": "$age"}}

// Bad
{"t": {"$sum": "$price"}}
{"x": {"$avg": "$age"}}
```

### 4. Test Stages Incrementally

```go
// Test stage by stage
// Stage 1 only
results1, _ := coll.Aggregate([]map[string]interface{}{
    {"$match": ...},
})

// Stages 1-2
results2, _ := coll.Aggregate([]map[string]interface{}{
    {"$match": ...},
    {"$group": ...},
})
```

## Future Enhancements

1. **$lookup**: Join collections
2. **$unwind**: Expand arrays
3. **$addFields**: Add computed fields
4. **Expression operators**: String manipulation, math, etc.
5. **$facet**: Multiple aggregations in one pipeline
6. **Index integration**: Use indexes in $match
7. **Pipeline optimization**: Reorder stages automatically
8. **Parallel execution**: Multi-threaded stage processing

## Summary

The aggregation pipeline provides:
- **Flexible data processing**: Multi-stage transformations
- **Powerful analytics**: Grouping and aggregation
- **Composable stages**: Build complex queries from simple parts
- **Familiar syntax**: Similar to MongoDB

Aggregation is essential for analytics, reporting, and data transformation tasks that go beyond simple queries.
