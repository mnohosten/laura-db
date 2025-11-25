package storage

import (
	"testing"
)

func TestFreePageHeader(t *testing.T) {
	page := NewPage(1, PageTypeFreeList)

	// Create and serialize header
	header := &FreePageHeader{
		NextFreeListPage: 42,
		EntryCount:       10,
	}

	SerializeFreePageHeader(page, header)

	// Deserialize and verify
	deserialized, err := DeserializeFreePageHeader(page)
	if err != nil {
		t.Fatalf("Failed to deserialize header: %v", err)
	}

	if deserialized.NextFreeListPage != 42 {
		t.Errorf("Expected NextFreeListPage=42, got %d", deserialized.NextFreeListPage)
	}
	if deserialized.EntryCount != 10 {
		t.Errorf("Expected EntryCount=10, got %d", deserialized.EntryCount)
	}
}

func TestFreePageHeaderInvalidPageType(t *testing.T) {
	page := NewPage(1, PageTypeData)

	_, err := DeserializeFreePageHeader(page)
	if err == nil {
		t.Error("Expected error for invalid page type, got nil")
	}
}

func TestWriteReadFreePageEntry(t *testing.T) {
	page := NewPage(1, PageTypeFreeList)
	InitializeFreeListPage(page)

	// Write entries
	entries := []PageID{100, 200, 300, 400, 500}
	for i, pageID := range entries {
		err := WriteFreePageEntry(page, uint32(i), pageID)
		if err != nil {
			t.Fatalf("Failed to write entry %d: %v", i, err)
		}
	}

	// Read and verify entries
	for i, expectedPageID := range entries {
		pageID, err := ReadFreePageEntry(page, uint32(i))
		if err != nil {
			t.Fatalf("Failed to read entry %d: %v", i, err)
		}
		if pageID != expectedPageID {
			t.Errorf("Entry %d: expected %d, got %d", i, expectedPageID, pageID)
		}
	}
}

func TestWriteFreePageEntryExceedsMax(t *testing.T) {
	page := NewPage(1, PageTypeFreeList)
	InitializeFreeListPage(page)

	// Try to write beyond max entries
	err := WriteFreePageEntry(page, MaxFreePageEntries, 100)
	if err == nil {
		t.Error("Expected error when writing beyond max entries, got nil")
	}
}

func TestReadAllFreePageEntries(t *testing.T) {
	page := NewPage(1, PageTypeFreeList)
	InitializeFreeListPage(page)

	// Add entries
	expectedEntries := []PageID{10, 20, 30, 40, 50}
	header := &FreePageHeader{
		NextFreeListPage: 0,
		EntryCount:       uint32(len(expectedEntries)),
	}

	for i, pageID := range expectedEntries {
		WriteFreePageEntry(page, uint32(i), pageID)
	}
	SerializeFreePageHeader(page, header)

	// Read all entries
	entries, err := ReadAllFreePageEntries(page)
	if err != nil {
		t.Fatalf("Failed to read all entries: %v", err)
	}

	if len(entries) != len(expectedEntries) {
		t.Fatalf("Expected %d entries, got %d", len(expectedEntries), len(entries))
	}

	for i, pageID := range entries {
		if pageID != expectedEntries[i] {
			t.Errorf("Entry %d: expected %d, got %d", i, expectedEntries[i], pageID)
		}
	}
}

func TestInitializeFreeListPage(t *testing.T) {
	page := NewPage(1, PageTypeData)
	InitializeFreeListPage(page)

	if page.Type != PageTypeFreeList {
		t.Errorf("Expected page type %v, got %v", PageTypeFreeList, page.Type)
	}

	if !page.IsDirty {
		t.Error("Expected page to be marked dirty")
	}

	header, err := DeserializeFreePageHeader(page)
	if err != nil {
		t.Fatalf("Failed to deserialize header: %v", err)
	}

	if header.NextFreeListPage != 0 {
		t.Errorf("Expected NextFreeListPage=0, got %d", header.NextFreeListPage)
	}
	if header.EntryCount != 0 {
		t.Errorf("Expected EntryCount=0, got %d", header.EntryCount)
	}
}

func TestAddFreePageToList(t *testing.T) {
	page := NewPage(1, PageTypeFreeList)
	InitializeFreeListPage(page)

	// Add pages until full
	var addedCount uint32
	for i := uint32(0); i < MaxFreePageEntries; i++ {
		added, err := AddFreePageToList(page, PageID(100+i))
		if err != nil {
			t.Fatalf("Failed to add page %d: %v", i, err)
		}
		if !added {
			break
		}
		addedCount++
	}

	if addedCount != MaxFreePageEntries {
		t.Errorf("Expected to add %d pages, added %d", MaxFreePageEntries, addedCount)
	}

	// Try to add one more (should fail)
	added, err := AddFreePageToList(page, 999)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if added {
		t.Error("Expected page to be full, but page was added")
	}

	// Verify header
	header, err := DeserializeFreePageHeader(page)
	if err != nil {
		t.Fatalf("Failed to deserialize header: %v", err)
	}
	if header.EntryCount != MaxFreePageEntries {
		t.Errorf("Expected EntryCount=%d, got %d", MaxFreePageEntries, header.EntryCount)
	}
}

