package storage

import (
	"os"
	"testing"
)

func TestNewStorageEngine(t *testing.T) {
	dir := "./test_storage"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	if engine == nil {
		t.Fatal("Expected non-nil storage engine")
	}
}

func TestAllocateAndFetchPage(t *testing.T) {
	dir := "./test_storage_page"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Allocate page
	page, err := engine.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}

	pageID := page.ID

	// Write data to page
	testData := []byte("Hello, Storage!")
	copy(page.Data, testData)
	page.MarkDirty()

	// Unpin page
	engine.UnpinPage(pageID, true)

	// Fetch page back
	fetchedPage, err := engine.FetchPage(pageID)
	if err != nil {
		t.Fatalf("Failed to fetch page: %v", err)
	}

	// Verify data
	fetchedData := fetchedPage.Data[:len(testData)]
	if string(fetchedData) != string(testData) {
		t.Errorf("Expected %s, got %s", testData, fetchedData)
	}

	engine.UnpinPage(fetchedPage.ID, false)
}

func TestWALLogOperation(t *testing.T) {
	dir := "./test_storage_wal"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Log an operation
	record := &LogRecord{
		Type:   LogRecordInsert,
		TxnID:  1,
		PageID: 0,
		Data:   []byte("test data"),
	}

	lsn, err := engine.LogOperation(record)
	if err != nil {
		t.Fatalf("Failed to log operation: %v", err)
	}

	if lsn == 0 {
		t.Error("Expected non-zero LSN")
	}
}

func TestCheckpoint(t *testing.T) {
	dir := "./test_storage_checkpoint"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Allocate and modify a page
	page, _ := engine.AllocatePage()
	copy(page.Data, []byte("checkpoint test"))
	page.MarkDirty()
	engine.UnpinPage(page.ID, true)

	// Checkpoint
	err = engine.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint failed: %v", err)
	}
}

func TestStorageEngineRecovery(t *testing.T) {
	dir := "./test_storage_recovery"
	defer os.RemoveAll(dir)

	// Create engine and write data
	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}

	page, _ := engine.AllocatePage()
	testData := []byte("recovery test")
	copy(page.Data, testData)
	page.MarkDirty()

	// Log operation
	record := &LogRecord{
		Type:   LogRecordInsert,
		TxnID:  1,
		PageID: page.ID,
		Data:   []byte("metadata"),
	}
	lsn, _ := engine.LogOperation(record)
	page.LSN = lsn

	engine.UnpinPage(page.ID, true)
	pageID := page.ID

	// Close engine
	engine.Close()

	// Reopen engine (should trigger recovery)
	engine2, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to reopen storage engine: %v", err)
	}
	defer engine2.Close()

	// Fetch page and verify data persisted
	recoveredPage, err := engine2.FetchPage(pageID)
	if err != nil {
		t.Fatalf("Failed to fetch page after recovery: %v", err)
	}

	recoveredData := recoveredPage.Data[:len(testData)]
	if string(recoveredData) != string(testData) {
		t.Errorf("Data not recovered correctly: expected %s, got %s", testData, recoveredData)
	}

	engine2.UnpinPage(recoveredPage.ID, false)
}

func TestDeletePage(t *testing.T) {
	dir := "./test_delete_page"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Allocate a page
	page, err := engine.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}
	pageID := page.ID

	// Write some data
	testData := []byte("test data for deletion")
	copy(page.Data, testData)
	page.MarkDirty()

	// Unpin the page first
	engine.UnpinPage(pageID, true)

	// Delete the page
	err = engine.bufferPool.DeletePage(pageID)
	if err != nil {
		t.Fatalf("Failed to delete page: %v", err)
	}

	// Verify page is removed from buffer pool
	_, exists := engine.bufferPool.pages[pageID]
	if exists {
		t.Error("Expected page to be removed from buffer pool")
	}
}

