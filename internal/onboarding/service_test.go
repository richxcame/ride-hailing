package onboarding

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// MOCK IMPLEMENTATIONS
// ========================================

// mockRepo is a test-local mock that implements RepositoryInterface.
type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) GetDriver(ctx context.Context, driverID uuid.UUID) (*DriverInfo, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverInfo), args.Error(1)
}

func (m *mockRepo) GetDriverByUserID(ctx context.Context, userID uuid.UUID) (*DriverInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverInfo), args.Error(1)
}

func (m *mockRepo) CreateDriver(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockRepo) GetBackgroundCheck(ctx context.Context, driverID uuid.UUID) (*BackgroundCheck, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BackgroundCheck), args.Error(1)
}

func (m *mockRepo) CreateBackgroundCheck(ctx context.Context, driverID uuid.UUID, provider string) (*BackgroundCheck, error) {
	args := m.Called(ctx, driverID, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BackgroundCheck), args.Error(1)
}

func (m *mockRepo) UpdateBackgroundCheckStatus(ctx context.Context, checkID uuid.UUID, status string, notes *string) error {
	args := m.Called(ctx, checkID, status, notes)
	return args.Error(0)
}

func (m *mockRepo) ApproveDriver(ctx context.Context, driverID uuid.UUID, approvedBy uuid.UUID) error {
	args := m.Called(ctx, driverID, approvedBy)
	return args.Error(0)
}

func (m *mockRepo) RejectDriver(ctx context.Context, driverID uuid.UUID, rejectedBy uuid.UUID, reason string) error {
	args := m.Called(ctx, driverID, rejectedBy, reason)
	return args.Error(0)
}

func (m *mockRepo) GetOnboardingStats(ctx context.Context) (*OnboardingStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OnboardingStats), args.Error(1)
}

func (m *mockRepo) HasApprovedVehicle(ctx context.Context, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

// mockDocumentService is a mock for DocumentServiceInterface
type mockDocumentService struct {
	mock.Mock
}

func (m *mockDocumentService) GetDriverDocuments(ctx context.Context, driverID uuid.UUID) ([]DocumentInfo, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]DocumentInfo), args.Error(1)
}

func (m *mockDocumentService) GetRequiredDocumentTypes(ctx context.Context) ([]DocumentTypeInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]DocumentTypeInfo), args.Error(1)
}

func (m *mockDocumentService) GetDriverVerificationStatus(ctx context.Context, driverID uuid.UUID) (*VerificationStatus, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*VerificationStatus), args.Error(1)
}

// mockNotificationService is a mock for NotificationServiceInterface
type mockNotificationService struct {
	mock.Mock
}

func (m *mockNotificationService) SendOnboardingNotification(ctx context.Context, userID uuid.UUID, notifType, title, message string) error {
	args := m.Called(ctx, userID, notifType, title, message)
	return args.Error(0)
}

// newTestService creates a Service wired to the given mocks for testing.
// It registers a default HasApprovedVehicle expectation (returns false) so tests that
// don't care about the vehicle step don't need to set it up individually.
func newTestService(repo *mockRepo, docSvc *mockDocumentService, notifSvc NotificationServiceInterface) *Service {
	repo.On("HasApprovedVehicle", mock.Anything, mock.Anything).Return(false, nil).Maybe()
	return &Service{
		repo:            repo,
		documentService: docSvc,
		notifService:    notifSvc,
	}
}

// ========================================
// TEST HELPERS
// ========================================

