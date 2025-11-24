package sharding

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/database"
)

func TestNewShard(t *testing.T) {
	db, err := database.Open(database.DefaultConfig("/tmp/test-shard"))
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	shard := NewShard("shard-1", db, "localhost:27017")
	if shard.ID != "shard-1" {
		t.Errorf("unexpected shard ID: %s", shard.ID)
	}
	if shard.Host != "localhost:27017" {
		t.Errorf("unexpected host: %s", shard.Host)
	}
	if shard.Database != db {
		t.Error("database not set correctly")
	}
}

func TestShardTags(t *testing.T) {
	db, _ := database.Open(database.DefaultConfig("/tmp/test-shard-tags"))
	defer db.Close()

	shard := NewShard("shard-1", db, "localhost:27017")

	// Set tags
	shard.SetTag("datacenter", "us-east-1")
	shard.SetTag("rack", "r1")

	// Get tags
	dc, ok := shard.GetTag("datacenter")
	if !ok || dc != "us-east-1" {
		t.Errorf("unexpected datacenter tag: %s", dc)
	}

	rack, ok := shard.GetTag("rack")
	if !ok || rack != "r1" {
		t.Errorf("unexpected rack tag: %s", rack)
	}

	// Non-existent tag
	_, ok = shard.GetTag("nonexistent")
	if ok {
		t.Error("expected false for non-existent tag")
	}
}

func TestShardMatchesTags(t *testing.T) {
	db, _ := database.Open(database.DefaultConfig("/tmp/test-shard-match"))
	defer db.Close()

	shard := NewShard("shard-1", db, "localhost:27017")
	shard.SetTag("datacenter", "us-east-1")
	shard.SetTag("rack", "r1")

	// Match all tags
	tags := map[string]string{
		"datacenter": "us-east-1",
		"rack":       "r1",
	}
	if !shard.MatchesTags(tags) {
		t.Error("expected shard to match tags")
	}

	// Match subset
	tags = map[string]string{
		"datacenter": "us-east-1",
	}
	if !shard.MatchesTags(tags) {
		t.Error("expected shard to match subset of tags")
	}

	// No match
	tags = map[string]string{
		"datacenter": "us-west-1",
	}
	if shard.MatchesTags(tags) {
		t.Error("expected shard not to match different tags")
	}
}

func TestNewChunk(t *testing.T) {
	chunk := NewChunk("chunk-1", "shard-1", int64(0), int64(1000))
	if chunk.ID != "chunk-1" {
		t.Errorf("unexpected chunk ID: %s", chunk.ID)
	}
	if chunk.ShardID != "shard-1" {
		t.Errorf("unexpected shard ID: %s", chunk.ShardID)
	}
	if chunk.MinKey != int64(0) {
		t.Errorf("unexpected min key: %v", chunk.MinKey)
	}
	if chunk.MaxKey != int64(1000) {
		t.Errorf("unexpected max key: %v", chunk.MaxKey)
	}
}

func TestChunkContains(t *testing.T) {
	sk := NewRangeShardKey("user_id")
	chunk := NewChunk("chunk-1", "shard-1", int64(100), int64(200))

	// Value within range
	if !chunk.Contains(sk, int64(150)) {
		t.Error("expected chunk to contain 150")
	}

	// Value at min boundary (inclusive)
	if !chunk.Contains(sk, int64(100)) {
		t.Error("expected chunk to contain min value 100")
	}

	// Value at max boundary (exclusive)
	if chunk.Contains(sk, int64(200)) {
		t.Error("expected chunk not to contain max value 200")
	}

	// Value below range
	if chunk.Contains(sk, int64(50)) {
		t.Error("expected chunk not to contain 50")
	}

	// Value above range
	if chunk.Contains(sk, int64(250)) {
		t.Error("expected chunk not to contain 250")
	}
}

func TestChunkUnbounded(t *testing.T) {
	sk := NewRangeShardKey("user_id")

	// Chunk with no min key (unbounded below)
	chunk := NewChunk("chunk-1", "shard-1", nil, int64(1000))
	if !chunk.Contains(sk, int64(-999999)) {
		t.Error("expected unbounded chunk to contain very small value")
	}
	if chunk.Contains(sk, int64(1000)) {
		t.Error("expected chunk not to contain max value")
	}

	// Chunk with no max key (unbounded above)
	chunk = NewChunk("chunk-2", "shard-1", int64(1000), nil)
	if !chunk.Contains(sk, int64(999999)) {
		t.Error("expected unbounded chunk to contain very large value")
	}
	if !chunk.Contains(sk, int64(1000)) {
		t.Error("expected chunk to contain min value")
	}
}

