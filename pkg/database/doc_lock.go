package database

import (
	"hash/fnv"
	"sync"
)

// DocumentLockManager manages fine-grained locks on individual documents
// Uses lock striping to reduce contention on the lock map
type DocumentLockManager struct {
	// Number of lock stripes (power of 2 for efficient modulo)
	numStripes int
	// Array of lock stripes, each managing a subset of document locks
	stripes []*lockStripe
}

// lockStripe manages locks for a subset of documents
type lockStripe struct {
	mu    sync.Mutex
	locks map[string]*sync.RWMutex
}

// NewDocumentLockManager creates a new document lock manager with the specified number of stripes
// numStripes should be a power of 2 for optimal performance (default: 256)
func NewDocumentLockManager(numStripes int) *DocumentLockManager {
	if numStripes <= 0 {
		numStripes = 256 // Default to 256 stripes
	}

	dlm := &DocumentLockManager{
		numStripes: numStripes,
		stripes:    make([]*lockStripe, numStripes),
	}

	for i := 0; i < numStripes; i++ {
		dlm.stripes[i] = &lockStripe{
			locks: make(map[string]*sync.RWMutex),
		}
	}

	return dlm
}

// getStripe returns the lock stripe for a given document ID
func (dlm *DocumentLockManager) getStripe(docID string) *lockStripe {
	// Use FNV-1a hash for consistent distribution
	h := fnv.New32a()
	h.Write([]byte(docID))
	hash := h.Sum32()
	return dlm.stripes[int(hash)%dlm.numStripes]
}

// getLock retrieves or creates a lock for a document ID
// Caller must hold the stripe's mutex
func (stripe *lockStripe) getLock(docID string) *sync.RWMutex {
	if lock, exists := stripe.locks[docID]; exists {
		return lock
	}
	lock := &sync.RWMutex{}
	stripe.locks[docID] = lock
	return lock
}

// RLock acquires a read lock on a document
func (dlm *DocumentLockManager) RLock(docID string) {
	stripe := dlm.getStripe(docID)
	stripe.mu.Lock()
	lock := stripe.getLock(docID)
	stripe.mu.Unlock()
	lock.RLock()
}

// RUnlock releases a read lock on a document
func (dlm *DocumentLockManager) RUnlock(docID string) {
	stripe := dlm.getStripe(docID)
	stripe.mu.Lock()
	if lock, exists := stripe.locks[docID]; exists {
		stripe.mu.Unlock()
		lock.RUnlock()
	} else {
		stripe.mu.Unlock()
	}
}

// Lock acquires a write lock on a document
func (dlm *DocumentLockManager) Lock(docID string) {
	stripe := dlm.getStripe(docID)
	stripe.mu.Lock()
	lock := stripe.getLock(docID)
	stripe.mu.Unlock()
	lock.Lock()
}

// Unlock releases a write lock on a document
func (dlm *DocumentLockManager) Unlock(docID string) {
	stripe := dlm.getStripe(docID)
	stripe.mu.Lock()
	if lock, exists := stripe.locks[docID]; exists {
		stripe.mu.Unlock()
		lock.Unlock()
	} else {
		stripe.mu.Unlock()
	}
}

// LockMultiple acquires write locks on multiple documents in a consistent order (sorted by ID)
// This prevents deadlocks when multiple goroutines try to lock the same set of documents
func (dlm *DocumentLockManager) LockMultiple(docIDs []string) {
	// Sort IDs to ensure consistent lock order
	sortedIDs := make([]string, len(docIDs))
	copy(sortedIDs, docIDs)
	sortStrings(sortedIDs)

	// Lock in sorted order
	for _, id := range sortedIDs {
		dlm.Lock(id)
	}
}

// UnlockMultiple releases write locks on multiple documents
func (dlm *DocumentLockManager) UnlockMultiple(docIDs []string) {
	// Unlock in any order (doesn't matter for release)
	for _, id := range docIDs {
		dlm.Unlock(id)
	}
}

// sortStrings is a simple insertion sort for small arrays
func sortStrings(arr []string) {
	n := len(arr)
	for i := 1; i < n; i++ {
		key := arr[i]
		j := i - 1
		for j >= 0 && arr[j] > key {
			arr[j+1] = arr[j]
			j--
		}
		arr[j+1] = key
	}
}

// Cleanup removes unused locks from the lock map to prevent memory leaks
// Should be called periodically when the collection is not under heavy load
func (dlm *DocumentLockManager) Cleanup() {
	for _, stripe := range dlm.stripes {
		stripe.mu.Lock()
		// Clear the entire map - locks that are still in use will be recreated on demand
		stripe.locks = make(map[string]*sync.RWMutex)
		stripe.mu.Unlock()
	}
}
