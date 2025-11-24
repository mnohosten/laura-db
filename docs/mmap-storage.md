# Memory-Mapped File Storage

LauraDB supports memory-mapped file (mmap) storage as an alternative to the standard disk manager. Memory-mapped files can provide better performance for read-heavy workloads by mapping file data directly into the process address space, reducing system calls.

## Overview

The `MmapDiskManager` implements the same interface as the standard `DiskManager` but uses memory-mapped files instead of traditional file I/O operations. This approach has several benefits:

1. **Reduced System Calls**: Page reads are direct memory accesses instead of `read()` system calls
2. **OS Page Cache Integration**: The OS automatically manages caching and prefetching
3. **Better Read Performance**: ~1.44x faster read operations compared to standard disk I/O
4. **Improved Mixed Workload Performance**: ~1.61x faster for 70% read / 30% write workloads

## When to Use Memory-Mapped Storage

Memory-mapped storage is ideal for:

- **Read-heavy workloads**: Applications that perform many more reads than writes
- **Large datasets**: When the working set is larger than available RAM
- **Sequential access patterns**: OS prefetching can significantly improve performance
- **Multi-threaded reads**: Multiple threads can read from the mmap region without blocking

Memory-mapped storage may not be ideal for:

- **Write-heavy workloads**: Standard disk I/O is ~1.6x faster for write operations
- **Very small files**: The overhead of mmap setup may outweigh benefits
- **Frequent sync operations**: Standard disk I/O sync is ~1.45x faster

## Architecture

### Memory-Mapped Region

The `MmapDiskManager` maintains a memory-mapped region that covers the entire database file:

```
┌─────────────────────────────────────────┐
│        Memory-Mapped Region             │
│  (mapped to process address space)      │
├─────────────────────────────────────────┤
│  Page 0  │  Page 1  │  ...  │  Page N  │
│  (4KB)   │  (4KB)   │  ...  │  (4KB)   │
└─────────────────────────────────────────┘
          ↕ (mmap syscall)
┌─────────────────────────────────────────┐
│         Database File on Disk            │
│            (data.db)                     │
└─────────────────────────────────────────┘
```

### Dynamic Expansion

The mmap region automatically expands as pages are allocated:

1. **Initial Size**: 256MB by default (configurable)
2. **Growth Size**: 64MB increments by default (configurable)
3. **Automatic Remapping**: When a page beyond the current region is accessed

### Page Operations

**Read Operation:**
```
1. Calculate offset: pageID * PageSize
2. Check if offset is within mmap region
3. Copy data directly from memory-mapped region
4. Deserialize page header and data
```

**Write Operation:**
```
1. Calculate offset: pageID * PageSize
2. Expand mmap region if needed
3. Serialize page data
4. Copy directly to memory-mapped region
5. OS handles flushing to disk
```

## Configuration

### Basic Configuration

```go
import "github.com/mnohosten/laura-db/pkg/storage"

// Use default configuration (256MB initial, 64MB growth)
dm, err := storage.NewMmapDiskManager("/path/to/data.db", nil)
if err != nil {
    log.Fatal(err)
}
defer dm.Close()
```

### Custom Configuration

```go
config := &storage.MmapConfig{
    InitialSize: 512 * 1024 * 1024,  // 512MB initial
    GrowthSize:  128 * 1024 * 1024,  // 128MB growth
}

dm, err := storage.NewMmapDiskManager("/path/to/data.db", config)
if err != nil {
    log.Fatal(err)
}
defer dm.Close()
```

## Memory Advise Hints

The `MmapDiskManager` provides methods to hint the OS about access patterns:

### Random Access Pattern

```go
// Hint that pages will be accessed randomly
dm.MadviseRandom()
```

This tells the OS:
- Don't prefetch sequential pages
- Optimize for random access
- Use less aggressive readahead

### Sequential Access Pattern

```go
// Hint that pages will be accessed sequentially
dm.MadviseSequential()
```

This tells the OS:
- Prefetch upcoming pages aggressively
- Can discard recently accessed pages
- Optimize for sequential scans

### Prefetch Specific Pages

```go
// Hint that pages 100-200 will be needed soon
dm.MadviseWillNeed(100, 200)
```

This triggers prefetching of specific page ranges into memory.

## Usage Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/mnohosten/laura-db/pkg/storage"
)

