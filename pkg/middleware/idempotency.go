package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	redisClient "github.com/richxcame/ride-hailing/pkg/redis"
	"go.uber.org/zap"
)

const (
	// IdempotencyKeyHeader is the HTTP header for idempotency keys
	IdempotencyKeyHeader = "Idempotency-Key"
	// idempotencyTTL is how long idempotency results are cached (24 hours)
	idempotencyTTL = 24 * time.Hour
	// idempotencyPrefix is the Redis key prefix
	idempotencyPrefix = "idempotency:"
)

// idempotencyEntry stores the cached response for a given idempotency key
type idempotencyEntry struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       json.RawMessage   `json:"body"`
	RequestHash string           `json:"request_hash"`
}

// idempotencyResponseWriter captures the response for caching
type idempotencyResponseWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (w *idempotencyResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *idempotencyResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Idempotency middleware ensures that POST/PATCH/PUT requests with the same
// Idempotency-Key header return the same response without re-executing the handler.
// This prevents duplicate ride requests, duplicate payments, etc.
func Idempotency(redis redisClient.ClientInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to mutating methods
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPatch && c.Request.Method != http.MethodPut {
			c.Next()
			return
		}

		idempotencyKey := c.GetHeader(IdempotencyKeyHeader)
		if idempotencyKey == "" {
			// No idempotency key provided - proceed without idempotency
			c.Next()
			return
		}

		// Build a request fingerprint to detect misuse (same key, different request)
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			common.ErrorResponse(c, http.StatusBadRequest, "failed to read request body")
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		requestHash := hashRequest(c.Request.Method, c.FullPath(), bodyBytes)

		// Get user identity for key scoping
		userID := ""
		if uid, err := GetUserID(c); err == nil {
			userID = uid.String()
		}

		redisKey := fmt.Sprintf("%s%s:%s", idempotencyPrefix, userID, idempotencyKey)

		// Check if we have a cached response for this key
		cached, err := redis.GetString(c.Request.Context(), redisKey)
		if err == nil && cached != "" {
			// Found cached response
			var entry idempotencyEntry
			if err := json.Unmarshal([]byte(cached), &entry); err == nil {
				// Verify the request hash matches (same key must be same request)
				if entry.RequestHash != requestHash {
					common.ErrorResponse(c, http.StatusUnprocessableEntity,
						"Idempotency-Key has already been used with a different request")
					c.Abort()
					return
				}

				// Return the cached response
				for k, v := range entry.Headers {
					c.Header(k, v)
				}
				c.Header("Idempotent-Replayed", "true")
				c.Data(entry.StatusCode, "application/json; charset=utf-8", entry.Body)
				c.Abort()
				return
			}
		}

		// No cached response - proceed with the request and capture the response
		writer := &idempotencyResponseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
			statusCode:     http.StatusOK,
		}
		c.Writer = writer

		c.Next()

		// Only cache successful responses (2xx)
		if writer.statusCode >= 200 && writer.statusCode < 300 {
			headers := map[string]string{
				"Content-Type": c.Writer.Header().Get("Content-Type"),
			}

			entry := idempotencyEntry{
				StatusCode:  writer.statusCode,
				Headers:     headers,
				Body:        writer.body.Bytes(),
				RequestHash: requestHash,
			}

			data, err := json.Marshal(entry)
			if err == nil {
				if err := redis.SetWithExpiration(c.Request.Context(), redisKey, data, idempotencyTTL); err != nil {
					logger.WarnContext(c.Request.Context(), "failed to cache idempotency response",
						zap.String("key", idempotencyKey),
						zap.Error(err),
					)
				}
			}
		}
	}
}

// hashRequest creates a SHA-256 hash of the request method, path, and body
func hashRequest(method, path string, body []byte) string {
	h := sha256.New()
	h.Write([]byte(method))
	h.Write([]byte(path))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}
