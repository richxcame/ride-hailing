package verification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// Service handles verification business logic
type Service struct {
	repo       *Repository
	cfg        *config.Config
	httpClient *http.Client
}

// NewService creates a new verification service
func NewService(repo *Repository, cfg *config.Config) *Service {
	return &Service{
		repo: repo,
		cfg:  cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ========================================
// BACKGROUND CHECK OPERATIONS
// ========================================

// InitiateBackgroundCheck starts a new background check for a driver
func (s *Service) InitiateBackgroundCheck(ctx context.Context, req *InitiateBackgroundCheckRequest) (*BackgroundCheckResponse, error) {
	// Check if there's already a pending check
	existing, _ := s.repo.GetLatestBackgroundCheck(ctx, req.DriverID)
	if existing != nil && (existing.Status == BGCheckStatusPending || existing.Status == BGCheckStatusInProgress) {
		return nil, common.NewBadRequestError("a background check is already in progress", nil)
	}

	// Set default provider
	provider := req.Provider
	if provider == "" {
		provider = ProviderCheckr
	}

	// Set default check type
	checkType := req.CheckType
	if checkType == "" {
		checkType = "driver_standard"
	}

	// Create background check record
	check := &BackgroundCheck{
		ID:        uuid.New(),
		DriverID:  req.DriverID,
		Provider:  provider,
		Status:    BGCheckStatusPending,
		CheckType: checkType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateBackgroundCheck(ctx, check); err != nil {
		return nil, common.NewInternalServerError("failed to create background check")
	}

	// Initiate with provider
	var externalID string
	var err error

	switch provider {
	case ProviderCheckr:
		externalID, err = s.initiateCheckrCheck(ctx, req, check.ID)
	case ProviderSterling:
		externalID, err = s.initiateSterlingCheck(ctx, req, check.ID)
	case ProviderOnfido:
		externalID, err = s.initiateOnfidoCheck(ctx, req, check.ID)
	case ProviderMock:
		externalID, err = s.initiateMockCheck(ctx, req, check.ID)
	default:
		return nil, common.NewBadRequestError("unsupported background check provider", nil)
	}

	if err != nil {
		// Update check status to failed
		failureReason := err.Error()
		_ = s.repo.UpdateBackgroundCheckStatus(ctx, check.ID, BGCheckStatusFailed, &failureReason, nil)
		return nil, common.NewInternalServerError("failed to initiate background check with provider")
	}

	// Update check with external ID and in-progress status
	if err := s.repo.UpdateBackgroundCheckStarted(ctx, check.ID, externalID); err != nil {
		logger.Error("failed to update background check status", zap.Error(err))
	}

	logger.Info("Background check initiated",
		zap.String("driver_id", req.DriverID.String()),
		zap.String("provider", string(provider)),
		zap.String("external_id", externalID),
	)

	return &BackgroundCheckResponse{
		CheckID:       check.ID,
		Status:        BGCheckStatusInProgress,
		Provider:      provider,
		ExternalID:    &externalID,
		EstimatedTime: "1-3 business days",
	}, nil
}

// GetBackgroundCheckStatus gets the status of a background check
func (s *Service) GetBackgroundCheckStatus(ctx context.Context, checkID uuid.UUID) (*BackgroundCheck, error) {
	check, err := s.repo.GetBackgroundCheck(ctx, checkID)
	if err != nil {
		return nil, common.NewNotFoundError("background check not found", err)
	}

	return check, nil
}

// GetDriverBackgroundStatus gets the latest background check status for a driver
func (s *Service) GetDriverBackgroundStatus(ctx context.Context, driverID uuid.UUID) (*BackgroundCheck, error) {
	check, err := s.repo.GetLatestBackgroundCheck(ctx, driverID)
	if err != nil {
		return nil, common.NewNotFoundError("no background check found for driver", err)
	}

	return check, nil
}

// ProcessWebhook processes webhooks from background check providers
func (s *Service) ProcessWebhook(ctx context.Context, payload *WebhookPayload) error {
	// Find the background check by external ID
	check, err := s.repo.GetBackgroundCheckByExternalID(ctx, payload.Provider, payload.ExternalID)
	if err != nil {
		return common.NewNotFoundError("background check not found", err)
	}

	// Map provider status to our status
	var newStatus BackgroundCheckStatus
	var failureReasons []string

	switch payload.Provider {
	case ProviderCheckr:
		newStatus, failureReasons = s.mapCheckrStatus(payload)
	case ProviderSterling:
		newStatus, failureReasons = s.mapSterlingStatus(payload)
	case ProviderOnfido:
		newStatus, failureReasons = s.mapOnfidoStatus(payload)
	default:
		return common.NewBadRequestError("unsupported provider", nil)
	}

	// Extract report URL if available
	var reportURL *string
	if url, ok := payload.Data["report_url"].(string); ok {
		reportURL = &url
	}

	// Calculate expiration (typically 1 year for passed checks)
	var expiresAt *time.Time
	if newStatus == BGCheckStatusPassed {
		exp := time.Now().AddDate(1, 0, 0) // 1 year from now
		expiresAt = &exp
	}

	// Update the background check
	if err := s.repo.UpdateBackgroundCheckCompleted(ctx, check.ID, newStatus, reportURL, expiresAt); err != nil {
		return common.NewInternalServerError("failed to update background check")
	}

	// Update failure reasons if any
	if len(failureReasons) > 0 {
		_ = s.repo.UpdateBackgroundCheckStatus(ctx, check.ID, newStatus, nil, failureReasons)
	}

	// Auto-approve driver if background check passed
	if newStatus == BGCheckStatusPassed {
		if err := s.repo.UpdateDriverApproval(ctx, check.DriverID, true, nil, nil); err != nil {
			logger.Error("failed to auto-approve driver", zap.Error(err))
		}
	}

	logger.Info("Background check webhook processed",
		zap.String("check_id", check.ID.String()),
		zap.String("new_status", string(newStatus)),
	)

	return nil
}

// ========================================
// SELFIE VERIFICATION OPERATIONS
// ========================================

// VerifySelfie verifies a driver's selfie against their reference photo
func (s *Service) VerifySelfie(ctx context.Context, req *SubmitSelfieRequest) (*SelfieVerificationResponse, error) {
	// Get driver's reference photo
	referenceURL, err := s.repo.GetDriverReferencePhoto(ctx, req.DriverID)
	if err != nil || referenceURL == nil {
		return nil, common.NewBadRequestError("driver reference photo not found, please upload identity documents first", nil)
	}

	// Create verification record
	verification := &SelfieVerification{
		ID:                uuid.New(),
		DriverID:          req.DriverID,
		RideID:            req.RideID,
		SelfieURL:         req.SelfieURL,
		ReferencePhotoURL: referenceURL,
		Status:            SelfieStatusPending,
		Provider:          "rekognition", // AWS Rekognition for face comparison
		CreatedAt:         time.Now(),
	}

	if err := s.repo.CreateSelfieVerification(ctx, verification); err != nil {
		return nil, common.NewInternalServerError("failed to create verification record")
	}

	// Perform face comparison
	confidenceScore, match, err := s.compareFaces(ctx, req.SelfieURL, *referenceURL)
	if err != nil {
		failureReason := err.Error()
		_ = s.repo.UpdateSelfieVerificationResult(ctx, verification.ID, SelfieStatusFailed, nil, nil, &failureReason)
		return nil, common.NewInternalServerError("face comparison failed")
	}

	// Determine verification status based on confidence score
	var status SelfieVerificationStatus
	var message string
	minConfidence := 90.0 // 90% minimum confidence for a match

	if match && confidenceScore >= minConfidence {
		status = SelfieStatusVerified
		message = "Identity verified successfully"
	} else {
		status = SelfieStatusFailed
		if !match {
			message = "Face does not match reference photo"
		} else {
			message = fmt.Sprintf("Confidence score too low: %.1f%% (minimum: %.1f%%)", confidenceScore, minConfidence)
		}
	}

	// Update verification result
	if err := s.repo.UpdateSelfieVerificationResult(ctx, verification.ID, status, &confidenceScore, &match, nil); err != nil {
		logger.Error("failed to update selfie verification", zap.Error(err))
	}

	logger.Info("Selfie verification completed",
		zap.String("driver_id", req.DriverID.String()),
		zap.String("status", string(status)),
		zap.Float64("confidence", confidenceScore),
	)

	return &SelfieVerificationResponse{
		VerificationID:  verification.ID,
		Status:          status,
		ConfidenceScore: &confidenceScore,
		MatchResult:     &match,
		Message:         message,
	}, nil
}

// GetSelfieVerificationStatus gets the status of a selfie verification
func (s *Service) GetSelfieVerificationStatus(ctx context.Context, verificationID uuid.UUID) (*SelfieVerification, error) {
	verification, err := s.repo.GetSelfieVerification(ctx, verificationID)
	if err != nil {
		return nil, common.NewNotFoundError("selfie verification not found", err)
	}

	return verification, nil
}

// RequiresSelfieVerification checks if a driver needs to verify their selfie today
func (s *Service) RequiresSelfieVerification(ctx context.Context, driverID uuid.UUID) (bool, error) {
	_, err := s.repo.GetTodaysSelfieVerification(ctx, driverID)
	if err != nil {
		// No verification today - requires verification
		return true, nil
	}

	return false, nil
}

// GetDriverVerificationStatus gets the overall verification status
func (s *Service) GetDriverVerificationStatus(ctx context.Context, driverID uuid.UUID) (*DriverVerificationStatus, error) {
	return s.repo.GetDriverVerificationStatus(ctx, driverID)
}

// ========================================
// PROVIDER INTEGRATIONS
// ========================================

// Checkr integration
func (s *Service) initiateCheckrCheck(ctx context.Context, req *InitiateBackgroundCheckRequest, checkID uuid.UUID) (string, error) {
	apiKey := s.cfg.Checkr.APIKey
	if apiKey == "" {
		return "", fmt.Errorf("checkr API key not configured")
	}

	// Create candidate
	candidatePayload := map[string]interface{}{
		"first_name":       req.FirstName,
		"last_name":        req.LastName,
		"email":            req.Email,
		"phone":            req.Phone,
		"dob":              req.DateOfBirth,
		"ssn":              req.SSN,
		"driver_license_number": req.LicenseNumber,
		"driver_license_state":  req.LicenseState,
		"zipcode":          req.ZipCode,
		"work_locations": []map[string]string{
			{
				"city":    req.City,
				"state":   req.State,
				"country": "US",
			},
		},
	}

	candidateBody, _ := json.Marshal(candidatePayload)
	candidateReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.checkr.com/v1/candidates", bytes.NewBuffer(candidateBody))
	candidateReq.SetBasicAuth(apiKey, "")
	candidateReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(candidateReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("checkr candidate creation failed: %s", string(body))
	}

	var candidateResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&candidateResp); err != nil {
		return "", err
	}

	// Create invitation (triggers background check)
	invitationPayload := map[string]interface{}{
		"candidate_id": candidateResp.ID,
		"package":      "driver_standard",
	}

	invitationBody, _ := json.Marshal(invitationPayload)
	invitationReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.checkr.com/v1/invitations", bytes.NewBuffer(invitationBody))
	invitationReq.SetBasicAuth(apiKey, "")
	invitationReq.Header.Set("Content-Type", "application/json")

	resp, err = s.httpClient.Do(invitationReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("checkr invitation creation failed: %s", string(body))
	}

	var invitationResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&invitationResp); err != nil {
		return "", err
	}

	return invitationResp.ID, nil
}

func (s *Service) mapCheckrStatus(payload *WebhookPayload) (BackgroundCheckStatus, []string) {
	switch payload.Status {
	case "clear":
		return BGCheckStatusPassed, nil
	case "consider":
		return BGCheckStatusFailed, []string{"Background check flagged for review"}
	case "pending":
		return BGCheckStatusInProgress, nil
	default:
		return BGCheckStatusFailed, []string{fmt.Sprintf("Unknown status: %s", payload.Status)}
	}
}

// Sterling integration
func (s *Service) initiateSterlingCheck(ctx context.Context, req *InitiateBackgroundCheckRequest, checkID uuid.UUID) (string, error) {
	// Sterling API implementation
	// This would follow Sterling's API documentation
	return fmt.Sprintf("sterling-%s", checkID.String()), nil
}

func (s *Service) mapSterlingStatus(payload *WebhookPayload) (BackgroundCheckStatus, []string) {
	switch payload.Status {
	case "Completed", "Pass":
		return BGCheckStatusPassed, nil
	case "Fail", "Alert":
		return BGCheckStatusFailed, []string{"Background check failed"}
	default:
		return BGCheckStatusInProgress, nil
	}
}

// Onfido integration
func (s *Service) initiateOnfidoCheck(ctx context.Context, req *InitiateBackgroundCheckRequest, checkID uuid.UUID) (string, error) {
	apiKey := s.cfg.Onfido.APIKey
	if apiKey == "" {
		return "", fmt.Errorf("onfido API key not configured")
	}

	// Create applicant
	applicantPayload := map[string]interface{}{
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"email":      req.Email,
		"dob":        req.DateOfBirth,
		"address": map[string]string{
			"street":      req.StreetAddress,
			"city":        req.City,
			"state":       req.State,
			"postcode":    req.ZipCode,
			"country":     "USA",
		},
	}

	applicantBody, _ := json.Marshal(applicantPayload)
	applicantReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.onfido.com/v3/applicants", bytes.NewBuffer(applicantBody))
	applicantReq.Header.Set("Authorization", "Token token="+apiKey)
	applicantReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(applicantReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("onfido applicant creation failed: %s", string(body))
	}

	var applicantResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&applicantResp); err != nil {
		return "", err
	}

	return applicantResp.ID, nil
}

