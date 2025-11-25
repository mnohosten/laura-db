# Index Persistence Design

## Overview

This document describes the design for persisting indexes to disk, enabling indexes to survive server restarts. Currently, all indexes (B+ tree, text, geo, etc.) are stored in memory. This design enables index persistence through the existing storage engine infrastructure.

## Goals

1. **Persistence**: Indexes survive server restarts
2. **Performance**: Minimal impact on query performance through caching
3. **Scalability**: Support indexes larger than available memory
4. **Compatibility**: Work with existing index types (B+ tree, text, geo, TTL, partial)
5. **MVCC Integration**: Support versioned index entries for snapshot isolation
6. **Efficient Updates**: Minimize disk I/O during index modifications

## Architecture Overview

```
┌──────────────────────────────────────────────────────┐
│              Collection Layer                         │
│  (CRUD operations, index management)                  │
└──────────────────────────────────────────────────────┘
                        ↓
┌──────────────────────────────────────────────────────┐
│              Index Layer                              │
│  (B+ tree, text, geo indexes)                         │
│  - In-memory: keys, values, structure                │
│  - Disk: B+ tree nodes, inverted index, R-tree       │
└──────────────────────────────────────────────────────┘
                        ↓
┌──────────────────────────────────────────────────────┐
│           Index Page Manager                          │
│  (node serialization, page allocation)                │
└──────────────────────────────────────────────────────┘
                        ↓
┌──────────────────────────────────────────────────────┐
│              Storage Engine                           │
│  (buffer pool, WAL, disk manager)                     │
└──────────────────────────────────────────────────────┘
                        ↓
┌──────────────────────────────────────────────────────┐
│                 Disk Files                            │
│  (data.db, wal.log, metadata.db)                      │
└──────────────────────────────────────────────────────┘
```

## B+ Tree Node Persistence

### Current In-Memory Structure

```go
type BTreeNode struct {
    isLeaf   bool
    keys     []interface{}        // In-memory keys
    values   []interface{}        // In-memory values (DocumentIDs)
    children []*BTreeNode         // In-memory pointers
    next     *BTreeNode           // In-memory pointer to next leaf
    parent   *BTreeNode           // In-memory pointer to parent
}
```

### Disk-Based Structure

```go
type BTreeNode struct {
    // Existing fields
    isLeaf   bool
    keys     []interface{}
    values   []interface{}
    children []*BTreeNode
    next     *BTreeNode
    parent   *BTreeNode

    // New fields for disk persistence
    pageID   PageID              // Page containing this node
    isDirty  bool                // Node modified in memory
    isLoaded bool                // Node loaded from disk

    // Child page IDs (for disk-based nodes)
    childPageIDs []PageID         // PageIDs of children (internal nodes)
    nextPageID   PageID           // PageID of next leaf (leaf nodes)
    parentPageID PageID           // PageID of parent node
}
```

### B+ Tree Node Page Format

#### Internal Node Page

