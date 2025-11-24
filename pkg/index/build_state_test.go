package index

import (
	"testing"
	"time"
)

func TestIndexBuildState_String(t *testing.T) {
	tests := []struct {
		state    IndexBuildState
		expected string
	}{
		{IndexStateReady, "ready"},
		{IndexStateBuilding, "building"},
		{IndexStateFailed, "failed"},
		{IndexBuildState(999), "unknown"}, // Invalid state
	}

	for _, tt := range tests {
		result := tt.state.String()
		if result != tt.expected {
			t.Errorf("Expected %s for state %v, got %s", tt.expected, tt.state, result)
		}
	}
}

func TestNewIndexBuildProgress(t *testing.T) {
	p := NewIndexBuildProgress()

	if p == nil {
		t.Fatal("NewIndexBuildProgress returned nil")
	}

	if p.GetState() != IndexStateReady {
		t.Errorf("Expected initial state to be Ready, got %v", p.GetState())
	}

	if p.TotalDocuments != 0 {
		t.Errorf("Expected TotalDocuments to be 0, got %d", p.TotalDocuments)
	}

	if p.ProcessedDocuments != 0 {
		t.Errorf("Expected ProcessedDocuments to be 0, got %d", p.ProcessedDocuments)
	}
}

func TestIndexBuildProgress_Start(t *testing.T) {
	p := NewIndexBuildProgress()

	p.Start(100)

	if p.GetState() != IndexStateBuilding {
		t.Errorf("Expected state to be Building after Start(), got %v", p.GetState())
	}

	progress := p.GetProgress()
	if progress["total"] != 100 {
		t.Errorf("Expected total to be 100, got %v", progress["total"])
	}

	if progress["processed"] != 0 {
		t.Errorf("Expected processed to be 0, got %v", progress["processed"])
	}

	if progress["state"] != "building" {
		t.Errorf("Expected state to be 'building', got %v", progress["state"])
	}
}

func TestIndexBuildProgress_Update(t *testing.T) {
	p := NewIndexBuildProgress()
	p.Start(100)

	p.Update(50)

	progress := p.GetProgress()
	if progress["processed"] != 50 {
		t.Errorf("Expected processed to be 50, got %v", progress["processed"])
	}

	// Update again
	p.Update(75)

	progress = p.GetProgress()
	if progress["processed"] != 75 {
		t.Errorf("Expected processed to be 75, got %v", progress["processed"])
	}
}

func TestIndexBuildProgress_Increment(t *testing.T) {
	p := NewIndexBuildProgress()
	p.Start(10)

	// Increment multiple times
	for i := 0; i < 5; i++ {
		p.Increment()
	}

	progress := p.GetProgress()
	if progress["processed"] != 5 {
		t.Errorf("Expected processed to be 5, got %v", progress["processed"])
	}

	// Increment more
	p.Increment()
	p.Increment()

	progress = p.GetProgress()
	if progress["processed"] != 7 {
		t.Errorf("Expected processed to be 7, got %v", progress["processed"])
	}
}

func TestIndexBuildProgress_Complete(t *testing.T) {
	p := NewIndexBuildProgress()
	p.Start(100)
	p.Update(100)

	// Small delay to ensure time difference
	time.Sleep(10 * time.Millisecond)

	p.Complete()

	if p.GetState() != IndexStateReady {
		t.Errorf("Expected state to be Ready after Complete(), got %v", p.GetState())
	}

	progress := p.GetProgress()

	// Should have end_time
	if _, ok := progress["end_time"]; !ok {
		t.Error("Expected end_time to be set after Complete()")
	}

	// Should have duration
	if _, ok := progress["duration_ms"]; !ok {
		t.Error("Expected duration_ms to be set after Complete()")
	}

	// Duration should be positive
	if duration, ok := progress["duration_ms"].(int64); ok {
		if duration <= 0 {
			t.Errorf("Expected positive duration, got %d", duration)
		}
	}
}

func TestIndexBuildProgress_Fail(t *testing.T) {
	p := NewIndexBuildProgress()
	p.Start(100)
	p.Update(50)

	errorMsg := "index build failed due to disk error"
	p.Fail(errorMsg)

	if p.GetState() != IndexStateFailed {
		t.Errorf("Expected state to be Failed after Fail(), got %v", p.GetState())
	}

	progress := p.GetProgress()

	// Should have error message
	if errMsg, ok := progress["error"].(string); !ok || errMsg != errorMsg {
		t.Errorf("Expected error message '%s', got %v", errorMsg, progress["error"])
	}

	// Should have end_time
	if _, ok := progress["end_time"]; !ok {
		t.Error("Expected end_time to be set after Fail()")
	}
}

