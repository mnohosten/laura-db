package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/query"
)

func main() {
	fmt.Println("=== LauraDB Parallel Query Execution Demo ===")
	fmt.Println()

	// Create a large dataset for demonstration
	fmt.Println("Creating dataset with 50,000 documents...")
	docs := make([]*document.Document, 50000)
	for i := 0; i < 50000; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("age", int64(20+i%60))
		doc.Set("score", int64(50+i%100))
		doc.Set("city", getCityName(i%10))
		doc.Set("active", i%3 == 0)
		docs[i] = doc
	}
	fmt.Printf("Created %d documents\n\n", len(docs))

	// Create executor
	executor := query.NewExecutor(docs)

	// Complex query
	q := query.NewQuery(map[string]interface{}{
		"$and": []interface{}{
			map[string]interface{}{
				"age": map[string]interface{}{"$gte": int64(40)},
			},
			map[string]interface{}{
				"score": map[string]interface{}{"$gt": int64(75)},
			},
			map[string]interface{}{
				"active": true,
			},
		},
	})
	q.WithSort([]query.SortField{{Field: "score", Ascending: false}})
	q.WithLimit(100)

	fmt.Printf("Query: Find active users with age >= 40 and score > 75, sorted by score desc, limit 100\n")
	fmt.Printf("System CPUs: %d\n\n", runtime.NumCPU())

	// Benchmark sequential execution
	fmt.Println("--- Sequential Execution ---")
	start := time.Now()
	seqResults, err := executor.Execute(q)
	seqDuration := time.Since(start)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Time: %v\n", seqDuration)
	fmt.Printf("Results: %d documents\n\n", len(seqResults))

	// Benchmark parallel execution with default config
	fmt.Println("--- Parallel Execution (Default Config) ---")
	start = time.Now()
	parallelResults, err := executor.ExecuteParallel(q, nil)
	parallelDuration := time.Since(start)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Time: %v\n", parallelDuration)
	fmt.Printf("Results: %d documents\n", len(parallelResults))

	speedup := float64(seqDuration) / float64(parallelDuration)
	fmt.Printf("Speedup: %.2fx\n\n", speedup)

	// Benchmark with custom config (more workers)
	fmt.Println("--- Parallel Execution (Custom Config: 8 workers) ---")
	config := &query.ParallelConfig{
		MinDocsForParallel: 1000,
		MaxWorkers:         8,
		ChunkSize:          0, // auto
	}
	start = time.Now()
	customResults, err := executor.ExecuteParallel(q, config)
	customDuration := time.Since(start)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Time: %v\n", customDuration)
	fmt.Printf("Results: %d documents\n", len(customResults))

	customSpeedup := float64(seqDuration) / float64(customDuration)
	fmt.Printf("Speedup: %.2fx\n\n", customSpeedup)

	// Show some sample results
	fmt.Println("--- Sample Results (Top 5) ---")
	for i := 0; i < 5 && i < len(parallelResults); i++ {
		doc := parallelResults[i]
		age, _ := doc.Get("age")
		score, _ := doc.Get("score")
		city, _ := doc.Get("city")
		active, _ := doc.Get("active")

		fmt.Printf("%d. Age: %d, Score: %d, City: %s, Active: %v\n",
			i+1, age, score, city, active)
	}

	// Verify consistency
	fmt.Println("\n--- Verification ---")
	if len(seqResults) != len(parallelResults) {
		fmt.Printf("❌ Result count mismatch: sequential=%d, parallel=%d\n",
			len(seqResults), len(parallelResults))
	} else {
		fmt.Printf("✓ Result count matches: %d documents\n", len(seqResults))
	}

	// Verify first result matches
	if len(seqResults) > 0 && len(parallelResults) > 0 {
		seq0Score, _ := seqResults[0].Get("score")
		par0Score, _ := parallelResults[0].Get("score")
		if seq0Score == par0Score {
			fmt.Printf("✓ Top result matches (score=%d)\n", seq0Score)
		} else {
			fmt.Printf("❌ Top result mismatch: seq=%d, par=%d\n", seq0Score, par0Score)
		}
	}

	fmt.Println("\n=== Performance Summary ===")
	fmt.Printf("Sequential:        %v\n", seqDuration)
	fmt.Printf("Parallel (auto):   %v (%.2fx faster)\n", parallelDuration, speedup)
	fmt.Printf("Parallel (8 workers): %v (%.2fx faster)\n", customDuration, customSpeedup)
}

func getCityName(index int) string {
	cities := []string{
		"New York", "Los Angeles", "Chicago", "Houston", "Phoenix",
		"Philadelphia", "San Antonio", "San Diego", "Dallas", "San Jose",
	}
	return cities[index%len(cities)]
}
