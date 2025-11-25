package storage

import (
	"encoding/binary"
	"fmt"
)

const (
	// SlottedPageHeaderSize is the size of the slotted page header (12 bytes)
	SlottedPageHeaderSize = 12

	// SlotEntrySize is the size of each slot directory entry (5 bytes)
	SlotEntrySize = 5

	// SlottedPageAvailableSpace is the available space for slots and data
	// PageSize - PageHeaderSize - SlottedPageHeaderSize
	SlottedPageAvailableSpace = PageSize - PageHeaderSize - SlottedPageHeaderSize

	// PageCompactionThreshold is the fragmentation percentage that triggers compaction
	PageCompactionThreshold = 0.25 // 25%
)

// SlotFlags represents flags for a slot
type SlotFlags uint8

const (
	// SlotFlagDeleted indicates the slot is deleted
	SlotFlagDeleted SlotFlags = 1 << 0

	// SlotFlagOverflow indicates the document spans multiple pages
	SlotFlagOverflow SlotFlags = 1 << 1

	// SlotFlagUpdated indicates the slot was updated (old version in overflow)
	SlotFlagUpdated SlotFlags = 1 << 2
)

// SlotEntry represents an entry in the slot directory
type SlotEntry struct {
	Offset uint16    // Byte offset to document data in page (0 = deleted)
	Length uint16    // Length of document in bytes
	Flags  SlotFlags // Slot flags
}

// IsDeleted returns true if the slot is marked as deleted
func (se *SlotEntry) IsDeleted() bool {
	return se.Flags&SlotFlagDeleted != 0
}

// IsOverflow returns true if the document spans multiple pages
func (se *SlotEntry) IsOverflow() bool {
	return se.Flags&SlotFlagOverflow != 0
}

// IsUpdated returns true if the slot was updated
func (se *SlotEntry) IsUpdated() bool {
	return se.Flags&SlotFlagUpdated != 0
}

// SlottedPageHeader represents the header of a slotted page
type SlottedPageHeader struct {
	SlotCount        uint16 // Number of slots in the slot directory
	FreeSpaceStart   uint16 // Offset where slot directory ends (grows down)
	FreeSpaceEnd     uint16 // Offset where document data starts (grows up)
	FragmentedSpace  uint16 // Total bytes of fragmented space (deleted slots)
	Flags            uint16 // Page-level flags
	Reserved         uint16 // Reserved for future use
}

// SlottedPage provides a slotted page structure for variable-length documents
type SlottedPage struct {
	page   *Page
	header SlottedPageHeader
	slots  []SlotEntry
}

// NewSlottedPage creates a new slotted page from an existing page
func NewSlottedPage(page *Page) (*SlottedPage, error) {
	if page.Type != PageTypeData {
		return nil, fmt.Errorf("invalid page type for slotted page: %s", page.Type)
	}

	sp := &SlottedPage{
		page: page,
	}

	// Initialize header
	sp.header.SlotCount = 0
	sp.header.FreeSpaceStart = 0
	sp.header.FreeSpaceEnd = uint16(len(page.Data))
	sp.header.FragmentedSpace = 0
	sp.header.Flags = 0
	sp.header.Reserved = 0
	sp.slots = []SlotEntry{}

	// Write initial header
	sp.serializeHeader()

	return sp, nil
}

// LoadSlottedPage loads an existing slotted page from a page
func LoadSlottedPage(page *Page) (*SlottedPage, error) {
	if page.Type != PageTypeData {
		return nil, fmt.Errorf("invalid page type for slotted page: %s", page.Type)
	}

	sp := &SlottedPage{
		page: page,
	}

	// Deserialize header
	if err := sp.deserializeHeader(); err != nil {
		return nil, fmt.Errorf("failed to deserialize header: %w", err)
	}

	// Deserialize slots
	if err := sp.deserializeSlots(); err != nil {
		return nil, fmt.Errorf("failed to deserialize slots: %w", err)
	}

	return sp, nil
}

