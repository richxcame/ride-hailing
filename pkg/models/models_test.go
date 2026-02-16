package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ==================== User Tests ====================

// TestUserRole constants
func TestUserRole_Constants(t *testing.T) {
	tests := []struct {
		name     string
		role     UserRole
		expected string
	}{
		{"rider role", RoleRider, "rider"},
		{"driver role", RoleDriver, "driver"},
		{"admin role", RoleAdmin, "admin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.role) != tt.expected {
				t.Errorf("Role = %s, want %s", string(tt.role), tt.expected)
			}
		})
	}
}

// TestUser_JSON_Marshaling tests User JSON marshaling
func TestUser_JSON_Marshaling(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	profileImage := "https://example.com/image.jpg"

	user := User{
		ID:           userID,
		Email:        "test@example.com",
		PhoneNumber:  "+1234567890",
		PasswordHash: "hashed_password",
		FirstName:    "John",
		LastName:     "Doe",
		Role:         RoleRider,
		IsActive:     true,
		IsVerified:   true,
		ProfileImage: &profileImage,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}

	// Password should not be in JSON output
	if jsonContains(data, "hashed_password") {
		t.Error("PasswordHash should not be in JSON output")
	}

	// Other fields should be present
	if !jsonContains(data, "test@example.com") {
		t.Error("Email should be in JSON output")
	}
	if !jsonContains(data, "John") {
		t.Error("FirstName should be in JSON output")
	}
}

// TestUser_JSON_Unmarshaling tests User JSON unmarshaling
func TestUser_JSON_Unmarshaling(t *testing.T) {
	jsonData := `{
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"email": "test@example.com",
		"phone_number": "+1234567890",
		"first_name": "John",
		"last_name": "Doe",
		"role": "rider",
		"is_active": true,
		"is_verified": false
	}`

	var user User
	err := json.Unmarshal([]byte(jsonData), &user)
	if err != nil {
		t.Fatalf("Failed to unmarshal user: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", user.Email)
	}
	if user.Role != RoleRider {
		t.Errorf("Role = %s, want rider", user.Role)
	}
	if user.IsActive != true {
		t.Error("IsActive should be true")
	}
	if user.IsVerified != false {
		t.Error("IsVerified should be false")
	}
}

// TestRegisterRequest_Validation tests RegisterRequest structure
func TestRegisterRequest_Fields(t *testing.T) {
	req := RegisterRequest{
		Email:       "test@example.com",
		Password:    "password123",
		PhoneNumber: "+1234567890",
		FirstName:   "John",
		LastName:    "Doe",
		Role:        RoleDriver,
	}

	if req.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", req.Email)
	}
	if req.Role != RoleDriver {
		t.Errorf("Role = %s, want driver", req.Role)
	}
}

// TestLoginRequest_Fields tests LoginRequest structure
func TestLoginRequest_Fields(t *testing.T) {
	req := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	if req.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", req.Email)
	}
	if req.Password != "password123" {
		t.Errorf("Password = %s, want password123", req.Password)
	}
}

// TestLoginResponse_Fields tests LoginResponse structure
func TestLoginResponse_Fields(t *testing.T) {
	user := &User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	resp := LoginResponse{
		User:  user,
		Token: "jwt-token-here",
	}

	if resp.User.Email != "test@example.com" {
		t.Error("User email mismatch")
	}
	if resp.Token != "jwt-token-here" {
		t.Error("Token mismatch")
	}
}