```
┌────────────────────────────────────────────────────────┐
│                Page Header (16 bytes)                   │  ← Storage engine header
│  - PageID (4 bytes)                                     │
│  - PageType (1 byte) = PageTypeBTreeInternal           │
│  - LSN (8 bytes)                                        │
│  - Checksum (2 bytes)                                   │
│  - Reserved (1 byte)                                    │
├────────────────────────────────────────────────────────┤
│            B+ Tree Node Header (32 bytes)              │
│  - NodeType (1 byte): 0 = internal, 1 = leaf           │
│  - Level (1 byte): Distance from leaves (0 = leaf)     │
│  - KeyCount (2 bytes): Number of keys in node          │
│  - ChildCount (2 bytes): Number of children            │
│  - IndexID (4 bytes): Parent index identifier          │
│  - CollectionID (4 bytes): Parent collection           │
│  - ParentPageID (4 bytes): Parent node page            │
│  - NextPageID (4 bytes): Next sibling (0 = none)       │
│  - PrevPageID (4 bytes): Previous sibling (0 = none)   │
│  - FreeSpaceOffset (2 bytes): Start of free space      │
│  - Reserved (2 bytes)                                   │
├────────────────────────────────────────────────────────┤
│          Child Page ID Array (variable)                │
│  ChildPageID[0] (4 bytes)                              │
│  ChildPageID[1] (4 bytes)                              │
│  ...                                                    │
│  ChildPageID[n] (4 bytes)                              │
│  Note: ChildCount = KeyCount + 1 (B+ tree property)    │
├────────────────────────────────────────────────────────┤
│          Key Directory (variable)                       │
│  For each key:                                          │
│    - KeyOffset (2 bytes): Offset to key data           │
│    - KeyLength (2 bytes): Length of key data           │
│    - KeyType (1 byte): Type of key                     │
│                                                         │
│  KeyType values:                                        │
│    0 = int64, 1 = float64, 2 = string,                │
│    3 = ObjectID, 4 = composite key                     │
├────────────────────────────────────────────────────────┤
│               Free Space                                │
├────────────────────────────────────────────────────────┤
│          Key Data Area (grows upward from bottom)      │
│  Key[n]: [serialized key data]                         │
│  ...                                                    │
│  Key[1]: [serialized key data]                         │
│  Key[0]: [serialized key data]                         │
└────────────────────────────────────────────────────────┘

Total Page Size: 4096 bytes
Available Space: 4096 - 16 (page header) - 32 (node header) = 4048 bytes
```

#### Leaf Node Page

```
┌────────────────────────────────────────────────────────┐
│                Page Header (16 bytes)                   │
│  - PageID (4 bytes)                                     │
│  - PageType (1 byte) = PageTypeBTreeLeaf               │
│  - LSN (8 bytes)                                        │
│  - Checksum (2 bytes)                                   │
│  - Reserved (1 byte)                                    │
├────────────────────────────────────────────────────────┤
│            B+ Tree Node Header (32 bytes)              │
│  - NodeType (1 byte): 1 = leaf                         │
│  - Level (1 byte): 0 (leaf level)                      │
│  - EntryCount (2 bytes): Number of key-value pairs     │
│  - Reserved (2 bytes)                                   │
│  - IndexID (4 bytes): Parent index identifier          │
│  - CollectionID (4 bytes): Parent collection           │
│  - ParentPageID (4 bytes): Parent node page            │
│  - NextPageID (4 bytes): Next leaf in chain            │
│  - PrevPageID (4 bytes): Previous leaf in chain        │
│  - FreeSpaceOffset (2 bytes): Start of free space      │
│  - Reserved (2 bytes)                                   │
├────────────────────────────────────────────────────────┤
│          Entry Directory (variable)                     │
│  For each entry:                                        │
│    - KeyOffset (2 bytes): Offset to key data           │
│    - KeyLength (2 bytes): Length of key data           │
│    - KeyType (1 byte): Type of key                     │
│    - ValueOffset (2 bytes): Offset to value data       │
│    - ValueLength (2 bytes): Length of value data       │
│    - Flags (1 byte): Entry flags                       │
│                                                         │
│  Entry size: 10 bytes per entry                        │
│                                                         │
│  Flags (1 byte):                                        │
│    - Bit 0: Deleted (for MVCC)                         │
│    - Bit 1: Versioned (has TxnID)                      │
│    - Bits 2-7: Reserved                                 │
├────────────────────────────────────────────────────────┤
│               Free Space                                │
├────────────────────────────────────────────────────────┤
│    Key-Value Data Area (grows upward from bottom)      │
│  Entry[n]:                                              │
│    - Key data [variable]                               │
│    - Value data [variable]                             │
│      * DocumentID (12 bytes)                           │
│      * TxnID (8 bytes, optional if versioned)          │
│  ...                                                    │
│  Entry[0]: ...                                          │
└────────────────────────────────────────────────────────┘
```

