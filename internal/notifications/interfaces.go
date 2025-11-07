package notifications

import (
	"context"

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
}

// FirebaseClientInterface defines the interface for Firebase push notifications
type FirebaseClientInterface interface {
	SendMulticastNotification(ctx context.Context, tokens []string, title, body string, data map[string]string) (*messaging.BatchResponse, error)
}

// TwilioClientInterface defines the interface for Twilio SMS
type TwilioClientInterface interface {
	SendSMS(to, message string) (string, error)
}

// EmailClientInterface defines the interface for email notifications
type EmailClientInterface interface {
	SendEmail(to, subject, body string) error
	SendHTMLEmail(to, subject, htmlBody string) error
	SendRideConfirmationEmail(to, userName string, rideDetails map[string]interface{}) error
	SendReceiptEmail(to, userName string, receiptData map[string]interface{}) error
}
