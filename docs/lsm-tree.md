# LSM Tree Storage

## Overview

LauraDB now includes an LSM (Log-Structured Merge) tree storage option, optimized for write-heavy workloads. LSM trees achieve high write throughput by buffering writes in memory and flushing sorted data to disk in sequential I/O operations.

## Key Components

### 1. MemTable (In-Memory)
- **Data Structure**: Skip list for O(log n) operations
- **Purpose**: Accept all writes with minimal latency
- **Size**: Configurable (default: 4MB)
- **Behavior**: When full, becomes immutable and triggers flush

### 2. SSTable (Sorted String Table)
- **Format**: Immutable sorted file on disk
- **Structure**:
  - Data section: Sorted key-value entries
  - Sparse index: Every Nth key for binary search
  - Bloom filter: Probabilistic membership test
  - Footer: Metadata (min/max keys, entry count)
- **Benefits**: Sequential writes, compression-friendly

### 3. Bloom Filter
- **Purpose**: Avoid disk reads for non-existent keys
- **Size**: ~10 bits per key
- **False Positive Rate**: ~1-3%
- **False Negatives**: Never (guaranteed)

### 4. Compaction
- **Trigger**: >4 SSTables accumulated
- **Strategy**: Merge oldest SSTables
- **Benefits**:
  - Removes deleted entries (tombstones)
  - Deduplicates keys
  - Reduces file count
  - Improves read performance

## Usage

```go
package main

import (
	"github.com/mnohosten/laura-db/pkg/lsm"
)

func main() {
	// Create LSM tree
	config := lsm.DefaultConfig("./data")
	config.MemTableSize = 4 * 1024 * 1024 // 4MB
	config.IndexInterval = 100             // Index every 100 keys

	tree, err := lsm.NewLSMTree(config)
	if err != nil {
		panic(err)
	}
	defer tree.Close()

	// Put key-value pairs
	tree.Put([]byte("user:1"), []byte("alice"))
	tree.Put([]byte("user:2"), []byte("bob"))

	// Get value
	value, found, err := tree.Get([]byte("user:1"))
	if found {
		println(string(value)) // "alice"
	}

	// Delete key
	tree.Delete([]byte("user:2"))

	// Flush pending writes
	tree.Flush()

	// Get statistics
	stats := tree.Stats()
	println(stats["num_sstables"].(int))
}
```

## Performance Characteristics

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Write | O(log n) | MemTable skip list insertion |
| Read (hit) | O(log n) | MemTable or index lookup |
| Read (miss) | O(k) | Check k SSTables (bloom filters help) |
| Flush | O(n log n) | Sort and write MemTable |
| Compaction | O(n log k) | k-way merge of SSTables |

## Write Amplification

LSM trees have write amplification due to compaction:
- Data written once to MemTable
- Flushed to L0 SSTable
- Potentially rewritten during compaction

**Write Amplification**: ~2-3x for typical workloads

## Read Amplification

Reads may need to check multiple SSTables:
- MemTable: 1 lookup
- Immutables: 0-N lookups
- SSTables: 0-M lookups (bloom filters reduce)

**Read Amplification**: 1-10 lookups depending on compaction lag

## Space Amplification

Multiple versions of data exist until compaction:
- Active data
- Deleted data (tombstones)
- Old versions in un-compacted SSTables

**Space Amplification**: ~1.5-2x until compaction runs

## Comparison with B+ Tree

| Aspect | LSM Tree | B+ Tree |
|--------|----------|---------|
| Write Speed | ★★★★★ | ★★★☆☆ |
| Read Speed | ★★★☆☆ | ★★★★★ |
| Space Efficiency | ★★★☆☆ | ★★★★☆ |
| Compression | ★★★★★ | ★★☆☆☆ |
| Write Amp | High (2-3x) | Low (1x) |
| Use Case | Write-heavy | Balanced |

## Best Use Cases

LSM trees excel at:
1. **Time-series data**: Append-only writes
2. **Logging systems**: High write throughput
3. **Metrics collection**: Continuous data ingestion
4. **Event sourcing**: Write-heavy event streams
5. **Cache backends**: Frequent updates

## Implementation Details

### Skip List (MemTable)
- Probabilistic data structure
- Expected O(log n) search/insert
- Better cache locality than trees
- Level probability: 0.25
- Max levels: 16

### SSTable Format
```
[Entry 1][Entry 2]...[Entry N][Footer][Footer Size]
```

Entry format:
```
keyLen(4) | key | valueLen(4) | value | timestamp(8) | deleted(1)
```

Footer format:
```
numEntries(4) | minKey | maxKey | indexEntries | bloomFilter
```

### Compaction Strategy
- **Level 0**: Unsorted SSTables from flushes
- **Compaction trigger**: >4 L0 SSTables
- **Merge strategy**: Oldest N SSTables
- **Output**: Single merged SSTable

## Configuration

```go
type Config struct {
	Dir           string // Data directory
	MemTableSize  int64  // Max MemTable size (default: 4MB)
	IndexInterval int    // Sparse index density (default: 100)
}
```

## Limitations

1. **No range scans**: Current implementation optimized for point queries
2. **Simple compaction**: Single-level compaction (not tiered)
3. **No compression**: SSTables stored uncompressed
4. **No WAL**: MemTable data lost on crash (before flush)

## Future Enhancements

- [ ] Leveled compaction strategy (LevelDB style)
- [ ] SSTable compression (Snappy/Zstd)
- [ ] Write-ahead log for durability
- [ ] Range query support
- [ ] Concurrent compaction
- [ ] Compaction admission control

## References

- ["The Log-Structured Merge-Tree (LSM-Tree)"](https://www.cs.umb.edu/~poneil/lsmtree.pdf) - O'Neil et al. (1996)
- [LevelDB Implementation](https://github.com/google/leveldb)
- [RocksDB Tuning Guide](https://github.com/facebook/rocksdb/wiki/RocksDB-Tuning-Guide)
