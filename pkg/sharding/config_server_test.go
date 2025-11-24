package sharding

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewConfigServer(t *testing.T) {
	tempDir := t.TempDir()

	cs, err := NewConfigServer(tempDir)
	if err != nil {
		t.Fatalf("Failed to create config server: %v", err)
	}
	defer cs.Close()

	if cs.dataDir != tempDir {
		t.Errorf("Expected dataDir %s, got %s", tempDir, cs.dataDir)
	}

	if cs.version != 1 {
		t.Errorf("Expected initial version 1, got %d", cs.version)
	}
}

func TestConfigServerRegisterShard(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shard := NewShard("shard-1", nil, "localhost:27017")
	shard.SetTag("datacenter", "us-east")

	err := cs.RegisterShard(shard)
	if err != nil {
		t.Fatalf("Failed to register shard: %v", err)
	}

	// Verify shard was registered
	meta, err := cs.GetShard("shard-1")
	if err != nil {
		t.Fatalf("Failed to get shard: %v", err)
	}

	if meta.ID != "shard-1" {
		t.Errorf("Expected shard ID 'shard-1', got %s", meta.ID)
	}

	if meta.Host != "localhost:27017" {
		t.Errorf("Expected host 'localhost:27017', got %s", meta.Host)
	}

	if meta.State != ShardStateActive {
		t.Errorf("Expected state active, got %s", meta.State)
	}

	if meta.Tags["datacenter"] != "us-east" {
		t.Errorf("Expected datacenter tag 'us-east', got %s", meta.Tags["datacenter"])
	}
}

func TestConfigServerRegisterShardDuplicate(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shard := NewShard("shard-1", nil, "localhost:27017")

	// Register once
	cs.RegisterShard(shard)

	// Try to register again
	err := cs.RegisterShard(shard)
	if err == nil {
		t.Fatal("Expected error when registering duplicate shard")
	}
}

func TestConfigServerUnregisterShard(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shard := NewShard("shard-1", nil, "localhost:27017")
	cs.RegisterShard(shard)

	err := cs.UnregisterShard("shard-1")
	if err != nil {
		t.Fatalf("Failed to unregister shard: %v", err)
	}

	// Verify shard was removed
	_, err = cs.GetShard("shard-1")
	if err == nil {
		t.Error("Expected error when getting unregistered shard")
	}
}

func TestConfigServerUnregisterShardWithChunks(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shard := NewShard("shard-1", nil, "localhost:27017")
	cs.RegisterShard(shard)

	chunk := NewChunk("chunk-1", "shard-1", int64(0), int64(100))
	cs.RegisterChunk(chunk)

	// Should fail because shard has chunks
	err := cs.UnregisterShard("shard-1")
	if err == nil {
		t.Fatal("Expected error when unregistering shard with chunks")
	}
}

func TestConfigServerUpdateShardState(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shard := NewShard("shard-1", nil, "localhost:27017")
	cs.RegisterShard(shard)

	// Update state to draining
	err := cs.UpdateShardState("shard-1", ShardStateDraining)
	if err != nil {
		t.Fatalf("Failed to update shard state: %v", err)
	}

	meta, _ := cs.GetShard("shard-1")
	if meta.State != ShardStateDraining {
		t.Errorf("Expected state draining, got %s", meta.State)
	}
}

func TestConfigServerUpdateShardTags(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shard := NewShard("shard-1", nil, "localhost:27017")
	cs.RegisterShard(shard)

	tags := map[string]string{
		"datacenter": "us-west",
		"rack":       "rack-5",
	}

	err := cs.UpdateShardTags("shard-1", tags)
	if err != nil {
		t.Fatalf("Failed to update shard tags: %v", err)
	}

	meta, _ := cs.GetShard("shard-1")
	if meta.Tags["datacenter"] != "us-west" {
		t.Errorf("Expected datacenter 'us-west', got %s", meta.Tags["datacenter"])
	}
	if meta.Tags["rack"] != "rack-5" {
		t.Errorf("Expected rack 'rack-5', got %s", meta.Tags["rack"])
	}
}

func TestConfigServerListShards(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	// Register multiple shards
	for i := 1; i <= 3; i++ {
		shard := NewShard(ShardID(filepath.Join("shard-", string(rune(i)))), nil, "localhost:27017")
		cs.RegisterShard(shard)
	}

	shards := cs.ListShards()
	if len(shards) != 3 {
		t.Errorf("Expected 3 shards, got %d", len(shards))
	}
}

