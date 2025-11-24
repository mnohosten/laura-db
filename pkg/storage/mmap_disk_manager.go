package storage

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

// MmapDiskManager handles physical disk I/O operations using memory-mapped files
// This provides better performance for read-heavy workloads by mapping the file
// directly into the process address space, reducing system calls.
type MmapDiskManager struct {
	dataFile     *os.File
	mmapData     []byte
	mmapSize     int64
	nextPageID   PageID
	freePages    []PageID
	mu           sync.RWMutex
	totalReads   int64
	totalWrites  int64
	useMmap      bool
}

// MmapConfig holds configuration for memory-mapped disk manager
type MmapConfig struct {
	InitialSize int64 // Initial mmap size in bytes (default: 256MB)
	GrowthSize  int64 // Size to grow by when expanding (default: 64MB)
}

// DefaultMmapConfig returns default mmap configuration
func DefaultMmapConfig() *MmapConfig {
	return &MmapConfig{
		InitialSize: 256 * 1024 * 1024, // 256MB
		GrowthSize:  64 * 1024 * 1024,  // 64MB
	}
}

// NewMmapDiskManager creates a new memory-mapped disk manager
func NewMmapDiskManager(path string, config *MmapConfig) (*MmapDiskManager, error) {
	if config == nil {
		config = DefaultMmapConfig()
	}

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

	currentSize := fileInfo.Size()
	nextPageID := PageID(currentSize / PageSize)

	dm := &MmapDiskManager{
		dataFile:   file,
		nextPageID: nextPageID,
		freePages:  make([]PageID, 0),
		useMmap:    true,
	}

	// Initialize mmap
	mmapSize := config.InitialSize
	if currentSize > mmapSize {
		mmapSize = currentSize
	}

	if err := dm.expandMmap(mmapSize); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to initialize mmap: %w", err)
	}

	return dm, nil
}

// expandMmap expands or initializes the memory-mapped region
func (dm *MmapDiskManager) expandMmap(newSize int64) error {
	// Unmap existing region if any
	if dm.mmapData != nil {
		if err := syscall.Munmap(dm.mmapData); err != nil {
			return fmt.Errorf("failed to unmap existing region: %w", err)
		}
		dm.mmapData = nil
	}

	// Ensure file is large enough
	if err := dm.dataFile.Truncate(newSize); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}

	// Map the file into memory
	data, err := syscall.Mmap(
		int(dm.dataFile.Fd()),
		0,
		int(newSize),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
	if err != nil {
		return fmt.Errorf("failed to mmap file: %w", err)
	}

	dm.mmapData = data
	dm.mmapSize = newSize
	return nil
}

// ReadPage reads a page from the memory-mapped region
func (dm *MmapDiskManager) ReadPage(pageID PageID) (*Page, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if !dm.useMmap {
		return nil, fmt.Errorf("mmap is disabled")
	}

	offset := int64(pageID) * PageSize

	// Check if page is beyond current mmap region
	if offset+PageSize > dm.mmapSize {
		return NewPage(pageID, PageTypeData), nil
	}

	// Read directly from memory-mapped region
	data := dm.mmapData[offset : offset+PageSize]

	page := NewPage(pageID, PageTypeData)
	if err := page.Deserialize(data); err != nil {
		return nil, fmt.Errorf("failed to deserialize page %d: %w", pageID, err)
	}

	dm.totalReads++
	return page, nil
}

// WritePage writes a page to the memory-mapped region
func (dm *MmapDiskManager) WritePage(page *Page) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if !dm.useMmap {
		return fmt.Errorf("mmap is disabled")
	}

	offset := int64(page.ID) * PageSize

	// Expand mmap if needed
	if offset+PageSize > dm.mmapSize {
		newSize := dm.mmapSize + DefaultMmapConfig().GrowthSize
		if offset+PageSize > newSize {
			newSize = offset + PageSize + DefaultMmapConfig().GrowthSize
		}
		if err := dm.expandMmap(newSize); err != nil {
			return fmt.Errorf("failed to expand mmap: %w", err)
		}
	}

	// Write directly to memory-mapped region
	data := page.Serialize()
	copy(dm.mmapData[offset:offset+PageSize], data)

	dm.totalWrites++
	return nil
}

