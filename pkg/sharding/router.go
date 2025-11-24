package sharding

import (
	"fmt"
	"sync"
)

// ShardRouter routes operations to the appropriate shard
type ShardRouter struct {
	shards        map[ShardID]*Shard
	shardKey      *ShardKey
	chunkManager  *ChunkManager // For range-based sharding
	numShards     int            // For hash-based sharding
	shardList     []*Shard       // Ordered list for hash-based routing
	mu            sync.RWMutex
}

// NewShardRouter creates a new shard router
func NewShardRouter(shardKey *ShardKey) (*ShardRouter, error) {
	if err := shardKey.Validate(); err != nil {
		return nil, fmt.Errorf("invalid shard key: %w", err)
	}

	sr := &ShardRouter{
		shards:   make(map[ShardID]*Shard),
		shardKey: shardKey,
	}

	// Initialize chunk manager for range-based sharding
	if shardKey.Type == ShardKeyTypeRange {
		sr.chunkManager = NewChunkManager(shardKey)
	}

	return sr, nil
}

// AddShard adds a new shard to the router
func (sr *ShardRouter) AddShard(shard *Shard) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if _, exists := sr.shards[shard.ID]; exists {
		return fmt.Errorf("shard already exists: %s", shard.ID)
	}

	sr.shards[shard.ID] = shard

	// Update shard list for hash-based sharding
	if sr.shardKey.Type == ShardKeyTypeHash {
		sr.shardList = append(sr.shardList, shard)
		sr.numShards = len(sr.shardList)
	}

	return nil
}

// RemoveShard removes a shard from the router
func (sr *ShardRouter) RemoveShard(shardID ShardID) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if _, exists := sr.shards[shardID]; !exists {
		return fmt.Errorf("shard not found: %s", shardID)
	}

	delete(sr.shards, shardID)

	// Update shard list for hash-based sharding
	if sr.shardKey.Type == ShardKeyTypeHash {
		newList := make([]*Shard, 0, len(sr.shardList)-1)
		for _, s := range sr.shardList {
			if s.ID != shardID {
				newList = append(newList, s)
			}
		}
		sr.shardList = newList
		sr.numShards = len(sr.shardList)
	}

	return nil
}

// GetShard gets a shard by ID
func (sr *ShardRouter) GetShard(shardID ShardID) (*Shard, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	shard, ok := sr.shards[shardID]
	if !ok {
		return nil, fmt.Errorf("shard not found: %s", shardID)
	}

	return shard, nil
}

// GetAllShards returns all shards
func (sr *ShardRouter) GetAllShards() []*Shard {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	result := make([]*Shard, 0, len(sr.shards))
	for _, shard := range sr.shards {
		result = append(result, shard)
	}

	return result
}

// Route routes a document to the appropriate shard based on shard key
func (sr *ShardRouter) Route(doc map[string]interface{}) (*Shard, error) {
	// Extract shard key value from document
	shardKeyValue, err := sr.shardKey.ExtractShardKeyValue(doc)
	if err != nil {
		return nil, err
	}

	// Route based on sharding strategy
	switch sr.shardKey.Type {
	case ShardKeyTypeRange:
		return sr.routeRange(shardKeyValue)
	case ShardKeyTypeHash:
		return sr.routeHash(shardKeyValue)
	default:
		return nil, fmt.Errorf("unknown shard key type: %v", sr.shardKey.Type)
	}
}

// RouteByShardKeyValue routes based on an extracted shard key value
func (sr *ShardRouter) RouteByShardKeyValue(shardKeyValue interface{}) (*Shard, error) {
	switch sr.shardKey.Type {
	case ShardKeyTypeRange:
		return sr.routeRange(shardKeyValue)
	case ShardKeyTypeHash:
		return sr.routeHash(shardKeyValue)
	default:
		return nil, fmt.Errorf("unknown shard key type: %v", sr.shardKey.Type)
	}
}

// routeRange routes using range-based sharding
func (sr *ShardRouter) routeRange(shardKeyValue interface{}) (*Shard, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	if sr.chunkManager == nil {
		return nil, fmt.Errorf("chunk manager not initialized")
	}

	// Find the chunk containing this value
	chunk := sr.chunkManager.FindChunk(shardKeyValue)
	if chunk == nil {
		return nil, fmt.Errorf("no chunk found for shard key value: %v", shardKeyValue)
	}

	// Get the shard for this chunk
	shard, ok := sr.shards[chunk.ShardID]
	if !ok {
		return nil, fmt.Errorf("shard not found for chunk: %s", chunk.ShardID)
	}

	return shard, nil
}

// routeHash routes using hash-based sharding
func (sr *ShardRouter) routeHash(shardKeyValue interface{}) (*Shard, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	if sr.numShards == 0 {
		return nil, fmt.Errorf("no shards available")
	}

	// Compute hash of shard key value
	hashValue := sr.shardKey.HashValue(shardKeyValue)

	// Map hash to shard using modulo
	shardIndex := int(hashValue % uint64(sr.numShards))
	return sr.shardList[shardIndex], nil
}