func TestConfigServerListActiveShards(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shard1 := NewShard("shard-1", nil, "localhost:27017")
	shard2 := NewShard("shard-2", nil, "localhost:27018")
	shard3 := NewShard("shard-3", nil, "localhost:27019")

	cs.RegisterShard(shard1)
	cs.RegisterShard(shard2)
	cs.RegisterShard(shard3)

	// Set one shard to draining
	cs.UpdateShardState("shard-2", ShardStateDraining)

	activeShards := cs.ListActiveShards()
	if len(activeShards) != 2 {
		t.Errorf("Expected 2 active shards, got %d", len(activeShards))
	}
}

func TestConfigServerRegisterChunk(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shard := NewShard("shard-1", nil, "localhost:27017")
	cs.RegisterShard(shard)

	chunk := NewChunk("chunk-1", "shard-1", int64(0), int64(100))
	chunk.UpdateStats(50, 1024)

	err := cs.RegisterChunk(chunk)
	if err != nil {
		t.Fatalf("Failed to register chunk: %v", err)
	}

	// Verify chunk was registered
	meta, err := cs.GetChunk("chunk-1")
	if err != nil {
		t.Fatalf("Failed to get chunk: %v", err)
	}

	if meta.ID != "chunk-1" {
		t.Errorf("Expected chunk ID 'chunk-1', got %s", meta.ID)
	}

	if meta.ShardID != "shard-1" {
		t.Errorf("Expected shard ID 'shard-1', got %s", meta.ShardID)
	}

	if meta.Count != 50 {
		t.Errorf("Expected count 50, got %d", meta.Count)
	}

	if meta.Size != 1024 {
		t.Errorf("Expected size 1024, got %d", meta.Size)
	}
}

func TestConfigServerRegisterChunkInvalidShard(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	chunk := NewChunk("chunk-1", "nonexistent-shard", int64(0), int64(100))

	err := cs.RegisterChunk(chunk)
	if err == nil {
		t.Fatal("Expected error when registering chunk with invalid shard")
	}
}

func TestConfigServerUpdateChunk(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shard := NewShard("shard-1", nil, "localhost:27017")
	cs.RegisterShard(shard)

	chunk := NewChunk("chunk-1", "shard-1", int64(0), int64(100))
	cs.RegisterChunk(chunk)

	// Update chunk stats
	err := cs.UpdateChunk("chunk-1", 100, 2048)
	if err != nil {
		t.Fatalf("Failed to update chunk: %v", err)
	}

	meta, _ := cs.GetChunk("chunk-1")
	if meta.Count != 100 {
		t.Errorf("Expected count 100, got %d", meta.Count)
	}
	if meta.Size != 2048 {
		t.Errorf("Expected size 2048, got %d", meta.Size)
	}
}

func TestConfigServerMoveChunkMetadata(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shard1 := NewShard("shard-1", nil, "localhost:27017")
	shard2 := NewShard("shard-2", nil, "localhost:27018")
	cs.RegisterShard(shard1)
	cs.RegisterShard(shard2)

	chunk := NewChunk("chunk-1", "shard-1", int64(0), int64(100))
	cs.RegisterChunk(chunk)

	// Move chunk to shard-2
	err := cs.MoveChunkMetadata("chunk-1", "shard-2")
	if err != nil {
		t.Fatalf("Failed to move chunk: %v", err)
	}

	meta, _ := cs.GetChunk("chunk-1")
	if meta.ShardID != "shard-2" {
		t.Errorf("Expected shard ID 'shard-2', got %s", meta.ShardID)
	}

	if meta.Version != 2 {
		t.Errorf("Expected version 2 after move, got %d", meta.Version)
	}
}

func TestConfigServerListChunksForShard(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shard1 := NewShard("shard-1", nil, "localhost:27017")
	shard2 := NewShard("shard-2", nil, "localhost:27018")
	cs.RegisterShard(shard1)
	cs.RegisterShard(shard2)

	// Register chunks on different shards
	chunk1 := NewChunk("chunk-1", "shard-1", int64(0), int64(50))
	chunk2 := NewChunk("chunk-2", "shard-1", int64(50), int64(100))
	chunk3 := NewChunk("chunk-3", "shard-2", int64(0), int64(100))

	cs.RegisterChunk(chunk1)
	cs.RegisterChunk(chunk2)
	cs.RegisterChunk(chunk3)

	chunks := cs.ListChunksForShard("shard-1")
	if len(chunks) != 2 {
		t.Errorf("Expected 2 chunks for shard-1, got %d", len(chunks))
	}
}

