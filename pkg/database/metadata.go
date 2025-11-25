package database

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/mnohosten/laura-db/pkg/storage"
)

// CollectionMetadata represents the persistent metadata for a collection
type CollectionMetadata struct {
	CollectionID        uint32
	Name                string
	CreatedTimestamp    time.Time
	DocumentCount       uint64
	DataSizeBytes       uint64
	IndexCount          uint16
	FirstDataPageID     storage.PageID
	StatisticsPageID    storage.PageID
	Schema              *CollectionSchema  // Optional schema validation
	Options             *CollectionOptions // Collection options
}

// CollectionSchema represents optional schema validation for a collection
type CollectionSchema struct {
	Version            uint16
	SchemaJSON         string // JSON Schema format
	ValidationLevel    string // "off", "moderate", "strict"
	ValidationAction   string // "error", "warn"
}

// CollectionOptions represents collection-level configuration
type CollectionOptions struct {
	Capped          bool
	MaxSize         int64
	MaxDocuments    int64
}

// IndexMetadata represents the persistent metadata for an index
type IndexMetadata struct {
	IndexID            uint32
	CollectionID       uint32
	Name               string
	IndexType          IndexType
	FieldPaths         []string
	IsUnique           bool
	IsSparse           bool
	IsPartial          bool
	PartialFilter      string // JSON-encoded filter expression
	CreatedTimestamp   time.Time
	RootPageID         storage.PageID // Root page of B+ tree (for future disk-based indexes)
	EntryCount         uint64
	Order              uint16 // B+ tree order
	Options            map[string]interface{} // Index-specific options (e.g., TTL seconds, text weights)
}

// IndexType represents the type of index
type IndexType uint8

const (
	IndexTypeBTree     IndexType = 0
	IndexTypeHash      IndexType = 1
	IndexTypeText      IndexType = 2
	IndexType2D        IndexType = 3
	IndexType2DSphere  IndexType = 4
)

// SerializeCollectionMetadata serializes collection metadata to bytes
func SerializeCollectionMetadata(meta *CollectionMetadata) ([]byte, error) {
	// Estimate buffer size: fixed fields + variable length strings
	bufSize := 64 + len(meta.Name) + 1024 // Room for schema and options
	buf := make([]byte, 0, bufSize)

	// CollectionID (4 bytes)
	buf = appendUint32(buf, meta.CollectionID)

	// Name (2 bytes length + string)
	buf = appendString(buf, meta.Name)

	// CreatedTimestamp (8 bytes, Unix nanoseconds)
	buf = appendInt64(buf, meta.CreatedTimestamp.UnixNano())

	// DocumentCount (8 bytes)
	buf = appendUint64(buf, meta.DocumentCount)

	// DataSizeBytes (8 bytes)
	buf = appendUint64(buf, meta.DataSizeBytes)

	// IndexCount (2 bytes)
	buf = appendUint16(buf, meta.IndexCount)

	// FirstDataPageID (4 bytes)
	buf = appendUint32(buf, uint32(meta.FirstDataPageID))

	// StatisticsPageID (4 bytes)
	buf = appendUint32(buf, uint32(meta.StatisticsPageID))

	// Schema (optional)
	if meta.Schema != nil {
		buf = append(buf, 1) // Has schema flag
		schemaJSON, err := json.Marshal(meta.Schema)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal schema: %w", err)
		}
		buf = appendBytes(buf, schemaJSON)
	} else {
		buf = append(buf, 0) // No schema flag
	}

	// Options (optional)
	if meta.Options != nil {
		buf = append(buf, 1) // Has options flag
		optionsJSON, err := json.Marshal(meta.Options)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal options: %w", err)
		}
		buf = appendBytes(buf, optionsJSON)
	} else {
		buf = append(buf, 0) // No options flag
	}

	return buf, nil
}