// serializeHeader writes the header to the page data
func (sp *SlottedPage) serializeHeader() {
	binary.LittleEndian.PutUint16(sp.page.Data[0:2], sp.header.SlotCount)
	binary.LittleEndian.PutUint16(sp.page.Data[2:4], sp.header.FreeSpaceStart)
	binary.LittleEndian.PutUint16(sp.page.Data[4:6], sp.header.FreeSpaceEnd)
	binary.LittleEndian.PutUint16(sp.page.Data[6:8], sp.header.FragmentedSpace)
	binary.LittleEndian.PutUint16(sp.page.Data[8:10], sp.header.Flags)
	binary.LittleEndian.PutUint16(sp.page.Data[10:12], sp.header.Reserved)
}

// deserializeHeader reads the header from the page data
func (sp *SlottedPage) deserializeHeader() error {
	if len(sp.page.Data) < SlottedPageHeaderSize {
		return fmt.Errorf("page data too small for slotted page header")
	}

	sp.header.SlotCount = binary.LittleEndian.Uint16(sp.page.Data[0:2])
	sp.header.FreeSpaceStart = binary.LittleEndian.Uint16(sp.page.Data[2:4])
	sp.header.FreeSpaceEnd = binary.LittleEndian.Uint16(sp.page.Data[4:6])
	sp.header.FragmentedSpace = binary.LittleEndian.Uint16(sp.page.Data[6:8])
	sp.header.Flags = binary.LittleEndian.Uint16(sp.page.Data[8:10])
	sp.header.Reserved = binary.LittleEndian.Uint16(sp.page.Data[10:12])

	return nil
}

// serializeSlot writes a single slot entry to the page data
func (sp *SlottedPage) serializeSlot(slotID uint16, slot *SlotEntry) {
	offset := SlottedPageHeaderSize + int(slotID)*SlotEntrySize
	binary.LittleEndian.PutUint16(sp.page.Data[offset:offset+2], slot.Offset)
	binary.LittleEndian.PutUint16(sp.page.Data[offset+2:offset+4], slot.Length)
	sp.page.Data[offset+4] = byte(slot.Flags)
}

// deserializeSlots reads all slot entries from the page data
func (sp *SlottedPage) deserializeSlots() error {
	sp.slots = make([]SlotEntry, sp.header.SlotCount)

	for i := uint16(0); i < sp.header.SlotCount; i++ {
		offset := SlottedPageHeaderSize + int(i)*SlotEntrySize
		if offset+SlotEntrySize > len(sp.page.Data) {
			return fmt.Errorf("slot directory extends beyond page data")
		}

		sp.slots[i].Offset = binary.LittleEndian.Uint16(sp.page.Data[offset : offset+2])
		sp.slots[i].Length = binary.LittleEndian.Uint16(sp.page.Data[offset+2 : offset+4])
		sp.slots[i].Flags = SlotFlags(sp.page.Data[offset+4])
	}

	return nil
}

// InsertSlot inserts data into a new slot and returns the slot ID
func (sp *SlottedPage) InsertSlot(data []byte) (uint16, error) {
	dataLen := len(data)
	if dataLen == 0 {
		return 0, fmt.Errorf("cannot insert empty data")
	}

	// Check if we need compaction
	if sp.NeedsCompaction() {
		if err := sp.Compact(); err != nil {
			return 0, fmt.Errorf("failed to compact page: %w", err)
		}
	}

	// Calculate space needed
	spaceNeeded := uint16(dataLen)
	slotNeeded := SlotEntrySize

	// Check if we have enough contiguous free space
	contiguousFree := sp.ContiguousFreeSpace()
	if int(contiguousFree) < int(spaceNeeded)+slotNeeded {
		return 0, fmt.Errorf("insufficient space: need %d bytes, have %d bytes", spaceNeeded+uint16(slotNeeded), contiguousFree)
	}

	// Allocate slot
	slotID := sp.header.SlotCount
	sp.header.SlotCount++

	// Update free space start (slot directory grows down)
	sp.header.FreeSpaceStart = SlottedPageHeaderSize + sp.header.SlotCount*SlotEntrySize

	// Allocate space for data (data grows up from bottom)
	dataOffset := sp.header.FreeSpaceEnd - spaceNeeded
	sp.header.FreeSpaceEnd = dataOffset

	// Create slot entry
	slot := SlotEntry{
		Offset: dataOffset,
		Length: uint16(dataLen),
		Flags:  0,
	}

	// Add slot to in-memory array
	sp.slots = append(sp.slots, slot)

	// Write data to page
	copy(sp.page.Data[dataOffset:dataOffset+spaceNeeded], data)

	// Serialize slot and header
	sp.serializeSlot(slotID, &slot)
	sp.serializeHeader()

	// Mark page as dirty
	sp.page.MarkDirty()

	return slotID, nil
}

