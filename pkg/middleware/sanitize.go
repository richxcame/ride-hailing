package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/security"
)

const maxSanitizedBodySize = 2 << 20 // 2 MB

// SanitizeRequest normalizes query parameters and JSON request bodies to guard
// against XSS/SQL injection payloads. It should be registered before handlers
// attempt to bind JSON payloads.
func SanitizeRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		sanitizeQueryParams(c)
		sanitizeJSONBody(c)
		c.Next()
	}
}

func sanitizeQueryParams(c *gin.Context) {
	query := c.Request.URL.Query()
	changed := false

	for key, values := range query {
		for i, value := range values {
			sanitized := security.SanitizeInput(value, 0)
			if sanitized != value {
				query[key][i] = sanitized
				changed = true
			}
		}
	}

	if changed {
		c.Request.URL.RawQuery = query.Encode()
	}
}

func sanitizeJSONBody(c *gin.Context) {
	if c.Request.Body == nil {
		return
	}

	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return
	}

	limited := io.LimitReader(c.Request.Body, maxSanitizedBodySize)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		resetRequestBody(c, nil)
		return
	}

	originalBody := append([]byte(nil), bodyBytes...)
	if len(bodyBytes) == 0 {
		resetRequestBody(c, originalBody)
		return
	}

	var payload interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		resetRequestBody(c, originalBody)
		return
	}

	sanitizeJSONValue(&payload)

	sanitizedBytes, err := json.Marshal(payload)
	if err != nil {
		resetRequestBody(c, originalBody)
		return
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(sanitizedBytes))
	c.Set("sanitizedBody", payload)
}

func resetRequestBody(c *gin.Context, body []byte) {
	if body == nil {
		c.Request.Body = http.NoBody
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
}

func sanitizeJSONValue(value *interface{}) {
	switch v := (*value).(type) {
	case string:
		*value = security.SanitizeInput(v, 0)
	case []interface{}:
		for i := range v {
			item := v[i]
			sanitizeJSONValue(&item)
			v[i] = item
		}
	case map[string]interface{}:
		for key, item := range v {
			sanitizeJSONValue(&item)
			v[key] = item
		}
	}
}
