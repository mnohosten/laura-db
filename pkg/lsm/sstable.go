package lsm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// SSTable represents a Sorted String Table - immutable on-disk sorted file
type SSTable struct {
	path        string
	index       *SSTableIndex // Sparse index for quick lookups
	bloomFilter *BloomFilter
	minKey      []byte
	maxKey      []byte
	numEntries  int
	dataEnd     int64 // Offset where data section ends and footer begins
}

// SSTableIndex is a sparse index for SSTable
// Maps key -> file offset for efficient seeks
type SSTableIndex struct {
	entries []IndexEntry
}

// IndexEntry represents an entry in the sparse index
type IndexEntry struct {
	Key    []byte
	Offset int64
}

// SSTableWriter writes a new SSTable from sorted entries
type SSTableWriter struct {
	file         *os.File
	path         string
	index        []IndexEntry
	bloomFilter  *BloomFilter
	minKey       []byte
	maxKey       []byte
	numEntries   int
	currentOffset int64
	indexInterval int // Write index entry every N keys
}

// NewSSTableWriter creates a new SSTable writer
func NewSSTableWriter(dir string, id int, indexInterval int) (*SSTableWriter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	path := filepath.Join(dir, fmt.Sprintf("sstable_%d.sst", id))
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create sstable file: %w", err)
	}

	return &SSTableWriter{
		file:          file,
		path:          path,
		index:         make([]IndexEntry, 0),
		bloomFilter:   NewBloomFilter(10000, 3), // 10K entries, 3 hash functions
		numEntries:    0,
		currentOffset: 0,
		indexInterval: indexInterval,
	}, nil
}

// Write writes an entry to the SSTable
// Entries must be written in sorted order
func (w *SSTableWriter) Write(entry *MemTableEntry) error {
	// Track min/max keys
	if w.minKey == nil {
		w.minKey = append([]byte(nil), entry.Key...)
	}
	w.maxKey = append([]byte(nil), entry.Key...)

	// Add to bloom filter
	w.bloomFilter.Add(entry.Key)

	// Write entry to file
	// Format: keyLen(4) | key | valueLen(4) | value | timestamp(8) | deleted(1)
	buf := new(bytes.Buffer)

	// Key length and key
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(entry.Key))); err != nil {
		return err
	}
	buf.Write(entry.Key)

	// Value length and value
	valueLen := uint32(0)
	if entry.Value != nil {
		valueLen = uint32(len(entry.Value))
	}
	if err := binary.Write(buf, binary.LittleEndian, valueLen); err != nil {
		return err
	}
	if valueLen > 0 {
		buf.Write(entry.Value)
	}

	// Timestamp
	if err := binary.Write(buf, binary.LittleEndian, entry.Timestamp); err != nil {
		return err
	}

	// Deleted flag
	deletedByte := byte(0)
	if entry.Deleted {
		deletedByte = 1
	}
	if err := buf.WriteByte(deletedByte); err != nil {
		return err
	}

	// Write to file
	n, err := w.file.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}

	// Add to sparse index
	if w.numEntries%w.indexInterval == 0 {
		w.index = append(w.index, IndexEntry{
			Key:    append([]byte(nil), entry.Key...),
			Offset: w.currentOffset,
		})
	}

	w.currentOffset += int64(n)
	w.numEntries++

	return nil
}