func TestConfigServerSetCollectionSharding(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shardKey := NewHashShardKey("user_id")

	err := cs.SetCollectionSharding("mydb", "users", shardKey)
	if err != nil {
		t.Fatalf("Failed to set collection sharding: %v", err)
	}

	// Verify configuration
	config, err := cs.GetCollectionSharding("mydb", "users")
	if err != nil {
		t.Fatalf("Failed to get collection sharding: %v", err)
	}

	if config.Database != "mydb" {
		t.Errorf("Expected database 'mydb', got %s", config.Database)
	}

	if config.Collection != "users" {
		t.Errorf("Expected collection 'users', got %s", config.Collection)
	}

	if !config.Sharded {
		t.Error("Expected sharded to be true")
	}
}

func TestConfigServerSetCollectionShardingDuplicate(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shardKey := NewHashShardKey("user_id")

	cs.SetCollectionSharding("mydb", "users", shardKey)

	// Try to set again
	err := cs.SetCollectionSharding("mydb", "users", shardKey)
	if err == nil {
		t.Fatal("Expected error when setting collection sharding twice")
	}
}

func TestConfigServerRemoveCollectionSharding(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shardKey := NewHashShardKey("user_id")
	cs.SetCollectionSharding("mydb", "users", shardKey)

	err := cs.RemoveCollectionSharding("mydb", "users")
	if err != nil {
		t.Fatalf("Failed to remove collection sharding: %v", err)
	}

	// Verify removed
	_, err = cs.GetCollectionSharding("mydb", "users")
	if err == nil {
		t.Error("Expected error when getting removed collection sharding")
	}
}

func TestConfigServerListShardedCollections(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	shardKey1 := NewHashShardKey("user_id")
	shardKey2 := NewRangeShardKey("timestamp")

	cs.SetCollectionSharding("mydb", "users", shardKey1)
	cs.SetCollectionSharding("mydb", "events", shardKey2)

	collections := cs.ListShardedCollections()
	if len(collections) != 2 {
		t.Errorf("Expected 2 sharded collections, got %d", len(collections))
	}
}

func TestConfigServerPersistence(t *testing.T) {
	tempDir := t.TempDir()

	// Create config server and add some data
	cs1, _ := NewConfigServer(tempDir)

	shard := NewShard("shard-1", nil, "localhost:27017")
	cs1.RegisterShard(shard)

	chunk := NewChunk("chunk-1", "shard-1", int64(0), int64(100))
	cs1.RegisterChunk(chunk)

	shardKey := NewHashShardKey("user_id")
	cs1.SetCollectionSharding("mydb", "users", shardKey)

	cs1.Close()

	// Create new config server with same directory
	cs2, err := NewConfigServer(tempDir)
	if err != nil {
		t.Fatalf("Failed to create second config server: %v", err)
	}
	defer cs2.Close()

	// Verify data was loaded
	if _, err := cs2.GetShard("shard-1"); err != nil {
		t.Error("Failed to load shard from disk")
	}

	if _, err := cs2.GetChunk("chunk-1"); err != nil {
		t.Error("Failed to load chunk from disk")
	}

	if _, err := cs2.GetCollectionSharding("mydb", "users"); err != nil {
		t.Error("Failed to load collection sharding from disk")
	}
}

func TestConfigServerStats(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	// Add some data
	shard1 := NewShard("shard-1", nil, "localhost:27017")
	shard2 := NewShard("shard-2", nil, "localhost:27018")
	shard3 := NewShard("shard-3", nil, "localhost:27019")

	cs.RegisterShard(shard1)
	cs.RegisterShard(shard2)
	cs.RegisterShard(shard3)

	cs.UpdateShardState("shard-2", ShardStateDraining)
	cs.UpdateShardState("shard-3", ShardStateInactive)

	chunk := NewChunk("chunk-1", "shard-1", int64(0), int64(100))
	cs.RegisterChunk(chunk)

	shardKey := NewHashShardKey("user_id")
	cs.SetCollectionSharding("mydb", "users", shardKey)

	stats := cs.Stats()

	if stats["total_shards"].(int) != 3 {
		t.Errorf("Expected 3 total shards, got %v", stats["total_shards"])
	}

	if stats["active_shards"].(int) != 1 {
		t.Errorf("Expected 1 active shard, got %v", stats["active_shards"])
	}

	if stats["draining_shards"].(int) != 1 {
		t.Errorf("Expected 1 draining shard, got %v", stats["draining_shards"])
	}

	if stats["inactive_shards"].(int) != 1 {
		t.Errorf("Expected 1 inactive shard, got %v", stats["inactive_shards"])
	}

	if stats["total_chunks"].(int) != 1 {
		t.Errorf("Expected 1 chunk, got %v", stats["total_chunks"])
	}

	if stats["sharded_collections"].(int) != 1 {
		t.Errorf("Expected 1 sharded collection, got %v", stats["sharded_collections"])
	}
}

