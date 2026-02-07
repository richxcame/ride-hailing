package loyalty

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// MockRepository implements RepositoryInterface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetRiderLoyalty(ctx context.Context, riderID uuid.UUID) (*RiderLoyalty, error) {
	args := m.Called(ctx, riderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RiderLoyalty), args.Error(1)
}

func (m *MockRepository) CreateRiderLoyalty(ctx context.Context, account *RiderLoyalty) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockRepository) UpdatePoints(ctx context.Context, riderID uuid.UUID, earnedPoints, tierPoints int) error {
	args := m.Called(ctx, riderID, earnedPoints, tierPoints)
	return args.Error(0)
}

func (m *MockRepository) DeductPoints(ctx context.Context, riderID uuid.UUID, points int) error {
	args := m.Called(ctx, riderID, points)
	return args.Error(0)
}

func (m *MockRepository) UpdateTier(ctx context.Context, riderID uuid.UUID, tierID uuid.UUID) error {
	args := m.Called(ctx, riderID, tierID)
	return args.Error(0)
}

func (m *MockRepository) UpdateStreak(ctx context.Context, riderID uuid.UUID, streakDays int) error {
	args := m.Called(ctx, riderID, streakDays)
	return args.Error(0)
}

func (m *MockRepository) GetTier(ctx context.Context, tierID uuid.UUID) (*LoyaltyTier, error) {
	args := m.Called(ctx, tierID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LoyaltyTier), args.Error(1)
}

func (m *MockRepository) GetTierByName(ctx context.Context, name TierName) (*LoyaltyTier, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LoyaltyTier), args.Error(1)
}

func (m *MockRepository) GetAllTiers(ctx context.Context) ([]*LoyaltyTier, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*LoyaltyTier), args.Error(1)
}

func (m *MockRepository) CreatePointsTransaction(ctx context.Context, tx *PointsTransaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockRepository) GetPointsHistory(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*PointsTransaction, int, error) {
	args := m.Called(ctx, riderID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*PointsTransaction), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetReward(ctx context.Context, rewardID uuid.UUID) (*RewardCatalogItem, error) {
	args := m.Called(ctx, rewardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RewardCatalogItem), args.Error(1)
}

func (m *MockRepository) GetAvailableRewards(ctx context.Context, tierID *uuid.UUID) ([]*RewardCatalogItem, error) {
	args := m.Called(ctx, tierID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RewardCatalogItem), args.Error(1)
}

func (m *MockRepository) GetUserRedemptionCount(ctx context.Context, riderID, rewardID uuid.UUID) (int, error) {
	args := m.Called(ctx, riderID, rewardID)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) CreateRedemption(ctx context.Context, redemption *Redemption) error {
	args := m.Called(ctx, redemption)
	return args.Error(0)
}

func (m *MockRepository) IncrementRewardRedemptionCount(ctx context.Context, rewardID uuid.UUID) error {
	args := m.Called(ctx, rewardID)
	return args.Error(0)
}

func (m *MockRepository) GetActiveChallenges(ctx context.Context, tierID *uuid.UUID) ([]*RiderChallenge, error) {
	args := m.Called(ctx, tierID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RiderChallenge), args.Error(1)
}

func (m *MockRepository) GetActiveChallengesByType(ctx context.Context, challengeType string, tierID *uuid.UUID) ([]*RiderChallenge, error) {
	args := m.Called(ctx, challengeType, tierID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RiderChallenge), args.Error(1)
}

func (m *MockRepository) GetChallengeProgress(ctx context.Context, riderID, challengeID uuid.UUID) (*ChallengeProgress, error) {
	args := m.Called(ctx, riderID, challengeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChallengeProgress), args.Error(1)
}

func (m *MockRepository) CreateChallengeProgress(ctx context.Context, progress *ChallengeProgress) error {
	args := m.Called(ctx, progress)
	return args.Error(0)
}

func (m *MockRepository) UpdateChallengeProgress(ctx context.Context, progressID uuid.UUID, currentValue int, completed bool) error {
	args := m.Called(ctx, progressID, currentValue, completed)
	return args.Error(0)
}

func (m *MockRepository) GetLoyaltyStats(ctx context.Context) (*LoyaltyStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LoyaltyStats), args.Error(1)
}

// ============================================================================
// Helper Functions
// ============================================================================

func setupTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req

	return c, w
}

