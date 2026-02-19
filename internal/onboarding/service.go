package onboarding

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// OnboardingStatus represents the driver onboarding status
type OnboardingStatus string

const (
	StatusNotStarted       OnboardingStatus = "not_started"
	StatusProfileIncomplete OnboardingStatus = "profile_incomplete"
	StatusDocumentsPending  OnboardingStatus = "documents_pending"
	StatusDocumentsReview   OnboardingStatus = "documents_under_review"
	StatusBackgroundCheck   OnboardingStatus = "background_check"
	StatusPendingApproval   OnboardingStatus = "pending_approval"
	StatusApproved          OnboardingStatus = "approved"
	StatusRejected          OnboardingStatus = "rejected"
	StatusSuspended         OnboardingStatus = "suspended"
)

// OnboardingStep represents a step in the onboarding process
type OnboardingStep struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Status      string           `json:"status"` // pending, in_progress, completed, failed
	Required    bool             `json:"required"`
	Order       int              `json:"order"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// OnboardingProgress represents the overall onboarding progress
type OnboardingProgress struct {
	DriverID        uuid.UUID         `json:"driver_id"`
	UserID          uuid.UUID         `json:"user_id"`
	Status          OnboardingStatus  `json:"status"`
	CurrentStep     string            `json:"current_step"`
	Steps           []*OnboardingStep `json:"steps"`
	CompletedSteps  int               `json:"completed_steps"`
	TotalSteps      int               `json:"total_steps"`
	ProgressPercent int               `json:"progress_percent"`
	CanDrive        bool              `json:"can_drive"`
	Message         string            `json:"message"`
	NextAction      *NextAction       `json:"next_action,omitempty"`
	StartedAt       *time.Time        `json:"started_at,omitempty"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
}

// NextAction represents the next required action for the driver
type NextAction struct {
	Type        string `json:"type"` // upload_document, complete_profile, wait, etc.
	Title       string `json:"title"`
	Description string `json:"description"`
	ActionURL   string `json:"action_url,omitempty"`
	Priority    string `json:"priority"` // high, medium, low
}

// DocumentRequirement represents a document requirement
type DocumentRequirement struct {
	DocumentTypeID   uuid.UUID  `json:"document_type_id"`
	DocumentTypeCode string     `json:"document_type_code"`
	Name             string     `json:"name"`
	Required         bool       `json:"required"`
	Status           string     `json:"status"` // not_submitted, pending, approved, rejected, expired
	DocumentID       *uuid.UUID `json:"document_id,omitempty"`
	SubmittedAt      *time.Time `json:"submitted_at,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	RejectionReason  *string    `json:"rejection_reason,omitempty"`
}

// Service handles driver onboarding logic
type Service struct {
	repo            RepositoryInterface
	documentService DocumentServiceInterface
	notifService    NotificationServiceInterface
}

// DocumentServiceInterface defines methods needed from document service
type DocumentServiceInterface interface {
	GetDriverDocuments(ctx context.Context, driverID uuid.UUID) ([]DocumentInfo, error)
	GetRequiredDocumentTypes(ctx context.Context) ([]DocumentTypeInfo, error)
	GetDriverVerificationStatus(ctx context.Context, driverID uuid.UUID) (*VerificationStatus, error)
}

// DocumentInfo represents document information
type DocumentInfo struct {
	ID             uuid.UUID  `json:"id"`
	DocumentTypeID uuid.UUID  `json:"document_type_id"`
	Status         string     `json:"status"`
	SubmittedAt    time.Time  `json:"submitted_at"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
	ExpiryDate     *time.Time `json:"expiry_date,omitempty"`
	RejectionReason *string   `json:"rejection_reason,omitempty"`
}

