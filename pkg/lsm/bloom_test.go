package lsm

import (
	"fmt"
	"testing"
)

func TestBloomFilterBasic(t *testing.T) {
	bf := NewBloomFilter(1000, 3)

	// Add some keys
	keys := [][]byte{
		[]byte("apple"),
		[]byte("banana"),
		[]byte("cherry"),
		[]byte("date"),
	}

	for _, key := range keys {
		bf.Add(key)
	}

	// All added keys should be found
	for _, key := range keys {
		if !bf.Contains(key) {
			t.Fatalf("key %s should be in bloom filter", key)
		}
	}
}

func TestBloomFilterFalseNegatives(t *testing.T) {
	bf := NewBloomFilter(1000, 3)

	// Add some keys
	bf.Add([]byte("key1"))
	bf.Add([]byte("key2"))

	// Keys that were added should always be found (no false negatives)
	if !bf.Contains([]byte("key1")) {
		t.Fatal("false negative: key1 should be found")
	}
	if !bf.Contains([]byte("key2")) {
		t.Fatal("false negative: key2 should be found")
	}
}

func TestBloomFilterFalsePositives(t *testing.T) {
	bf := NewBloomFilter(100, 3) // Small size to increase false positive rate

	// Add many keys
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		bf.Add(key)
	}

	// Check for keys that were not added
	falsePositives := 0
	testKeys := 1000

	for i := 1000; i < 1000+testKeys; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		if bf.Contains(key) {
			falsePositives++
		}
	}

	// Should have some false positives, but not too many
	fpr := float64(falsePositives) / float64(testKeys)
	if fpr > 0.5 {
		t.Fatalf("false positive rate too high: %.2f%%", fpr*100)
	}

	t.Logf("False positive rate: %.2f%% (%d/%d)", fpr*100, falsePositives, testKeys)
}

func TestBloomFilterMarshalUnmarshal(t *testing.T) {
	bf := NewBloomFilter(1000, 3)

	// Add some keys
	keys := [][]byte{
		[]byte("test1"),
		[]byte("test2"),
		[]byte("test3"),
	}

	for _, key := range keys {
		bf.Add(key)
	}

	// Marshal
	data := bf.Marshal()

	// Unmarshal
	bf2, err := UnmarshalBloomFilter(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify keys are still found
	for _, key := range keys {
		if !bf2.Contains(key) {
			t.Fatalf("key %s not found after unmarshal", key)
		}
	}

	// Verify parameters match
	if bf2.size != bf.size {
		t.Fatalf("size mismatch: %d != %d", bf2.size, bf.size)
	}
	if bf2.numHashes != bf.numHashes {
		t.Fatalf("numHashes mismatch: %d != %d", bf2.numHashes, bf.numHashes)
	}
}

func TestBloomFilterEmpty(t *testing.T) {
	bf := NewBloomFilter(1000, 3)

	// Empty bloom filter should not contain any key
	if bf.Contains([]byte("any-key")) {
		t.Fatal("empty bloom filter should not contain any key")
	}
}

func TestBloomFilterStats(t *testing.T) {
	bf := NewBloomFilter(1000, 3)

	// Add some keys
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		bf.Add(key)
	}

	stats := bf.Stats()

	if stats["size"].(int) != 10000 {
		t.Fatalf("expected size 10000, got %d", stats["size"])
	}

	if stats["num_hashes"].(int) != 3 {
		t.Fatalf("expected 3 hashes, got %d", stats["num_hashes"])
	}

	fillRatio := stats["fill_ratio"].(float64)
	if fillRatio <= 0 || fillRatio >= 1 {
		t.Fatalf("invalid fill ratio: %.2f", fillRatio)
	}

	t.Logf("Bloom filter stats: %+v", stats)
}

func TestBloomFilterInvalidUnmarshal(t *testing.T) {
	// Too short
	_, err := UnmarshalBloomFilter([]byte{1, 2, 3})
	if err != ErrInvalidBloomFilter {
		t.Fatalf("expected ErrInvalidBloomFilter, got %v", err)
	}
}