// Finalize writes metadata and closes the SSTable
func (w *SSTableWriter) Finalize() (*SSTable, error) {
	// Write metadata footer
	// Format: numEntries(4) | minKeyLen(4) | minKey | maxKeyLen(4) | maxKey | numIndexEntries(4) | indexEntries | bloomFilterSize(4) | bloomFilter

	footer := new(bytes.Buffer)

	// Number of entries
	if err := binary.Write(footer, binary.LittleEndian, uint32(w.numEntries)); err != nil {
		return nil, err
	}

	// Min key
	if err := binary.Write(footer, binary.LittleEndian, uint32(len(w.minKey))); err != nil {
		return nil, err
	}
	footer.Write(w.minKey)

	// Max key
	if err := binary.Write(footer, binary.LittleEndian, uint32(len(w.maxKey))); err != nil {
		return nil, err
	}
	footer.Write(w.maxKey)

	// Index entries
	if err := binary.Write(footer, binary.LittleEndian, uint32(len(w.index))); err != nil {
		return nil, err
	}
	for _, entry := range w.index {
		if err := binary.Write(footer, binary.LittleEndian, uint32(len(entry.Key))); err != nil {
			return nil, err
		}
		footer.Write(entry.Key)
		if err := binary.Write(footer, binary.LittleEndian, entry.Offset); err != nil {
			return nil, err
		}
	}

	// Bloom filter
	bloomData := w.bloomFilter.Marshal()
	if err := binary.Write(footer, binary.LittleEndian, uint32(len(bloomData))); err != nil {
		return nil, err
	}
	footer.Write(bloomData)

	// Footer size (for reading footer from end of file)
	footerSize := uint32(footer.Len())
	if err := binary.Write(footer, binary.LittleEndian, footerSize); err != nil {
		return nil, err
	}

	// Write footer
	if _, err := w.file.Write(footer.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to write footer: %w", err)
	}

	// Sync and close
	if err := w.file.Sync(); err != nil {
		return nil, fmt.Errorf("failed to sync file: %w", err)
	}
	if err := w.file.Close(); err != nil {
		return nil, fmt.Errorf("failed to close file: %w", err)
	}

	// Return SSTable
	dataEnd := w.currentOffset // Save where data ends before footer
	return &SSTable{
		path:        w.path,
		index:       &SSTableIndex{entries: w.index},
		bloomFilter: w.bloomFilter,
		minKey:      w.minKey,
		maxKey:      w.maxKey,
		numEntries:  w.numEntries,
		dataEnd:     dataEnd,
	}, nil
}

// OpenSSTable opens an existing SSTable
func OpenSSTable(path string) (*SSTable, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sstable: %w", err)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	fileSize := stat.Size()

	// Read footer size (last 4 bytes)
	if _, err := file.Seek(fileSize-4, io.SeekStart); err != nil {
		return nil, err
	}
	var footerSize uint32
	if err := binary.Read(file, binary.LittleEndian, &footerSize); err != nil {
		return nil, err
	}

	// Read footer
	footerStart := fileSize - int64(footerSize) - 4
	if _, err := file.Seek(footerStart, io.SeekStart); err != nil {
		return nil, err
	}

	// Parse footer
	var numEntries uint32
	if err := binary.Read(file, binary.LittleEndian, &numEntries); err != nil {
		return nil, err
	}

	// Min key
	var minKeyLen uint32
	if err := binary.Read(file, binary.LittleEndian, &minKeyLen); err != nil {
		return nil, err
	}
	minKey := make([]byte, minKeyLen)
	if _, err := io.ReadFull(file, minKey); err != nil {
		return nil, err
	}

	// Max key
	var maxKeyLen uint32
	if err := binary.Read(file, binary.LittleEndian, &maxKeyLen); err != nil {
		return nil, err
	}
	maxKey := make([]byte, maxKeyLen)
	if _, err := io.ReadFull(file, maxKey); err != nil {
		return nil, err
	}

	// Index entries
	var numIndexEntries uint32
	if err := binary.Read(file, binary.LittleEndian, &numIndexEntries); err != nil {
		return nil, err
	}
	indexEntries := make([]IndexEntry, numIndexEntries)
	for i := uint32(0); i < numIndexEntries; i++ {
		var keyLen uint32
		if err := binary.Read(file, binary.LittleEndian, &keyLen); err != nil {
			return nil, err
		}
		key := make([]byte, keyLen)
		if _, err := io.ReadFull(file, key); err != nil {
			return nil, err
		}
		var offset int64
		if err := binary.Read(file, binary.LittleEndian, &offset); err != nil {
			return nil, err
		}
		indexEntries[i] = IndexEntry{Key: key, Offset: offset}
	}

	// Bloom filter
	var bloomSize uint32
	if err := binary.Read(file, binary.LittleEndian, &bloomSize); err != nil {
		return nil, err
	}
	bloomData := make([]byte, bloomSize)
	if _, err := io.ReadFull(file, bloomData); err != nil {
		return nil, err
	}
	bloomFilter, err := UnmarshalBloomFilter(bloomData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal bloom filter: %w", err)
	}

	// Data ends where footer starts
	dataEnd := footerStart

	return &SSTable{
		path:        path,
		index:       &SSTableIndex{entries: indexEntries},
		bloomFilter: bloomFilter,
		minKey:      minKey,
		maxKey:      maxKey,
		numEntries:  int(numEntries),
		dataEnd:     dataEnd,
	}, nil
}

