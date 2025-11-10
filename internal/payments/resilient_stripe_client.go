package payments

import (
	"context"
	"time"

	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"github.com/stripe/stripe-go/v83"
	"go.uber.org/zap"
)

// ResilientStripeClient wraps StripeClient with circuit breaker and retry logic
type ResilientStripeClient struct {
	client  StripeClientInterface
	breaker *resilience.CircuitBreaker
	retry   resilience.RetryConfig
}

// NewResilientStripeClient creates a new resilient Stripe client
func NewResilientStripeClient(apiKey string, breaker *resilience.CircuitBreaker) *ResilientStripeClient {
	if breaker == nil {
		breakerSettings := resilience.Settings{
			Name:             "stripe-api",
			Interval:         60 * time.Second,
			Timeout:          30 * time.Second,
			FailureThreshold: 5,
			SuccessThreshold: 2,
		}

		breaker = resilience.NewCircuitBreaker(breakerSettings, func(ctx context.Context, err error) (interface{}, error) {
			logger.Get().Error("Stripe circuit breaker open, payment operation failed",
				zap.Error(err),
			)
			return nil, common.NewServiceUnavailableError("payments are temporarily unavailable, please try again")
		})
	}

	// Configure retry with exponential backoff
	retryConfig := resilience.DefaultRetryConfig()
	retryConfig.MaxAttempts = 3
	retryConfig.InitialBackoff = 1 * time.Second
	retryConfig.MaxBackoff = 10 * time.Second
	retryConfig.RetryableChecker = isStripeRetryable

	return &ResilientStripeClient{
		client:  NewStripeClient(apiKey),
		breaker: breaker,
		retry:   retryConfig,
	}
}

// NewResilientStripeClientWithClient creates a resilient wrapper around an existing client (for testing)
func NewResilientStripeClientWithClient(client StripeClientInterface, breaker *resilience.CircuitBreaker) *ResilientStripeClient {
	if breaker == nil {
		breakerSettings := resilience.Settings{
			Name:             "stripe-api-test",
			Interval:         60 * time.Second,
			Timeout:          30 * time.Second,
			FailureThreshold: 5,
			SuccessThreshold: 2,
		}
		breaker = resilience.NewCircuitBreaker(breakerSettings, resilience.NoopFallback)
	}
	retryConfig := resilience.DefaultRetryConfig()
	retryConfig.RetryableChecker = isStripeRetryable

	return &ResilientStripeClient{
		client:  client,
		breaker: breaker,
		retry:   retryConfig,
	}
}

// CreateCustomer creates a Stripe customer with resilience
func (r *ResilientStripeClient) CreateCustomer(email, name string, metadata map[string]string) (*stripe.Customer, error) {
	ctx := context.Background()

	result, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return r.client.CreateCustomer(email, name, metadata)
	})

	if err != nil {
		logger.Get().Error("Failed to create Stripe customer after retries",
			zap.Error(err),
			zap.String("email", email),
		)
		return nil, err
	}

	return result.(*stripe.Customer), nil
}

// CreatePaymentIntent creates a payment intent with resilience
func (r *ResilientStripeClient) CreatePaymentIntent(amount int64, currency, customerID, description string, metadata map[string]string) (*stripe.PaymentIntent, error) {
	ctx := context.Background()

	result, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return r.client.CreatePaymentIntent(amount, currency, customerID, description, metadata)
	})

	if err != nil {
		logger.Get().Error("Failed to create payment intent after retries",
			zap.Error(err),
			zap.Int64("amount", amount),
			zap.String("currency", currency),
		)
		return nil, err
	}

	logger.Get().Info("Successfully created payment intent",
		zap.String("payment_intent_id", result.(*stripe.PaymentIntent).ID),
		zap.Int64("amount", amount),
	)

	return result.(*stripe.PaymentIntent), nil
}

// ConfirmPaymentIntent confirms a payment intent with resilience
func (r *ResilientStripeClient) ConfirmPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	ctx := context.Background()

	result, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return r.client.ConfirmPaymentIntent(paymentIntentID)
	})

	if err != nil {
		logger.Get().Error("Failed to confirm payment intent after retries",
			zap.Error(err),
			zap.String("payment_intent_id", paymentIntentID),
		)
		return nil, err
	}

	return result.(*stripe.PaymentIntent), nil
}

