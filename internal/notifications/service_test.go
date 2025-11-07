package notifications

import (
	"context"
	"errors"
	"testing"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ===== Helper for Async Tests =====

// setupAsyncMocks sets up lenient mocks for async processNotification calls
func setupAsyncMocks(mockRepo *mocks.MockNotificationsRepository, mockFirebase *mocks.MockFirebaseClient,
	mockTwilio *mocks.MockTwilioClient, mockEmail *mocks.MockEmailClient) {
	// Push notifications
	mockRepo.On("GetUserDeviceTokens", mock.Anything, mock.Anything).Return([]string{"token1"}, nil).Maybe()
	mockFirebase.On("SendMulticastNotification", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&messaging.BatchResponse{SuccessCount: 1}, nil).Maybe()

	// SMS notifications
	mockRepo.On("GetUserPhoneNumber", mock.Anything, mock.Anything).Return("+1234567890", nil).Maybe()
	mockTwilio.On("SendSMS", mock.Anything, mock.Anything).Return("SMS123", nil).Maybe()

	// Email notifications
	mockRepo.On("GetUserEmail", mock.Anything, mock.Anything).Return("user@example.com", nil).Maybe()
	mockEmail.On("SendEmail", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	mockEmail.On("SendHTMLEmail", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	mockEmail.On("SendRideConfirmationEmail", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	mockEmail.On("SendReceiptEmail", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// Status updates
	mockRepo.On("UpdateNotificationStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
}

// ===== Core Notification Sending Tests =====
// Note: We test the synchronous creation part and the helper methods separately
// to avoid issues with async goroutines in processNotification

func TestService_SendNotification_CreateSuccess(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()
	data := map[string]interface{}{"key": "value"}

	mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).
		Return(nil)

	// Set up lenient mocks for async processNotification
	setupAsyncMocks(mockRepo, mockFirebase, mockTwilio, mockEmail)

	// Act
	notification, err := service.SendNotification(ctx, userID, "test_type", "push", "Test Title", "Test Body", data)

	// Assert - Only test the synchronous part
	assert.NoError(t, err)
	assert.NotNil(t, notification)
	assert.Equal(t, userID, notification.UserID)
	assert.Equal(t, "test_type", notification.Type)
	assert.Equal(t, "push", notification.Channel)
	assert.Equal(t, "Test Title", notification.Title)
	assert.Equal(t, "Test Body", notification.Body)
	assert.Equal(t, "pending", notification.Status)

	// Sleep briefly to let goroutine finish
	time.Sleep(20 * time.Millisecond)
}

func TestService_SendNotification_CreateError(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).
		Return(errors.New("database error"))

	// Act
	notification, err := service.SendNotification(ctx, userID, "test_type", "push", "Test", "Test", nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, notification)
	assert.Contains(t, err.Error(), "database error")
	mockRepo.AssertExpectations(t)
}

// ===== Push Notification Tests =====

func TestService_SendPushNotification_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()
	tokens := []string{"token1", "token2"}

	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  userID,
		Type:    "test",
		Channel: "push",
		Title:   "Test",
		Body:    "Test body",
		Data:    map[string]interface{}{"key": "value"},
		Status:  "pending",
	}

	mockRepo.On("GetUserDeviceTokens", ctx, userID).Return(tokens, nil)
	mockFirebase.On("SendMulticastNotification", ctx, tokens, "Test", "Test body", mock.AnythingOfType("map[string]string")).
		Return(&messaging.BatchResponse{SuccessCount: 2}, nil)

	// Act
	err := service.sendPushNotification(ctx, notification)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockFirebase.AssertExpectations(t)
}

func TestService_SendPushNotification_NoFirebaseClient(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, nil, mockTwilio, mockEmail)

	ctx := context.Background()
	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		Channel: "push",
		Title:   "Test",
		Body:    "Test",
	}

	// Act
	err := service.sendPushNotification(ctx, notification)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "firebase client not initialized")
}

func TestService_SendPushNotification_NoDeviceTokens(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  userID,
		Channel: "push",
		Title:   "Test",
		Body:    "Test",
	}

	mockRepo.On("GetUserDeviceTokens", ctx, userID).Return([]string{}, nil)

	// Act
	err := service.sendPushNotification(ctx, notification)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no device tokens found")
	mockRepo.AssertExpectations(t)
}

