package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/mnohosten/laura-db/pkg/server"
)

// This example demonstrates LauraDB's Prometheus metrics integration
//
// The example:
// 1. Starts LauraDB HTTP server with Prometheus metrics endpoint
// 2. Simulates realistic database workload
// 3. Shows how to configure Prometheus to scrape metrics
// 4. Demonstrates Grafana dashboard setup
//
// To use with Prometheus:
// 1. Run this example: go run main.go
// 2. Add to prometheus.yml:
//    scrape_configs:
//      - job_name: 'laura_db'
//        static_configs:
//          - targets: ['localhost:8080']
//        metrics_path: '/_metrics'
//        scrape_interval: 5s
// 3. Access metrics at: http://localhost:8080/_metrics
// 4. View in Prometheus UI: http://localhost:9090
// 5. Import LauraDB dashboard to Grafana (dashboard config below)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║        LauraDB Prometheus/Grafana Integration Demo            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create server configuration
	config := server.DefaultConfig()
	config.Host = "localhost"
	config.Port = 8080
	config.DataDir = "./prometheus-demo-data"
	config.EnableLogging = true

	// Create server
	srv, err := server.New(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start background workload simulator
	go simulateWorkload(srv)

	// Print usage instructions
	printInstructions()

	// Start server (blocks until shutdown)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// simulateWorkload generates realistic database activity
func simulateWorkload(srv *server.Server) {
	// Wait for server to start
	time.Sleep(2 * time.Second)

	fmt.Println("\n🔄 Starting workload simulator...")
	fmt.Println("   Generating realistic database traffic for Prometheus metrics")
	fmt.Println()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	collections := []string{"users", "orders", "products", "logs"}
	rand.Seed(time.Now().UnixNano())

	iteration := 0
	for range ticker.C {
		iteration++

		// Simulate different types of operations
		coll := collections[rand.Intn(len(collections))]
		collection := srv.GetDatabase().Collection(coll)

		// Insert operations (60% of traffic)
		if rand.Float64() < 0.6 {
			doc := generateRandomDocument(coll, iteration)
			srv.GetMetricsCollector().RecordInsert(
				time.Duration(rand.Intn(20))*time.Millisecond,
				rand.Float64() < 0.95, // 95% success rate
			)
			collection.InsertOne(doc)
		}

		// Query operations (30% of traffic)
		if rand.Float64() < 0.3 {
			filter := generateRandomFilter(coll)
			srv.GetMetricsCollector().RecordQuery(
				time.Duration(rand.Intn(50))*time.Millisecond,
				rand.Float64() < 0.98, // 98% success rate
			)
			collection.Find(filter)

			// Cache hits/misses
			if rand.Float64() < 0.7 {
				srv.GetMetricsCollector().RecordCacheHit()
			} else {
				srv.GetMetricsCollector().RecordCacheMiss()
			}
		}

		// Update operations (7% of traffic)
		if rand.Float64() < 0.07 {
			srv.GetMetricsCollector().RecordUpdate(
				time.Duration(rand.Intn(30))*time.Millisecond,
				rand.Float64() < 0.96,
			)
		}

		// Delete operations (3% of traffic)
		if rand.Float64() < 0.03 {
			srv.GetMetricsCollector().RecordDelete(
				time.Duration(rand.Intn(15))*time.Millisecond,
				rand.Float64() < 0.99,
			)
		}

		// Transaction operations
		if rand.Float64() < 0.1 {
			srv.GetMetricsCollector().RecordTransactionStart()
			if rand.Float64() < 0.9 {
				srv.GetMetricsCollector().RecordTransactionCommit()
			} else {
				srv.GetMetricsCollector().RecordTransactionAbort()
			}
		}

		// Index vs collection scans
		if rand.Float64() < 0.8 {
			srv.GetMetricsCollector().RecordIndexScan()
		} else {
			srv.GetMetricsCollector().RecordCollectionScan()
		}

		// Connection events
		if rand.Float64() < 0.2 {
			srv.GetMetricsCollector().RecordConnectionStart()
			if rand.Float64() < 0.5 {
				srv.GetMetricsCollector().RecordConnectionEnd()
			}
		}

		// I/O operations
		srv.GetResourceTracker().RecordRead(uint64(rand.Intn(10000) + 1000))
		srv.GetResourceTracker().RecordWrite(uint64(rand.Intn(5000) + 500))

		if iteration%10 == 0 {
			fmt.Printf("   ⚡ Generated %d operations...\n", iteration)
		}
	}
}

func generateRandomDocument(collection string, iteration int) map[string]interface{} {
	switch collection {
	case "users":
		return map[string]interface{}{
			"name":      fmt.Sprintf("User %d", iteration),
			"email":     fmt.Sprintf("user%d@example.com", iteration),
			"age":       int64(rand.Intn(60) + 18),
			"created":   time.Now(),
			"active":    rand.Float64() < 0.8,
		}
	case "orders":
		return map[string]interface{}{
			"order_id":  fmt.Sprintf("ORD-%d", iteration),
			"user_id":   fmt.Sprintf("user%d", rand.Intn(1000)),
			"amount":    rand.Float64() * 1000,
			"status":    []string{"pending", "completed", "cancelled"}[rand.Intn(3)],
			"timestamp": time.Now(),
		}
	case "products":
		return map[string]interface{}{
			"name":     fmt.Sprintf("Product %d", iteration),
			"category": []string{"electronics", "books", "clothing"}[rand.Intn(3)],
			"price":    rand.Float64() * 500,
			"stock":    int64(rand.Intn(100)),
		}
	case "logs":
		return map[string]interface{}{
			"level":   []string{"info", "warn", "error"}[rand.Intn(3)],
			"message": fmt.Sprintf("Log message %d", iteration),
			"timestamp": time.Now(),
		}
	default:
		return map[string]interface{}{"data": "test"}
	}
}

func generateRandomFilter(collection string) map[string]interface{} {
	switch collection {
	case "users":
		return map[string]interface{}{
			"age": map[string]interface{}{
				"$gte": int64(25),
			},
		}
	case "orders":
		return map[string]interface{}{
			"status": "completed",
		}
	case "products":
		return map[string]interface{}{
			"stock": map[string]interface{}{
				"$gt": int64(0),
			},
		}
	default:
		return map[string]interface{}{}
	}
}

func printInstructions() {
	fmt.Println("📊 Prometheus Metrics Integration")
	fmt.Println("=" + string(make([]byte, 70)))
	fmt.Println()
	fmt.Println("✅ Server running on: http://localhost:8080")
	fmt.Println("✅ Metrics endpoint:  http://localhost:8080/_metrics")
	fmt.Println("✅ Admin console:     http://localhost:8080/")
	fmt.Println()
	fmt.Println("📝 Prometheus Configuration (prometheus.yml):")
	fmt.Println("   scrape_configs:")
	fmt.Println("     - job_name: 'laura_db'")
	fmt.Println("       static_configs:")
	fmt.Println("         - targets: ['localhost:8080']")
	fmt.Println("       metrics_path: '/_metrics'")
	fmt.Println("       scrape_interval: 5s")
	fmt.Println()
	fmt.Println("📊 Available Metrics:")
	fmt.Println("   • Operation counters: queries, inserts, updates, deletes")
	fmt.Println("   • Latency histograms: p50, p95, p99 percentiles")
	fmt.Println("   • Cache metrics: hit rate, hits, misses")
	fmt.Println("   • Transaction metrics: started, committed, aborted")
	fmt.Println("   • Scan metrics: index scans, collection scans")
	fmt.Println("   • Connection metrics: active, total")
	fmt.Println("   • Resource metrics: memory, goroutines, I/O, GC")
	fmt.Println()
	fmt.Println("🎨 Grafana Dashboard Setup:")
	fmt.Println("   1. Add Prometheus data source in Grafana")
	fmt.Println("   2. Create new dashboard or import JSON below")
	fmt.Println("   3. Add panels for key metrics:")
	fmt.Println("      - Query rate: rate(laura_db_queries_total[1m])")
	fmt.Println("      - Cache hit rate: laura_db_cache_hit_rate")
	fmt.Println("      - P95 latency: laura_db_query_duration_seconds_p95")
	fmt.Println("      - Memory usage: laura_db_memory_heap_bytes")
	fmt.Println("      - Active connections: laura_db_active_connections")
	fmt.Println()
	fmt.Println("📈 Useful PromQL Queries:")
	fmt.Println("   • Query throughput:")
	fmt.Println("     rate(laura_db_queries_total[1m])")
	fmt.Println("   • Error rate:")
	fmt.Println("     rate(laura_db_queries_failed_total[1m]) / rate(laura_db_queries_total[1m])")
	fmt.Println("   • Average latency:")
	fmt.Println("     rate(laura_db_query_duration_nanoseconds_total[1m]) / rate(laura_db_queries_total[1m]) / 1000000")
	fmt.Println("   • Index usage percentage:")
	fmt.Println("     laura_db_index_usage_rate * 100")
	fmt.Println()
	fmt.Println("🔄 Workload simulator is running in the background")
	fmt.Println("   Press Ctrl+C to stop")
	fmt.Println("=" + string(make([]byte, 70)))
	fmt.Println()
}
