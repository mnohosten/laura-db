package mvcc

import (
	"sync"
)

// VersionStore stores multiple versions of each key
type VersionStore struct {
	data map[string]*VersionChain
	mu   sync.RWMutex
}

// VersionChain is a linked list of versions for a key
type VersionChain struct {
	Head *VersionNode
	mu   sync.RWMutex
}

// VersionNode represents a single version in the chain
type VersionNode struct {
	Value *VersionedValue
	Next  *VersionNode
}

// NewVersionStore creates a new version store
func NewVersionStore() *VersionStore {
	return &VersionStore{
		data: make(map[string]*VersionChain),
	}
}

// Put adds a new version for a key
func (vs *VersionStore) Put(key string, value *VersionedValue) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	chain, exists := vs.data[key]
	if !exists {
		chain = &VersionChain{
			Head: nil,
		}
		vs.data[key] = chain
	}

	chain.mu.Lock()
	defer chain.mu.Unlock()

	// Insert at head (most recent version)
	newNode := &VersionNode{
		Value: value,
		Next:  chain.Head,
	}
	chain.Head = newNode
}

// GetVersion retrieves a version visible to the given snapshot
func (vs *VersionStore) GetVersion(key string, snapshotVersion uint64) (interface{}, bool, error) {
	vs.mu.RLock()
	chain, exists := vs.data[key]
	vs.mu.RUnlock()

	if !exists {
		return nil, false, nil
	}

	chain.mu.RLock()
	defer chain.mu.RUnlock()

	// Traverse version chain to find visible version
	for node := chain.Head; node != nil; node = node.Next {
		version := node.Value

		// Check if this version is visible to the snapshot
		// A version is visible if:
		// 1. It was created before the snapshot
		// 2. It wasn't deleted before the snapshot
		if version.Version <= snapshotVersion {
			// Check if deleted
			if version.DeletedBy != 0 {
				return nil, false, nil
			}
			return version.Value, true, nil
		}
	}

	return nil, false, nil
}

// GetLatest retrieves the latest version of a key
func (vs *VersionStore) GetLatest(key string) (interface{}, bool) {
	vs.mu.RLock()
	chain, exists := vs.data[key]
	vs.mu.RUnlock()

	if !exists {
		return nil, false
	}

	chain.mu.RLock()
	defer chain.mu.RUnlock()

	if chain.Head == nil {
		return nil, false
	}

	// Check if deleted
	if chain.Head.Value.DeletedBy != 0 {
		return nil, false
	}

	return chain.Head.Value.Value, true
}

// GarbageCollect removes versions older than minVersion
func (vs *VersionStore) GarbageCollect(minVersion uint64) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	for key, chain := range vs.data {
		chain.mu.Lock()

		// Keep at least one version
		if chain.Head == nil || chain.Head.Next == nil {
			chain.mu.Unlock()
			continue
		}

		// Traverse and remove old versions
		prev := chain.Head
		current := chain.Head.Next

		for current != nil {
			if current.Value.Version < minVersion {
				// Remove this version
				prev.Next = current.Next
				current = prev.Next
			} else {
				prev = current
				current = current.Next
			}
		}

		chain.mu.Unlock()

		// If chain is empty, remove the key
		if chain.Head == nil {
			delete(vs.data, key)
		}
	}
}

// GetAllKeys returns all keys in the version store
func (vs *VersionStore) GetAllKeys() []string {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	keys := make([]string, 0, len(vs.data))
	for key := range vs.data {
		keys = append(keys, key)
	}
	return keys
}

// GetVersionCount returns the number of versions for a key
func (vs *VersionStore) GetVersionCount(key string) int {
	vs.mu.RLock()
	chain, exists := vs.data[key]
	vs.mu.RUnlock()

	if !exists {
		return 0
	}

	chain.mu.RLock()
	defer chain.mu.RUnlock()

	count := 0
	for node := chain.Head; node != nil; node = node.Next {
		count++
	}
	return count
}
