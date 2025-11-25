package index

import (
	"testing"
)

func TestNodeCacheBasicOperations(t *testing.T) {
	cache := NewNodeCache(3)

	// Create test nodes
	node1 := &BTreeNode{pageID: 1, isLeaf: true}
	node2 := &BTreeNode{pageID: 2, isLeaf: true}
	node3 := &BTreeNode{pageID: 3, isLeaf: true}

	// Test Put and Get
	cache.Put(1, node1)
	cache.Put(2, node2)
	cache.Put(3, node3)

	// Verify all nodes are in cache
	if cache.Size() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.Size())
	}

	// Test Get
	node, found := cache.Get(1)
	if !found {
		t.Error("Expected to find node 1")
	}
	if node.pageID != 1 {
		t.Errorf("Expected pageID 1, got %d", node.pageID)
	}

	// Test cache hit/miss stats
	hits, misses, hitRate := cache.Stats()
	if hits != 1 || misses != 0 {
		t.Errorf("Expected 1 hit and 0 misses, got %d hits and %d misses", hits, misses)
	}
	if hitRate != 1.0 {
		t.Errorf("Expected hit rate 1.0, got %f", hitRate)
	}

	// Test miss
	_, found = cache.Get(999)
	if found {
		t.Error("Expected not to find node 999")
	}

	hits, misses, _ = cache.Stats()
	if hits != 1 || misses != 1 {
		t.Errorf("Expected 1 hit and 1 miss, got %d hits and %d misses", hits, misses)
	}
}

func TestNodeCacheEviction(t *testing.T) {
	cache := NewNodeCache(2) // Small capacity

	node1 := &BTreeNode{pageID: 1, isLeaf: true}
	node2 := &BTreeNode{pageID: 2, isLeaf: true}
	node3 := &BTreeNode{pageID: 3, isLeaf: true}

	// Add 3 nodes to cache with capacity 2
	cache.Put(1, node1)
	cache.Put(2, node2)
	cache.Put(3, node3) // Should evict node1 (LRU)

	// Verify cache size
	if cache.Size() != 2 {
		t.Errorf("Expected cache size 2, got %d", cache.Size())
	}

	// Node 1 should have been evicted
	_, found := cache.Get(1)
	if found {
		t.Error("Expected node 1 to be evicted")
	}

	// Nodes 2 and 3 should still be in cache
	_, found = cache.Get(2)
	if !found {
		t.Error("Expected node 2 to be in cache")
	}

	_, found = cache.Get(3)
	if !found {
		t.Error("Expected node 3 to be in cache")
	}
}

func TestNodeCacheLRUOrdering(t *testing.T) {
	cache := NewNodeCache(3)

	node1 := &BTreeNode{pageID: 1, isLeaf: true}
	node2 := &BTreeNode{pageID: 2, isLeaf: true}
	node3 := &BTreeNode{pageID: 3, isLeaf: true}
	node4 := &BTreeNode{pageID: 4, isLeaf: true}

	// Add 3 nodes
	cache.Put(1, node1)
	cache.Put(2, node2)
	cache.Put(3, node3)

	// Access node 1 to make it most recently used
	cache.Get(1)

	// Add node 4, should evict node 2 (least recently used)
	cache.Put(4, node4)

	// Node 2 should be evicted
	_, found := cache.Get(2)
	if found {
		t.Error("Expected node 2 to be evicted")
	}

	// Nodes 1, 3, and 4 should be in cache
	_, found = cache.Get(1)
	if !found {
		t.Error("Expected node 1 to be in cache")
	}

	_, found = cache.Get(3)
	if !found {
		t.Error("Expected node 3 to be in cache")
	}

	_, found = cache.Get(4)
	if !found {
		t.Error("Expected node 4 to be in cache")
	}
}

func TestNodeCacheUpdate(t *testing.T) {
	cache := NewNodeCache(3)

	node1 := &BTreeNode{pageID: 1, isLeaf: true, keys: []interface{}{int64(10)}}
	cache.Put(1, node1)

	// Update node 1
	node1Updated := &BTreeNode{pageID: 1, isLeaf: true, keys: []interface{}{int64(20)}}
	cache.Put(1, node1Updated)

	// Retrieve and verify update
	node, found := cache.Get(1)
	if !found {
		t.Error("Expected to find node 1")
	}

	if len(node.keys) != 1 || node.keys[0] != int64(20) {
		t.Error("Node was not updated correctly")
	}

	// Cache size should still be 1
	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}
}