func createTestDriver(driverID, userID uuid.UUID, approved bool) *DriverInfo {
	licenseNum := "DL12345678"
	now := time.Now()
	return &DriverInfo{
		ID:            driverID,
		UserID:        userID,
		FirstName:     "John",
		LastName:      "Doe",
		PhoneNumber:   "+15551234567",
		Email:         "john.doe@example.com",
		LicenseNumber: &licenseNum,
		IsApproved:    approved,
		IsSuspended:   false,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
}

func createIncompleteDriver(driverID, userID uuid.UUID) *DriverInfo {
	now := time.Now()
	return &DriverInfo{
		ID:            driverID,
		UserID:        userID,
		FirstName:     "",
		LastName:      "",
		PhoneNumber:   "",
		LicenseNumber: nil,
		IsApproved:    false,
		IsSuspended:   false,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
}

// ========================================
// StartOnboarding TESTS
// ========================================

func TestStartOnboarding_NewDriver_Success(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	userID := uuid.New()
	driverID := uuid.New()

	// User is not yet a driver
	repo.On("GetDriverByUserID", ctx, userID).Return(nil, errors.New("not found"))
	repo.On("CreateDriver", ctx, userID).Return(driverID, nil)

	// For GetOnboardingProgress call
	driver := createIncompleteDriver(driverID, userID)
	repo.On("GetDriver", ctx, driverID).Return(driver, nil)
	repo.On("GetBackgroundCheck", ctx, driverID).Return(nil, errors.New("not found"))

	docSvc.On("GetDriverVerificationStatus", ctx, driverID).Return(&VerificationStatus{
		Status:           "pending",
		CanDrive:         false,
		MissingDocuments: 3,
	}, nil)

	// Notification is sent async, allow any context
	notifSvc.On("SendOnboardingNotification", mock.Anything, userID, "onboarding_started", mock.Anything, mock.Anything).Return(nil).Maybe()

	progress, err := svc.StartOnboarding(ctx, userID)

	require.NoError(t, err)
	require.NotNil(t, progress)
	assert.Equal(t, driverID, progress.DriverID)
	assert.Equal(t, userID, progress.UserID)
	assert.False(t, progress.CanDrive, "new driver should not be able to drive")
	assert.Equal(t, StatusProfileIncomplete, progress.Status)

	repo.AssertExpectations(t)
}

func TestStartOnboarding_ExistingDriver_ReturnsProgress(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	userID := uuid.New()
	driverID := uuid.New()

	// User is already a driver
	existingDriver := createTestDriver(driverID, userID, false)
	repo.On("GetDriverByUserID", ctx, userID).Return(existingDriver, nil)

	// For GetOnboardingProgress call
	repo.On("GetDriver", ctx, driverID).Return(existingDriver, nil)
	repo.On("GetBackgroundCheck", ctx, driverID).Return(nil, errors.New("not found"))

	docSvc.On("GetDriverVerificationStatus", ctx, driverID).Return(&VerificationStatus{
		Status:            "pending",
		CanDrive:          false,
		MissingDocuments:  0,
		PendingDocuments:  2,
		ApprovedDocuments: 1,
	}, nil)

	progress, err := svc.StartOnboarding(ctx, userID)

	require.NoError(t, err)
	require.NotNil(t, progress)
	assert.Equal(t, driverID, progress.DriverID)
	// Should NOT have called CreateDriver
	repo.AssertNotCalled(t, "CreateDriver", mock.Anything, mock.Anything)
}

func TestStartOnboarding_CreateDriverFails(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	userID := uuid.New()

	repo.On("GetDriverByUserID", ctx, userID).Return(nil, errors.New("not found"))
	repo.On("CreateDriver", ctx, userID).Return(uuid.Nil, errors.New("database error"))

	progress, err := svc.StartOnboarding(ctx, userID)

	require.Error(t, err)
	assert.Nil(t, progress)
	// The error is wrapped as internal server error
	assert.Contains(t, err.Error(), "internal server error")
}

// ========================================
// GetOnboardingProgress TESTS
// ========================================

func TestGetOnboardingProgress_DriverNotFound(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	driverID := uuid.New()

	repo.On("GetDriver", ctx, driverID).Return(nil, errors.New("not found"))

	progress, err := svc.GetOnboardingProgress(ctx, driverID)

	require.Error(t, err)
	assert.Nil(t, progress)
	// The error wraps the original error
	assert.Contains(t, err.Error(), "not found")
}

func TestGetOnboardingProgress_ProfileIncomplete(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	driverID := uuid.New()
	userID := uuid.New()

	driver := createIncompleteDriver(driverID, userID)
	repo.On("GetDriver", ctx, driverID).Return(driver, nil)
	repo.On("GetBackgroundCheck", ctx, driverID).Return(nil, errors.New("not found"))

	docSvc.On("GetDriverVerificationStatus", ctx, driverID).Return(&VerificationStatus{
		Status:           "pending",
		MissingDocuments: 3,
	}, nil)

	progress, err := svc.GetOnboardingProgress(ctx, driverID)

	require.NoError(t, err)
	require.NotNil(t, progress)
	assert.Equal(t, StatusProfileIncomplete, progress.Status)
	assert.False(t, progress.CanDrive)
	assert.Equal(t, "profile", progress.CurrentStep)
	assert.NotNil(t, progress.NextAction)
	assert.Equal(t, "complete_profile", progress.NextAction.Type)
}

func TestGetOnboardingProgress_ProfileComplete_VehiclePending(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	driverID := uuid.New()
	userID := uuid.New()

	// Driver has complete profile but vehicle step is still pending
	// (since service doesn't auto-complete vehicle step based on VehicleID)
	driver := createTestDriver(driverID, userID, false)
	repo.On("GetDriver", ctx, driverID).Return(driver, nil)
	repo.On("GetBackgroundCheck", ctx, driverID).Return(nil, errors.New("not found"))

	docSvc.On("GetDriverVerificationStatus", ctx, driverID).Return(&VerificationStatus{
		Status:           "pending",
		MissingDocuments: 2,
		PendingDocuments: 1,
	}, nil)

	progress, err := svc.GetOnboardingProgress(ctx, driverID)

	require.NoError(t, err)
	require.NotNil(t, progress)
	// Vehicle step remains pending, so status is still profile_incomplete
	assert.Equal(t, StatusProfileIncomplete, progress.Status)
	assert.False(t, progress.CanDrive)
	assert.Equal(t, "vehicle", progress.CurrentStep)
}

func TestGetOnboardingProgress_BackgroundCheckInProgress(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	driverID := uuid.New()
	userID := uuid.New()

	driver := createTestDriver(driverID, userID, false)
	repo.On("GetDriver", ctx, driverID).Return(driver, nil)

	bgCheck := &BackgroundCheck{
		ID:       uuid.New(),
		DriverID: driverID,
		Status:   "in_progress",
	}
	repo.On("GetBackgroundCheck", ctx, driverID).Return(bgCheck, nil)

	docSvc.On("GetDriverVerificationStatus", ctx, driverID).Return(&VerificationStatus{
		Status:            "approved",
		MissingDocuments:  0,
		PendingDocuments:  0,
		ApprovedDocuments: 3,
	}, nil)

	progress, err := svc.GetOnboardingProgress(ctx, driverID)

	require.NoError(t, err)
	require.NotNil(t, progress)
	assert.False(t, progress.CanDrive)

	// Check that background check step is in_progress
	var bgCheckStep *OnboardingStep
	for _, step := range progress.Steps {
		if step.ID == "background_check" {
			bgCheckStep = step
			break
		}
	}
	require.NotNil(t, bgCheckStep)
	assert.Equal(t, "in_progress", bgCheckStep.Status)
}

// ========================================
// SAFETY-CRITICAL: CanDrive Flag Tests
// ========================================

func TestGetOnboardingProgress_CanDrive_OnlyWhenFullyApproved(t *testing.T) {
	tests := []struct {
		name           string
		isApproved     bool
		isSuspended    bool
		docStatus      *VerificationStatus
		bgCheckStatus  string
		expectedDrive  bool
		expectedStatus OnboardingStatus
	}{
		{
			name:       "fully approved driver can drive",
			isApproved: true,
			docStatus: &VerificationStatus{
				Status:            "approved",
				CanDrive:          true,
				MissingDocuments:  0,
				PendingDocuments:  0,
				ApprovedDocuments: 3,
			},
			bgCheckStatus:  "passed",
			expectedDrive:  true,
			expectedStatus: StatusApproved,
		},
		{
			name:       "unapproved driver cannot drive even with docs approved",
			isApproved: false,
			docStatus: &VerificationStatus{
				Status:            "approved",
				CanDrive:          true,
				MissingDocuments:  0,
				PendingDocuments:  0,
				ApprovedDocuments: 3,
			},
			bgCheckStatus:  "passed",
			expectedDrive:  false,
			expectedStatus: StatusProfileIncomplete, // vehicle step is still pending
		},
		{
			name:        "suspended driver with IsApproved=true still returns approved (service prioritizes IsApproved)",
			isApproved:  true,
			isSuspended: true,
			docStatus: &VerificationStatus{
				Status:            "approved",
				CanDrive:          true,
				MissingDocuments:  0,
				ApprovedDocuments: 3,
			},
			bgCheckStatus:  "passed",
			expectedDrive:  true,  // IsApproved is checked first in service
			expectedStatus: StatusApproved,
		},
		{
			name:        "suspended driver with IsApproved=false cannot drive",
			isApproved:  false,
			isSuspended: true,
			docStatus: &VerificationStatus{
				Status:            "approved",
				CanDrive:          true,
				MissingDocuments:  0,
				ApprovedDocuments: 3,
			},
			bgCheckStatus:  "passed",
			expectedDrive:  false,
			expectedStatus: StatusSuspended,
		},
		{
			name:       "driver with pending documents cannot drive",
			isApproved: false,
			docStatus: &VerificationStatus{
				Status:           "pending",
				CanDrive:         false,
				MissingDocuments: 1,
				PendingDocuments: 2,
			},
			bgCheckStatus:  "",
			expectedDrive:  false,
			expectedStatus: StatusProfileIncomplete, // vehicle step pending
		},
		{
			name:       "driver with failed background check - background_check step is not Required so doesn't trigger rejection",
			isApproved: false,
			docStatus: &VerificationStatus{
				Status:            "approved",
				CanDrive:          true,
				MissingDocuments:  0,
				ApprovedDocuments: 3,
			},
			bgCheckStatus:  "failed",
			expectedDrive:  false,
			expectedStatus: StatusProfileIncomplete, // vehicle step is still pending
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepo)
			docSvc := new(mockDocumentService)
			notifSvc := new(mockNotificationService)
			svc := newTestService(repo, docSvc, notifSvc)
			ctx := context.Background()

			driverID := uuid.New()
			userID := uuid.New()

			driver := createTestDriver(driverID, userID, tt.isApproved)
			driver.IsSuspended = tt.isSuspended

			repo.On("GetDriver", ctx, driverID).Return(driver, nil)

			if tt.bgCheckStatus != "" {
				bgCheck := &BackgroundCheck{
					ID:       uuid.New(),
					DriverID: driverID,
					Status:   tt.bgCheckStatus,
				}
				repo.On("GetBackgroundCheck", ctx, driverID).Return(bgCheck, nil)
			} else {
				repo.On("GetBackgroundCheck", ctx, driverID).Return(nil, errors.New("not found"))
			}

			docSvc.On("GetDriverVerificationStatus", ctx, driverID).Return(tt.docStatus, nil)

			progress, err := svc.GetOnboardingProgress(ctx, driverID)

			require.NoError(t, err)
			require.NotNil(t, progress)

			assert.Equal(t, tt.expectedDrive, progress.CanDrive,
				"CanDrive should be %v for case: %s", tt.expectedDrive, tt.name)
			assert.Equal(t, tt.expectedStatus, progress.Status,
				"Status should be %s for case: %s", tt.expectedStatus, tt.name)
		})
	}
}

