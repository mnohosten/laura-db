# Phase 6: Query Engine

**Status**: ✅ Complete
**Duration**: Major development phase
**Completion**: 100%

## Overview

Phase 6 implemented a comprehensive query engine with support for MongoDB-like query operators, query planning, index optimization, and execution. This phase transforms the database from a simple key-value store into a powerful document query system.

## Goals

- Implement MongoDB-compatible query operators
- Build query parser and validator
- Create query executor with filtering logic
- Implement projection, sorting, and pagination
- Add query planner for index optimization
- Provide query explain functionality

## Architecture

```
Query Flow:
┌─────────────┐
│   Query     │  (MongoDB-like filter)
│   Filter    │  {"age": {"$gte": 18}, "city": "NYC"}
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Query     │  Parse and validate
│   Parser    │  Create Query object
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Query     │  Analyze filter, select best plan
│   Planner   │  Decide: index scan vs collection scan
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Query     │  Execute plan
│   Executor  │  Apply filters, sort, project
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Results   │  Matching documents
└─────────────┘
```

## Implementation

### 1. Query Structure

```go
type Query struct {
    filter     map[string]interface{}
    projection map[string]bool
    sort       []SortField
    limit      int
    skip       int
}
```

### 2. Query Operators

#### Comparison Operators
- `$eq`: Equal to
- `$ne`: Not equal to
- `$gt`: Greater than
- `$gte`: Greater than or equal
- `$lt`: Less than
- `$lte`: Less than or equal

#### Logical Operators
- `$and`: Logical AND of conditions
- `$or`: Logical OR of conditions
- `$not`: Logical negation

#### Array Operators
- `$in`: Value in array
- `$nin`: Value not in array
- `$all`: Array contains all values

#### Element Operators
- `$exists`: Field exists check
- `$type`: Type checking

### 3. Query Planner (New!)

The query planner analyzes queries and generates optimal execution plans:

```go
type QueryPlan struct {
    UseIndex      bool
    IndexName     string
    Index         *index.Index
    ScanType      ScanType        // Collection, IndexExact, IndexRange
    ScanKey       interface{}     // For exact scans
    ScanStart     interface{}     // For range scans
    ScanEnd       interface{}     // For range scans
    EstimatedCost int
    FilterSteps   []string
}
```

**Cost Model:**
- Exact index match: Cost = 10 (lowest)
- Range index scan: Cost = 50 (low)
- Collection scan: Cost = 1,000,000 (high)

**Index Selection Logic:**
1. Analyze all available indexes
2. Check if query filter matches indexed fields
3. Calculate cost for each possible plan
4. Select plan with lowest cost
5. Track remaining filters after index scan

### 4. Query Executor

Two execution modes:

**Without Index (Collection Scan):**
```go
for _, doc := range documents {
    if query.Matches(doc) {
        results = append(results, doc)
    }
}
```

**With Index (Index Scan):**
```go
// 1. Use index to get candidate document IDs
docIDs := plan.Index.Search(plan.ScanKey)

// 2. Retrieve documents by ID (O(1) lookup)
for _, id := range docIDs {
    doc := documentsMap[id]

    // 3. Apply remaining filters
    if query.Matches(doc) {
        results = append(results, doc)
    }
}
```

## Key Features

### 1. Operator Implementation

Example: `$gt` operator
```go
func evaluateGt(docValue, queryValue interface{}) bool {
    docNum := toFloat64(docValue)
    queryNum := toFloat64(queryValue)
    return docNum > queryNum
}
```

### 2. Compound Filters

Supports nested logical operators:
```go
{
    "$and": [
        {"age": {"$gte": 18}},
        {"$or": [
            {"city": "NYC"},
            {"city": "LA"}
        ]}
    ]
}
```

### 3. Projection

Field inclusion/exclusion:
```go
// Include specific fields
{"name": true, "age": true}

// Exclude specific fields
{"password": false, "ssn": false}
```

### 4. Sorting

Multi-field sorting:
```go
query.WithSort([]SortField{
    {Field: "age", Ascending: false},    // Primary sort
    {Field: "name", Ascending: true},    // Secondary sort
})
```

### 5. Pagination

```go
query.WithSkip(20).WithLimit(10)  // Page 3, 10 per page
```

## Query Optimization

### Index Usage Examples

**Query 1: Exact Match**
```go
// Query: {"age": 30}
// Plan: INDEX_EXACT scan on age_1
// Cost: 10
// Speedup: O(log n) vs O(n)
```

