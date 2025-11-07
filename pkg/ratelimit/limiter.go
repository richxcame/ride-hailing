package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/richxcame/ride-hailing/pkg/config"
)

// IdentityType represents the subject of a rate limit decision.
type IdentityType int

const (
	// IdentityAnonymous represents unauthenticated traffic keyed by IP address.
	IdentityAnonymous IdentityType = iota
	// IdentityAuthenticated represents authenticated users keyed by user ID.
	IdentityAuthenticated
)

// Rule defines a rate limiting policy for a single identity and endpoint.
type Rule struct {
	Limit  int
	Burst  int
	Window time.Duration
}

// Result captures the outcome of a rate limiting decision.
type Result struct {
	Allowed      bool
	Remaining    int
	RetryAfter   time.Duration
	Limit        int
	Window       time.Duration
	ResetAfter   time.Duration
	IdentityKey  string
	EndpointKey  string
	IdentityType IdentityType
}

// Limiter implements a Redis-backed token bucket rate limiter.
type Limiter struct {
	client redis.Cmdable
	cfg    config.RateLimitConfig
	script *redis.Script
	now    func() time.Time
}

const tokenBucketScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local refillRate = tonumber(ARGV[2])
local capacity = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

local data = redis.call("HMGET", key, "tokens", "timestamp")
local tokens = tonumber(data[1])
local timestamp = tonumber(data[2])

if tokens == nil then
    tokens = capacity
    timestamp = now
else
    if timestamp == nil then
        timestamp = now
    end
    local delta = now - timestamp
    if delta > 0 then
        tokens = math.min(capacity, tokens + (delta * refillRate))
        timestamp = now
    end
end

local allowed = 0
if tokens >= 1 then
    allowed = 1
    tokens = tokens - 1
end

redis.call("HMSET", key, "tokens", tokens, "timestamp", now)
redis.call("PEXPIRE", key, ttl)

local retryAfter = 0
if allowed == 0 then
    retryAfter = math.ceil((1 - tokens) / refillRate)
end

return {allowed, tokens, retryAfter}
`

// NewLimiter creates a new Limiter instance.
func NewLimiter(client redis.Cmdable, cfg config.RateLimitConfig) *Limiter {
	return &Limiter{
		client: client,
		cfg:    cfg,
		script: redis.NewScript(tokenBucketScript),
		now:    time.Now,
	}
}

// RuleFor determines the effective rule for the provided endpoint and identity type.
func (l *Limiter) RuleFor(endpoint string, identityType IdentityType) Rule {
	window := l.cfg.Window()
	limit := l.cfg.DefaultLimit
	burst := l.cfg.DefaultBurst

	if identityType == IdentityAnonymous {
		limit = l.cfg.AnonymousLimit
		burst = l.cfg.AnonymousBurst
	}

	if override, ok := l.cfg.EndpointOverrides[endpoint]; ok {
		if override.WindowSeconds > 0 {
			window = time.Duration(override.WindowSeconds) * time.Second
		}
		if identityType == IdentityAnonymous {
			if override.AnonymousLimit > 0 {
				limit = override.AnonymousLimit
			}
			if override.AnonymousBurst >= 0 {
				burst = override.AnonymousBurst
			}
		} else {
			if override.AuthenticatedLimit > 0 {
				limit = override.AuthenticatedLimit
			}
			if override.AuthenticatedBurst >= 0 {
				burst = override.AuthenticatedBurst
			}
		}
	}

	if limit <= 0 {
		return Rule{Limit: 0, Burst: burst, Window: window}
	}

	if burst < 0 {
		burst = 0
	}

	return Rule{Limit: limit, Burst: burst, Window: window}
}

// Allow determines whether the request should be allowed for the provided key.
func (l *Limiter) Allow(ctx context.Context, endpointKey, identityKey string, rule Rule, identityType IdentityType) (Result, error) {
	if !l.cfg.Enabled || rule.Limit <= 0 {
		return Result{
			Allowed:      true,
			Remaining:    rule.Limit,
			Limit:        rule.Limit,
			Window:       rule.Window,
			ResetAfter:   0,
			RetryAfter:   0,
			IdentityKey:  identityKey,
			EndpointKey:  endpointKey,
			IdentityType: identityType,
		}, nil
	}

	if rule.Window <= 0 {
		rule.Window = l.cfg.Window()
	}

	key := fmt.Sprintf("%s:%s:%s", l.cfg.RedisPrefix, endpointKey, identityKey)

	now := l.now().UnixMilli()
	windowMillis := rule.Window.Milliseconds()
	if windowMillis <= 0 {
		windowMillis = int64(time.Minute / time.Millisecond)
	}

	refillRate := float64(rule.Limit) / float64(windowMillis)
	if refillRate <= 0 {
		refillRate = 1.0 / float64(windowMillis)
	}

	capacity := float64(rule.Limit + rule.Burst)
	if capacity < 1 {
		capacity = float64(rule.Limit)
	}
	if capacity < 1 {
		capacity = 1
	}

	ttl := windowMillis * 2
	if ttl <= 0 {
		ttl = windowMillis
	}

	cmd := l.script.Run(ctx, l.client, []string{key}, now, formatFloat(refillRate), formatFloat(capacity), ttl)
	raw, err := cmd.Result()
	if err != nil {
		return Result{}, err
	}

	values, ok := raw.([]interface{})
	if !ok || len(values) != 3 {
		return Result{}, errors.New("unexpected script response")
	}

	allowed := toInt(values[0])
	remainingTokens := toFloat(values[1])
	retryAfterMillis := toInt(values[2])

	result := Result{
		Allowed:      allowed == 1,
		Remaining:    int(math.Max(0, math.Floor(remainingTokens))),
		RetryAfter:   time.Duration(retryAfterMillis) * time.Millisecond,
		Limit:        rule.Limit,
		Window:       rule.Window,
		ResetAfter:   time.Duration(retryAfterMillis) * time.Millisecond,
		IdentityKey:  identityKey,
		EndpointKey:  endpointKey,
		IdentityType: identityType,
	}

	if result.Allowed {
		missing := capacity - remainingTokens
		if missing < 0 {
			missing = 0
		}
		resetMillis := missing / refillRate
		if resetMillis < 0 {
			resetMillis = 0
		}
		result.ResetAfter = time.Duration(int(math.Ceil(resetMillis))) * time.Millisecond
		result.RetryAfter = 0
	}

	return result, nil
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 10, 64)
}

func toInt(value interface{}) int {
	switch v := value.(type) {
	case int64:
		return int(v)
	case int:
		return v
	case string:
		i, _ := strconv.Atoi(v)
		return i
	case float64:
		return int(v)
	default:
		return 0
	}
}

func toFloat(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	case int:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

// WithNow overrides the time source (useful for tests).
func (l *Limiter) WithNow(now func() time.Time) {
	l.now = now
}