// TestDriver_Fields tests Driver structure
func TestDriver_Fields(t *testing.T) {
	driverID := uuid.New()
	userID := uuid.New()
	latitude := 40.7128
	longitude := -74.0060
	now := time.Now()

	driver := Driver{
		ID:                 driverID,
		UserID:             userID,
		LicenseNumber:      "DL12345678",
		VehicleModel:       "Toyota Camry",
		VehiclePlate:       "ABC123",
		VehicleColor:       "Black",
		VehicleYear:        2022,
		IsAvailable:        true,
		IsOnline:           true,
		Rating:             4.8,
		TotalRides:         150,
		CurrentLatitude:    &latitude,
		CurrentLongitude:   &longitude,
		LastLocationUpdate: &now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if driver.LicenseNumber != "DL12345678" {
		t.Errorf("LicenseNumber = %s, want DL12345678", driver.LicenseNumber)
	}
	if driver.Rating != 4.8 {
		t.Errorf("Rating = %f, want 4.8", driver.Rating)
	}
	if *driver.CurrentLatitude != 40.7128 {
		t.Errorf("CurrentLatitude = %f, want 40.7128", *driver.CurrentLatitude)
	}
}

// ==================== Ride Tests ====================

// TestRideStatus_Constants tests ride status constants
func TestRideStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   RideStatus
		expected string
	}{
		{"requested", RideStatusRequested, "requested"},
		{"accepted", RideStatusAccepted, "accepted"},
		{"in_progress", RideStatusInProgress, "in_progress"},
		{"completed", RideStatusCompleted, "completed"},
		{"cancelled", RideStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Status = %s, want %s", string(tt.status), tt.expected)
			}
		})
	}
}

// TestRide_JSON_Marshaling tests Ride JSON marshaling
func TestRide_JSON_Marshaling(t *testing.T) {
	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	ride := Ride{
		ID:                rideID,
		RiderID:           riderID,
		DriverID:          &driverID,
		Status:            RideStatusInProgress,
		PickupLatitude:    40.7128,
		PickupLongitude:   -74.0060,
		PickupAddress:     "123 Main St",
		DropoffLatitude:   40.7580,
		DropoffLongitude:  -73.9855,
		DropoffAddress:    "456 Broadway",
		EstimatedDistance: 5.5,
		EstimatedDuration: 15,
		EstimatedFare:     25.50,
		SurgeMultiplier:   1.0,
		RequestedAt:       now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	data, err := json.Marshal(ride)
	if err != nil {
		t.Fatalf("Failed to marshal ride: %v", err)
	}

	if !jsonContains(data, "in_progress") {
		t.Error("Status should be in JSON output")
	}
	if !jsonContains(data, "123 Main St") {
		t.Error("PickupAddress should be in JSON output")
	}
}

// TestRide_JSON_Unmarshaling tests Ride JSON unmarshaling
func TestRide_JSON_Unmarshaling(t *testing.T) {
	jsonData := `{
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"rider_id": "660e8400-e29b-41d4-a716-446655440000",
		"status": "requested",
		"pickup_latitude": 40.7128,
		"pickup_longitude": -74.0060,
		"pickup_address": "123 Main St",
		"dropoff_latitude": 40.7580,
		"dropoff_longitude": -73.9855,
		"dropoff_address": "456 Broadway",
		"estimated_distance": 5.5,
		"estimated_duration": 15,
		"estimated_fare": 25.50,
		"surge_multiplier": 1.5
	}`

	var ride Ride
	err := json.Unmarshal([]byte(jsonData), &ride)
	if err != nil {
		t.Fatalf("Failed to unmarshal ride: %v", err)
	}

	if ride.Status != RideStatusRequested {
		t.Errorf("Status = %s, want requested", ride.Status)
	}
	if ride.PickupLatitude != 40.7128 {
		t.Errorf("PickupLatitude = %f, want 40.7128", ride.PickupLatitude)
	}
	if ride.SurgeMultiplier != 1.5 {
		t.Errorf("SurgeMultiplier = %f, want 1.5", ride.SurgeMultiplier)
	}
}

// TestRideRequest_Fields tests RideRequest structure
func TestRideRequest_Fields(t *testing.T) {
	rideTypeID := uuid.New()
	scheduledAt := time.Now().Add(24 * time.Hour)

	req := RideRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		PickupAddress:    "123 Main St",
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		DropoffAddress:   "456 Broadway",
		RideTypeID:       &rideTypeID,
		PromoCode:        "SAVE10",
		ScheduledAt:      &scheduledAt,
		IsScheduled:      true,
	}

	if req.PickupAddress != "123 Main St" {
		t.Errorf("PickupAddress = %s, want 123 Main St", req.PickupAddress)
	}
	if req.PromoCode != "SAVE10" {
		t.Errorf("PromoCode = %s, want SAVE10", req.PromoCode)
	}
	if !req.IsScheduled {
		t.Error("IsScheduled should be true")
	}
}

