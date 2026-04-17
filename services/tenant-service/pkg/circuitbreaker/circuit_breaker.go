package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name             string
	maxFailures      int
	timeout          time.Duration
	resetTimeout     time.Duration
	state            State
	failures         int
	successCount     int
	lastFailureTime  time.Time
	mu               sync.RWMutex
}

// Config holds circuit breaker configuration
type Config struct {
	Name         string
	MaxFailures  int           // Number of failures before opening
	Timeout      time.Duration // Timeout for operations
	ResetTimeout time.Duration // Time before moving from open to half-open
}

var (
	ErrCircuitOpen     = errors.New("circuit breaker is open")
	ErrTimeout         = errors.New("operation timeout")
	ErrTooManyFailures = errors.New("too many failures")
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		name:         config.Name,
		maxFailures:  config.MaxFailures,
		timeout:      config.Timeout,
		resetTimeout: config.ResetTimeout,
		state:        StateClosed,
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	// Check if circuit is open
	if !cb.canExecute() {
		return ErrCircuitOpen
	}

	// Execute with timeout
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		if err != nil {
			cb.recordFailure()
			return err
		}
		cb.recordSuccess()
		return nil
	case <-time.After(cb.timeout):
		cb.recordFailure()
		return ErrTimeout
	}
}

// canExecute checks if execution is allowed
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if we should move to half-open
		if now.Sub(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.failures = 0
			cb.successCount = 0
			return true
		}
		return false
	case StateHalfOpen:
		return true
	}

	return false
}

// recordFailure records a failure
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.state == StateHalfOpen {
		// Go back to open if we fail in half-open state
		cb.state = StateOpen
		cb.successCount = 0
	} else if cb.failures >= cb.maxFailures {
		// Open the circuit
		cb.state = StateOpen
	}
}

// recordSuccess records a success
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.successCount++
		// Close circuit after successful executions in half-open
		if cb.successCount >= 3 {
			cb.state = StateClosed
			cb.failures = 0
			cb.successCount = 0
		}
	} else {
		// Reset failure count on success
		cb.failures = 0
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStateName returns the state as a string
func (cb *CircuitBreaker) GetStateName() string {
	state := cb.GetState()
	switch state {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successCount = 0
}
