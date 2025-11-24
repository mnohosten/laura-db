package main

import (
	"fmt"
	"os"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/sharding"
)

func main() {
	fmt.Println("=== LauraDB Sharding Demo ===")
	fmt.Println()

	// Demo 1: Hash-based sharding
	demo1HashBasedSharding()

	fmt.Println()

	// Demo 2: Range-based sharding
	demo2RangeBasedSharding()

	fmt.Println()

	// Demo 3: Compound shard key
	demo3CompoundShardKey()

	fmt.Println()

	// Demo 4: Chunk splitting
	demo4ChunkSplitting()

	fmt.Println()

	// Demo 5: Chunk migration
	demo5ChunkMigration()

	fmt.Println("\n=== Demo Complete ===")
}

func demo1HashBasedSharding() {
	fmt.Println("Demo 1: Hash-based Sharding")
	fmt.Println("----------------------------")

	// Create shard key
	shardKey := sharding.NewHashShardKey("user_id")
	router, err := sharding.NewShardRouter(shardKey)
	if err != nil {
		fmt.Printf("Error creating router: %v\n", err)
		return
	}

	// Create 3 shards
	cleanup1 := setupShard(router, "shard-1", "/tmp/sharding-demo-hash-1", "localhost:27017")
	cleanup2 := setupShard(router, "shard-2", "/tmp/sharding-demo-hash-2", "localhost:27018")
	cleanup3 := setupShard(router, "shard-3", "/tmp/sharding-demo-hash-3", "localhost:27019")
	defer cleanup1()
	defer cleanup2()
	defer cleanup3()

	fmt.Println("Created 3 shards with hash-based routing")

	// Insert documents - they will be distributed across shards based on hash
	users := []map[string]interface{}{
		{"user_id": "user001", "name": "Alice", "country": "US"},
		{"user_id": "user002", "name": "Bob", "country": "UK"},
		{"user_id": "user003", "name": "Charlie", "country": "CA"},
		{"user_id": "user004", "name": "Diana", "country": "US"},
		{"user_id": "user005", "name": "Eve", "country": "UK"},
		{"user_id": "user006", "name": "Frank", "country": "CA"},
		{"user_id": "user007", "name": "Grace", "country": "US"},
		{"user_id": "user008", "name": "Henry", "country": "UK"},
	}

	shardCounts := make(map[sharding.ShardID]int)
	for _, user := range users {
		shard, err := router.Route(user)
		if err != nil {
			fmt.Printf("Error routing user %v: %v\n", user["user_id"], err)
			continue
		}
		shardCounts[shard.ID]++
		fmt.Printf("  User %s -> %s\n", user["user_id"], shard.ID)
	}

	fmt.Println("\nDistribution across shards:")
	for shardID, count := range shardCounts {
		fmt.Printf("  %s: %d documents (%.1f%%)\n", shardID, count, float64(count)/float64(len(users))*100)
	}

	// Verify same user_id always routes to same shard
	shard1, _ := router.Route(map[string]interface{}{"user_id": "user001"})
	shard2, _ := router.Route(map[string]interface{}{"user_id": "user001"})
	fmt.Printf("\nConsistency check: user001 routes to %s and %s (should be same)\n", shard1.ID, shard2.ID)
}

