package storage

import (
	"fmt"
	"os"
	"sync"
)

// DiskManager handles physical disk I/O operations
type DiskManager struct {
	dataFile     *os.File
	nextPageID   PageID
	freePageList *FreePageList
	mu           sync.Mutex
	totalReads   int64
	totalWrites  int64
}

// NewDiskManager creates a new disk manager
func NewDiskManager(path string) (*DiskManager, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open data file: %w", err)
	}

	// Get file size to determine next page ID
	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat data file: %w", err)
	}

	nextPageID := PageID(fileInfo.Size() / PageSize)

	dm := &DiskManager{
		dataFile:     file,
		nextPageID:   nextPageID,
		freePageList: NewFreePageList(),
	}

	// If the file exists and has pages, try to load the free page list from page 0
	if nextPageID > 0 {
		if err := dm.loadFreePageList(); err != nil {
			// If we can't load the free page list, start with an empty one
			// This handles the case where we're opening an existing database without free page tracking
			dm.freePageList = NewFreePageList()
		}
	}

	return dm, nil
}

// ReadPage reads a page from disk
func (dm *DiskManager) ReadPage(pageID PageID) (*Page, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	return dm.readPageInternal(pageID)
}

// readPageInternal reads a page from disk without acquiring the lock
// Must be called with dm.mu held
func (dm *DiskManager) readPageInternal(pageID PageID) (*Page, error) {
	offset := int64(pageID) * PageSize
	data := make([]byte, PageSize)

	n, err := dm.dataFile.ReadAt(data, offset)
	if err != nil && err.Error() != "EOF" {
		return nil, fmt.Errorf("failed to read page %d: %w", pageID, err)
	}

	// If file is smaller, this is a new page
	if n < PageSize {
		return NewPage(pageID, PageTypeData), nil
	}

	page := NewPage(pageID, PageTypeData)
	if err := page.Deserialize(data); err != nil {
		return nil, fmt.Errorf("failed to deserialize page %d: %w", pageID, err)
	}

	dm.totalReads++
	return page, nil
}

// WritePage writes a page to disk
func (dm *DiskManager) WritePage(page *Page) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	return dm.writePageInternal(page)
}

// writePageInternal writes a page to disk without acquiring the lock
// Must be called with dm.mu held
func (dm *DiskManager) writePageInternal(page *Page) error {
	offset := int64(page.ID) * PageSize
	data := page.Serialize()

	if _, err := dm.dataFile.WriteAt(data, offset); err != nil {
		return fmt.Errorf("failed to write page %d: %w", page.ID, err)
	}

	dm.totalWrites++
	return nil
}

// AllocatePage allocates a new page
func (dm *DiskManager) AllocatePage() (PageID, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Try to reuse a free page if available
	if dm.freePageList.PageCount > 0 {
		pageID, ok, err := dm.popFreePage()
		if err != nil {
			return 0, fmt.Errorf("failed to pop free page: %w", err)
		}
		if ok {
			return pageID, nil
		}
	} else if dm.freePageList.HeadPageID != 0 {
		// Edge case: free list page exists but PageCount is 0
		// Try to reclaim the free list page itself
		pageID, ok, err := dm.popFreePage()
		if err != nil {
			return 0, fmt.Errorf("failed to pop free page: %w", err)
		}
		if ok {
			return pageID, nil
		}
	}

	// Allocate a new page
	pageID := dm.nextPageID
	dm.nextPageID++
	return pageID, nil
}

// DeallocatePage marks a page as free
func (dm *DiskManager) DeallocatePage(pageID PageID) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Validate the page ID
	if pageID >= dm.nextPageID {
		return fmt.Errorf("invalid page ID: %d (next page ID: %d)", pageID, dm.nextPageID)
	}

	// Add the page to the free page list
	if err := dm.pushFreePage(pageID); err != nil {
		return fmt.Errorf("failed to add page to free list: %w", err)
	}

	return nil
}

// Sync flushes all data to disk
func (dm *DiskManager) Sync() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	return dm.dataFile.Sync()
}

// Close closes the data file
func (dm *DiskManager) Close() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if err := dm.dataFile.Sync(); err != nil {
		return err
	}

	return dm.dataFile.Close()
}

// Stats returns disk manager statistics
func (dm *DiskManager) Stats() map[string]interface{} {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	return map[string]interface{}{
		"next_page_id": dm.nextPageID,
		"free_pages":   dm.freePageList.PageCount,
		"total_reads":  dm.totalReads,
		"total_writes": dm.totalWrites,
	}
}

// loadFreePageList loads the free page list from disk
// The free page list metadata is stored in a special location (we'll use a simple approach)
func (dm *DiskManager) loadFreePageList() error {
	// For now, we'll build the free page list by scanning for free list pages
	// In a production system, we'd store the head page ID in a metadata page
	// This is a simplified implementation that will be enhanced later
	dm.freePageList = NewFreePageList()
	return nil
}