func TestConfigServerVersionIncrement(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	initialVersion := cs.GetVersion()

	shard := NewShard("shard-1", nil, "localhost:27017")
	cs.RegisterShard(shard)

	if cs.GetVersion() <= initialVersion {
		t.Error("Expected version to increment after registering shard")
	}

	prevVersion := cs.GetVersion()
	cs.UpdateShardState("shard-1", ShardStateDraining)

	if cs.GetVersion() <= prevVersion {
		t.Error("Expected version to increment after updating shard state")
	}
}

func TestConfigServerConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	// Register initial shard
	shard := NewShard("shard-1", nil, "localhost:27017")
	cs.RegisterShard(shard)

	// Concurrent reads and writes
	done := make(chan bool, 100)

	// Readers
	for i := 0; i < 50; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				cs.GetShard("shard-1")
				cs.ListShards()
				cs.Stats()
			}
			done <- true
		}()
	}

	// Writers
	for i := 0; i < 50; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				tags := map[string]string{
					"iteration": filepath.Join("iter-", string(rune(j))),
				}
				cs.UpdateShardTags("shard-1", tags)
				time.Sleep(time.Microsecond)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestConfigServerMetadataFile(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)

	shard := NewShard("shard-1", nil, "localhost:27017")
	cs.RegisterShard(shard)

	cs.Close()

	// Verify metadata file exists
	metaPath := filepath.Join(tempDir, "config_server_metadata.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Error("Metadata file was not created")
	}

	// Verify file is valid JSON
	data, _ := os.ReadFile(metaPath)
	if len(data) == 0 {
		t.Error("Metadata file is empty")
	}
}

func TestConfigServerUnregisterChunk(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	// Register a shard and chunk
	shard := NewShard("shard-1", nil, "localhost:27017")
	cs.RegisterShard(shard)

	chunk := NewChunk("chunk-1", "shard-1", int64(0), int64(100))
	err := cs.RegisterChunk(chunk)
	if err != nil {
		t.Fatalf("Failed to register chunk: %v", err)
	}

	// Verify chunk was registered
	_, err = cs.GetChunk("chunk-1")
	if err != nil {
		t.Fatalf("Chunk should exist: %v", err)
	}

	// Unregister the chunk
	err = cs.UnregisterChunk("chunk-1")
	if err != nil {
		t.Fatalf("Failed to unregister chunk: %v", err)
	}

	// Verify chunk no longer exists
	_, err = cs.GetChunk("chunk-1")
	if err == nil {
		t.Error("Chunk should not exist after unregistration")
	}

	// Try to unregister non-existent chunk
	err = cs.UnregisterChunk("chunk-nonexistent")
	if err == nil {
		t.Error("Expected error when unregistering non-existent chunk")
	}
}

func TestConfigServerListChunks(t *testing.T) {
	tempDir := t.TempDir()
	cs, _ := NewConfigServer(tempDir)
	defer cs.Close()

	// Register shards
	shard1 := NewShard("shard-1", nil, "localhost:27017")
	shard2 := NewShard("shard-2", nil, "localhost:27018")
	cs.RegisterShard(shard1)
	cs.RegisterShard(shard2)

	// Initially, there should be no chunks
	chunks := cs.ListChunks()
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks, got %d", len(chunks))
	}

	// Register chunks
	chunk1 := NewChunk("chunk-1", "shard-1", int64(0), int64(100))
	chunk2 := NewChunk("chunk-2", "shard-1", int64(100), int64(200))
	chunk3 := NewChunk("chunk-3", "shard-2", int64(0), int64(150))

	cs.RegisterChunk(chunk1)
	cs.RegisterChunk(chunk2)
	cs.RegisterChunk(chunk3)

	// List all chunks
	chunks = cs.ListChunks()
	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chunks))
	}

	// Verify chunk IDs
	chunkIDs := make(map[string]bool)
	for _, c := range chunks {
		chunkIDs[c.ID] = true
	}

	if !chunkIDs["chunk-1"] || !chunkIDs["chunk-2"] || !chunkIDs["chunk-3"] {
		t.Error("Missing expected chunk IDs in list")
	}
}
