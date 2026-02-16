package eventbus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewEvent
// ---------------------------------------------------------------------------

func TestNewEvent_Success(t *testing.T) {
	data := map[string]string{"ride_id": "abc"}

	event, err := NewEvent("rides.requested", "ride-service", data)
	require.NoError(t, err)
	require.NotNil(t, event)

	assert.Equal(t, "rides.requested", event.Type)
	assert.Equal(t, "ride-service", event.Source)
	assert.NotEmpty(t, event.ID)
	assert.False(t, event.Timestamp.IsZero())

	// ID should be a valid UUID
	_, err = uuid.Parse(event.ID)
	assert.NoError(t, err)

	// Data should be valid JSON
	var decoded map[string]string
	err = json.Unmarshal(event.Data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "abc", decoded["ride_id"])
}

func TestNewEvent_NilData(t *testing.T) {
	event, err := NewEvent("test.event", "test-source", nil)
	require.NoError(t, err)
	assert.Equal(t, json.RawMessage("null"), event.Data)
}

func TestNewEvent_ComplexData(t *testing.T) {
	rideTypeID := uuid.New()
	data := RideRequestedData{
		RideID:            uuid.New(),
		RiderID:           uuid.New(),
		RiderName:         "John Doe",
		RiderRating:       4.8,
		PickupLatitude:         40.7128,
		PickupLongitude:         -74.0060,
		PickupAddress:     "123 Main St, New York, NY",
		DropoffLatitude:        40.7580,
		DropoffLongitude:        -73.9855,
		DropoffAddress:    "456 Park Ave, New York, NY",
		RideTypeID:        rideTypeID,
		RideTypeName:      "Economy",
		EstimatedFare:     25.50,
		EstimatedDistance: 5.2,
		EstimatedDuration: 15,
		Currency:          "USD",
		RequestedAt:       time.Now(),
	}

	event, err := NewEvent(SubjectRideRequested, "ride-service", data)
	require.NoError(t, err)

	// Deserialize and verify
	var decoded RideRequestedData
	err = json.Unmarshal(event.Data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, data.RideID, decoded.RideID)
	assert.Equal(t, data.RiderName, decoded.RiderName)
	assert.Equal(t, data.RiderRating, decoded.RiderRating)
	assert.Equal(t, data.PickupLatitude, decoded.PickupLatitude)
	assert.Equal(t, data.PickupAddress, decoded.PickupAddress)
	assert.Equal(t, data.RideTypeID, decoded.RideTypeID)
	assert.Equal(t, data.RideTypeName, decoded.RideTypeName)
	assert.Equal(t, data.EstimatedFare, decoded.EstimatedFare)
	assert.Equal(t, data.Currency, decoded.Currency)
}

func TestNewEvent_UnmarshalableData(t *testing.T) {
	// Channels cannot be marshaled to JSON
	event, err := NewEvent("test", "src", make(chan int))
	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestNewEvent_UniqueIDs(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		event, err := NewEvent("test", "src", nil)
		require.NoError(t, err)
		assert.False(t, ids[event.ID], "duplicate event ID generated")
		ids[event.ID] = true
	}
}

func TestNewEvent_TimestampIsUTC(t *testing.T) {
	event, err := NewEvent("test", "src", nil)
	require.NoError(t, err)
	assert.Equal(t, time.UTC, event.Timestamp.Location())
}

// ---------------------------------------------------------------------------
// Event JSON serialization round-trip
// ---------------------------------------------------------------------------

func TestEvent_JSONRoundTrip(t *testing.T) {
	original, err := NewEvent("rides.completed", "ride-service", map[string]int{"fare": 25})
	require.NoError(t, err)

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored Event
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.Type, restored.Type)
	assert.Equal(t, original.Source, restored.Source)
	assert.JSONEq(t, string(original.Data), string(restored.Data))
}

// ---------------------------------------------------------------------------
// Subject constants
// ---------------------------------------------------------------------------

func TestSubjectConstants(t *testing.T) {
	tests := []struct {
		name     string
		subject  string
		expected string
	}{
		{"RideRequested", SubjectRideRequested, "rides.requested"},
		{"RideAccepted", SubjectRideAccepted, "rides.accepted"},
		{"RideStarted", SubjectRideStarted, "rides.started"},
		{"RideCompleted", SubjectRideCompleted, "rides.completed"},
		{"RideCancelled", SubjectRideCancelled, "rides.cancelled"},
		{"PaymentProcessed", SubjectPaymentProcessed, "payments.processed"},
		{"PaymentFailed", SubjectPaymentFailed, "payments.failed"},
		{"DriverLocationUpdated", SubjectDriverLocationUpdated, "drivers.location.updated"},
		{"DriverOnline", SubjectDriverOnline, "drivers.online"},
		{"DriverOffline", SubjectDriverOffline, "drivers.offline"},
		{"FraudDetected", SubjectFraudDetected, "fraud.detected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.subject)
		})
	}
}