### Key Serialization Format

Different key types are serialized differently:

#### int64 Key
```
[int64 value] (8 bytes, little-endian)
```

#### float64 Key
```
[float64 value] (8 bytes, IEEE 754, little-endian)
```

#### string Key
```
[length (2 bytes)][UTF-8 bytes (variable)]
```

#### ObjectID Key
```
[12 bytes] (raw ObjectID bytes)
```

#### Composite Key (Compound Indexes)
```
[field count (1 byte)]
For each field:
  [field type (1 byte)]
  [field length (2 bytes)]
  [field data (variable)]
```

### Value Serialization Format

Index values store DocumentIDs (references to documents):

```
DocumentID (12 bytes):
┌────────────────────────────────────────┐
│ CollectionID (4 bytes)                 │
│ PageID (4 bytes)                       │
│ SlotID (2 bytes)                       │
│ Reserved (2 bytes)                     │
└────────────────────────────────────────┘

For MVCC-enabled indexes (optional):
┌────────────────────────────────────────┐
│ DocumentID (12 bytes)                  │
│ CreatedTxnID (8 bytes)                 │
│ DeletedTxnID (8 bytes, 0 = not deleted)│
└────────────────────────────────────────┘

Total: 12 bytes (basic) or 28 bytes (with MVCC)
```

## Index-to-Document Pointer Mapping

### Current: In-Memory Pointers

Currently, indexes store direct Go pointers to documents in memory:

```go
// Current (in-memory)
btree.Insert(key, documentPointer)  // documentPointer is *map[string]interface{}
```

### Disk-Based: DocumentID References

With disk storage, indexes will store DocumentIDs instead:

```go
// Disk-based
type DocumentID struct {
    CollectionID uint32
    PageID       uint32
    SlotID       uint16
    Reserved     uint16
}

btree.Insert(key, documentID)  // documentID is a 12-byte struct
```

### Lookup Flow

**Before (in-memory)**:
```
Query → Index.Search(key) → Document pointer → Return document
```

**After (disk-based)**:
```
Query → Index.Search(key) → DocumentID → Collection.FetchDocument(documentID) → Return document
                                               ↓
                                        Page read from disk/cache
                                               ↓
                                        Deserialize document from page slot
```

## Node Caching Strategy

### Two-Level Caching

To minimize disk I/O, use a two-level caching strategy:

1. **Page Cache (Buffer Pool)**: Already exists, caches raw pages
2. **Node Cache**: New layer, caches deserialized B+ tree nodes

```go
type NodeCache struct {
    cache    map[PageID]*BTreeNode  // PageID → deserialized node
    lru      *list.List              // LRU eviction list
    maxSize  int                     // Max nodes in cache
    mu       sync.RWMutex
}
```

### Cache Hit Flow

```
1. Index.Search(key)
2. Check NodeCache for root node
   - Hit: Use cached node
   - Miss: Read page from BufferPool → Deserialize → Cache node
3. Traverse to child
4. Check NodeCache for child node
   - Hit: Use cached node
   - Miss: Read page from BufferPool → Deserialize → Cache node
5. Repeat until leaf found
6. Return value (DocumentID)
```

### Cache Invalidation

- **Node modified**: Mark node as dirty, write back on eviction or checkpoint
- **Node split**: Invalidate parent node cache entry
- **Transaction rollback**: Invalidate all modified node cache entries

### Cache Size Tuning

Default configuration:
- **Buffer Pool**: 1,000 pages (4 MB)
- **Node Cache**: 500 nodes (~2-3 MB depending on node size)
- **Combined**: ~6-7 MB of index cache

For read-heavy workloads, increase cache sizes:
- **Buffer Pool**: 10,000 pages (40 MB)
- **Node Cache**: 5,000 nodes (~20-30 MB)

## Lazy Node Loading

