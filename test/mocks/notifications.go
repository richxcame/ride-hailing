package mocks

import (
	"context"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/mock"
)

// MockNotificationsRepository is a mock implementation of notifications.RepositoryInterface
type MockNotificationsRepository struct {
	mock.Mock
}

func (m *MockNotificationsRepository) CreateNotification(ctx context.Context, notification *models.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockNotificationsRepository) UpdateNotificationStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error {
	args := m.Called(ctx, id, status, errorMsg)
	return args.Error(0)
}

func (m *MockNotificationsRepository) GetUserDeviceTokens(ctx context.Context, userID uuid.UUID) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockNotificationsRepository) GetUserPhoneNumber(ctx context.Context, userID uuid.UUID) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockNotificationsRepository) GetUserEmail(ctx context.Context, userID uuid.UUID) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockNotificationsRepository) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Notification, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Notification), args.Error(1)
}

func (m *MockNotificationsRepository) MarkNotificationAsRead(ctx context.Context, notificationID uuid.UUID) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockNotificationsRepository) GetUnreadNotificationCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockNotificationsRepository) GetPendingNotifications(ctx context.Context, limit int) ([]*models.Notification, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Notification), args.Error(1)
}

func (m *MockNotificationsRepository) ScheduleNotificationRetry(ctx context.Context, id uuid.UUID, retryAt time.Time, errorMsg string) error {
	args := m.Called(ctx, id, retryAt, errorMsg)
	return args.Error(0)
}

// MockFirebaseClient is a mock implementation of notifications.FirebaseClientInterface
type MockFirebaseClient struct {
	mock.Mock
}

func (m *MockFirebaseClient) SendPushNotification(ctx context.Context, token, title, body string, data map[string]string) (string, error) {
	args := m.Called(ctx, token, title, body, data)
	return args.String(0), args.Error(1)
}

func (m *MockFirebaseClient) SendMulticastNotification(ctx context.Context, tokens []string, title, body string, data map[string]string) (*messaging.BatchResponse, error) {
	args := m.Called(ctx, tokens, title, body, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messaging.BatchResponse), args.Error(1)
}

func (m *MockFirebaseClient) SendTopicNotification(ctx context.Context, topic, title, body string, data map[string]string) (string, error) {
	args := m.Called(ctx, topic, title, body, data)
	return args.String(0), args.Error(1)
}

func (m *MockFirebaseClient) SubscribeToTopic(ctx context.Context, tokens []string, topic string) error {
	args := m.Called(ctx, tokens, topic)
	return args.Error(0)
}

func (m *MockFirebaseClient) UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) error {
	args := m.Called(ctx, tokens, topic)
	return args.Error(0)
}

// MockTwilioClient is a mock implementation of notifications.TwilioClientInterface
type MockTwilioClient struct {
	mock.Mock
}

func (m *MockTwilioClient) SendSMS(to, message string) (string, error) {
	args := m.Called(to, message)
	return args.String(0), args.Error(1)
}

func (m *MockTwilioClient) SendBulkSMS(recipients []string, body string) ([]string, []error) {
	args := m.Called(recipients, body)
	if args.Get(0) == nil {
		return nil, args.Get(1).([]error)
	}
	return args.Get(0).([]string), args.Get(1).([]error)
}

func (m *MockTwilioClient) GetMessageStatus(messageSid string) (string, error) {
	args := m.Called(messageSid)
	return args.String(0), args.Error(1)
}

func (m *MockTwilioClient) SendOTP(to, otp string) (string, error) {
	args := m.Called(to, otp)
	return args.String(0), args.Error(1)
}

func (m *MockTwilioClient) SendRideNotification(to, message string) (string, error) {
	args := m.Called(to, message)
	return args.String(0), args.Error(1)
}

// MockEmailClient is a mock implementation of notifications.EmailClientInterface
type MockEmailClient struct {
	mock.Mock
}

func (m *MockEmailClient) SendEmail(to, subject, body string) error {
	args := m.Called(to, subject, body)
	return args.Error(0)
}

func (m *MockEmailClient) SendHTMLEmail(to, subject, htmlBody string) error {
	args := m.Called(to, subject, htmlBody)
	return args.Error(0)
}

func (m *MockEmailClient) SendRideConfirmationEmail(to, userName string, rideDetails map[string]interface{}) error {
	args := m.Called(to, userName, rideDetails)
	return args.Error(0)
}

func (m *MockEmailClient) SendReceiptEmail(to, userName string, receiptData map[string]interface{}) error {
	args := m.Called(to, userName, receiptData)
	return args.Error(0)
}
