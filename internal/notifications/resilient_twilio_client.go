package notifications

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"go.uber.org/zap"
)

// ResilientTwilioClient wraps TwilioClient with circuit breaker and retry logic
type ResilientTwilioClient struct {
	client  TwilioClientInterface
	breaker *resilience.CircuitBreaker
	retry   resilience.RetryConfig
}

// NewResilientTwilioClient creates a new resilient Twilio client
func NewResilientTwilioClient(accountSid, authToken, fromNumber string, breaker *resilience.CircuitBreaker) *ResilientTwilioClient {
	client := NewTwilioClient(accountSid, authToken, fromNumber)
	return NewResilientTwilioClientWithClient(client, breaker)
}

// NewResilientTwilioClientWithClient creates a resilient wrapper around an existing client
func NewResilientTwilioClientWithClient(client TwilioClientInterface, breaker *resilience.CircuitBreaker) *ResilientTwilioClient {
	if breaker == nil {
		breakerSettings := resilience.Settings{
			Name:             "twilio-sms",
			Interval:         60 * time.Second,
			Timeout:          30 * time.Second,
			FailureThreshold: 5,
			SuccessThreshold: 2,
		}

		breaker = resilience.NewCircuitBreaker(breakerSettings, func(ctx context.Context, err error) (interface{}, error) {
			logger.Get().Error("Twilio circuit breaker open, SMS send failed",
				zap.Error(err),
			)
			// For SMS, we can queue them for later retry
			return "", err
		})
	}

	// Configure retry with exponential backoff
	retryConfig := resilience.DefaultRetryConfig()
	retryConfig.MaxAttempts = 3
	retryConfig.InitialBackoff = 1 * time.Second
	retryConfig.MaxBackoff = 10 * time.Second
	retryConfig.RetryableChecker = isTwilioRetryable

	return &ResilientTwilioClient{
		client:  client,
		breaker: breaker,
		retry:   retryConfig,
	}
}

// SendSMS sends an SMS message with retry and circuit breaker
func (r *ResilientTwilioClient) SendSMS(to, body string) (string, error) {
	ctx := context.Background()

	result, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return r.client.SendSMS(to, body)
	})

	if err != nil {
		logger.Get().Error("Failed to send SMS after retries",
			zap.Error(err),
			zap.String("to", maskPhoneNumber(to)),
		)
		return "", err
	}

	logger.Get().Debug("Successfully sent SMS",
		zap.String("message_sid", result.(string)),
		zap.String("to", maskPhoneNumber(to)),
	)

	return result.(string), nil
}

// SendBulkSMS sends SMS to multiple recipients with retry and circuit breaker
func (r *ResilientTwilioClient) SendBulkSMS(recipients []string, body string) ([]string, []error) {
	var messageIds []string
	var errors []error

	for _, recipient := range recipients {
		sid, err := r.SendSMS(recipient, body)
		messageIds = append(messageIds, sid)
		errors = append(errors, err)
	}

	return messageIds, errors
}

// GetMessageStatus retrieves the status of a sent message with retry and circuit breaker
func (r *ResilientTwilioClient) GetMessageStatus(messageSid string) (string, error) {
	ctx := context.Background()

	result, err := resilience.RetryWithBreaker(ctx, r.retry, r.breaker, func(ctx context.Context) (interface{}, error) {
		return r.client.GetMessageStatus(messageSid)
	})

	if err != nil {
		logger.Get().Error("Failed to get message status after retries",
			zap.Error(err),
			zap.String("message_sid", messageSid),
		)
		return "", err
	}

	return result.(string), nil
}

// SendOTP sends a one-time password via SMS with retry and circuit breaker
func (r *ResilientTwilioClient) SendOTP(to, otp string) (string, error) {
	body := fmt.Sprintf("Your verification code is: %s. This code expires in 10 minutes.", otp)
	return r.SendSMS(to, body)
}

// SendRideNotification sends a ride-related SMS notification with retry and circuit breaker
func (r *ResilientTwilioClient) SendRideNotification(to, message string) (string, error) {
	return r.SendSMS(to, message)
}

// isTwilioRetryable determines if a Twilio error should be retried
func isTwilioRetryable(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// Retry on specific Twilio error codes
	retryableErrors := []string{
		"20429", // Too Many Requests
		"20500", // Internal Server Error
		"20503", // Service Unavailable
		"30001", // Queue overflow
		"30002", // Account suspended (might be temporary)
		"30003", // Unreachable destination handset (might be temporary)
		"30005", // Unknown destination handset (might be temporary)
		"timeout",
		"connection",
		"network",
	}

	for _, code := range retryableErrors {
		if strings.Contains(errMsg, code) {
			return true
		}
	}

	// Don't retry on validation errors or permanent failures
	nonRetryableErrors := []string{
		"21211", // Invalid 'To' phone number
		"21212", // Invalid 'From' phone number
		"21408", // Permission denied
		"21606", // Phone number is not a mobile number
		"21610", // Attempt to send to unsubscribed recipient
		"21614", // 'To' number is not a valid mobile number
		"invalid",
		"unauthorized",
		"forbidden",
	}

	for _, code := range nonRetryableErrors {
		if strings.Contains(errMsg, code) {
			return false
		}
	}

	// By default, retry network-related errors
	return true
}

// maskPhoneNumber masks phone number for logging (show only last 4 digits)
func maskPhoneNumber(phoneNumber string) string {
	if len(phoneNumber) <= 4 {
		return "***"
	}
	return "***" + phoneNumber[len(phoneNumber)-4:]
}