// ---------------------------------------------------------------------------
// DefaultConfig
// ---------------------------------------------------------------------------

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "nats://127.0.0.1:4222", cfg.URL)
	assert.Equal(t, "ride-hailing", cfg.Name)
	assert.Equal(t, "RIDEHAILING", cfg.StreamName)
}

// ---------------------------------------------------------------------------
// Config struct
// ---------------------------------------------------------------------------

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		URL:        "nats://custom:4222",
		Name:       "my-service",
		StreamName: "MYSTREAM",
	}

	assert.Equal(t, "nats://custom:4222", cfg.URL)
	assert.Equal(t, "my-service", cfg.Name)
	assert.Equal(t, "MYSTREAM", cfg.StreamName)
}

// ---------------------------------------------------------------------------
// HandlerFunc type
// ---------------------------------------------------------------------------

func TestHandlerFunc_Invocation(t *testing.T) {
	var called bool
	var receivedEvent *Event

	handler := HandlerFunc(func(ctx context.Context, event *Event) error {
		called = true
		receivedEvent = event
		return nil
	})

	event, _ := NewEvent("test.event", "test", map[string]string{"key": "value"})
	err := handler(context.Background(), event)

	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, event.ID, receivedEvent.ID)
}

func TestHandlerFunc_ReturnsError(t *testing.T) {
	handler := HandlerFunc(func(ctx context.Context, event *Event) error {
		return assert.AnError
	})

	event, _ := NewEvent("test", "src", nil)
	err := handler(context.Background(), event)

	assert.ErrorIs(t, err, assert.AnError)
}

// ---------------------------------------------------------------------------
// Event data types – serialization
// ---------------------------------------------------------------------------

func TestRideRequestedData_Serialization(t *testing.T) {
	rideTypeID := uuid.New()
	data := RideRequestedData{
		RideID:            uuid.New(),
		RiderID:           uuid.New(),
		RiderName:         "Alice Smith",
		RiderRating:       4.9,
		PickupLatitude:         37.7749,
		PickupLongitude:         -122.4194,
		PickupAddress:     "789 Market St, San Francisco, CA",
		DropoffLatitude:        37.3382,
		DropoffLongitude:        -121.8863,
		DropoffAddress:    "1 Apple Park Way, Cupertino, CA",
		RideTypeID:        rideTypeID,
		RideTypeName:      "Premium",
		EstimatedFare:     85.00,
		EstimatedDistance: 45.2,
		EstimatedDuration: 50,
		Currency:          "USD",
		RequestedAt:       time.Now().UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded RideRequestedData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.RideID, decoded.RideID)
	assert.Equal(t, data.RiderID, decoded.RiderID)
	assert.Equal(t, data.RiderName, decoded.RiderName)
	assert.Equal(t, data.RiderRating, decoded.RiderRating)
	assert.Equal(t, data.PickupLatitude, decoded.PickupLatitude)
	assert.Equal(t, data.PickupLongitude, decoded.PickupLongitude)
	assert.Equal(t, data.PickupAddress, decoded.PickupAddress)
	assert.Equal(t, data.DropoffLatitude, decoded.DropoffLatitude)
	assert.Equal(t, data.DropoffLongitude, decoded.DropoffLongitude)
	assert.Equal(t, data.DropoffAddress, decoded.DropoffAddress)
	assert.Equal(t, data.RideTypeID, decoded.RideTypeID)
	assert.Equal(t, data.RideTypeName, decoded.RideTypeName)
	assert.Equal(t, data.EstimatedFare, decoded.EstimatedFare)
	assert.Equal(t, data.EstimatedDistance, decoded.EstimatedDistance)
	assert.Equal(t, data.EstimatedDuration, decoded.EstimatedDuration)
	assert.Equal(t, data.Currency, decoded.Currency)
	assert.Equal(t, data.RequestedAt, decoded.RequestedAt)
}

func TestRideAcceptedData_Serialization(t *testing.T) {
	data := RideAcceptedData{
		RideID:     uuid.New(),
		RiderID:    uuid.New(),
		DriverID:   uuid.New(),
		PickupLatitude:  40.7128,
		PickupLongitude:  -74.0060,
		DropoffLatitude: 40.7580,
		DropoffLongitude: -73.9855,
		AcceptedAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded RideAcceptedData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.RideID, decoded.RideID)
	assert.Equal(t, data.DriverID, decoded.DriverID)
}

