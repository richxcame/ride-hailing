package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
	redisclient "github.com/richxcame/ride-hailing/pkg/redis"
)

// FraudSuspender is the subset of the fraud service used by the admin service.
// Using a narrow interface avoids an import cycle.
type FraudSuspender interface {
	SuspendUser(ctx context.Context, userID, adminID uuid.UUID, reason string) error
}

// Service handles business logic for admin operations
type Service struct {
	repo         RepositoryInterface
	redis        *redisclient.Client
	fraudService FraudSuspender
}

// NewService creates a new admin service.
// redis and fraudService may be nil.
func NewService(repo RepositoryInterface, redis *redisclient.Client, fraudService FraudSuspender) *Service {
	return &Service{repo: repo, redis: redis, fraudService: fraudService}
}

// GetAllUsers retrieves all users with pagination and filters
func (s *Service) GetAllUsers(ctx context.Context, limit, offset int, filter *UserFilter) ([]*models.User, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Validate role filter if provided
	if filter != nil && filter.Role != "" {
		validRoles := map[string]bool{"admin": true, "driver": true, "rider": true}
		if !validRoles[filter.Role] {
			filter.Role = "" // Ignore invalid role
		}
	}

	return s.repo.GetAllUsers(ctx, limit, offset, filter)
}

// GetUserStats retrieves user statistics by role
func (s *Service) GetUserStats(ctx context.Context) (*UserStats, error) {
	return s.repo.GetUserStats(ctx)
}

// GetUser retrieves a specific user
func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

// SuspendUser suspends a user account. When a fraud service is wired in it
// also updates the risk profile and creates a fraud alert for the suspension.
func (s *Service) SuspendUser(ctx context.Context, adminID, userID uuid.UUID, reason string) error {
	if err := s.repo.UpdateUserStatus(ctx, userID, false); err != nil {
		return err
	}
	s.repo.InsertAuditLog(ctx, adminID, "suspend_user", "user", userID, nil)
	if s.fraudService != nil {
		// Best-effort — don't fail the suspension if fraud tracking errors.
		_ = s.fraudService.SuspendUser(ctx, userID, adminID, reason)
	}
	return nil
}

// ActivateUser activates a user account
func (s *Service) ActivateUser(ctx context.Context, adminID, userID uuid.UUID) error {
	if err := s.repo.UpdateUserStatus(ctx, userID, true); err != nil {
		return err
	}
	s.repo.InsertAuditLog(ctx, adminID, "activate_user", "user", userID, nil)
	return nil
}

// GetAllDrivers retrieves all drivers with pagination and filters
func (s *Service) GetAllDrivers(ctx context.Context, limit, offset int, filter *DriverFilter) ([]*models.Driver, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Validate status filter if provided
	if filter != nil && filter.Status != "" {
		validStatuses := map[string]bool{"online": true, "offline": true, "available": true, "pending": true}
		if !validStatuses[filter.Status] {
			filter.Status = "" // Ignore invalid status
		}
	}

	drivers, total, err := s.repo.GetAllDriversWithTotal(ctx, limit, offset, filter)
	if err != nil {
		return nil, 0, err
	}

	s.enrichDriversWithRedisStatus(ctx, drivers)
	return drivers, total, nil
}

// GetDriverStats retrieves driver statistics for stats cards.
// When Redis is available, online/available/offline counts come from real-time status.
func (s *Service) GetDriverStats(ctx context.Context) (*DriverStats, error) {
	stats, err := s.repo.GetDriverStats(ctx)
	if err != nil {
		return nil, err
	}

	if s.redis == nil {
		return stats, nil
	}

	// Fetch all approved driver user_ids to check their Redis status
	drivers, _, err := s.repo.GetAllDriversWithTotal(ctx, 10000, 0, nil)
	if err != nil {
		return stats, nil // Fall back to DB stats
	}

	// Only check approved drivers
	var approvedDrivers []*models.Driver
	for _, d := range drivers {
		if d.ApprovalStatus == "approved" {
			approvedDrivers = append(approvedDrivers, d)
		}
	}

	if len(approvedDrivers) == 0 {
		return stats, nil
	}

	keys := make([]string, len(approvedDrivers))
	for i, d := range approvedDrivers {
		keys[i] = fmt.Sprintf("driver:status:%s", d.UserID.String())
	}

	statuses, err := s.redis.MGetStrings(ctx, keys...)
	if err != nil {
		return stats, nil // Fall back to DB stats
	}

	online, available, offline := 0, 0, 0
	for _, raw := range statuses {
		if raw == "" {
			offline++
			continue
		}
		var statusData struct {
			Status string `json:"status"`
		}
		if json.Unmarshal([]byte(raw), &statusData) != nil {
			offline++
			continue
		}
		switch statusData.Status {
		case "available":
			online++
			available++
		case "busy":
			online++
		default:
			offline++
		}
	}

	stats.OnlineDrivers = online
	stats.AvailableDrivers = available
	stats.OfflineDrivers = len(approvedDrivers) - online

	return stats, nil
}

