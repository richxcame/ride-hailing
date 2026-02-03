package preferences

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles preferences data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new preferences repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// RIDER PREFERENCES
// ========================================

// GetRiderPreferences retrieves a rider's default preferences
func (r *Repository) GetRiderPreferences(ctx context.Context, userID uuid.UUID) (*RiderPreferences, error) {
	p := &RiderPreferences{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, temperature, music, conversation, route,
			child_seat, pet_friendly, wheelchair_access, prefer_female_driver,
			luggage_assistance, max_passengers, preferred_language, special_needs,
			created_at, updated_at
		FROM rider_preferences
		WHERE user_id = $1`, userID,
	).Scan(
		&p.ID, &p.UserID, &p.Temperature, &p.Music, &p.Conversation, &p.Route,
		&p.ChildSeat, &p.PetFriendly, &p.WheelchairAccess, &p.PreferFemaleDriver,
		&p.LuggageAssistance, &p.MaxPassengers, &p.PreferredLanguage, &p.SpecialNeeds,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// UpsertRiderPreferences creates or updates rider preferences
func (r *Repository) UpsertRiderPreferences(ctx context.Context, p *RiderPreferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO rider_preferences (
			id, user_id, temperature, music, conversation, route,
			child_seat, pet_friendly, wheelchair_access, prefer_female_driver,
			luggage_assistance, max_passengers, preferred_language, special_needs,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (user_id) DO UPDATE SET
			temperature = EXCLUDED.temperature,
			music = EXCLUDED.music,
			conversation = EXCLUDED.conversation,
			route = EXCLUDED.route,
			child_seat = EXCLUDED.child_seat,
			pet_friendly = EXCLUDED.pet_friendly,
			wheelchair_access = EXCLUDED.wheelchair_access,
			prefer_female_driver = EXCLUDED.prefer_female_driver,
			luggage_assistance = EXCLUDED.luggage_assistance,
			max_passengers = EXCLUDED.max_passengers,
			preferred_language = EXCLUDED.preferred_language,
			special_needs = EXCLUDED.special_needs,
			updated_at = EXCLUDED.updated_at`,
		p.ID, p.UserID, p.Temperature, p.Music, p.Conversation, p.Route,
		p.ChildSeat, p.PetFriendly, p.WheelchairAccess, p.PreferFemaleDriver,
		p.LuggageAssistance, p.MaxPassengers, p.PreferredLanguage, p.SpecialNeeds,
		p.CreatedAt, p.UpdatedAt,
	)
	return err
}

// ========================================
// RIDE OVERRIDES
// ========================================

// SetRideOverride saves per-ride preference overrides
func (r *Repository) SetRideOverride(ctx context.Context, o *RidePreferenceOverride) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ride_preference_overrides (
			id, ride_id, user_id, temperature, music, conversation, route,
			child_seat, pet_friendly, notes, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (ride_id) DO UPDATE SET
			temperature = EXCLUDED.temperature,
			music = EXCLUDED.music,
			conversation = EXCLUDED.conversation,
			route = EXCLUDED.route,
			child_seat = EXCLUDED.child_seat,
			pet_friendly = EXCLUDED.pet_friendly,
			notes = EXCLUDED.notes`,
		o.ID, o.RideID, o.UserID, o.Temperature, o.Music, o.Conversation, o.Route,
		o.ChildSeat, o.PetFriendly, o.Notes, o.CreatedAt,
	)
	return err
}

// GetRideOverride retrieves per-ride preference overrides
func (r *Repository) GetRideOverride(ctx context.Context, rideID uuid.UUID) (*RidePreferenceOverride, error) {
	o := &RidePreferenceOverride{}
	err := r.db.QueryRow(ctx, `
		SELECT id, ride_id, user_id, temperature, music, conversation, route,
			child_seat, pet_friendly, notes, created_at
		FROM ride_preference_overrides
		WHERE ride_id = $1`, rideID,
	).Scan(
		&o.ID, &o.RideID, &o.UserID, &o.Temperature, &o.Music, &o.Conversation, &o.Route,
		&o.ChildSeat, &o.PetFriendly, &o.Notes, &o.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return o, nil
}

// ========================================
// DRIVER CAPABILITIES
// ========================================

// GetDriverCapabilities retrieves a driver's vehicle capabilities
func (r *Repository) GetDriverCapabilities(ctx context.Context, driverID uuid.UUID) (*DriverCapabilities, error) {
	dc := &DriverCapabilities{}
	err := r.db.QueryRow(ctx, `
		SELECT id, driver_id, has_child_seat, pet_friendly, wheelchair_access,
			luggage_capacity, max_passengers, languages, created_at, updated_at
		FROM driver_capabilities
		WHERE driver_id = $1`, driverID,
	).Scan(
		&dc.ID, &dc.DriverID, &dc.HasChildSeat, &dc.PetFriendly, &dc.WheelchairAccess,
		&dc.LuggageCapacity, &dc.MaxPassengers, &dc.LanguagesJSON, &dc.CreatedAt, &dc.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return dc, nil
}

// UpsertDriverCapabilities creates or updates driver capabilities
func (r *Repository) UpsertDriverCapabilities(ctx context.Context, dc *DriverCapabilities) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO driver_capabilities (
			id, driver_id, has_child_seat, pet_friendly, wheelchair_access,
			luggage_capacity, max_passengers, languages, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (driver_id) DO UPDATE SET
			has_child_seat = EXCLUDED.has_child_seat,
			pet_friendly = EXCLUDED.pet_friendly,
			wheelchair_access = EXCLUDED.wheelchair_access,
			luggage_capacity = EXCLUDED.luggage_capacity,
			max_passengers = EXCLUDED.max_passengers,
			languages = EXCLUDED.languages,
			updated_at = EXCLUDED.updated_at`,
		dc.ID, dc.DriverID, dc.HasChildSeat, dc.PetFriendly, dc.WheelchairAccess,
		dc.LuggageCapacity, dc.MaxPassengers, dc.LanguagesJSON, dc.CreatedAt, dc.UpdatedAt,
	)
	return err
}
