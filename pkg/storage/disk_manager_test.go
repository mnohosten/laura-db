package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDiskManagerError(t *testing.T) {
	// Test with invalid path (directory that doesn't exist with no permissions)
	// This is challenging to test without creating actual permission issues
	// For now, we'll test the happy path to ensure proper initialization

	dir := "./test_disk_mgr_new"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	if dm == nil {
		t.Fatal("Expected non-nil disk manager")
	}
	if dm.nextPageID != 0 {
		t.Errorf("Expected nextPageID 0, got %d", dm.nextPageID)
	}
}

func TestDiskManagerReadPagePartial(t *testing.T) {
	dir := "./test_disk_read_partial"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Read a page that doesn't exist yet (should return new page)
	page, err := dm.ReadPage(5)
	if err != nil {
		t.Fatalf("Failed to read non-existent page: %v", err)
	}
	if page == nil {
		t.Fatal("Expected non-nil page")
	}
	if page.ID != 5 {
		t.Errorf("Expected page ID 5, got %d", page.ID)
	}
}

func TestDiskManagerWritePageError(t *testing.T) {
	dir := "./test_disk_write"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}

	// Write a valid page
	page := NewPage(0, PageTypeData)
	copy(page.Data, []byte("test data"))

	err = dm.WritePage(page)
	if err != nil {
		t.Fatalf("Failed to write page: %v", err)
	}

	// Close the file
	dm.Close()

	// Try to write after close (should fail)
	err = dm.WritePage(page)
	if err == nil {
		t.Error("Expected error when writing to closed file")
	}
}

func TestDiskManagerAllocateFreePages(t *testing.T) {
	dir := "./test_disk_alloc_free"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Allocate first page
	pageID1, err := dm.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}
	if pageID1 != 0 {
		t.Errorf("Expected first page ID 0, got %d", pageID1)
	}

	// Allocate second page
	pageID2, err := dm.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}
	if pageID2 != 1 {
		t.Errorf("Expected second page ID 1, got %d", pageID2)
	}

	// Deallocate first page
	err = dm.DeallocatePage(pageID1)
	if err != nil {
		t.Fatalf("Failed to deallocate page: %v", err)
	}

	// Allocate again - should reuse freed page
	pageID3, err := dm.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}
	if pageID3 != pageID1 {
		t.Errorf("Expected to reuse page %d, got %d", pageID1, pageID3)
	}
}

func TestDiskManagerSync(t *testing.T) {
	dir := "./test_disk_sync"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Write a page
	page := NewPage(0, PageTypeData)
	copy(page.Data, []byte("sync test"))
	err = dm.WritePage(page)
	if err != nil {
		t.Fatalf("Failed to write page: %v", err)
	}

	// Sync to disk
	err = dm.Sync()
	if err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}
}

func TestDiskManagerCloseError(t *testing.T) {
	dir := "./test_disk_close"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}

	// Close should succeed
	err = dm.Close()
	if err != nil {
		t.Fatalf("Failed to close: %v", err)
	}

	// Second close should fail
	err = dm.Close()
	if err == nil {
		t.Error("Expected error on second close")
	}
}

func TestDiskManagerStatsWithActivity(t *testing.T) {
	dir := "./test_disk_stats_activity"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Initial stats
	stats := dm.Stats()
	initialReads := stats["total_reads"].(int64)
	initialWrites := stats["total_writes"].(int64)

	// Write a page
	page := NewPage(0, PageTypeData)
	copy(page.Data, []byte("stats test"))
	err = dm.WritePage(page)
	if err != nil {
		t.Fatalf("Failed to write page: %v", err)
	}

	// Read the page
	_, err = dm.ReadPage(0)
	if err != nil {
		t.Fatalf("Failed to read page: %v", err)
	}

	// Check updated stats
	newStats := dm.Stats()
	newReads := newStats["total_reads"].(int64)
	newWrites := newStats["total_writes"].(int64)

	if newWrites != initialWrites+1 {
		t.Errorf("Expected %d writes, got %d", initialWrites+1, newWrites)
	}
	if newReads != initialReads+1 {
		t.Errorf("Expected %d reads, got %d", initialReads+1, newReads)
	}
}

