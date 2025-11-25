package index

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/storage"
)

// TestBTreeDiskLoadAndWrite tests loading and writing nodes to disk
func TestBTreeDiskLoadAndWrite(t *testing.T) {
	// Create temporary directory and disk manager
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/btree_test.db"

	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create a test node
	node := &BTreeNode{
		isLeaf: true,
		keys: []interface{}{
			int64(100),
			int64(200),
			int64(300),
		},
		values: []interface{}{
			make([]byte, 12),
			make([]byte, 12),
			make([]byte, 12),
		},
	}

	// Write node to disk
	err = WriteNodeToDisk(diskMgr, node, 1, 1)
	if err != nil {
		t.Fatalf("Failed to write node to disk: %v", err)
	}

	// Note: pageID can be 0 (first page), so just verify it was written successfully
	// The fact that WriteNodeToDisk didn't return an error means the node was persisted

	// Verify node is not dirty
	if node.isDirty {
		t.Error("Node should not be dirty after writing")
	}

	pageID := node.pageID

	// Load node from disk (without cache)
	loadedNode, err := LoadNodeFromDisk(diskMgr, storage.PageID(pageID), nil)
	if err != nil {
		t.Fatalf("Failed to load node from disk: %v", err)
	}

	// Verify loaded node properties
	if !loadedNode.isLeaf {
		t.Error("Loaded node should be a leaf")
	}

	if len(loadedNode.keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(loadedNode.keys))
	}

	// Verify keys match
	for i, key := range node.keys {
		if loadedNode.keys[i] != key {
			t.Errorf("Key %d mismatch: got %v, want %v", i, loadedNode.keys[i], key)
		}
	}

	if !loadedNode.isLoaded {
		t.Error("Loaded node should have isLoaded = true")
	}

	if loadedNode.pageID != pageID {
		t.Errorf("PageID mismatch: got %d, want %d", loadedNode.pageID, pageID)
	}
}

// TestBTreeDiskWithCache tests node caching
func TestBTreeDiskWithCache(t *testing.T) {
	// Create temporary directory and disk manager
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/btree_test.db"

	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create node cache
	cache := NewNodeCache(10)

	// Create a test node
	node := &BTreeNode{
		isLeaf: true,
		keys:   []interface{}{int64(42)},
		values: []interface{}{make([]byte, 12)},
	}

	// Write node to disk
	err = WriteNodeToDisk(diskMgr, node, 1, 1)
	if err != nil {
		t.Fatalf("Failed to write node to disk: %v", err)
	}

	pageID := node.pageID

	// First load - should be a cache miss
	loadedNode1, err := LoadNodeFromDisk(diskMgr, storage.PageID(pageID), cache)
	if err != nil {
		t.Fatalf("Failed to load node from disk: %v", err)
	}

	hits, misses, _ := cache.Stats()
	if hits != 0 || misses != 1 {
		t.Errorf("Expected 0 hits and 1 miss, got %d hits and %d misses", hits, misses)
	}

	// Second load - should be a cache hit
	loadedNode2, err := LoadNodeFromDisk(diskMgr, storage.PageID(pageID), cache)
	if err != nil {
		t.Fatalf("Failed to load node from disk: %v", err)
	}

	hits, misses, _ = cache.Stats()
	if hits != 1 || misses != 1 {
		t.Errorf("Expected 1 hit and 1 miss, got %d hits and %d misses", hits, misses)
	}

	// Verify both loads returned the same node (from cache)
	if loadedNode1 != loadedNode2 {
		t.Error("Second load should return cached node")
	}

	// Verify cache contains the node
	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}
}

