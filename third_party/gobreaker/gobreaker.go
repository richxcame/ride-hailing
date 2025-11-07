package gobreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the state of the circuit breaker.
type State int

const (
	// StateClosed allows all requests.
	StateClosed State = iota
	// StateHalfOpen allows a limited number of requests to test recovery.
	StateHalfOpen
	// StateOpen rejects requests until timeout expires.
	StateOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// ErrOpenState is returned when the breaker is open and requests are rejected.
var ErrOpenState = errors.New("circuit breaker is open")

// ErrTooManyRequests is returned in half-open state when the probe budget is exhausted.
var ErrTooManyRequests = errors.New("too many requests in half-open state")

// Counts stores breaker statistics.
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

func (c *Counts) onRequest() {
	c.Requests++
}

func (c *Counts) onSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

func (c *Counts) onFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

func (c *Counts) clear() {
	*c = Counts{}
}

// Settings configures a CircuitBreaker.
type Settings struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(Counts) bool
	OnStateChange func(name string, from State, to State)
	IsSuccessful  func(error) bool
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(Counts) bool
	onStateChange func(name string, from State, to State)
	isSuccessful  func(error) bool

	mutex         sync.Mutex
	state         State
	counts        Counts
	expiry        time.Time
	intervalTimer time.Time
}

// NewCircuitBreaker builds a new breaker.
func NewCircuitBreaker(settings Settings) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          settings.Name,
		maxRequests:   settings.MaxRequests,
		interval:      settings.Interval,
		timeout:       settings.Timeout,
		readyToTrip:   settings.ReadyToTrip,
		onStateChange: settings.OnStateChange,
		isSuccessful:  settings.IsSuccessful,
		state:         StateClosed,
	}

	if cb.readyToTrip == nil {
		cb.readyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 5
		}
	}

	if cb.onStateChange == nil {
		cb.onStateChange = func(string, State, State) {}
	}

	if cb.isSuccessful == nil {
		cb.isSuccessful = func(err error) bool {
			return err == nil
		}
	}

	cb.intervalTimer = time.Now()
	return cb
}

// Name returns the breaker name.
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// State returns the current state.
func (cb *CircuitBreaker) State() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.state
}

// Counts returns a snapshot of breaker statistics.
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.counts
}

// Execute runs a request if allowed by the breaker.
func (cb *CircuitBreaker) Execute(request func() (interface{}, error)) (interface{}, error) {
	if err := cb.beforeRequest(); err != nil {
		return nil, err
	}

	result, err := request()
	cb.afterRequest(err)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (cb *CircuitBreaker) beforeRequest() error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	cb.updateIntervals(now)

	switch cb.state {
	case StateOpen:
		if cb.timeout <= 0 {
			return ErrOpenState
		}
		if now.After(cb.expiry) {
			cb.setStateLocked(StateHalfOpen)
			break
		}
		return ErrOpenState
	case StateHalfOpen:
		if cb.maxRequests > 0 && cb.counts.Requests >= cb.maxRequests {
			return ErrTooManyRequests
		}
	}

	cb.counts.onRequest()
	return nil
}

func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	success := cb.isSuccessful(err)
	if success {
		cb.counts.onSuccess()
		if cb.state == StateHalfOpen {
			if cb.maxRequests == 0 || cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
				cb.setStateLocked(StateClosed)
			}
		}
	} else {
		cb.counts.onFailure()
		if cb.state == StateHalfOpen {
			cb.setStateLocked(StateOpen)
		} else if cb.readyToTrip(cb.counts) {
			cb.setStateLocked(StateOpen)
		}
	}

	if cb.state == StateOpen {
		cb.expiry = time.Now().Add(cb.timeout)
	}
}

func (cb *CircuitBreaker) setStateLocked(state State) {
	if cb.state == state {
		return
	}
	previous := cb.state
	cb.state = state
	cb.counts.clear()
	cb.intervalTimer = time.Now()
	if cb.state == StateOpen {
		cb.expiry = time.Now().Add(cb.timeout)
	}
	cb.onStateChange(cb.name, previous, state)
}

func (cb *CircuitBreaker) updateIntervals(now time.Time) {
	if cb.state != StateClosed || cb.interval <= 0 {
		return
	}
	if now.Sub(cb.intervalTimer) >= cb.interval {
		cb.counts.clear()
		cb.intervalTimer = now
	}
}
