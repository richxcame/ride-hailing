package vehicle

import (
	"time"

	"github.com/google/uuid"
)

// VehicleStatus represents the registration status
type VehicleStatus string

const (
	VehicleStatusPending   VehicleStatus = "pending"
	VehicleStatusApproved  VehicleStatus = "approved"
	VehicleStatusRejected  VehicleStatus = "rejected"
	VehicleStatusSuspended VehicleStatus = "suspended"
	VehicleStatusRetired   VehicleStatus = "retired"
)

// VehicleCategory represents the vehicle class for ride matching
type VehicleCategory string

const (
	VehicleCategoryEconomy  VehicleCategory = "economy"
	VehicleCategoryComfort  VehicleCategory = "comfort"
	VehicleCategoryPremium  VehicleCategory = "premium"
	VehicleCategoryLux      VehicleCategory = "lux"
	VehicleCategoryXL       VehicleCategory = "xl"
	VehicleCategoryWAV      VehicleCategory = "wav" // Wheelchair accessible
	VehicleCategoryElectric VehicleCategory = "electric"
)

// FuelType represents the fuel type
type FuelType string

const (
	FuelTypeGasoline FuelType = "gasoline"
	FuelTypeDiesel   FuelType = "diesel"
	FuelTypeElectric FuelType = "electric"
	FuelTypeHybrid   FuelType = "hybrid"
	FuelTypeCNG      FuelType = "cng"
	FuelTypeLPG      FuelType = "lpg"
)

