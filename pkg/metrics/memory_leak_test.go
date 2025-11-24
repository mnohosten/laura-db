package metrics

import (
	"runtime"
	"testing"
	"time"
)

// MemorySnapshot captures a point-in-time memory state
type MemorySnapshot struct {
	HeapAlloc    uint64
	HeapInUse    uint64
	HeapObjects  uint64
	NumGoroutine int
	Timestamp    time.Time
}

// TakeMemorySnapshot captures current memory statistics
func TakeMemorySnapshot() MemorySnapshot {
	runtime.GC() // Force GC to get accurate stats
	runtime.GC() // Run twice to ensure cleanup

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return MemorySnapshot{
		HeapAlloc:    m.HeapAlloc,
		HeapInUse:    m.HeapInuse,
		HeapObjects:  m.HeapObjects,
		NumGoroutine: runtime.NumGoroutine(),
		Timestamp:    time.Now(),
	}
}

// MemoryLeakDetector provides utilities for detecting memory leaks in tests
type MemoryLeakDetector struct {
	baseline         MemorySnapshot
	threshold        float64 // Percentage threshold for memory growth (e.g., 0.1 = 10%)
	goroutineBuffer  int     // Allow some goroutine variance
}

// NewMemoryLeakDetector creates a new leak detector with the given threshold
func NewMemoryLeakDetector(threshold float64, goroutineBuffer int) *MemoryLeakDetector {
	return &MemoryLeakDetector{
		threshold:       threshold,
		goroutineBuffer: goroutineBuffer,
	}
}

// SetBaseline captures the baseline memory state
func (d *MemoryLeakDetector) SetBaseline() {
	d.baseline = TakeMemorySnapshot()
}

// CheckLeak compares current memory to baseline and returns true if leak detected
func (d *MemoryLeakDetector) CheckLeak(t *testing.T, operation string) bool {
	current := TakeMemorySnapshot()

	// Check heap allocation growth
	heapGrowth := float64(current.HeapAlloc) - float64(d.baseline.HeapAlloc)
	heapGrowthPercent := heapGrowth / float64(d.baseline.HeapAlloc)

	// Check goroutine leak
	goroutineDiff := current.NumGoroutine - d.baseline.NumGoroutine

	leaked := false

	if heapGrowthPercent > d.threshold {
		t.Logf("LEAK DETECTED in %s: Heap grew by %.2f%% (%.2f MB)",
			operation,
			heapGrowthPercent*100,
			heapGrowth/(1024*1024))
		leaked = true
	}

	if goroutineDiff > d.goroutineBuffer {
		t.Logf("GOROUTINE LEAK DETECTED in %s: %d goroutines leaked",
			operation,
			goroutineDiff)
		leaked = true
	}

	if !leaked {
		t.Logf("PASS %s: Heap: %.2f MB (%.1f%% growth), Goroutines: %d->%d",
			operation,
			float64(current.HeapAlloc)/(1024*1024),
			heapGrowthPercent*100,
			d.baseline.NumGoroutine,
			current.NumGoroutine)
	}

	return leaked
}

// TestMemoryLeak_ResourceTracker verifies ResourceTracker doesn't leak memory
func TestMemoryLeak_ResourceTracker(t *testing.T) {
	detector := NewMemoryLeakDetector(0.2, 2) // 20% threshold, 2 goroutine buffer

	// Set baseline
	detector.SetBaseline()

	// Run operations that should not leak
	const iterations = 100
	for i := 0; i < iterations; i++ {
		rt := NewResourceTracker(&ResourceTrackerConfig{
			Enabled:        true,
			SampleInterval: 10 * time.Millisecond,
			MaxSamples:     5,
		})

		// Do some operations
		rt.RecordRead(1024)
		rt.RecordWrite(512)
		_ = rt.GetStats()
		_ = rt.GetSamples()

		// Clean up
		rt.Close()
	}

	// Wait for goroutines to finish
	time.Sleep(100 * time.Millisecond)

	// Check for leaks
	if detector.CheckLeak(t, "ResourceTracker lifecycle") {
		t.Error("Memory leak detected in ResourceTracker")
	}
}

// TestMemoryLeak_ResourceTrackerSampling verifies sampling doesn't leak
func TestMemoryLeak_ResourceTrackerSampling(t *testing.T) {
	detector := NewMemoryLeakDetector(0.15, 2)
	detector.SetBaseline()

	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled:        true,
		SampleInterval: 5 * time.Millisecond,
		MaxSamples:     10,
	})
	defer rt.Close()

	// Let it sample for a while
	time.Sleep(200 * time.Millisecond)

	// Get samples multiple times
	for i := 0; i < 50; i++ {
		_ = rt.GetSamples()
		_ = rt.GetTrends()
	}

	if detector.CheckLeak(t, "ResourceTracker sampling") {
		t.Error("Memory leak detected in ResourceTracker sampling")
	}
}