func TestRideCompletedData_Serialization(t *testing.T) {
	data := RideCompletedData{
		RideID:      uuid.New(),
		RiderID:     uuid.New(),
		DriverID:    uuid.New(),
		FareAmount:  25.50,
		DistanceKm:  12.3,
		DurationMin: 18.5,
		CompletedAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded RideCompletedData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.FareAmount, decoded.FareAmount)
	assert.Equal(t, data.DistanceKm, decoded.DistanceKm)
	assert.Equal(t, data.DurationMin, decoded.DurationMin)
}

func TestRideCancelledData_Serialization(t *testing.T) {
	data := RideCancelledData{
		RideID:      uuid.New(),
		RiderID:     uuid.New(),
		DriverID:    uuid.New(),
		CancelledBy: "rider",
		Reason:      "changed mind",
		CancelledAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded RideCancelledData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.CancelledBy, decoded.CancelledBy)
	assert.Equal(t, data.Reason, decoded.Reason)
}

func TestPaymentProcessedData_Serialization(t *testing.T) {
	data := PaymentProcessedData{
		PaymentID:   uuid.New(),
		RideID:      uuid.New(),
		RiderID:     uuid.New(),
		DriverID:    uuid.New(),
		Amount:      45.99,
		Currency:    "USD",
		Method:      "card",
		ProcessedAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded PaymentProcessedData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.Amount, decoded.Amount)
	assert.Equal(t, data.Currency, decoded.Currency)
	assert.Equal(t, data.Method, decoded.Method)
}

func TestPaymentFailedData_Serialization(t *testing.T) {
	data := PaymentFailedData{
		PaymentID: uuid.New(),
		RideID:    uuid.New(),
		RiderID:   uuid.New(),
		Amount:    30.00,
		Error:     "insufficient funds",
		FailedAt:  time.Now().UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded PaymentFailedData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.Error, decoded.Error)
}

func TestDriverLocationUpdatedData_Serialization(t *testing.T) {
	data := DriverLocationUpdatedData{
		DriverID:  uuid.New(),
		Latitude:  37.7749,
		Longitude: -122.4194,
		Heading:   90.0,
		Speed:     35.5,
		H3Cell:    "8928308280fffff",
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded DriverLocationUpdatedData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.H3Cell, decoded.H3Cell)
	assert.Equal(t, data.Speed, decoded.Speed)
	assert.Equal(t, data.Heading, decoded.Heading)
}

func TestFraudDetectedData_Serialization(t *testing.T) {
	data := FraudDetectedData{
		UserID:     uuid.New(),
		AlertType:  "suspicious_login",
		Severity:   "high",
		Details:    "Multiple failed login attempts from different locations",
		DetectedAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded FraudDetectedData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.AlertType, decoded.AlertType)
	assert.Equal(t, data.Severity, decoded.Severity)
	assert.Equal(t, data.Details, decoded.Details)
}

// ---------------------------------------------------------------------------
// Negotiation event types – serialization
// ---------------------------------------------------------------------------

func TestNegotiationStartedData_Serialization(t *testing.T) {
	cityID := uuid.New()
	initialOffer := 15.50

	data := NegotiationStartedData{
		SessionID:         uuid.New(),
		RiderID:           uuid.New(),
		PickupLatitude:         40.7128,
		PickupLongitude:         -74.0060,
		DropoffLatitude:        40.7580,
		DropoffLongitude:        -73.9855,
		PickupAddress:     "123 Main St",
		DropoffAddress:    "456 Broadway",
		CityID:            &cityID,
		EstimatedFare:     20.00,
		FairPriceMin:      15.00,
		FairPriceMax:      25.00,
		CurrencyCode:      "USD",
		RiderInitialOffer: &initialOffer,
		ExpiresAt:         time.Now().Add(5 * time.Minute).UTC().Truncate(time.Millisecond),
		StartedAt:         time.Now().UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded NegotiationStartedData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.SessionID, decoded.SessionID)
	assert.Equal(t, data.EstimatedFare, decoded.EstimatedFare)
	require.NotNil(t, decoded.CityID)
	assert.Equal(t, cityID, *decoded.CityID)
	require.NotNil(t, decoded.RiderInitialOffer)
	assert.Equal(t, initialOffer, *decoded.RiderInitialOffer)
}