// saveFreePageList persists the free page list to disk
func (dm *DiskManager) saveFreePageList() error {
	// This will be implemented when we add metadata persistence
	// For now, the free page list is rebuilt on startup
	return nil
}

// pushFreePage adds a page to the free page list
// Must be called with dm.mu held
func (dm *DiskManager) pushFreePage(pageID PageID) error {
	// If we don't have a head page for the free list, we need to allocate one
	if dm.freePageList.HeadPageID == 0 {
		// Allocate a new page for the free list
		freeListPageID := dm.nextPageID
		dm.nextPageID++

		// Initialize it as a free list page
		page := NewPage(freeListPageID, PageTypeFreeList)
		InitializeFreeListPage(page)

		// Add the page being freed to this free list page
		_, err := AddFreePageToList(page, pageID)
		if err != nil {
			return fmt.Errorf("failed to add page to new free list: %w", err)
		}

		// Write the page to disk
		if err := dm.writePageInternal(page); err != nil {
			return fmt.Errorf("failed to write free list page: %w", err)
		}

		dm.freePageList.HeadPageID = freeListPageID
		dm.freePageList.PageCount = 1
		return nil
	}

	// Read the head free list page
	headPage, err := dm.readPageInternal(dm.freePageList.HeadPageID)
	if err != nil {
		return fmt.Errorf("failed to read head free list page: %w", err)
	}

	// Try to add the page to the head free list page
	added, err := AddFreePageToList(headPage, pageID)
	if err != nil {
		return fmt.Errorf("failed to add page to free list: %w", err)
	}

	if added {
		// Successfully added to the head page
		if err := dm.writePageInternal(headPage); err != nil {
			return fmt.Errorf("failed to write free list page: %w", err)
		}
		dm.freePageList.PageCount++
		return nil
	}

	// Head page is full, need to create a new free list page
	// Use the page being freed as the new head
	newHeadPage := NewPage(pageID, PageTypeFreeList)
	InitializeFreeListPage(newHeadPage)

	// Link the new head to the old head
	header := &FreePageHeader{
		NextFreeListPage: dm.freePageList.HeadPageID,
		EntryCount:       0,
	}
	SerializeFreePageHeader(newHeadPage, header)

	// Write the new head page
	if err := dm.writePageInternal(newHeadPage); err != nil {
		return fmt.Errorf("failed to write new free list head page: %w", err)
	}

	// Update the free page list head
	dm.freePageList.HeadPageID = pageID

	return nil
}

// popFreePage removes a page from the free page list
// Returns (pageID, success, error)
// Must be called with dm.mu held
func (dm *DiskManager) popFreePage() (PageID, bool, error) {
	if dm.freePageList.HeadPageID == 0 || dm.freePageList.PageCount == 0 {
		return 0, false, nil // No free pages available
	}

	// Read the head free list page
	headPage, err := dm.readPageInternal(dm.freePageList.HeadPageID)
	if err != nil {
		return 0, false, fmt.Errorf("failed to read head free list page: %w", err)
	}

	// Try to remove a page from the head free list page
	pageID, removed, err := RemoveFreePageFromList(headPage)
	if err != nil {
		return 0, false, fmt.Errorf("failed to remove page from free list: %w", err)
	}

	if !removed {
		// Head page is empty, this shouldn't happen if PageCount is correct
		// But handle it gracefully - move to next free list page
		header, err := DeserializeFreePageHeader(headPage)
		if err != nil {
			return 0, false, fmt.Errorf("failed to deserialize free page header: %w", err)
		}

		oldHeadPageID := dm.freePageList.HeadPageID

		// Move to the next free list page
		dm.freePageList.HeadPageID = header.NextFreeListPage

		// Return the now-empty free list page as a regular free page
		return oldHeadPageID, true, nil
	}

	// Successfully removed from the head page
	if err := dm.writePageInternal(headPage); err != nil {
		return 0, false, fmt.Errorf("failed to write free list page: %w", err)
	}
	dm.freePageList.PageCount--

	// Check if the head page is now empty and should be reclaimed
	isEmpty, err := IsFreeListPageEmpty(headPage)
	if err == nil && isEmpty {
		// Head page is now empty, we can reclaim it on the next allocation
		// For now, just leave it - it will be reused when the next page is freed
	}

	return pageID, true, nil
}

// CompactionStats holds statistics about page compaction
type CompactionStats struct {
	PagesScanned       int64
	PagesCompacted     int64
	BytesReclaimed     int64
	Errors             int64
	LastCompactionTime int64 // Unix timestamp
}