func TestChunkUpdateStats(t *testing.T) {
	chunk := NewChunk("chunk-1", "shard-1", int64(0), int64(1000))

	chunk.UpdateStats(500, 1024000)
	if chunk.Count != 500 {
		t.Errorf("unexpected count: %d", chunk.Count)
	}
	if chunk.Size != 1024000 {
		t.Errorf("unexpected size: %d", chunk.Size)
	}
}

func TestChunkIncrementCount(t *testing.T) {
	chunk := NewChunk("chunk-1", "shard-1", int64(0), int64(1000))

	chunk.IncrementCount(10)
	if chunk.Count != 10 {
		t.Errorf("unexpected count: %d", chunk.Count)
	}

	chunk.IncrementCount(5)
	if chunk.Count != 15 {
		t.Errorf("unexpected count: %d", chunk.Count)
	}

	chunk.IncrementCount(-3)
	if chunk.Count != 12 {
		t.Errorf("unexpected count: %d", chunk.Count)
	}
}

func TestNewChunkManager(t *testing.T) {
	sk := NewRangeShardKey("user_id")
	cm := NewChunkManager(sk)

	if cm.shardKey != sk {
		t.Error("shard key not set correctly")
	}
	if len(cm.chunks) != 0 {
		t.Error("expected no chunks initially")
	}
}

func TestChunkManagerAddChunk(t *testing.T) {
	sk := NewRangeShardKey("user_id")
	cm := NewChunkManager(sk)

	chunk1 := NewChunk("chunk-1", "shard-1", int64(0), int64(1000))
	err := cm.AddChunk(chunk1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	chunks := cm.GetAllChunks()
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
}

func TestChunkManagerOverlapDetection(t *testing.T) {
	sk := NewRangeShardKey("user_id")
	cm := NewChunkManager(sk)

	chunk1 := NewChunk("chunk-1", "shard-1", int64(0), int64(1000))
	cm.AddChunk(chunk1)

	// Overlapping chunk (same shard)
	chunk2 := NewChunk("chunk-2", "shard-1", int64(500), int64(1500))
	err := cm.AddChunk(chunk2)
	if err == nil {
		t.Error("expected error for overlapping chunk")
	}

	// Non-overlapping chunk
	chunk3 := NewChunk("chunk-3", "shard-1", int64(1000), int64(2000))
	err = cm.AddChunk(chunk3)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChunkManagerFindChunk(t *testing.T) {
	sk := NewRangeShardKey("user_id")
	cm := NewChunkManager(sk)

	chunk1 := NewChunk("chunk-1", "shard-1", int64(0), int64(1000))
	chunk2 := NewChunk("chunk-2", "shard-2", int64(1000), int64(2000))
	cm.AddChunk(chunk1)
	cm.AddChunk(chunk2)

	// Find in first chunk
	found := cm.FindChunk(int64(500))
	if found == nil || found.ID != "chunk-1" {
		t.Error("expected to find chunk-1")
	}

	// Find in second chunk
	found = cm.FindChunk(int64(1500))
	if found == nil || found.ID != "chunk-2" {
		t.Error("expected to find chunk-2")
	}

	// Not found
	found = cm.FindChunk(int64(2500))
	if found != nil {
		t.Error("expected not to find chunk")
	}
}

func TestChunkManagerGetChunksForShard(t *testing.T) {
	sk := NewRangeShardKey("user_id")
	cm := NewChunkManager(sk)

	chunk1 := NewChunk("chunk-1", "shard-1", int64(0), int64(1000))
	chunk2 := NewChunk("chunk-2", "shard-1", int64(1000), int64(2000))
	chunk3 := NewChunk("chunk-3", "shard-2", int64(0), int64(1000))
	cm.AddChunk(chunk1)
	cm.AddChunk(chunk2)
	cm.AddChunk(chunk3)

	// Get chunks for shard-1
	chunks := cm.GetChunksForShard("shard-1")
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks for shard-1, got %d", len(chunks))
	}

	// Get chunks for shard-2
	chunks = cm.GetChunksForShard("shard-2")
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for shard-2, got %d", len(chunks))
	}
}

