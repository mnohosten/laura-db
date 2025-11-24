package database

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDocumentLockManager_BasicLocking(t *testing.T) {
	dlm := NewDocumentLockManager(16)

	// Test write lock
	dlm.Lock("doc1")
	dlm.Unlock("doc1")

	// Test read lock
	dlm.RLock("doc1")
	dlm.RUnlock("doc1")
}

func TestDocumentLockManager_ConcurrentReads(t *testing.T) {
	dlm := NewDocumentLockManager(16)
	docID := "doc1"

	var wg sync.WaitGroup
	readers := 10
	wg.Add(readers)

	start := time.Now()
	for i := 0; i < readers; i++ {
		go func(id int) {
			defer wg.Done()
			dlm.RLock(docID)
			time.Sleep(10 * time.Millisecond) // Simulate read operation
			dlm.RUnlock(docID)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// All readers should run concurrently, so total time should be close to 10ms
	// Allow some overhead
	if elapsed > 50*time.Millisecond {
		t.Errorf("Concurrent reads took too long: %v (expected ~10ms)", elapsed)
	}
}

func TestDocumentLockManager_ExclusiveWrite(t *testing.T) {
	dlm := NewDocumentLockManager(16)
	docID := "doc1"

	var counter int64
	var wg sync.WaitGroup
	writers := 10
	wg.Add(writers)

	for i := 0; i < writers; i++ {
		go func() {
			defer wg.Done()
			dlm.Lock(docID)
			// Read, increment, write - should be atomic due to lock
			current := atomic.LoadInt64(&counter)
			time.Sleep(1 * time.Millisecond) // Small delay to test exclusivity
			atomic.StoreInt64(&counter, current+1)
			dlm.Unlock(docID)
		}()
	}

	wg.Wait()

	if counter != int64(writers) {
		t.Errorf("Counter should be %d, got %d", writers, counter)
	}
}

func TestDocumentLockManager_ReadWriteExclusion(t *testing.T) {
	dlm := NewDocumentLockManager(16)
	docID := "doc1"

	var value int64
	var wg sync.WaitGroup

	// Start writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		dlm.Lock(docID)
		atomic.StoreInt64(&value, 100)
		time.Sleep(50 * time.Millisecond) // Hold write lock
		dlm.Unlock(docID)
	}()

	// Give writer time to acquire lock
	time.Sleep(10 * time.Millisecond)

	// Start reader - should wait for writer
	wg.Add(1)
	start := time.Now()
	go func() {
		defer wg.Done()
		dlm.RLock(docID)
		val := atomic.LoadInt64(&value)
		if val != 100 {
			t.Errorf("Reader got incorrect value: %d (expected 100)", val)
		}
		dlm.RUnlock(docID)
	}()

	wg.Wait()
	elapsed := time.Since(start)

	// Reader should have waited for writer (allow some timing tolerance)
	if elapsed < 35*time.Millisecond {
		t.Errorf("Reader didn't wait for writer: %v", elapsed)
	}
}

func TestDocumentLockManager_MultipleDocuments(t *testing.T) {
	dlm := NewDocumentLockManager(16)

	var wg sync.WaitGroup
	docs := []string{"doc1", "doc2", "doc3", "doc4", "doc5"}

	// Lock different documents concurrently
	for _, docID := range docs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			dlm.Lock(id)
			time.Sleep(10 * time.Millisecond)
			dlm.Unlock(id)
		}(docID)
	}

	start := time.Now()
	wg.Wait()
	elapsed := time.Since(start)

	// Different documents should be locked independently
	// Total time should be close to 10ms, not 50ms
	if elapsed > 50*time.Millisecond {
		t.Errorf("Locking different documents took too long: %v", elapsed)
	}
}

func TestDocumentLockManager_LockMultiple(t *testing.T) {
	dlm := NewDocumentLockManager(16)

	docIDs := []string{"doc3", "doc1", "doc2"} // Unsorted

	// Lock multiple documents
	dlm.LockMultiple(docIDs)

	// Try to lock one of them from another goroutine - should block
	done := make(chan bool, 1)
	go func() {
		dlm.Lock("doc2")
		dlm.Unlock("doc2")
		done <- true
	}()

	// Should not complete within 50ms
	select {
	case <-done:
		t.Error("Lock should have been blocked")
	case <-time.After(50 * time.Millisecond):
		// Expected - lock is blocked
	}

	// Unlock all
	dlm.UnlockMultiple(docIDs)

	// Now the goroutine should complete
	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Lock should have been released")
	}
}

