package safety

import (
	"time"

	"github.com/google/uuid"
)

// EmergencyType represents the type of emergency
type EmergencyType string

const (
	EmergencySOS       EmergencyType = "sos"
	EmergencyAccident  EmergencyType = "accident"
	EmergencyMedical   EmergencyType = "medical"
	EmergencyThreat    EmergencyType = "threat"
	EmergencyOther     EmergencyType = "other"
)

// EmergencyStatus represents the status of an emergency alert
type EmergencyStatus string

const (
	EmergencyStatusActive    EmergencyStatus = "active"
	EmergencyStatusResponded EmergencyStatus = "responded"
	EmergencyStatusResolved  EmergencyStatus = "resolved"
	EmergencyStatusCancelled EmergencyStatus = "cancelled"
	EmergencyStatusFalse     EmergencyStatus = "false_alarm"
)

// EmergencyAlert represents an emergency SOS alert
type EmergencyAlert struct {
	ID              uuid.UUID       `json:"id"`
	UserID          uuid.UUID       `json:"user_id"`
	RideID          *uuid.UUID      `json:"ride_id,omitempty"`
	Type            EmergencyType   `json:"type"`
	Status          EmergencyStatus `json:"status"`

	// Location at time of alert
	Latitude        float64         `json:"latitude"`
	Longitude       float64         `json:"longitude"`
	Address         string          `json:"address,omitempty"`

	// Alert details
	Description     string          `json:"description,omitempty"`
	AudioRecordingURL *string       `json:"audio_recording_url,omitempty"`
	VideoRecordingURL *string       `json:"video_recording_url,omitempty"`

	// Response tracking
	RespondedAt     *time.Time      `json:"responded_at,omitempty"`
	RespondedBy     *uuid.UUID      `json:"responded_by,omitempty"` // Admin user
	ResolvedAt      *time.Time      `json:"resolved_at,omitempty"`
	Resolution      string          `json:"resolution,omitempty"`

	// Emergency services
	PoliceNotified  bool            `json:"police_notified"`
	PoliceRefNumber string          `json:"police_ref_number,omitempty"`

	// Contacts notified
	ContactsNotified []uuid.UUID   `json:"contacts_notified,omitempty"`

	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// EmergencyContact represents a user's emergency contact
type EmergencyContact struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email,omitempty"`
	Relationship string   `json:"relationship"` // family, friend, etc.
	IsPrimary   bool      `json:"is_primary"`
	NotifyOnRide bool     `json:"notify_on_ride"` // Notify when ride starts
	NotifyOnSOS  bool     `json:"notify_on_sos"`  // Notify on emergency
	Verified    bool      `json:"verified"`       // Phone verified
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RideShareLink represents a shareable link for live ride tracking
type RideShareLink struct {
	ID          uuid.UUID  `json:"id"`
	RideID      uuid.UUID  `json:"ride_id"`
	UserID      uuid.UUID  `json:"user_id"` // The rider who created the link
	Token       string     `json:"token"`   // Unique secure token
	ExpiresAt   time.Time  `json:"expires_at"`
	IsActive    bool       `json:"is_active"`

	// Recipients
	SharedWith  []ShareRecipient `json:"shared_with,omitempty"`

	// Tracking what was shared
	ShareLocation bool      `json:"share_location"`
	ShareDriver   bool      `json:"share_driver"` // Driver name/photo
	ShareETA      bool      `json:"share_eta"`
	ShareRoute    bool      `json:"share_route"`

	// Analytics
	ViewCount   int        `json:"view_count"`
	LastViewedAt *time.Time `json:"last_viewed_at,omitempty"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ShareRecipient represents someone the ride was shared with
type ShareRecipient struct {
	ContactID   *uuid.UUID `json:"contact_id,omitempty"` // If shared with saved contact
	Name        string     `json:"name,omitempty"`
	Phone       string     `json:"phone,omitempty"`
	Email       string     `json:"email,omitempty"`
	NotifiedAt  time.Time  `json:"notified_at"`
	ViewedAt    *time.Time `json:"viewed_at,omitempty"`
}

// SafetyCheck represents a periodic safety check during a ride
type SafetyCheck struct {
	ID          uuid.UUID       `json:"id"`
	RideID      uuid.UUID       `json:"ride_id"`
	UserID      uuid.UUID       `json:"user_id"` // Who received the check
	Type        SafetyCheckType `json:"type"`
	Status      SafetyCheckStatus `json:"status"`

	// For periodic check-ins
	SentAt      time.Time       `json:"sent_at"`
	RespondedAt *time.Time      `json:"responded_at,omitempty"`
	Response    string          `json:"response,omitempty"` // "safe", "help", etc.

	// Auto-triggered checks
	TriggerReason string        `json:"trigger_reason,omitempty"` // route_deviation, long_stop, etc.

	// Escalation
	Escalated   bool            `json:"escalated"`
	EscalatedAt *time.Time      `json:"escalated_at,omitempty"`

	CreatedAt   time.Time       `json:"created_at"`
}

// SafetyCheckType represents the type of safety check
type SafetyCheckType string

const (
	SafetyCheckPeriodic     SafetyCheckType = "periodic"      // Regular check-in
	SafetyCheckRouteDeviation SafetyCheckType = "route_deviation" // Off route
	SafetyCheckLongStop     SafetyCheckType = "long_stop"     // Stopped too long
	SafetyCheckSpeedAlert   SafetyCheckType = "speed_alert"   // Excessive speed
	SafetyCheckManual       SafetyCheckType = "manual"        // Admin triggered
)

// SafetyCheckStatus represents the status of a safety check
type SafetyCheckStatus string

const (
	SafetyCheckPending   SafetyCheckStatus = "pending"
	SafetyCheckSafe      SafetyCheckStatus = "safe"       // User confirmed safe
	SafetyCheckHelp      SafetyCheckStatus = "help"       // User needs help
	SafetyCheckNoResponse SafetyCheckStatus = "no_response" // No response
	SafetyCheckExpired   SafetyCheckStatus = "expired"
)

// RouteDeviationAlert represents when a driver goes off route
type RouteDeviationAlert struct {
	ID              uuid.UUID `json:"id"`
	RideID          uuid.UUID `json:"ride_id"`
	DriverID        uuid.UUID `json:"driver_id"`

	// Expected vs actual location
	ExpectedLat     float64   `json:"expected_lat"`
	ExpectedLng     float64   `json:"expected_lng"`
	ActualLat       float64   `json:"actual_lat"`
	ActualLng       float64   `json:"actual_lng"`
	DeviationMeters int       `json:"deviation_meters"`

	// Context
	ExpectedRoute   string    `json:"expected_route,omitempty"` // Polyline
	ActualRoute     string    `json:"actual_route,omitempty"`

	// Response
	RiderNotified   bool      `json:"rider_notified"`
	DriverResponse  string    `json:"driver_response,omitempty"` // "traffic", "road_closed", etc.
	Acknowledged    bool      `json:"acknowledged"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at,omitempty"`

	CreatedAt       time.Time `json:"created_at"`
}

// SpeedAlert represents when a driver exceeds speed limits
type SpeedAlert struct {
	ID          uuid.UUID `json:"id"`
	RideID      uuid.UUID `json:"ride_id"`
	DriverID    uuid.UUID `json:"driver_id"`

	SpeedKmh    float64   `json:"speed_kmh"`
	SpeedLimit  int       `json:"speed_limit"`    // If known
	Location    Coordinate `json:"location"`
	RoadName    string    `json:"road_name,omitempty"`

	Severity    string    `json:"severity"` // minor, moderate, severe
	Duration    int       `json:"duration_seconds"` // How long over limit

	CreatedAt   time.Time `json:"created_at"`
}

// Coordinate represents a geographic point
type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// SafetySettings represents a user's safety preferences
type SafetySettings struct {
	UserID                uuid.UUID `json:"user_id"`

	// SOS settings
	SOSEnabled            bool      `json:"sos_enabled"`
	SOSGesture            string    `json:"sos_gesture"` // shake, power_button, etc.
	SOSAutoCall911        bool      `json:"sos_auto_call_911"`

	// Share settings
	AutoShareWithContacts bool      `json:"auto_share_with_contacts"`
	ShareOnRideStart      bool      `json:"share_on_ride_start"`
	ShareOnEmergency      bool      `json:"share_on_emergency"`

	// Check-in settings
	PeriodicCheckIn       bool      `json:"periodic_check_in"`
	CheckInIntervalMins   int       `json:"check_in_interval_mins"` // 0 = disabled

	// Alert preferences
	RouteDeviationAlert   bool      `json:"route_deviation_alert"`
	SpeedAlertEnabled     bool      `json:"speed_alert_enabled"`
	LongStopAlertMins     int       `json:"long_stop_alert_mins"` // 0 = disabled

	// Audio/Video
	AllowRideRecording    bool      `json:"allow_ride_recording"`
	RecordAudioOnSOS      bool      `json:"record_audio_on_sos"`

	// Trusted contacts required for night rides
	NightRideProtection   bool      `json:"night_ride_protection"`
	NightProtectionStart  string    `json:"night_protection_start"` // "22:00"
	NightProtectionEnd    string    `json:"night_protection_end"`   // "06:00"

	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// TrustedDriver represents a driver marked as trusted by a rider
type TrustedDriver struct {
	ID        uuid.UUID `json:"id"`
	RiderID   uuid.UUID `json:"rider_id"`
	DriverID  uuid.UUID `json:"driver_id"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// BlockedDriver represents a driver blocked by a rider
type BlockedDriver struct {
	ID        uuid.UUID `json:"id"`
	RiderID   uuid.UUID `json:"rider_id"`
	DriverID  uuid.UUID `json:"driver_id"`
	Reason    string    `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// SafetyIncidentReport represents a safety-related incident report
type SafetyIncidentReport struct {
	ID              uuid.UUID `json:"id"`
	ReporterID      uuid.UUID `json:"reporter_id"`
	ReportedUserID  *uuid.UUID `json:"reported_user_id,omitempty"`
	RideID          *uuid.UUID `json:"ride_id,omitempty"`

	Category        string    `json:"category"` // harassment, unsafe_driving, etc.
	Severity        string    `json:"severity"` // low, medium, high, critical
	Description     string    `json:"description"`

	// Evidence
	Photos          []string  `json:"photos,omitempty"`
	AudioURL        string    `json:"audio_url,omitempty"`
	VideoURL        string    `json:"video_url,omitempty"`

	// Admin handling
	Status          string    `json:"status"` // pending, investigating, resolved, dismissed
	AssignedTo      *uuid.UUID `json:"assigned_to,omitempty"`
	Resolution      string    `json:"resolution,omitempty"`
	ActionTaken     string    `json:"action_taken,omitempty"`

	// Timeline
	CreatedAt       time.Time `json:"created_at"`
	InvestigatedAt  *time.Time `json:"investigated_at,omitempty"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
}

// API Request/Response types

// TriggerSOSRequest represents a request to trigger an emergency alert
type TriggerSOSRequest struct {
	Type        EmergencyType `json:"type" binding:"required"`
	RideID      *uuid.UUID    `json:"ride_id,omitempty"`
	Latitude    float64       `json:"latitude" binding:"required"`
	Longitude   float64       `json:"longitude" binding:"required"`
	Description string        `json:"description,omitempty"`
}

// TriggerSOSResponse represents the response to an SOS trigger
type TriggerSOSResponse struct {
	AlertID         uuid.UUID `json:"alert_id"`
	Status          string    `json:"status"`
	Message         string    `json:"message"`
	ContactsNotified int      `json:"contacts_notified"`
	EmergencyNumber string    `json:"emergency_number"` // Local 911 equivalent
	ShareLink       string    `json:"share_link,omitempty"`
}

// CreateEmergencyContactRequest represents a request to add an emergency contact
type CreateEmergencyContactRequest struct {
	Name         string `json:"name" binding:"required"`
	Phone        string `json:"phone" binding:"required"`
	Email        string `json:"email,omitempty"`
	Relationship string `json:"relationship" binding:"required"`
	IsPrimary    bool   `json:"is_primary"`
	NotifyOnRide bool   `json:"notify_on_ride"`
	NotifyOnSOS  bool   `json:"notify_on_sos"`
}

// CreateShareLinkRequest represents a request to create a ride share link
type CreateShareLinkRequest struct {
	RideID        uuid.UUID `json:"ride_id" binding:"required"`
	ExpiryMinutes int       `json:"expiry_minutes,omitempty"` // 0 = until ride ends
	ShareLocation bool      `json:"share_location"`
	ShareDriver   bool      `json:"share_driver"`
	ShareETA      bool      `json:"share_eta"`
	ShareRoute    bool      `json:"share_route"`
	SendTo        []ShareRecipientInput `json:"send_to,omitempty"`
}

// ShareRecipientInput represents a recipient for ride sharing
type ShareRecipientInput struct {
	ContactID *uuid.UUID `json:"contact_id,omitempty"`
	Phone     string     `json:"phone,omitempty"`
	Email     string     `json:"email,omitempty"`
}

// ShareLinkResponse represents a ride share link
type ShareLinkResponse struct {
	LinkID    uuid.UUID `json:"link_id"`
	ShareURL  string    `json:"share_url"`
	ExpiresAt time.Time `json:"expires_at"`
	Token     string    `json:"token"`
}

// SharedRideView represents what a recipient sees when viewing a shared ride
type SharedRideView struct {
	RideID        uuid.UUID  `json:"ride_id"`
	Status        string     `json:"status"`

	// Location (if shared)
	CurrentLocation *Coordinate `json:"current_location,omitempty"`

	// Driver info (if shared)
	DriverFirstName string     `json:"driver_first_name,omitempty"`
	DriverPhoto     string     `json:"driver_photo,omitempty"`
	VehicleMake     string     `json:"vehicle_make,omitempty"`
	VehicleModel    string     `json:"vehicle_model,omitempty"`
	VehicleColor    string     `json:"vehicle_color,omitempty"`
	LicensePlate    string     `json:"license_plate,omitempty"`

	// Route (if shared)
	PickupAddress   string     `json:"pickup_address,omitempty"`
	DropoffAddress  string     `json:"dropoff_address,omitempty"`
	RoutePolyline   string     `json:"route_polyline,omitempty"`

	// ETA (if shared)
	ETAMinutes      int        `json:"eta_minutes,omitempty"`
	EstimatedArrival *time.Time `json:"estimated_arrival,omitempty"`

	// Safety
	EmergencyContact string    `json:"emergency_contact"` // Local emergency number
	ReportConcernURL string    `json:"report_concern_url"`

	LastUpdated      time.Time `json:"last_updated"`
}

// UpdateSafetySettingsRequest represents a request to update safety settings
type UpdateSafetySettingsRequest struct {
	SOSEnabled            *bool   `json:"sos_enabled,omitempty"`
	SOSGesture            *string `json:"sos_gesture,omitempty"`
	SOSAutoCall911        *bool   `json:"sos_auto_call_911,omitempty"`
	AutoShareWithContacts *bool   `json:"auto_share_with_contacts,omitempty"`
	ShareOnRideStart      *bool   `json:"share_on_ride_start,omitempty"`
	ShareOnEmergency      *bool   `json:"share_on_emergency,omitempty"`
	PeriodicCheckIn       *bool   `json:"periodic_check_in,omitempty"`
	CheckInIntervalMins   *int    `json:"check_in_interval_mins,omitempty"`
	RouteDeviationAlert   *bool   `json:"route_deviation_alert,omitempty"`
	SpeedAlertEnabled     *bool   `json:"speed_alert_enabled,omitempty"`
	LongStopAlertMins     *int    `json:"long_stop_alert_mins,omitempty"`
	AllowRideRecording    *bool   `json:"allow_ride_recording,omitempty"`
	RecordAudioOnSOS      *bool   `json:"record_audio_on_sos,omitempty"`
	NightRideProtection   *bool   `json:"night_ride_protection,omitempty"`
	NightProtectionStart  *string `json:"night_protection_start,omitempty"`
	NightProtectionEnd    *string `json:"night_protection_end,omitempty"`
}

// SafetyCheckResponse represents a response to a safety check
type SafetyCheckResponse struct {
	CheckID  uuid.UUID `json:"check_id" binding:"required"`
	Response string    `json:"response" binding:"required"` // "safe" or "help"
	Notes    string    `json:"notes,omitempty"`
}

// ReportIncidentRequest represents a safety incident report
type ReportIncidentRequest struct {
	RideID         *uuid.UUID `json:"ride_id,omitempty"`
	ReportedUserID *uuid.UUID `json:"reported_user_id,omitempty"`
	Category       string     `json:"category" binding:"required"`
	Severity       string     `json:"severity" binding:"required"`
	Description    string     `json:"description" binding:"required"`
	Photos         []string   `json:"photos,omitempty"`
}
