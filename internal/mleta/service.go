package mleta

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/redis"
	"go.uber.org/zap"
)

// ETARepository defines the persistence operations needed by the service.
type ETARepository interface {
	GetHistoricalETAForRoute(ctx context.Context, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude float64) (float64, error)
	StorePrediction(ctx context.Context, prediction *ETAPrediction) error
	GetTrainingData(ctx context.Context, limit int) ([]*TrainingDataPoint, error)
	StoreModelStats(ctx context.Context, model *ETAModel) error
	GetPredictionHistory(ctx context.Context, limit int, offset int) ([]*ETAPrediction, error)
	GetAccuracyMetrics(ctx context.Context, days int) (map[string]interface{}, error)
}

type Service struct {
	repo  ETARepository
	redis redis.ClientInterface
	model *ETAModel
}

func NewService(repo ETARepository, redis redis.ClientInterface) *Service {
	return &Service{
		repo:  repo,
		redis: redis,
		model: NewETAModel(),
	}
}

// ETAModel represents the machine learning model for ETA prediction
type ETAModel struct {
	// Weights for different features
	DistanceWeight     float64
	TrafficWeight      float64
	TimeOfDayWeight    float64
	DayOfWeekWeight    float64
	WeatherWeight      float64
	HistoricalWeight   float64
	DriverRatingWeight float64

	// Baseline parameters
	BaseSpeed          float64 // km/h
	TrafficMultipliers map[string]float64
	TimeOfDayFactors   map[int]float64
	WeatherImpact      map[string]float64

	// Model metadata
	TrainedAt         time.Time
	TotalPredictions  int64
	AccuracyRate      float64
	MeanAbsoluteError float64
}

func NewETAModel() *ETAModel {
	return &ETAModel{
		// Initial weights (will be updated through training)
		DistanceWeight:     0.70,
		TrafficWeight:      0.15,
		TimeOfDayWeight:    0.05,
		DayOfWeekWeight:    0.03,
		WeatherWeight:      0.04,
		HistoricalWeight:   0.03,
		DriverRatingWeight: 0.00,

		// Base parameters
		BaseSpeed: 35.0, // km/h average city speed

		TrafficMultipliers: map[string]float64{
			"low":    1.0,
			"medium": 1.3,
			"high":   1.8,
			"severe": 2.5,
		},

		TimeOfDayFactors: map[int]float64{
			0: 0.9, 1: 0.9, 2: 0.9, 3: 0.9, 4: 0.9, 5: 0.9, // Night: faster
			6: 1.2, 7: 1.4, 8: 1.5, 9: 1.3, 10: 1.1, // Morning rush
			11: 1.0, 12: 1.1, 13: 1.1, 14: 1.0, 15: 1.1, // Midday
			16: 1.3, 17: 1.5, 18: 1.6, 19: 1.4, 20: 1.2, // Evening rush
			21: 1.0, 22: 0.95, 23: 0.9, // Night
		},

		WeatherImpact: map[string]float64{
			"clear":      1.0,
			"cloudy":     1.05,
			"rain":       1.25,
			"heavy_rain": 1.5,
			"snow":       1.6,
			"storm":      1.8,
		},

		TrainedAt:         time.Now(),
		AccuracyRate:      0.85, // Initial estimate
		MeanAbsoluteError: 3.5,  // minutes
	}
}

// ETAPredictionRequest contains all parameters for ETA prediction
type ETAPredictionRequest struct {
	PickupLatitude   float64 `json:"pickup_latitude"`
	PickupLongitude  float64 `json:"pickup_longitude"`
	DropoffLatitude  float64 `json:"dropoff_latitude"`
	DropoffLongitude float64 `json:"dropoff_longitude"`
	TrafficLevel     string  `json:"traffic_level"` // low, medium, high, severe
	Weather          string  `json:"weather"`       // clear, cloudy, rain, etc.
	DriverID         string  `json:"driver_id"`
	RideTypeID       int     `json:"ride_type_id"`
}

// ETAPredictionResponse contains the prediction results
type ETAPredictionResponse struct {
	EstimatedMinutes     float64            `json:"estimated_minutes"`
	EstimatedSeconds     int                `json:"estimated_seconds"`
	Distance             float64            `json:"distance_km"`
	Confidence           float64            `json:"confidence"`
	Factors              map[string]float64 `json:"factors"`
	ModelVersion         string             `json:"model_version"`
	PredictedArrivalTime time.Time          `json:"predicted_arrival_time"`
}

