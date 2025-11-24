package lsm

import (
	"bytes"
	"fmt"
	"testing"
)

func TestSSTableWriteAndRead(t *testing.T) {
	dir := t.TempDir()

	// Create writer
	writer, err := NewSSTableWriter(dir, 1, 10)
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}

	// Write some entries
	entries := []*MemTableEntry{
		{Key: []byte("apple"), Value: []byte("red"), Timestamp: 1, Deleted: false},
		{Key: []byte("banana"), Value: []byte("yellow"), Timestamp: 2, Deleted: false},
		{Key: []byte("cherry"), Value: []byte("red"), Timestamp: 3, Deleted: false},
	}

	for _, entry := range entries {
		if err := writer.Write(entry); err != nil {
			t.Fatalf("failed to write entry: %v", err)
		}
	}

	// Finalize
	sst, err := writer.Finalize()
	if err != nil {
		t.Fatalf("failed to finalize: %v", err)
	}

	t.Logf("SSTable created: path=%s, entries=%d, dataEnd=%d", sst.path, sst.numEntries, sst.dataEnd)

	// Reopen
	sst2, err := OpenSSTable(sst.path)
	if err != nil {
		t.Fatalf("failed to open sstable: %v", err)
	}

	t.Logf("SSTable opened: path=%s, entries=%d, dataEnd=%d", sst2.path, sst2.numEntries, sst2.dataEnd)

	// Test Get
	for _, expected := range entries {
		got, found, err := sst2.Get(expected.Key)
		if err != nil {
			t.Fatalf("failed to get key %s: %v", expected.Key, err)
		}
		if !found {
			t.Fatalf("key %s not found", expected.Key)
		}
		if !bytes.Equal(got.Value, expected.Value) {
			t.Fatalf("key %s: expected value %s, got %s", expected.Key, expected.Value, got.Value)
		}
	}

	// Test iterator
	iter, err := sst2.Iterator()
	if err != nil {
		t.Fatalf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	for iter.Next() {
		entry := iter.Entry()
		t.Logf("Iterator entry: %s = %s", entry.Key, entry.Value)
		count++
	}

	if count != len(entries) {
		t.Fatalf("expected %d entries from iterator, got %d", len(entries), count)
	}
}

func TestSSTableBloomFilter(t *testing.T) {
	dir := t.TempDir()

	writer, err := NewSSTableWriter(dir, 1, 10)
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}

	// Write entries
	for i := 0; i < 100; i++ {
		entry := &MemTableEntry{
			Key:       []byte(fmt.Sprintf("key-%04d", i)),
			Value:     []byte(fmt.Sprintf("value-%04d", i)),
			Timestamp: int64(i),
			Deleted:   false,
		}
		writer.Write(entry)
	}

	sst, err := writer.Finalize()
	if err != nil {
		t.Fatalf("failed to finalize: %v", err)
	}

	// Test bloom filter prevents unnecessary lookups
	_, found, err := sst.Get([]byte("nonexistent-key"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatal("nonexistent key should not be found")
	}

	// Bloom filter stats
	stats := sst.bloomFilter.Stats()
	t.Logf("Bloom filter stats: %+v", stats)
}
