# Indexing

## Overview

Indexes dramatically speed up queries by allowing the database to locate documents without scanning the entire collection. This implementation uses B+ trees, the most common indexing structure in databases.

## B+ Tree Structure

### Why B+ Trees?

B+ trees are self-balancing trees optimized for systems that read and write large blocks of data. They're ideal for databases because:

1. **Balanced**: All leaf nodes are at the same depth → O(log n) lookups
2. **Sequential access**: Leaf nodes are linked → Efficient range scans
3. **High fanout**: Each node stores many keys → Shallow trees
4. **Disk-friendly**: Nodes map well to disk pages

### Structure

```
                    [25|50]
                   /   |   \
                  /    |    \
         [10|15|20] [30|40] [60|70|80]
            ↓         ↓         ↓
         Leaf → Leaf → Leaf (linked list)
```

**Internal Nodes**: Store keys and child pointers
**Leaf Nodes**: Store keys, values, and next-leaf pointer

## Implementation

### Node Structure

```go
type BTreeNode struct {
    isLeaf   bool
    keys     []interface{}
    values   []interface{} // Only in leaf nodes
    children []*BTreeNode  // Only in internal nodes
    next     *BTreeNode    // Only in leaf nodes
    parent   *BTreeNode
}
```

### Order

The **order** of a B+ tree determines how many keys a node can hold:
- Order = 32 means each node can store up to 32 keys
- Higher order = shallower tree, fewer disk reads
- Lower order = more splits, more overhead

**Our default**: Order 32 (good balance for in-memory operations)

## Operations

### Insert

**Algorithm**:
1. Find the appropriate leaf node
2. Insert key-value pair in sorted order
3. If node is full (keys ≥ order), split:
   - Create new node with right half of keys
   - Promote middle key to parent
   - Recursively split parent if needed
4. If root splits, create new root (tree grows)

**Time complexity**: O(log n)

**Example**:
```
Insert 15 into tree of order 3:

Before:
    [10|20|30]  (full!)

After split:
       [20]           (new root)
      /    \
  [10|15] [20|30]
```

### Search

**Algorithm**:
1. Start at root
2. Binary search keys to find child to descend
3. Repeat until reaching leaf
4. Binary search leaf keys for exact match

**Time complexity**: O(log n)

**Example**:
```
Search for 35:
    [25|50]
   /   |   \
[10] [30|40] [60]

Step 1: 35 > 25, 35 < 50 → Middle child
Step 2: Search [30|40] leaf → Not found
```

### Range Scan

**Algorithm**:
1. Find leaf containing start key
2. Follow next-leaf pointers, collecting values
3. Stop when reaching end key

**Time complexity**: O(log n + k) where k = results returned

**Example**:
```
Range scan [25, 65]:

         Root
        /    \
   [20] → [40] → [60] → [80]
           ↑      ↑      ↑
        Start   Keep   End

Scan: [40] → [60] (following next pointers)
```

### Delete

Our implementation uses simplified deletion (educational):
1. Find and remove key from leaf
2. Does NOT handle underflow or rebalancing

**Production databases** would:
- Borrow keys from siblings
- Merge nodes if too empty
- Rebalance tree

**Why simplified?** Educational clarity. Adding full rebalancing adds complexity without teaching new concepts.

## Index Types

### Unique Index

Enforces uniqueness constraint:
```go
index := NewIndex(&IndexConfig{
    Name:      "email_1",
    FieldPath: "email",
    Unique:    true,
})

// This will fail if email already exists
index.Insert("alice@example.com", docID)
```

**Use cases**:
- Primary keys (_id)
- Email addresses
- Usernames
- Any field requiring uniqueness

### Non-Unique Index

Allows duplicate keys:
```go
index := NewIndex(&IndexConfig{
    Name:      "city_1",
    FieldPath: "city",
    Unique:    false,
})

// Multiple documents can have same city
index.Insert("New York", docID1)
index.Insert("New York", docID2)
```

**Use cases**:
- Categories
- Status fields
- Any field with repeated values

## Performance Characteristics

### Time Complexity

| Operation | Average | Worst Case |
|-----------|---------|------------|
| Insert | O(log n) | O(log n) |
| Search | O(log n) | O(log n) |
| Delete | O(log n) | O(log n) |
| Range Scan | O(log n + k) | O(log n + k) |

Where n = number of keys, k = results returned

### Space Complexity

