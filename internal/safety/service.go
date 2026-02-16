package safety

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/eventbus"
	"github.com/richxcame/ride-hailing/pkg/httpclient"
	"github.com/richxcame/ride-hailing/pkg/logger"
	redisclient "github.com/richxcame/ride-hailing/pkg/redis"
	"github.com/richxcame/ride-hailing/pkg/websocket"
	"go.uber.org/zap"
)

// Service handles safety business logic
type Service struct {
	repo               RepositoryInterface
	eventBus           *eventbus.Bus
	wsHub              *websocket.Hub
	redis              redisclient.ClientInterface
	notificationClient *httpclient.Client
	mapsClient         *httpclient.Client
	ridesClient        *httpclient.Client
	baseShareURL       string
	emergencyNumber    string // Local emergency number (e.g., "911", "112")
}

// Config holds service configuration
type Config struct {
	BaseShareURL    string
	EmergencyNumber string
	NotificationURL string
	MapsServiceURL  string
	RidesServiceURL string
}

// NewService creates a new safety service
func NewService(repo RepositoryInterface, cfg Config) *Service {
	if repo == nil {
		panic("safety: repository cannot be nil")
	}
	s := &Service{
		repo:               repo,
		baseShareURL:       cfg.BaseShareURL,
		emergencyNumber:    cfg.EmergencyNumber,
		notificationClient: httpclient.NewClient(cfg.NotificationURL),
		mapsClient:         httpclient.NewClient(cfg.MapsServiceURL),
	}
	if cfg.RidesServiceURL != "" {
		s.ridesClient = httpclient.NewClient(cfg.RidesServiceURL)
	}
	return s
}

// SetEventBus sets the NATS event bus
func (s *Service) SetEventBus(bus *eventbus.Bus) {
	s.eventBus = bus
}

// SetWebSocketHub sets the WebSocket hub for real-time notifications
func (s *Service) SetWebSocketHub(hub *websocket.Hub) {
	s.wsHub = hub
}

// SetRedis sets the Redis client for caching
func (s *Service) SetRedis(redis redisclient.ClientInterface) {
	s.redis = redis
}

// ========================================
// EMERGENCY SOS
// ========================================

