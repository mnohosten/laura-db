package storage

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewSlottedPage(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	if sp.header.SlotCount != 0 {
		t.Errorf("Expected SlotCount = 0, got %d", sp.header.SlotCount)
	}

	if sp.header.FreeSpaceStart != 0 {
		t.Errorf("Expected FreeSpaceStart = 0, got %d", sp.header.FreeSpaceStart)
	}

	expectedFreeSpaceEnd := uint16(len(page.Data))
	if sp.header.FreeSpaceEnd != expectedFreeSpaceEnd {
		t.Errorf("Expected FreeSpaceEnd = %d, got %d", expectedFreeSpaceEnd, sp.header.FreeSpaceEnd)
	}

	if sp.header.FragmentedSpace != 0 {
		t.Errorf("Expected FragmentedSpace = 0, got %d", sp.header.FragmentedSpace)
	}
}

func TestNewSlottedPage_InvalidPageType(t *testing.T) {
	page := NewPage(1, PageTypeIndex)
	_, err := NewSlottedPage(page)
	if err == nil {
		t.Error("Expected error for invalid page type, got nil")
	}
}

func TestInsertSlot(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert data
	data := []byte("Hello, World!")
	slotID, err := sp.InsertSlot(data)
	if err != nil {
		t.Fatalf("Failed to insert slot: %v", err)
	}

	if slotID != 0 {
		t.Errorf("Expected slotID = 0, got %d", slotID)
	}

	if sp.header.SlotCount != 1 {
		t.Errorf("Expected SlotCount = 1, got %d", sp.header.SlotCount)
	}

	// Verify slot entry
	slot := sp.slots[0]
	if slot.Length != uint16(len(data)) {
		t.Errorf("Expected slot length = %d, got %d", len(data), slot.Length)
	}

	if slot.IsDeleted() {
		t.Error("Slot should not be marked as deleted")
	}

	// Page should be marked dirty
	if !page.IsDirty {
		t.Error("Page should be marked as dirty after insert")
	}
}

func TestInsertSlot_EmptyData(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	_, err = sp.InsertSlot([]byte{})
	if err == nil {
		t.Error("Expected error for empty data, got nil")
	}
}

func TestGetSlot(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert data
	originalData := []byte("Test data for retrieval")
	slotID, err := sp.InsertSlot(originalData)
	if err != nil {
		t.Fatalf("Failed to insert slot: %v", err)
	}

	// Retrieve data
	retrievedData, err := sp.GetSlot(slotID)
	if err != nil {
		t.Fatalf("Failed to get slot: %v", err)
	}

	if !bytes.Equal(originalData, retrievedData) {
		t.Errorf("Data mismatch: expected %s, got %s", originalData, retrievedData)
	}
}

func TestGetSlot_InvalidSlotID(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	_, err = sp.GetSlot(0)
	if err == nil {
		t.Error("Expected error for invalid slot ID, got nil")
	}
}

func TestUpdateSlot_InPlace(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert data
	originalData := []byte("Original data")
	slotID, err := sp.InsertSlot(originalData)
	if err != nil {
		t.Fatalf("Failed to insert slot: %v", err)
	}

	// Update with smaller data (in-place update)
	newData := []byte("New data")
	err = sp.UpdateSlot(slotID, newData)
	if err != nil {
		t.Fatalf("Failed to update slot: %v", err)
	}

	// Verify updated data
	retrievedData, err := sp.GetSlot(slotID)
	if err != nil {
		t.Fatalf("Failed to get slot: %v", err)
	}

	if !bytes.Equal(newData, retrievedData) {
		t.Errorf("Data mismatch: expected %s, got %s", newData, retrievedData)
	}

	// Check fragmented space was tracked
	if sp.header.FragmentedSpace == 0 {
		t.Error("Expected fragmented space > 0 after shrinking update")
	}
}

