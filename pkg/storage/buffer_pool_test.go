package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBufferPoolEviction(t *testing.T) {
	dir := "./test_buffer_eviction"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	// Create a small buffer pool
	diskPath := filepath.Join(dir, "test.db")
	diskMgr, err := NewDiskManager(diskPath)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bp := NewBufferPool(3, diskMgr) // Only 3 pages

	// Allocate 3 pages (fills the buffer)
	page1, _ := bp.NewPage()
	page2, _ := bp.NewPage()
	page3, _ := bp.NewPage()

	// Unpin all pages
	bp.UnpinPage(page1.ID, false)
	bp.UnpinPage(page2.ID, false)
	bp.UnpinPage(page3.ID, false)

	// Allocate another page - should trigger eviction
	page4, err := bp.NewPage()
	if err != nil {
		t.Fatalf("Failed to allocate page after buffer full: %v", err)
	}
	if page4 == nil {
		t.Fatal("Expected non-nil page")
	}

	// Check that eviction occurred
	stats := bp.Stats()
	evictions := stats["evictions"].(int)
	if evictions == 0 {
		t.Error("Expected at least one eviction")
	}
}

func TestBufferPoolEvictionWithDirtyPage(t *testing.T) {
	dir := "./test_buffer_eviction_dirty"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	diskPath := filepath.Join(dir, "test.db")
	diskMgr, err := NewDiskManager(diskPath)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bp := NewBufferPool(2, diskMgr)

	// Allocate two pages
	page1, _ := bp.NewPage()
	page2, _ := bp.NewPage()

	// Mark page1 as dirty and unpin
	copy(page1.Data, []byte("dirty data"))
	page1.MarkDirty()
	bp.UnpinPage(page1.ID, true)
	bp.UnpinPage(page2.ID, false)

	// Allocate third page - should evict dirty page1 and flush it
	page3, err := bp.NewPage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}
	if page3 == nil {
		t.Fatal("Expected non-nil page")
	}

	// Verify dirty page was flushed by reading it back
	fetchedPage, err := bp.FetchPage(page1.ID)
	if err != nil {
		t.Fatalf("Failed to fetch evicted page: %v", err)
	}
	fetchedData := fetchedPage.Data[:len("dirty data")]
	if string(fetchedData) != "dirty data" {
		t.Errorf("Expected 'dirty data', got '%s'", string(fetchedData))
	}
}

func TestBufferPoolFetchNonExistent(t *testing.T) {
	dir := "./test_buffer_fetch_missing"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	diskPath := filepath.Join(dir, "test.db")
	diskMgr, err := NewDiskManager(diskPath)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bp := NewBufferPool(10, diskMgr)

	// Fetch a page that doesn't exist - should create new page
	page, err := bp.FetchPage(100)
	if err != nil {
		t.Fatalf("Failed to fetch non-existent page: %v", err)
	}
	if page == nil {
		t.Fatal("Expected non-nil page")
	}
	if page.ID != 100 {
		t.Errorf("Expected page ID 100, got %d", page.ID)
	}
}

func TestBufferPoolFlushNonExistentPage(t *testing.T) {
	dir := "./test_buffer_flush_missing"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	diskPath := filepath.Join(dir, "test.db")
	diskMgr, err := NewDiskManager(diskPath)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bp := NewBufferPool(10, diskMgr)

	// Try to flush a page that's not in buffer pool
	err = bp.FlushPage(999)
	if err == nil {
		t.Error("Expected error when flushing non-existent page")
	}
}

func TestBufferPoolFlushCleanPage(t *testing.T) {
	dir := "./test_buffer_flush_clean"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	diskPath := filepath.Join(dir, "test.db")
	diskMgr, err := NewDiskManager(diskPath)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bp := NewBufferPool(10, diskMgr)

	// Create a clean page (not dirty)
	page, _ := bp.NewPage()
	bp.UnpinPage(page.ID, false) // Unpin without marking dirty

	// Flush clean page (should not error)
	err = bp.FlushPage(page.ID)
	if err != nil {
		t.Fatalf("Failed to flush clean page: %v", err)
	}
}

func TestBufferPoolDeletePageNotInPool(t *testing.T) {
	dir := "./test_buffer_delete_missing"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	diskPath := filepath.Join(dir, "test.db")
	diskMgr, err := NewDiskManager(diskPath)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bp := NewBufferPool(10, diskMgr)

	// Delete a page that's not in buffer pool and doesn't exist (should error with new validation)
	err = bp.DeletePage(999)
	if err == nil {
		t.Fatal("Expected error when deleting non-existent page, got nil")
	}
}

