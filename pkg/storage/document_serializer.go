package storage

import (
	"fmt"

	"github.com/mnohosten/laura-db/pkg/document"
)

const (
	// MaxDocumentSize is the maximum size of a document (16MB, MongoDB-compatible)
	MaxDocumentSize = 16 * 1024 * 1024

	// MaxSinglePageDocumentSize is the maximum size that can fit in a single page
	// Accounts for page header (16 bytes), slotted page header (12 bytes), and slot entry (5 bytes)
	MaxSinglePageDocumentSize = PageSize - PageHeaderSize - SlottedPageHeaderSize - SlotEntrySize
)

// DocumentSerializer provides high-level document serialization/deserialization for disk storage
type DocumentSerializer struct {
	encoder *document.Encoder
}

// NewDocumentSerializer creates a new document serializer
func NewDocumentSerializer() *DocumentSerializer {
	return &DocumentSerializer{
		encoder: document.NewEncoder(),
	}
}

// SerializeDocument serializes a document to BSON format for disk storage
// Returns the serialized bytes and any error encountered
func (ds *DocumentSerializer) SerializeDocument(doc *document.Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("cannot serialize nil document")
	}

	// Encode document to BSON
	data, err := ds.encoder.Encode(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to encode document: %w", err)
	}

	// Validate size
	if len(data) > MaxDocumentSize {
		return nil, fmt.Errorf("document size %d exceeds maximum %d bytes", len(data), MaxDocumentSize)
	}

	return data, nil
}

// DeserializeDocument deserializes a document from BSON format
// Returns the deserialized document and any error encountered
func (ds *DocumentSerializer) DeserializeDocument(data []byte) (*document.Document, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("cannot deserialize empty data")
	}

	// Validate size
	if len(data) > MaxDocumentSize {
		return nil, fmt.Errorf("document data size %d exceeds maximum %d bytes", len(data), MaxDocumentSize)
	}

	// Decode BSON to document
	decoder := document.NewDecoder(data)
	doc, err := decoder.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode document: %w", err)
	}

	return doc, nil
}

// SerializeDocumentFromMap is a convenience method that converts a map to a document and serializes it
func (ds *DocumentSerializer) SerializeDocumentFromMap(m map[string]interface{}) ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf("cannot serialize nil map")
	}

	doc := document.NewDocumentFromMap(m)
	return ds.SerializeDocument(doc)
}

// DeserializeDocumentToMap deserializes a document and converts it to a map
func (ds *DocumentSerializer) DeserializeDocumentToMap(data []byte) (map[string]interface{}, error) {
	doc, err := ds.DeserializeDocument(data)
	if err != nil {
		return nil, err
	}

	return doc.ToMap(), nil
}

// EstimateDocumentSize estimates the serialized size of a document without actually encoding it
// This is useful for checking if a document will fit in a page before attempting insertion
// Note: This is an approximation and may not be exact
func (ds *DocumentSerializer) EstimateDocumentSize(doc *document.Document) int {
	if doc == nil {
		return 0
	}

	// Try to encode and return actual size
	// In a production system, we might use a more efficient estimation algorithm
	data, err := ds.encoder.Encode(doc)
	if err != nil {
		// Return a large estimate to be safe
		return MaxDocumentSize
	}

	return len(data)
}

// CanFitInSinglePage checks if a document can fit in a single page
func (ds *DocumentSerializer) CanFitInSinglePage(doc *document.Document) (bool, error) {
	data, err := ds.SerializeDocument(doc)
	if err != nil {
		return false, err
	}

	return len(data) <= MaxSinglePageDocumentSize, nil
}

// DocumentPageManager provides high-level operations for storing documents in slotted pages
type DocumentPageManager struct {
	serializer *DocumentSerializer
}

// NewDocumentPageManager creates a new document page manager
func NewDocumentPageManager() *DocumentPageManager {
	return &DocumentPageManager{
		serializer: NewDocumentSerializer(),
	}
}

// InsertDocument inserts a document into a slotted page and returns the slot ID
func (dpm *DocumentPageManager) InsertDocument(page *SlottedPage, doc *document.Document) (uint16, error) {
	if page == nil {
		return 0, fmt.Errorf("cannot insert into nil page")
	}

	if doc == nil {
		return 0, fmt.Errorf("cannot insert nil document")
	}

	// Serialize document
	data, err := dpm.serializer.SerializeDocument(doc)
	if err != nil {
		return 0, fmt.Errorf("failed to serialize document: %w", err)
	}

	// Check if document can fit in a single page
	if len(data) > MaxSinglePageDocumentSize {
		return 0, fmt.Errorf("document size %d bytes exceeds single page limit %d bytes (overflow pages not yet implemented)",
			len(data), MaxSinglePageDocumentSize)
	}

	// Insert into slotted page
	slotID, err := page.InsertSlot(data)
	if err != nil {
		return 0, fmt.Errorf("failed to insert document into page: %w", err)
	}

	return slotID, nil
}