func main() {
    // Create mmap disk manager
    dm, err := storage.NewMmapDiskManager("./data/mydb.db", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer dm.Close()

    // Hint sequential access for bulk load
    dm.MadviseSequential()

    // Allocate and write pages
    for i := 0; i < 1000; i++ {
        pageID, err := dm.AllocatePage()
        if err != nil {
            log.Fatal(err)
        }

        page := storage.NewPage(pageID, storage.PageTypeData)
        page.LSN = uint64(i)
        copy(page.Data, []byte(fmt.Sprintf("Page %d data", i)))

        if err := dm.WritePage(page); err != nil {
            log.Fatal(err)
        }
    }

    // Switch to random access for queries
    dm.MadviseRandom()

    // Read pages
    page, err := dm.ReadPage(42)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Page 42 LSN: %d\n", page.LSN)

    // Sync to disk
    if err := dm.Sync(); err != nil {
        log.Fatal(err)
    }
}
```

## Performance Characteristics

### Benchmark Results

Tested on Apple M4 Max (macOS):

| Operation | Standard Disk I/O | Memory-Mapped | Speedup |
|-----------|------------------|---------------|---------|
| Read Page | 1078 ns/op | 748 ns/op | 1.44x |
| Write Page | 3089 ns/op | 4935 ns/op | 0.63x |
| Mixed (70% read) | 951 ns/op | 590 ns/op | 1.61x |
| Sync | 321 ns/op | 465 ns/op | 0.69x |

### Memory Usage

- **Memory-mapped region**: Allocated but may not be resident in RAM
- **OS managed**: The OS decides which pages to keep in memory
- **Virtual memory**: The full mmap size counts as virtual memory, not necessarily physical memory

### Scalability

- **Read scalability**: Excellent - multiple threads can read concurrently
- **Write scalability**: Good - writes are independent with proper locking
- **Large files**: Handles multi-GB files efficiently with dynamic expansion

## Implementation Details

### Thread Safety

The `MmapDiskManager` uses read-write locks:
- **Read operations**: Use read lock (allows concurrent reads)
- **Write operations**: Use write lock for expansion, read lock for actual writes
- **Stats operations**: Use read lock

### Error Handling

Common error scenarios:
- **File creation failure**: Ensure directory exists and has proper permissions
- **Mmap failure**: Check system mmap limits (`ulimit -v` on Unix)
- **Expansion failure**: Verify sufficient disk space

### Platform Support

Memory-mapped file support is implemented using:
- **Unix/Linux/macOS**: `syscall.Mmap`, `syscall.Munmap`
- **System calls**: `SYS_MSYNC`, `SYS_MADVISE`

The implementation uses direct syscalls for msync and madvise operations to ensure platform compatibility.

## Comparison with Standard Disk Manager

| Feature | Standard DiskManager | MmapDiskManager |
|---------|---------------------|-----------------|
| Read Performance | Good | Better (1.44x) |
| Write Performance | Better (1.6x) | Good |
| Mixed Workload | Good | Better (1.61x) |
| Memory Usage | Lower | Higher (virtual) |
| Setup Complexity | Simple | Moderate |
| OS Integration | Limited | Deep (page cache) |
| Large File Support | Good | Excellent |

## Best Practices

1. **Choose based on workload**: Use mmap for read-heavy, standard for write-heavy
2. **Configure appropriate initial size**: Avoid frequent expansions
3. **Use access hints**: MadviseRandom/Sequential can significantly improve performance
4. **Monitor virtual memory**: Mmap uses virtual address space
5. **Sync explicitly**: Call Sync() before critical checkpoints
6. **Test both**: Benchmark with your specific workload to determine which is better

## Limitations

1. **Virtual address space**: Very large databases (>256GB) may hit address space limits on 32-bit systems
2. **Write performance**: Slightly slower than standard disk I/O for write operations
3. **Sync performance**: Msync is slower than fsync for standard files
4. **Platform-specific**: Implementation uses Unix/Linux/macOS system calls

## Future Enhancements

Potential improvements:
- [ ] Read-only mmap mode (MAP_PRIVATE) for reader processes
- [ ] Huge page support (MAP_HUGETLB on Linux) for reduced TLB misses
- [ ] Lock-free concurrent access for different pages
- [ ] Async msync (MS_ASYNC) for better write throughput
- [ ] Windows support using CreateFileMapping/MapViewOfFile

## References

- [mmap(2) man page](https://man7.org/linux/man-pages/man2/mmap.2.html)
- [msync(2) man page](https://man7.org/linux/man-pages/man2/msync.2.html)
- [madvise(2) man page](https://man7.org/linux/man-pages/man2/madvise.2.html)
- [Virtual Memory in the Linux Kernel](https://www.kernel.org/doc/html/latest/admin-guide/mm/index.html)
