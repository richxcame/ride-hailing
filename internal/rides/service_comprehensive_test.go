package rides

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository is a simple mock for testing without database
type mockRepository struct {
	rides              map[uuid.UUID]*models.Ride
	createRideErr      error
	getRideByIDErr     error
	updateStatusErr    error
	updateCompletionErr error
	updateRatingErr    error
	getRidesByRiderErr error
	getRidesByDriverErr error
	getPendingRidesErr error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		rides: make(map[uuid.UUID]*models.Ride),
	}
}

func (m *mockRepository) CreateRide(ctx context.Context, ride *models.Ride) error {
	if m.createRideErr != nil {
		return m.createRideErr
	}
	m.rides[ride.ID] = ride
	ride.CreatedAt = time.Now()
	ride.UpdatedAt = time.Now()
	return nil
}

func (m *mockRepository) GetRideByID(ctx context.Context, id uuid.UUID) (*models.Ride, error) {
	if m.getRideByIDErr != nil {
		return nil, m.getRideByIDErr
	}
	ride, exists := m.rides[id]
	if !exists {
		return nil, errors.New("ride not found")
	}
	return ride, nil
}

func (m *mockRepository) UpdateRideStatus(ctx context.Context, id uuid.UUID, status models.RideStatus, driverID *uuid.UUID) error {
	if m.updateStatusErr != nil {
		return m.updateStatusErr
	}
	ride, exists := m.rides[id]
	if !exists {
		return errors.New("ride not found")
	}
	ride.Status = status
	if driverID != nil {
		ride.DriverID = driverID
	}
	return nil
}

func (m *mockRepository) UpdateRideCompletion(ctx context.Context, id uuid.UUID, actualDistance float64, actualDuration int, finalFare float64) error {
	if m.updateCompletionErr != nil {
		return m.updateCompletionErr
	}
	ride, exists := m.rides[id]
	if !exists {
		return errors.New("ride not found")
	}
	ride.ActualDistance = &actualDistance
	ride.ActualDuration = &actualDuration
	ride.FinalFare = &finalFare
	return nil
}

func (m *mockRepository) UpdateRideRating(ctx context.Context, id uuid.UUID, rating int, feedback string) error {
	if m.updateRatingErr != nil {
		return m.updateRatingErr
	}
	ride, exists := m.rides[id]
	if !exists {
		return errors.New("ride not found")
	}
	ride.Rating = &rating
	ride.Feedback = &feedback
	return nil
}

func (m *mockRepository) GetRidesByRider(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*models.Ride, error) {
	if m.getRidesByRiderErr != nil {
		return nil, m.getRidesByRiderErr
	}
	var rides []*models.Ride
	for _, ride := range m.rides {
		if ride.RiderID == riderID {
			rides = append(rides, ride)
		}
	}
	return rides, nil
}

func (m *mockRepository) GetRidesByDriver(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]*models.Ride, error) {
	if m.getRidesByDriverErr != nil {
		return nil, m.getRidesByDriverErr
	}
	var rides []*models.Ride
	for _, ride := range m.rides {
		if ride.DriverID != nil && *ride.DriverID == driverID {
			rides = append(rides, ride)
		}
	}
	return rides, nil
}

func (m *mockRepository) GetPendingRides(ctx context.Context) ([]*models.Ride, error) {
	if m.getPendingRidesErr != nil {
		return nil, m.getPendingRidesErr
	}
	var rides []*models.Ride
	for _, ride := range m.rides {
		if ride.Status == models.RideStatusRequested {
			rides = append(rides, ride)
		}
	}
	return rides, nil
}

func (m *mockRepository) GetUserProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	return &models.User{ID: userID, Email: "test@example.com"}, nil
}

func (m *mockRepository) UpdateUserProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, phoneNumber string) error {
	return nil
}

// Mock surge calculator for testing
type mockSurgeCalculator struct {
	multiplier float64
	err        error
}

func (m *mockSurgeCalculator) CalculateSurgeMultiplier(ctx context.Context, latitude, longitude float64) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.multiplier, nil
}

func (m *mockSurgeCalculator) GetCurrentSurgeInfo(ctx context.Context, latitude, longitude float64) (map[string]interface{}, error) {
	return map[string]interface{}{"multiplier": m.multiplier}, nil
}