func TestBufferPoolNewPageWhenFull(t *testing.T) {
	dir := "./test_buffer_new_full"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	diskPath := filepath.Join(dir, "test.db")
	diskMgr, err := NewDiskManager(diskPath)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bp := NewBufferPool(2, diskMgr)

	// Fill buffer with pinned pages
	page1, _ := bp.NewPage()
	page2, _ := bp.NewPage()

	// Don't unpin them
	if page1.PinCount != 1 || page2.PinCount != 1 {
		t.Error("Expected pages to be pinned")
	}

	// Unpin one
	bp.UnpinPage(page1.ID, false)

	// Try to allocate another - should succeed by evicting page1
	page3, err := bp.NewPage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}
	if page3 == nil {
		t.Fatal("Expected non-nil page")
	}
}

func TestBufferPoolUnpinNonExistentPage(t *testing.T) {
	dir := "./test_buffer_unpin_missing"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	diskPath := filepath.Join(dir, "test.db")
	diskMgr, err := NewDiskManager(diskPath)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bp := NewBufferPool(10, diskMgr)

	// Try to unpin a page that's not in buffer pool (should not panic)
	err = bp.UnpinPage(999, false)
	if err == nil {
		t.Error("Expected error when unpinning non-existent page")
	}
}

func TestBufferPoolMultiplePinUnpin(t *testing.T) {
	dir := "./test_buffer_multi_pin"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	diskPath := filepath.Join(dir, "test.db")
	diskMgr, err := NewDiskManager(diskPath)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bp := NewBufferPool(10, diskMgr)

	// Create a page
	page, _ := bp.NewPage()
	pageID := page.ID

	// Pin it multiple times
	bp.FetchPage(pageID) // Pin count = 2
	bp.FetchPage(pageID) // Pin count = 3

	// Unpin once
	bp.UnpinPage(pageID, false) // Pin count = 2

	// Page should still be pinned
	frame := bp.pages[pageID]
	if frame.page.PinCount != 2 {
		t.Errorf("Expected pin count 2, got %d", frame.page.PinCount)
	}

	// Unpin again
	bp.UnpinPage(pageID, false) // Pin count = 1
	bp.UnpinPage(pageID, false) // Pin count = 0

	// Now page should be unpinned
	if frame.page.IsPinned() {
		t.Error("Expected page to be unpinned")
	}
}

func TestBufferPoolStatsHitRate(t *testing.T) {
	dir := "./test_buffer_hit_rate"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	diskPath := filepath.Join(dir, "test.db")
	diskMgr, err := NewDiskManager(diskPath)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bp := NewBufferPool(10, diskMgr)

	// Create a page
	page, _ := bp.NewPage()
	pageID := page.ID
	bp.UnpinPage(pageID, false)

	// Fetch it again (hit)
	bp.FetchPage(pageID)
	bp.UnpinPage(pageID, false)

	// Check stats
	stats := bp.Stats()
	hits := stats["hits"].(int)
	if hits == 0 {
		t.Error("Expected at least one cache hit")
	}

	hitRate := stats["hit_rate"].(float64)
	if hitRate == 0.0 {
		t.Error("Expected non-zero hit rate")
	}
}

func TestBufferPoolLRUOrdering(t *testing.T) {
	dir := "./test_buffer_lru"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	diskPath := filepath.Join(dir, "test.db")
	diskMgr, err := NewDiskManager(diskPath)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bp := NewBufferPool(3, diskMgr)

	// Create 3 pages
	page1, _ := bp.NewPage()
	page2, _ := bp.NewPage()
	page3, _ := bp.NewPage()

	// Unpin all
	bp.UnpinPage(page1.ID, false)
	bp.UnpinPage(page2.ID, false)
	bp.UnpinPage(page3.ID, false)

	// Access page1 again (makes it most recently used)
	bp.FetchPage(page1.ID)
	bp.UnpinPage(page1.ID, false)

	// Now allocate a new page - should evict page2 (least recently used)
	page4, _ := bp.NewPage()
	bp.UnpinPage(page4.ID, false)

	// Verify page2 was evicted (not in buffer)
	_, exists := bp.pages[page2.ID]
	if exists {
		t.Error("Expected page2 to be evicted")
	}

	// Verify page1 and page3 are still in buffer
	_, exists1 := bp.pages[page1.ID]
	_, exists3 := bp.pages[page3.ID]
	if !exists1 || !exists3 {
		t.Error("Expected page1 and page3 to still be in buffer")
	}
}
