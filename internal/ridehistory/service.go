package ridehistory

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
)

// Service handles ride history business logic
type Service struct {
	repo *Repository
}

// NewService creates a new ride history service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetRiderHistory returns paginated ride history for a rider
func (s *Service) GetRiderHistory(ctx context.Context, riderID uuid.UUID, filters *HistoryFilters, page, pageSize int) (*HistoryResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rides, total, err := s.repo.GetRiderHistory(ctx, riderID, filters, pageSize, offset)
	if err != nil {
		return nil, err
	}
	if rides == nil {
		rides = []RideHistoryEntry{}
	}

	return &HistoryResponse{
		Rides:    rides,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetDriverHistory returns paginated ride history for a driver
func (s *Service) GetDriverHistory(ctx context.Context, driverID uuid.UUID, filters *HistoryFilters, page, pageSize int) (*HistoryResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rides, total, err := s.repo.GetDriverHistory(ctx, driverID, filters, pageSize, offset)
	if err != nil {
		return nil, err
	}
	if rides == nil {
		rides = []RideHistoryEntry{}
	}

	return &HistoryResponse{
		Rides:    rides,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetRideDetails returns full details of a specific ride
func (s *Service) GetRideDetails(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) (*RideHistoryEntry, error) {
	ride, err := s.repo.GetRideByID(ctx, rideID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("ride not found", nil)
		}
		return nil, err
	}

	// Verify access: must be the rider or driver
	if ride.RiderID != userID && (ride.DriverID == nil || *ride.DriverID != userID) {
		return nil, common.NewForbiddenError("you don't have access to this ride")
	}

	return ride, nil
}

// GetReceipt generates a receipt for a completed ride
func (s *Service) GetReceipt(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) (*Receipt, error) {
	ride, err := s.GetRideDetails(ctx, rideID, userID)
	if err != nil {
		return nil, err
	}

	if ride.Status != "completed" {
		return nil, common.NewBadRequestError("receipts are only available for completed rides", nil)
	}

	receipt := &Receipt{
		ReceiptID:     generateReceiptID(),
		RideID:        ride.ID,
		IssuedAt:      time.Now(),
		PickupAddress: ride.PickupAddress,
		DropoffAddress: ride.DropoffAddress,
		Distance:      ride.Distance,
		Duration:      ride.Duration,
		Currency:      ride.Currency,
		PaymentMethod: ride.PaymentMethod,
		DriverName:    ride.DriverName,
		LicensePlate:  ride.LicensePlate,
	}

	// Format timestamps
	receipt.TripDate = ride.RequestedAt.Format("January 2, 2006")
	receipt.TripStartTime = ride.RequestedAt.Format("3:04 PM")
	if ride.CompletedAt != nil {
		receipt.TripEndTime = ride.CompletedAt.Format("3:04 PM")
	}

	// Vehicle info
	if ride.VehicleMake != nil && ride.VehicleModel != nil {
		info := fmt.Sprintf("%s %s", *ride.VehicleMake, *ride.VehicleModel)
		if ride.VehicleColor != nil {
			info = *ride.VehicleColor + " " + info
		}
		receipt.VehicleInfo = &info
	}

	// Build fare breakdown
	var breakdown []FareLineItem

	if ride.BaseFare > 0 {
		breakdown = append(breakdown, FareLineItem{Label: "Base fare", Amount: ride.BaseFare, Type: "charge"})
	}
	if ride.DistanceFare > 0 {
		breakdown = append(breakdown, FareLineItem{
			Label:  fmt.Sprintf("Distance (%.1f km)", ride.Distance),
			Amount: ride.DistanceFare, Type: "charge",
		})
	}
	if ride.TimeFare > 0 {
		breakdown = append(breakdown, FareLineItem{
			Label:  fmt.Sprintf("Time (%d min)", ride.Duration),
			Amount: ride.TimeFare, Type: "charge",
		})
	}
	if ride.SurgeAmount > 0 {
		breakdown = append(breakdown, FareLineItem{
			Label:  fmt.Sprintf("Surge (%.1fx)", ride.SurgeMultiplier),
			Amount: ride.SurgeAmount, Type: "charge",
		})
	}
	if ride.WaitTimeCharge > 0 {
		breakdown = append(breakdown, FareLineItem{Label: "Wait time", Amount: ride.WaitTimeCharge, Type: "fee"})
	}
	if ride.TollFees > 0 {
		breakdown = append(breakdown, FareLineItem{Label: "Tolls", Amount: ride.TollFees, Type: "fee"})
	}
	if ride.DiscountAmount > 0 {
		label := "Discount"
		if ride.PromoCode != nil {
			label = fmt.Sprintf("Promo (%s)", *ride.PromoCode)
		}
		breakdown = append(breakdown, FareLineItem{Label: label, Amount: -ride.DiscountAmount, Type: "discount"})
	}
	if ride.TipAmount > 0 {
		breakdown = append(breakdown, FareLineItem{Label: "Tip", Amount: ride.TipAmount, Type: "tip"})
	}

	receipt.FareBreakdown = breakdown
	receipt.Subtotal = ride.BaseFare + ride.DistanceFare + ride.TimeFare + ride.SurgeAmount
	receipt.Fees = ride.TollFees + ride.WaitTimeCharge
	receipt.Discounts = ride.DiscountAmount
	receipt.Tip = ride.TipAmount
	receipt.Total = ride.TotalFare

	return receipt, nil
}

// GetRiderStats returns aggregated stats for a rider
func (s *Service) GetRiderStats(ctx context.Context, riderID uuid.UUID, period string) (*RideStats, error) {
	from, to := s.periodToTimeRange(period)

	stats, err := s.repo.GetRiderStats(ctx, riderID, from, to)
	if err != nil {
		return nil, err
	}
	stats.Period = period

	return stats, nil
}

// GetFrequentRoutes returns commonly taken routes
func (s *Service) GetFrequentRoutes(ctx context.Context, riderID uuid.UUID) ([]FrequentRoute, error) {
	routes, err := s.repo.GetFrequentRoutes(ctx, riderID, 10)
	if err != nil {
		return nil, err
	}
	if routes == nil {
		routes = []FrequentRoute{}
	}
	return routes, nil
}

func (s *Service) periodToTimeRange(period string) (time.Time, time.Time) {
	now := time.Now()
	to := now

	switch period {
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
	case "last_month":
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		from := firstOfMonth.AddDate(0, -1, 0)
		return from, firstOfMonth
	case "this_year":
		from := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return from, to
	default: // all_time
		from := time.Date(2020, 1, 1, 0, 0, 0, 0, now.Location())
		return from, to
	}
}

func generateReceiptID() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	id := make([]byte, 12)
	for i := range id {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		id[i] = chars[n.Int64()]
	}
	return fmt.Sprintf("RCP-%s-%s", string(id[:6]), string(id[6:]))
}
