package index

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/storage"
)

// KeyType represents the type of a key stored in a B+ tree node
type KeyType byte

const (
	KeyTypeInt64     KeyType = 0
	KeyTypeFloat64   KeyType = 1
	KeyTypeString    KeyType = 2
	KeyTypeObjectID  KeyType = 3
	KeyTypeComposite KeyType = 4
)

// BTreeNodeHeader represents the header of a B+ tree node page (32 bytes)
type BTreeNodeHeader struct {
	NodeType        byte   // 0 = internal, 1 = leaf
	Level           byte   // Distance from leaves (0 = leaf)
	KeyCount        uint16 // Number of keys in node
	ChildCount      uint16 // Number of children (internal nodes only)
	IndexID         uint32 // Parent index identifier
	CollectionID    uint32 // Parent collection
	ParentPageID    uint32 // Parent node page
	NextPageID      uint32 // Next sibling (leaf: next leaf in chain)
	PrevPageID      uint32 // Previous sibling
	FreeSpaceOffset uint16 // Start of free space
	Reserved        uint16 // Reserved for future use
}

// DocumentID represents a reference to a document on disk
type DocumentID struct {
	CollectionID uint32
	PageID       uint32
	SlotID       uint16
	Reserved     uint16
}

// KeyEntry represents a key directory entry
type KeyEntry struct {
	KeyOffset uint16
	KeyLength uint16
	KeyType   KeyType
}

// ValueEntry represents a value directory entry (leaf nodes only)
type ValueEntry struct {
	ValueOffset uint16
	ValueLength uint16
	Flags       byte // Entry flags (deleted, versioned, etc.)
}

// Entry flags
const (
	EntryFlagDeleted   = 1 << 0 // 0x01
	EntryFlagVersioned = 1 << 1 // 0x02
)

// SerializeBTreeNode serializes a B+ tree node to a page
func SerializeBTreeNode(node *BTreeNode, page *storage.Page, indexID, collectionID uint32) error {
	if node == nil {
		return fmt.Errorf("node is nil")
	}

	// Start after page header
	offset := storage.PageHeaderSize

	// Write B+ tree node header
	header := BTreeNodeHeader{
		NodeType:        boolToByte(node.isLeaf),
		Level:           0, // Will be set by caller
		KeyCount:        uint16(len(node.keys)),
		ChildCount:      uint16(len(node.children)),
		IndexID:         indexID,
		CollectionID:    collectionID,
		ParentPageID:    0, // Will be set when linking nodes
		NextPageID:      0, // Will be set when linking nodes
		PrevPageID:      0,
		FreeSpaceOffset: 0, // Will be calculated
		Reserved:        0,
	}

	// Write header fields
	page.Data[offset] = header.NodeType
	offset++
	page.Data[offset] = header.Level
	offset++
	binary.LittleEndian.PutUint16(page.Data[offset:offset+2], header.KeyCount)
	offset += 2
	binary.LittleEndian.PutUint16(page.Data[offset:offset+2], header.ChildCount)
	offset += 2
	binary.LittleEndian.PutUint32(page.Data[offset:offset+4], header.IndexID)
	offset += 4
	binary.LittleEndian.PutUint32(page.Data[offset:offset+4], header.CollectionID)
	offset += 4
	binary.LittleEndian.PutUint32(page.Data[offset:offset+4], header.ParentPageID)
	offset += 4
	binary.LittleEndian.PutUint32(page.Data[offset:offset+4], header.NextPageID)
	offset += 4
	binary.LittleEndian.PutUint32(page.Data[offset:offset+4], header.PrevPageID)
	offset += 4
	binary.LittleEndian.PutUint16(page.Data[offset:offset+2], header.FreeSpaceOffset)
	offset += 2
	binary.LittleEndian.PutUint16(page.Data[offset:offset+2], header.Reserved)
	offset += 2

	// For internal nodes, write child page IDs
	if !node.isLeaf {
		return serializeInternalNode(node, page, offset)
	}

	// For leaf nodes, write key-value pairs
	return serializeLeafNode(node, page, offset)
}