// TestRideResponse_Fields tests RideResponse structure
func TestRideResponse_Fields(t *testing.T) {
	ride := &Ride{
		ID:     uuid.New(),
		Status: RideStatusCompleted,
	}
	rider := &User{
		ID:        uuid.New(),
		FirstName: "John",
	}
	driver := &Driver{
		ID:           uuid.New(),
		VehicleModel: "Tesla Model 3",
	}

	resp := RideResponse{
		Ride:   ride,
		Rider:  rider,
		Driver: driver,
	}

	if resp.Status != RideStatusCompleted {
		t.Error("Status mismatch")
	}
	if resp.Rider.FirstName != "John" {
		t.Error("Rider first name mismatch")
	}
	if resp.Driver.VehicleModel != "Tesla Model 3" {
		t.Error("Driver vehicle model mismatch")
	}
}

// TestRideUpdateRequest_Fields tests RideUpdateRequest structure
func TestRideUpdateRequest_Fields(t *testing.T) {
	reason := "Driver not available"
	req := RideUpdateRequest{
		Status:             RideStatusCancelled,
		CancellationReason: &reason,
	}

	if req.Status != RideStatusCancelled {
		t.Errorf("Status = %s, want cancelled", req.Status)
	}
	if *req.CancellationReason != "Driver not available" {
		t.Error("CancellationReason mismatch")
	}
}

// TestRideRatingRequest_Fields tests RideRatingRequest structure
func TestRideRatingRequest_Fields(t *testing.T) {
	feedback := "Great ride!"
	req := RideRatingRequest{
		Rating:   5,
		Feedback: &feedback,
	}

	if req.Rating != 5 {
		t.Errorf("Rating = %d, want 5", req.Rating)
	}
	if *req.Feedback != "Great ride!" {
		t.Error("Feedback mismatch")
	}
}

// ==================== Payment Tests ====================

// TestPaymentStatus_Constants tests payment status constants
func TestPaymentStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   PaymentStatus
		expected string
	}{
		{"pending", PaymentStatusPending, "pending"},
		{"completed", PaymentStatusCompleted, "completed"},
		{"failed", PaymentStatusFailed, "failed"},
		{"refunded", PaymentStatusRefunded, "refunded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Status = %s, want %s", string(tt.status), tt.expected)
			}
		})
	}
}

// TestPaymentMethod_Constants tests payment method constants
func TestPaymentMethod_Constants(t *testing.T) {
	tests := []struct {
		name     string
		method   PaymentMethod
		expected string
	}{
		{"card", PaymentMethodCard, "card"},
		{"wallet", PaymentMethodWallet, "wallet"},
		{"cash", PaymentMethodCash, "cash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.method) != tt.expected {
				t.Errorf("Method = %s, want %s", string(tt.method), tt.expected)
			}
		})
	}
}

