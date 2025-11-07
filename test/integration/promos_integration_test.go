//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/richxcame/ride-hailing/internal/promos"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

const promosServiceKey = "promos"

func startPromosService() *serviceInstance {
	repo := promos.NewRepository(dbPool)
	service := promos.NewService(repo)
	handler := promos.NewHandler(service)

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.CorrelationID())

	router.GET("/healthz", handler.HealthCheck)

	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddleware("integration-secret"))
	{
		// Promo code endpoints
		api.POST("/promos/validate", handler.ValidatePromoCode)
		api.GET("/ride-types", handler.GetRideTypes)
		api.POST("/calculate-fare", handler.CalculateFare)

		// Referral endpoints
		api.GET("/referrals/my-code", handler.GetMyReferralCode)
		api.POST("/referrals/apply", handler.ApplyReferralCode)

		// Admin endpoints
		admin := api.Group("/admin")
		admin.Use(middleware.RequireAdmin())
		{
			admin.POST("/promos", handler.CreatePromoCode)
		}
	}

	server := httptest.NewServer(router)
	return &serviceInstance{server: server, client: server.Client(), baseURL: server.URL}
}

func TestPromosIntegration_PromoCodeValidation(t *testing.T) {
	truncateTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	rider := registerAndLogin(t, models.RoleRider)

	// Create a percentage promo code
	now := time.Now()
	validUntil := now.Add(30 * 24 * time.Hour)
	maxDiscountAmount := 20.0
	maxUses := 100

	createPromoReq := map[string]interface{}{
		"code":                "SAVE20",
		"description":         "20% off rides",
		"discount_type":       "percentage",
		"discount_value":      20.0,
		"max_discount_amount": maxDiscountAmount,
		"valid_from":          now.Format(time.RFC3339),
		"valid_until":         validUntil.Format(time.RFC3339),
		"max_uses":            maxUses,
		"uses_per_user":       3,
		"is_active":           true,
	}

	createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
	require.True(t, createResp.Success)

	// Validate promo code with valid ride amount
	validateReq := map[string]interface{}{
		"code":        "SAVE20",
		"ride_amount": 50.0,
	}

	type promoValidation struct {
		Valid          bool    `json:"valid"`
		DiscountAmount float64 `json:"discount_amount"`
		FinalAmount    float64 `json:"final_amount"`
		Message        string  `json:"message"`
	}

	validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
	require.True(t, validateResp.Success)
	require.True(t, validateResp.Data.Valid)
	require.InEpsilon(t, 10.0, validateResp.Data.DiscountAmount, 1e-6) // 20% of $50 = $10
	require.InEpsilon(t, 40.0, validateResp.Data.FinalAmount, 1e-6)   // $50 - $10 = $40

	// Test max discount cap with higher ride amount
	validateReq2 := map[string]interface{}{
		"code":        "SAVE20",
		"ride_amount": 200.0,
	}

	validateResp2 := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq2, authHeaders(rider.Token))
	require.True(t, validateResp2.Success)
	require.True(t, validateResp2.Data.Valid)
	require.InEpsilon(t, 20.0, validateResp2.Data.DiscountAmount, 1e-6) // Capped at max discount
	require.InEpsilon(t, 180.0, validateResp2.Data.FinalAmount, 1e-6)

	// Test invalid promo code
	validateReq3 := map[string]interface{}{
		"code":        "INVALID",
		"ride_amount": 50.0,
	}

	validateResp3 := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq3, authHeaders(rider.Token))
	require.True(t, validateResp3.Success)
	require.False(t, validateResp3.Data.Valid)
	require.Equal(t, "Invalid promo code", validateResp3.Data.Message)
}

func TestPromosIntegration_FixedAmountPromoCode(t *testing.T) {
	truncateTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	rider := registerAndLogin(t, models.RoleRider)

	// Create a fixed amount promo code
	now := time.Now()
	validUntil := now.Add(30 * 24 * time.Hour)
	minRideAmount := 20.0
	maxUses := 50

	createPromoReq := map[string]interface{}{
		"code":            "FLAT15",
		"description":     "$15 off rides",
		"discount_type":   "fixed_amount",
		"discount_value":  15.0,
		"min_ride_amount": minRideAmount,
		"valid_from":      now.Format(time.RFC3339),
		"valid_until":     validUntil.Format(time.RFC3339),
		"max_uses":        maxUses,
		"uses_per_user":   2,
		"is_active":       true,
	}

	createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
	require.True(t, createResp.Success)

	// Validate with valid amount
	validateReq := map[string]interface{}{
		"code":        "FLAT15",
		"ride_amount": 30.0,
	}

	type promoValidation struct {
		Valid          bool    `json:"valid"`
		DiscountAmount float64 `json:"discount_amount"`
		FinalAmount    float64 `json:"final_amount"`
		Message        string  `json:"message"`
	}

	validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
	require.True(t, validateResp.Success)
	require.True(t, validateResp.Data.Valid)
	require.InEpsilon(t, 15.0, validateResp.Data.DiscountAmount, 1e-6)
	require.InEpsilon(t, 15.0, validateResp.Data.FinalAmount, 1e-6)

	// Test with amount below minimum
	validateReq2 := map[string]interface{}{
		"code":        "FLAT15",
		"ride_amount": 10.0,
	}

	validateResp2 := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq2, authHeaders(rider.Token))
	require.True(t, validateResp2.Success)
	require.False(t, validateResp2.Data.Valid)
	require.Contains(t, validateResp2.Data.Message, "Minimum ride amount")
}