func TestIndexBuildProgress_GetPercentComplete(t *testing.T) {
	p := NewIndexBuildProgress()

	// Before starting (0 total documents)
	percent := p.GetPercentComplete()
	if percent != 100.0 {
		t.Errorf("Expected 100%% for 0 total documents, got %.2f", percent)
	}

	// Start with 100 documents
	p.Start(100)

	// 0 processed
	percent = p.GetPercentComplete()
	if percent != 0.0 {
		t.Errorf("Expected 0%% for 0/100 processed, got %.2f", percent)
	}

	// 50 processed
	p.Update(50)
	percent = p.GetPercentComplete()
	if percent != 50.0 {
		t.Errorf("Expected 50%% for 50/100 processed, got %.2f", percent)
	}

	// 100 processed
	p.Update(100)
	percent = p.GetPercentComplete()
	if percent != 100.0 {
		t.Errorf("Expected 100%% for 100/100 processed, got %.2f", percent)
	}
}

func TestIndexBuildProgress_GetProgress_PercentComplete(t *testing.T) {
	p := NewIndexBuildProgress()
	p.Start(200)
	p.Update(100)

	progress := p.GetProgress()

	// Should have percent_complete when building
	if percent, ok := progress["percent_complete"].(float64); !ok {
		t.Error("Expected percent_complete in progress")
	} else if percent != 50.0 {
		t.Errorf("Expected percent_complete to be 50.0, got %.2f", percent)
	}

	// After completion, percent_complete should not be in GetProgress
	p.Complete()
	progress = p.GetProgress()

	if _, ok := progress["percent_complete"]; ok {
		t.Error("Expected percent_complete to not be in progress after completion")
	}
}

func TestIndexBuildProgress_Concurrent(t *testing.T) {
	p := NewIndexBuildProgress()
	p.Start(1000)

	// Concurrent increments
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				p.Increment()
			}
			done <- true
		}()
	}

	// Wait for all increments
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should have incremented 1000 times (100 goroutines * 10 increments each)
	progress := p.GetProgress()
	if progress["processed"] != 1000 {
		t.Errorf("Expected 1000 processed after concurrent increments, got %v", progress["processed"])
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		go func() {
			p.GetProgress()
			p.GetState()
			p.GetPercentComplete()
			done <- true
		}()
	}

	// Wait for all reads
	for i := 0; i < 50; i++ {
		<-done
	}
}

func TestIndexBuildProgress_RestartAfterComplete(t *testing.T) {
	p := NewIndexBuildProgress()

	// First build
	p.Start(100)
	p.Update(100)
	p.Complete()

	if p.GetState() != IndexStateReady {
		t.Error("Expected state to be Ready after first build")
	}

	// Start again
	time.Sleep(10 * time.Millisecond)
	p.Start(200)

	if p.GetState() != IndexStateBuilding {
		t.Error("Expected state to be Building after restart")
	}

	progress := p.GetProgress()
	if progress["total"] != 200 {
		t.Errorf("Expected total to be 200 after restart, got %v", progress["total"])
	}

	if progress["processed"] != 0 {
		t.Errorf("Expected processed to be 0 after restart, got %v", progress["processed"])
	}

	// Error message should be cleared
	if _, ok := progress["error"]; ok {
		t.Error("Expected error to be cleared after restart")
	}
}

func TestIndexBuildProgress_RestartAfterFail(t *testing.T) {
	p := NewIndexBuildProgress()

	// First build fails
	p.Start(100)
	p.Update(50)
	p.Fail("some error")

	if p.GetState() != IndexStateFailed {
		t.Error("Expected state to be Failed")
	}

	// Restart
	time.Sleep(10 * time.Millisecond)
	p.Start(150)

	if p.GetState() != IndexStateBuilding {
		t.Error("Expected state to be Building after restart")
	}

	progress := p.GetProgress()

	// Error message should be cleared
	if _, ok := progress["error"]; ok {
		t.Error("Expected error to be cleared after restart")
	}

	// New values
	if progress["total"] != 150 {
		t.Errorf("Expected total to be 150, got %v", progress["total"])
	}
}

func TestIndexBuildProgress_DurationCalculation(t *testing.T) {
	p := NewIndexBuildProgress()

	p.Start(10)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	p.Update(5)

	// Wait more
	time.Sleep(50 * time.Millisecond)

	p.Complete()

	progress := p.GetProgress()

	duration, ok := progress["duration_ms"].(int64)
	if !ok {
		t.Fatal("Expected duration_ms to be int64")
	}

	// Duration should be at least 100ms (we waited 50ms + 50ms)
	if duration < 100 {
		t.Errorf("Expected duration >= 100ms, got %dms", duration)
	}

	// But shouldn't be too large (give 200ms buffer for slow systems)
	if duration > 300 {
		t.Errorf("Expected duration < 300ms, got %dms", duration)
	}
}

func TestIndexBuildProgress_ZeroDocuments(t *testing.T) {
	p := NewIndexBuildProgress()

	// Start with 0 documents
	p.Start(0)

	percent := p.GetPercentComplete()
	if percent != 100.0 {
		t.Errorf("Expected 100%% for 0 total documents, got %.2f", percent)
	}

	// Complete immediately
	p.Complete()

	if p.GetState() != IndexStateReady {
		t.Error("Expected state to be Ready")
	}
}
