# Config Server - Sharding Metadata Management

## Overview

The Config Server is a critical component of LauraDB's sharding infrastructure that stores and manages cluster metadata. Similar to MongoDB's config servers, it maintains the authoritative source of truth for:

- **Shard Registry**: Information about all shards in the cluster
- **Chunk Metadata**: Range mappings and chunk distribution
- **Collection Configuration**: Sharding configuration per collection
- **Cluster Topology**: Shard states, tags, and locations

## Architecture

```
┌─────────────────────────────────────────────────┐
│           Config Server                         │
│                                                 │
│  ┌──────────────┐  ┌──────────────┐           │
│  │    Shard     │  │    Chunk     │           │
│  │   Registry   │  │   Registry   │           │
│  └──────────────┘  └──────────────┘           │
│                                                 │
│  ┌──────────────────────────────────┐         │
│  │   Collection Sharding Config     │         │
│  └──────────────────────────────────┘         │
│                                                 │
│  ┌──────────────────────────────────┐         │
│  │   Persistent JSON Storage        │         │
│  └──────────────────────────────────┘         │
└─────────────────────────────────────────────────┘
         │              │              │
         ▼              ▼              ▼
   ┌─────────┐    ┌─────────┐    ┌─────────┐
   │ Shard 1 │    │ Shard 2 │    │ Shard 3 │
   └─────────┘    └─────────┘    └─────────┘
```

## Key Components

### 1. ConfigServer

The main struct that manages all metadata operations.

```go
type ConfigServer struct {
    dataDir        string
    shardRegistry  map[ShardID]*ShardMetadata
    chunkRegistry  map[string]*ChunkMetadata
    collectionMeta map[string]*CollectionShardingConfig
    mu             sync.RWMutex
    version        int64
}
```

### 2. ShardMetadata

Persistent information about a shard in the cluster.

```go
type ShardMetadata struct {
    ID        ShardID
    Host      string
    Tags      map[string]string
    State     ShardState
    AddedAt   time.Time
    UpdatedAt time.Time
}
```

**Shard States:**
- `active`: Shard is online and serving traffic
- `draining`: Shard is being removed, chunks being migrated
- `inactive`: Temporarily offline for maintenance
- `unreachable`: Cannot be contacted

### 3. ChunkMetadata

Metadata about range-based sharding chunks.

```go
type ChunkMetadata struct {
    ID        string
    ShardID   ShardID
    MinKey    interface{}
    MaxKey    interface{}
    Count     int64
    Size      int64
    Version   int64
    UpdatedAt time.Time
}
```

### 4. CollectionShardingConfig

Sharding configuration for a collection.

```go
type CollectionShardingConfig struct {
    Database   string
    Collection string
    ShardKey   *ShardKey
    Sharded    bool
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

## API Reference

### Creating a Config Server

```go
cs, err := sharding.NewConfigServer("/path/to/config-data")
if err != nil {
    log.Fatal(err)
}
defer cs.Close()
```

### Shard Management

#### Register a Shard
```go
shard := sharding.NewShard("shard-1", db, "localhost:27017")
shard.SetTag("datacenter", "us-east")
shard.SetTag("region", "us")

err := cs.RegisterShard(shard)
```

#### Update Shard State
```go
err := cs.UpdateShardState("shard-1", sharding.ShardStateDraining)
```

#### Update Shard Tags
```go
tags := map[string]string{
    "datacenter": "us-west",
    "rack": "rack-5",
}
err := cs.UpdateShardTags("shard-1", tags)
```

#### Get Shard Information
```go
meta, err := cs.GetShard("shard-1")
fmt.Printf("Shard: %s, Host: %s, State: %s\n", meta.ID, meta.Host, meta.State)
```

#### List All Shards
```go
shards := cs.ListShards()
for _, meta := range shards {
    fmt.Printf("Shard %s: %s (%s)\n", meta.ID, meta.Host, meta.State)
}
```

#### List Active Shards Only
```go
activeShards := cs.ListActiveShards()
```

#### Unregister a Shard
```go
err := cs.UnregisterShard("shard-1")
```

**Note:** Cannot unregister a shard that has chunks assigned to it.

### Chunk Management

#### Register a Chunk
```go
chunk := sharding.NewChunk("chunk-1", "shard-1", int64(0), int64(1000))
chunk.UpdateStats(450, 92160) // 450 docs, ~90KB

