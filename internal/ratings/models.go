package ratings

import (
	"time"

	"github.com/google/uuid"
)

// RaterType represents who is giving the rating
type RaterType string

const (
	RaterTypeRider  RaterType = "rider"
	RaterTypeDriver RaterType = "driver"
)

// RatingTag represents a predefined feedback tag
type RatingTag string

const (
	// Positive rider-to-driver tags
	TagGreatConversation RatingTag = "great_conversation"
	TagSmoothDriving     RatingTag = "smooth_driving"
	TagCleanCar          RatingTag = "clean_car"
	TagKnowsRoute        RatingTag = "knows_route"
	TagFriendly          RatingTag = "friendly"
	TagProfessional      RatingTag = "professional"
	TagGoodMusic         RatingTag = "good_music"
	TagSafeDriver        RatingTag = "safe_driver"

	// Negative rider-to-driver tags
	TagRoughDriving  RatingTag = "rough_driving"
	TagDirtyCar      RatingTag = "dirty_car"
	TagRude          RatingTag = "rude"
	TagUnsafe        RatingTag = "unsafe"
	TagLostRoute     RatingTag = "lost_route"
	TagPhoneUse      RatingTag = "phone_use"
	TagLateArrival   RatingTag = "late_arrival"

	// Positive driver-to-rider tags
	TagPoliteRider   RatingTag = "polite_rider"
	TagOnTime        RatingTag = "on_time"
	TagRespectful    RatingTag = "respectful"
	TagGoodDirections RatingTag = "good_directions"

	// Negative driver-to-rider tags
	TagRudeRider     RatingTag = "rude_rider"
	TagMessyRider    RatingTag = "messy_rider"
	TagLatePickup    RatingTag = "late_pickup"
	TagSlammedDoor   RatingTag = "slammed_door"
)

// Rating represents a ride rating
type Rating struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	RideID     uuid.UUID  `json:"ride_id" db:"ride_id"`
	RaterID    uuid.UUID  `json:"rater_id" db:"rater_id"`
	RateeID    uuid.UUID  `json:"ratee_id" db:"ratee_id"`
	RaterType  RaterType  `json:"rater_type" db:"rater_type"`
	Score      int        `json:"score" db:"score"` // 1-5
	Comment    *string    `json:"comment,omitempty" db:"comment"`
	Tags       []string   `json:"tags,omitempty" db:"tags"`
	IsPublic   bool       `json:"is_public" db:"is_public"` // Visible to ratee
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// RatingResponse represents a response to a rating (driver responding to rider feedback)
type RatingResponse struct {
	ID        uuid.UUID `json:"id" db:"id"`
	RatingID  uuid.UUID `json:"rating_id" db:"rating_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Comment   string    `json:"comment" db:"comment"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// UserRatingProfile represents aggregated ratings for a user
type UserRatingProfile struct {
	UserID           uuid.UUID          `json:"user_id"`
	AverageRating    float64            `json:"average_rating"`
	TotalRatings     int                `json:"total_ratings"`
	RatingDistribution map[int]int      `json:"rating_distribution"` // 1-5 star counts
	TopTags          []TagCount         `json:"top_tags"`
	RecentRatings    []Rating           `json:"recent_ratings,omitempty"`
	RatingTrend      float64            `json:"rating_trend"` // Change from last month
}

// TagCount represents a tag with its frequency
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// SubmitRatingRequest submits a rating for a ride
type SubmitRatingRequest struct {
	RideID  uuid.UUID `json:"ride_id" binding:"required"`
	Score   int       `json:"score" binding:"required,min=1,max=5"`
	Comment *string   `json:"comment,omitempty"`
	Tags    []string  `json:"tags,omitempty"`
}

// RespondToRatingRequest responds to a received rating
type RespondToRatingRequest struct {
	Comment string `json:"comment" binding:"required"`
}

// RatingPrompt represents what's shown to a user after a ride
type RatingPrompt struct {
	RideID          uuid.UUID   `json:"ride_id"`
	OtherUserName   string      `json:"other_user_name"`
	OtherUserPhoto  *string     `json:"other_user_photo,omitempty"`
	SuggestedTags   []RatingTag `json:"suggested_tags"`
	AlreadyRated    bool        `json:"already_rated"`
}
