package mleta

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richxcame/ride-hailing/pkg/redis"
)

type Repository struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// Ensure Repository satisfies the service interface.
var _ ETARepository = (*Repository)(nil)

func NewRepository(db *pgxpool.Pool, redis *redis.Client) *Repository {
	return &Repository{db: db, redis: redis}
}

// ETAPrediction represents a stored ETA prediction
type ETAPrediction struct {
	ID               int       `json:"id"`
	PickupLatitude   float64   `json:"pickup_latitude"`
	PickupLongitude  float64   `json:"pickup_longitude"`
	DropoffLatitude  float64   `json:"dropoff_latitude"`
	DropoffLongitude float64   `json:"dropoff_longitude"`
	PredictedMinutes float64   `json:"predicted_minutes"`
	ActualMinutes    float64   `json:"actual_minutes,omitempty"`
	Distance         float64   `json:"distance"`
	TrafficLevel     string    `json:"traffic_level"`
	Weather          string    `json:"weather"`
	TimeOfDay        int       `json:"time_of_day"`
	DayOfWeek        int       `json:"day_of_week"`
	Confidence       float64   `json:"confidence"`
	RideID           string    `json:"ride_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

// TrainingDataPoint represents data used for model training
type TrainingDataPoint struct {
	PickupLatitude   float64 `json:"pickup_latitude"`
	PickupLongitude  float64 `json:"pickup_longitude"`
	DropoffLatitude  float64 `json:"dropoff_latitude"`
	DropoffLongitude float64 `json:"dropoff_longitude"`
	ActualMinutes    float64 `json:"actual_minutes"`
	Distance         float64 `json:"distance"`
	TrafficLevel     string  `json:"traffic_level"`
	Weather          string  `json:"weather"`
	TimeOfDay        int     `json:"time_of_day"`
	DayOfWeek        int     `json:"day_of_day"`
}

// StorePrediction stores an ETA prediction
func (r *Repository) StorePrediction(ctx context.Context, prediction *ETAPrediction) error {
	query := `
		INSERT INTO eta_predictions (
			pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
			predicted_minutes, distance, traffic_level, weather,
			time_of_day, day_of_week, confidence, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		prediction.PickupLatitude,
		prediction.PickupLongitude,
		prediction.DropoffLatitude,
		prediction.DropoffLongitude,
		prediction.PredictedMinutes,
		prediction.Distance,
		prediction.TrafficLevel,
		prediction.Weather,
		prediction.TimeOfDay,
		prediction.DayOfWeek,
		prediction.Confidence,
		time.Now(),
	).Scan(&prediction.ID)

	return err
}

// UpdatePredictionWithActual updates a prediction with actual ride duration
func (r *Repository) UpdatePredictionWithActual(ctx context.Context, rideID string, actualMinutes float64) error {
	query := `
		UPDATE eta_predictions
		SET actual_minutes = $1, ride_id = $2, updated_at = $3
		WHERE id = (
			SELECT id FROM eta_predictions
			WHERE ride_id IS NULL
			ORDER BY created_at DESC
			LIMIT 1
		)
	`

	_, err := r.db.Exec(ctx, query, actualMinutes, rideID, time.Now())
	return err
}

// GetHistoricalETAForRoute gets average historical ETA for similar routes
func (r *Repository) GetHistoricalETAForRoute(ctx context.Context, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude float64) (float64, error) {
	// Query for rides in similar area (within 0.5km radius)
	query := `
        SELECT COALESCE(AVG(actual_minutes), 0) as avg_eta
		FROM eta_predictions
		WHERE actual_minutes IS NOT NULL
			AND actual_minutes > 0
			AND ABS(pickup_latitude - $1) < 0.005
			AND ABS(pickup_longitude - $2) < 0.005
			AND ABS(dropoff_latitude - $3) < 0.005
			AND ABS(dropoff_longitude - $4) < 0.005
			AND created_at > NOW() - INTERVAL '90 days'
		GROUP BY DATE_TRUNC('hour', created_at)
		ORDER BY DATE_TRUNC('hour', created_at) DESC
		LIMIT 10
	`

	var avgETA float64
	err := r.db.QueryRow(ctx, query, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude).Scan(&avgETA)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}

	return avgETA, nil
}

