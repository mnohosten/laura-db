package database

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/mnohosten/laura-db/pkg/storage"
)

// Magic number for LauraDB file format validation: "LAUR"
const CatalogMagicNumber uint32 = 0x4C415552

// Catalog schema version
const CatalogVersion uint16 = 1

// Catalog page is always page 0
const CatalogPageID storage.PageID = 0

// CatalogHeader represents the header of the collection catalog (32 bytes)
type CatalogHeader struct {
	MagicNumber          uint32 // 0x4C415552 ("LAUR")
	Version              uint16 // Schema version (1)
	CollectionCount      uint32 // Total number of collections
	NextCollectionID     uint32 // Next available collection ID
	FirstMetadataPageID  uint32 // First page in metadata page chain
	FreeMetadataPageID   uint32 // First free metadata page (for reuse)
	LastCheckpointTxnID  uint64 // Transaction ID of last checkpoint
	Reserved             uint16 // Reserved for future use
}

// CollectionDirectoryEntry represents an entry in the collection directory
type CollectionDirectoryEntry struct {
	CollectionID   uint32
	Name           string
	MetadataPageID storage.PageID
	Flags          uint16 // Bit 0: IsActive, Bit 1: IsSystem
}

// Flag bits for CollectionDirectoryEntry
const (
	CollectionFlagActive = 1 << 0 // 0x01
	CollectionFlagSystem = 1 << 1 // 0x02
)

// CollectionCatalog manages the central registry of all collections
type CollectionCatalog struct {
	diskMgr     *storage.DiskManager
	header      *CatalogHeader
	collections map[string]*CollectionDirectoryEntry // name -> entry
	mu          sync.RWMutex
}

// NewCollectionCatalog creates a new collection catalog
func NewCollectionCatalog(diskMgr *storage.DiskManager) (*CollectionCatalog, error) {
	catalog := &CollectionCatalog{
		diskMgr:     diskMgr,
		collections: make(map[string]*CollectionDirectoryEntry),
	}

	// Try to load existing catalog
	if err := catalog.load(); err != nil {
		// If load fails, initialize new catalog
		if err := catalog.initialize(); err != nil {
			return nil, fmt.Errorf("failed to initialize catalog: %w", err)
		}
	}

	return catalog, nil
}

// initialize creates a new catalog page
func (c *CollectionCatalog) initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create catalog header
	c.header = &CatalogHeader{
		MagicNumber:          CatalogMagicNumber,
		Version:              CatalogVersion,
		CollectionCount:      0,
		NextCollectionID:     1, // Start from 1 (0 is reserved)
		FirstMetadataPageID:  0,
		FreeMetadataPageID:   0,
		LastCheckpointTxnID:  0,
		Reserved:             0,
	}

	// Save catalog to disk
	return c.saveUnlocked()
}

// load reads the catalog from disk
func (c *CollectionCatalog) load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Read catalog page
	page, err := c.diskMgr.ReadPage(CatalogPageID)
	if err != nil {
		return fmt.Errorf("failed to read catalog page: %w", err)
	}

	// Parse catalog header (starts after page header)
	offset := storage.PageHeaderSize
	data := page.Data

	if len(data) < offset+32 {
		return fmt.Errorf("catalog page too short")
	}

	// Read header
	c.header = &CatalogHeader{}
	c.header.MagicNumber = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	// Validate magic number
	if c.header.MagicNumber != CatalogMagicNumber {
		return fmt.Errorf("invalid magic number: expected 0x%X, got 0x%X", CatalogMagicNumber, c.header.MagicNumber)
	}

	c.header.Version = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2
	c.header.CollectionCount = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	c.header.NextCollectionID = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	c.header.FirstMetadataPageID = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	c.header.FreeMetadataPageID = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	c.header.LastCheckpointTxnID = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	c.header.Reserved = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2

	// Read collection directory entries
	c.collections = make(map[string]*CollectionDirectoryEntry)
	for i := uint32(0); i < c.header.CollectionCount; i++ {
		if offset+12 > len(data) {
			return fmt.Errorf("unexpected end of catalog data")
		}

		entry := &CollectionDirectoryEntry{}
		entry.CollectionID = binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4

		nameLen := binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2

		if offset+int(nameLen) > len(data) {
			return fmt.Errorf("unexpected end of catalog data reading name")
		}
		entry.Name = string(data[offset : offset+int(nameLen)])
		offset += int(nameLen)

		if offset+6 > len(data) {
			return fmt.Errorf("unexpected end of catalog data reading metadata page ID")
		}
		entry.MetadataPageID = storage.PageID(binary.LittleEndian.Uint32(data[offset : offset+4]))
		offset += 4

		entry.Flags = binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2

		// Only add active collections
		if entry.Flags&CollectionFlagActive != 0 {
			c.collections[entry.Name] = entry
		}
	}

	return nil
}