func (s *Service) mapOnfidoStatus(payload *WebhookPayload) (BackgroundCheckStatus, []string) {
	switch payload.Status {
	case "complete":
		if result, ok := payload.Data["result"].(string); ok && result == "clear" {
			return BGCheckStatusPassed, nil
		}
		return BGCheckStatusFailed, []string{"Document verification failed"}
	case "in_progress", "awaiting_data":
		return BGCheckStatusInProgress, nil
	default:
		return BGCheckStatusFailed, []string{fmt.Sprintf("Unknown status: %s", payload.Status)}
	}
}

// Mock provider for testing
func (s *Service) initiateMockCheck(ctx context.Context, req *InitiateBackgroundCheckRequest, checkID uuid.UUID) (string, error) {
	// Simulate async check completion
	go func() {
		time.Sleep(5 * time.Second)
		expiresAt := time.Now().AddDate(1, 0, 0)
		_ = s.repo.UpdateBackgroundCheckCompleted(context.Background(), checkID, BGCheckStatusPassed, nil, &expiresAt)
		_ = s.repo.UpdateDriverApproval(context.Background(), req.DriverID, true, nil, nil)
	}()

	return fmt.Sprintf("mock-%s", checkID.String()), nil
}

// Face comparison using AWS Rekognition
func (s *Service) compareFaces(ctx context.Context, selfieURL, referenceURL string) (float64, bool, error) {
	// For production, this would use AWS Rekognition's CompareFaces API
	// For now, return a mock result

	// Mock implementation - in production use AWS SDK:
	// client := rekognition.NewFromConfig(awsCfg)
	// output, err := client.CompareFaces(ctx, &rekognition.CompareFacesInput{
	//     SourceImage: &types.Image{S3Object: &types.S3Object{...}},
	//     TargetImage: &types.Image{S3Object: &types.S3Object{...}},
	//     SimilarityThreshold: aws.Float32(90),
	// })

	// Mock: 95% confidence, successful match
	return 95.5, true, nil
}
