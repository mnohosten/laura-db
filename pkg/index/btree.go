package index

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/mnohosten/laura-db/pkg/document"
)

// BTree is a B+ tree index
type BTree struct {
	root         *BTreeNode
	order        int // Maximum number of keys per node
	mu           sync.RWMutex
	size         int
	height       int
	lastSplitKey interface{} // Temporarily stores the separator/promoted key from a split
}

// BTreeNode represents a node in the B+ tree
type BTreeNode struct {
	isLeaf   bool
	keys     []interface{}
	values   []interface{} // Only used in leaf nodes
	children []*BTreeNode  // Only used in internal nodes
	next     *BTreeNode    // Only used in leaf nodes (linked list)
	parent   *BTreeNode
}

// NewBTree creates a new B+ tree with the given order
func NewBTree(order int) *BTree {
	if order < 3 {
		order = 3 // Minimum order
	}

	return &BTree{
		root: &BTreeNode{
			isLeaf:   true,
			keys:     make([]interface{}, 0),
			values:   make([]interface{}, 0),
			children: nil,
			next:     nil,
		},
		order:  order,
		size:   0,
		height: 1,
	}
}

// Insert inserts a key-value pair into the B+ tree
func (bt *BTree) Insert(key interface{}, value interface{}) error {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	// Check if key already exists
	if _, exists := bt.searchNode(bt.root, key); exists {
		return ErrDuplicateKey
	}

	// Insert into the tree
	newRoot, err := bt.insertIntoNode(bt.root, key, value)
	if err != nil {
		return err
	}

	if newRoot != nil {
		bt.root = newRoot
		bt.height++
	}

	bt.size++
	return nil
}

// insertIntoNode inserts a key-value pair into a node, returns new root if split
func (bt *BTree) insertIntoNode(node *BTreeNode, key interface{}, value interface{}) (*BTreeNode, error) {
	if node.isLeaf {
		return bt.insertIntoLeaf(node, key, value)
	}
	return bt.insertIntoInternal(node, key, value)
}

// insertIntoLeaf inserts into a leaf node
func (bt *BTree) insertIntoLeaf(leaf *BTreeNode, key interface{}, value interface{}) (*BTreeNode, error) {
	// Find insertion position
	pos := bt.findPosition(leaf.keys, key)

	// Insert key and value at position
	leaf.keys = append(leaf.keys[:pos], append([]interface{}{key}, leaf.keys[pos:]...)...)
	leaf.values = append(leaf.values[:pos], append([]interface{}{value}, leaf.values[pos:]...)...)

	// Check if split is needed
	if len(leaf.keys) >= bt.order {
		return bt.splitLeaf(leaf)
	}

	return nil, nil
}

// insertIntoInternal inserts into an internal node
func (bt *BTree) insertIntoInternal(node *BTreeNode, key interface{}, value interface{}) (*BTreeNode, error) {
	// Find child to insert into
	pos := bt.findPosition(node.keys, key)
	child := node.children[pos]

	// Recursively insert
	newChild, err := bt.insertIntoNode(child, key, value)
	if err != nil {
		return nil, err
	}

	// No split occurred
	if newChild == nil {
		return nil, nil
	}

	// Split occurred, use the stored split key
	splitKey := bt.lastSplitKey
	pos = bt.findPosition(node.keys, splitKey)

	// Insert new key and child
	node.keys = append(node.keys[:pos], append([]interface{}{splitKey}, node.keys[pos:]...)...)
	node.children = append(node.children[:pos+1], append([]*BTreeNode{newChild}, node.children[pos+1:]...)...)

	// Check if this node needs to split
	if len(node.keys) >= bt.order {
		return bt.splitInternal(node)
	}

	return nil, nil
}

// splitLeaf splits a leaf node and returns the new leaf and separator key
func (bt *BTree) splitLeaf(leaf *BTreeNode) (*BTreeNode, error) {
	mid := len(leaf.keys) / 2

	// Create new leaf with right half
	newLeaf := &BTreeNode{
		isLeaf: true,
		keys:   append([]interface{}{}, leaf.keys[mid:]...),
		values: append([]interface{}{}, leaf.values[mid:]...),
		next:   leaf.next,
		parent: leaf.parent,
	}

	// Separator key is the first key of the right half
	separatorKey := newLeaf.keys[0]

	// Update original leaf with left half
	leaf.keys = leaf.keys[:mid]
	leaf.values = leaf.values[:mid]
	leaf.next = newLeaf

	// If this is the root, create new root
	if leaf.parent == nil {
		newRoot := &BTreeNode{
			isLeaf:   false,
			keys:     []interface{}{separatorKey},
			children: []*BTreeNode{leaf, newLeaf},
		}
		leaf.parent = newRoot
		newLeaf.parent = newRoot
		return newRoot, nil
	}

	// Store the separator key for the parent to use
	bt.lastSplitKey = separatorKey
	return newLeaf, nil
}

// splitInternal splits an internal node and returns the new node and promoted key
func (bt *BTree) splitInternal(node *BTreeNode) (*BTreeNode, error) {
	mid := len(node.keys) / 2

	// Promote middle key
	promoteKey := node.keys[mid]

	// Create new internal node with right half (NOT including the promoted key)
	newNode := &BTreeNode{
		isLeaf:   false,
		keys:     append([]interface{}{}, node.keys[mid+1:]...),
		children: append([]*BTreeNode{}, node.children[mid+1:]...),
		parent:   node.parent,
	}

	// Update children's parent pointers
	for _, child := range newNode.children {
		child.parent = newNode
	}

	// Update original node with left half
	node.keys = node.keys[:mid]
	node.children = node.children[:mid+1]

	// If this is the root, create new root
	if node.parent == nil {
		newRoot := &BTreeNode{
			isLeaf:   false,
			keys:     []interface{}{promoteKey},
			children: []*BTreeNode{node, newNode},
		}
		node.parent = newRoot
		newNode.parent = newRoot
		return newRoot, nil
	}

	// Store the promoted key for the parent to use
	bt.lastSplitKey = promoteKey
	return newNode, nil
}

