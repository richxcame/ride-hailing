package errors

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

// SentryConfig holds configuration for Sentry integration
type SentryConfig struct {
	DSN              string
	Environment      string
	Release          string
	SampleRate       float64
	TracesSampleRate float64
	Debug            bool
	EnableTracing    bool
	ServerName       string
	AttachStacktrace bool
}

// DefaultSentryConfig returns a default Sentry configuration
func DefaultSentryConfig() *SentryConfig {
	return &SentryConfig{
		DSN:              os.Getenv("SENTRY_DSN"),
		Environment:      getEnvironment(),
		Release:          os.Getenv("SENTRY_RELEASE"),
		SampleRate:       getSampleRate(),
		TracesSampleRate: getTracesSampleRate(),
		Debug:            os.Getenv("SENTRY_DEBUG") == "true",
		EnableTracing:    os.Getenv("SENTRY_ENABLE_TRACING") != "false", // enabled by default
		ServerName:       os.Getenv("SERVICE_NAME"),
		AttachStacktrace: true,
	}
}

// InitSentry initializes the Sentry SDK with the given configuration
func InitSentry(config *SentryConfig) error {
	// Skip initialization if DSN is not set
	if config.DSN == "" {
		return fmt.Errorf("sentry DSN is not configured")
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              config.DSN,
		Environment:      config.Environment,
		Release:          config.Release,
		SampleRate:       config.SampleRate,
		TracesSampleRate: config.TracesSampleRate,
		Debug:            config.Debug,
		EnableTracing:    config.EnableTracing,
		ServerName:       config.ServerName,
		AttachStacktrace: config.AttachStacktrace,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Filter out business logic errors (validation failures)
			if event.Level == sentry.LevelInfo || event.Level == sentry.LevelDebug {
				return nil
			}
			return event
		},
		BeforeBreadcrumb: func(breadcrumb *sentry.Breadcrumb, hint *sentry.BreadcrumbHint) *sentry.Breadcrumb {
			// Filter sensitive data from breadcrumbs
			if breadcrumb.Category == "http" {
				// Remove sensitive headers
				if breadcrumb.Data != nil {
					delete(breadcrumb.Data, "Authorization")
					delete(breadcrumb.Data, "Cookie")
					delete(breadcrumb.Data, "X-API-Key")
				}
			}
			return breadcrumb
		},
	})

	if err != nil {
		return fmt.Errorf("failed to initialize sentry: %w", err)
	}

	return nil
}

// Flush flushes the Sentry buffer
func Flush(timeout time.Duration) bool {
	return sentry.Flush(timeout)
}

// CaptureError captures an error and sends it to Sentry
func CaptureError(err error) *sentry.EventID {
	if err == nil {
		return nil
	}
	return sentry.CaptureException(err)
}

// CaptureErrorWithContext captures an error with additional context
func CaptureErrorWithContext(ctx context.Context, err error, extras map[string]interface{}) *sentry.EventID {
	if err == nil {
		return nil
	}

	scope := sentry.NewScope()

	// Add context data
	if extras != nil {
		for key, value := range extras {
			scope.SetExtra(key, value)
		}
	}

	// Extract user context from Gin context
	if ginCtx, ok := ctx.(*gin.Context); ok {
		addGinContextToScope(scope, ginCtx)
	}

	// Extract correlation ID
	if correlationID := ctx.Value("correlation_id"); correlationID != nil {
		scope.SetTag("correlation_id", fmt.Sprintf("%v", correlationID))
	}

	return sentry.CaptureException(err)
}

// CaptureMessage captures a message and sends it to Sentry
func CaptureMessage(message string, level sentry.Level) *sentry.EventID {
	return sentry.CaptureMessage(message)
}

// CaptureMessageWithContext captures a message with additional context
func CaptureMessageWithContext(ctx context.Context, message string, level sentry.Level, extras map[string]interface{}) *sentry.EventID {
	scope := sentry.NewScope()
	scope.SetLevel(level)

	// Add context data
	if extras != nil {
		for key, value := range extras {
			scope.SetExtra(key, value)
		}
	}

	// Extract user context from Gin context
	if ginCtx, ok := ctx.(*gin.Context); ok {
		addGinContextToScope(scope, ginCtx)
	}

	return sentry.CaptureMessage(message)
}

// RecoverWithSentry recovers from panic and sends to Sentry
func RecoverWithSentry() {
	if err := recover(); err != nil {
		sentry.CurrentHub().Recover(err)
		sentry.Flush(2 * time.Second)
		panic(err) // re-panic after sending to Sentry
	}
}

