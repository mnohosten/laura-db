package database

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestWorkerPool_BasicSubmit tests basic task submission and execution
func TestWorkerPool_BasicSubmit(t *testing.T) {
	config := &WorkerPoolConfig{
		NumWorkers: 2,
		QueueSize:  10,
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	var counter atomic.Int64

	// Submit a simple task
	submitted := pool.SubmitFunc(func() error {
		counter.Add(1)
		return nil
	})

	if !submitted {
		t.Fatal("Expected task to be submitted")
	}

	// Wait for task to complete
	time.Sleep(50 * time.Millisecond)

	if counter.Load() != 1 {
		t.Errorf("Expected counter to be 1, got %d", counter.Load())
	}
}

// TestWorkerPool_MultipleTasks tests submitting multiple tasks
func TestWorkerPool_MultipleTasks(t *testing.T) {
	config := &WorkerPoolConfig{
		NumWorkers: 4,
		QueueSize:  50,
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	var counter atomic.Int64
	numTasks := 20

	// Submit multiple tasks
	for i := 0; i < numTasks; i++ {
		submitted := pool.SubmitFunc(func() error {
			counter.Add(1)
			return nil
		})
		if !submitted {
			t.Errorf("Failed to submit task %d", i)
		}
	}

	// Wait for all tasks to complete
	time.Sleep(100 * time.Millisecond)

	if counter.Load() != int64(numTasks) {
		t.Errorf("Expected counter to be %d, got %d", numTasks, counter.Load())
	}
}

// TestWorkerPool_ConcurrentSubmit tests concurrent task submissions
func TestWorkerPool_ConcurrentSubmit(t *testing.T) {
	config := &WorkerPoolConfig{
		NumWorkers: 8,
		QueueSize:  200,
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	var counter atomic.Int64
	numGoroutines := 10
	tasksPerGoroutine := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch multiple goroutines that submit tasks concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < tasksPerGoroutine; j++ {
				pool.SubmitFunc(func() error {
					counter.Add(1)
					time.Sleep(1 * time.Millisecond) // Simulate some work
					return nil
				})
			}
		}()
	}

	wg.Wait()

	// Wait for all tasks to complete
	time.Sleep(500 * time.Millisecond)

	expected := int64(numGoroutines * tasksPerGoroutine)
	if counter.Load() != expected {
		t.Errorf("Expected counter to be %d, got %d", expected, counter.Load())
	}
}

// TestWorkerPool_Shutdown tests graceful shutdown
func TestWorkerPool_Shutdown(t *testing.T) {
	config := &WorkerPoolConfig{
		NumWorkers: 2,
		QueueSize:  10,
	}
	pool := NewWorkerPool(config)

	var counter atomic.Int64

	// Submit some tasks
	for i := 0; i < 5; i++ {
		pool.SubmitFunc(func() error {
			time.Sleep(10 * time.Millisecond)
			counter.Add(1)
			return nil
		})
	}

	// Shutdown the pool
	pool.Shutdown()

	// Verify workers have stopped (pool is shutting down)
	if !pool.IsShuttingDown() {
		t.Error("Expected pool to be shutting down")
	}

	// Try to submit after shutdown (should fail)
	submitted := pool.SubmitFunc(func() error {
		counter.Add(1)
		return nil
	})

	if submitted {
		t.Error("Should not be able to submit after shutdown")
	}
}

// TestWorkerPool_ShutdownAndDrain tests draining all queued tasks
func TestWorkerPool_ShutdownAndDrain(t *testing.T) {
	config := &WorkerPoolConfig{
		NumWorkers: 1, // Single worker to ensure queuing
		QueueSize:  20,
	}
	pool := NewWorkerPool(config)

	var counter atomic.Int64
	numTasks := 10

	// Submit tasks
	for i := 0; i < numTasks; i++ {
		pool.SubmitFunc(func() error {
			time.Sleep(10 * time.Millisecond)
			counter.Add(1)
			return nil
		})
	}

	// Shutdown and drain (wait for all tasks)
	pool.ShutdownAndDrain()

	// All tasks should have completed
	if counter.Load() != int64(numTasks) {
		t.Errorf("Expected all %d tasks to complete, got %d", numTasks, counter.Load())
	}
}

// TestWorkerPool_Stats tests worker pool statistics
func TestWorkerPool_Stats(t *testing.T) {
	config := &WorkerPoolConfig{
		NumWorkers: 2,
		QueueSize:  10,
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	// Submit tasks
	for i := 0; i < 5; i++ {
		pool.SubmitFunc(func() error {
			time.Sleep(20 * time.Millisecond)
			return nil
		})
	}

	// Get stats immediately (some tasks should be queued)
	stats := pool.Stats()

	if stats.NumWorkers != 2 {
		t.Errorf("Expected 2 workers, got %d", stats.NumWorkers)
	}

	if stats.TasksTotal != 5 {
		t.Errorf("Expected 5 total tasks, got %d", stats.TasksTotal)
	}

	// Wait for tasks to complete
	time.Sleep(200 * time.Millisecond)

	// Get stats after completion
	stats = pool.Stats()
	if stats.TasksDone != 5 {
		t.Errorf("Expected 5 completed tasks, got %d", stats.TasksDone)
	}

	if stats.TasksActive != 0 {
		t.Errorf("Expected 0 active tasks, got %d", stats.TasksActive)
	}
}

// TestWorkerPool_SubmitBlocking tests blocking task submission
func TestWorkerPool_SubmitBlocking(t *testing.T) {
	config := &WorkerPoolConfig{
		NumWorkers: 1,
		QueueSize:  2,
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	var counter atomic.Int64

	// Submit tasks using blocking submission
	submitted := pool.SubmitBlocking(TaskFunc(func() error {
		time.Sleep(10 * time.Millisecond)
		counter.Add(1)
		return nil
	}))

	if !submitted {
		t.Fatal("Expected task to be submitted")
	}

	// Wait for task to complete
	time.Sleep(50 * time.Millisecond)

	if counter.Load() != 1 {
		t.Errorf("Expected counter to be 1, got %d", counter.Load())
	}
}

// TestWorkerPool_QueueFull tests behavior when queue is full
func TestWorkerPool_QueueFull(t *testing.T) {
	config := &WorkerPoolConfig{
		NumWorkers: 1,  // Single worker
		QueueSize:  3,  // Small queue
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	// Submit slow tasks rapidly to fill the queue
	successCount := 0
	failCount := 0

	for i := 0; i < 20; i++ {
		submitted := pool.Submit(TaskFunc(func() error {
			time.Sleep(200 * time.Millisecond)
			return nil
		}))

		if submitted {
			successCount++
		} else {
			failCount++
		}
	}

	// We should have some successful submissions and some failures
	if failCount == 0 {
		t.Error("Expected some submissions to fail when queue is full")
	}

	if successCount == 0 {
		t.Error("Expected some submissions to succeed")
	}

	t.Logf("Submitted: %d succeeded, %d failed", successCount, failCount)
}

// TestWorkerPool_DefaultConfig tests default configuration
func TestWorkerPool_DefaultConfig(t *testing.T) {
	pool := NewWorkerPool(nil) // Use default config
	defer pool.Shutdown()

	stats := pool.Stats()
	if stats.NumWorkers != 4 {
		t.Errorf("Expected 4 workers (default), got %d", stats.NumWorkers)
	}
}

// TestWorkerPool_MinWorkers tests that at least 1 worker is created
func TestWorkerPool_MinWorkers(t *testing.T) {
	config := &WorkerPoolConfig{
		NumWorkers: 0, // Invalid: should be corrected to 1
		QueueSize:  10,
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	stats := pool.Stats()
	if stats.NumWorkers != 1 {
		t.Errorf("Expected at least 1 worker, got %d", stats.NumWorkers)
	}
}

// TestWorkerPool_TaskExecution tests that tasks actually execute
func TestWorkerPool_TaskExecution(t *testing.T) {
	config := &WorkerPoolConfig{
		NumWorkers: 2,
		QueueSize:  5,
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	executed := make(chan int, 5)

	// Submit tasks that report their execution
	for i := 0; i < 5; i++ {
		id := i
		pool.SubmitFunc(func() error {
			executed <- id
			return nil
		})
	}

	// Collect results
	results := make(map[int]bool)
	timeout := time.After(1 * time.Second)

	for len(results) < 5 {
		select {
		case id := <-executed:
			results[id] = true
		case <-timeout:
			t.Fatal("Timeout waiting for tasks to execute")
		}
	}

	// Verify all tasks executed
	for i := 0; i < 5; i++ {
		if !results[i] {
			t.Errorf("Task %d did not execute", i)
		}
	}
}

// TestWorkerPool_HighLoad tests worker pool under high load
func TestWorkerPool_HighLoad(t *testing.T) {
	config := &WorkerPoolConfig{
		NumWorkers: 16,
		QueueSize:  1000,
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	var counter atomic.Int64
	numTasks := 500

	// Submit many tasks
	for i := 0; i < numTasks; i++ {
		submitted := pool.SubmitFunc(func() error {
			counter.Add(1)
			time.Sleep(1 * time.Millisecond)
			return nil
		})
		if !submitted {
			t.Errorf("Failed to submit task under high load")
		}
	}

	// Wait for completion
	time.Sleep(2 * time.Second)

	if counter.Load() != int64(numTasks) {
		t.Errorf("Expected %d tasks to complete, got %d", numTasks, counter.Load())
	}
}

// TestWorkerPool_NoRaceConditions tests for race conditions with -race flag
func TestWorkerPool_NoRaceConditions(t *testing.T) {
	config := &WorkerPoolConfig{
		NumWorkers: 8,
		QueueSize:  150,
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	sharedMap := make(map[int]int)
	var mu sync.Mutex
	var wg sync.WaitGroup

	numTasks := 100

	// Submit tasks that access shared state
	for i := 0; i < numTasks; i++ {
		id := i
		wg.Add(1)
		pool.SubmitFunc(func() error {
			defer wg.Done()
			mu.Lock()
			sharedMap[id] = id * 2
			mu.Unlock()
			return nil
		})
	}

	// Wait for all tasks to complete
	wg.Wait()

	// Verify results
	mu.Lock()
	if len(sharedMap) != numTasks {
		t.Errorf("Expected %d entries in map, got %d", numTasks, len(sharedMap))
	}
	mu.Unlock()
}

// TestWorkerPool_IsFull tests the IsFull() method
func TestWorkerPool_IsFull(t *testing.T) {
	config := &WorkerPoolConfig{
		NumWorkers: 1,
		QueueSize:  3, // Small queue
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	// Initially, pool should not be full
	if pool.IsFull() {
		t.Error("Expected pool to not be full initially")
	}

	// Submit blocking tasks to fill the queue
	blockChan := make(chan struct{})
	workerStarted := make(chan struct{})

	// Submit first task and wait for it to start processing
	submitted := pool.Submit(TaskFunc(func() error {
		close(workerStarted)
		<-blockChan
		return nil
	}))
	if !submitted {
		t.Fatal("Failed to submit first task")
	}

	// Wait for worker to start processing the first task
	<-workerStarted
	time.Sleep(10 * time.Millisecond) // Extra time to ensure worker is blocked

	// Now submit 3 more tasks to fill the queue (they should all succeed)
	for i := 0; i < 3; i++ {
		submitted := pool.Submit(TaskFunc(func() error {
			<-blockChan
			return nil
		}))
		if !submitted {
			t.Errorf("Task %d should have been accepted but was rejected", i+1)
		}
	}

	// Try to submit one more task - this should fail since queue is full
	submitted = pool.Submit(TaskFunc(func() error {
		<-blockChan
		return nil
	}))
	if submitted {
		t.Error("Expected 5th task to be rejected but it was accepted")
	}

	// Give a moment for queue state to stabilize
	time.Sleep(10 * time.Millisecond)

	// Now pool should be full (3 items in queue, 1 being processed)
	if !pool.IsFull() {
		stats := pool.Stats()
		t.Logf("Queue size: %d, Capacity: %d", stats.QueuedTasks, cap(pool.taskQueue))
		t.Error("Expected pool to be full after filling queue")
	}

	// Unblock tasks
	close(blockChan)

	// Wait for queue to drain
	time.Sleep(200 * time.Millisecond)

	// Pool should no longer be full
	if pool.IsFull() {
		t.Error("Expected pool to not be full after draining")
	}
}
