package verification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"go.uber.org/zap"
)

// ResilientHTTPClient wraps HTTP client with circuit breaker and retry logic
// for external verification provider API calls (Checkr, Onfido, Sterling)
type ResilientHTTPClient struct {
	client  *http.Client
	breaker *resilience.CircuitBreaker
	retry   resilience.RetryConfig
	name    string
}

// NewResilientHTTPClient creates a new resilient HTTP client for verification providers
func NewResilientHTTPClient(name string, timeout time.Duration) *ResilientHTTPClient {
	breakerSettings := resilience.Settings{
		Name:             fmt.Sprintf("verification-%s", name),
		Interval:         60 * time.Second,
		Timeout:          30 * time.Second,
		FailureThreshold: 5,
		SuccessThreshold: 2,
	}

	breaker := resilience.NewCircuitBreaker(breakerSettings, func(ctx context.Context, err error) (interface{}, error) {
		logger.Get().Error("Verification circuit breaker open",
			zap.String("provider", name),
			zap.Error(err),
		)
		return nil, err
	})

	// Use conservative retry for verification APIs (they are not idempotent)
	retryConfig := resilience.ConservativeRetryConfig()
	retryConfig.RetryableChecker = isVerificationRetryable

	return &ResilientHTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		breaker: breaker,
		retry:   retryConfig,
		name:    name,
	}
}

// NewResilientHTTPClientWithBreaker creates a resilient client with a custom breaker
func NewResilientHTTPClientWithBreaker(name string, timeout time.Duration, breaker *resilience.CircuitBreaker) *ResilientHTTPClient {
	retryConfig := resilience.ConservativeRetryConfig()
	retryConfig.RetryableChecker = isVerificationRetryable

	return &ResilientHTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		breaker: breaker,
		retry:   retryConfig,
		name:    name,
	}
}

// DoRequest executes an HTTP request with circuit breaker and retry protection
func (r *ResilientHTTPClient) DoRequest(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	operationName := fmt.Sprintf("%s-%s", r.name, req.Method)

	result, err := resilience.RetryWithName(ctx, r.retry, func(ctx context.Context) (interface{}, error) {
		return r.breaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			// Clone the request for retry safety
			reqClone := req.Clone(ctx)
			if req.Body != nil {
				bodyBytes, _ := io.ReadAll(req.Body)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				reqClone.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			resp, err := r.client.Do(reqClone)
			if err != nil {
				return nil, err
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, err
			}

			// Check for retryable HTTP status codes
			if resilience.IsRetryableHTTPStatus(resp.StatusCode) {
				return nil, &httpError{
					statusCode: resp.StatusCode,
					body:       string(body),
				}
			}

			return &httpResult{
				response: resp,
				body:     body,
			}, nil
		})
	}, operationName)

	if err != nil {
		logger.Get().Error("Verification API request failed",
			zap.String("provider", r.name),
			zap.String("method", req.Method),
			zap.String("url", req.URL.String()),
			zap.Error(err),
		)
		return nil, nil, err
	}

	hr := result.(*httpResult)
	return hr.response, hr.body, nil
}

// Post performs a POST request with JSON body
func (r *ResilientHTTPClient) Post(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, respBody, err := r.DoRequest(ctx, req)
	if err != nil {
		return nil, 0, err
	}

	return respBody, resp.StatusCode, nil
}

// Get performs a GET request
func (r *ResilientHTTPClient) Get(ctx context.Context, url string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, respBody, err := r.DoRequest(ctx, req)
	if err != nil {
		return nil, 0, err
	}

	return respBody, resp.StatusCode, nil
}

// Allow reports whether the circuit breaker would allow a request
func (r *ResilientHTTPClient) Allow() bool {
	return r.breaker.Allow()
}

// httpResult holds successful HTTP response data
type httpResult struct {
	response *http.Response
	body     []byte
}

// httpError represents an HTTP error that might be retryable
type httpError struct {
	statusCode int
	body       string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.statusCode, e.body)
}

// isVerificationRetryable determines if an error from verification APIs should be retried
func isVerificationRetryable(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Retry on transient errors
	retryablePatterns := []string{
		"timeout",
		"connection",
		"network",
		"temporary",
		"503",
		"502",
		"504",
		"429", // rate limited
		"service unavailable",
		"bad gateway",
		"gateway timeout",
		"too many requests",
		"econnrefused",
		"econnreset",
		"etimedout",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	// Don't retry on permanent errors
	nonRetryablePatterns := []string{
		"400",
		"401",
		"403",
		"404",
		"invalid",
		"unauthorized",
		"forbidden",
		"not found",
		"bad request",
		"unprocessable",
	}

	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return false
		}
	}

	// Default to retry for unknown errors (network issues, etc.)
	return true
}
