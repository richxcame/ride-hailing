package common

import (
	"errors"
	"net/http"
)

// Common error types
var (
	ErrNotFound           = errors.New("resource not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrBadRequest         = errors.New("bad request")
	ErrInternalServer     = errors.New("internal server error")
	ErrConflict           = errors.New("resource conflict")
	ErrValidation         = errors.New("validation error")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("expired token")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// ErrorCode constants for machine-readable error identification.
const (
	// Auth errors
	ErrCodeUnauthorized       = "AUTH_UNAUTHORIZED"
	ErrCodeForbidden          = "AUTH_FORBIDDEN"
	ErrCodeInvalidToken       = "AUTH_INVALID_TOKEN"
	ErrCodeExpiredToken       = "AUTH_EXPIRED_TOKEN"
	ErrCodeInvalidCredentials = "AUTH_INVALID_CREDENTIALS"

	// Validation errors
	ErrCodeValidation  = "VALIDATION_ERROR"
	ErrCodeBadRequest  = "BAD_REQUEST"

	// Resource errors
	ErrCodeNotFound = "RESOURCE_NOT_FOUND"
	ErrCodeConflict = "RESOURCE_CONFLICT"

	// Ride errors
	ErrCodeRideNotAvailable = "RIDE_NOT_AVAILABLE"
	ErrCodeRideNotStarted   = "RIDE_NOT_STARTED"
	ErrCodeRideAlreadyDone  = "RIDE_ALREADY_COMPLETED"

	// Payment errors
	ErrCodePaymentFailed    = "PAYMENT_FAILED"
	ErrCodeInsufficientFunds = "PAYMENT_INSUFFICIENT_FUNDS"

	// Driver errors
	ErrCodeDriverUnauthorized = "DRIVER_UNAUTHORIZED"
	ErrCodeDriverUnavailable  = "DRIVER_UNAVAILABLE"

	// System errors
	ErrCodeInternal           = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeRateLimited        = "RATE_LIMITED"
)

// AppError represents an application error with HTTP status code and error code.
type AppError struct {
	Code      int    `json:"code"`
	ErrorCode string `json:"error_code,omitempty"`
	Message   string `json:"message"`
	Err       error  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// NewAppError creates a new AppError
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common error constructors
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

func NewBadRequestError(message string, err error) *AppError {
	return &AppError{
		Code:      http.StatusBadRequest,
		ErrorCode: ErrCodeBadRequest,
		Message:   message,
		Err:       err,
	}
}

func NewInternalError(message string, err error) *AppError {
	return &AppError{
		Code:      http.StatusInternalServerError,
		ErrorCode: ErrCodeInternal,
		Message:   message,
		Err:       err,
	}
}

func NewInternalErrorWithError(message string, err error) *AppError {
	return &AppError{
		Code:      http.StatusInternalServerError,
		ErrorCode: ErrCodeInternal,
		Message:   message,
		Err:       err,
	}
}

func NewInternalServerError(message string) *AppError {
	return &AppError{
		Code:      http.StatusInternalServerError,
		ErrorCode: ErrCodeInternal,
		Message:   message,
		Err:       ErrInternalServer,
	}
}

func NewConflictError(message string) *AppError {
	return &AppError{
		Code:      http.StatusConflict,
		ErrorCode: ErrCodeConflict,
		Message:   message,
		Err:       ErrConflict,
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

func NewServiceUnavailableError(message string) *AppError {
	return &AppError{
		Code:      http.StatusServiceUnavailable,
		ErrorCode: ErrCodeServiceUnavailable,
		Message:   message,
		Err:       errors.New("service unavailable"),
	}
}

func NewTooManyRequestsError(message string) *AppError {
	return &AppError{
		Code:      http.StatusTooManyRequests,
		ErrorCode: ErrCodeRateLimited,
		Message:   message,
		Err:       errors.New("rate limit exceeded"),
	}
}

func NewForbiddenError(message string) *AppError {
	return &AppError{
		Code:      http.StatusForbidden,
		ErrorCode: ErrCodeForbidden,
		Message:   message,
		Err:       ErrForbidden,
	}
}

// NewErrorWithCode creates an AppError with a custom error code.
func NewErrorWithCode(httpCode int, errorCode, message string, err error) *AppError {
	return &AppError{
		Code:      httpCode,
		ErrorCode: errorCode,
		Message:   message,
		Err:       err,
	}
}