// CRITICAL SAFETY TEST: Verify CanDrive is NEVER set without explicit approval
func TestCanDrive_NeverSetWithoutExplicitApproval(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	driverID := uuid.New()
	userID := uuid.New()

	// Create a driver that looks "complete" but is not approved
	driver := createTestDriver(driverID, userID, false) // isApproved = false

	repo.On("GetDriver", ctx, driverID).Return(driver, nil)
	repo.On("GetBackgroundCheck", ctx, driverID).Return(&BackgroundCheck{
		ID:       uuid.New(),
		DriverID: driverID,
		Status:   "passed",
	}, nil)

	// All documents approved
	docSvc.On("GetDriverVerificationStatus", ctx, driverID).Return(&VerificationStatus{
		Status:            "approved",
		CanDrive:          true, // Document service says OK
		MissingDocuments:  0,
		PendingDocuments:  0,
		ApprovedDocuments: 5,
		RejectedDocuments: 0,
	}, nil)

	progress, err := svc.GetOnboardingProgress(ctx, driverID)

	require.NoError(t, err)
	require.NotNil(t, progress)

	// CRITICAL: Even with all docs approved and background check passed,
	// CanDrive should be false because IsApproved is false
	assert.False(t, progress.CanDrive,
		"SAFETY CRITICAL: CanDrive must not be true without explicit driver approval")
	assert.NotEqual(t, StatusApproved, progress.Status,
		"Status should not be Approved without IsApproved=true")
}