// GetDocument retrieves a document from a slotted page by slot ID
func (dpm *DocumentPageManager) GetDocument(page *SlottedPage, slotID uint16) (*document.Document, error) {
	if page == nil {
		return nil, fmt.Errorf("cannot get from nil page")
	}

	// Get data from slot
	data, err := page.GetSlot(slotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slot %d: %w", slotID, err)
	}

	// Deserialize document
	doc, err := dpm.serializer.DeserializeDocument(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize document from slot %d: %w", slotID, err)
	}

	return doc, nil
}

// UpdateDocument updates a document in a slotted page
func (dpm *DocumentPageManager) UpdateDocument(page *SlottedPage, slotID uint16, doc *document.Document) error {
	if page == nil {
		return fmt.Errorf("cannot update in nil page")
	}

	if doc == nil {
		return fmt.Errorf("cannot update with nil document")
	}

	// Serialize document
	data, err := dpm.serializer.SerializeDocument(doc)
	if err != nil {
		return fmt.Errorf("failed to serialize document: %w", err)
	}

	// Check if document can fit in a single page
	if len(data) > MaxSinglePageDocumentSize {
		return fmt.Errorf("document size %d bytes exceeds single page limit %d bytes (overflow pages not yet implemented)",
			len(data), MaxSinglePageDocumentSize)
	}

	// Update slot in page
	if err := page.UpdateSlot(slotID, data); err != nil {
		return fmt.Errorf("failed to update slot %d: %w", slotID, err)
	}

	return nil
}

// DeleteDocument marks a document as deleted in a slotted page
func (dpm *DocumentPageManager) DeleteDocument(page *SlottedPage, slotID uint16) error {
	if page == nil {
		return fmt.Errorf("cannot delete from nil page")
	}

	// Delete slot
	if err := page.DeleteSlot(slotID); err != nil {
		return fmt.Errorf("failed to delete slot %d: %w", slotID, err)
	}

	return nil
}

// InsertDocumentFromMap is a convenience method that inserts a document from a map
func (dpm *DocumentPageManager) InsertDocumentFromMap(page *SlottedPage, m map[string]interface{}) (uint16, error) {
	doc := document.NewDocumentFromMap(m)
	return dpm.InsertDocument(page, doc)
}

// GetDocumentAsMap retrieves a document and returns it as a map
func (dpm *DocumentPageManager) GetDocumentAsMap(page *SlottedPage, slotID uint16) (map[string]interface{}, error) {
	doc, err := dpm.GetDocument(page, slotID)
	if err != nil {
		return nil, err
	}

	return doc.ToMap(), nil
}

// UpdateDocumentFromMap updates a document from a map
func (dpm *DocumentPageManager) UpdateDocumentFromMap(page *SlottedPage, slotID uint16, m map[string]interface{}) error {
	doc := document.NewDocumentFromMap(m)
	return dpm.UpdateDocument(page, slotID, doc)
}

// GetPageCapacity returns information about how many documents can fit in a page
type PageCapacity struct {
	TotalSpace         int // Total available space in bytes
	ContiguousFreeSpace int // Contiguous free space available
	FragmentedSpace    int // Fragmented space from deletions
	EstimatedSlots     int // Estimated number of average-sized documents that can fit
	CurrentSlotCount   int // Current number of slots (active + deleted)
	ActiveSlotCount    int // Number of active slots
}

// GetPageCapacity analyzes a page and returns capacity information
func (dpm *DocumentPageManager) GetPageCapacity(page *SlottedPage) PageCapacity {
	if page == nil {
		return PageCapacity{}
	}

	stats := page.Stats()

	contiguousFree := int(page.ContiguousFreeSpace())
	fragmentedSpace := int(page.FragmentedSpace())
	totalFree := int(page.TotalFreeSpace())

	// Estimate how many average-sized documents can fit
	// Assuming an average document size of 256 bytes + slot overhead
	avgDocSize := 256 + SlotEntrySize
	estimatedSlots := totalFree / avgDocSize

	return PageCapacity{
		TotalSpace:         totalFree,
		ContiguousFreeSpace: contiguousFree,
		FragmentedSpace:    fragmentedSpace,
		EstimatedSlots:     estimatedSlots,
		CurrentSlotCount:   int(stats["slot_count"].(uint16)),
		ActiveSlotCount:    stats["active_slots"].(int),
	}
}