// InitializeRangeSharding initializes range-based sharding with initial chunks
func (sr *ShardRouter) InitializeRangeSharding(initialChunks []*Chunk) error {
	if sr.shardKey.Type != ShardKeyTypeRange {
		return fmt.Errorf("not using range-based sharding")
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	if sr.chunkManager == nil {
		return fmt.Errorf("chunk manager not initialized")
	}

	// Add all initial chunks
	for _, chunk := range initialChunks {
		// Validate shard exists
		if _, ok := sr.shards[chunk.ShardID]; !ok {
			return fmt.Errorf("shard not found for chunk: %s", chunk.ShardID)
		}

		if err := sr.chunkManager.AddChunk(chunk); err != nil {
			return err
		}
	}

	return nil
}

// CreateChunk creates a new chunk for range-based sharding
func (sr *ShardRouter) CreateChunk(shardID ShardID, minKey, maxKey interface{}) (*Chunk, error) {
	if sr.shardKey.Type != ShardKeyTypeRange {
		return nil, fmt.Errorf("not using range-based sharding")
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	if sr.chunkManager == nil {
		return nil, fmt.Errorf("chunk manager not initialized")
	}

	// Validate shard exists
	if _, ok := sr.shards[shardID]; !ok {
		return nil, fmt.Errorf("shard not found: %s", shardID)
	}

	return sr.chunkManager.CreateChunk(shardID, minKey, maxKey)
}

// SplitChunk splits a chunk at the given split key
func (sr *ShardRouter) SplitChunk(chunkID string, splitKey interface{}) (*Chunk, *Chunk, error) {
	if sr.shardKey.Type != ShardKeyTypeRange {
		return nil, nil, fmt.Errorf("not using range-based sharding")
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	if sr.chunkManager == nil {
		return nil, nil, fmt.Errorf("chunk manager not initialized")
	}

	return sr.chunkManager.SplitChunk(chunkID, splitKey)
}

// MoveChunk moves a chunk to a different shard
func (sr *ShardRouter) MoveChunk(chunkID string, targetShardID ShardID) error {
	if sr.shardKey.Type != ShardKeyTypeRange {
		return fmt.Errorf("not using range-based sharding")
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	if sr.chunkManager == nil {
		return fmt.Errorf("chunk manager not initialized")
	}

	// Validate target shard exists
	if _, ok := sr.shards[targetShardID]; !ok {
		return fmt.Errorf("shard not found: %s", targetShardID)
	}

	return sr.chunkManager.MoveChunk(chunkID, targetShardID)
}

// GetChunks returns all chunks (for range-based sharding)
func (sr *ShardRouter) GetChunks() []*Chunk {
	if sr.shardKey.Type != ShardKeyTypeRange || sr.chunkManager == nil {
		return nil
	}

	return sr.chunkManager.GetAllChunks()
}

// GetChunksForShard returns all chunks for a specific shard
func (sr *ShardRouter) GetChunksForShard(shardID ShardID) []*Chunk {
	if sr.shardKey.Type != ShardKeyTypeRange || sr.chunkManager == nil {
		return nil
	}

	return sr.chunkManager.GetChunksForShard(shardID)
}

// RouteQuery routes a query to the appropriate shards
// Returns all shards that might contain matching documents
func (sr *ShardRouter) RouteQuery(filter map[string]interface{}) ([]*Shard, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	// Try to extract shard key from filter
	shardKeyValue, err := sr.shardKey.ExtractShardKeyValue(filter)
	if err != nil {
		// Filter doesn't contain full shard key - must query all shards
		result := make([]*Shard, 0, len(sr.shards))
		for _, shard := range sr.shards {
			result = append(result, shard)
		}
		return result, nil
	}

	// Filter contains shard key - route to specific shard
	shard, err := sr.RouteByShardKeyValue(shardKeyValue)
	if err != nil {
		return nil, err
	}

	return []*Shard{shard}, nil
}

// Stats returns statistics about the shard router
func (sr *ShardRouter) Stats() map[string]interface{} {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	shardStats := make([]map[string]interface{}, 0, len(sr.shards))
	for _, shard := range sr.shards {
		shardStats = append(shardStats, shard.Stats())
	}

	stats := map[string]interface{}{
		"shard_key":    sr.shardKey.String(),
		"shard_type":   sr.shardKey.Type.String(),
		"total_shards": len(sr.shards),
		"shards":       shardStats,
	}

	// Add chunk manager stats for range-based sharding
	if sr.shardKey.Type == ShardKeyTypeRange && sr.chunkManager != nil {
		stats["chunk_manager"] = sr.chunkManager.Stats()
	}

	return stats
}
