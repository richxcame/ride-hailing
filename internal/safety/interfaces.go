package safety

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RepositoryInterface defines all public methods of the Repository
type RepositoryInterface interface {
	// Emergency Alerts
	CreateEmergencyAlert(ctx context.Context, alert *EmergencyAlert) error
	GetEmergencyAlert(ctx context.Context, id uuid.UUID) (*EmergencyAlert, error)
	UpdateEmergencyAlertStatus(ctx context.Context, id uuid.UUID, status EmergencyStatus, respondedBy *uuid.UUID, resolution string) error
	GetActiveEmergencyAlerts(ctx context.Context) ([]*EmergencyAlert, error)
	GetUserEmergencyAlerts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*EmergencyAlert, error)

	// Emergency Contacts
	CreateEmergencyContact(ctx context.Context, contact *EmergencyContact) error
	GetEmergencyContact(ctx context.Context, id uuid.UUID) (*EmergencyContact, error)
	GetUserEmergencyContacts(ctx context.Context, userID uuid.UUID) ([]*EmergencyContact, error)
	GetSOSContacts(ctx context.Context, userID uuid.UUID) ([]*EmergencyContact, error)
	UpdateEmergencyContact(ctx context.Context, contact *EmergencyContact) error
	DeleteEmergencyContact(ctx context.Context, id uuid.UUID) error
	VerifyEmergencyContact(ctx context.Context, id uuid.UUID) error

	// Ride Share Links
	CreateRideShareLink(ctx context.Context, link *RideShareLink) error
	GetRideShareLinkByToken(ctx context.Context, token string) (*RideShareLink, error)
	GetRideShareLinks(ctx context.Context, rideID uuid.UUID) ([]*RideShareLink, error)
	IncrementShareLinkViewCount(ctx context.Context, id uuid.UUID) error
	DeactivateShareLink(ctx context.Context, id uuid.UUID) error
	DeactivateRideShareLinks(ctx context.Context, rideID uuid.UUID) error

	// Safety Settings
	GetSafetySettings(ctx context.Context, userID uuid.UUID) (*SafetySettings, error)
	UpdateSafetySettings(ctx context.Context, settings *SafetySettings) error

	// Safety Checks
	CreateSafetyCheck(ctx context.Context, check *SafetyCheck) error
	UpdateSafetyCheckResponse(ctx context.Context, id uuid.UUID, status SafetyCheckStatus, response string) error
	GetPendingSafetyChecks(ctx context.Context, userID uuid.UUID) ([]*SafetyCheck, error)
	EscalateSafetyCheck(ctx context.Context, id uuid.UUID) error
	GetExpiredPendingSafetyChecks(ctx context.Context, timeout time.Duration) ([]*SafetyCheck, error)

	// Route Deviation Alerts
	CreateRouteDeviationAlert(ctx context.Context, alert *RouteDeviationAlert) error
	UpdateRouteDeviationAcknowledgement(ctx context.Context, id uuid.UUID, response string) error
	GetRecentRouteDeviations(ctx context.Context, rideID uuid.UUID, since time.Duration) ([]*RouteDeviationAlert, error)

	// Safety Incident Reports
	CreateSafetyIncidentReport(ctx context.Context, report *SafetyIncidentReport) error
	GetSafetyIncidentReport(ctx context.Context, id uuid.UUID) (*SafetyIncidentReport, error)
	UpdateSafetyIncidentStatus(ctx context.Context, id uuid.UUID, status string, resolution, actionTaken string) error

	// Trusted/Blocked Drivers
	AddTrustedDriver(ctx context.Context, riderID, driverID uuid.UUID, note string) error
	RemoveTrustedDriver(ctx context.Context, riderID, driverID uuid.UUID) error
	IsDriverTrusted(ctx context.Context, riderID, driverID uuid.UUID) (bool, error)
	AddBlockedDriver(ctx context.Context, riderID, driverID uuid.UUID, reason string) error
	RemoveBlockedDriver(ctx context.Context, riderID, driverID uuid.UUID) error
	IsDriverBlocked(ctx context.Context, riderID, driverID uuid.UUID) (bool, error)
	GetBlockedDrivers(ctx context.Context, riderID uuid.UUID) ([]uuid.UUID, error)

	// Statistics
	CountUserEmergencyAlerts(ctx context.Context, userID uuid.UUID) (int, error)
	GetEmergencyAlertStats(ctx context.Context, since time.Time) (map[string]interface{}, error)
}