// CompactPage compacts a specific page if it's a data page with a slotted page structure
func (dm *DiskManager) CompactPage(pageID PageID) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Read the page
	page, err := dm.readPageInternal(pageID)
	if err != nil {
		return fmt.Errorf("failed to read page %d: %w", pageID, err)
	}

	// Only compact data pages
	if page.Type != PageTypeData {
		return fmt.Errorf("cannot compact non-data page (type: %s)", page.Type)
	}

	// Load as slotted page
	sp, err := LoadSlottedPage(page)
	if err != nil {
		return fmt.Errorf("failed to load slotted page: %w", err)
	}

	// Check if compaction is needed
	if !sp.NeedsCompaction() {
		return nil // No compaction needed
	}

	// Perform compaction
	if err := sp.Compact(); err != nil {
		return fmt.Errorf("failed to compact page: %w", err)
	}

	// Write the compacted page back to disk
	if err := dm.writePageInternal(sp.GetPage()); err != nil {
		return fmt.Errorf("failed to write compacted page: %w", err)
	}

	return nil
}

// CompactPages compacts multiple pages in batch
func (dm *DiskManager) CompactPages(pageIDs []PageID) (*CompactionStats, error) {
	stats := &CompactionStats{}

	for _, pageID := range pageIDs {
		stats.PagesScanned++

		// Compact the page
		err := dm.CompactPage(pageID)
		if err != nil {
			stats.Errors++
			// Continue with other pages instead of failing completely
			continue
		}

		stats.PagesCompacted++
	}

	return stats, nil
}

// ScanForCompaction scans all data pages and compacts those that need it
// Returns statistics about the compaction operation
func (dm *DiskManager) ScanForCompaction() (*CompactionStats, error) {
	dm.mu.Lock()
	totalPages := dm.nextPageID
	dm.mu.Unlock()

	stats := &CompactionStats{
		LastCompactionTime: 0, // Will be set by caller if needed
	}

	// Scan all pages
	for pageID := PageID(0); pageID < totalPages; pageID++ {
		dm.mu.Lock()
		page, err := dm.readPageInternal(pageID)
		dm.mu.Unlock()

		stats.PagesScanned++

		if err != nil {
			stats.Errors++
			continue
		}

		// Skip non-data pages
		if page.Type != PageTypeData {
			continue
		}

		// Load as slotted page
		sp, err := LoadSlottedPage(page)
		if err != nil {
			stats.Errors++
			continue
		}

		// Check if compaction is needed
		if !sp.NeedsCompaction() {
			continue
		}

		// Record fragmented space before compaction
		fragmentedBefore := sp.FragmentedSpace()

		// Perform compaction
		if err := sp.Compact(); err != nil {
			stats.Errors++
			continue
		}

		// Write the compacted page back to disk
		dm.mu.Lock()
		err = dm.writePageInternal(sp.GetPage())
		dm.mu.Unlock()

		if err != nil {
			stats.Errors++
			continue
		}

		stats.PagesCompacted++
		stats.BytesReclaimed += int64(fragmentedBefore)
	}

	return stats, nil
}

// CompactPageRange compacts pages within a specific range
func (dm *DiskManager) CompactPageRange(startPageID, endPageID PageID) (*CompactionStats, error) {
	if startPageID > endPageID {
		return nil, fmt.Errorf("invalid page range: start %d > end %d", startPageID, endPageID)
	}

	dm.mu.Lock()
	totalPages := dm.nextPageID
	dm.mu.Unlock()

	if endPageID >= totalPages {
		endPageID = totalPages - 1
	}

	stats := &CompactionStats{}

	for pageID := startPageID; pageID <= endPageID; pageID++ {
		stats.PagesScanned++

		dm.mu.Lock()
		page, err := dm.readPageInternal(pageID)
		dm.mu.Unlock()

		if err != nil {
			stats.Errors++
			continue
		}

		// Skip non-data pages
		if page.Type != PageTypeData {
			continue
		}

		// Load as slotted page
		sp, err := LoadSlottedPage(page)
		if err != nil {
			stats.Errors++
			continue
		}

		// Check if compaction is needed
		if !sp.NeedsCompaction() {
			continue
		}

		// Record fragmented space before compaction
		fragmentedBefore := sp.FragmentedSpace()

		// Perform compaction
		if err := sp.Compact(); err != nil {
			stats.Errors++
			continue
		}

		// Write the compacted page back to disk
		dm.mu.Lock()
		err = dm.writePageInternal(sp.GetPage())
		dm.mu.Unlock()

		if err != nil {
			stats.Errors++
			continue
		}

		stats.PagesCompacted++
		stats.BytesReclaimed += int64(fragmentedBefore)
	}

	return stats, nil
}