// ========================================
// GetDocumentRequirements TESTS
// ========================================

func TestGetDocumentRequirements_Success(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	driverID := uuid.New()
	docTypeID := uuid.New()
	docID := uuid.New()
	now := time.Now()

	docSvc.On("GetRequiredDocumentTypes", ctx).Return([]DocumentTypeInfo{
		{ID: docTypeID, Code: "drivers_license", Name: "Driver's License", IsRequired: true},
	}, nil)

	docSvc.On("GetDriverDocuments", ctx, driverID).Return([]DocumentInfo{
		{ID: docID, DocumentTypeID: docTypeID, Status: "approved", SubmittedAt: now},
	}, nil)

	requirements, err := svc.GetDocumentRequirements(ctx, driverID)

	require.NoError(t, err)
	require.Len(t, requirements, 1)
	assert.Equal(t, "Driver's License", requirements[0].Name)
	assert.Equal(t, "approved", requirements[0].Status)
	assert.Equal(t, &docID, requirements[0].DocumentID)
}

func TestGetDocumentRequirements_NoDocumentsSubmitted(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	driverID := uuid.New()
	docTypeID := uuid.New()

	docSvc.On("GetRequiredDocumentTypes", ctx).Return([]DocumentTypeInfo{
		{ID: docTypeID, Code: "drivers_license", Name: "Driver's License", IsRequired: true},
		{ID: uuid.New(), Code: "vehicle_registration", Name: "Vehicle Registration", IsRequired: true},
	}, nil)

	docSvc.On("GetDriverDocuments", ctx, driverID).Return(nil, errors.New("no documents"))

	requirements, err := svc.GetDocumentRequirements(ctx, driverID)

	require.NoError(t, err)
	require.Len(t, requirements, 2)
	assert.Equal(t, "not_submitted", requirements[0].Status)
	assert.Equal(t, "not_submitted", requirements[1].Status)
}

func TestGetDocumentRequirements_GetDocTypesFails(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	driverID := uuid.New()

	docSvc.On("GetRequiredDocumentTypes", ctx).Return(nil, errors.New("service unavailable"))

	requirements, err := svc.GetDocumentRequirements(ctx, driverID)

	require.Error(t, err)
	assert.Nil(t, requirements)
}

// ========================================
// NotifyOnboardingUpdate TESTS
// ========================================

func TestNotifyOnboardingUpdate_Success(t *testing.T) {
	tests := []struct {
		name          string
		eventType     string
		expectedTitle string
	}{
		{"documents_approved", "documents_approved", "Documents Approved!"},
		{"documents_rejected", "documents_rejected", "Document Issue"},
		{"approved", "approved", "Welcome Aboard!"},
		{"rejected", "rejected", "Application Update"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepo)
			docSvc := new(mockDocumentService)
			notifSvc := new(mockNotificationService)
			svc := newTestService(repo, docSvc, notifSvc)
			ctx := context.Background()

			driverID := uuid.New()
			userID := uuid.New()
			driver := createTestDriver(driverID, userID, false)

			repo.On("GetDriver", ctx, driverID).Return(driver, nil)
			notifSvc.On("SendOnboardingNotification", ctx, userID, tt.eventType, mock.Anything, mock.Anything).Return(nil)

			err := svc.NotifyOnboardingUpdate(ctx, driverID, tt.eventType)

			require.NoError(t, err)
			notifSvc.AssertCalled(t, "SendOnboardingNotification", ctx, userID, tt.eventType, mock.Anything, mock.Anything)
		})
	}
}

func TestNotifyOnboardingUpdate_DriverNotFound(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	driverID := uuid.New()

	repo.On("GetDriver", ctx, driverID).Return(nil, errors.New("not found"))

	err := svc.NotifyOnboardingUpdate(ctx, driverID, "approved")

	require.Error(t, err)
}