// TestFlushDirtyNodes tests flushing dirty nodes to disk
func TestFlushDirtyNodes(t *testing.T) {
	// Create temporary directory and disk manager
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/btree_test.db"

	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create node cache
	cache := NewNodeCache(10)

	// Create and write multiple nodes
	nodes := make([]*BTreeNode, 3)
	for i := 0; i < 3; i++ {
		nodes[i] = &BTreeNode{
			isLeaf: true,
			keys:   []interface{}{int64(i * 100)},
			values: []interface{}{make([]byte, 12)},
		}

		err = WriteNodeToDisk(diskMgr, nodes[i], 1, 1)
		if err != nil {
			t.Fatalf("Failed to write node %d to disk: %v", i, err)
		}

		// Add to cache
		cache.Put(nodes[i].pageID, nodes[i])
	}

	// Mark some nodes as dirty
	nodes[0].isDirty = true
	nodes[2].isDirty = true

	// Update cache with dirty nodes
	cache.Put(nodes[0].pageID, nodes[0])
	cache.Put(nodes[2].pageID, nodes[2])

	// Verify dirty nodes
	dirtyNodes := cache.GetDirtyNodes()
	if len(dirtyNodes) != 2 {
		t.Errorf("Expected 2 dirty nodes, got %d", len(dirtyNodes))
	}

	// Flush dirty nodes
	err = FlushDirtyNodes(diskMgr, cache, 1, 1)
	if err != nil {
		t.Fatalf("Failed to flush dirty nodes: %v", err)
	}

	// Verify no dirty nodes remain
	dirtyNodes = cache.GetDirtyNodes()
	if len(dirtyNodes) != 0 {
		t.Errorf("Expected 0 dirty nodes after flush, got %d", len(dirtyNodes))
	}

	// Verify all nodes are still in cache
	if cache.Size() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.Size())
	}
}

// TestBTreeDiskInternalNode tests internal node serialization
func TestBTreeDiskInternalNode(t *testing.T) {
	// Create temporary directory and disk manager
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/btree_test.db"

	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create an internal node
	node := &BTreeNode{
		isLeaf: false,
		keys: []interface{}{
			int64(50),
			int64(100),
		},
		children: []*BTreeNode{
			{pageID: 10}, // Placeholder children
			{pageID: 20},
			{pageID: 30},
		},
	}

	// Write node to disk
	err = WriteNodeToDisk(diskMgr, node, 1, 1)
	if err != nil {
		t.Fatalf("Failed to write node to disk: %v", err)
	}

	pageID := node.pageID

	// Load node from disk
	loadedNode, err := LoadNodeFromDisk(diskMgr, storage.PageID(pageID), nil)
	if err != nil {
		t.Fatalf("Failed to load node from disk: %v", err)
	}

	// Verify loaded node properties
	if loadedNode.isLeaf {
		t.Error("Loaded node should be internal (not leaf)")
	}

	if len(loadedNode.keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(loadedNode.keys))
	}

	// Verify keys match
	expectedKeys := []int64{50, 100}
	for i, expectedKey := range expectedKeys {
		if loadedNode.keys[i] != expectedKey {
			t.Errorf("Key %d mismatch: got %v, want %v", i, loadedNode.keys[i], expectedKey)
		}
	}
}

// TestBTreeDiskMultipleNodeTypes tests different key types
func TestBTreeDiskMultipleNodeTypes(t *testing.T) {
	// Create temporary directory and disk manager
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/btree_test.db"

	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	testCases := []struct {
		name string
		keys []interface{}
	}{
		{
			name: "int64 keys",
			keys: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			name: "string keys",
			keys: []interface{}{"apple", "banana", "cherry"},
		},
		{
			name: "float64 keys",
			keys: []interface{}{float64(1.1), float64(2.2), float64(3.3)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create node with specific key type
			node := &BTreeNode{
				isLeaf: true,
				keys:   tc.keys,
				values: []interface{}{
					make([]byte, 12),
					make([]byte, 12),
					make([]byte, 12),
				},
			}

			// Write to disk
			err := WriteNodeToDisk(diskMgr, node, 1, 1)
			if err != nil {
				t.Fatalf("Failed to write node: %v", err)
			}

			// Load from disk
			loadedNode, err := LoadNodeFromDisk(diskMgr, storage.PageID(node.pageID), nil)
			if err != nil {
				t.Fatalf("Failed to load node: %v", err)
			}

			// Verify keys match
			if len(loadedNode.keys) != len(tc.keys) {
				t.Errorf("Key count mismatch: got %d, want %d", len(loadedNode.keys), len(tc.keys))
			}

			for i, expectedKey := range tc.keys {
				if loadedNode.keys[i] != expectedKey {
					t.Errorf("Key %d mismatch: got %v, want %v", i, loadedNode.keys[i], expectedKey)
				}
			}
		})
	}
}