err := cs.RegisterChunk(chunk)
```

#### Update Chunk Statistics
```go
err := cs.UpdateChunk("chunk-1", 500, 100000) // new count, new size
```

#### Move Chunk to Different Shard
```go
err := cs.MoveChunkMetadata("chunk-1", "shard-2")
```

This increments the chunk version for migration tracking.

#### Get Chunk Information
```go
meta, err := cs.GetChunk("chunk-1")
fmt.Printf("Chunk: %s, Shard: %s, Range: [%v, %v)\n",
    meta.ID, meta.ShardID, meta.MinKey, meta.MaxKey)
```

#### List All Chunks
```go
chunks := cs.ListChunks()
```

#### List Chunks for Specific Shard
```go
chunks := cs.ListChunksForShard("shard-1")
totalDocs := int64(0)
for _, chunk := range chunks {
    totalDocs += chunk.Count
}
```

#### Unregister a Chunk
```go
err := cs.UnregisterChunk("chunk-1")
```

### Collection Sharding Configuration

#### Configure Collection Sharding
```go
// Hash-based sharding
shardKey := sharding.NewHashShardKey("user_id")
err := cs.SetCollectionSharding("myapp", "users", shardKey)

// Range-based sharding
shardKey := sharding.NewRangeShardKey("timestamp")
err := cs.SetCollectionSharding("myapp", "events", shardKey)

// Compound shard key
shardKey := sharding.NewRangeShardKey("customer_id", "order_date")
err := cs.SetCollectionSharding("myapp", "orders", shardKey)
```

#### Get Collection Configuration
```go
config, err := cs.GetCollectionSharding("myapp", "users")
fmt.Printf("Database: %s, Collection: %s\n", config.Database, config.Collection)
fmt.Printf("Shard Key: %v, Type: %s\n", config.ShardKey.Fields, config.ShardKey.Type)
```

#### List All Sharded Collections
```go
collections := cs.ListShardedCollections()
for _, config := range collections {
    fmt.Printf("%s.%s sharded by %v (%s)\n",
        config.Database, config.Collection,
        config.ShardKey.Fields, config.ShardKey.Type)
}
```

#### Remove Collection Sharding
```go
err := cs.RemoveCollectionSharding("myapp", "users")
```

### Metadata Versioning

The config server maintains a version number that increments with each metadata change. This enables optimistic concurrency control and change detection.

```go
version := cs.GetVersion()
```

### Statistics

Get comprehensive statistics about the config server:

```go
stats := cs.Stats()
fmt.Printf("Version: %d\n", stats["version"])
fmt.Printf("Total Shards: %d\n", stats["total_shards"])
fmt.Printf("Active Shards: %d\n", stats["active_shards"])
fmt.Printf("Draining Shards: %d\n", stats["draining_shards"])
fmt.Printf("Total Chunks: %d\n", stats["total_chunks"])
fmt.Printf("Sharded Collections: %d\n", stats["sharded_collections"])
```

## Persistence and Recovery

### Automatic Persistence

All metadata changes are automatically persisted to disk in JSON format:

```
/path/to/config-data/
  └── config_server_metadata.json
```

The file is written atomically using a temporary file + rename pattern to prevent corruption.

### Metadata Format

```json
{
  "version": 6,
  "shards": {
    "shard-1": {
      "id": "shard-1",
      "host": "localhost:27017",
      "tags": {
        "datacenter": "us-east",
        "region": "us"
      },
      "state": "active",
      "added_at": "2025-11-24T10:00:00Z",
      "updated_at": "2025-11-24T10:00:00Z"
    }
  },
  "chunks": {
    "chunk-1": {
      "id": "chunk-1",
      "shard_id": "shard-1",
      "min_key": 0,
      "max_key": 1000,
      "count": 450,
      "size": 92160,
      "version": 1,
      "updated_at": "2025-11-24T10:00:00Z"
    }
  },
  "collection_meta": {
    "mydb.users": {
      "database": "mydb",
      "collection": "users",
      "shard_key": {
        "Fields": ["user_id"],
        "Type": 1,
        "Unique": false
      },
      "sharded": true,
      "created_at": "2025-11-24T10:00:00Z",
      "updated_at": "2025-11-24T10:00:00Z"
    }
  },
  "updated_at": "2025-11-24T10:00:00Z"
}
```

### Recovery

On startup, the config server automatically loads metadata from disk:

```go
cs, err := sharding.NewConfigServer("/path/to/config-data")
// All metadata is loaded from disk if the file exists
```

If no metadata file exists, the config server starts with an empty state.

## Use Cases

### 1. Shard Topology Management

Track all shards in the cluster with their locations and states:

```go
cs := sharding.NewConfigServer("/data/config")