func TestService_SendPushNotification_GetTokensError(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  userID,
		Channel: "push",
	}

	mockRepo.On("GetUserDeviceTokens", ctx, userID).Return(nil, errors.New("database error"))

	// Act
	err := service.sendPushNotification(ctx, notification)

	// Assert
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_SendPushNotification_FirebaseError(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()
	tokens := []string{"token1"}

	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  userID,
		Channel: "push",
		Title:   "Test",
		Body:    "Test",
	}

	mockRepo.On("GetUserDeviceTokens", ctx, userID).Return(tokens, nil)
	mockFirebase.On("SendMulticastNotification", ctx, tokens, "Test", "Test", mock.AnythingOfType("map[string]string")).
		Return(nil, errors.New("firebase error"))

	// Act
	err := service.sendPushNotification(ctx, notification)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "firebase error")
	mockRepo.AssertExpectations(t)
	mockFirebase.AssertExpectations(t)
}

// ===== SMS Notification Tests =====

func TestService_SendSMSNotification_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  userID,
		Channel: "sms",
		Title:   "Test Title",
		Body:    "Test Body",
	}

	mockRepo.On("GetUserPhoneNumber", ctx, userID).Return("+1234567890", nil)
	mockTwilio.On("SendSMS", "+1234567890", "Test Title: Test Body").Return("SMS123", nil)

	// Act
	err := service.sendSMSNotification(ctx, notification)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockTwilio.AssertExpectations(t)
}

func TestService_SendSMSNotification_NoTwilioClient(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, nil, mockEmail)

	ctx := context.Background()
	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		Channel: "sms",
	}

	// Act
	err := service.sendSMSNotification(ctx, notification)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "twilio client not initialized")
}

func TestService_SendSMSNotification_GetPhoneError(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  userID,
		Channel: "sms",
	}

	mockRepo.On("GetUserPhoneNumber", ctx, userID).Return("", errors.New("user not found"))

	// Act
	err := service.sendSMSNotification(ctx, notification)

	// Assert
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_SendSMSNotification_TwilioError(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  userID,
		Channel: "sms",
		Title:   "Test",
		Body:    "Test",
	}

	mockRepo.On("GetUserPhoneNumber", ctx, userID).Return("+1234567890", nil)
	mockTwilio.On("SendSMS", "+1234567890", "Test: Test").Return("", errors.New("twilio error"))

	// Act
	err := service.sendSMSNotification(ctx, notification)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "twilio error")
	mockRepo.AssertExpectations(t)
	mockTwilio.AssertExpectations(t)
}

// ===== Email Notification Tests =====

func TestService_SendEmailNotification_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  userID,
		Channel: "email",
		Type:    "general",
		Title:   "Test Subject",
		Body:    "Test Body",
	}

	mockRepo.On("GetUserEmail", ctx, userID).Return("user@example.com", nil)
	mockEmail.On("SendEmail", "user@example.com", "Test Subject", "Test Body").Return(nil)

	// Act
	err := service.sendEmailNotification(ctx, notification)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockEmail.AssertExpectations(t)
}

func TestService_SendEmailNotification_RideConfirmation(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	rideDetails := map[string]interface{}{
		"pickup":  "123 Main St",
		"dropoff": "456 Oak Ave",
	}

	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  userID,
		Channel: "email",
		Type:    "ride_confirmed",
		Title:   "Ride Confirmed",
		Body:    "Your ride is confirmed",
		Data: map[string]interface{}{
			"details": rideDetails,
		},
	}

	mockRepo.On("GetUserEmail", ctx, userID).Return("user@example.com", nil)
	mockEmail.On("SendRideConfirmationEmail", "user@example.com", "User", rideDetails).Return(nil)

	// Act
	err := service.sendEmailNotification(ctx, notification)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockEmail.AssertExpectations(t)
}

func TestService_SendEmailNotification_Receipt(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	receiptData := map[string]interface{}{
		"Fare":   "$25.00",
		"Date":   "Nov 7, 2025",
		"Status": "Completed",
	}

	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  userID,
		Channel: "email",
		Type:    "ride_receipt",
		Title:   "Your Receipt",
		Body:    "Receipt",
		Data: map[string]interface{}{
			"receipt": receiptData,
		},
	}

	mockRepo.On("GetUserEmail", ctx, userID).Return("user@example.com", nil)
	mockEmail.On("SendReceiptEmail", "user@example.com", "User", receiptData).Return(nil)

	// Act
	err := service.sendEmailNotification(ctx, notification)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockEmail.AssertExpectations(t)
}

func TestService_SendEmailNotification_NoEmailClient(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, nil)

	ctx := context.Background()
	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		Channel: "email",
	}

	// Act
	err := service.sendEmailNotification(ctx, notification)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email client not initialized")
}

