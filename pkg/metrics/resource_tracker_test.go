package metrics

import (
	"runtime"
	"testing"
	"time"
)

func TestResourceTracker_EnableDisable(t *testing.T) {
	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled:        true,
		SampleInterval: 100 * time.Millisecond,
		MaxSamples:     10,
	})
	defer rt.Close()

	if !rt.IsEnabled() {
		t.Error("Expected tracker to be enabled")
	}

	rt.Disable()

	if rt.IsEnabled() {
		t.Error("Expected tracker to be disabled")
	}

	rt.Enable()

	if !rt.IsEnabled() {
		t.Error("Expected tracker to be enabled")
	}
}

func TestResourceTracker_GetStats(t *testing.T) {
	rt := NewResourceTracker(DefaultResourceTrackerConfig())
	defer rt.Close()

	stats := rt.GetStats()

	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if stats.NumCPU == 0 {
		t.Error("Expected non-zero CPU count")
	}

	if stats.GoVersion == "" {
		t.Error("Expected non-empty Go version")
	}

	if stats.NumGoroutines == 0 {
		t.Error("Expected non-zero goroutine count")
	}

	// Memory stats should be reasonable
	if stats.HeapInUse == 0 {
		t.Error("Expected non-zero heap in use")
	}

	if stats.AllocBytes == 0 {
		t.Error("Expected non-zero allocated bytes")
	}
}

func TestResourceTracker_RecordIO(t *testing.T) {
	rt := NewResourceTracker(DefaultResourceTrackerConfig())
	defer rt.Close()

	rt.RecordRead(1024)
	rt.RecordRead(2048)
	rt.RecordWrite(512)
	rt.RecordWrite(1024)

	stats := rt.GetStats()

	if stats.BytesRead != 3072 {
		t.Errorf("Expected 3072 bytes read, got %d", stats.BytesRead)
	}

	if stats.BytesWritten != 1536 {
		t.Errorf("Expected 1536 bytes written, got %d", stats.BytesWritten)
	}

	if stats.ReadsCompleted != 2 {
		t.Errorf("Expected 2 reads completed, got %d", stats.ReadsCompleted)
	}

	if stats.WritesCompleted != 2 {
		t.Errorf("Expected 2 writes completed, got %d", stats.WritesCompleted)
	}
}

func TestResourceTracker_Sampling(t *testing.T) {
	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled:        true,
		SampleInterval: 50 * time.Millisecond,
		MaxSamples:     5,
	})
	defer rt.Close()

	// Wait for a few samples to be collected
	time.Sleep(250 * time.Millisecond)

	samples := rt.GetSamples()

	if len(samples) == 0 {
		t.Error("Expected at least one sample")
	}

	if len(samples) > 5 {
		t.Errorf("Expected at most 5 samples, got %d", len(samples))
	}

	// Verify sample structure
	for _, sample := range samples {
		if sample.Timestamp.IsZero() {
			t.Error("Expected non-zero timestamp")
		}
		if sample.NumGoroutines == 0 {
			t.Error("Expected non-zero goroutine count")
		}
	}
}

func TestResourceTracker_MaxSamples(t *testing.T) {
	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled:        true,
		SampleInterval: 20 * time.Millisecond,
		MaxSamples:     3,
	})
	defer rt.Close()

	// Wait for more than maxSamples intervals
	time.Sleep(150 * time.Millisecond)

	samples := rt.GetSamples()

	if len(samples) > 3 {
		t.Errorf("Expected at most 3 samples, got %d", len(samples))
	}
}

func TestResourceTracker_GetTrends(t *testing.T) {
	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled:        true,
		SampleInterval: 50 * time.Millisecond,
		MaxSamples:     10,
	})
	defer rt.Close()

	// Wait for samples
	time.Sleep(200 * time.Millisecond)

	trends := rt.GetTrends()

	if trends["samples"].(int) == 0 {
		t.Error("Expected at least one sample")
	}

	// Should have trend data
	if _, ok := trends["heap_growth_bytes"]; !ok {
		t.Error("Expected heap_growth_bytes in trends")
	}

	if _, ok := trends["avg_heap_bytes"]; !ok {
		t.Error("Expected avg_heap_bytes in trends")
	}

	if _, ok := trends["current_goroutines"]; !ok {
		t.Error("Expected current_goroutines in trends")
	}
}

func TestResourceTracker_EmptyTrends(t *testing.T) {
	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled:        false,
		SampleInterval: 1 * time.Second,
		MaxSamples:     10,
	})
	defer rt.Close()

	trends := rt.GetTrends()

	if trends["samples"].(int) != 0 {
		t.Errorf("Expected 0 samples, got %v", trends["samples"])
	}
}

func TestResourceTracker_ClearSamples(t *testing.T) {
	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled:        true,
		SampleInterval: 50 * time.Millisecond,
		MaxSamples:     10,
	})
	defer rt.Close()

	// Wait for samples
	time.Sleep(150 * time.Millisecond)

	samples := rt.GetSamples()
	if len(samples) == 0 {
		t.Error("Expected at least one sample before clear")
	}

	rt.ClearSamples()

	samples = rt.GetSamples()
	if len(samples) != 0 {
		t.Errorf("Expected 0 samples after clear, got %d", len(samples))
	}
}

