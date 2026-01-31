package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code      int    `json:"code"`
	ErrorCode string `json:"error_code,omitempty"`
	Message   string `json:"message"`
}

// Meta contains metadata for paginated responses
type Meta struct {
	Page       int         `json:"page,omitempty"`
	PerPage    int         `json:"per_page,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
	Total      int64       `json:"total,omitempty"`
	TotalPages int         `json:"total_pages,omitempty"`
	Stats      interface{} `json:"stats,omitempty"`
}

// SuccessResponse sends a successful response (backward compatibility)
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessResponseWithStatus sends a successful response with custom status code
func SuccessResponseWithStatus(c *gin.Context, statusCode int, data interface{}, message string) {
	c.JSON(statusCode, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessResponseWithMeta sends a successful response with metadata
func SuccessResponseWithMeta(c *gin.Context, data interface{}, meta *Meta) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// SuccessResponseWithMetaAndStatus sends a successful response with metadata and status
func SuccessResponseWithMetaAndStatus(c *gin.Context, statusCode int, data interface{}, meta *Meta, message string) {
	c.JSON(statusCode, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// CreatedResponse sends a created response
func CreatedResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// ErrorResponse sends an error response
func ErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    statusCode,
			Message: message,
		},
	})
}

// AppErrorResponse sends an AppError response
func AppErrorResponse(c *gin.Context, err *AppError) {
	c.JSON(err.Code, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:      err.Code,
			ErrorCode: err.ErrorCode,
			Message:   err.Message,
		},
	})
}
