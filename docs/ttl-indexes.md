# TTL (Time-To-Live) Indexes

## Overview

TTL (Time-To-Live) indexes in LauraDB provide automatic expiration and deletion of documents after a specified time period. This is useful for managing temporary data such as:

- Session data
- Cache entries
- Temporary user tokens
- Event logs
- Ephemeral notifications
- Time-limited offers or content

## How TTL Indexes Work

A TTL index monitors a date/time field in documents and automatically deletes documents when:

```
current_time > (document_timestamp + ttl_seconds)
```

### Key Features

- **Automatic Cleanup**: Documents are deleted automatically without manual intervention
- **Background Process**: Cleanup runs every 60 seconds in a background goroutine
- **Multiple Indexes**: You can create multiple TTL indexes on different fields
- **Flexible Timestamps**: Supports `time.Time`, RFC3339 strings, and Unix timestamps
- **Zero Overhead**: TTL indexes are maintained efficiently during normal CRUD operations

### Cleanup Process

1. **Periodic Scanning**: Every 60 seconds, the database scans all TTL indexes
2. **Expiration Check**: Documents with timestamps older than `current_time - ttl_seconds` are identified
3. **Automatic Deletion**: Expired documents are removed from the collection and all indexes
4. **Cache Invalidation**: Query cache is cleared when documents are deleted

## Creating TTL Indexes

### Basic Usage

```go
db, _ := database.Open(database.DefaultConfig("./data"))
coll := db.Collection("sessions")

// Create TTL index on 'createdAt' field with 3600 second (1 hour) expiration
err := coll.CreateTTLIndex("createdAt", 3600)
if err != nil {
    log.Fatal(err)
}

// Insert a document with timestamp
coll.InsertOne(map[string]interface{}{
    "user":      "alice",
    "createdAt": time.Now(),  // Will expire in 1 hour
})
```

### Index Naming Convention

TTL indexes follow the naming pattern: `{field}_ttl`

Example:
- Field: `createdAt` → Index: `createdAt_ttl`
- Field: `expiresAt` → Index: `expiresAt_ttl`

## Supported Timestamp Formats

TTL indexes accept three timestamp formats:

### 1. time.Time (Native Go)

```go
coll.InsertOne(map[string]interface{}{
    "data": "session data",
    "createdAt": time.Now(),
})
```

### 2. RFC3339 String

```go
coll.InsertOne(map[string]interface{}{
    "data": "log entry",
    "timestamp": "2024-01-15T10:30:00Z",  // RFC3339 format
})
```

### 3. Unix Timestamp (int64)

```go
coll.InsertOne(map[string]interface{}{
    "data": "event",
    "eventTime": time.Now().Unix(),  // Unix seconds since epoch
})
```

## Expiration Examples

### Session Management

```go
// Sessions expire after 30 minutes of inactivity
sessionsColl := db.Collection("sessions")
sessionsColl.CreateTTLIndex("lastActivity", 1800)  // 30 minutes

// Create session
sessionsColl.InsertOne(map[string]interface{}{
    "sessionId":    "abc123",
    "user":         "alice",
    "lastActivity": time.Now(),
})

// Update session activity
sessionsColl.UpdateOne(
    map[string]interface{}{"sessionId": "abc123"},
    map[string]interface{}{
        "$set": map[string]interface{}{
            "lastActivity": time.Now(),  // Resets expiration timer
        },
    },
)
```

### Temporary Tokens

```go
// Verification tokens expire after 24 hours
tokensColl := db.Collection("verification_tokens")
tokensColl.CreateTTLIndex("createdAt", 86400)  // 24 hours

tokensColl.InsertOne(map[string]interface{}{
    "token":     "verify_abc123",
    "email":     "user@example.com",
    "createdAt": time.Now(),
})
// Token automatically deleted after 24 hours
```

### Event Logs with Multiple TTLs