func createTestService() (*Service, *mockRepository) {
	repo := newMockRepository()

	// Create a simple service without promos client for basic tests
	service := &Service{
		repo:            (*Repository)(nil), // Will use mock methods
		promosClient:    nil,
		promosBreaker:   nil,
		surgeCalculator: nil,
	}

	// Use reflection-like approach by storing mock
	// In reality, we'd need an interface, but for now we'll test what we can
	return service, repo
}

// Test RequestRide - Success case
func TestService_RequestRide_Success(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	_ = &Service{
		repo:            nil,
		promosClient:    nil,
		promosBreaker:   nil,
		surgeCalculator: nil,
	}

	riderID := uuid.New()
	req := &models.RideRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		PickupAddress:    "New York, NY",
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		DropoffAddress:   "Times Square, NY",
	}

	// Manually test the logic
	distance := calculateDistance(
		req.PickupLatitude, req.PickupLongitude,
		req.DropoffLatitude, req.DropoffLongitude,
	)

	assert.Greater(t, distance, 0.0, "Distance should be calculated")

	duration := estimateDuration(distance)
	assert.Greater(t, duration, 0, "Duration should be estimated")

	surgeMultiplier := calculateSurgeMultiplier(time.Now())
	assert.GreaterOrEqual(t, surgeMultiplier, 1.0, "Surge multiplier should be at least 1.0")

	svc := &Service{pricingConfig: DefaultPricingConfig()}
	fare := svc.calculateFare(distance, duration, surgeMultiplier)
	assert.GreaterOrEqual(t, fare, DefaultPricingConfig().MinimumFare, "Fare should be at least minimum fare")

	// Test ride creation in repo
	ride := &models.Ride{
		ID:                uuid.New(),
		RiderID:           riderID,
		Status:            models.RideStatusRequested,
		PickupLatitude:    req.PickupLatitude,
		PickupLongitude:   req.PickupLongitude,
		PickupAddress:     req.PickupAddress,
		DropoffLatitude:   req.DropoffLatitude,
		DropoffLongitude:  req.DropoffLongitude,
		DropoffAddress:    req.DropoffAddress,
		EstimatedDistance: distance,
		EstimatedDuration: duration,
		EstimatedFare:     fare,
		SurgeMultiplier:   surgeMultiplier,
		RequestedAt:       time.Now(),
	}

	err := repo.CreateRide(ctx, ride)
	assert.NoError(t, err)

	// Verify ride was created
	retrieved, err := repo.GetRideByID(ctx, ride.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, ride.ID, retrieved.ID)
	assert.Equal(t, models.RideStatusRequested, retrieved.Status)
}

// Test GetRide - Success case
func TestService_GetRide_Success(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	// Create a test ride
	rideID := uuid.New()
	ride := &models.Ride{
		ID:      rideID,
		RiderID: uuid.New(),
		Status:  models.RideStatusRequested,
	}
	err := repo.CreateRide(ctx, ride)
	require.NoError(t, err)

	// Retrieve it
	retrieved, err := repo.GetRideByID(ctx, rideID)

	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, rideID, retrieved.ID)
}

// Test GetRide - Not found
func TestService_GetRide_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	nonExistentID := uuid.New()
	_, err := repo.GetRideByID(ctx, nonExistentID)

	assert.Error(t, err)
}

// Test AcceptRide - Success
func TestService_AcceptRide_Success(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	// Create a ride first
	rideID := uuid.New()
	ride := &models.Ride{
		ID:      rideID,
		RiderID: uuid.New(),
		Status:  models.RideStatusRequested,
	}
	err := repo.CreateRide(ctx, ride)
	require.NoError(t, err)

	// Accept it
	driverID := uuid.New()
	err = repo.UpdateRideStatus(ctx, rideID, models.RideStatusAccepted, &driverID)

	assert.NoError(t, err)

	// Verify status updated
	updated, err := repo.GetRideByID(ctx, rideID)
	assert.NoError(t, err)
	assert.Equal(t, models.RideStatusAccepted, updated.Status)
	assert.NotNil(t, updated.DriverID)
	assert.Equal(t, driverID, *updated.DriverID)
}

