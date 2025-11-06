package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/security"
	"go.uber.org/zap"
)

type responseRecorder struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	r.body.Write(data)
	return r.ResponseWriter.Write(data)
}

func (r *responseRecorder) WriteString(data string) (int, error) {
	r.body.WriteString(data)
	return r.ResponseWriter.WriteString(data)
}

// RequestLogger logs HTTP requests
func RequestLogger(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestBody := captureRequestBody(c)
		recorder := &responseRecorder{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = recorder

		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		fields := []zap.Field{
			zap.String("service", serviceName),
			zap.Int("status", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.Int("response_size", recorder.body.Len()),
		}

		if requestBody != "" {
			fields = append(fields, zap.String("request_body", requestBody))
		}

		if responseBody := sanitizePayload(recorder.body.Bytes()); responseBody != "" {
			fields = append(fields, zap.String("response_body", responseBody))
		}

		reqLogger := logger.WithContext(c.Request.Context())

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
			reqLogger.Error("Request completed with errors", fields...)
		} else {
			reqLogger.Info("Request completed", fields...)
		}
	}
}

func captureRequestBody(c *gin.Context) string {
	if c.Request == nil || c.Request.Body == nil {
		return ""
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return sanitizePayload(bodyBytes)
}

func sanitizePayload(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}

	sanitized := security.StripHTMLTags(string(payload))
	sanitized = security.SanitizeString(sanitized)
	sanitized = strings.Join(strings.Fields(sanitized), " ")

	const maxPayloadLength = 512
	if len(sanitized) > maxPayloadLength {
		sanitized = sanitized[:maxPayloadLength] + "...(truncated)"
	}

	return sanitized
}
