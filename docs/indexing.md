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

```go
indexes := coll.ListIndexes()
for _, idx := range indexes {
    fmt.Printf("Index: %s\n", idx["name"])
    fmt.Printf("  Field: %s\n", idx["field_path"])
    fmt.Printf("  Unique: %v\n", idx["unique"])
    fmt.Printf("  Size: %v keys\n", idx["size"])
    fmt.Printf("  Height: %v levels\n", idx["height"])
}
```

## Query Optimization

### Index Selection

When executing a query, the database should choose the best index:

```go
// This query can use city index
coll.Find(map[string]interface{}{
    "city": "New York",
    "age": map[string]interface{}{"$gt": 25},
})

// Strategy:
// 1. Use city index to find "New York" documents
// 2. Filter by age > 25 in memory
// Better than scanning all documents!
```

**Our implementation**: Currently scans all documents (simpler). Production databases use query optimizers to choose best index.

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

### Compound Indexes (Future Enhancement)

Our implementation supports single-field indexes. Production databases support compound indexes:

```go
// Hypothetical compound index
CreateIndex([]string{"city", "age"}, false)

// Efficient for queries like:
Find({"city": "New York", "age": {"$gt": 25}})
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
2. **Compound indexes**: Multiple fields per index
3. **Partial indexes**: Index only matching documents
4. **Text indexes**: Full-text search capability
5. **Geospatial indexes**: Location-based queries
6. **Index statistics**: Track index usage and selectivity
7. **Index hints**: Let users specify which index to use
8. **Covering indexes**: Store all queried fields in index

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