// Add shards across datacenters
shard1 := sharding.NewShard("us-east-1", db1, "10.0.1.10:27017")
shard1.SetTag("datacenter", "us-east")
cs.RegisterShard(shard1)

shard2 := sharding.NewShard("eu-west-1", db2, "10.0.2.10:27017")
shard2.SetTag("datacenter", "eu-west")
cs.RegisterShard(shard2)

// Query active shards
activeShards := cs.ListActiveShards()
```

### 2. Chunk Distribution Tracking

Maintain chunk metadata for range-based sharding:

```go
// Register chunks
chunk1 := sharding.NewChunk("chunk-1", "shard-1", int64(0), int64(10000))
chunk1.UpdateStats(5000, 1024000)
cs.RegisterChunk(chunk1)

chunk2 := sharding.NewChunk("chunk-2", "shard-2", int64(10000), int64(20000))
chunk2.UpdateStats(4800, 980000)
cs.RegisterChunk(chunk2)

// Monitor chunk distribution
for _, shardID := range []string{"shard-1", "shard-2"} {
    chunks := cs.ListChunksForShard(shardID)
    fmt.Printf("Shard %s has %d chunks\n", shardID, len(chunks))
}
```

### 3. Chunk Migration

Track chunk movements during balancing:

```go
// Before migration
chunk, _ := cs.GetChunk("chunk-1")
fmt.Printf("Chunk on %s (version %d)\n", chunk.ShardID, chunk.Version)

// Migrate chunk
cs.MoveChunkMetadata("chunk-1", "shard-2")

// After migration
chunk, _ = cs.GetChunk("chunk-1")
fmt.Printf("Chunk on %s (version %d)\n", chunk.ShardID, chunk.Version)
```

### 4. Collection-Level Sharding

Configure which collections are sharded and how:

```go
// Users collection - hash by user_id
userKey := sharding.NewHashShardKey("user_id")
cs.SetCollectionSharding("myapp", "users", userKey)

// Events collection - range by timestamp
eventKey := sharding.NewRangeShardKey("timestamp")
cs.SetCollectionSharding("myapp", "events", eventKey)

// Check if collection is sharded
config, err := cs.GetCollectionSharding("myapp", "users")
if err == nil {
    fmt.Printf("Collection is sharded by %v\n", config.ShardKey.Fields)
}
```

### 5. Disaster Recovery

Recover cluster metadata after failure:

```go
// Original config server
cs1, _ := sharding.NewConfigServer("/data/config")
// ... populate with shards, chunks, collections ...
cs1.Close()

// After restart or failover
cs2, _ := sharding.NewConfigServer("/data/config")
// All metadata automatically loaded from disk

// Verify recovery
shards := cs2.ListShards()
chunks := cs2.ListChunks()
collections := cs2.ListShardedCollections()
fmt.Printf("Recovered: %d shards, %d chunks, %d collections\n",
    len(shards), len(chunks), len(collections))
```

## Performance Characteristics

- **Read Operations**: O(1) for Get operations, O(N) for List operations
- **Write Operations**: O(1) for updates + disk I/O for persistence
- **Persistence**: Atomic write with temporary file + rename
- **Concurrency**: RWMutex for concurrent reads, exclusive writes
- **Memory**: Entire metadata kept in memory for fast access
- **Metadata Size**: Typically < 1MB for thousands of shards/chunks

## Best Practices

### 1. Tag-Based Organization

Use tags to organize shards by datacenter, rack, or region:

```go
shard.SetTag("datacenter", "us-east-1a")
shard.SetTag("rack", "rack-5")
shard.SetTag("purpose", "analytics")
```

### 2. Chunk Size Monitoring

Regularly update chunk statistics for balancing decisions:

```go
// After significant inserts/deletes
newCount := collection.Count()
newSize := calculateCollectionSize()
cs.UpdateChunk("chunk-1", newCount, newSize)
```

### 3. Graceful Shard Removal

Use draining state before removing shards:

```go
// 1. Mark as draining
cs.UpdateShardState("shard-1", sharding.ShardStateDraining)

