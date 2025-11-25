package chaos

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Scenario represents a chaos engineering scenario
type Scenario struct {
	Name        string
	Description string
	Duration    time.Duration
	Steps       []ScenarioStep
	Assertions  []Assertion
}

// ScenarioStep represents a single step in a chaos scenario
type ScenarioStep struct {
	Name        string
	Delay       time.Duration        // Wait before executing this step
	Action      func(ctx context.Context) error
	FaultConfig *FaultConfig         // Optional fault to inject during this step
	Duration    time.Duration        // How long to maintain fault
}

// Assertion represents a post-scenario assertion
type Assertion struct {
	Name     string
	Check    func() error
	Critical bool // If true, scenario fails if assertion fails
}

// ScenarioResult contains the results of running a scenario
type ScenarioResult struct {
	Scenario       *Scenario
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	Success        bool
	StepResults    []StepResult
	AssertionResults []AssertionResult
	Errors         []error
	Events         []ChaosEvent
}

// StepResult contains the result of a single step
type StepResult struct {
	Step      *ScenarioStep
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Error     error
	Success   bool
}

// AssertionResult contains the result of an assertion
type AssertionResult struct {
	Assertion *Assertion
	Error     error
	Success   bool
}

// ScenarioRunner executes chaos scenarios
type ScenarioRunner struct {
	injector *FaultInjector
	mu       sync.Mutex
	results  []*ScenarioResult
}

// NewScenarioRunner creates a new scenario runner
func NewScenarioRunner(injector *FaultInjector) *ScenarioRunner {
	return &ScenarioRunner{
		injector: injector,
		results:  make([]*ScenarioResult, 0),
	}
}

// Run executes a scenario
func (sr *ScenarioRunner) Run(ctx context.Context, scenario *Scenario) (*ScenarioResult, error) {
	result := &ScenarioResult{
		Scenario:    scenario,
		StartTime:   time.Now(),
		Success:     true,
		StepResults: make([]StepResult, 0, len(scenario.Steps)),
		AssertionResults: make([]AssertionResult, 0, len(scenario.Assertions)),
		Errors:      make([]error, 0),
		Events:      make([]ChaosEvent, 0),
	}

	// Set up event collection
	eventsChan := make(chan ChaosEvent, 100)
	sr.injector.AddEventCallback(func(event ChaosEvent) {
		select {
		case eventsChan <- event:
		default:
			// Channel full, skip event
		}
	})

	// Collect events in background
	eventsDone := make(chan struct{})
	go func() {
		defer close(eventsDone)
		for {
			select {
			case event := <-eventsChan:
				result.Events = append(result.Events, event)
			case <-ctx.Done():
				// Drain remaining events
				for len(eventsChan) > 0 {
					result.Events = append(result.Events, <-eventsChan)
				}
				return
			}
		}
	}()

	// Execute scenario steps
	for _, step := range scenario.Steps {
		// Check context cancellation
		select {
		case <-ctx.Done():
			result.Success = false
			result.Errors = append(result.Errors, ctx.Err())
			goto finish
		default:
		}

		// Wait for step delay
		if step.Delay > 0 {
			time.Sleep(step.Delay)
		}

		// Execute step
		stepResult := sr.executeStep(ctx, &step)
		result.StepResults = append(result.StepResults, stepResult)

		if !stepResult.Success {
			result.Success = false
			result.Errors = append(result.Errors, stepResult.Error)
		}
	}

	// Run assertions
	for _, assertion := range scenario.Assertions {
		assertionResult := AssertionResult{
			Assertion: &assertion,
			Success:   true,
		}

		if err := assertion.Check(); err != nil {
			assertionResult.Success = false
			assertionResult.Error = err

			if assertion.Critical {
				result.Success = false
				result.Errors = append(result.Errors, fmt.Errorf("critical assertion failed: %s: %w", assertion.Name, err))
			}
		}

		result.AssertionResults = append(result.AssertionResults, assertionResult)
	}

finish:
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Stop event collection
	<-eventsDone

	// Store result
	sr.mu.Lock()
	sr.results = append(sr.results, result)
	sr.mu.Unlock()

	if !result.Success {
		return result, fmt.Errorf("scenario failed: %s", scenario.Name)
	}

	return result, nil
}

