package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestQueryProfiler_EnableDisable(t *testing.T) {
	qp := NewQueryProfiler(true)

	if !qp.IsEnabled() {
		t.Error("Expected profiler to be enabled")
	}

	qp.Disable()

	if qp.IsEnabled() {
		t.Error("Expected profiler to be disabled")
	}

	qp.Enable()

	if !qp.IsEnabled() {
		t.Error("Expected profiler to be enabled")
	}
}

func TestQueryProfiler_StartProfile(t *testing.T) {
	qp := NewQueryProfiler(true)

	session := qp.StartProfile()
	if session == nil {
		t.Error("Expected non-nil profile session when enabled")
	}

	qp.Disable()
	session = qp.StartProfile()
	if session != nil {
		t.Error("Expected nil profile session when disabled")
	}
}

func TestProfileSession_AddMetadata(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	session.AddMetadata("collection", "users")
	session.AddMetadata("operation", "query")
	session.AddMetadata("user_id", 123)

	result := session.Finish()

	if result.Metadata["collection"] != "users" {
		t.Errorf("Expected collection 'users', got %v", result.Metadata["collection"])
	}
	if result.Metadata["operation"] != "query" {
		t.Errorf("Expected operation 'query', got %v", result.Metadata["operation"])
	}
	if result.Metadata["user_id"] != 123 {
		t.Errorf("Expected user_id 123, got %v", result.Metadata["user_id"])
	}
}

func TestProfileSession_StartEndStage(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	session.StartStage("parse")
	time.Sleep(10 * time.Millisecond)
	session.EndStage()

	session.StartStage("execute")
	time.Sleep(20 * time.Millisecond)
	session.EndStage()

	result := session.Finish()

	if len(result.Stages) != 2 {
		t.Errorf("Expected 2 stages, got %d", len(result.Stages))
	}

	if result.Stages[0].Name != "parse" {
		t.Errorf("Expected first stage 'parse', got '%s'", result.Stages[0].Name)
	}

	if result.Stages[1].Name != "execute" {
		t.Errorf("Expected second stage 'execute', got '%s'", result.Stages[1].Name)
	}

	// Check durations (should be >= sleep times)
	if result.Stages[0].Duration < 10*time.Millisecond {
		t.Errorf("Expected parse stage >= 10ms, got %v", result.Stages[0].Duration)
	}

	if result.Stages[1].Duration < 20*time.Millisecond {
		t.Errorf("Expected execute stage >= 20ms, got %v", result.Stages[1].Duration)
	}
}

func TestProfileSession_AutoEndPreviousStage(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	session.StartStage("stage1")
	time.Sleep(10 * time.Millisecond)

	// Starting a new stage should automatically end the previous one
	session.StartStage("stage2")
	time.Sleep(10 * time.Millisecond)

	result := session.Finish()

	if len(result.Stages) != 2 {
		t.Errorf("Expected 2 stages, got %d", len(result.Stages))
	}

	// First stage should have been auto-ended
	if result.Stages[0].EndTime.IsZero() {
		t.Error("Expected first stage to have end time")
	}

	if result.Stages[0].Duration == 0 {
		t.Error("Expected first stage to have non-zero duration")
	}
}

func TestProfileSession_AddStageDetail(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	session.StartStage("query")
	session.AddStageDetail("docs_examined", 1000)
	session.AddStageDetail("docs_returned", 50)
	session.AddStageDetail("index_used", "age_idx")
	session.EndStage()

	result := session.Finish()

	if len(result.Stages) != 1 {
		t.Errorf("Expected 1 stage, got %d", len(result.Stages))
	}

	details := result.Stages[0].Details
	if details["docs_examined"] != 1000 {
		t.Errorf("Expected docs_examined 1000, got %v", details["docs_examined"])
	}
	if details["docs_returned"] != 50 {
		t.Errorf("Expected docs_returned 50, got %v", details["docs_returned"])
	}
	if details["index_used"] != "age_idx" {
		t.Errorf("Expected index_used 'age_idx', got %v", details["index_used"])
	}
}

func TestProfileSession_RecordStage(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	details := map[string]interface{}{
		"count": 100,
		"type":  "scan",
	}

	session.RecordStage("collection_scan", 50*time.Millisecond, details)

	result := session.Finish()

	if len(result.Stages) != 1 {
		t.Errorf("Expected 1 stage, got %d", len(result.Stages))
	}

	stage := result.Stages[0]
	if stage.Name != "collection_scan" {
		t.Errorf("Expected stage 'collection_scan', got '%s'", stage.Name)
	}

	if stage.Duration != 50*time.Millisecond {
		t.Errorf("Expected duration 50ms, got %v", stage.Duration)
	}

	if stage.Details["count"] != 100 {
		t.Errorf("Expected count 100, got %v", stage.Details["count"])
	}
}

