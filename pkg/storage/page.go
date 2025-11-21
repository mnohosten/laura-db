package storage

import (
	"encoding/binary"
	"fmt"
)

const (
	// PageSize is the size of each page (4KB, typical OS page size)
	PageSize = 4096

	// PageHeaderSize is the size of the page header
	PageHeaderSize = 16
)

// PageType represents the type of page
type PageType uint8

const (
	PageTypeData PageType = iota
	PageTypeIndex
	PageTypeFreeList
	PageTypeOverflow
)

// PageID is a unique identifier for a page
type PageID uint32

// Page represents a fixed-size block of data
type Page struct {
	ID       PageID
	Type     PageType
	Flags    uint8
	LSN      uint64 // Log Sequence Number for recovery
	Data     []byte
	IsDirty  bool
	PinCount int
}

// NewPage creates a new page
func NewPage(id PageID, pageType PageType) *Page {
	return &Page{
		ID:       id,
		Type:     pageType,
		Flags:    0,
		LSN:      0,
		Data:     make([]byte, PageSize-PageHeaderSize),
		IsDirty:  false,
		PinCount: 0,
	}
}

// Serialize converts the page to bytes for storage
func (p *Page) Serialize() []byte {
	buf := make([]byte, PageSize)

	// Header: [4-byte ID][1-byte Type][1-byte Flags][8-byte LSN][2-byte reserved]
	binary.LittleEndian.PutUint32(buf[0:4], uint32(p.ID))
	buf[4] = byte(p.Type)
	buf[5] = p.Flags
	binary.LittleEndian.PutUint64(buf[6:14], p.LSN)
	// bytes 14-16 reserved

	// Data
	copy(buf[PageHeaderSize:], p.Data)

	return buf
}

// Deserialize loads page data from bytes
func (p *Page) Deserialize(data []byte) error {
	if len(data) != PageSize {
		return fmt.Errorf("invalid page size: expected %d, got %d", PageSize, len(data))
	}

	// Read header
	p.ID = PageID(binary.LittleEndian.Uint32(data[0:4]))
	p.Type = PageType(data[4])
	p.Flags = data[5]
	p.LSN = binary.LittleEndian.Uint64(data[6:14])

	// Read data
	p.Data = make([]byte, PageSize-PageHeaderSize)
	copy(p.Data, data[PageHeaderSize:])

	return nil
}

// Pin increments the pin count (page is in use)
func (p *Page) Pin() {
	p.PinCount++
}

// Unpin decrements the pin count
func (p *Page) Unpin() {
	if p.PinCount > 0 {
		p.PinCount--
	}
}

// IsPinned returns true if the page is pinned
func (p *Page) IsPinned() bool {
	return p.PinCount > 0
}

// MarkDirty marks the page as modified
func (p *Page) MarkDirty() {
	p.IsDirty = true
}

// FreeSpace returns the amount of free space in the page
func (p *Page) FreeSpace() int {
	// Simple implementation - in practice would track used space
	return len(p.Data)
}