- O(n) for storing all keys
- Each node: Order × (key size + pointer size)
- Overhead: ~30% compared to unsorted array

### Tree Height

For order m and n keys:
- Height ≤ log_m(n)
- Example: 1 million keys, order 32
  - Height = log₃₂(1,000,000) ≈ 4
  - Maximum 4 disk reads to find any key!

## Usage Examples

### Creating Indexes

```go
coll := db.Collection("users")

// Create unique index on email
err := coll.CreateIndex("email", true)

// Create non-unique index on city
err = coll.CreateIndex("city", false)

// Create index on age
err = coll.CreateIndex("age", false)
```

### Automatic Index Maintenance

Indexes are automatically updated on document operations:

```go
// Insert updates all indexes
id, _ := coll.InsertOne(map[string]interface{}{
    "email": "alice@example.com",
    "city": "New York",
    "age": 30,
})
// email index: "alice@example.com" → id
// city index: "New York" → id
// age index: 30 → id

// Update maintains indexes
coll.UpdateOne(
    map[string]interface{}{"email": "alice@example.com"},
    map[string]interface{}{"$set": map[string]interface{}{"city": "Boston"}},
)
// city index updated: "New York" removed, "Boston" added

// Delete removes from indexes
coll.DeleteOne(map[string]interface{}{"email": "alice@example.com"})
// All index entries for this document removed
```

### Index Statistics

Indexes track statistics to help the query planner choose the most efficient execution strategy:

```go
// Analyze index to collect statistics
coll.Analyze()

// View index statistics
indexes := coll.ListIndexes()
for _, idx := range indexes {
    fmt.Printf("Index: %s\n", idx["name"])
    fmt.Printf("  Field: %s\n", idx["field_path"])
    fmt.Printf("  Unique: %v\n", idx["unique"])
    fmt.Printf("  Size: %v keys\n", idx["size"])
    fmt.Printf("  Height: %v levels\n", idx["height"])

    // Statistics for query optimization
    fmt.Printf("  Total Entries: %v\n", idx["total_entries"])
    fmt.Printf("  Unique Keys (Cardinality): %v\n", idx["cardinality"])
    fmt.Printf("  Selectivity: %.3f\n", idx["selectivity"])
    fmt.Printf("  Min Value: %v\n", idx["min_value"])
    fmt.Printf("  Max Value: %v\n", idx["max_value"])
    fmt.Printf("  Last Updated: %v\n", idx["last_updated"])
    fmt.Printf("  Stats Stale: %v\n", idx["is_stale"])
}
```

**Statistics Tracked:**
- **Total Entries**: Number of entries in the index
- **Cardinality**: Number of unique keys (higher = more selective)
- **Selectivity**: Ratio of unique keys to total entries (0.0-1.0)
- **Min/Max Values**: Range of indexed values
- **Stale Flag**: Whether statistics need to be recalculated

**Updating Statistics:**

Statistics are automatically marked as stale when data is modified:

```go
// Statistics are fresh after analyze
coll.Analyze()

// Inserts/deletes mark stats as stale
coll.InsertOne(doc)  // Stats now stale

// Re-analyze to refresh
coll.Analyze()  // Stats fresh again
```

## Query Optimization

### Statistics-Based Index Selection

The query planner uses index statistics to choose the most efficient execution strategy:

```go
// Create two indexes
coll.CreateIndex("city", false)   // Low cardinality (few unique cities)
coll.CreateIndex("email", true)   // High cardinality (unique emails)

// Analyze indexes to collect statistics
coll.Analyze()

// Query that could use either index
coll.Find(map[string]interface{}{
    "city": "New York",
    "email": "alice@example.com",
})

// Planner chooses email index because:
// 1. Higher cardinality (more unique values)
// 2. Lower estimated cost (fewer documents to scan)
// 3. More selective (better filtering)
```

**How It Works:**

1. **Cost Estimation**: For each usable index, estimate query cost based on:
   - Cardinality (higher = better for exact matches)
   - Scan type (exact match vs range scan)
   - Statistics freshness (stale stats use default estimates)

2. **Index Comparison**: Choose index with lowest estimated cost

3. **Fallback**: If statistics are stale, use default cost estimation

**Cost Tiers for Exact Matches:**
- Very high cardinality (>1000 unique keys): Cost = 5
- High cardinality (>100 unique keys): Cost = 8
- Medium cardinality (>10 unique keys): Cost = 12
- Low cardinality (≤10 unique keys): Cost = 20

