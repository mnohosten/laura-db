# Disk Storage Design

## Overview

This document describes the design for persisting documents to disk through the storage engine. Currently, documents are stored in memory (in-memory map in `Collection`). This design enables documents to survive server restarts by persisting them to disk pages.

## Goals

1. **Persistence**: Documents survive server restarts
2. **Efficiency**: Minimize disk I/O through caching and page management
3. **MVCC Integration**: Support version chains for snapshot isolation
4. **Scalability**: Handle datasets larger than available memory
5. **Compatibility**: Work with existing storage engine infrastructure

## Architecture Overview

```
┌──────────────────────────────────────────────────────┐
│              Collection Layer                         │
│  (CRUD operations, document cache)                    │
└──────────────────────────────────────────────────────┘
                        ↓
┌──────────────────────────────────────────────────────┐
│           Document Page Manager                       │
│  (slotted pages, document addressing)                 │
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

## Document Storage Format on Pages

### Slotted Page Structure

We use a **slotted page** structure to store variable-length documents efficiently. This allows:
- Efficient space utilization for variable-length documents
- Fast document lookup by slot ID
- In-place updates when possible
- Easy compaction to reclaim fragmented space

#### Page Layout

```
┌────────────────────────────────────────────────────────┐
│                Page Header (16 bytes)                   │  ← Storage engine header
├────────────────────────────────────────────────────────┤
│            Slotted Page Header (12 bytes)              │
│  - SlotCount (2 bytes)                                 │
│  - FreeSpaceStart (2 bytes)                            │
│  - FreeSpaceEnd (2 bytes)                              │
│  - FragmentedSpace (2 bytes)                           │
│  - Flags (2 bytes)                                     │
│  - Reserved (2 bytes)                                  │
├────────────────────────────────────────────────────────┤
│              Slot Directory (grows downward)            │
│  Slot 0: [Offset (2B)][Length (2B)][Flags (1B)]       │  ← 5 bytes per slot
│  Slot 1: [Offset (2B)][Length (2B)][Flags (1B)]       │
│  Slot 2: [Offset (2B)][Length (2B)][Flags (1B)]       │
│  ...                                                   │
│                                                        │
│               Free Space (middle)                      │
│                                                        │
├────────────────────────────────────────────────────────┤
│          Document Data (grows upward from bottom)      │
│  Document N: [BSON data]                               │
│  ...                                                   │
│  Document 1: [BSON data]                               │
│  Document 0: [BSON data]                               │
└────────────────────────────────────────────────────────┘

