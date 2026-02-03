package tips

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
)

const (
	maxTipAmount     = 200.0
	minTipAmount     = 1.0
	tipWindowHours   = 72 // Hours after ride to allow tipping
	defaultCurrency  = "USD"
)

// Service handles tipping business logic
type Service struct {
	repo *Repository
}

// NewService creates a new tips service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// SendTip creates a tip from a rider to a driver
func (s *Service) SendTip(ctx context.Context, riderID uuid.UUID, driverID uuid.UUID, req *SendTipRequest) (*Tip, error) {
	if req.Amount < minTipAmount {
		return nil, common.NewBadRequestError(fmt.Sprintf("minimum tip is %.2f", minTipAmount), nil)
	}
	if req.Amount > maxTipAmount {
		return nil, common.NewBadRequestError(fmt.Sprintf("maximum tip is %.2f", maxTipAmount), nil)
	}

	// Check if already tipped for this ride
	existing, err := s.repo.GetTipByRideAndRider(ctx, req.RideID, riderID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}
	if existing != nil {
		return nil, common.NewConflictError("you have already tipped for this ride")
	}

	now := time.Now()
	tip := &Tip{
		ID:          uuid.New(),
		RideID:      req.RideID,
		RiderID:     riderID,
		DriverID:    driverID,
		Amount:      req.Amount,
		Currency:    defaultCurrency,
		Status:      TipStatusCompleted,
		Message:     req.Message,
		IsAnonymous: req.IsAnonymous,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateTip(ctx, tip); err != nil {
		return nil, fmt.Errorf("create tip: %w", err)
	}

	return tip, nil
}

// GetTipPresets returns suggested tip amounts based on ride fare
func (s *Service) GetTipPresets(fareAmount float64) *TipPresetsResponse {
	presets := []TipPreset{
		{Amount: fareAmount * 0.10, Label: "10%", IsPercent: true, IsDefault: false},
		{Amount: fareAmount * 0.15, Label: "15%", IsPercent: true, IsDefault: true},
		{Amount: fareAmount * 0.20, Label: "20%", IsPercent: true, IsDefault: false},
		{Amount: fareAmount * 0.25, Label: "25%", IsPercent: true, IsDefault: false},
	}

	// Round to 2 decimal places
	for i := range presets {
		presets[i].Amount = float64(int(presets[i].Amount*100)) / 100
		if presets[i].Amount < minTipAmount {
			presets[i].Amount = minTipAmount
		}
	}

	return &TipPresetsResponse{
		Presets:  presets,
		MaxTip:   maxTipAmount,
		Currency: defaultCurrency,
	}
}

// GetDriverTipSummary returns tip stats for a driver
func (s *Service) GetDriverTipSummary(ctx context.Context, driverID uuid.UUID, period string) (*DriverTipSummary, error) {
	from, to := s.periodToTimeRange(period)

	summary, err := s.repo.GetDriverTipSummary(ctx, driverID, from, to)
	if err != nil {
		return nil, err
	}
	summary.Period = period

	return summary, nil
}

// GetRiderTipHistory returns tips given by a rider
func (s *Service) GetRiderTipHistory(ctx context.Context, riderID uuid.UUID, limit, offset int) (*RiderTipHistory, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	tips, total, err := s.repo.GetTipsByRider(ctx, riderID, limit, offset)
	if err != nil {
		return nil, err
	}
	if tips == nil {
		tips = []Tip{}
	}

	return &RiderTipHistory{
		Tips:  tips,
		Total: total,
	}, nil
}

// GetDriverTips returns tips received by a driver
func (s *Service) GetDriverTips(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]Tip, int, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	tips, total, err := s.repo.GetTipsByDriver(ctx, driverID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if tips == nil {
		tips = []Tip{}
	}

	return tips, total, nil
}

func (s *Service) periodToTimeRange(period string) (time.Time, time.Time) {
	now := time.Now()
	to := now

	switch period {
	case "today":
		from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return from, to
	case "this_week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		from := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		return from, to
	case "this_month":
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return from, to
	default:
		// Default to all time
		from := time.Date(2020, 1, 1, 0, 0, 0, 0, now.Location())
		return from, to
	}
}
