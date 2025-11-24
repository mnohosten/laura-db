package storage

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

// TestBufferPoolConcurrentReads tests that multiple goroutines can read concurrently
func TestBufferPoolConcurrentReads(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := tempDir + "/test.db"

	// Create disk manager and buffer pool
	diskMgr, err := NewDiskManager(dbFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bufferPool := NewBufferPool(100, diskMgr)

	// Create a page and write it to disk
	page, err := bufferPool.NewPage()
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}
	pageID := page.ID
	copy(page.Data[:], []byte("test data"))
	page.MarkDirty()

	if err := bufferPool.UnpinPage(pageID, true); err != nil {
		t.Fatalf("Failed to unpin page: %v", err)
	}
	if err := bufferPool.FlushPage(pageID); err != nil {
		t.Fatalf("Failed to flush page: %v", err)
	}

	// Concurrent reads
	const numReaders = 100
	const readsPerReader = 100
	var wg sync.WaitGroup
	errors := make(chan error, numReaders)

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < readsPerReader; j++ {
				p, err := bufferPool.FetchPage(pageID)
				if err != nil {
					errors <- fmt.Errorf("reader %d: failed to fetch page: %w", readerID, err)
					return
				}
				// Read data
				_ = p.Data[0]
				if err := bufferPool.UnpinPage(pageID, false); err != nil {
					errors <- fmt.Errorf("reader %d: failed to unpin page: %w", readerID, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify high hit rate (most reads should be from cache)
	stats := bufferPool.Stats()
	hitRate := stats["hit_rate"].(float64)
	if hitRate < 99.0 {
		t.Errorf("Expected hit rate > 99%%, got %.2f%%", hitRate)
	}
}

// TestBufferPoolMixedWorkload tests concurrent reads and writes
func TestBufferPoolMixedWorkload(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := tempDir + "/test.db"

	diskMgr, err := NewDiskManager(dbFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bufferPool := NewBufferPool(50, diskMgr)

	// Create initial pages
	const numPages = 10
	pageIDs := make([]PageID, numPages)
	for i := 0; i < numPages; i++ {
		page, err := bufferPool.NewPage()
		if err != nil {
			t.Fatalf("Failed to create page %d: %v", i, err)
		}
		pageIDs[i] = page.ID
		copy(page.Data[:], []byte(fmt.Sprintf("page-%d", i)))
		page.MarkDirty()
		if err := bufferPool.UnpinPage(page.ID, true); err != nil {
			t.Fatalf("Failed to unpin page %d: %v", i, err)
		}
		if err := bufferPool.FlushPage(page.ID); err != nil {
			t.Fatalf("Failed to flush page %d: %v", i, err)
		}
	}

	// Mixed workload: 80% reads, 20% writes
	// Each worker gets its own page to avoid data races
	const numWorkers = 10 // Reduced to match numPages
	const opsPerWorker = 100
	var wg sync.WaitGroup
	errors := make(chan error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			// Each worker has its own dedicated page
			pageID := pageIDs[workerID%numPages]

			for j := 0; j < opsPerWorker; j++ {
				// 80% reads, 20% writes
				if j%5 == 0 {
					// Write operation
					p, err := bufferPool.FetchPage(pageID)
					if err != nil {
						errors <- fmt.Errorf("worker %d: failed to fetch page for write: %w", workerID, err)
						return
					}
					copy(p.Data[:], []byte(fmt.Sprintf("updated-by-%d", workerID)))
					p.MarkDirty()
					if err := bufferPool.UnpinPage(pageID, true); err != nil {
						errors <- fmt.Errorf("worker %d: failed to unpin page after write: %w", workerID, err)
						return
					}
				} else {
					// Read operation
					p, err := bufferPool.FetchPage(pageID)
					if err != nil {
						errors <- fmt.Errorf("worker %d: failed to fetch page for read: %w", workerID, err)
						return
					}
					_ = p.Data[0]
					if err := bufferPool.UnpinPage(pageID, false); err != nil {
						errors <- fmt.Errorf("worker %d: failed to unpin page after read: %w", workerID, err)
						return
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify reasonable hit rate
	stats := bufferPool.Stats()
	hitRate := stats["hit_rate"].(float64)
	if hitRate < 90.0 {
		t.Errorf("Expected hit rate > 90%%, got %.2f%%", hitRate)
	}
}

// TestBufferPoolLockUpgrade tests the lock upgrade path
func TestBufferPoolLockUpgrade(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := tempDir + "/test.db"

	diskMgr, err := NewDiskManager(dbFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bufferPool := NewBufferPool(10, diskMgr)

	// Create a page
	page, err := bufferPool.NewPage()
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}
	pageID := page.ID
	if err := bufferPool.UnpinPage(pageID, false); err != nil {
		t.Fatalf("Failed to unpin page: %v", err)
	}

	// Test that lock upgrade works correctly when page is in pool
	for i := 0; i < 100; i++ {
		p, err := bufferPool.FetchPage(pageID)
		if err != nil {
			t.Fatalf("Iteration %d: failed to fetch page: %v", i, err)
		}
		if err := bufferPool.UnpinPage(pageID, false); err != nil {
			t.Fatalf("Iteration %d: failed to unpin page: %v", i, err)
		}
		_ = p
	}

	// Verify we got all hits (no misses after first fetch)
	stats := bufferPool.Stats()
	hits := stats["hits"].(int)
	if hits != 100 {
		t.Errorf("Expected 100 hits, got %d", hits)
	}
}

// TestBufferPoolEvictionUnderContention tests eviction under concurrent load
func TestBufferPoolEvictionUnderContention(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := tempDir + "/test.db"

	diskMgr, err := NewDiskManager(dbFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Small buffer pool to force evictions
	bufferPool := NewBufferPool(5, diskMgr)

	// Create more pages than buffer capacity
	const numPages = 20
	pageIDs := make([]PageID, numPages)
	for i := 0; i < numPages; i++ {
		page, err := bufferPool.NewPage()
		if err != nil {
			t.Fatalf("Failed to create page %d: %v", i, err)
		}
		pageIDs[i] = page.ID
		copy(page.Data[:], []byte(fmt.Sprintf("page-%d", i)))
		if err := bufferPool.UnpinPage(page.ID, true); err != nil {
			t.Fatalf("Failed to unpin page %d: %v", i, err)
		}
	}

	// Concurrent access to different pages
	const numWorkers = 10
	const opsPerWorker = 50
	var wg sync.WaitGroup
	errors := make(chan error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < opsPerWorker; j++ {
				pageIdx := (workerID*opsPerWorker + j) % numPages
				pageID := pageIDs[pageIdx]

				p, err := bufferPool.FetchPage(pageID)
				if err != nil {
					errors <- fmt.Errorf("worker %d: failed to fetch page: %w", workerID, err)
					return
				}
				_ = p.Data[0]
				if err := bufferPool.UnpinPage(pageID, false); err != nil {
					errors <- fmt.Errorf("worker %d: failed to unpin page: %w", workerID, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify evictions occurred
	stats := bufferPool.Stats()
	evictions := stats["evictions"].(int)
	if evictions == 0 {
		t.Error("Expected some evictions to occur")
	}
}

// BenchmarkBufferPoolConcurrentReads benchmarks concurrent read performance
func BenchmarkBufferPoolConcurrentReads(b *testing.B) {
	tempDir := b.TempDir()
	dbFile := tempDir + "/bench.db"

	diskMgr, err := NewDiskManager(dbFile)
	if err != nil {
		b.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bufferPool := NewBufferPool(100, diskMgr)

	// Create pages
	const numPages = 10
	pageIDs := make([]PageID, numPages)
	for i := 0; i < numPages; i++ {
		page, err := bufferPool.NewPage()
		if err != nil {
			b.Fatalf("Failed to create page: %v", err)
		}
		pageIDs[i] = page.ID
		if err := bufferPool.UnpinPage(page.ID, false); err != nil {
			b.Fatalf("Failed to unpin page: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			pageID := pageIDs[i%numPages]
			p, err := bufferPool.FetchPage(pageID)
			if err != nil {
				b.Fatalf("Failed to fetch page: %v", err)
			}
			_ = p.Data[0]
			if err := bufferPool.UnpinPage(pageID, false); err != nil {
				b.Fatalf("Failed to unpin page: %v", err)
			}
			i++
		}
	})
}

// BenchmarkBufferPoolMixedWorkload benchmarks mixed read/write workload
func BenchmarkBufferPoolMixedWorkload(b *testing.B) {
	tempDir := b.TempDir()
	dbFile := tempDir + "/bench.db"

	diskMgr, err := NewDiskManager(dbFile)
	if err != nil {
		b.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bufferPool := NewBufferPool(100, diskMgr)

	// Create pages
	const numPages = 10
	pageIDs := make([]PageID, numPages)
	for i := 0; i < numPages; i++ {
		page, err := bufferPool.NewPage()
		if err != nil {
			b.Fatalf("Failed to create page: %v", err)
		}
		pageIDs[i] = page.ID
		if err := bufferPool.UnpinPage(page.ID, false); err != nil {
			b.Fatalf("Failed to unpin page: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			pageID := pageIDs[i%numPages]
			isWrite := i%5 == 0 // 20% writes, 80% reads

			p, err := bufferPool.FetchPage(pageID)
			if err != nil {
				b.Fatalf("Failed to fetch page: %v", err)
			}

			if isWrite {
				copy(p.Data[:], []byte("write"))
				p.MarkDirty()
			} else {
				_ = p.Data[0]
			}

			if err := bufferPool.UnpinPage(pageID, isWrite); err != nil {
				b.Fatalf("Failed to unpin page: %v", err)
			}
			i++
		}
	})
}

// TestBufferPoolRaceDetector tests for race conditions with -race flag
func TestBufferPoolRaceDetector(t *testing.T) {
	if os.Getenv("SKIP_RACE_TESTS") != "" {
		t.Skip("Skipping race detector test")
	}

	tempDir := t.TempDir()
	dbFile := tempDir + "/test.db"

	diskMgr, err := NewDiskManager(dbFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	bufferPool := NewBufferPool(10, diskMgr)

	// Create a single page
	page, err := bufferPool.NewPage()
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}
	pageID := page.ID
	if err := bufferPool.UnpinPage(pageID, false); err != nil {
		t.Fatalf("Failed to unpin page: %v", err)
	}

	// Hammer the same page from multiple goroutines
	const numGoroutines = 20
	const opsPerGoroutine = 100
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				p, err := bufferPool.FetchPage(pageID)
				if err != nil {
					t.Errorf("Failed to fetch page: %v", err)
					return
				}
				_ = p.Data[0]
				if err := bufferPool.UnpinPage(pageID, false); err != nil {
					t.Errorf("Failed to unpin page: %v", err)
					return
				}
			}
		}()
	}

	wg.Wait()
}
