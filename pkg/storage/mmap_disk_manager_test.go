package storage

import (
	"path/filepath"
	"testing"
)

func TestMmapDiskManager_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mmap.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		t.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Allocate a page
	pageID, err := dm.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}

	if pageID != 0 {
		t.Errorf("Expected first page ID to be 0, got %d", pageID)
	}

	// Write a page
	page := NewPage(pageID, PageTypeData)
	copy(page.Data, []byte("Hello, mmap!"))
	page.LSN = 42

	if err := dm.WritePage(page); err != nil {
		t.Fatalf("Failed to write page: %v", err)
	}

	// Read the page back
	readPage, err := dm.ReadPage(pageID)
	if err != nil {
		t.Fatalf("Failed to read page: %v", err)
	}

	if readPage.LSN != 42 {
		t.Errorf("Expected LSN 42, got %d", readPage.LSN)
	}

	data := string(readPage.Data[:12])
	if data != "Hello, mmap!" {
		t.Errorf("Expected 'Hello, mmap!', got '%s'", data)
	}
}

func TestMmapDiskManager_MultiplePages(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mmap_multi.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		t.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Allocate and write multiple pages
	const numPages = 100
	for i := 0; i < numPages; i++ {
		pageID, err := dm.AllocatePage()
		if err != nil {
			t.Fatalf("Failed to allocate page %d: %v", i, err)
		}

		page := NewPage(pageID, PageTypeData)
		page.LSN = uint64(i + 1000)
		copy(page.Data, []byte{byte(i % 256)})

		if err := dm.WritePage(page); err != nil {
			t.Fatalf("Failed to write page %d: %v", i, err)
		}
	}

	// Read all pages back
	for i := 0; i < numPages; i++ {
		page, err := dm.ReadPage(PageID(i))
		if err != nil {
			t.Fatalf("Failed to read page %d: %v", i, err)
		}

		expectedLSN := uint64(i + 1000)
		if page.LSN != expectedLSN {
			t.Errorf("Page %d: expected LSN %d, got %d", i, expectedLSN, page.LSN)
		}

		expectedByte := byte(i % 256)
		if page.Data[0] != expectedByte {
			t.Errorf("Page %d: expected first byte %d, got %d", i, expectedByte, page.Data[0])
		}
	}
}

func TestMmapDiskManager_Sync(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mmap_sync.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		t.Fatalf("Failed to create mmap disk manager: %v", err)
	}

	// Write a page
	pageID, _ := dm.AllocatePage()
	page := NewPage(pageID, PageTypeData)
	copy(page.Data, []byte("sync test"))
	dm.WritePage(page)

	// Sync
	if err := dm.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	// Close and reopen
	dm.Close()

	dm2, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		t.Fatalf("Failed to reopen mmap disk manager: %v", err)
	}
	defer dm2.Close()

	// Read page back
	readPage, err := dm2.ReadPage(pageID)
	if err != nil {
		t.Fatalf("Failed to read page after reopen: %v", err)
	}

	data := string(readPage.Data[:9])
	if data != "sync test" {
		t.Errorf("Expected 'sync test', got '%s'", data)
	}
}

func TestMmapDiskManager_DeallocateAndReuse(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mmap_dealloc.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		t.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Allocate three pages
	page1, _ := dm.AllocatePage()
	page2, _ := dm.AllocatePage()
	page3, _ := dm.AllocatePage()

	if page1 != 0 || page2 != 1 || page3 != 2 {
		t.Errorf("Unexpected page IDs: %d, %d, %d", page1, page2, page3)
	}

	// Deallocate middle page
	dm.DeallocatePage(page2)

	// Next allocation should reuse deallocated page
	page4, _ := dm.AllocatePage()
	if page4 != page2 {
		t.Errorf("Expected to reuse page %d, got %d", page2, page4)
	}

	// Next allocation should get a new page
	page5, _ := dm.AllocatePage()
	if page5 != 3 {
		t.Errorf("Expected new page ID 3, got %d", page5)
	}
}

func TestMmapDiskManager_Expansion(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mmap_expand.db")

	// Create with small initial size
	config := &MmapConfig{
		InitialSize: 4 * PageSize, // Only 4 pages initially
		GrowthSize:  2 * PageSize,
	}

	dm, err := NewMmapDiskManager(dbPath, config)
	if err != nil {
		t.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Allocate more pages than initial size
	for i := 0; i < 10; i++ {
		pageID, err := dm.AllocatePage()
		if err != nil {
			t.Fatalf("Failed to allocate page %d: %v", i, err)
		}

		page := NewPage(pageID, PageTypeData)
		page.Data[0] = byte(i)

		if err := dm.WritePage(page); err != nil {
			t.Fatalf("Failed to write page %d: %v", i, err)
		}
	}

	// Verify all pages can be read
	for i := 0; i < 10; i++ {
		page, err := dm.ReadPage(PageID(i))
		if err != nil {
			t.Fatalf("Failed to read expanded page %d: %v", i, err)
		}

		if page.Data[0] != byte(i) {
			t.Errorf("Page %d: expected data %d, got %d", i, i, page.Data[0])
		}
	}

	// Check that mmap was expanded
	stats := dm.Stats()
	mmapSize := stats["mmap_size"].(int64)
	if mmapSize < 10*PageSize {
		t.Errorf("Expected mmap size >= %d, got %d", 10*PageSize, mmapSize)
	}
}

func TestMmapDiskManager_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mmap_stats.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		t.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Initial stats
	stats := dm.Stats()
	if stats["total_reads"].(int64) != 0 {
		t.Errorf("Expected 0 reads initially")
	}
	if stats["total_writes"].(int64) != 0 {
		t.Errorf("Expected 0 writes initially")
	}

	// Allocate and write a page
	pageID, _ := dm.AllocatePage()
	page := NewPage(pageID, PageTypeData)
	dm.WritePage(page)

	// Read the page
	dm.ReadPage(pageID)

	// Check updated stats
	stats = dm.Stats()
	if stats["total_reads"].(int64) != 1 {
		t.Errorf("Expected 1 read, got %d", stats["total_reads"].(int64))
	}
	if stats["total_writes"].(int64) != 1 {
		t.Errorf("Expected 1 write, got %d", stats["total_writes"].(int64))
	}
}

