package payments

import (
	"fmt"

	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/charge"
	"github.com/stripe/stripe-go/v83/customer"
	"github.com/stripe/stripe-go/v83/paymentintent"
	"github.com/stripe/stripe-go/v83/refund"
	"github.com/stripe/stripe-go/v83/transfer"
)

// StripeClient wraps Stripe API operations
type StripeClient struct {
	apiKey string
}

// NewStripeClient creates a new Stripe client
func NewStripeClient(apiKey string) *StripeClient {
	stripe.Key = apiKey
	return &StripeClient{apiKey: apiKey}
}

// CreateCustomer creates a new Stripe customer
func (s *StripeClient) CreateCustomer(email, name string, metadata map[string]string) (*stripe.Customer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}

	for key, value := range metadata {
		params.AddMetadata(key, value)
	}

	cust, err := customer.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	return cust, nil
}

// CreatePaymentIntent creates a payment intent for a ride
func (s *StripeClient) CreatePaymentIntent(amount int64, currency, customerID, description string, metadata map[string]string) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentParams{
		Amount:      stripe.Int64(amount), // Amount in cents
		Currency:    stripe.String(currency),
		Customer:    stripe.String(customerID),
		Description: stripe.String(description),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	for key, value := range metadata {
		params.AddMetadata(key, value)
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	return pi, nil
}

// ConfirmPaymentIntent confirms a payment intent
func (s *StripeClient) ConfirmPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentConfirmParams{}
	pi, err := paymentintent.Confirm(paymentIntentID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to confirm payment intent: %w", err)
	}

	return pi, nil
}

// CapturePaymentIntent captures a payment intent
func (s *StripeClient) CapturePaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentCaptureParams{}
	pi, err := paymentintent.Capture(paymentIntentID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to capture payment intent: %w", err)
	}

	return pi, nil
}

// CreateCharge creates a direct charge (legacy method, prefer PaymentIntent)
func (s *StripeClient) CreateCharge(amount int64, currency, customerID, description string, metadata map[string]string) (*stripe.Charge, error) {
	params := &stripe.ChargeParams{
		Amount:      stripe.Int64(amount),
		Currency:    stripe.String(currency),
		Customer:    stripe.String(customerID),
		Description: stripe.String(description),
	}

	for key, value := range metadata {
		params.AddMetadata(key, value)
	}

	ch, err := charge.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create charge: %w", err)
	}

	return ch, nil
}

// CreateRefund creates a refund for a charge or payment intent
func (s *StripeClient) CreateRefund(chargeID string, amount *int64, reason string) (*stripe.Refund, error) {
	params := &stripe.RefundParams{
		Charge: stripe.String(chargeID),
	}

	if amount != nil {
		params.Amount = stripe.Int64(*amount)
	}

	if reason != "" {
		params.Reason = stripe.String(reason)
	}

	r, err := refund.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create refund: %w", err)
	}

	return r, nil
}

// CreateTransfer creates a transfer to a driver's connected account
func (s *StripeClient) CreateTransfer(amount int64, currency, destination, description string, metadata map[string]string) (*stripe.Transfer, error) {
	params := &stripe.TransferParams{
		Amount:      stripe.Int64(amount),
		Currency:    stripe.String(currency),
		Destination: stripe.String(destination),
		Description: stripe.String(description),
	}

	for key, value := range metadata {
		params.AddMetadata(key, value)
	}

	t, err := transfer.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer: %w", err)
	}

	return t, nil
}

// GetPaymentIntent retrieves a payment intent
func (s *StripeClient) GetPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	pi, err := paymentintent.Get(paymentIntentID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment intent: %w", err)
	}

	return pi, nil
}

// CancelPaymentIntent cancels a payment intent
func (s *StripeClient) CancelPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentCancelParams{}
	pi, err := paymentintent.Cancel(paymentIntentID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel payment intent: %w", err)
	}

	return pi, nil
}
