package concurrent

import (
	"container/list"
	"crypto/sha256"
	"encoding/binary"
	"sync"
	"time"
)

// ShardedLRUCache is a sharded LRU cache that reduces lock contention
// by partitioning the cache into multiple shards, each with its own lock
type ShardedLRUCache struct {
	shards    []*shard
	shardMask uint32
	capacity  int
	ttl       time.Duration
}

// shard represents a single partition of the cache
type shard struct {
	mu        sync.RWMutex
	items     map[string]*cacheEntry
	lruList   *list.List
	capacity  int
	hits      uint64
	misses    uint64
	evictions uint64
}

// cacheEntry represents a cached item
type cacheEntry struct {
	key       string
	value     interface{}
	expiresAt time.Time
	element   *list.Element
}

// NewShardedLRUCache creates a new sharded LRU cache
// shardCount must be a power of 2 for efficient modulo operation
func NewShardedLRUCache(capacity int, ttl time.Duration, shardCount uint32) *ShardedLRUCache {
	// Ensure shardCount is a power of 2
	if shardCount == 0 || (shardCount&(shardCount-1)) != 0 {
		// Round up to next power of 2
		shardCount = nextPowerOfTwo(shardCount)
	}

	shards := make([]*shard, shardCount)
	shardCapacity := capacity / int(shardCount)
	if shardCapacity == 0 {
		shardCapacity = 1
	}

	for i := range shards {
		shards[i] = &shard{
			items:    make(map[string]*cacheEntry),
			lruList:  list.New(),
			capacity: shardCapacity,
		}
	}

	return &ShardedLRUCache{
		shards:    shards,
		shardMask: shardCount - 1,
		capacity:  capacity,
		ttl:       ttl,
	}
}

// getShard returns the shard for a given key
func (c *ShardedLRUCache) getShard(key string) *shard {
	// Hash the key and use bitwise AND for fast modulo
	hash := fnv32(key)
	return c.shards[hash&c.shardMask]
}

// Get retrieves a value from the cache
func (c *ShardedLRUCache) Get(key string) (interface{}, bool) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	entry, exists := shard.items[key]
	if !exists {
		shard.misses++
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		shard.removeElement(entry.element)
		delete(shard.items, key)
		shard.misses++
		return nil, false
	}

	// Move to front (most recently used)
	shard.lruList.MoveToFront(entry.element)
	shard.hits++
	return entry.value, true
}

// Put adds a value to the cache
func (c *ShardedLRUCache) Put(key string, value interface{}) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Check if key already exists
	if entry, exists := shard.items[key]; exists {
		entry.value = value
		entry.expiresAt = time.Now().Add(c.ttl)
		shard.lruList.MoveToFront(entry.element)
		return
	}

	// Create new entry
	entry := &cacheEntry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}

	// Add to front of list
	entry.element = shard.lruList.PushFront(entry)
	shard.items[key] = entry

	// Evict if over capacity
	if shard.lruList.Len() > shard.capacity {
		shard.evictOldest()
	}
}

// Clear removes all entries from the cache
func (c *ShardedLRUCache) Clear() {
	for _, shard := range c.shards {
		shard.mu.Lock()
		shard.items = make(map[string]*cacheEntry)
		shard.lruList = list.New()
		shard.mu.Unlock()
	}
}

// Size returns the current number of items in the cache
func (c *ShardedLRUCache) Size() int {
	total := 0
	for _, shard := range c.shards {
		shard.mu.RLock()
		total += len(shard.items)
		shard.mu.RUnlock()
	}
	return total
}

// Stats returns cache statistics
func (c *ShardedLRUCache) Stats() map[string]interface{} {
	var totalHits, totalMisses, totalEvictions uint64
	var totalSize int

	for _, shard := range c.shards {
		shard.mu.RLock()
		totalHits += shard.hits
		totalMisses += shard.misses
		totalEvictions += shard.evictions
		totalSize += len(shard.items)
		shard.mu.RUnlock()
	}

	total := totalHits + totalMisses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(totalHits) / float64(total) * 100
	}

	return map[string]interface{}{
		"capacity":    c.capacity,
		"shard_count": len(c.shards),
		"size":        totalSize,
		"hits":        totalHits,
		"misses":      totalMisses,
		"evictions":   totalEvictions,
		"hit_rate":    hitRate,
		"ttl_seconds": c.ttl.Seconds(),
	}
}

// CleanupExpired removes all expired entries
func (c *ShardedLRUCache) CleanupExpired() int {
	total := 0
	for _, shard := range c.shards {
		shard.mu.Lock()
		total += shard.cleanupExpired()
		shard.mu.Unlock()
	}
	return total
}

// shard methods

func (s *shard) evictOldest() {
	oldest := s.lruList.Back()
	if oldest != nil {
		entry := oldest.Value.(*cacheEntry)
		s.removeElement(oldest)
		delete(s.items, entry.key)
		s.evictions++
	}
}

func (s *shard) removeElement(e *list.Element) {
	s.lruList.Remove(e)
}

func (s *shard) cleanupExpired() int {
	now := time.Now()
	removed := 0

	for key, entry := range s.items {
		if now.After(entry.expiresAt) {
			s.removeElement(entry.element)
			delete(s.items, key)
			removed++
		}
	}

	return removed
}

// Utility functions

// fnv32 is a fast non-cryptographic hash function
func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	return hash
}

// fnv32a is an alternate FNV-1a hash (used for comparison)
func fnv32a(key string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		hash = (hash ^ uint32(key[i])) * 16777619
	}
	return hash
}

// hashKey uses SHA256 for better distribution (slower but more uniform)
func hashKey(key string) uint32 {
	h := sha256.Sum256([]byte(key))
	return binary.BigEndian.Uint32(h[:4])
}

// nextPowerOfTwo rounds up to the next power of 2
func nextPowerOfTwo(n uint32) uint32 {
	if n == 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}