// TestMemoryLeak_ResourceTrackerIO verifies I/O tracking doesn't leak
func TestMemoryLeak_ResourceTrackerIO(t *testing.T) {
	detector := NewMemoryLeakDetector(0.15, 2)

	rt := NewResourceTracker(DefaultResourceTrackerConfig())
	defer rt.Close()

	detector.SetBaseline()

	// Record lots of I/O operations
	for i := 0; i < 10000; i++ {
		rt.RecordRead(1024)
		rt.RecordWrite(512)
	}

	if detector.CheckLeak(t, "ResourceTracker I/O recording") {
		t.Error("Memory leak detected in I/O recording")
	}
}

// TestMemoryLeak_MultipleTrackers verifies concurrent trackers don't leak
func TestMemoryLeak_MultipleTrackers(t *testing.T) {
	detector := NewMemoryLeakDetector(0.3, 5)
	detector.SetBaseline()

	const numTrackers = 50
	trackers := make([]*ResourceTracker, numTrackers)

	// Create multiple trackers
	for i := 0; i < numTrackers; i++ {
		trackers[i] = NewResourceTracker(&ResourceTrackerConfig{
			Enabled:        true,
			SampleInterval: 20 * time.Millisecond,
			MaxSamples:     5,
		})
	}

	// Use them
	for i := 0; i < numTrackers; i++ {
		trackers[i].RecordRead(uint64(i * 100))
		_ = trackers[i].GetStats()
	}

	// Clean up all
	for i := 0; i < numTrackers; i++ {
		trackers[i].Close()
	}

	// Wait for cleanup
	time.Sleep(150 * time.Millisecond)

	if detector.CheckLeak(t, "Multiple concurrent trackers") {
		t.Error("Memory leak detected with multiple trackers")
	}
}

// TestMemoryLeak_SampleClearing verifies sample clearing releases memory
func TestMemoryLeak_SampleClearing(t *testing.T) {
	detector := NewMemoryLeakDetector(0.2, 2)

	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled:        true,
		SampleInterval: 5 * time.Millisecond,
		MaxSamples:     100,
	})
	defer rt.Close()

	// Collect many samples
	time.Sleep(600 * time.Millisecond)

	detector.SetBaseline()

	// Clear samples multiple times
	for i := 0; i < 20; i++ {
		rt.ClearSamples()
		time.Sleep(50 * time.Millisecond)
		rt.ClearSamples()
	}

	if detector.CheckLeak(t, "Sample clearing") {
		t.Error("Memory leak detected in sample clearing")
	}
}

// TestMemoryLeak_EnableDisableCycle verifies enable/disable doesn't leak
func TestMemoryLeak_EnableDisableCycle(t *testing.T) {
	detector := NewMemoryLeakDetector(0.2, 2)

	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled:        true,
		SampleInterval: 10 * time.Millisecond,
		MaxSamples:     10,
	})
	defer rt.Close()

	detector.SetBaseline()

	// Cycle enable/disable many times
	for i := 0; i < 50; i++ {
		rt.Disable()
		time.Sleep(5 * time.Millisecond)
		rt.Enable()
		time.Sleep(5 * time.Millisecond)
	}

	// Final disable to stop sampling
	rt.Disable()
	time.Sleep(50 * time.Millisecond)

	if detector.CheckLeak(t, "Enable/Disable cycling") {
		t.Error("Memory leak detected in enable/disable cycle")
	}
}

// TestMemoryGrowth_Trends verifies trend calculations don't grow unbounded
func TestMemoryGrowth_Trends(t *testing.T) {
	detector := NewMemoryLeakDetector(0.15, 2)

	rt := NewResourceTracker(&ResourceTrackerConfig{
		Enabled:        true,
		SampleInterval: 5 * time.Millisecond,
		MaxSamples:     20,
	})
	defer rt.Close()

	// Wait for samples
	time.Sleep(150 * time.Millisecond)

	detector.SetBaseline()

	// Call GetTrends many times
	for i := 0; i < 1000; i++ {
		_ = rt.GetTrends()
	}

	if detector.CheckLeak(t, "Trend calculations") {
		t.Error("Memory leak detected in trend calculations")
	}
}

// TestMemorySnapshot_Consistency verifies snapshot consistency
func TestMemorySnapshot_Consistency(t *testing.T) {
	s1 := TakeMemorySnapshot()
	time.Sleep(10 * time.Millisecond)
	s2 := TakeMemorySnapshot()

	if s2.Timestamp.Before(s1.Timestamp) {
		t.Error("Timestamps should be chronological")
	}

	// Basic sanity checks
	if s1.HeapAlloc == 0 {
		t.Error("Expected non-zero heap allocation")
	}

	if s1.NumGoroutine == 0 {
		t.Error("Expected non-zero goroutine count")
	}
}

// BenchmarkMemorySnapshot measures snapshot overhead
func BenchmarkMemorySnapshot(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = TakeMemorySnapshot()
	}
}

// BenchmarkLeakDetector measures detector overhead
func BenchmarkLeakDetector(b *testing.B) {
	detector := NewMemoryLeakDetector(0.1, 2)
	detector.SetBaseline()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.CheckLeak(&testing.T{}, "benchmark")
	}
}