**Range Query Costs:**
- Estimated at 30% of total entries
- Capped between 20-500

### Covered Queries

A query is "covered" if the index contains all queried fields:

```go
// Not covered (needs to fetch document)
coll.Find(map[string]interface{}{
    "email": "alice@example.com",
}).WithProjection(map[string]bool{
    "name": true,
    "age": true,
})

// Potentially covered (if we stored values in index)
coll.Find(map[string]interface{}{
    "email": "alice@example.com",
}).WithProjection(map[string]bool{
    "email": true,
})
```

## Index Design Best Practices

### When to Create Indexes

**Good candidates**:
- Fields used in WHERE clauses frequently
- Fields used for sorting
- Foreign keys
- Fields with high selectivity (many unique values)

**Poor candidates**:
- Fields with low selectivity (few unique values)
- Fields rarely queried
- Small collections (< 1000 documents)

### Index Cost

Indexes have costs:
- **Storage**: Each index adds overhead
- **Write speed**: Inserts/updates must modify indexes
- **Memory**: Indexes consume RAM

**Rule of thumb**: Don't over-index. Create indexes for common queries only.

### Compound Indexes

Compound indexes are indexes on multiple fields, allowing efficient queries that filter on multiple criteria:

```go
// Create compound index on city and age
coll.CreateCompoundIndex([]string{"city", "age"}, false)

// Efficient for queries like:
coll.Find(map[string]interface{}{
    "city": "New York",
    "age": 30,
})
```

**How Compound Indexes Work:**

Compound indexes use **composite keys** that combine multiple field values into a single sortable key. The fields are ordered, and the index is sorted lexicographically by each field in sequence.

**Example:**
```go
// Index on [city, age]
// Stores keys like:
// ("Boston", 25) → id1
// ("Boston", 30) → id2
// ("NYC", 25) → id3
// ("NYC", 30) → id4
// ("Seattle", 28) → id5
```

**Field Ordering Matters:**

The order of fields in a compound index determines which queries can use it efficiently. The index can be used for queries that:
1. Match all indexed fields (full match)
2. Match a prefix of indexed fields (prefix match)

```go
// Index: [city, age, salary]

// ✓ Uses index (full match)
Find({"city": "NYC", "age": 30, "salary": 100000})

// ✓ Uses index (2-field prefix)
Find({"city": "NYC", "age": 30})

// ✓ Uses index (1-field prefix)
Find({"city": "NYC"})

// ✗ Cannot use index (doesn't start with city)
Find({"age": 30})

// ✗ Cannot use index (skips city)
Find({"age": 30, "salary": 100000})
```

**Prefix Matching:**

When a query matches only the first N fields of a compound index, the database uses **prefix matching** to scan all entries with that prefix:

```go
// Index has: (NYC, 25), (NYC, 30), (NYC, 35), (Boston, 25)

// Query: Find({"city": "NYC"})
// Scans all entries where city = "NYC"
// Returns 3 documents
```

Implementation: The query executor performs a range scan over the entire index and filters results using the `MatchesPrefix()` method on composite keys.

**Creating Compound Indexes:**

```go
coll := db.Collection("employees")

// 2-field compound index
err := coll.CreateCompoundIndex([]string{"department", "level"}, false)

// 3-field compound index
err = coll.CreateCompoundIndex([]string{"department", "level", "salary"}, false)

// Unique compound index
err = coll.CreateCompoundIndex([]string{"email", "username"}, true)
```

**Unique Compound Indexes:**

Unique compound indexes enforce uniqueness on the combination of fields:

```go
coll.CreateCompoundIndex([]string{"email", "username"}, true)

// ✓ Allowed (different combination)
coll.InsertOne({"email": "alice@example.com", "username": "alice123"})
coll.InsertOne({"email": "alice@example.com", "username": "alice456"})

// ✗ Fails (duplicate combination)
coll.InsertOne({"email": "alice@example.com", "username": "alice123"})
```

**Automatic Maintenance:**

Like single-field indexes, compound indexes are automatically maintained during updates:

```go
// Insert adds to compound index
coll.InsertOne({"city": "NYC", "age": 25})

// Update removes old composite key, adds new one
coll.UpdateOne(
    {"city": "NYC"},
    {"$set": {"age": 26}},
)
// Index updated: (NYC, 25) → (NYC, 26)

// Delete removes from compound index
coll.DeleteOne({"city": "NYC"})
```