func TestService_SendEmailNotification_GetEmailError(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  userID,
		Channel: "email",
		Type:    "general",
	}

	mockRepo.On("GetUserEmail", ctx, userID).Return("", errors.New("user not found"))

	// Act
	err := service.sendEmailNotification(ctx, notification)

	// Assert
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_SendEmailNotification_SendError(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	notification := &models.Notification{
		ID:      uuid.New(),
		UserID:  userID,
		Channel: "email",
		Type:    "general",
		Title:   "Test",
		Body:    "Test",
	}

	mockRepo.On("GetUserEmail", ctx, userID).Return("user@example.com", nil)
	mockEmail.On("SendEmail", "user@example.com", "Test", "Test").Return(errors.New("smtp error"))

	// Act
	err := service.sendEmailNotification(ctx, notification)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "smtp error")
	mockRepo.AssertExpectations(t)
	mockEmail.AssertExpectations(t)
}

// ===== Ride Event Notification Tests =====
// Note: These tests only verify the synchronous part (creating notifications in DB)
// The async processing is tested separately via helper methods

func TestService_NotifyRideRequested_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	setupAsyncMocks(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	driverID := uuid.New()
	rideID := uuid.New()

	// Push notification
	mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n *models.Notification) bool {
		return n.UserID == driverID && n.Type == "ride_requested" && n.Channel == "push"
	})).Return(nil).Once()

	// SMS notification (best effort)
	mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n *models.Notification) bool {
		return n.UserID == driverID && n.Type == "ride_requested" && n.Channel == "sms"
	})).Return(nil).Once()

	// Act
	err := service.NotifyRideRequested(ctx, driverID, rideID, "123 Main St")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	time.Sleep(10 * time.Millisecond) // Let goroutines finish
}

func TestService_NotifyRideRequested_PushError(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	driverID := uuid.New()
	rideID := uuid.New()

	mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).
		Return(errors.New("database error"))

	// Act
	err := service.NotifyRideRequested(ctx, driverID, rideID, "123 Main St")

	// Assert
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_NotifyRideAccepted_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	setupAsyncMocks(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	riderID := uuid.New()

	mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n *models.Notification) bool {
		return n.UserID == riderID && n.Type == "ride_accepted" && n.Channel == "push"
	})).Return(nil)

	// Act
	err := service.NotifyRideAccepted(ctx, riderID, "John Driver", 5)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	time.Sleep(10 * time.Millisecond)
}

func TestService_NotifyRideStarted_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	setupAsyncMocks(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	riderID := uuid.New()

	mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n *models.Notification) bool {
		return n.UserID == riderID && n.Type == "ride_started" && n.Channel == "push"
	})).Return(nil)

	// Act
	err := service.NotifyRideStarted(ctx, riderID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	time.Sleep(10 * time.Millisecond)
}

func TestService_NotifyRideCompleted_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	setupAsyncMocks(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	riderID := uuid.New()
	driverID := uuid.New()

	// Rider push notification
	mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n *models.Notification) bool {
		return n.UserID == riderID && n.Type == "ride_completed" && n.Channel == "push"
	})).Return(nil).Once()

	// Rider receipt email (best effort)
	mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n *models.Notification) bool {
		return n.UserID == riderID && n.Type == "ride_receipt" && n.Channel == "email"
	})).Return(nil).Once()

	// Driver notification
	mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n *models.Notification) bool {
		return n.UserID == driverID && n.Type == "ride_completed" && n.Channel == "push"
	})).Return(nil).Once()

	// Act
	err := service.NotifyRideCompleted(ctx, riderID, driverID, 50.0)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	time.Sleep(10 * time.Millisecond)
}

func TestService_NotifyRideCancelled_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	setupAsyncMocks(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n *models.Notification) bool {
		return n.UserID == userID && n.Type == "ride_cancelled" && n.Channel == "push"
	})).Return(nil)

	// Act
	err := service.NotifyRideCancelled(ctx, userID, "driver")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	time.Sleep(10 * time.Millisecond)
}

// ===== Payment Notification Tests =====

func TestService_NotifyPaymentReceived_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	setupAsyncMocks(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n *models.Notification) bool {
		return n.UserID == userID && n.Type == "payment_received" && n.Channel == "push"
	})).Return(nil)

	// Act
	err := service.NotifyPaymentReceived(ctx, userID, 100.0)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	time.Sleep(10 * time.Millisecond)
}

// ===== User Notification Management Tests =====

func TestService_GetUserNotifications_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	expectedNotifications := []*models.Notification{
		{
			ID:      uuid.New(),
			UserID:  userID,
			Type:    "test",
			Channel: "push",
			Title:   "Test",
			Body:    "Test",
			Status:  "sent",
		},
	}

	mockRepo.On("GetUserNotifications", ctx, userID, 10, 0).Return(expectedNotifications, nil)

	// Act
	notifications, err := service.GetUserNotifications(ctx, userID, 10, 0)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedNotifications, notifications)
	mockRepo.AssertExpectations(t)
}