func TestMmapDiskManager_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mmap_persist.db")

	// Create and write data
	dm1, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		t.Fatalf("Failed to create mmap disk manager: %v", err)
	}

	const numPages = 50
	for i := 0; i < numPages; i++ {
		pageID, _ := dm1.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		page.LSN = uint64(i * 10)
		copy(page.Data, []byte{byte(i % 256), byte((i + 1) % 256)})
		dm1.WritePage(page)
	}

	dm1.Close()

	// Reopen and verify data
	dm2, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		t.Fatalf("Failed to reopen mmap disk manager: %v", err)
	}
	defer dm2.Close()

	for i := 0; i < numPages; i++ {
		page, err := dm2.ReadPage(PageID(i))
		if err != nil {
			t.Fatalf("Failed to read page %d after reopen: %v", i, err)
		}

		expectedLSN := uint64(i * 10)
		if page.LSN != expectedLSN {
			t.Errorf("Page %d: expected LSN %d, got %d", i, expectedLSN, page.LSN)
		}

		if page.Data[0] != byte(i%256) || page.Data[1] != byte((i+1)%256) {
			t.Errorf("Page %d: data mismatch", i)
		}
	}
}

func TestMmapDiskManager_MadviseRandom(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mmap_madvise.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		t.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Test madvise random (should not error)
	if err := dm.MadviseRandom(); err != nil {
		t.Errorf("MadviseRandom failed: %v", err)
	}
}

func TestMmapDiskManager_MadviseSequential(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mmap_madvise_seq.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		t.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Test madvise sequential (should not error)
	if err := dm.MadviseSequential(); err != nil {
		t.Errorf("MadviseSequential failed: %v", err)
	}
}

func TestMmapDiskManager_MadviseWillNeed(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mmap_willneed.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		t.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Allocate some pages
	for i := 0; i < 10; i++ {
		pageID, _ := dm.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		dm.WritePage(page)
	}

	// Test madvise will need
	if err := dm.MadviseWillNeed(0, 5); err != nil {
		t.Errorf("MadviseWillNeed failed: %v", err)
	}

	// Test with out-of-range pages (should error)
	// DefaultMmapConfig creates 256MB, which is 64K pages (256MB / 4KB)
	// So we need to request pages beyond 64K
	err = dm.MadviseWillNeed(100000, 200000)
	if err == nil {
		t.Error("Expected error for out-of-range pages")
	} else {
		// Verify it's the right kind of error
		if err.Error() != "page range exceeds mmap size" {
			t.Errorf("Expected 'page range exceeds mmap size' error, got: %v", err)
		}
	}
}

func TestMmapDiskManager_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mmap_concurrent.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		t.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Allocate pages first
	const numPages = 50
	for i := 0; i < numPages; i++ {
		dm.AllocatePage()
	}

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 5; j++ {
				pageID := PageID(id*5 + j)
				page := NewPage(pageID, PageTypeData)
				page.Data[0] = byte(id)
				dm.WritePage(page)
			}
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 5; j++ {
				pageID := PageID(id*5 + j)
				page, err := dm.ReadPage(pageID)
				if err != nil {
					t.Errorf("Concurrent read failed: %v", err)
				}
				if page.Data[0] != byte(id) {
					t.Errorf("Concurrent read data mismatch")
				}
			}
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMmapDiskManager_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mmap_existing.db")

	// Create file with standard disk manager first
	standardDM, err := NewDiskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create standard disk manager: %v", err)
	}

	// Write some pages with standard disk manager
	for i := 0; i < 5; i++ {
		pageID, _ := standardDM.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		page.LSN = uint64(i + 100)
		copy(page.Data, []byte{byte(i * 10)})
		standardDM.WritePage(page)
	}
	standardDM.Close()

	// Open with mmap disk manager
	mmapDM, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		t.Fatalf("Failed to open existing file with mmap: %v", err)
	}
	defer mmapDM.Close()

	// Verify existing pages can be read
	for i := 0; i < 5; i++ {
		page, err := mmapDM.ReadPage(PageID(i))
		if err != nil {
			t.Fatalf("Failed to read existing page %d: %v", i, err)
		}

		expectedLSN := uint64(i + 100)
		if page.LSN != expectedLSN {
			t.Errorf("Page %d: expected LSN %d, got %d", i, expectedLSN, page.LSN)
		}

		if page.Data[0] != byte(i*10) {
			t.Errorf("Page %d: expected data %d, got %d", i, i*10, page.Data[0])
		}
	}
}