func TestUpdateSlot_Relocation(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert data
	originalData := []byte("Short")
	slotID, err := sp.InsertSlot(originalData)
	if err != nil {
		t.Fatalf("Failed to insert slot: %v", err)
	}

	// Update with larger data (requires relocation)
	newData := []byte("Much longer data that requires relocation")
	err = sp.UpdateSlot(slotID, newData)
	if err != nil {
		t.Fatalf("Failed to update slot: %v", err)
	}

	// Verify updated data
	retrievedData, err := sp.GetSlot(slotID)
	if err != nil {
		t.Fatalf("Failed to get slot: %v", err)
	}

	if !bytes.Equal(newData, retrievedData) {
		t.Errorf("Data mismatch: expected %s, got %s", newData, retrievedData)
	}

	// Check slot was marked as updated
	if !sp.slots[slotID].IsUpdated() {
		t.Error("Slot should be marked as updated after relocation")
	}
}

func TestDeleteSlot(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert data
	data := []byte("Data to delete")
	slotID, err := sp.InsertSlot(data)
	if err != nil {
		t.Fatalf("Failed to insert slot: %v", err)
	}

	// Delete slot
	err = sp.DeleteSlot(slotID)
	if err != nil {
		t.Fatalf("Failed to delete slot: %v", err)
	}

	// Verify slot is deleted
	if !sp.slots[slotID].IsDeleted() {
		t.Error("Slot should be marked as deleted")
	}

	// Verify offset is zeroed
	if sp.slots[slotID].Offset != 0 {
		t.Error("Deleted slot offset should be 0")
	}

	// Verify fragmented space increased
	if sp.header.FragmentedSpace == 0 {
		t.Error("Expected fragmented space > 0 after deletion")
	}

	// Try to get deleted slot
	_, err = sp.GetSlot(slotID)
	if err == nil {
		t.Error("Expected error when getting deleted slot, got nil")
	}
}

func TestDeleteSlot_AlreadyDeleted(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert and delete
	data := []byte("Data")
	slotID, _ := sp.InsertSlot(data)
	sp.DeleteSlot(slotID)

	// Try to delete again
	err = sp.DeleteSlot(slotID)
	if err == nil {
		t.Error("Expected error when deleting already deleted slot, got nil")
	}
}

func TestMultipleInserts(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert multiple items
	items := [][]byte{
		[]byte("First item"),
		[]byte("Second item"),
		[]byte("Third item"),
		[]byte("Fourth item"),
	}

	slotIDs := []uint16{}
	for _, item := range items {
		slotID, err := sp.InsertSlot(item)
		if err != nil {
			t.Fatalf("Failed to insert slot: %v", err)
		}
		slotIDs = append(slotIDs, slotID)
	}

	// Verify all items
	for i, slotID := range slotIDs {
		data, err := sp.GetSlot(slotID)
		if err != nil {
			t.Fatalf("Failed to get slot %d: %v", slotID, err)
		}

		if !bytes.Equal(items[i], data) {
			t.Errorf("Data mismatch for slot %d: expected %s, got %s", slotID, items[i], data)
		}
	}

	if sp.header.SlotCount != uint16(len(items)) {
		t.Errorf("Expected SlotCount = %d, got %d", len(items), sp.header.SlotCount)
	}
}