func TestService_GetUserNotifications_Error(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("GetUserNotifications", ctx, userID, 10, 0).Return(nil, errors.New("database error"))

	// Act
	notifications, err := service.GetUserNotifications(ctx, userID, 10, 0)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, notifications)
	mockRepo.AssertExpectations(t)
}

func TestService_MarkAsRead_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	notificationID := uuid.New()

	mockRepo.On("MarkNotificationAsRead", ctx, notificationID).Return(nil)

	// Act
	err := service.MarkAsRead(ctx, notificationID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_MarkAsRead_Error(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	notificationID := uuid.New()

	mockRepo.On("MarkNotificationAsRead", ctx, notificationID).Return(errors.New("not found"))

	// Act
	err := service.MarkAsRead(ctx, notificationID)

	// Assert
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_GetUnreadCount_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("GetUnreadNotificationCount", ctx, userID).Return(5, nil)

	// Act
	count, err := service.GetUnreadCount(ctx, userID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
	mockRepo.AssertExpectations(t)
}

func TestService_GetUnreadCount_Error(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("GetUnreadNotificationCount", ctx, userID).Return(0, errors.New("database error"))

	// Act
	count, err := service.GetUnreadCount(ctx, userID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	mockRepo.AssertExpectations(t)
}

// ===== Scheduled and Bulk Notification Tests =====

func TestService_ScheduleNotification_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()
	scheduledAt := time.Now().Add(1 * time.Hour)

	mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n *models.Notification) bool {
		return n.UserID == userID && n.Status == "pending" && n.ScheduledAt != nil
	})).Return(nil)

	// Act
	notification, err := service.ScheduleNotification(ctx, userID, "scheduled", "push", "Test", "Test", nil, scheduledAt)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, notification)
	assert.Equal(t, "pending", notification.Status)
	assert.NotNil(t, notification.ScheduledAt)
	mockRepo.AssertExpectations(t)
}

func TestService_ScheduleNotification_Error(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userID := uuid.New()
	scheduledAt := time.Now().Add(1 * time.Hour)

	mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).
		Return(errors.New("database error"))

	// Act
	notification, err := service.ScheduleNotification(ctx, userID, "scheduled", "push", "Test", "Test", nil, scheduledAt)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, notification)
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessPendingNotifications_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	setupAsyncMocks(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()

	pendingNotifications := []*models.Notification{
		{
			ID:      uuid.New(),
			UserID:  uuid.New(),
			Type:    "test",
			Channel: "push",
			Title:   "Test",
			Body:    "Test",
			Status:  "pending",
		},
	}

	mockRepo.On("GetPendingNotifications", ctx, 100).Return(pendingNotifications, nil)

	// Act
	err := service.ProcessPendingNotifications(ctx)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// Note: processNotification runs asynchronously, so we can't test its effects here
}

func TestService_ProcessPendingNotifications_Error(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()

	mockRepo.On("GetPendingNotifications", ctx, 100).Return(nil, errors.New("database error"))

	// Act
	err := service.ProcessPendingNotifications(ctx)

	// Assert
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_SendBulkNotification_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	setupAsyncMocks(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	// Each user gets a notification created
	mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).
		Return(nil).Times(3)

	// Act
	err := service.SendBulkNotification(ctx, userIDs, "bulk", "push", "Test", "Test", nil)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	time.Sleep(50 * time.Millisecond) // Let all goroutines finish
}

func TestService_SendBulkNotification_PartialFailure(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockNotificationsRepository)
	mockFirebase := new(mocks.MockFirebaseClient)
	mockTwilio := new(mocks.MockTwilioClient)
	mockEmail := new(mocks.MockEmailClient)
	service := NewService(mockRepo, mockFirebase, mockTwilio, mockEmail)

	setupAsyncMocks(mockRepo, mockFirebase, mockTwilio, mockEmail)

	ctx := context.Background()
	userIDs := []uuid.UUID{uuid.New(), uuid.New()}

	// First succeeds, second fails (but bulk operation continues)
	mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).
		Return(nil).Once()
	mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).
		Return(errors.New("database error")).Once()

	// Act
	err := service.SendBulkNotification(ctx, userIDs, "bulk", "push", "Test", "Test", nil)

	// Assert
	assert.NoError(t, err) // Bulk operation doesn't fail on individual errors
	mockRepo.AssertExpectations(t)
	time.Sleep(30 * time.Millisecond) // Let goroutines finish
}