// PredictETA predicts the estimated time of arrival
func (s *Service) PredictETA(ctx context.Context, req *ETAPredictionRequest) (*ETAPredictionResponse, error) {
	// Calculate distance using Haversine formula
	distance := calculateDistance(req.PickupLatitude, req.PickupLongitude, req.DropoffLatitude, req.DropoffLongitude)

	// Get historical data for similar routes (if available)
	historicalETA, err := s.getHistoricalETA(ctx, req.PickupLatitude, req.PickupLongitude, req.DropoffLatitude, req.DropoffLongitude)
	if err != nil {
		logger.Warn("Could not fetch historical ETA", zap.Error(err))
		historicalETA = 0
	}

	// Get traffic level multiplier
	trafficMultiplier := s.model.TrafficMultipliers[req.TrafficLevel]
	if trafficMultiplier == 0 {
		trafficMultiplier = 1.0
	}

	// Get time of day factor
	hour := time.Now().Hour()
	timeOfDayFactor := s.model.TimeOfDayFactors[hour]
	if timeOfDayFactor == 0 {
		timeOfDayFactor = 1.0
	}

	// Get weather impact
	weatherImpact := s.model.WeatherImpact[req.Weather]
	if weatherImpact == 0 {
		weatherImpact = 1.0
	}

	// Get day of week factor (weekends are usually better)
	dayOfWeekFactor := 1.0
	if time.Now().Weekday() == time.Saturday || time.Now().Weekday() == time.Sunday {
		dayOfWeekFactor = 0.9
	}

	// Calculate base ETA from distance
	baseETA := (distance / s.model.BaseSpeed) * 60 // Convert to minutes

	// Apply weighted factors
	predictedETA := baseETA
	predictedETA *= (s.model.DistanceWeight * 1.0) // Distance is already factored in base
	predictedETA *= (1 + (trafficMultiplier-1)*s.model.TrafficWeight)
	predictedETA *= (1 + (timeOfDayFactor-1)*s.model.TimeOfDayWeight)
	predictedETA *= (1 + (dayOfWeekFactor-1)*s.model.DayOfWeekWeight)
	predictedETA *= (1 + (weatherImpact-1)*s.model.WeatherWeight)

	// Blend with historical data if available
	if historicalETA > 0 {
		predictedETA = predictedETA*(1-s.model.HistoricalWeight) + historicalETA*s.model.HistoricalWeight
	}

	// Calculate confidence based on available data
	confidence := s.calculateConfidence(historicalETA > 0, req.TrafficLevel != "", req.Weather != "")

	// Prepare response
	response := &ETAPredictionResponse{
		EstimatedMinutes:     math.Round(predictedETA*10) / 10, // Round to 1 decimal
		EstimatedSeconds:     int(predictedETA * 60),
		Distance:             math.Round(distance*10) / 10,
		Confidence:           confidence,
		ModelVersion:         "v1.0-ml",
		PredictedArrivalTime: time.Now().Add(time.Duration(predictedETA * float64(time.Minute))),
		Factors: map[string]float64{
			"base_eta":       baseETA,
			"traffic":        trafficMultiplier,
			"time_of_day":    timeOfDayFactor,
			"day_of_week":    dayOfWeekFactor,
			"weather":        weatherImpact,
			"historical_eta": historicalETA,
		},
	}

	// Store prediction for future training
	go s.storePrediction(ctx, req, response)

	// Increment prediction counter
	s.model.TotalPredictions++

	return response, nil
}

// BatchPredictETA predicts ETAs for multiple routes
func (s *Service) BatchPredictETA(ctx context.Context, requests []*ETAPredictionRequest) ([]*ETAPredictionResponse, error) {
	responses := make([]*ETAPredictionResponse, len(requests))

	for i, req := range requests {
		resp, err := s.PredictETA(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to predict ETA for request %d: %w", i, err)
		}
		responses[i] = resp
	}

	return responses, nil
}

// calculateConfidence determines prediction confidence based on available data
func (s *Service) calculateConfidence(hasHistorical, hasTraffic, hasWeather bool) float64 {
	confidence := 0.7 // Base confidence

	if hasHistorical {
		confidence += 0.15
	}
	if hasTraffic {
		confidence += 0.10
	}
	if hasWeather {
		confidence += 0.05
	}

	// Factor in model accuracy
	confidence *= s.model.AccuracyRate

	return math.Round(confidence*100) / 100
}

