# MVCC (Multi-Version Concurrency Control)

## Overview

MVCC allows multiple transactions to access the database concurrently without blocking each other by maintaining multiple versions of data. Readers never block writers, and writers never block readers.

## Key Concepts

### Snapshot Isolation

Each transaction sees a consistent snapshot of the database as it existed when the transaction started.

```
Timeline:
T1: ─────[Start]────────────[Read X=10]──────[Commit]────
T2: ──────────[Start]──[Write X=20]──[Commit]──────────

T1 reads X=10 (its snapshot)
T2 writes X=20 (new version)
Both succeed without blocking!
```

### Version Chains

Each key maintains a linked list of versions, ordered by version number (most recent first).

```
Key "user:123"
    ↓
[V3: {age:31}] → [V2: {age:30}] → [V1: {age:29}] → null
 Created: T3      Created: T2      Created: T1
 Commit: 103      Commit: 102      Commit: 101
```

When reading, traverse the chain to find the first version visible to your snapshot.

## Architecture

```
┌──────────────────────────────────────────┐
│       Transaction Manager                 │
│  - Assign transaction IDs                 │
│  - Manage transaction lifecycle           │
│  - Coordinate commits/aborts              │
└──────────────────────────────────────────┘
               │
               ├──────────────────┐
               ▼                   ▼
┌────────────────────┐   ┌──────────────────┐
│   Version Store    │   │  Transactions    │
│  - Version chains  │   │  - Active txns   │
│  - Get/Put         │   │  - Write sets    │
│  - GC old versions │   │  - Read versions │
└────────────────────┘   └──────────────────┘
```

## Transaction Lifecycle

### 1. Begin

```go
txn := txnMgr.Begin()
// Assigns:
// - Unique transaction ID
// - Read version (current latest version)
// - Empty write set
```

### 2. Read (Snapshot Isolation)

```go
value, exists, err := txnMgr.Read(txn, "user:123")
```

**Read Algorithm**:
1. Check write set (read your own writes)
2. If not in write set, query version store
3. Find version where `version.Version <= txn.ReadVersion`
4. Return that version's value

**Example**:
```
Transaction T1 (ReadVersion: 100)
Version chain for "key":
  [V105: "new"] → [V98: "old"] → null

T1 reads → Returns "old" (V98)
V105 not visible because 105 > 100
```

### 3. Write

```go
err := txnMgr.Write(txn, "user:123", newValue)
```

**Write Algorithm**:
1. Add to transaction's write set
2. Does NOT immediately persist
3. Only local to this transaction

**Write Set**:
```go
txn.WriteSet = map[string]*VersionedValue{
    "user:123": {Value: {...}, CreatedBy: txn.ID},
    "user:456": {Value: {...}, CreatedBy: txn.ID},
}
```

### 4. Commit

```go
err := txnMgr.Commit(txn)
```

**Commit Algorithm**:
1. Acquire global lock (brief)
2. Assign commit version (atomic increment)
3. Apply write set to version store
4. Update each version's commit time
5. Mark transaction as committed
6. Release lock
7. Trigger garbage collection

**Example**:
```
T1 write set: {"user:123": {age: 31}}

Commit:
  - Assign version: 106
  - Add to version chain:
    [V106: {age:31}] → [V105: {age:30}] → ...
```

### 5. Abort

```go
err := txnMgr.Abort(txn)
```

Discards write set, no changes applied.

## Garbage Collection

Old versions that no transaction can see are periodically removed.

### Algorithm

1. Find minimum read version among all active transactions
2. Remove versions older than this minimum
3. Keep at least one version per key

**Example**:
```
Active transactions:
  T1: ReadVersion = 100
  T2: ReadVersion = 105

Minimum read version = 100

Version chain for "key":
  [V110] → [V107] → [V102] → [V95] → [V90]
                                ↑       ↑
                             Keep   Remove (< 100)

After GC:
  [V110] → [V107] → [V102]
```

### When to GC

- After each commit (async)
- Periodically (background thread)
- When version chains grow too long

## Concurrency Examples

### Example 1: Read Your Own Writes

```go
txn := txnMgr.Begin()

// Write
txnMgr.Write(txn, "x", 100)

// Read (sees own write)
val, _, _ := txnMgr.Read(txn, "x")
// val = 100 (from write set)

txnMgr.Commit(txn)
```

### Example 2: Snapshot Isolation