func demo2RangeBasedSharding() {
	fmt.Println("Demo 2: Range-based Sharding")
	fmt.Println("----------------------------")

	// Create shard key
	shardKey := sharding.NewRangeShardKey("user_id")
	router, err := sharding.NewShardRouter(shardKey)
	if err != nil {
		fmt.Printf("Error creating router: %v\n", err)
		return
	}

	// Create 3 shards
	cleanup1 := setupShard(router, "shard-1", "/tmp/sharding-demo-range-1", "localhost:27017")
	cleanup2 := setupShard(router, "shard-2", "/tmp/sharding-demo-range-2", "localhost:27018")
	cleanup3 := setupShard(router, "shard-3", "/tmp/sharding-demo-range-3", "localhost:27019")
	defer cleanup1()
	defer cleanup2()
	defer cleanup3()

	// Define chunks (ranges)
	chunk1, _ := router.CreateChunk("shard-1", int64(0), int64(1000))
	chunk2, _ := router.CreateChunk("shard-2", int64(1000), int64(2000))
	chunk3, _ := router.CreateChunk("shard-3", int64(2000), int64(3000))

	fmt.Println("Created 3 shards with range-based chunks:")
	fmt.Printf("  Chunk %s: [0, 1000) -> shard-1\n", chunk1.ID)
	fmt.Printf("  Chunk %s: [1000, 2000) -> shard-2\n", chunk2.ID)
	fmt.Printf("  Chunk %s: [2000, 3000) -> shard-3\n", chunk3.ID)

	// Insert documents with different user_id ranges
	users := []map[string]interface{}{
		{"user_id": int64(500), "name": "Alice"},
		{"user_id": int64(1500), "name": "Bob"},
		{"user_id": int64(2500), "name": "Charlie"},
		{"user_id": int64(750), "name": "Diana"},
		{"user_id": int64(1750), "name": "Eve"},
		{"user_id": int64(2750), "name": "Frank"},
	}

	fmt.Println("\nRouting documents:")
	for _, user := range users {
		shard, err := router.Route(user)
		if err != nil {
			fmt.Printf("Error routing user_id %v: %v\n", user["user_id"], err)
			continue
		}
		fmt.Printf("  user_id=%d (%s) -> %s\n", user["user_id"], user["name"], shard.ID)
	}
}

func demo3CompoundShardKey() {
	fmt.Println("Demo 3: Compound Shard Key (country + user_id)")
	fmt.Println("-----------------------------------------------")

	// Create compound shard key
	shardKey := sharding.NewRangeShardKey("country", "user_id")
	router, err := sharding.NewShardRouter(shardKey)
	if err != nil {
		fmt.Printf("Error creating router: %v\n", err)
		return
	}

	// Create shards
	cleanup1 := setupShard(router, "us-shard", "/tmp/sharding-demo-compound-us", "localhost:27017")
	cleanup2 := setupShard(router, "uk-shard", "/tmp/sharding-demo-compound-uk", "localhost:27018")
	cleanup3 := setupShard(router, "ca-shard", "/tmp/sharding-demo-compound-ca", "localhost:27019")
	defer cleanup1()
	defer cleanup2()
	defer cleanup3()

	// Define chunks per country
	router.CreateChunk("us-shard",
		map[string]interface{}{"country": "US", "user_id": int64(0)},
		map[string]interface{}{"country": "US", "user_id": int64(10000)})

	router.CreateChunk("uk-shard",
		map[string]interface{}{"country": "UK", "user_id": int64(0)},
		map[string]interface{}{"country": "UK", "user_id": int64(10000)})

	router.CreateChunk("ca-shard",
		map[string]interface{}{"country": "CA", "user_id": int64(0)},
		map[string]interface{}{"country": "CA", "user_id": int64(10000)})

	fmt.Println("Created geo-distributed shards:")
	fmt.Println("  US users -> us-shard")
	fmt.Println("  UK users -> uk-shard")
	fmt.Println("  CA users -> ca-shard")

	// Insert documents
	users := []map[string]interface{}{
		{"country": "US", "user_id": int64(1001), "name": "Alice"},
		{"country": "UK", "user_id": int64(2001), "name": "Bob"},
		{"country": "CA", "user_id": int64(3001), "name": "Charlie"},
		{"country": "US", "user_id": int64(1002), "name": "Diana"},
		{"country": "UK", "user_id": int64(2002), "name": "Eve"},
	}

	fmt.Println("\nRouting documents:")
	for _, user := range users {
		shard, err := router.Route(user)
		if err != nil {
			fmt.Printf("Error routing user: %v\n", err)
			continue
		}
		fmt.Printf("  %s user_id=%d (%s) -> %s\n",
			user["country"], user["user_id"], user["name"], shard.ID)
	}

	// Query routing
	fmt.Println("\nQuery routing:")

	// Query with shard key - routes to single shard
	filter := map[string]interface{}{"country": "US", "user_id": int64(1001)}
	shards, _ := router.RouteQuery(filter)
	fmt.Printf("  Query {country: US, user_id: 1001} -> %d shard(s): %s\n",
		len(shards), shards[0].ID)

	// Query without shard key - routes to all shards
	filter = map[string]interface{}{"name": "Alice"}
	shards, _ = router.RouteQuery(filter)
	fmt.Printf("  Query {name: Alice} -> %d shard(s) (scatter-gather)\n", len(shards))
}