Total Page Size: 4096 bytes
Available Space: 4096 - 16 (page header) - 12 (slotted header) = 4068 bytes
```

### Slotted Page Header (12 bytes)

| Field | Size | Description |
|-------|------|-------------|
| SlotCount | 2 bytes | Number of slots in the slot directory |
| FreeSpaceStart | 2 bytes | Offset where slot directory ends (grows down) |
| FreeSpaceEnd | 2 bytes | Offset where document data starts (grows up) |
| FragmentedSpace | 2 bytes | Total bytes of fragmented space (deleted slots) |
| Flags | 2 bytes | Page-level flags (e.g., needs compaction) |
| Reserved | 2 bytes | Reserved for future use |

### Slot Directory Entry (5 bytes per slot)

| Field | Size | Description |
|-------|------|-------------|
| Offset | 2 bytes | Byte offset to document data in page (0 = deleted) |
| Length | 2 bytes | Length of document in bytes |
| Flags | 1 byte | Slot flags (deleted, overflow chain, etc.) |

#### Slot Flags (1 byte)

- Bit 0: Deleted (1 = slot is deleted, 0 = active)
- Bit 1: Overflow (1 = document spans multiple pages)
- Bit 2: Updated (1 = slot was updated, old version in overflow)
- Bits 3-7: Reserved

### Document Serialization

Documents are serialized using the existing BSON encoding (see `pkg/document/bson.go`):

```
Document Format (BSON):
┌────────────────────────────────────────┐
│ Size (4 bytes, int32)                  │  ← Total document size
├────────────────────────────────────────┤
│ Element 1:                             │
│   - Type (1 byte)                      │
│   - Key (C-string, null-terminated)    │
│   - Value (type-dependent)             │
├────────────────────────────────────────┤
│ Element 2: ...                         │
├────────────────────────────────────────┤
│ ...                                    │
├────────────────────────────────────────┤
│ Terminator (0x00)                      │
└────────────────────────────────────────┘
```

**Note**: The existing `Encoder` and `Decoder` in `pkg/document/bson.go` already implement this format and will be reused.

## Document Addressing Scheme

### DocumentID (Logical Identifier)

Documents are uniquely identified by:
```go
type DocumentID struct {
    CollectionID uint32  // Unique collection identifier
    PageID       uint32  // Page containing the document (PageID)
    SlotID       uint16  // Slot number within the page
    Reserved     uint16  // Reserved for future use (e.g., partition ID)
}
```

Total size: 12 bytes

This provides:
- **CollectionID**: Supports multiple collections in the same database
- **PageID**: Direct page lookup without index scan
- **SlotID**: Direct slot access within page
- **4 billion collections** (2^32)
- **4 billion pages per collection** (2^32)
- **65,536 slots per page** (2^16, though typically limited to ~800 by space)

### ObjectID to DocumentID Mapping

The `_id` field (ObjectID) maps to DocumentID through the primary index:
- Primary `_id` index: `ObjectID → DocumentID`
- Secondary indexes: `IndexKey → DocumentID`

## Free Space Tracking Within Pages

### Free Space Management

Each slotted page tracks:
1. **Contiguous Free Space**: Between FreeSpaceStart and FreeSpaceEnd
2. **Fragmented Space**: Sum of deleted slot spaces

```
Contiguous Free Space = FreeSpaceEnd - FreeSpaceStart
Total Free Space = Contiguous Free Space + Fragmented Space
```

### Compaction Trigger

A page needs compaction when:
```
FragmentedSpace > (PageSize * 0.25)  // More than 25% fragmented
```

Compaction process:
1. Create new slot directory
2. Copy active documents to end of page (defragmented)
3. Update slot offsets
4. Reset FreeSpaceStart and FreeSpaceEnd
5. Set FragmentedSpace = 0

## Document Size Limits and Overflow Handling

### Size Limits

- **Maximum document size without overflow**: ~4,000 bytes (single page)
- **Maximum document size with overflow**: 16 MB (MongoDB-compatible limit)

### Overflow Page Chain

For documents larger than a single page:

```
Primary Page (has slot):
┌────────────────────────────────────────┐
│ Slot N:                                │
│   Offset → Overflow Header             │
│   Length = overflow header size        │
│   Flags = OVERFLOW bit set             │
├────────────────────────────────────────┤
│ Overflow Header (16 bytes):            │
│   - TotalSize (4 bytes)                │
│   - FirstPageID (4 bytes)              │
│   - PageCount (2 bytes)                │
│   - Reserved (6 bytes)                 │
└────────────────────────────────────────┘
          ↓
Overflow Page 1:
┌────────────────────────────────────────┐
│ Page Type: PageTypeOverflow            │
├────────────────────────────────────────┤
│ Overflow Page Header (8 bytes):        │
│   - NextPageID (4 bytes)               │
│   - DataLength (2 bytes)               │
│   - Reserved (2 bytes)                 │
├────────────────────────────────────────┤
│ Document Data (part 1)                 │
│ ...                                    │
└────────────────────────────────────────┘
          ↓
Overflow Page 2:
┌────────────────────────────────────────┐
│ Overflow Page Header                   │
│   - NextPageID = 0 (last page)         │
├────────────────────────────────────────┤
│ Document Data (part 2)                 │
│ ...                                    │
└────────────────────────────────────────┘
```

### Overflow Operations

**Write**:
1. Serialize document to BSON
2. If size > single page capacity:
   - Allocate overflow pages
   - Write overflow header in primary page
   - Chain overflow pages
   - Write data chunks to overflow pages

**Read**:
1. Read slot in primary page
2. If OVERFLOW flag set:
   - Read overflow header
   - Read all overflow pages in chain
   - Reassemble document from chunks
   - Deserialize BSON

## Free Page Management

### Free Page List

Track deallocated pages for reuse:

```go
type FreePageList struct {
    HeadPageID PageID  // First free page
    PageCount  uint32  // Number of free pages
}
```

Free pages form a linked list:
```
FreePageList → Page 7 → Page 12 → Page 15 → nil
```

Each free page stores:
```
┌────────────────────────────────────────┐
│ Page Type: PageTypeFreeList            │
├────────────────────────────────────────┤
│ NextFreePageID (4 bytes)               │
└────────────────────────────────────────┘
```

### Page Allocation Algorithm

```
AllocatePage():
  if FreePageList is not empty:
    pop page from free list
    return page
  else:
    extend data file
    return new page
```

### Page Deallocation

```
DeallocatePage(pageID):
  clear page data
  set page type = PageTypeFreeList
  set NextFreePageID = current head
  update FreePageList head = pageID
  increment PageCount
```

## MVCC Integration

### Document Versioning

Each document can have multiple versions stored on disk:

```
Current Version (in slotted page):
┌────────────────────────────────────────┐
│ Document with TxnID=105                │
│ {_id: 1, name: "Alice", age: 30}       │
└────────────────────────────────────────┘
          ↓ (has older version pointer)
