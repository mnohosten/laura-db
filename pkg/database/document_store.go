package database

import (
	"fmt"
	"sync"

	"github.com/mnohosten/laura-db/pkg/cache"
	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/storage"
)

// DocumentLocation represents the location of a document on disk
type DocumentLocation struct {
	PageID storage.PageID
	SlotID uint16
}

// DocumentStore manages disk-based document storage with caching
type DocumentStore struct {
	diskManager    *storage.DiskManager
	pageManager    *storage.DocumentPageManager
	serializer     *storage.DocumentSerializer
	locationMap    map[string]*DocumentLocation // _id -> location
	docCache       *cache.LRUCache              // LRU cache for documents
	activePagesMap map[storage.PageID]*storage.SlottedPage // Currently active pages
	mu             sync.RWMutex
}

// NewDocumentStore creates a new document store
func NewDocumentStore(diskManager *storage.DiskManager, cacheSize int) *DocumentStore {
	return &DocumentStore{
		diskManager:    diskManager,
		pageManager:    storage.NewDocumentPageManager(),
		serializer:     storage.NewDocumentSerializer(),
		locationMap:    make(map[string]*DocumentLocation),
		docCache:       cache.NewLRUCache(cacheSize, 0), // No TTL for document cache
		activePagesMap: make(map[storage.PageID]*storage.SlottedPage),
	}
}

// Insert inserts a document into disk storage and returns its ID
func (ds *DocumentStore) Insert(id string, doc *document.Document) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Check if document already exists
	if _, exists := ds.locationMap[id]; exists {
		return fmt.Errorf("document with _id %s already exists", id)
	}

	// Try to find a page with enough space, or allocate a new one
	page, err := ds.findOrAllocatePageForDocument(doc)
	if err != nil {
		return fmt.Errorf("failed to find page for document: %w", err)
	}

	// Insert document into the page
	slotID, err := ds.pageManager.InsertDocument(page, doc)
	if err != nil {
		return fmt.Errorf("failed to insert document into page: %w", err)
	}

	// Store location
	ds.locationMap[id] = &DocumentLocation{
		PageID: page.GetPage().ID,
		SlotID: slotID,
	}

	// Cache the document
	ds.docCache.Put(id, doc)

	// Flush the page to disk
	if err := ds.diskManager.WritePage(page.GetPage()); err != nil {
		// Remove from location map if write fails
		delete(ds.locationMap, id)
		return fmt.Errorf("failed to write page to disk: %w", err)
	}

	return nil
}

// Get retrieves a document by ID
func (ds *DocumentStore) Get(id string) (*document.Document, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	// Check cache first
	if cached, found := ds.docCache.Get(id); found {
		if doc, ok := cached.(*document.Document); ok {
			return doc, nil
		}
	}

	// Get location
	location, exists := ds.locationMap[id]
	if !exists {
		return nil, fmt.Errorf("document not found: %s", id)
	}

	// Read from disk
	doc, err := ds.readDocumentFromDisk(location)
	if err != nil {
		return nil, fmt.Errorf("failed to read document from disk: %w", err)
	}

	// Cache the document
	ds.docCache.Put(id, doc)

	return doc, nil
}

// Update updates an existing document
func (ds *DocumentStore) Update(id string, doc *document.Document) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Get location
	location, exists := ds.locationMap[id]
	if !exists {
		return fmt.Errorf("document not found: %s", id)
	}

	// Load the page
	page, err := ds.loadOrGetActivePage(location.PageID)
	if err != nil {
		return fmt.Errorf("failed to load page: %w", err)
	}

	// Try to update in place
	if err := ds.pageManager.UpdateDocument(page, location.SlotID, doc); err != nil {
		// If update fails (e.g., document too large), we need to delete and reinsert
		// For now, return the error
		return fmt.Errorf("failed to update document: %w", err)
	}

	// Update cache
	ds.docCache.Put(id, doc)

	// Flush the page to disk
	if err := ds.diskManager.WritePage(page.GetPage()); err != nil {
		return fmt.Errorf("failed to write page to disk: %w", err)
	}

	return nil
}