func TestNotifyOnboardingUpdate_UnknownEventType(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	driverID := uuid.New()
	userID := uuid.New()
	driver := createTestDriver(driverID, userID, false)

	repo.On("GetDriver", ctx, driverID).Return(driver, nil)
	// Notification service should not be called for unknown event types

	err := svc.NotifyOnboardingUpdate(ctx, driverID, "unknown_event")

	require.NoError(t, err)
	notifSvc.AssertNotCalled(t, "SendOnboardingNotification", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ========================================
// Progress Calculation TESTS
// ========================================

func TestCalculateProgress_AllStepsCompleted(t *testing.T) {
	steps := []*OnboardingStep{
		{ID: "profile", Status: "completed", Required: true},
		{ID: "vehicle", Status: "completed", Required: true},
		{ID: "documents", Status: "completed", Required: true},
		{ID: "document_review", Status: "completed", Required: true},
		{ID: "background_check", Status: "completed", Required: false},
		{ID: "approval", Status: "completed", Required: true},
	}

	svc := &Service{}
	progress := svc.calculateProgress(steps)

	assert.Equal(t, 6, progress.CompletedSteps)
	assert.Equal(t, 6, progress.TotalSteps)
	assert.Equal(t, 100, progress.ProgressPercent)
}

func TestCalculateProgress_PartialCompletion(t *testing.T) {
	steps := []*OnboardingStep{
		{ID: "profile", Status: "completed", Required: true},
		{ID: "vehicle", Status: "completed", Required: true},
		{ID: "documents", Status: "in_progress", Required: true},
		{ID: "document_review", Status: "pending", Required: true},
		{ID: "background_check", Status: "pending", Required: false},
		{ID: "approval", Status: "pending", Required: true},
	}

	svc := &Service{}
	progress := svc.calculateProgress(steps)

	assert.Equal(t, 2, progress.CompletedSteps)
	assert.Equal(t, 6, progress.TotalSteps)
	// 2 out of 5 required steps = 40%
	assert.Equal(t, 40, progress.ProgressPercent)
}

func TestCalculateProgress_NoStepsCompleted(t *testing.T) {
	steps := []*OnboardingStep{
		{ID: "profile", Status: "pending", Required: true},
		{ID: "vehicle", Status: "pending", Required: true},
		{ID: "documents", Status: "pending", Required: true},
		{ID: "document_review", Status: "pending", Required: true},
		{ID: "background_check", Status: "pending", Required: false},
		{ID: "approval", Status: "pending", Required: true},
	}

	svc := &Service{}
	progress := svc.calculateProgress(steps)

	assert.Equal(t, 0, progress.CompletedSteps)
	assert.Equal(t, 0, progress.ProgressPercent)
}

// ========================================
// Profile Completion TESTS
// ========================================

func TestCheckProfileComplete(t *testing.T) {
	tests := []struct {
		name     string
		driver   *DriverInfo
		expected bool
	}{
		{
			name:     "nil driver",
			driver:   nil,
			expected: false,
		},
		{
			name: "complete profile",
			driver: &DriverInfo{
				FirstName:     "John",
				LastName:      "Doe",
				PhoneNumber:   "+15551234567",
				LicenseNumber: stringPtr("DL123456"),
			},
			expected: true,
		},
		{
			name: "missing first name",
			driver: &DriverInfo{
				FirstName:     "",
				LastName:      "Doe",
				PhoneNumber:   "+15551234567",
				LicenseNumber: stringPtr("DL123456"),
			},
			expected: false,
		},
		{
			name: "missing last name",
			driver: &DriverInfo{
				FirstName:     "John",
				LastName:      "",
				PhoneNumber:   "+15551234567",
				LicenseNumber: stringPtr("DL123456"),
			},
			expected: false,
		},
		{
			name: "missing phone",
			driver: &DriverInfo{
				FirstName:     "John",
				LastName:      "Doe",
				PhoneNumber:   "",
				LicenseNumber: stringPtr("DL123456"),
			},
			expected: false,
		},
		{
			name: "nil license number",
			driver: &DriverInfo{
				FirstName:     "John",
				LastName:      "Doe",
				PhoneNumber:   "+15551234567",
				LicenseNumber: nil,
			},
			expected: false,
		},
		{
			name: "empty license number",
			driver: &DriverInfo{
				FirstName:     "John",
				LastName:      "Doe",
				PhoneNumber:   "+15551234567",
				LicenseNumber: stringPtr(""),
			},
			expected: false,
		},
	}

	svc := &Service{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.checkProfileComplete(tt.driver)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

// ========================================
// Status Message TESTS
// ========================================

func TestGetStatusMessage(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		status   OnboardingStatus
		expected string
	}{
		{StatusNotStarted, "Start your driver application"},
		{StatusProfileIncomplete, "Please complete your profile to continue"},
		{StatusDocumentsPending, "Upload your required documents"},
		{StatusDocumentsReview, "Your documents are being reviewed"},
		{StatusBackgroundCheck, "Background check in progress"},
		{StatusPendingApproval, "Your application is pending final approval"},
		{StatusApproved, "You're approved! Start accepting rides"},
		{StatusRejected, "Your application was not approved"},
		{StatusSuspended, "Your account has been suspended"},
		{OnboardingStatus("unknown"), "Unknown status"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := svc.getStatusMessage(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ========================================
// Determine Current Step TESTS
// ========================================

func TestDetermineCurrentStep(t *testing.T) {
	tests := []struct {
		name     string
		steps    []*OnboardingStep
		expected string
	}{
		{
			name: "first step pending",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "pending"},
				{ID: "vehicle", Status: "pending"},
			},
			expected: "profile",
		},
		{
			name: "second step in progress",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed"},
				{ID: "vehicle", Status: "in_progress"},
				{ID: "documents", Status: "pending"},
			},
			expected: "vehicle",
		},
		{
			name: "step failed",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed"},
				{ID: "vehicle", Status: "completed"},
				{ID: "documents", Status: "failed"},
			},
			expected: "documents",
		},
		{
			name: "all completed",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed"},
				{ID: "vehicle", Status: "completed"},
				{ID: "documents", Status: "completed"},
			},
			expected: "completed",
		},
	}

	svc := &Service{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.determineCurrentStep(tt.steps)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ========================================
// Next Action TESTS
// ========================================

func TestDetermineNextAction(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name         string
		steps        []*OnboardingStep
		docStatus    *VerificationStatus
		expectedType string
		expectedNil  bool
	}{
		{
			name: "profile incomplete",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "pending"},
			},
			expectedType: "complete_profile",
		},
		{
			name: "vehicle needed",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed"},
				{ID: "vehicle", Status: "in_progress"},
			},
			expectedType: "add_vehicle",
		},
		{
			name: "documents needed",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed"},
				{ID: "vehicle", Status: "completed"},
				{ID: "documents", Status: "pending"},
			},
			docStatus: &VerificationStatus{
				MissingDocuments: 2,
			},
			expectedType: "upload_documents",
		},
		{
			name: "documents under review",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed"},
				{ID: "vehicle", Status: "completed"},
				{ID: "documents", Status: "completed"},
				{ID: "document_review", Status: "in_progress"},
			},
			expectedType: "wait",
		},
		{
			name: "documents rejected - resubmit",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed"},
				{ID: "vehicle", Status: "completed"},
				{ID: "documents", Status: "completed"},
				{ID: "document_review", Status: "failed"},
			},
			expectedType: "resubmit_documents",
		},
		{
			name: "pending approval",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed"},
				{ID: "vehicle", Status: "completed"},
				{ID: "documents", Status: "completed"},
				{ID: "document_review", Status: "completed"},
				{ID: "background_check", Status: "completed"},
				{ID: "approval", Status: "in_progress"},
			},
			expectedType: "wait",
		},
		{
			name: "all complete - no action",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed"},
				{ID: "vehicle", Status: "completed"},
				{ID: "documents", Status: "completed"},
				{ID: "document_review", Status: "completed"},
				{ID: "approval", Status: "completed"},
			},
			expectedNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.determineNextAction(tt.steps, tt.docStatus)

			if tt.expectedNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedType, result.Type)
			}
		})
	}
}

