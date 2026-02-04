# Error Handling

## Response Envelope

All API responses use `pkg/common.Response`:

```json
// Success                              // Error
{                                       {
  "success": true,                        "success": false,
  "data": { ... },                        "error": {
  "meta": { "page": 1, "total": 50 }       "code": 400,
}                                           "error_code": "VALIDATION_ERROR",
                                            "message": "Invalid email format"
                                          }
                                        }
```

Use the helpers in `pkg/common/response.go`:

```go
common.SuccessResponse(c, data)                              // 200 + data
common.SuccessResponseWithMeta(c, data, &common.Meta{...})   // 200 + data + pagination
common.CreatedResponse(c, data)                              // 201 + data
common.ErrorResponse(c, http.StatusBadRequest, "bad input")  // status + message
common.AppErrorResponse(c, appErr)                           // uses AppError fields
```

## AppError Type

Defined in `pkg/common/errors.go`. Carries HTTP status, machine-readable code, and message:

```go
type AppError struct {
    Code      int    `json:"code"`
    ErrorCode string `json:"error_code,omitempty"`
    Message   string `json:"message"`
    Err       error  `json:"-"`              // internal, never serialized
}
```

Constructors (all return `*AppError`):

| Constructor                   | HTTP | ErrorCode              |
|-------------------------------|------|------------------------|
| `NewBadRequestError`          | 400  | `BAD_REQUEST`          |
| `NewValidationError`          | 400  | `VALIDATION_ERROR`     |
| `NewUnauthorizedError`        | 401  | `AUTH_UNAUTHORIZED`    |
| `NewForbiddenError`           | 403  | `AUTH_FORBIDDEN`       |
| `NewNotFoundError`            | 404  | `RESOURCE_NOT_FOUND`   |
| `NewConflictError`            | 409  | `RESOURCE_CONFLICT`    |
| `NewTooManyRequestsError`     | 429  | `RATE_LIMITED`         |
| `NewInternalError`            | 500  | `INTERNAL_ERROR`       |
| `NewServiceUnavailableError`  | 503  | `SERVICE_UNAVAILABLE`  |
| `NewErrorWithCode`            | any  | any (custom)           |

Domain-specific error codes (use with `NewErrorWithCode`):

```
RIDE_NOT_AVAILABLE / RIDE_NOT_STARTED / RIDE_ALREADY_COMPLETED
PAYMENT_FAILED / PAYMENT_INSUFFICIENT_FUNDS
DRIVER_UNAUTHORIZED / DRIVER_UNAVAILABLE
AUTH_INVALID_TOKEN / AUTH_EXPIRED_TOKEN / AUTH_INVALID_CREDENTIALS
```

## Sentinel Errors

`pkg/common/errors.go` exports sentinel errors for `errors.Is()`: `ErrNotFound`, `ErrUnauthorized`, `ErrForbidden`, `ErrBadRequest`, `ErrInternalServer`, `ErrConflict`, `ErrValidation`, `ErrInvalidToken`, `ErrExpiredToken`, `ErrInvalidCredentials`. Packages may define their own (e.g. `internal/favorites/errors.go`).

## Sentry: When to Report

Handled by `pkg/errors.ShouldReportError(err, statusCode)` and the `ErrorTracking` middleware.

**Report to Sentry:**
- 5xx errors (server/infra failures)
- Panics (caught by `RecoveryWithSentry` middleware)
- 429 rate-limit hits (possible abuse)

**Do NOT report:**
- 4xx errors (except 429) -- these are expected business/client errors
- Errors matching `IsBusinessError` substrings: `"validation failed"`, `"invalid input"`, `"unauthorized"`, `"forbidden"`, `"not found"`, `"conflict"`, `"bad request"`

### Reporting API (`pkg/errors`)

```go
errors.CaptureError(err)                                         // simple
errors.CaptureErrorWithContext(ctx, err, map[string]interface{}{ // with context
    "user_id": userID, "operation": "payment_processing",
})
if errors.ShouldReportError(err, statusCode) {                   // conditional
    errors.CaptureError(err)
}
```

## Handler Pattern

```go
func (h *Handler) CreateRide(c *gin.Context) {
    var req CreateRideRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        common.ErrorResponse(c, 400, err.Error())     // business error, no Sentry
        return
    }

    ride, err := h.service.CreateRide(c.Request.Context(), &req)
    if err != nil {
        if errors.Is(err, rides.ErrDriverNotFound) {
            common.ErrorResponse(c, 404, "No drivers available")
            return
        }
        logger.Error("Failed to create ride", zap.Error(err))
        errors.CaptureError(err)                        // server error -> Sentry
        common.ErrorResponse(c, 500, "Failed to create ride")
        return
    }

    common.CreatedResponse(c, ride)
}
```
