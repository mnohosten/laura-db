package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mnohosten/laura-db/pkg/lsm"
)

func main() {
	fmt.Println("=== LauraDB LSM Tree Demo ===")
	fmt.Println()

	// Clean up any existing data
	dataDir := "./lsm-data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	// Demo 1: Basic Operations
	demo1BasicOperations(dataDir)

	// Demo 2: Write-Heavy Workload
	demo2WriteHeavy(dataDir + "-write")

	// Demo 3: Persistence
	demo3Persistence(dataDir + "-persist")

	// Demo 4: Statistics
	demo4Statistics(dataDir + "-stats")

	fmt.Println("\n=== Demo Complete ===")
}

func demo1BasicOperations(dir string) {
	fmt.Println("Demo 1: Basic LSM Tree Operations")
	fmt.Println("-----------------------------------")

	config := lsm.DefaultConfig(dir)
	tree, err := lsm.NewLSMTree(config)
	if err != nil {
		log.Fatal(err)
	}
	defer tree.Close()

	// Put some key-value pairs
	fmt.Println("Inserting key-value pairs...")
	pairs := map[string]string{
		"name":    "LauraDB",
		"type":    "LSM-Tree",
		"version": "1.0",
		"author":  "Demo",
	}

	for key, value := range pairs {
		if err := tree.Put([]byte(key), []byte(value)); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  PUT %s = %s\n", key, value)
	}

	// Get values
	fmt.Println("\nRetrieving values...")
	for key := range pairs {
		value, found, err := tree.Get([]byte(key))
		if err != nil {
			log.Fatal(err)
		}
		if found {
			fmt.Printf("  GET %s = %s\n", key, value)
		} else {
			fmt.Printf("  GET %s = NOT FOUND\n", key)
		}
	}

	// Delete a key
	fmt.Println("\nDeleting 'version' key...")
	if err := tree.Delete([]byte("version")); err != nil {
		log.Fatal(err)
	}

	value, found, _ := tree.Get([]byte("version"))
	fmt.Printf("  GET version = found:%v, value:%s\n", found, value)

	fmt.Println()
}

func demo2WriteHeavy(dir string) {
	fmt.Println("Demo 2: Write-Heavy Workload (LSM Advantage)")
	fmt.Println("----------------------------------------------")

	config := lsm.DefaultConfig(dir)
	config.MemTableSize = 1024 * 1024 // 1MB memtable
	tree, err := lsm.NewLSMTree(config)
	if err != nil {
		log.Fatal(err)
	}
	defer tree.Close()

	// Insert many keys quickly
	numKeys := 1000
	fmt.Printf("Inserting %d keys...\n", numKeys)
	start := time.Now()

	for i := 0; i < numKeys; i++ {
		key := []byte(fmt.Sprintf("user:%06d", i))
		value := []byte(fmt.Sprintf("data-for-user-%06d", i))
		if err := tree.Put(key, value); err != nil {
			log.Fatal(err)
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("Inserted %d keys in %v\n", numKeys, elapsed)
	fmt.Printf("Throughput: %.0f writes/sec\n", float64(numKeys)/elapsed.Seconds())

	// Flush to SSTables
	fmt.Println("\nFlushing to SSTables...")
	tree.Flush()

	stats := tree.Stats()
	fmt.Printf("Stats: %+v\n", stats)

	// Read some keys
	fmt.Println("\nReading sample keys...")
	sampleKeys := []string{"user:000000", "user:000500", "user:000999"}
	for _, key := range sampleKeys {
		value, found, _ := tree.Get([]byte(key))
		if found {
			fmt.Printf("  %s = %s\n", key, value)
		}
	}

	fmt.Println()
}

func demo3Persistence(dir string) {
	fmt.Println("Demo 3: Persistence and Recovery")
	fmt.Println("---------------------------------")

	// Create and populate LSM tree
	config := lsm.DefaultConfig(dir)
	tree, err := lsm.NewLSMTree(config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Writing data...")
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("persistent-key-%03d", i))
		value := []byte(fmt.Sprintf("value-%03d", i))
		tree.Put(key, value)
	}

	tree.Flush()
	fmt.Println("Closing database...")
	tree.Close()

	// Reopen and verify
	fmt.Println("Reopening database...")
	tree, err = lsm.NewLSMTree(config)
	if err != nil {
		log.Fatal(err)
	}
	defer tree.Close()

	stats := tree.Stats()
	fmt.Printf("Reopened with %d SSTables\n", stats["num_sstables"])

	// Verify some keys
	fmt.Println("\nVerifying persisted data...")
	testKeys := []string{"persistent-key-000", "persistent-key-050", "persistent-key-099"}
	for _, key := range testKeys {
		value, found, _ := tree.Get([]byte(key))
		if found {
			fmt.Printf("  ✓ %s = %s\n", key, value)
		} else {
			fmt.Printf("  ✗ %s NOT FOUND\n", key)
		}
	}

	fmt.Println()
}

func demo4Statistics(dir string) {
	fmt.Println("Demo 4: LSM Tree Statistics")
	fmt.Println("----------------------------")

	config := lsm.DefaultConfig(dir)
	config.MemTableSize = 512 * 1024 // 512KB
	tree, err := lsm.NewLSMTree(config)
	if err != nil {
		log.Fatal(err)
	}
	defer tree.Close()

	// Insert enough data to trigger flushes and compaction
	fmt.Println("Inserting data to trigger flushes...")
	for i := 0; i < 500; i++ {
		key := []byte(fmt.Sprintf("metric:%04d", i))
		value := []byte(fmt.Sprintf("measurement-%04d-with-some-data", i))
		tree.Put(key, value)
	}

	tree.Flush()
	time.Sleep(100 * time.Millisecond) // Allow compaction to run

	stats := tree.Stats()
	fmt.Println("\nLSM Tree Statistics:")
	fmt.Printf("  MemTable size: %d bytes\n", stats["memtable_size"])
	fmt.Printf("  Immutable memtables: %d\n", stats["num_immutables"])
	fmt.Printf("  SSTables: %d\n", stats["num_sstables"])
	fmt.Printf("  Total entries: %d\n", stats["total_entries"])
	fmt.Printf("  Next SSTable ID: %d\n", stats["next_sstable_id"])

	fmt.Println("\nLSM Tree Architecture:")
	fmt.Println("  Write Path: MemTable (in-memory) → Flush → SSTable (on-disk)")
	fmt.Println("  Read Path: MemTable → Immutables → SSTables (newest to oldest)")
	fmt.Println("  Compaction: Background merge of SSTables to reduce file count")
	fmt.Println("  Bloom Filters: Skip SSTable reads for non-existent keys")

	fmt.Println()
}
