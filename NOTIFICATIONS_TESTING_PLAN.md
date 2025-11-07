# Notifications Service Testing Plan

## Current Status

The Notifications service is ready for testing but requires interface-based dependency injection similar to what was implemented for Auth, Geo, and Payments services.

## Service Overview

The notifications service handles:
- **Push Notifications** (via Firebase)
- **SMS Notifications** (via Twilio)
- **Email Notifications** (custom email client)
- **Scheduled Notifications**
- **Bulk Notifications**
- **Ride Event Notifications** (requested, accepted, started, completed, cancelled)
- **Payment Notifications**

## Implementation Steps

### 1. Interface Setup ✅
Created [internal/notifications/interfaces.go](internal/notifications/interfaces.go) with:
- `RepositoryInterface` - Database operations
- `FirebaseClientInterface` - Push notifications
- `TwilioClientInterface` - SMS notifications
- `EmailClientInterface` - Email notifications

### 2. Refactor Service (TODO)
Update [internal/notifications/service.go](internal/notifications/service.go):
```go
type Service struct {
    repo           RepositoryInterface
    firebaseClient FirebaseClientInterface
    twilioClient   TwilioClientInterface
    emailClient    EmailClientInterface
}
```

Add production constructor:
```go
func NewServiceWithClients(repo *Repository, firebaseClient *FirebaseClient,
    twilioClient *TwilioClient, emailClient *EmailClient) *Service
```

### 3. Create Mocks
File: `test/mocks/notifications.go`

Required mocks:
```go
type MockNotificationsRepository struct {
    mock.Mock
}

type MockFirebaseClient struct {
    mock.Mock
}

type MockTwilioClient struct {
    mock.Mock
}

type MockEmailClient struct {
    mock.Mock
}
```

### 4. Test Categories

#### Core Notification Tests
- `TestService_SendNotification_Push_Success`
- `TestService_SendNotification_SMS_Success`
- `TestService_SendNotification_Email_Success`
- `TestService_SendNotification_UnsupportedChannel`
- `TestService_SendNotification_RepositoryError`

#### Push Notification Tests
- `TestService_SendPushNotification_Success`
- `TestService_SendPushNotification_NoDeviceTokens`
- `TestService_SendPushNotification_FirebaseError`
- `TestService_SendPushNotification_ClientNotInitialized`

#### SMS Notification Tests
- `TestService_SendSMSNotification_Success`
- `TestService_SendSMSNotification_NoPhoneNumber`
- `TestService_SendSMSNotification_TwilioError`
- `TestService_SendSMSNotification_ClientNotInitialized`

#### Email Notification Tests
- `TestService_SendEmailNotification_Success`
- `TestService_SendEmailNotification_RideConfirmation`
- `TestService_SendEmailNotification_Receipt`
- `TestService_SendEmailNotification_NoEmail`
- `TestService_SendEmailNotification_ClientNotInitialized`

#### Ride Event Notification Tests
- `TestService_NotifyRideRequested_Success`
- `TestService_NotifyRideAccepted_Success`
- `TestService_NotifyRideStarted_Success`
- `TestService_NotifyRideCompleted_Success`
- `TestService_NotifyRideCancelled_Success`

#### User Notification Tests
- `TestService_GetUserNotifications_Success`
- `TestService_MarkAsRead_Success`
- `TestService_GetUnreadCount_Success`

#### Scheduled Notification Tests
- `TestService_ScheduleNotification_Success`
- `TestService_ProcessPendingNotifications_Success`

#### Bulk Notification Tests
- `TestService_SendBulkNotification_Success`
- `TestService_SendBulkNotification_PartialFailure`

### 5. Expected Coverage

| Component | Target Coverage |
|-----------|----------------|
| Service Layer | 90%+ |
| Business Logic | 95%+ |
| Error Handling | 100% |
| Overall Package | 40-60% |

## Test Example

```go
func TestService_SendNotification_Push_Success(t *testing.T) {
    // Arrange
    mockRepo := new(mocks.MockNotificationsRepository)
    mockFirebase := new(mocks.MockFirebaseClient)
    service := NewService(mockRepo, mockFirebase, nil, nil)
    ctx := context.Background()

    userID := uuid.New()
    tokens := []string{"token1", "token2"}

    mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).
        Return(nil)
    mockRepo.On("GetUserDeviceTokens", ctx, userID).Return(tokens, nil)
    mockFirebase.On("SendMulticastNotification", mock.Anything, tokens, "Test", "Test body", mock.Anything).
        Return(2, nil)
    mockRepo.On("UpdateNotificationStatus", mock.Anything, mock.Anything, "sent", mock.Anything).
        Return(nil)

    // Act
    notification, err := service.SendNotification(ctx, userID, "test", "push",
        "Test", "Test body", nil)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, notification)
    assert.Equal(t, "push", notification.Channel)

    // Wait for async processing
    time.Sleep(100 * time.Millisecond)
    mockRepo.AssertExpectations(t)
    mockFirebase.AssertExpectations(t)
}
```

## Challenges

### Asynchronous Processing
The service uses `go s.processNotification()` for async processing. Tests need to:
- Use `time.Sleep()` or channels to wait for completion
- Or refactor to make testable (dependency injection of executor)

### Multiple External Dependencies
- Firebase (push notifications)
- Twilio (SMS)
- Email client
- Database

All need proper mocking.

### Data Type Conversions
The service converts `map[string]interface{}` to `map[string]string` for Firebase.
Tests should verify these conversions.

## Next Steps

1. **Create Mock Implementations**
   - MockNotificationsRepository
   - MockFirebaseClient
   - MockTwilioClient
   - MockEmailClient

2. **Refactor Service for Testability**
   - Use interfaces instead of concrete types
   - Add NewServiceWithClients for production
   - Update cmd/notifications/main.go

3. **Write Comprehensive Tests**
   - Start with core SendNotification tests
   - Add channel-specific tests
   - Add ride event tests
   - Add error path tests

4. **Handle Async Testing**
   - Add synchronization mechanism
   - Or refactor to inject notification processor

5. **Integration Tests** (Optional)
   - Test with real Firebase (test project)
   - Test with Twilio test credentials
   - Test email with test SMTP server

## Estimated Effort

- Mock creation: 1 hour
- Service refactoring: 30 minutes
- Test writing: 3-4 hours
- **Total: ~5 hours**

## Benefits

Once completed:
- Confidence in notification delivery
- Easier to add new notification channels
- Clear documentation of behavior
- Regression protection
- Facilitates future refactoring

## Similar Services Completed

✅ **Auth Service**: 37.2% package coverage, ~90% service layer
✅ **Geo Service**: 59.2% package coverage, ~95% service layer
✅ **Payments Service**: 27.8% package coverage, ~100% service layer

The same patterns and infrastructure can be applied to Notifications.
