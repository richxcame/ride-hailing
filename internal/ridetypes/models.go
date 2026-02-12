package ridetypes

import (
	"time"

	"github.com/google/uuid"
)

// RideType represents a global ride type product (Economy, Premium, XL, etc.)
// Pricing is managed through the hierarchical pricing engine (pricing_configs),
// NOT on the ride type itself.
type RideType struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	Icon        *string   `json:"icon,omitempty" db:"icon"`
	Capacity    int       `json:"capacity" db:"capacity"`
	SortOrder   int       `json:"sort_order" db:"sort_order"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CountryRideType maps a ride type to a country (country-level availability).
// If no city_ride_types rows exist for a country's cities, the ride type is
// available in ALL cities of that country.
type CountryRideType struct {
	CountryID  uuid.UUID `json:"country_id" db:"country_id"`
	RideTypeID uuid.UUID `json:"ride_type_id" db:"ride_type_id"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	SortOrder  int       `json:"sort_order" db:"sort_order"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// CountryRideTypeWithDetails includes the ride type details for a country mapping
type CountryRideTypeWithDetails struct {
	CountryRideType
	RideTypeName        string  `json:"ride_type_name" db:"ride_type_name"`
	RideTypeDescription *string `json:"ride_type_description,omitempty" db:"ride_type_description"`
	RideTypeIcon        *string `json:"ride_type_icon,omitempty" db:"ride_type_icon"`
	RideTypeCapacity    int     `json:"ride_type_capacity" db:"ride_type_capacity"`
}

// CityRideType maps a ride type to a city (overrides country-level availability)
type CityRideType struct {
	CityID     uuid.UUID `json:"city_id" db:"city_id"`
	RideTypeID uuid.UUID `json:"ride_type_id" db:"ride_type_id"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	SortOrder  int       `json:"sort_order" db:"sort_order"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// CityRideTypeWithDetails includes the ride type details for a city mapping
type CityRideTypeWithDetails struct {
	CityRideType
	RideTypeName        string  `json:"ride_type_name" db:"ride_type_name"`
	RideTypeDescription *string `json:"ride_type_description,omitempty" db:"ride_type_description"`
	RideTypeIcon        *string `json:"ride_type_icon,omitempty" db:"ride_type_icon"`
	RideTypeCapacity    int     `json:"ride_type_capacity" db:"ride_type_capacity"`
}

// CreateRideTypeRequest is the request body for creating a ride type
type CreateRideTypeRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	Capacity    int     `json:"capacity" binding:"required,gt=0"`
	SortOrder   int     `json:"sort_order"`
	IsActive    bool    `json:"is_active"`
}

// UpdateRideTypeRequest is the request body for updating a ride type
type UpdateRideTypeRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	Capacity    *int    `json:"capacity,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// CountryRideTypeRequest is the request body for adding a ride type to a country
type CountryRideTypeRequest struct {
	RideTypeID uuid.UUID `json:"ride_type_id" binding:"required"`
	IsActive   bool      `json:"is_active"`
	SortOrder  int       `json:"sort_order"`
}

// UpdateCountryRideTypeRequest is the request body for updating a country ride type mapping
type UpdateCountryRideTypeRequest struct {
	IsActive  *bool `json:"is_active,omitempty"`
	SortOrder *int  `json:"sort_order,omitempty"`
}

// CityRideTypeRequest is the request body for adding/updating a ride type in a city
type CityRideTypeRequest struct {
	RideTypeID uuid.UUID `json:"ride_type_id" binding:"required"`
	IsActive   bool      `json:"is_active"`
	SortOrder  int       `json:"sort_order"`
}

// UpdateCityRideTypeRequest is the request body for updating a city ride type mapping
type UpdateCityRideTypeRequest struct {
	IsActive  *bool `json:"is_active,omitempty"`
	SortOrder *int  `json:"sort_order,omitempty"`
}