func TestResourceTracker_DisabledRecordIO(t *testing.T) {
	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled: false,
	})
	defer rt.Close()

	rt.RecordRead(1024)
	rt.RecordWrite(512)

	stats := rt.GetStats()

	// Should not record when disabled
	if stats.BytesRead != 0 {
		t.Errorf("Expected 0 bytes read when disabled, got %d", stats.BytesRead)
	}

	if stats.BytesWritten != 0 {
		t.Errorf("Expected 0 bytes written when disabled, got %d", stats.BytesWritten)
	}
}

func TestResourceTracker_DefaultConfig(t *testing.T) {
	config := DefaultResourceTrackerConfig()

	if !config.Enabled {
		t.Error("Expected default config to be enabled")
	}

	if config.SampleInterval != 1*time.Second {
		t.Errorf("Expected 1s sample interval, got %v", config.SampleInterval)
	}

	if config.MaxSamples != 60 {
		t.Errorf("Expected 60 max samples, got %d", config.MaxSamples)
	}
}

func TestResourceTracker_MemoryAllocations(t *testing.T) {
	rt := NewResourceTracker(DefaultResourceTrackerConfig())
	defer rt.Close()

	// Get initial stats
	initialStats := rt.GetStats()
	initialAlloc := initialStats.AllocBytes

	// Allocate some memory
	_ = make([]byte, 1024*1024) // 1MB

	// Force GC to get accurate stats
	runtime.GC()

	// Get new stats
	newStats := rt.GetStats()

	// Allocations should have increased
	if newStats.AllocBytes <= initialAlloc {
		t.Error("Expected allocations to increase")
	}
}

func TestResourceTracker_SampleTimestamps(t *testing.T) {
	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled:        true,
		SampleInterval: 50 * time.Millisecond,
		MaxSamples:     5,
	})
	defer rt.Close()

	time.Sleep(200 * time.Millisecond)

	samples := rt.GetSamples()
	if len(samples) < 2 {
		t.Skip("Need at least 2 samples for this test")
	}

	// Verify timestamps are in order
	for i := 1; i < len(samples); i++ {
		if !samples[i].Timestamp.After(samples[i-1].Timestamp) {
			t.Error("Expected timestamps to be in chronological order")
		}
	}
}

func TestResourceTracker_Close(t *testing.T) {
	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled:        true,
		SampleInterval: 50 * time.Millisecond,
		MaxSamples:     10,
	})

	if !rt.IsEnabled() {
		t.Error("Expected tracker to be enabled")
	}

	rt.Close()

	if rt.IsEnabled() {
		t.Error("Expected tracker to be disabled after close")
	}

	// Should be safe to call Close multiple times
	rt.Close()
}

func TestResourceTracker_StatsFormat(t *testing.T) {
	rt := NewResourceTracker(DefaultResourceTrackerConfig())
	defer rt.Close()

	stats := rt.GetStats()

	// Check MB conversions are reasonable
	if stats.AllocMB != float64(stats.AllocBytes)/1024/1024 {
		t.Error("Incorrect AllocMB calculation")
	}

	if stats.HeapInUseMB != float64(stats.HeapInUse)/1024/1024 {
		t.Error("Incorrect HeapInUseMB calculation")
	}

	if stats.StackInUseMB != float64(stats.StackInUse)/1024/1024 {
		t.Error("Incorrect StackInUseMB calculation")
	}
}

func TestResourceTracker_ConcurrentIO(t *testing.T) {
	rt := NewResourceTracker(DefaultResourceTrackerConfig())
	defer rt.Close()

	done := make(chan bool, 2)

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			rt.RecordRead(1024)
		}
		done <- true
	}()

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			rt.RecordWrite(512)
		}
		done <- true
	}()

	// Wait for completion
	<-done
	<-done

	stats := rt.GetStats()

	if stats.BytesRead != 102400 {
		t.Errorf("Expected 102400 bytes read, got %d", stats.BytesRead)
	}

	if stats.BytesWritten != 51200 {
		t.Errorf("Expected 51200 bytes written, got %d", stats.BytesWritten)
	}

	if stats.ReadsCompleted != 100 {
		t.Errorf("Expected 100 reads, got %d", stats.ReadsCompleted)
	}

	if stats.WritesCompleted != 100 {
		t.Errorf("Expected 100 writes, got %d", stats.WritesCompleted)
	}
}

func TestResourceTracker_GCStats(t *testing.T) {
	rt := NewResourceTracker(DefaultResourceTrackerConfig())
	defer rt.Close()

	// Force a GC run
	runtime.GC()

	stats := rt.GetStats()

	// Should have at least one GC run
	if stats.GCRuns == 0 {
		t.Error("Expected at least one GC run")
	}

	// GC pause time should be non-zero
	if stats.GCPauseTotalMs < 0 {
		t.Error("Expected non-negative GC pause time")
	}
}

func TestResourceTracker_EnableAfterDisable(t *testing.T) {
	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled:        true,
		SampleInterval: 50 * time.Millisecond,
		MaxSamples:     10,
	})
	defer rt.Close()

	// Wait for samples
	time.Sleep(100 * time.Millisecond)
	samples1 := rt.GetSamples()

	// Disable
	rt.Disable()
	time.Sleep(100 * time.Millisecond)

	// No new samples should be added
	samples2 := rt.GetSamples()
	if len(samples2) != len(samples1) {
		// Could be equal or differ by 1 due to timing
		if len(samples2) > len(samples1)+1 {
			t.Error("Expected no new samples while disabled")
		}
	}

	// Re-enable
	rt.Enable()
	time.Sleep(100 * time.Millisecond)

	// Should have new samples
	samples3 := rt.GetSamples()
	if len(samples3) <= len(samples2) {
		t.Error("Expected new samples after re-enabling")
	}
}
