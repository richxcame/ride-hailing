package resilience

import (
	"context"
	"errors"
	"time"

	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

// ErrCircuitOpen is returned when the breaker refuses a request because it is open.
var ErrCircuitOpen = errors.New("circuit breaker open")

// Operation represents a call wrapped by the circuit breaker.
type Operation func(ctx context.Context) (interface{}, error)

// Settings defines runtime options for the circuit breaker.
type Settings struct {
	Name             string
	Interval         time.Duration
	Timeout          time.Duration
	FailureThreshold uint32
	SuccessThreshold uint32
}

// CircuitBreaker wraps gobreaker with defaults suitable for our services.
type CircuitBreaker struct {
	breaker  *gobreaker.CircuitBreaker
	fallback FallbackFunc
}

// NewCircuitBreaker constructs a breaker with logging and optional fallback behaviour.
func NewCircuitBreaker(settings Settings, fallback FallbackFunc) *CircuitBreaker {
	readyToTrip := func(counts gobreaker.Counts) bool {
		threshold := settings.FailureThreshold
		if threshold == 0 {
			threshold = 5
		}
		return counts.ConsecutiveFailures >= threshold
	}

	breakerSettings := gobreaker.Settings{
		Name:        settings.Name,
		Timeout:     settings.Timeout,
		Interval:    settings.Interval,
		ReadyToTrip: readyToTrip,
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.Get().Info("circuit breaker state change",
				zap.String("breaker", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	}

	if settings.SuccessThreshold > 0 {
		breakerSettings.MaxRequests = settings.SuccessThreshold
	}

	return &CircuitBreaker{
		breaker:  gobreaker.NewCircuitBreaker(breakerSettings),
		fallback: fallback,
	}
}

// Execute runs the supplied operation through the breaker.
func (c *CircuitBreaker) Execute(ctx context.Context, operation Operation) (interface{}, error) {
	if operation == nil {
		return nil, errors.New("operation cannot be nil")
	}

	if c == nil || c.breaker == nil {
		return operation(ctx)
	}

	result, err := c.breaker.Execute(func() (interface{}, error) {
		return operation(ctx)
	})
	if err == nil {
		return result, nil
	}

	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		if c.fallback != nil {
			return c.fallback(ctx, err)
		}
		return nil, ErrCircuitOpen
	}

	return nil, err
}

// Allow reports whether the breaker would allow a request without executing it.
func (c *CircuitBreaker) Allow() bool {
	if c == nil || c.breaker == nil {
		return true
	}
	return c.breaker.State() != gobreaker.StateOpen
}
