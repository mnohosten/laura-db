package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CacheEntry represents a cached query result
type CacheEntry struct {
	Key       string
	Value     interface{}
	ExpiresAt time.Time
	element   *list.Element
}

// LRUCache is a thread-safe LRU cache with TTL support
type LRUCache struct {
	mu       sync.RWMutex
	capacity int
	ttl      time.Duration
	items    map[string]*CacheEntry
	lruList  *list.List
	hits     uint64
	misses   uint64
	evictions uint64
}

// NewLRUCache creates a new LRU cache
func NewLRUCache(capacity int, ttl time.Duration) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[string]*CacheEntry),
		lruList:  list.New(),
		hits:     0,
		misses:   0,
		evictions: 0,
	}
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.items[key]
	if !exists {
		c.misses++
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		// Remove expired entry
		c.removeElement(entry.element)
		delete(c.items, key)
		c.misses++
		return nil, false
	}

	// Move to front (most recently used)
	c.lruList.MoveToFront(entry.element)
	c.hits++
	return entry.Value, true
}

// Put adds a value to the cache
func (c *LRUCache) Put(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key already exists
	if entry, exists := c.items[key]; exists {
		// Update existing entry
		entry.Value = value
		entry.ExpiresAt = time.Now().Add(c.ttl)
		c.lruList.MoveToFront(entry.element)
		return
	}

	// Create new entry
	entry := &CacheEntry{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}

	// Add to front of list
	entry.element = c.lruList.PushFront(entry)
	c.items[key] = entry

	// Evict if over capacity
	if c.lruList.Len() > c.capacity {
		c.evictOldest()
	}
}

// evictOldest removes the least recently used item
func (c *LRUCache) evictOldest() {
	oldest := c.lruList.Back()
	if oldest != nil {
		entry := oldest.Value.(*CacheEntry)
		c.removeElement(oldest)
		delete(c.items, entry.Key)
		c.evictions++
	}
}

// removeElement removes an element from the LRU list
func (c *LRUCache) removeElement(e *list.Element) {
	c.lruList.Remove(e)
}

// Clear removes all entries from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheEntry)
	c.lruList = list.New()
}

// Size returns the current number of items in the cache
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Stats returns cache statistics
func (c *LRUCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(c.hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"capacity":   c.capacity,
		"size":       len(c.items),
		"hits":       c.hits,
		"misses":     c.misses,
		"evictions":  c.evictions,
		"hit_rate":   fmt.Sprintf("%.2f%%", hitRate),
		"ttl_seconds": c.ttl.Seconds(),
	}
}

// GenerateKey creates a deterministic cache key from query parameters
func GenerateKey(filter map[string]interface{}, sort []interface{}, skip, limit int, projection map[string]bool) string {
	// Create a struct to hash
	keyData := struct {
		Filter     map[string]interface{}
		Sort       []interface{}
		Skip       int
		Limit      int
		Projection map[string]bool
	}{
		Filter:     filter,
		Sort:       sort,
		Skip:       skip,
		Limit:      limit,
		Projection: projection,
	}

	// Convert to JSON for deterministic serialization
	jsonBytes, err := json.Marshal(keyData)
	if err != nil {
		// Fallback to string representation
		return fmt.Sprintf("%v_%v_%d_%d_%v", filter, sort, skip, limit, projection)
	}

	// Hash the JSON
	hash := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%x", hash)
}

// CleanupExpired removes all expired entries (should be called periodically)
func (c *LRUCache) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0

	// Iterate through all items and remove expired ones
	for key, entry := range c.items {
		if now.After(entry.ExpiresAt) {
			c.removeElement(entry.element)
			delete(c.items, key)
			removed++
		}
	}

	return removed
}