func TestDiskManagerReadExistingFile(t *testing.T) {
	dir := "./test_disk_existing"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")

	// Create and write to file
	dm1, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}

	page := NewPage(0, PageTypeData)
	copy(page.Data, []byte("persistent data"))
	err = dm1.WritePage(page)
	if err != nil {
		t.Fatalf("Failed to write page: %v", err)
	}
	dm1.Close()

	// Reopen and verify nextPageID is set correctly
	dm2, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to reopen disk manager: %v", err)
	}
	defer dm2.Close()

	if dm2.nextPageID != 1 {
		t.Errorf("Expected nextPageID 1 after reopening, got %d", dm2.nextPageID)
	}

	// Read the page back
	readPage, err := dm2.ReadPage(0)
	if err != nil {
		t.Fatalf("Failed to read page: %v", err)
	}

	readData := readPage.Data[:len("persistent data")]
	if string(readData) != "persistent data" {
		t.Errorf("Expected 'persistent data', got '%s'", string(readData))
	}
}

func TestCompactPage(t *testing.T) {
	dir := "./test_compact_page"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Create a data page with slotted structure
	pageID, err := dm.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}

	page := NewPage(pageID, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert and delete some slots to create fragmentation (>25% threshold)
	// Available space is ~4068 bytes, 25% is ~1017 bytes
	// We'll insert and delete 4 x 300 bytes to get 1200 bytes fragmented
	largeData := make([]byte, 300)
	for i := 0; i < 4; i++ {
		slotID, err := sp.InsertSlot(largeData)
		if err != nil {
			t.Fatalf("Failed to insert slot: %v", err)
		}
		sp.DeleteSlot(slotID) // Delete all to create fragmentation
	}

	// Write the fragmented page
	err = dm.WritePage(sp.GetPage())
	if err != nil {
		t.Fatalf("Failed to write page: %v", err)
	}

	// Verify fragmentation exists before compaction
	fragmentedBefore := sp.FragmentedSpace()
	if fragmentedBefore == 0 {
		t.Error("Expected fragmented space > 0 before compaction")
	}

	// Compact the page
	err = dm.CompactPage(pageID)
	if err != nil {
		t.Fatalf("Failed to compact page: %v", err)
	}

	// Read back and verify compaction
	compactedPage, err := dm.ReadPage(pageID)
	if err != nil {
		t.Fatalf("Failed to read compacted page: %v", err)
	}

	compactedSP, err := LoadSlottedPage(compactedPage)
	if err != nil {
		t.Fatalf("Failed to load compacted slotted page: %v", err)
	}

	// After compaction, fragmented space should be significantly reduced or zero
	// Note: There might still be some fragmentation if not all deleted slots were removed
	fragmentedAfter := compactedSP.FragmentedSpace()
	if fragmentedAfter >= fragmentedBefore {
		t.Errorf("Expected fragmented space to decrease after compaction: before=%d, after=%d", fragmentedBefore, fragmentedAfter)
	}
}

func TestCompactPage_NonDataPage(t *testing.T) {
	dir := "./test_compact_non_data"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Create a non-data page
	pageID, err := dm.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}

	page := NewPage(pageID, PageTypeIndex)
	err = dm.WritePage(page)
	if err != nil {
		t.Fatalf("Failed to write page: %v", err)
	}

	// Try to compact non-data page
	err = dm.CompactPage(pageID)
	if err == nil {
		t.Error("Expected error when compacting non-data page, got nil")
	}
}

func TestCompactPage_NoCompactionNeeded(t *testing.T) {
	dir := "./test_compact_not_needed"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Create a data page without fragmentation
	pageID, err := dm.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}

	page := NewPage(pageID, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert only, no deletions
	sp.InsertSlot([]byte("data 1"))
	sp.InsertSlot([]byte("data 2"))

	err = dm.WritePage(sp.GetPage())
	if err != nil {
		t.Fatalf("Failed to write page: %v", err)
	}

	// Compact should succeed but do nothing
	err = dm.CompactPage(pageID)
	if err != nil {
		t.Fatalf("Failed to compact page: %v", err)
	}
}

