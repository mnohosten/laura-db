// Package chaos provides fault injection and chaos engineering utilities for LauraDB.
//
// This package enables testing system resilience by simulating various failure scenarios
// including disk failures, network partitions, process crashes, and resource exhaustion.
package chaos

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// FaultType represents different types of failures that can be injected
type FaultType int

const (
	FaultTypeNone FaultType = iota
	FaultTypeDiskRead
	FaultTypeDiskWrite
	FaultTypeDiskFull
	FaultTypeDiskCorruption
	FaultTypeNetworkPartition
	FaultTypeProcessCrash
	FaultTypeSlowIO
	FaultTypeMemoryPressure
)

func (ft FaultType) String() string {
	switch ft {
	case FaultTypeNone:
		return "None"
	case FaultTypeDiskRead:
		return "DiskRead"
	case FaultTypeDiskWrite:
		return "DiskWrite"
	case FaultTypeDiskFull:
		return "DiskFull"
	case FaultTypeDiskCorruption:
		return "DiskCorruption"
	case FaultTypeNetworkPartition:
		return "NetworkPartition"
	case FaultTypeProcessCrash:
		return "ProcessCrash"
	case FaultTypeSlowIO:
		return "SlowIO"
	case FaultTypeMemoryPressure:
		return "MemoryPressure"
	default:
		return "Unknown"
	}
}

// FaultInjector manages fault injection for chaos testing
type FaultInjector struct {
	mu              sync.RWMutex
	enabled         bool
	faults          map[FaultType]*FaultConfig
	triggerCount    map[FaultType]*int64
	rng             *rand.Rand
	eventCallbacks  []EventCallback
	activeScenarios []string
}

// FaultConfig defines the configuration for a specific fault type
type FaultConfig struct {
	Type        FaultType
	Enabled     bool
	Probability float64       // 0.0 to 1.0 (probability of fault occurring)
	Duration    time.Duration // How long fault lasts (0 = permanent until disabled)
	Delay       time.Duration // Delay before returning error (simulates slow I/O)
	ErrorMsg    string        // Custom error message
	Callback    func()        // Optional callback when fault is triggered
}

// EventCallback is called when chaos events occur
type EventCallback func(event ChaosEvent)

// ChaosEvent represents a chaos engineering event
type ChaosEvent struct {
	Timestamp time.Time
	Type      FaultType
	Scenario  string
	Message   string
	Triggered bool
}

// NewFaultInjector creates a new fault injector
func NewFaultInjector(seed int64) *FaultInjector {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	return &FaultInjector{
		enabled:      false,
		faults:       make(map[FaultType]*FaultConfig),
		triggerCount: make(map[FaultType]*int64),
		rng:          rand.New(rand.NewSource(seed)),
	}
}

// Enable enables fault injection
func (fi *FaultInjector) Enable() {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.enabled = true
	fi.notifyEvent(ChaosEvent{
		Timestamp: time.Now(),
		Type:      FaultTypeNone,
		Message:   "Fault injection enabled",
	})
}

// Disable disables all fault injection
func (fi *FaultInjector) Disable() {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.enabled = false
	fi.notifyEvent(ChaosEvent{
		Timestamp: time.Now(),
		Type:      FaultTypeNone,
		Message:   "Fault injection disabled",
	})
}

// IsEnabled returns whether fault injection is enabled
func (fi *FaultInjector) IsEnabled() bool {
	fi.mu.RLock()
	defer fi.mu.RUnlock()
	return fi.enabled
}

// ConfigureFault configures a specific fault type
func (fi *FaultInjector) ConfigureFault(config *FaultConfig) {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.faults[config.Type] = config
}

// EnableFault enables a specific fault type with given probability
func (fi *FaultInjector) EnableFault(faultType FaultType, probability float64) {
	fi.ConfigureFault(&FaultConfig{
		Type:        faultType,
		Enabled:     true,
		Probability: probability,
		ErrorMsg:    fmt.Sprintf("Chaos: %s fault injected", faultType),
	})
}

// DisableFault disables a specific fault type
func (fi *FaultInjector) DisableFault(faultType FaultType) {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	if config, exists := fi.faults[faultType]; exists {
		config.Enabled = false
	}
}

// ShouldInjectFault determines if a fault should be injected
func (fi *FaultInjector) ShouldInjectFault(faultType FaultType) (bool, error) {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	if !fi.enabled {
		return false, nil
	}

	config, exists := fi.faults[faultType]
	if !exists || !config.Enabled {
		return false, nil
	}

	// Check probability
	if fi.rng.Float64() > config.Probability {
		return false, nil
	}

	// Increment trigger count
	fi.incrementTriggerCount(faultType)

	// Apply delay if configured
	if config.Delay > 0 {
		time.Sleep(config.Delay)
	}

	// Notify event
	fi.notifyEvent(ChaosEvent{
		Timestamp: time.Now(),
		Type:      faultType,
		Message:   config.ErrorMsg,
		Triggered: true,
	})

	// Call callback if configured
	if config.Callback != nil {
		config.Callback()
	}

	// Return error
	if config.ErrorMsg != "" {
		return true, errors.New(config.ErrorMsg)
	}

	return true, fmt.Errorf("chaos: %s fault", faultType)
}

// InjectFault forces a fault to occur (ignoring probability)
func (fi *FaultInjector) InjectFault(faultType FaultType) error {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	if !fi.enabled {
		return nil
	}

	config, exists := fi.faults[faultType]
	if !exists || !config.Enabled {
		return nil
	}

	fi.incrementTriggerCount(faultType)

	if config.Delay > 0 {
		time.Sleep(config.Delay)
	}

	fi.notifyEvent(ChaosEvent{
		Timestamp: time.Now(),
		Type:      faultType,
		Message:   config.ErrorMsg,
		Triggered: true,
	})

	if config.Callback != nil {
		config.Callback()
	}

	if config.ErrorMsg != "" {
		return errors.New(config.ErrorMsg)
	}

	return fmt.Errorf("chaos: %s fault", faultType)
}