// DeserializeCollectionMetadata deserializes collection metadata from bytes
func DeserializeCollectionMetadata(data []byte) (*CollectionMetadata, error) {
	if len(data) < 40 {
		return nil, fmt.Errorf("data too short for collection metadata")
	}

	meta := &CollectionMetadata{}
	offset := 0

	// CollectionID
	meta.CollectionID, offset = readUint32(data, offset)

	// Name
	meta.Name, offset = readString(data, offset)

	// CreatedTimestamp
	timestamp, newOffset := readInt64(data, offset)
	meta.CreatedTimestamp = time.Unix(0, timestamp)
	offset = newOffset

	// DocumentCount
	meta.DocumentCount, offset = readUint64(data, offset)

	// DataSizeBytes
	meta.DataSizeBytes, offset = readUint64(data, offset)

	// IndexCount
	meta.IndexCount, offset = readUint16(data, offset)

	// FirstDataPageID
	pageID, newOffset := readUint32(data, offset)
	meta.FirstDataPageID = storage.PageID(pageID)
	offset = newOffset

	// StatisticsPageID
	pageID, offset = readUint32(data, offset)
	meta.StatisticsPageID = storage.PageID(pageID)

	// Schema (optional)
	if offset < len(data) {
		hasSchema := data[offset]
		offset++
		if hasSchema == 1 {
			schemaBytes, newOffset := readBytes(data, offset)
			offset = newOffset
			meta.Schema = &CollectionSchema{}
			if err := json.Unmarshal(schemaBytes, meta.Schema); err != nil {
				return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
			}
		}
	}

	// Options (optional)
	if offset < len(data) {
		hasOptions := data[offset]
		offset++
		if hasOptions == 1 {
			optionsBytes, newOffset := readBytes(data, offset)
			offset = newOffset
			meta.Options = &CollectionOptions{}
			if err := json.Unmarshal(optionsBytes, meta.Options); err != nil {
				return nil, fmt.Errorf("failed to unmarshal options: %w", err)
			}
		}
	}

	return meta, nil
}

// SerializeIndexMetadata serializes index metadata to bytes
func SerializeIndexMetadata(meta *IndexMetadata) ([]byte, error) {
	// Estimate buffer size
	bufSize := 128 + len(meta.Name) + len(meta.PartialFilter)
	for _, fp := range meta.FieldPaths {
		bufSize += len(fp) + 2
	}
	buf := make([]byte, 0, bufSize)

	// IndexID (4 bytes)
	buf = appendUint32(buf, meta.IndexID)

	// CollectionID (4 bytes)
	buf = appendUint32(buf, meta.CollectionID)

	// Name (2 bytes length + string)
	buf = appendString(buf, meta.Name)

	// IndexType (1 byte)
	buf = append(buf, byte(meta.IndexType))

	// FieldPaths (2 bytes count + strings)
	buf = appendUint16(buf, uint16(len(meta.FieldPaths)))
	for _, fp := range meta.FieldPaths {
		buf = appendString(buf, fp)
	}

	// Flags (1 byte)
	var flags byte
	if meta.IsUnique {
		flags |= 0x01
	}
	if meta.IsSparse {
		flags |= 0x02
	}
	if meta.IsPartial {
		flags |= 0x04
	}
	buf = append(buf, flags)

	// PartialFilter (2 bytes length + string)
	buf = appendString(buf, meta.PartialFilter)

	// CreatedTimestamp (8 bytes)
	buf = appendInt64(buf, meta.CreatedTimestamp.UnixNano())

	// RootPageID (4 bytes)
	buf = appendUint32(buf, uint32(meta.RootPageID))

	// EntryCount (8 bytes)
	buf = appendUint64(buf, meta.EntryCount)

	// Order (2 bytes)
	buf = appendUint16(buf, meta.Order)

	// Options (optional)
	if len(meta.Options) > 0 {
		buf = append(buf, 1) // Has options flag
		optionsJSON, err := json.Marshal(meta.Options)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal options: %w", err)
		}
		buf = appendBytes(buf, optionsJSON)
	} else {
		buf = append(buf, 0) // No options flag
	}

	return buf, nil
}

