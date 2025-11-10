package notifications

import (
	"context"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// RepositoryInterface defines the interface for notifications repository operations
type RepositoryInterface interface {
	CreateNotification(ctx context.Context, notification *models.Notification) error
	UpdateNotificationStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error
	GetUserDeviceTokens(ctx context.Context, userID uuid.UUID) ([]string, error)
	GetUserPhoneNumber(ctx context.Context, userID uuid.UUID) (string, error)
	GetUserEmail(ctx context.Context, userID uuid.UUID) (string, error)
	GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Notification, error)
	MarkNotificationAsRead(ctx context.Context, notificationID uuid.UUID) error
	GetUnreadNotificationCount(ctx context.Context, userID uuid.UUID) (int, error)
	GetPendingNotifications(ctx context.Context, limit int) ([]*models.Notification, error)
	ScheduleNotificationRetry(ctx context.Context, id uuid.UUID, retryAt time.Time, errorMsg string) error
}

// FirebaseClientInterface defines the interface for Firebase push notifications
type FirebaseClientInterface interface {
	SendPushNotification(ctx context.Context, token, title, body string, data map[string]string) (string, error)
	SendMulticastNotification(ctx context.Context, tokens []string, title, body string, data map[string]string) (*messaging.BatchResponse, error)
	SendTopicNotification(ctx context.Context, topic, title, body string, data map[string]string) (string, error)
	SubscribeToTopic(ctx context.Context, tokens []string, topic string) error
	UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) error
}

// TwilioClientInterface defines the interface for Twilio SMS
type TwilioClientInterface interface {
	SendSMS(to, message string) (string, error)
	SendBulkSMS(recipients []string, body string) ([]string, []error)
	GetMessageStatus(messageSid string) (string, error)
	SendOTP(to, otp string) (string, error)
	SendRideNotification(to, message string) (string, error)
}

// EmailClientInterface defines the interface for email notifications
type EmailClientInterface interface {
	SendEmail(to, subject, body string) error
	SendHTMLEmail(to, subject, htmlBody string) error
	SendRideConfirmationEmail(to, userName string, rideDetails map[string]interface{}) error
	SendReceiptEmail(to, userName string, receiptData map[string]interface{}) error
}
