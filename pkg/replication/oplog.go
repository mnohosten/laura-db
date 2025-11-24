package replication

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/mnohosten/laura-db/pkg/document"
)

// OpType represents the type of operation in the oplog
type OpType uint8

const (
	OpTypeInsert OpType = iota
	OpTypeUpdate
	OpTypeDelete
	OpTypeCreateCollection
	OpTypeDropCollection
	OpTypeCreateIndex
	OpTypeDropIndex
	OpTypeNoop // No-operation (used for heartbeats)
)

// String returns the string representation of OpType
func (t OpType) String() string {
	switch t {
	case OpTypeInsert:
		return "insert"
	case OpTypeUpdate:
		return "update"
	case OpTypeDelete:
		return "delete"
	case OpTypeCreateCollection:
		return "createCollection"
	case OpTypeDropCollection:
		return "dropCollection"
	case OpTypeCreateIndex:
		return "createIndex"
	case OpTypeDropIndex:
		return "dropIndex"
	case OpTypeNoop:
		return "noop"
	default:
		return "unknown"
	}
}

// OpID is a unique identifier for an operation
type OpID uint64

// OplogEntry represents a single operation in the replication log
type OplogEntry struct {
	OpID       OpID                   `json:"op_id"`
	Timestamp  time.Time              `json:"ts"`
	OpType     OpType                 `json:"op"`
	Database   string                 `json:"db"`
	Collection string                 `json:"coll"`
	DocID      interface{}            `json:"doc_id,omitempty"`      // _id of the document
	Document   map[string]interface{} `json:"doc,omitempty"`         // For insert operations
	Update     map[string]interface{} `json:"update,omitempty"`      // For update operations
	Filter     map[string]interface{} `json:"filter,omitempty"`      // For update/delete operations
	IndexDef   map[string]interface{} `json:"index_def,omitempty"`   // For index operations
}

// Oplog manages the operation log for replication
type Oplog struct {
	file       *os.File
	mu         sync.RWMutex
	currentID  OpID
	path       string
	entries    []*OplogEntry // In-memory cache of recent entries
	maxEntries int           // Maximum number of entries to keep in memory
}

// NewOplog creates a new operation log
func NewOplog(path string) (*Oplog, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open oplog file: %w", err)
	}

	oplog := &Oplog{
		file:       file,
		currentID:  0,
		path:       path,
		entries:    make([]*OplogEntry, 0),
		maxEntries: 10000, // Keep last 10k entries in memory
	}

	// Load existing entries to determine current ID
	if err := oplog.loadEntries(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to load oplog entries: %w", err)
	}

	return oplog, nil
}

// Append adds a new operation to the log
func (o *Oplog) Append(entry *OplogEntry) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Assign OpID and timestamp
	o.currentID++
	entry.OpID = o.currentID
	entry.Timestamp = time.Now()

	// Serialize entry
	data, err := o.serializeEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to serialize entry: %w", err)
	}

	// Write to file
	if _, err := o.file.Write(data); err != nil {
		return fmt.Errorf("failed to write oplog entry: %w", err)
	}

	// Add to in-memory cache
	o.entries = append(o.entries, entry)

	// Trim cache if needed
	if len(o.entries) > o.maxEntries {
		o.entries = o.entries[len(o.entries)-o.maxEntries:]
	}

	return nil
}

// serializeEntry converts an oplog entry to bytes
// Format: [4-byte length][JSON data]
func (o *Oplog) serializeEntry(entry *OplogEntry) ([]byte, error) {
	// Marshal to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}

	// Create buffer with length prefix
	buf := make([]byte, 4+len(jsonData))
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(jsonData)))
	copy(buf[4:], jsonData)

	return buf, nil
}