// CapturePaymentIntent captures a payment intent with resilience
func (r *ResilientStripeClient) CapturePaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	ctx := context.Background()

	result, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return r.client.CapturePaymentIntent(paymentIntentID)
	})

	if err != nil {
		logger.Get().Error("Failed to capture payment intent after retries",
			zap.Error(err),
			zap.String("payment_intent_id", paymentIntentID),
		)
		return nil, err
	}

	return result.(*stripe.PaymentIntent), nil
}

// CreateRefund creates a refund with resilience
func (r *ResilientStripeClient) CreateRefund(chargeID string, amount *int64, reason string) (*stripe.Refund, error) {
	ctx := context.Background()

	result, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return r.client.CreateRefund(chargeID, amount, reason)
	})

	if err != nil {
		logger.Get().Error("Failed to create refund after retries",
			zap.Error(err),
			zap.String("charge_id", chargeID),
		)
		return nil, err
	}

	return result.(*stripe.Refund), nil
}

// CreateTransfer creates a transfer with resilience
func (r *ResilientStripeClient) CreateTransfer(amount int64, currency, destination, description string, metadata map[string]string) (*stripe.Transfer, error) {
	ctx := context.Background()

	result, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return r.client.CreateTransfer(amount, currency, destination, description, metadata)
	})

	if err != nil {
		logger.Get().Error("Failed to create transfer after retries",
			zap.Error(err),
			zap.Int64("amount", amount),
			zap.String("destination", destination),
		)
		return nil, err
	}

	return result.(*stripe.Transfer), nil
}

// GetPaymentIntent retrieves a payment intent with resilience
func (r *ResilientStripeClient) GetPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	ctx := context.Background()

	result, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return r.client.GetPaymentIntent(paymentIntentID)
	})

	if err != nil {
		logger.Get().Error("Failed to get payment intent after retries",
			zap.Error(err),
			zap.String("payment_intent_id", paymentIntentID),
		)
		return nil, err
	}

	return result.(*stripe.PaymentIntent), nil
}

// CancelPaymentIntent cancels a payment intent with resilience
func (r *ResilientStripeClient) CancelPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	ctx := context.Background()

	result, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return r.client.CancelPaymentIntent(paymentIntentID)
	})

	if err != nil {
		logger.Get().Error("Failed to cancel payment intent after retries",
			zap.Error(err),
			zap.String("payment_intent_id", paymentIntentID),
		)
		return nil, err
	}

	return result.(*stripe.PaymentIntent), nil
}

// isStripeRetryable determines if a Stripe error should be retried
func isStripeRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a Stripe error
	if stripeErr, ok := err.(*stripe.Error); ok {
		// Retry on API errors (temporary issues)
		if stripeErr.Type == stripe.ErrorTypeAPI {
			return true
		}

		// Retry on certain HTTP status codes
		if stripeErr.HTTPStatusCode >= 500 && stripeErr.HTTPStatusCode < 600 {
			return true // Retry on 5xx errors (server errors)
		}

		if stripeErr.HTTPStatusCode == 429 {
			return true // Retry on rate limit (429 Too Many Requests)
		}

		if stripeErr.HTTPStatusCode == 408 {
			return true // Retry on request timeout
		}

		if stripeErr.HTTPStatusCode == 503 {
			return true // Retry on service unavailable
		}

		// Don't retry on client errors (4xx except 408, 429)
		// These include card errors, invalid requests, authentication failures, etc.
		if stripeErr.HTTPStatusCode >= 400 && stripeErr.HTTPStatusCode < 500 {
			return false
		}

		// Don't retry on card errors - these are customer issues, not transient
		if stripeErr.Type == stripe.ErrorTypeCard {
			return false
		}

		// Don't retry on invalid request errors - these won't succeed on retry
		if stripeErr.Type == stripe.ErrorTypeInvalidRequest {
			return false
		}
	}

	// For non-Stripe errors (network issues, etc.), use default retry logic
	return true
}