func serializeInternalNode(node *BTreeNode, page *storage.Page, startOffset int) error {
	offset := startOffset

	// Write child page IDs (placeholder - will be set when nodes have pageIDs)
	for i := 0; i < len(node.children); i++ {
		binary.LittleEndian.PutUint32(page.Data[offset:offset+4], 0) // Placeholder
		offset += 4
	}

	// Write key directory
	keyDirOffset := offset
	keyDataOffset := len(page.Data) // Start from bottom of available data
	offset = keyDirOffset + (len(node.keys) * 5) // 5 bytes per key entry

	// Write keys from bottom up (but iterate forward)
	for i := 0; i < len(node.keys); i++ {
		keyData, keyType, err := serializeKey(node.keys[i])
		if err != nil {
			return fmt.Errorf("failed to serialize key %d: %w", i, err)
		}

		keyDataOffset -= len(keyData)
		copy(page.Data[keyDataOffset:], keyData)

		// Write key directory entry
		entryOffset := keyDirOffset + (i * 5)
		binary.LittleEndian.PutUint16(page.Data[entryOffset:entryOffset+2], uint16(keyDataOffset))
		binary.LittleEndian.PutUint16(page.Data[entryOffset+2:entryOffset+4], uint16(len(keyData)))
		page.Data[entryOffset+4] = byte(keyType)
	}

	// Update free space offset in header
	freeSpaceOffset := offset
	headerOffset := storage.PageHeaderSize + 14 // Offset to FreeSpaceOffset field
	binary.LittleEndian.PutUint16(page.Data[headerOffset:headerOffset+2], uint16(freeSpaceOffset))

	return nil
}

func serializeLeafNode(node *BTreeNode, page *storage.Page, startOffset int) error {
	offset := startOffset

	// Write entry directory (10 bytes per entry)
	entryDirOffset := offset
	entryDataOffset := len(page.Data) // Start from bottom of available data
	offset = entryDirOffset + (len(node.keys) * 10)

	// Write entries from bottom up (but iterate forward)
	for i := 0; i < len(node.keys); i++ {
		// Serialize key
		keyData, keyType, err := serializeKey(node.keys[i])
		if err != nil {
			return fmt.Errorf("failed to serialize key %d: %w", i, err)
		}

		// Serialize value (DocumentID for now - interface{} will be converted)
		valueData, err := serializeValue(node.values[i])
		if err != nil {
			return fmt.Errorf("failed to serialize value %d: %w", i, err)
		}

		// Write value first (at bottom, growing upward)
		entryDataOffset -= len(valueData)
		valueOffset := entryDataOffset
		copy(page.Data[valueOffset:], valueData)

		// Write key
		entryDataOffset -= len(keyData)
		keyOffset := entryDataOffset
		copy(page.Data[keyOffset:], keyData)

		// Write entry directory entry (10 bytes)
		entryOffset := entryDirOffset + (i * 10)
		binary.LittleEndian.PutUint16(page.Data[entryOffset:entryOffset+2], uint16(keyOffset))
		binary.LittleEndian.PutUint16(page.Data[entryOffset+2:entryOffset+4], uint16(len(keyData)))
		page.Data[entryOffset+4] = byte(keyType)
		binary.LittleEndian.PutUint16(page.Data[entryOffset+5:entryOffset+7], uint16(valueOffset))
		binary.LittleEndian.PutUint16(page.Data[entryOffset+7:entryOffset+9], uint16(len(valueData)))
		page.Data[entryOffset+9] = 0 // Flags (no MVCC for now)
	}

	// Update free space offset in header
	freeSpaceOffset := offset
	headerOffset := storage.PageHeaderSize + 14 // Offset to FreeSpaceOffset field
	binary.LittleEndian.PutUint16(page.Data[headerOffset:headerOffset+2], uint16(freeSpaceOffset))

	return nil
}

func serializeKey(key interface{}) ([]byte, KeyType, error) {
	switch v := key.(type) {
	case int64:
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, uint64(v))
		return data, KeyTypeInt64, nil

	case float64:
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, math.Float64bits(v))
		return data, KeyTypeFloat64, nil

	case string:
		strBytes := []byte(v)
		data := make([]byte, 2+len(strBytes))
		binary.LittleEndian.PutUint16(data[0:2], uint16(len(strBytes)))
		copy(data[2:], strBytes)
		return data, KeyTypeString, nil

	case document.ObjectID:
		// ObjectID is 12 bytes
		return v[:], KeyTypeObjectID, nil

	default:
		return nil, 0, fmt.Errorf("unsupported key type: %T", key)
	}
}

func serializeValue(value interface{}) ([]byte, error) {
	// For now, values are expected to be document pointers (interface{})
	// In the disk-based system, we'll use DocumentID
	// For initial implementation, serialize as placeholder 12-byte DocumentID
	data := make([]byte, 12)
	// Placeholder: will be replaced with actual DocumentID when integrated
	return data, nil
}

