package config

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/richxcame/ride-hailing/pkg/secrets"
)

// Config holds all application configuration
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	JWT           JWTConfig
	PubSub        PubSubConfig
	Firebase      FirebaseConfig
	Payments      PaymentsConfig
	Notifications NotificationsConfig
	RateLimit     RateLimitConfig
	Resilience    ResilienceConfig
	Timeout       TimeoutConfig
	Secrets       SecretsSettings
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
	Host        string
	Port        string
	User        string
	Password    string
	DBName      string
	SSLMode     string
	MaxConns    int
	MinConns    int
	ServiceName string
	Breaker     DatabaseBreakerConfig
}

// DatabaseBreakerConfig guards database connectivity when upstream issues occur.
type DatabaseBreakerConfig struct {
	Enabled          bool
	FailureThreshold int
	SuccessThreshold int
	TimeoutSeconds   int
	IntervalSeconds  int
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
	Secret         string
	Expiration     int // in hours
	KeyFile        string
	RotationHours  int
	GraceHours     int
	RefreshMinutes int
	VaultAddress   string
	VaultToken     string
	VaultPath      string
	VaultNamespace string
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
	CredentialsJSON string
	Enabled         bool
}

// PaymentsConfig holds payment provider configuration.
type PaymentsConfig struct {
	StripeAPIKey string
}

// NotificationsConfig stores third-party notification credentials.
type NotificationsConfig struct {
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string
	SMTPHost         string
	SMTPPort         string
	SMTPUsername     string
	SMTPPassword     string
	SMTPFromEmail    string
	SMTPFromName     string
}

// SecretsSettings configures the optional secrets manager.
type SecretsSettings struct {
	Provider        secrets.ProviderType
	CacheTTLSeconds int
	RotationDays    int
	AuditEnabled    bool
	Vault           secrets.VaultConfig
	AWS             secrets.AWSConfig
	GCP             secrets.GCPConfig
	Kubernetes      secrets.KubernetesConfig
	References      SecretReferences

	manager secrets.Manager
}

// SecretReferences list the secret paths consumed by the platform.
type SecretReferences struct {
	Database *secrets.Reference
	Stripe   *secrets.Reference
	Twilio   *secrets.Reference
	SMTP     *secrets.Reference
	Firebase *secrets.Reference
	JWTKeys  *secrets.Reference
}