func TestPromosIntegration_ExpiredPromoCode(t *testing.T) {
	truncateTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	rider := registerAndLogin(t, models.RoleRider)

	// Create an expired promo code
	validFrom := time.Now().Add(-7 * 24 * time.Hour)
	validUntil := time.Now().Add(-24 * time.Hour)

	createPromoReq := map[string]interface{}{
		"code":           "EXPIRED",
		"description":    "Expired promo",
		"discount_type":  "percentage",
		"discount_value": 25.0,
		"valid_from":     validFrom.Format(time.RFC3339),
		"valid_until":    validUntil.Format(time.RFC3339),
		"max_uses":       100,
		"uses_per_user":  1,
		"is_active":      true,
	}

	createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
	require.True(t, createResp.Success)

	// Try to validate expired promo
	validateReq := map[string]interface{}{
		"code":        "EXPIRED",
		"ride_amount": 50.0,
	}

	type promoValidation struct {
		Valid   bool   `json:"valid"`
		Message string `json:"message"`
	}

	validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
	require.True(t, validateResp.Success)
	require.False(t, validateResp.Data.Valid)
	require.Equal(t, "This promo code has expired", validateResp.Data.Message)
}

func TestPromosIntegration_ReferralCodeGeneration(t *testing.T) {
	truncateTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	rider := registerAndLogin(t, models.RoleRider)

	// Generate referral code
	type referralCodeResponse struct {
		ID             string  `json:"id"`
		UserID         string  `json:"user_id"`
		Code           string  `json:"code"`
		TotalReferrals int     `json:"total_referrals"`
		TotalEarnings  float64 `json:"total_earnings"`
	}

	referralResp := doRequest[referralCodeResponse](t, promosServiceKey, http.MethodGet, "/api/v1/referrals/my-code", nil, authHeaders(rider.Token))
	require.True(t, referralResp.Success)
	require.NotEmpty(t, referralResp.Data.Code)
	require.Equal(t, rider.User.ID.String(), referralResp.Data.UserID)
	require.Equal(t, 0, referralResp.Data.TotalReferrals)

	// Request again should return same code
	referralResp2 := doRequest[referralCodeResponse](t, promosServiceKey, http.MethodGet, "/api/v1/referrals/my-code", nil, authHeaders(rider.Token))
	require.True(t, referralResp2.Success)
	require.Equal(t, referralResp.Data.Code, referralResp2.Data.Code)
	require.Equal(t, referralResp.Data.ID, referralResp2.Data.ID)
}

func TestPromosIntegration_ApplyReferralCode(t *testing.T) {
	truncateTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	referrer := registerAndLogin(t, models.RoleRider)
	newUser := registerAndLogin(t, models.RoleRider)

	// Get referrer's referral code
	type referralCodeResponse struct {
		Code string `json:"code"`
	}

	referralResp := doRequest[referralCodeResponse](t, promosServiceKey, http.MethodGet, "/api/v1/referrals/my-code", nil, authHeaders(referrer.Token))
	require.True(t, referralResp.Success)
	referralCode := referralResp.Data.Code

	// Apply referral code as new user
	applyReq := map[string]interface{}{
		"referral_code": referralCode,
	}

	type applyResponse struct {
		Message string  `json:"message"`
		Bonus   float64 `json:"bonus"`
	}

	applyResp := doRequest[applyResponse](t, promosServiceKey, http.MethodPost, "/api/v1/referrals/apply", applyReq, authHeaders(newUser.Token))
	require.True(t, applyResp.Success)
	require.Contains(t, applyResp.Data.Message, "successfully")
	require.Equal(t, 10.0, applyResp.Data.Bonus)

	// Verify referral relationship in database
	var referralCount int
	err := dbPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM referrals WHERE referrer_id = $1 AND referred_id = $2",
		referrer.User.ID, newUser.User.ID).Scan(&referralCount)
	require.NoError(t, err)
	require.Equal(t, 1, referralCount)

	// Try to use own referral code (should fail)
	applyReq2 := map[string]interface{}{
		"referral_code": referralCode,
	}

	applyResp2 := doRawRequest(t, promosServiceKey, http.MethodPost, "/api/v1/referrals/apply", applyReq2, authHeaders(referrer.Token))
	defer applyResp2.Body.Close()
	require.Equal(t, http.StatusBadRequest, applyResp2.StatusCode)
}