func setUserContext(c *gin.Context, userID uuid.UUID) {
	c.Set("user_id", userID)
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestHandler(mockRepo *MockRepository) *Handler {
	service := NewService(mockRepo)
	return NewHandler(service)
}

func createTestLoyaltyTier() *LoyaltyTier {
	return &LoyaltyTier{
		ID:                uuid.New(),
		Name:              TierBronze,
		DisplayName:       "Bronze",
		MinPoints:         0,
		Multiplier:        1.0,
		FreeCancellations: 2,
		FreeUpgrades:      1,
		PrioritySupport:   false,
		Benefits:          []string{"Basic support", "Standard rewards"},
		IsActive:          true,
		CreatedAt:         time.Now(),
	}
}

func createTestRiderLoyalty(riderID uuid.UUID, tier *LoyaltyTier) *RiderLoyalty {
	return &RiderLoyalty{
		RiderID:         riderID,
		CurrentTierID:   &tier.ID,
		CurrentTier:     tier,
		TotalPoints:     500,
		AvailablePoints: 300,
		LifetimePoints:  1000,
		TierPoints:      500,
		TierPeriodStart: time.Now().AddDate(-1, 0, 0),
		TierPeriodEnd:   time.Now().AddDate(1, 0, 0),
		StreakDays:      5,
		JoinedAt:        time.Now().AddDate(-2, 0, 0),
		CreatedAt:       time.Now().AddDate(-2, 0, 0),
		UpdatedAt:       time.Now(),
	}
}

func createTestRewardHandler() *RewardCatalogItem {
	return &RewardCatalogItem{
		ID:             uuid.New(),
		Name:           "Free Ride",
		RewardType:     "discount",
		PointsRequired: 100,
		ValidDays:      30,
		IsActive:       true,
		CreatedAt:      time.Now(),
	}
}

func createTestChallengeHandler() *RiderChallenge {
	return &RiderChallenge{
		ID:            uuid.New(),
		Name:          "Complete 10 Rides",
		ChallengeType: "ride_count",
		TargetValue:   10,
		RewardPoints:  50,
		StartDate:     time.Now().AddDate(0, 0, -7),
		EndDate:       time.Now().AddDate(0, 0, 7),
		IsActive:      true,
		CreatedAt:     time.Now(),
	}
}

// ============================================================================
// GetStatus Handler Tests
// ============================================================================

func TestHandler_GetStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()
	account := createTestRiderLoyalty(riderID, tier)

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
	mockRepo.On("GetTier", mock.Anything, tier.ID).Return(tier, nil)
	mockRepo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tier}, nil)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/status", nil)
	setUserContext(c, riderID)

	handler.GetStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetStatus_Unauthorized_NoUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/status", nil)
	// Don't set user context

	handler.GetStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetStatus_NewAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()

	// First call returns error (no account), second call returns the created account
	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(nil, errors.New("not found")).Once()
	mockRepo.On("GetTierByName", mock.Anything, TierBronze).Return(tier, nil)
	mockRepo.On("CreateRiderLoyalty", mock.Anything, mock.AnythingOfType("*loyalty.RiderLoyalty")).Return(nil)
	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(createTestRiderLoyalty(riderID, tier), nil).Maybe()
	mockRepo.On("GetTier", mock.Anything, mock.Anything).Return(tier, nil).Maybe()
	mockRepo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tier}, nil)
	// For the signup bonus goroutine
	mockRepo.On("CreatePointsTransaction", mock.Anything, mock.AnythingOfType("*loyalty.PointsTransaction")).Return(nil).Maybe()
	mockRepo.On("UpdatePoints", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/status", nil)
	setUserContext(c, riderID)

	handler.GetStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetStatus_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(nil, errors.New("not found"))
	mockRepo.On("GetTierByName", mock.Anything, TierBronze).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/status", nil)
	setUserContext(c, riderID)

	handler.GetStatus(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetPointsHistory Handler Tests
// ============================================================================

func TestHandler_GetPointsHistory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	transactions := []*PointsTransaction{
		{
			ID:              uuid.New(),
			RiderID:         riderID,
			TransactionType: TransactionEarn,
			Points:          50,
			BalanceAfter:    550,
			Source:          SourceRide,
			CreatedAt:       time.Now(),
		},
	}

	mockRepo.On("GetPointsHistory", mock.Anything, riderID, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(transactions, 1, nil)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/points/history", nil)
	setUserContext(c, riderID)

	handler.GetPointsHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetPointsHistory_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/points/history", nil)
	// Don't set user context

	handler.GetPointsHistory(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetPointsHistory_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()

	mockRepo.On("GetPointsHistory", mock.Anything, riderID, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return([]*PointsTransaction{}, 0, nil)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/points/history", nil)
	setUserContext(c, riderID)

	handler.GetPointsHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetPointsHistory_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()

	mockRepo.On("GetPointsHistory", mock.Anything, riderID, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil, 0, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/points/history", nil)
	setUserContext(c, riderID)

	handler.GetPointsHistory(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetRewards Handler Tests
// ============================================================================

func TestHandler_GetRewards_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()
	account := createTestRiderLoyalty(riderID, tier)
	rewards := []*RewardCatalogItem{createTestRewardHandler()}

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
	mockRepo.On("GetAvailableRewards", mock.Anything, mock.Anything).Return(rewards, nil)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/rewards", nil)
	setUserContext(c, riderID)

	handler.GetRewards(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetRewards_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/rewards", nil)
	// Don't set user context

	handler.GetRewards(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetRewards_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()
	account := createTestRiderLoyalty(riderID, tier)

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
	mockRepo.On("GetAvailableRewards", mock.Anything, mock.Anything).Return([]*RewardCatalogItem{}, nil)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/rewards", nil)
	setUserContext(c, riderID)

	handler.GetRewards(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetRewards_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(nil, errors.New("not found"))
	mockRepo.On("GetAvailableRewards", mock.Anything, (*uuid.UUID)(nil)).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/rewards", nil)
	setUserContext(c, riderID)

	handler.GetRewards(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// RedeemReward Handler Tests
// ============================================================================

func TestHandler_RedeemReward_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()
	account := createTestRiderLoyalty(riderID, tier)
	reward := createTestRewardHandler()

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
	mockRepo.On("GetReward", mock.Anything, reward.ID).Return(reward, nil)
	mockRepo.On("CreateRedemption", mock.Anything, mock.AnythingOfType("*loyalty.Redemption")).Return(nil)
	mockRepo.On("CreatePointsTransaction", mock.Anything, mock.AnythingOfType("*loyalty.PointsTransaction")).Return(nil)
	mockRepo.On("DeductPoints", mock.Anything, riderID, reward.PointsRequired).Return(nil)
	mockRepo.On("IncrementRewardRedemptionCount", mock.Anything, reward.ID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/rider/loyalty/rewards/"+reward.ID.String()+"/redeem", nil)
	c.Params = gin.Params{{Key: "id", Value: reward.ID.String()}}
	setUserContext(c, riderID)

	handler.RedeemReward(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_RedeemReward_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rewardID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/rider/loyalty/rewards/"+rewardID.String()+"/redeem", nil)
	c.Params = gin.Params{{Key: "id", Value: rewardID.String()}}
	// Don't set user context

	handler.RedeemReward(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RedeemReward_InvalidRewardID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/rider/loyalty/rewards/invalid-uuid/redeem", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, riderID)

	handler.RedeemReward(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid reward ID")
}

func TestHandler_RedeemReward_RewardNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()
	account := createTestRiderLoyalty(riderID, tier)
	rewardID := uuid.New()

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
	mockRepo.On("GetReward", mock.Anything, rewardID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("POST", "/api/v1/rider/loyalty/rewards/"+rewardID.String()+"/redeem", nil)
	c.Params = gin.Params{{Key: "id", Value: rewardID.String()}}
	setUserContext(c, riderID)

	handler.RedeemReward(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_RedeemReward_InsufficientPoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()
	account := createTestRiderLoyalty(riderID, tier)
	account.AvailablePoints = 50 // Less than required
	reward := createTestRewardHandler()
	reward.PointsRequired = 100

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
	mockRepo.On("GetReward", mock.Anything, reward.ID).Return(reward, nil)

	c, w := setupTestContext("POST", "/api/v1/rider/loyalty/rewards/"+reward.ID.String()+"/redeem", nil)
	c.Params = gin.Params{{Key: "id", Value: reward.ID.String()}}
	setUserContext(c, riderID)

	handler.RedeemReward(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "insufficient points")
}

func TestHandler_RedeemReward_InactiveReward(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()
	account := createTestRiderLoyalty(riderID, tier)
	reward := createTestRewardHandler()
	reward.IsActive = false

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
	mockRepo.On("GetReward", mock.Anything, reward.ID).Return(reward, nil)

	c, w := setupTestContext("POST", "/api/v1/rider/loyalty/rewards/"+reward.ID.String()+"/redeem", nil)
	c.Params = gin.Params{{Key: "id", Value: reward.ID.String()}}
	setUserContext(c, riderID)

	handler.RedeemReward(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// GetChallenges Handler Tests
// ============================================================================

func TestHandler_GetChallenges_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()
	account := createTestRiderLoyalty(riderID, tier)
	challenges := []*RiderChallenge{createTestChallengeHandler()}

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
	mockRepo.On("GetActiveChallenges", mock.Anything, mock.Anything).Return(challenges, nil)
	mockRepo.On("GetChallengeProgress", mock.Anything, riderID, mock.Anything).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/challenges", nil)
	setUserContext(c, riderID)

	handler.GetChallenges(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetChallenges_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/challenges", nil)
	// Don't set user context

	handler.GetChallenges(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetChallenges_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()
	account := createTestRiderLoyalty(riderID, tier)

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
	mockRepo.On("GetActiveChallenges", mock.Anything, mock.Anything).Return([]*RiderChallenge{}, nil)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/challenges", nil)
	setUserContext(c, riderID)

	handler.GetChallenges(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetChallenges_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()
	account := createTestRiderLoyalty(riderID, tier)

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
	mockRepo.On("GetActiveChallenges", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/challenges", nil)
	setUserContext(c, riderID)

	handler.GetChallenges(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetTiers Handler Tests
// ============================================================================

func TestHandler_GetTiers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	tiers := []*LoyaltyTier{
		createTestLoyaltyTier(),
		{
			ID:        uuid.New(),
			Name:      TierSilver,
			MinPoints: 1000,
			IsActive:  true,
		},
	}

	mockRepo.On("GetAllTiers", mock.Anything).Return(tiers, nil)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/tiers", nil)

	handler.GetTiers(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetTiers_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{}, nil)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/tiers", nil)

	handler.GetTiers(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetTiers_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetAllTiers", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/tiers", nil)

	handler.GetTiers(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Admin GetLoyaltyStats Handler Tests
// ============================================================================

func TestHandler_GetLoyaltyStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	stats := &LoyaltyStats{
		TotalMembers:           1000,
		BronzeCount:            600,
		SilverCount:            250,
		GoldCount:              100,
		PlatinumCount:          40,
		DiamondCount:           10,
		TotalPointsEarned:      500000,
		TotalPointsOutstanding: 150000,
	}

	mockRepo.On("GetLoyaltyStats", mock.Anything).Return(stats, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/loyalty/stats", nil)

	handler.GetLoyaltyStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetLoyaltyStats_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetLoyaltyStats", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/loyalty/stats", nil)

	handler.GetLoyaltyStats(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Admin AwardPoints Handler Tests
// ============================================================================

func TestHandler_AwardPoints_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()
	account := createTestRiderLoyalty(riderID, tier)

	reqBody := map[string]interface{}{
		"rider_id":    riderID.String(),
		"points":      100,
		"description": "Promotional bonus",
	}

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
	mockRepo.On("CreatePointsTransaction", mock.Anything, mock.AnythingOfType("*loyalty.PointsTransaction")).Return(nil)
	mockRepo.On("UpdatePoints", mock.Anything, riderID, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil)
	mockRepo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tier}, nil).Maybe()
	mockRepo.On("UpdateTier", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	c, w := setupTestContext("POST", "/api/v1/admin/loyalty/award", reqBody)

	handler.AwardPoints(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_AwardPoints_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("POST", "/api/v1/admin/loyalty/award", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/loyalty/award", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.AwardPoints(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AwardPoints_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := map[string]interface{}{
		"rider_id": uuid.New().String(),
		// Missing points and description
	}

	c, w := setupTestContext("POST", "/api/v1/admin/loyalty/award", reqBody)

	handler.AwardPoints(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AwardPoints_InvalidPoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()

	reqBody := map[string]interface{}{
		"rider_id":    riderID.String(),
		"points":      0, // Invalid - must be > 0
		"description": "Test",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/loyalty/award", reqBody)

	handler.AwardPoints(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AwardPoints_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()
	account := createTestRiderLoyalty(riderID, tier)

	reqBody := map[string]interface{}{
		"rider_id":    riderID.String(),
		"points":      100,
		"description": "Test bonus",
	}

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
	mockRepo.On("CreatePointsTransaction", mock.Anything, mock.AnythingOfType("*loyalty.PointsTransaction")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/admin/loyalty/award", reqBody)

	handler.AwardPoints(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_GetStatus_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockRepository, uuid.UUID)
		setUserID      bool
		userID         uuid.UUID
		expectedStatus int
	}{
		{
			name: "success with existing account",
			setupMock: func(m *MockRepository, riderID uuid.UUID) {
				tier := createTestLoyaltyTier()
				account := createTestRiderLoyalty(riderID, tier)
				m.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
				m.On("GetTier", mock.Anything, tier.ID).Return(tier, nil)
				m.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tier}, nil)
			},
			setUserID:      true,
			userID:         uuid.New(),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthorized - no user ID",
			setupMock:      func(m *MockRepository, riderID uuid.UUID) {},
			setUserID:      false,
			userID:         uuid.Nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "service error - tier not found",
			setupMock: func(m *MockRepository, riderID uuid.UUID) {
				m.On("GetRiderLoyalty", mock.Anything, riderID).Return(nil, errors.New("not found"))
				m.On("GetTierByName", mock.Anything, TierBronze).Return(nil, errors.New("tier not found"))
			},
			setUserID:      true,
			userID:         uuid.New(),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			if tt.setUserID && tt.userID == uuid.Nil {
				tt.userID = uuid.New()
			}

			tt.setupMock(mockRepo, tt.userID)

			c, w := setupTestContext("GET", "/api/v1/rider/loyalty/status", nil)
			if tt.setUserID {
				setUserContext(c, tt.userID)
			}

			handler.GetStatus(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_RedeemReward_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		rewardID       string
		setupMock      func(*MockRepository, uuid.UUID, uuid.UUID)
		setUserID      bool
		expectedStatus int
	}{
		{
			name:     "success",
			rewardID: uuid.New().String(),
			setupMock: func(m *MockRepository, riderID uuid.UUID, rewardID uuid.UUID) {
				tier := createTestLoyaltyTier()
				account := createTestRiderLoyalty(riderID, tier)
				reward := createTestRewardHandler()
				reward.ID = rewardID
				m.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
				m.On("GetReward", mock.Anything, rewardID).Return(reward, nil)
				m.On("CreateRedemption", mock.Anything, mock.AnythingOfType("*loyalty.Redemption")).Return(nil)
				m.On("CreatePointsTransaction", mock.Anything, mock.AnythingOfType("*loyalty.PointsTransaction")).Return(nil)
				m.On("DeductPoints", mock.Anything, riderID, reward.PointsRequired).Return(nil)
				m.On("IncrementRewardRedemptionCount", mock.Anything, reward.ID).Return(nil)
			},
			setUserID:      true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid reward ID",
			rewardID:       "invalid-uuid",
			setupMock:      func(m *MockRepository, riderID uuid.UUID, rewardID uuid.UUID) {},
			setUserID:      true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized",
			rewardID:       uuid.New().String(),
			setupMock:      func(m *MockRepository, riderID uuid.UUID, rewardID uuid.UUID) {},
			setUserID:      false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:     "reward not found",
			rewardID: uuid.New().String(),
			setupMock: func(m *MockRepository, riderID uuid.UUID, rewardID uuid.UUID) {
				tier := createTestLoyaltyTier()
				account := createTestRiderLoyalty(riderID, tier)
				m.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
				m.On("GetReward", mock.Anything, rewardID).Return(nil, errors.New("not found"))
			},
			setUserID:      true,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			riderID := uuid.New()
			rewardID, _ := uuid.Parse(tt.rewardID)

			tt.setupMock(mockRepo, riderID, rewardID)

			c, w := setupTestContext("POST", "/api/v1/rider/loyalty/rewards/"+tt.rewardID+"/redeem", nil)
			c.Params = gin.Params{{Key: "id", Value: tt.rewardID}}
			if tt.setUserID {
				setUserContext(c, riderID)
			}

			handler.RedeemReward(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_GetStatus_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	tier := createTestLoyaltyTier()
	account := createTestRiderLoyalty(riderID, tier)

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil)
	mockRepo.On("GetTier", mock.Anything, tier.ID).Return(tier, nil)
	mockRepo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tier}, nil)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/status", nil)
	setUserContext(c, riderID)

	handler.GetStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()

	mockRepo.On("GetRiderLoyalty", mock.Anything, riderID).Return(nil, errors.New("not found"))
	mockRepo.On("GetTierByName", mock.Anything, TierBronze).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/status", nil)
	setUserContext(c, riderID)

	handler.GetStatus(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)

	// Verify error response structure
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestHandler_GetPointsHistory_WithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()
	transactions := []*PointsTransaction{
		{ID: uuid.New(), RiderID: riderID, Points: 10, CreatedAt: time.Now()},
	}

	mockRepo.On("GetPointsHistory", mock.Anything, riderID, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(transactions, 50, nil)

	c, w := setupTestContext("GET", "/api/v1/rider/loyalty/points/history?limit=10&offset=20", nil)
	c.Request.URL.RawQuery = "limit=10&offset=20"
	setUserContext(c, riderID)

	handler.GetPointsHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_RedeemReward_EmptyRewardID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	riderID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/rider/loyalty/rewards//redeem", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, riderID)

	handler.RedeemReward(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
