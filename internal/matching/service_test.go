package matching

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ─── mocks ───────────────────────────────────────────────────────────────────

type mockGeoService struct{ mock.Mock }

func (m *mockGeoService) FindAvailableDrivers(ctx context.Context, latitude, longitude float64, maxDrivers int) ([]*GeoDriverLocation, error) {
	args := m.Called(ctx, latitude, longitude, maxDrivers)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*GeoDriverLocation), args.Error(1)
}

func (m *mockGeoService) CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	args := m.Called(lat1, lon1, lat2, lon2)
	return args.Get(0).(float64)
}

type mockRidesRepo struct{ mock.Mock }

func (m *mockRidesRepo) UpdateRideDriver(ctx context.Context, rideID, driverID uuid.UUID) error {
	args := m.Called(ctx, rideID, driverID)
	return args.Error(0)
}

func (m *mockRidesRepo) GetRideByID(ctx context.Context, rideID uuid.UUID) (interface{}, error) {
	args := m.Called(ctx, rideID)
	return args.Get(0), args.Error(1)
}

type mockRedis struct{ mock.Mock }

func (m *mockRedis) SetWithExpiration(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *mockRedis) GetString(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *mockRedis) GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error {
	args := m.Called(ctx, key, longitude, latitude, member)
	return args.Error(0)
}

func (m *mockRedis) GeoRadius(ctx context.Context, key string, longitude, latitude, radius float64, count int) ([]string, error) {
	args := m.Called(ctx, key, longitude, latitude, radius, count)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockRedis) GeoRemove(ctx context.Context, key string, member string) error {
	args := m.Called(ctx, key, member)
	return args.Error(0)
}

func (m *mockRedis) Delete(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func (m *mockRedis) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *mockRedis) Expire(ctx context.Context, key string, expiration time.Duration) error {
	args := m.Called(ctx, key, expiration)
	return args.Error(0)
}