func TestPromosIntegration_RideTypesAndFareCalculation(t *testing.T) {
	truncateTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	rider := registerAndLogin(t, models.RoleRider)

	// Insert ride types if not exist
	_, err := dbPool.Exec(context.Background(), `
		INSERT INTO ride_types (id, name, description, base_fare, per_km_rate, per_minute_rate, minimum_fare, capacity)
		VALUES
			($1, 'Economy', 'Budget-friendly rides', 2.50, 1.20, 0.25, 5.00, 4),
			($2, 'Premium', 'Comfortable premium rides', 5.00, 2.00, 0.40, 10.00, 4),
			($3, 'XL', 'Rides for larger groups', 4.00, 1.80, 0.35, 8.00, 6)
		ON CONFLICT (id) DO NOTHING`,
		uuid.New(), uuid.New(), uuid.New())
	require.NoError(t, err)

	// Get all ride types
	type rideTypesResponse struct {
		RideTypes []map[string]interface{} `json:"ride_types"`
	}

	rideTypesResp := doRequest[rideTypesResponse](t, promosServiceKey, http.MethodGet, "/api/v1/ride-types", nil, authHeaders(rider.Token))
	require.True(t, rideTypesResp.Success)
	require.GreaterOrEqual(t, len(rideTypesResp.Data.RideTypes), 3)

	// Get first ride type ID
	rideTypeID := rideTypesResp.Data.RideTypes[0]["id"].(string)

	// Calculate fare
	fareReq := map[string]interface{}{
		"ride_type_id":      rideTypeID,
		"distance":          10.5,
		"duration":          20,
		"surge_multiplier":  1.0,
	}

	type fareResponse struct {
		Fare            float64 `json:"fare"`
		Distance        float64 `json:"distance"`
		Duration        int     `json:"duration"`
		SurgeMultiplier float64 `json:"surge_multiplier"`
	}

	fareResp := doRequest[fareResponse](t, promosServiceKey, http.MethodPost, "/api/v1/calculate-fare", fareReq, authHeaders(rider.Token))
	require.True(t, fareResp.Success)
	require.Greater(t, fareResp.Data.Fare, 0.0)
	require.Equal(t, 10.5, fareResp.Data.Distance)
	require.Equal(t, 20, fareResp.Data.Duration)

	// Calculate fare with surge pricing
	fareReq2 := map[string]interface{}{
		"ride_type_id":      rideTypeID,
		"distance":          10.5,
		"duration":          20,
		"surge_multiplier":  1.5,
	}

	fareResp2 := doRequest[fareResponse](t, promosServiceKey, http.MethodPost, "/api/v1/calculate-fare", fareReq2, authHeaders(rider.Token))
	require.True(t, fareResp2.Success)
	require.Greater(t, fareResp2.Data.Fare, fareResp.Data.Fare) // Should be higher with surge
	require.Equal(t, 1.5, fareResp2.Data.SurgeMultiplier)
}

func TestPromosIntegration_PromoCodeUsageLimit(t *testing.T) {
	truncateTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	rider := registerAndLogin(t, models.RoleRider)

	// Create promo with usage limit of 1 per user
	now := time.Now()
	validUntil := now.Add(30 * 24 * time.Hour)

	createPromoReq := map[string]interface{}{
		"code":           "ONETIME",
		"description":    "One-time use promo",
		"discount_type":  "percentage",
		"discount_value": 50.0,
		"valid_from":     now.Format(time.RFC3339),
		"valid_until":    validUntil.Format(time.RFC3339),
		"max_uses":       100,
		"uses_per_user":  1,
		"is_active":      true,
	}

	createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
	require.True(t, createResp.Success)

	// Get promo code ID
	var promoCodeID uuid.UUID
	err := dbPool.QueryRow(context.Background(), "SELECT id FROM promo_codes WHERE code = 'ONETIME'").Scan(&promoCodeID)
	require.NoError(t, err)

	// First validation should succeed
	validateReq := map[string]interface{}{
		"code":        "ONETIME",
		"ride_amount": 50.0,
	}

	type promoValidation struct {
		Valid          bool    `json:"valid"`
		DiscountAmount float64 `json:"discount_amount"`
		FinalAmount    float64 `json:"final_amount"`
		Message        string  `json:"message"`
	}

	validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
	require.True(t, validateResp.Success)
	require.True(t, validateResp.Data.Valid)

	// Simulate promo code usage by inserting into promo_code_uses
	rideID := uuid.New()
	_, err = dbPool.Exec(context.Background(), `
		INSERT INTO promo_code_uses (promo_code_id, user_id, ride_id, discount_amount, original_amount, final_amount)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		promoCodeID, rider.User.ID, rideID, 25.0, 50.0, 25.0)
	require.NoError(t, err)

	// Second validation should fail (exceeded per-user limit)
	validateResp2 := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
	require.True(t, validateResp2.Success)
	require.False(t, validateResp2.Data.Valid)
	require.Contains(t, validateResp2.Data.Message, "already used")
}
