# Index Intersection

Index intersection is a query optimization technique that uses multiple indexes together to efficiently answer multi-field queries. Instead of using a single index and filtering the rest, the query planner can use multiple indexes simultaneously and intersect their results.

## Overview

When a query has multiple filter conditions, LauraDB's query planner can choose to:
1. **Collection Scan**: Scan all documents and filter (slowest)
2. **Single Index**: Use one index and filter remaining conditions
3. **Index Intersection**: Use multiple indexes and intersect the results (often fastest)

## How It Works

### Example Query

```go
query := map[string]interface{}{
    "age":  int64(25),
    "city": "NYC",
}
```

With separate indexes on `age` and `city`:

1. **Index Scan 1**: Use age index to find all documents with `age=25` → Set A
2. **Index Scan 2**: Use city index to find all documents with `city="NYC"` → Set B
3. **Intersection**: Find common document IDs: `Result = A ∩ B`
4. **Document Fetch**: Retrieve only the documents in the intersection

### Algorithm

The intersection uses an efficient set-based algorithm:

```
1. Execute each index scan independently
2. Collect document IDs from each index into sets
3. Start with the smallest set (most selective)
4. For each ID in the smallest set:
   - Check if it exists in all other sets
   - If yes, add to result
5. Fetch documents for result IDs
```

## When to Use

Index intersection is most beneficial when:

- ✅ Query has **2 or more equality/range conditions**
- ✅ Each condition has a **separate single-field index**
- ✅ Both/all conditions are **highly selective** (filter out most documents)
- ✅ Dataset is **large** (>1000 documents typically)

Index intersection may NOT be chosen when:

- ❌ Only one condition has an index
- ❌ Compound index exists covering all fields (compound is preferred)
- ❌ Dataset is very small (single index + filter is cheaper)
- ❌ One index is much more selective than others (use that one index)

## Cost-Based Selection

The query planner uses statistics to estimate costs:

```go
// Cost for index intersection
intersectionCost = sum(indexScanCosts) + intersectionOverhead

// Compare with alternatives:
// - Single index cost: ~5-50 (depends on selectivity)
// - Collection scan cost: ~1,000,000 (proportional to collection size)
```

The planner automatically chooses the lowest-cost option.

## Examples

### Two-Field Query

```go
// Create indexes
collection.CreateIndex("age_idx", "age", false)
collection.CreateIndex("city_idx", "city", false)

// Query with both conditions
results, err := collection.Find(map[string]interface{}{
    "age":  int64(25),
    "city": "NYC",
})

// Planner may use index intersection automatically
```

### Three-Field Query

```go
// Create three indexes
collection.CreateIndex("age_idx", "age", false)
collection.CreateIndex("city_idx", "city", false)
collection.CreateIndex("status_idx", "status", false)

// Query with three conditions
results, err := collection.Find(map[string]interface{}{
    "age":    int64(25),
    "city":   "NYC",
    "status": "active",
})

// All three indexes can be intersected
```

### Range Queries

Index intersection also works with range operators:

```go
// Indexes on age and salary
collection.CreateIndex("age_idx", "age", false)
collection.CreateIndex("salary_idx", "salary", false)

// Range query on both fields
results, err := collection.Find(map[string]interface{}{
    "age":    map[string]interface{}{"$gte": int64(30)},
    "salary": map[string]interface{}{"$lte": int64(70000)},
})

// Uses range scans on both indexes + intersection
```

## Query Explanation

Use the `Explain()` method to see if intersection was used:

```go
planner := query.NewQueryPlanner(indexes)
plan := planner.Plan(query)
explanation := plan.Explain()

fmt.Printf("%+v\n", explanation)
```

Output with intersection:
```json
{
  "scanType": "INDEX_INTERSECTION",
  "indexes": ["age_idx", "city_idx"],
  "fields": ["age", "city"],
  "estimatedCost": 15,
  "note": "Using multiple indexes with set intersection"
}
```

Output with single index:
```json
{
  "scanType": "INDEX_EXACT",
  "indexName": "age_idx",
  "indexedField": "age",
  "estimatedCost": 10,
  "additionalFilters": ["city"]
}
```

## Performance Characteristics

### Time Complexity

- **Index scans**: O(log n) per index (B+ tree lookup)
- **Set intersection**: O(min(|A|, |B|, ...)) where A, B are result sets
- **Document fetch**: O(|result|)

**Overall**: O(k·log n + |result|) where k = number of indexes

### Space Complexity

- **Temporary sets**: O(k·|avg_results|) where k = number of indexes
- **Final result**: O(|result|)

### Performance Benchmarks

Based on 10,000 documents:

| Method | Time | Speedup |
|--------|------|---------|
| Collection Scan | ~500 µs | 1x (baseline) |
| Single Index | ~127 ns | 3,937x |
| Index Intersection | ~127 ns | 3,937x |

Note: Intersection has similar performance to single index for small result sets, but scales better with multiple selective conditions.

### Selectivity Impact

High selectivity (100 unique values per field):
- Index intersection: ~107 ns/op
- Highly efficient, minimal set intersection overhead

Low selectivity (5 unique values per field):
- Index intersection: ~470 ns/op
- More overhead due to larger intermediate sets
- Still much faster than collection scan

## Implementation Details

