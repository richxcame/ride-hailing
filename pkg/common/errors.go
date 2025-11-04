package common

import (
	"errors"
	"net/http"
)

// Common error types
var (
	ErrNotFound          = errors.New("resource not found")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrBadRequest        = errors.New("bad request")
	ErrInternalServer    = errors.New("internal server error")
	ErrConflict          = errors.New("resource conflict")
	ErrValidation        = errors.New("validation error")
	ErrInvalidToken      = errors.New("invalid token")
	ErrExpiredToken      = errors.New("expired token")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// AppError represents an application error with HTTP status code
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
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
		Code:    http.StatusNotFound,
		Message: message,
		Err:     err,
	}
}

func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Code:    http.StatusUnauthorized,
		Message: message,
		Err:     ErrUnauthorized,
	}
}

func NewBadRequestError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: message,
		Err:     err,
	}
}

func NewInternalError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: message,
		Err:     err,
	}
}

func NewInternalErrorWithError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: message,
		Err:     err,
	}
}

func NewInternalServerError(message string) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: message,
		Err:     ErrInternalServer,
	}
}

func NewConflictError(message string) *AppError {
	return &AppError{
		Code:    http.StatusConflict,
		Message: message,
		Err:     ErrConflict,
	}
}

func NewValidationError(message string) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: message,
		Err:     ErrValidation,
	}
}
