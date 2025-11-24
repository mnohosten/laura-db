package main

import (
	"fmt"
	"os"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/sharding"
)

func main() {
	fmt.Println("=== LauraDB Config Server Demo ===")
	fmt.Println()

	// Demo 1: Basic config server setup
	demo1BasicSetup()

	fmt.Println()

	// Demo 2: Shard registration and management
	demo2ShardManagement()

	fmt.Println()

	// Demo 3: Chunk metadata tracking
	demo3ChunkMetadata()

	fmt.Println()

	// Demo 4: Collection sharding configuration
	demo4CollectionSharding()

	fmt.Println()

	// Demo 5: Config server persistence and recovery
	demo5Persistence()

	fmt.Println("\n=== Demo Complete ===")
}

func demo1BasicSetup() {
	fmt.Println("Demo 1: Basic Config Server Setup")
	fmt.Println("-----------------------------------")

	// Create config server
	configDir := "/tmp/laura-config-server"
	os.RemoveAll(configDir)
	defer os.RemoveAll(configDir)

	cs, err := sharding.NewConfigServer(configDir)
	if err != nil {
		fmt.Printf("Error creating config server: %v\n", err)
		return
	}
	defer cs.Close()

	fmt.Printf("✓ Config server created at: %s\n", configDir)
	fmt.Printf("  Initial version: %d\n", cs.GetVersion())

	// Display stats
	stats := cs.Stats()
	fmt.Println("\nConfig server stats:")
	fmt.Printf("  Total shards: %d\n", stats["total_shards"])
	fmt.Printf("  Total chunks: %d\n", stats["total_chunks"])
	fmt.Printf("  Sharded collections: %d\n", stats["sharded_collections"])
}

