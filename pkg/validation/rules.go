package validation

import "time"

// Common validation rules and request structs

// CreateRideRequest represents a ride creation request with validation rules
type CreateRideRequest struct {
	PickupLatitude   float64    `json:"pickup_latitude" validate:"required,latitude"`
	PickupLongitude  float64    `json:"pickup_longitude" validate:"required,longitude"`
	PickupAddr       string     `json:"pickup_address" validate:"required,min=5,max=500"`
	DropoffLatitude  float64    `json:"dropoff_latitude" validate:"required,latitude"`
	DropoffLongitude float64    `json:"dropoff_longitude" validate:"required,longitude"`
	DropoffAddr      string     `json:"dropoff_address" validate:"required,min=5,max=500"`
	RideType         string     `json:"ride_type" validate:"required,oneof=economy premium xl"`
	PromoCode        string     `json:"promo_code" validate:"omitempty,alphanum,max=20"`
	ScheduledFor     *time.Time `json:"scheduled_for" validate:"omitempty"`
}

// RegisterUserRequest represents a user registration request
type RegisterUserRequest struct {
	Email       string `json:"email" validate:"required,email,max=255"`
	PhoneNumber string `json:"phone_number" validate:"required,phone,max=20"`
	Password    string `json:"password" validate:"required,min=8,max=72"`
	FirstName   string `json:"first_name" validate:"required,min=1,max=100"`
	LastName    string `json:"last_name" validate:"required,min=1,max=100"`
	Role        string `json:"role" validate:"required,user_role"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// UpdateLocationRequest represents a driver location update
type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude" validate:"required,latitude"`
	Longitude float64 `json:"longitude" validate:"required,longitude"`
}

// CreatePaymentRequest represents a payment creation request
type CreatePaymentRequest struct {
	RideID  string  `json:"ride_id" validate:"required,uuid"`
	Amount  float64 `json:"amount" validate:"required,gt=0,lte=100000"`
	Method  string  `json:"method" validate:"required,payment_method"`
	TokenID string  `json:"token_id" validate:"omitempty,min=1,max=255"`
}

// UpdateRideStatusRequest represents a ride status update
type UpdateRideStatusRequest struct {
	Status string `json:"status" validate:"required,ride_status"`
	Reason string `json:"reason" validate:"omitempty,max=500"`
}

// RatingRequest represents a ride rating request
type RatingRequest struct {
	RideID   string `json:"ride_id" validate:"required,uuid"`
	Rating   int    `json:"rating" validate:"required,gte=1,lte=5"`
	Feedback string `json:"feedback" validate:"omitempty,max=1000"`
}

// CreateDriverRequest represents a driver registration request
type CreateDriverRequest struct {
	UserID        string `json:"user_id" validate:"required,uuid"`
	LicenseNumber string `json:"license_number" validate:"required,alphanum,min=5,max=50"`
	VehicleModel  string `json:"vehicle_model" validate:"required,min=2,max=100"`
	VehiclePlate  string `json:"vehicle_plate" validate:"required,alphanum,min=2,max=20"`
	VehicleColor  string `json:"vehicle_color" validate:"required,min=2,max=50"`
	VehicleYear   int    `json:"vehicle_year" validate:"required,vehicle_year"`
}

// CreatePromoCodeRequest represents a promo code creation request
type CreatePromoCodeRequest struct {
	Code             string    `json:"code" validate:"required,alphanum,min=3,max=20"`
	DiscountType     string    `json:"discount_type" validate:"required,oneof=percentage fixed"`
	DiscountValue    float64   `json:"discount_value" validate:"required,gt=0"`
	MaxUsage         int       `json:"max_usage" validate:"required,gte=1"`
	MaxDiscountValue float64   `json:"max_discount_value" validate:"omitempty,gt=0"`
	MinRideAmount    float64   `json:"min_ride_amount" validate:"omitempty,gte=0"`
	ValidFrom        time.Time `json:"valid_from" validate:"required"`
	ValidUntil       time.Time `json:"valid_until" validate:"required"`
}

// WalletTopUpRequest represents a wallet top-up request
type WalletTopUpRequest struct {
	Amount  float64 `json:"amount" validate:"required,gt=0,lte=10000"`
	TokenID string  `json:"token_id" validate:"required,min=1,max=255"`
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	FirstName    string `json:"first_name" validate:"omitempty,min=1,max=100"`
	LastName     string `json:"last_name" validate:"omitempty,min=1,max=100"`
	PhoneNumber  string `json:"phone_number" validate:"omitempty,phone"`
	ProfileImage string `json:"profile_image" validate:"omitempty,url"`
}

// PaginationRequest represents common pagination parameters
type PaginationRequest struct {
	Limit   int    `json:"limit" validate:"omitempty,gte=1,lte=100"`
	Offset  int    `json:"offset" validate:"omitempty,gte=0"`
	SortBy  string `json:"sort_by" validate:"omitempty,alpha"`
	SortDir string `json:"sort_dir" validate:"omitempty,oneof=asc desc"`
}

// DateRangeRequest represents a date range filter
type DateRangeRequest struct {
	StartDate time.Time `json:"start_date" validate:"omitempty"`
	EndDate   time.Time `json:"end_date" validate:"omitempty"`
}

// ValidateRideRequest validates a ride request and checks business rules
func ValidateRideRequest(req *CreateRideRequest) error {
	// First, validate struct tags
	if err := ValidateStruct(req); err != nil {
		return err
	}

	// Additional business logic validation
	validationErr := &ValidationError{Errors: make(map[string]string)}

	// Check if pickup and dropoff are not the same
	if req.PickupLatitude == req.DropoffLatitude && req.PickupLongitude == req.DropoffLongitude {
		validationErr.AddError("location", "Pickup and dropoff locations cannot be the same")
	}

	// Check if scheduled time is in the future
	if req.ScheduledFor != nil && req.ScheduledFor.Before(time.Now()) {
		validationErr.AddError("scheduled_for", "Scheduled time must be in the future")
	}

	if validationErr.HasErrors() {
		return validationErr
	}

	return nil
}

// ValidatePasswordStrength validates password complexity
func ValidatePasswordStrength(password string) error {
	validationErr := &ValidationError{Errors: make(map[string]string)}

	if len(password) < 8 {
		validationErr.AddError("password", "Password must be at least 8 characters long")
	}

	// Could add more complexity requirements here:
	// - At least one uppercase letter
	// - At least one lowercase letter
	// - At least one digit
	// - At least one special character

	if validationErr.HasErrors() {
		return validationErr
	}

	return nil
}

// ValidateDateRange validates that end date is after start date
func ValidateDateRange(start, end time.Time) error {
	if end.Before(start) {
		return &ValidationError{
			Errors: map[string]string{
				"date_range": "End date must be after start date",
			},
		}
	}
	return nil
}
