package index

import (
	"container/list"
	"sync"
)

// NodeCache implements an LRU cache for B+ tree nodes
type NodeCache struct {
	capacity int
	cache    map[uint32]*list.Element // pageID -> list element
	lru      *list.List                // LRU list
	mu       sync.RWMutex
	hits     uint64 // Cache hit counter
	misses   uint64 // Cache miss counter
}

// cacheEntry represents an entry in the cache
type cacheEntry struct {
	pageID uint32
	node   *BTreeNode
}

// NewNodeCache creates a new node cache with the given capacity
func NewNodeCache(capacity int) *NodeCache {
	if capacity <= 0 {
		capacity = 500 // Default capacity
	}

	return &NodeCache{
		capacity: capacity,
		cache:    make(map[uint32]*list.Element),
		lru:      list.New(),
		hits:     0,
		misses:   0,
	}
}

// Get retrieves a node from the cache
func (nc *NodeCache) Get(pageID uint32) (*BTreeNode, bool) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	if elem, found := nc.cache[pageID]; found {
		// Move to front (most recently used)
		nc.lru.MoveToFront(elem)
		nc.hits++
		return elem.Value.(*cacheEntry).node, true
	}

	nc.misses++
	return nil, false
}

// Put adds a node to the cache
func (nc *NodeCache) Put(pageID uint32, node *BTreeNode) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	// Check if already in cache
	if elem, found := nc.cache[pageID]; found {
		// Update and move to front
		elem.Value.(*cacheEntry).node = node
		nc.lru.MoveToFront(elem)
		return
	}

	// Add new entry
	entry := &cacheEntry{
		pageID: pageID,
		node:   node,
	}
	elem := nc.lru.PushFront(entry)
	nc.cache[pageID] = elem

	// Evict if over capacity
	if nc.lru.Len() > nc.capacity {
		nc.evict()
	}
}

// evict removes the least recently used entry
func (nc *NodeCache) evict() {
	elem := nc.lru.Back()
	if elem != nil {
		entry := elem.Value.(*cacheEntry)
		nc.lru.Remove(elem)
		delete(nc.cache, entry.pageID)
	}
}

// Remove removes a node from the cache
func (nc *NodeCache) Remove(pageID uint32) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	if elem, found := nc.cache[pageID]; found {
		nc.lru.Remove(elem)
		delete(nc.cache, pageID)
	}
}

// Clear removes all entries from the cache
func (nc *NodeCache) Clear() {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.cache = make(map[uint32]*list.Element)
	nc.lru.Init()
	nc.hits = 0
	nc.misses = 0
}

// Size returns the current number of entries in the cache
func (nc *NodeCache) Size() int {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	return nc.lru.Len()
}

// Stats returns cache statistics
func (nc *NodeCache) Stats() (hits, misses uint64, hitRate float64) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	hits = nc.hits
	misses = nc.misses
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}
	return hits, misses, hitRate
}

// GetDirtyNodes returns all dirty nodes in the cache
func (nc *NodeCache) GetDirtyNodes() []*BTreeNode {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	dirtyNodes := make([]*BTreeNode, 0)
	for elem := nc.lru.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*cacheEntry)
		if entry.node.isDirty {
			dirtyNodes = append(dirtyNodes, entry.node)
		}
	}
	return dirtyNodes
}

// Flush marks all nodes as clean (after they've been written to disk)
func (nc *NodeCache) Flush() {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	for elem := nc.lru.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*cacheEntry)
		entry.node.isDirty = false
	}
}
