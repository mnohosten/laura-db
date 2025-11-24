package metrics

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ResourceTracker tracks CPU, memory, and I/O usage
type ResourceTracker struct {
	enabled bool
	mu      sync.RWMutex

	// Memory tracking
	allocBytes      uint64 // Total bytes allocated
	allocObjects    uint64 // Total objects allocated
	heapInUse       uint64 // Current heap in use
	stackInUse      uint64 // Current stack in use

	// Goroutine tracking
	numGoroutines   uint64 // Current number of goroutines

	// I/O tracking
	bytesRead       uint64 // Total bytes read
	bytesWritten    uint64 // Total bytes written
	readsCompleted  uint64 // Total read operations
	writesCompleted uint64 // Total write operations

	// CPU tracking (approximated via goroutine count and GC stats)
	gcPauseTotal    uint64 // Total GC pause time in nanoseconds
	gcRuns          uint64 // Total GC runs

	// Sampling history for trend analysis
	sampleInterval  time.Duration
	maxSamples      int
	samples         []ResourceSample
	stopChan        chan struct{}
	wg              sync.WaitGroup
}

// ResourceSample represents a point-in-time resource snapshot
type ResourceSample struct {
	Timestamp       time.Time
	HeapInUse       uint64
	StackInUse      uint64
	NumGoroutines   int
	AllocBytes      uint64
	AllocObjects    uint64
	GCPauseNs       uint64
	GCRuns          uint32
}

// ResourceStats contains current resource usage statistics
type ResourceStats struct {
	// Memory
	AllocBytes       uint64  `json:"alloc_bytes"`
	AllocMB          float64 `json:"alloc_mb"`
	HeapInUse        uint64  `json:"heap_in_use_bytes"`
	HeapInUseMB      float64 `json:"heap_in_use_mb"`
	StackInUse       uint64  `json:"stack_in_use_bytes"`
	StackInUseMB     float64 `json:"stack_in_use_mb"`
	AllocObjects     uint64  `json:"alloc_objects"`

	// Goroutines
	NumGoroutines    int     `json:"num_goroutines"`

	// I/O
	BytesRead        uint64  `json:"bytes_read"`
	BytesWritten     uint64  `json:"bytes_written"`
	ReadsCompleted   uint64  `json:"reads_completed"`
	WritesCompleted  uint64  `json:"writes_completed"`

	// GC
	GCPauseTotalMs   float64 `json:"gc_pause_total_ms"`
	GCRuns           uint32  `json:"gc_runs"`
	LastGCTimeNs     uint64  `json:"last_gc_time_ns"`

	// Runtime
	NumCPU           int     `json:"num_cpu"`
	GoVersion        string  `json:"go_version"`
}

// ResourceTrackerConfig holds configuration for the resource tracker
type ResourceTrackerConfig struct {
	Enabled        bool
	SampleInterval time.Duration // How often to sample resources (default: 1s)
	MaxSamples     int           // Maximum samples to keep in history (default: 60)
}

// DefaultResourceTrackerConfig returns default configuration
func DefaultResourceTrackerConfig() *ResourceTrackerConfig {
	return &ResourceTrackerConfig{
		Enabled:        true,
		SampleInterval: 1 * time.Second,
		MaxSamples:     60, // Keep 1 minute of samples at 1s interval
	}
}

// NewResourceTracker creates a new resource tracker
func NewResourceTracker(config *ResourceTrackerConfig) *ResourceTracker {
	if config == nil {
		config = DefaultResourceTrackerConfig()
	}

	rt := &ResourceTracker{
		enabled:        config.Enabled,
		sampleInterval: config.SampleInterval,
		maxSamples:     config.MaxSamples,
		samples:        make([]ResourceSample, 0, config.MaxSamples),
		stopChan:       make(chan struct{}),
	}

	if rt.enabled {
		rt.startSampling()
	}

	return rt
}

// Enable enables resource tracking
func (rt *ResourceTracker) Enable() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if !rt.enabled {
		rt.enabled = true
		rt.startSampling()
	}
}

// Disable disables resource tracking
func (rt *ResourceTracker) Disable() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if rt.enabled {
		rt.enabled = false
		close(rt.stopChan)
		rt.wg.Wait()
		rt.stopChan = make(chan struct{}) // Reset for potential re-enable
	}
}

// IsEnabled returns whether tracking is enabled
func (rt *ResourceTracker) IsEnabled() bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.enabled
}

// startSampling starts the background sampling goroutine
func (rt *ResourceTracker) startSampling() {
	rt.wg.Add(1)
	go rt.samplingLoop()
}

// samplingLoop runs the periodic resource sampling
func (rt *ResourceTracker) samplingLoop() {
	defer rt.wg.Done()

	ticker := time.NewTicker(rt.sampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rt.takeSample()
		case <-rt.stopChan:
			return
		}
	}
}

