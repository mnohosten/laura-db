package storage

import (
	"encoding/binary"
	"fmt"
)

const (
	// FreePageHeaderSize is the size of the free page header
	FreePageHeaderSize = 8

	// MaxFreePageEntries is the maximum number of free page IDs that can be stored in a single free list page
	// (PageSize - PageHeaderSize - FreePageHeaderSize) / 4 bytes per PageID
	MaxFreePageEntries = (PageSize - PageHeaderSize - FreePageHeaderSize) / 4
)

// FreePageHeader represents the header of a free page list page
type FreePageHeader struct {
	NextFreeListPage PageID // Next page in the free list chain (0 = end of chain)
	EntryCount       uint32 // Number of free page entries in this page
}

// FreePageList manages the list of free pages available for reuse
type FreePageList struct {
	HeadPageID PageID // First free list page (0 = no free pages)
	PageCount  uint32 // Total number of free pages across all free list pages
}

// NewFreePageList creates a new free page list
func NewFreePageList() *FreePageList {
	return &FreePageList{
		HeadPageID: 0,
		PageCount:  0,
	}
}

// SerializeFreePageHeader writes the free page header to a page
func SerializeFreePageHeader(page *Page, header *FreePageHeader) {
	if page.Type != PageTypeFreeList {
		return
	}
	binary.LittleEndian.PutUint32(page.Data[0:4], uint32(header.NextFreeListPage))
	binary.LittleEndian.PutUint32(page.Data[4:8], header.EntryCount)
}

// DeserializeFreePageHeader reads the free page header from a page
func DeserializeFreePageHeader(page *Page) (*FreePageHeader, error) {
	if page.Type != PageTypeFreeList {
		return nil, fmt.Errorf("invalid page type for free page: %s", page.Type)
	}
	if len(page.Data) < FreePageHeaderSize {
		return nil, fmt.Errorf("page data too small for free page header")
	}

	header := &FreePageHeader{
		NextFreeListPage: PageID(binary.LittleEndian.Uint32(page.Data[0:4])),
		EntryCount:       binary.LittleEndian.Uint32(page.Data[4:8]),
	}
	return header, nil
}

// WriteFreePageEntry writes a free page ID at the specified index
func WriteFreePageEntry(page *Page, index uint32, pageID PageID) error {
	if page.Type != PageTypeFreeList {
		return fmt.Errorf("invalid page type for free page: %s", page.Type)
	}
	if index >= MaxFreePageEntries {
		return fmt.Errorf("free page entry index %d exceeds maximum %d", index, MaxFreePageEntries)
	}

	offset := FreePageHeaderSize + int(index)*4
	if offset+4 > len(page.Data) {
		return fmt.Errorf("offset %d exceeds page data size", offset)
	}

	binary.LittleEndian.PutUint32(page.Data[offset:offset+4], uint32(pageID))
	return nil
}

// ReadFreePageEntry reads a free page ID at the specified index
func ReadFreePageEntry(page *Page, index uint32) (PageID, error) {
	if page.Type != PageTypeFreeList {
		return 0, fmt.Errorf("invalid page type for free page: %s", page.Type)
	}
	if index >= MaxFreePageEntries {
		return 0, fmt.Errorf("free page entry index %d exceeds maximum %d", index, MaxFreePageEntries)
	}

	offset := FreePageHeaderSize + int(index)*4
	if offset+4 > len(page.Data) {
		return 0, fmt.Errorf("offset %d exceeds page data size", offset)
	}

	pageID := PageID(binary.LittleEndian.Uint32(page.Data[offset : offset+4]))
	return pageID, nil
}

// ReadAllFreePageEntries reads all free page IDs from a free list page
func ReadAllFreePageEntries(page *Page) ([]PageID, error) {
	header, err := DeserializeFreePageHeader(page)
	if err != nil {
		return nil, err
	}

	entries := make([]PageID, 0, header.EntryCount)
	for i := uint32(0); i < header.EntryCount; i++ {
		pageID, err := ReadFreePageEntry(page, i)
		if err != nil {
			return nil, err
		}
		entries = append(entries, pageID)
	}

	return entries, nil
}

// InitializeFreeListPage initializes a page as a free list page
func InitializeFreeListPage(page *Page) {
	page.Type = PageTypeFreeList
	header := &FreePageHeader{
		NextFreeListPage: 0,
		EntryCount:       0,
	}
	SerializeFreePageHeader(page, header)
	page.MarkDirty()
}

// AddFreePageToList adds a free page ID to a free list page
// Returns true if the page was added, false if the page is full
func AddFreePageToList(page *Page, pageID PageID) (bool, error) {
	header, err := DeserializeFreePageHeader(page)
	if err != nil {
		return false, err
	}

	if header.EntryCount >= MaxFreePageEntries {
		return false, nil // Page is full
	}

	if err := WriteFreePageEntry(page, header.EntryCount, pageID); err != nil {
		return false, err
	}

	header.EntryCount++
	SerializeFreePageHeader(page, header)
	page.MarkDirty()

	return true, nil
}

// RemoveFreePageFromList removes the last free page ID from a free list page
// Returns the removed page ID and true if successful, 0 and false if the page is empty
func RemoveFreePageFromList(page *Page) (PageID, bool, error) {
	header, err := DeserializeFreePageHeader(page)
	if err != nil {
		return 0, false, err
	}

	if header.EntryCount == 0 {
		return 0, false, nil // Page is empty
	}

	header.EntryCount--
	pageID, err := ReadFreePageEntry(page, header.EntryCount)
	if err != nil {
		return 0, false, err
	}

	SerializeFreePageHeader(page, header)
	page.MarkDirty()

	return pageID, true, nil
}

// IsFreeListPageFull returns true if the free list page is full
func IsFreeListPageFull(page *Page) (bool, error) {
	header, err := DeserializeFreePageHeader(page)
	if err != nil {
		return false, err
	}
	return header.EntryCount >= MaxFreePageEntries, nil
}

// IsFreeListPageEmpty returns true if the free list page is empty
func IsFreeListPageEmpty(page *Page) (bool, error) {
	header, err := DeserializeFreePageHeader(page)
	if err != nil {
		return false, err
	}
	return header.EntryCount == 0, nil
}
