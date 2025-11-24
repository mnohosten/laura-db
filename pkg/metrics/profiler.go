package metrics

import (
	"fmt"
	"sync"
	"time"
)

// QueryProfiler profiles query execution with detailed timing breakdown
type QueryProfiler struct {
	enabled bool
	mu      sync.RWMutex
}

// ProfileSession represents a single profiling session for a query
type ProfileSession struct {
	startTime      time.Time
	stages         []ProfileStage
	currentStage   *ProfileStage
	metadata       map[string]interface{}
	mu             sync.Mutex
}

// ProfileStage represents a single stage in query execution
type ProfileStage struct {
	Name      string                 `json:"name"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Duration  time.Duration          `json:"duration_ns"`
	DurationMS float64               `json:"duration_ms"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// ProfileResult contains the complete profile of a query execution
type ProfileResult struct {
	TotalDuration  time.Duration          `json:"total_duration_ns"`
	TotalDurationMS float64               `json:"total_duration_ms"`
	Stages         []ProfileStage         `json:"stages"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
}

// NewQueryProfiler creates a new query profiler
func NewQueryProfiler(enabled bool) *QueryProfiler {
	return &QueryProfiler{
		enabled: enabled,
	}
}

// Enable enables profiling
func (qp *QueryProfiler) Enable() {
	qp.mu.Lock()
	defer qp.mu.Unlock()
	qp.enabled = true
}

// Disable disables profiling
func (qp *QueryProfiler) Disable() {
	qp.mu.Lock()
	defer qp.mu.Unlock()
	qp.enabled = false
}

// IsEnabled returns whether profiling is enabled
func (qp *QueryProfiler) IsEnabled() bool {
	qp.mu.RLock()
	defer qp.mu.RUnlock()
	return qp.enabled
}

// StartProfile starts a new profiling session
func (qp *QueryProfiler) StartProfile() *ProfileSession {
	if !qp.IsEnabled() {
		return nil
	}

	return &ProfileSession{
		startTime: time.Now(),
		stages:    make([]ProfileStage, 0, 10),
		metadata:  make(map[string]interface{}),
	}
}

// AddMetadata adds metadata to the profile session
func (ps *ProfileSession) AddMetadata(key string, value interface{}) {
	if ps == nil {
		return
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.metadata[key] = value
}

// StartStage starts a new profiling stage
func (ps *ProfileSession) StartStage(name string) {
	if ps == nil {
		return
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	// End previous stage if it's still running
	if ps.currentStage != nil && ps.currentStage.EndTime.IsZero() {
		ps.currentStage.EndTime = time.Now()
		ps.currentStage.Duration = ps.currentStage.EndTime.Sub(ps.currentStage.StartTime)
		ps.currentStage.DurationMS = float64(ps.currentStage.Duration.Nanoseconds()) / 1e6

		// Update the stage in the stages slice
		if len(ps.stages) > 0 {
			ps.stages[len(ps.stages)-1] = *ps.currentStage
		}
	}

	// Start new stage
	stage := ProfileStage{
		Name:      name,
		StartTime: time.Now(),
		Details:   make(map[string]interface{}),
	}
	ps.stages = append(ps.stages, stage)
	// Point to the stage in the slice
	ps.currentStage = &ps.stages[len(ps.stages)-1]
}

// EndStage ends the current profiling stage
func (ps *ProfileSession) EndStage() {
	if ps == nil {
		return
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.currentStage != nil && ps.currentStage.EndTime.IsZero() {
		ps.currentStage.EndTime = time.Now()
		ps.currentStage.Duration = ps.currentStage.EndTime.Sub(ps.currentStage.StartTime)
		ps.currentStage.DurationMS = float64(ps.currentStage.Duration.Nanoseconds()) / 1e6

		// Update the stage in the stages slice
		if len(ps.stages) > 0 {
			ps.stages[len(ps.stages)-1] = *ps.currentStage
		}
	}
}

// AddStageDetail adds a detail to the current stage
func (ps *ProfileSession) AddStageDetail(key string, value interface{}) {
	if ps == nil {
		return
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.currentStage != nil {
		ps.currentStage.Details[key] = value
	}
}

// RecordStage is a convenience method to record a complete stage
func (ps *ProfileSession) RecordStage(name string, duration time.Duration, details map[string]interface{}) {
	if ps == nil {
		return
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	stage := ProfileStage{
		Name:       name,
		StartTime:  time.Now().Add(-duration), // Approximate start time
		EndTime:    time.Now(),
		Duration:   duration,
		DurationMS: float64(duration.Nanoseconds()) / 1e6,
		Details:    details,
	}

	ps.stages = append(ps.stages, stage)
}

// Finish completes the profiling session and returns the result
func (ps *ProfileSession) Finish() *ProfileResult {
	if ps == nil {
		return nil
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	// End current stage if still running
	if ps.currentStage != nil && ps.currentStage.EndTime.IsZero() {
		ps.currentStage.EndTime = time.Now()
		ps.currentStage.Duration = ps.currentStage.EndTime.Sub(ps.currentStage.StartTime)
		ps.currentStage.DurationMS = float64(ps.currentStage.Duration.Nanoseconds()) / 1e6

		// Update the stage in the stages slice
		if len(ps.stages) > 0 {
			ps.stages[len(ps.stages)-1] = *ps.currentStage
		}
	}

	endTime := time.Now()
	totalDuration := endTime.Sub(ps.startTime)

	return &ProfileResult{
		TotalDuration:   totalDuration,
		TotalDurationMS: float64(totalDuration.Nanoseconds()) / 1e6,
		Stages:          ps.stages,
		Metadata:        ps.metadata,
		StartTime:       ps.startTime,
		EndTime:         endTime,
	}
}

// GetSummary returns a human-readable summary of the profile
func (pr *ProfileResult) GetSummary() string {
	if pr == nil {
		return "No profile data"
	}

	summary := fmt.Sprintf("Total Duration: %.2fms\n", pr.TotalDurationMS)
	summary += fmt.Sprintf("Start Time: %s\n", pr.StartTime.Format(time.RFC3339Nano))
	summary += fmt.Sprintf("End Time: %s\n\n", pr.EndTime.Format(time.RFC3339Nano))
	summary += "Stages:\n"

	for i, stage := range pr.Stages {
		percentage := (float64(stage.Duration.Nanoseconds()) / float64(pr.TotalDuration.Nanoseconds())) * 100
		summary += fmt.Sprintf("  %d. %s: %.2fms (%.1f%%)\n", i+1, stage.Name, stage.DurationMS, percentage)

		if len(stage.Details) > 0 {
			for key, value := range stage.Details {
				summary += fmt.Sprintf("     - %s: %v\n", key, value)
			}
		}
	}

	if len(pr.Metadata) > 0 {
		summary += "\nMetadata:\n"
		for key, value := range pr.Metadata {
			summary += fmt.Sprintf("  - %s: %v\n", key, value)
		}
	}

	return summary
}

// GetSlowStages returns stages that took longer than the threshold
func (pr *ProfileResult) GetSlowStages(threshold time.Duration) []ProfileStage {
	if pr == nil {
		return nil
	}

	var slowStages []ProfileStage
	for _, stage := range pr.Stages {
		if stage.Duration >= threshold {
			slowStages = append(slowStages, stage)
		}
	}
	return slowStages
}

// GetStagePercentages returns the percentage of total time for each stage
func (pr *ProfileResult) GetStagePercentages() map[string]float64 {
	if pr == nil {
		return nil
	}

	percentages := make(map[string]float64)
	totalNs := float64(pr.TotalDuration.Nanoseconds())

	for _, stage := range pr.Stages {
		stageNs := float64(stage.Duration.Nanoseconds())
		percentages[stage.Name] = (stageNs / totalNs) * 100
	}

	return percentages
}

// GetBottleneck returns the slowest stage
func (pr *ProfileResult) GetBottleneck() *ProfileStage {
	if pr == nil || len(pr.Stages) == 0 {
		return nil
	}

	var bottleneck *ProfileStage
	var maxDuration time.Duration

	for i := range pr.Stages {
		if pr.Stages[i].Duration > maxDuration {
			maxDuration = pr.Stages[i].Duration
			bottleneck = &pr.Stages[i]
		}
	}

	return bottleneck
}

// ProfilerHelper provides convenient methods for common profiling scenarios
type ProfilerHelper struct {
	profiler *QueryProfiler
}

// NewProfilerHelper creates a new profiler helper
func NewProfilerHelper(profiler *QueryProfiler) *ProfilerHelper {
	return &ProfilerHelper{
		profiler: profiler,
	}
}

// ProfileQuery profiles a complete query execution
func (ph *ProfilerHelper) ProfileQuery(collection string, operation string, fn func(*ProfileSession) error) (*ProfileResult, error) {
	session := ph.profiler.StartProfile()
	if session != nil {
		session.AddMetadata("collection", collection)
		session.AddMetadata("operation", operation)
		defer session.EndStage()
	}

	err := fn(session)

	var result *ProfileResult
	if session != nil {
		result = session.Finish()
	}

	return result, err
}

// TimeStage is a helper to time a single stage with defer
func TimeStage(session *ProfileSession, name string) func() {
	if session == nil {
		return func() {}
	}

	session.StartStage(name)
	return func() {
		session.EndStage()
	}
}

// Example usage:
// defer TimeStage(session, "parse_query")()