```go
logsColl := db.Collection("logs")

// Keep audit logs for 90 days
logsColl.CreateTTLIndex("auditTime", 90*24*3600)

// But keep detailed debug logs for only 7 days
logsColl.CreateTTLIndex("debugTime", 7*24*3600)

logsColl.InsertOne(map[string]interface{}{
    "level":     "INFO",
    "message":   "User logged in",
    "auditTime": time.Now(),   // Expires in 90 days
    "debugTime": time.Now(),   // Expires in 7 days (will delete first)
})
```

## Index Management

### List TTL Indexes

```go
indexes := coll.ListIndexes()
for _, idx := range indexes {
    if idx["type"] == "ttl" {
        fmt.Printf("TTL Index: %s, Field: %s, TTL: %d seconds\n",
            idx["name"], idx["field"], idx["ttlSeconds"])
    }
}
```

### Drop TTL Index

```go
err := coll.DropIndex("createdAt_ttl")
if err != nil {
    log.Fatal(err)
}
```

### Manual Cleanup (Testing)

```go
// Manually trigger cleanup (usually happens automatically every 60 seconds)
deletedCount := coll.CleanupExpiredDocuments()
fmt.Printf("Deleted %d expired documents\n", deletedCount)
```

## Performance Characteristics

### Insertion

- **With TTL Index**: ~14µs per insert (7% overhead)
- **Without TTL Index**: ~13µs per insert

TTL indexes add minimal overhead during document insertion.

### Cleanup

- **500 expired docs**: ~7ms cleanup time
- **1000 expired docs**: ~14ms cleanup time

Cleanup is O(n) where n is the number of expired documents, not total documents.

### Memory Overhead

Each TTL index maintains:
- Document ID → Expiration Time mapping
- Average: ~48 bytes per document

For 100,000 documents: ~4.8MB memory overhead

## Best Practices

### 1. Choose Appropriate TTL Values

```go
// Good: Reasonable TTL for session data
sessionsColl.CreateTTLIndex("lastActivity", 3600)  // 1 hour

// Avoid: Very short TTLs may cause frequent cleanup overhead
cacheColl.CreateTTLIndex("timestamp", 1)  // 1 second - may be too aggressive
```

### 2. Use Explicit Timestamp Fields

```go
// Good: Explicit field name
coll.CreateTTLIndex("expiresAt", 300)

// Avoid: Ambiguous field names
coll.CreateTTLIndex("time", 300)
```

### 3. Document the Expiration Policy

```go
// Document your TTL choices
const (
    SESSION_TTL = 1800      // 30 minutes
    TOKEN_TTL   = 86400     // 24 hours
    CACHE_TTL   = 300       // 5 minutes
)

sessionsColl.CreateTTLIndex("lastActivity", SESSION_TTL)
```

### 4. Handle Missing Timestamps

Documents without the TTL field are simply not indexed and won't expire:

```go
// Document WITH timestamp - will expire
coll.InsertOne(map[string]interface{}{
    "data": "expires",
    "createdAt": time.Now(),
})

// Document WITHOUT timestamp - won't expire (just not in TTL index)
coll.InsertOne(map[string]interface{}{
    "data": "permanent",
})
```

### 5. Test Expiration Behavior

```go
func TestSessionExpiration(t *testing.T) {
    coll := db.Collection("test_sessions")
    coll.CreateTTLIndex("createdAt", 2)  // 2 second TTL for testing

    // Insert expired session
    coll.InsertOne(map[string]interface{}{
        "user": "test",
        "createdAt": time.Now().Add(-5 * time.Second),
    })

    // Trigger cleanup
    count := coll.CleanupExpiredDocuments()

    // Verify deletion
    if count != 1 {
        t.Errorf("Expected 1 deleted, got %d", count)
    }
}
```

## Common Patterns

### Pattern 1: Session with Sliding Expiration

```go
// Session expires 30 minutes after LAST activity
func UpdateSession(sessionId string) {
    sessionsColl.UpdateOne(
        map[string]interface{}{"sessionId": sessionId},
        map[string]interface{}{
            "$set": map[string]interface{}{
                "lastActivity": time.Now(),  // Resets expiration
            },
        },
    )
}
```

### Pattern 2: Fixed Expiration Time