// AddBreadcrumb adds a breadcrumb for tracking user actions
func AddBreadcrumb(breadcrumb *sentry.Breadcrumb) {
	sentry.AddBreadcrumb(breadcrumb)
}

// AddBreadcrumbForRequest adds a breadcrumb for HTTP request
func AddBreadcrumbForRequest(method, url string, statusCode int, duration time.Duration) {
	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Type:      "http",
		Category:  "http.request",
		Level:     sentry.LevelInfo,
		Message:   fmt.Sprintf("%s %s", method, url),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"method":      method,
			"url":         url,
			"status_code": statusCode,
			"duration_ms": duration.Milliseconds(),
		},
	})
}

// SetUser sets the user context
func SetUser(userID, email, username, ipAddress string) {
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID:        userID,
			Email:     email,
			Username:  username,
			IPAddress: ipAddress,
		})
	})
}

// SetTag sets a tag
func SetTag(key, value string) {
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag(key, value)
	})
}

// SetContext sets additional context
func SetContext(key string, value map[string]interface{}) {
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetContext(key, value)
	})
}

// IsBusinessError checks if an error is a business logic error that shouldn't be reported
func IsBusinessError(err error) bool {
	if err == nil {
		return false
	}

	// List of business error types that shouldn't be reported to Sentry
	businessErrors := []string{
		"validation failed",
		"invalid input",
		"unauthorized",
		"forbidden",
		"not found",
		"conflict",
		"bad request",
	}

	errMsg := err.Error()
	for _, businessErr := range businessErrors {
		if contains(errMsg, businessErr) {
			return true
		}
	}

	return false
}

// ShouldReportError determines if an error should be reported to Sentry
func ShouldReportError(err error, statusCode int) bool {
	if err == nil {
		return false
	}

	// Don't report business logic errors
	if IsBusinessError(err) {
		return false
	}

	// Don't report client errors (4xx) except 429 (rate limit)
	if statusCode >= 400 && statusCode < 500 && statusCode != 429 {
		return false
	}

	return true
}

// Helper functions

func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = os.Getenv("SENTRY_ENVIRONMENT")
	}
	if env == "" {
		env = "development"
	}
	return env
}

func getSampleRate() float64 {
	rate := os.Getenv("SENTRY_SAMPLE_RATE")
	if rate == "" {
		return 1.0 // 100% in development
	}

	var sampleRate float64
	fmt.Sscanf(rate, "%f", &sampleRate)
	return sampleRate
}

func getTracesSampleRate() float64 {
	rate := os.Getenv("SENTRY_TRACES_SAMPLE_RATE")
	if rate == "" {
		env := getEnvironment()
		if env == "production" {
			return 0.1 // 10% in production
		}
		return 1.0 // 100% in dev/staging
	}

	var sampleRate float64
	fmt.Sscanf(rate, "%f", &sampleRate)
	return sampleRate
}

func addGinContextToScope(scope *sentry.Scope, c *gin.Context) {
	// Set request context
	scope.SetRequest(c.Request)

	// Set user context if available
	if userID, exists := c.Get("user_id"); exists {
		scope.SetUser(sentry.User{
			ID: fmt.Sprintf("%v", userID),
		})
	}

	// Set correlation ID
	if correlationID := c.GetHeader("X-Request-ID"); correlationID != "" {
		scope.SetTag("correlation_id", correlationID)
	}

	// Set trace context if available
	if traceID := c.GetHeader("X-Trace-ID"); traceID != "" {
		scope.SetTag("trace_id", traceID)
	}
	if spanID := c.GetHeader("X-Span-ID"); spanID != "" {
		scope.SetTag("span_id", spanID)
	}

	// Set additional context
	scope.SetContext("http", map[string]interface{}{
		"method":      c.Request.Method,
		"url":         c.Request.URL.String(),
		"query":       c.Request.URL.RawQuery,
		"headers":     sanitizeHeaders(c.Request.Header),
		"remote_addr": c.ClientIP(),
		"user_agent":  c.Request.UserAgent(),
	})
}

func sanitizeHeaders(headers http.Header) map[string]string {
	sanitized := make(map[string]string)
	sensitiveHeaders := map[string]bool{
		"Authorization": true,
		"Cookie":        true,
		"X-API-Key":     true,
		"X-Auth-Token":  true,
	}

	for key, values := range headers {
		if _, isSensitive := sensitiveHeaders[key]; isSensitive {
			sanitized[key] = "[REDACTED]"
		} else if len(values) > 0 {
			sanitized[key] = values[0]
		}
	}

	return sanitized
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		len(s) > len(substr)*2))
}