// GetSlot retrieves data from a slot
func (sp *SlottedPage) GetSlot(slotID uint16) ([]byte, error) {
	if slotID >= sp.header.SlotCount {
		return nil, fmt.Errorf("invalid slot ID: %d (max: %d)", slotID, sp.header.SlotCount-1)
	}

	slot := &sp.slots[slotID]

	if slot.IsDeleted() {
		return nil, fmt.Errorf("slot %d is deleted", slotID)
	}

	// Read data from page
	data := make([]byte, slot.Length)
	copy(data, sp.page.Data[slot.Offset:slot.Offset+slot.Length])

	return data, nil
}

// UpdateSlot updates data in an existing slot
func (sp *SlottedPage) UpdateSlot(slotID uint16, data []byte) error {
	if slotID >= sp.header.SlotCount {
		return fmt.Errorf("invalid slot ID: %d (max: %d)", slotID, sp.header.SlotCount-1)
	}

	slot := &sp.slots[slotID]

	if slot.IsDeleted() {
		return fmt.Errorf("slot %d is deleted", slotID)
	}

	dataLen := uint16(len(data))

	// Check if new data fits in existing slot
	if dataLen <= slot.Length {
		// In-place update
		copy(sp.page.Data[slot.Offset:slot.Offset+dataLen], data)

		// Update fragmented space if new data is smaller
		if dataLen < slot.Length {
			sp.header.FragmentedSpace += slot.Length - dataLen
			slot.Length = dataLen
		}

		sp.serializeSlot(slotID, slot)
		sp.serializeHeader()
		sp.page.MarkDirty()
		return nil
	}

	// Data doesn't fit, need to allocate new space
	// Mark old space as fragmented
	sp.header.FragmentedSpace += slot.Length

	// Check if we have enough space after compaction
	totalFree := sp.TotalFreeSpace()
	if dataLen > totalFree {
		return fmt.Errorf("insufficient space for update: need %d bytes, have %d bytes", dataLen, totalFree)
	}

	// Try compaction if needed
	if sp.NeedsCompaction() {
		if err := sp.Compact(); err != nil {
			return fmt.Errorf("failed to compact page: %w", err)
		}
	}

	// Allocate new space
	contiguousFree := sp.ContiguousFreeSpace()
	if dataLen > contiguousFree {
		return fmt.Errorf("insufficient contiguous space after compaction: need %d bytes, have %d bytes", dataLen, contiguousFree)
	}

	// Allocate space for new data
	dataOffset := sp.header.FreeSpaceEnd - dataLen
	sp.header.FreeSpaceEnd = dataOffset

	// Write data
	copy(sp.page.Data[dataOffset:dataOffset+dataLen], data)

	// Update slot
	slot.Offset = dataOffset
	slot.Length = dataLen
	slot.Flags |= SlotFlagUpdated

	sp.serializeSlot(slotID, slot)
	sp.serializeHeader()
	sp.page.MarkDirty()

	return nil
}

// DeleteSlot marks a slot as deleted
func (sp *SlottedPage) DeleteSlot(slotID uint16) error {
	if slotID >= sp.header.SlotCount {
		return fmt.Errorf("invalid slot ID: %d (max: %d)", slotID, sp.header.SlotCount-1)
	}

	slot := &sp.slots[slotID]

	if slot.IsDeleted() {
		return fmt.Errorf("slot %d is already deleted", slotID)
	}

	// Mark as deleted
	slot.Flags |= SlotFlagDeleted

	// Add to fragmented space
	sp.header.FragmentedSpace += slot.Length

	// Zero out offset to indicate deletion
	slot.Offset = 0

	sp.serializeSlot(slotID, slot)
	sp.serializeHeader()
	sp.page.MarkDirty()

	return nil
}

// ContiguousFreeSpace returns the contiguous free space between slot directory and data
func (sp *SlottedPage) ContiguousFreeSpace() uint16 {
	if sp.header.FreeSpaceEnd < sp.header.FreeSpaceStart {
		return 0
	}
	return sp.header.FreeSpaceEnd - sp.header.FreeSpaceStart
}