// DeserializeIndexMetadata deserializes index metadata from bytes
func DeserializeIndexMetadata(data []byte) (*IndexMetadata, error) {
	if len(data) < 40 {
		return nil, fmt.Errorf("data too short for index metadata")
	}

	meta := &IndexMetadata{}
	offset := 0

	// IndexID
	meta.IndexID, offset = readUint32(data, offset)

	// CollectionID
	meta.CollectionID, offset = readUint32(data, offset)

	// Name
	meta.Name, offset = readString(data, offset)

	// IndexType
	if offset >= len(data) {
		return nil, fmt.Errorf("unexpected end of data")
	}
	meta.IndexType = IndexType(data[offset])
	offset++

	// FieldPaths
	fieldCount, newOffset := readUint16(data, offset)
	offset = newOffset
	meta.FieldPaths = make([]string, fieldCount)
	for i := uint16(0); i < fieldCount; i++ {
		meta.FieldPaths[i], offset = readString(data, offset)
	}

	// Flags
	if offset >= len(data) {
		return nil, fmt.Errorf("unexpected end of data")
	}
	flags := data[offset]
	offset++
	meta.IsUnique = (flags & 0x01) != 0
	meta.IsSparse = (flags & 0x02) != 0
	meta.IsPartial = (flags & 0x04) != 0

	// PartialFilter
	meta.PartialFilter, offset = readString(data, offset)

	// CreatedTimestamp
	timestamp, newOffset := readInt64(data, offset)
	meta.CreatedTimestamp = time.Unix(0, timestamp)
	offset = newOffset

	// RootPageID
	pageID, newOffset := readUint32(data, offset)
	meta.RootPageID = storage.PageID(pageID)
	offset = newOffset

	// EntryCount
	meta.EntryCount, offset = readUint64(data, offset)

	// Order
	meta.Order, offset = readUint16(data, offset)

	// Options (optional)
	if offset < len(data) {
		hasOptions := data[offset]
		offset++
		if hasOptions == 1 {
			optionsBytes, newOffset := readBytes(data, offset)
			offset = newOffset
			meta.Options = make(map[string]interface{})
			if err := json.Unmarshal(optionsBytes, &meta.Options); err != nil {
				return nil, fmt.Errorf("failed to unmarshal options: %w", err)
			}
		}
	}

	return meta, nil
}

// Helper functions for serialization

func appendUint16(buf []byte, val uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, val)
	return append(buf, b...)
}

func appendUint32(buf []byte, val uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, val)
	return append(buf, b...)
}

func appendUint64(buf []byte, val uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, val)
	return append(buf, b...)
}

func appendInt64(buf []byte, val int64) []byte {
	return appendUint64(buf, uint64(val))
}

func appendString(buf []byte, s string) []byte {
	buf = appendUint16(buf, uint16(len(s)))
	return append(buf, []byte(s)...)
}

func appendBytes(buf []byte, data []byte) []byte {
	buf = appendUint16(buf, uint16(len(data)))
	return append(buf, data...)
}

func readUint16(data []byte, offset int) (uint16, int) {
	if offset+2 > len(data) {
		return 0, offset
	}
	val := binary.LittleEndian.Uint16(data[offset : offset+2])
	return val, offset + 2
}

func readUint32(data []byte, offset int) (uint32, int) {
	if offset+4 > len(data) {
		return 0, offset
	}
	val := binary.LittleEndian.Uint32(data[offset : offset+4])
	return val, offset + 4
}

func readUint64(data []byte, offset int) (uint64, int) {
	if offset+8 > len(data) {
		return 0, offset
	}
	val := binary.LittleEndian.Uint64(data[offset : offset+8])
	return val, offset + 8
}

func readInt64(data []byte, offset int) (int64, int) {
	val, newOffset := readUint64(data, offset)
	return int64(val), newOffset
}

func readString(data []byte, offset int) (string, int) {
	length, newOffset := readUint16(data, offset)
	if newOffset+int(length) > len(data) {
		return "", newOffset
	}
	s := string(data[newOffset : newOffset+int(length)])
	return s, newOffset + int(length)
}

func readBytes(data []byte, offset int) ([]byte, int) {
	length, newOffset := readUint16(data, offset)
	if newOffset+int(length) > len(data) {
		return nil, newOffset
	}
	bytes := data[newOffset : newOffset+int(length)]
	return bytes, newOffset + int(length)
}

// SaveCollectionMetadata saves collection metadata to a disk page
func SaveCollectionMetadata(diskMgr *storage.DiskManager, pageID storage.PageID, meta *CollectionMetadata) error {
	// Serialize metadata
	data, err := SerializeCollectionMetadata(meta)
	if err != nil {
		return fmt.Errorf("failed to serialize collection metadata: %w", err)
	}

	// Create or load page
	page := storage.NewPage(pageID, storage.PageTypeData) // Using PageTypeData for now

	// Write serialized data to page (starting after page header)
	if len(data) > storage.PageSize-storage.PageHeaderSize {
		return fmt.Errorf("metadata too large for single page: %d bytes", len(data))
	}

	copy(page.Data[storage.PageHeaderSize:], data)

	// Write page to disk
	if err := diskMgr.WritePage(page); err != nil {
		return fmt.Errorf("failed to write metadata page: %w", err)
	}

	return nil
}