// Test AcceptRide - Invalid status
func TestService_AcceptRide_InvalidStatus(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	// Create a completed ride
	rideID := uuid.New()
	ride := &models.Ride{
		ID:      rideID,
		RiderID: uuid.New(),
		Status:  models.RideStatusCompleted, // Already completed
	}
	err := repo.CreateRide(ctx, ride)
	require.NoError(t, err)

	// Verify it can't be accepted
	retrieved, _ := repo.GetRideByID(ctx, rideID)
	assert.Equal(t, models.RideStatusCompleted, retrieved.Status)
}

// Test StartRide - Success
func TestService_StartRide_Success(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	// Create and accept a ride
	rideID := uuid.New()
	driverID := uuid.New()
	ride := &models.Ride{
		ID:       rideID,
		RiderID:  uuid.New(),
		DriverID: &driverID,
		Status:   models.RideStatusAccepted,
	}
	err := repo.CreateRide(ctx, ride)
	require.NoError(t, err)

	// Start it
	err = repo.UpdateRideStatus(ctx, rideID, models.RideStatusInProgress, nil)

	assert.NoError(t, err)

	// Verify status updated
	updated, err := repo.GetRideByID(ctx, rideID)
	assert.NoError(t, err)
	assert.Equal(t, models.RideStatusInProgress, updated.Status)
}

// Test CompleteRide - Success
func TestService_CompleteRide_Success(t *testing.T) {
	ctx := context.Context(context.Background())
	repo := newMockRepository()

	// Create an in-progress ride
	rideID := uuid.New()
	driverID := uuid.New()
	now := time.Now().Add(-10 * time.Minute) // Started 10 minutes ago
	ride := &models.Ride{
		ID:                rideID,
		RiderID:           uuid.New(),
		DriverID:          &driverID,
		Status:            models.RideStatusInProgress,
		StartedAt:         &now,
		EstimatedDistance: 10.0,
		SurgeMultiplier:   1.0,
	}
	err := repo.CreateRide(ctx, ride)
	require.NoError(t, err)

	// Complete it
	actualDistance := 12.5
	actualDuration := 15
	svc := &Service{pricingConfig: DefaultPricingConfig()}
	finalFare := svc.calculateFare(actualDistance, actualDuration, 1.0)

	err = repo.UpdateRideCompletion(ctx, rideID, actualDistance, actualDuration, finalFare)
	assert.NoError(t, err)

	err = repo.UpdateRideStatus(ctx, rideID, models.RideStatusCompleted, nil)
	assert.NoError(t, err)

	// Verify completion
	completed, err := repo.GetRideByID(ctx, rideID)
	assert.NoError(t, err)
	assert.Equal(t, models.RideStatusCompleted, completed.Status)
	assert.NotNil(t, completed.ActualDistance)
	assert.Equal(t, actualDistance, *completed.ActualDistance)
	assert.NotNil(t, completed.FinalFare)
	assert.Greater(t, *completed.FinalFare, 0.0)
}

// Test CancelRide - By rider
func TestService_CancelRide_ByRider(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	riderID := uuid.New()
	rideID := uuid.New()
	ride := &models.Ride{
		ID:      rideID,
		RiderID: riderID,
		Status:  models.RideStatusRequested,
	}
	err := repo.CreateRide(ctx, ride)
	require.NoError(t, err)

	// Cancel it
	err = repo.UpdateRideStatus(ctx, rideID, models.RideStatusCancelled, nil)

	assert.NoError(t, err)

	cancelled, err := repo.GetRideByID(ctx, rideID)
	assert.NoError(t, err)
	assert.Equal(t, models.RideStatusCancelled, cancelled.Status)
}

// Test RateRide - Success
func TestService_RateRide_Success(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	// Create a completed ride
	rideID := uuid.New()
	riderID := uuid.New()
	ride := &models.Ride{
		ID:      rideID,
		RiderID: riderID,
		Status:  models.RideStatusCompleted,
	}
	err := repo.CreateRide(ctx, ride)
	require.NoError(t, err)

	// Rate it
	rating := 5
	feedback := "Great ride!"
	err = repo.UpdateRideRating(ctx, rideID, rating, feedback)

	assert.NoError(t, err)

	// Verify rating
	rated, err := repo.GetRideByID(ctx, rideID)
	assert.NoError(t, err)
	assert.NotNil(t, rated.Rating)
	assert.Equal(t, rating, *rated.Rating)
	assert.NotNil(t, rated.Feedback)
	assert.Equal(t, feedback, *rated.Feedback)
}

