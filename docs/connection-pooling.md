# Connection Pooling in LauraDB

## Overview

LauraDB implements two types of pooling to optimize performance and resource usage:

1. **Session Pool** - Reuses transactional session objects using `sync.Pool`
2. **Worker Pool** - Manages background task execution with configurable workers

## Session Pool

### Purpose

The Session Pool reduces allocation overhead by reusing Session objects for transactional operations. Since sessions maintain transaction state, operations buffers, and snapshot caches, creating them from scratch for each transaction can be expensive.

### Architecture

```
┌──────────────────┐
│   SessionPool    │
├──────────────────┤
│ sync.Pool        │ ◄── Reusable Session objects
│ Database *       │
└──────────────────┘
         │
         │ Get() / Put()
         ▼
┌──────────────────┐
│    Session       │
├──────────────────┤
│ txn              │ ◄── Fresh MVCC transaction
│ operations []    │ ◄── Reset on return to pool
│ collections {}   │ ◄── Cleared
│ snapshotDocs {}  │ ◄── Cleared
└──────────────────┘
```

### Key Features

- **Automatic Reset**: Session state is cleared when returned to the pool
- **Transaction Management**: New MVCC transaction created on Get()
- **Capacity Pre-allocation**: Operations slice pre-allocated with capacity 16
- **Zero GC Pressure**: Maps and slices reused by clearing instead of reallocating

### Usage

#### Basic Get/Put Pattern

```go
pool := NewSessionPool(db)

// Get session from pool
session := pool.Get()
defer pool.Put(session) // Always return to pool

// Use session for operations
id, err := session.InsertOne("users", map[string]interface{}{
    "name": "Alice",
    "age": int64(30),
})

// Commit transaction
if err := session.CommitTransaction(); err != nil {
    session.AbortTransaction()
    return err
}
```

#### Convenience Method

```go
pool := NewSessionPool(db)

// Automatic lifecycle management
err := pool.WithTransactionPooled(func(session *Session) error {
    // Insert documents
    _, err := session.InsertOne("products", map[string]interface{}{
        "name": "Widget",
        "price": int64(100),
    })
    if err != nil {
        return err // Automatically aborted
    }

    _, err = session.InsertOne("products", map[string]interface{}{
        "name": "Gadget",
        "price": int64(200),
    })
    return err // Automatically committed
})
```

### Performance

Based on benchmarks:

| Operation | Pooled | Direct | Improvement |
|-----------|--------|--------|-------------|
| Get/Put | 106 ns/op | 135 ns/op | **1.27x faster** |
| Get/Put (concurrent) | 405 ns/op | N/A | Highly concurrent |
| Memory | 208 B/op, 3 allocs/op | N/A | Low allocation |

**Key Insight**: Session pooling reduces allocation overhead by ~22% and is safe for concurrent use.

### Thread Safety

- **Concurrent Get/Put**: `sync.Pool` is thread-safe for concurrent Get/Put operations
- **Session Usage**: Each session should only be used by ONE goroutine at a time
- **Commit/Abort**: Must be called before returning session to pool

### Best Practices

1. **Always use defer**: Ensure Put() is called even if transaction fails
   ```go
   session := pool.Get()
   defer pool.Put(session)
   ```

2. **Commit or Abort before Put**: Don't return active transactions to pool
   ```go
   if err := session.CommitTransaction(); err != nil {
       session.AbortTransaction() // Cleanup before Put
   }
   ```

3. **Use WithTransactionPooled**: Simplifies lifecycle management
   ```go
   err := pool.WithTransactionPooled(func(s *Session) error {
       // Your transactional code here
       return nil
   })
   ```

4. **Don't retain references**: Once Put() is called, don't use the session
   ```go
   session := pool.Get()
   // ... use session ...
   pool.Put(session)
   // DON'T use session here! It may be reused by another goroutine
   ```

## Worker Pool

### Purpose

The Worker Pool provides controlled concurrency for background tasks such as:
- TTL cleanup
- Index building
- Defragmentation
- Batch operations
- Background compaction

### Architecture

```
┌─────────────────────────────────────────┐
│           WorkerPool                    │
├─────────────────────────────────────────┤
│ taskQueue: chan Task (buffered)         │
│ numWorkers: int                         │
│ ctx: context.Context                    │
│ wg: sync.WaitGroup                      │
│ stats: atomic counters                  │
└─────────────────────────────────────────┘
          │         │         │         │
          ▼         ▼         ▼         ▼
    ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
    │Worker 1│ │Worker 2│ │Worker 3│ │Worker 4│
    └────────┘ └────────┘ └────────┘ └────────┘
         │         │         │         │
         └─────────┴─────────┴─────────┘
                    │
                    ▼
            ┌──────────────┐
            │ Task.Execute()│
            └──────────────┘
```