// LoadCollectionMetadata loads collection metadata from a disk page
func LoadCollectionMetadata(diskMgr *storage.DiskManager, pageID storage.PageID) (*CollectionMetadata, error) {
	// Read page from disk
	page, err := diskMgr.ReadPage(pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata page: %w", err)
	}

	// Extract data (skip page header)
	data := page.Data[storage.PageHeaderSize:]

	// Deserialize metadata
	meta, err := DeserializeCollectionMetadata(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize collection metadata: %w", err)
	}

	return meta, nil
}

// SaveIndexMetadata saves index metadata to a disk page
func SaveIndexMetadata(diskMgr *storage.DiskManager, pageID storage.PageID, meta *IndexMetadata) error {
	// Serialize metadata
	data, err := SerializeIndexMetadata(meta)
	if err != nil {
		return fmt.Errorf("failed to serialize index metadata: %w", err)
	}

	// Create or load page
	page := storage.NewPage(pageID, storage.PageTypeData)

	// Write serialized data to page
	if len(data) > storage.PageSize-storage.PageHeaderSize {
		return fmt.Errorf("metadata too large for single page: %d bytes", len(data))
	}

	copy(page.Data[storage.PageHeaderSize:], data)

	// Write page to disk
	if err := diskMgr.WritePage(page); err != nil {
		return fmt.Errorf("failed to write index metadata page: %w", err)
	}

	return nil
}

// LoadIndexMetadata loads index metadata from a disk page
func LoadIndexMetadata(diskMgr *storage.DiskManager, pageID storage.PageID) (*IndexMetadata, error) {
	// Read page from disk
	page, err := diskMgr.ReadPage(pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to read index metadata page: %w", err)
	}

	// Extract data (skip page header)
	data := page.Data[storage.PageHeaderSize:]

	// Deserialize metadata
	meta, err := DeserializeIndexMetadata(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize index metadata: %w", err)
	}

	return meta, nil
}

// IndexStatistics holds detailed statistics about an index
type IndexStatistics struct {
	IndexID          uint32
	TotalEntries     uint64    // Total number of entries in the index
	UniqueKeys       uint64    // Number of unique keys (estimate)
	TreeHeight       uint16    // Height of the B+ tree
	LeafNodeCount    uint32    // Number of leaf nodes
	InternalNodeCount uint32   // Number of internal nodes
	AvgKeySize       uint32    // Average key size in bytes
	AvgValueSize     uint32    // Average value size in bytes
	MinKey           interface{} // Minimum key value (for range statistics)
	MaxKey           interface{} // Maximum key value (for range statistics)
	LastUpdated      time.Time   // Last time statistics were updated
	Cardinality      float64     // Selectivity estimate (unique keys / total entries)
	
	// Query optimization statistics
	IndexScans       uint64    // Number of times index was scanned
	IndexSeeks       uint64    // Number of times index was sought
	LastAccessTime   time.Time // Last time index was accessed
}

// SerializeIndexStatistics serializes index statistics to bytes
func SerializeIndexStatistics(stats *IndexStatistics) ([]byte, error) {
	buf := make([]byte, 0, 256)
	
	// IndexID (4 bytes)
	buf = appendUint32(buf, stats.IndexID)
	
	// TotalEntries (8 bytes)
	buf = appendUint64(buf, stats.TotalEntries)
	
	// UniqueKeys (8 bytes)
	buf = appendUint64(buf, stats.UniqueKeys)
	
	// TreeHeight (2 bytes)
	buf = appendUint16(buf, stats.TreeHeight)
	
	// LeafNodeCount (4 bytes)
	buf = appendUint32(buf, stats.LeafNodeCount)
	
	// InternalNodeCount (4 bytes)
	buf = appendUint32(buf, stats.InternalNodeCount)
	
	// AvgKeySize (4 bytes)
	buf = appendUint32(buf, stats.AvgKeySize)
	
	// AvgValueSize (4 bytes)
	buf = appendUint32(buf, stats.AvgValueSize)
	
	// MinKey and MaxKey (JSON encoded for flexibility)
	minKeyJSON, err := json.Marshal(stats.MinKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal min key: %w", err)
	}
	buf = appendBytes(buf, minKeyJSON)
	
	maxKeyJSON, err := json.Marshal(stats.MaxKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal max key: %w", err)
	}
	buf = appendBytes(buf, maxKeyJSON)
	
	// LastUpdated (8 bytes)
	buf = appendInt64(buf, stats.LastUpdated.UnixNano())
	
	// Cardinality (8 bytes, float64 as bits)
	cardinalityBits := math.Float64bits(stats.Cardinality)
	buf = appendUint64(buf, cardinalityBits)
	
	// IndexScans (8 bytes)
	buf = appendUint64(buf, stats.IndexScans)
	
	// IndexSeeks (8 bytes)
	buf = appendUint64(buf, stats.IndexSeeks)
	
	// LastAccessTime (8 bytes)
	buf = appendInt64(buf, stats.LastAccessTime.UnixNano())
	
	return buf, nil
}

