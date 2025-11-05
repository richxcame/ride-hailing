package analytics

import (
	"time"

	"github.com/google/uuid"
)

// RevenueMetrics represents revenue statistics
type RevenueMetrics struct {
	Period          string  `json:"period"`
	TotalRevenue    float64 `json:"total_revenue"`
	TotalRides      int     `json:"total_rides"`
	AvgFarePerRide  float64 `json:"avg_fare_per_ride"`
	TotalDiscounts  float64 `json:"total_discounts"`
	PlatformEarnings float64 `json:"platform_earnings"`
	DriverEarnings  float64 `json:"driver_earnings"`
}

// PromoCodePerformance represents promo code usage statistics
type PromoCodePerformance struct {
	PromoCodeID     uuid.UUID `json:"promo_code_id"`
	Code            string    `json:"code"`
	TotalUses       int       `json:"total_uses"`
	TotalDiscount   float64   `json:"total_discount"`
	AvgDiscount     float64   `json:"avg_discount"`
	UniqueUsers     int       `json:"unique_users"`
	TotalRideValue  float64   `json:"total_ride_value"`
}

// RideTypeStats represents ride type usage statistics
type RideTypeStats struct {
	RideTypeID     uuid.UUID `json:"ride_type_id"`
	Name           string    `json:"name"`
	TotalRides     int       `json:"total_rides"`
	TotalRevenue   float64   `json:"total_revenue"`
	AvgFare        float64   `json:"avg_fare"`
	Percentage     float64   `json:"percentage"`
}

// ReferralMetrics represents referral program statistics
type ReferralMetrics struct {
	TotalReferrals       int     `json:"total_referrals"`
	CompletedReferrals   int     `json:"completed_referrals"`
	TotalBonusesPaid     float64 `json:"total_bonuses_paid"`
	AvgBonusPerReferral  float64 `json:"avg_bonus_per_referral"`
	ConversionRate       float64 `json:"conversion_rate"`
}

// DriverPerformance represents driver performance metrics
type DriverPerformance struct {
	DriverID          uuid.UUID `json:"driver_id"`
	DriverName        string    `json:"driver_name"`
	TotalRides        int       `json:"total_rides"`
	TotalEarnings     float64   `json:"total_earnings"`
	AvgRating         float64   `json:"avg_rating"`
	CompletionRate    float64   `json:"completion_rate"`
	CancellationRate  float64   `json:"cancellation_rate"`
}

// DateRange represents a time period for analytics
type DateRange struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// DashboardMetrics represents overall platform metrics
type DashboardMetrics struct {
	TotalRides       int               `json:"total_rides"`
	ActiveRides      int               `json:"active_rides"`
	CompletedToday   int               `json:"completed_today"`
	RevenueToday     float64           `json:"revenue_today"`
	ActiveDrivers    int               `json:"active_drivers"`
	ActiveRiders     int               `json:"active_riders"`
	AvgRating        float64           `json:"avg_rating"`
	TopPromoCode     *PromoCodePerformance `json:"top_promo_code,omitempty"`
	TopRideType      *RideTypeStats    `json:"top_ride_type,omitempty"`
}

// DemandHeatMap represents geographic demand data
type DemandHeatMap struct {
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	RideCount     int     `json:"ride_count"`
	AvgWaitTime   int     `json:"avg_wait_time_minutes"`
	AvgFare       float64 `json:"avg_fare"`
	DemandLevel   string  `json:"demand_level"` // low, medium, high, very_high
	SurgeActive   bool    `json:"surge_active"`
}

// FinancialReport represents financial summary for a period
type FinancialReport struct {
	Period                 string  `json:"period"`
	GrossRevenue           float64 `json:"gross_revenue"`
	NetRevenue             float64 `json:"net_revenue"`
	PlatformCommission     float64 `json:"platform_commission"`
	DriverPayouts          float64 `json:"driver_payouts"`
	PromoDiscounts         float64 `json:"promo_discounts"`
	ReferralBonuses        float64 `json:"referral_bonuses"`
	Refunds                float64 `json:"refunds"`
	TotalExpenses          float64 `json:"total_expenses"`
	Profit                 float64 `json:"profit"`
	ProfitMargin           float64 `json:"profit_margin_percent"`
	TotalRides             int     `json:"total_rides"`
	CompletedRides         int     `json:"completed_rides"`
	CancelledRides         int     `json:"cancelled_rides"`
	AvgRevenuePerRide      float64 `json:"avg_revenue_per_ride"`
	TopRevenueDay          string  `json:"top_revenue_day,omitempty"`
	TopRevenueDayAmount    float64 `json:"top_revenue_day_amount"`
}

// DemandZone represents a high-demand geographic zone
type DemandZone struct {
	ZoneName      string  `json:"zone_name"`
	CenterLat     float64 `json:"center_latitude"`
	CenterLon     float64 `json:"center_longitude"`
	RadiusKm      float64 `json:"radius_km"`
	TotalRides    int     `json:"total_rides"`
	AvgSurge      float64 `json:"avg_surge_multiplier"`
	PeakHours     string  `json:"peak_hours"`
}