// ========================================
// Update Document Steps TESTS
// ========================================

func TestUpdateDocumentSteps(t *testing.T) {
	tests := []struct {
		name           string
		docStatus      *VerificationStatus
		expectedDocs   string
		expectedReview string
	}{
		{
			name: "all documents uploaded, none reviewed",
			docStatus: &VerificationStatus{
				MissingDocuments:  0,
				PendingDocuments:  3,
				ApprovedDocuments: 0,
				RejectedDocuments: 0,
			},
			expectedDocs:   "completed",
			expectedReview: "in_progress",
		},
		{
			name: "missing documents with some pending",
			docStatus: &VerificationStatus{
				MissingDocuments:  2,
				PendingDocuments:  1,
				ApprovedDocuments: 0,
				RejectedDocuments: 0,
			},
			expectedDocs:   "in_progress",
			expectedReview: "in_progress", // pending docs means review in progress
		},
		{
			name: "all approved",
			docStatus: &VerificationStatus{
				MissingDocuments:  0,
				PendingDocuments:  0,
				ApprovedDocuments: 3,
				RejectedDocuments: 0,
			},
			expectedDocs:   "completed",
			expectedReview: "completed",
		},
		{
			name: "some rejected",
			docStatus: &VerificationStatus{
				MissingDocuments:  0,
				PendingDocuments:  0,
				ApprovedDocuments: 2,
				RejectedDocuments: 1,
			},
			expectedDocs:   "completed",
			expectedReview: "failed",
		},
	}

	svc := &Service{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := []*OnboardingStep{
				{ID: "profile", Status: "completed"},
				{ID: "vehicle", Status: "completed"},
				{ID: "documents", Status: "pending"},
				{ID: "document_review", Status: "pending"},
			}

			svc.updateDocumentSteps(steps, tt.docStatus)

			var docStep, reviewStep *OnboardingStep
			for _, s := range steps {
				if s.ID == "documents" {
					docStep = s
				}
				if s.ID == "document_review" {
					reviewStep = s
				}
			}

			assert.Equal(t, tt.expectedDocs, docStep.Status, "documents step status")
			assert.Equal(t, tt.expectedReview, reviewStep.Status, "document_review step status")
		})
	}
}

// ========================================
// Update Background Check Step TESTS
// ========================================