// DeserializeIndexStatistics deserializes index statistics from bytes
func DeserializeIndexStatistics(data []byte) (*IndexStatistics, error) {
	if len(data) < 40 {
		return nil, fmt.Errorf("data too short for index statistics")
	}
	
	stats := &IndexStatistics{}
	offset := 0
	
	// IndexID
	stats.IndexID, offset = readUint32(data, offset)
	
	// TotalEntries
	stats.TotalEntries, offset = readUint64(data, offset)
	
	// UniqueKeys
	stats.UniqueKeys, offset = readUint64(data, offset)
	
	// TreeHeight
	stats.TreeHeight, offset = readUint16(data, offset)
	
	// LeafNodeCount
	stats.LeafNodeCount, offset = readUint32(data, offset)
	
	// InternalNodeCount
	stats.InternalNodeCount, offset = readUint32(data, offset)
	
	// AvgKeySize
	stats.AvgKeySize, offset = readUint32(data, offset)
	
	// AvgValueSize
	stats.AvgValueSize, offset = readUint32(data, offset)
	
	// MinKey
	minKeyJSON, newOffset := readBytes(data, offset)
	if len(minKeyJSON) > 0 {
		if err := json.Unmarshal(minKeyJSON, &stats.MinKey); err != nil {
			return nil, fmt.Errorf("failed to unmarshal min key: %w", err)
		}
	}
	offset = newOffset
	
	// MaxKey
	maxKeyJSON, newOffset := readBytes(data, offset)
	if len(maxKeyJSON) > 0 {
		if err := json.Unmarshal(maxKeyJSON, &stats.MaxKey); err != nil {
			return nil, fmt.Errorf("failed to unmarshal max key: %w", err)
		}
	}
	offset = newOffset
	
	// LastUpdated
	lastUpdatedNano, newOffset := readInt64(data, offset)
	stats.LastUpdated = time.Unix(0, lastUpdatedNano)
	offset = newOffset
	
	// Cardinality
	cardinalityBits, newOffset := readUint64(data, offset)
	stats.Cardinality = math.Float64frombits(cardinalityBits)
	offset = newOffset
	
	// IndexScans
	stats.IndexScans, offset = readUint64(data, offset)
	
	// IndexSeeks
	stats.IndexSeeks, offset = readUint64(data, offset)
	
	// LastAccessTime
	lastAccessNano, newOffset := readInt64(data, offset)
	stats.LastAccessTime = time.Unix(0, lastAccessNano)
	offset = newOffset
	
	return stats, nil
}

// SaveIndexStatistics saves index statistics to a disk page
func SaveIndexStatistics(diskMgr *storage.DiskManager, pageID storage.PageID, stats *IndexStatistics) error {
	// Serialize statistics
	data, err := SerializeIndexStatistics(stats)
	if err != nil {
		return fmt.Errorf("failed to serialize statistics: %w", err)
	}
	
	// Create page
	page := storage.NewPage(pageID, storage.PageTypeData)
	
	// Check if data fits in page
	if len(data) > len(page.Data) {
		return fmt.Errorf("statistics data too large for page")
	}
	
	// Copy data to page
	copy(page.Data, data)
	
	// Write page to disk
	if err := diskMgr.WritePage(page); err != nil {
		return fmt.Errorf("failed to write statistics page: %w", err)
	}
	
	return nil
}

// LoadIndexStatistics loads index statistics from a disk page
func LoadIndexStatistics(diskMgr *storage.DiskManager, pageID storage.PageID) (*IndexStatistics, error) {
	// Read page from disk
	page, err := diskMgr.ReadPage(pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to read statistics page: %w", err)
	}
	
	// Deserialize statistics
	stats, err := DeserializeIndexStatistics(page.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize statistics: %w", err)
	}
	
	return stats, nil
}
