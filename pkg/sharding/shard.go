package sharding

import (
	"fmt"
	"sync"

	"github.com/mnohosten/laura-db/pkg/database"
)

// ShardID is a unique identifier for a shard
type ShardID string

// Shard represents a single shard in the sharded cluster
type Shard struct {
	ID       ShardID
	Database *database.Database
	Host     string // Host address (for distributed shards)
	Tags     map[string]string // Shard tags for targeting
	mu       sync.RWMutex
}

// NewShard creates a new shard
func NewShard(id ShardID, db *database.Database, host string) *Shard {
	return &Shard{
		ID:       id,
		Database: db,
		Host:     host,
		Tags:     make(map[string]string),
	}
}

// SetTag sets a tag on the shard
func (s *Shard) SetTag(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Tags[key] = value
}

// GetTag gets a tag value
func (s *Shard) GetTag(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.Tags[key]
	return val, ok
}

// MatchesTags checks if shard matches all given tags
func (s *Shard) MatchesTags(tags map[string]string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for k, v := range tags {
		if s.Tags[k] != v {
			return false
		}
	}
	return true
}

// Stats returns statistics about the shard
func (s *Shard) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"shard_id": s.ID,
		"host":     s.Host,
		"tags":     s.Tags,
	}
}

// Chunk represents a contiguous range of shard key values
// Used for range-based sharding
type Chunk struct {
	ID       string
	ShardID  ShardID
	MinKey   interface{} // Minimum shard key value (inclusive)
	MaxKey   interface{} // Maximum shard key value (exclusive)
	Count    int64       // Number of documents in chunk
	Size     int64       // Size in bytes
	mu       sync.RWMutex
}

// NewChunk creates a new chunk
func NewChunk(id string, shardID ShardID, minKey, maxKey interface{}) *Chunk {
	return &Chunk{
		ID:      id,
		ShardID: shardID,
		MinKey:  minKey,
		MaxKey:  maxKey,
		Count:   0,
		Size:    0,
	}
}

// Contains checks if a shard key value falls within this chunk
func (c *Chunk) Contains(shardKey *ShardKey, value interface{}) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if value >= MinKey
	if c.MinKey != nil {
		cmp := shardKey.CompareValues(value, c.MinKey)
		if cmp < 0 {
			return false
		}
	}

	// Check if value < MaxKey
	if c.MaxKey != nil {
		cmp := shardKey.CompareValues(value, c.MaxKey)
		if cmp >= 0 {
			return false
		}
	}

	return true
}

// UpdateStats updates chunk statistics
func (c *Chunk) UpdateStats(count int64, size int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Count = count
	c.Size = size
}

// IncrementCount increments the document count
func (c *Chunk) IncrementCount(delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Count += delta
}

// Stats returns chunk statistics
func (c *Chunk) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"chunk_id": c.ID,
		"shard_id": c.ShardID,
		"min_key":  c.MinKey,
		"max_key":  c.MaxKey,
		"count":    c.Count,
		"size":     c.Size,
	}
}

// ChunkManager manages chunks for range-based sharding
type ChunkManager struct {
	chunks      []*Chunk
	shardKey    *ShardKey
	mu          sync.RWMutex
	nextChunkID int
}

// NewChunkManager creates a new chunk manager
func NewChunkManager(shardKey *ShardKey) *ChunkManager {
	return &ChunkManager{
		chunks:      make([]*Chunk, 0),
		shardKey:    shardKey,
		nextChunkID: 1,
	}
}

// AddChunk adds a new chunk
func (cm *ChunkManager) AddChunk(chunk *Chunk) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validate chunk doesn't overlap with existing chunks
	for _, existing := range cm.chunks {
		if existing.ShardID == chunk.ShardID && cm.chunksOverlap(existing, chunk) {
			return fmt.Errorf("chunk overlaps with existing chunk %s", existing.ID)
		}
	}

	cm.chunks = append(cm.chunks, chunk)
	return nil
}