// saveUnlocked writes the catalog to disk (caller must hold lock)
func (c *CollectionCatalog) saveUnlocked() error {
	// Create page
	page := storage.NewPage(CatalogPageID, storage.PageTypeData)
	offset := storage.PageHeaderSize

	// Write header
	binary.LittleEndian.PutUint32(page.Data[offset:offset+4], c.header.MagicNumber)
	offset += 4
	binary.LittleEndian.PutUint16(page.Data[offset:offset+2], c.header.Version)
	offset += 2
	binary.LittleEndian.PutUint32(page.Data[offset:offset+4], c.header.CollectionCount)
	offset += 4
	binary.LittleEndian.PutUint32(page.Data[offset:offset+4], c.header.NextCollectionID)
	offset += 4
	binary.LittleEndian.PutUint32(page.Data[offset:offset+4], c.header.FirstMetadataPageID)
	offset += 4
	binary.LittleEndian.PutUint32(page.Data[offset:offset+4], c.header.FreeMetadataPageID)
	offset += 4
	binary.LittleEndian.PutUint64(page.Data[offset:offset+8], c.header.LastCheckpointTxnID)
	offset += 8
	binary.LittleEndian.PutUint16(page.Data[offset:offset+2], c.header.Reserved)
	offset += 2

	// Write collection directory entries
	for _, entry := range c.collections {
		// Check if we have enough space
		entrySize := 12 + len(entry.Name)
		if offset+entrySize > storage.PageSize {
			return fmt.Errorf("catalog page full")
		}

		binary.LittleEndian.PutUint32(page.Data[offset:offset+4], entry.CollectionID)
		offset += 4

		binary.LittleEndian.PutUint16(page.Data[offset:offset+2], uint16(len(entry.Name)))
		offset += 2

		copy(page.Data[offset:], []byte(entry.Name))
		offset += len(entry.Name)

		binary.LittleEndian.PutUint32(page.Data[offset:offset+4], uint32(entry.MetadataPageID))
		offset += 4

		binary.LittleEndian.PutUint16(page.Data[offset:offset+2], entry.Flags)
		offset += 2
	}

	// Write page to disk
	if err := c.diskMgr.WritePage(page); err != nil {
		return fmt.Errorf("failed to write catalog page: %w", err)
	}

	return nil
}

// RegisterCollection adds a new collection to the catalog
func (c *CollectionCatalog) RegisterCollection(name string, metadataPageID storage.PageID) (uint32, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if collection already exists
	if _, exists := c.collections[name]; exists {
		return 0, fmt.Errorf("collection %s already exists", name)
	}

	// Validate collection name
	if err := validateCollectionName(name); err != nil {
		return 0, err
	}

	// Assign collection ID
	collectionID := c.header.NextCollectionID
	c.header.NextCollectionID++

	// Create directory entry
	entry := &CollectionDirectoryEntry{
		CollectionID:   collectionID,
		Name:           name,
		MetadataPageID: metadataPageID,
		Flags:          CollectionFlagActive, // Set active flag
	}

	// Add to in-memory map
	c.collections[name] = entry
	c.header.CollectionCount++

	// Save catalog to disk
	if err := c.saveUnlocked(); err != nil {
		// Rollback in-memory changes
		delete(c.collections, name)
		c.header.CollectionCount--
		c.header.NextCollectionID--
		return 0, fmt.Errorf("failed to save catalog: %w", err)
	}

	return collectionID, nil
}

// GetCollection retrieves a collection entry by name
func (c *CollectionCatalog) GetCollection(name string) (*CollectionDirectoryEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.collections[name]
	if !exists {
		return nil, fmt.Errorf("collection %s not found", name)
	}

	return entry, nil
}

// ListCollections returns all collection names
func (c *CollectionCatalog) ListCollections() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.collections))
	for name := range c.collections {
		names = append(names, name)
	}
	return names
}

// DropCollection removes a collection from the catalog
func (c *CollectionCatalog) DropCollection(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if collection exists
	entry, exists := c.collections[name]
	if !exists {
		return fmt.Errorf("collection %s not found", name)
	}

	// Mark as inactive (soft delete)
	entry.Flags &= ^uint16(CollectionFlagActive) // Clear active flag
	delete(c.collections, name)
	c.header.CollectionCount--

	// Save catalog to disk
	if err := c.saveUnlocked(); err != nil {
		// Rollback in-memory changes
		entry.Flags |= CollectionFlagActive
		c.collections[name] = entry
		c.header.CollectionCount++
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	return nil
}

// GetCollectionCount returns the number of active collections
func (c *CollectionCatalog) GetCollectionCount() uint32 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.header.CollectionCount
}

// validateCollectionName checks if a collection name is valid
func validateCollectionName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("collection name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("collection name too long (max 255 characters)")
	}

	// Check for valid characters (alphanumeric, underscore, dash)
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			return fmt.Errorf("invalid character '%c' in collection name", ch)
		}
	}

	// Cannot start with "system."
	if len(name) >= 7 && name[:7] == "system." {
		return fmt.Errorf("collection name cannot start with 'system.' (reserved)")
	}

	return nil
}
