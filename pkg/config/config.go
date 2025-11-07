package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	PubSub     PubSubConfig
	Firebase   FirebaseConfig
	RateLimit  RateLimitConfig
	Resilience ResilienceConfig
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port         string
	Environment  string
	ServiceName  string
	ReadTimeout  int
	WriteTimeout int
	CORSOrigins  string // Comma-separated list of allowed origins
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int
	MinConns int
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool
	WindowSeconds     int
	DefaultLimit      int
	DefaultBurst      int
	AnonymousLimit    int
	AnonymousBurst    int
	RedisPrefix       string
	EndpointOverrides map[string]EndpointRateLimitConfig
}

// EndpointRateLimitConfig allows customizing limits per endpoint
type EndpointRateLimitConfig struct {
	AuthenticatedLimit int `json:"authenticated_limit"`
	AuthenticatedBurst int `json:"authenticated_burst"`
	AnonymousLimit     int `json:"anonymous_limit"`
	AnonymousBurst     int `json:"anonymous_burst"`
	WindowSeconds      int `json:"window_seconds"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret     string
	Expiration int // in hours
}

// PubSubConfig holds Google Pub/Sub configuration
type PubSubConfig struct {
	ProjectID string
	Enabled   bool
}

// FirebaseConfig holds Firebase configuration
type FirebaseConfig struct {
	ProjectID       string
	CredentialsPath string
	Enabled         bool
}

// ResilienceConfig groups runtime resilience controls
type ResilienceConfig struct {
	CircuitBreaker CircuitBreakerConfig
}

// CircuitBreakerConfig captures default and per-service breaker tuning
type CircuitBreakerConfig struct {
	Enabled          bool
	FailureThreshold int
	SuccessThreshold int
	TimeoutSeconds   int
	IntervalSeconds  int
	ServiceOverrides map[string]CircuitBreakerSettings
}

// CircuitBreakerSettings overrides defaults for a specific upstream service
type CircuitBreakerSettings struct {
	FailureThreshold int `json:"failure_threshold"`
	SuccessThreshold int `json:"success_threshold"`
	TimeoutSeconds   int `json:"timeout_seconds"`
	IntervalSeconds  int `json:"interval_seconds"`
}

// Load loads configuration from environment variables
func Load(serviceName string) (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			Environment:  getEnv("ENVIRONMENT", "development"),
			ServiceName:  serviceName,
			ReadTimeout:  getEnvAsInt("READ_TIMEOUT", 10),
			WriteTimeout: getEnvAsInt("WRITE_TIMEOUT", 10),
			CORSOrigins:  getEnv("CORS_ORIGINS", "http://localhost:3000"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "ridehailing"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			MaxConns: getEnvAsInt("DB_MAX_CONNS", 25),
			MinConns: getEnvAsInt("DB_MIN_CONNS", 5),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			Expiration: getEnvAsInt("JWT_EXPIRATION", 24),
		},
		PubSub: PubSubConfig{
			ProjectID: getEnv("PUBSUB_PROJECT_ID", ""),
			Enabled:   getEnvAsBool("PUBSUB_ENABLED", false),
		},
		Firebase: FirebaseConfig{
			ProjectID:       getEnv("FIREBASE_PROJECT_ID", ""),
			CredentialsPath: getEnv("FIREBASE_CREDENTIALS_PATH", ""),
			Enabled:         getEnvAsBool("FIREBASE_ENABLED", false),
		},
		RateLimit: RateLimitConfig{
			Enabled:        getEnvAsBool("RATE_LIMIT_ENABLED", false),
			WindowSeconds:  getEnvAsInt("RATE_LIMIT_WINDOW_SECONDS", 60),
			DefaultLimit:   getEnvAsInt("RATE_LIMIT_DEFAULT_LIMIT", 120),
			DefaultBurst:   getEnvAsInt("RATE_LIMIT_DEFAULT_BURST", 40),
			AnonymousLimit: getEnvAsInt("RATE_LIMIT_ANON_LIMIT", 60),
			AnonymousBurst: getEnvAsInt("RATE_LIMIT_ANON_BURST", 20),
			RedisPrefix:    getEnv("RATE_LIMIT_REDIS_PREFIX", "rate-limit"),
		},
		Resilience: ResilienceConfig{
			CircuitBreaker: CircuitBreakerConfig{
				Enabled:          getEnvAsBool("CB_ENABLED", false),
				FailureThreshold: getEnvAsInt("CB_FAILURE_THRESHOLD", 5),
				SuccessThreshold: getEnvAsInt("CB_SUCCESS_THRESHOLD", 1),
				TimeoutSeconds:   getEnvAsInt("CB_TIMEOUT_SECONDS", 30),
				IntervalSeconds:  getEnvAsInt("CB_INTERVAL_SECONDS", 60),
			},
		},
	}

	if overrides := getEnv("RATE_LIMIT_ENDPOINTS", ""); overrides != "" {
		var endpointConfig map[string]EndpointRateLimitConfig
		if err := json.Unmarshal([]byte(overrides), &endpointConfig); err != nil {
			return nil, fmt.Errorf("invalid RATE_LIMIT_ENDPOINTS value: %w", err)
		}
		cfg.RateLimit.EndpointOverrides = endpointConfig
	}

	if breakerOverrides := getEnv("CB_SERVICE_OVERRIDES", ""); breakerOverrides != "" {
		var serviceConfig map[string]CircuitBreakerSettings
		if err := json.Unmarshal([]byte(breakerOverrides), &serviceConfig); err != nil {
			return nil, fmt.Errorf("invalid CB_SERVICE_OVERRIDES value: %w", err)
		}
		cfg.Resilience.CircuitBreaker.ServiceOverrides = serviceConfig
	}

	if cfg.RateLimit.WindowSeconds <= 0 {
		cfg.RateLimit.WindowSeconds = int((time.Minute).Seconds())
	}

	if cfg.Resilience.CircuitBreaker.TimeoutSeconds <= 0 {
		cfg.Resilience.CircuitBreaker.TimeoutSeconds = 30
	}

	if cfg.Resilience.CircuitBreaker.IntervalSeconds <= 0 {
		cfg.Resilience.CircuitBreaker.IntervalSeconds = 60
	}

	if cfg.Resilience.CircuitBreaker.FailureThreshold <= 0 {
		cfg.Resilience.CircuitBreaker.FailureThreshold = 5
	}

	if cfg.Resilience.CircuitBreaker.SuccessThreshold <= 0 {
		cfg.Resilience.CircuitBreaker.SuccessThreshold = 1
	}

	return cfg, nil
}

// SettingsFor returns effective breaker settings for a specific upstream service name
func (c CircuitBreakerConfig) SettingsFor(service string) CircuitBreakerSettings {
	settings := CircuitBreakerSettings{
		FailureThreshold: c.FailureThreshold,
		SuccessThreshold: c.SuccessThreshold,
		TimeoutSeconds:   c.TimeoutSeconds,
		IntervalSeconds:  c.IntervalSeconds,
	}

	if c.ServiceOverrides != nil {
		if override, ok := c.ServiceOverrides[service]; ok {
			if override.FailureThreshold > 0 {
				settings.FailureThreshold = override.FailureThreshold
			}
			if override.SuccessThreshold > 0 {
				settings.SuccessThreshold = override.SuccessThreshold
			}
			if override.TimeoutSeconds > 0 {
				settings.TimeoutSeconds = override.TimeoutSeconds
			}
			if override.IntervalSeconds > 0 {
				settings.IntervalSeconds = override.IntervalSeconds
			}
		}
	}

	if settings.SuccessThreshold <= 0 {
		settings.SuccessThreshold = 1
	}
	if settings.FailureThreshold <= 0 {
		settings.FailureThreshold = 5
	}
	if settings.TimeoutSeconds <= 0 {
		settings.TimeoutSeconds = 30
	}
	if settings.IntervalSeconds <= 0 {
		settings.IntervalSeconds = 60
	}

	return settings
}

// DSN returns the database connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// RedisAddr returns the Redis address
func (c *RedisConfig) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// Window returns the configured rate limit window duration
func (c RateLimitConfig) Window() time.Duration {
	if c.WindowSeconds <= 0 {
		return time.Minute
	}
	return time.Duration(c.WindowSeconds) * time.Second
}
