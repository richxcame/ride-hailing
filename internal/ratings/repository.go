package ratings

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles ratings data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new ratings repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateRating creates a new rating
func (r *Repository) CreateRating(ctx context.Context, rating *Rating) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ratings (
			id, ride_id, rater_id, ratee_id, rater_type,
			score, comment, tags, is_public, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		rating.ID, rating.RideID, rating.RaterID, rating.RateeID, rating.RaterType,
		rating.Score, rating.Comment, rating.Tags, rating.IsPublic, rating.CreatedAt,
	)
	return err
}

// GetRatingByRideAndRater checks if a user already rated a ride
func (r *Repository) GetRatingByRideAndRater(ctx context.Context, rideID, raterID uuid.UUID) (*Rating, error) {
	rating := &Rating{}
	err := r.db.QueryRow(ctx, `
		SELECT id, ride_id, rater_id, ratee_id, rater_type,
			score, comment, tags, is_public, created_at
		FROM ratings
		WHERE ride_id = $1 AND rater_id = $2`,
		rideID, raterID,
	).Scan(
		&rating.ID, &rating.RideID, &rating.RaterID, &rating.RateeID, &rating.RaterType,
		&rating.Score, &rating.Comment, &rating.Tags, &rating.IsPublic, &rating.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rating, nil
}

// GetRatingByID retrieves a rating by ID
func (r *Repository) GetRatingByID(ctx context.Context, id uuid.UUID) (*Rating, error) {
	rating := &Rating{}
	err := r.db.QueryRow(ctx, `
		SELECT id, ride_id, rater_id, ratee_id, rater_type,
			score, comment, tags, is_public, created_at
		FROM ratings WHERE id = $1`, id,
	).Scan(
		&rating.ID, &rating.RideID, &rating.RaterID, &rating.RateeID, &rating.RaterType,
		&rating.Score, &rating.Comment, &rating.Tags, &rating.IsPublic, &rating.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rating, nil
}

// GetAverageRating returns the average rating for a user
func (r *Repository) GetAverageRating(ctx context.Context, userID uuid.UUID) (float64, int, error) {
	var avg float64
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(score), 0), COUNT(*)
		FROM ratings WHERE ratee_id = $1`, userID,
	).Scan(&avg, &count)
	return avg, count, err
}

// GetRatingDistribution returns count per score (1-5)
func (r *Repository) GetRatingDistribution(ctx context.Context, userID uuid.UUID) (map[int]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT score, COUNT(*)
		FROM ratings WHERE ratee_id = $1
		GROUP BY score ORDER BY score`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dist := map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	for rows.Next() {
		var score, count int
		if err := rows.Scan(&score, &count); err != nil {
			return nil, err
		}
		dist[score] = count
	}
	return dist, nil
}

// GetTopTags returns the most common tags for a user
func (r *Repository) GetTopTags(ctx context.Context, userID uuid.UUID, limit int) ([]TagCount, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tag, COUNT(*) as cnt
		FROM ratings, UNNEST(tags) as tag
		WHERE ratee_id = $1
		GROUP BY tag
		ORDER BY cnt DESC
		LIMIT $2`, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []TagCount
	for rows.Next() {
		tc := TagCount{}
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, err
		}
		tags = append(tags, tc)
	}
	return tags, nil
}

// GetRecentRatings returns recent ratings received by a user
func (r *Repository) GetRecentRatings(ctx context.Context, userID uuid.UUID, limit int) ([]Rating, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, ride_id, rater_id, ratee_id, rater_type,
			score, comment, tags, is_public, created_at
		FROM ratings
		WHERE ratee_id = $1 AND is_public = true
		ORDER BY created_at DESC
		LIMIT $2`, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ratings []Rating
	for rows.Next() {
		rating := Rating{}
		if err := rows.Scan(
			&rating.ID, &rating.RideID, &rating.RaterID, &rating.RateeID, &rating.RaterType,
			&rating.Score, &rating.Comment, &rating.Tags, &rating.IsPublic, &rating.CreatedAt,
		); err != nil {
			return nil, err
		}
		ratings = append(ratings, rating)
	}
	return ratings, nil
}

// GetRatingsGiven returns ratings a user has given
func (r *Repository) GetRatingsGiven(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Rating, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM ratings WHERE rater_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, ride_id, rater_id, ratee_id, rater_type,
			score, comment, tags, is_public, created_at
		FROM ratings
		WHERE rater_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var ratings []Rating
	for rows.Next() {
		rating := Rating{}
		if err := rows.Scan(
			&rating.ID, &rating.RideID, &rating.RaterID, &rating.RateeID, &rating.RaterType,
			&rating.Score, &rating.Comment, &rating.Tags, &rating.IsPublic, &rating.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		ratings = append(ratings, rating)
	}
	return ratings, total, nil
}

// GetRatingTrend compares current month average to last month
func (r *Repository) GetRatingTrend(ctx context.Context, userID uuid.UUID) (float64, error) {
	now := time.Now()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	firstOfLastMonth := firstOfMonth.AddDate(0, -1, 0)

	var currentAvg, lastAvg float64

	r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(score), 0) FROM ratings
		WHERE ratee_id = $1 AND created_at >= $2`,
		userID, firstOfMonth,
	).Scan(&currentAvg)

	r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(score), 0) FROM ratings
		WHERE ratee_id = $1 AND created_at >= $2 AND created_at < $3`,
		userID, firstOfLastMonth, firstOfMonth,
	).Scan(&lastAvg)

	if lastAvg == 0 {
		return 0, nil
	}
	return currentAvg - lastAvg, nil
}

// CreateRatingResponse creates a response to a rating
func (r *Repository) CreateRatingResponse(ctx context.Context, resp *RatingResponse) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO rating_responses (id, rating_id, user_id, comment, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		resp.ID, resp.RatingID, resp.UserID, resp.Comment, resp.CreatedAt,
	)
	return err
}