```go
// Token expires at specific future time
expiresAt := time.Now().Add(24 * time.Hour)

tokensColl.InsertOne(map[string]interface{}{
    "token": "abc123",
    "expiresAt": expiresAt,
})

// Create TTL index with 0 seconds - expires exactly at expiresAt timestamp
tokensColl.CreateTTLIndex("expiresAt", 0)
```

### Pattern 3: Conditional Expiration

```go
// Only expire "temporary" documents
coll.InsertOne(map[string]interface{}{
    "type": "temporary",
    "expiresAt": time.Now(),  // Will be indexed and expire
})

coll.InsertOne(map[string]interface{}{
    "type": "permanent",
    // No expiresAt field - won't expire
})
```

## Architecture Details

### TTL Index Structure

```go
type TTLIndex struct {
    name       string
    fieldPath  string
    ttlSeconds int64

    // Maps document ID to calculated expiration time
    expirationTimes map[string]time.Time
}
```

### Cleanup Goroutine

```go
// Runs in background every 60 seconds
func (db *Database) ttlCleanupLoop() {
    ticker := time.NewTicker(60 * time.Second)
    for {
        select {
        case <-ticker.C:
            db.cleanupExpiredDocuments()
        case <-db.ttlStopChan:
            return
        }
    }
}
```

### Expiration Calculation

```
expiration_time = timestamp_field_value + ttl_seconds

if current_time > expiration_time:
    document is expired and will be deleted
```

## Current Limitations

1. **Cleanup Frequency**: Fixed 60-second intervals (not configurable)
2. **Precision**: Expiration is not exact - documents may persist up to 60 seconds past expiration
3. **Single Field**: Each TTL index monitors only one timestamp field
4. **No Timezone Support**: All times are in UTC
5. **No Pause/Resume**: TTL cleanup runs continuously while database is open

## Future Enhancements

Planned improvements for TTL indexes:

- [ ] Configurable cleanup interval
- [ ] TTL statistics and metrics
- [ ] Expiration callbacks/hooks
- [ ] Conditional TTL (based on field values)
- [ ] TTL index compaction
- [ ] Per-collection cleanup scheduling

## Comparison with Manual Deletion

| Aspect | TTL Index | Manual Deletion |
|--------|-----------|----------------|
| **Automation** | Fully automatic | Requires cron jobs or scheduled tasks |
| **Precision** | ~60 second granularity | Exact control |
| **Overhead** | Minimal (background process) | Depends on implementation |
| **Complexity** | Simple API | More code to maintain |
| **Flexibility** | Fixed behavior | Full control over logic |

## Troubleshooting

### Documents Not Expiring

**Check 1**: Verify TTL index exists
```go
indexes := coll.ListIndexes()
// Look for your TTL index
```

**Check 2**: Verify timestamp field format
```go
// Ensure documents have correct timestamp field
doc, _ := coll.FindOne(map[string]interface{}{})
timestamp, exists := doc.Get("createdAt")
fmt.Printf("Timestamp: %v, Type: %T\n", timestamp, timestamp)
```

**Check 3**: Wait for cleanup cycle (60 seconds)
```go
// Or trigger manual cleanup for testing
count := coll.CleanupExpiredDocuments()
```

### Unexpected Deletions

**Check**: Verify TTL duration
```go
indexes := coll.ListIndexes()
for _, idx := range indexes {
    if idx["type"] == "ttl" {
        fmt.Printf("TTL: %d seconds\n", idx["ttlSeconds"])
    }
}
```

### Performance Issues

**Monitor**: Cleanup duration
```go
start := time.Now()
count := coll.CleanupExpiredDocuments()
duration := time.Since(start)
fmt.Printf("Deleted %d docs in %v\n", count, duration)
```

If cleanup is slow, consider:
- Shorter TTL values to spread deletions over time
- Batch archival before deletion
- Partitioning data across collections

## Related Documentation

- [Indexing Guide](./indexing.md) - General index concepts
- [Query Optimization](./query-engine.md) - Query performance with indexes
- [Statistics](./statistics-optimization.md) - Index statistics

## Examples

See `pkg/database/ttl_test.go` for comprehensive usage examples and test cases.