To minimize startup time and memory usage, use lazy loading:

### Node Loading Strategy

```go
type BTreeNode struct {
    // ... existing fields

    // Lazy loading state
    isLoaded   bool      // Node structure loaded from disk
    childrenLoaded bool  // Children pointers loaded
}

// Load node from disk on first access
func (bt *BTree) loadNode(pageID PageID) (*BTreeNode, error) {
    // Check cache first
    if node := bt.nodeCache.Get(pageID); node != nil {
        return node, nil
    }

    // Read page from disk
    page, err := bt.storage.ReadPage(pageID)
    if err != nil {
        return nil, err
    }

    // Deserialize node
    node, err := bt.deserializeNode(page)
    if err != nil {
        return nil, err
    }

    // Cache node
    bt.nodeCache.Put(pageID, node)

    return node, nil
}

// Load children lazily when traversing
func (bt *BTree) loadChildren(node *BTreeNode) error {
    if node.childrenLoaded {
        return nil  // Already loaded
    }

    node.children = make([]*BTreeNode, len(node.childPageIDs))
    for i, childPageID := range node.childPageIDs {
        // Don't load child yet, just set pageID
        node.children[i] = &BTreeNode{
            pageID: childPageID,
            isLoaded: false,
        }
    }

    node.childrenLoaded = true
    return nil
}
```

### Lazy Loading Benefits

- **Fast startup**: Only load root node metadata
- **Memory efficient**: Load nodes on-demand
- **Scalable**: Support very large indexes (millions of nodes)

## Index Operations with Disk Persistence

### Insert Operation

```
Insert(key, documentID):
1. Begin transaction
2. Traverse tree to find leaf (load nodes from disk as needed)
3. Insert key-value into leaf
4. If leaf is full:
   a. Split leaf node
   b. Allocate new page for new leaf
   c. Write both leaf pages to disk
   d. Propagate split up to parent (may cascade)
5. Mark all modified nodes as dirty
6. Log operation to WAL:
   - WAL: InsertIndexEntry(indexID, key, documentID)
   - WAL: WriteNodePage(pageID, nodeData)
7. Commit transaction
8. Update node cache with modified nodes
9. Background: Flush dirty pages to disk
```

### Search Operation

```
Search(key):
1. Load root node (from cache or disk)
2. Traverse to child based on key comparison
3. Load child node (from cache or disk)
4. Repeat until leaf node
5. Binary search in leaf for key
6. If found: Return DocumentID
7. If not found: Return nil
```

### Delete Operation

```
Delete(key):
1. Begin transaction
2. Traverse tree to find leaf containing key
3. Remove key-value from leaf
4. If leaf underflows (optional, for full B+ tree):
   a. Redistribute with sibling, or
   b. Merge with sibling
   c. Update parent keys
5. Mark modified nodes as dirty
6. Log to WAL:
   - WAL: DeleteIndexEntry(indexID, key)
   - WAL: WriteNodePage(pageID, nodeData)
7. Commit transaction
8. Update node cache
```

### Range Scan Operation

```
RangeScan(start, end):
1. Find leaf containing start key
2. Load leaf node from cache/disk
3. Collect entries in range [start, end]
4. Follow next pointers to subsequent leaves
5. Load next leaf from cache/disk (prefetch for performance)
6. Continue until end key or end of range
7. Return all matching entries (DocumentIDs)
```

**Optimization**: Prefetch next N leaf pages in background during range scan.

## Handling Node Splits on Disk

### Node Split Process

When a node is full and requires a split:

```
SplitNode(node):
1. Check if node is full (len(keys) >= order)
2. Calculate midpoint: mid = len(keys) / 2
3. Allocate new page for new node
4. Create new node:
   - Copy keys[mid:] to new node
   - Copy values[mid:] to new node (if leaf)
   - Copy children[mid:] to new node (if internal)
5. Update original node:
   - Truncate keys to keys[:mid]
   - Truncate values/children accordingly
6. Update sibling pointers:
   - Original node's next = new node
   - New node's next = original node's old next
7. Mark both nodes as dirty
8. Return separator key to promote to parent
9. Write both nodes to disk:
   - WritePage(original.pageID, serialize(original))
   - WritePage(newNode.pageID, serialize(newNode))
10. Log to WAL:
    - WAL: SplitNode(originalPageID, newPageID, separatorKey)
```

### Parent Update After Split

```
UpdateParentAfterSplit(parent, separatorKey, newChildPageID):
1. Find position in parent for separator key
2. Insert separator key at position
3. Insert new child page ID at position+1
4. If parent is full:
   a. Recursively split parent
   b. May cascade up to root
5. If root splits:
   a. Create new root node
   b. New root has 2 children (old root, new node)
   c. Allocate page for new root
   d. Update index metadata with new root page ID
6. Mark parent as dirty
7. Write parent to disk
```

## Text Index Persistence

Text indexes use an inverted index structure:

### Inverted Index Page Format

```
┌────────────────────────────────────────────────────────┐
│                Page Header (16 bytes)                   │
├────────────────────────────────────────────────────────┤
│        Text Index Node Header (32 bytes)               │
│  - NodeType (1 byte): 0 = term node, 1 = posting list │
│  - IndexID (4 bytes): Parent text index ID             │
│  - TermCount (2 bytes): Number of terms (if term node) │
│  - PostingCount (4 bytes): Number of postings          │
│  - TotalFrequency (4 bytes): Sum of term frequencies   │
│  - Reserved (17 bytes)                                  │
├────────────────────────────────────────────────────────┤
│          Term Directory (variable)                      │
│  For each term:                                         │
│    - TermLength (2 bytes)                              │
│    - TermOffset (2 bytes): Offset to term string       │
│    - PostingListPageID (4 bytes): Page with postings   │
│    - DocumentFrequency (4 bytes): # docs with term     │
├────────────────────────────────────────────────────────┤
│               Free Space                                │
├────────────────────────────────────────────────────────┤
│          Term String Data (from bottom)                 │
│  Term[n]: [UTF-8 string]                               │
│  ...                                                    │
│  Term[0]: [UTF-8 string]                               │
└────────────────────────────────────────────────────────┘
```

### Posting List Page Format

```
┌────────────────────────────────────────────────────────┐
│                Page Header (16 bytes)                   │
├────────────────────────────────────────────────────────┤
│        Posting List Header (16 bytes)                  │
│  - Term (4 bytes hash): Hash of term for validation    │
│  - PostingCount (4 bytes): Number of postings          │
│  - NextPageID (4 bytes): Overflow page (0 = none)      │
│  - Reserved (4 bytes)                                   │
├────────────────────────────────────────────────────────┤
│          Posting Entries (variable)                     │
│  For each posting:                                      │
│    - DocumentID (12 bytes): Reference to document      │
│    - TermFrequency (2 bytes): Frequency in document    │
│    - FieldLength (2 bytes): Length of field            │
│                                                         │
│  Entry size: 16 bytes per posting                      │
│  Capacity: ~250 postings per page                      │
└────────────────────────────────────────────────────────┘
```

### Text Index Storage Strategy

1. **Term Dictionary**: B+ tree with terms as keys, posting list page IDs as values
2. **Posting Lists**: Separate pages containing document references and frequencies
3. **BM25 Scoring**: Calculate scores on-the-fly using stored frequencies

## Geospatial Index Persistence (R-tree)

Geospatial indexes use an R-tree structure:

### R-tree Node Page Format