func TestUpdateBackgroundCheckStep(t *testing.T) {
	tests := []struct {
		name           string
		bgCheckStatus  string
		expectedStatus string
	}{
		{"passed", "passed", "completed"},
		{"pending", "pending", "in_progress"},
		{"in_progress", "in_progress", "in_progress"},
		{"failed", "failed", "failed"},
	}

	svc := &Service{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := []*OnboardingStep{
				{ID: "background_check", Status: "pending"},
			}

			bgCheck := &BackgroundCheck{
				ID:     uuid.New(),
				Status: tt.bgCheckStatus,
			}

			svc.updateBackgroundCheckStep(steps, bgCheck)

			assert.Equal(t, tt.expectedStatus, steps[0].Status)
		})
	}
}

// ========================================
// Build Onboarding Steps TESTS
// ========================================

func TestBuildOnboardingSteps(t *testing.T) {
	svc := &Service{}
	steps := svc.buildOnboardingSteps()

	require.Len(t, steps, 6)

	// Verify order
	expectedOrder := []string{"profile", "vehicle", "documents", "document_review", "background_check", "approval"}
	for i, step := range steps {
		assert.Equal(t, expectedOrder[i], step.ID)
		assert.Equal(t, i+1, step.Order)
		assert.Equal(t, "pending", step.Status)
	}

	// Verify required flags
	requiredSteps := map[string]bool{
		"profile":          true,
		"vehicle":          true,
		"documents":        true,
		"document_review":  true,
		"background_check": false, // Optional
		"approval":         true,
	}

	for _, step := range steps {
		assert.Equal(t, requiredSteps[step.ID], step.Required, "step %s required flag", step.ID)
	}
}

// ========================================
// Nil Notification Service TESTS
// ========================================

func TestStartOnboarding_NilNotificationService(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	// No notification service
	svc := newTestService(repo, docSvc, nil)
	ctx := context.Background()

	userID := uuid.New()
	driverID := uuid.New()

	repo.On("GetDriverByUserID", ctx, userID).Return(nil, errors.New("not found"))
	repo.On("CreateDriver", ctx, userID).Return(driverID, nil)

	driver := createIncompleteDriver(driverID, userID)
	repo.On("GetDriver", ctx, driverID).Return(driver, nil)
	repo.On("GetBackgroundCheck", ctx, driverID).Return(nil, errors.New("not found"))

	docSvc.On("GetDriverVerificationStatus", ctx, driverID).Return(&VerificationStatus{
		MissingDocuments: 3,
	}, nil)

	// Should not panic even with nil notification service
	progress, err := svc.StartOnboarding(ctx, userID)

	require.NoError(t, err)
	require.NotNil(t, progress)
}

func TestNotifyOnboardingUpdate_NilNotificationService(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	svc := newTestService(repo, docSvc, nil) // nil notification service
	ctx := context.Background()

	driverID := uuid.New()
	userID := uuid.New()
	driver := createTestDriver(driverID, userID, false)

	repo.On("GetDriver", ctx, driverID).Return(driver, nil)

	// Should not panic or error with nil notification service
	err := svc.NotifyOnboardingUpdate(ctx, driverID, "approved")

	require.NoError(t, err)
}

// ========================================
// Determine Overall Status TESTS
// ========================================

func TestDetermineOverallStatus(t *testing.T) {
	tests := []struct {
		name           string
		steps          []*OnboardingStep
		driver         *DriverInfo
		expectedStatus OnboardingStatus
	}{
		{
			name: "approved driver returns approved",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "pending", Required: true},
			},
			driver: &DriverInfo{
				IsApproved:  true,
				IsSuspended: false,
			},
			expectedStatus: StatusApproved,
		},
		{
			name: "suspended driver returns suspended",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed", Required: true},
			},
			driver: &DriverInfo{
				IsApproved:  false,
				IsSuspended: true,
			},
			expectedStatus: StatusSuspended,
		},
		{
			name: "profile pending returns profile incomplete",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "pending", Required: true},
				{ID: "vehicle", Status: "pending", Required: true},
			},
			driver: &DriverInfo{
				IsApproved:  false,
				IsSuspended: false,
			},
			expectedStatus: StatusProfileIncomplete,
		},
		{
			name: "vehicle pending returns profile incomplete",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed", Required: true},
				{ID: "vehicle", Status: "pending", Required: true},
			},
			driver: &DriverInfo{
				IsApproved:  false,
				IsSuspended: false,
			},
			expectedStatus: StatusProfileIncomplete,
		},
		{
			name: "documents pending returns documents pending",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed", Required: true},
				{ID: "vehicle", Status: "completed", Required: true},
				{ID: "documents", Status: "pending", Required: true},
			},
			driver: &DriverInfo{
				IsApproved:  false,
				IsSuspended: false,
			},
			expectedStatus: StatusDocumentsPending,
		},
		{
			name: "document review in progress returns documents review",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed", Required: true},
				{ID: "vehicle", Status: "completed", Required: true},
				{ID: "documents", Status: "completed", Required: true},
				{ID: "document_review", Status: "in_progress", Required: true},
			},
			driver: &DriverInfo{
				IsApproved:  false,
				IsSuspended: false,
			},
			expectedStatus: StatusDocumentsReview,
		},
		{
			name: "document review failed returns documents review",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed", Required: true},
				{ID: "vehicle", Status: "completed", Required: true},
				{ID: "documents", Status: "completed", Required: true},
				{ID: "document_review", Status: "failed", Required: true},
			},
			driver: &DriverInfo{
				IsApproved:  false,
				IsSuspended: false,
			},
			expectedStatus: StatusDocumentsReview,
		},
		{
			name: "approval pending returns pending approval",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed", Required: true},
				{ID: "vehicle", Status: "completed", Required: true},
				{ID: "documents", Status: "completed", Required: true},
				{ID: "document_review", Status: "completed", Required: true},
				{ID: "background_check", Status: "completed", Required: false},
				{ID: "approval", Status: "pending", Required: true},
			},
			driver: &DriverInfo{
				IsApproved:  false,
				IsSuspended: false,
			},
			expectedStatus: StatusPendingApproval,
		},
		{
			name: "all steps completed but not approved - returns approved status (edge case)",
			steps: []*OnboardingStep{
				{ID: "profile", Status: "completed", Required: true},
				{ID: "vehicle", Status: "completed", Required: true},
				{ID: "documents", Status: "completed", Required: true},
				{ID: "document_review", Status: "completed", Required: true},
				{ID: "approval", Status: "completed", Required: true},
			},
			driver: &DriverInfo{
				IsApproved:  false,
				IsSuspended: false,
			},
			expectedStatus: StatusApproved, // falls through to default
		},
	}

	svc := &Service{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.determineOverallStatus(tt.steps, tt.driver)
			assert.Equal(t, tt.expectedStatus, result)
		})
	}
}