// GetPendingDrivers retrieves drivers awaiting approval
func (s *Service) GetPendingDrivers(ctx context.Context, limit, offset int) ([]*models.Driver, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.GetPendingDriversWithTotal(ctx, limit, offset)
}

// GetDriver retrieves a specific driver by ID
func (s *Service) GetDriver(ctx context.Context, driverID uuid.UUID) (*models.Driver, error) {
	driver, err := s.repo.GetDriverByID(ctx, driverID)
	if err != nil {
		return nil, err
	}

	s.enrichDriversWithRedisStatus(ctx, []*models.Driver{driver})
	return driver, nil
}

// enrichDriversWithRedisStatus overrides is_online/is_available on each driver
// using real-time data from Redis driver:status:{user_id} keys.
func (s *Service) enrichDriversWithRedisStatus(ctx context.Context, drivers []*models.Driver) {
	if s.redis == nil || len(drivers) == 0 {
		return
	}

	keys := make([]string, len(drivers))
	for i, d := range drivers {
		keys[i] = fmt.Sprintf("driver:status:%s", d.UserID.String())
	}

	statuses, err := s.redis.MGetStrings(ctx, keys...)
	if err != nil {
		return // Silently fall back to DB values
	}

	for i, raw := range statuses {
		if raw == "" {
			// No Redis key → driver is offline
			drivers[i].IsOnline = false
			drivers[i].IsAvailable = false
			continue
		}

		var statusData struct {
			Status string `json:"status"`
		}
		if json.Unmarshal([]byte(raw), &statusData) != nil {
			continue // Keep DB values on parse error
		}

		switch statusData.Status {
		case "available":
			drivers[i].IsOnline = true
			drivers[i].IsAvailable = true
		case "busy":
			drivers[i].IsOnline = true
			drivers[i].IsAvailable = false
		default: // "offline" or unknown
			drivers[i].IsOnline = false
			drivers[i].IsAvailable = false
		}
	}
}

// ApproveDriver approves a driver application
func (s *Service) ApproveDriver(ctx context.Context, adminID, driverID uuid.UUID) error {
	if err := s.repo.ApproveDriver(ctx, driverID); err != nil {
		return err
	}
	s.repo.InsertAuditLog(ctx, adminID, "approve_driver", "driver", driverID, nil)
	return nil
}

// RejectDriver rejects a driver application
func (s *Service) RejectDriver(ctx context.Context, adminID, driverID uuid.UUID) error {
	if err := s.repo.RejectDriver(ctx, driverID); err != nil {
		return err
	}
	s.repo.InsertAuditLog(ctx, adminID, "reject_driver", "driver", driverID, nil)
	return nil
}

// GetDashboardStats retrieves overall dashboard statistics
func (s *Service) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	// Get user stats
	userStats, err := s.repo.GetUserStats(ctx)
	if err != nil {
		return nil, err
	}

	// Get ride stats (all time)
	rideStats, err := s.repo.GetRideStats(ctx, nil, nil)
	if err != nil {
		return nil, err
	}

	// Get today's stats
	today := time.Now().Truncate(24 * time.Hour)
	todayStats, err := s.repo.GetRideStats(ctx, &today, nil)
	if err != nil {
		return nil, err
	}

	return &DashboardStats{
		Users:      userStats,
		Rides:      rideStats,
		TodayRides: todayStats,
	}, nil
}

// GetRideStats retrieves ride statistics for a date range
func (s *Service) GetRideStats(ctx context.Context, startDate, endDate *time.Time) (*RideStats, error) {
	return s.repo.GetRideStats(ctx, startDate, endDate)
}

// GetRide retrieves a specific ride by ID with rider and driver details
func (s *Service) GetRide(ctx context.Context, rideID uuid.UUID) (*AdminRideDetail, error) {
	return s.repo.GetRideByID(ctx, rideID)
}

// GetRecentRides retrieves recent rides for monitoring with rider and driver details
func (s *Service) GetRecentRides(ctx context.Context, limit, offset int) ([]*AdminRideDetail, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.repo.GetRecentRidesWithDetails(ctx, limit, offset)
}