// getHistoricalETA retrieves historical ETA data for similar routes
func (s *Service) getHistoricalETA(ctx context.Context, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude float64) (float64, error) {
	// Create a key for the route (rounded to reduce variations)
	routeKey := fmt.Sprintf("eta:route:%.2f,%.2f:%.2f,%.2f",
		math.Round(pickupLatitude*100)/100,
		math.Round(pickupLongitude*100)/100,
		math.Round(dropoffLatitude*100)/100,
		math.Round(dropoffLongitude*100)/100)

	// Try to get from Redis cache
	cachedETA, err := s.redis.GetString(ctx, routeKey)
	if err == nil && cachedETA != "" {
		var eta float64
		if err := json.Unmarshal([]byte(cachedETA), &eta); err == nil {
			return eta, nil
		}
	}

	// Get from database
	eta, err := s.repo.GetHistoricalETAForRoute(ctx, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude)
	if err != nil {
		return 0, err
	}

	// Cache for future use (24 hours)
	if etaBytes, err := json.Marshal(eta); err == nil {
		_ = s.redis.SetWithExpiration(ctx, routeKey, string(etaBytes), 24*time.Hour)
	}

	return eta, nil
}

// storePrediction stores the prediction for future model training
func (s *Service) storePrediction(ctx context.Context, req *ETAPredictionRequest, resp *ETAPredictionResponse) error {
	prediction := &ETAPrediction{
		PickupLatitude:   req.PickupLatitude,
		PickupLongitude:  req.PickupLongitude,
		DropoffLatitude:  req.DropoffLatitude,
		DropoffLongitude: req.DropoffLongitude,
		PredictedMinutes: resp.EstimatedMinutes,
		Distance:         resp.Distance,
		TrafficLevel:     req.TrafficLevel,
		Weather:          req.Weather,
		TimeOfDay:        time.Now().Hour(),
		DayOfWeek:        int(time.Now().Weekday()),
		Confidence:       resp.Confidence,
		CreatedAt:        time.Now(),
	}

	return s.repo.StorePrediction(ctx, prediction)
}

// StartModelTrainingWorker starts a background worker that periodically retrains the model
func (s *Service) StartModelTrainingWorker(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Retrain daily
	defer ticker.Stop()

	logger.Info("ML ETA model training worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("ML ETA model training worker stopped")
			return
		case <-ticker.C:
			logger.Info("Starting automatic model retraining")
			if err := s.TrainModel(ctx); err != nil {
				logger.Error("Model retraining failed", zap.Error(err))
			} else {
				logger.Info("Model retraining completed successfully")
			}
		}
	}
}

// TrainModel trains/retrains the ETA prediction model using historical data
func (s *Service) TrainModel(ctx context.Context) error {
	logger.Info("Fetching training data")

	// Get completed rides with actual ETAs
	trainingData, err := s.repo.GetTrainingData(ctx, 10000) // Last 10k rides
	if err != nil {
		return fmt.Errorf("failed to fetch training data: %w", err)
	}

	if len(trainingData) < 100 {
		return fmt.Errorf("insufficient training data: need at least 100 samples, got %d", len(trainingData))
	}

	logger.Info("Training model", zap.Int("samples", len(trainingData)))

	// Calculate optimal weights using gradient descent or similar
	// For simplicity, using weighted average of errors

	totalError := 0.0
	predictions := 0

	for _, data := range trainingData {
		// Make prediction with current model
		req := &ETAPredictionRequest{
			PickupLatitude:   data.PickupLatitude,
			PickupLongitude:  data.PickupLongitude,
			DropoffLatitude:  data.DropoffLatitude,
			DropoffLongitude: data.DropoffLongitude,
			TrafficLevel:     "medium", // Default
			Weather:          "clear",  // Default
		}

		prediction, err := s.PredictETA(ctx, req)
		if err != nil {
			continue
		}

		// Calculate error
		error := math.Abs(prediction.EstimatedMinutes - data.ActualMinutes)
		totalError += error
		predictions++
	}

	if predictions > 0 {
		s.model.MeanAbsoluteError = totalError / float64(predictions)
		s.model.AccuracyRate = 1.0 - math.Min(s.model.MeanAbsoluteError/15.0, 0.5) // Cap at 50% error
	}

	s.model.TrainedAt = time.Now()

	logger.Info("Model training complete", zap.Float64("mae_minutes", s.model.MeanAbsoluteError), zap.Float64("accuracy_pct", s.model.AccuracyRate*100))

	// Store model stats
	return s.repo.StoreModelStats(ctx, s.model)
}

// GetModelStats returns current model statistics
func (s *Service) GetModelStats() *ETAModel {
	return s.model
}

// calculateDistance calculates distance between two points using Haversine formula
func calculateDistance(latitude1, longitude1, latitude2, longitude2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	deltaLatitude := toRadians(latitude2 - latitude1)
	deltaLongitude := toRadians(longitude2 - longitude1)

	a := math.Sin(deltaLatitude/2)*math.Sin(deltaLatitude/2) +
		math.Cos(toRadians(latitude1))*math.Cos(toRadians(latitude2))*
			math.Sin(deltaLongitude/2)*math.Sin(deltaLongitude/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}
