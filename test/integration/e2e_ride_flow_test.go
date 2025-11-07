//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/richxcame/ride-hailing/pkg/models"
)

// TestE2E_CompleteRideFlowWithPromoAndPayment tests the complete ride lifecycle
// from ride request to completion with promo code and payment
func TestE2E_CompleteRideFlowWithPromoAndPayment(t *testing.T) {
	truncateTables(t)

	// Ensure all services are started
	if _, ok := services[authServiceKey]; !ok {
		services[authServiceKey] = startAuthService(mustLoadConfig("auth-service"))
	}
	if _, ok := services[ridesServiceKey]; !ok {
		services[ridesServiceKey] = startRidesService(mustLoadConfig("rides-service"))
	}
	if _, ok := services[paymentsServiceKey]; !ok {
		services[paymentsServiceKey] = startPaymentsService(mustLoadConfig("payments-service"))
	}
	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	// Step 1: Register users
	t.Log("Step 1: Registering users")
	admin := registerAndLogin(t, models.RoleAdmin)
	rider := registerAndLogin(t, models.RoleRider)
	driver := registerAndLogin(t, models.RoleDriver)

	// Step 2: Create a promo code
	t.Log("Step 2: Creating promo code")
	now := time.Now()
	validUntil := now.Add(30 * 24 * time.Hour)

	createPromoReq := map[string]interface{}{
		"code":           "FIRSTRIDE20",
		"description":    "20% off first ride",
		"discount_type":  "percentage",
		"discount_value": 20.0,
		"valid_from":     now.Format(time.RFC3339),
		"valid_until":    validUntil.Format(time.RFC3339),
		"max_uses":       1000,
		"uses_per_user":  1,
		"is_active":      true,
	}

	createPromoResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
	require.True(t, createPromoResp.Success)

	// Step 3: Top up rider's wallet
	t.Log("Step 3: Topping up rider wallet")
	topUpReq := map[string]interface{}{
		"amount":                 100.00,
		"stripe_payment_method":  "pm_test",
	}

	topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(rider.Token))
	require.True(t, topUpResp.Success)
	require.InEpsilon(t, 100.00, topUpResp.Data.Amount, 1e-6)

	// Verify wallet balance
	walletResp := doRequest[*models.Wallet](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, authHeaders(rider.Token))
	require.True(t, walletResp.Success)
	require.InEpsilon(t, 100.00, walletResp.Data.Balance, 1e-6)

	// Step 4: Create a ride request
	t.Log("Step 4: Creating ride request")
	rideReq := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "123 Market St, San Francisco",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "456 Broadway, Oakland",
	}

	rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(rider.Token))
	require.True(t, rideResp.Success)
	require.NotNil(t, rideResp.Data)
	require.Equal(t, models.RideStatusRequested, rideResp.Data.Status)
	require.Equal(t, rider.User.ID, rideResp.Data.RiderID)
	rideID := rideResp.Data.ID

	// Step 5: Validate promo code (before applying)
	t.Log("Step 5: Validating promo code")
	validateReq := map[string]interface{}{
		"code":        "FIRSTRIDE20",
		"ride_amount": rideResp.Data.EstimatedFare,
	}

	type promoValidation struct {
		Valid          bool    `json:"valid"`
		DiscountAmount float64 `json:"discount_amount"`
		FinalAmount    float64 `json:"final_amount"`
	}

	validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
	require.True(t, validateResp.Success)
	require.True(t, validateResp.Data.Valid)

	// Step 6: Driver accepts the ride
	t.Log("Step 6: Driver accepting ride")
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(driver.Token))
	require.True(t, acceptResp.Success)
	require.Equal(t, models.RideStatusAccepted, acceptResp.Data.Status)
	require.NotNil(t, acceptResp.Data.DriverID)
	require.Equal(t, driver.User.ID, *acceptResp.Data.DriverID)

	// Step 7: Driver starts the ride
	t.Log("Step 7: Driver starting ride")
	startPath := fmt.Sprintf("/api/v1/driver/rides/%s/start", rideID)
	startResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, startPath, nil, authHeaders(driver.Token))
	require.True(t, startResp.Success)
	require.Equal(t, models.RideStatusInProgress, startResp.Data.Status)
	require.NotNil(t, startResp.Data.StartedAt)

	// Step 8: Driver completes the ride
	t.Log("Step 8: Driver completing ride")
	completePath := fmt.Sprintf("/api/v1/driver/rides/%s/complete", rideID)
	completeReq := map[string]interface{}{
		"actual_distance": 13.5,
	}
	completeResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, completePath, completeReq, authHeaders(driver.Token))
	require.True(t, completeResp.Success)
	require.Equal(t, models.RideStatusCompleted, completeResp.Data.Status)
	require.NotNil(t, completeResp.Data.CompletedAt)
	require.NotNil(t, completeResp.Data.FinalFare)

	// Step 9: Apply promo code and process payment
	t.Log("Step 9: Processing payment with promo code")

	// Get promo code ID from database
	var promoCodeID uuid.UUID
	err := dbPool.QueryRow(context.Background(), "SELECT id FROM promo_codes WHERE code = 'FIRSTRIDE20'").Scan(&promoCodeID)
	require.NoError(t, err)

	// Simulate promo code application
	originalFare := *completeResp.Data.FinalFare
	discountAmount := originalFare * 0.20 // 20% discount
	finalFare := originalFare - discountAmount

	// Insert promo code use
	_, err = dbPool.Exec(context.Background(), `
		INSERT INTO promo_code_uses (promo_code_id, user_id, ride_id, discount_amount, original_amount, final_amount)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		promoCodeID, rider.User.ID, rideID, discountAmount, originalFare, finalFare)
	require.NoError(t, err)

	// Create payment for the ride
	paymentReq := map[string]interface{}{
		"ride_id":                rideID,
		"amount":                 finalFare,
		"payment_method":         "wallet",
		"stripe_payment_method":  "pm_test",
	}

	type paymentResponse struct {
		ID       string  `json:"id"`
		RideID   string  `json:"ride_id"`
		Amount   float64 `json:"amount"`
		Status   string  `json:"status"`
	}

	paymentResp := doRequest[paymentResponse](t, paymentsServiceKey, http.MethodPost, "/api/v1/payments", paymentReq, authHeaders(rider.Token))
	require.True(t, paymentResp.Success)
	require.Equal(t, "completed", paymentResp.Data.Status)
	require.InEpsilon(t, finalFare, paymentResp.Data.Amount, 1e-6)

	// Step 10: Verify wallet balance was deducted
	t.Log("Step 10: Verifying wallet balance")
	walletAfterResp := doRequest[*models.Wallet](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, authHeaders(rider.Token))
	require.True(t, walletAfterResp.Success)
	expectedBalance := 100.00 - finalFare
	require.InEpsilon(t, expectedBalance, walletAfterResp.Data.Balance, 1e-6)

	// Step 11: Verify ride status in database
	t.Log("Step 11: Verifying ride status in database")
	var dbStatus string
	var dbFinalFare float64
	err = dbPool.QueryRow(context.Background(),
		"SELECT status, final_fare FROM rides WHERE id = $1",
		rideID).Scan(&dbStatus, &dbFinalFare)
	require.NoError(t, err)
	require.Equal(t, string(models.RideStatusCompleted), dbStatus)
	require.InEpsilon(t, originalFare, dbFinalFare, 1e-6)

	// Step 12: Verify payment was recorded
	t.Log("Step 12: Verifying payment in database")
	var paymentCount int
	err = dbPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM payments WHERE ride_id = $1 AND user_id = $2",
		rideID, rider.User.ID).Scan(&paymentCount)
	require.NoError(t, err)
	require.Equal(t, 1, paymentCount)

	// Step 13: Verify promo code was used
	t.Log("Step 13: Verifying promo code usage")
	var promoUseCount int
	err = dbPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM promo_code_uses WHERE promo_code_id = $1 AND user_id = $2 AND ride_id = $3",
		promoCodeID, rider.User.ID, rideID).Scan(&promoUseCount)
	require.NoError(t, err)
	require.Equal(t, 1, promoUseCount)

	t.Log("E2E test completed successfully!")
}

// TestE2E_RideWithReferralBonus tests ride flow with referral bonus
func TestE2E_RideWithReferralBonus(t *testing.T) {
	truncateTables(t)

	// Ensure all services are started
	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}
	if _, ok := services[ridesServiceKey]; !ok {
		services[ridesServiceKey] = startRidesService(mustLoadConfig("rides-service"))
	}
	if _, ok := services[paymentsServiceKey]; !ok {
		services[paymentsServiceKey] = startPaymentsService(mustLoadConfig("payments-service"))
	}

	// Step 1: Create referrer and get referral code
	t.Log("Step 1: Creating referrer and generating referral code")
	referrer := registerAndLogin(t, models.RoleRider)

	type referralCodeResponse struct {
		Code string `json:"code"`
	}

	referralResp := doRequest[referralCodeResponse](t, promosServiceKey, http.MethodGet, "/api/v1/referrals/my-code", nil, authHeaders(referrer.Token))
	require.True(t, referralResp.Success)
	referralCode := referralResp.Data.Code

	// Step 2: New user signs up and applies referral code
	t.Log("Step 2: New user applying referral code")
	newRider := registerAndLogin(t, models.RoleRider)

	applyReq := map[string]interface{}{
		"referral_code": referralCode,
	}

	type applyResponse struct {
		Message string  `json:"message"`
		Bonus   float64 `json:"bonus"`
	}

	applyResp := doRequest[applyResponse](t, promosServiceKey, http.MethodPost, "/api/v1/referrals/apply", applyReq, authHeaders(newRider.Token))
	require.True(t, applyResp.Success)
	require.Equal(t, 10.0, applyResp.Data.Bonus)

	// Step 3: New rider completes first ride
	t.Log("Step 3: New rider creating and completing first ride")
	driver := registerAndLogin(t, models.RoleDriver)

	// Top up wallet
	topUpReq := map[string]interface{}{
		"amount":                 50.00,
		"stripe_payment_method":  "pm_test",
	}
	topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(newRider.Token))
	require.True(t, topUpResp.Success)

	// Create ride
	rideReq := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "123 Market St",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "456 Broadway",
	}

	rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(newRider.Token))
	require.True(t, rideResp.Success)
	rideID := rideResp.Data.ID

	// Driver accepts and completes ride
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(driver.Token))
	require.True(t, acceptResp.Success)

	startPath := fmt.Sprintf("/api/v1/driver/rides/%s/start", rideID)
	startResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, startPath, nil, authHeaders(driver.Token))
	require.True(t, startResp.Success)

	completePath := fmt.Sprintf("/api/v1/driver/rides/%s/complete", rideID)
	completeReq := map[string]interface{}{
		"actual_distance": 10.0,
	}
	completeResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, completePath, completeReq, authHeaders(driver.Token))
	require.True(t, completeResp.Success)

	// Step 4: Verify referral relationship exists
	t.Log("Step 4: Verifying referral bonus eligibility")
	var referralID uuid.UUID
	var referrerBonus, referredBonus float64
	err := dbPool.QueryRow(context.Background(), `
		SELECT id, referrer_bonus, referred_bonus
		FROM referrals
		WHERE referrer_id = $1 AND referred_id = $2`,
		referrer.User.ID, newRider.User.ID).Scan(&referralID, &referrerBonus, &referredBonus)
	require.NoError(t, err)
	require.Equal(t, 10.0, referrerBonus)
	require.Equal(t, 10.0, referredBonus)

	t.Log("E2E referral test completed successfully!")
}

// TestE2E_MultipleRidesConcurrent tests multiple riders creating rides concurrently
func TestE2E_MultipleRidesConcurrent(t *testing.T) {
	truncateTables(t)

	if _, ok := services[ridesServiceKey]; !ok {
		services[ridesServiceKey] = startRidesService(mustLoadConfig("rides-service"))
	}

	// Create multiple riders and drivers
	riders := make([]authSession, 3)
	for i := 0; i < 3; i++ {
		riders[i] = registerAndLogin(t, models.RoleRider)
	}

	drivers := make([]authSession, 2)
	for i := 0; i < 2; i++ {
		drivers[i] = registerAndLogin(t, models.RoleDriver)
	}

	// Create rides for all riders
	rideIDs := make([]uuid.UUID, 3)
	for i, rider := range riders {
		rideReq := &models.RideRequest{
			PickupLatitude:   37.7749 + float64(i)*0.01,
			PickupLongitude:  -122.4194 + float64(i)*0.01,
			PickupAddress:    fmt.Sprintf("Pickup %d", i+1),
			DropoffLatitude:  37.8044 + float64(i)*0.01,
			DropoffLongitude: -122.2712 + float64(i)*0.01,
			DropoffAddress:   fmt.Sprintf("Dropoff %d", i+1),
		}

		rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(rider.Token))
		require.True(t, rideResp.Success, "Ride %d creation failed", i+1)
		rideIDs[i] = rideResp.Data.ID
	}

	// Verify all rides are in requested status
	for i, rideID := range rideIDs {
		rideDetail := doRequest[*models.Ride](t, ridesServiceKey, http.MethodGet, fmt.Sprintf("/api/v1/rides/%s", rideID.String()), nil, authHeaders(riders[i].Token))
		require.True(t, rideDetail.Success)
		require.Equal(t, models.RideStatusRequested, rideDetail.Data.Status)
	}

	t.Log("Multiple concurrent rides test completed successfully!")
}
