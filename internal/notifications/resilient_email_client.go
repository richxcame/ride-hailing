package notifications

import (
	"context"
	"strings"
	"time"

	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"go.uber.org/zap"
)

// ResilientEmailClient wraps EmailClient with circuit breaker and retry logic
type ResilientEmailClient struct {
	client  EmailClientInterface
	breaker *resilience.CircuitBreaker
	retry   resilience.RetryConfig
}

// NewResilientEmailClient creates a new resilient email client
func NewResilientEmailClient(smtpHost, smtpPort, username, password, fromEmail, fromName string, breaker *resilience.CircuitBreaker) *ResilientEmailClient {
	client := NewEmailClient(smtpHost, smtpPort, username, password, fromEmail, fromName)
	return NewResilientEmailClientWithClient(client, breaker)
}

// NewResilientEmailClientWithClient creates a resilient wrapper around an existing client
func NewResilientEmailClientWithClient(client EmailClientInterface, breaker *resilience.CircuitBreaker) *ResilientEmailClient {
	if breaker == nil {
		breakerSettings := resilience.Settings{
			Name:             "smtp-email",
			Interval:         60 * time.Second,
			Timeout:          30 * time.Second,
			FailureThreshold: 5,
			SuccessThreshold: 2,
		}

		breaker = resilience.NewCircuitBreaker(breakerSettings, func(ctx context.Context, err error) (interface{}, error) {
			logger.Get().Error("Email circuit breaker open, email send failed",
				zap.Error(err),
			)
			// For emails, we can queue them for later retry or just log the failure
			return nil, err
		})
	}

	// Configure retry with exponential backoff
	// Email retries are less aggressive since emails can be delayed
	retryConfig := resilience.ConservativeRetryConfig()
	retryConfig.MaxAttempts = 3
	retryConfig.InitialBackoff = 2 * time.Second
	retryConfig.MaxBackoff = 15 * time.Second
	retryConfig.RetryableChecker = isEmailRetryable

	return &ResilientEmailClient{
		client:  client,
		breaker: breaker,
		retry:   retryConfig,
	}
}

// SendEmail sends a plain text email with retry and circuit breaker
func (r *ResilientEmailClient) SendEmail(to, subject, body string) error {
	ctx := context.Background()

	_, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return nil, r.client.SendEmail(to, subject, body)
	})

	if err != nil {
		logger.Get().Error("Failed to send email after retries",
			zap.Error(err),
			zap.String("to", maskEmail(to)),
			zap.String("subject", subject),
		)
		return err
	}

	logger.Get().Debug("Successfully sent email",
		zap.String("to", maskEmail(to)),
		zap.String("subject", subject),
	)

	return nil
}

// SendHTMLEmail sends an HTML email with retry and circuit breaker
func (r *ResilientEmailClient) SendHTMLEmail(to, subject, htmlBody string) error {
	ctx := context.Background()

	_, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return nil, r.client.SendHTMLEmail(to, subject, htmlBody)
	})

	if err != nil {
		logger.Get().Error("Failed to send HTML email after retries",
			zap.Error(err),
			zap.String("to", maskEmail(to)),
			zap.String("subject", subject),
		)
		return err
	}

	logger.Get().Debug("Successfully sent HTML email",
		zap.String("to", maskEmail(to)),
		zap.String("subject", subject),
	)

	return nil
}

// SendRideConfirmationEmail sends ride confirmation email with retry and circuit breaker
func (r *ResilientEmailClient) SendRideConfirmationEmail(to, userName string, rideDetails map[string]interface{}) error {
	ctx := context.Background()

	_, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return nil, r.client.SendRideConfirmationEmail(to, userName, rideDetails)
	})

	if err != nil {
		logger.Get().Error("Failed to send ride confirmation email after retries",
			zap.Error(err),
			zap.String("to", maskEmail(to)),
			zap.String("user", userName),
		)
		return err
	}

	logger.Get().Debug("Successfully sent ride confirmation email",
		zap.String("to", maskEmail(to)),
	)

	return nil
}

// SendReceiptEmail sends receipt email with retry and circuit breaker
func (r *ResilientEmailClient) SendReceiptEmail(to, userName string, receiptData map[string]interface{}) error {
	ctx := context.Background()

	_, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return nil, r.client.SendReceiptEmail(to, userName, receiptData)
	})

	if err != nil {
		logger.Get().Error("Failed to send receipt email after retries",
			zap.Error(err),
			zap.String("to", maskEmail(to)),
			zap.String("user", userName),
		)
		return err
	}

	logger.Get().Debug("Successfully sent receipt email",
		zap.String("to", maskEmail(to)),
	)

	return nil
}

// isEmailRetryable determines if an email error should be retried
func isEmailRetryable(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Retry on SMTP transient errors
	retryableMessages := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"broken pipe",
		"temporary failure",
		"421", // Service not available (temporary)
		"450", // Requested mail action not taken: mailbox unavailable (temporary)
		"451", // Requested action aborted: local error in processing
		"452", // Requested action not taken: insufficient system storage
		"network is unreachable",
		"i/o timeout",
		"eof",
		"too many connections",
		"server closed",
	}

	for _, msg := range retryableMessages {
		if strings.Contains(errMsg, msg) {
			return true
		}
	}

	// Don't retry on permanent SMTP errors
	nonRetryableMessages := []string{
		"550", // Requested action not taken: mailbox unavailable (permanent)
		"551", // User not local
		"552", // Requested mail action aborted: exceeded storage allocation
		"553", // Requested action not taken: mailbox name not allowed
		"554", // Transaction failed / No SMTP service here
		"invalid address",
		"invalid email",
		"mailbox not found",
		"user unknown",
		"domain not found",
		"authentication failed",
		"auth failed",
		"bad username",
		"bad password",
		"access denied",
	}

	for _, msg := range nonRetryableMessages {
		if strings.Contains(errMsg, msg) {
			return false
		}
	}

	// By default, retry network-related errors
	return true
}

// maskEmail masks email address for logging (show only first char and domain)
func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}
	if len(parts[0]) == 0 {
		return "***@" + parts[1]
	}
	return string(parts[0][0]) + "***@" + parts[1]
}