// takeSample captures a resource snapshot
func (rt *ResourceTracker) takeSample() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	sample := ResourceSample{
		Timestamp:     time.Now(),
		HeapInUse:     m.HeapInuse,
		StackInUse:    m.StackInuse,
		NumGoroutines: runtime.NumGoroutine(),
		AllocBytes:    m.TotalAlloc,
		AllocObjects:  m.Mallocs - m.Frees,
		GCPauseNs:     m.PauseTotalNs,
		GCRuns:        m.NumGC,
	}

	// Update current values
	atomic.StoreUint64(&rt.heapInUse, sample.HeapInUse)
	atomic.StoreUint64(&rt.stackInUse, sample.StackInUse)
	atomic.StoreUint64(&rt.numGoroutines, uint64(sample.NumGoroutines))
	atomic.StoreUint64(&rt.allocBytes, sample.AllocBytes)
	atomic.StoreUint64(&rt.allocObjects, sample.AllocObjects)
	atomic.StoreUint64(&rt.gcPauseTotal, sample.GCPauseNs)
	atomic.StoreUint64(&rt.gcRuns, uint64(sample.GCRuns))

	// Add to history
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if len(rt.samples) >= rt.maxSamples {
		// Remove oldest sample
		rt.samples = rt.samples[1:]
	}
	rt.samples = append(rt.samples, sample)
}

// RecordRead records a read operation
func (rt *ResourceTracker) RecordRead(bytes uint64) {
	if !rt.IsEnabled() {
		return
	}
	atomic.AddUint64(&rt.bytesRead, bytes)
	atomic.AddUint64(&rt.readsCompleted, 1)
}

// RecordWrite records a write operation
func (rt *ResourceTracker) RecordWrite(bytes uint64) {
	if !rt.IsEnabled() {
		return
	}
	atomic.AddUint64(&rt.bytesWritten, bytes)
	atomic.AddUint64(&rt.writesCompleted, 1)
}

// GetStats returns current resource statistics
func (rt *ResourceTracker) GetStats() *ResourceStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	bytesRead := atomic.LoadUint64(&rt.bytesRead)
	bytesWritten := atomic.LoadUint64(&rt.bytesWritten)
	readsCompleted := atomic.LoadUint64(&rt.readsCompleted)
	writesCompleted := atomic.LoadUint64(&rt.writesCompleted)

	return &ResourceStats{
		AllocBytes:      m.TotalAlloc,
		AllocMB:         float64(m.TotalAlloc) / 1024 / 1024,
		HeapInUse:       m.HeapInuse,
		HeapInUseMB:     float64(m.HeapInuse) / 1024 / 1024,
		StackInUse:      m.StackInuse,
		StackInUseMB:    float64(m.StackInuse) / 1024 / 1024,
		AllocObjects:    m.Mallocs - m.Frees,
		NumGoroutines:   runtime.NumGoroutine(),
		BytesRead:       bytesRead,
		BytesWritten:    bytesWritten,
		ReadsCompleted:  readsCompleted,
		WritesCompleted: writesCompleted,
		GCPauseTotalMs:  float64(m.PauseTotalNs) / 1e6,
		GCRuns:          m.NumGC,
		LastGCTimeNs:    m.LastGC,
		NumCPU:          runtime.NumCPU(),
		GoVersion:       runtime.Version(),
	}
}

// GetSamples returns recent resource samples
func (rt *ResourceTracker) GetSamples() []ResourceSample {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	// Return a copy
	samples := make([]ResourceSample, len(rt.samples))
	copy(samples, rt.samples)
	return samples
}

// GetTrends returns trend analysis of resource usage
func (rt *ResourceTracker) GetTrends() map[string]interface{} {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	if len(rt.samples) == 0 {
		return map[string]interface{}{
			"samples": 0,
		}
	}

	// Calculate trends
	first := rt.samples[0]
	last := rt.samples[len(rt.samples)-1]

	heapGrowth := int64(last.HeapInUse) - int64(first.HeapInUse)
	goroutineGrowth := last.NumGoroutines - first.NumGoroutines

	var avgHeap, avgStack, avgGoroutines float64
	for _, sample := range rt.samples {
		avgHeap += float64(sample.HeapInUse)
		avgStack += float64(sample.StackInUse)
		avgGoroutines += float64(sample.NumGoroutines)
	}
	count := float64(len(rt.samples))
	avgHeap /= count
	avgStack /= count
	avgGoroutines /= count

	return map[string]interface{}{
		"samples":           len(rt.samples),
		"time_range_sec":    last.Timestamp.Sub(first.Timestamp).Seconds(),
		"heap_growth_bytes": heapGrowth,
		"heap_growth_mb":    float64(heapGrowth) / 1024 / 1024,
		"goroutine_growth":  goroutineGrowth,
		"avg_heap_bytes":    avgHeap,
		"avg_heap_mb":       avgHeap / 1024 / 1024,
		"avg_stack_bytes":   avgStack,
		"avg_stack_mb":      avgStack / 1024 / 1024,
		"avg_goroutines":    avgGoroutines,
		"current_heap":      last.HeapInUse,
		"current_heap_mb":   float64(last.HeapInUse) / 1024 / 1024,
		"current_goroutines": last.NumGoroutines,
	}
}

// ClearSamples clears the sample history
func (rt *ResourceTracker) ClearSamples() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.samples = make([]ResourceSample, 0, rt.maxSamples)
}

// Close stops the resource tracker
func (rt *ResourceTracker) Close() {
	rt.Disable()
}