// 2. Migrate all chunks away
chunks := cs.ListChunksForShard("shard-1")
for _, chunk := range chunks {
    cs.MoveChunkMetadata(chunk.ID, "shard-2")
}

// 3. Verify no chunks remain
chunks = cs.ListChunksForShard("shard-1")
if len(chunks) == 0 {
    cs.UnregisterShard("shard-1")
}
```

### 4. Version-Based Change Detection

Use version numbers to detect metadata changes:

```go
oldVersion := cs.GetVersion()

// ... make changes ...

newVersion := cs.GetVersion()
if newVersion > oldVersion {
    fmt.Println("Metadata changed, reload routing tables")
}
```

### 5. Regular Backups

Since metadata is critical, backup the metadata file:

```go
// Backup config server data directory
cp /data/config/config_server_metadata.json /backup/config_$(date +%Y%m%d).json
```

## Limitations and Future Enhancements

### Current Limitations

1. **Single Instance**: No built-in replication (MongoDB uses 3-node replica set)
2. **No Network API**: Local file-based only (no RPC/HTTP interface)
3. **No Transactions**: Metadata changes are individual operations
4. **Manual Synchronization**: Client must query for changes

### Planned Enhancements

1. **Config Server Replica Set**: 3-node deployment for high availability
2. **Network Protocol**: gRPC or HTTP API for remote access
3. **Change Streams**: Publish/subscribe for metadata changes
4. **Snapshot Isolation**: Multi-operation metadata transactions
5. **Compact History**: Retain historical metadata versions

## Integration with ShardRouter

The config server works alongside ShardRouter for complete sharding:

```go
// Create config server
cs, _ := sharding.NewConfigServer("/data/config")

// Register shards
shard1 := sharding.NewShard("shard-1", db1, "host1:27017")
cs.RegisterShard(shard1)
shard2 := sharding.NewShard("shard-2", db2, "host2:27017")
cs.RegisterShard(shard2)

// Configure collection sharding
shardKey := sharding.NewHashShardKey("user_id")
cs.SetCollectionSharding("mydb", "users", shardKey)

// Create router
router, _ := sharding.NewShardRouter(shardKey)
router.AddShard(shard1)
router.AddShard(shard2)

// Route operations
doc := map[string]interface{}{"user_id": "user123", "name": "Alice"}
shard, _ := router.Route(doc)

// Update chunk metadata after insert
chunk, _ := cs.GetChunk("chunk-1")
cs.UpdateChunk(chunk.ID, chunk.Count+1, chunk.Size+docSize)
```

## Comparison with MongoDB Config Servers

| Feature | LauraDB | MongoDB |
|---------|---------|---------|
| Deployment | Single instance | 3-node replica set |
| Storage | JSON file | BSON + WiredTiger |
| Network API | Local only | MongoDB protocol |
| Change Streams | No | Yes |
| Elections | N/A | Yes (Raft-like) |
| Transactions | No | Yes |
| Sharding Types | Hash, Range | Hash, Range, Zone |

## Troubleshooting

### Metadata File Corrupted

If metadata file is corrupted, it can be manually edited (JSON format) or restored from backup:

```bash
# Restore from backup
cp /backup/config_20251124.json /data/config/config_server_metadata.json
```

### Cannot Unregister Shard

Error: "cannot unregister shard: still has chunks assigned"

**Solution**: Migrate all chunks first:
```go
chunks := cs.ListChunksForShard("shard-1")
for _, chunk := range chunks {
    cs.MoveChunkMetadata(chunk.ID, "target-shard")
}
cs.UnregisterShard("shard-1")
```

### Metadata Version Mismatch

If clients have stale routing tables, check version:

```go
currentVersion := cs.GetVersion()
if clientVersion < currentVersion {
    // Reload metadata
    shards = cs.ListShards()
    chunks = cs.ListChunks()
}
```

## See Also

- [Sharding Documentation](sharding.md)
- [Range-Based Sharding](range-sharding.md)
- [Hash-Based Sharding](hash-sharding.md)
- [Chunk Migration](chunk-migration.md)