// GetRealtimeMetrics retrieves real-time dashboard metrics
func (s *Service) GetRealtimeMetrics(ctx context.Context) (*RealtimeMetrics, error) {
	metrics, err := s.repo.GetRealtimeMetrics(ctx)
	if err != nil {
		return nil, err
	}

	// Enrich with Redis driver counts
	if s.redis != nil {
		stats, statsErr := s.GetDriverStats(ctx)
		if statsErr == nil {
			metrics.OnlineDrivers = stats.OnlineDrivers
			metrics.AvailableDrivers = stats.AvailableDrivers
		}
	}

	return metrics, nil
}

// GetDashboardSummary retrieves comprehensive dashboard summary
func (s *Service) GetDashboardSummary(ctx context.Context, period string) (*DashboardSummary, error) {
	summary, err := s.repo.GetDashboardSummary(ctx, period)
	if err != nil {
		return nil, err
	}

	// Enrich driver section with Redis real-time counts
	if s.redis != nil && summary.Drivers != nil {
		stats, statsErr := s.GetDriverStats(ctx)
		if statsErr == nil {
			summary.Drivers.OnlineNow = stats.OnlineDrivers
			summary.Drivers.AvailableNow = stats.AvailableDrivers
			summary.Drivers.BusyNow = stats.OnlineDrivers - stats.AvailableDrivers
		}
	}

	return summary, nil
}

// GetRevenueTrend retrieves revenue trend data
func (s *Service) GetRevenueTrend(ctx context.Context, period, groupBy string) (*RevenueTrend, error) {
	return s.repo.GetRevenueTrend(ctx, period, groupBy)
}

// GetActionItems retrieves items requiring admin attention
func (s *Service) GetActionItems(ctx context.Context) (*ActionItems, error) {
	return s.repo.GetActionItems(ctx)
}

// GetActivityFeed retrieves recent activity events across the system
func (s *Service) GetActivityFeed(ctx context.Context, limit, offset int) ([]*ActivityFeedItem, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetActivityFeed(ctx, limit, offset)
}

// GetAuditLogs retrieves audit logs with pagination and filters
func (s *Service) GetAuditLogs(ctx context.Context, limit, offset int, filter *AuditLogFilter) ([]*AuditLog, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetAuditLogs(ctx, limit, offset, filter)
}

// DashboardStats represents overall dashboard statistics
type DashboardStats struct {
	Users      *UserStats `json:"users"`
	Rides      *RideStats `json:"rides"`
	TodayRides *RideStats `json:"today_rides"`
}

// RealtimeMetrics represents real-time dashboard metrics
type RealtimeMetrics struct {
	ActiveRides        int     `json:"active_rides"`
	AvailableDrivers   int     `json:"available_drivers"`
	PendingRequests    int     `json:"pending_requests"`
	TodayRevenue       float64 `json:"today_revenue"`
	TodayRevenueChange float64 `json:"today_revenue_change"`
	OnlineDrivers      int     `json:"online_drivers"`
	TotalRidersActive  int     `json:"total_riders_active"`
	AvgWaitTime        float64 `json:"avg_wait_time"`
	AvgETA             float64 `json:"avg_eta"`
}

// DashboardSummary represents comprehensive dashboard metrics
type DashboardSummary struct {
	Rides   *SummaryRides   `json:"rides"`
	Drivers *SummaryDrivers `json:"drivers"`
	Riders  *SummaryRiders  `json:"riders"`
	Revenue *SummaryRevenue `json:"revenue"`
	Alerts  *SummaryAlerts  `json:"alerts"`
}

type SummaryRides struct {
	Total               int     `json:"total"`
	Completed           int     `json:"completed"`
	Cancelled           int     `json:"cancelled"`
	InProgress          int     `json:"in_progress"`
	Pending             int     `json:"pending"`
	CompletionRate      float64 `json:"completion_rate"`
	CancellationRate    float64 `json:"cancellation_rate"`
	CancelledByRider    int     `json:"cancelled_by_rider"`
	CancelledByDriver   int     `json:"cancelled_by_driver"`
	AvgDuration         float64 `json:"avg_duration"`
	AvgDistance         float64 `json:"avg_distance"`
	ChangeVsPrevious    float64 `json:"change_vs_previous"`
}

type SummaryDrivers struct {
	TotalActive       int     `json:"total_active"`
	OnlineNow         int     `json:"online_now"`
	AvailableNow      int     `json:"available_now"`
	BusyNow           int     `json:"busy_now"`
	PendingApprovals  int     `json:"pending_approvals"`
	AvgRating         float64 `json:"avg_rating"`
	UtilizationRate   float64 `json:"utilization_rate"`
	NewSignups        int     `json:"new_signups"`
}

type SummaryRiders struct {
	TotalActive   int     `json:"total_active"`
	NewSignups    int     `json:"new_signups"`
	ActiveToday   int     `json:"active_today"`
	RetentionRate float64 `json:"retention_rate"`
}