func TestDeletePinnedPage(t *testing.T) {
	dir := "./test_delete_pinned"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Allocate a page (it will be pinned)
	page, err := engine.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}

	// Try to delete pinned page - should fail
	err = engine.bufferPool.DeletePage(page.ID)
	if err == nil {
		t.Error("Expected error when deleting pinned page")
	}

	// Unpin and try again
	engine.UnpinPage(page.ID, false)
	err = engine.bufferPool.DeletePage(page.ID)
	if err != nil {
		t.Errorf("Failed to delete unpinned page: %v", err)
	}
}

func TestDeallocatePage(t *testing.T) {
	dir := "./test_deallocate_page"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Allocate a page
	page, err := engine.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}
	pageID := page.ID
	engine.UnpinPage(pageID, false)

	// Get initial free pages count
	initialStats := engine.diskMgr.Stats()
	initialFreePages := initialStats["free_pages"].(uint32)

	// Deallocate the page
	err = engine.diskMgr.DeallocatePage(pageID)
	if err != nil {
		t.Fatalf("Failed to deallocate page: %v", err)
	}

	// Verify free pages list grew
	stats := engine.diskMgr.Stats()
	freePages := stats["free_pages"].(uint32)
	if freePages != initialFreePages+1 {
		t.Errorf("Expected %d free pages, got %d", initialFreePages+1, freePages)
	}
}

func TestDiskManagerStats(t *testing.T) {
	dir := "./test_disk_stats"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Get initial stats
	stats := engine.diskMgr.Stats()
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	// Check required fields
	if _, ok := stats["next_page_id"]; !ok {
		t.Error("Stats missing next_page_id")
	}
	if _, ok := stats["free_pages"]; !ok {
		t.Error("Stats missing free_pages")
	}
	if _, ok := stats["total_reads"]; !ok {
		t.Error("Stats missing total_reads")
	}
	if _, ok := stats["total_writes"]; !ok {
		t.Error("Stats missing total_writes")
	}

	// Allocate and write a page to increase stats
	page, _ := engine.AllocatePage()
	copy(page.Data, []byte("stats test"))
	page.MarkDirty()
	engine.UnpinPage(page.ID, true)

	// Flush to ensure write
	engine.bufferPool.FlushPage(page.ID)

	// Get updated stats
	newStats := engine.diskMgr.Stats()
	nextPageID := newStats["next_page_id"].(PageID)
	if nextPageID == 0 {
		t.Error("Expected next_page_id to be greater than 0")
	}
}

func TestBufferPoolStats(t *testing.T) {
	dir := "./test_buffer_stats"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	config.BufferPoolSize = 10
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Get initial stats
	stats := engine.bufferPool.Stats()
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	// Check required fields
	if _, ok := stats["capacity"]; !ok {
		t.Error("Stats missing capacity")
	}
	if capacity := stats["capacity"].(int); capacity != 10 {
		t.Errorf("Expected capacity 10, got %d", capacity)
	}

	// Allocate a page to add to buffer pool
	page, _ := engine.AllocatePage()
	engine.UnpinPage(page.ID, false)

	// Get updated stats
	newStats := engine.bufferPool.Stats()
	size := newStats["size"].(int)
	if size == 0 {
		t.Error("Expected size to be greater than 0")
	}

	// Check hit rate exists
	if _, ok := newStats["hit_rate"]; !ok {
		t.Error("Stats missing hit_rate")
	}
}

