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
	freePages    []PageID
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

	return &DiskManager{
		dataFile:   file,
		nextPageID: nextPageID,
		freePages:  make([]PageID, 0),
	}, nil
}

// ReadPage reads a page from disk
func (dm *DiskManager) ReadPage(pageID PageID) (*Page, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

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

	// Reuse a free page if available
	if len(dm.freePages) > 0 {
		pageID := dm.freePages[len(dm.freePages)-1]
		dm.freePages = dm.freePages[:len(dm.freePages)-1]
		return pageID, nil
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

	dm.freePages = append(dm.freePages, pageID)
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
		"free_pages":   len(dm.freePages),
		"total_reads":  dm.totalReads,
		"total_writes": dm.totalWrites,
	}
}