// TotalFreeSpace returns total free space (contiguous + fragmented)
func (sp *SlottedPage) TotalFreeSpace() uint16 {
	return sp.ContiguousFreeSpace() + sp.header.FragmentedSpace
}

// NeedsCompaction returns true if the page needs compaction
func (sp *SlottedPage) NeedsCompaction() bool {
	if sp.header.FragmentedSpace == 0 {
		return false
	}
	pageSize := float64(SlottedPageAvailableSpace)
	fragmented := float64(sp.header.FragmentedSpace)
	return fragmented/pageSize > PageCompactionThreshold
}

// Compact defragments the page by removing deleted slots and reorganizing data
func (sp *SlottedPage) Compact() error {
	// Create new slots array (only active slots)
	newSlots := []SlotEntry{}
	newSlotMapping := make(map[uint16]uint16) // old slot ID -> new slot ID

	// Collect active slots
	for i := uint16(0); i < sp.header.SlotCount; i++ {
		if !sp.slots[i].IsDeleted() {
			newSlotID := uint16(len(newSlots))
			newSlotMapping[i] = newSlotID
			newSlots = append(newSlots, sp.slots[i])
		}
	}

	// If no active slots, reset page
	if len(newSlots) == 0 {
		sp.header.SlotCount = 0
		sp.header.FreeSpaceStart = SlottedPageHeaderSize
		sp.header.FreeSpaceEnd = uint16(len(sp.page.Data))
		sp.header.FragmentedSpace = 0
		sp.slots = []SlotEntry{}
		sp.serializeHeader()
		sp.page.MarkDirty()
		return nil
	}

	// Create temporary buffer for reorganized data
	tempData := make([]byte, len(sp.page.Data))

	// Copy data in reverse order (from end of page upward)
	newDataEnd := uint16(len(sp.page.Data))
	for i := len(newSlots) - 1; i >= 0; i-- {
		slot := &newSlots[i]

		// Skip deleted slots (shouldn't happen but be safe)
		if slot.IsDeleted() {
			continue
		}

		// Read old data
		oldData := sp.page.Data[slot.Offset : slot.Offset+slot.Length]

		// Write to new position
		newOffset := newDataEnd - slot.Length
		copy(tempData[newOffset:newDataEnd], oldData)

		// Update slot offset
		slot.Offset = newOffset
		newDataEnd = newOffset
	}

	// Copy reorganized data back to page
	copy(sp.page.Data, tempData)

	// Update header
	sp.header.SlotCount = uint16(len(newSlots))
	sp.header.FreeSpaceStart = SlottedPageHeaderSize + sp.header.SlotCount*SlotEntrySize
	sp.header.FreeSpaceEnd = newDataEnd
	sp.header.FragmentedSpace = 0

	// Update slots
	sp.slots = newSlots

	// Serialize everything
	sp.serializeHeader()
	for i := uint16(0); i < sp.header.SlotCount; i++ {
		sp.serializeSlot(i, &sp.slots[i])
	}

	sp.page.MarkDirty()

	return nil
}

// SlotCount returns the number of slots
func (sp *SlottedPage) SlotCount() uint16 {
	return sp.header.SlotCount
}

// FragmentedSpace returns the fragmented space
func (sp *SlottedPage) FragmentedSpace() uint16 {
	return sp.header.FragmentedSpace
}

// GetPage returns the underlying page
func (sp *SlottedPage) GetPage() *Page {
	return sp.page
}

// Stats returns statistics about the slotted page
func (sp *SlottedPage) Stats() map[string]interface{} {
	activeSlots := 0
	deletedSlots := 0
	for i := uint16(0); i < sp.header.SlotCount; i++ {
		if sp.slots[i].IsDeleted() {
			deletedSlots++
		} else {
			activeSlots++
		}
	}

	return map[string]interface{}{
		"slot_count":         sp.header.SlotCount,
		"active_slots":       activeSlots,
		"deleted_slots":      deletedSlots,
		"free_space_start":   sp.header.FreeSpaceStart,
		"free_space_end":     sp.header.FreeSpaceEnd,
		"contiguous_free":    sp.ContiguousFreeSpace(),
		"fragmented_space":   sp.header.FragmentedSpace,
		"total_free":         sp.TotalFreeSpace(),
		"needs_compaction":   sp.NeedsCompaction(),
	}
}
