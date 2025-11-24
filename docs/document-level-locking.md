# Document-Level Locking - Design Analysis

## Overview

This document describes the research and findings from investigating document-level locking as an alternative to collection-level locking for improved concurrency in LauraDB.

## Current Implementation

LauraDB currently uses **collection-level locking** with a single `sync.RWMutex` per collection:
- Read operations (Find, FindOne, Count) acquire a read lock on the entire collection
- Write operations (InsertOne, UpdateOne, DeleteOne) acquire a write lock on the entire collection
- This creates contention when multiple goroutines try to access different documents simultaneously

## Proposed Document-Level Locking

The goal of document-level locking is to allow concurrent operations on different documents while still preventing race conditions.

### Design Approach

A document-level locking mechanism was prototyped with the following components:

1. **DocumentLockManager**: Manages fine-grained locks on individual documents
   - Uses lock striping (256 stripes) to reduce contention on the lock map itself
   - Each stripe manages a subset of document locks using FNV-1a hashing
   - Provides Lock/Unlock and RLock/RUnlock methods per document ID
   - LockMultiple/UnlockMultiple for deadlock prevention (locks in sorted order)

2. **Implementation**: `pkg/database/doc_lock.go`
   - 150 LOC implementing the lock manager
   - Comprehensive test coverage (11 tests, all passing)
   - Tests cover concurrent reads, exclusive writes, deadlock prevention, and cleanup

### Key Challenge: Map Concurrency

The fundamental challenge is that Go's built-in `map` type is not thread-safe. The current implementation uses `map[string]*document.Document` to store documents.

**Problem**: Even with document-level locks protecting individual documents, we still need synchronization when modifying the map structure itself (adding/removing entries).

**Attempted Solutions**:

1. **Hybrid Approach**: Use collection-level write lock for map modifications, document-level locks for updates
   - Problem: Inserts and deletes still require collection-level write locks, losing most of the concurrency benefit
   - Updates could use document locks, but this creates complexity with mixed locking strategies

2. **sync.Map**: Replace `map[string]*document.Document` with `sync.Map`
   - Problem: Requires significant refactoring throughout the codebase
   - Many operations iterate over all documents (e.g., Find with filters, Count, Aggregate)
   - sync.Map's Range() method holds a lock during iteration, potentially worse than collection-level locking
   - Performance characteristics differ from regular maps (optimized for specific access patterns)

3. **Sharded Maps**: Use multiple maps with separate locks
   - Problem: Adds complexity without addressing the fundamental issue
   - Operations spanning multiple shards still need coordination

## Performance Implications

Document-level locking would provide benefits primarily for:
- ✅ Concurrent updates to different documents (high write concurrency)
- ✅ Mixed read/write workloads on different documents

However, it would not help (and could hurt) for:
- ❌ Operations that scan all documents (Find, Count, Aggregate)
- ❌ Inserts and deletes (still need map-level synchronization)
- ❌ Operations on the same document (serialized anyway)

## Benchmarking Results

The document lock manager itself performs well:
- Lock acquisition: ~40-100ns overhead
- High concurrency (1000 operations, 100 documents): completes reliably
- Deadlock prevention works correctly with sorted locking

However, integration with Collection showed:
- ❌ Concurrent inserts cause "concurrent map writes" panic
- ❌ Hybrid locking (collection + document) negates benefits
- ✅ Original collection-level locking works correctly for all operations

## Recommendation

**Do not implement document-level locking** at this time for the following reasons:

1. **Fundamental Conflict**: Go maps require external synchronization for writes
2. **Limited Benefit**: Most operations (Find, Aggregate, Count) scan multiple documents anyway
3. **Complexity**: Mixing locking strategies or migrating to sync.Map adds significant complexity
4. **Alternative Solutions**: For high concurrency, consider:
   - Sharding at the database level (multiple collections)
   - Application-level caching
   - Read replicas for read-heavy workloads
   - Optimistic concurrency control (already implemented via MVCC transactions)

5. **MVCC Alternative**: LauraDB already has MVCC transactions which provide snapshot isolation without blocking reads. This is a better solution for most concurrency scenarios.

## Future Possibilities

Document-level locking could be reconsidered if:
1. Go adds a concurrent map implementation with fine-grained locking to the standard library
2. LauraDB migrates to a different storage engine (e.g., LSM tree, B+ tree on disk) where the in-memory map is just an index
3. The workload profile changes to be predominantly single-document updates (rare in practice)

## Files Created

- `pkg/database/doc_lock.go`: Document lock manager implementation (~150 LOC)
- `pkg/database/doc_lock_test.go`: Comprehensive tests (11 tests, all passing)
- `pkg/database/collection_concurrency_test.go`: Integration tests (attempted, revealed map concurrency issues)
- `docs/document-level-locking.md`: This design document

## Conclusion

While document-level locking is theoretically appealing, the practical implementation challenges and limited performance benefits do not justify the added complexity. LauraDB's current collection-level locking combined with MVCC transactions provides a good balance of correctness, simplicity, and performance for the target use cases.

The document lock manager code remains in the codebase as a reference implementation and can be enabled in the future if the storage architecture changes significantly.