// DeserializeBTreeNode deserializes a B+ tree node from a page
func DeserializeBTreeNode(page *storage.Page) (*BTreeNode, error) {
	// Start after page header
	offset := storage.PageHeaderSize

	// Read B+ tree node header
	header := BTreeNodeHeader{}
	header.NodeType = page.Data[offset]
	offset++
	header.Level = page.Data[offset]
	offset++
	header.KeyCount = binary.LittleEndian.Uint16(page.Data[offset : offset+2])
	offset += 2
	header.ChildCount = binary.LittleEndian.Uint16(page.Data[offset : offset+2])
	offset += 2
	header.IndexID = binary.LittleEndian.Uint32(page.Data[offset : offset+4])
	offset += 4
	header.CollectionID = binary.LittleEndian.Uint32(page.Data[offset : offset+4])
	offset += 4
	header.ParentPageID = binary.LittleEndian.Uint32(page.Data[offset : offset+4])
	offset += 4
	header.NextPageID = binary.LittleEndian.Uint32(page.Data[offset : offset+4])
	offset += 4
	header.PrevPageID = binary.LittleEndian.Uint32(page.Data[offset : offset+4])
	offset += 4
	header.FreeSpaceOffset = binary.LittleEndian.Uint16(page.Data[offset : offset+2])
	offset += 2
	header.Reserved = binary.LittleEndian.Uint16(page.Data[offset : offset+2])
	offset += 2

	// Create node
	node := &BTreeNode{
		isLeaf: header.NodeType == 1,
		keys:   make([]interface{}, 0, header.KeyCount),
		values: nil,
	}

	// Deserialize based on node type
	if !node.isLeaf {
		return deserializeInternalNode(node, page, offset, header)
	}

	return deserializeLeafNode(node, page, offset, header)
}

func deserializeInternalNode(node *BTreeNode, page *storage.Page, startOffset int, header BTreeNodeHeader) (*BTreeNode, error) {
	offset := startOffset

	// Read child page IDs
	childPageIDs := make([]uint32, header.ChildCount)
	for i := 0; i < int(header.ChildCount); i++ {
		childPageIDs[i] = binary.LittleEndian.Uint32(page.Data[offset : offset+4])
		offset += 4
	}

	// Read key directory
	keyDirOffset := offset
	for i := 0; i < int(header.KeyCount); i++ {
		entryOffset := keyDirOffset + (i * 5)
		keyOffset := binary.LittleEndian.Uint16(page.Data[entryOffset : entryOffset+2])
		keyLength := binary.LittleEndian.Uint16(page.Data[entryOffset+2 : entryOffset+4])
		keyType := KeyType(page.Data[entryOffset+4])

		// Deserialize key
		keyData := page.Data[keyOffset : keyOffset+keyLength]
		key, err := deserializeKey(keyData, keyType)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize key %d: %w", i, err)
		}

		node.keys = append(node.keys, key)
	}

	// Note: Children are not loaded yet (lazy loading)
	// Store child page IDs for later loading
	node.children = make([]*BTreeNode, len(childPageIDs))
	// For now, children are nil - will be loaded lazily

	return node, nil
}

func deserializeLeafNode(node *BTreeNode, page *storage.Page, startOffset int, header BTreeNodeHeader) (*BTreeNode, error) {
	offset := startOffset
	node.values = make([]interface{}, 0, header.KeyCount)

	// Read entry directory
	entryDirOffset := offset
	for i := 0; i < int(header.KeyCount); i++ {
		entryOffset := entryDirOffset + (i * 10)
		keyOffset := binary.LittleEndian.Uint16(page.Data[entryOffset : entryOffset+2])
		keyLength := binary.LittleEndian.Uint16(page.Data[entryOffset+2 : entryOffset+4])
		keyType := KeyType(page.Data[entryOffset+4])
		valueOffset := binary.LittleEndian.Uint16(page.Data[entryOffset+5 : entryOffset+7])
		valueLength := binary.LittleEndian.Uint16(page.Data[entryOffset+7 : entryOffset+9])
		flags := page.Data[entryOffset+9]

		// Check if entry is deleted
		if flags&EntryFlagDeleted != 0 {
			continue // Skip deleted entries
		}

		// Deserialize key
		keyData := page.Data[keyOffset : keyOffset+keyLength]
		key, err := deserializeKey(keyData, keyType)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize key %d: %w", i, err)
		}

		// Deserialize value
		valueData := page.Data[valueOffset : valueOffset+valueLength]
		value, err := deserializeValue(valueData)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize value %d: %w", i, err)
		}

		node.keys = append(node.keys, key)
		node.values = append(node.values, value)
	}

	return node, nil
}

