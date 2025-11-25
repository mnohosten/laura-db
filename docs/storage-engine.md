# Storage Engine

## Overview

The storage engine is responsible for persisting data to disk reliably and efficiently. It consists of four main components:

1. **Disk Manager**: Low-level I/O operations
2. **Page Management**: Fixed-size data blocks
3. **Buffer Pool**: In-memory page cache
4. **Write-Ahead Log (WAL)**: Durability guarantee

## Architecture

```
┌─────────────────────────────────────────────────┐
│           Storage Engine API                     │
└─────────────────────────────────────────────────┘
                      │
         ┌────────────┴────────────┐
         ▼                          ▼
┌─────────────────┐        ┌─────────────────┐
│   Buffer Pool   │        │      WAL        │
│  (LRU Cache)    │        │  (Durability)   │
└─────────────────┘        └─────────────────┘
         │                          │
         └────────────┬─────────────┘
                      ▼
              ┌─────────────────┐
              │  Disk Manager   │
              │   (I/O Layer)   │
              └─────────────────┘
                      │
                      ▼
              ┌─────────────────┐
              │   Disk Files    │
              │ data.db, wal.log│
              └─────────────────┘
```

## Page Management

### Page Structure

Pages are the fundamental unit of storage. Each page is 4KB (typical OS page size for optimal I/O).

```
┌─────────────────────────────────────┐
│           Page Header (16 bytes)     │
├─────────────────────────────────────┤
│  PageID (4 bytes)                    │
│  Type (1 byte)                       │
│  Flags (1 byte)                      │
│  LSN (8 bytes)                       │
│  Reserved (2 bytes)                  │
├─────────────────────────────────────┤
│           Page Data (4080 bytes)     │
│                                      │
│  Actual document/index data          │
│                                      │
└─────────────────────────────────────┘
```

### Page Types

- **Data Page**: Stores document data using slotted page structure
  - Slotted page layout for variable-length documents
  - Slot directory maps slot IDs to document offsets
  - Supports in-place updates and compaction
  - Managed by DocumentStore and DocumentPageManager
- **Index Page**: B+ tree nodes
- **Free List Page**: Tracks available pages
- **Overflow Page**: Large documents spanning multiple pages

### LSN (Log Sequence Number)

Each page has an LSN that indicates the last WAL record applied to it. This is critical for recovery:

- If page LSN < WAL record LSN: Need to replay the record
- If page LSN >= WAL record LSN: Page is up-to-date

## Buffer Pool

The buffer pool is an in-memory cache of pages using LRU (Least Recently Used) eviction.

### LRU Eviction Policy

```
Most Recent                            Least Recent
    ↓                                       ↓
┌─────┐  ┌─────┐  ┌─────┐  ┌─────┐  ┌─────┐
│ P10 │→│ P5  │→│ P3  │→│ P7  │→│ P2  │
└─────┘  └─────┘  └─────┘  └─────┘  └─────┘
```

When a page is accessed, it moves to the front. When evicting, we take from the back.

### Pin Count

Pages have a "pin count" that prevents eviction while in use:

```go
page, _ := bufferPool.FetchPage(pageID) // Pin count: 1
// Use the page...
bufferPool.UnpinPage(pageID, isDirty)   // Pin count: 0
// Now eligible for eviction
```

This prevents evicting a page that's actively being modified.

### Buffer Pool Operations

**Fetch Page**:
1. Check if page is in buffer pool (cache hit)
2. If not, check if buffer pool is full
3. If full, evict LRU unpinned page
4. Read page from disk
5. Add to buffer pool and pin it

**Evict Page**:
1. Find LRU page with pin count = 0
2. If dirty, flush to disk
3. Remove from buffer pool

### Performance Metrics

- **Hit Rate**: (Hits / (Hits + Misses)) × 100%
- **Eviction Count**: Number of pages evicted
- **Dirty Page Ratio**: Dirty pages / Total pages

Good hit rate: > 95% for typical workloads

## Write-Ahead Logging (WAL)

WAL ensures durability: all changes are logged before being applied to data pages.

### WAL Protocol

```
1. Generate log record for operation
2. Append log record to WAL (in memory buffer)
3. Flush WAL to disk (on commit or buffer full)
4. Apply changes to pages in buffer pool
5. Mark pages as dirty
6. Eventually flush dirty pages to disk
```

### Log Record Format

```
┌──────────────────────────────────────────┐
│ LSN (8 bytes) - Log Sequence Number       │
├──────────────────────────────────────────┤
│ Type (1 byte) - Insert/Update/Delete/etc  │
├──────────────────────────────────────────┤
│ TxnID (8 bytes) - Transaction ID          │
├──────────────────────────────────────────┤
│ PageID (4 bytes) - Affected page          │
├──────────────────────────────────────────┤
│ PrevLSN (8 bytes) - Previous LSN for txn  │
├──────────────────────────────────────────┤
│ DataLen (4 bytes) - Length of data        │
├──────────────────────────────────────────┤
│ Data (variable) - Operation-specific data │
└──────────────────────────────────────────┘
```