func TestChunkManagerSplitChunk(t *testing.T) {
	sk := NewRangeShardKey("user_id")
	cm := NewChunkManager(sk)

	chunk := NewChunk("chunk-1", "shard-1", int64(0), int64(1000))
	cm.AddChunk(chunk)

	// Split at 500
	left, right, err := cm.SplitChunk("chunk-1", int64(500))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if left.MinKey != int64(0) || left.MaxKey != int64(500) {
		t.Errorf("unexpected left chunk range: [%v, %v)", left.MinKey, left.MaxKey)
	}

	if right.MinKey != int64(500) || right.MaxKey != int64(1000) {
		t.Errorf("unexpected right chunk range: [%v, %v)", right.MinKey, right.MaxKey)
	}

	// Original chunk should be replaced
	chunks := cm.GetAllChunks()
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks after split, got %d", len(chunks))
	}
}

func TestChunkManagerMoveChunk(t *testing.T) {
	sk := NewRangeShardKey("user_id")
	cm := NewChunkManager(sk)

	chunk := NewChunk("chunk-1", "shard-1", int64(0), int64(1000))
	cm.AddChunk(chunk)

	// Move to shard-2
	err := cm.MoveChunk("chunk-1", "shard-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	chunks := cm.GetChunksForShard("shard-2")
	if len(chunks) != 1 || chunks[0].ID != "chunk-1" {
		t.Error("chunk not moved to shard-2")
	}

	chunks = cm.GetChunksForShard("shard-1")
	if len(chunks) != 0 {
		t.Error("chunk still on shard-1")
	}
}

func TestChunkStats(t *testing.T) {
	chunk := NewChunk("chunk-1", "shard-1", int64(0), int64(100))
	chunk.UpdateStats(int64(150), int64(5000))
	chunk.IncrementCount(int64(10))

	stats := chunk.Stats()

	if stats["chunk_id"].(string) != "chunk-1" {
		t.Errorf("Expected chunk_id 'chunk-1', got %v", stats["chunk_id"])
	}

	if stats["shard_id"].(ShardID) != "shard-1" {
		t.Errorf("Expected shard_id 'shard-1', got %v", stats["shard_id"])
	}

	if stats["min_key"].(int64) != 0 {
		t.Errorf("Expected min_key 0, got %v", stats["min_key"])
	}

	if stats["max_key"].(int64) != 100 {
		t.Errorf("Expected max_key 100, got %v", stats["max_key"])
	}

	if stats["count"].(int64) != 160 {
		t.Errorf("Expected count 160 (150 + 10), got %v", stats["count"])
	}

	if stats["size"].(int64) != 5000 {
		t.Errorf("Expected size 5000, got %v", stats["size"])
	}
}

func TestChunkManagerStats(t *testing.T) {
	shardKey := NewRangeShardKey("user_id")
	cm := NewChunkManager(shardKey)

	// Add some chunks
	chunk1 := NewChunk("chunk-1", "shard-1", int64(0), int64(100))
	chunk1.UpdateStats(int64(50), int64(2000))
	cm.AddChunk(chunk1)

	chunk2 := NewChunk("chunk-2", "shard-2", int64(100), int64(200))
	chunk2.UpdateStats(int64(75), int64(3000))
	cm.AddChunk(chunk2)

	stats := cm.Stats()

	if stats["total_chunks"].(int) != 2 {
		t.Errorf("Expected 2 total chunks, got %v", stats["total_chunks"])
	}

	chunkStats := stats["chunks"].([]map[string]interface{})
	if len(chunkStats) != 2 {
		t.Errorf("Expected 2 chunk stats, got %d", len(chunkStats))
	}

	// Verify chunk stats are included
	foundChunk1 := false
	foundChunk2 := false
	for _, cs := range chunkStats {
		if cs["chunk_id"].(string) == "chunk-1" {
			foundChunk1 = true
			if cs["count"].(int64) != 50 {
				t.Errorf("Expected chunk-1 count 50, got %v", cs["count"])
			}
		}
		if cs["chunk_id"].(string) == "chunk-2" {
			foundChunk2 = true
			if cs["count"].(int64) != 75 {
				t.Errorf("Expected chunk-2 count 75, got %v", cs["count"])
			}
		}
	}

	if !foundChunk1 || !foundChunk2 {
		t.Error("Missing expected chunk stats")
	}
}
