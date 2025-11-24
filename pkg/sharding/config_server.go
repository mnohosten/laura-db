package sharding

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ConfigServer stores and manages sharding metadata
// In a distributed system, this would be a replicated service
type ConfigServer struct {
	dataDir        string
	shardRegistry  map[ShardID]*ShardMetadata
	chunkRegistry  map[string]*ChunkMetadata
	collectionMeta map[string]*CollectionShardingConfig
	mu             sync.RWMutex
	version        int64 // Metadata version for optimistic concurrency
}

// ShardMetadata contains persistent metadata about a shard
type ShardMetadata struct {
	ID        ShardID           `json:"id"`
	Host      string            `json:"host"`
	Tags      map[string]string `json:"tags"`
	State     ShardState        `json:"state"`
	AddedAt   time.Time         `json:"added_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// ShardState represents the state of a shard
type ShardState string

const (
	ShardStateActive      ShardState = "active"
	ShardStateDraining    ShardState = "draining"    // Being removed, chunks being migrated
	ShardStateInactive    ShardState = "inactive"    // Temporarily offline
	ShardStateUnreachable ShardState = "unreachable" // Cannot be contacted
)

// ChunkMetadata contains persistent metadata about a chunk
type ChunkMetadata struct {
	ID        string      `json:"id"`
	ShardID   ShardID     `json:"shard_id"`
	MinKey    interface{} `json:"min_key"`
	MaxKey    interface{} `json:"max_key"`
	Count     int64       `json:"count"`
	Size      int64       `json:"size"`
	Version   int64       `json:"version"` // For migration tracking
	UpdatedAt time.Time   `json:"updated_at"`
}

// CollectionShardingConfig stores sharding configuration for a collection
type CollectionShardingConfig struct {
	Database   string       `json:"database"`
	Collection string       `json:"collection"`
	ShardKey   *ShardKey    `json:"shard_key"`
	Sharded    bool         `json:"sharded"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

// ConfigServerMetadata is the root metadata structure for persistence
type ConfigServerMetadata struct {
	Version        int64                                  `json:"version"`
	Shards         map[ShardID]*ShardMetadata             `json:"shards"`
	Chunks         map[string]*ChunkMetadata              `json:"chunks"`
	CollectionMeta map[string]*CollectionShardingConfig   `json:"collection_meta"`
	UpdatedAt      time.Time                              `json:"updated_at"`
}

// NewConfigServer creates a new config server
func NewConfigServer(dataDir string) (*ConfigServer, error) {
	cs := &ConfigServer{
		dataDir:        dataDir,
		shardRegistry:  make(map[ShardID]*ShardMetadata),
		chunkRegistry:  make(map[string]*ChunkMetadata),
		collectionMeta: make(map[string]*CollectionShardingConfig),
		version:        1,
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config server data directory: %w", err)
	}

	// Load existing metadata from disk
	if err := cs.loadMetadata(); err != nil {
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}

	return cs, nil
}

// RegisterShard registers a new shard in the config server
func (cs *ConfigServer) RegisterShard(shard *Shard) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, exists := cs.shardRegistry[shard.ID]; exists {
		return fmt.Errorf("shard already registered: %s", shard.ID)
	}

	meta := &ShardMetadata{
		ID:        shard.ID,
		Host:      shard.Host,
		Tags:      make(map[string]string),
		State:     ShardStateActive,
		AddedAt:   time.Now(),
		UpdatedAt: time.Now(),
	}

	// Copy tags
	for k, v := range shard.Tags {
		meta.Tags[k] = v
	}

	cs.shardRegistry[shard.ID] = meta
	cs.version++

	return cs.persistMetadata()
}

// UnregisterShard removes a shard from the config server
func (cs *ConfigServer) UnregisterShard(shardID ShardID) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, exists := cs.shardRegistry[shardID]; !exists {
		return fmt.Errorf("shard not found: %s", shardID)
	}

	// Check if shard has any chunks
	hasChunks := false
	for _, chunk := range cs.chunkRegistry {
		if chunk.ShardID == shardID {
			hasChunks = true
			break
		}
	}

	if hasChunks {
		return fmt.Errorf("cannot unregister shard %s: still has chunks assigned", shardID)
	}

	delete(cs.shardRegistry, shardID)
	cs.version++

	return cs.persistMetadata()
}

// UpdateShardState updates the state of a shard
func (cs *ConfigServer) UpdateShardState(shardID ShardID, state ShardState) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	meta, exists := cs.shardRegistry[shardID]
	if !exists {
		return fmt.Errorf("shard not found: %s", shardID)
	}

	meta.State = state
	meta.UpdatedAt = time.Now()
	cs.version++

	return cs.persistMetadata()
}