func TestFreeSpaceTracking(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	initialFree := sp.ContiguousFreeSpace()

	// Insert data
	data := []byte("Test data")
	slotID, err := sp.InsertSlot(data)
	if err != nil {
		t.Fatalf("Failed to insert slot: %v", err)
	}

	// Free space should decrease by data length + slot entry + slot directory header space
	afterInsertFree := sp.ContiguousFreeSpace()
	// The decrease includes: data length, slot entry (which moves FreeSpaceStart)
	actualDecrease := initialFree - afterInsertFree
	expectedMinimumDecrease := uint16(len(data)) + SlotEntrySize
	if actualDecrease < expectedMinimumDecrease {
		t.Errorf("Expected free space to decrease by at least %d, decreased by %d", expectedMinimumDecrease, actualDecrease)
	}

	// Delete slot
	sp.DeleteSlot(slotID)

	// Fragmented space should increase
	if sp.header.FragmentedSpace != uint16(len(data)) {
		t.Errorf("Expected FragmentedSpace = %d, got %d", len(data), sp.header.FragmentedSpace)
	}

	// Total free space should include fragmented
	totalFree := sp.TotalFreeSpace()
	expectedTotal := afterInsertFree + uint16(len(data))
	if totalFree != expectedTotal {
		t.Errorf("Expected TotalFreeSpace = %d, got %d", expectedTotal, totalFree)
	}
}

func TestCompaction(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert multiple items
	items := [][]byte{
		[]byte("Item 1"),
		[]byte("Item 2"),
		[]byte("Item 3"),
		[]byte("Item 4"),
		[]byte("Item 5"),
	}

	slotIDs := []uint16{}
	for _, item := range items {
		slotID, err := sp.InsertSlot(item)
		if err != nil {
			t.Fatalf("Failed to insert slot: %v", err)
		}
		slotIDs = append(slotIDs, slotID)
	}

	// Delete some slots (create fragmentation)
	sp.DeleteSlot(slotIDs[1]) // Delete "Item 2"
	sp.DeleteSlot(slotIDs[3]) // Delete "Item 4"

	fragmentedBefore := sp.header.FragmentedSpace
	if fragmentedBefore == 0 {
		t.Error("Expected fragmented space > 0 after deletions")
	}

	// Compact the page
	err = sp.Compact()
	if err != nil {
		t.Fatalf("Failed to compact page: %v", err)
	}

	// Fragmented space should be 0 after compaction
	if sp.header.FragmentedSpace != 0 {
		t.Errorf("Expected FragmentedSpace = 0 after compaction, got %d", sp.header.FragmentedSpace)
	}

	// Should have only 3 active slots
	if sp.header.SlotCount != 3 {
		t.Errorf("Expected SlotCount = 3 after compaction, got %d", sp.header.SlotCount)
	}

	// Verify remaining data is intact
	remainingItems := [][]byte{items[0], items[2], items[4]}
	for i := uint16(0); i < sp.header.SlotCount; i++ {
		data, err := sp.GetSlot(i)
		if err != nil {
			t.Fatalf("Failed to get slot %d after compaction: %v", i, err)
		}

		if !bytes.Equal(remainingItems[i], data) {
			t.Errorf("Data mismatch after compaction for slot %d: expected %s, got %s", i, remainingItems[i], data)
		}
	}
}

func TestCompaction_EmptyPage(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert and delete all
	slotID, _ := sp.InsertSlot([]byte("Data"))
	sp.DeleteSlot(slotID)

	// Compact
	err = sp.Compact()
	if err != nil {
		t.Fatalf("Failed to compact page: %v", err)
	}

	// Should be completely empty
	if sp.header.SlotCount != 0 {
		t.Errorf("Expected SlotCount = 0 after compacting empty page, got %d", sp.header.SlotCount)
	}

	if sp.header.FragmentedSpace != 0 {
		t.Errorf("Expected FragmentedSpace = 0, got %d", sp.header.FragmentedSpace)
	}

	// Free space should be maximized
	expectedFree := uint16(len(page.Data)) - SlottedPageHeaderSize
	actualFree := sp.ContiguousFreeSpace()
	if actualFree != expectedFree {
		t.Errorf("Expected free space = %d, got %d", expectedFree, actualFree)
	}
}

