package notifications

import (
	"context"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"go.uber.org/zap"
)

// ResilientFirebaseClient wraps FirebaseClient with circuit breaker and retry logic
type ResilientFirebaseClient struct {
	client  FirebaseClientInterface
	breaker *resilience.CircuitBreaker
	retry   resilience.RetryConfig
}

// NewResilientFirebaseClient creates a new resilient Firebase client
func NewResilientFirebaseClient(credentialsPath string, breaker *resilience.CircuitBreaker) (*ResilientFirebaseClient, error) {
	client, err := NewFirebaseClient(credentialsPath)
	if err != nil {
		return nil, err
	}

	return NewResilientFirebaseClientWithClient(client, breaker), nil
}

// NewResilientFirebaseClientFromJSON creates a resilient Firebase client from JSON credentials
func NewResilientFirebaseClientFromJSON(credentialsJSON []byte, breaker *resilience.CircuitBreaker) (*ResilientFirebaseClient, error) {
	client, err := NewFirebaseClientFromJSON(credentialsJSON)
	if err != nil {
		return nil, err
	}

	return NewResilientFirebaseClientWithClient(client, breaker), nil
}

// NewResilientFirebaseClientWithClient creates a resilient wrapper around an existing client
func NewResilientFirebaseClientWithClient(client FirebaseClientInterface, breaker *resilience.CircuitBreaker) *ResilientFirebaseClient {
	if breaker == nil {
		breakerSettings := resilience.Settings{
			Name:             "firebase-fcm",
			Interval:         60 * time.Second,
			Timeout:          30 * time.Second,
			FailureThreshold: 5,
			SuccessThreshold: 2,
		}

		breaker = resilience.NewCircuitBreaker(breakerSettings, func(ctx context.Context, err error) (interface{}, error) {
			logger.Get().Error("Firebase circuit breaker open, notification failed",
				zap.Error(err),
			)
			// For notifications, we can queue them for later retry or just log the failure
			return "", err
		})
	}

	// Configure retry with exponential backoff
	retryConfig := resilience.DefaultRetryConfig()
	retryConfig.MaxAttempts = 3
	retryConfig.InitialBackoff = 1 * time.Second
	retryConfig.MaxBackoff = 10 * time.Second
	retryConfig.RetryableChecker = isFirebaseRetryable

	return &ResilientFirebaseClient{
		client:  client,
		breaker: breaker,
		retry:   retryConfig,
	}
}

// SendPushNotification sends a push notification with retry and circuit breaker
func (r *ResilientFirebaseClient) SendPushNotification(ctx context.Context, token, title, body string, data map[string]string) (string, error) {
	result, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return r.client.SendPushNotification(ctx, token, title, body, data)
	})

	if err != nil {
		logger.Get().Error("Failed to send push notification after retries",
			zap.Error(err),
			zap.String("token", maskToken(token)),
			zap.String("title", title),
		)
		return "", err
	}

	logger.Get().Debug("Successfully sent push notification",
		zap.String("message_id", result.(string)),
		zap.String("title", title),
	)

	return result.(string), nil
}

// SendMulticastNotification sends notification to multiple devices with retry and circuit breaker
func (r *ResilientFirebaseClient) SendMulticastNotification(ctx context.Context, tokens []string, title, body string, data map[string]string) (*messaging.BatchResponse, error) {
	result, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return r.client.SendMulticastNotification(ctx, tokens, title, body, data)
	})

	if err != nil {
		logger.Get().Error("Failed to send multicast notification after retries",
			zap.Error(err),
			zap.Int("token_count", len(tokens)),
			zap.String("title", title),
		)
		return nil, err
	}

	batchResp := result.(*messaging.BatchResponse)
	logger.Get().Info("Successfully sent multicast notification",
		zap.Int("success_count", batchResp.SuccessCount),
		zap.Int("failure_count", batchResp.FailureCount),
		zap.String("title", title),
	)

	return batchResp, nil
}

// SendTopicNotification sends notification to a topic with retry and circuit breaker
func (r *ResilientFirebaseClient) SendTopicNotification(ctx context.Context, topic, title, body string, data map[string]string) (string, error) {
	result, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return r.client.SendTopicNotification(ctx, topic, title, body, data)
	})

	if err != nil {
		logger.Get().Error("Failed to send topic notification after retries",
			zap.Error(err),
			zap.String("topic", topic),
			zap.String("title", title),
		)
		return "", err
	}

	logger.Get().Debug("Successfully sent topic notification",
		zap.String("message_id", result.(string)),
		zap.String("topic", topic),
	)

	return result.(string), nil
}

// SubscribeToTopic subscribes tokens to a topic with retry and circuit breaker
func (r *ResilientFirebaseClient) SubscribeToTopic(ctx context.Context, tokens []string, topic string) error {
	_, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return nil, r.client.SubscribeToTopic(ctx, tokens, topic)
	})

	if err != nil {
		logger.Get().Error("Failed to subscribe to topic after retries",
			zap.Error(err),
			zap.String("topic", topic),
			zap.Int("token_count", len(tokens)),
		)
		return err
	}

	logger.Get().Debug("Successfully subscribed tokens to topic",
		zap.String("topic", topic),
		zap.Int("token_count", len(tokens)),
	)

	return nil
}

// UnsubscribeFromTopic unsubscribes tokens from a topic with retry and circuit breaker
func (r *ResilientFirebaseClient) UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) error {
	_, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return nil, r.client.UnsubscribeFromTopic(ctx, tokens, topic)
	})

	if err != nil {
		logger.Get().Error("Failed to unsubscribe from topic after retries",
			zap.Error(err),
			zap.String("topic", topic),
			zap.Int("token_count", len(tokens)),
		)
		return err
	}

	logger.Get().Debug("Successfully unsubscribed tokens from topic",
		zap.String("topic", topic),
		zap.Int("token_count", len(tokens)),
	)

	return nil
}

// isFirebaseRetryable determines if a Firebase error should be retried
func isFirebaseRetryable(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// Retry on specific Firebase error codes
	retryableMessages := []string{
		"unavailable",
		"internal",
		"deadline-exceeded",
		"resource-exhausted",
		"connection",
		"timeout",
		"temporarily unavailable",
	}

	for _, msg := range retryableMessages {
		if contains(errMsg, msg) {
			return true
		}
	}

	// Don't retry on invalid token errors
	nonRetryableMessages := []string{
		"invalid-argument",
		"invalid-registration-token",
		"registration-token-not-registered",
		"mismatched-credential",
		"invalid-credential",
	}

	for _, msg := range nonRetryableMessages {
		if contains(errMsg, msg) {
			return false
		}
	}

	// By default, retry network-related errors
	return true
}

// maskToken masks FCM token for logging (show only first/last 4 chars)
func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	// Simple substring search
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