### Key Features

- **Goroutine Pooling**: Fixed number of worker goroutines
- **Buffered Queue**: Handles bursts without blocking
- **Graceful Shutdown**: Wait for active tasks or drain queue
- **Statistics**: Track tasks submitted, active, and completed
- **Context-based Cancellation**: Clean shutdown via context

### Configuration

```go
config := &WorkerPoolConfig{
    NumWorkers: 4,    // Number of worker goroutines
    QueueSize:  100,  // Buffer size for task queue
}
pool := NewWorkerPool(config)
defer pool.Shutdown()
```

**Default Configuration**:
- **NumWorkers**: 4 (good for I/O-bound tasks)
- **QueueSize**: 100 (handles moderate bursts)

### Usage Patterns

#### Submit and Forget

```go
pool := NewWorkerPool(&WorkerPoolConfig{
    NumWorkers: 4,
    QueueSize:  100,
})
defer pool.Shutdown()

// Submit task (non-blocking if queue has space)
submitted := pool.SubmitFunc(func() error {
    // Background work here
    cleanupExpiredDocuments()
    return nil
})

if !submitted {
    // Queue full or pool shutting down
    log.Println("Failed to submit task")
}
```

#### Blocking Submit

```go
// Block until task is queued or pool shuts down
submitted := pool.SubmitBlocking(TaskFunc(func() error {
    rebuildIndex()
    return nil
}))
```

#### Custom Task Type

```go
type DefragmentTask struct {
    collectionName string
}

func (t *DefragmentTask) Execute() error {
    // Implement Task interface
    return defragmentCollection(t.collectionName)
}

pool.Submit(&DefragmentTask{collectionName: "users"})
```

### Shutdown Modes

#### Immediate Shutdown

```go
// Stop accepting new tasks, wait for active tasks
pool.Shutdown()
```

#### Drain and Shutdown

```go
// Process all queued tasks, then shutdown
pool.ShutdownAndDrain()
```

### Performance

Based on benchmarks:

| Operation | Performance | Notes |
|-----------|-------------|-------|
| Submit | 95 ns/op | Task submission overhead |
| Execute | 210 ns/op | End-to-end task execution |
| Concurrent Submit | 51 ns/op | Multiple goroutines submitting |
| Memory | 0 B/op, 0 allocs/op | Zero allocation for submission |

**Comparison with Raw Goroutines**:
- Worker Pool: 83 ns/op
- Raw Goroutines: 134 ns/op
- **1.62x faster** due to goroutine reuse

### Worker Scaling

Benchmark results for different worker counts (processing simple tasks):

| Workers | Time per Task |
|---------|--------------|
| 1 | 48 ns/op |
| 2 | 77 ns/op |
| 4 | 90 ns/op |
| 8 | 111 ns/op |
| 16 | 103 ns/op |
| 32 | 122 ns/op |

**Recommendation**: Use 4-8 workers for most workloads. More workers add overhead without proportional benefit for CPU-bound tasks.

### Queue Sizing

Benchmark results for different queue sizes:

| Queue Size | Time per Task |
|------------|--------------|
| 10 | 88 ns/op |
| 100 | 102 ns/op |
| 1,000 | 95 ns/op |
| 10,000 | 95 ns/op |

**Recommendation**: Queue size has minimal impact on performance. Use 100-1000 for most cases.

### Statistics

Monitor pool health with stats:

```go
stats := pool.Stats()
fmt.Printf("Workers: %d\n", stats.NumWorkers)
fmt.Printf("Total submitted: %d\n", stats.TasksTotal)
fmt.Printf("Currently active: %d\n", stats.TasksActive)
fmt.Printf("Completed: %d\n", stats.TasksDone)
fmt.Printf("Queued: %d\n", stats.QueuedTasks)
```

### Best Practices

1. **Choose appropriate worker count**
   - **CPU-bound**: NumCPUs or NumCPUs * 2
   - **I/O-bound**: NumCPUs * 4 to 8
   - **Mixed**: Start with 4, tune based on metrics