// TestPayment_JSON_Marshaling tests Payment JSON marshaling
func TestPayment_JSON_Marshaling(t *testing.T) {
	paymentID := uuid.New()
	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	stripePaymentID := "pi_123456"
	now := time.Now().UTC().Truncate(time.Second)

	payment := Payment{
		ID:              paymentID,
		RideID:          rideID,
		RiderID:         riderID,
		DriverID:        driverID,
		Amount:          25.50,
		Currency:        "USD",
		PaymentMethod:   "card",
		Status:          "completed",
		StripePaymentID: &stripePaymentID,
		Commission:      2.55,
		DriverEarnings:  22.95,
		Method:          PaymentMethodCard,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	data, err := json.Marshal(payment)
	if err != nil {
		t.Fatalf("Failed to marshal payment: %v", err)
	}

	if !jsonContains(data, "25.5") {
		t.Error("Amount should be in JSON output")
	}
	if !jsonContains(data, "USD") {
		t.Error("Currency should be in JSON output")
	}
}

// TestPayment_JSON_Unmarshaling tests Payment JSON unmarshaling
func TestPayment_JSON_Unmarshaling(t *testing.T) {
	jsonData := `{
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"ride_id": "660e8400-e29b-41d4-a716-446655440000",
		"rider_id": "770e8400-e29b-41d4-a716-446655440000",
		"driver_id": "880e8400-e29b-41d4-a716-446655440000",
		"amount": 30.00,
		"currency": "EUR",
		"payment_method": "wallet",
		"status": "pending",
		"commission": 3.00,
		"driver_earnings": 27.00
	}`

	var payment Payment
	err := json.Unmarshal([]byte(jsonData), &payment)
	if err != nil {
		t.Fatalf("Failed to unmarshal payment: %v", err)
	}

	if payment.Amount != 30.00 {
		t.Errorf("Amount = %f, want 30.00", payment.Amount)
	}
	if payment.Currency != "EUR" {
		t.Errorf("Currency = %s, want EUR", payment.Currency)
	}
	if payment.Commission != 3.00 {
		t.Errorf("Commission = %f, want 3.00", payment.Commission)
	}
}

// TestWallet_Fields tests Wallet structure
func TestWallet_Fields(t *testing.T) {
	walletID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	wallet := Wallet{
		ID:        walletID,
		UserID:    userID,
		Balance:   100.50,
		Currency:  "USD",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if wallet.Balance != 100.50 {
		t.Errorf("Balance = %f, want 100.50", wallet.Balance)
	}
	if !wallet.IsActive {
		t.Error("IsActive should be true")
	}
}

// TestWalletTransaction_Fields tests WalletTransaction structure
func TestWalletTransaction_Fields(t *testing.T) {
	transactionID := uuid.New()
	walletID := uuid.New()
	referenceID := uuid.New()
	now := time.Now()

	transaction := WalletTransaction{
		ID:            transactionID,
		WalletID:      walletID,
		Type:          "credit",
		Amount:        50.00,
		Description:   "Wallet top-up",
		ReferenceType: "topup",
		ReferenceID:   &referenceID,
		BalanceBefore: 100.00,
		BalanceAfter:  150.00,
		CreatedAt:     now,
	}

	if transaction.Type != "credit" {
		t.Errorf("Type = %s, want credit", transaction.Type)
	}
	if transaction.Amount != 50.00 {
		t.Errorf("Amount = %f, want 50.00", transaction.Amount)
	}
	if transaction.BalanceAfter != 150.00 {
		t.Errorf("BalanceAfter = %f, want 150.00", transaction.BalanceAfter)
	}
}

// TestPaymentRequest_Fields tests PaymentRequest structure
func TestPaymentRequest_Fields(t *testing.T) {
	rideID := uuid.New()
	token := "tok_visa"

	req := PaymentRequest{
		RideID: rideID,
		Method: PaymentMethodCard,
		Token:  &token,
	}

	if req.Method != PaymentMethodCard {
		t.Errorf("Method = %s, want card", req.Method)
	}
	if *req.Token != "tok_visa" {
		t.Error("Token mismatch")
	}
}

// ==================== Notification Tests ====================

// TestNotificationType_Constants tests notification type constants
func TestNotificationType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		notifType NotificationType
		expected string
	}{
		{"ride_request", NotificationTypeRideRequest, "ride_request"},
		{"ride_accepted", NotificationTypeRideAccepted, "ride_accepted"},
		{"ride_started", NotificationTypeRideStarted, "ride_started"},
		{"ride_completed", NotificationTypeRideCompleted, "ride_completed"},
		{"ride_cancelled", NotificationTypeRideCancelled, "ride_cancelled"},
		{"payment", NotificationTypePayment, "payment"},
		{"promotion", NotificationTypePromotion, "promotion"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.notifType) != tt.expected {
				t.Errorf("Type = %s, want %s", string(tt.notifType), tt.expected)
			}
		})
	}
}