func TestProfileSession_Finish(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	time.Sleep(50 * time.Millisecond)

	session.StartStage("stage1")
	time.Sleep(10 * time.Millisecond)
	session.EndStage()

	result := session.Finish()

	if result.TotalDuration < 50*time.Millisecond {
		t.Errorf("Expected total duration >= 50ms, got %v", result.TotalDuration)
	}

	if result.TotalDurationMS < 50.0 {
		t.Errorf("Expected total duration >= 50ms, got %.2fms", result.TotalDurationMS)
	}

	if result.StartTime.IsZero() {
		t.Error("Expected non-zero start time")
	}

	if result.EndTime.IsZero() {
		t.Error("Expected non-zero end time")
	}

	if !result.EndTime.After(result.StartTime) {
		t.Error("Expected end time to be after start time")
	}
}

func TestProfileSession_FinishEndsCurrentStage(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	session.StartStage("unclosed_stage")
	time.Sleep(10 * time.Millisecond)

	// Finish without explicitly ending the stage
	result := session.Finish()

	if len(result.Stages) != 1 {
		t.Errorf("Expected 1 stage, got %d", len(result.Stages))
	}

	// The stage should have been auto-ended
	if result.Stages[0].EndTime.IsZero() {
		t.Error("Expected stage to have end time")
	}

	if result.Stages[0].Duration == 0 {
		t.Error("Expected stage to have non-zero duration")
	}
}

func TestProfileResult_GetSummary(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	session.AddMetadata("collection", "users")
	session.StartStage("parse")
	time.Sleep(10 * time.Millisecond)
	session.EndStage()

	session.StartStage("execute")
	time.Sleep(20 * time.Millisecond)
	session.EndStage()

	result := session.Finish()
	summary := result.GetSummary()

	if !strings.Contains(summary, "Total Duration:") {
		t.Error("Expected summary to contain 'Total Duration:'")
	}

	if !strings.Contains(summary, "parse") {
		t.Error("Expected summary to contain 'parse' stage")
	}

	if !strings.Contains(summary, "execute") {
		t.Error("Expected summary to contain 'execute' stage")
	}

	if !strings.Contains(summary, "users") {
		t.Error("Expected summary to contain metadata 'users'")
	}
}

func TestProfileResult_GetSlowStages(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	session.RecordStage("fast", 5*time.Millisecond, nil)
	session.RecordStage("slow1", 50*time.Millisecond, nil)
	session.RecordStage("slow2", 100*time.Millisecond, nil)
	session.RecordStage("fast2", 10*time.Millisecond, nil)

	result := session.Finish()
	slowStages := result.GetSlowStages(20 * time.Millisecond)

	if len(slowStages) != 2 {
		t.Errorf("Expected 2 slow stages, got %d", len(slowStages))
	}

	if slowStages[0].Name != "slow1" {
		t.Errorf("Expected first slow stage 'slow1', got '%s'", slowStages[0].Name)
	}

	if slowStages[1].Name != "slow2" {
		t.Errorf("Expected second slow stage 'slow2', got '%s'", slowStages[1].Name)
	}
}

func TestProfileResult_GetStagePercentages(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	// Use actual stages with timing
	session.StartStage("stage1")
	time.Sleep(10 * time.Millisecond)
	session.EndStage()

	session.StartStage("stage2")
	time.Sleep(30 * time.Millisecond)
	session.EndStage()

	result := session.Finish()
	percentages := result.GetStagePercentages()

	// Stage1 should be ~25% (10ms of 40ms), stage2 ~75% (30ms of 40ms)
	// Allow generous variance due to timing imprecision
	if percentages["stage1"] < 15.0 || percentages["stage1"] > 35.0 {
		t.Errorf("Expected stage1 15-35%%, got %.2f%%", percentages["stage1"])
	}

	if percentages["stage2"] < 65.0 || percentages["stage2"] > 85.0 {
		t.Errorf("Expected stage2 65-85%%, got %.2f%%", percentages["stage2"])
	}
}

func TestProfileResult_GetBottleneck(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	session.RecordStage("stage1", 10*time.Millisecond, nil)
	session.RecordStage("bottleneck", 100*time.Millisecond, nil)
	session.RecordStage("stage3", 20*time.Millisecond, nil)

	result := session.Finish()
	bottleneck := result.GetBottleneck()

	if bottleneck == nil {
		t.Fatal("Expected non-nil bottleneck")
	}

	if bottleneck.Name != "bottleneck" {
		t.Errorf("Expected bottleneck 'bottleneck', got '%s'", bottleneck.Name)
	}

	if bottleneck.Duration != 100*time.Millisecond {
		t.Errorf("Expected bottleneck duration 100ms, got %v", bottleneck.Duration)
	}
}