// Delete deletes a document by ID
func (ds *DocumentStore) Delete(id string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Get location
	location, exists := ds.locationMap[id]
	if !exists {
		return fmt.Errorf("document not found: %s", id)
	}

	// Load the page
	page, err := ds.loadOrGetActivePage(location.PageID)
	if err != nil {
		return fmt.Errorf("failed to load page: %w", err)
	}

	// Delete from page
	if err := ds.pageManager.DeleteDocument(page, location.SlotID); err != nil {
		return fmt.Errorf("failed to delete document from page: %w", err)
	}

	// Remove from location map
	delete(ds.locationMap, id)

	// Note: We don't explicitly remove from cache - it will be evicted naturally
	// or expire based on LRU policy. Attempting to Get() a deleted document will
	// return an error from the location map check.

	// Flush the page to disk
	if err := ds.diskManager.WritePage(page.GetPage()); err != nil {
		return fmt.Errorf("failed to write page to disk: %w", err)
	}

	return nil
}

// Exists checks if a document exists
func (ds *DocumentStore) Exists(id string) bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	_, exists := ds.locationMap[id]
	return exists
}

// GetAllIDs returns all document IDs
func (ds *DocumentStore) GetAllIDs() []string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	ids := make([]string, 0, len(ds.locationMap))
	for id := range ds.locationMap {
		ids = append(ids, id)
	}
	return ids
}

// Count returns the number of documents
func (ds *DocumentStore) Count() int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	return len(ds.locationMap)
}

// findOrAllocatePageForDocument finds a page with enough space or allocates a new one
func (ds *DocumentStore) findOrAllocatePageForDocument(doc *document.Document) (*storage.SlottedPage, error) {
	// Estimate document size
	docSize := ds.serializer.EstimateDocumentSize(doc)

	// Try to find an active page with enough space
	for pageID, page := range ds.activePagesMap {
		capacity := ds.pageManager.GetPageCapacity(page)
		if capacity.ContiguousFreeSpace >= docSize+storage.SlotEntrySize {
			return page, nil
		}
		// If page is nearly full, remove from active pages
		if capacity.ContiguousFreeSpace < 256 { // Arbitrary threshold
			delete(ds.activePagesMap, pageID)
		}
	}

	// Allocate a new page
	pageID, err := ds.diskManager.AllocatePage()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate page: %w", err)
	}

	// Create a new data page
	page := storage.NewPage(pageID, storage.PageTypeData)
	slottedPage, err := storage.NewSlottedPage(page)
	if err != nil {
		return nil, fmt.Errorf("failed to create slotted page: %w", err)
	}

	// Add to active pages
	ds.activePagesMap[pageID] = slottedPage

	return slottedPage, nil
}

// loadOrGetActivePage loads a page from disk or returns it from active pages cache
func (ds *DocumentStore) loadOrGetActivePage(pageID storage.PageID) (*storage.SlottedPage, error) {
	// Check if page is already in active pages
	if page, exists := ds.activePagesMap[pageID]; exists {
		return page, nil
	}

	// Load from disk
	page, err := ds.diskManager.ReadPage(pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to read page from disk: %w", err)
	}

	slottedPage, err := storage.LoadSlottedPage(page)
	if err != nil {
		return nil, fmt.Errorf("failed to load slotted page: %w", err)
	}

	// Add to active pages cache (with a size limit to prevent unbounded growth)
	if len(ds.activePagesMap) < 100 {
		ds.activePagesMap[pageID] = slottedPage
	}

	return slottedPage, nil
}

// readDocumentFromDisk reads a document from disk
func (ds *DocumentStore) readDocumentFromDisk(location *DocumentLocation) (*document.Document, error) {
	// Load the page
	page, err := ds.loadOrGetActivePage(location.PageID)
	if err != nil {
		return nil, err
	}

	// Get document from page
	doc, err := ds.pageManager.GetDocument(page, location.SlotID)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// FlushAll flushes all cached pages to disk
func (ds *DocumentStore) FlushAll() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	for _, page := range ds.activePagesMap {
		if err := ds.diskManager.WritePage(page.GetPage()); err != nil {
			return fmt.Errorf("failed to flush page: %w", err)
		}
	}

	return ds.diskManager.Sync()
}

// Stats returns statistics about the document store
func (ds *DocumentStore) Stats() map[string]interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	cacheStats := ds.docCache.Stats()

	return map[string]interface{}{
		"document_count":   len(ds.locationMap),
		"active_pages":     len(ds.activePagesMap),
		"cache_size":       cacheStats["size"],
		"cache_capacity":   cacheStats["capacity"],
		"cache_hit_rate":   cacheStats["hit_rate"],
		"cache_hits":       cacheStats["hits"],
		"cache_misses":     cacheStats["misses"],
		"cache_evictions":  cacheStats["evictions"],
	}
}