// UpdateShardTags updates tags for a shard
func (cs *ConfigServer) UpdateShardTags(shardID ShardID, tags map[string]string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	meta, exists := cs.shardRegistry[shardID]
	if !exists {
		return fmt.Errorf("shard not found: %s", shardID)
	}

	meta.Tags = make(map[string]string)
	for k, v := range tags {
		meta.Tags[k] = v
	}
	meta.UpdatedAt = time.Now()
	cs.version++

	return cs.persistMetadata()
}

// GetShard retrieves shard metadata
func (cs *ConfigServer) GetShard(shardID ShardID) (*ShardMetadata, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	meta, exists := cs.shardRegistry[shardID]
	if !exists {
		return nil, fmt.Errorf("shard not found: %s", shardID)
	}

	// Return a copy to prevent external modification
	copy := *meta
	copy.Tags = make(map[string]string)
	for k, v := range meta.Tags {
		copy.Tags[k] = v
	}

	return &copy, nil
}

// ListShards returns all registered shards
func (cs *ConfigServer) ListShards() []*ShardMetadata {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make([]*ShardMetadata, 0, len(cs.shardRegistry))
	for _, meta := range cs.shardRegistry {
		copy := *meta
		copy.Tags = make(map[string]string)
		for k, v := range meta.Tags {
			copy.Tags[k] = v
		}
		result = append(result, &copy)
	}

	return result
}

// ListActiveShards returns all shards in active state
func (cs *ConfigServer) ListActiveShards() []*ShardMetadata {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make([]*ShardMetadata, 0)
	for _, meta := range cs.shardRegistry {
		if meta.State == ShardStateActive {
			copy := *meta
			copy.Tags = make(map[string]string)
			for k, v := range meta.Tags {
				copy.Tags[k] = v
			}
			result = append(result, &copy)
		}
	}

	return result
}

// RegisterChunk registers chunk metadata
func (cs *ConfigServer) RegisterChunk(chunk *Chunk) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, exists := cs.chunkRegistry[chunk.ID]; exists {
		return fmt.Errorf("chunk already registered: %s", chunk.ID)
	}

	// Verify shard exists
	if _, exists := cs.shardRegistry[chunk.ShardID]; !exists {
		return fmt.Errorf("shard not found for chunk: %s", chunk.ShardID)
	}

	meta := &ChunkMetadata{
		ID:        chunk.ID,
		ShardID:   chunk.ShardID,
		MinKey:    chunk.MinKey,
		MaxKey:    chunk.MaxKey,
		Count:     chunk.Count,
		Size:      chunk.Size,
		Version:   1,
		UpdatedAt: time.Now(),
	}

	cs.chunkRegistry[chunk.ID] = meta
	cs.version++

	return cs.persistMetadata()
}

// UnregisterChunk removes chunk metadata
func (cs *ConfigServer) UnregisterChunk(chunkID string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, exists := cs.chunkRegistry[chunkID]; !exists {
		return fmt.Errorf("chunk not found: %s", chunkID)
	}

	delete(cs.chunkRegistry, chunkID)
	cs.version++

	return cs.persistMetadata()
}

// UpdateChunk updates chunk metadata
func (cs *ConfigServer) UpdateChunk(chunkID string, count, size int64) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	meta, exists := cs.chunkRegistry[chunkID]
	if !exists {
		return fmt.Errorf("chunk not found: %s", chunkID)
	}

	meta.Count = count
	meta.Size = size
	meta.UpdatedAt = time.Now()
	cs.version++

	return cs.persistMetadata()
}

// MoveChunkMetadata updates chunk's shard assignment
func (cs *ConfigServer) MoveChunkMetadata(chunkID string, targetShardID ShardID) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	meta, exists := cs.chunkRegistry[chunkID]
	if !exists {
		return fmt.Errorf("chunk not found: %s", chunkID)
	}

	// Verify target shard exists
	if _, exists := cs.shardRegistry[targetShardID]; !exists {
		return fmt.Errorf("target shard not found: %s", targetShardID)
	}

	meta.ShardID = targetShardID
	meta.Version++
	meta.UpdatedAt = time.Now()
	cs.version++

	return cs.persistMetadata()
}

// GetChunk retrieves chunk metadata
func (cs *ConfigServer) GetChunk(chunkID string) (*ChunkMetadata, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	meta, exists := cs.chunkRegistry[chunkID]
	if !exists {
		return nil, fmt.Errorf("chunk not found: %s", chunkID)
	}

	copy := *meta
	return &copy, nil
}

// ListChunks returns all chunks
func (cs *ConfigServer) ListChunks() []*ChunkMetadata {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make([]*ChunkMetadata, 0, len(cs.chunkRegistry))
	for _, meta := range cs.chunkRegistry {
		copy := *meta
		result = append(result, &copy)
	}

	return result
}

