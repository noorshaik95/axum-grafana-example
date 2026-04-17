package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_InitialStateClosed(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:         "test",
		MaxFailures:  3,
		Timeout:      1 * time.Second,
		ResetTimeout: 5 * time.Second,
	})

	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state Closed, got %v", cb.GetStateName())
	}
}

func TestCircuitBreaker_SuccessfulExecution(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:         "test",
		MaxFailures:  3,
		Timeout:      1 * time.Second,
		ResetTimeout: 5 * time.Second,
	})

	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state Closed after success, got %v", cb.GetStateName())
	}
}

func TestCircuitBreaker_OpensAfterMaxFailures(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:         "test",
		MaxFailures:  3,
		Timeout:      1 * time.Second,
		ResetTimeout: 5 * time.Second,
	})

	testError := errors.New("test error")

	// Execute 3 failing operations
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return testError
		})
	}

	// Circuit should be open now
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state Open after %d failures, got %v", 3, cb.GetStateName())
	}

	// Next execution should fail immediately
	err := cb.Execute(func() error {
		t.Error("Function should not execute when circuit is open")
		return nil
	})

	if err != ErrCircuitOpen {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:         "test",
		MaxFailures:  2,
		Timeout:      1 * time.Second,
		ResetTimeout: 1 * time.Second, // Short timeout for test
	})

	testError := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testError
		})
	}

	if cb.GetState() != StateOpen {
		t.Error("Circuit should be open")
	}

	// Wait for reset timeout
	time.Sleep(1100 * time.Millisecond)

	// Next attempt should be allowed (half-open state)
	executed := false
	_ = cb.Execute(func() error {
		executed = true
		return nil // Success
	})

	if !executed {
		t.Error("Function should execute in half-open state")
	}
}

func TestCircuitBreaker_ClosesAfterSuccessInHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:         "test",
		MaxFailures:  2,
		Timeout:      1 * time.Second,
		ResetTimeout: 1 * time.Second,
	})

	testError := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testError
		})
	}

	// Wait for reset timeout (half-open)
	time.Sleep(1100 * time.Millisecond)

	// Execute 3 successful operations in half-open
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return nil
		})
	}

	// Circuit should be closed now
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state Closed after successes in half-open, got %v", cb.GetStateName())
	}
}

func TestCircuitBreaker_ReopensOnFailureInHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:         "test",
		MaxFailures:  2,
		Timeout:      1 * time.Second,
		ResetTimeout: 1 * time.Second,
	})

	testError := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testError
		})
	}

	// Wait for reset timeout (half-open)
	time.Sleep(1100 * time.Millisecond)

	// Fail in half-open state
	_ = cb.Execute(func() error {
		return testError
	})

	// Circuit should be open again
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state Open after failure in half-open, got %v", cb.GetStateName())
	}
}

func TestCircuitBreaker_Timeout(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:         "test",
		MaxFailures:  3,
		Timeout:      100 * time.Millisecond,
		ResetTimeout: 5 * time.Second,
	})

	err := cb.Execute(func() error {
		time.Sleep(200 * time.Millisecond) // Exceed timeout
		return nil
	})

	if err != ErrTimeout {
		t.Errorf("Expected ErrTimeout, got %v", err)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:         "test",
		MaxFailures:  2,
		Timeout:      1 * time.Second,
		ResetTimeout: 5 * time.Second,
	})

	testError := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testError
		})
	}

	if cb.GetState() != StateOpen {
		t.Error("Circuit should be open")
	}

	// Reset
	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state Closed after reset, got %v", cb.GetStateName())
	}

	// Should allow execution
	executed := false
	_ = cb.Execute(func() error {
		executed = true
		return nil
	})

	if !executed {
		t.Error("Function should execute after reset")
	}
}

func TestCircuitBreaker_PartialFailures(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:         "test",
		MaxFailures:  3,
		Timeout:      1 * time.Second,
		ResetTimeout: 5 * time.Second,
	})

	testError := errors.New("test error")

	// Mix of success and failure (not reaching threshold)
	_ = cb.Execute(func() error { return testError })
	_ = cb.Execute(func() error { return nil })       // Success resets count
	_ = cb.Execute(func() error { return testError })
	_ = cb.Execute(func() error { return testError })

	// Should still be closed (failures reset after success)
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state Closed with partial failures, got %v", cb.GetStateName())
	}
}

func TestCircuitBreaker_ConcurrentExecutions(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:         "test",
		MaxFailures:  10,
		Timeout:      1 * time.Second,
		ResetTimeout: 5 * time.Second,
	})

	// Execute multiple concurrent operations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			err := cb.Execute(func() error {
				time.Sleep(10 * time.Millisecond)
				return nil
			})
			if err != nil {
				t.Errorf("Concurrent execution failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state Closed after concurrent successes, got %v", cb.GetStateName())
	}
}

func TestCircuitBreaker_GetStateName(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:         "test",
		MaxFailures:  1,
		Timeout:      1 * time.Second,
		ResetTimeout: 1 * time.Second,
	})

	// Test state names
	if cb.GetStateName() != "closed" {
		t.Errorf("Expected 'closed', got '%s'", cb.GetStateName())
	}

	// Open circuit
	_ = cb.Execute(func() error { return errors.New("error") })

	if cb.GetStateName() != "open" {
		t.Errorf("Expected 'open', got '%s'", cb.GetStateName())
	}

	// Wait for half-open
	time.Sleep(1100 * time.Millisecond)
	_ = cb.Execute(func() error { return nil })

	// Should transition through half-open
	// (might be closed if successful operations completed)
	state := cb.GetStateName()
	if state != "closed" && state != "half-open" {
		t.Errorf("Expected 'closed' or 'half-open', got '%s'", state)
	}
}