// GetTrainingData retrieves historical data for model training
func (r *Repository) GetTrainingData(ctx context.Context, limit int) ([]*TrainingDataPoint, error) {
	query := `
		SELECT
			pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
			actual_minutes, distance,
			COALESCE(traffic_level, 'medium') as traffic_level,
			COALESCE(weather, 'clear') as weather,
			time_of_day, day_of_week
		FROM eta_predictions
		WHERE actual_minutes IS NOT NULL
			AND actual_minutes > 0
			AND actual_minutes < 180  -- Filter outliers (> 3 hours)
			AND distance > 0
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dataPoints := make([]*TrainingDataPoint, 0)
	for rows.Next() {
		var dp TrainingDataPoint
		err := rows.Scan(
			&dp.PickupLatitude,
			&dp.PickupLongitude,
			&dp.DropoffLatitude,
			&dp.DropoffLongitude,
			&dp.ActualMinutes,
			&dp.Distance,
			&dp.TrafficLevel,
			&dp.Weather,
			&dp.TimeOfDay,
			&dp.DayOfWeek,
		)
		if err != nil {
			return nil, err
		}
		dataPoints = append(dataPoints, &dp)
	}

	return dataPoints, rows.Err()
}

// StoreModelStats stores model training statistics
func (r *Repository) StoreModelStats(ctx context.Context, model *ETAModel) error {
	// Store in database
	query := `
		INSERT INTO eta_model_stats (
			trained_at, total_predictions, accuracy_rate, mean_absolute_error,
			model_config
		) VALUES ($1, $2, $3, $4, $5)
	`

	configJSON, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("failed to marshal model config: %w", err)
	}

	_, err = r.db.Exec(ctx, query,
		model.TrainedAt,
		model.TotalPredictions,
		model.AccuracyRate,
		model.MeanAbsoluteError,
		configJSON,
	)
	if err != nil {
		return err
	}

	// Also cache in Redis for quick access
	cacheKey := "eta:model:latest"
	return r.redis.SetWithExpiration(ctx, cacheKey, string(configJSON), 24*time.Hour)
}

// GetModelStats retrieves the latest model statistics
func (r *Repository) GetModelStats(ctx context.Context) (*ETAModel, error) {
	// Try Redis cache first
	cacheKey := "eta:model:latest"
	cached, err := r.redis.GetString(ctx, cacheKey)
	if err == nil && cached != "" {
		var model ETAModel
		if err := json.Unmarshal([]byte(cached), &model); err == nil {
			return &model, nil
		}
	}

	// Fallback to database
	query := `
		SELECT model_config, trained_at, total_predictions, accuracy_rate, mean_absolute_error
		FROM eta_model_stats
		ORDER BY trained_at DESC
		LIMIT 1
	`

	var configJSON []byte
	var model ETAModel
	err = r.db.QueryRow(ctx, query).Scan(
		&configJSON,
		&model.TrainedAt,
		&model.TotalPredictions,
		&model.AccuracyRate,
		&model.MeanAbsoluteError,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(configJSON, &model); err != nil {
		return nil, fmt.Errorf("failed to unmarshal model config: %w", err)
	}

	return &model, nil
}

// GetPredictionHistory retrieves historical predictions
func (r *Repository) GetPredictionHistory(ctx context.Context, limit int, offset int) ([]*ETAPrediction, error) {
	query := `
		SELECT
			id, pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
            predicted_minutes, COALESCE(actual_minutes, 0), distance, traffic_level,
            weather, time_of_day, day_of_week, confidence, COALESCE(ride_id, ''), created_at
		FROM eta_predictions
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	predictions := make([]*ETAPrediction, 0)
	for rows.Next() {
		var p ETAPrediction
		err := rows.Scan(
			&p.ID,
			&p.PickupLatitude,
			&p.PickupLongitude,
			&p.DropoffLatitude,
			&p.DropoffLongitude,
			&p.PredictedMinutes,
			&p.ActualMinutes,
			&p.Distance,
			&p.TrafficLevel,
			&p.Weather,
			&p.TimeOfDay,
			&p.DayOfWeek,
			&p.Confidence,
			&p.RideID,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		predictions = append(predictions, &p)
	}

	return predictions, rows.Err()
}

// GetAccuracyMetrics calculates accuracy metrics over time
func (r *Repository) GetAccuracyMetrics(ctx context.Context, days int) (map[string]interface{}, error) {
	query := `
		SELECT
			DATE(created_at) as date,
			COUNT(*) as total_predictions,
			COUNT(CASE WHEN actual_minutes IS NOT NULL THEN 1 END) as completed_rides,
			AVG(CASE
				WHEN actual_minutes IS NOT NULL
				THEN ABS(predicted_minutes - actual_minutes)
			END) as mean_absolute_error,
			AVG(CASE
				WHEN actual_minutes IS NOT NULL
				THEN (1 - LEAST(ABS(predicted_minutes - actual_minutes) / actual_minutes, 1)) * 100
			END) as accuracy_percentage
		FROM eta_predictions
        WHERE created_at > NOW() - ($1::int * INTERVAL '1 day')
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`

	rows, err := r.db.Query(ctx, query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type DailyMetric struct {
		Date               string  `json:"date"`
		TotalPredictions   int     `json:"total_predictions"`
		CompletedRides     int     `json:"completed_rides"`
		MeanAbsoluteError  float64 `json:"mean_absolute_error"`
		AccuracyPercentage float64 `json:"accuracy_percentage"`
	}

	var metrics []DailyMetric
	for rows.Next() {
		var m DailyMetric
		var mae, acc sql.NullFloat64

		err := rows.Scan(
			&m.Date,
			&m.TotalPredictions,
			&m.CompletedRides,
			&mae,
			&acc,
		)
		if err != nil {
			return nil, err
		}

		if mae.Valid {
			m.MeanAbsoluteError = mae.Float64
		}
		if acc.Valid {
			m.AccuracyPercentage = acc.Float64
		}

		metrics = append(metrics, m)
	}

	return map[string]interface{}{
		"daily_metrics": metrics,
		"period_days":   days,
	}, rows.Err()
}
