package index

import (
	"sync"
	"time"
)

// IndexBuildState represents the state of index building
type IndexBuildState int

const (
	// IndexStateReady means index is ready to use
	IndexStateReady IndexBuildState = iota
	// IndexStateBuilding means index is currently being built
	IndexStateBuilding
	// IndexStateFailed means index build failed
	IndexStateFailed
)

// String returns the string representation of the build state
func (s IndexBuildState) String() string {
	switch s {
	case IndexStateReady:
		return "ready"
	case IndexStateBuilding:
		return "building"
	case IndexStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// IndexBuildProgress tracks the progress of index building
type IndexBuildProgress struct {
	State              IndexBuildState
	TotalDocuments     int
	ProcessedDocuments int
	ErrorMessage       string
	StartTime          time.Time
	EndTime            time.Time
	mu                 sync.RWMutex
}

// NewIndexBuildProgress creates a new progress tracker
func NewIndexBuildProgress() *IndexBuildProgress {
	return &IndexBuildProgress{
		State:     IndexStateReady,
		StartTime: time.Time{},
		EndTime:   time.Time{},
	}
}

// Start marks the index build as started
func (p *IndexBuildProgress) Start(totalDocs int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.State = IndexStateBuilding
	p.TotalDocuments = totalDocs
	p.ProcessedDocuments = 0
	p.ErrorMessage = ""
	p.StartTime = time.Now()
	p.EndTime = time.Time{}
}

// Update increments the processed document count
func (p *IndexBuildProgress) Update(processed int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ProcessedDocuments = processed
}

// Increment increments the processed document count by one
func (p *IndexBuildProgress) Increment() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ProcessedDocuments++
}

// Complete marks the index build as completed successfully
func (p *IndexBuildProgress) Complete() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.State = IndexStateReady
	p.EndTime = time.Now()
}

// Fail marks the index build as failed
func (p *IndexBuildProgress) Fail(err string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.State = IndexStateFailed
	p.ErrorMessage = err
	p.EndTime = time.Now()
}

// GetState returns the current build state
func (p *IndexBuildProgress) GetState() IndexBuildState {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.State
}

// GetProgress returns a snapshot of the current progress
func (p *IndexBuildProgress) GetProgress() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	progress := map[string]interface{}{
		"state":      p.State.String(),
		"total":      p.TotalDocuments,
		"processed":  p.ProcessedDocuments,
		"start_time": p.StartTime,
	}

	if !p.EndTime.IsZero() {
		progress["end_time"] = p.EndTime
		progress["duration_ms"] = p.EndTime.Sub(p.StartTime).Milliseconds()
	}

	if p.State == IndexStateBuilding && p.TotalDocuments > 0 {
		progress["percent_complete"] = float64(p.ProcessedDocuments) / float64(p.TotalDocuments) * 100.0
	}

	if p.ErrorMessage != "" {
		progress["error"] = p.ErrorMessage
	}

	return progress
}

// GetPercentComplete returns the percentage of documents processed
func (p *IndexBuildProgress) GetPercentComplete() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.TotalDocuments == 0 {
		return 100.0
	}
	return float64(p.ProcessedDocuments) / float64(p.TotalDocuments) * 100.0
}