// deserializeEntry converts bytes to an oplog entry
func (o *Oplog) deserializeEntry(data []byte) (*OplogEntry, error) {
	var entry OplogEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// GetEntriesSince returns all entries after the given OpID
func (o *Oplog) GetEntriesSince(afterID OpID) ([]*OplogEntry, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// First check in-memory cache
	result := make([]*OplogEntry, 0)
	for _, entry := range o.entries {
		if entry.OpID > afterID {
			result = append(result, entry)
		}
	}

	// If we found entries in cache, return them
	if len(result) > 0 {
		return result, nil
	}

	// Otherwise, read from disk
	return o.readEntriesFromDisk(afterID)
}

// readEntriesFromDisk reads entries from disk that are after the given OpID
func (o *Oplog) readEntriesFromDisk(afterID OpID) ([]*OplogEntry, error) {
	// Open file for reading
	file, err := os.Open(o.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := make([]*OplogEntry, 0)
	lengthBuf := make([]byte, 4)

	for {
		// Read length
		n, err := file.Read(lengthBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if n < 4 {
			break // Incomplete entry
		}

		length := binary.LittleEndian.Uint32(lengthBuf)

		// Read data
		data := make([]byte, length)
		if _, err := io.ReadFull(file, data); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// Deserialize
		entry, err := o.deserializeEntry(data)
		if err != nil {
			return nil, err
		}

		// Add if after specified ID
		if entry.OpID > afterID {
			result = append(result, entry)
		}
	}

	return result, nil
}

// loadEntries loads all entries from disk into memory
func (o *Oplog) loadEntries() error {
	// Seek to beginning
	if _, err := o.file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	lengthBuf := make([]byte, 4)

	for {
		// Read length
		n, err := o.file.Read(lengthBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if n < 4 {
			break // Incomplete entry
		}

		length := binary.LittleEndian.Uint32(lengthBuf)

		// Read data
		data := make([]byte, length)
		if _, err := io.ReadFull(o.file, data); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// Deserialize
		entry, err := o.deserializeEntry(data)
		if err != nil {
			return err
		}

		// Update current ID
		if entry.OpID > o.currentID {
			o.currentID = entry.OpID
		}

		// Add to cache (keep only recent entries)
		o.entries = append(o.entries, entry)
		if len(o.entries) > o.maxEntries {
			o.entries = o.entries[1:]
		}
	}

	// Seek back to end for appending
	if _, err := o.file.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	return nil
}

// GetCurrentID returns the current OpID
func (o *Oplog) GetCurrentID() OpID {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.currentID
}

// Flush ensures all data is written to disk
func (o *Oplog) Flush() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.file.Sync()
}

// Close closes the oplog
func (o *Oplog) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if err := o.file.Sync(); err != nil {
		return err
	}
	return o.file.Close()
}

// CreateInsertEntry creates an oplog entry for an insert operation
func CreateInsertEntry(db, coll string, doc map[string]interface{}) *OplogEntry {
	docID := doc["_id"]
	if docID == nil {
		// Generate new ObjectID if not present
		docID = document.NewObjectID()
		doc["_id"] = docID
	}

	return &OplogEntry{
		OpType:     OpTypeInsert,
		Database:   db,
		Collection: coll,
		DocID:      docID,
		Document:   doc,
	}
}

// CreateUpdateEntry creates an oplog entry for an update operation
func CreateUpdateEntry(db, coll string, filter, update map[string]interface{}) *OplogEntry {
	return &OplogEntry{
		OpType:     OpTypeUpdate,
		Database:   db,
		Collection: coll,
		Filter:     filter,
		Update:     update,
	}
}

// CreateDeleteEntry creates an oplog entry for a delete operation
func CreateDeleteEntry(db, coll string, filter map[string]interface{}) *OplogEntry {
	return &OplogEntry{
		OpType:     OpTypeDelete,
		Database:   db,
		Collection: coll,
		Filter:     filter,
	}
}

// CreateCollectionEntry creates an oplog entry for collection operations
func CreateCollectionEntry(db, coll string, create bool) *OplogEntry {
	opType := OpTypeCreateCollection
	if !create {
		opType = OpTypeDropCollection
	}
	return &OplogEntry{
		OpType:     opType,
		Database:   db,
		Collection: coll,
	}
}

// CreateIndexEntry creates an oplog entry for index operations
func CreateIndexEntry(db, coll string, indexDef map[string]interface{}, create bool) *OplogEntry {
	opType := OpTypeCreateIndex
	if !create {
		opType = OpTypeDropIndex
	}
	return &OplogEntry{
		OpType:     opType,
		Database:   db,
		Collection: coll,
		IndexDef:   indexDef,
	}
}

// CreateNoopEntry creates a no-operation entry (for heartbeats)
func CreateNoopEntry(db string) *OplogEntry {
	return &OplogEntry{
		OpType:   OpTypeNoop,
		Database: db,
	}
}