Old Version (in overflow/separate page):
┌────────────────────────────────────────┐
│ Document with TxnID=100                │
│ {_id: 1, name: "Alice", age: 25}       │
└────────────────────────────────────────┘
```

### Version Chain Storage

Option 1: **In-Place Updates with Overflow Chain** (Recommended)
- Current version: In primary slot
- Old versions: In overflow pages (linked list)
- Benefits: Fast current version access, clear separation

Option 2: **Version Slots in Same Page**
- All versions in same page if space allows
- Benefits: Locality for recent versions
- Drawbacks: More complex page management

**Decision**: Use Option 1 for simplicity and performance.

### Version Metadata

Add version metadata to each document on disk:
```
Version Header (12 bytes):
┌────────────────────────────────────────┐
│ CreatedTxnID (8 bytes)                 │
│ DeletedTxnID (8 bytes, 0 = not deleted)│
│ PrevVersionPageID (4 bytes)            │  ← Overflow page with old version
│ PrevVersionSlotID (2 bytes)            │
│ Reserved (2 bytes)                     │
└────────────────────────────────────────┘
```

This header is prepended to the BSON document data in each slot.

### Garbage Collection

Old versions are cleaned up when no active transaction needs them:

```
GarbageCollect():
  minActiveTxn = min(all active transaction IDs)
  for each document with old versions:
    for each version in chain:
      if version.DeletedTxnID != 0 and version.DeletedTxnID < minActiveTxn:
        deallocate version page/slot
        remove from version chain
