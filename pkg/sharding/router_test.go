package sharding

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/database"
)

func TestNewShardRouter(t *testing.T) {
	sk := NewHashShardKey("user_id")
	router, err := NewShardRouter(sk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if router.shardKey != sk {
		t.Error("shard key not set correctly")
	}
}

func TestShardRouterAddShard(t *testing.T) {
	sk := NewHashShardKey("user_id")
	router, _ := NewShardRouter(sk)

	db1, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db1.Close()

	shard1 := NewShard("shard-1", db1, "localhost:27017")
	err := router.AddShard(shard1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try to add duplicate
	err = router.AddShard(shard1)
	if err == nil {
		t.Error("expected error for duplicate shard")
	}
}

func TestShardRouterRemoveShard(t *testing.T) {
	sk := NewHashShardKey("user_id")
	router, _ := NewShardRouter(sk)

	db1, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db1.Close()

	shard1 := NewShard("shard-1", db1, "localhost:27017")
	router.AddShard(shard1)

	err := router.RemoveShard("shard-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try to remove non-existent
	err = router.RemoveShard("shard-1")
	if err == nil {
		t.Error("expected error for non-existent shard")
	}
}

func TestShardRouterGetShard(t *testing.T) {
	sk := NewHashShardKey("user_id")
	router, _ := NewShardRouter(sk)

	db1, _ := database.Open(database.DefaultConfig("/tmp/test-router-remove"))
	defer db1.Close()

	shard1 := NewShard("shard-1", db1, "localhost:27017")
	router.AddShard(shard1)

	// Get existing shard
	shard, err := router.GetShard("shard-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shard.ID != "shard-1" {
		t.Errorf("unexpected shard ID: %s", shard.ID)
	}

	// Get non-existent shard
	_, err = router.GetShard("shard-999")
	if err == nil {
		t.Error("expected error for non-existent shard")
	}
}

func TestShardRouterHashBasedRouting(t *testing.T) {
	sk := NewHashShardKey("user_id")
	router, _ := NewShardRouter(sk)

	// Add 3 shards
	db1, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db1.Close()
	db2, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db2.Close()
	db3, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db3.Close()

	router.AddShard(NewShard("shard-1", db1, "localhost:27017"))
	router.AddShard(NewShard("shard-2", db2, "localhost:27018"))
	router.AddShard(NewShard("shard-3", db3, "localhost:27019"))

	// Route same user_id multiple times - should always go to same shard
	doc := map[string]interface{}{"user_id": "user123", "name": "Alice"}
	shard1, err := router.Route(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	shard2, err := router.Route(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if shard1.ID != shard2.ID {
		t.Error("same document should route to same shard")
	}

	// Different user_id might route to different shard
	doc2 := map[string]interface{}{"user_id": "user456", "name": "Bob"}
	shard3, err := router.Route(doc2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Just verify it routes successfully (might be same or different shard)
	if shard3 == nil {
		t.Error("expected valid shard")
	}
}

func TestShardRouterRangeBasedRouting(t *testing.T) {
	sk := NewRangeShardKey("user_id")
	router, _ := NewShardRouter(sk)

	// Add shards
	db1, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db1.Close()
	db2, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db2.Close()

	router.AddShard(NewShard("shard-1", db1, "localhost:27017"))
	router.AddShard(NewShard("shard-2", db2, "localhost:27018"))

	// Initialize chunks
	chunk1, _ := router.CreateChunk("shard-1", int64(0), int64(1000))
	chunk2, _ := router.CreateChunk("shard-2", int64(1000), int64(2000))

	if chunk1 == nil || chunk2 == nil {
		t.Fatal("failed to create chunks")
	}

	// Route document with user_id in first chunk
	doc1 := map[string]interface{}{"user_id": int64(500), "name": "Alice"}
	shard, err := router.Route(doc1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shard.ID != "shard-1" {
		t.Errorf("expected shard-1, got %s", shard.ID)
	}

	// Route document with user_id in second chunk
	doc2 := map[string]interface{}{"user_id": int64(1500), "name": "Bob"}
	shard, err = router.Route(doc2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shard.ID != "shard-2" {
		t.Errorf("expected shard-2, got %s", shard.ID)
	}
}

func TestShardRouterCreateChunk(t *testing.T) {
	sk := NewRangeShardKey("user_id")
	router, _ := NewShardRouter(sk)

	db1, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db1.Close()

	router.AddShard(NewShard("shard-1", db1, "localhost:27017"))

	chunk, err := router.CreateChunk("shard-1", int64(0), int64(1000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chunk.ShardID != "shard-1" {
		t.Errorf("unexpected shard ID: %s", chunk.ShardID)
	}

	// Verify chunk is in router
	chunks := router.GetChunks()
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
}

func TestShardRouterSplitChunk(t *testing.T) {
	sk := NewRangeShardKey("user_id")
	router, _ := NewShardRouter(sk)

	db1, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db1.Close()

	router.AddShard(NewShard("shard-1", db1, "localhost:27017"))
	chunk, _ := router.CreateChunk("shard-1", int64(0), int64(1000))

	left, right, err := router.SplitChunk(chunk.ID, int64(500))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if left.MaxKey != int64(500) {
		t.Errorf("unexpected left max key: %v", left.MaxKey)
	}
	if right.MinKey != int64(500) {
		t.Errorf("unexpected right min key: %v", right.MinKey)
	}

	// Verify 2 chunks now exist
	chunks := router.GetChunks()
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}
}

func TestShardRouterMoveChunk(t *testing.T) {
	sk := NewRangeShardKey("user_id")
	router, _ := NewShardRouter(sk)

	db1, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db1.Close()
	db2, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db2.Close()

	router.AddShard(NewShard("shard-1", db1, "localhost:27017"))
	router.AddShard(NewShard("shard-2", db2, "localhost:27018"))

	chunk, _ := router.CreateChunk("shard-1", int64(0), int64(1000))

	err := router.MoveChunk(chunk.ID, "shard-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify chunk now belongs to shard-2
	chunks := router.GetChunksForShard("shard-2")
	if len(chunks) != 1 || chunks[0].ID != chunk.ID {
		t.Error("chunk not moved to shard-2")
	}
}

func TestShardRouterRouteQuery(t *testing.T) {
	sk := NewHashShardKey("user_id")
	router, _ := NewShardRouter(sk)

	db1, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db1.Close()
	db2, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db2.Close()

	router.AddShard(NewShard("shard-1", db1, "localhost:27017"))
	router.AddShard(NewShard("shard-2", db2, "localhost:27018"))

	// Query with shard key - should route to single shard
	filter := map[string]interface{}{"user_id": "user123"}
	shards, err := router.RouteQuery(filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shards) != 1 {
		t.Errorf("expected 1 shard, got %d", len(shards))
	}

	// Query without shard key - should route to all shards
	filter = map[string]interface{}{"name": "Alice"}
	shards, err = router.RouteQuery(filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shards) != 2 {
		t.Errorf("expected 2 shards, got %d", len(shards))
	}
}

func TestShardRouterStats(t *testing.T) {
	sk := NewHashShardKey("user_id")
	router, _ := NewShardRouter(sk)

	db1, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db1.Close()

	router.AddShard(NewShard("shard-1", db1, "localhost:27017"))

	stats := router.Stats()
	if stats["total_shards"] != 1 {
		t.Errorf("unexpected total shards: %v", stats["total_shards"])
	}
	if stats["shard_type"] != "hash" {
		t.Errorf("unexpected shard type: %v", stats["shard_type"])
	}
}

func TestShardRouterHashDistribution(t *testing.T) {
	sk := NewHashShardKey("user_id")
	router, _ := NewShardRouter(sk)

	// Add 4 shards
	for i := 1; i <= 4; i++ {
		db, _ := database.Open(database.DefaultConfig("/tmp/test-router-dist" + string(rune('0'+i))))
		defer db.Close()
		router.AddShard(NewShard(ShardID("shard-"+string(rune('0'+i))), db, "localhost:27017"))
	}

	// Route many documents and check distribution
	shardCounts := make(map[ShardID]int)
	for i := 0; i < 1000; i++ {
		doc := map[string]interface{}{"user_id": int64(i)}
		shard, err := router.Route(doc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		shardCounts[shard.ID]++
	}

	// Each shard should get roughly 250 documents (allow 20% variance)
	for shardID, count := range shardCounts {
		if count < 200 || count > 300 {
			t.Errorf("shard %s has poor distribution: %d documents", shardID, count)
		}
	}
}

func TestShardRouterCompoundKey(t *testing.T) {
	sk := NewRangeShardKey("country", "user_id")
	router, _ := NewShardRouter(sk)

	db1, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db1.Close()
	db2, _ := database.Open(database.DefaultConfig("/tmp/test-router-shard1"))
	defer db2.Close()

	router.AddShard(NewShard("shard-1", db1, "localhost:27017"))
	router.AddShard(NewShard("shard-2", db2, "localhost:27018"))

	// Create chunks based on country
	router.CreateChunk("shard-1",
		map[string]interface{}{"country": "US", "user_id": int64(0)},
		map[string]interface{}{"country": "US", "user_id": int64(1000)})

	router.CreateChunk("shard-2",
		map[string]interface{}{"country": "UK", "user_id": int64(0)},
		map[string]interface{}{"country": "UK", "user_id": int64(1000)})

	// Route US user
	doc := map[string]interface{}{"country": "US", "user_id": int64(500), "name": "Alice"}
	shard, err := router.Route(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shard.ID != "shard-1" {
		t.Errorf("expected shard-1 for US user, got %s", shard.ID)
	}

	// Route UK user
	doc = map[string]interface{}{"country": "UK", "user_id": int64(500), "name": "Bob"}
	shard, err = router.Route(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shard.ID != "shard-2" {
		t.Errorf("expected shard-2 for UK user, got %s", shard.ID)
	}
}

func TestShardRouterGetAllShards(t *testing.T) {
	shardKey := NewHashShardKey("user_id")
	router, err := NewShardRouter(shardKey)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	// Initially no shards
	shards := router.GetAllShards()
	if len(shards) != 0 {
		t.Errorf("Expected 0 shards, got %d", len(shards))
	}

	// Add some shards
	shard1 := NewShard("shard-1", nil, "localhost:27017")
	shard2 := NewShard("shard-2", nil, "localhost:27018")
	shard3 := NewShard("shard-3", nil, "localhost:27019")

	router.AddShard(shard1)
	router.AddShard(shard2)
	router.AddShard(shard3)

	// Get all shards
	shards = router.GetAllShards()
	if len(shards) != 3 {
		t.Errorf("Expected 3 shards, got %d", len(shards))
	}

	// Verify shard IDs
	shardIDs := make(map[ShardID]bool)
	for _, s := range shards {
		shardIDs[s.ID] = true
	}

	if !shardIDs["shard-1"] || !shardIDs["shard-2"] || !shardIDs["shard-3"] {
		t.Error("Missing expected shard IDs in list")
	}
}

func TestShardRouterInitializeRangeSharding(t *testing.T) {
	shardKey := NewRangeShardKey("user_id")
	router, err := NewShardRouter(shardKey)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	// Add shards
	shard1 := NewShard("shard-1", nil, "localhost:27017")
	shard2 := NewShard("shard-2", nil, "localhost:27018")
	router.AddShard(shard1)
	router.AddShard(shard2)

	// Create initial chunks
	chunk1 := NewChunk("chunk-1", "shard-1", int64(0), int64(500))
	chunk2 := NewChunk("chunk-2", "shard-2", int64(500), int64(1000))

	initialChunks := []*Chunk{chunk1, chunk2}

	// Initialize range sharding
	err = router.InitializeRangeSharding(initialChunks)
	if err != nil {
		t.Fatalf("Failed to initialize range sharding: %v", err)
	}

	// Test routing with the initialized chunks
	doc := map[string]interface{}{"user_id": int64(250)}
	shard, err := router.Route(doc)
	if err != nil {
		t.Fatalf("Failed to route document: %v", err)
	}
	if shard.ID != "shard-1" {
		t.Errorf("Expected shard-1, got %s", shard.ID)
	}

	doc = map[string]interface{}{"user_id": int64(750)}
	shard, err = router.Route(doc)
	if err != nil {
		t.Fatalf("Failed to route document: %v", err)
	}
	if shard.ID != "shard-2" {
		t.Errorf("Expected shard-2, got %s", shard.ID)
	}
}

func TestShardRouterInitializeRangeShardingErrors(t *testing.T) {
	// Test with hash-based router (should fail)
	hashKey := NewHashShardKey("user_id")
	router, err := NewShardRouter(hashKey)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	shard1 := NewShard("shard-1", nil, "localhost:27017")
	router.AddShard(shard1)

	chunk1 := NewChunk("chunk-1", "shard-1", int64(0), int64(500))
	err = router.InitializeRangeSharding([]*Chunk{chunk1})
	if err == nil {
		t.Error("Expected error when initializing range sharding on hash-based router")
	}

	// Test with non-existent shard
	rangeKey := NewRangeShardKey("user_id")
	router2, err := NewShardRouter(rangeKey)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	chunk2 := NewChunk("chunk-2", "nonexistent-shard", int64(0), int64(500))
	err = router2.InitializeRangeSharding([]*Chunk{chunk2})
	if err == nil {
		t.Error("Expected error when chunk references non-existent shard")
	}
}