func deserializeKey(data []byte, keyType KeyType) (interface{}, error) {
	switch keyType {
	case KeyTypeInt64:
		if len(data) != 8 {
			return nil, fmt.Errorf("invalid int64 key length: %d", len(data))
		}
		return int64(binary.LittleEndian.Uint64(data)), nil

	case KeyTypeFloat64:
		if len(data) != 8 {
			return nil, fmt.Errorf("invalid float64 key length: %d", len(data))
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(data)), nil

	case KeyTypeString:
		if len(data) < 2 {
			return nil, fmt.Errorf("invalid string key length: %d", len(data))
		}
		strLen := binary.LittleEndian.Uint16(data[0:2])
		if len(data) < int(2+strLen) {
			return nil, fmt.Errorf("string key data truncated")
		}
		return string(data[2 : 2+strLen]), nil

	case KeyTypeObjectID:
		if len(data) != 12 {
			return nil, fmt.Errorf("invalid ObjectID key length: %d", len(data))
		}
		var oid document.ObjectID
		copy(oid[:], data)
		return oid, nil

	default:
		return nil, fmt.Errorf("unsupported key type: %d", keyType)
	}
}

func deserializeValue(data []byte) (interface{}, error) {
	// For now, return placeholder
	// Will be replaced with actual DocumentID deserialization
	if len(data) != 12 {
		return nil, fmt.Errorf("invalid value length: %d", len(data))
	}
	return data, nil
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}
// LoadNodeFromDisk loads a B+ tree node from disk
// diskMgr should be *storage.DiskManager but we use interface{} to avoid import cycle
func LoadNodeFromDisk(diskMgr interface{}, pageID storage.PageID, cache *NodeCache) (*BTreeNode, error) {
	// Check cache first
	if cache != nil {
		if node, found := cache.Get(uint32(pageID)); found {
			return node, nil
		}
	}

	// Cast diskMgr to the correct type (this will be done properly when integrated)
	dm, ok := diskMgr.(interface {
		ReadPage(storage.PageID) (*storage.Page, error)
	})
	if !ok {
		return nil, fmt.Errorf("invalid disk manager type")
	}

	// Read page from disk
	page, err := dm.ReadPage(pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to read page %d: %w", pageID, err)
	}

	// Deserialize node
	node, err := DeserializeBTreeNode(page)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize node: %w", err)
	}

	// Set disk persistence fields
	node.pageID = uint32(pageID)
	node.isLoaded = true
	node.isDirty = false

	// Add to cache
	if cache != nil {
		cache.Put(uint32(pageID), node)
	}

	return node, nil
}

// WriteNodeToDisk writes a B+ tree node to disk
func WriteNodeToDisk(diskMgr interface{}, node *BTreeNode, indexID, collectionID uint32) error {
	// Cast diskMgr to the correct type
	dm, ok := diskMgr.(interface {
		WritePage(*storage.Page) error
		AllocatePage() (storage.PageID, error)
	})
	if !ok {
		return fmt.Errorf("invalid disk manager type")
	}

	// Allocate page if node doesn't have one
	if node.pageID == 0 {
		pageID, err := dm.AllocatePage()
		if err != nil {
			return fmt.Errorf("failed to allocate page: %w", err)
		}
		node.pageID = uint32(pageID)
	}

	// Create page
	page := storage.NewPage(storage.PageID(node.pageID), storage.PageTypeIndex)

	// Serialize node to page
	if err := SerializeBTreeNode(node, page, indexID, collectionID); err != nil {
		return fmt.Errorf("failed to serialize node: %w", err)
	}

	// Write page to disk
	if err := dm.WritePage(page); err != nil {
		return fmt.Errorf("failed to write page: %w", err)
	}

	// Mark as clean
	node.isDirty = false

	return nil
}

// FlushDirtyNodes writes all dirty nodes in the cache to disk
func FlushDirtyNodes(diskMgr interface{}, cache *NodeCache, indexID, collectionID uint32) error {
	if cache == nil {
		return nil
	}

	dirtyNodes := cache.GetDirtyNodes()
	for _, node := range dirtyNodes {
		if err := WriteNodeToDisk(diskMgr, node, indexID, collectionID); err != nil {
			return fmt.Errorf("failed to flush node %d: %w", node.pageID, err)
		}
	}

	// Mark all nodes as clean
	cache.Flush()

	return nil
}
