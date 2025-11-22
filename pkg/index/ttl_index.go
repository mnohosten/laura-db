package index

import (
	"fmt"
	"sync"
	"time"
)

// TTLIndex represents a time-to-live index that tracks document expiration times
type TTLIndex struct {
	name       string
	fieldPath  string
	ttlSeconds int64 // Documents expire after this many seconds from the timestamp field

	// Maps document ID to expiration timestamp
	expirationTimes map[string]time.Time

	mu sync.RWMutex
}

// NewTTLIndex creates a new TTL index
func NewTTLIndex(name, fieldPath string, ttlSeconds int64) *TTLIndex {
	return &TTLIndex{
		name:            name,
		fieldPath:       fieldPath,
		ttlSeconds:      ttlSeconds,
		expirationTimes: make(map[string]time.Time),
	}
}

// Index adds a document with its timestamp to the TTL index
func (idx *TTLIndex) Index(docID string, timestamp time.Time) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Calculate expiration time
	expirationTime := timestamp.Add(time.Duration(idx.ttlSeconds) * time.Second)
	idx.expirationTimes[docID] = expirationTime

	return nil
}

// Remove removes a document from the TTL index
func (idx *TTLIndex) Remove(docID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	delete(idx.expirationTimes, docID)
}

// GetExpiredDocuments returns a list of document IDs that have expired
func (idx *TTLIndex) GetExpiredDocuments(currentTime time.Time) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	expired := make([]string, 0)

	for docID, expirationTime := range idx.expirationTimes {
		if currentTime.After(expirationTime) {
			expired = append(expired, docID)
		}
	}

	return expired
}

// Name returns the index name
func (idx *TTLIndex) Name() string {
	return idx.name
}

// FieldPath returns the indexed field path
func (idx *TTLIndex) FieldPath() string {
	return idx.fieldPath
}

// TTLSeconds returns the TTL duration in seconds
func (idx *TTLIndex) TTLSeconds() int64 {
	return idx.ttlSeconds
}

// Count returns the number of documents tracked in the TTL index
func (idx *TTLIndex) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return len(idx.expirationTimes)
}

// GetExpirationTime returns the expiration time for a specific document
func (idx *TTLIndex) GetExpirationTime(docID string) (time.Time, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	expirationTime, exists := idx.expirationTimes[docID]
	return expirationTime, exists
}

// String returns a string representation of the TTL index
func (idx *TTLIndex) String() string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return fmt.Sprintf("TTLIndex{name: %s, field: %s, ttl: %ds, docs: %d}",
		idx.name, idx.fieldPath, idx.ttlSeconds, len(idx.expirationTimes))
}