// TestNotificationChannel_Constants tests notification channel constants
func TestNotificationChannel_Constants(t *testing.T) {
	tests := []struct {
		name     string
		channel  NotificationChannel
		expected string
	}{
		{"push", NotificationChannelPush, "push"},
		{"sms", NotificationChannelSMS, "sms"},
		{"email", NotificationChannelEmail, "email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.channel) != tt.expected {
				t.Errorf("Channel = %s, want %s", string(tt.channel), tt.expected)
			}
		})
	}
}

// TestNotification_JSON_Marshaling tests Notification JSON marshaling
func TestNotification_JSON_Marshaling(t *testing.T) {
	notificationID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	sentAt := now.Add(-5 * time.Minute)

	notification := Notification{
		ID:        notificationID,
		UserID:    userID,
		Type:      "ride_request",
		Channel:   "push",
		Title:     "New Ride Request",
		Body:      "You have a new ride request nearby",
		Data:      map[string]interface{}{"ride_id": "123"},
		Status:    "sent",
		SentAt:    &sentAt,
		IsRead:    false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("Failed to marshal notification: %v", err)
	}

	if !jsonContains(data, "New Ride Request") {
		t.Error("Title should be in JSON output")
	}
	if !jsonContains(data, "ride_request") {
		t.Error("Type should be in JSON output")
	}
}

// TestNotification_JSON_Unmarshaling tests Notification JSON unmarshaling
func TestNotification_JSON_Unmarshaling(t *testing.T) {
	jsonData := `{
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"user_id": "660e8400-e29b-41d4-a716-446655440000",
		"type": "payment",
		"channel": "email",
		"title": "Payment Received",
		"body": "Your payment of $25.00 has been processed",
		"status": "delivered",
		"is_read": true
	}`

	var notification Notification
	err := json.Unmarshal([]byte(jsonData), &notification)
	if err != nil {
		t.Fatalf("Failed to unmarshal notification: %v", err)
	}

	if notification.Type != "payment" {
		t.Errorf("Type = %s, want payment", notification.Type)
	}
	if notification.Channel != "email" {
		t.Errorf("Channel = %s, want email", notification.Channel)
	}
	if !notification.IsRead {
		t.Error("IsRead should be true")
	}
}

// TestNotificationRequest_Fields tests NotificationRequest structure
func TestNotificationRequest_Fields(t *testing.T) {
	userID := uuid.New()

	req := NotificationRequest{
		UserID:  userID,
		Type:    NotificationTypeRideAccepted,
		Channel: NotificationChannelPush,
		Title:   "Ride Accepted",
		Body:    "Your driver is on the way",
		Data:    map[string]string{"driver_name": "John"},
	}

	if req.Type != NotificationTypeRideAccepted {
		t.Errorf("Type = %s, want ride_accepted", req.Type)
	}
	if req.Channel != NotificationChannelPush {
		t.Errorf("Channel = %s, want push", req.Channel)
	}
	if req.Data["driver_name"] != "John" {
		t.Error("Data driver_name mismatch")
	}
}

// ==================== Helper Tests ====================

// TestOptionalFields tests optional field handling
func TestOptionalFields(t *testing.T) {
	// Test User with nil optional fields
	user := User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		PhoneNumber:  "+1234567890",
		FirstName:    "John",
		LastName:     "Doe",
		Role:         RoleRider,
		ProfileImage: nil,
		DeletedAt:    nil,
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}

	// Optional nil fields should be omitted
	if jsonContains(data, "profile_image") {
		t.Error("profile_image should be omitted when nil")
	}
	if jsonContains(data, "deleted_at") {
		t.Error("deleted_at should be omitted when nil")
	}
}