// Manager returns the active secrets manager instance.
func (s *SecretsSettings) Manager() secrets.Manager {
	return s.manager
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

const (
	DefaultHTTPClientTimeout          = 30
	DefaultDatabaseQueryTimeout       = 10
	DefaultRedisOperationTimeout      = 5
	DefaultRedisReadTimeout           = 5
	DefaultRedisWriteTimeout          = 5
	DefaultWebSocketConnectionTimeout = 60
	DefaultRequestTimeout             = 30

	// Maximum allowed timeouts (prevent misconfigurations)
	MaxHTTPClientTimeout          = 300 // 5 minutes
	MaxDatabaseQueryTimeout       = 60  // 1 minute
	MaxRedisOperationTimeout      = 30  // 30 seconds
	MaxWebSocketConnectionTimeout = 300 // 5 minutes
	MaxRequestTimeout             = 300 // 5 minutes
)

// TimeoutConfig holds timeout configuration for various operations
type TimeoutConfig struct {
	HTTPClientTimeout         int
	DatabaseQueryTimeout     int
	RedisOperationTimeout    int
	RedisReadTimeout         int
	RedisWriteTimeout        int
	WebSocketConnectionTimeout int
	DefaultRequestTimeout    int
	RouteOverrides           map[string]int // Route pattern -> timeout in seconds (e.g., "POST:/api/v1/rides" -> 60)
}

func (t TimeoutConfig) HTTPClientTimeoutDuration() time.Duration {
	return time.Duration(t.HTTPClientTimeout) * time.Second
}

func (t TimeoutConfig) DatabaseQueryTimeoutDuration() time.Duration {
	return time.Duration(t.DatabaseQueryTimeout) * time.Second
}

func (t TimeoutConfig) RedisOperationTimeoutDuration() time.Duration {
	return time.Duration(t.RedisOperationTimeout) * time.Second
}

func (t TimeoutConfig) RedisReadTimeoutDuration() time.Duration {
	if t.RedisReadTimeout > 0 {
		return time.Duration(t.RedisReadTimeout) * time.Second
	}
	return t.RedisOperationTimeoutDuration()
}

func (t TimeoutConfig) RedisWriteTimeoutDuration() time.Duration {
	if t.RedisWriteTimeout > 0 {
		return time.Duration(t.RedisWriteTimeout) * time.Second
	}
	return t.RedisOperationTimeoutDuration()
}

func DefaultRedisReadTimeoutDuration() time.Duration {
	return time.Duration(DefaultRedisReadTimeout) * time.Second
}

func DefaultRedisWriteTimeoutDuration() time.Duration {
	return time.Duration(DefaultRedisWriteTimeout) * time.Second
}

func DefaultHTTPClientTimeoutDuration() time.Duration {
	return time.Duration(DefaultHTTPClientTimeout) * time.Second
}

func (t TimeoutConfig) WebSocketConnectionTimeoutDuration() time.Duration {
	return time.Duration(t.WebSocketConnectionTimeout) * time.Second
}

func (t TimeoutConfig) DefaultRequestTimeoutDuration() time.Duration {
	return time.Duration(t.DefaultRequestTimeout) * time.Second
}

// TimeoutForRoute returns the timeout duration for a specific route
// Route format: "METHOD:/path" (e.g., "POST:/api/v1/rides")
// Returns the route-specific timeout if found, otherwise returns the default timeout
func (t TimeoutConfig) TimeoutForRoute(method, path string) time.Duration {
	if t.RouteOverrides == nil {
		return t.DefaultRequestTimeoutDuration()
	}

	routeKey := fmt.Sprintf("%s:%s", method, path)
	if timeoutSeconds, ok := t.RouteOverrides[routeKey]; ok && timeoutSeconds > 0 {
		return time.Duration(timeoutSeconds) * time.Second
	}

	return t.DefaultRequestTimeoutDuration()
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
			Host:        getEnv("DB_HOST", "localhost"),
			Port:        getEnv("DB_PORT", "5432"),
			User:        getEnv("DB_USER", "postgres"),
			Password:    getEnv("DB_PASSWORD", "postgres"),
			DBName:      getEnv("DB_NAME", "ridehailing"),
			SSLMode:     getEnv("DB_SSLMODE", "disable"),
			MaxConns:    getEnvAsInt("DB_MAX_CONNS", 25),
			MinConns:    getEnvAsInt("DB_MIN_CONNS", 5),
			ServiceName: serviceName,
			Breaker: DatabaseBreakerConfig{
				Enabled:          getEnvAsBool("DB_BREAKER_ENABLED", false),
				FailureThreshold: getEnvAsInt("DB_BREAKER_FAILURE_THRESHOLD", 5),
				SuccessThreshold: getEnvAsInt("DB_BREAKER_SUCCESS_THRESHOLD", 1),
				TimeoutSeconds:   getEnvAsInt("DB_BREAKER_TIMEOUT_SECONDS", 30),
				IntervalSeconds:  getEnvAsInt("DB_BREAKER_INTERVAL_SECONDS", 60),
			},
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:         getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			Expiration:     getEnvAsInt("JWT_EXPIRATION", 24),
			KeyFile:        getEnv("JWT_KEYS_FILE", "config/jwt_keys.json"),
			RotationHours:  getEnvAsInt("JWT_ROTATION_HOURS", 24*30),
			GraceHours:     getEnvAsInt("JWT_ROTATION_GRACE_HOURS", 24*30),
			RefreshMinutes: getEnvAsInt("JWT_KEY_REFRESH_MINUTES", 5),
			VaultAddress:   getEnv("JWT_KEYS_VAULT_ADDR", ""),
			VaultToken:     getEnv("JWT_KEYS_VAULT_TOKEN", ""),
			VaultPath:      getEnv("JWT_KEYS_VAULT_PATH", ""),
			VaultNamespace: getEnv("JWT_KEYS_VAULT_NAMESPACE", ""),
		},
		PubSub: PubSubConfig{
			ProjectID: getEnv("PUBSUB_PROJECT_ID", ""),
			Enabled:   getEnvAsBool("PUBSUB_ENABLED", false),
		},
		Firebase: FirebaseConfig{
			ProjectID:       getEnv("FIREBASE_PROJECT_ID", ""),
			CredentialsPath: getEnv("FIREBASE_CREDENTIALS_PATH", ""),
			CredentialsJSON: getEnv("FIREBASE_CREDENTIALS_JSON", ""),
			Enabled:         getEnvAsBool("FIREBASE_ENABLED", false),
		},
		Payments: PaymentsConfig{
			StripeAPIKey: getEnv("STRIPE_API_KEY", ""),
		},
		Notifications: NotificationsConfig{
			TwilioAccountSID: getEnv("TWILIO_ACCOUNT_SID", ""),
			TwilioAuthToken:  getEnv("TWILIO_AUTH_TOKEN", ""),
			TwilioFromNumber: getEnv("TWILIO_FROM_NUMBER", ""),
			SMTPHost:         getEnv("SMTP_HOST", ""),
			SMTPPort:         getEnv("SMTP_PORT", ""),
			SMTPUsername:     getEnv("SMTP_USERNAME", ""),
			SMTPPassword:     getEnv("SMTP_PASSWORD", ""),
			SMTPFromEmail:    getEnv("SMTP_FROM_EMAIL", ""),
			SMTPFromName:     getEnv("SMTP_FROM_NAME", ""),
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
		Timeout: TimeoutConfig{
			HTTPClientTimeout:         getEnvAsInt("HTTP_CLIENT_TIMEOUT", DefaultHTTPClientTimeout),
			DatabaseQueryTimeout:      getEnvAsInt("DB_QUERY_TIMEOUT", DefaultDatabaseQueryTimeout),
			RedisOperationTimeout:      getEnvAsInt("REDIS_OPERATION_TIMEOUT", DefaultRedisOperationTimeout),
			RedisReadTimeout:          getEnvAsInt("REDIS_READ_TIMEOUT", DefaultRedisReadTimeout),
			RedisWriteTimeout:         getEnvAsInt("REDIS_WRITE_TIMEOUT", DefaultRedisWriteTimeout),
			WebSocketConnectionTimeout: getEnvAsInt("WS_CONNECTION_TIMEOUT", DefaultWebSocketConnectionTimeout),
			DefaultRequestTimeout:      getEnvAsInt("DEFAULT_REQUEST_TIMEOUT", DefaultRequestTimeout),
			RouteOverrides:             make(map[string]int),
		},
		Secrets: SecretsSettings{
			Provider:        secrets.ProviderType(getEnv("SECRETS_PROVIDER", "")),
			CacheTTLSeconds: getEnvAsInt("SECRETS_CACHE_TTL_SECONDS", 300),
			RotationDays:    getEnvAsInt("SECRETS_ROTATION_DAYS", 90),
			AuditEnabled:    getEnvAsBool("SECRETS_AUDIT_ENABLED", true),
			Vault: secrets.VaultConfig{
				Address:       getEnv("SECRETS_VAULT_ADDR", ""),
				Token:         getEnv("SECRETS_VAULT_TOKEN", ""),
				Namespace:     getEnv("SECRETS_VAULT_NAMESPACE", ""),
				MountPath:     getEnv("SECRETS_VAULT_MOUNT", "secret"),
				CACert:        getEnv("SECRETS_VAULT_CACERT", ""),
				CAPath:        getEnv("SECRETS_VAULT_CAPATH", ""),
				ClientCert:    getEnv("SECRETS_VAULT_CLIENT_CERT", ""),
				ClientKey:     getEnv("SECRETS_VAULT_CLIENT_KEY", ""),
				TLSSkipVerify: getEnvAsBool("SECRETS_VAULT_TLS_SKIP_VERIFY", false),
			},
			AWS: secrets.AWSConfig{
				Region:          getEnv("SECRETS_AWS_REGION", ""),
				AccessKeyID:     getEnv("SECRETS_AWS_ACCESS_KEY_ID", ""),
				SecretAccessKey: getEnv("SECRETS_AWS_SECRET_ACCESS_KEY", ""),
				SessionToken:    getEnv("SECRETS_AWS_SESSION_TOKEN", ""),
				Profile:         getEnv("SECRETS_AWS_PROFILE", ""),
				Endpoint:        getEnv("SECRETS_AWS_ENDPOINT", ""),
			},
			GCP: secrets.GCPConfig{
				ProjectID:       getEnv("SECRETS_GCP_PROJECT_ID", ""),
				CredentialsJSON: getEnv("SECRETS_GCP_CREDENTIALS_JSON", ""),
				CredentialsFile: getEnv("SECRETS_GCP_CREDENTIALS_FILE", ""),
			},
			Kubernetes: secrets.KubernetesConfig{
				BasePath: getEnv("SECRETS_K8S_BASE_PATH", "/var/run/secrets/ride-hailing"),
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

	if timeoutOverrides := getEnv("ROUTE_TIMEOUT_OVERRIDES", ""); timeoutOverrides != "" {
		var routeTimeouts map[string]int
		if err := json.Unmarshal([]byte(timeoutOverrides), &routeTimeouts); err != nil {
			return nil, fmt.Errorf("invalid ROUTE_TIMEOUT_OVERRIDES value: %w", err)
		}
		for route, timeout := range routeTimeouts {
			if timeout <= 0 {
				delete(routeTimeouts, route)
			}
		}
		cfg.Timeout.RouteOverrides = routeTimeouts
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

	// Validate and set HTTP client timeout
	if cfg.Timeout.HTTPClientTimeout <= 0 {
		cfg.Timeout.HTTPClientTimeout = DefaultHTTPClientTimeout
	} else if cfg.Timeout.HTTPClientTimeout > MaxHTTPClientTimeout {
		return nil, fmt.Errorf("HTTP_CLIENT_TIMEOUT (%d seconds) exceeds maximum allowed value of %d seconds", cfg.Timeout.HTTPClientTimeout, MaxHTTPClientTimeout)
	}

	// Validate and set database query timeout
	if cfg.Timeout.DatabaseQueryTimeout <= 0 {
		cfg.Timeout.DatabaseQueryTimeout = DefaultDatabaseQueryTimeout
	} else if cfg.Timeout.DatabaseQueryTimeout > MaxDatabaseQueryTimeout {
		return nil, fmt.Errorf("DB_QUERY_TIMEOUT (%d seconds) exceeds maximum allowed value of %d seconds", cfg.Timeout.DatabaseQueryTimeout, MaxDatabaseQueryTimeout)
	}

	// Validate and set Redis operation timeout
	if cfg.Timeout.RedisOperationTimeout <= 0 {
		cfg.Timeout.RedisOperationTimeout = DefaultRedisOperationTimeout
	} else if cfg.Timeout.RedisOperationTimeout > MaxRedisOperationTimeout {
		return nil, fmt.Errorf("REDIS_OPERATION_TIMEOUT (%d seconds) exceeds maximum allowed value of %d seconds", cfg.Timeout.RedisOperationTimeout, MaxRedisOperationTimeout)
	}

	// Validate and set Redis read timeout
	if cfg.Timeout.RedisReadTimeout <= 0 {
		cfg.Timeout.RedisReadTimeout = DefaultRedisReadTimeout
	} else if cfg.Timeout.RedisReadTimeout > MaxRedisOperationTimeout {
		return nil, fmt.Errorf("REDIS_READ_TIMEOUT (%d seconds) exceeds maximum allowed value of %d seconds", cfg.Timeout.RedisReadTimeout, MaxRedisOperationTimeout)
	}

	// Validate and set Redis write timeout
	if cfg.Timeout.RedisWriteTimeout <= 0 {
		cfg.Timeout.RedisWriteTimeout = DefaultRedisWriteTimeout
	} else if cfg.Timeout.RedisWriteTimeout > MaxRedisOperationTimeout {
		return nil, fmt.Errorf("REDIS_WRITE_TIMEOUT (%d seconds) exceeds maximum allowed value of %d seconds", cfg.Timeout.RedisWriteTimeout, MaxRedisOperationTimeout)
	}

	// Validate and set WebSocket connection timeout
	if cfg.Timeout.WebSocketConnectionTimeout <= 0 {
		cfg.Timeout.WebSocketConnectionTimeout = DefaultWebSocketConnectionTimeout
	} else if cfg.Timeout.WebSocketConnectionTimeout > MaxWebSocketConnectionTimeout {
		return nil, fmt.Errorf("WS_CONNECTION_TIMEOUT (%d seconds) exceeds maximum allowed value of %d seconds", cfg.Timeout.WebSocketConnectionTimeout, MaxWebSocketConnectionTimeout)
	}

	// Validate and set default request timeout
	if cfg.Timeout.DefaultRequestTimeout <= 0 {
		cfg.Timeout.DefaultRequestTimeout = DefaultRequestTimeout
	} else if cfg.Timeout.DefaultRequestTimeout > MaxRequestTimeout {
		return nil, fmt.Errorf("DEFAULT_REQUEST_TIMEOUT (%d seconds) exceeds maximum allowed value of %d seconds", cfg.Timeout.DefaultRequestTimeout, MaxRequestTimeout)
	}

	// Validate route-specific timeout overrides
	for route, timeout := range cfg.Timeout.RouteOverrides {
		if timeout > MaxRequestTimeout {
			return nil, fmt.Errorf("route timeout for '%s' (%d seconds) exceeds maximum allowed value of %d seconds", route, timeout, MaxRequestTimeout)
		}
	}

	if err := cfg.populateSecretReferences(); err != nil {
		return nil, err
	}

	if err := cfg.initializeSecrets(); err != nil {
		return nil, err
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

// Close releases resources associated with the configuration (e.g., secret manager clients).
func (c *Config) Close() error {
	if c == nil || c.Secrets.manager == nil {
		return nil
	}
	return c.Secrets.manager.Close()
}

func (c *Config) populateSecretReferences() error {
	var err error
	refs := SecretReferences{}

	if refs.Database, err = parseSecretReference("database_credentials", secrets.SecretDatabase, getEnv("SECRETS_DB_PATH", "")); err != nil {
		return fmt.Errorf("invalid SECRETS_DB_PATH: %w", err)
	}
	if refs.Stripe, err = parseSecretReference("stripe_api_key", secrets.SecretStripe, getEnv("SECRETS_STRIPE_PATH", "")); err != nil {
		return fmt.Errorf("invalid SECRETS_STRIPE_PATH: %w", err)
	}
	if refs.Twilio, err = parseSecretReference("twilio_credentials", secrets.SecretTwilio, getEnv("SECRETS_TWILIO_PATH", "")); err != nil {
		return fmt.Errorf("invalid SECRETS_TWILIO_PATH: %w", err)
	}
	if refs.SMTP, err = parseSecretReference("smtp_credentials", secrets.SecretSMTP, getEnv("SECRETS_SMTP_PATH", "")); err != nil {
		return fmt.Errorf("invalid SECRETS_SMTP_PATH: %w", err)
	}
	if refs.Firebase, err = parseSecretReference("firebase_credentials", secrets.SecretFirebase, getEnv("SECRETS_FIREBASE_PATH", "")); err != nil {
		return fmt.Errorf("invalid SECRETS_FIREBASE_PATH: %w", err)
	}
	if refs.JWTKeys, err = parseSecretReference("jwt_signing_keys", secrets.SecretJWTKeys, getEnv("SECRETS_JWT_KEYS_PATH", "")); err != nil {
		return fmt.Errorf("invalid SECRETS_JWT_KEYS_PATH: %w", err)
	}

	c.Secrets.References = refs
	return nil
}

func (c *Config) initializeSecrets() error {
	if c.Secrets.Provider == secrets.ProviderNone || c.Secrets.Provider == "" {
		return nil
	}

	cacheTTL := time.Duration(c.Secrets.CacheTTLSeconds) * time.Second
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}

	rotation := time.Duration(c.Secrets.RotationDays) * 24 * time.Hour
	if rotation <= 0 {
		rotation = 90 * 24 * time.Hour
	}

	manager, err := secrets.NewManager(secrets.Config{
		Provider:         c.Secrets.Provider,
		CacheTTL:         cacheTTL,
		RotationInterval: rotation,
		AuditEnabled:     c.Secrets.AuditEnabled,
		Vault:            c.Secrets.Vault,
		AWS:              c.Secrets.AWS,
		GCP:              c.Secrets.GCP,
		Kubernetes:       c.Secrets.Kubernetes,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize secrets manager: %w", err)
	}

	c.Secrets.manager = manager

	if err := c.applySecretOverrides(context.Background()); err != nil {
		return err
	}

	return nil
}

func (c *Config) applySecretOverrides(ctx context.Context) error {
	if c.Secrets.manager == nil {
		return nil
	}

	if ref := c.Secrets.References.Database; ref != nil {
		secret, err := c.Secrets.manager.GetSecret(ctx, *ref)
		if err != nil {
			return fmt.Errorf("fetch database secret: %w", err)
		}
		overrideString(&c.Database.Host, secret.Data["host"])
		overrideString(&c.Database.Port, secret.Data["port"])
		overrideString(&c.Database.User, firstNonEmpty(secret.Data["username"], secret.Data["user"]))
		overrideString(&c.Database.Password, secret.Data["password"])
		overrideString(&c.Database.DBName, firstNonEmpty(secret.Data["dbname"], secret.Data["database"]))
		overrideString(&c.Database.SSLMode, secret.Data["sslmode"])
	}

	if ref := c.Secrets.References.Stripe; ref != nil {
		secret, err := c.Secrets.manager.GetSecret(ctx, *ref)
		if err != nil {
			return fmt.Errorf("fetch stripe secret: %w", err)
		}
		if apiKey := firstNonEmpty(secret.Data["api_key"], secret.Data["key"], secret.Data["stripe_api_key"]); apiKey != "" {
			c.Payments.StripeAPIKey = apiKey
		}
	}

	if ref := c.Secrets.References.Twilio; ref != nil {
		secret, err := c.Secrets.manager.GetSecret(ctx, *ref)
		if err != nil {
			return fmt.Errorf("fetch twilio secret: %w", err)
		}
		overrideString(&c.Notifications.TwilioAccountSID, firstNonEmpty(secret.Data["account_sid"], secret.Data["sid"]))
		overrideString(&c.Notifications.TwilioAuthToken, firstNonEmpty(secret.Data["auth_token"], secret.Data["token"]))
		overrideString(&c.Notifications.TwilioFromNumber, firstNonEmpty(secret.Data["from_number"], secret.Data["phone_number"]))
	}

	if ref := c.Secrets.References.SMTP; ref != nil {
		secret, err := c.Secrets.manager.GetSecret(ctx, *ref)
		if err != nil {
			return fmt.Errorf("fetch smtp secret: %w", err)
		}
		overrideString(&c.Notifications.SMTPHost, secret.Data["host"])
		overrideString(&c.Notifications.SMTPPort, secret.Data["port"])
		overrideString(&c.Notifications.SMTPUsername, firstNonEmpty(secret.Data["username"], secret.Data["user"]))
		overrideString(&c.Notifications.SMTPPassword, secret.Data["password"])
		overrideString(&c.Notifications.SMTPFromEmail, secret.Data["from_email"])
		overrideString(&c.Notifications.SMTPFromName, secret.Data["from_name"])
	}

	if ref := c.Secrets.References.Firebase; ref != nil {
		secret, err := c.Secrets.manager.GetSecret(ctx, *ref)
		if err != nil {
			return fmt.Errorf("fetch firebase secret: %w", err)
		}

		if raw := firstNonEmpty(secret.Data["credentials_json"], secret.Data["service_account"]); raw != "" {
			c.Firebase.CredentialsJSON = raw
		} else if encoded := secret.Data["credentials_b64"]; encoded != "" {
			decoded, decodeErr := base64.StdEncoding.DecodeString(encoded)
			if decodeErr != nil {
				return fmt.Errorf("decode firebase credentials_b64: %w", decodeErr)
			}
			c.Firebase.CredentialsJSON = string(decoded)
		}

		overrideString(&c.Firebase.CredentialsPath, secret.Data["credentials_path"])
	}

	if ref := c.Secrets.References.JWTKeys; ref != nil {
		mount := ref.Mount
		if mount == "" {
			mount = c.Secrets.Vault.MountPath
		}
		mount = strings.Trim(mount, "/")
		path := strings.Trim(ref.Path, "/")
		if c.JWT.VaultPath == "" && path != "" {
			if mount != "" {
				c.JWT.VaultPath = fmt.Sprintf("%s/%s", mount, path)
			} else {
				c.JWT.VaultPath = path
			}
		}
		if c.JWT.VaultAddress == "" {
			c.JWT.VaultAddress = c.Secrets.Vault.Address
		}
		if c.JWT.VaultToken == "" {
			c.JWT.VaultToken = c.Secrets.Vault.Token
		}
		if c.JWT.VaultNamespace == "" {
			c.JWT.VaultNamespace = c.Secrets.Vault.Namespace
		}
	}

	return nil
}

func parseSecretReference(name string, secretType secrets.SecretType, raw string) (*secrets.Reference, error) {
	if raw == "" {
		return nil, nil
	}
	ref, err := secrets.ParseReference(name, secretType, raw)
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func overrideString(target *string, candidate string) {
	if candidate != "" && target != nil {
		*target = candidate
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
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
