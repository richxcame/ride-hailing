package notifications

import (
	"fmt"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// TwilioClient handles Twilio SMS operations
type TwilioClient struct {
	client      *twilio.RestClient
	fromNumber  string
	accountSid  string
	authToken   string
}

// NewTwilioClient creates a new Twilio client
func NewTwilioClient(accountSid, authToken, fromNumber string) *TwilioClient {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	return &TwilioClient{
		client:      client,
		fromNumber:  fromNumber,
		accountSid:  accountSid,
		authToken:   authToken,
	}
}

// SendSMS sends an SMS message
func (t *TwilioClient) SendSMS(to, body string) (string, error) {
	params := &twilioApi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(t.fromNumber)
	params.SetBody(body)

	resp, err := t.client.Api.CreateMessage(params)
	if err != nil {
		return "", fmt.Errorf("failed to send SMS: %w", err)
	}

	if resp.Sid == nil {
		return "", fmt.Errorf("no message SID returned")
	}

	return *resp.Sid, nil
}

// SendBulkSMS sends SMS to multiple recipients
func (t *TwilioClient) SendBulkSMS(recipients []string, body string) ([]string, []error) {
	var messageIds []string
	var errors []error

	for _, recipient := range recipients {
		sid, err := t.SendSMS(recipient, body)
		if err != nil {
			errors = append(errors, err)
			messageIds = append(messageIds, "")
		} else {
			messageIds = append(messageIds, sid)
			errors = append(errors, nil)
		}
	}

	return messageIds, errors
}

// GetMessageStatus retrieves the status of a sent message
func (t *TwilioClient) GetMessageStatus(messageSid string) (string, error) {
	params := &twilioApi.FetchMessageParams{}
	params.SetPathAccountSid(t.accountSid)

	resp, err := t.client.Api.FetchMessage(messageSid, params)
	if err != nil {
		return "", fmt.Errorf("failed to get message status: %w", err)
	}

	if resp.Status == nil {
		return "", fmt.Errorf("no status returned")
	}

	return *resp.Status, nil
}

// SendOTP sends a one-time password via SMS
func (t *TwilioClient) SendOTP(to, otp string) (string, error) {
	body := fmt.Sprintf("Your verification code is: %s. This code expires in 10 minutes.", otp)
	return t.SendSMS(to, body)
}

// SendRideNotification sends a ride-related SMS notification
func (t *TwilioClient) SendRideNotification(to, message string) (string, error) {
	return t.SendSMS(to, message)
}