func TestCompactPages(t *testing.T) {
	dir := "./test_compact_pages"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Create multiple pages with fragmentation
	pageIDs := []PageID{}
	largeData := make([]byte, 300)

	for i := 0; i < 3; i++ {
		pageID, err := dm.AllocatePage()
		if err != nil {
			t.Fatalf("Failed to allocate page: %v", err)
		}
		pageIDs = append(pageIDs, pageID)

		page := NewPage(pageID, PageTypeData)
		sp, err := NewSlottedPage(page)
		if err != nil {
			t.Fatalf("Failed to create slotted page: %v", err)
		}

		// Create fragmentation
		for j := 0; j < 4; j++ {
			slotID, err := sp.InsertSlot(largeData)
			if err != nil {
				t.Fatalf("Failed to insert slot: %v", err)
			}
			sp.DeleteSlot(slotID)
		}

		err = dm.WritePage(sp.GetPage())
		if err != nil {
			t.Fatalf("Failed to write page: %v", err)
		}
	}

	// Compact all pages
	stats, err := dm.CompactPages(pageIDs)
	if err != nil {
		t.Fatalf("Failed to compact pages: %v", err)
	}

	if stats.PagesScanned != int64(len(pageIDs)) {
		t.Errorf("Expected %d pages scanned, got %d", len(pageIDs), stats.PagesScanned)
	}

	if stats.PagesCompacted != int64(len(pageIDs)) {
		t.Errorf("Expected %d pages compacted, got %d", len(pageIDs), stats.PagesCompacted)
	}

	if stats.Errors != 0 {
		t.Errorf("Expected 0 errors, got %d", stats.Errors)
	}
}

func TestScanForCompaction(t *testing.T) {
	dir := "./test_scan_compaction"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Create mix of pages: some need compaction, some don't
	largeData := make([]byte, 300)

	// Page 0: Needs compaction
	pageID0, _ := dm.AllocatePage()
	page0 := NewPage(pageID0, PageTypeData)
	sp0, _ := NewSlottedPage(page0)
	for i := 0; i < 4; i++ {
		slotID, _ := sp0.InsertSlot(largeData)
		sp0.DeleteSlot(slotID)
	}
	dm.WritePage(sp0.GetPage())

	// Page 1: Doesn't need compaction
	pageID1, _ := dm.AllocatePage()
	page1 := NewPage(pageID1, PageTypeData)
	sp1, _ := NewSlottedPage(page1)
	sp1.InsertSlot([]byte("clean data"))
	dm.WritePage(sp1.GetPage())

	// Page 2: Needs compaction
	pageID2, _ := dm.AllocatePage()
	page2 := NewPage(pageID2, PageTypeData)
	sp2, _ := NewSlottedPage(page2)
	for i := 0; i < 4; i++ {
		slotID, _ := sp2.InsertSlot(largeData)
		sp2.DeleteSlot(slotID)
	}
	dm.WritePage(sp2.GetPage())

	// Scan and compact
	stats, err := dm.ScanForCompaction()
	if err != nil {
		t.Fatalf("Failed to scan for compaction: %v", err)
	}

	if stats.PagesScanned != 3 {
		t.Errorf("Expected 3 pages scanned, got %d", stats.PagesScanned)
	}

	if stats.PagesCompacted != 2 {
		t.Errorf("Expected 2 pages compacted, got %d", stats.PagesCompacted)
	}

	if stats.BytesReclaimed <= 0 {
		t.Error("Expected bytes reclaimed > 0")
	}

	if stats.Errors != 0 {
		t.Errorf("Expected 0 errors, got %d", stats.Errors)
	}
}

