package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

type contextKey string

const (
	correlationIDContextKey contextKey = "correlation_id"
)

// Init initializes the global logger
func Init(environment string) error {
	var err error
	var config zap.Config

	if environment == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	log, err = config.Build()
	if err != nil {
		return err
	}

	return nil
}

// Get returns the global logger instance
func Get() *zap.Logger {
	if log == nil {
		// Fallback to a basic logger if Init wasn't called
		log, _ = zap.NewDevelopment()
	}
	return log
}

// WithContext returns a logger enriched with context-aware fields like correlation ID.
func WithContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return Get()
	}

	if correlationID := CorrelationIDFromContext(ctx); correlationID != "" {
		return Get().With(zap.String(string(correlationIDContextKey), correlationID))
	}

	return Get()
}

// ContextWithCorrelationID returns a context containing the provided correlation ID.
func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, correlationIDContextKey, correlationID)
}

// CorrelationIDFromContext extracts a correlation ID from the provided context if available.
func CorrelationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if value := ctx.Value(correlationIDContextKey); value != nil {
		if correlationID, ok := value.(string); ok {
			return correlationID
		}
	}

	return ""
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// InfoContext logs an info message enriched with context-aware fields.
func InfoContext(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Info(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// ErrorContext logs an error message enriched with context-aware fields.
func ErrorContext(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Error(msg, fields...)
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// DebugContext logs a debug message enriched with context-aware fields.
func DebugContext(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Debug(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// WarnContext logs a warning message enriched with context-aware fields.
func WarnContext(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Warn(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// FatalContext logs a fatal message enriched with context-aware fields and exits.
func FatalContext(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Fatal(msg, fields...)
}

// Sync flushes any buffered log entries
func Sync() error {
	if log != nil {
		return log.Sync()
	}
	return nil
}