// CreateChunk creates a new chunk with auto-generated ID
func (cm *ChunkManager) CreateChunk(shardID ShardID, minKey, maxKey interface{}) (*Chunk, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	chunkID := fmt.Sprintf("chunk-%d", cm.nextChunkID)
	cm.nextChunkID++

	chunk := NewChunk(chunkID, shardID, minKey, maxKey)

	// Validate chunk doesn't overlap
	for _, existing := range cm.chunks {
		if existing.ShardID == shardID && cm.chunksOverlap(existing, chunk) {
			return nil, fmt.Errorf("chunk overlaps with existing chunk %s", existing.ID)
		}
	}

	cm.chunks = append(cm.chunks, chunk)
	return chunk, nil
}

// FindChunk finds the chunk containing the given shard key value
func (cm *ChunkManager) FindChunk(value interface{}) *Chunk {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, chunk := range cm.chunks {
		if chunk.Contains(cm.shardKey, value) {
			return chunk
		}
	}

	return nil
}

// GetChunksForShard returns all chunks for a given shard
func (cm *ChunkManager) GetChunksForShard(shardID ShardID) []*Chunk {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]*Chunk, 0)
	for _, chunk := range cm.chunks {
		if chunk.ShardID == shardID {
			result = append(result, chunk)
		}
	}

	return result
}

// GetAllChunks returns all chunks
func (cm *ChunkManager) GetAllChunks() []*Chunk {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]*Chunk, len(cm.chunks))
	copy(result, cm.chunks)
	return result
}

// SplitChunk splits a chunk into two chunks at the given split point
func (cm *ChunkManager) SplitChunk(chunkID string, splitKey interface{}) (*Chunk, *Chunk, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Find the chunk
	var chunk *Chunk
	var chunkIndex int
	for i, c := range cm.chunks {
		if c.ID == chunkID {
			chunk = c
			chunkIndex = i
			break
		}
	}

	if chunk == nil {
		return nil, nil, fmt.Errorf("chunk not found: %s", chunkID)
	}

	// Validate split key is within chunk range
	if !chunk.Contains(cm.shardKey, splitKey) {
		return nil, nil, fmt.Errorf("split key not within chunk range")
	}

	// Create two new chunks
	leftChunkID := fmt.Sprintf("chunk-%d", cm.nextChunkID)
	cm.nextChunkID++
	rightChunkID := fmt.Sprintf("chunk-%d", cm.nextChunkID)
	cm.nextChunkID++

	leftChunk := NewChunk(leftChunkID, chunk.ShardID, chunk.MinKey, splitKey)
	rightChunk := NewChunk(rightChunkID, chunk.ShardID, splitKey, chunk.MaxKey)

	// Replace old chunk with new chunks
	cm.chunks[chunkIndex] = leftChunk
	cm.chunks = append(cm.chunks, rightChunk)

	return leftChunk, rightChunk, nil
}

// MoveChunk moves a chunk to a different shard
func (cm *ChunkManager) MoveChunk(chunkID string, targetShardID ShardID) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, chunk := range cm.chunks {
		if chunk.ID == chunkID {
			chunk.mu.Lock()
			chunk.ShardID = targetShardID
			chunk.mu.Unlock()
			return nil
		}
	}

	return fmt.Errorf("chunk not found: %s", chunkID)
}

// chunksOverlap checks if two chunks overlap
func (cm *ChunkManager) chunksOverlap(a, b *Chunk) bool {
	// If either chunk has unbounded ranges, they might overlap
	// Check if b.min < a.max and a.min < b.max

	if a.MaxKey != nil && b.MinKey != nil {
		if cm.shardKey.CompareValues(b.MinKey, a.MaxKey) >= 0 {
			return false
		}
	}

	if a.MinKey != nil && b.MaxKey != nil {
		if cm.shardKey.CompareValues(a.MinKey, b.MaxKey) >= 0 {
			return false
		}
	}

	return true
}

// Stats returns chunk manager statistics
func (cm *ChunkManager) Stats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	chunkStats := make([]map[string]interface{}, len(cm.chunks))
	for i, chunk := range cm.chunks {
		chunkStats[i] = chunk.Stats()
	}

	return map[string]interface{}{
		"total_chunks": len(cm.chunks),
		"shard_key":    cm.shardKey.String(),
		"chunks":       chunkStats,
	}
}