func TestNeedsCompaction(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Initially shouldn't need compaction
	if sp.NeedsCompaction() {
		t.Error("Empty page should not need compaction")
	}

	// Insert and delete enough to trigger compaction threshold
	// Available space is ~4068 bytes, 25% threshold is ~1017 bytes
	largeData := make([]byte, 300)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// Insert 4 items and delete all 4 (1200 bytes fragmented, ~29.5%)
	for i := 0; i < 4; i++ {
		slotID, err := sp.InsertSlot(largeData)
		if err != nil {
			t.Fatalf("Failed to insert slot: %v", err)
		}
		sp.DeleteSlot(slotID)
	}

	// Should need compaction now
	if !sp.NeedsCompaction() {
		t.Errorf("Page should need compaction with %d bytes fragmented (%.2f%%)",
			sp.header.FragmentedSpace,
			float64(sp.header.FragmentedSpace)/float64(SlottedPageAvailableSpace)*100)
	}
}

func TestLoadSlottedPage(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert some data
	items := [][]byte{
		[]byte("Item 1"),
		[]byte("Item 2"),
		[]byte("Item 3"),
	}

	for _, item := range items {
		_, err := sp.InsertSlot(item)
		if err != nil {
			t.Fatalf("Failed to insert slot: %v", err)
		}
	}

	// Load the page
	loadedSP, err := LoadSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to load slotted page: %v", err)
	}

	// Verify header
	if loadedSP.header.SlotCount != sp.header.SlotCount {
		t.Errorf("SlotCount mismatch: expected %d, got %d", sp.header.SlotCount, loadedSP.header.SlotCount)
	}

	if loadedSP.header.FreeSpaceStart != sp.header.FreeSpaceStart {
		t.Errorf("FreeSpaceStart mismatch: expected %d, got %d", sp.header.FreeSpaceStart, loadedSP.header.FreeSpaceStart)
	}

	if loadedSP.header.FreeSpaceEnd != sp.header.FreeSpaceEnd {
		t.Errorf("FreeSpaceEnd mismatch: expected %d, got %d", sp.header.FreeSpaceEnd, loadedSP.header.FreeSpaceEnd)
	}

	// Verify data
	for i := uint16(0); i < loadedSP.header.SlotCount; i++ {
		data, err := loadedSP.GetSlot(i)
		if err != nil {
			t.Fatalf("Failed to get slot %d from loaded page: %v", i, err)
		}

		if !bytes.Equal(items[i], data) {
			t.Errorf("Data mismatch for slot %d: expected %s, got %s", i, items[i], data)
		}
	}
}

func TestSlotFlags(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert data
	slotID, _ := sp.InsertSlot([]byte("Test"))

	// Check initial flags
	slot := &sp.slots[slotID]
	if slot.IsDeleted() {
		t.Error("New slot should not be deleted")
	}
	if slot.IsOverflow() {
		t.Error("New slot should not be overflow")
	}
	if slot.IsUpdated() {
		t.Error("New slot should not be updated")
	}

	// Delete and check flag
	sp.DeleteSlot(slotID)
	if !sp.slots[slotID].IsDeleted() {
		t.Error("Deleted slot should have deleted flag")
	}
}

func TestStats(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Insert and delete some slots
	slotID1, _ := sp.InsertSlot([]byte("Active 1"))
	slotID2, _ := sp.InsertSlot([]byte("Active 2"))
	slotID3, _ := sp.InsertSlot([]byte("Deleted"))
	sp.DeleteSlot(slotID3)

	stats := sp.Stats()

	if stats["slot_count"] != uint16(3) {
		t.Errorf("Expected slot_count = 3, got %v", stats["slot_count"])
	}

	if stats["active_slots"] != 2 {
		t.Errorf("Expected active_slots = 2, got %v", stats["active_slots"])
	}

	if stats["deleted_slots"] != 1 {
		t.Errorf("Expected deleted_slots = 1, got %v", stats["deleted_slots"])
	}

	// Verify we can still access active slots
	_, err = sp.GetSlot(slotID1)
	if err != nil {
		t.Errorf("Should be able to access active slot: %v", err)
	}

	_, err = sp.GetSlot(slotID2)
	if err != nil {
		t.Errorf("Should be able to access active slot: %v", err)
	}
}