// executeStep executes a single scenario step
func (sr *ScenarioRunner) executeStep(ctx context.Context, step *ScenarioStep) StepResult {
	result := StepResult{
		Step:      step,
		StartTime: time.Now(),
		Success:   true,
	}

	// Configure fault if specified
	if step.FaultConfig != nil {
		sr.injector.ConfigureFault(step.FaultConfig)
		defer sr.injector.DisableFault(step.FaultConfig.Type)
	}

	// Execute action with timeout
	actionCtx := ctx
	if step.Duration > 0 {
		var cancel context.CancelFunc
		actionCtx, cancel = context.WithTimeout(ctx, step.Duration)
		defer cancel()
	}

	// Run action
	if step.Action != nil {
		if err := step.Action(actionCtx); err != nil {
			result.Success = false
			result.Error = err
		}
	}

	// Maintain fault for duration
	if step.Duration > 0 && step.FaultConfig != nil {
		select {
		case <-actionCtx.Done():
		case <-time.After(step.Duration):
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// GetResults returns all scenario results
func (sr *ScenarioRunner) GetResults() []*ScenarioResult {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	return sr.results
}

// PrintResult prints a scenario result
func (sr *ScenarioRunner) PrintResult(result *ScenarioResult) string {
	output := fmt.Sprintf("Scenario: %s\n", result.Scenario.Name)
	output += fmt.Sprintf("  Duration: %v\n", result.Duration)
	output += fmt.Sprintf("  Success: %v\n", result.Success)
	output += fmt.Sprintf("  Steps: %d/%d passed\n", sr.countSuccessfulSteps(result), len(result.StepResults))
	output += fmt.Sprintf("  Assertions: %d/%d passed\n", sr.countSuccessfulAssertions(result), len(result.AssertionResults))

	if len(result.Errors) > 0 {
		output += "  Errors:\n"
		for _, err := range result.Errors {
			output += fmt.Sprintf("    - %v\n", err)
		}
	}

	output += fmt.Sprintf("  Events: %d chaos events recorded\n", len(result.Events))

	return output
}

func (sr *ScenarioRunner) countSuccessfulSteps(result *ScenarioResult) int {
	count := 0
	for _, stepResult := range result.StepResults {
		if stepResult.Success {
			count++
		}
	}
	return count
}

func (sr *ScenarioRunner) countSuccessfulAssertions(result *ScenarioResult) int {
	count := 0
	for _, assertionResult := range result.AssertionResults {
		if assertionResult.Success {
			count++
		}
	}
	return count
}

// Predefined scenario builders

// DiskFailureScenario creates a disk failure scenario
func DiskFailureScenario(name string, actions ...func(ctx context.Context) error) *Scenario {
	steps := make([]ScenarioStep, len(actions))
	for i, action := range actions {
		steps[i] = ScenarioStep{
			Name:   fmt.Sprintf("Action %d", i+1),
			Action: action,
			FaultConfig: &FaultConfig{
				Type:        FaultTypeDiskWrite,
				Enabled:     true,
				Probability: 0.3, // 30% chance of failure
				ErrorMsg:    "disk write failure injected",
			},
		}
	}

	return &Scenario{
		Name:        name,
		Description: "Simulates random disk write failures",
		Steps:       steps,
	}
}

// SlowDiskScenario creates a slow disk I/O scenario
func SlowDiskScenario(name string, delay time.Duration, actions ...func(ctx context.Context) error) *Scenario {
	steps := make([]ScenarioStep, len(actions))
	for i, action := range actions {
		steps[i] = ScenarioStep{
			Name:   fmt.Sprintf("Action %d", i+1),
			Action: action,
			FaultConfig: &FaultConfig{
				Type:        FaultTypeSlowIO,
				Enabled:     true,
				Probability: 1.0, // Always inject delay
				Delay:       delay,
				ErrorMsg:    "slow I/O injected",
			},
		}
	}

	return &Scenario{
		Name:        name,
		Description: fmt.Sprintf("Simulates slow disk I/O with %v delay", delay),
		Steps:       steps,
	}
}

// ProcessCrashScenario creates a process crash simulation scenario
func ProcessCrashScenario(name string, crashAction func(ctx context.Context) error, recoveryActions ...func(ctx context.Context) error) *Scenario {
	steps := []ScenarioStep{
		{
			Name:   "Normal operation before crash",
			Action: crashAction,
		},
		{
			Name:  "Simulate crash",
			Delay: 100 * time.Millisecond,
			FaultConfig: &FaultConfig{
				Type:        FaultTypeProcessCrash,
				Enabled:     true,
				Probability: 1.0,
				ErrorMsg:    "process crash injected",
			},
		},
	}

	// Add recovery steps
	for i, action := range recoveryActions {
		steps = append(steps, ScenarioStep{
			Name:   fmt.Sprintf("Recovery action %d", i+1),
			Delay:  100 * time.Millisecond,
			Action: action,
		})
	}

	return &Scenario{
		Name:        name,
		Description: "Simulates process crash and recovery",
		Steps:       steps,
	}
}

// NetworkPartitionScenario creates a network partition scenario
func NetworkPartitionScenario(name string, duration time.Duration, actions ...func(ctx context.Context) error) *Scenario {
	steps := []ScenarioStep{
		{
			Name: "Enable network partition",
			FaultConfig: &FaultConfig{
				Type:        FaultTypeNetworkPartition,
				Enabled:     true,
				Probability: 1.0,
				Duration:    duration,
				ErrorMsg:    "network partition active",
			},
			Duration: duration,
		},
	}

	// Add actions during partition
	for i, action := range actions {
		steps = append(steps, ScenarioStep{
			Name:   fmt.Sprintf("Action during partition %d", i+1),
			Action: action,
		})
	}

	// Recovery step
	steps = append(steps, ScenarioStep{
		Name:  "Partition healed",
		Delay: 100 * time.Millisecond,
	})

	return &Scenario{
		Name:        name,
		Description: fmt.Sprintf("Simulates network partition for %v", duration),
		Steps:       steps,
	}
}
