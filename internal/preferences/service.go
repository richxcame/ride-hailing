package preferences

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Service handles ride preferences business logic
type Service struct {
	repo *Repository
}

// NewService creates a new preferences service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetPreferences retrieves a rider's default preferences (creates defaults if none exist)
func (s *Service) GetPreferences(ctx context.Context, userID uuid.UUID) (*RiderPreferences, error) {
	prefs, err := s.repo.GetRiderPreferences(ctx, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Return defaults
			return s.createDefaults(ctx, userID)
		}
		return nil, err
	}
	return prefs, nil
}

// UpdatePreferences updates a rider's default preferences
func (s *Service) UpdatePreferences(ctx context.Context, userID uuid.UUID, req *UpdatePreferencesRequest) (*RiderPreferences, error) {
	prefs, err := s.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Temperature != nil {
		prefs.Temperature = *req.Temperature
	}
	if req.Music != nil {
		prefs.Music = *req.Music
	}
	if req.Conversation != nil {
		prefs.Conversation = *req.Conversation
	}
	if req.Route != nil {
		prefs.Route = *req.Route
	}
	if req.ChildSeat != nil {
		prefs.ChildSeat = *req.ChildSeat
	}
	if req.PetFriendly != nil {
		prefs.PetFriendly = *req.PetFriendly
	}
	if req.WheelchairAccess != nil {
		prefs.WheelchairAccess = *req.WheelchairAccess
	}
	if req.PreferFemaleDriver != nil {
		prefs.PreferFemaleDriver = *req.PreferFemaleDriver
	}
	if req.LuggageAssistance != nil {
		prefs.LuggageAssistance = *req.LuggageAssistance
	}
	if req.MaxPassengers != nil {
		prefs.MaxPassengers = req.MaxPassengers
	}
	if req.PreferredLanguage != nil {
		prefs.PreferredLanguage = req.PreferredLanguage
	}
	if req.SpecialNeeds != nil {
		prefs.SpecialNeeds = req.SpecialNeeds
	}

	prefs.UpdatedAt = time.Now()

	if err := s.repo.UpsertRiderPreferences(ctx, prefs); err != nil {
		return nil, err
	}
	return prefs, nil
}

