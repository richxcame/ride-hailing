package favorites

import (
	"context"

	"github.com/google/uuid"
)

// Service handles business logic for favorite locations
type Service struct {
	repo *Repository
}

// NewService creates a new favorite locations service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateFavoriteLocation creates a new favorite location for a user
func (s *Service) CreateFavoriteLocation(ctx context.Context, userID uuid.UUID, name, address string, latitude, longitude float64) (*FavoriteLocation, error) {
	// Validate input
	if name == "" {
		return nil, ErrInvalidName
	}

	if address == "" {
		return nil, ErrInvalidAddress
	}

	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return nil, ErrInvalidCoordinates
	}

	favorite := &FavoriteLocation{
		UserID:    userID,
		Name:      name,
		Address:   address,
		Latitude:  latitude,
		Longitude: longitude,
	}

	if err := s.repo.CreateFavorite(ctx, favorite); err != nil {
		return nil, err
	}

	return favorite, nil
}

// GetFavoriteLocations retrieves all favorite locations for a user
func (s *Service) GetFavoriteLocations(ctx context.Context, userID uuid.UUID) ([]*FavoriteLocation, error) {
	return s.repo.GetFavoritesByUser(ctx, userID)
}

// GetFavoriteLocation retrieves a specific favorite location
func (s *Service) GetFavoriteLocation(ctx context.Context, id, userID uuid.UUID) (*FavoriteLocation, error) {
	favorite, err := s.repo.GetFavoriteByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Ensure the favorite belongs to the user
	if favorite.UserID != userID {
		return nil, ErrUnauthorized
	}

	return favorite, nil
}

// UpdateFavoriteLocation updates a favorite location
func (s *Service) UpdateFavoriteLocation(ctx context.Context, id, userID uuid.UUID, name, address string, latitude, longitude float64) (*FavoriteLocation, error) {
	// Validate input
	if name == "" {
		return nil, ErrInvalidName
	}

	if address == "" {
		return nil, ErrInvalidAddress
	}

	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return nil, ErrInvalidCoordinates
	}

	// Get existing favorite to verify ownership
	existing, err := s.repo.GetFavoriteByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Update fields
	existing.Name = name
	existing.Address = address
	existing.Latitude = latitude
	existing.Longitude = longitude

	if err := s.repo.UpdateFavorite(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

// DeleteFavoriteLocation deletes a favorite location
func (s *Service) DeleteFavoriteLocation(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.DeleteFavorite(ctx, id, userID)
}