2. **Size the queue appropriately**
   - Small queue (10-50): Backpressure on submission
   - Medium queue (100-500): Good for typical workloads
   - Large queue (1000+): Burst handling, more memory

3. **Handle submission failures**
   ```go
   if !pool.Submit(task) {
       // Queue full or shutting down
       // Log, retry, or handle synchronously
   }
   ```

4. **Use graceful shutdown**
   ```go
   // Allow in-flight tasks to complete
   defer pool.Shutdown()
   ```

5. **Monitor statistics**
   ```go
   if pool.IsFull() {
       log.Warn("Worker pool queue is full")
   }
   ```

## Integration with LauraDB

### HTTP Server

The HTTP server can benefit from both pools:

```go
// In pkg/server/server.go
type Server struct {
    db          *database.Database
    sessionPool *database.SessionPool
    workerPool  *database.WorkerPool
}

func New(config *Config, db *database.Database) *Server {
    return &Server{
        db:          db,
        sessionPool: database.NewSessionPool(db),
        workerPool: database.NewWorkerPool(&database.WorkerPoolConfig{
            NumWorkers: 8,
            QueueSize:  200,
        }),
    }
}

// In handler
func (s *Server) handleTransaction(w http.ResponseWriter, r *http.Request) {
    err := s.sessionPool.WithTransactionPooled(func(session *database.Session) error {
        // Handle multi-document transaction
        return nil
    })
    // ... response handling ...
}

// Background tasks
func (s *Server) scheduleCleanup() {
    s.workerPool.SubmitFunc(func() error {
        return s.db.CleanupExpired()
    })
}
```

### Database Operations

Use worker pool for long-running operations:

```go
// Background index build
func (db *Database) CreateIndexAsync(coll string, field string) error {
    return db.workerPool.SubmitFunc(func() error {
        return db.Collection(coll).CreateIndex(field, &IndexConfig{
            Unique: false,
        })
    })
}

// TTL cleanup
func (db *Database) startTTLCleanup() {
    ticker := time.NewTicker(60 * time.Second)
    go func() {
        for range ticker.C {
            db.workerPool.SubmitFunc(func() error {
                return db.cleanupExpiredDocuments()
            })
        }
    }()
}
```

## Testing

Both pools have comprehensive test coverage:

**Session Pool Tests** (10 tests):
- Basic Get/Put
- Transaction commit/rollback
- Concurrent usage (50 goroutines)
- Session reset
- Multiple operations
- Delete operations

**Worker Pool Tests** (13 tests):
- Basic submission
- Multiple tasks
- Concurrent submission
- Graceful shutdown
- Queue full handling
- Statistics
- High load (500 tasks)
- Race condition detection

**Benchmarks** (14 benchmarks):
- Get/Put performance
- Transaction throughput
- Worker scaling
- Queue scaling
- Memory allocation

## Performance Summary

| Metric | Session Pool | Worker Pool |
|--------|--------------|-------------|
| Overhead | 106 ns/op | 95 ns/op |
| Memory | 208 B/op | 0 B/op |
| Allocations | 3 allocs/op | 0 allocs/op |
| vs. Direct | 1.27x faster | 1.62x faster |
| Concurrency | Thread-safe | Thread-safe |

## Limitations

### Session Pool
- Sessions must be used by single goroutine at a time
- Transaction must be committed/aborted before returning to pool
- Update operations have limited support (known session limitation)

### Worker Pool
- Fixed number of workers (no dynamic scaling)
- Tasks should be idempotent (no automatic retry)
- No task prioritization (FIFO queue)
- Shutdown discards queued tasks (use ShutdownAndDrain for graceful)

## Future Enhancements

Potential improvements:

1. **Session Pool**
   - Metrics: track pool hit rate, session reuse count
   - Timeout: auto-abort transactions after timeout
   - Validation: detect leaked sessions

2. **Worker Pool**
   - Priority queues for important tasks
   - Dynamic worker scaling based on load
   - Task retry with exponential backoff
   - Task dependencies and ordering

3. **Integration**
   - Connection pool for network operations
   - Buffer pool for I/O operations
   - Request context pooling for HTTP handlers

## References

- Session implementation: `pkg/database/session.go`
- Session pool: `pkg/database/session_pool.go`
- Worker pool: `pkg/database/worker_pool.go`
- Tests: `pkg/database/session_pool_test.go`, `pkg/database/worker_pool_test.go`
- Benchmarks: `pkg/database/pool_bench_test.go`