// TriggerSOS triggers an emergency SOS alert
func (s *Service) TriggerSOS(ctx context.Context, userID uuid.UUID, req *TriggerSOSRequest) (*TriggerSOSResponse, error) {
	logger.Info("SOS triggered", zap.String("user_id", userID.String()), zap.String("type", string(req.Type)))

	// Create emergency alert
	alert := &EmergencyAlert{
		ID:             uuid.New(),
		UserID:         userID,
		RideID:         req.RideID,
		Type:           req.Type,
		Status:         EmergencyStatusActive,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		Description:    req.Description,
		PoliceNotified: false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Get address from coordinates (async, don't block)
	go s.enrichAlertWithAddress(context.Background(), alert.ID, req.Latitude, req.Longitude)

	if err := s.repo.CreateEmergencyAlert(ctx, alert); err != nil {
		logger.Error("Failed to create emergency alert", zap.Error(err))
		return nil, fmt.Errorf("failed to create emergency alert: %w", err)
	}

	// Get contacts to notify
	contacts, err := s.repo.GetSOSContacts(ctx, userID)
	if err != nil {
		logger.Warn("Failed to get SOS contacts", zap.Error(err))
	}

	// Notify emergency contacts
	contactsNotified := 0
	for _, contact := range contacts {
		if err := s.notifyEmergencyContact(ctx, contact, alert); err != nil {
			logger.Warn("Failed to notify contact", zap.String("contact_id", contact.ID.String()), zap.Error(err))
		} else {
			contactsNotified++
		}
	}

	// If ride is active, also notify the driver or rider (depending on who triggered)
	if req.RideID != nil {
		go s.notifyRideParticipants(context.Background(), *req.RideID, userID, alert)
	}

	// Publish event
	s.publishSOSEvent(alert)

	// Create share link for emergency contacts
	var shareLink string
	if req.RideID != nil {
		link, err := s.CreateShareLink(ctx, userID, &CreateShareLinkRequest{
			RideID:        *req.RideID,
			ExpiryMinutes: 60, // 1 hour for emergency
			ShareLocation: true,
			ShareDriver:   true,
			ShareETA:      true,
			ShareRoute:    true,
		})
		if err == nil {
			shareLink = link.ShareURL
		}
	}

	// Notify admin dashboard via WebSocket
	s.notifyAdminDashboard(alert)

	return &TriggerSOSResponse{
		AlertID:          alert.ID,
		Status:           string(alert.Status),
		Message:          "Emergency alert activated. Your contacts have been notified.",
		ContactsNotified: contactsNotified,
		EmergencyNumber:  s.emergencyNumber,
		ShareLink:        shareLink,
	}, nil
}

// CancelSOS cancels an emergency alert
func (s *Service) CancelSOS(ctx context.Context, userID uuid.UUID, alertID uuid.UUID) error {
	alert, err := s.repo.GetEmergencyAlert(ctx, alertID)
	if err != nil {
		return err
	}
	if alert == nil {
		return fmt.Errorf("alert not found")
	}
	if alert.UserID != userID {
		return fmt.Errorf("unauthorized")
	}
	if alert.Status != EmergencyStatusActive && alert.Status != EmergencyStatusResponded {
		return fmt.Errorf("alert is not active")
	}

	return s.repo.UpdateEmergencyAlertStatus(ctx, alertID, EmergencyStatusCancelled, nil, "Cancelled by user")
}

// ResolveEmergencyAlert resolves an emergency alert (admin action)
func (s *Service) ResolveEmergencyAlert(ctx context.Context, adminID, alertID uuid.UUID, resolution string, isFalseAlarm bool) error {
	status := EmergencyStatusResolved
	if isFalseAlarm {
		status = EmergencyStatusFalse
	}
	return s.repo.UpdateEmergencyAlertStatus(ctx, alertID, status, &adminID, resolution)
}

// GetActiveEmergencies retrieves all active emergencies (for admin)
func (s *Service) GetActiveEmergencies(ctx context.Context) ([]*EmergencyAlert, error) {
	return s.repo.GetActiveEmergencyAlerts(ctx)
}

// GetUserEmergencies retrieves emergencies for a user
func (s *Service) GetUserEmergencies(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*EmergencyAlert, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.GetUserEmergencyAlerts(ctx, userID, limit, offset)
}

// ========================================
// EMERGENCY CONTACTS
// ========================================

// CreateEmergencyContact creates a new emergency contact
func (s *Service) CreateEmergencyContact(ctx context.Context, userID uuid.UUID, req *CreateEmergencyContactRequest) (*EmergencyContact, error) {
	// Check contact limit (max 5)
	contacts, err := s.repo.GetUserEmergencyContacts(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(contacts) >= 5 {
		return nil, fmt.Errorf("maximum 5 emergency contacts allowed")
	}

	contact := &EmergencyContact{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         req.Name,
		Phone:        req.Phone,
		Email:        req.Email,
		Relationship: req.Relationship,
		IsPrimary:    req.IsPrimary,
		NotifyOnRide: req.NotifyOnRide,
		NotifyOnSOS:  req.NotifyOnSOS,
		Verified:     false, // Will need phone verification
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateEmergencyContact(ctx, contact); err != nil {
		return nil, err
	}

	// Send verification SMS
	go s.sendContactVerification(context.Background(), contact)

	return contact, nil
}

// GetEmergencyContacts retrieves all emergency contacts for a user
func (s *Service) GetEmergencyContacts(ctx context.Context, userID uuid.UUID) ([]*EmergencyContact, error) {
	return s.repo.GetUserEmergencyContacts(ctx, userID)
}

// UpdateEmergencyContact updates an emergency contact
func (s *Service) UpdateEmergencyContact(ctx context.Context, userID, contactID uuid.UUID, req *CreateEmergencyContactRequest) (*EmergencyContact, error) {
	contact, err := s.repo.GetEmergencyContact(ctx, contactID)
	if err != nil {
		return nil, err
	}
	if contact == nil {
		return nil, fmt.Errorf("contact not found")
	}
	if contact.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	// If phone changed, need re-verification
	phoneChanged := contact.Phone != req.Phone

	contact.Name = req.Name
	contact.Phone = req.Phone
	contact.Email = req.Email
	contact.Relationship = req.Relationship
	contact.IsPrimary = req.IsPrimary
	contact.NotifyOnRide = req.NotifyOnRide
	contact.NotifyOnSOS = req.NotifyOnSOS

	if phoneChanged {
		contact.Verified = false
		contact.VerifiedAt = nil
	}

	if err := s.repo.UpdateEmergencyContact(ctx, contact); err != nil {
		return nil, err
	}

	if phoneChanged {
		go s.sendContactVerification(context.Background(), contact)
	}

	return contact, nil
}

// DeleteEmergencyContact deletes an emergency contact
func (s *Service) DeleteEmergencyContact(ctx context.Context, userID, contactID uuid.UUID) error {
	contact, err := s.repo.GetEmergencyContact(ctx, contactID)
	if err != nil {
		return err
	}
	if contact == nil {
		return fmt.Errorf("contact not found")
	}
	if contact.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	return s.repo.DeleteEmergencyContact(ctx, contactID)
}

// VerifyEmergencyContact verifies an emergency contact's phone
func (s *Service) VerifyEmergencyContact(ctx context.Context, contactID uuid.UUID, code string) error {
	// Verify code from Redis
	if s.redis != nil {
		key := fmt.Sprintf("contact_verify:%s", contactID.String())
		storedCode, err := s.redis.GetString(ctx, key)
		if err != nil || storedCode != code {
			return fmt.Errorf("invalid or expired verification code")
		}
		s.redis.Delete(ctx, key)
	}

	return s.repo.VerifyEmergencyContact(ctx, contactID)
}

// ========================================
// RIDE SHARING
// ========================================

// CreateShareLink creates a shareable link for live ride tracking
func (s *Service) CreateShareLink(ctx context.Context, userID uuid.UUID, req *CreateShareLinkRequest) (*ShareLinkResponse, error) {
	// Generate secure token
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Calculate expiry
	expiry := time.Now().Add(24 * time.Hour) // Default 24 hours
	if req.ExpiryMinutes > 0 {
		expiry = time.Now().Add(time.Duration(req.ExpiryMinutes) * time.Minute)
	}

	link := &RideShareLink{
		ID:            uuid.New(),
		RideID:        req.RideID,
		UserID:        userID,
		Token:         token,
		ExpiresAt:     expiry,
		IsActive:      true,
		ShareLocation: req.ShareLocation,
		ShareDriver:   req.ShareDriver,
		ShareETA:      req.ShareETA,
		ShareRoute:    req.ShareRoute,
		ViewCount:     0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.CreateRideShareLink(ctx, link); err != nil {
		return nil, err
	}

	// Notify recipients
	for _, recipient := range req.SendTo {
		go s.notifyShareRecipient(context.Background(), link, recipient)
	}

	return &ShareLinkResponse{
		LinkID:    link.ID,
		ShareURL:  fmt.Sprintf("%s/share/%s", s.baseShareURL, token),
		ExpiresAt: link.ExpiresAt,
		Token:     token,
	}, nil
}

// rideDataResponse represents the response from the rides service
type rideDataResponse struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	CurrentLatitude   float64 `json:"current_latitude"`
	CurrentLongitude  float64 `json:"current_longitude"`
	DriverFirstName   string  `json:"driver_first_name"`
	DriverPhoto     string  `json:"driver_photo"`
	VehicleMake     string  `json:"vehicle_make"`
	VehicleModel    string  `json:"vehicle_model"`
	VehicleColor    string  `json:"vehicle_color"`
	LicensePlate    string  `json:"license_plate"`
	PickupAddress   string  `json:"pickup_address"`
	DropoffAddress  string  `json:"dropoff_address"`
	RoutePolyline   string  `json:"route_polyline"`
	ETAMinutes      int     `json:"eta_minutes"`
	EstimatedArrival *time.Time `json:"estimated_arrival"`
}

// GetSharedRide retrieves the shared ride view for a token
func (s *Service) GetSharedRide(ctx context.Context, token string) (*SharedRideView, error) {
	link, err := s.repo.GetRideShareLinkByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if link == nil {
		return nil, fmt.Errorf("share link not found or expired")
	}

	// Increment view count
	go s.repo.IncrementShareLinkViewCount(context.Background(), link.ID)

	// Initialize the view with base data
	view := &SharedRideView{
		RideID:           link.RideID,
		Status:           "in_progress",
		EmergencyContact: s.emergencyNumber,
		ReportConcernURL: fmt.Sprintf("%s/safety/report", s.baseShareURL),
		LastUpdated:      time.Now(),
	}

	// Fetch ride data from rides service if client is configured
	if s.ridesClient != nil {
		rideData, err := s.fetchRideData(ctx, link.RideID)
		if err != nil {
			logger.WithContext(ctx).Warn("failed to fetch ride data for shared view",
				zap.String("ride_id", link.RideID.String()),
				zap.Error(err))
			// Continue with partial data - don't fail the request
		} else if rideData != nil {
			// Update status from actual ride data
			view.Status = rideData.Status

			// Conditionally include data based on link permissions
			if link.ShareLocation && rideData.CurrentLatitude != 0 && rideData.CurrentLongitude != 0 {
				view.CurrentLocation = &Coordinate{
					Latitude:  rideData.CurrentLatitude,
					Longitude: rideData.CurrentLongitude,
				}
			}

			if link.ShareDriver {
				view.DriverFirstName = rideData.DriverFirstName
				view.DriverPhoto = rideData.DriverPhoto
				view.VehicleMake = rideData.VehicleMake
				view.VehicleModel = rideData.VehicleModel
				view.VehicleColor = rideData.VehicleColor
				view.LicensePlate = rideData.LicensePlate
			}

			if link.ShareRoute {
				view.PickupAddress = rideData.PickupAddress
				view.DropoffAddress = rideData.DropoffAddress
				view.RoutePolyline = rideData.RoutePolyline
			}

			if link.ShareETA {
				view.ETAMinutes = rideData.ETAMinutes
				view.EstimatedArrival = rideData.EstimatedArrival
			}
		}
	} else {
		logger.WithContext(ctx).Warn("rides client not configured, returning partial shared ride view",
			zap.String("ride_id", link.RideID.String()))
	}

	return view, nil
}

// fetchRideData fetches ride data from the rides service
func (s *Service) fetchRideData(ctx context.Context, rideID uuid.UUID) (*rideDataResponse, error) {
	path := fmt.Sprintf("/api/v1/rides/%s", rideID.String())
	respBody, err := s.ridesClient.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ride data: %w", err)
	}

	var rideData rideDataResponse
	if err := json.Unmarshal(respBody, &rideData); err != nil {
		return nil, fmt.Errorf("failed to parse ride data: %w", err)
	}

	return &rideData, nil
}

// DeactivateShareLink deactivates a share link
func (s *Service) DeactivateShareLink(ctx context.Context, userID, linkID uuid.UUID) error {
	// Verify ownership would happen here
	return s.repo.DeactivateShareLink(ctx, linkID)
}

// ========================================
// SAFETY SETTINGS
// ========================================

// GetSafetySettings retrieves safety settings for a user
func (s *Service) GetSafetySettings(ctx context.Context, userID uuid.UUID) (*SafetySettings, error) {
	return s.repo.GetSafetySettings(ctx, userID)
}

// UpdateSafetySettings updates safety settings for a user
func (s *Service) UpdateSafetySettings(ctx context.Context, userID uuid.UUID, req *UpdateSafetySettingsRequest) (*SafetySettings, error) {
	settings, err := s.repo.GetSafetySettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.SOSEnabled != nil {
		settings.SOSEnabled = *req.SOSEnabled
	}
	if req.SOSGesture != nil {
		settings.SOSGesture = *req.SOSGesture
	}
	if req.SOSAutoCall911 != nil {
		settings.SOSAutoCall911 = *req.SOSAutoCall911
	}
	if req.AutoShareWithContacts != nil {
		settings.AutoShareWithContacts = *req.AutoShareWithContacts
	}
	if req.ShareOnRideStart != nil {
		settings.ShareOnRideStart = *req.ShareOnRideStart
	}
	if req.ShareOnEmergency != nil {
		settings.ShareOnEmergency = *req.ShareOnEmergency
	}
	if req.PeriodicCheckIn != nil {
		settings.PeriodicCheckIn = *req.PeriodicCheckIn
	}
	if req.CheckInIntervalMins != nil {
		settings.CheckInIntervalMins = *req.CheckInIntervalMins
	}
	if req.RouteDeviationAlert != nil {
		settings.RouteDeviationAlert = *req.RouteDeviationAlert
	}
	if req.SpeedAlertEnabled != nil {
		settings.SpeedAlertEnabled = *req.SpeedAlertEnabled
	}
	if req.LongStopAlertMins != nil {
		settings.LongStopAlertMins = *req.LongStopAlertMins
	}
	if req.AllowRideRecording != nil {
		settings.AllowRideRecording = *req.AllowRideRecording
	}
	if req.RecordAudioOnSOS != nil {
		settings.RecordAudioOnSOS = *req.RecordAudioOnSOS
	}
	if req.NightRideProtection != nil {
		settings.NightRideProtection = *req.NightRideProtection
	}
	if req.NightProtectionStart != nil {
		settings.NightProtectionStart = *req.NightProtectionStart
	}
	if req.NightProtectionEnd != nil {
		settings.NightProtectionEnd = *req.NightProtectionEnd
	}

	if err := s.repo.UpdateSafetySettings(ctx, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// ========================================
// SAFETY CHECKS
// ========================================

// SendSafetyCheck sends a safety check to a user
func (s *Service) SendSafetyCheck(ctx context.Context, rideID, userID uuid.UUID, checkType SafetyCheckType, reason string) (*SafetyCheck, error) {
	check := &SafetyCheck{
		ID:            uuid.New(),
		RideID:        rideID,
		UserID:        userID,
		Type:          checkType,
		Status:        SafetyCheckPending,
		SentAt:        time.Now(),
		TriggerReason: reason,
		CreatedAt:     time.Now(),
	}

	if err := s.repo.CreateSafetyCheck(ctx, check); err != nil {
		return nil, err
	}

	// Send push notification
	go s.sendSafetyCheckNotification(context.Background(), check)

	// Send WebSocket message
	if s.wsHub != nil {
		s.wsHub.SendToUser(userID.String(), &websocket.Message{
			Type: "safety_check",
			Data: map[string]interface{}{
				"check_id":       check.ID.String(),
				"type":           check.Type,
				"trigger_reason": check.TriggerReason,
			},
		})
	}

	return check, nil
}

// RespondToSafetyCheck responds to a safety check
func (s *Service) RespondToSafetyCheck(ctx context.Context, userID uuid.UUID, req *SafetyCheckResponse) error {
	status := SafetyCheckSafe
	if req.Response == "help" {
		status = SafetyCheckHelp
	}

	if err := s.repo.UpdateSafetyCheckResponse(ctx, req.CheckID, status, req.Notes); err != nil {
		return err
	}

	// If user needs help, trigger escalation
	if status == SafetyCheckHelp {
		go s.escalateSafetyCheck(context.Background(), req.CheckID, userID)
	}

	return nil
}

// ProcessExpiredSafetyChecks processes expired safety checks (called by worker)
func (s *Service) ProcessExpiredSafetyChecks(ctx context.Context) (int, error) {
	timeout := 5 * time.Minute
	checks, err := s.repo.GetExpiredPendingSafetyChecks(ctx, timeout)
	if err != nil {
		return 0, err
	}

	escalated := 0
	for _, check := range checks {
		// Mark as no response
		s.repo.UpdateSafetyCheckResponse(ctx, check.ID, SafetyCheckNoResponse, "")

		// Escalate
		if !check.Escalated {
			s.escalateSafetyCheck(ctx, check.ID, check.UserID)
			escalated++
		}
	}

	return escalated, nil
}

// ========================================
// TRUSTED/BLOCKED DRIVERS
// ========================================

// AddTrustedDriver adds a driver to a rider's trusted list
func (s *Service) AddTrustedDriver(ctx context.Context, riderID, driverID uuid.UUID, note string) error {
	return s.repo.AddTrustedDriver(ctx, riderID, driverID, note)
}

// RemoveTrustedDriver removes a driver from the trusted list
func (s *Service) RemoveTrustedDriver(ctx context.Context, riderID, driverID uuid.UUID) error {
	return s.repo.RemoveTrustedDriver(ctx, riderID, driverID)
}

// BlockDriver blocks a driver
func (s *Service) BlockDriver(ctx context.Context, riderID, driverID uuid.UUID, reason string) error {
	// Also remove from trusted if exists
	s.repo.RemoveTrustedDriver(ctx, riderID, driverID)
	return s.repo.AddBlockedDriver(ctx, riderID, driverID, reason)
}

// UnblockDriver unblocks a driver
func (s *Service) UnblockDriver(ctx context.Context, riderID, driverID uuid.UUID) error {
	return s.repo.RemoveBlockedDriver(ctx, riderID, driverID)
}

// GetBlockedDrivers retrieves all blocked drivers for a rider
func (s *Service) GetBlockedDrivers(ctx context.Context, riderID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.GetBlockedDrivers(ctx, riderID)
}

// IsDriverBlocked checks if a driver is blocked
func (s *Service) IsDriverBlocked(ctx context.Context, riderID, driverID uuid.UUID) (bool, error) {
	return s.repo.IsDriverBlocked(ctx, riderID, driverID)
}

// ========================================
// INCIDENT REPORTS
// ========================================

// ReportIncident creates a new safety incident report
func (s *Service) ReportIncident(ctx context.Context, userID uuid.UUID, req *ReportIncidentRequest) (*SafetyIncidentReport, error) {
	report := &SafetyIncidentReport{
		ID:             uuid.New(),
		ReporterID:     userID,
		ReportedUserID: req.ReportedUserID,
		RideID:         req.RideID,
		Category:       req.Category,
		Severity:       req.Severity,
		Description:    req.Description,
		Photos:         req.Photos,
		Status:         "pending",
		CreatedAt:      time.Now(),
	}

	if err := s.repo.CreateSafetyIncidentReport(ctx, report); err != nil {
		return nil, err
	}

	// Notify safety team
	go s.notifySafetyTeam(context.Background(), report)

	return report, nil
}

// ========================================
// ROUTE DEVIATION
// ========================================

// CheckRouteDeviation checks if driver deviated from route
func (s *Service) CheckRouteDeviation(ctx context.Context, rideID, driverID uuid.UUID, actualLatitude, actualLongitude, expectedLatitude, expectedLongitude float64) (*RouteDeviationAlert, error) {
	// Calculate deviation distance
	deviation := haversineDistance(actualLatitude, actualLongitude, expectedLatitude, expectedLongitude)
	deviationMeters := int(deviation * 1000)

	// Only alert if deviation is significant (> 500 meters)
	if deviationMeters < 500 {
		return nil, nil
	}

	alert := &RouteDeviationAlert{
		ID:              uuid.New(),
		RideID:          rideID,
		DriverID:        driverID,
		ExpectedLatitude:  expectedLatitude,
		ExpectedLongitude: expectedLongitude,
		ActualLatitude:    actualLatitude,
		ActualLongitude:   actualLongitude,
		DeviationMeters: deviationMeters,
		RiderNotified:   false,
		Acknowledged:    false,
		CreatedAt:       time.Now(),
	}

	if err := s.repo.CreateRouteDeviationAlert(ctx, alert); err != nil {
		return nil, err
	}

	// Notify rider
	go s.notifyRiderOfDeviation(context.Background(), alert)

	return alert, nil
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *Service) enrichAlertWithAddress(ctx context.Context, alertID uuid.UUID, latitude, longitude float64) {
	// Call maps service to reverse geocode
	// Update alert with address
}

func (s *Service) notifyEmergencyContact(ctx context.Context, contact *EmergencyContact, alert *EmergencyAlert) error {
	// Send SMS notification
	message := fmt.Sprintf(
		"EMERGENCY ALERT: %s has triggered an SOS alert. Location: https://maps.google.com/?q=%f,%f",
		"Your contact", // Would use actual user name
		alert.Latitude,
		alert.Longitude,
	)

	// Call notification service
	payload := map[string]interface{}{
		"type":    "sms",
		"to":      contact.Phone,
		"message": message,
	}

	_, err := s.notificationClient.Post(ctx, "/api/v1/notifications/send", payload, nil)
	return err
}

func (s *Service) notifyRideParticipants(ctx context.Context, rideID, triggeredByID uuid.UUID, alert *EmergencyAlert) {
	// Get ride details and notify the other participant
	// If rider triggered, notify driver and vice versa
}

func (s *Service) publishSOSEvent(alert *EmergencyAlert) {
	if s.eventBus == nil {
		return
	}

	event, _ := eventbus.NewEvent("safety.sos.triggered", "safety-service", map[string]interface{}{
		"alert_id":  alert.ID.String(),
		"user_id":   alert.UserID.String(),
		"ride_id":   alert.RideID,
		"type":      alert.Type,
		"latitude":  alert.Latitude,
		"longitude": alert.Longitude,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.eventBus.Publish(ctx, "safety.sos", event)
}

func (s *Service) notifyAdminDashboard(alert *EmergencyAlert) {
	if s.wsHub == nil {
		return
	}

	// Send to all admin clients
	s.wsHub.SendToAll(&websocket.Message{
		Type: "emergency_alert",
		Data: map[string]interface{}{
			"alert_id":  alert.ID.String(),
			"user_id":   alert.UserID.String(),
			"type":      alert.Type,
			"status":    alert.Status,
			"latitude":  alert.Latitude,
			"longitude": alert.Longitude,
			"created_at": alert.CreatedAt,
		},
	})
}

func (s *Service) sendContactVerification(ctx context.Context, contact *EmergencyContact) {
	// Generate verification code
	code := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)

	// Store in Redis with TTL
	if s.redis != nil {
		key := fmt.Sprintf("contact_verify:%s", contact.ID.String())
		s.redis.SetWithExpiration(ctx, key, code, 10*time.Minute)
	}

	// Send SMS
	payload := map[string]interface{}{
		"type":    "sms",
		"to":      contact.Phone,
		"message": fmt.Sprintf("Your emergency contact verification code is: %s", code),
	}

	if _, err := s.notificationClient.Post(ctx, "/api/v1/notifications/send", payload, nil); err != nil {
		logger.WithContext(ctx).Warn("failed to send verification code SMS",
			zap.String("contact_id", contact.ID.String()),
			zap.Error(err))
	}
}

func (s *Service) notifyShareRecipient(ctx context.Context, link *RideShareLink, recipient ShareRecipientInput) {
	shareURL := fmt.Sprintf("%s/share/%s", s.baseShareURL, link.Token)
	message := fmt.Sprintf("Someone shared their ride with you. Track their journey: %s", shareURL)

	if recipient.Phone != "" {
		payload := map[string]interface{}{
			"type":    "sms",
			"to":      recipient.Phone,
			"message": message,
		}
		if _, err := s.notificationClient.Post(ctx, "/api/v1/notifications/send", payload, nil); err != nil {
			logger.WithContext(ctx).Warn("failed to send ride share SMS notification",
				zap.String("link_id", link.ID.String()),
				zap.Error(err))
		}
	}

	if recipient.Email != "" {
		payload := map[string]interface{}{
			"type":    "email",
			"to":      recipient.Email,
			"subject": "Ride Shared With You",
			"body":    message,
		}
		if _, err := s.notificationClient.Post(ctx, "/api/v1/notifications/send", payload, nil); err != nil {
			logger.WithContext(ctx).Warn("failed to send ride share email notification",
				zap.String("link_id", link.ID.String()),
				zap.Error(err))
		}
	}
}

func (s *Service) sendSafetyCheckNotification(ctx context.Context, check *SafetyCheck) {
	// Send push notification asking user to confirm they're safe
}

func (s *Service) escalateSafetyCheck(ctx context.Context, checkID, userID uuid.UUID) {
	// Mark as escalated
	s.repo.EscalateSafetyCheck(ctx, checkID)

	// Notify safety team
	logger.Warn("Safety check escalated", zap.String("check_id", checkID.String()), zap.String("user_id", userID.String()))

	// Could trigger SOS if needed
}

func (s *Service) notifySafetyTeam(ctx context.Context, report *SafetyIncidentReport) {
	// Notify safety team about new incident
}

func (s *Service) notifyRiderOfDeviation(ctx context.Context, alert *RouteDeviationAlert) {
	logger.Info("Notifying rider of route deviation",
		zap.String("alert_id", alert.ID.String()),
		zap.String("ride_id", alert.RideID.String()),
		zap.Int("deviation_meters", alert.DeviationMeters))

	// Build notification message
	message := fmt.Sprintf(
		"Route Alert: Your driver has deviated from the expected route by %d meters. "+
			"If you feel unsafe, use the SOS button or contact us.",
		alert.DeviationMeters,
	)

	// Send push notification via notification service
	payload := map[string]interface{}{
		"type":             "push",
		"ride_id":          alert.RideID.String(),
		"alert_type":       "route_deviation",
		"deviation_meters": alert.DeviationMeters,
		"title":            "Route Deviation Detected",
		"body":             message,
	}

	if _, err := s.notificationClient.Post(ctx, "/api/v1/notifications/send", payload, nil); err != nil {
		logger.WithContext(ctx).Warn("failed to send route deviation notification",
			zap.String("alert_id", alert.ID.String()),
			zap.String("ride_id", alert.RideID.String()),
			zap.Error(err))
		return
	}

	// Mark rider as notified in the alert
	if err := s.repo.UpdateRouteDeviationAcknowledgement(ctx, alert.ID, ""); err != nil {
		logger.WithContext(ctx).Warn("failed to update route deviation alert notification status",
			zap.String("alert_id", alert.ID.String()),
			zap.Error(err))
	}

	logger.Info("Route deviation notification sent successfully",
		zap.String("alert_id", alert.ID.String()),
		zap.String("ride_id", alert.RideID.String()))
}

// haversineDistance calculates the distance between two points in km
func haversineDistance(latitude1, longitude1, latitude2, longitude2 float64) float64 {
	const earthRadius = 6371.0

	deltaLatitude := (latitude2 - latitude1) * 3.141592653589793 / 180.0
	deltaLongitude := (longitude2 - longitude1) * 3.141592653589793 / 180.0

	a := (deltaLatitude/2)*(deltaLatitude/2) + (latitude1*3.141592653589793/180.0)*(latitude2*3.141592653589793/180.0)*(deltaLongitude/2)*(deltaLongitude/2)
	c := 2 * (a)

	return earthRadius * c
}