```

Run periodically (e.g., every 60 seconds) or on checkpoint.

## Performance Optimizations

### 1. Document Cache (LRU)

Two-level caching:
- **Page Cache (Buffer Pool)**: Already exists, caches pages
- **Document Cache**: Cache deserialized documents

```go
type DocumentCache struct {
    cache     map[DocumentID]*Document
    lruList   *list.List
    maxSize   int
    mu        sync.RWMutex
}
```

Benefits:
- Avoid repeated BSON deserialization
- Fast repeated reads of same document
- Configurable size (default: 10,000 documents)

### 2. Clustering by Primary Key

Store documents in approximate `_id` order:
- Sequential inserts → nearby pages
- Range scans → sequential page reads
- Better buffer pool hit rate

### 3. Prefetching

For range scans and sequential access:
```go
// Read ahead next N pages
prefetchPages(startPageID, count)
```

Strategies:
- **Sequential scan**: Prefetch next 8 pages
- **Index range scan**: Prefetch pages for next N DocumentIDs
- **Adaptive**: Adjust based on access patterns

### 4. Write Buffering

Batch small writes:
- Buffer updates in memory
- Flush to WAL periodically (e.g., every 100ms)
- Group commit: Write multiple transactions' WAL entries together

### 5. Page Compression (Optional, Future)

Compress pages before writing to disk:
- Use Snappy or LZ4 (fast compression)
- Trade CPU for disk I/O and space
- Compress in buffer pool on eviction

## Space Efficiency Analysis

### Overhead per Document

Assuming average document size: 500 bytes

| Component | Size | Percentage |
|-----------|------|------------|
| BSON document data | 500 bytes | 96.2% |
| Version header | 12 bytes | 2.3% |
| Slot directory entry | 5 bytes | 1.0% |
| Amortized page header | ~2 bytes | 0.4% |
| **Total** | **519 bytes** | **100%** |

**Overhead: ~3.8%** (very efficient)

### Documents per Page

Available space per page: 4068 bytes (4096 - 28 for headers)

For 500-byte documents:
- Document + version header: 512 bytes
- Slot entry: 5 bytes
- Total per document: 517 bytes
- **Documents per page**: ~7-8 documents

For 100-byte documents:
- Document + version header: 112 bytes
- Slot entry: 5 bytes
- Total per document: 117 bytes
- **Documents per page**: ~34 documents

### Fragmentation

Expected fragmentation after mixed workload:
- **Internal fragmentation**: 5-15% (slot deletions)
- **Compaction threshold**: 25%
- **Post-compaction**: <5%

## Implementation Phases

This design supports the phased implementation plan in TODO.md:

### Phase 1: Design & Planning ✓ (This Document)
- Document storage format on pages ✓
- Document addressing scheme ✓
- Free space tracking ✓
- Document size limits and overflow handling ✓

### Phase 2: Storage Layer Enhancements
- Implement slotted page structure (`slotted_page.go`)
- Implement free page management (extend `DiskManager`)
- Implement document serialization helpers (wrapper for existing BSON)
- Extend DiskManager with AllocatePage/FreePage

### Phase 3: Collection Layer Refactoring
- Replace in-memory map with disk-based storage
- Implement DocumentID addressing
- Add document cache (LRU)
- Update CRUD operations for disk I/O

### Phase 4: Index Layer Refactoring
- Persist B+ tree nodes to pages
- Update index values to store DocumentID instead of in-memory pointers
- Implement node cache

### Phase 5: Transaction & MVCC Integration
- Store version chains on disk
- Implement disk-based garbage collection
- Add version metadata to documents

## Testing Strategy

### Unit Tests
- Slotted page operations (insert, delete, update, compact)
- Free page list management
- Document serialization/deserialization
- Overflow page chain operations
- Version chain operations

### Integration Tests
- Insert → Close → Reopen → Verify data persists
- CRUD operations with disk storage
- Concurrent reads and writes
- Large document handling (overflow)
- Page compaction under load

### Performance Tests
- Insert throughput (disk vs. in-memory)
- Query latency (disk vs. in-memory)
- Cache hit rate analysis
- Mixed workload (read-heavy, write-heavy, balanced)

### Stress Tests
- Fill disk to capacity
- Crash during write → Verify recovery
- Concurrent transactions with version chains
- Fragmentation and compaction under load

## Migration Plan

### Backwards Compatibility

**Option 1: New Data Directory**
- Users create new data directory for disk-based storage
- Old in-memory mode still available (flag: `--memory-mode`)
- No migration needed, clean start

**Option 2: Automatic Migration**
- On first run with new version, detect old format
- Migrate in-memory data to disk format
- Create backup before migration

**Recommendation**: Use Option 1 initially, add Option 2 later if needed.

### Migration Tool (Future)

For users who need to migrate existing data:
```bash
laura-migrate --input=/old/data --output=/new/data
```

Steps:
1. Read collections from old format
2. Create new disk-based collections
3. Insert documents with disk storage
4. Rebuild indexes
5. Verify data integrity

## Configuration Options

### Storage Configuration

```go
type StorageConfig struct {
    // Existing
    DataDir        string
    BufferPoolSize int

    // New
    DocumentCacheSize int     // Number of documents to cache (default: 10000)
    PageCompactionThreshold float64  // Fragmentation % to trigger compaction (default: 0.25)
    PrefetchSize int          // Number of pages to prefetch (default: 8)
    MaxDocumentSize int       // Max document size in bytes (default: 16MB)
    EnableCompression bool    // Enable page compression (default: false)
}
```

### Tuning for Workloads

**Read-Heavy Workload**:
- Increase BufferPoolSize (more page cache)
- Increase DocumentCacheSize (more document cache)
- Increase PrefetchSize (more aggressive prefetching)

**Write-Heavy Workload**:
- Smaller BufferPoolSize (faster flush)
- Lower PageCompactionThreshold (keep pages tight)
- Enable write buffering

**Memory-Constrained**:
- Smaller BufferPoolSize and DocumentCacheSize
- Enable compression
- More aggressive checkpointing

## Performance Expectations

### Disk-Based vs. In-Memory

| Operation | In-Memory | Disk (Cold) | Disk (Cached) |
|-----------|-----------|-------------|---------------|
| Insert | 50-100µs | 300-500µs | 100-200µs |
| Find by ID | 20-50µs | 500-1000µs | 50-100µs |
| Range Scan (100 docs) | 500-1000µs | 5-10ms | 1-2ms |
| Update | 50-100µs | 500-1000µs | 150-300µs |
| Delete | 50-100µs | 300-500µs | 100-200µs |

**Cache Hit Rate Target**: >95% for typical workloads

### Scalability

- **Dataset size**: Limited only by disk space
- **Collection size**: Up to 16TB per collection (4B pages × 4KB)
- **Document count**: Trillions (limited by 64-bit DocumentID space)
- **Concurrent connections**: Same as current (limited by goroutines)

## Summary

This design provides:

✅ **Persistence**: Documents stored on disk, survive restarts
✅ **Efficiency**: Slotted pages minimize fragmentation, caching reduces I/O
✅ **MVCC Compatible**: Version chains stored on disk
✅ **Scalable**: Handle datasets larger than memory
✅ **Production-Ready**: WAL, crash recovery, comprehensive testing

**Key Design Decisions**:
1. **Slotted Page Structure**: Efficient variable-length storage
2. **DocumentID Addressing**: Direct page/slot lookup
3. **Two-Level Caching**: Buffer pool + document cache
4. **Overflow Pages**: Support large documents (up to 16MB)
5. **Version Chain in Overflow**: Clean separation of current/old versions
6. **Free Page List**: Efficient page reuse

**Next Steps**: Proceed to Phase 2 implementation (storage layer enhancements).