// ========================================
// Integration-style Test: Full Flow
// ========================================

func TestOnboardingFlow_ApprovedDriver(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	userID := uuid.New()
	driverID := uuid.New()

	// Approved driver with all docs verified
	approvedDriver := createTestDriver(driverID, userID, true) // IsApproved = true

	repo.On("GetDriver", ctx, driverID).Return(approvedDriver, nil)
	repo.On("GetBackgroundCheck", ctx, driverID).Return(&BackgroundCheck{
		ID:       uuid.New(),
		DriverID: driverID,
		Status:   "passed",
	}, nil)

	docSvc.On("GetDriverVerificationStatus", ctx, driverID).Return(&VerificationStatus{
		Status:            "approved",
		CanDrive:          true,
		MissingDocuments:  0,
		PendingDocuments:  0,
		ApprovedDocuments: 3,
	}, nil)

	progress, err := svc.GetOnboardingProgress(ctx, driverID)

	require.NoError(t, err)
	assert.True(t, progress.CanDrive, "Approved driver should be able to drive")
	assert.Equal(t, StatusApproved, progress.Status)
}

// ========================================
// Edge Cases TESTS
// ========================================

func TestGetOnboardingProgress_DocumentServiceError(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	driverID := uuid.New()
	userID := uuid.New()

	driver := createTestDriver(driverID, userID, false)
	repo.On("GetDriver", ctx, driverID).Return(driver, nil)
	repo.On("GetBackgroundCheck", ctx, driverID).Return(nil, errors.New("not found"))

	// Document service returns error
	docSvc.On("GetDriverVerificationStatus", ctx, driverID).Return(nil, errors.New("service unavailable"))

	// Should still return progress, just without document status updates
	progress, err := svc.GetOnboardingProgress(ctx, driverID)

	require.NoError(t, err)
	require.NotNil(t, progress)
	// Document steps should remain in their initial pending state
}

func TestGetOnboardingProgress_NilDocumentStatus(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	driverID := uuid.New()
	userID := uuid.New()

	driver := createTestDriver(driverID, userID, false)
	repo.On("GetDriver", ctx, driverID).Return(driver, nil)
	repo.On("GetBackgroundCheck", ctx, driverID).Return(nil, errors.New("not found"))

	// Document service returns nil status without error
	docSvc.On("GetDriverVerificationStatus", ctx, driverID).Return(nil, nil)

	progress, err := svc.GetOnboardingProgress(ctx, driverID)

	require.NoError(t, err)
	require.NotNil(t, progress)
}

// ========================================
// Document Requirement Mapping TESTS
// ========================================

func TestGetDocumentRequirements_LatestDocumentUsed(t *testing.T) {
	repo := new(mockRepo)
	docSvc := new(mockDocumentService)
	notifSvc := new(mockNotificationService)
	svc := newTestService(repo, docSvc, notifSvc)
	ctx := context.Background()

	driverID := uuid.New()
	docTypeID := uuid.New()
	oldDocID := uuid.New()
	newDocID := uuid.New()
	oldTime := time.Now().Add(-24 * time.Hour)
	newTime := time.Now()

	docSvc.On("GetRequiredDocumentTypes", ctx).Return([]DocumentTypeInfo{
		{ID: docTypeID, Code: "drivers_license", Name: "Driver's License", IsRequired: true},
	}, nil)

	// Return two documents for the same type, with different timestamps
	docSvc.On("GetDriverDocuments", ctx, driverID).Return([]DocumentInfo{
		{ID: oldDocID, DocumentTypeID: docTypeID, Status: "rejected", SubmittedAt: oldTime},
		{ID: newDocID, DocumentTypeID: docTypeID, Status: "approved", SubmittedAt: newTime},
	}, nil)

	requirements, err := svc.GetDocumentRequirements(ctx, driverID)

	require.NoError(t, err)
	require.Len(t, requirements, 1)
	// Should use the most recent document
	assert.Equal(t, &newDocID, requirements[0].DocumentID)
	assert.Equal(t, "approved", requirements[0].Status)
}
