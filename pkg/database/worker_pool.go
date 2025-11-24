package database

import (
	"context"
	"sync"
	"sync/atomic"
)

// WorkerPool manages a pool of worker goroutines for executing background tasks
// such as TTL cleanup, index building, defragmentation, etc.
// Provides controlled concurrency and graceful shutdown.
type WorkerPool struct {
	numWorkers  int
	taskQueue   chan Task
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	tasksTotal  atomic.Int64 // Total tasks submitted
	tasksActive atomic.Int64 // Currently executing tasks
	tasksDone   atomic.Int64 // Completed tasks
	closeOnce   sync.Once    // Ensures channel is only closed once
}

// Task represents a unit of work to be executed by the worker pool
type Task interface {
	Execute() error
}

// TaskFunc is a function that implements the Task interface
type TaskFunc func() error

// Execute implements the Task interface for TaskFunc
func (f TaskFunc) Execute() error {
	return f()
}

// WorkerPoolConfig holds configuration for creating a worker pool
type WorkerPoolConfig struct {
	// NumWorkers is the number of worker goroutines to spawn
	NumWorkers int

	// QueueSize is the buffer size for the task queue
	// A larger queue can handle bursts better but uses more memory
	QueueSize int
}

// DefaultWorkerPoolConfig returns a sensible default configuration
func DefaultWorkerPoolConfig() *WorkerPoolConfig {
	return &WorkerPoolConfig{
		NumWorkers: 4,        // 4 background workers by default
		QueueSize:  100,      // Buffer up to 100 tasks
	}
}

// NewWorkerPool creates a new worker pool with the given configuration
func NewWorkerPool(config *WorkerPoolConfig) *WorkerPool {
	if config == nil {
		config = DefaultWorkerPoolConfig()
	}

	// Ensure at least 1 worker
	if config.NumWorkers < 1 {
		config.NumWorkers = 1
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	wp := &WorkerPool{
		numWorkers: config.NumWorkers,
		taskQueue:  make(chan Task, config.QueueSize),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Start worker goroutines
	for i := 0; i < config.NumWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	return wp
}

// worker is the main loop for a worker goroutine
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.ctx.Done():
			// Shutdown signal received
			return

		case task, ok := <-wp.taskQueue:
			if !ok {
				// Channel closed, exit worker
				return
			}

			// Execute the task
			wp.tasksActive.Add(1)
			task.Execute() // Ignoring error for now, tasks should handle their own errors
			wp.tasksActive.Add(-1)
			wp.tasksDone.Add(1)
		}
	}
}

// Submit adds a task to the worker pool queue
// Returns true if the task was successfully queued, false if the pool is shutting down
func (wp *WorkerPool) Submit(task Task) bool {
	// Check if already shutting down
	if wp.IsShuttingDown() {
		return false
	}

	// Try to submit (using select with default for non-blocking behavior)
	// This will handle both full queue and closed channel cases
	select {
	case wp.taskQueue <- task:
		wp.tasksTotal.Add(1)
		return true
	default:
		// Queue is full or closed
		return false
	}
}

// SubmitFunc is a convenience method to submit a function as a task
func (wp *WorkerPool) SubmitFunc(fn func() error) bool {
	return wp.Submit(TaskFunc(fn))
}

// SubmitBlocking submits a task and blocks until it's queued or the pool shuts down
// Returns true if the task was successfully queued
func (wp *WorkerPool) SubmitBlocking(task Task) bool {
	select {
	case <-wp.ctx.Done():
		return false
	case wp.taskQueue <- task:
		wp.tasksTotal.Add(1)
		return true
	}
}

// Shutdown gracefully stops the worker pool
// Waits for all currently executing tasks to complete
// Does not wait for queued tasks (they will be discarded)
func (wp *WorkerPool) Shutdown() {
	// Signal workers to stop
	wp.cancel()

	// Close the task queue to unblock workers (only once)
	wp.closeOnce.Do(func() {
		close(wp.taskQueue)
	})

	// Wait for all workers to finish
	wp.wg.Wait()
}

// ShutdownAndDrain gracefully stops the worker pool and waits for all queued tasks
func (wp *WorkerPool) ShutdownAndDrain() {
	// Close the task queue (no more submissions - only once)
	wp.closeOnce.Do(func() {
		close(wp.taskQueue)
	})

	// Wait for all workers to finish (they'll drain the queue)
	wp.wg.Wait()

	// Cancel context
	wp.cancel()
}

// Stats returns statistics about the worker pool
func (wp *WorkerPool) Stats() WorkerPoolStats {
	return WorkerPoolStats{
		NumWorkers:  wp.numWorkers,
		TasksTotal:  wp.tasksTotal.Load(),
		TasksActive: wp.tasksActive.Load(),
		TasksDone:   wp.tasksDone.Load(),
		QueuedTasks: int64(len(wp.taskQueue)),
	}
}

// WorkerPoolStats holds statistics about the worker pool
type WorkerPoolStats struct {
	NumWorkers  int
	TasksTotal  int64 // Total tasks submitted
	TasksActive int64 // Currently executing
	TasksDone   int64 // Completed tasks
	QueuedTasks int64 // Waiting in queue
}

// IsFull returns true if the task queue is full
func (wp *WorkerPool) IsFull() bool {
	return len(wp.taskQueue) >= cap(wp.taskQueue)
}

// IsShuttingDown returns true if the pool is shutting down
func (wp *WorkerPool) IsShuttingDown() bool {
	select {
	case <-wp.ctx.Done():
		return true
	default:
		return false
	}
}
