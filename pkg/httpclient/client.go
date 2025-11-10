package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/resilience"
)

// Client wraps http.Client with convenience methods and retry support
type Client struct {
	httpClient  *http.Client
	baseURL     string
	retryConfig *resilience.RetryConfig
}

// Option configures the HTTP client
type Option func(*Client)

// WithRetry enables retry logic with the given configuration
func WithRetry(config resilience.RetryConfig) Option {
	return func(c *Client) {
		c.retryConfig = &config
	}
}

// WithDefaultRetry enables default retry configuration
func WithDefaultRetry() Option {
	config := resilience.DefaultRetryConfig()
	config.RetryableChecker = isHTTPRetryable
	return func(c *Client) {
		c.retryConfig = &config
	}
}

// NewClient creates a new HTTP client
func NewClient(baseURL string, timeout time.Duration, opts ...Option) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Post makes a POST request with JSON body
func (c *Client) Post(ctx context.Context, path string, body interface{}, headers map[string]string) ([]byte, error) {
	if c.retryConfig != nil {
		return c.postWithRetry(ctx, path, body, headers)
	}
	return c.doPost(ctx, path, body, headers)
}

// PostWithIdempotency makes a POST request with an idempotency key for safe retries
func (c *Client) PostWithIdempotency(ctx context.Context, path string, body interface{}, headers map[string]string, idempotencyKey string) ([]byte, error) {
	if headers == nil {
		headers = make(map[string]string)
	}
	// Add idempotency key if not already present
	if idempotencyKey != "" {
		headers["Idempotency-Key"] = idempotencyKey
	} else {
		// Generate a unique idempotency key if not provided
		headers["Idempotency-Key"] = uuid.New().String()
	}

	return c.Post(ctx, path, body, headers)
}

func (c *Client) postWithRetry(ctx context.Context, path string, body interface{}, headers map[string]string) ([]byte, error) {
	result, err := resilience.Retry(ctx, *c.retryConfig, func(ctx context.Context) (interface{}, error) {
		return c.doPost(ctx, path, body, headers)
	})

	if err != nil {
		return nil, err
	}

	return result.([]byte), nil
}

func (c *Client) doPost(ctx context.Context, path string, body interface{}, headers map[string]string) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	injectCorrelationID(ctx, req)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	return respBody, nil
}

// Get makes a GET request
func (c *Client) Get(ctx context.Context, path string, headers map[string]string) ([]byte, error) {
	if c.retryConfig != nil {
		return c.getWithRetry(ctx, path, headers)
	}
	return c.doGet(ctx, path, headers)
}

func (c *Client) getWithRetry(ctx context.Context, path string, headers map[string]string) ([]byte, error) {
	result, err := resilience.Retry(ctx, *c.retryConfig, func(ctx context.Context) (interface{}, error) {
		return c.doGet(ctx, path, headers)
	})

	if err != nil {
		return nil, err
	}

	return result.([]byte), nil
}

func (c *Client) doGet(ctx context.Context, path string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	injectCorrelationID(ctx, req)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	return respBody, nil
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// isHTTPRetryable determines if an HTTP error is retryable
func isHTTPRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's an HTTPError
	if httpErr, ok := err.(*HTTPError); ok {
		return resilience.IsRetryableHTTPStatus(httpErr.StatusCode)
	}

	// For other errors (network issues, timeouts), retry by default
	return true
}

func injectCorrelationID(ctx context.Context, req *http.Request) {
	if ctx == nil || req == nil {
		return
	}

	if correlationID := logger.CorrelationIDFromContext(ctx); correlationID != "" {
		req.Header.Set(middleware.CorrelationIDHeader, correlationID)
	}
}
