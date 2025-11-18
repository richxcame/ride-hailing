package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/errors"
)

// SentryMiddleware returns a middleware that integrates Sentry error tracking
// This middleware automatically captures:
// - Panics with full stack traces
// - HTTP errors (5xx status codes)
// - Request context (headers, body, user info)
// - Breadcrumbs for request flow tracking
func SentryMiddleware() gin.HandlerFunc {
	return sentrygin.New(sentrygin.Options{
		Repanic:         true,
		WaitForDelivery: false,
		Timeout:         2 * time.Second,
	})
}

// ErrorHandler is a custom error handler middleware that captures errors and sends them to Sentry
// It should be placed after other middleware in the chain
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Capture request duration
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// Add breadcrumb for this request
		errors.AddBreadcrumbForRequest(
			c.Request.Method,
			c.Request.URL.Path,
			statusCode,
			duration,
		)

		// Check if there were any errors during request processing
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				// Only report unexpected errors to Sentry
				if errors.ShouldReportError(err.Err, statusCode) {
					captureErrorWithContext(c, err.Err, statusCode, duration)
				}
			}
		}

		// Capture 5xx errors even if no explicit error was set
		if statusCode >= 500 && len(c.Errors) == 0 {
			captureHTTPError(c, statusCode, duration)
		}
	}
}

// RecoveryWithSentry returns a middleware that recovers from panics and reports them to Sentry
func RecoveryWithSentry() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Capture the panic
				hub := sentrygin.GetHubFromContext(c)
				if hub == nil {
					hub = sentry.CurrentHub().Clone()
				}

				// Add request context
				hub.Scope().SetRequest(c.Request)
				hub.Scope().SetContext("panic", map[string]interface{}{
					"value":      fmt.Sprintf("%v", err),
					"stacktrace": string(debug.Stack()),
				})

				// Set user context if available
				if userID, exists := c.Get("user_id"); exists {
					hub.Scope().SetUser(sentry.User{
						ID: fmt.Sprintf("%v", userID),
					})
				}

				// Capture the panic
				hub.RecoverWithContext(c.Request.Context(), err)
				hub.Flush(2 * time.Second)

				// Return 500 error
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal Server Error",
					"message": "An unexpected error occurred",
				})
			}
		}()

		c.Next()
	}
}

// captureErrorWithContext captures an error with full context
func captureErrorWithContext(c *gin.Context, err error, statusCode int, duration time.Duration) {
	hub := sentrygin.GetHubFromContext(c)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
	}

	// Configure scope with request details
	hub.Scope().SetRequest(c.Request)
	hub.Scope().SetLevel(getSentryLevel(statusCode))

	// Set user context
	if userID, exists := c.Get("user_id"); exists {
		userIDStr := fmt.Sprintf("%v", userID)
		email, _ := c.Get("user_email")
		username, _ := c.Get("user_name")

		hub.Scope().SetUser(sentry.User{
			ID:        userIDStr,
			Email:     fmt.Sprintf("%v", email),
			Username:  fmt.Sprintf("%v", username),
			IPAddress: c.ClientIP(),
		})
	}

	// Set tags
	hub.Scope().SetTag("http.method", c.Request.Method)
	hub.Scope().SetTag("http.status_code", fmt.Sprintf("%d", statusCode))
	hub.Scope().SetTag("endpoint", c.Request.URL.Path)

	// Set correlation ID
	if correlationID := c.GetHeader("X-Request-ID"); correlationID != "" {
		hub.Scope().SetTag("correlation_id", correlationID)
	}

	// Set trace context
	if traceID := c.GetHeader("X-Trace-ID"); traceID != "" {
		hub.Scope().SetTag("trace_id", traceID)
	}
	if spanID := c.GetHeader("X-Span-ID"); spanID != "" {
		hub.Scope().SetTag("span_id", spanID)
	}

	// Add extra context
	hub.Scope().SetContext("http", map[string]interface{}{
		"method":       c.Request.Method,
		"url":          c.Request.URL.String(),
		"status_code":  statusCode,
		"duration_ms":  duration.Milliseconds(),
		"remote_addr":  c.ClientIP(),
		"user_agent":   c.Request.UserAgent(),
		"content_type": c.ContentType(),
	})

	// Add route context
	hub.Scope().SetContext("route", map[string]interface{}{
		"path":    c.Request.URL.Path,
		"handler": c.HandlerName(),
	})

	// Capture the error
	hub.CaptureException(err)
}

// captureHTTPError captures HTTP errors without explicit Go errors
func captureHTTPError(c *gin.Context, statusCode int, duration time.Duration) {
	hub := sentrygin.GetHubFromContext(c)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
	}

	// Configure scope
	hub.Scope().SetRequest(c.Request)
	hub.Scope().SetLevel(getSentryLevel(statusCode))

	// Set tags
	hub.Scope().SetTag("http.method", c.Request.Method)
	hub.Scope().SetTag("http.status_code", fmt.Sprintf("%d", statusCode))
	hub.Scope().SetTag("endpoint", c.Request.URL.Path)

	// Capture as a message
	message := fmt.Sprintf("HTTP %d: %s %s", statusCode, c.Request.Method, c.Request.URL.Path)
	hub.CaptureMessage(message)
}

// getSentryLevel maps HTTP status codes to Sentry severity levels
func getSentryLevel(statusCode int) sentry.Level {
	switch {
	case statusCode >= 500:
		return sentry.LevelError
	case statusCode == 429:
		return sentry.LevelWarning
	case statusCode >= 400:
		return sentry.LevelInfo
	default:
		return sentry.LevelInfo
	}
}

// SentryBreadcrumbMiddleware adds breadcrumbs for each request
func SentryBreadcrumbMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add breadcrumb before processing
		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Type:      "http",
			Category:  "http.request.start",
			Message:   fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
			Level:     sentry.LevelInfo,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"method": c.Request.Method,
				"url":    c.Request.URL.Path,
			},
		})

		c.Next()

		// Add breadcrumb after processing
		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Type:      "http",
			Category:  "http.request.end",
			Message:   fmt.Sprintf("%s %s - %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status()),
			Level:     sentry.LevelInfo,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"method":      c.Request.Method,
				"url":         c.Request.URL.Path,
				"status_code": c.Writer.Status(),
			},
		})
	}
}

// SetSentryUser sets the user context in Sentry from authentication middleware
// This should be called after authentication middleware
func SetSentryUser(c *gin.Context) {
	hub := sentrygin.GetHubFromContext(c)
	if hub == nil {
		return
	}

	// Extract user info from context
	userID, userIDExists := c.Get("user_id")
	email, _ := c.Get("user_email")
	username, _ := c.Get("user_name")
	role, _ := c.Get("user_role")

	if userIDExists {
		hub.Scope().SetUser(sentry.User{
			ID:        fmt.Sprintf("%v", userID),
			Email:     fmt.Sprintf("%v", email),
			Username:  fmt.Sprintf("%v", username),
			IPAddress: c.ClientIP(),
		})

		// Set role as a tag
		if role != nil {
			hub.Scope().SetTag("user.role", fmt.Sprintf("%v", role))
		}
	}
}