func demo4ChunkSplitting() {
	fmt.Println("Demo 4: Chunk Splitting")
	fmt.Println("------------------------")

	shardKey := sharding.NewRangeShardKey("user_id")
	router, _ := sharding.NewShardRouter(shardKey)

	cleanup := setupShard(router, "shard-1", "/tmp/sharding-demo-split", "localhost:27017")
	defer cleanup()

	// Create initial chunk
	chunk, _ := router.CreateChunk("shard-1", int64(0), int64(1000))
	fmt.Printf("Initial chunk: %s [0, 1000) on shard-1\n", chunk.ID)

	// Simulate chunk growing too large and needing split
	chunk.UpdateStats(10000, 10485760) // 10,000 docs, 10MB
	fmt.Printf("  Stats: %d documents, %d bytes\n", chunk.Count, chunk.Size)

	// Split chunk at midpoint
	fmt.Println("\nSplitting chunk at 500...")
	left, right, err := router.SplitChunk(chunk.ID, int64(500))
	if err != nil {
		fmt.Printf("Error splitting chunk: %v\n", err)
		return
	}

	fmt.Printf("  Left chunk: %s [%v, %v) on shard-1\n", left.ID, left.MinKey, left.MaxKey)
	fmt.Printf("  Right chunk: %s [%v, %v) on shard-1\n", right.ID, right.MinKey, right.MaxKey)

	// Show all chunks
	chunks := router.GetChunks()
	fmt.Printf("\nTotal chunks after split: %d\n", len(chunks))
	for _, c := range chunks {
		fmt.Printf("  %s: [%v, %v) on %s\n", c.ID, c.MinKey, c.MaxKey, c.ShardID)
	}
}

func demo5ChunkMigration() {
	fmt.Println("Demo 5: Chunk Migration (Balancing)")
	fmt.Println("------------------------------------")

	shardKey := sharding.NewRangeShardKey("user_id")
	router, _ := sharding.NewShardRouter(shardKey)

	// Create 2 shards
	cleanup1 := setupShard(router, "shard-1", "/tmp/sharding-demo-migrate-1", "localhost:27017")
	cleanup2 := setupShard(router, "shard-2", "/tmp/sharding-demo-migrate-2", "localhost:27018")
	defer cleanup1()
	defer cleanup2()

	// Create chunks all on shard-1 (imbalanced)
	chunk1, _ := router.CreateChunk("shard-1", int64(0), int64(1000))
	chunk2, _ := router.CreateChunk("shard-1", int64(1000), int64(2000))
	chunk3, _ := router.CreateChunk("shard-1", int64(2000), int64(3000))

	fmt.Println("Initial distribution (imbalanced):")
	printShardDistribution(router)

	// Simulate chunk growth
	chunk1.UpdateStats(5000, 5242880)
	chunk2.UpdateStats(8000, 8388608)
	chunk3.UpdateStats(3000, 3145728)

	// Balance by moving chunk to shard-2
	fmt.Println("\nMigrating chunk-2 to shard-2 for balance...")
	err := router.MoveChunk(chunk2.ID, "shard-2")
	if err != nil {
		fmt.Printf("Error moving chunk: %v\n", err)
		return
	}

	fmt.Println("\nFinal distribution (balanced):")
	printShardDistribution(router)
}

// Helper functions

func setupShard(router *sharding.ShardRouter, id sharding.ShardID, path, host string) func() {
	// Clean up old data
	os.RemoveAll(path)

	db, err := database.Open(database.DefaultConfig(path))
	if err != nil {
		panic(err)
	}

	shard := sharding.NewShard(id, db, host)
	router.AddShard(shard)

	return func() {
		db.Close()
		os.RemoveAll(path)
	}
}

func printShardDistribution(router *sharding.ShardRouter) {
	shards := router.GetAllShards()
	for _, shard := range shards {
		chunks := router.GetChunksForShard(shard.ID)
		fmt.Printf("  %s: %d chunk(s)\n", shard.ID, len(chunks))
		for _, chunk := range chunks {
			fmt.Printf("    %s: [%v, %v) - %d docs, %d bytes\n",
				chunk.ID, chunk.MinKey, chunk.MaxKey, chunk.Count, chunk.Size)
		}
	}
}
