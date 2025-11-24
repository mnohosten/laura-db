package lsm

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestLSMTreeSimple(t *testing.T) {
	dir := t.TempDir()
	config := DefaultConfig(dir)
	config.MemTableSize = 256

	lsm, _ := NewLSMTree(config)
	defer lsm.Close()

	// Insert a few keys
	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("k%02d", i))
		val := []byte(fmt.Sprintf("v%02d", i))
		lsm.Put(key, val)
	}

	stats1 := lsm.Stats()
	t.Logf("Before flush: %+v", stats1)

	lsm.Flush()

	stats2 := lsm.Stats()
	t.Logf("After flush: %+v", stats2)

	// Try to get first key
	val, found, err := lsm.Get([]byte("k00"))
	t.Logf("Get k00: found=%v, err=%v, val=%s", found, err, val)

	if !found {
		t.Fatal("key k00 not found")
	}
}

func TestLSMTreeBasicOperations(t *testing.T) {
	dir := t.TempDir()
	config := DefaultConfig(dir)
	config.MemTableSize = 1024 // Small size to trigger flushes

	lsm, err := NewLSMTree(config)
	if err != nil {
		t.Fatalf("failed to create LSM tree: %v", err)
	}
	defer lsm.Close()

	// Test Put and Get
	key := []byte("test-key")
	value := []byte("test-value")

	if err := lsm.Put(key, value); err != nil {
		t.Fatalf("failed to put: %v", err)
	}

	got, found, err := lsm.Get(key)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if !found {
		t.Fatal("key not found")
	}
	if !bytes.Equal(got, value) {
		t.Fatalf("expected %s, got %s", value, got)
	}
}

func TestLSMTreeDelete(t *testing.T) {
	dir := t.TempDir()
	lsm, err := NewLSMTree(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("failed to create LSM tree: %v", err)
	}
	defer lsm.Close()

	key := []byte("key-to-delete")
	value := []byte("value")

	// Put and verify
	lsm.Put(key, value)
	if _, found, _ := lsm.Get(key); !found {
		t.Fatal("key should exist")
	}

	// Delete and verify
	if err := lsm.Delete(key); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	if _, found, _ := lsm.Get(key); found {
		t.Fatal("key should not exist after delete")
	}
}

func TestLSMTreeMemTableFlush(t *testing.T) {
	dir := t.TempDir()
	config := DefaultConfig(dir)
	config.MemTableSize = 512 // Very small to trigger flush

	lsm, err := NewLSMTree(config)
	if err != nil {
		t.Fatalf("failed to create LSM tree: %v", err)
	}
	defer lsm.Close()

	// Insert enough data to trigger flush
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key-%04d", i))
		value := []byte(fmt.Sprintf("value-%04d", i))
		if err := lsm.Put(key, value); err != nil {
			t.Fatalf("failed to put: %v", err)
		}
	}

	// Wait for flush to complete
	if err := lsm.Flush(); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}

	// Verify data is still accessible
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key-%04d", i))
		expectedValue := []byte(fmt.Sprintf("value-%04d", i))

		got, found, err := lsm.Get(key)
		if err != nil {
			t.Fatalf("failed to get key %s: %v", key, err)
		}
		if !found {
			t.Fatalf("key %s not found", key)
		}
		if !bytes.Equal(got, expectedValue) {
			t.Fatalf("key %s: expected %s, got %s", key, expectedValue, got)
		}
	}

	// Check that SSTables were created
	stats := lsm.Stats()
	numSSTables := stats["num_sstables"].(int)
	if numSSTables == 0 {
		t.Fatal("expected SSTables to be created")
	}
}

func TestLSMTreeCompaction(t *testing.T) {
	dir := t.TempDir()
	config := DefaultConfig(dir)
	config.MemTableSize = 256 // Very small

	lsm, err := NewLSMTree(config)
	if err != nil {
		t.Fatalf("failed to create LSM tree: %v", err)
	}
	defer lsm.Close()

	// Insert enough data to trigger multiple flushes and compaction
	for i := 0; i < 200; i++ {
		key := []byte(fmt.Sprintf("key-%04d", i))
		value := []byte(fmt.Sprintf("value-%04d", i))
		lsm.Put(key, value)
	}

	// Wait for flushes and compaction
	if err := lsm.Flush(); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	time.Sleep(200 * time.Millisecond) // Wait for potential compaction

	// Verify all data is still accessible
	for i := 0; i < 200; i++ {
		key := []byte(fmt.Sprintf("key-%04d", i))
		expectedValue := []byte(fmt.Sprintf("value-%04d", i))

		got, found, err := lsm.Get(key)
		if err != nil {
			t.Fatalf("failed to get key %s: %v", key, err)
		}
		if !found {
			t.Fatalf("key %s not found", key)
		}
		if !bytes.Equal(got, expectedValue) {
			t.Fatalf("key %s: expected %s, got %s", key, expectedValue, got)
		}
	}

	// Compaction should have reduced number of SSTables
	stats := lsm.Stats()
	t.Logf("Stats after compaction: %+v", stats)
}