**Query Planning with Compound Indexes:**

The query planner intelligently chooses between compound and single-field indexes:

```go
// Create both types of indexes
coll.CreateIndex("city", false)              // Single-field index
coll.CreateCompoundIndex([]string{"city", "age"}, false)  // Compound index

// Query matching both fields
coll.Find({"city": "NYC", "age": 30})
// Chooses compound index (lower cost, no additional filtering needed)

// Query matching only city
coll.Find({"city": "NYC"})
// May choose single-field index (simpler, lower overhead)
```

**Performance Characteristics:**

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Full match | O(log n) | Exact composite key lookup |
| Prefix match | O(log n + k) | Range scan with prefix filtering |
| Non-prefix | O(n) | Cannot use index |

**Best Practices:**

1. **Order fields by selectivity**: Put most selective fields first
   ```go
   // Good: email is unique, department has few values
   CreateCompoundIndex([]string{"email", "department"}, false)

   // Poor: department first limits usefulness
   CreateCompoundIndex([]string{"department", "email"}, false)
   ```

2. **Match your query patterns**: Design compound indexes for common queries
   ```go
   // Common query: Find users by city and age
   CreateCompoundIndex([]string{"city", "age"}, false)
   ```

3. **Avoid redundancy**: Don't create both compound and single-field indexes on same prefix
   ```go
   // Redundant: compound index can handle city-only queries
   CreateIndex("city", false)
   CreateCompoundIndex([]string{"city", "age"}, false)
   ```

4. **Limit field count**: More fields = larger keys = more overhead
   - Maximum recommended: 3-4 fields
   - Beyond this, consider document design changes

**Statistics for Compound Indexes:**

Compound indexes track the same statistics as single-field indexes:

```go
indexes := coll.ListIndexes()
for _, idx := range indexes {
    if idx["is_compound"].(bool) {
        fmt.Printf("Compound Index: %s\n", idx["name"])
        fmt.Printf("  Fields: %v\n", idx["field_paths"])
        fmt.Printf("  Cardinality: %v\n", idx["cardinality"])
        fmt.Printf("  Selectivity: %.3f\n", idx["selectivity"])
    }
}
```

## Comparison: With vs Without Index

### Without Index (Collection Scan)

```
Query: Find user with email = "alice@example.com"

Process:
1. Read document 1 → Check email → No match
2. Read document 2 → Check email → No match
...
999. Read document 999 → Check email → No match
1000. Read document 1000 → Check email → MATCH!

Time: O(n) = 1000 operations
```

### With Index

```
Query: Find user with email = "alice@example.com"

Process:
1. Search index root → Find child
2. Search internal node → Find leaf
3. Search leaf → Find document ID
4. Read document by ID → MATCH!

Time: O(log n) = ~10 operations
100× faster!
```

## Future Enhancements

Potential improvements for educational exploration:

1. **Full delete rebalancing**: Borrow/merge nodes on underflow
2. **Partial indexes**: Index only matching documents
3. **Text indexes**: Full-text search capability
4. **Geospatial indexes**: Location-based queries
5. **Index hints**: Let users specify which index to use
6. **Covering index expansion**: Store additional fields in index for more covered queries
7. **Range queries on compound index suffix**: Support range operators on the last field of compound indexes

## Debugging Tips

### Visualize Tree Structure

```go
btree := index.NewBTree(3)
// ... insert data ...
btree.Print()

// Output:
// B+ Tree Structure:
//   Internal: [25 50]
//     Leaf: [10 15 20]
//     Leaf: [30 40]
//     Leaf: [60 70 80]
```

### Check Index Size

```go
stats := coll.Stats()
for _, idx := range stats["index_details"].([]map[string]interface{}) {
    size := idx["size"]
    height := idx["height"]
    fmt.Printf("%s: %v keys, height %v\n", idx["name"], size, height)
}
```

### Verify Index Integrity

After operations, check that document count matches index size:
```go
docCount := coll.Count(map[string]interface{}{})
indexSize := index.Size()

if docCount != indexSize {
    log.Printf("Warning: Index may be corrupt!")
}
```

## Summary

- **B+ trees** provide O(log n) search with efficient range scans
- **Automatic maintenance** keeps indexes synchronized with data
- **Trade-offs**: Faster reads, slower writes, more storage
- **Best practice**: Index frequently-queried fields only

Indexes are crucial for database performance. A well-indexed database can be 100× faster than an unindexed one!
