package storage

import (
	"container/list"
	"fmt"
	"sync"
)

// BufferPool manages a cache of pages in memory
// Uses LRU (Least Recently Used) eviction policy
type BufferPool struct {
	capacity  int
	pages     map[PageID]*bufferFrame
	lruList   *list.List
	mu        sync.RWMutex
	diskMgr   *DiskManager
	evictions int
	hits      int
	misses    int
}

// bufferFrame represents a page in the buffer pool
type bufferFrame struct {
	page    *Page
	lruNode *list.Element
}

// NewBufferPool creates a new buffer pool
func NewBufferPool(capacity int, diskMgr *DiskManager) *BufferPool {
	return &BufferPool{
		capacity: capacity,
		pages:    make(map[PageID]*bufferFrame, capacity),
		lruList:  list.New(),
		diskMgr:  diskMgr,
	}
}

// FetchPage retrieves a page from the buffer pool or disk
func (bp *BufferPool) FetchPage(pageID PageID) (*Page, error) {
	// Fast path: check if page is already in buffer pool (read lock)
	bp.mu.RLock()
	if _, exists := bp.pages[pageID]; exists {
		// Upgrade to write lock for LRU update
		bp.mu.RUnlock()
		bp.mu.Lock()

		// Double-check page still exists after lock upgrade
		if frame, exists := bp.pages[pageID]; exists {
			// Move to front of LRU list (most recently used)
			bp.lruList.MoveToFront(frame.lruNode)
			frame.page.Pin()
			bp.hits++
			bp.mu.Unlock()
			return frame.page, nil
		}
		bp.mu.Unlock()

		// Page was evicted between locks, fall through to slow path
		bp.mu.Lock()
	} else {
		// Upgrade to write lock for disk read and insertion
		bp.mu.RUnlock()
		bp.mu.Lock()
	}
	defer bp.mu.Unlock()

	// Slow path: page not in pool - need to fetch from disk
	// Double-check after acquiring write lock
	if frame, exists := bp.pages[pageID]; exists {
		bp.lruList.MoveToFront(frame.lruNode)
		frame.page.Pin()
		bp.hits++
		return frame.page, nil
	}

	bp.misses++

	// Page not in pool - need to fetch from disk
	// First, check if we need to evict a page
	if len(bp.pages) >= bp.capacity {
		if err := bp.evictPage(); err != nil {
			return nil, fmt.Errorf("failed to evict page: %w", err)
		}
	}

	// Read page from disk
	page, err := bp.diskMgr.ReadPage(pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to read page from disk: %w", err)
	}

	// Add to buffer pool
	frame := &bufferFrame{
		page:    page,
		lruNode: bp.lruList.PushFront(pageID),
	}
	bp.pages[pageID] = frame
	page.Pin()

	return page, nil
}

// NewPage creates a new page and adds it to the buffer pool
func (bp *BufferPool) NewPage() (*Page, error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Check if we need to evict
	if len(bp.pages) >= bp.capacity {
		if err := bp.evictPage(); err != nil {
			return nil, fmt.Errorf("failed to evict page: %w", err)
		}
	}

	// Allocate new page on disk
	pageID, err := bp.diskMgr.AllocatePage()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate page: %w", err)
	}

	page := NewPage(pageID, PageTypeData)
	page.MarkDirty()

	// Add to buffer pool
	frame := &bufferFrame{
		page:    page,
		lruNode: bp.lruList.PushFront(pageID),
	}
	bp.pages[pageID] = frame
	page.Pin()

	return page, nil
}

// UnpinPage decrements the pin count of a page
func (bp *BufferPool) UnpinPage(pageID PageID, isDirty bool) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	frame, exists := bp.pages[pageID]
	if !exists {
		return fmt.Errorf("page %d not in buffer pool", pageID)
	}

	frame.page.Unpin()
	if isDirty {
		frame.page.MarkDirty()
	}

	return nil
}

// FlushPage writes a page to disk
func (bp *BufferPool) FlushPage(pageID PageID) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	frame, exists := bp.pages[pageID]
	if !exists {
		return fmt.Errorf("page %d not in buffer pool", pageID)
	}

	if frame.page.IsDirty {
		if err := bp.diskMgr.WritePage(frame.page); err != nil {
			return fmt.Errorf("failed to write page to disk: %w", err)
		}
		frame.page.IsDirty = false
	}

	return nil
}

// FlushAllPages writes all dirty pages to disk
func (bp *BufferPool) FlushAllPages() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	for pageID := range bp.pages {
		frame := bp.pages[pageID]
		if frame.page.IsDirty {
			if err := bp.diskMgr.WritePage(frame.page); err != nil {
				return fmt.Errorf("failed to write page %d to disk: %w", pageID, err)
			}
			frame.page.IsDirty = false
		}
	}

	return nil
}

// evictPage removes the least recently used unpinned page
func (bp *BufferPool) evictPage() error {
	// Find an unpinned page from the back of the LRU list
	for elem := bp.lruList.Back(); elem != nil; elem = elem.Prev() {
		pageID := elem.Value.(PageID)
		frame := bp.pages[pageID]

		if !frame.page.IsPinned() {
			// Flush if dirty
			if frame.page.IsDirty {
				if err := bp.diskMgr.WritePage(frame.page); err != nil {
					return fmt.Errorf("failed to flush page during eviction: %w", err)
				}
			}

			// Remove from buffer pool
			bp.lruList.Remove(elem)
			delete(bp.pages, pageID)
			bp.evictions++
			return nil
		}
	}

	return fmt.Errorf("no unpinned pages available for eviction")
}

// DeletePage removes a page from the buffer pool and disk
func (bp *BufferPool) DeletePage(pageID PageID) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Remove from buffer pool if present
	if frame, exists := bp.pages[pageID]; exists {
		if frame.page.IsPinned() {
			return fmt.Errorf("cannot delete pinned page %d", pageID)
		}
		bp.lruList.Remove(frame.lruNode)
		delete(bp.pages, pageID)
	}

	// Mark as free on disk
	return bp.diskMgr.DeallocatePage(pageID)
}

// Stats returns buffer pool statistics
func (bp *BufferPool) Stats() map[string]interface{} {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	total := bp.hits + bp.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(bp.hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"capacity":   bp.capacity,
		"size":       len(bp.pages),
		"hits":       bp.hits,
		"misses":     bp.misses,
		"evictions":  bp.evictions,
		"hit_rate":   hitRate,
	}
}
