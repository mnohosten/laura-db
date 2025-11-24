package database

import (
	"fmt"
	"testing"
)

// BenchmarkSessionPool_GetPut benchmarks session pool get/put operations
func BenchmarkSessionPool_GetPut(b *testing.B) {
	db := setupPoolTestDB(&testing.T{})
	defer db.Close()

	pool := NewSessionPool(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session := pool.Get()
		session.AbortTransaction()
		pool.Put(session)
	}
}

// BenchmarkSessionPool_WithTransactionPooled benchmarks pooled transactions
func BenchmarkSessionPool_WithTransactionPooled(b *testing.B) {
	db := setupPoolTestDB(&testing.T{})
	defer db.Close()

	pool := NewSessionPool(db)
	db.Collection("bench_test") // Pre-create collection

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.WithTransactionPooled(func(session *Session) error {
			_, err := session.InsertOne("bench_test", map[string]interface{}{
				"id":    int64(i),
				"value": fmt.Sprintf("value_%d", i),
			})
			return err
		})
	}
}

// BenchmarkSessionPool_vs_Direct compares pooled vs direct session creation
func BenchmarkSessionPool_vs_Direct(b *testing.B) {
	db := setupPoolTestDB(&testing.T{})
	defer db.Close()

	b.Run("Pooled", func(b *testing.B) {
		pool := NewSessionPool(db)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			session := pool.Get()
			session.AbortTransaction()
			pool.Put(session)
		}
	})

	b.Run("Direct", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			session := db.StartSession()
			session.AbortTransaction()
		}
	})
}

// BenchmarkSessionPool_ConcurrentGetPut benchmarks concurrent pool usage
func BenchmarkSessionPool_ConcurrentGetPut(b *testing.B) {
	db := setupPoolTestDB(&testing.T{})
	defer db.Close()

	pool := NewSessionPool(db)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			session := pool.Get()
			session.AbortTransaction()
			pool.Put(session)
		}
	})
}

// BenchmarkSessionPool_TransactionInsert benchmarks transactional inserts
func BenchmarkSessionPool_TransactionInsert(b *testing.B) {
	db := setupPoolTestDB(&testing.T{})
	defer db.Close()

	pool := NewSessionPool(db)
	coll := db.Collection("insert_bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.WithTransactionPooled(func(session *Session) error {
			_, err := session.InsertOne("insert_bench", map[string]interface{}{
				"counter": int64(i),
				"data":    "benchmark data",
			})
			return err
		})
	}

	b.StopTimer()
	// Cleanup
	coll.DeleteMany(map[string]interface{}{})
}

// BenchmarkWorkerPool_Submit benchmarks task submission
func BenchmarkWorkerPool_Submit(b *testing.B) {
	config := &WorkerPoolConfig{
		NumWorkers: 4,
		QueueSize:  1000,
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.SubmitFunc(func() error {
			return nil
		})
	}
}

// BenchmarkWorkerPool_Execute benchmarks task execution throughput
func BenchmarkWorkerPool_Execute(b *testing.B) {
	config := &WorkerPoolConfig{
		NumWorkers: 8,
		QueueSize:  10000,
	}
	pool := NewWorkerPool(config)
	defer pool.ShutdownAndDrain()

	completed := make(chan struct{}, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.SubmitFunc(func() error {
			completed <- struct{}{}
			return nil
		})
	}

	// Wait for all tasks to complete
	for i := 0; i < b.N; i++ {
		<-completed
	}
}

// BenchmarkWorkerPool_ConcurrentSubmit benchmarks concurrent submissions
func BenchmarkWorkerPool_ConcurrentSubmit(b *testing.B) {
	config := &WorkerPoolConfig{
		NumWorkers: 8,
		QueueSize:  10000,
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.SubmitFunc(func() error {
				return nil
			})
		}
	})
}

// BenchmarkWorkerPool_WorkerScaling benchmarks different worker counts
func BenchmarkWorkerPool_WorkerScaling(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8, 16, 32}

	for _, numWorkers := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", numWorkers), func(b *testing.B) {
			config := &WorkerPoolConfig{
				NumWorkers: numWorkers,
				QueueSize:  10000,
			}
			pool := NewWorkerPool(config)
			defer pool.Shutdown()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				pool.SubmitFunc(func() error {
					// Simulate small amount of work
					sum := 0
					for j := 0; j < 100; j++ {
						sum += j
					}
					return nil
				})
			}
		})
	}
}

// BenchmarkWorkerPool_QueueScaling benchmarks different queue sizes
func BenchmarkWorkerPool_QueueScaling(b *testing.B) {
	queueSizes := []int{10, 100, 1000, 10000}

	for _, queueSize := range queueSizes {
		b.Run(fmt.Sprintf("Queue_%d", queueSize), func(b *testing.B) {
			config := &WorkerPoolConfig{
				NumWorkers: 4,
				QueueSize:  queueSize,
			}
			pool := NewWorkerPool(config)
			defer pool.Shutdown()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				pool.Submit(TaskFunc(func() error {
					return nil
				}))
			}
		})
	}
}

// BenchmarkWorkerPool_vs_Goroutines compares worker pool vs raw goroutines
func BenchmarkWorkerPool_vs_Goroutines(b *testing.B) {
	b.Run("WorkerPool", func(b *testing.B) {
		config := &WorkerPoolConfig{
			NumWorkers: 8,
			QueueSize:  1000,
		}
		pool := NewWorkerPool(config)
		defer pool.Shutdown()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pool.SubmitFunc(func() error {
				sum := 0
				for j := 0; j < 100; j++ {
					sum += j
				}
				return nil
			})
		}
	})

	b.Run("RawGoroutines", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			go func() {
				sum := 0
				for j := 0; j < 100; j++ {
					sum += j
				}
			}()
		}
	})
}

// BenchmarkSessionPool_TransactionUpdate benchmarks transactional updates
func BenchmarkSessionPool_TransactionUpdate(b *testing.B) {
	db := setupPoolTestDB(&testing.T{})
	defer db.Close()

	pool := NewSessionPool(db)
	coll := db.Collection("update_bench")

	// Pre-insert documents
	for i := 0; i < 100; i++ {
		coll.InsertOne(map[string]interface{}{
			"id":      int64(i),
			"counter": int64(0),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.WithTransactionPooled(func(session *Session) error {
			return session.UpdateOne("update_bench",
				map[string]interface{}{"id": int64(i % 100)},
				map[string]interface{}{
					"$inc": map[string]interface{}{"counter": int64(1)},
				})
		})
	}
}

// BenchmarkWorkerPool_MemoryUsage benchmarks memory allocation
func BenchmarkWorkerPool_MemoryUsage(b *testing.B) {
	config := &WorkerPoolConfig{
		NumWorkers: 4,
		QueueSize:  100,
	}
	pool := NewWorkerPool(config)
	defer pool.Shutdown()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pool.Submit(TaskFunc(func() error {
			return nil
		}))
	}
}

// BenchmarkSessionPool_MemoryUsage benchmarks session pool memory allocation
func BenchmarkSessionPool_MemoryUsage(b *testing.B) {
	db := setupPoolTestDB(&testing.T{})
	defer db.Close()

	pool := NewSessionPool(db)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		session := pool.Get()
		session.AbortTransaction()
		pool.Put(session)
	}
}
