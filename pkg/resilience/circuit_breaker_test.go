package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerTripsAndReturnsOpenError(t *testing.T) {
	breaker := NewCircuitBreaker(Settings{
		Name:             "test-breaker",
		Timeout:          50 * time.Millisecond,
		Interval:         50 * time.Millisecond,
		FailureThreshold: 2,
		SuccessThreshold: 1,
	}, nil)

	ctx := context.Background()
	failingOp := func(context.Context) (interface{}, error) {
		return nil, errors.New("boom")
	}

	for i := 0; i < 2; i++ {
		if _, err := breaker.Execute(ctx, failingOp); err == nil {
			t.Fatalf("expected failure on iteration %d", i)
		}
	}

	if breaker.Allow() {
		t.Fatalf("breaker should be open after consecutive failures")
	}

	if _, err := breaker.Execute(ctx, func(context.Context) (interface{}, error) {
		return "ok", nil
	}); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreakerPassesThroughOnSuccess(t *testing.T) {
	breaker := NewCircuitBreaker(Settings{
		Name:             "success-breaker",
		Timeout:          time.Second,
		Interval:         time.Second,
		FailureThreshold: 5,
		SuccessThreshold: 1,
	}, nil)

	ctx := context.Background()
	result, err := breaker.Execute(ctx, func(context.Context) (interface{}, error) {
		return "response", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.(string) != "response" {
		t.Fatalf("expected response, got %v", result)
	}
}