### Crash Recovery

**Recovery Process**:

1. Read WAL from beginning
2. For each log record:
   - If page LSN < record LSN: Apply the change
   - If page LSN >= record LSN: Skip (already applied)
3. Flush all recovered pages to disk
4. Database is now consistent

**Example**:

```
WAL contains:
  LSN=100: Insert doc into page 5
  LSN=101: Update doc in page 5
  LSN=102: Insert doc into page 7
  LSN=103: Checkpoint

Crash occurs before page 5 flushed to disk

Recovery:
  - Read page 5: LSN=99 (old)
  - Replay LSN=100: Insert doc
  - Replay LSN=101: Update doc
  - Page 5 now at LSN=101
  - Read page 7: LSN=102 (already flushed)
  - Skip replay for page 7
```

### Checkpointing

Checkpoints reduce recovery time by persisting all dirty pages:

1. Flush all dirty pages to disk
2. Write checkpoint record to WAL
3. Sync disk
4. Recovery can start from checkpoint instead of beginning

**Checkpoint Frequency**: Balance between:
- Too frequent: High I/O overhead
- Too infrequent: Long recovery time

Typical: Every 1-5 minutes or after N transactions

## Disk Manager

Low-level I/O operations:

```go
// Read page from disk
page, err := diskMgr.ReadPage(pageID)

// Write page to disk
err := diskMgr.WritePage(page)

// Allocate new page
pageID, err := diskMgr.AllocatePage()

// Free page
err := diskMgr.DeallocatePage(pageID)
```

### Page Allocation

**Free List**: Track deallocated pages for reuse

```
Free Pages: [7, 12, 15, 23]

AllocatePage() → Returns 23 (last in list)
Free Pages: [7, 12, 15]

DeallocatePage(20) → Add to free list
Free Pages: [7, 12, 15, 20]
```

This prevents fragmentation and reuses space efficiently.

## Usage Example

```go
// Create storage engine
config := storage.DefaultConfig("./data")
engine, err := storage.NewStorageEngine(config)
if err != nil {
    log.Fatal(err)
}
defer engine.Close()

// Allocate a new page
page, err := engine.AllocatePage()
if err != nil {
    log.Fatal(err)
}

// Write data to page
copy(page.Data, []byte("Hello, world!"))
page.MarkDirty()

// Log the operation
record := &storage.LogRecord{
    Type:   storage.LogRecordInsert,
    TxnID:  1,
    PageID: page.ID,
    Data:   []byte("metadata"),
}
lsn, err := engine.LogOperation(record)
page.LSN = lsn

// Unpin the page (allow eviction)
engine.UnpinPage(page.ID, true)

// Checkpoint
engine.Checkpoint()

// View stats
stats := engine.Stats()
fmt.Printf("Buffer pool hit rate: %.2f%%\n",
    stats["buffer_pool"].(map[string]interface{})["hit_rate"])
```

## Design Trade-offs

### Page Size: 4KB

**Advantages**:
- Matches OS page size (efficient I/O)
- Good balance for most workloads
- Reduces internal fragmentation

**Disadvantages**:
- Large documents may span multiple pages
- More pages = more overhead

### LRU Eviction

**Advantages**:
- Simple to implement
- Works well for most access patterns
- O(1) operations with hash map + linked list

**Disadvantages**:
- Can perform poorly with sequential scans
- Doesn't consider page importance
- Alternative: LRU-K, 2Q, ARC (more complex)

### Write-Ahead Logging

**Advantages**:
- Strong durability guarantee
- Enables fast recovery
- Sequential writes (fast)

**Disadvantages**:
- Double write penalty (log + data)
- Extra storage space for logs
- Needs periodic log truncation

## Performance Characteristics

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Fetch page (cached) | O(1) | Hash map lookup |
| Fetch page (uncached) | O(log n) + I/O | Eviction + disk read |
| Write page | O(1) + I/O | Mark dirty, flush later |
| WAL append | O(1) + I/O | Sequential write |
| Checkpoint | O(n) | Flush all dirty pages |

## Related Documentation

For detailed information on document storage implementation:
- **[disk-storage-design.md](disk-storage-design.md)**: Slotted page structure and document persistence
- **[collection-metadata-design.md](collection-metadata-design.md)**: Collection catalog and metadata management
- **[index-persistence-design.md](index-persistence-design.md)**: B+ tree node persistence

## Future Enhancements

1. **Group Commit**: Batch multiple transactions' WAL writes
2. **Async I/O**: Non-blocking disk operations
3. **Compression**: Compress pages to reduce disk usage
4. **Direct I/O**: Bypass OS page cache for better control
5. **Multiple Buffer Pools**: Per-table or per-index pools
6. **Log Compression**: Reduce WAL size
7. **Parallel Recovery**: Multi-threaded WAL replay
