package async

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// contextKey is a private type for context keys to prevent collisions
type contextKey string

const (
	// correlationIDKey is the context key for correlation ID
	correlationIDKey contextKey = "correlation_id"
	// spanIDKey is the context key for span ID
	spanIDKey contextKey = "span_id"
	// userIDKey is the context key for user ID
	userIDKey contextKey = "user_id"
)

// TaskContext holds context values that should be propagated to async tasks
type TaskContext struct {
	CorrelationID string
	SpanID        string
	UserID        string
	StartTime     time.Time
	TaskName      string
}

// CaptureContext captures the current context values for async propagation
func CaptureContext(ctx context.Context, taskName string) TaskContext {
	tc := TaskContext{
		StartTime: time.Now(),
		TaskName:  taskName,
	}

	// Extract correlation ID from logger package
	tc.CorrelationID = logger.CorrelationIDFromContext(ctx)

	// Extract other values if present
	if spanID, ok := ctx.Value(spanIDKey).(string); ok {
		tc.SpanID = spanID
	}
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		tc.UserID = userID
	}

	return tc
}

// NewContext creates a new context with the captured values
func (tc TaskContext) NewContext() context.Context {
	ctx := context.Background()

	// Inject correlation ID using logger package
	if tc.CorrelationID != "" {
		ctx = logger.ContextWithCorrelationID(ctx, tc.CorrelationID)
	}

	// Inject other values
	if tc.SpanID != "" {
		ctx = context.WithValue(ctx, spanIDKey, tc.SpanID)
	}
	if tc.UserID != "" {
		ctx = context.WithValue(ctx, userIDKey, tc.UserID)
	}

	return ctx
}

// NewContextWithTimeout creates a new context with timeout and captured values
func (tc TaskContext) NewContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := tc.NewContext()
	return context.WithTimeout(ctx, timeout)
}

// Go runs a function in a goroutine with context propagation and panic recovery
// This is the recommended way to start async tasks that need correlation ID tracking
//
// Usage:
//
//	async.Go(ctx, "send-notification", func(ctx context.Context) {
//	    notificationService.Send(ctx, message)
//	})
func Go(ctx context.Context, taskName string, fn func(ctx context.Context)) {
	tc := CaptureContext(ctx, taskName)

	go func() {
		defer recoverWithLogging(tc)

		newCtx := tc.NewContext()
		fn(newCtx)

		logger.DebugContext(newCtx, "async task completed",
			zap.String("task", tc.TaskName),
			zap.Duration("duration", time.Since(tc.StartTime)),
		)
	}()
}

// GoWithTimeout runs a function in a goroutine with context propagation,
// timeout, and panic recovery
//
// Usage:
//
//	async.GoWithTimeout(ctx, "process-payment", 30*time.Second, func(ctx context.Context) {
//	    paymentService.Process(ctx, payment)
//	})
func GoWithTimeout(ctx context.Context, taskName string, timeout time.Duration, fn func(ctx context.Context)) {
	tc := CaptureContext(ctx, taskName)

	go func() {
		defer recoverWithLogging(tc)

		newCtx, cancel := tc.NewContextWithTimeout(timeout)
		defer cancel()

		done := make(chan struct{})
		go func() {
			defer close(done)
			fn(newCtx)
		}()

		select {
		case <-done:
			logger.DebugContext(newCtx, "async task completed",
				zap.String("task", tc.TaskName),
				zap.Duration("duration", time.Since(tc.StartTime)),
			)
		case <-newCtx.Done():
			logger.WarnContext(newCtx, "async task timed out",
				zap.String("task", tc.TaskName),
				zap.Duration("timeout", timeout),
			)
		}
	}()
}

// GoWithCallback runs a function in a goroutine with a callback for completion
//
// Usage:
//
//	async.GoWithCallback(ctx, "fetch-data", func(ctx context.Context) error {
//	    return dataService.Fetch(ctx, id)
//	}, func(err error) {
//	    if err != nil {
//	        log.Error("fetch failed", zap.Error(err))
//	    }
//	})
func GoWithCallback(ctx context.Context, taskName string, fn func(ctx context.Context) error, callback func(error)) {
	tc := CaptureContext(ctx, taskName)

	go func() {
		defer recoverWithLogging(tc)

		newCtx := tc.NewContext()
		err := fn(newCtx)

		if callback != nil {
			callback(err)
		}

		if err != nil {
			logger.ErrorContext(newCtx, "async task failed",
				zap.String("task", tc.TaskName),
				zap.Duration("duration", time.Since(tc.StartTime)),
				zap.Error(err),
			)
		} else {
			logger.DebugContext(newCtx, "async task completed",
				zap.String("task", tc.TaskName),
				zap.Duration("duration", time.Since(tc.StartTime)),
			)
		}
	}()
}

// recoverWithLogging recovers from panics and logs them with context
func recoverWithLogging(tc TaskContext) {
	if r := recover(); r != nil {
		ctx := tc.NewContext()
		logger.ErrorContext(ctx, "async task panicked",
			zap.String("task", tc.TaskName),
			zap.Any("panic", r),
			zap.String("stack", string(debug.Stack())),
		)
	}
}

// RunAll runs multiple functions concurrently and waits for all to complete
// All functions share the same context propagation
//
// Usage:
//
//	async.RunAll(ctx, "batch-operations",
//	    func(ctx context.Context) { service1.Do(ctx) },
//	    func(ctx context.Context) { service2.Do(ctx) },
//	)
func RunAll(ctx context.Context, taskName string, fns ...func(ctx context.Context)) {
	tc := CaptureContext(ctx, taskName)
	newCtx := tc.NewContext()

	done := make(chan struct{}, len(fns))

	for i, fn := range fns {
		go func(idx int, f func(ctx context.Context)) {
			defer func() {
				if r := recover(); r != nil {
					logger.ErrorContext(newCtx, "async task panicked",
						zap.String("task", tc.TaskName),
						zap.Int("index", idx),
						zap.Any("panic", r),
					)
				}
				done <- struct{}{}
			}()
			f(newCtx)
		}(i, fn)
	}

	// Wait for all to complete
	for range fns {
		<-done
	}

	logger.DebugContext(newCtx, "all async tasks completed",
		zap.String("task", tc.TaskName),
		zap.Int("count", len(fns)),
		zap.Duration("duration", time.Since(tc.StartTime)),
	)
}

// WithCorrelationID adds or replaces the correlation ID in a context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return logger.ContextWithCorrelationID(ctx, correlationID)
}

// GetCorrelationID extracts the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	return logger.CorrelationIDFromContext(ctx)
}

// WithUserID adds user ID to context for async propagation
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}
	return ""
}