func demo2ShardManagement() {
	fmt.Println("Demo 2: Shard Registration and Management")
	fmt.Println("------------------------------------------")

	configDir := "/tmp/laura-config-server-2"
	os.RemoveAll(configDir)
	defer os.RemoveAll(configDir)

	cs, _ := sharding.NewConfigServer(configDir)
	defer cs.Close()

	// Register multiple shards
	fmt.Println("Registering shards:")

	shard1 := sharding.NewShard("shard-us-east", nil, "us-east-1.example.com:27017")
	shard1.SetTag("datacenter", "us-east")
	shard1.SetTag("region", "us")
	cs.RegisterShard(shard1)
	fmt.Printf("  ✓ Registered %s at %s (datacenter: %s)\n", shard1.ID, shard1.Host, "us-east")

	shard2 := sharding.NewShard("shard-us-west", nil, "us-west-1.example.com:27017")
	shard2.SetTag("datacenter", "us-west")
	shard2.SetTag("region", "us")
	cs.RegisterShard(shard2)
	fmt.Printf("  ✓ Registered %s at %s (datacenter: %s)\n", shard2.ID, shard2.Host, "us-west")

	shard3 := sharding.NewShard("shard-eu-central", nil, "eu-central-1.example.com:27017")
	shard3.SetTag("datacenter", "eu-central")
	shard3.SetTag("region", "eu")
	cs.RegisterShard(shard3)
	fmt.Printf("  ✓ Registered %s at %s (datacenter: %s)\n", shard3.ID, shard3.Host, "eu-central")

	// List all shards
	fmt.Println("\nAll registered shards:")
	for _, meta := range cs.ListShards() {
		fmt.Printf("  - %s: %s (%s)\n", meta.ID, meta.Host, meta.State)
		for k, v := range meta.Tags {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}

	// Update shard state
	fmt.Println("\nUpdating shard state:")
	cs.UpdateShardState("shard-us-west", sharding.ShardStateDraining)
	fmt.Printf("  ✓ Set %s to draining state\n", "shard-us-west")

	// List active shards only
	fmt.Println("\nActive shards:")
	for _, meta := range cs.ListActiveShards() {
		fmt.Printf("  - %s (%s)\n", meta.ID, meta.State)
	}

	stats := cs.Stats()
	fmt.Println("\nShard statistics:")
	fmt.Printf("  Active shards: %d\n", stats["active_shards"])
	fmt.Printf("  Draining shards: %d\n", stats["draining_shards"])
	fmt.Printf("  Metadata version: %d\n", stats["version"])
}

func demo3ChunkMetadata() {
	fmt.Println("Demo 3: Chunk Metadata Tracking")
	fmt.Println("---------------------------------")

	configDir := "/tmp/laura-config-server-3"
	os.RemoveAll(configDir)
	defer os.RemoveAll(configDir)

	cs, _ := sharding.NewConfigServer(configDir)
	defer cs.Close()

	// Register shards
	shard1 := sharding.NewShard("shard-1", nil, "localhost:27017")
	shard2 := sharding.NewShard("shard-2", nil, "localhost:27018")
	cs.RegisterShard(shard1)
	cs.RegisterShard(shard2)

	// Register chunks with range-based sharding
	fmt.Println("Registering chunks:")

	chunk1 := sharding.NewChunk("chunk-1", "shard-1", int64(0), int64(1000))
	chunk1.UpdateStats(450, 92160) // 450 docs, ~90KB
	cs.RegisterChunk(chunk1)
	fmt.Printf("  ✓ Chunk %s on %s: range [%v, %v), %d docs, %d bytes\n",
		chunk1.ID, chunk1.ShardID, chunk1.MinKey, chunk1.MaxKey, chunk1.Count, chunk1.Size)

	chunk2 := sharding.NewChunk("chunk-2", "shard-1", int64(1000), int64(2000))
	chunk2.UpdateStats(520, 106496) // 520 docs, ~104KB
	cs.RegisterChunk(chunk2)
	fmt.Printf("  ✓ Chunk %s on %s: range [%v, %v), %d docs, %d bytes\n",
		chunk2.ID, chunk2.ShardID, chunk2.MinKey, chunk2.MaxKey, chunk2.Count, chunk2.Size)

	chunk3 := sharding.NewChunk("chunk-3", "shard-2", int64(2000), int64(3000))
	chunk3.UpdateStats(380, 77824) // 380 docs, ~76KB
	cs.RegisterChunk(chunk3)
	fmt.Printf("  ✓ Chunk %s on %s: range [%v, %v), %d docs, %d bytes\n",
		chunk3.ID, chunk3.ShardID, chunk3.MinKey, chunk3.MaxKey, chunk3.Count, chunk3.Size)

	// List chunks per shard
	fmt.Println("\nChunks by shard:")
	for _, shardID := range []sharding.ShardID{"shard-1", "shard-2"} {
		chunks := cs.ListChunksForShard(shardID)
		totalDocs := int64(0)
		totalSize := int64(0)
		for _, chunk := range chunks {
			totalDocs += chunk.Count
			totalSize += chunk.Size
		}
		fmt.Printf("  %s: %d chunks, %d docs, %d bytes\n",
			shardID, len(chunks), totalDocs, totalSize)
	}

	// Simulate chunk migration
	fmt.Println("\nMigrating chunk-2 to shard-2:")
	oldMeta, _ := cs.GetChunk("chunk-2")
	fmt.Printf("  Before: chunk-2 on %s (version %d)\n", oldMeta.ShardID, oldMeta.Version)

	cs.MoveChunkMetadata("chunk-2", "shard-2")

	newMeta, _ := cs.GetChunk("chunk-2")
	fmt.Printf("  After:  chunk-2 on %s (version %d)\n", newMeta.ShardID, newMeta.Version)

	// Update chunk statistics after migration
	fmt.Println("\nUpdating chunk statistics:")
	cs.UpdateChunk("chunk-2", 530, 108544)
	updated, _ := cs.GetChunk("chunk-2")
	fmt.Printf("  ✓ Chunk %s now has %d docs, %d bytes\n", updated.ID, updated.Count, updated.Size)
}

func demo4CollectionSharding() {
	fmt.Println("Demo 4: Collection Sharding Configuration")
	fmt.Println("------------------------------------------")

	configDir := "/tmp/laura-config-server-4"
	os.RemoveAll(configDir)
	defer os.RemoveAll(configDir)

	cs, _ := sharding.NewConfigServer(configDir)
	defer cs.Close()

	fmt.Println("Configuring sharded collections:")

	// Hash-based sharding for users
	usersShardKey := sharding.NewHashShardKey("user_id")
	cs.SetCollectionSharding("myapp", "users", usersShardKey)
	fmt.Printf("  ✓ Collection myapp.users sharded by %s (hash)\n", "user_id")

	// Range-based sharding for events
	eventsShardKey := sharding.NewRangeShardKey("timestamp")
	cs.SetCollectionSharding("myapp", "events", eventsShardKey)
	fmt.Printf("  ✓ Collection myapp.events sharded by %s (range)\n", "timestamp")

	// Compound shard key for orders
	ordersShardKey := sharding.NewRangeShardKey("customer_id", "order_date")
	cs.SetCollectionSharding("myapp", "orders", ordersShardKey)
	fmt.Printf("  ✓ Collection myapp.orders sharded by %v (range)\n", ordersShardKey.Fields)

	// List all sharded collections
	fmt.Println("\nAll sharded collections:")
	for _, config := range cs.ListShardedCollections() {
		fmt.Printf("  - %s.%s: shard key %s, type %s\n",
			config.Database, config.Collection,
			config.ShardKey.Fields, config.ShardKey.Type)
	}

	// Get specific collection config
	fmt.Println("\nRetrieving specific collection config:")
	usersConfig, _ := cs.GetCollectionSharding("myapp", "users")
	fmt.Printf("  Collection: %s.%s\n", usersConfig.Database, usersConfig.Collection)
	fmt.Printf("  Shard key: %v\n", usersConfig.ShardKey.Fields)
	fmt.Printf("  Type: %s\n", usersConfig.ShardKey.Type)
	fmt.Printf("  Created: %s\n", usersConfig.CreatedAt.Format("2006-01-02 15:04:05"))
}

func demo5Persistence() {
	fmt.Println("Demo 5: Config Server Persistence and Recovery")
	fmt.Println("-----------------------------------------------")

	configDir := "/tmp/laura-config-server-5"
	os.RemoveAll(configDir)
	defer os.RemoveAll(configDir)

	// Phase 1: Create config server and populate data
	fmt.Println("Phase 1: Creating config server and adding metadata")

	cs1, _ := sharding.NewConfigServer(configDir)

	// Add shards
	shard1 := sharding.NewShard("shard-1", nil, "localhost:27017")
	shard2 := sharding.NewShard("shard-2", nil, "localhost:27018")
	cs1.RegisterShard(shard1)
	cs1.RegisterShard(shard2)
	fmt.Printf("  ✓ Registered 2 shards\n")

	// Add chunks
	chunk1 := sharding.NewChunk("chunk-1", "shard-1", int64(0), int64(1000))
	chunk2 := sharding.NewChunk("chunk-2", "shard-2", int64(1000), int64(2000))
	cs1.RegisterChunk(chunk1)
	cs1.RegisterChunk(chunk2)
	fmt.Printf("  ✓ Registered 2 chunks\n")

	// Add collection sharding
	shardKey := sharding.NewHashShardKey("user_id")
	cs1.SetCollectionSharding("mydb", "users", shardKey)
	fmt.Printf("  ✓ Configured sharding for mydb.users\n")

	version1 := cs1.GetVersion()
	fmt.Printf("  Metadata version: %d\n", version1)

	// Close config server
	cs1.Close()
	fmt.Println("  ✓ Config server closed, metadata persisted to disk")

	// Phase 2: Recover from disk
	fmt.Println("\nPhase 2: Recovering config server from disk")

	cs2, err := sharding.NewConfigServer(configDir)
	if err != nil {
		fmt.Printf("Error recovering config server: %v\n", err)
		return
	}
	defer cs2.Close()

	version2 := cs2.GetVersion()
	fmt.Printf("  ✓ Config server recovered\n")
	fmt.Printf("  Metadata version: %d (matches: %v)\n", version2, version1 == version2)

	// Verify all data was recovered
	shards := cs2.ListShards()
	fmt.Printf("  Recovered %d shards\n", len(shards))

	chunks := cs2.ListChunks()
	fmt.Printf("  Recovered %d chunks\n", len(chunks))

	collections := cs2.ListShardedCollections()
	fmt.Printf("  Recovered %d sharded collections\n", len(collections))

	// Verify specific data
	fmt.Println("\nVerifying recovered data:")
	if meta, err := cs2.GetShard("shard-1"); err == nil {
		fmt.Printf("  ✓ Shard %s found (host: %s)\n", meta.ID, meta.Host)
	}

	if chunk, err := cs2.GetChunk("chunk-1"); err == nil {
		fmt.Printf("  ✓ Chunk %s found (shard: %s, range: [%v, %v))\n",
			chunk.ID, chunk.ShardID, chunk.MinKey, chunk.MaxKey)
	}

	if config, err := cs2.GetCollectionSharding("mydb", "users"); err == nil {
		fmt.Printf("  ✓ Collection %s.%s found (shard key: %v)\n",
			config.Database, config.Collection, config.ShardKey.Fields)
	}

	stats := cs2.Stats()
	fmt.Println("\nFinal statistics:")
	fmt.Printf("  Total shards: %d\n", stats["total_shards"])
	fmt.Printf("  Active shards: %d\n", stats["active_shards"])
	fmt.Printf("  Total chunks: %d\n", stats["total_chunks"])
	fmt.Printf("  Sharded collections: %d\n", stats["sharded_collections"])
	fmt.Printf("  Metadata version: %d\n", stats["version"])
}

func setupDatabase(dataDir string) *database.Database {
	os.RemoveAll(dataDir)
	config := &database.Config{
		DataDir:        dataDir,
		BufferPoolSize: 100,
	}
	db, err := database.Open(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to open database: %v", err))
	}
	return db
}

func cleanup(db *database.Database, dataDir string) {
	if db != nil {
		db.Close()
	}
	os.RemoveAll(dataDir)
}