// TestRide_OptionalFields tests Ride optional field handling
func TestRide_OptionalFields(t *testing.T) {
	ride := Ride{
		ID:                uuid.New(),
		RiderID:           uuid.New(),
		DriverID:          nil, // Not assigned yet
		Status:            RideStatusRequested,
		PickupLatitude:    40.7128,
		PickupLongitude:   -74.0060,
		PickupAddress:     "123 Main St",
		DropoffLatitude:   40.7580,
		DropoffLongitude:  -73.9855,
		DropoffAddress:    "456 Broadway",
		EstimatedDistance: 5.5,
		EstimatedDuration: 15,
		EstimatedFare:     25.50,
		SurgeMultiplier:   1.0,
		RequestedAt:       time.Now(),
		AcceptedAt:        nil,
		StartedAt:         nil,
		CompletedAt:       nil,
		ActualDistance:    nil,
		ActualDuration:    nil,
		FinalFare:         nil,
	}

	data, err := json.Marshal(ride)
	if err != nil {
		t.Fatalf("Failed to marshal ride: %v", err)
	}

	// Optional nil fields should be omitted
	if jsonContains(data, "driver_id") {
		t.Error("driver_id should be omitted when nil")
	}
	if jsonContains(data, "accepted_at") {
		t.Error("accepted_at should be omitted when nil")
	}
	if jsonContains(data, "final_fare") {
		t.Error("final_fare should be omitted when nil")
	}
}

// ==================== Edge Case Tests ====================

// TestUser_EmptyFields tests User with empty string fields
func TestUser_EmptyFields(t *testing.T) {
	user := User{
		ID:          uuid.New(),
		Email:       "",
		PhoneNumber: "",
		FirstName:   "",
		LastName:    "",
		Role:        "",
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user with empty fields: %v", err)
	}

	if len(data) == 0 {
		t.Error("JSON output should not be empty")
	}
}

// TestPayment_ZeroValues tests Payment with zero values
func TestPayment_ZeroValues(t *testing.T) {
	payment := Payment{
		ID:             uuid.New(),
		RideID:         uuid.New(),
		RiderID:        uuid.New(),
		DriverID:       uuid.New(),
		Amount:         0,
		Currency:       "USD",
		Commission:     0,
		DriverEarnings: 0,
	}

	data, err := json.Marshal(payment)
	if err != nil {
		t.Fatalf("Failed to marshal payment with zero values: %v", err)
	}

	// Zero values should still be present
	if !jsonContains(data, `"amount":0`) {
		t.Error("Zero amount should be in JSON output")
	}
}

// ==================== Benchmark Tests ====================

func BenchmarkUser_JSON_Marshal(b *testing.B) {
	user := User{
		ID:          uuid.New(),
		Email:       "test@example.com",
		PhoneNumber: "+1234567890",
		FirstName:   "John",
		LastName:    "Doe",
		Role:        RoleRider,
		IsActive:    true,
		IsVerified:  true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(user)
	}
}

func BenchmarkRide_JSON_Marshal(b *testing.B) {
	ride := Ride{
		ID:                uuid.New(),
		RiderID:           uuid.New(),
		Status:            RideStatusInProgress,
		PickupLatitude:    40.7128,
		PickupLongitude:   -74.0060,
		PickupAddress:     "123 Main St",
		DropoffLatitude:   40.7580,
		DropoffLongitude:  -73.9855,
		DropoffAddress:    "456 Broadway",
		EstimatedDistance: 5.5,
		EstimatedDuration: 15,
		EstimatedFare:     25.50,
		SurgeMultiplier:   1.0,
		RequestedAt:       time.Now(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(ride)
	}
}

func BenchmarkPayment_JSON_Marshal(b *testing.B) {
	payment := Payment{
		ID:             uuid.New(),
		RideID:         uuid.New(),
		RiderID:        uuid.New(),
		DriverID:       uuid.New(),
		Amount:         25.50,
		Currency:       "USD",
		Commission:     2.55,
		DriverEarnings: 22.95,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(payment)
	}
}

// Helper function to check if JSON contains a string
func jsonContains(data []byte, substr string) bool {
	return json.Valid(data) && contains(string(data), substr)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