func TestInsufficientSpace(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Try to insert data larger than page
	largeData := make([]byte, SlottedPageAvailableSpace+100)
	_, err = sp.InsertSlot(largeData)
	if err == nil {
		t.Error("Expected error when inserting data larger than page, got nil")
	}
}

func TestAutoCompaction(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Fill with data and delete to create fragmentation
	largeData := make([]byte, 300)
	for i := 0; i < 4; i++ {
		slotID, err := sp.InsertSlot(largeData)
		if err != nil {
			t.Fatalf("Failed to insert slot: %v", err)
		}
		sp.DeleteSlot(slotID)
	}

	// Should need compaction
	if !sp.NeedsCompaction() {
		t.Errorf("Page should need compaction with %d bytes fragmented", sp.header.FragmentedSpace)
	}

	// Try to insert - should trigger auto-compaction
	newData := []byte("New data after compaction")
	slotID, err := sp.InsertSlot(newData)
	if err != nil {
		t.Fatalf("Failed to insert slot (should auto-compact): %v", err)
	}

	// Fragmented space should be reduced after auto-compaction
	if sp.NeedsCompaction() {
		t.Error("Page should not need compaction after auto-compact")
	}

	// Verify new data is retrievable
	retrievedData, err := sp.GetSlot(slotID)
	if err != nil {
		t.Fatalf("Failed to get slot: %v", err)
	}

	if !bytes.Equal(newData, retrievedData) {
		t.Errorf("Data mismatch: expected %s, got %s", newData, retrievedData)
	}
}

// TestLoadSlottedPageWithInvalidPageType tests LoadSlottedPage with wrong page type
func TestLoadSlottedPageWithInvalidPageType(t *testing.T) {
	// Create a free list page instead of data page
	page := NewPage(1, PageTypeFreeList)

	// Try to load as slotted page - should fail
	_, err := LoadSlottedPage(page)
	if err == nil {
		t.Error("Expected error when loading slotted page from non-data page")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid page type") {
		t.Errorf("Expected 'invalid page type' error, got: %v", err)
	}
}

// TestLoadSlottedPageWithCorruptedHeader tests LoadSlottedPage with corrupted header
func TestLoadSlottedPageWithCorruptedHeader(t *testing.T) {
	page := NewPage(1, PageTypeData)
	// Make the page data too small for a proper header
	page.Data = make([]byte, 4) // Less than SlottedPageHeaderSize

	_, err := LoadSlottedPage(page)
	if err == nil {
		t.Error("Expected error when loading slotted page with corrupted header")
	}
}

// TestContiguousFreeSpace tests the ContiguousFreeSpace method
func TestContiguousFreeSpace(t *testing.T) {
	page := NewPage(1, PageTypeData)
	sp, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Initially should have maximum contiguous space
	initialSpace := sp.ContiguousFreeSpace()
	if initialSpace <= 0 {
		t.Error("Expected positive contiguous free space initially")
	}

	// Insert some data
	data := []byte("Test data")
	_, err = sp.InsertSlot(data)
	if err != nil {
		t.Fatalf("Failed to insert slot: %v", err)
	}

	// Contiguous space should be less now
	afterInsertSpace := sp.ContiguousFreeSpace()
	if afterInsertSpace >= initialSpace {
		t.Error("Contiguous space should decrease after insertion")
	}
}

// TestNewSlottedPageWithNonDataPage tests error handling for wrong page type
func TestNewSlottedPageWithNonDataPage(t *testing.T) {
	page := NewPage(1, PageTypeFreeList)

	_, err := NewSlottedPage(page)
	if err == nil {
		t.Error("Expected error when creating slotted page from non-data page")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid page type") {
		t.Errorf("Expected 'invalid page type' error, got: %v", err)
	}
}
