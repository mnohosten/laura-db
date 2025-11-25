package chaos

import (
	"testing"
	"time"
)

func TestFaultInjector_EnableDisable(t *testing.T) {
	injector := NewFaultInjector(42)

	if injector.IsEnabled() {
		t.Error("Injector should be disabled by default")
	}

	injector.Enable()
	if !injector.IsEnabled() {
		t.Error("Injector should be enabled")
	}

	injector.Disable()
	if injector.IsEnabled() {
		t.Error("Injector should be disabled")
	}
}

func TestFaultInjector_ConfigureFault(t *testing.T) {
	injector := NewFaultInjector(42)
	injector.Enable()

	config := &FaultConfig{
		Type:        FaultTypeDiskWrite,
		Enabled:     true,
		Probability: 1.0,
		ErrorMsg:    "test error",
	}

	injector.ConfigureFault(config)

	// Should always trigger with 100% probability
	shouldInject, err := injector.ShouldInjectFault(FaultTypeDiskWrite)
	if !shouldInject {
		t.Error("Fault should be injected with 100% probability")
	}
	if err == nil {
		t.Error("Expected error from fault injection")
	}
	if err.Error() != "test error" {
		t.Errorf("Expected error message 'test error', got '%s'", err.Error())
	}
}

func TestFaultInjector_Probability(t *testing.T) {
	injector := NewFaultInjector(42)
	injector.Enable()

	// Test 0% probability
	injector.EnableFault(FaultTypeDiskRead, 0.0)

	triggered := false
	for i := 0; i < 100; i++ {
		if shouldInject, _ := injector.ShouldInjectFault(FaultTypeDiskRead); shouldInject {
			triggered = true
			break
		}
	}

	if triggered {
		t.Error("Fault should never trigger with 0% probability")
	}

	// Test 100% probability
	injector.EnableFault(FaultTypeDiskWrite, 1.0)

	allTriggered := true
	for i := 0; i < 10; i++ {
		if shouldInject, _ := injector.ShouldInjectFault(FaultTypeDiskWrite); !shouldInject {
			allTriggered = false
			break
		}
	}

	if !allTriggered {
		t.Error("Fault should always trigger with 100% probability")
	}
}

func TestFaultInjector_TriggerCount(t *testing.T) {
	injector := NewFaultInjector(42)
	injector.Enable()
	injector.EnableFault(FaultTypeDiskWrite, 1.0)

	initialCount := injector.GetTriggerCount(FaultTypeDiskWrite)
	if initialCount != 0 {
		t.Errorf("Expected initial count of 0, got %d", initialCount)
	}

	for i := 0; i < 10; i++ {
		injector.ShouldInjectFault(FaultTypeDiskWrite)
	}

	count := injector.GetTriggerCount(FaultTypeDiskWrite)
	if count != 10 {
		t.Errorf("Expected trigger count of 10, got %d", count)
	}

	injector.Reset()
	if injector.GetTriggerCount(FaultTypeDiskWrite) != 0 {
		t.Error("Trigger count should be reset to 0")
	}
}

func TestFaultInjector_FaultDelay(t *testing.T) {
	injector := NewFaultInjector(42)
	injector.Enable()

	delay := 50 * time.Millisecond
	injector.ConfigureFault(&FaultConfig{
		Type:        FaultTypeSlowIO,
		Enabled:     true,
		Probability: 1.0,
		Delay:       delay,
		ErrorMsg:    "slow io",
	})

	start := time.Now()
	injector.ShouldInjectFault(FaultTypeSlowIO)
	elapsed := time.Since(start)

	if elapsed < delay {
		t.Errorf("Expected at least %v delay, got %v", delay, elapsed)
	}
}

func TestFaultInjector_EventCallbacks(t *testing.T) {
	injector := NewFaultInjector(42)

	eventReceived := false
	var receivedEvent ChaosEvent

	injector.AddEventCallback(func(event ChaosEvent) {
		eventReceived = true
		receivedEvent = event
	})

	injector.Enable()
	injector.EnableFault(FaultTypeDiskWrite, 1.0)
	injector.ShouldInjectFault(FaultTypeDiskWrite)

	// Give callback time to execute
	time.Sleep(10 * time.Millisecond)

	if !eventReceived {
		t.Error("Event callback was not called")
	}

	if receivedEvent.Type != FaultTypeDiskWrite {
		t.Errorf("Expected event type %v, got %v", FaultTypeDiskWrite, receivedEvent.Type)
	}

	if !receivedEvent.Triggered {
		t.Error("Event should be marked as triggered")
	}
}

func TestFaultTypes_String(t *testing.T) {
	tests := []struct {
		faultType FaultType
		expected  string
	}{
		{FaultTypeNone, "None"},
		{FaultTypeDiskRead, "DiskRead"},
		{FaultTypeDiskWrite, "DiskWrite"},
		{FaultTypeDiskFull, "DiskFull"},
		{FaultTypeDiskCorruption, "DiskCorruption"},
		{FaultTypeNetworkPartition, "NetworkPartition"},
		{FaultTypeProcessCrash, "ProcessCrash"},
		{FaultTypeSlowIO, "SlowIO"},
		{FaultTypeMemoryPressure, "MemoryPressure"},
	}

	for _, tt := range tests {
		if got := tt.faultType.String(); got != tt.expected {
			t.Errorf("FaultType(%d).String() = %s, expected %s", tt.faultType, got, tt.expected)
		}
	}
}