// AllocatePage allocates a new page
func (dm *MmapDiskManager) AllocatePage() (PageID, error) {
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

	// Ensure mmap is large enough for the new page
	offset := int64(pageID) * PageSize
	if offset+PageSize > dm.mmapSize {
		newSize := dm.mmapSize + DefaultMmapConfig().GrowthSize
		if err := dm.expandMmap(newSize); err != nil {
			return 0, fmt.Errorf("failed to expand mmap for new page: %w", err)
		}
	}

	return pageID, nil
}

// DeallocatePage marks a page as free
func (dm *MmapDiskManager) DeallocatePage(pageID PageID) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.freePages = append(dm.freePages, pageID)
	return nil
}

// Sync flushes all changes to disk
func (dm *MmapDiskManager) Sync() error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if !dm.useMmap || dm.mmapData == nil {
		return nil
	}

	// Use msync to flush memory-mapped region to disk
	_, _, errno := syscall.Syscall(syscall.SYS_MSYNC, uintptr(unsafe.Pointer(&dm.mmapData[0])), uintptr(len(dm.mmapData)), uintptr(syscall.MS_SYNC))
	if errno != 0 {
		return fmt.Errorf("failed to msync: %v", errno)
	}

	return nil
}

// Close closes the memory-mapped file
func (dm *MmapDiskManager) Close() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Unmap the memory region
	if dm.mmapData != nil {
		// Sync before unmapping
		_, _, errno := syscall.Syscall(syscall.SYS_MSYNC, uintptr(unsafe.Pointer(&dm.mmapData[0])), uintptr(len(dm.mmapData)), uintptr(syscall.MS_SYNC))
		if errno != 0 {
			return fmt.Errorf("failed to sync before close: %v", errno)
		}
		if err := syscall.Munmap(dm.mmapData); err != nil {
			return fmt.Errorf("failed to unmap: %w", err)
		}
		dm.mmapData = nil
	}

	// Close the file
	if err := dm.dataFile.Sync(); err != nil {
		return err
	}

	dm.useMmap = false
	return dm.dataFile.Close()
}

// Stats returns disk manager statistics
func (dm *MmapDiskManager) Stats() map[string]interface{} {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return map[string]interface{}{
		"next_page_id": dm.nextPageID,
		"free_pages":   len(dm.freePages),
		"total_reads":  dm.totalReads,
		"total_writes": dm.totalWrites,
		"mmap_size":    dm.mmapSize,
		"use_mmap":     dm.useMmap,
	}
}

// MadviseRandom hints that page access will be random
func (dm *MmapDiskManager) MadviseRandom() error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if dm.mmapData == nil {
		return fmt.Errorf("mmap not initialized")
	}

	_, _, errno := syscall.Syscall(syscall.SYS_MADVISE, uintptr(unsafe.Pointer(&dm.mmapData[0])), uintptr(len(dm.mmapData)), uintptr(syscall.MADV_RANDOM))
	if errno != 0 {
		return fmt.Errorf("madvise random failed: %v", errno)
	}
	return nil
}

// MadviseSequential hints that page access will be sequential
func (dm *MmapDiskManager) MadviseSequential() error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if dm.mmapData == nil {
		return fmt.Errorf("mmap not initialized")
	}

	_, _, errno := syscall.Syscall(syscall.SYS_MADVISE, uintptr(unsafe.Pointer(&dm.mmapData[0])), uintptr(len(dm.mmapData)), uintptr(syscall.MADV_SEQUENTIAL))
	if errno != 0 {
		return fmt.Errorf("madvise sequential failed: %v", errno)
	}
	return nil
}

// MadviseWillNeed hints that pages will be needed soon (prefetch)
func (dm *MmapDiskManager) MadviseWillNeed(startPage, endPage PageID) error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if dm.mmapData == nil {
		return fmt.Errorf("mmap not initialized")
	}

	startOffset := int64(startPage) * PageSize
	endOffset := int64(endPage) * PageSize

	if startOffset >= dm.mmapSize || endOffset > dm.mmapSize {
		return fmt.Errorf("page range exceeds mmap size")
	}

	length := int(endOffset - startOffset)
	_, _, errno := syscall.Syscall(syscall.SYS_MADVISE, uintptr(unsafe.Pointer(&dm.mmapData[startOffset])), uintptr(length), uintptr(syscall.MADV_WILLNEED))
	if errno != 0 {
		return fmt.Errorf("madvise willneed failed: %v", errno)
	}
	return nil
}
