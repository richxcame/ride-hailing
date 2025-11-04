package favorites

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FavoriteLocation represents a user's favorite location
type FavoriteLocation struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`       // "Home", "Work", "Gym", etc.
	Address   string    `json:"address"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Repository handles database operations for favorite locations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new favorite locations repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateFavorite creates a new favorite location
func (r *Repository) CreateFavorite(ctx context.Context, favorite *FavoriteLocation) error {
	query := `
		INSERT INTO favorite_locations (id, user_id, name, address, latitude, longitude, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`

	favorite.ID = uuid.New()
	now := time.Now()
	favorite.CreatedAt = now
	favorite.UpdatedAt = now

	err := r.db.QueryRow(ctx, query,
		favorite.ID,
		favorite.UserID,
		favorite.Name,
		favorite.Address,
		favorite.Latitude,
		favorite.Longitude,
		favorite.CreatedAt,
		favorite.UpdatedAt,
	).Scan(&favorite.CreatedAt, &favorite.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create favorite location: %w", err)
	}

	return nil
}

// GetFavoriteByID retrieves a favorite location by ID
func (r *Repository) GetFavoriteByID(ctx context.Context, id uuid.UUID) (*FavoriteLocation, error) {
	query := `
		SELECT id, user_id, name, address, latitude, longitude, created_at, updated_at
		FROM favorite_locations
		WHERE id = $1
	`

	favorite := &FavoriteLocation{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&favorite.ID,
		&favorite.UserID,
		&favorite.Name,
		&favorite.Address,
		&favorite.Latitude,
		&favorite.Longitude,
		&favorite.CreatedAt,
		&favorite.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get favorite location: %w", err)
	}

	return favorite, nil
}

// GetFavoritesByUser retrieves all favorite locations for a user
func (r *Repository) GetFavoritesByUser(ctx context.Context, userID uuid.UUID) ([]*FavoriteLocation, error) {
	query := `
		SELECT id, user_id, name, address, latitude, longitude, created_at, updated_at
		FROM favorite_locations
		WHERE user_id = $1
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get favorite locations: %w", err)
	}
	defer rows.Close()

	var favorites []*FavoriteLocation
	for rows.Next() {
		favorite := &FavoriteLocation{}
		err := rows.Scan(
			&favorite.ID,
			&favorite.UserID,
			&favorite.Name,
			&favorite.Address,
			&favorite.Latitude,
			&favorite.Longitude,
			&favorite.CreatedAt,
			&favorite.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan favorite location: %w", err)
		}
		favorites = append(favorites, favorite)
	}

	return favorites, nil
}

// UpdateFavorite updates a favorite location
func (r *Repository) UpdateFavorite(ctx context.Context, favorite *FavoriteLocation) error {
	query := `
		UPDATE favorite_locations
		SET name = $1, address = $2, latitude = $3, longitude = $4, updated_at = $5
		WHERE id = $6 AND user_id = $7
	`

	favorite.UpdatedAt = time.Now()

	result, err := r.db.Exec(ctx, query,
		favorite.Name,
		favorite.Address,
		favorite.Latitude,
		favorite.Longitude,
		favorite.UpdatedAt,
		favorite.ID,
		favorite.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update favorite location: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("favorite location not found")
	}

	return nil
}

// DeleteFavorite deletes a favorite location
func (r *Repository) DeleteFavorite(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM favorite_locations WHERE id = $1 AND user_id = $2`

	result, err := r.db.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete favorite location: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("favorite location not found")
	}

	return nil
}