// ListChunksForShard returns all chunks on a specific shard
func (cs *ConfigServer) ListChunksForShard(shardID ShardID) []*ChunkMetadata {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make([]*ChunkMetadata, 0)
	for _, meta := range cs.chunkRegistry {
		if meta.ShardID == shardID {
			copy := *meta
			result = append(result, &copy)
		}
	}

	return result
}

// SetCollectionSharding configures sharding for a collection
func (cs *ConfigServer) SetCollectionSharding(database, collection string, shardKey *ShardKey) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	key := fmt.Sprintf("%s.%s", database, collection)

	if _, exists := cs.collectionMeta[key]; exists {
		return fmt.Errorf("collection already sharded: %s", key)
	}

	config := &CollectionShardingConfig{
		Database:   database,
		Collection: collection,
		ShardKey:   shardKey,
		Sharded:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	cs.collectionMeta[key] = config
	cs.version++

	return cs.persistMetadata()
}

// GetCollectionSharding retrieves sharding configuration for a collection
func (cs *ConfigServer) GetCollectionSharding(database, collection string) (*CollectionShardingConfig, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	key := fmt.Sprintf("%s.%s", database, collection)
	config, exists := cs.collectionMeta[key]
	if !exists {
		return nil, fmt.Errorf("collection not sharded: %s", key)
	}

	// Return a copy
	copy := *config
	return &copy, nil
}

// RemoveCollectionSharding removes sharding configuration for a collection
func (cs *ConfigServer) RemoveCollectionSharding(database, collection string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	key := fmt.Sprintf("%s.%s", database, collection)
	if _, exists := cs.collectionMeta[key]; !exists {
		return fmt.Errorf("collection not sharded: %s", key)
	}

	delete(cs.collectionMeta, key)
	cs.version++

	return cs.persistMetadata()
}

// ListShardedCollections returns all sharded collections
func (cs *ConfigServer) ListShardedCollections() []*CollectionShardingConfig {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make([]*CollectionShardingConfig, 0, len(cs.collectionMeta))
	for _, config := range cs.collectionMeta {
		copy := *config
		result = append(result, &copy)
	}

	return result
}

// GetVersion returns current metadata version
func (cs *ConfigServer) GetVersion() int64 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.version
}

// Stats returns statistics about the config server
func (cs *ConfigServer) Stats() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	activeShards := 0
	drainingShards := 0
	inactiveShards := 0
	unreachableShards := 0

	for _, meta := range cs.shardRegistry {
		switch meta.State {
		case ShardStateActive:
			activeShards++
		case ShardStateDraining:
			drainingShards++
		case ShardStateInactive:
			inactiveShards++
		case ShardStateUnreachable:
			unreachableShards++
		}
	}

	return map[string]interface{}{
		"version":             cs.version,
		"total_shards":        len(cs.shardRegistry),
		"active_shards":       activeShards,
		"draining_shards":     drainingShards,
		"inactive_shards":     inactiveShards,
		"unreachable_shards":  unreachableShards,
		"total_chunks":        len(cs.chunkRegistry),
		"sharded_collections": len(cs.collectionMeta),
		"data_dir":            cs.dataDir,
	}
}

// persistMetadata saves metadata to disk
func (cs *ConfigServer) persistMetadata() error {
	metadata := &ConfigServerMetadata{
		Version:        cs.version,
		Shards:         cs.shardRegistry,
		Chunks:         cs.chunkRegistry,
		CollectionMeta: cs.collectionMeta,
		UpdatedAt:      time.Now(),
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metaPath := filepath.Join(cs.dataDir, "config_server_metadata.json")
	tempPath := metaPath + ".tmp"

	// Write to temporary file first
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, metaPath); err != nil {
		return fmt.Errorf("failed to rename metadata file: %w", err)
	}

	return nil
}

// loadMetadata loads metadata from disk
func (cs *ConfigServer) loadMetadata() error {
	metaPath := filepath.Join(cs.dataDir, "config_server_metadata.json")

	// Check if file exists
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		// No existing metadata - start fresh
		return nil
	}

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata ConfigServerMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	cs.version = metadata.Version
	cs.shardRegistry = metadata.Shards
	cs.chunkRegistry = metadata.Chunks
	cs.collectionMeta = metadata.CollectionMeta

	// Ensure maps are initialized
	if cs.shardRegistry == nil {
		cs.shardRegistry = make(map[ShardID]*ShardMetadata)
	}
	if cs.chunkRegistry == nil {
		cs.chunkRegistry = make(map[string]*ChunkMetadata)
	}
	if cs.collectionMeta == nil {
		cs.collectionMeta = make(map[string]*CollectionShardingConfig)
	}

	return nil
}

// Close cleanly shuts down the config server
func (cs *ConfigServer) Close() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Final metadata persist
	return cs.persistMetadata()
}
