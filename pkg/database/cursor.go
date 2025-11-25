package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/query"
)

// Cursor represents an iterator over query results
type Cursor struct {
	id           string
	collection   *Collection
	query        *query.Query
	results      []*document.Document
	position     int
	batchSize    int
	timeout      time.Duration
	lastAccessed time.Time
	exhausted    bool
	mu           sync.RWMutex
}

// CursorOptions contains options for cursor creation
type CursorOptions struct {
	BatchSize int           // Number of documents to fetch per batch (default: 100)
	Timeout   time.Duration // Cursor idle timeout (default: 10 minutes)
}

// DefaultCursorOptions returns default cursor options
func DefaultCursorOptions() *CursorOptions {
	return &CursorOptions{
		BatchSize: 100,
		Timeout:   10 * time.Minute,
	}
}

// NewCursor creates a new cursor for a query
func NewCursor(collection *Collection, q *query.Query, options *CursorOptions) (*Cursor, error) {
	if options == nil {
		options = DefaultCursorOptions()
	}

	// Create default query if nil
	if q == nil {
		q = query.NewQuery(map[string]interface{}{})
	}

	// Generate unique cursor ID
	cursorID, err := generateCursorID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate cursor ID: %w", err)
	}

	cursor := &Cursor{
		id:           cursorID,
		collection:   collection,
		query:        q,
		position:     0,
		batchSize:    options.BatchSize,
		timeout:      options.Timeout,
		lastAccessed: time.Now(),
		exhausted:    false,
	}

	// Execute query and store results
	// Note: In a production implementation, this would use iterators
	// to avoid loading all results into memory at once
	results, err := collection.executeQuery(q)
	if err != nil {
		return nil, err
	}

	cursor.results = results
	// Don't mark as exhausted if empty - let position tracking handle it
	if len(results) == 0 {
		cursor.exhausted = false
	}

	return cursor, nil
}

// ID returns the cursor's unique identifier
func (c *Cursor) ID() string {
	return c.id
}

// HasNext returns true if there are more documents to fetch
func (c *Cursor) HasNext() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return !c.exhausted && c.position < len(c.results)
}

// Next returns the next document in the result set
func (c *Cursor) Next() (*document.Document, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastAccessed = time.Now()

	if c.exhausted {
		return nil, fmt.Errorf("cursor exhausted")
	}

	if c.position >= len(c.results) {
		c.exhausted = true
		return nil, fmt.Errorf("no more documents")
	}

	doc := c.results[c.position]
	c.position++

	// Mark as exhausted if we've reached the end
	if c.position >= len(c.results) {
		c.exhausted = true
	}

	return doc, nil
}

// NextBatch returns the next batch of documents
func (c *Cursor) NextBatch() ([]*document.Document, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastAccessed = time.Now()

	// Return empty batch if already exhausted or no more results
	if c.exhausted || c.position >= len(c.results) {
		c.exhausted = true
		return []*document.Document{}, nil
	}

	// Calculate batch end position
	endPos := c.position + c.batchSize
	if endPos > len(c.results) {
		endPos = len(c.results)
	}

	// Extract batch
	batch := c.results[c.position:endPos]
	c.position = endPos

	// Mark as exhausted if we've reached the end
	if c.position >= len(c.results) {
		c.exhausted = true
	}

	return batch, nil
}

// Count returns the total number of documents in the result set
func (c *Cursor) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.results)
}

// Position returns the current position in the result set
func (c *Cursor) Position() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.position
}

// BatchSize returns the cursor's batch size
func (c *Cursor) BatchSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.batchSize
}

// Remaining returns the number of documents remaining
func (c *Cursor) Remaining() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.results) - c.position
}

// IsExhausted returns true if the cursor is exhausted
func (c *Cursor) IsExhausted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.exhausted
}

// Close closes the cursor and releases resources
func (c *Cursor) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.exhausted = true
	c.results = nil
}

// IsTimedOut returns true if the cursor has exceeded its idle timeout
func (c *Cursor) IsTimedOut() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.lastAccessed) > c.timeout
}

// CursorManager manages server-side cursors
type CursorManager struct {
	cursors map[string]*Cursor
	mu      sync.RWMutex
}

// NewCursorManager creates a new cursor manager
func NewCursorManager() *CursorManager {
	return &CursorManager{
		cursors: make(map[string]*Cursor),
	}
}

// CreateCursor creates and registers a new cursor
func (cm *CursorManager) CreateCursor(collection *Collection, q *query.Query, options *CursorOptions) (*Cursor, error) {
	cursor, err := NewCursor(collection, q, options)
	if err != nil {
		return nil, err
	}

	cm.mu.Lock()
	cm.cursors[cursor.ID()] = cursor
	cm.mu.Unlock()

	return cursor, nil
}

// GetCursor retrieves a cursor by ID
func (cm *CursorManager) GetCursor(cursorID string) (*Cursor, error) {
	cm.mu.RLock()
	cursor, exists := cm.cursors[cursorID]
	cm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("cursor not found: %s", cursorID)
	}

	// Check if cursor has timed out
	if cursor.IsTimedOut() {
		cm.CloseCursor(cursorID)
		return nil, fmt.Errorf("cursor timed out: %s", cursorID)
	}

	return cursor, nil
}

// CloseCursor closes and removes a cursor
func (cm *CursorManager) CloseCursor(cursorID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cursor, exists := cm.cursors[cursorID]
	if !exists {
		return fmt.Errorf("cursor not found: %s", cursorID)
	}

	cursor.Close()
	delete(cm.cursors, cursorID)
	return nil
}

// CleanupTimedOutCursors removes cursors that have exceeded their timeout
func (cm *CursorManager) CleanupTimedOutCursors() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	removed := 0
	for id, cursor := range cm.cursors {
		if cursor.IsTimedOut() || cursor.IsExhausted() {
			cursor.Close()
			delete(cm.cursors, id)
			removed++
		}
	}

	return removed
}

// ActiveCursors returns the number of active cursors
func (cm *CursorManager) ActiveCursors() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.cursors)
}

// generateCursorID generates a unique cursor identifier
func generateCursorID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