```go
// Initial: x = 10

// T1 starts
t1 := txnMgr.Begin() // ReadVersion: 100

// T2 starts, modifies x, commits
t2 := txnMgr.Begin()
txnMgr.Write(t2, "x", 20)
txnMgr.Commit(t2) // Creates V101

// T1 reads x
val, _, _ := txnMgr.Read(t1, "x")
// val = 10 (T1's snapshot, V100)

txnMgr.Commit(t1)
```

T1 doesn't see T2's changes because they happened after T1 started.

### Example 3: Non-Repeatable Reads Prevented

```go
t1 := txnMgr.Begin() // ReadVersion: 100

// First read
val1, _, _ := txnMgr.Read(t1, "x") // 10

// T2 modifies x
t2 := txnMgr.Begin()
txnMgr.Write(t2, "x", 20)
txnMgr.Commit(t2)

// Second read (same transaction)
val2, _, _ := txnMgr.Read(t1, "x") // Still 10!

// val1 == val2 (repeatable read)
```

## Write Conflicts

MVCC provides snapshot isolation, which can have write conflicts.

**Write Conflict Example**:
```
Initial: balance = 100

T1: Read balance (100)
T2: Read balance (100)
T1: Write balance = 150 (deposit $50)
T2: Write balance = 80  (withdraw $20)
T1: Commit ✓
T2: Commit ✓

Final: balance = 80 (Lost T1's update!)
```

**Solutions** (not yet implemented):
1. **First-Committer-Wins**: Abort T2 if T1 committed first
2. **Optimistic Locking**: Check version at commit time
3. **Serializable Snapshot Isolation (SSI)**: Detect conflicts

Our current implementation allows this for simplicity. In production, would add conflict detection.

## Memory Management

### Version Chain Growth

Problem: Long-running transactions prevent GC, causing chains to grow.

**Mitigation**:
- Transaction timeouts
- Version chain length limits
- Force-abort old transactions

### Write Set Size

Large transactions can have huge write sets.

**Mitigation**:
- Spill to disk if write set exceeds threshold
- Transaction size limits
- Batch smaller transactions

## Performance Characteristics

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Begin | O(1) | Atomic increment |
| Read | O(v) | v = versions to traverse |
| Write | O(1) | Add to write set |
| Commit | O(w) | w = write set size |
| Garbage Collect | O(k × v) | k = keys, v = avg versions |

**Typical Performance**:
- Begin/Write: Microseconds
- Read: Microseconds (few versions)
- Commit: Milliseconds (depends on write set size)

## Usage Example

```go
// Create transaction manager
txnMgr := mvcc.NewTransactionManager()

// Transaction 1: Transfer money
t1 := txnMgr.Begin()

// Read balances
aliceBalance, _, _ := txnMgr.Read(t1, "balance:alice")
bobBalance, _, _ := txnMgr.Read(t1, "balance:bob")

// Transfer $50
txnMgr.Write(t1, "balance:alice", aliceBalance.(int)-50)
txnMgr.Write(t1, "balance:bob", bobBalance.(int)+50)

// Commit
if err := txnMgr.Commit(t1); err != nil {
    txnMgr.Abort(t1)
}

// Transaction 2: Read balances (concurrent)
t2 := txnMgr.Begin()
balance, _, _ := txnMgr.Read(t2, "balance:alice")
// Sees snapshot depending on when T2 started
txnMgr.Commit(t2)
```

## Design Trade-offs

### Advantages

- **No read locks**: Readers never block
- **High concurrency**: Multiple writers don't block readers
- **Consistent snapshots**: No phantom reads
- **Simple rollback**: Just discard write set

### Disadvantages

- **Space overhead**: Multiple versions stored
- **GC complexity**: Need to clean old versions
- **Write conflicts**: Snapshot isolation anomalies
- **Memory usage**: Large write sets

## Comparison: MVCC vs 2PL

| Aspect | MVCC | 2PL (Two-Phase Locking) |
|--------|------|-------------------------|
| Readers block writers | No | Yes |
| Writers block readers | No | Yes |
| Deadlocks | Rare | Common |
| Space overhead | High | Low |
| Complexity | Higher | Lower |

MVCC trades space for concurrency.

## Future Enhancements

1. **Serializable Snapshot Isolation**: Detect write conflicts
2. **Index-based GC**: Track versions in indexes
3. **Vacuum process**: Background GC thread
4. **Version compression**: Compress old versions
5. **Distributed MVCC**: Multi-node transactions
6. **Read-only optimization**: Skip write set for read-only txns
