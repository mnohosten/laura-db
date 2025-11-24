package lsm

import (
	"bytes"
	"math/rand"
	"time"
)

const (
	maxLevel    = 16   // Maximum number of levels in skip list
	probability = 0.25 // Probability for level increase
)

// SkipList is a probabilistic data structure for sorted data
type SkipList struct {
	head   *SkipListNode
	level  int
	size   int
	random *rand.Rand
}

// SkipListNode represents a node in the skip list
type SkipListNode struct {
	key     []byte
	value   interface{}
	forward []*SkipListNode
}

// NewSkipList creates a new skip list
func NewSkipList() *SkipList {
	return &SkipList{
		head:   newSkipListNode(nil, nil, maxLevel),
		level:  1,
		size:   0,
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// newSkipListNode creates a new skip list node
func newSkipListNode(key []byte, value interface{}, level int) *SkipListNode {
	return &SkipListNode{
		key:     key,
		value:   value,
		forward: make([]*SkipListNode, level),
	}
}

// randomLevel generates a random level for a new node
func (sl *SkipList) randomLevel() int {
	level := 1
	for level < maxLevel && sl.random.Float32() < probability {
		level++
	}
	return level
}

// Insert inserts or updates a key-value pair
func (sl *SkipList) Insert(key []byte, value interface{}) {
	update := make([]*SkipListNode, maxLevel)
	current := sl.head

	// Find insertion point
	for i := sl.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && bytes.Compare(current.forward[i].key, key) < 0 {
			current = current.forward[i]
		}
		update[i] = current
	}

	current = current.forward[0]

	// Update existing key
	if current != nil && bytes.Equal(current.key, key) {
		current.value = value
		return
	}

	// Insert new node
	newLevel := sl.randomLevel()
	if newLevel > sl.level {
		for i := sl.level; i < newLevel; i++ {
			update[i] = sl.head
		}
		sl.level = newLevel
	}

	newNode := newSkipListNode(key, value, newLevel)
	for i := 0; i < newLevel; i++ {
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}

	sl.size++
}

// Search finds a value by key
func (sl *SkipList) Search(key []byte) (interface{}, bool) {
	current := sl.head

	for i := sl.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && bytes.Compare(current.forward[i].key, key) < 0 {
			current = current.forward[i]
		}
	}

	current = current.forward[0]
	if current != nil && bytes.Equal(current.key, key) {
		return current.value, true
	}

	return nil, false
}

// Delete removes a key from the skip list
func (sl *SkipList) Delete(key []byte) bool {
	update := make([]*SkipListNode, maxLevel)
	current := sl.head

	// Find node to delete
	for i := sl.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && bytes.Compare(current.forward[i].key, key) < 0 {
			current = current.forward[i]
		}
		update[i] = current
	}

	current = current.forward[0]

	if current == nil || !bytes.Equal(current.key, key) {
		return false
	}

	// Remove node from all levels
	for i := 0; i < sl.level; i++ {
		if update[i].forward[i] != current {
			break
		}
		update[i].forward[i] = current.forward[i]
	}

	// Update level if needed
	for sl.level > 1 && sl.head.forward[sl.level-1] == nil {
		sl.level--
	}

	sl.size--
	return true
}

// Size returns the number of entries
func (sl *SkipList) Size() int {
	return sl.size
}