func TestNegotiationStartedData_NilOptionalFields(t *testing.T) {
	data := NegotiationStartedData{
		SessionID:     uuid.New(),
		RiderID:       uuid.New(),
		EstimatedFare: 20.00,
		CurrencyCode:  "USD",
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded NegotiationStartedData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.CityID)
	assert.Nil(t, decoded.RiderInitialOffer)
}

func TestNegotiationOfferData_Serialization(t *testing.T) {
	pickupTime := 5
	driverRating := 4.8

	data := NegotiationOfferData{
		OfferID:             uuid.New(),
		SessionID:           uuid.New(),
		DriverID:            uuid.New(),
		RiderID:             uuid.New(),
		OfferedPrice:        18.50,
		CurrencyCode:        "USD",
		EstimatedPickupTime: &pickupTime,
		DriverRating:        &driverRating,
		IsCounterOffer:      true,
		CreatedAt:           time.Now().UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded NegotiationOfferData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.OfferedPrice, decoded.OfferedPrice)
	assert.True(t, decoded.IsCounterOffer)
	require.NotNil(t, decoded.EstimatedPickupTime)
	assert.Equal(t, 5, *decoded.EstimatedPickupTime)
	require.NotNil(t, decoded.DriverRating)
	assert.Equal(t, 4.8, *decoded.DriverRating)
}

func TestNegotiationOfferAcceptedData_Serialization(t *testing.T) {
	data := NegotiationOfferAcceptedData{
		SessionID:     uuid.New(),
		OfferID:       uuid.New(),
		RiderID:       uuid.New(),
		DriverID:      uuid.New(),
		AcceptedPrice: 22.00,
		CurrencyCode:  "USD",
		AcceptedAt:    time.Now().UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded NegotiationOfferAcceptedData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.AcceptedPrice, decoded.AcceptedPrice)
}

func TestNegotiationExpiredData_Serialization(t *testing.T) {
	data := NegotiationExpiredData{
		SessionID:   uuid.New(),
		RiderID:     uuid.New(),
		OffersCount: 3,
		ExpiredAt:   time.Now().UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded NegotiationExpiredData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 3, decoded.OffersCount)
}

func TestNegotiationCancelledData_Serialization(t *testing.T) {
	data := NegotiationCancelledData{
		SessionID:   uuid.New(),
		RiderID:     uuid.New(),
		Reason:      "found another ride",
		CancelledAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded NegotiationCancelledData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "found another ride", decoded.Reason)
}

func TestDriverInviteData_Serialization(t *testing.T) {
	data := DriverInviteData{
		SessionID:     uuid.New(),
		DriverIDs:     []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		PickupLatitude:     40.7128,
		PickupLongitude:     -74.0060,
		EstimatedFare: 18.00,
		FairPriceMin:  14.00,
		FairPriceMax:  22.00,
		ExpiresAt:     time.Now().Add(3 * time.Minute).UTC().Truncate(time.Millisecond),
	}

	b, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded DriverInviteData
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.DriverIDs, 3)
	assert.Equal(t, data.EstimatedFare, decoded.EstimatedFare)
}

// ---------------------------------------------------------------------------
// NewEvent with each event data type – integration
// ---------------------------------------------------------------------------

func TestNewEvent_WithRideStartedData(t *testing.T) {
	data := RideStartedData{
		RideID:    uuid.New(),
		RiderID:   uuid.New(),
		DriverID:  uuid.New(),
		StartedAt: time.Now().UTC(),
	}

	event, err := NewEvent(SubjectRideStarted, "ride-service", data)
	require.NoError(t, err)
	assert.Equal(t, SubjectRideStarted, event.Type)

	var decoded RideStartedData
	err = json.Unmarshal(event.Data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, data.RideID, decoded.RideID)
}

// ---------------------------------------------------------------------------
// Bus struct – nil-safety of Connected()
// ---------------------------------------------------------------------------

func TestBus_Connected_NilConn(t *testing.T) {
	bus := &Bus{}
	assert.False(t, bus.Connected())
}

// ---------------------------------------------------------------------------
// Bus struct – Close with empty subs
// ---------------------------------------------------------------------------

func TestBus_Close_NoSubs(t *testing.T) {
	bus := &Bus{}
	// Should not panic
	bus.Close()
}

// ---------------------------------------------------------------------------
// Event struct – zero value
// ---------------------------------------------------------------------------

func TestEvent_ZeroValue(t *testing.T) {
	var event Event
	assert.Empty(t, event.ID)
	assert.Empty(t, event.Type)
	assert.Empty(t, event.Source)
	assert.True(t, event.Timestamp.IsZero())
	assert.Nil(t, event.Data)
}