### Query Plan Structure

```go
type QueryPlan struct {
    UseIntersection bool                  // True if using intersection
    IntersectPlans  []*IndexIntersectPlan // Plans for each index
    // ... other fields
}

type IndexIntersectPlan struct {
    IndexName string
    Index     *index.Index
    Field     string
    ScanType  ScanType    // Exact or Range
    ScanKey   interface{} // For exact match
    ScanStart interface{} // For range scans
    ScanEnd   interface{} // For range scans
}
```

### Execution Flow

1. **Planning** (in `planner.go`):
   - `planIndexIntersection()` - Finds usable indexes for each field
   - `createIntersectPlan()` - Creates scan plan for each index
   - `estimateIntersectionCost()` - Computes cost estimate
   - Compare with single index and collection scan costs

2. **Execution** (in `executor.go`):
   - `executeIndexIntersection()` - Orchestrates intersection
   - `executeIntersectIndexScan()` - Scans individual indexes
   - `intersectSets()` - Performs set intersection
   - Fetch documents for final result IDs

### Set Intersection Optimization

The `intersectSets()` function optimizes by:
1. Starting with the smallest set (most selective index)
2. Checking membership in other sets using hash maps (O(1) lookup)
3. Early termination if any ID is missing from any set

```go
// Start with smallest set for efficiency
for id := range smallestSet {
    inAll := true
    for otherSet := range otherSets {
        if !otherSet[id] {
            inAll = false
            break // Early exit
        }
    }
    if inAll {
        result[id] = true
    }
}
```

## Limitations

Current limitations:

1. **Compound indexes not used**: Intersection currently uses only single-field indexes
   - Compound indexes are preferred when available
   - Future: Support compound index + single-field index intersection

2. **Equality and ranges only**: Supports `$eq`, `$gt`, `$gte`, `$lt`, `$lte`
   - Not yet supported: `$in`, `$nin`, `$regex`, `$exists`
   - Future: Extend to more operators

3. **Top-level fields only**: Doesn't handle `$and`/`$or` operators yet
   - Extracts simple field conditions from filter
   - Future: Support complex logical expressions

4. **Statistics required**: Relies on index statistics for cost estimation
   - Call `collection.AnalyzeIndexes()` to update statistics
   - Stale stats may lead to suboptimal plans

## Best Practices

### 1. Create Appropriate Indexes

```go
// Create single-field indexes for commonly queried fields
collection.CreateIndex("age_idx", "age", false)
collection.CreateIndex("city_idx", "city", false)
collection.CreateIndex("status_idx", "status", false)
```

### 2. Keep Statistics Updated

```go
// Analyze indexes periodically (especially after bulk inserts)
for _, indexName := range collection.ListIndexes() {
    idx := collection.GetIndex(indexName)
    idx.Analyze()
}
```

### 3. Use Compound Indexes When Appropriate

```go
// If you ALWAYS query age+city together, use a compound index
collection.CreateCompoundIndex("age_city_idx", []string{"age", "city"}, false)

// Compound index is better than intersection for this specific query
```

### 4. Monitor Query Plans

```go
// Check if intersection is being used
plan := planner.Plan(query)
if plan.UseIntersection {
    fmt.Printf("Using %d indexes\n", len(plan.IntersectPlans))
} else {
    fmt.Printf("Using plan type: %v\n", plan.ScanType)
}
```

### 5. Profile with Benchmarks

```go
// Benchmark your specific queries
func BenchmarkMyQuery(b *testing.B) {
    for i := 0; i < b.N; i++ {
        collection.Find(myFilter)
    }
}
```

## Comparison with Other Databases

| Database | Index Intersection | Notes |
|----------|-------------------|-------|
| **LauraDB** | ✅ Automatic | Cost-based optimizer |
| **MongoDB** | ✅ Automatic | Since version 2.6 |
| **PostgreSQL** | ✅ Automatic | Bitmap index scans |
| **MySQL** | ✅ Manual/Automatic | index_merge optimization |
| **SQLite** | ❌ No | Uses single index only |

## Future Enhancements

Planned improvements:

1. **Compound + single-field intersection**
   - Intersect compound index with single-field indexes
   - Example: `{city,age}` compound + `status` single-field

2. **More operators**
   - Support `$in` (multi-value exact match)
   - Support `$nin` (exclusion)
   - Support `$regex` (text pattern matching)

3. **Parallel intersection**
   - Execute index scans concurrently
   - Parallel set intersection for large sets

4. **Index union**
   - Use `$or` with multiple indexes
   - Union results instead of intersection

5. **Adaptive statistics**
   - Auto-update statistics based on query patterns
   - Machine learning for cost estimation

## References

- [MongoDB Index Intersection](https://www.mongodb.com/docs/manual/core/index-intersection/)
- [PostgreSQL Bitmap Scans](https://www.postgresql.org/docs/current/indexes-bitmap-scans.html)
- [MySQL Index Merge](https://dev.mysql.com/doc/refman/8.0/en/index-merge-optimization.html)

## Related Documentation

- [Query Engine](query-engine.md) - Overall query execution architecture
- [Indexing](indexing.md) - Index types and creation
- [Statistics and Optimization](statistics-optimization.md) - Cost-based query planning
- [Covered Queries](covered-queries.md) - Index-only query execution