```
┌────────────────────────────────────────────────────────┐
│                Page Header (16 bytes)                   │
├────────────────────────────────────────────────────────┤
│          R-tree Node Header (32 bytes)                 │
│  - NodeType (1 byte): 0 = internal, 1 = leaf           │
│  - Level (1 byte): Distance from leaves                │
│  - EntryCount (2 bytes): Number of entries             │
│  - IndexID (4 bytes): Parent geo index ID              │
│  - ParentPageID (4 bytes): Parent node page            │
│  - MBR (Minimum Bounding Rectangle) (16 bytes):       │
│    * MinX (4 bytes, float32)                           │
│    * MinY (4 bytes, float32)                           │
│    * MaxX (4 bytes, float32)                           │
│    * MaxY (4 bytes, float32)                           │
│  - Reserved (4 bytes)                                   │
├────────────────────────────────────────────────────────┤
│          Entry Directory (variable)                     │
│  For each entry:                                        │
│    - MBR (16 bytes): Bounding rectangle                │
│    - ChildPageID (4 bytes): Child page (internal node) │
│      OR DocumentID (12 bytes): Document (leaf node)    │
│    - Reserved (4 bytes)                                 │
│                                                         │
│  Entry size: 24 bytes per entry (internal)             │
│              32 bytes per entry (leaf)                 │
└────────────────────────────────────────────────────────┘
```

### Geospatial Coordinate Storage

```
2D Point:
┌────────────────────────────────────────┐
│ X (4 bytes, float32)                   │
│ Y (4 bytes, float32)                   │
└────────────────────────────────────────┘

2D Sphere (longitude, latitude):
┌────────────────────────────────────────┐
│ Longitude (8 bytes, float64)           │  -180 to 180
│ Latitude (8 bytes, float64)            │  -90 to 90
└────────────────────────────────────────┘
```

## MVCC Integration with Indexes

### Versioned Index Entries

For snapshot isolation, index entries need version information:

```
Versioned Index Entry:
┌────────────────────────────────────────┐
│ Key (variable)                         │
│ DocumentID (12 bytes)                  │
│ CreatedTxnID (8 bytes)                 │
│ DeletedTxnID (8 bytes, 0 = active)     │
└────────────────────────────────────────┘
```

### Index Visibility Rules

When reading from an index in a transaction:

```
IsVisible(entry, txn):
  return entry.CreatedTxnID <= txn.ReadVersion
     AND (entry.DeletedTxnID == 0 OR entry.DeletedTxnID > txn.ReadVersion)
```

### Index Update Strategy

**Option 1: In-Place Updates (Recommended)**
- Update existing entry with new DeletedTxnID
- Insert new entry with new CreatedTxnID
- Garbage collect old entries when safe

**Option 2: Append-Only**
- Never modify entries
- Always insert new entry for updates
- More entries, more garbage to collect

**Decision**: Use Option 1 (in-place updates) for space efficiency.

## Free Space Management

### Index Page Compaction

Similar to document pages, index node pages can become fragmented:

```
CompactIndexNode(node):
1. Create new key and value arrays
2. Copy active entries (non-deleted)
3. Rebuild directory with contiguous offsets
4. Update FreeSpaceOffset
5. Mark node as dirty
6. Write compacted node to disk
```

Trigger compaction when:
- Free space > 25% of page size
- Entry count < 50% of capacity

## Index Rebuild

For corrupted or heavily fragmented indexes, provide rebuild capability:

```
RebuildIndex(indexID):
1. Create new empty index
2. Scan all documents in collection
3. For each document:
   a. Extract indexed fields
   b. Insert into new index
4. Replace old index with new index
5. Update metadata with new root page ID
6. Deallocate old index pages
```

## Performance Optimizations

### 1. Node Prefetching

During range scans, prefetch next N nodes in background:

```go
func (bt *BTree) prefetchNodes(pageIDs []PageID) {
    go func() {
        for _, pageID := range pageIDs {
            bt.loadNode(pageID)  // Load into cache
        }
    }()
}
```

### 2. Bulk Loading

For initial index creation, use bulk loading:

```
BulkLoad(sortedEntries):
1. Sort all entries by key
2. Build leaf nodes from bottom up
3. Write leaves to consecutive pages (locality)
4. Build internal nodes from leaf keys
5. Create root node
6. All nodes written sequentially (fast writes)
```

**Performance**: 10-100x faster than incremental inserts.

### 3. Write Buffering

Batch index updates in memory before flushing:

```go
type IndexWriteBuffer struct {
    updates  []IndexUpdate   // Pending updates
    maxSize  int             // Max buffer size
    flushFn  func([]IndexUpdate) error
}

// Flush when buffer full or on commit
func (buf *IndexWriteBuffer) Flush() error {
    return buf.flushFn(buf.updates)
}
```

### 4. Node Compression

For leaf nodes with repetitive keys (e.g., partial indexes), use prefix compression:

```
Original keys:
  "user_email_john@example.com"
  "user_email_jane@example.com"
  "user_email_bob@example.com"

Prefix compressed:
  Prefix: "user_email_"
  Keys: "john@...", "jane@...", "bob@..."
```

## Space Efficiency Analysis

### B+ Tree Node Overhead

Assuming average node:
- Order: 32
- Average key count: 20
- Average key size: 20 bytes
- Value size: 12 bytes (DocumentID)

| Component | Size | Percentage |
|-----------|------|------------|
| Page header | 16 bytes | 0.4% |
| Node header | 32 bytes | 0.8% |
| Key directory (20 entries) | 100 bytes | 2.4% |
| Key data (20 × 20 bytes) | 400 bytes | 9.8% |
| Value data (20 × 12 bytes) | 240 bytes | 5.9% |
| Child pointers (21 × 4 bytes) | 84 bytes | 2.1% |
| Free space | ~3,220 bytes | 78.6% |
| **Total** | **4,096 bytes** | **100%** |

**Overhead**: ~15% (very efficient for sparse nodes)

### Index Size Estimation

For 1 million documents with indexed field:
- Leaf nodes needed: ~50,000 (20 entries per leaf)
- Internal nodes (3 levels): ~2,500
- Total nodes: ~52,500
- **Index size**: 52,500 × 4 KB = **210 MB**

Compare to in-memory: ~150 MB (pointers vs. DocumentIDs)

## Configuration Options

```go
type IndexConfig struct {
    // Node cache size
    NodeCacheSize int // default: 500 nodes

    // Prefetch size
    PrefetchSize int // default: 8 nodes

    // Enable MVCC versioning
    EnableMVCC bool // default: false (for now)

    // Write buffer size
    WriteBufferSize int // default: 1000 updates

    // Compaction threshold
    CompactionThreshold float64 // default: 0.25 (25% free space)

    // Bulk load batch size
    BulkLoadBatchSize int // default: 10000 entries
}
```

## Testing Strategy

### Unit Tests

- B+ tree node serialization/deserialization
- Node split with disk persistence
- Lazy node loading
- Node cache operations
- Text index persistence
- R-tree node persistence

### Integration Tests

1. **Insert → Restart → Search**
   - Insert 10,000 entries
   - Restart database
   - Verify all entries searchable

2. **Index rebuild**
   - Create index
   - Insert documents
   - Rebuild index
   - Verify correctness

3. **Range scan with disk nodes**
   - Insert 100,000 entries
   - Range scan [1000, 2000]
   - Verify all entries returned

4. **Node split cascade**
   - Fill tree until root splits
   - Verify tree structure correct
   - Restart and verify

### Performance Tests

- Insert throughput (disk vs. in-memory)
- Search latency (disk vs. in-memory)
- Range scan performance
- Cache hit rate analysis
- Bulk load performance

### Stress Tests

- Large index (100M+ entries)
- High write rate (10K+ inserts/sec)
- Crash during node split
- Concurrent index updates

## Migration Plan

### Phase 1: Add Page Types and Structures
- Define B+ tree node page types
- Implement node serialization/deserialization
- Add unit tests