func TestProfileResult_GetBottleneckEmpty(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	result := session.Finish()
	bottleneck := result.GetBottleneck()

	if bottleneck != nil {
		t.Error("Expected nil bottleneck for empty stages")
	}
}

func TestProfilerHelper_ProfileQuery(t *testing.T) {
	qp := NewQueryProfiler(true)
	helper := NewProfilerHelper(qp)

	result, err := helper.ProfileQuery("users", "find", func(session *ProfileSession) error {
		if session != nil {
			session.StartStage("parse")
			time.Sleep(10 * time.Millisecond)
			session.EndStage()

			session.StartStage("execute")
			time.Sleep(20 * time.Millisecond)
			session.EndStage()
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Metadata["collection"] != "users" {
		t.Errorf("Expected collection 'users', got %v", result.Metadata["collection"])
	}

	if result.Metadata["operation"] != "find" {
		t.Errorf("Expected operation 'find', got %v", result.Metadata["operation"])
	}

	if len(result.Stages) != 2 {
		t.Errorf("Expected 2 stages, got %d", len(result.Stages))
	}
}

func TestTimeStage(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	func() {
		defer TimeStage(session, "test_stage")()
		time.Sleep(10 * time.Millisecond)
	}()

	result := session.Finish()

	if len(result.Stages) != 1 {
		t.Errorf("Expected 1 stage, got %d", len(result.Stages))
	}

	if result.Stages[0].Name != "test_stage" {
		t.Errorf("Expected stage 'test_stage', got '%s'", result.Stages[0].Name)
	}

	if result.Stages[0].Duration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", result.Stages[0].Duration)
	}
}

func TestTimeStageNilSession(t *testing.T) {
	// Should not panic with nil session
	func() {
		defer TimeStage(nil, "test")()
		time.Sleep(10 * time.Millisecond)
	}()
}

func TestProfileSession_NilOperations(t *testing.T) {
	// All operations should be safe with nil session
	var session *ProfileSession

	session.AddMetadata("key", "value")
	session.StartStage("test")
	session.EndStage()
	session.AddStageDetail("key", "value")
	session.RecordStage("test", time.Second, nil)
	result := session.Finish()

	if result != nil {
		t.Error("Expected nil result for nil session")
	}
}

func TestProfileResult_NilOperations(t *testing.T) {
	// All operations should be safe with nil result
	var result *ProfileResult

	summary := result.GetSummary()
	if summary != "No profile data" {
		t.Errorf("Expected 'No profile data', got '%s'", summary)
	}

	slowStages := result.GetSlowStages(time.Second)
	if slowStages != nil {
		t.Error("Expected nil slow stages for nil result")
	}

	percentages := result.GetStagePercentages()
	if percentages != nil {
		t.Error("Expected nil percentages for nil result")
	}

	bottleneck := result.GetBottleneck()
	if bottleneck != nil {
		t.Error("Expected nil bottleneck for nil result")
	}
}

func TestProfileSession_MultipleStagesWithDetails(t *testing.T) {
	qp := NewQueryProfiler(true)
	session := qp.StartProfile()

	session.AddMetadata("query", "find all users")

	session.StartStage("parse_query")
	session.AddStageDetail("complexity", "simple")
	time.Sleep(5 * time.Millisecond)
	session.EndStage()

	session.StartStage("optimize")
	session.AddStageDetail("indexes_available", 3)
	session.AddStageDetail("index_selected", "age_idx")
	time.Sleep(3 * time.Millisecond)
	session.EndStage()

	session.StartStage("execute")
	session.AddStageDetail("docs_scanned", 1000)
	session.AddStageDetail("docs_returned", 50)
	time.Sleep(20 * time.Millisecond)
	session.EndStage()

	result := session.Finish()

	if len(result.Stages) != 3 {
		t.Errorf("Expected 3 stages, got %d", len(result.Stages))
	}

	// Check parse stage
	if result.Stages[0].Details["complexity"] != "simple" {
		t.Error("Expected complexity 'simple'")
	}

	// Check optimize stage
	if result.Stages[1].Details["indexes_available"] != 3 {
		t.Error("Expected 3 indexes available")
	}

	// Check execute stage
	if result.Stages[2].Details["docs_scanned"] != 1000 {
		t.Error("Expected 1000 docs scanned")
	}
}