func TestDocumentLockManager_DeadlockPrevention(t *testing.T) {
	dlm := NewDocumentLockManager(16)

	docIDs1 := []string{"doc1", "doc2", "doc3"}
	docIDs2 := []string{"doc3", "doc1", "doc2"} // Different order

	var wg sync.WaitGroup
	completed := make(chan int, 2)

	// Goroutine 1: Lock in one order
	wg.Add(1)
	go func() {
		defer wg.Done()
		dlm.LockMultiple(docIDs1)
		time.Sleep(10 * time.Millisecond)
		dlm.UnlockMultiple(docIDs1)
		completed <- 1
	}()

	// Goroutine 2: Lock in different order (but LockMultiple sorts them)
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Millisecond) // Small delay
		dlm.LockMultiple(docIDs2)
		time.Sleep(10 * time.Millisecond)
		dlm.UnlockMultiple(docIDs2)
		completed <- 2
	}()

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Both should complete without deadlock
		if len(completed) != 2 {
			t.Error("Not all goroutines completed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Deadlock detected - operations did not complete")
	}
}

func TestDocumentLockManager_StripeDistribution(t *testing.T) {
	dlm := NewDocumentLockManager(16)

	// Test that different document IDs map to stripes
	stripe1 := dlm.getStripe("doc1")
	stripe2 := dlm.getStripe("doc2")
	stripe3 := dlm.getStripe("doc1") // Same ID should map to same stripe

	if stripe1 != stripe3 {
		t.Error("Same document ID should map to same stripe")
	}

	// Check that stripes are different (not guaranteed but likely with 16 stripes)
	// Just verify the function returns valid stripe indices
	if stripe1 == nil || stripe2 == nil {
		t.Error("Stripe should not be nil")
	}
}

func TestDocumentLockManager_Cleanup(t *testing.T) {
	dlm := NewDocumentLockManager(4) // Small number for testing

	// Lock and unlock some documents
	for i := 0; i < 10; i++ {
		docID := fmt.Sprintf("doc%d", i)
		dlm.Lock(docID)
		dlm.Unlock(docID)
	}

	// Verify locks exist
	totalLocks := 0
	for _, stripe := range dlm.stripes {
		stripe.mu.Lock()
		totalLocks += len(stripe.locks)
		stripe.mu.Unlock()
	}

	if totalLocks != 10 {
		t.Errorf("Expected 10 locks, got %d", totalLocks)
	}

	// Cleanup
	dlm.Cleanup()

	// Verify locks are cleared
	totalLocksAfter := 0
	for _, stripe := range dlm.stripes {
		stripe.mu.Lock()
		totalLocksAfter += len(stripe.locks)
		stripe.mu.Unlock()
	}

	if totalLocksAfter != 0 {
		t.Errorf("Expected 0 locks after cleanup, got %d", totalLocksAfter)
	}
}

func TestDocumentLockManager_HighConcurrency(t *testing.T) {
	dlm := NewDocumentLockManager(256)

	var wg sync.WaitGroup
	operations := 1000
	numDocs := 100

	wg.Add(operations)

	for i := 0; i < operations; i++ {
		go func(id int) {
			defer wg.Done()
			docID := fmt.Sprintf("doc%d", id%numDocs)

			if id%2 == 0 {
				// Write operation
				dlm.Lock(docID)
				time.Sleep(time.Microsecond)
				dlm.Unlock(docID)
			} else {
				// Read operation
				dlm.RLock(docID)
				time.Sleep(time.Microsecond)
				dlm.RUnlock(docID)
			}
		}(i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(10 * time.Second):
		t.Fatal("High concurrency test did not complete in time")
	}
}

func TestDocumentLockManager_DefaultStripeCount(t *testing.T) {
	dlm := NewDocumentLockManager(0) // Should use default

	if dlm.numStripes != 256 {
		t.Errorf("Expected default stripe count 256, got %d", dlm.numStripes)
	}

	if len(dlm.stripes) != 256 {
		t.Errorf("Expected 256 stripes, got %d", len(dlm.stripes))
	}
}