// Get retrieves a value from the SSTable
func (sst *SSTable) Get(key []byte) (*MemTableEntry, bool, error) {
	// Check bloom filter first
	if !sst.bloomFilter.Contains(key) {
		return nil, false, nil
	}

	// Check if key is in range
	if bytes.Compare(key, sst.minKey) < 0 || bytes.Compare(key, sst.maxKey) > 0 {
		return nil, false, nil
	}

	// Find nearest index entry
	idx := sort.Search(len(sst.index.entries), func(i int) bool {
		return bytes.Compare(sst.index.entries[i].Key, key) > 0
	})
	if idx > 0 {
		idx--
	}

	// Open file and seek to offset
	file, err := os.Open(sst.path)
	if err != nil {
		return nil, false, fmt.Errorf("failed to open sstable: %w", err)
	}
	defer file.Close()

	offset := int64(0)
	if idx < len(sst.index.entries) {
		offset = sst.index.entries[idx].Offset
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, false, err
	}

	// Scan from offset to find key (don't read past dataEnd)
	for {
		// Check if we're at or past the footer
		currentPos, err := file.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, false, err
		}
		if currentPos >= sst.dataEnd {
			return nil, false, nil // Reached end of data section
		}

		entry, err := readEntry(file)
		if err != nil {
			if err == io.EOF {
				return nil, false, nil
			}
			return nil, false, err
		}

		cmp := bytes.Compare(entry.Key, key)
		if cmp == 0 {
			return entry, true, nil
		}
		if cmp > 0 {
			return nil, false, nil // Key not found
		}
	}
}

// readEntry reads a single entry from file
func readEntry(r io.Reader) (*MemTableEntry, error) {
	var keyLen uint32
	if err := binary.Read(r, binary.LittleEndian, &keyLen); err != nil {
		return nil, err
	}

	key := make([]byte, keyLen)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}

	var valueLen uint32
	if err := binary.Read(r, binary.LittleEndian, &valueLen); err != nil {
		return nil, err
	}

	var value []byte
	if valueLen > 0 {
		value = make([]byte, valueLen)
		if _, err := io.ReadFull(r, value); err != nil {
			return nil, err
		}
	}

	var timestamp int64
	if err := binary.Read(r, binary.LittleEndian, &timestamp); err != nil {
		return nil, err
	}

	deletedByte, err := readByte(r)
	if err != nil {
		return nil, err
	}
	deleted := deletedByte == 1

	return &MemTableEntry{
		Key:       key,
		Value:     value,
		Timestamp: timestamp,
		Deleted:   deleted,
	}, nil
}

// readByte reads a single byte
func readByte(r io.Reader) (byte, error) {
	buf := make([]byte, 1)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return buf[0], nil
}

// Iterator returns an iterator over all entries
func (sst *SSTable) Iterator() (*SSTableIterator, error) {
	file, err := os.Open(sst.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sstable: %w", err)
	}

	return &SSTableIterator{
		file:    file,
		current: nil,
		dataEnd: sst.dataEnd,
	}, nil
}

// SSTableIterator iterates over entries in an SSTable
type SSTableIterator struct {
	file    *os.File
	current *MemTableEntry
	dataEnd int64
}

// Next advances the iterator
func (it *SSTableIterator) Next() bool {
	// Check if we're at or past the footer
	currentPos, err := it.file.Seek(0, io.SeekCurrent)
	if err != nil || currentPos >= it.dataEnd {
		it.current = nil
		return false
	}

	entry, err := readEntry(it.file)
	if err != nil {
		it.current = nil
		return false
	}
	it.current = entry
	return true
}

// Entry returns the current entry
func (it *SSTableIterator) Entry() *MemTableEntry {
	return it.current
}

// Close closes the iterator
func (it *SSTableIterator) Close() error {
	return it.file.Close()
}