// GetTriggerCount returns the number of times a fault was triggered
func (fi *FaultInjector) GetTriggerCount(faultType FaultType) int64 {
	fi.mu.RLock()
	defer fi.mu.RUnlock()
	if counter, exists := fi.triggerCount[faultType]; exists {
		return atomic.LoadInt64(counter)
	}
	return 0
}

// incrementTriggerCount safely increments the trigger count for a fault type
// Must be called while holding at least a read lock
func (fi *FaultInjector) incrementTriggerCount(faultType FaultType) {
	counter, exists := fi.triggerCount[faultType]
	if !exists {
		// Need to upgrade to write lock temporarily
		fi.mu.RUnlock()
		fi.mu.Lock()
		// Double-check after getting write lock
		if _, exists := fi.triggerCount[faultType]; !exists {
			var zero int64
			fi.triggerCount[faultType] = &zero
		}
		counter = fi.triggerCount[faultType]
		fi.mu.Unlock()
		fi.mu.RLock()
	}

	atomic.AddInt64(counter, 1)
}

// Reset resets all fault counters
func (fi *FaultInjector) Reset() {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	for _, counter := range fi.triggerCount {
		atomic.StoreInt64(counter, 0)
	}
}

// AddEventCallback adds a callback for chaos events
func (fi *FaultInjector) AddEventCallback(callback EventCallback) {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.eventCallbacks = append(fi.eventCallbacks, callback)
}

// notifyEvent notifies all registered callbacks
func (fi *FaultInjector) notifyEvent(event ChaosEvent) {
	for _, callback := range fi.eventCallbacks {
		callback(event)
	}
}

// ChaosDisk wraps an io.ReadWriteSeeker with fault injection
type ChaosDisk struct {
	inner    io.ReadWriteSeeker
	injector *FaultInjector
	path     string
}

// NewChaosDisk creates a new ChaosDisk wrapper
func NewChaosDisk(inner io.ReadWriteSeeker, injector *FaultInjector, path string) *ChaosDisk {
	return &ChaosDisk{
		inner:    inner,
		injector: injector,
		path:     path,
	}
}

// Read implements io.Reader with fault injection
func (cd *ChaosDisk) Read(p []byte) (n int, err error) {
	if shouldInject, err := cd.injector.ShouldInjectFault(FaultTypeDiskRead); shouldInject {
		return 0, err
	}

	if shouldInject, err := cd.injector.ShouldInjectFault(FaultTypeSlowIO); shouldInject {
		// Delay already applied in ShouldInjectFault
		_ = err
	}

	return cd.inner.Read(p)
}

// Write implements io.Writer with fault injection
func (cd *ChaosDisk) Write(p []byte) (n int, err error) {
	if shouldInject, err := cd.injector.ShouldInjectFault(FaultTypeDiskWrite); shouldInject {
		return 0, err
	}

	if shouldInject, err := cd.injector.ShouldInjectFault(FaultTypeDiskFull); shouldInject {
		return 0, err
	}

	if shouldInject, err := cd.injector.ShouldInjectFault(FaultTypeSlowIO); shouldInject {
		_ = err
	}

	return cd.inner.Write(p)
}

// Seek implements io.Seeker
func (cd *ChaosDisk) Seek(offset int64, whence int) (int64, error) {
	return cd.inner.Seek(offset, whence)
}

// ChaosFile wraps *os.File with fault injection
type ChaosFile struct {
	*os.File
	injector *FaultInjector
}

// NewChaosFile creates a new ChaosFile wrapper
func NewChaosFile(file *os.File, injector *FaultInjector) *ChaosFile {
	return &ChaosFile{
		File:     file,
		injector: injector,
	}
}

// Read implements io.Reader with fault injection
func (cf *ChaosFile) Read(p []byte) (n int, err error) {
	if shouldInject, err := cf.injector.ShouldInjectFault(FaultTypeDiskRead); shouldInject {
		return 0, err
	}
	return cf.File.Read(p)
}

// Write implements io.Writer with fault injection
func (cf *ChaosFile) Write(p []byte) (n int, err error) {
	if shouldInject, err := cf.injector.ShouldInjectFault(FaultTypeDiskWrite); shouldInject {
		return 0, err
	}

	if shouldInject, err := cf.injector.ShouldInjectFault(FaultTypeDiskFull); shouldInject {
		return 0, err
	}

	return cf.File.Write(p)
}

// WriteAt implements io.WriterAt with fault injection
func (cf *ChaosFile) WriteAt(p []byte, off int64) (n int, err error) {
	if shouldInject, err := cf.injector.ShouldInjectFault(FaultTypeDiskWrite); shouldInject {
		return 0, err
	}

	if shouldInject, err := cf.injector.ShouldInjectFault(FaultTypeDiskFull); shouldInject {
		return 0, err
	}

	return cf.File.WriteAt(p, off)
}

// ReadAt implements io.ReaderAt with fault injection
func (cf *ChaosFile) ReadAt(p []byte, off int64) (n int, err error) {
	if shouldInject, err := cf.injector.ShouldInjectFault(FaultTypeDiskRead); shouldInject {
		return 0, err
	}
	return cf.File.ReadAt(p, off)
}

// Sync implements file sync with fault injection
func (cf *ChaosFile) Sync() error {
	if shouldInject, err := cf.injector.ShouldInjectFault(FaultTypeDiskWrite); shouldInject {
		return err
	}
	return cf.File.Sync()
}
