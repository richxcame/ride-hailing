package resilience

import "context"

// FallbackFunc is executed when the breaker is open or overloaded.
type FallbackFunc func(ctx context.Context, err error) (interface{}, error)

// NoopFallback returns the breaker open error without additional handling.
func NoopFallback(ctx context.Context, err error) (interface{}, error) {
	return nil, ErrCircuitOpen
}
