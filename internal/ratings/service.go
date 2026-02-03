package ratings

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
)

// Service handles ratings business logic
type Service struct {
	repo *Repository
}

// NewService creates a new ratings service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// SubmitRating submits a rating for a ride
func (s *Service) SubmitRating(ctx context.Context, raterID uuid.UUID, rateeID uuid.UUID, raterType RaterType, req *SubmitRatingRequest) (*Rating, error) {
	if req.Score < 1 || req.Score > 5 {
		return nil, common.NewBadRequestError("score must be between 1 and 5", nil)
	}

	// Check if already rated
	existing, err := s.repo.GetRatingByRideAndRater(ctx, req.RideID, raterID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}
	if existing != nil {
		return nil, common.NewConflictError("you have already rated this ride")
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	now := time.Now()
	rating := &Rating{
		ID:        uuid.New(),
		RideID:    req.RideID,
		RaterID:   raterID,
		RateeID:   rateeID,
		RaterType: raterType,
		Score:     req.Score,
		Comment:   req.Comment,
		Tags:      tags,
		IsPublic:  true,
		CreatedAt: now,
	}

	if err := s.repo.CreateRating(ctx, rating); err != nil {
		return nil, fmt.Errorf("create rating: %w", err)
	}

	return rating, nil
}

// GetMyRatingProfile returns the user's rating profile
func (s *Service) GetMyRatingProfile(ctx context.Context, userID uuid.UUID) (*UserRatingProfile, error) {
	avg, total, err := s.repo.GetAverageRating(ctx, userID)
	if err != nil {
		return nil, err
	}

	dist, err := s.repo.GetRatingDistribution(ctx, userID)
	if err != nil {
		dist = map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	}

	topTags, err := s.repo.GetTopTags(ctx, userID, 10)
	if err != nil {
		topTags = []TagCount{}
	}

	recentRatings, err := s.repo.GetRecentRatings(ctx, userID, 10)
	if err != nil {
		recentRatings = []Rating{}
	}

	trend, _ := s.repo.GetRatingTrend(ctx, userID)

	return &UserRatingProfile{
		UserID:             userID,
		AverageRating:      avg,
		TotalRatings:       total,
		RatingDistribution: dist,
		TopTags:            topTags,
		RecentRatings:      recentRatings,
		RatingTrend:        trend,
	}, nil
}

// GetUserRating returns the public rating profile for another user
func (s *Service) GetUserRating(ctx context.Context, userID uuid.UUID) (*UserRatingProfile, error) {
	avg, total, err := s.repo.GetAverageRating(ctx, userID)
	if err != nil {
		return nil, err
	}

	topTags, err := s.repo.GetTopTags(ctx, userID, 5)
	if err != nil {
		topTags = []TagCount{}
	}

	return &UserRatingProfile{
		UserID:        userID,
		AverageRating: avg,
		TotalRatings:  total,
		TopTags:       topTags,
	}, nil
}

// RespondToRating responds to a received rating
func (s *Service) RespondToRating(ctx context.Context, userID uuid.UUID, ratingID uuid.UUID, req *RespondToRatingRequest) (*RatingResponse, error) {
	rating, err := s.repo.GetRatingByID(ctx, ratingID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("rating not found", nil)
		}
		return nil, err
	}

	if rating.RateeID != userID {
		return nil, common.NewForbiddenError("you can only respond to your own ratings")
	}

	resp := &RatingResponse{
		ID:        uuid.New(),
		RatingID:  ratingID,
		UserID:    userID,
		Comment:   req.Comment,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateRatingResponse(ctx, resp); err != nil {
		return nil, fmt.Errorf("create rating response: %w", err)
	}

	return resp, nil
}

// GetRatingsGiven returns ratings the user has given
func (s *Service) GetRatingsGiven(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Rating, int, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	ratings, total, err := s.repo.GetRatingsGiven(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if ratings == nil {
		ratings = []Rating{}
	}
	return ratings, total, nil
}

// GetSuggestedTags returns appropriate tags based on rater type
func (s *Service) GetSuggestedTags(raterType RaterType) []RatingTag {
	if raterType == RaterTypeRider {
		return []RatingTag{
			TagGreatConversation, TagSmoothDriving, TagCleanCar,
			TagKnowsRoute, TagFriendly, TagProfessional,
			TagGoodMusic, TagSafeDriver,
		}
	}
	return []RatingTag{
		TagPoliteRider, TagOnTime, TagRespectful, TagGoodDirections,
	}
}