// DocumentTypeInfo represents document type information
type DocumentTypeInfo struct {
	ID         uuid.UUID `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	IsRequired bool      `json:"is_required"`
}

// VerificationStatus represents driver verification status
type VerificationStatus struct {
	Status             string `json:"status"`
	CanDrive           bool   `json:"can_drive"`
	MissingDocuments   int    `json:"missing_documents"`
	PendingDocuments   int    `json:"pending_documents"`
	ApprovedDocuments  int    `json:"approved_documents"`
	RejectedDocuments  int    `json:"rejected_documents"`
}

// NotificationServiceInterface defines methods needed from notification service
type NotificationServiceInterface interface {
	SendOnboardingNotification(ctx context.Context, userID uuid.UUID, notifType, title, message string) error
}

// NewService creates a new onboarding service
func NewService(repo RepositoryInterface, docService DocumentServiceInterface, notifService NotificationServiceInterface) *Service {
	return &Service{
		repo:            repo,
		documentService: docService,
		notifService:    notifService,
	}
}

// GetOnboardingProgress returns the current onboarding progress for a driver
func (s *Service) GetOnboardingProgress(ctx context.Context, driverID uuid.UUID) (*OnboardingProgress, error) {
	// Get driver info
	driver, err := s.repo.GetDriver(ctx, driverID)
	if err != nil {
		return nil, common.NewNotFoundError("driver not found", err)
	}

	// Initialize steps
	steps := s.buildOnboardingSteps()

	// Check profile completion
	profileComplete := s.checkProfileComplete(driver)
	if profileComplete {
		steps[0].Status = "completed"
		steps[0].CompletedAt = driver.UpdatedAt
	} else {
		steps[0].Status = "in_progress"
	}

	// Check vehicle step.
	// vehicles.driver_id is a FK to users.id (not drivers.id), so query by user_id.
	if hasVehicle, err := s.repo.HasApprovedVehicle(ctx, driver.UserID); err == nil && hasVehicle {
		steps[1].Status = "completed"
	}

	// Check document status
	docStatus, err := s.documentService.GetDriverVerificationStatus(ctx, driverID)
	if err == nil && docStatus != nil {
		s.updateDocumentSteps(steps, docStatus)
	}

	// Check background check (if applicable)
	bgCheck, _ := s.repo.GetBackgroundCheck(ctx, driverID)
	if bgCheck != nil {
		s.updateBackgroundCheckStep(steps, bgCheck)
	}

	// Check approval step
	if driver.IsApproved {
		steps[5].Status = "completed"
		steps[5].CompletedAt = driver.ApprovedAt
	}

	// Calculate overall progress
	progress := s.calculateProgress(steps)
	progress.DriverID = driverID
	progress.UserID = driver.UserID

	// Determine current step and next action
	progress.CurrentStep = s.determineCurrentStep(steps)
	progress.NextAction = s.determineNextAction(steps, docStatus)
	progress.Status = s.determineOverallStatus(steps, driver)
	progress.CanDrive = progress.Status == StatusApproved
	progress.Message = s.getStatusMessage(progress.Status)

	return progress, nil
}

// buildOnboardingSteps creates the list of onboarding steps
func (s *Service) buildOnboardingSteps() []*OnboardingStep {
	return []*OnboardingStep{
		{
			ID:          "profile",
			Name:        "Complete Profile",
			Description: "Add your personal information and contact details",
			Status:      "pending",
			Required:    true,
			Order:       1,
		},
		{
			ID:          "vehicle",
			Name:        "Add Vehicle",
			Description: "Register your vehicle details",
			Status:      "pending",
			Required:    true,
			Order:       2,
		},
		{
			ID:          "documents",
			Name:        "Upload Documents",
			Description: "Upload required documents (license, registration, insurance)",
			Status:      "pending",
			Required:    true,
			Order:       3,
		},
		{
			ID:          "document_review",
			Name:        "Document Review",
			Description: "Our team is reviewing your documents",
			Status:      "pending",
			Required:    true,
			Order:       4,
		},
		{
			ID:          "background_check",
			Name:        "Background Check",
			Description: "Verification of your driving history",
			Status:      "pending",
			Required:    false,
			Order:       5,
		},
		{
			ID:          "approval",
			Name:        "Final Approval",
			Description: "Account approval and activation",
			Status:      "pending",
			Required:    true,
			Order:       6,
		},
	}
}

// checkProfileComplete checks if driver profile is complete
func (s *Service) checkProfileComplete(driver *DriverInfo) bool {
	if driver == nil {
		return false
	}
	return driver.FirstName != "" &&
		driver.LastName != "" &&
		driver.PhoneNumber != "" &&
		driver.LicenseNumber != nil && *driver.LicenseNumber != ""
}

// updateDocumentSteps updates document-related steps
func (s *Service) updateDocumentSteps(steps []*OnboardingStep, status *VerificationStatus) {
	for _, step := range steps {
		switch step.ID {
		case "documents":
			if status.MissingDocuments == 0 {
				step.Status = "completed"
			} else if status.PendingDocuments > 0 || status.ApprovedDocuments > 0 {
				step.Status = "in_progress"
				step.Metadata = map[string]interface{}{
					"missing":  status.MissingDocuments,
					"pending":  status.PendingDocuments,
					"approved": status.ApprovedDocuments,
				}
			}
		case "document_review":
			if status.MissingDocuments == 0 && status.PendingDocuments == 0 && status.RejectedDocuments == 0 {
				step.Status = "completed"
			} else if status.PendingDocuments > 0 {
				step.Status = "in_progress"
			} else if status.RejectedDocuments > 0 {
				step.Status = "failed"
			}
		}
	}
}

// updateBackgroundCheckStep updates the background check step
func (s *Service) updateBackgroundCheckStep(steps []*OnboardingStep, bgCheck *BackgroundCheck) {
	for _, step := range steps {
		if step.ID == "background_check" {
			switch bgCheck.Status {
			case "passed":
				step.Status = "completed"
				step.CompletedAt = bgCheck.CompletedAt
			case "pending", "in_progress":
				step.Status = "in_progress"
			case "failed":
				step.Status = "failed"
			}
		}
	}
}

// calculateProgress calculates overall progress
func (s *Service) calculateProgress(steps []*OnboardingStep) *OnboardingProgress {
	progress := &OnboardingProgress{
		Steps:      steps,
		TotalSteps: len(steps),
	}

	for _, step := range steps {
		if step.Status == "completed" {
			progress.CompletedSteps++
		}
	}

	// Count only required steps for percentage
	requiredTotal := 0
	requiredCompleted := 0
	for _, step := range steps {
		if step.Required {
			requiredTotal++
			if step.Status == "completed" {
				requiredCompleted++
			}
		}
	}

	if requiredTotal > 0 {
		progress.ProgressPercent = (requiredCompleted * 100) / requiredTotal
	}

	return progress
}

// determineCurrentStep determines which step the driver is currently on
func (s *Service) determineCurrentStep(steps []*OnboardingStep) string {
	for _, step := range steps {
		if step.Status == "pending" || step.Status == "in_progress" || step.Status == "failed" {
			return step.ID
		}
	}
	return "completed"
}

// determineNextAction determines the next action for the driver
func (s *Service) determineNextAction(steps []*OnboardingStep, docStatus *VerificationStatus) *NextAction {
	for _, step := range steps {
		switch {
		case step.ID == "profile" && step.Status != "completed":
			return &NextAction{
				Type:        "complete_profile",
				Title:       "Complete Your Profile",
				Description: "Please fill in your personal details to continue",
				ActionURL:   "/driver/profile",
				Priority:    "high",
			}
		case step.ID == "vehicle" && step.Status != "completed":
			return &NextAction{
				Type:        "add_vehicle",
				Title:       "Add Your Vehicle",
				Description: "Register your vehicle details",
				ActionURL:   "/driver/vehicle",
				Priority:    "high",
			}
		case step.ID == "documents" && step.Status != "completed":
			msg := "Upload your required documents"
			if docStatus != nil && docStatus.MissingDocuments > 0 {
				msg = fmt.Sprintf("Upload %d missing document(s)", docStatus.MissingDocuments)
			}
			return &NextAction{
				Type:        "upload_documents",
				Title:       "Upload Documents",
				Description: msg,
				ActionURL:   "/driver/documents",
				Priority:    "high",
			}
		case step.ID == "document_review" && step.Status == "in_progress":
			return &NextAction{
				Type:        "wait",
				Title:       "Documents Under Review",
				Description: "We're reviewing your documents. This usually takes 1-2 business days.",
				Priority:    "low",
			}
		case step.ID == "document_review" && step.Status == "failed":
			return &NextAction{
				Type:        "resubmit_documents",
				Title:       "Resubmit Documents",
				Description: "Some documents were rejected. Please review and resubmit.",
				ActionURL:   "/driver/documents",
				Priority:    "high",
			}
		case step.ID == "approval" && step.Status == "in_progress":
			return &NextAction{
				Type:        "wait",
				Title:       "Pending Approval",
				Description: "Your account is being reviewed for final approval.",
				Priority:    "low",
			}
		}
	}

	return nil
}

// determineOverallStatus determines the overall onboarding status
func (s *Service) determineOverallStatus(steps []*OnboardingStep, driver *DriverInfo) OnboardingStatus {
	if driver.IsApproved {
		return StatusApproved
	}

	if driver.IsSuspended {
		return StatusSuspended
	}

	// Check steps
	for _, step := range steps {
		if !step.Required {
			continue
		}
		switch step.Status {
		case "failed":
			if step.ID == "document_review" {
				return StatusDocumentsReview
			}
			return StatusRejected
		case "pending", "in_progress":
			switch step.ID {
			case "profile":
				return StatusProfileIncomplete
			case "vehicle":
				return StatusProfileIncomplete
			case "documents":
				return StatusDocumentsPending
			case "document_review":
				return StatusDocumentsReview
			case "background_check":
				return StatusBackgroundCheck
			case "approval":
				return StatusPendingApproval
			}
		}
	}

	return StatusApproved
}

// getStatusMessage returns a user-friendly status message
func (s *Service) getStatusMessage(status OnboardingStatus) string {
	messages := map[OnboardingStatus]string{
		StatusNotStarted:        "Start your driver application",
		StatusProfileIncomplete: "Please complete your profile to continue",
		StatusDocumentsPending:  "Upload your required documents",
		StatusDocumentsReview:   "Your documents are being reviewed",
		StatusBackgroundCheck:   "Background check in progress",
		StatusPendingApproval:   "Your application is pending final approval",
		StatusApproved:          "You're approved! Start accepting rides",
		StatusRejected:          "Your application was not approved",
		StatusSuspended:         "Your account has been suspended",
	}

	if msg, ok := messages[status]; ok {
		return msg
	}
	return "Unknown status"
}

// GetDocumentRequirements returns the document requirements for a driver
func (s *Service) GetDocumentRequirements(ctx context.Context, driverID uuid.UUID) ([]*DocumentRequirement, error) {
	// Get required document types
	docTypes, err := s.documentService.GetRequiredDocumentTypes(ctx)
	if err != nil {
		return nil, err
	}

	// Get driver's documents
	docs, err := s.documentService.GetDriverDocuments(ctx, driverID)
	if err != nil {
		docs = []DocumentInfo{} // Continue with empty list
	}

	// Map documents by type
	docByType := make(map[uuid.UUID]DocumentInfo)
	for _, doc := range docs {
		if existing, ok := docByType[doc.DocumentTypeID]; !ok || doc.SubmittedAt.After(existing.SubmittedAt) {
			docByType[doc.DocumentTypeID] = doc
		}
	}

	// Build requirements
	var requirements []*DocumentRequirement
	for _, dt := range docTypes {
		req := &DocumentRequirement{
			DocumentTypeID:   dt.ID,
			DocumentTypeCode: dt.Code,
			Name:             dt.Name,
			Required:         dt.IsRequired,
			Status:           "not_submitted",
		}

		if doc, ok := docByType[dt.ID]; ok {
			req.DocumentID = &doc.ID
			req.Status = doc.Status
			req.SubmittedAt = &doc.SubmittedAt
			req.ReviewedAt = doc.ReviewedAt
			req.ExpiresAt = doc.ExpiryDate
			req.RejectionReason = doc.RejectionReason
		}

		requirements = append(requirements, req)
	}

	return requirements, nil
}

// StartOnboarding initializes the onboarding process for a new driver
func (s *Service) StartOnboarding(ctx context.Context, userID uuid.UUID) (*OnboardingProgress, error) {
	// Check if user is already a driver
	existing, _ := s.repo.GetDriverByUserID(ctx, userID)
	if existing != nil {
		return s.GetOnboardingProgress(ctx, existing.ID)
	}

	// Create new driver record
	driverID, err := s.repo.CreateDriver(ctx, userID)
	if err != nil {
		return nil, common.NewInternalServerError("failed to create driver record")
	}

	// Send welcome notification
	if s.notifService != nil {
		go func() {
			_ = s.notifService.SendOnboardingNotification(
				context.Background(),
				userID,
				"onboarding_started",
				"Welcome to Driver Onboarding!",
				"Complete your profile and upload documents to start driving.",
			)
		}()
	}

	logger.Info("Driver onboarding started",
		zap.String("user_id", userID.String()),
		zap.String("driver_id", driverID.String()),
	)

	return s.GetOnboardingProgress(ctx, driverID)
}

// NotifyOnboardingUpdate sends a notification about onboarding status change
func (s *Service) NotifyOnboardingUpdate(ctx context.Context, driverID uuid.UUID, eventType string) error {
	driver, err := s.repo.GetDriver(ctx, driverID)
	if err != nil {
		return err
	}

	var title, message string
	switch eventType {
	case "documents_approved":
		title = "Documents Approved!"
		message = "Your documents have been verified. We're completing your background check."
	case "documents_rejected":
		title = "Document Issue"
		message = "One or more documents need to be resubmitted. Please check your document status."
	case "approved":
		title = "Welcome Aboard! 🎉"
		message = "Your driver account is now approved. You can start accepting rides!"
	case "rejected":
		title = "Application Update"
		message = "Unfortunately, we couldn't approve your application at this time."
	}

	if s.notifService != nil && title != "" {
		return s.notifService.SendOnboardingNotification(ctx, driver.UserID, eventType, title, message)
	}

	return nil
}
