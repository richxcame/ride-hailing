package mleta

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/pagination"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// PredictETA handles ETA prediction requests
func (h *Handler) PredictETA(c *gin.Context) {
	var req ETAPredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate coordinates
	if req.PickupLatitude == 0 || req.PickupLongitude == 0 || req.DropoffLatitude == 0 || req.DropoffLongitude == 0 {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid coordinates")
		return
	}

	prediction, err := h.service.PredictETA(c.Request.Context(), &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to predict ETA")
		return
	}

	common.SuccessResponse(c, gin.H{
		"prediction": prediction,
	})
}

// BatchPredictETA handles batch ETA prediction requests
func (h *Handler) BatchPredictETA(c *gin.Context) {
	var requests []*ETAPredictionRequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if len(requests) == 0 {
		common.ErrorResponse(c, http.StatusBadRequest, "No requests provided")
		return
	}

	if len(requests) > 100 {
		common.ErrorResponse(c, http.StatusBadRequest, "Maximum 100 requests allowed per batch")
		return
	}

	predictions, err := h.service.BatchPredictETA(c.Request.Context(), requests)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to predict ETAs")
		return
	}

	common.SuccessResponse(c, gin.H{
		"predictions": predictions,
		"count":       len(predictions),
	})
}

// TriggerModelTraining manually triggers model retraining (admin only)
func (h *Handler) TriggerModelTraining(c *gin.Context) {
	go func() {
		if err := h.service.TrainModel(c.Request.Context()); err != nil {
			// Log error but don't fail the request
			c.Error(err)
		}
	}()

	common.SuccessResponseWithStatus(c, http.StatusAccepted, gin.H{
		"message": "Model training started in background",
	}, "")
}

// GetModelStats returns current model statistics
func (h *Handler) GetModelStats(c *gin.Context) {
	stats := h.service.GetModelStats()

	common.SuccessResponse(c, gin.H{
		"stats": stats,
	})
}

// GetModelAccuracy returns model accuracy metrics
func (h *Handler) GetModelAccuracy(c *gin.Context) {
	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	metrics, err := h.service.repo.GetAccuracyMetrics(c.Request.Context(), days)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to get accuracy metrics")
		return
	}

	common.SuccessResponse(c, gin.H{
		"metrics": metrics,
	})
}

// TuneHyperparameters allows admins to adjust model hyperparameters
func (h *Handler) TuneHyperparameters(c *gin.Context) {
	var params struct {
		DistanceWeight   *float64 `json:"distance_weight"`
		TrafficWeight    *float64 `json:"traffic_weight"`
		TimeOfDayWeight  *float64 `json:"time_of_day_weight"`
		DayOfWeekWeight  *float64 `json:"day_of_week_weight"`
		WeatherWeight    *float64 `json:"weather_weight"`
		HistoricalWeight *float64 `json:"historical_weight"`
		BaseSpeed        *float64 `json:"base_speed"`
	}

	if err := c.ShouldBindJSON(&params); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	model := h.service.model

	// Update weights if provided
	if params.DistanceWeight != nil && *params.DistanceWeight >= 0 && *params.DistanceWeight <= 1 {
		model.DistanceWeight = *params.DistanceWeight
	}
	if params.TrafficWeight != nil && *params.TrafficWeight >= 0 && *params.TrafficWeight <= 1 {
		model.TrafficWeight = *params.TrafficWeight
	}
	if params.TimeOfDayWeight != nil && *params.TimeOfDayWeight >= 0 && *params.TimeOfDayWeight <= 1 {
		model.TimeOfDayWeight = *params.TimeOfDayWeight
	}
	if params.DayOfWeekWeight != nil && *params.DayOfWeekWeight >= 0 && *params.DayOfWeekWeight <= 1 {
		model.DayOfWeekWeight = *params.DayOfWeekWeight
	}
	if params.WeatherWeight != nil && *params.WeatherWeight >= 0 && *params.WeatherWeight <= 1 {
		model.WeatherWeight = *params.WeatherWeight
	}
	if params.HistoricalWeight != nil && *params.HistoricalWeight >= 0 && *params.HistoricalWeight <= 1 {
		model.HistoricalWeight = *params.HistoricalWeight
	}
	if params.BaseSpeed != nil && *params.BaseSpeed > 0 && *params.BaseSpeed < 200 {
		model.BaseSpeed = *params.BaseSpeed
	}

	// Validate that weights sum to reasonable value
	totalWeight := model.DistanceWeight + model.TrafficWeight + model.TimeOfDayWeight +
		model.DayOfWeekWeight + model.WeatherWeight + model.HistoricalWeight

	if totalWeight < 0.8 || totalWeight > 1.2 {
		common.ErrorResponse(c, http.StatusBadRequest, "Weights must sum to approximately 1.0")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Hyperparameters updated successfully",
		"model":   model,
	})
}

// GetPredictionHistory returns historical predictions
func (h *Handler) GetPredictionHistory(c *gin.Context) {
	params := pagination.ParseParams(c)

	predictions, err := h.service.repo.GetPredictionHistory(c.Request.Context(), params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to get prediction history")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(len(predictions)))
	common.SuccessResponseWithMeta(c, gin.H{
		"predictions": predictions,
		"count":       len(predictions),
	}, meta)
}

// GetAccuracyTrends returns accuracy trends over time
func (h *Handler) GetAccuracyTrends(c *gin.Context) {
	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	metrics, err := h.service.repo.GetAccuracyMetrics(c.Request.Context(), days)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to get accuracy trends")
		return
	}

	common.SuccessResponse(c, gin.H{
		"trends": metrics,
	})
}

// GetFeatureImportance returns feature importance for the model
func (h *Handler) GetFeatureImportance(c *gin.Context) {
	model := h.service.model

	features := map[string]float64{
		"distance":    model.DistanceWeight,
		"traffic":     model.TrafficWeight,
		"time_of_day": model.TimeOfDayWeight,
		"day_of_week": model.DayOfWeekWeight,
		"weather":     model.WeatherWeight,
		"historical":  model.HistoricalWeight,
	}

	common.SuccessResponse(c, gin.H{
		"features":      features,
		"model_version": "v1.0-ml",
		"trained_at":    model.TrainedAt,
	})
}