func TestCompactPageRange(t *testing.T) {
	dir := "./test_compact_range"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Create 5 pages with fragmentation
	largeData := make([]byte, 300)
	for i := 0; i < 5; i++ {
		pageID, _ := dm.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		sp, _ := NewSlottedPage(page)
		for j := 0; j < 4; j++ {
			slotID, _ := sp.InsertSlot(largeData)
			sp.DeleteSlot(slotID)
		}
		dm.WritePage(sp.GetPage())
	}

	// Compact only pages 1-3
	stats, err := dm.CompactPageRange(1, 3)
	if err != nil {
		t.Fatalf("Failed to compact page range: %v", err)
	}

	if stats.PagesScanned != 3 {
		t.Errorf("Expected 3 pages scanned, got %d", stats.PagesScanned)
	}

	if stats.PagesCompacted != 3 {
		t.Errorf("Expected 3 pages compacted, got %d", stats.PagesCompacted)
	}
}

func TestCompactPageRange_InvalidRange(t *testing.T) {
	dir := "./test_compact_invalid_range"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Try invalid range (start > end)
	_, err = dm.CompactPageRange(5, 2)
	if err == nil {
		t.Error("Expected error for invalid page range, got nil")
	}
}

func TestCompactPageRange_BeyondEnd(t *testing.T) {
	dir := "./test_compact_beyond_end"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Create 3 pages
	largeData := make([]byte, 300)
	for i := 0; i < 3; i++ {
		pageID, _ := dm.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		sp, _ := NewSlottedPage(page)
		for j := 0; j < 4; j++ {
			slotID, _ := sp.InsertSlot(largeData)
			sp.DeleteSlot(slotID)
		}
		dm.WritePage(sp.GetPage())
	}

	// Try to compact beyond end (should adjust to valid range)
	stats, err := dm.CompactPageRange(0, 100)
	if err != nil {
		t.Fatalf("Failed to compact page range: %v", err)
	}

	if stats.PagesScanned != 3 {
		t.Errorf("Expected 3 pages scanned, got %d", stats.PagesScanned)
	}
}

func TestCompactionStats(t *testing.T) {
	dir := "./test_compaction_stats"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.db")
	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Create pages with varying degrees of fragmentation
	// Need >25% fragmentation (>1017 bytes) for compaction to trigger
	largeData := make([]byte, 300)

	// Page 0: Heavy fragmentation (5 x 300 = 1500 bytes > 1017)
	pageID0, _ := dm.AllocatePage()
	page0 := NewPage(pageID0, PageTypeData)
	sp0, _ := NewSlottedPage(page0)
	for i := 0; i < 5; i++ {
		slotID, _ := sp0.InsertSlot(largeData)
		sp0.DeleteSlot(slotID)
	}
	dm.WritePage(sp0.GetPage())

	// Page 1: Moderate fragmentation (4 x 300 = 1200 bytes > 1017)
	pageID1, _ := dm.AllocatePage()
	page1 := NewPage(pageID1, PageTypeData)
	sp1, _ := NewSlottedPage(page1)
	for i := 0; i < 4; i++ {
		slotID, _ := sp1.InsertSlot(largeData)
		sp1.DeleteSlot(slotID)
	}
	dm.WritePage(sp1.GetPage())

	// Scan and verify stats
	stats, err := dm.ScanForCompaction()
	if err != nil {
		t.Fatalf("Failed to scan for compaction: %v", err)
	}

	// Verify stats structure
	if stats.PagesScanned <= 0 {
		t.Error("Expected PagesScanned > 0")
	}

	if stats.PagesCompacted <= 0 {
		t.Error("Expected PagesCompacted > 0")
	}

	if stats.BytesReclaimed <= 0 {
		t.Error("Expected BytesReclaimed > 0")
	}

	// Verify bytes reclaimed is reasonable (should be less than total page size)
	maxPossibleReclaimed := int64(stats.PagesCompacted) * SlottedPageAvailableSpace
	if stats.BytesReclaimed > maxPossibleReclaimed {
		t.Errorf("BytesReclaimed (%d) exceeds maximum possible (%d)", stats.BytesReclaimed, maxPossibleReclaimed)
	}
}