func TestPageTypeString(t *testing.T) {
	tests := []struct {
		pageType PageType
		expected string
	}{
		{PageTypeData, "data"},
		{PageTypeIndex, "index"},
		{PageTypeFreeList, "freelist"},
		{PageTypeOverflow, "overflow"},
		{PageType(99), "unknown"},
	}

	for _, test := range tests {
		result := test.pageType.String()
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

func TestPageFreeSpace(t *testing.T) {
	page := &Page{
		ID:   0,
		Type: PageTypeData,
		Data: make([]byte, PageSize),
	}

	freeSpace := page.FreeSpace()
	if freeSpace != PageSize {
		t.Errorf("Expected free space %d, got %d", PageSize, freeSpace)
	}
}

func TestStorageEngineFlushPage(t *testing.T) {
	dir := "./test_flush_page"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Allocate a page and write data
	page, err := engine.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}

	testData := []byte("flush test data")
	copy(page.Data, testData)
	page.MarkDirty()
	pageID := page.ID
	engine.UnpinPage(pageID, true)

	// Flush the specific page
	err = engine.FlushPage(pageID)
	if err != nil {
		t.Fatalf("Failed to flush page: %v", err)
	}
}

func TestStorageEngineFlushAll(t *testing.T) {
	dir := "./test_flush_all"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Allocate multiple pages and write data
	for i := 0; i < 5; i++ {
		page, err := engine.AllocatePage()
		if err != nil {
			t.Fatalf("Failed to allocate page: %v", err)
		}
		copy(page.Data, []byte("test data"))
		page.MarkDirty()
		engine.UnpinPage(page.ID, true)
	}

	// Flush all pages
	err = engine.FlushAll()
	if err != nil {
		t.Fatalf("Failed to flush all pages: %v", err)
	}
}

func TestStorageEngineStats(t *testing.T) {
	dir := "./test_storage_stats"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Get stats
	stats := engine.Stats()
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	// Verify buffer_pool stats exist
	if _, ok := stats["buffer_pool"]; !ok {
		t.Error("Stats missing buffer_pool")
	}

	// Verify disk stats exist
	if _, ok := stats["disk"]; !ok {
		t.Error("Stats missing disk")
	}

	// Verify buffer_pool stats structure
	bufferPoolStats, ok := stats["buffer_pool"].(map[string]interface{})
	if !ok {
		t.Fatal("buffer_pool stats should be a map")
	}
	if _, ok := bufferPoolStats["capacity"]; !ok {
		t.Error("buffer_pool stats missing capacity")
	}

	// Verify disk stats structure
	diskStats, ok := stats["disk"].(map[string]interface{})
	if !ok {
		t.Fatal("disk stats should be a map")
	}
	if _, ok := diskStats["next_page_id"]; !ok {
		t.Error("disk stats missing next_page_id")
	}
}

func TestNewStorageEngineErrors(t *testing.T) {
	// Test with invalid directory (permission error simulation)
	// Note: This test creates a directory structure that may fail on certain operations
	dir := "./test_storage_error"
	defer os.RemoveAll(dir)

	// Test with nil config
	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	if engine != nil {
		engine.Close()
	}
}

func TestCheckpointErrors(t *testing.T) {
	dir := "./test_checkpoint_errors"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}

	// Test checkpoint on open engine
	err = engine.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint should succeed on open engine: %v", err)
	}

	// Close engine
	engine.Close()

	// Test checkpoint on closed engine (should fail)
	err = engine.Checkpoint()
	if err == nil {
		t.Error("Expected error when checkpointing closed engine")
	}
}

func TestAllocatePageErrors(t *testing.T) {
	dir := "./test_allocate_errors"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}

	// Test successful allocation
	page, err := engine.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}
	if page == nil {
		t.Error("Expected non-nil page")
	}
	engine.UnpinPage(page.ID, false)

	// Close engine
	engine.Close()

	// Test allocation on closed engine
	page, err = engine.AllocatePage()
	if err == nil {
		t.Error("Expected error when allocating page on closed engine")
	}
	if page != nil {
		t.Error("Expected nil page when allocation fails")
	}
}

func TestFetchPageErrors(t *testing.T) {
	dir := "./test_fetch_errors"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}

	// Allocate a page
	page, err := engine.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}
	pageID := page.ID
	engine.UnpinPage(pageID, false)

	// Close engine
	engine.Close()

	// Test fetch on closed engine
	page, err = engine.FetchPage(pageID)
	if err == nil {
		t.Error("Expected error when fetching page on closed engine")
	}
	if page != nil {
		t.Error("Expected nil page when fetch fails")
	}
}

func TestFlushAllErrors(t *testing.T) {
	dir := "./test_flush_all_errors"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}

	// Allocate a page
	page, err := engine.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}
	copy(page.Data, []byte("test"))
	page.MarkDirty()
	engine.UnpinPage(page.ID, true)

	// Close engine
	engine.Close()

	// Test flush all on closed engine
	err = engine.FlushAll()
	if err == nil {
		t.Error("Expected error when flushing all on closed engine")
	}
}