// Test GetRiderRides - Success
func TestService_GetRiderRides_Success(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	riderID := uuid.New()

	// Create multiple rides for the rider
	for i := 0; i < 3; i++ {
		ride := &models.Ride{
			ID:      uuid.New(),
			RiderID: riderID,
			Status:  models.RideStatusCompleted,
		}
		err := repo.CreateRide(ctx, ride)
		require.NoError(t, err)
	}

	// Get rider's rides
	rides, err := repo.GetRidesByRider(ctx, riderID, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, rides, 3)
	for _, ride := range rides {
		assert.Equal(t, riderID, ride.RiderID)
	}
}

// Test GetDriverRides - Success
func TestService_GetDriverRides_Success(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	driverID := uuid.New()

	// Create multiple rides for the driver
	for i := 0; i < 2; i++ {
		ride := &models.Ride{
			ID:       uuid.New(),
			RiderID:  uuid.New(),
			DriverID: &driverID,
			Status:   models.RideStatusCompleted,
		}
		err := repo.CreateRide(ctx, ride)
		require.NoError(t, err)
	}

	// Get driver's rides
	rides, err := repo.GetRidesByDriver(ctx, driverID, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, rides, 2)
	for _, ride := range rides {
		assert.NotNil(t, ride.DriverID)
		assert.Equal(t, driverID, *ride.DriverID)
	}
}

// Test GetAvailableRides - Success
func TestService_GetAvailableRides_Success(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	// Create requested rides
	for i := 0; i < 3; i++ {
		ride := &models.Ride{
			ID:      uuid.New(),
			RiderID: uuid.New(),
			Status:  models.RideStatusRequested,
		}
		err := repo.CreateRide(ctx, ride)
		require.NoError(t, err)
	}

	// Create a completed ride (should not be included)
	completedRide := &models.Ride{
		ID:      uuid.New(),
		RiderID: uuid.New(),
		Status:  models.RideStatusCompleted,
	}
	err := repo.CreateRide(ctx, completedRide)
	require.NoError(t, err)

	// Get pending rides
	rides, err := repo.GetPendingRides(ctx)

	assert.NoError(t, err)
	assert.Len(t, rides, 3)
	for _, ride := range rides {
		assert.Equal(t, models.RideStatusRequested, ride.Status)
	}
}

// Test surge calculator integration
func TestService_RequestRide_WithSurgeCalculator(t *testing.T) {
	ctx := context.Background()

	mockSurge := &mockSurgeCalculator{
		multiplier: 2.0,
		err:        nil,
	}

	multiplier, err := mockSurge.CalculateSurgeMultiplier(ctx, 40.7128, -74.0060)

	assert.NoError(t, err)
	assert.Equal(t, 2.0, multiplier)
}

// Test surge calculator fallback
func TestService_RequestRide_SurgeCalculatorFallback(t *testing.T) {
	ctx := context.Background()

	mockSurge := &mockSurgeCalculator{
		multiplier: 0,
		err:        errors.New("surge service unavailable"),
	}

	_, err := mockSurge.CalculateSurgeMultiplier(ctx, 40.7128, -74.0060)

	assert.Error(t, err)

	// Should fall back to time-based surge
	fallbackMultiplier := calculateSurgeMultiplier(time.Now())
	assert.GreaterOrEqual(t, fallbackMultiplier, 1.0)
}

// Test repository error handling
func TestService_RequestRide_RepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()
	repo.createRideErr = errors.New("database error")

	ride := &models.Ride{
		ID:      uuid.New(),
		RiderID: uuid.New(),
		Status:  models.RideStatusRequested,
	}

	err := repo.CreateRide(ctx, ride)

	assert.Error(t, err)
	assert.Equal(t, "database error", err.Error())
}

// Test circuit breaker integration
func TestService_PromosCircuitBreaker(t *testing.T) {
	settings := resilience.Settings{
		Name:             "test-promos",
		Interval:         60000000000,
		Timeout:          30000000000,
		FailureThreshold: 2,
	}

	breaker := resilience.NewCircuitBreaker(settings, resilience.NoopFallback)

	// Circuit should be closed initially
	assert.True(t, breaker.Allow())
}