func TestRemoveFreePageFromList(t *testing.T) {
	page := NewPage(1, PageTypeFreeList)
	InitializeFreeListPage(page)

	// Add some pages
	pages := []PageID{100, 200, 300}
	for _, pageID := range pages {
		added, err := AddFreePageToList(page, pageID)
		if err != nil {
			t.Fatalf("Failed to add page: %v", err)
		}
		if !added {
			t.Fatal("Failed to add page")
		}
	}

	// Remove pages (LIFO order)
	for i := len(pages) - 1; i >= 0; i-- {
		expectedPageID := pages[i]
		pageID, removed, err := RemoveFreePageFromList(page)
		if err != nil {
			t.Fatalf("Failed to remove page: %v", err)
		}
		if !removed {
			t.Fatal("Expected page to be removed")
		}
		if pageID != expectedPageID {
			t.Errorf("Expected page %d, got %d", expectedPageID, pageID)
		}
	}

	// Try to remove from empty list
	_, removed, err := RemoveFreePageFromList(page)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if removed {
		t.Error("Expected removal to fail on empty list")
	}
}

func TestIsFreeListPageFull(t *testing.T) {
	page := NewPage(1, PageTypeFreeList)
	InitializeFreeListPage(page)

	// Initially not full
	full, err := IsFreeListPageFull(page)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if full {
		t.Error("Expected page to not be full initially")
	}

	// Fill the page
	for i := uint32(0); i < MaxFreePageEntries; i++ {
		AddFreePageToList(page, PageID(100+i))
	}

	// Now should be full
	full, err = IsFreeListPageFull(page)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !full {
		t.Error("Expected page to be full")
	}
}

func TestIsFreeListPageEmpty(t *testing.T) {
	page := NewPage(1, PageTypeFreeList)
	InitializeFreeListPage(page)

	// Initially empty
	empty, err := IsFreeListPageEmpty(page)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !empty {
		t.Error("Expected page to be empty initially")
	}

	// Add a page
	AddFreePageToList(page, 100)

	// Now should not be empty
	empty, err = IsFreeListPageEmpty(page)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if empty {
		t.Error("Expected page to not be empty")
	}

	// Remove the page
	RemoveFreePageFromList(page)

	// Should be empty again
	empty, err = IsFreeListPageEmpty(page)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !empty {
		t.Error("Expected page to be empty after removal")
	}
}

func TestFreePageListCapacity(t *testing.T) {
	// Verify that we can store a reasonable number of entries
	expectedMin := 1000 // Should be able to store at least 1000 page IDs per free list page

	if MaxFreePageEntries < uint32(expectedMin) {
		t.Errorf("Expected MaxFreePageEntries to be at least %d, got %d", expectedMin, MaxFreePageEntries)
	}

	t.Logf("MaxFreePageEntries: %d", MaxFreePageEntries)
	t.Logf("Free list page overhead: %d bytes", PageHeaderSize+FreePageHeaderSize)
	t.Logf("Usable space: %d bytes", PageSize-PageHeaderSize-FreePageHeaderSize)
}

// TestReadFreePageEntryOutOfBounds tests reading entry beyond valid range
func TestReadFreePageEntryOutOfBounds(t *testing.T) {
	page := NewPage(1, PageTypeFreeList)

	// Initialize the free list page
	InitializeFreeListPage(page, 0)

	// Try to read entry beyond max entries
	_, err := ReadFreePageEntry(page, MaxFreePageEntries+1)
	if err == nil {
		t.Error("Expected error when reading entry out of bounds")
	}
}

// TestSerializeFreePageHeaderEdgeCases tests edge cases for header serialization
func TestSerializeFreePageHeaderEdgeCases(t *testing.T) {
	page := NewPage(1, PageTypeFreeList)

	// Create header with maximum values
	header := &FreePageHeader{
		NextFreeListPage: ^PageID(0), // Max PageID
		EntryCount:       MaxFreePageEntries,
	}

	SerializeFreePageHeader(page, header)

	// Deserialize it back
	desHeader := DeserializeFreePageHeader(page)

	if desHeader.NextFreeListPage != header.NextFreeListPage {
		t.Errorf("NextFreeListPage mismatch: expected %d, got %d", header.NextFreeListPage, desHeader.NextFreeListPage)
	}

	if desHeader.EntryCount != header.EntryCount {
		t.Errorf("EntryCount mismatch: expected %d, got %d", header.EntryCount, desHeader.EntryCount)
	}
}