// Search finds a value by key
func (bt *BTree) Search(key interface{}) (interface{}, bool) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	return bt.searchNode(bt.root, key)
}

// searchNode searches for a key in a subtree
func (bt *BTree) searchNode(node *BTreeNode, key interface{}) (interface{}, bool) {
	if node.isLeaf {
		// Search in leaf node
		for i, k := range node.keys {
			cmp := bt.compare(key, k)
			if cmp == 0 {
				return node.values[i], true
			}
			if cmp < 0 {
				return nil, false
			}
		}
		return nil, false
	}

	// Search in internal node
	pos := bt.findPosition(node.keys, key)
	return bt.searchNode(node.children[pos], key)
}

// Delete removes a key from the tree
func (bt *BTree) Delete(key interface{}) error {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	if _, exists := bt.searchNode(bt.root, key); !exists {
		return ErrKeyNotFound
	}

	// Simple deletion without rebalancing (for educational purposes)
	// Production implementation would handle underflow
	bt.deleteFromNode(bt.root, key)
	bt.size--
	return nil
}

// deleteFromNode deletes a key from a node
func (bt *BTree) deleteFromNode(node *BTreeNode, key interface{}) bool {
	if node.isLeaf {
		// Find and remove from leaf
		for i, k := range node.keys {
			if bt.compare(key, k) == 0 {
				node.keys = append(node.keys[:i], node.keys[i+1:]...)
				node.values = append(node.values[:i], node.values[i+1:]...)
				return true
			}
		}
		return false
	}

	// Find child and recursively delete
	pos := bt.findPosition(node.keys, key)
	return bt.deleteFromNode(node.children[pos], key)
}

// RangeScan returns all key-value pairs in the range [start, end]
func (bt *BTree) RangeScan(start, end interface{}) ([]interface{}, []interface{}) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	keys := make([]interface{}, 0)
	values := make([]interface{}, 0)

	// If start is nil, begin from the leftmost leaf
	var leaf *BTreeNode
	if start == nil {
		// Find leftmost leaf
		leaf = bt.root
		for !leaf.isLeaf {
			leaf = leaf.children[0]
		}
	} else {
		// Find the first leaf containing start
		leaf = bt.findLeaf(bt.root, start)
	}

	// Scan through leaves using the linked list
	for leaf != nil {
		for i, k := range leaf.keys {
			// Check if key is >= start (or start is nil)
			includeStart := start == nil || bt.compare(k, start) >= 0

			// Check if key is <= end (or end is nil)
			includeEnd := end == nil || bt.compare(k, end) <= 0

			if includeStart && includeEnd {
				keys = append(keys, k)
				values = append(values, leaf.values[i])
			}

			// Stop if we've passed the end
			if end != nil && bt.compare(k, end) > 0 {
				return keys, values
			}
		}
		leaf = leaf.next
	}

	return keys, values
}

// findLeaf finds the leaf node that should contain the key
func (bt *BTree) findLeaf(node *BTreeNode, key interface{}) *BTreeNode {
	if node.isLeaf {
		return node
	}

	pos := bt.findPosition(node.keys, key)
	return bt.findLeaf(node.children[pos], key)
}

// findPosition finds the position where key should be inserted
func (bt *BTree) findPosition(keys []interface{}, key interface{}) int {
	for i, k := range keys {
		if bt.compare(key, k) < 0 {
			return i
		}
	}
	return len(keys)
}

// compare compares two keys
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func (bt *BTree) compare(a, b interface{}) int {
	// Handle CompositeKey first (for compound indexes)
	if va, ok := a.(*CompositeKey); ok {
		if vb, ok := b.(*CompositeKey); ok {
			return va.Compare(vb)
		}
		// If one is composite and other isn't, they're not comparable
		return 0
	}

	// Need to import document package for ObjectID
	// So we'll handle it by type assertion
	switch va := a.(type) {
	case int64:
		if vb, ok := b.(int64); ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case int32:
		if vb, ok := b.(int32); ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case float64:
		if vb, ok := b.(float64); ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case string:
		if vb, ok := b.(string); ok {
			return bytes.Compare([]byte(va), []byte(vb))
		}
	case []byte:
		if vb, ok := b.([]byte); ok {
			return bytes.Compare(va, vb)
		}
	case document.ObjectID:
		if vb, ok := b.(document.ObjectID); ok {
			return bytes.Compare(va[:], vb[:])
		}
	}

	// Default: treat as equal if types don't match
	return 0
}

// Size returns the number of keys in the tree
func (bt *BTree) Size() int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.size
}

// Height returns the height of the tree
func (bt *BTree) Height() int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.height
}

// Print prints the tree structure (for debugging)
func (bt *BTree) Print() {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	fmt.Println("B+ Tree Structure:")
	bt.printNode(bt.root, 0)
}

// printNode prints a node recursively
func (bt *BTree) printNode(node *BTreeNode, level int) {
	if node == nil {
		return
	}

	indent := ""
	for i := 0; i < level; i++ {
		indent += "  "
	}

	if node.isLeaf {
		fmt.Printf("%sLeaf: %v\n", indent, node.keys)
	} else {
		fmt.Printf("%sInternal: %v\n", indent, node.keys)
		for _, child := range node.children {
			bt.printNode(child, level+1)
		}
	}
}
