package storage

import (
	"path/filepath"
	"testing"
)

func BenchmarkMmapDiskManager_WritePage(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_mmap_write.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		b.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Pre-allocate pages
	pageIDs := make([]PageID, b.N)
	for i := 0; i < b.N; i++ {
		pageID, _ := dm.AllocatePage()
		pageIDs[i] = pageID
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		page := NewPage(pageIDs[i], PageTypeData)
		page.LSN = uint64(i)
		copy(page.Data, []byte("benchmark data"))
		if err := dm.WritePage(page); err != nil {
			b.Fatalf("Failed to write page: %v", err)
		}
	}
}

func BenchmarkDiskManager_WritePage(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_disk_write.db")

	dm, err := NewDiskManager(dbPath)
	if err != nil {
		b.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Pre-allocate pages
	pageIDs := make([]PageID, b.N)
	for i := 0; i < b.N; i++ {
		pageID, _ := dm.AllocatePage()
		pageIDs[i] = pageID
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		page := NewPage(pageIDs[i], PageTypeData)
		page.LSN = uint64(i)
		copy(page.Data, []byte("benchmark data"))
		if err := dm.WritePage(page); err != nil {
			b.Fatalf("Failed to write page: %v", err)
		}
	}
}

func BenchmarkMmapDiskManager_ReadPage(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_mmap_read.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		b.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Create pages to read
	const numPages = 1000
	for i := 0; i < numPages; i++ {
		pageID, _ := dm.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		page.LSN = uint64(i)
		dm.WritePage(page)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pageID := PageID(i % numPages)
		if _, err := dm.ReadPage(pageID); err != nil {
			b.Fatalf("Failed to read page: %v", err)
		}
	}
}

func BenchmarkDiskManager_ReadPage(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_disk_read.db")

	dm, err := NewDiskManager(dbPath)
	if err != nil {
		b.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Create pages to read
	const numPages = 1000
	for i := 0; i < numPages; i++ {
		pageID, _ := dm.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		page.LSN = uint64(i)
		dm.WritePage(page)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pageID := PageID(i % numPages)
		if _, err := dm.ReadPage(pageID); err != nil {
			b.Fatalf("Failed to read page: %v", err)
		}
	}
}

func BenchmarkMmapDiskManager_RandomRead(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_mmap_random.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		b.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Create many pages
	const numPages = 10000
	for i := 0; i < numPages; i++ {
		pageID, _ := dm.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		page.Data[0] = byte(i % 256)
		dm.WritePage(page)
	}

	// Set random access hint
	dm.MadviseRandom()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Pseudo-random access pattern
		pageID := PageID((i*7919 + 12345) % numPages)
		if _, err := dm.ReadPage(pageID); err != nil {
			b.Fatalf("Failed to read page: %v", err)
		}
	}
}

func BenchmarkMmapDiskManager_SequentialRead(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_mmap_sequential.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		b.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Create many pages
	const numPages = 10000
	for i := 0; i < numPages; i++ {
		pageID, _ := dm.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		page.Data[0] = byte(i % 256)
		dm.WritePage(page)
	}

	// Set sequential access hint
	dm.MadviseSequential()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pageID := PageID(i % numPages)
		if _, err := dm.ReadPage(pageID); err != nil {
			b.Fatalf("Failed to read page: %v", err)
		}
	}
}

func BenchmarkMmapDiskManager_MixedWorkload(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_mmap_mixed.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		b.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Create initial pages
	const numPages = 1000
	for i := 0; i < numPages; i++ {
		pageID, _ := dm.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		dm.WritePage(page)
	}

	b.ResetTimer()

	// 70% reads, 30% writes
	for i := 0; i < b.N; i++ {
		pageID := PageID(i % numPages)

		if i%10 < 7 {
			// Read
			if _, err := dm.ReadPage(pageID); err != nil {
				b.Fatalf("Failed to read: %v", err)
			}
		} else {
			// Write
			page := NewPage(pageID, PageTypeData)
			page.LSN = uint64(i)
			if err := dm.WritePage(page); err != nil {
				b.Fatalf("Failed to write: %v", err)
			}
		}
	}
}

func BenchmarkDiskManager_MixedWorkload(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_disk_mixed.db")

	dm, err := NewDiskManager(dbPath)
	if err != nil {
		b.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Create initial pages
	const numPages = 1000
	for i := 0; i < numPages; i++ {
		pageID, _ := dm.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		dm.WritePage(page)
	}

	b.ResetTimer()

	// 70% reads, 30% writes
	for i := 0; i < b.N; i++ {
		pageID := PageID(i % numPages)

		if i%10 < 7 {
			// Read
			if _, err := dm.ReadPage(pageID); err != nil {
				b.Fatalf("Failed to read: %v", err)
			}
		} else {
			// Write
			page := NewPage(pageID, PageTypeData)
			page.LSN = uint64(i)
			if err := dm.WritePage(page); err != nil {
				b.Fatalf("Failed to write: %v", err)
			}
		}
	}
}

func BenchmarkMmapDiskManager_Expansion(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_mmap_expand.db")

	// Small initial size to force expansion
	config := &MmapConfig{
		InitialSize: 10 * PageSize,
		GrowthSize:  10 * PageSize,
	}

	dm, err := NewMmapDiskManager(dbPath, config)
	if err != nil {
		b.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	b.ResetTimer()

	// This will trigger multiple expansions
	for i := 0; i < b.N; i++ {
		pageID, _ := dm.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		if err := dm.WritePage(page); err != nil {
			b.Fatalf("Failed to write: %v", err)
		}
	}
}

func BenchmarkMmapDiskManager_Sync(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_mmap_sync.db")

	dm, err := NewMmapDiskManager(dbPath, DefaultMmapConfig())
	if err != nil {
		b.Fatalf("Failed to create mmap disk manager: %v", err)
	}
	defer dm.Close()

	// Write some pages
	for i := 0; i < 100; i++ {
		pageID, _ := dm.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		dm.WritePage(page)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := dm.Sync(); err != nil {
			b.Fatalf("Failed to sync: %v", err)
		}
	}
}

func BenchmarkDiskManager_Sync(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_disk_sync.db")

	dm, err := NewDiskManager(dbPath)
	if err != nil {
		b.Fatalf("Failed to create disk manager: %v", err)
	}
	defer dm.Close()

	// Write some pages
	for i := 0; i < 100; i++ {
		pageID, _ := dm.AllocatePage()
		page := NewPage(pageID, PageTypeData)
		dm.WritePage(page)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := dm.Sync(); err != nil {
			b.Fatalf("Failed to sync: %v", err)
		}
	}
}