### Phase 2: Implement Node Cache
- Create NodeCache struct
- LRU eviction policy
- Integration with storage engine

### Phase 3: Add Disk Persistence to B+ Tree
- Modify BTreeNode to support pageID
- Implement lazy node loading
- Write nodes to disk on modifications

### Phase 4: Update Index Operations
- Insert with disk persistence
- Search with lazy loading
- Delete with disk updates
- Range scan with prefetching

### Phase 5: Text and Geo Indexes
- Persist inverted index to disk
- Persist R-tree to disk
- Update search operations

### Phase 6: MVCC Integration
- Add version info to index entries
- Implement visibility checks
- Garbage collection for old versions

### Phase 7: Optimization
- Bulk loading
- Prefetching
- Write buffering
- Node compression

## API Changes

### Index API

```go
type Index struct {
    id           uint32      // IndexID (from metadata)
    collectionID uint32      // Parent collection ID
    rootPageID   PageID      // Root node page ID (NEW)
    nodeCache    *NodeCache  // Node cache (NEW)
    storage      *StorageEngine // Storage engine (NEW)

    // Existing fields
    btree        *BTree
    indexType    IndexType
    // ...
}

// Persist root node page ID
func (idx *Index) syncRootPage() error

// Load index from disk
func (idx *Index) loadFromDisk() error

// Flush dirty nodes to disk
func (idx *Index) flushNodes() error
```

### B+ Tree API

```go
type BTree struct {
    // Existing fields
    root   *BTreeNode
    order  int

    // New fields
    rootPageID  PageID       // Root node page ID (NEW)
    nodeCache   *NodeCache   // Node cache (NEW)
    storage     *StorageEngine // Storage engine (NEW)
}

// Load node from disk
func (bt *BTree) loadNode(pageID PageID) (*BTreeNode, error)

// Write node to disk
func (bt *BTree) writeNode(node *BTreeNode) error

// Flush all dirty nodes
func (bt *BTree) flush() error
```

## Future Enhancements

### 1. Index Compression
- Prefix compression for strings
- Delta encoding for integers
- Dictionary encoding for low-cardinality fields

### 2. Index Partitioning
- Split large indexes across multiple files
- Parallel index scans
- Distributed indexing

### 3. Adaptive Indexing
- Auto-create indexes based on query patterns
- Drop unused indexes
- Rebalance hot indexes

### 4. Index Statistics Histograms
- Store value distribution histograms
- Better query optimization
- Adaptive tree balancing

## Summary

This design provides:

✅ **Persistence**: Indexes stored on disk, survive restarts
✅ **Performance**: Node caching and lazy loading minimize I/O
✅ **Scalability**: Support indexes larger than memory
✅ **Compatibility**: Works with all existing index types
✅ **MVCC Ready**: Version tracking for snapshot isolation
✅ **Efficient**: Minimal overhead (~15% for B+ tree nodes)

**Key Design Decisions**:
1. **Page-Based Storage**: B+ tree nodes stored in 4KB pages
2. **Lazy Loading**: Load nodes on-demand to reduce memory usage
3. **Node Cache**: LRU cache for frequently accessed nodes
4. **DocumentID References**: Indexes store DocumentIDs instead of pointers
5. **Slotted Node Format**: Variable-length keys with directory structure
6. **Prefetching**: Read-ahead for range scans
7. **Write Buffering**: Batch updates for performance

**Index Size Examples**:
- 1M documents, simple index: ~210 MB
- 10M documents, compound index: ~2.1 GB
- 100M documents, text index: ~20-30 GB

**Performance Expectations** (with caching):
- Insert: 100-300µs (disk), 50-100µs (cached)
- Search: 50-200µs (disk), 20-50µs (cached)
- Range Scan: 500-2000µs per 100 entries

**Next Steps**: Proceed to Phase 2 implementation (storage layer enhancements in TODO.md).
