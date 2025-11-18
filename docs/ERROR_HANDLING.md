# Error Handling Best Practices

This document outlines error handling best practices for the ride-hailing platform.

## Overview

Effective error handling is critical for:
- **Debugging**: Quick identification and resolution of issues
- **User Experience**: Clear, actionable error messages
- **Reliability**: Graceful degradation and recovery
- **Observability**: Tracking and monitoring errors in production

## Error Handling Stack

The platform uses a multi-layered approach:

1. **Application Errors**: Go error types with context
2. **HTTP Status Codes**: RESTful status codes for API responses
3. **Structured Logging**: Zap logger with error context
4. **Error Tracking**: Sentry for centralized error monitoring
5. **Metrics**: Prometheus counters for error rates
6. **Tracing**: OpenTelemetry spans with error status

## Error Categories

### 1. Business Errors (4xx)

These are expected errors resulting from user actions or business rules:

```go
// Validation errors
return c.JSON(400, gin.H{"error": "invalid_input", "message": "Email format is invalid"})

// Authentication errors
return c.JSON(401, gin.H{"error": "unauthorized", "message": "Invalid credentials"})

// Authorization errors
return c.JSON(403, gin.H{"error": "forbidden", "message": "Insufficient permissions"})

// Not found errors
return c.JSON(404, gin.H{"error": "not_found", "message": "User not found"})

// Conflict errors
return c.JSON(409, gin.H{"error": "conflict", "message": "User already exists"})

// Rate limiting
return c.JSON(429, gin.H{"error": "rate_limit_exceeded", "message": "Too many requests"})
```

**Handling**: Log at INFO/WARN level, return to user, do NOT report to Sentry

### 2. Server Errors (5xx)

Unexpected errors indicating system issues:

```go
// Internal server error
return c.JSON(500, gin.H{"error": "internal_error", "message": "An unexpected error occurred"})

// Service unavailable
return c.JSON(503, gin.H{"error": "service_unavailable", "message": "Database connection failed"})

// Gateway timeout
return c.JSON(504, gin.H{"error": "gateway_timeout", "message": "External service timeout"})
```

**Handling**: Log at ERROR level, report to Sentry, return generic message to user

## Error Handling Patterns

### Pattern 1: Simple Error Return

```go
func (s *Service) GetUser(ctx context.Context, userID string) (*User, error) {
    user, err := s.repo.FindByID(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get user %s: %w", userID, err)
    }
    return user, nil
}
```

### Pattern 2: Error with Context

```go
func (s *Service) ProcessPayment(ctx context.Context, req *PaymentRequest) error {
    err := s.stripe.Charge(req.Amount, req.Token)
    if err != nil {
        // Add context before returning
        return fmt.Errorf("stripe charge failed for user %s, amount %f: %w",
            req.UserID, req.Amount, err)
    }
    return nil
}
```

### Pattern 3: Error Classification

```go
func (h *Handler) CreateRide(c *gin.Context) {
    var req CreateRideRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        // Business error - validation failure
        c.JSON(400, gin.H{
            "error": "validation_failed",
            "message": err.Error(),
        })
        return
    }

    ride, err := h.service.CreateRide(c.Request.Context(), &req)
    if err != nil {
        // Check error type
        if errors.Is(err, ErrDriverNotFound) {
            // Business error
            c.JSON(404, gin.H{
                "error": "driver_not_found",
                "message": "No drivers available",
            })
            return
        }

        // Server error - unexpected
        logger.Error("Failed to create ride", zap.Error(err))
        errors.CaptureError(err)
        c.JSON(500, gin.H{
            "error": "internal_error",
            "message": "Failed to create ride",
        })
        return
    }

    c.JSON(201, ride)
}
```

### Pattern 4: Error Recovery

```go
func (s *Service) GetETAWithFallback(ctx context.Context, origin, dest Location) (int, error) {
    // Try ML service
    eta, err := s.mlService.PredictETA(ctx, origin, dest)
    if err != nil {
        logger.Warn("ML ETA prediction failed, using fallback", zap.Error(err))
        // Fallback to simple calculation
        eta = calculateSimpleETA(origin, dest)
    }
    return eta, nil
}
```

### Pattern 5: Panic Recovery

```go
func (s *Service) ProcessCriticalOperation(ctx context.Context) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic recovered: %v", r)
            errors.CaptureError(err)
        }
    }()

    // Critical operation that might panic
    result := s.riskyOperation()
    return nil
}
```

## Custom Error Types

Define custom errors for common business cases:

```go
package rides

import "errors"

var (
    ErrDriverNotFound    = errors.New("no drivers available")
    ErrRideNotFound      = errors.New("ride not found")
    ErrInvalidStatus     = errors.New("invalid ride status")
    ErrUnauthorized      = errors.New("user not authorized for this ride")
    ErrPaymentFailed     = errors.New("payment processing failed")
)

// Usage
func (s *Service) AcceptRide(ctx context.Context, rideID, driverID string) error {
    ride, err := s.repo.GetRide(ctx, rideID)
    if err != nil {
        return fmt.Errorf("get ride: %w", err)
    }

    if ride.Status != StatusPending {
        return ErrInvalidStatus
    }

    // ... rest of logic
}

// In handler
if errors.Is(err, rides.ErrDriverNotFound) {
    c.JSON(404, gin.H{"error": "driver_not_found"})
    return
}
```

## Error Response Format

### Standard Error Response

