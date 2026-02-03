package preferences

import (
	"time"

	"github.com/google/uuid"
)

// TemperaturePreference represents temperature setting
type TemperaturePreference string

const (
	TempCool   TemperaturePreference = "cool"
	TempNormal TemperaturePreference = "normal"
	TempWarm   TemperaturePreference = "warm"
	TempNoPreference TemperaturePreference = "no_preference"
)

// MusicPreference represents music setting
type MusicPreference string

const (
	MusicQuiet        MusicPreference = "quiet"         // No music
	MusicLow          MusicPreference = "low"            // Low volume background
	MusicDriverChoice MusicPreference = "driver_choice"  // Driver picks
	MusicNoPreference MusicPreference = "no_preference"
)

// ConversationPreference represents talk preference
type ConversationPreference string

const (
	ConversationQuiet        ConversationPreference = "quiet"         // Minimal conversation
	ConversationFriendly     ConversationPreference = "friendly"      // Open to chat
	ConversationNoPreference ConversationPreference = "no_preference"
)

// RoutePreference represents route selection preference
type RoutePreference string

const (
	RouteFastest     RoutePreference = "fastest"      // Minimize time (may use highways/tolls)
	RouteCheapest    RoutePreference = "cheapest"      // Avoid tolls, minimize distance
	RouteScenic      RoutePreference = "scenic"        // Prefer scenic routes
	RouteNoPreference RoutePreference = "no_preference"
)

// RiderPreferences stores a rider's default preferences for rides
type RiderPreferences struct {
	ID                  uuid.UUID              `json:"id" db:"id"`
	UserID              uuid.UUID              `json:"user_id" db:"user_id"`
	Temperature         TemperaturePreference  `json:"temperature" db:"temperature"`
	Music               MusicPreference        `json:"music" db:"music"`
	Conversation        ConversationPreference `json:"conversation" db:"conversation"`
	Route               RoutePreference        `json:"route" db:"route"`
	ChildSeat           bool                   `json:"child_seat" db:"child_seat"`
	PetFriendly         bool                   `json:"pet_friendly" db:"pet_friendly"`
	WheelchairAccess    bool                   `json:"wheelchair_access" db:"wheelchair_access"`
	PreferFemaleDriver  bool                   `json:"prefer_female_driver" db:"prefer_female_driver"`
	LuggageAssistance   bool                   `json:"luggage_assistance" db:"luggage_assistance"`
	MaxPassengers       *int                   `json:"max_passengers,omitempty" db:"max_passengers"`
	PreferredLanguage   *string                `json:"preferred_language,omitempty" db:"preferred_language"`
	SpecialNeeds        *string                `json:"special_needs,omitempty" db:"special_needs"`
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at" db:"updated_at"`
}

// RidePreferenceOverride allows per-ride preference overrides
type RidePreferenceOverride struct {
	ID           uuid.UUID               `json:"id" db:"id"`
	RideID       uuid.UUID               `json:"ride_id" db:"ride_id"`
	UserID       uuid.UUID               `json:"user_id" db:"user_id"`
	Temperature  *TemperaturePreference  `json:"temperature,omitempty" db:"temperature"`
	Music        *MusicPreference        `json:"music,omitempty" db:"music"`
	Conversation *ConversationPreference `json:"conversation,omitempty" db:"conversation"`
	Route        *RoutePreference        `json:"route,omitempty" db:"route"`
	ChildSeat    *bool                   `json:"child_seat,omitempty" db:"child_seat"`
	PetFriendly  *bool                   `json:"pet_friendly,omitempty" db:"pet_friendly"`
	Notes        *string                 `json:"notes,omitempty" db:"notes"` // Free-text notes to driver
	CreatedAt    time.Time               `json:"created_at" db:"created_at"`
}

// DriverCapabilities tracks what a driver's vehicle supports
type DriverCapabilities struct {
	ID               uuid.UUID `json:"id" db:"id"`
	DriverID         uuid.UUID `json:"driver_id" db:"driver_id"`
	HasChildSeat     bool      `json:"has_child_seat" db:"has_child_seat"`
	PetFriendly      bool      `json:"pet_friendly" db:"pet_friendly"`
	WheelchairAccess bool      `json:"wheelchair_access" db:"wheelchair_access"`
	LuggageCapacity  int       `json:"luggage_capacity" db:"luggage_capacity"` // Number of large bags
	MaxPassengers    int       `json:"max_passengers" db:"max_passengers"`
	Languages        []string  `json:"languages" db:"-"` // Loaded separately or JSON
	LanguagesJSON    *string   `json:"-" db:"languages"` // Stored as JSON string
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// UpdatePreferencesRequest updates rider's default preferences
type UpdatePreferencesRequest struct {
	Temperature        *TemperaturePreference  `json:"temperature,omitempty"`
	Music              *MusicPreference        `json:"music,omitempty"`
	Conversation       *ConversationPreference `json:"conversation,omitempty"`
	Route              *RoutePreference        `json:"route,omitempty"`
	ChildSeat          *bool                   `json:"child_seat,omitempty"`
	PetFriendly        *bool                   `json:"pet_friendly,omitempty"`
	WheelchairAccess   *bool                   `json:"wheelchair_access,omitempty"`
	PreferFemaleDriver *bool                   `json:"prefer_female_driver,omitempty"`
	LuggageAssistance  *bool                   `json:"luggage_assistance,omitempty"`
	MaxPassengers      *int                    `json:"max_passengers,omitempty"`
	PreferredLanguage  *string                 `json:"preferred_language,omitempty"`
	SpecialNeeds       *string                 `json:"special_needs,omitempty"`
}

// SetRidePreferencesRequest overrides preferences for a specific ride
type SetRidePreferencesRequest struct {
	RideID       uuid.UUID               `json:"ride_id" binding:"required"`
	Temperature  *TemperaturePreference  `json:"temperature,omitempty"`
	Music        *MusicPreference        `json:"music,omitempty"`
	Conversation *ConversationPreference `json:"conversation,omitempty"`
	Route        *RoutePreference        `json:"route,omitempty"`
	ChildSeat    *bool                   `json:"child_seat,omitempty"`
	PetFriendly  *bool                   `json:"pet_friendly,omitempty"`
	Notes        *string                 `json:"notes,omitempty"`
}

// UpdateDriverCapabilitiesRequest updates driver's vehicle capabilities
type UpdateDriverCapabilitiesRequest struct {
	HasChildSeat     *bool    `json:"has_child_seat,omitempty"`
	PetFriendly      *bool    `json:"pet_friendly,omitempty"`
	WheelchairAccess *bool    `json:"wheelchair_access,omitempty"`
	LuggageCapacity  *int     `json:"luggage_capacity,omitempty"`
	MaxPassengers    *int     `json:"max_passengers,omitempty"`
	Languages        []string `json:"languages,omitempty"`
}

// PreferenceSummary combines rider preferences with matched driver capabilities
type PreferenceSummary struct {
	Preferences  *RiderPreferences       `json:"preferences"`
	Override     *RidePreferenceOverride  `json:"override,omitempty"`
	Capabilities *DriverCapabilities     `json:"driver_capabilities,omitempty"`
	Matches      []PreferenceMatch       `json:"matches,omitempty"`
}

// PreferenceMatch shows whether a preference matches driver capability
type PreferenceMatch struct {
	Preference string `json:"preference"`
	Requested  bool   `json:"requested"`
	Available  bool   `json:"available"`
	Match      bool   `json:"match"`
}
