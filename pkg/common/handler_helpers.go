package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// HandleServiceError handles service errors with consistent patterns.
// Returns true if an error was handled (and response was sent), false otherwise.
// This standardizes error handling across all handlers.
//
// Usage:
//
//	result, err := h.service.DoSomething(ctx, req)
//	if HandleServiceError(c, err, "failed to do something") {
//	    return
//	}
func HandleServiceError(c *gin.Context, err error, fallbackMessage string) bool {
	if err == nil {
		return false
	}

	// Check for AppError first (typed business errors)
	if appErr, ok := err.(*AppError); ok {
		AppErrorResponse(c, appErr)
		return true
	}

	// Log the unexpected error for debugging
	logger.ErrorContext(c.Request.Context(), fallbackMessage,
		zap.Error(err),
	)

	// Return generic internal server error
	ErrorResponse(c, http.StatusInternalServerError, fallbackMessage)
	return true
}

// HandleServiceErrorWithCode handles service errors with a custom fallback status code.
// Useful when the default behavior should be something other than 500.
func HandleServiceErrorWithCode(c *gin.Context, err error, fallbackCode int, fallbackMessage string) bool {
	if err == nil {
		return false
	}

	// Check for AppError first (typed business errors)
	if appErr, ok := err.(*AppError); ok {
		AppErrorResponse(c, appErr)
		return true
	}

	// Log the unexpected error for debugging
	logger.ErrorContext(c.Request.Context(), fallbackMessage,
		zap.Error(err),
	)

	// Return error with custom code
	ErrorResponse(c, fallbackCode, fallbackMessage)
	return true
}

// ParseUUIDParam parses a UUID from a URL parameter.
// Returns the UUID and true on success, or sends an error response and returns false on failure.
//
// Usage:
//
//	rideID, ok := ParseUUIDParam(c, "id", "ride ID")
//	if !ok {
//	    return
//	}
func ParseUUIDParam(c *gin.Context, paramName, displayName string) (uuid.UUID, bool) {
	paramValue := c.Param(paramName)
	if paramValue == "" {
		ErrorResponse(c, http.StatusBadRequest, displayName+" is required")
		return uuid.Nil, false
	}

	id, err := uuid.Parse(paramValue)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid "+displayName)
		return uuid.Nil, false
	}

	return id, true
}

// ParseUUIDQuery parses a UUID from a query parameter.
// Returns the UUID and true on success, or sends an error response and returns false on failure.
// If the parameter is optional and not provided, returns uuid.Nil and true.
//
// Usage:
//
//	driverID, ok := ParseUUIDQuery(c, "driver_id", "driver ID", false)
//	if !ok {
//	    return
//	}
func ParseUUIDQuery(c *gin.Context, paramName, displayName string, required bool) (uuid.UUID, bool) {
	paramValue := c.Query(paramName)
	if paramValue == "" {
		if required {
			ErrorResponse(c, http.StatusBadRequest, displayName+" is required")
			return uuid.Nil, false
		}
		return uuid.Nil, true
	}

	id, err := uuid.Parse(paramValue)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid "+displayName)
		return uuid.Nil, false
	}

	return id, true
}

// BindJSON binds JSON request body and sends error response on failure.
// Returns true on success, false on failure (response already sent).
//
// Usage:
//
//	var req CreateRideRequest
//	if !BindJSON(c, &req) {
//	    return
//	}
func BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}

// BindQuery binds query parameters and sends error response on failure.
// Returns true on success, false on failure (response already sent).
//
// Usage:
//
//	var req ListRidesRequest
//	if !BindQuery(c, &req) {
//	    return
//	}
func BindQuery(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}

// RequireUserID extracts and validates user ID from context.
// Returns the user ID and true on success, or sends an unauthorized response and returns false.
//
// Usage:
//
//	userID, ok := RequireUserID(c, getUserIDFunc)
//	if !ok {
//	    return
//	}
func RequireUserID(c *gin.Context, getUserID func(*gin.Context) (uuid.UUID, error)) (uuid.UUID, bool) {
	userID, err := getUserID(c)
	if err != nil {
		ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, false
	}
	return userID, true
}

// ValidateNotEmpty checks if a string value is not empty and sends error response if it is.
// Returns true if valid, false if empty (response already sent).
func ValidateNotEmpty(c *gin.Context, value, fieldName string) bool {
	if value == "" {
		ErrorResponse(c, http.StatusBadRequest, fieldName+" is required")
		return false
	}
	return true
}

// ValidatePositive checks if a number is positive and sends error response if not.
// Returns true if valid, false if invalid (response already sent).
func ValidatePositive(c *gin.Context, value float64, fieldName string) bool {
	if value <= 0 {
		ErrorResponse(c, http.StatusBadRequest, fieldName+" must be positive")
		return false
	}
	return true
}

// ValidateInRange checks if a number is within a range and sends error response if not.
// Returns true if valid, false if invalid (response already sent).
func ValidateInRange(c *gin.Context, value, min, max float64, fieldName string) bool {
	if value < min || value > max {
		ErrorResponse(c, http.StatusBadRequest, fieldName+" must be between "+formatFloat(min)+" and "+formatFloat(max))
		return false
	}
	return true
}

// formatFloat formats a float64 for display in error messages
func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return string(rune(int64(f) + '0'))
	}
	// Simple formatting for common cases
	return string(rune(int64(f)))
}