```json
{
  "error": "error_code",
  "message": "Human-readable error message",
  "details": {
    "field": "email",
    "issue": "Invalid format"
  },
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Validation Error Response

```json
{
  "error": "validation_failed",
  "message": "Request validation failed",
  "errors": [
    {
      "field": "email",
      "message": "Invalid email format"
    },
    {
      "field": "amount",
      "message": "Amount must be positive"
    }
  ],
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

## Logging Errors

### Log Levels

```go
// DEBUG - detailed information for debugging
logger.Debug("Processing payment", zap.String("payment_id", id))

// INFO - general informational messages
logger.Info("Payment processed successfully", zap.String("payment_id", id))

// WARN - warning messages for recoverable issues
logger.Warn("Cache miss, fetching from database", zap.String("key", key))

// ERROR - error messages for unexpected issues
logger.Error("Failed to process payment",
    zap.String("payment_id", id),
    zap.Error(err))

// FATAL - critical errors requiring service shutdown
logger.Fatal("Failed to connect to database", zap.Error(err))
```

### Error Context

Always include context when logging errors:

```go
logger.Error("Database query failed",
    zap.Error(err),
    zap.String("query", "SELECT * FROM users"),
    zap.String("user_id", userID),
    zap.String("correlation_id", correlationID),
)
```

## Reporting Errors to Sentry

### When to Report

**DO Report**:
- Server errors (5xx)
- Panics
- Database connection errors
- External service failures
- Unexpected application errors
- Rate limiting (429) - indicates potential abuse

**DON'T Report**:
- Validation errors (400)
- Authentication failures (401)
- Authorization failures (403)
- Not found errors (404)
- Conflict errors (409)
- Other expected business errors

### How to Report

```go
import "github.com/richxcame/ride-hailing/pkg/errors"

// Simple error reporting
if err != nil {
    errors.CaptureError(err)
    return err
}

// Error with context
if err != nil {
    errors.CaptureErrorWithContext(ctx, err, map[string]interface{}{
        "user_id": userID,
        "operation": "payment_processing",
        "amount": amount,
        "payment_method": "stripe",
    })
    return err
}

// Check if error should be reported
if err != nil {
    statusCode := determineStatusCode(err)
    if errors.ShouldReportError(err, statusCode) {
        errors.CaptureError(err)
    }
    return err
}
```

## Error Handling Checklist

For each error in your code:

- [ ] **Classify the error**: Is it a business error or server error?
- [ ] **Add context**: Include relevant information (user ID, operation, etc.)
- [ ] **Log appropriately**: Use correct log level with context
- [ ] **Report if needed**: Send to Sentry if it's a server error
- [ ] **Return user-friendly message**: Don't expose internal details
- [ ] **Set correct HTTP status**: Use appropriate status code
- [ ] **Add correlation ID**: Include request ID in response
- [ ] **Consider recovery**: Can the operation be retried or fallback used?

## Common Pitfalls

### 1. Swallowing Errors

❌ **Bad**:
```go
user, err := s.GetUser(ctx, userID)
if err != nil {
    // Error ignored!
}
// Continue with nil user
```

✅ **Good**:
```go
user, err := s.GetUser(ctx, userID)
if err != nil {
    return nil, fmt.Errorf("get user: %w", err)
}
```

### 2. Generic Error Messages

❌ **Bad**:
```go
return errors.New("error occurred")
```

✅ **Good**:
```go
return fmt.Errorf("failed to create ride for user %s: %w", userID, err)
```

### 3. Exposing Internal Details

❌ **Bad**:
```go
c.JSON(500, gin.H{"error": "database error: connection timeout on pg-primary-1.internal"})
```

✅ **Good**:
```go
logger.Error("Database connection timeout", zap.Error(err))
errors.CaptureError(err)
c.JSON(500, gin.H{"error": "internal_error", "message": "Service temporarily unavailable"})
```

### 4. Not Adding Context

❌ **Bad**:
```go
return err
```

✅ **Good**:
```go
return fmt.Errorf("process payment for user %s, amount %f: %w", userID, amount, err)
```

### 5. Incorrect Error Reporting

❌ **Bad**:
```go
// Reporting validation errors to Sentry
if err := validate(req); err != nil {
    errors.CaptureError(err) // Don't report business errors!
    return c.JSON(400, gin.H{"error": err.Error()})
}
```

✅ **Good**:
```go
if err := validate(req); err != nil {
    logger.Info("Validation failed", zap.Error(err))
    return c.JSON(400, gin.H{"error": "validation_failed", "message": err.Error()})
}
```

## Testing Error Handling

### Unit Tests

```go
func TestService_CreateRide_DriverNotFound(t *testing.T) {
    service := NewService(mockRepo)

    _, err := service.CreateRide(context.Background(), &req)

    assert.Error(t, err)
    assert.True(t, errors.Is(err, ErrDriverNotFound))
}
```

### Integration Tests

```go
func TestAPI_CreateRide_Returns404WhenNoDrivers(t *testing.T) {
    resp := makeRequest("POST", "/api/v1/rides", rideRequest)

    assert.Equal(t, 404, resp.StatusCode)

    var errResp ErrorResponse
    json.Unmarshal(resp.Body, &errResp)
    assert.Equal(t, "driver_not_found", errResp.Error)
}
```

## Related Documentation

- [Health Checks Guide](./HEALTH_CHECKS.md) - Service health monitoring
- [Observability Guide](./observability.md) - Logging, metrics, and tracing
- [Sentry Integration Guide](./SENTRY_INTEGRATION_GUIDE.md) - Error tracking setup
- [API Documentation](./API.md) - API error responses

## References

- [Go Error Handling Best Practices](https://go.dev/blog/error-handling-and-go)
- [HTTP Status Codes](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status)
- [Sentry Error Tracking](https://docs.sentry.io/platforms/go/)
