# ADR-0005: Standardized Error Handling Pattern

## Status

Accepted

## Context

A distributed ride-hailing platform generates errors across 14 microservices. Challenges include:

1. **Debugging complexity**: Tracing errors across service boundaries requires correlation.
2. **Client consistency**: Mobile apps need predictable error response formats.
3. **Error classification**: Business errors (invalid promo code) vs. system errors (database timeout) need different handling.
4. **Monitoring**: Error rates and types must be trackable in observability systems.
5. **Security**: Stack traces and internal details must not leak to clients.

## Decision

Implement a standardized `AppError` type with correlation IDs across all services.

### AppError Structure

```go
// From pkg/common/errors.go
type AppError struct {
    Code      int    `json:"code"`       // HTTP status code
    ErrorCode string `json:"error_code"` // Machine-readable error code
    Message   string `json:"message"`    // Human-readable message
    Err       error  `json:"-"`          // Internal error (not serialized)
}

func (e *AppError) Error() string {
    if e.Err != nil {
        return e.Err.Error()
    }
    return e.Message
}
```

### Error Code Constants

```go
// From pkg/common/errors.go
const (
    // Auth errors
    ErrCodeUnauthorized       = "AUTH_UNAUTHORIZED"
    ErrCodeForbidden          = "AUTH_FORBIDDEN"
    ErrCodeInvalidToken       = "AUTH_INVALID_TOKEN"
    ErrCodeExpiredToken       = "AUTH_EXPIRED_TOKEN"

    // Validation errors
    ErrCodeValidation  = "VALIDATION_ERROR"
    ErrCodeBadRequest  = "BAD_REQUEST"

    // Resource errors
    ErrCodeNotFound = "RESOURCE_NOT_FOUND"
    ErrCodeConflict = "RESOURCE_CONFLICT"

    // Ride errors
    ErrCodeRideNotAvailable = "RIDE_NOT_AVAILABLE"
    ErrCodeRideAlreadyDone  = "RIDE_ALREADY_COMPLETED"

    // Payment errors
    ErrCodePaymentFailed     = "PAYMENT_FAILED"
    ErrCodeInsufficientFunds = "PAYMENT_INSUFFICIENT_FUNDS"

    // System errors
    ErrCodeInternal           = "INTERNAL_ERROR"
    ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
    ErrCodeRateLimited        = "RATE_LIMITED"
)
```

### Error Constructors

```go
// From pkg/common/errors.go
func NewNotFoundError(message string, err error) *AppError {
    return &AppError{
        Code:      http.StatusNotFound,
        ErrorCode: ErrCodeNotFound,
        Message:   message,
        Err:       err,
    }
}

func NewUnauthorizedError(message string) *AppError {
    return &AppError{
        Code:      http.StatusUnauthorized,
        ErrorCode: ErrCodeUnauthorized,
        Message:   message,
        Err:       ErrUnauthorized,
    }
}

func NewValidationError(message string) *AppError {
    return &AppError{
        Code:      http.StatusBadRequest,
        ErrorCode: ErrCodeValidation,
        Message:   message,
        Err:       ErrValidation,
    }
}
```

### Correlation ID Propagation

```go
// From pkg/middleware/correlation_id.go
const (
    CorrelationIDHeader = "X-Request-ID"
    CorrelationIDKey    = "correlation_id"
)

func CorrelationID() gin.HandlerFunc {
    return func(c *gin.Context) {
        correlationID := strings.TrimSpace(c.GetHeader(CorrelationIDHeader))

        // Validate or generate UUID
        if correlationID != "" {
            if _, err := uuid.Parse(correlationID); err != nil {
                correlationID = ""
            }
        }
        if correlationID == "" {
            correlationID = uuid.New().String()
        }

        // Store in context
        c.Set(CorrelationIDKey, correlationID)
        ctx := logger.ContextWithCorrelationID(c.Request.Context(), correlationID)
        c.Request = c.Request.WithContext(ctx)

        // Return in response header
        c.Writer.Header().Set(CorrelationIDHeader, correlationID)
        c.Next()
    }
}
```

### Handler Error Pattern

```go
// From internal/geo/handler.go
func (h *Handler) FindNearbyDrivers(c *gin.Context) {
    // ... parameter validation ...

    drivers, err := h.service.FindAvailableDrivers(c.Request.Context(), latitude, longitude, limit)
    if err != nil {
        if appErr, ok := err.(*common.AppError); ok {
            common.AppErrorResponse(c, appErr)
            return
        }
        common.ErrorResponse(c, http.StatusInternalServerError, "failed to find nearby drivers")
        return
    }

    common.SuccessResponse(c, gin.H{"drivers": drivers})
}
```

### Error Response Format

All error responses follow this JSON structure:

```json
{
    "success": false,
    "error": {
        "code": 404,
        "error_code": "RESOURCE_NOT_FOUND",
        "message": "Driver not found",
        "correlation_id": "550e8400-e29b-41d4-a716-446655440000"
    }
}
```

### Sentry Integration

```go
// From pkg/errors/sentry.go
func CaptureErrorWithContext(ctx context.Context, err error, extras map[string]interface{}) *sentry.EventID {
    scope := sentry.NewScope()

    // Add extras
    for key, value := range extras {
        scope.SetExtra(key, value)
    }

    // Extract correlation ID
    if correlationID := ctx.Value("correlation_id"); correlationID != nil {
        scope.SetTag("correlation_id", fmt.Sprintf("%v", correlationID))
    }

    return sentry.CaptureException(err)
}

// Business errors are filtered from Sentry
func IsBusinessError(err error) bool {
    businessErrors := []string{
        "validation failed", "invalid input", "unauthorized",
        "forbidden", "not found", "conflict", "bad request",
    }
    // ... matching logic ...
}
```

## Consequences

### Positive

- **Consistent client handling**: Mobile apps parse errors uniformly across all endpoints.
- **Distributed tracing**: Correlation IDs enable end-to-end request tracking.
- **Error classification**: Machine-readable codes enable automated alerting.
- **Secure by default**: Internal errors never leak to clients.
- **Testable**: Tests can assert on specific error codes.

### Negative

- **Boilerplate**: Every handler includes error type checking.
- **Error wrapping discipline**: Developers must use constructors consistently.
- **Translation overhead**: Some errors need mapping to AppError.

### Test Assertions

```go
// From internal/auth/service_test.go
func TestService_Register_UserAlreadyExists(t *testing.T) {
    // ... setup ...
    user, err := service.Register(ctx, req)

    assert.Error(t, err)
    assert.Nil(t, user)

    var appErr *common.AppError
    assert.True(t, errors.As(err, &appErr))
    assert.Equal(t, 409, appErr.Code)  // Conflict status
}
```

## References

- [pkg/common/errors.go](/pkg/common/errors.go) - AppError definition
- [pkg/middleware/correlation_id.go](/pkg/middleware/correlation_id.go) - Correlation ID middleware
- [pkg/errors/sentry.go](/pkg/errors/sentry.go) - Sentry integration
- [internal/geo/handler.go](/internal/geo/handler.go) - Error handling in handlers
- [internal/auth/service_test.go](/internal/auth/service_test.go) - Error assertions in tests