// SetRidePreferences sets per-ride preference overrides
func (s *Service) SetRidePreferences(ctx context.Context, userID uuid.UUID, req *SetRidePreferencesRequest) (*RidePreferenceOverride, error) {
	override := &RidePreferenceOverride{
		ID:           uuid.New(),
		RideID:       req.RideID,
		UserID:       userID,
		Temperature:  req.Temperature,
		Music:        req.Music,
		Conversation: req.Conversation,
		Route:        req.Route,
		ChildSeat:    req.ChildSeat,
		PetFriendly:  req.PetFriendly,
		Notes:        req.Notes,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.SetRideOverride(ctx, override); err != nil {
		return nil, err
	}
	return override, nil
}

// GetRidePreferences retrieves the effective preferences for a ride
// (ride override merged with rider defaults)
func (s *Service) GetRidePreferences(ctx context.Context, rideID, userID uuid.UUID) (*PreferenceSummary, error) {
	prefs, _ := s.repo.GetRiderPreferences(ctx, userID)
	override, _ := s.repo.GetRideOverride(ctx, rideID)

	return &PreferenceSummary{
		Preferences: prefs,
		Override:    override,
	}, nil
}

// GetPreferencesForDriver retrieves the effective preferences for a ride
// from the driver's perspective, including preference-to-capability matching
func (s *Service) GetPreferencesForDriver(ctx context.Context, rideID, driverID, riderID uuid.UUID) (*PreferenceSummary, error) {
	prefs, _ := s.repo.GetRiderPreferences(ctx, riderID)
	override, _ := s.repo.GetRideOverride(ctx, rideID)
	caps, _ := s.repo.GetDriverCapabilities(ctx, driverID)

	summary := &PreferenceSummary{
		Preferences:  prefs,
		Override:     override,
		Capabilities: caps,
	}

	// Build match report
	if prefs != nil && caps != nil {
		summary.Matches = buildMatchReport(prefs, override, caps)
	}

	return summary, nil
}

// UpdateDriverCapabilities updates a driver's vehicle capabilities
func (s *Service) UpdateDriverCapabilities(ctx context.Context, driverID uuid.UUID, req *UpdateDriverCapabilitiesRequest) (*DriverCapabilities, error) {
	caps, err := s.repo.GetDriverCapabilities(ctx, driverID)
	if err != nil && caps == nil {
		// Create new
		caps = &DriverCapabilities{
			ID:        uuid.New(),
			DriverID:  driverID,
			CreatedAt: time.Now(),
		}
	}

	if req.HasChildSeat != nil {
		caps.HasChildSeat = *req.HasChildSeat
	}
	if req.PetFriendly != nil {
		caps.PetFriendly = *req.PetFriendly
	}
	if req.WheelchairAccess != nil {
		caps.WheelchairAccess = *req.WheelchairAccess
	}
	if req.LuggageCapacity != nil {
		caps.LuggageCapacity = *req.LuggageCapacity
	}
	if req.MaxPassengers != nil {
		caps.MaxPassengers = *req.MaxPassengers
	}
	if req.Languages != nil {
		langJSON, _ := json.Marshal(req.Languages)
		langStr := string(langJSON)
		caps.LanguagesJSON = &langStr
		caps.Languages = req.Languages
	}

	caps.UpdatedAt = time.Now()

	if err := s.repo.UpsertDriverCapabilities(ctx, caps); err != nil {
		return nil, err
	}
	return caps, nil
}

// GetDriverCapabilities retrieves a driver's capabilities
func (s *Service) GetDriverCapabilities(ctx context.Context, driverID uuid.UUID) (*DriverCapabilities, error) {
	caps, err := s.repo.GetDriverCapabilities(ctx, driverID)
	if err != nil {
		return nil, err
	}
	if caps == nil {
		return &DriverCapabilities{DriverID: driverID}, nil
	}

	// Unmarshal languages JSON
	if caps.LanguagesJSON != nil {
		json.Unmarshal([]byte(*caps.LanguagesJSON), &caps.Languages)
	}

	return caps, nil
}

// ========================================
// HELPERS
// ========================================

func (s *Service) createDefaults(ctx context.Context, userID uuid.UUID) (*RiderPreferences, error) {
	now := time.Now()
	prefs := &RiderPreferences{
		ID:           uuid.New(),
		UserID:       userID,
		Temperature:  TempNoPreference,
		Music:        MusicNoPreference,
		Conversation: ConversationNoPreference,
		Route:        RouteNoPreference,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.UpsertRiderPreferences(ctx, prefs); err != nil {
		return nil, err
	}
	return prefs, nil
}

// buildMatchReport compares rider preferences to driver capabilities
func buildMatchReport(prefs *RiderPreferences, override *RidePreferenceOverride, caps *DriverCapabilities) []PreferenceMatch {
	var matches []PreferenceMatch

	childSeatWanted := prefs.ChildSeat
	if override != nil && override.ChildSeat != nil {
		childSeatWanted = *override.ChildSeat
	}
	if childSeatWanted {
		matches = append(matches, PreferenceMatch{
			Preference: "child_seat",
			Requested:  true,
			Available:  caps.HasChildSeat,
			Match:      caps.HasChildSeat,
		})
	}

	petWanted := prefs.PetFriendly
	if override != nil && override.PetFriendly != nil {
		petWanted = *override.PetFriendly
	}
	if petWanted {
		matches = append(matches, PreferenceMatch{
			Preference: "pet_friendly",
			Requested:  true,
			Available:  caps.PetFriendly,
			Match:      caps.PetFriendly,
		})
	}

	if prefs.WheelchairAccess {
		matches = append(matches, PreferenceMatch{
			Preference: "wheelchair_access",
			Requested:  true,
			Available:  caps.WheelchairAccess,
			Match:      caps.WheelchairAccess,
		})
	}

	return matches
}
