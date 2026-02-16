package models

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents user role type
type UserRole string

const (
	RoleRider  UserRole = "rider"
	RoleDriver UserRole = "driver"
	RoleAdmin  UserRole = "admin"
)

// User represents a user in the system (both riders and drivers)
type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	PhoneNumber  string     `json:"phone_number" db:"phone_number"`
	PasswordHash string     `json:"-" db:"password_hash"`
	FirstName    string     `json:"first_name" db:"first_name"`
	LastName     string     `json:"last_name" db:"last_name"`
	Role         UserRole   `json:"role" db:"role"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	IsVerified   bool       `json:"is_verified" db:"is_verified"`
	ProfileImage *string    `json:"profile_image,omitempty" db:"profile_image"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// Driver represents driver-specific information
type Driver struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	UserID             uuid.UUID  `json:"user_id" db:"user_id"`
	LicenseNumber      string     `json:"license_number" db:"license_number"`
	VehicleModel       string     `json:"vehicle_model" db:"vehicle_model"`
	VehiclePlate       string     `json:"vehicle_plate" db:"vehicle_plate"`
	VehicleColor       string     `json:"vehicle_color" db:"vehicle_color"`
	VehicleYear        int        `json:"vehicle_year" db:"vehicle_year"`
	IsAvailable        bool       `json:"is_available" db:"is_available"`       // Available for accepting rides (online/offline)
	IsOnline           bool       `json:"is_online" db:"is_online"`             // Currently online
	ApprovalStatus     string     `json:"approval_status" db:"approval_status"` // pending, approved, rejected
	ApprovedBy         *uuid.UUID `json:"approved_by,omitempty" db:"approved_by"`
	ApprovedAt         *time.Time `json:"approved_at,omitempty" db:"approved_at"`
	RejectionReason    *string    `json:"rejection_reason,omitempty" db:"rejection_reason"`
	RejectedAt         *time.Time `json:"rejected_at,omitempty" db:"rejected_at"`
	Rating             float64    `json:"rating" db:"rating"`
	TotalRides         int        `json:"total_rides" db:"total_rides"`
	CurrentLatitude    *float64   `json:"current_latitude,omitempty" db:"current_latitude"`
	CurrentLongitude   *float64   `json:"current_longitude,omitempty" db:"current_longitude"`
	LastLocationUpdate *time.Time `json:"last_location_update,omitempty" db:"last_location_update"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// RegisterRequest represents registration request
type RegisterRequest struct {
	Email       string   `json:"email" binding:"required,email"`
	Password    string   `json:"password" binding:"required,min=8"`
	PhoneNumber string   `json:"phone_number" binding:"required"`
	FirstName   string   `json:"first_name" binding:"required"`
	LastName    string   `json:"last_name" binding:"required"`
	Role        UserRole `json:"role" binding:"required,oneof=rider driver"`
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response
type LoginResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}
