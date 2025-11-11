package resilience

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// RetryConfig defines the configuration for retry behavior
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts (including the initial attempt)
	MaxAttempts int
	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff (typically 2.0)
	BackoffMultiplier float64
	// EnableJitter adds randomization to prevent thundering herd
	EnableJitter bool
	// RetryableErrors is a list of errors that should trigger a retry
	RetryableErrors []error
	// RetryableChecker is a function that determines if an error is retryable
	RetryableChecker func(error) bool
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		EnableJitter:      true,
		RetryableChecker:  nil,
	}
}

// AggressiveRetryConfig returns a more aggressive retry configuration for critical operations
func AggressiveRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       5,
		InitialBackoff:    500 * time.Millisecond,
		MaxBackoff:        16 * time.Second,
		BackoffMultiplier: 2.0,
		EnableJitter:      true,
		RetryableChecker:  nil,
	}
}

// ConservativeRetryConfig returns a conservative retry configuration
func ConservativeRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       2,
		InitialBackoff:    2 * time.Second,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
		EnableJitter:      true,
		RetryableChecker:  nil,
	}
}

// Retry executes the given operation with exponential backoff retry logic
func Retry(ctx context.Context, config RetryConfig, operation Operation) (interface{}, error) {
	return RetryWithName(ctx, config, operation, "unknown")
}

// RetryWithName executes the operation with retry logic and records metrics with the given operation name
func RetryWithName(ctx context.Context, config RetryConfig, operation Operation, operationName string) (interface{}, error) {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 1
	}

	startTime := time.Now()
	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Check if context is already cancelled
		select {
		case <-ctx.Done():
			RecordRetryOperation(operationName, time.Since(startTime).Seconds(), attempt, false)
			return nil, ctx.Err()
		default:
		}

		// Execute the operation
		result, err := operation(ctx)
		if err == nil {
			// Success - record metrics
			RecordRetryAttempt(operationName, true)
			RecordRetryOperation(operationName, time.Since(startTime).Seconds(), attempt, true)

			if attempt > 1 {
				logger.Get().Info("operation succeeded after retry",
					zap.Int("attempt", attempt),
					zap.Int("max_attempts", config.MaxAttempts),
					zap.String("operation", operationName),
				)
			}
			return result, nil
		}

		// Failure - record attempt
		RecordRetryAttempt(operationName, false)
		lastErr = err

		// Check if we should retry
		if !shouldRetry(err, config) {
			logger.Get().Debug("error is not retryable",
				zap.Error(err),
				zap.Int("attempt", attempt),
				zap.String("operation", operationName),
			)
			RecordRetryOperation(operationName, time.Since(startTime).Seconds(), attempt, false)
			return nil, err
		}

		// Don't sleep after the last attempt
		if attempt == config.MaxAttempts {
			logger.Get().Warn("operation failed after all retry attempts",
				zap.Error(err),
				zap.Int("attempts", attempt),
				zap.String("operation", operationName),
			)
			break
		}

		// Calculate backoff duration
		backoff := calculateBackoff(attempt, config)
		RecordRetryBackoff(operationName, backoff.Seconds())

		logger.Get().Info("retrying operation after backoff",
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", config.MaxAttempts),
			zap.Duration("backoff", backoff),
			zap.String("operation", operationName),
			zap.Error(err),
		)

		// Wait for backoff duration or context cancellation
		select {
		case <-ctx.Done():
			RecordRetryOperation(operationName, time.Since(startTime).Seconds(), attempt+1, false)
			return nil, ctx.Err()
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	RecordRetryOperation(operationName, time.Since(startTime).Seconds(), config.MaxAttempts, false)
	return nil, lastErr
}

// RetryWithBreaker combines retry logic with circuit breaker
func RetryWithBreaker(ctx context.Context, config RetryConfig, breaker *CircuitBreaker, operation Operation) (interface{}, error) {
	return Retry(ctx, config, func(ctx context.Context) (interface{}, error) {
		return breaker.Execute(ctx, operation)
	})
}

// calculateBackoff calculates the backoff duration for a given attempt
func calculateBackoff(attempt int, config RetryConfig) time.Duration {
	// Exponential backoff: initial * (multiplier ^ (attempt - 1))
	backoff := float64(config.InitialBackoff) * math.Pow(config.BackoffMultiplier, float64(attempt-1))

	// Apply max backoff cap
	if backoff > float64(config.MaxBackoff) {
		backoff = float64(config.MaxBackoff)
	}

	duration := time.Duration(backoff)

	// Add jitter to prevent thundering herd
	if config.EnableJitter {
		duration = addJitter(duration)
	}

	return duration
}

// addJitter adds randomization to the backoff duration
// Uses "Full Jitter" algorithm: random value between 0 and backoff
func addJitter(duration time.Duration) time.Duration {
	if duration <= 0 {
		return duration
	}
	jitter := rand.Int63n(int64(duration))
	return time.Duration(jitter)
}

// shouldRetry determines if an error is retryable based on the configuration
func shouldRetry(err error, config RetryConfig) bool {
	if err == nil {
		return false
	}

	// Check custom retryable checker first
	if config.RetryableChecker != nil {
		return config.RetryableChecker(err)
	}

	// Check against list of retryable errors
	if len(config.RetryableErrors) > 0 {
		for _, retryableErr := range config.RetryableErrors {
			if errors.Is(err, retryableErr) {
				return true
			}
		}
		return false
	}

	// By default, retry all errors except context cancellation
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Don't retry circuit breaker open errors
	if errors.Is(err, ErrCircuitOpen) {
		return false
	}

	return true
}

// IsRetryableHTTPStatus determines if an HTTP status code is retryable
func IsRetryableHTTPStatus(statusCode int) bool {
	// Retry on:
	// - 408 Request Timeout
	// - 429 Too Many Requests
	// - 500 Internal Server Error
	// - 502 Bad Gateway
	// - 503 Service Unavailable
	// - 504 Gateway Timeout
	switch statusCode {
	case 408, 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}