func (m *mockRedis) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockRedis) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	args := m.Called(ctx, keys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *mockRedis) MGetStrings(ctx context.Context, keys ...string) ([]string, error) {
	args := m.Called(ctx, keys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newTestService(t *testing.T) (*Service, *mockGeoService, *mockRidesRepo, *mockRedis, *websocket.Hub) {
	t.Helper()
	geo := &mockGeoService{}
	repo := &mockRidesRepo{}
	redis := &mockRedis{}
	hub := websocket.NewHub()
	cfg := DefaultMatchingConfig()
	cfg.OfferTimeoutSeconds = 30
	cfg.FirstBatchSize = 3
	cfg.MaxDriversToNotify = 10
	svc := NewService(geo, repo, hub, nil, redis, cfg)
	return svc, geo, repo, redis, hub
}

func newRideRequestedEvent() *RideRequestedEvent {
	return &RideRequestedEvent{
		RideID:          uuid.New(),
		RiderID:         uuid.New(),
		RiderName:       "Test Rider",
		RiderRating:     4.8,
		PickupLatitude:  37.7749,
		PickupLongitude: -122.4194,
		PickupAddress:   "123 Main St",
		RideTypeName:    "Standard",
		EstimatedFare:   15.50,
		EstimatedDistance: 5.2,
		EstimatedDuration: 12,
		Currency:        "USD",
		RequestedAt:     time.Now(),
	}
}

// drainBroadcast reads all pending messages from the hub's broadcast channel.
func drainBroadcast(hub *websocket.Hub) []*websocket.BroadcastMessage {
	var msgs []*websocket.BroadcastMessage
	for {
		select {
		case msg := <-hub.Broadcast:
			msgs = append(msgs, msg)
		default:
			return msgs
		}
	}
}

// ─── tests: findAvailableDrivers ─────────────────────────────────────────────

func TestFindAvailableDrivers_Success(t *testing.T) {
	svc, geo, _, _, _ := newTestService(t)
	ctx := context.Background()

	d1 := uuid.New()
	d2 := uuid.New()
	geo.On("FindAvailableDrivers", ctx, 37.7749, -122.4194, 10).Return([]*GeoDriverLocation{
		{DriverID: d1, Latitude: 37.78, Longitude: -122.42, Status: "available"},
		{DriverID: d2, Latitude: 37.77, Longitude: -122.41, Status: "available"},
	}, nil)

	drivers, err := svc.findAvailableDrivers(ctx, 37.7749, -122.4194)

	require.NoError(t, err)
	assert.Len(t, drivers, 2)
	assert.Equal(t, d1, drivers[0].DriverID)
	assert.Equal(t, d2, drivers[1].DriverID)
	geo.AssertExpectations(t)
}

func TestFindAvailableDrivers_NoDrivers(t *testing.T) {
	svc, geo, _, _, _ := newTestService(t)
	ctx := context.Background()

	geo.On("FindAvailableDrivers", ctx, 37.7749, -122.4194, 10).Return([]*GeoDriverLocation{}, nil)

	drivers, err := svc.findAvailableDrivers(ctx, 37.7749, -122.4194)

	require.NoError(t, err)
	assert.Empty(t, drivers)
	geo.AssertExpectations(t)
}

func TestFindAvailableDrivers_GeoError(t *testing.T) {
	svc, geo, _, _, _ := newTestService(t)
	ctx := context.Background()

	geo.On("FindAvailableDrivers", ctx, 37.7749, -122.4194, 10).Return(nil, errors.New("geo service down"))

	drivers, err := svc.findAvailableDrivers(ctx, 37.7749, -122.4194)

	require.Error(t, err)
	assert.Nil(t, drivers)
	assert.Contains(t, err.Error(), "failed to find available drivers")
	geo.AssertExpectations(t)
}

// ─── tests: onRideRequested ───────────────────────────────────────────────────

func TestOnRideRequested_DriversFound_SendsOffers(t *testing.T) {
	svc, geo, _, redis, hub := newTestService(t)
	ctx := context.Background()
	event := newRideRequestedEvent()

	driverID := uuid.New()
	geo.On("FindAvailableDrivers", ctx, event.PickupLatitude, event.PickupLongitude, 10).Return([]*GeoDriverLocation{
		{DriverID: driverID, Latitude: 37.78, Longitude: -122.42},
	}, nil)
	geo.On("CalculateDistance", mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), event.PickupLatitude, event.PickupLongitude).Return(1.5)

	// trackOffer calls
	offerKey := "ride_offer:" + event.RideID.String() + ":" + driverID.String()
	driversKey := "ride_offer_drivers:" + event.RideID.String()
	redis.On("SetWithExpiration", ctx, offerKey, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)
	redis.On("GetString", ctx, driversKey).Return("", errors.New("not found"))
	redis.On("SetWithExpiration", ctx, driversKey, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	svc.onRideRequested(ctx, event)

	msgs := drainBroadcast(hub)
	require.Len(t, msgs, 1)
	assert.Equal(t, "user", msgs[0].Target)
	assert.Equal(t, driverID.String(), msgs[0].TargetID)
	assert.Equal(t, "ride.offer", msgs[0].Message.Type)
	geo.AssertExpectations(t)
	redis.AssertExpectations(t)
}

func TestOnRideRequested_NoDrivers_NotifiesRider(t *testing.T) {
	svc, geo, _, _, hub := newTestService(t)
	ctx := context.Background()
	event := newRideRequestedEvent()

	geo.On("FindAvailableDrivers", ctx, event.PickupLatitude, event.PickupLongitude, 10).Return([]*GeoDriverLocation{}, nil)

	svc.onRideRequested(ctx, event)

	msgs := drainBroadcast(hub)
	require.Len(t, msgs, 1)
	assert.Equal(t, "user", msgs[0].Target)
	assert.Equal(t, event.RiderID.String(), msgs[0].TargetID)
	assert.Equal(t, "ride.no_drivers", msgs[0].Message.Type)
	geo.AssertExpectations(t)
}

func TestOnRideRequested_GeoError_NoMessages(t *testing.T) {
	svc, geo, _, _, hub := newTestService(t)
	ctx := context.Background()
	event := newRideRequestedEvent()

	geo.On("FindAvailableDrivers", ctx, event.PickupLatitude, event.PickupLongitude, 10).Return(nil, errors.New("timeout"))

	svc.onRideRequested(ctx, event)

	msgs := drainBroadcast(hub)
	assert.Empty(t, msgs)
	geo.AssertExpectations(t)
}

// ─── tests: notifyRiderNoDrivers ──────────────────────────────────────────────

func TestNotifyRiderNoDrivers_SendsCorrectMessage(t *testing.T) {
	svc, _, _, _, hub := newTestService(t)
	riderID := uuid.New()
	rideID := uuid.New()

	svc.notifyRiderNoDrivers(riderID, rideID)

	msgs := drainBroadcast(hub)
	require.Len(t, msgs, 1)
	msg := msgs[0].Message
	assert.Equal(t, "ride.no_drivers", msg.Type)
	assert.Equal(t, riderID.String(), msg.UserID)
	assert.Equal(t, rideID.String(), msg.RideID)
	assert.NotZero(t, msg.Timestamp)

	data := msg.Data
	assert.Equal(t, rideID.String(), data["ride_id"])
	assert.NotEmpty(t, data["message"])
}

// ─── tests: trackOffer ───────────────────────────────────────────────────────

func TestTrackOffer_StoresOfferAndDriverSet(t *testing.T) {
	svc, _, _, redis, _ := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()
	driverID := uuid.New()
	expiresAt := time.Now().Add(30 * time.Second)

	offerKey := "ride_offer:" + rideID.String() + ":" + driverID.String()
	driversKey := "ride_offer_drivers:" + rideID.String()

	redis.On("SetWithExpiration", ctx, offerKey, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)
	redis.On("GetString", ctx, driversKey).Return("", errors.New("not found"))
	redis.On("SetWithExpiration", ctx, driversKey, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	err := svc.trackOffer(ctx, rideID, driverID, expiresAt)

	require.NoError(t, err)
	redis.AssertExpectations(t)
}

func TestTrackOffer_AppendsToPreviousDrivers(t *testing.T) {
	svc, _, _, redis, _ := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()
	driverID1 := uuid.New()
	driverID2 := uuid.New()
	expiresAt := time.Now().Add(30 * time.Second)

	offerKey := "ride_offer:" + rideID.String() + ":" + driverID2.String()
	driversKey := "ride_offer_drivers:" + rideID.String()

	existingJSON, _ := json.Marshal([]string{driverID1.String()})

	redis.On("SetWithExpiration", ctx, offerKey, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)
	redis.On("GetString", ctx, driversKey).Return(string(existingJSON), nil)
	redis.On("SetWithExpiration", ctx, driversKey, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).
		Run(func(args mock.Arguments) {
			// Verify new list contains both drivers
			var ids []string
			_ = json.Unmarshal([]byte(args.String(2)), &ids)
			assert.Contains(t, ids, driverID1.String())
			assert.Contains(t, ids, driverID2.String())
		}).Return(nil)

	err := svc.trackOffer(ctx, rideID, driverID2, expiresAt)

	require.NoError(t, err)
	redis.AssertExpectations(t)
}

func TestTrackOffer_RedisSetError_ReturnsError(t *testing.T) {
	svc, _, _, redis, _ := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()
	driverID := uuid.New()
	expiresAt := time.Now().Add(30 * time.Second)

	offerKey := "ride_offer:" + rideID.String() + ":" + driverID.String()
	redis.On("SetWithExpiration", ctx, offerKey, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(errors.New("redis error"))

	err := svc.trackOffer(ctx, rideID, driverID, expiresAt)

	assert.Error(t, err)
	redis.AssertExpectations(t)
}

// ─── tests: cancelPendingOffers ───────────────────────────────────────────────

func TestCancelPendingOffers_NotifiesAllDriversExceptAccepted(t *testing.T) {
	svc, _, _, redis, hub := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()
	acceptedDriverID := uuid.New()
	otherDriverID := uuid.New()

	driversKey := "ride_offer_drivers:" + rideID.String()
	driverIDs := []string{acceptedDriverID.String(), otherDriverID.String()}
	driversJSON, _ := json.Marshal(driverIDs)

	redis.On("GetString", ctx, driversKey).Return(string(driversJSON), nil)
	// Delete calls: one per non-accepted driver offer key + the driversKey itself
	offerKey := "ride_offer:" + rideID.String() + ":" + otherDriverID.String()
	redis.On("Delete", ctx, []string{offerKey}).Return(nil)
	redis.On("Delete", ctx, []string{driversKey}).Return(nil)

	err := svc.cancelPendingOffers(ctx, rideID, acceptedDriverID)

	require.NoError(t, err)

	msgs := drainBroadcast(hub)
	require.Len(t, msgs, 1)
	assert.Equal(t, otherDriverID.String(), msgs[0].TargetID)
	assert.Equal(t, "ride.offer_cancelled", msgs[0].Message.Type)
	redis.AssertExpectations(t)
}

func TestCancelPendingOffers_NoTrackedDrivers_ReturnsNil(t *testing.T) {
	svc, _, _, redis, hub := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()

	driversKey := "ride_offer_drivers:" + rideID.String()
	redis.On("GetString", ctx, driversKey).Return("", errors.New("key not found"))

	err := svc.cancelPendingOffers(ctx, rideID, uuid.Nil)

	require.NoError(t, err)
	msgs := drainBroadcast(hub)
	assert.Empty(t, msgs)
	redis.AssertExpectations(t)
}

func TestCancelPendingOffers_RideCancelled_NotifiesAllDrivers(t *testing.T) {
	svc, _, _, redis, hub := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()
	driver1 := uuid.New()
	driver2 := uuid.New()

	driversKey := "ride_offer_drivers:" + rideID.String()
	driverIDs := []string{driver1.String(), driver2.String()}
	driversJSON, _ := json.Marshal(driverIDs)

	redis.On("GetString", ctx, driversKey).Return(string(driversJSON), nil)
	redis.On("Delete", ctx, []string{"ride_offer:" + rideID.String() + ":" + driver1.String()}).Return(nil)
	redis.On("Delete", ctx, []string{"ride_offer:" + rideID.String() + ":" + driver2.String()}).Return(nil)
	redis.On("Delete", ctx, []string{driversKey}).Return(nil)

	// uuid.Nil means ride was cancelled (not accepted by a specific driver)
	err := svc.cancelPendingOffers(ctx, rideID, uuid.Nil)

	require.NoError(t, err)

	msgs := drainBroadcast(hub)
	assert.Len(t, msgs, 2)
	targets := []string{msgs[0].TargetID, msgs[1].TargetID}
	assert.Contains(t, targets, driver1.String())
	assert.Contains(t, targets, driver2.String())
	redis.AssertExpectations(t)
}

// ─── tests: isRidePending ─────────────────────────────────────────────────────

func TestIsRidePending_PendingInRedis(t *testing.T) {
	svc, _, _, redis, _ := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()

	redis.On("GetString", ctx, "ride_status:"+rideID.String()).Return("pending", nil)

	pending, err := svc.isRidePending(ctx, rideID)

	require.NoError(t, err)
	assert.True(t, pending)
	redis.AssertExpectations(t)
}

func TestIsRidePending_SearchingInRedis(t *testing.T) {
	svc, _, _, redis, _ := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()

	redis.On("GetString", ctx, "ride_status:"+rideID.String()).Return("searching", nil)

	pending, err := svc.isRidePending(ctx, rideID)

	require.NoError(t, err)
	assert.True(t, pending)
}

func TestIsRidePending_AcceptedInRedis(t *testing.T) {
	svc, _, _, redis, _ := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()

	redis.On("GetString", ctx, "ride_status:"+rideID.String()).Return("accepted", nil)

	pending, err := svc.isRidePending(ctx, rideID)

	require.NoError(t, err)
	assert.False(t, pending)
}

func TestIsRidePending_NotInRedis_FallsBackToDatabase(t *testing.T) {
	svc, _, repo, redis, _ := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()

	redis.On("GetString", ctx, "ride_status:"+rideID.String()).Return("", errors.New("not found"))
	repo.On("GetRideByID", ctx, rideID).Return(struct{}{}, nil)

	pending, err := svc.isRidePending(ctx, rideID)

	require.NoError(t, err)
	assert.True(t, pending) // simplified: non-nil ride returns true
	redis.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestIsRidePending_NotInRedis_DatabaseError(t *testing.T) {
	svc, _, repo, redis, _ := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()

	redis.On("GetString", ctx, "ride_status:"+rideID.String()).Return("", errors.New("not found"))
	repo.On("GetRideByID", ctx, rideID).Return(nil, errors.New("db error"))

	pending, err := svc.isRidePending(ctx, rideID)

	require.Error(t, err)
	assert.False(t, pending)
	assert.Contains(t, err.Error(), "failed to get ride from database")
}

func TestIsRidePending_NotInRedis_NilRide(t *testing.T) {
	svc, _, repo, redis, _ := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()

	redis.On("GetString", ctx, "ride_status:"+rideID.String()).Return("", errors.New("not found"))
	repo.On("GetRideByID", ctx, rideID).Return(nil, nil)

	pending, err := svc.isRidePending(ctx, rideID)

	require.NoError(t, err)
	assert.False(t, pending)
}

// ─── tests: onRideAccepted ────────────────────────────────────────────────────

func TestOnRideAccepted_CancelsPendingOffers(t *testing.T) {
	svc, _, _, redis, hub := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()
	acceptedDriverID := uuid.New()
	otherDriverID := uuid.New()

	driversKey := "ride_offer_drivers:" + rideID.String()
	driverIDs := []string{acceptedDriverID.String(), otherDriverID.String()}
	driversJSON, _ := json.Marshal(driverIDs)

	redis.On("GetString", ctx, driversKey).Return(string(driversJSON), nil)
	redis.On("Delete", ctx, []string{"ride_offer:" + rideID.String() + ":" + otherDriverID.String()}).Return(nil)
	redis.On("Delete", ctx, []string{driversKey}).Return(nil)

	event := &RideAcceptedEvent{
		RideID:     rideID,
		DriverID:   acceptedDriverID,
		AcceptedAt: time.Now(),
	}

	svc.onRideAccepted(ctx, event)

	msgs := drainBroadcast(hub)
	require.Len(t, msgs, 1)
	assert.Equal(t, otherDriverID.String(), msgs[0].TargetID)
	assert.Equal(t, "ride.offer_cancelled", msgs[0].Message.Type)
	redis.AssertExpectations(t)
}

// ─── tests: onRideCancelled ───────────────────────────────────────────────────

func TestOnRideCancelled_NotifiesAllDrivers(t *testing.T) {
	svc, _, _, redis, hub := newTestService(t)
	ctx := context.Background()
	rideID := uuid.New()
	driverID := uuid.New()

	driversKey := "ride_offer_drivers:" + rideID.String()
	driverIDs := []string{driverID.String()}
	driversJSON, _ := json.Marshal(driverIDs)

	redis.On("GetString", ctx, driversKey).Return(string(driversJSON), nil)
	redis.On("Delete", ctx, []string{"ride_offer:" + rideID.String() + ":" + driverID.String()}).Return(nil)
	redis.On("Delete", ctx, []string{driversKey}).Return(nil)

	event := &RideCancelledEvent{
		RideID:      rideID,
		CancelledBy: uuid.New(),
		Reason:      "rider_cancelled",
		CancelledAt: time.Now(),
	}

	svc.onRideCancelled(ctx, event)

	msgs := drainBroadcast(hub)
	require.Len(t, msgs, 1)
	assert.Equal(t, driverID.String(), msgs[0].TargetID)
	assert.Equal(t, "ride.offer_cancelled", msgs[0].Message.Type)
	redis.AssertExpectations(t)
}

// ─── tests: sendOfferToDriver ─────────────────────────────────────────────────

func TestSendOfferToDriver_IncludesDropoffWhenProvided(t *testing.T) {
	svc, geo, _, redis, hub := newTestService(t)
	ctx := context.Background()
	event := newRideRequestedEvent()
	dropoffLat := 37.80
	dropoffLon := -122.43
	dropoffAddr := "456 End St"
	event.DropoffLatitude = &dropoffLat
	event.DropoffLongitude = &dropoffLon
	event.DropoffAddress = &dropoffAddr

	driverID := uuid.New()
	driver := DriverLocation{
		UserID:    driverID,
		DriverID:  driverID,
		Latitude:  37.78,
		Longitude: -122.42,
	}
	expiresAt := time.Now().Add(30 * time.Second)

	geo.On("CalculateDistance", driver.Latitude, driver.Longitude, event.PickupLatitude, event.PickupLongitude).Return(2.0)
	offerKey := "ride_offer:" + event.RideID.String() + ":" + driverID.String()
	driversKey := "ride_offer_drivers:" + event.RideID.String()
	redis.On("SetWithExpiration", ctx, offerKey, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)
	redis.On("GetString", ctx, driversKey).Return("", errors.New("not found"))
	redis.On("SetWithExpiration", ctx, driversKey, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	svc.sendOfferToDriver(ctx, event, driver, expiresAt)

	msgs := drainBroadcast(hub)
	require.Len(t, msgs, 1)
	data := msgs[0].Message.Data
	dropoff := data["dropoff_location"].(*Location)
	assert.Equal(t, dropoffLat, dropoff.Latitude)
	assert.Equal(t, dropoffLon, dropoff.Longitude)
	geo.AssertExpectations(t)
	redis.AssertExpectations(t)
}

func TestSendOfferToDriver_NoDropoff(t *testing.T) {
	svc, geo, _, redis, hub := newTestService(t)
	ctx := context.Background()
	event := newRideRequestedEvent()
	// No dropoff set

	driverID := uuid.New()
	driver := DriverLocation{
		UserID:    driverID,
		DriverID:  driverID,
		Latitude:  37.78,
		Longitude: -122.42,
	}
	expiresAt := time.Now().Add(30 * time.Second)

	geo.On("CalculateDistance", driver.Latitude, driver.Longitude, event.PickupLatitude, event.PickupLongitude).Return(1.5)
	offerKey := "ride_offer:" + event.RideID.String() + ":" + driverID.String()
	driversKey := "ride_offer_drivers:" + event.RideID.String()
	redis.On("SetWithExpiration", ctx, offerKey, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)
	redis.On("GetString", ctx, driversKey).Return("", errors.New("not found"))
	redis.On("SetWithExpiration", ctx, driversKey, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	svc.sendOfferToDriver(ctx, event, driver, expiresAt)

	msgs := drainBroadcast(hub)
	require.Len(t, msgs, 1)
	data := msgs[0].Message.Data
	assert.Nil(t, data["dropoff_location"])
	geo.AssertExpectations(t)
}