// Vehicle represents a driver's vehicle
type Vehicle struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	DriverID         uuid.UUID       `json:"driver_id" db:"driver_id"`
	Status           VehicleStatus   `json:"status" db:"status"`
	Category         VehicleCategory `json:"category" db:"category"`

	// Vehicle info
	Make             string     `json:"make" db:"make"`
	Model            string     `json:"model" db:"model"`
	Year             int        `json:"year" db:"year"`
	Color            string     `json:"color" db:"color"`
	LicensePlate     string     `json:"license_plate" db:"license_plate"`
	VIN              *string    `json:"vin,omitempty" db:"vin"`

	// Capabilities
	FuelType         FuelType   `json:"fuel_type" db:"fuel_type"`
	MaxPassengers    int        `json:"max_passengers" db:"max_passengers"`
	HasChildSeat     bool       `json:"has_child_seat" db:"has_child_seat"`
	HasWheelchairAccess bool   `json:"has_wheelchair_access" db:"has_wheelchair_access"`
	HasWifi          bool       `json:"has_wifi" db:"has_wifi"`
	HasCharger       bool       `json:"has_charger" db:"has_charger"`
	PetFriendly      bool       `json:"pet_friendly" db:"pet_friendly"`
	LuggageCapacity  int        `json:"luggage_capacity" db:"luggage_capacity"` // Number of bags

	// Documents
	RegistrationPhotoURL *string `json:"registration_photo_url,omitempty" db:"registration_photo_url"`
	InsurancePhotoURL    *string `json:"insurance_photo_url,omitempty" db:"insurance_photo_url"`
	FrontPhotoURL        *string `json:"front_photo_url,omitempty" db:"front_photo_url"`
	BackPhotoURL         *string `json:"back_photo_url,omitempty" db:"back_photo_url"`
	SidePhotoURL         *string `json:"side_photo_url,omitempty" db:"side_photo_url"`
	InteriorPhotoURL     *string `json:"interior_photo_url,omitempty" db:"interior_photo_url"`

	// Insurance & Registration
	InsuranceExpiry      *time.Time `json:"insurance_expiry,omitempty" db:"insurance_expiry"`
	RegistrationExpiry   *time.Time `json:"registration_expiry,omitempty" db:"registration_expiry"`
	InspectionExpiry     *time.Time `json:"inspection_expiry,omitempty" db:"inspection_expiry"`

	// Admin
	RejectionReason *string    `json:"rejection_reason,omitempty" db:"rejection_reason"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	IsPrimary       bool       `json:"is_primary" db:"is_primary"` // Driver's current vehicle

	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// VehicleInspection represents a periodic vehicle inspection
type VehicleInspection struct {
	ID           uuid.UUID `json:"id" db:"id"`
	VehicleID    uuid.UUID `json:"vehicle_id" db:"vehicle_id"`
	InspectorID  *uuid.UUID `json:"inspector_id,omitempty" db:"inspector_id"`
	Status       string    `json:"status" db:"status"` // scheduled, passed, failed, missed
	ScheduledAt  time.Time `json:"scheduled_at" db:"scheduled_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	Notes        *string   `json:"notes,omitempty" db:"notes"`
	FailReasons  []string  `json:"fail_reasons,omitempty" db:"fail_reasons"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// MaintenanceReminder represents a maintenance reminder for a vehicle
type MaintenanceReminder struct {
	ID           uuid.UUID `json:"id" db:"id"`
	VehicleID    uuid.UUID `json:"vehicle_id" db:"vehicle_id"`
	Type         string    `json:"type" db:"type"` // oil_change, tire_rotation, brake_check, etc.
	Description  string    `json:"description" db:"description"`
	DueDate      *time.Time `json:"due_date,omitempty" db:"due_date"`
	DueMileage   *int      `json:"due_mileage,omitempty" db:"due_mileage"`
	IsCompleted  bool      `json:"is_completed" db:"is_completed"`
	CompletedAt  *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// RegisterVehicleRequest registers a new vehicle
type RegisterVehicleRequest struct {
	Make             string          `json:"make" binding:"required"`
	Model            string          `json:"model" binding:"required"`
	Year             int             `json:"year" binding:"required"`
	Color            string          `json:"color" binding:"required"`
	LicensePlate     string          `json:"license_plate" binding:"required"`
	VIN              *string         `json:"vin,omitempty"`
	Category         VehicleCategory `json:"category" binding:"required"`
	FuelType         FuelType        `json:"fuel_type" binding:"required"`
	MaxPassengers    int             `json:"max_passengers"`
	HasChildSeat     bool            `json:"has_child_seat"`
	HasWheelchairAccess bool         `json:"has_wheelchair_access"`
	HasWifi          bool            `json:"has_wifi"`
	HasCharger       bool            `json:"has_charger"`
	PetFriendly      bool            `json:"pet_friendly"`
	LuggageCapacity  int             `json:"luggage_capacity"`
}

// UpdateVehicleRequest updates vehicle details
type UpdateVehicleRequest struct {
	Color            *string          `json:"color,omitempty"`
	Category         *VehicleCategory `json:"category,omitempty"`
	MaxPassengers    *int             `json:"max_passengers,omitempty"`
	HasChildSeat     *bool            `json:"has_child_seat,omitempty"`
	HasWheelchairAccess *bool         `json:"has_wheelchair_access,omitempty"`
	HasWifi          *bool            `json:"has_wifi,omitempty"`
	HasCharger       *bool            `json:"has_charger,omitempty"`
	PetFriendly      *bool            `json:"pet_friendly,omitempty"`
	LuggageCapacity  *int             `json:"luggage_capacity,omitempty"`
}

// UploadVehiclePhotosRequest uploads vehicle photos
type UploadVehiclePhotosRequest struct {
	RegistrationPhotoURL *string `json:"registration_photo_url,omitempty"`
	InsurancePhotoURL    *string `json:"insurance_photo_url,omitempty"`
	FrontPhotoURL        *string `json:"front_photo_url,omitempty"`
	BackPhotoURL         *string `json:"back_photo_url,omitempty"`
	SidePhotoURL         *string `json:"side_photo_url,omitempty"`
	InteriorPhotoURL     *string `json:"interior_photo_url,omitempty"`
}

// AdminReviewRequest represents admin vehicle review
type AdminReviewRequest struct {
	Approved        bool    `json:"approved"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

// VehicleListResponse returns a list of vehicles
type VehicleListResponse struct {
	Vehicles []Vehicle `json:"vehicles"`
	Count    int       `json:"count"`
}

// VehicleStats represents vehicle-related statistics
type VehicleStats struct {
	TotalVehicles     int `json:"total_vehicles"`
	PendingReview     int `json:"pending_review"`
	ApprovedVehicles  int `json:"approved_vehicles"`
	SuspendedVehicles int `json:"suspended_vehicles"`
	ExpiringInsurance int `json:"expiring_insurance"` // Within 30 days
	ExpiringRegistration int `json:"expiring_registration"`
}