**Query 2: Range**
```go
// Query: {"age": {"$gte": 25, "$lte": 40}}
// Plan: INDEX_RANGE scan on age_1
// Cost: 50
// Speedup: O(k log n) vs O(n), where k = matching documents
```

**Query 3: Multiple Indexes**
```go
// Query: {"age": 30, "name": {"$gte": "A"}}
// Available: age_1 (exact, cost 10), name_1 (range, cost 50)
// Plan: Use age_1 (lower cost), filter name in memory
```

## Testing

### Test Coverage
- 20+ query operator tests
- 10+ query planner tests
- Integration tests with real indexes
- Edge cases (empty results, missing fields, type mismatches)

### Test Examples

```go
func TestQueryPlannerExactMatch(t *testing.T) {
    idx := createAgeIndex()
    planner := NewQueryPlanner(map[string]*index.Index{"age_1": idx})

    query := NewQuery(map[string]interface{}{"age": 30})
    plan := planner.Plan(query)

    assert.True(t, plan.UseIndex)
    assert.Equal(t, ScanTypeIndexExact, plan.ScanType)
    assert.Equal(t, 30, plan.ScanKey)
}
```

## Performance Characteristics

| Operation | Without Index | With Index | Improvement |
|-----------|---------------|------------|-------------|
| Exact match | O(n) | O(log n) | ~100-1000x |
| Range query | O(n) | O(k log n) | ~10-100x |
| Sort | O(n log n) | O(k) if indexed | ~10x |

*n = total documents, k = matching documents*

## Challenges

### Challenge 1: Type Coercion
**Problem**: MongoDB allows comparing numbers as int, float, int64

**Solution**:
```go
func toFloat64(v interface{}) (float64, bool) {
    switch val := v.(type) {
    case float64: return val, true
    case int: return float64(val), true
    case int64: return float64(val), true
    default: return 0, false
    }
}
```

### Challenge 2: Query Plan Selection
**Problem**: Multiple indexes available, which to use?

**Solution**: Cost-based optimizer
- Assign costs to different scan types
- Calculate cost for each available index
- Select minimum cost plan
- Consider operator types ($eq cheaper than $gt)

### Challenge 3: Nested Operators
**Problem**: Handling complex nested logical operators

**Solution**: Recursive evaluation
```go
func (q *Query) Matches(doc *document.Document) bool {
    return q.evaluateFilter(doc, q.filter)
}

func evaluateFilter(doc, filter) bool {
    if op, ok := filter["$and"]; ok {
        for _, subfilter := range op {
            if !evaluateFilter(doc, subfilter) {
                return false
            }
        }
        return true
    }
    // ... handle other operators
}
```

## Learning Points

### 1. Query Optimization
- Index selection is critical for performance
- Cost-based optimization outperforms heuristics
- Statistics would improve plan quality

### 2. B+ Tree Advantages
- Efficient point queries: O(log n)
- Efficient range scans: O(k log n)
- Leaf nodes form linked list for scans

### 3. Trade-offs
- Index scans faster but require index maintenance
- Multiple indexes increase write cost
- Query planner adds complexity but major performance gain

### 4. Type Systems
- Dynamic typing requires careful comparison logic
- Type coercion rules must be well-defined
- Missing fields need special handling

## Query Explain

Example explain output:
```json
{
  "collection": "users",
  "totalDocuments": 10000,
  "availableIndexes": ["_id_", "age_1", "email_1"],
  "useIndex": true,
  "indexName": "age_1",
  "scanType": "INDEX_RANGE",
  "scanStart": 25,
  "scanEnd": 40,
  "estimatedCost": 50,
  "additionalFilters": ["city"]
}
```

## Next Steps

Phase 7 built on the query engine to implement full CRUD operations with update operators.

**See**: [Phase 7: Database Operations](./phase-07-database-operations.md)

---

## Related Files

- `pkg/query/query.go` - Query structure and matching
- `pkg/query/operators.go` - Operator implementations
- `pkg/query/executor.go` - Query execution
- `pkg/query/planner.go` - Query planning and optimization
- `pkg/query/query_test.go` - Query tests
- `pkg/query/planner_test.go` - Planner tests

## Metrics

- **Query operators**: 12+
- **Lines of code**: ~1,200
- **Test cases**: 30+
- **Performance**: 100-1000x with indexes

## Educational Value

This phase teaches:
- Query language design
- Query optimization techniques
- Cost-based query planning
- Index utilization strategies
- Type system handling in dynamic languages
- Recursive algorithm design
- Performance analysis and trade-offs
