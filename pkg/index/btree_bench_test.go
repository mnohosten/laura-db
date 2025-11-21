package index

import (
	"fmt"
	"testing"
)

// BenchmarkBTreeInsert benchmarks B+ tree insertion
func BenchmarkBTreeInsert(b *testing.B) {
	// Create fresh index for each run to avoid duplicate keys
	b.StopTimer()
	idx := NewIndex(&IndexConfig{
		Name:      "bench_idx",
		FieldPath: "field",
		Type:      IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		// Use unique keys to avoid conflicts
		key := fmt.Sprintf("key_%d_%d", b.N, i)
		value := fmt.Sprintf("doc_%d", i)
		err := idx.Insert(key, value)
		if err != nil {
			b.Fatalf("Insert failed: %v", err)
		}
	}
}

// BenchmarkBTreeSearch benchmarks B+ tree search
func BenchmarkBTreeSearch(b *testing.B) {
	idx := NewIndex(&IndexConfig{
		Name:      "bench_idx",
		FieldPath: "field",
		Type:      IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Insert test data
	for i := 0; i < 10000; i++ {
		key := i
		value := fmt.Sprintf("doc_%d", i)
		idx.Insert(key, value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := i % 10000
		_, exists := idx.Search(key)
		if !exists {
			b.Fatalf("Key %d not found", key)
		}
	}
}

// BenchmarkBTreeRangeScan benchmarks range scans
func BenchmarkBTreeRangeScan(b *testing.B) {
	idx := NewIndex(&IndexConfig{
		Name:      "bench_idx",
		FieldPath: "field",
		Type:      IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Insert test data
	for i := 0; i < 10000; i++ {
		key := i
		value := fmt.Sprintf("doc_%d", i)
		idx.Insert(key, value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := (i % 9000)
		end := start + 1000
		_, _ = idx.RangeScan(start, end)
	}
}

// BenchmarkBTreeDelete benchmarks deletion
func BenchmarkBTreeDelete(b *testing.B) {
	idx := NewIndex(&IndexConfig{
		Name:      "bench_idx",
		FieldPath: "field",
		Type:      IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Pre-populate with enough data
	for i := 0; i < b.N+1000; i++ {
		idx.Insert(i, fmt.Sprintf("doc_%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Delete different keys
		key := i
		idx.Delete(key)
	}
}

// BenchmarkBTreeMixedOperations benchmarks mixed workload
func BenchmarkBTreeMixedOperations(b *testing.B) {
	idx := NewIndex(&IndexConfig{
		Name:      "bench_idx",
		FieldPath: "field",
		Type:      IndexTypeBTree,
		Unique:    false,
		Order:     32,
	})

	// Pre-populate
	for i := 0; i < 5000; i++ {
		idx.Insert(i, fmt.Sprintf("doc_%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		op := i % 4
		key := i % 10000

		switch op {
		case 0: // Insert
			idx.Insert(key, fmt.Sprintf("doc_%d", key))
		case 1: // Search
			idx.Search(key)
		case 2: // Range scan (small)
			idx.RangeScan(key, key+100)
		case 3: // Delete
			idx.Delete(key)
		}
	}
}
