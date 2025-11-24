package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mnohosten/laura-db/pkg/storage"
)

func main() {
	fmt.Println("=== LauraDB Memory-Mapped Storage Demo ===")
	fmt.Println()

	// Clean up from previous runs
	os.RemoveAll("./data")
	os.MkdirAll("./data", 0755)

	// Demo 1: Basic mmap operations
	fmt.Println("Demo 1: Basic Memory-Mapped Operations")
	fmt.Println("---------------------------------------")
	basicMmapDemo()

	// Demo 2: Performance comparison
	fmt.Println("\nDemo 2: Performance Comparison")
	fmt.Println("-------------------------------")
	performanceComparison()

	// Demo 3: Access pattern hints
	fmt.Println("\nDemo 3: Access Pattern Hints")
	fmt.Println("-----------------------------")
	accessPatternDemo()

	// Demo 4: Dynamic expansion
	fmt.Println("\nDemo 4: Dynamic Expansion")
	fmt.Println("-------------------------")
	expansionDemo()

	// Demo 5: Persistence
	fmt.Println("\nDemo 5: Persistence Test")
	fmt.Println("------------------------")
	persistenceDemo()

	fmt.Println("\n=== Demo Complete ===")
}

func basicMmapDemo() {
	dm, err := storage.NewMmapDiskManager("./data/basic.db", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer dm.Close()

	// Allocate and write pages
	const numPages = 10
	fmt.Printf("Writing %d pages...\n", numPages)
	for i := 0; i < numPages; i++ {
		pageID, err := dm.AllocatePage()
		if err != nil {
			log.Fatal(err)
		}

		page := storage.NewPage(pageID, storage.PageTypeData)
		page.LSN = uint64(i * 100)
		copy(page.Data, []byte(fmt.Sprintf("Page %d: Hello from mmap!", i)))

		if err := dm.WritePage(page); err != nil {
			log.Fatal(err)
		}
	}

	// Read pages back
	fmt.Printf("Reading pages back...\n")
	for i := 0; i < numPages; i++ {
		page, err := dm.ReadPage(storage.PageID(i))
		if err != nil {
			log.Fatal(err)
		}

		// Find null terminator
		data := page.Data
		end := 0
		for end < len(data) && data[end] != 0 {
			end++
		}

		fmt.Printf("  Page %d (LSN=%d): %s\n", i, page.LSN, string(data[:end]))
	}

	// Show stats
	stats := dm.Stats()
	fmt.Printf("\nStatistics:\n")
	fmt.Printf("  Total reads:  %d\n", stats["total_reads"].(int64))
	fmt.Printf("  Total writes: %d\n", stats["total_writes"].(int64))
	fmt.Printf("  Mmap size:    %d bytes (%.2f MB)\n",
		stats["mmap_size"].(int64),
		float64(stats["mmap_size"].(int64))/(1024*1024))
}

func performanceComparison() {
	const numPages = 1000

	// Test standard disk manager
	standardDM, err := storage.NewDiskManager("./data/standard.db")
	if err != nil {
		log.Fatal(err)
	}

	startStandard := time.Now()
	for i := 0; i < numPages; i++ {
		pageID, _ := standardDM.AllocatePage()
		page := storage.NewPage(pageID, storage.PageTypeData)
		page.LSN = uint64(i)
		standardDM.WritePage(page)
	}
	writeStandard := time.Since(startStandard)

	startStandard = time.Now()
	for i := 0; i < numPages; i++ {
		standardDM.ReadPage(storage.PageID(i))
	}
	readStandard := time.Since(startStandard)
	standardDM.Close()

	// Test mmap disk manager
	mmapDM, err := storage.NewMmapDiskManager("./data/mmap.db", nil)
	if err != nil {
		log.Fatal(err)
	}

	startMmap := time.Now()
	for i := 0; i < numPages; i++ {
		pageID, _ := mmapDM.AllocatePage()
		page := storage.NewPage(pageID, storage.PageTypeData)
		page.LSN = uint64(i)
		mmapDM.WritePage(page)
	}
	writeMmap := time.Since(startMmap)

	startMmap = time.Now()
	for i := 0; i < numPages; i++ {
		mmapDM.ReadPage(storage.PageID(i))
	}
	readMmap := time.Since(startMmap)
	mmapDM.Close()

	// Print comparison
	fmt.Printf("Testing with %d pages:\n\n", numPages)

	fmt.Printf("Write Performance:\n")
	fmt.Printf("  Standard: %v (%.2f µs/page)\n",
		writeStandard, float64(writeStandard.Microseconds())/float64(numPages))
	fmt.Printf("  Mmap:     %v (%.2f µs/page)\n",
		writeMmap, float64(writeMmap.Microseconds())/float64(numPages))
	fmt.Printf("  Speedup:  %.2fx\n\n",
		float64(writeStandard)/float64(writeMmap))

	fmt.Printf("Read Performance:\n")
	fmt.Printf("  Standard: %v (%.2f µs/page)\n",
		readStandard, float64(readStandard.Microseconds())/float64(numPages))
	fmt.Printf("  Mmap:     %v (%.2f µs/page)\n",
		readMmap, float64(readMmap.Microseconds())/float64(numPages))
	fmt.Printf("  Speedup:  %.2fx\n",
		float64(readStandard)/float64(readMmap))
}

func accessPatternDemo() {
	dm, err := storage.NewMmapDiskManager("./data/pattern.db", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer dm.Close()

	// Create pages
	const numPages = 100
	fmt.Printf("Creating %d pages...\n", numPages)
	for i := 0; i < numPages; i++ {
		pageID, _ := dm.AllocatePage()
		page := storage.NewPage(pageID, storage.PageTypeData)
		page.Data[0] = byte(i)
		dm.WritePage(page)
	}

	// Test sequential access with hint
	fmt.Printf("\nTesting sequential access with MadviseSequential()...\n")
	dm.MadviseSequential()

	start := time.Now()
	for i := 0; i < numPages; i++ {
		dm.ReadPage(storage.PageID(i))
	}
	sequentialTime := time.Since(start)
	fmt.Printf("  Time: %v (%.2f µs/page)\n",
		sequentialTime, float64(sequentialTime.Microseconds())/float64(numPages))

	// Test random access with hint
	fmt.Printf("\nTesting random access with MadviseRandom()...\n")
	dm.MadviseRandom()

	start = time.Now()
	for i := 0; i < numPages; i++ {
		pageID := storage.PageID((i*7 + 13) % numPages) // Pseudo-random
		dm.ReadPage(pageID)
	}
	randomTime := time.Since(start)
	fmt.Printf("  Time: %v (%.2f µs/page)\n",
		randomTime, float64(randomTime.Microseconds())/float64(numPages))

	// Test prefetch
	fmt.Printf("\nTesting MadviseWillNeed() prefetch...\n")
	// Prefetch pages 50-60
	if err := dm.MadviseWillNeed(50, 60); err != nil {
		fmt.Printf("  Warning: %v\n", err)
	} else {
		fmt.Printf("  Successfully prefetched pages 50-60\n")
	}
}

func expansionDemo() {
	// Create with small initial size
	config := &storage.MmapConfig{
		InitialSize: 10 * storage.PageSize, // Only 10 pages initially
		GrowthSize:  5 * storage.PageSize,  // Grow by 5 pages
	}

	dm, err := storage.NewMmapDiskManager("./data/expand.db", config)
	if err != nil {
		log.Fatal(err)
	}
	defer dm.Close()

	fmt.Printf("Initial mmap size: %d bytes (%d pages)\n",
		config.InitialSize, config.InitialSize/storage.PageSize)

	// Allocate more pages than initial size
	const numPages = 25
	fmt.Printf("\nAllocating %d pages (will trigger expansion)...\n", numPages)

	for i := 0; i < numPages; i++ {
		pageID, err := dm.AllocatePage()
		if err != nil {
			log.Fatal(err)
		}

		page := storage.NewPage(pageID, storage.PageTypeData)
		page.Data[0] = byte(i)
		dm.WritePage(page)

		if i == 9 || i == 14 || i == 24 {
			stats := dm.Stats()
			fmt.Printf("  After page %d: mmap size = %d bytes (%d pages)\n",
				i+1, stats["mmap_size"].(int64), stats["mmap_size"].(int64)/storage.PageSize)
		}
	}

	// Verify all pages can be read
	fmt.Printf("\nVerifying all pages are readable...\n")
	for i := 0; i < numPages; i++ {
		page, err := dm.ReadPage(storage.PageID(i))
		if err != nil {
			log.Fatalf("Failed to read page %d: %v", i, err)
		}
		if page.Data[0] != byte(i) {
			log.Fatalf("Page %d data mismatch", i)
		}
	}
	fmt.Printf("  ✓ All %d pages verified successfully\n", numPages)
}

func persistenceDemo() {
	const numPages = 20

	// Create and write data
	fmt.Printf("Creating database with %d pages...\n", numPages)
	dm1, err := storage.NewMmapDiskManager("./data/persist.db", nil)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < numPages; i++ {
		pageID, _ := dm1.AllocatePage()
		page := storage.NewPage(pageID, storage.PageTypeData)
		page.LSN = uint64(i * 42)
		copy(page.Data, []byte(fmt.Sprintf("Persistent data %d", i)))
		dm1.WritePage(page)
	}

	dm1.Sync()
	dm1.Close()
	fmt.Printf("  ✓ Database closed\n")

	// Reopen and verify
	fmt.Printf("\nReopening database...\n")
	dm2, err := storage.NewMmapDiskManager("./data/persist.db", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer dm2.Close()

	stats := dm2.Stats()
	fmt.Printf("  Next page ID: %d\n", stats["next_page_id"].(storage.PageID))

	fmt.Printf("\nVerifying persisted data...\n")
	for i := 0; i < numPages; i++ {
		page, err := dm2.ReadPage(storage.PageID(i))
		if err != nil {
			log.Fatalf("Failed to read page %d: %v", i, err)
		}

		if page.LSN != uint64(i*42) {
			log.Fatalf("Page %d LSN mismatch: expected %d, got %d", i, i*42, page.LSN)
		}

		// Find null terminator
		data := page.Data
		end := 0
		for end < len(data) && data[end] != 0 {
			end++
		}

		expected := fmt.Sprintf("Persistent data %d", i)
		actual := string(data[:end])
		if actual != expected {
			log.Fatalf("Page %d data mismatch: expected '%s', got '%s'", i, expected, actual)
		}
	}
	fmt.Printf("  ✓ All %d pages verified successfully\n", numPages)
	fmt.Printf("  ✓ Data persisted correctly across database restart\n")
}