func TestNodeCacheRemove(t *testing.T) {
	cache := NewNodeCache(3)

	node1 := &BTreeNode{pageID: 1, isLeaf: true}
	node2 := &BTreeNode{pageID: 2, isLeaf: true}

	cache.Put(1, node1)
	cache.Put(2, node2)

	if cache.Size() != 2 {
		t.Errorf("Expected cache size 2, got %d", cache.Size())
	}

	// Remove node 1
	cache.Remove(1)

	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1 after remove, got %d", cache.Size())
	}

	// Node 1 should not be found
	_, found := cache.Get(1)
	if found {
		t.Error("Expected node 1 to be removed")
	}

	// Node 2 should still be in cache
	_, found = cache.Get(2)
	if !found {
		t.Error("Expected node 2 to still be in cache")
	}
}

func TestNodeCacheClear(t *testing.T) {
	cache := NewNodeCache(3)

	node1 := &BTreeNode{pageID: 1, isLeaf: true}
	node2 := &BTreeNode{pageID: 2, isLeaf: true}
	node3 := &BTreeNode{pageID: 3, isLeaf: true}

	cache.Put(1, node1)
	cache.Put(2, node2)
	cache.Put(3, node3)

	// Some accesses to generate stats
	cache.Get(1)
	cache.Get(2)
	cache.Get(999) // Miss

	if cache.Size() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.Size())
	}

	// Clear cache
	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cache.Size())
	}

	// Stats should be reset
	hits, misses, _ := cache.Stats()
	if hits != 0 || misses != 0 {
		t.Errorf("Expected stats to be reset, got %d hits and %d misses", hits, misses)
	}

	// No nodes should be found
	_, found := cache.Get(1)
	if found {
		t.Error("Expected no nodes after clear")
	}
}

func TestNodeCacheGetDirtyNodes(t *testing.T) {
	cache := NewNodeCache(5)

	// Create nodes with different dirty states
	node1 := &BTreeNode{pageID: 1, isLeaf: true, isDirty: true}
	node2 := &BTreeNode{pageID: 2, isLeaf: true, isDirty: false}
	node3 := &BTreeNode{pageID: 3, isLeaf: true, isDirty: true}
	node4 := &BTreeNode{pageID: 4, isLeaf: true, isDirty: false}

	cache.Put(1, node1)
	cache.Put(2, node2)
	cache.Put(3, node3)
	cache.Put(4, node4)

	// Get dirty nodes
	dirtyNodes := cache.GetDirtyNodes()

	// Should have 2 dirty nodes
	if len(dirtyNodes) != 2 {
		t.Errorf("Expected 2 dirty nodes, got %d", len(dirtyNodes))
	}

	// Verify they are the correct nodes
	dirtyPageIDs := make(map[uint32]bool)
	for _, node := range dirtyNodes {
		dirtyPageIDs[node.pageID] = true
	}

	if !dirtyPageIDs[1] || !dirtyPageIDs[3] {
		t.Error("Incorrect dirty nodes returned")
	}
}

func TestNodeCacheFlush(t *testing.T) {
	cache := NewNodeCache(3)

	// Create dirty nodes
	node1 := &BTreeNode{pageID: 1, isLeaf: true, isDirty: true}
	node2 := &BTreeNode{pageID: 2, isLeaf: true, isDirty: true}

	cache.Put(1, node1)
	cache.Put(2, node2)

	// Verify nodes are dirty
	dirtyNodes := cache.GetDirtyNodes()
	if len(dirtyNodes) != 2 {
		t.Errorf("Expected 2 dirty nodes, got %d", len(dirtyNodes))
	}

	// Flush cache
	cache.Flush()

	// Verify nodes are no longer dirty
	dirtyNodes = cache.GetDirtyNodes()
	if len(dirtyNodes) != 0 {
		t.Errorf("Expected 0 dirty nodes after flush, got %d", len(dirtyNodes))
	}

	// Nodes should still be in cache
	if cache.Size() != 2 {
		t.Errorf("Expected cache size 2 after flush, got %d", cache.Size())
	}
}

func TestNodeCacheDefaultCapacity(t *testing.T) {
	// Test with invalid capacity
	cache := NewNodeCache(0)
	if cache.capacity != 500 {
		t.Errorf("Expected default capacity 500, got %d", cache.capacity)
	}

	cache = NewNodeCache(-10)
	if cache.capacity != 500 {
		t.Errorf("Expected default capacity 500, got %d", cache.capacity)
	}
}

func TestNodeCacheHitRate(t *testing.T) {
	cache := NewNodeCache(3)

	node1 := &BTreeNode{pageID: 1, isLeaf: true}
	cache.Put(1, node1)

	// 3 hits
	cache.Get(1)
	cache.Get(1)
	cache.Get(1)

	// 2 misses
	cache.Get(999)
	cache.Get(888)

	hits, misses, hitRate := cache.Stats()
	if hits != 3 || misses != 2 {
		t.Errorf("Expected 3 hits and 2 misses, got %d hits and %d misses", hits, misses)
	}

	expectedHitRate := 3.0 / 5.0
	if hitRate != expectedHitRate {
		t.Errorf("Expected hit rate %f, got %f", expectedHitRate, hitRate)
	}
}
