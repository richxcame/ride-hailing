package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/ratelimit"
	"go.uber.org/zap"
)

// RateLimit applies rate limiting to incoming requests using the provided limiter configuration.
func RateLimit(limiter *ratelimit.Limiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	if limiter == nil || !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		endpointPath := c.FullPath()
		if endpointPath == "" {
			endpointPath = c.Request.URL.Path
		}

		endpointKey := fmt.Sprintf("%s:%s", c.Request.Method, endpointPath)

		identityType := ratelimit.IdentityAnonymous
		identity := c.ClientIP()
		if identity == "" {
			identity = "unknown"
		}

		if userID, err := GetUserID(c); err == nil && userID != uuid.Nil {
			identityType = ratelimit.IdentityAuthenticated
			identity = userID.String()
		}

		rule := limiter.RuleFor(endpointKey, identityType)
		if rule.Limit <= 0 {
			c.Next()
			return
		}

		result, err := limiter.Allow(c.Request.Context(), endpointKey, identity, rule, identityType)
		if err != nil {
			logger.WarnContext(c.Request.Context(), "rate limit evaluation failed",
				zap.String("endpoint", endpointKey),
				zap.String("identity", identity),
				zap.Error(err),
			)
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		remaining := result.Remaining
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		resetSeconds := int(result.ResetAfter.Round(time.Second) / time.Second)
		if resetSeconds < 0 {
			resetSeconds = 0
		}
		c.Header("X-RateLimit-Reset", strconv.Itoa(resetSeconds))
		c.Header("X-RateLimit-Resource", endpointKey)

		if result.Allowed {
			c.Next()
			return
		}

		retrySeconds := int(result.RetryAfter.Round(time.Second) / time.Second)
		if retrySeconds <= 0 {
			retrySeconds = 1
		}
		c.Header("Retry-After", strconv.Itoa(retrySeconds))

		logger.WarnContext(c.Request.Context(), "rate limit exceeded",
			zap.String("endpoint", endpointKey),
			zap.String("identity", identity),
			zap.Int("retry_after_seconds", retrySeconds),
		)

		common.ErrorResponse(c, http.StatusTooManyRequests, "rate limit exceeded")
		c.Abort()
	}
}