func TestLSMTreePersistence(t *testing.T) {
	dir := t.TempDir()
	config := DefaultConfig(dir)

	// Create and populate LSM tree
	lsm, err := NewLSMTree(config)
	if err != nil {
		t.Fatalf("failed to create LSM tree: %v", err)
	}

	for i := 0; i < 50; i++ {
		key := []byte(fmt.Sprintf("persist-key-%04d", i))
		value := []byte(fmt.Sprintf("persist-value-%04d", i))
		lsm.Put(key, value)
	}

	// Flush and close
	if err := lsm.Flush(); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	if err := lsm.Close(); err != nil {
		t.Fatalf("failed to close LSM tree: %v", err)
	}

	// Reopen and verify data
	lsm, err = NewLSMTree(config)
	if err != nil {
		t.Fatalf("failed to reopen LSM tree: %v", err)
	}
	defer lsm.Close()

	for i := 0; i < 50; i++ {
		key := []byte(fmt.Sprintf("persist-key-%04d", i))
		expectedValue := []byte(fmt.Sprintf("persist-value-%04d", i))

		got, found, err := lsm.Get(key)
		if err != nil {
			t.Fatalf("failed to get key %s: %v", key, err)
		}
		if !found {
			t.Fatalf("key %s not found after reopen", key)
		}
		if !bytes.Equal(got, expectedValue) {
			t.Fatalf("key %s: expected %s, got %s", key, expectedValue, got)
		}
	}
}

func TestLSMTreeUpdate(t *testing.T) {
	dir := t.TempDir()
	lsm, err := NewLSMTree(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("failed to create LSM tree: %v", err)
	}
	defer lsm.Close()

	key := []byte("update-key")
	value1 := []byte("value-1")
	value2 := []byte("value-2")

	// Insert initial value
	lsm.Put(key, value1)
	got, _, _ := lsm.Get(key)
	if !bytes.Equal(got, value1) {
		t.Fatalf("expected %s, got %s", value1, got)
	}

	// Update value
	lsm.Put(key, value2)
	got, _, _ = lsm.Get(key)
	if !bytes.Equal(got, value2) {
		t.Fatalf("expected %s, got %s", value2, got)
	}
}

func TestLSMTreeNotFound(t *testing.T) {
	dir := t.TempDir()
	lsm, err := NewLSMTree(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("failed to create LSM tree: %v", err)
	}
	defer lsm.Close()

	key := []byte("nonexistent-key")
	_, found, err := lsm.Get(key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatal("nonexistent key should not be found")
	}
}

func TestLSMTreeStats(t *testing.T) {
	dir := t.TempDir()
	lsm, err := NewLSMTree(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("failed to create LSM tree: %v", err)
	}
	defer lsm.Close()

	// Insert some data
	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("stats-key-%d", i))
		value := []byte(fmt.Sprintf("stats-value-%d", i))
		lsm.Put(key, value)
	}

	stats := lsm.Stats()
	if stats["memtable_size"].(int64) == 0 {
		t.Fatal("memtable size should be > 0")
	}

	t.Logf("LSM Stats: %+v", stats)
}

func TestLSMTreeClosed(t *testing.T) {
	dir := t.TempDir()
	lsm, err := NewLSMTree(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("failed to create LSM tree: %v", err)
	}

	lsm.Close()

	// Operations on closed LSM should fail
	err = lsm.Put([]byte("key"), []byte("value"))
	if err != ErrClosed {
		t.Fatalf("expected ErrClosed, got %v", err)
	}

	_, _, err = lsm.Get([]byte("key"))
	if err != ErrClosed {
		t.Fatalf("expected ErrClosed, got %v", err)
	}

	err = lsm.Delete([]byte("key"))
	if err != ErrClosed {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestLSMTreeLoadExistingSSTables(t *testing.T) {
	dir := t.TempDir()
	config := DefaultConfig(dir)

	// Create first LSM tree and write data
	lsm1, err := NewLSMTree(config)
	if err != nil {
		t.Fatalf("failed to create LSM tree: %v", err)
	}

	for i := 0; i < 30; i++ {
		key := []byte(fmt.Sprintf("load-key-%04d", i))
		value := []byte(fmt.Sprintf("load-value-%04d", i))
		lsm1.Put(key, value)
	}

	if err := lsm1.Flush(); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	lsm1.Close()

	// Check that SSTable files exist
	matches, _ := filepath.Glob(filepath.Join(dir, "sstable_*.sst"))
	if len(matches) == 0 {
		t.Fatal("no sstable files created")
	}

	// Create second LSM tree - should load existing SSTables
	lsm2, err := NewLSMTree(config)
	if err != nil {
		t.Fatalf("failed to create second LSM tree: %v", err)
	}
	defer lsm2.Close()

	// Verify data is loaded
	for i := 0; i < 30; i++ {
		key := []byte(fmt.Sprintf("load-key-%04d", i))
		expectedValue := []byte(fmt.Sprintf("load-value-%04d", i))

		got, found, err := lsm2.Get(key)
		if err != nil {
			t.Fatalf("failed to get key %s: %v", key, err)
		}
		if !found {
			t.Fatalf("key %s not found after reload", key)
		}
		if !bytes.Equal(got, expectedValue) {
			t.Fatalf("key %s: expected %s, got %s", key, expectedValue, got)
		}
	}

	stats := lsm2.Stats()
	if stats["num_sstables"].(int) == 0 {
		t.Fatal("expected SSTables to be loaded")
	}
}