type SummaryRevenue struct {
	Total              float64                `json:"total"`
	Commission         float64                `json:"commission"`
	DriverEarnings     float64                `json:"driver_earnings"`
	AvgFare            float64                `json:"avg_fare"`
	ChangeVsPrevious   float64                `json:"change_vs_previous"`
	ByPaymentMethod    []PaymentMethodRevenue `json:"by_payment_method"`
}

type PaymentMethodRevenue struct {
	Method     string  `json:"method"`
	Amount     float64 `json:"amount"`
	Percentage float64 `json:"percentage"`
}

type SummaryAlerts struct {
	FraudAlerts           int `json:"fraud_alerts"`
	CriticalAlerts        int `json:"critical_alerts"`
	PendingInvestigations int `json:"pending_investigations"`
}

// RevenueTrend represents revenue trend data
type RevenueTrend struct {
	Period           string             `json:"period"`
	GroupBy          string             `json:"group_by"`
	TotalRevenue     float64            `json:"total_revenue"`
	AvgDailyRevenue  float64            `json:"avg_daily_revenue"`
	Trend            []RevenueTrendData `json:"trend"`
}

type RevenueTrendData struct {
	Date       string  `json:"date"`
	Revenue    float64 `json:"revenue"`
	Rides      int     `json:"rides"`
	AvgFare    float64 `json:"avg_fare"`
	Commission float64 `json:"commission"`
}

// ActionItems represents items requiring admin attention
type ActionItems struct {
	PendingDriverApprovals *ActionDriverApprovals  `json:"pending_driver_approvals"`
	FraudAlerts            *ActionFraudAlerts      `json:"fraud_alerts"`
	NegativeFeedback       *ActionNegativeFeedback `json:"negative_feedback"`
	LowBalanceDrivers      *ActionLowBalance       `json:"low_balance_drivers"`
	ExpiredDocuments       *ActionExpiredDocs      `json:"expired_documents"`
}

type ActionDriverApprovals struct {
	Count       int                   `json:"count"`
	UrgentCount int                   `json:"urgent_count"`
	Items       []PendingDriverItem   `json:"items"`
}

type PendingDriverItem struct {
	DriverID    uuid.UUID `json:"driver_id"`
	DriverName  string    `json:"driver_name"`
	SubmittedAt time.Time `json:"submitted_at"`
	DaysWaiting int       `json:"days_waiting"`
}

type ActionFraudAlerts struct {
	Count         int               `json:"count"`
	CriticalCount int               `json:"critical_count"`
	HighCount     int               `json:"high_count"`
	Items         []FraudAlertItem  `json:"items"`
}

type FraudAlertItem struct {
	AlertID    uuid.UUID `json:"alert_id"`
	AlertType  string    `json:"alert_type"`
	AlertLevel string    `json:"alert_level"`
	UserID     uuid.UUID `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type ActionNegativeFeedback struct {
	Count        int                  `json:"count"`
	OneStarCount int                  `json:"one_star_count"`
	Items        []NegativeFeedbackItem `json:"items"`
}

type NegativeFeedbackItem struct {
	RideID     uuid.UUID  `json:"ride_id"`
	DriverID   *uuid.UUID `json:"driver_id,omitempty"`
	DriverName string     `json:"driver_name"`
	Rating     *int       `json:"rating,omitempty"`
	Feedback   *string    `json:"feedback,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type ActionLowBalance struct {
	Count int                  `json:"count"`
	Items []LowBalanceItem     `json:"items"`
}

type LowBalanceItem struct {
	DriverID   uuid.UUID  `json:"driver_id"`
	DriverName string     `json:"driver_name"`
	Balance    float64    `json:"balance"`
	LastRide   *time.Time `json:"last_ride,omitempty"`
}

type ActionExpiredDocs struct {
	Count int                 `json:"count"`
	Items []ExpiredDocItem    `json:"items"`
}

type ExpiredDocItem struct {
	DriverID     uuid.UUID `json:"driver_id"`
	DriverName   string    `json:"driver_name"`
	DocumentType string    `json:"document_type"`
	ExpiredAt    time.Time `json:"expired_at"`
}

// ActivityFeedItem represents a single event in the activity feed
type ActivityFeedItem struct {
	ID          uuid.UUID  `json:"id"`
	EventType   string     `json:"event_type"`
	Description string     `json:"description"`
	ActorName   string     `json:"actor_name"`
	ActorRole   string     `json:"actor_role"`
	EntityType  string     `json:"entity_type"`
	EntityID    *uuid.UUID `json:"entity_id,omitempty"`
	Metadata    string     `json:"metadata,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
