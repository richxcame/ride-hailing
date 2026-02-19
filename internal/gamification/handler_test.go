package gamification

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

func (m *MockRepository) GetDriverGamification(ctx context.Context, driverID uuid.UUID) (*DriverGamification, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverGamification), args.Error(1)
}

func (m *MockRepository) CreateDriverGamification(ctx context.Context, profile *DriverGamification) error {
	args := m.Called(ctx, profile)
	return args.Error(0)
}

func (m *MockRepository) UpdateDriverStats(ctx context.Context, driverID uuid.UUID, rides int, earnings float64, rating float64) error {
	args := m.Called(ctx, driverID, rides, earnings, rating)
	return args.Error(0)
}

func (m *MockRepository) UpdateDriverStreak(ctx context.Context, driverID uuid.UUID, currentStreak int) error {
	args := m.Called(ctx, driverID, currentStreak)
	return args.Error(0)
}

func (m *MockRepository) UpdateDriverTier(ctx context.Context, driverID uuid.UUID, tierID uuid.UUID) error {
	args := m.Called(ctx, driverID, tierID)
	return args.Error(0)
}

func (m *MockRepository) IncrementQuestsCompleted(ctx context.Context, driverID uuid.UUID) error {
	args := m.Called(ctx, driverID)
	return args.Error(0)
}

func (m *MockRepository) AddBonusEarned(ctx context.Context, driverID uuid.UUID, bonus float64) error {
	args := m.Called(ctx, driverID, bonus)
	return args.Error(0)
}

func (m *MockRepository) AddAchievementPoints(ctx context.Context, driverID uuid.UUID, points int) error {
	args := m.Called(ctx, driverID, points)
	return args.Error(0)
}

func (m *MockRepository) ResetWeeklyStats(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepository) ResetMonthlyStats(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepository) GetTier(ctx context.Context, tierID uuid.UUID) (*DriverTier, error) {
	args := m.Called(ctx, tierID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverTier), args.Error(1)
}

func (m *MockRepository) GetTierByName(ctx context.Context, name DriverTierName) (*DriverTier, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverTier), args.Error(1)
}

func (m *MockRepository) GetAllTiers(ctx context.Context) ([]*DriverTier, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverTier), args.Error(1)
}

func (m *MockRepository) GetActiveQuests(ctx context.Context, tierID *uuid.UUID, city *string) ([]*DriverQuest, error) {
	args := m.Called(ctx, tierID, city)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverQuest), args.Error(1)
}

func (m *MockRepository) GetQuest(ctx context.Context, questID uuid.UUID) (*DriverQuest, error) {
	args := m.Called(ctx, questID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverQuest), args.Error(1)
}

func (m *MockRepository) GetQuestsByType(ctx context.Context, questType QuestType) ([]*DriverQuest, error) {
	args := m.Called(ctx, questType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverQuest), args.Error(1)
}

func (m *MockRepository) GetQuestProgress(ctx context.Context, driverID, questID uuid.UUID) (*DriverQuestProgress, error) {
	args := m.Called(ctx, driverID, questID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverQuestProgress), args.Error(1)
}

func (m *MockRepository) GetDriverActiveQuests(ctx context.Context, driverID uuid.UUID) ([]*DriverQuestProgress, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverQuestProgress), args.Error(1)
}

func (m *MockRepository) CreateQuestProgress(ctx context.Context, progress *DriverQuestProgress) error {
	args := m.Called(ctx, progress)
	return args.Error(0)
}

func (m *MockRepository) UpdateQuestProgress(ctx context.Context, progressID uuid.UUID, currentValue int, status QuestStatus) error {
	args := m.Called(ctx, progressID, currentValue, status)
	return args.Error(0)
}

func (m *MockRepository) ClaimQuestReward(ctx context.Context, progressID uuid.UUID, rewardAmount float64) error {
	args := m.Called(ctx, progressID, rewardAmount)
	return args.Error(0)
}

func (m *MockRepository) GetAchievement(ctx context.Context, achievementID uuid.UUID) (*Achievement, error) {
	args := m.Called(ctx, achievementID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Achievement), args.Error(1)
}

func (m *MockRepository) GetAllAchievements(ctx context.Context) ([]*Achievement, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Achievement), args.Error(1)
}

func (m *MockRepository) GetDriverAchievements(ctx context.Context, driverID uuid.UUID) ([]*DriverAchievement, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverAchievement), args.Error(1)
}

func (m *MockRepository) HasAchievement(ctx context.Context, driverID, achievementID uuid.UUID) (bool, error) {
	args := m.Called(ctx, driverID, achievementID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) AwardAchievement(ctx context.Context, driverID, achievementID uuid.UUID) error {
	args := m.Called(ctx, driverID, achievementID)
	return args.Error(0)
}

func (m *MockRepository) GetLeaderboard(ctx context.Context, period string, category string, limit int) ([]LeaderboardEntry, error) {
	args := m.Called(ctx, period, category, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]LeaderboardEntry), args.Error(1)
}

func (m *MockRepository) GetDriverLeaderboardPosition(ctx context.Context, driverID uuid.UUID, period string, category string) (*LeaderboardEntry, error) {
	args := m.Called(ctx, driverID, period, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LeaderboardEntry), args.Error(1)
}

func (m *MockRepository) GetDriverIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockRepository) GetUserIDByDriverID(ctx context.Context, driverID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, driverID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockRepository) GetWeeklyStats(ctx context.Context, userID uuid.UUID) (*WeeklyStats, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WeeklyStats), args.Error(1)
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

func createTestDriverTier() *DriverTier {
	return &DriverTier{
		ID:               uuid.New(),
		Name:             DriverTierNew,
		DisplayName:      "New Driver",
		MinRides:         0,
		MinRating:        0,
		CommissionRate:   0.2,
		BonusMultiplier:  1.0,
		PriorityDispatch: false,
		Benefits:         []string{"Standard support"},
		IsActive:         true,
		CreatedAt:        time.Now(),
	}
}

func createTestDriverGamification(driverID uuid.UUID, tier *DriverTier) *DriverGamification {
	return &DriverGamification{
		DriverID:             driverID,
		CurrentTierID:        &tier.ID,
		CurrentTier:          tier,
		TotalPoints:          500,
		WeeklyPoints:         150,
		MonthlyPoints:        450,
		CurrentStreak:        5,
		LongestStreak:        10,
		TotalQuestsCompleted: 3,
		TotalBonusEarned:     250.00,
		CreatedAt:            time.Now().AddDate(0, -6, 0),
		UpdatedAt:            time.Now(),
	}
}

func createTestQuest() *DriverQuest {
	return &DriverQuest{
		ID:          uuid.New(),
		Name:        "Complete 10 Rides",
		QuestType:   QuestTypeRideCount,
		TargetValue: 10,
		RewardType:  "bonus",
		RewardValue: 50.00,
		StartDate:   time.Now().AddDate(0, 0, -7),
		EndDate:     time.Now().AddDate(0, 0, 7),
		IsActive:    true,
		IsFeatured:  true,
		CreatedAt:   time.Now(),
	}
}

func createTestQuestProgress(driverID uuid.UUID, quest *DriverQuest, status QuestStatus) *DriverQuestProgress {
	return &DriverQuestProgress{
		ID:           uuid.New(),
		DriverID:     driverID,
		QuestID:      quest.ID,
		Quest:        quest,
		CurrentValue: 5,
		Status:       status,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func createTestAchievement() *Achievement {
	return &Achievement{
		ID:       uuid.New(),
		Name:     "Century Rider",
		Category: AchievementCategoryMilestone,
		Points:   100,
		Rarity:   "rare",
		IsActive: true,
	}
}

func createTestDriverAchievement(driverID uuid.UUID, achievement *Achievement) *DriverAchievement {
	return &DriverAchievement{
		ID:            uuid.New(),
		DriverID:      driverID,
		AchievementID: achievement.ID,
		Achievement:   achievement,
		EarnedAt:      time.Now(),
	}
}

// ============================================================================
// GetStatus Handler Tests
// ============================================================================

func TestHandler_GetStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	tier := createTestDriverTier()
	profile := createTestDriverGamification(driverID, tier)
	quests := []*DriverQuest{createTestQuest()}
	achievements := []*DriverAchievement{createTestDriverAchievement(driverID, createTestAchievement())}

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
	mockRepo.On("GetAllTiers", mock.Anything).Return([]*DriverTier{tier}, nil)
	mockRepo.On("GetActiveQuests", mock.Anything, mock.Anything, mock.Anything).Return(quests, nil)
	mockRepo.On("GetQuestProgress", mock.Anything, driverID, mock.Anything).Return(nil, errors.New("not found"))
	mockRepo.On("GetDriverAchievements", mock.Anything, driverID).Return(achievements, nil)
	mockRepo.On("GetUserIDByDriverID", mock.Anything, driverID).Return(userID, nil)
	mockRepo.On("GetWeeklyStats", mock.Anything, userID).Return(&WeeklyStats{Rides: 7, Earnings: 41.46}, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/status", nil)
	setUserContext(c, userID)

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

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/status", nil)
	// Don't set user context

	handler.GetStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetStatus_NewProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	tier := createTestDriverTier()

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	// First call returns error (no profile), triggers creation
	mockRepo.On("GetDriverGamification", mock.Anything, driverID).Return(nil, errors.New("not found")).Once()
	// Second call from GetActiveQuestsWithProgress (returns nil profile, but that's ok)
	mockRepo.On("GetDriverGamification", mock.Anything, driverID).Return(nil, errors.New("not found")).Maybe()
	mockRepo.On("GetTierByName", mock.Anything, DriverTierNew).Return(tier, nil)
	mockRepo.On("CreateDriverGamification", mock.Anything, mock.AnythingOfType("*gamification.DriverGamification")).Return(nil)
	mockRepo.On("GetAllTiers", mock.Anything).Return([]*DriverTier{tier}, nil)
	mockRepo.On("GetActiveQuests", mock.Anything, mock.Anything, mock.Anything).Return([]*DriverQuest{}, nil)
	mockRepo.On("GetDriverAchievements", mock.Anything, driverID).Return([]*DriverAchievement{}, nil)
	mockRepo.On("GetUserIDByDriverID", mock.Anything, driverID).Return(userID, nil)
	mockRepo.On("GetWeeklyStats", mock.Anything, userID).Return(&WeeklyStats{}, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/status", nil)
	setUserContext(c, userID)

	handler.GetStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetStatus_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetDriverGamification", mock.Anything, driverID).Return(nil, errors.New("not found"))
	mockRepo.On("GetTierByName", mock.Anything, DriverTierNew).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/status", nil)
	setUserContext(c, userID)

	handler.GetStatus(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetQuests Handler Tests
// ============================================================================

func TestHandler_GetQuests_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	tier := createTestDriverTier()
	profile := createTestDriverGamification(driverID, tier)
	quests := []*DriverQuest{createTestQuest()}

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
	mockRepo.On("GetActiveQuests", mock.Anything, mock.Anything, mock.Anything).Return(quests, nil)
	mockRepo.On("GetQuestProgress", mock.Anything, driverID, mock.Anything).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/quests", nil)
	setUserContext(c, userID)

	handler.GetQuests(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetQuests_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/quests", nil)
	// Don't set user context

	handler.GetQuests(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetQuests_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	tier := createTestDriverTier()
	profile := createTestDriverGamification(driverID, tier)

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
	mockRepo.On("GetActiveQuests", mock.Anything, mock.Anything, mock.Anything).Return([]*DriverQuest{}, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/quests", nil)
	setUserContext(c, userID)

	handler.GetQuests(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetQuests_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	tier := createTestDriverTier()
	profile := createTestDriverGamification(driverID, tier)

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
	mockRepo.On("GetActiveQuests", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/quests", nil)
	setUserContext(c, userID)

	handler.GetQuests(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// JoinQuest Handler Tests
// ============================================================================

func TestHandler_JoinQuest_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	quest := createTestQuest()

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetQuestProgress", mock.Anything, driverID, quest.ID).Return(nil, errors.New("not found"))
	mockRepo.On("CreateQuestProgress", mock.Anything, mock.AnythingOfType("*gamification.DriverQuestProgress")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/gamification/quests/"+quest.ID.String()+"/join", nil)
	c.Params = gin.Params{{Key: "id", Value: quest.ID.String()}}
	setUserContext(c, userID)

	handler.JoinQuest(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_JoinQuest_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	questID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/gamification/quests/"+questID.String()+"/join", nil)
	c.Params = gin.Params{{Key: "id", Value: questID.String()}}
	// Don't set user context

	handler.JoinQuest(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_JoinQuest_InvalidQuestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/gamification/quests/invalid-uuid/join", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID)

	handler.JoinQuest(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid quest ID")
}

func TestHandler_JoinQuest_AlreadyJoined(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	quest := createTestQuest()
	progress := createTestQuestProgress(driverID, quest, QuestStatusActive)

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetQuestProgress", mock.Anything, driverID, quest.ID).Return(progress, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/gamification/quests/"+quest.ID.String()+"/join", nil)
	c.Params = gin.Params{{Key: "id", Value: quest.ID.String()}}
	setUserContext(c, userID)

	handler.JoinQuest(c)

	// Should succeed even if already joined
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_JoinQuest_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	quest := createTestQuest()

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetQuestProgress", mock.Anything, driverID, quest.ID).Return(nil, errors.New("not found"))
	mockRepo.On("CreateQuestProgress", mock.Anything, mock.AnythingOfType("*gamification.DriverQuestProgress")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/gamification/quests/"+quest.ID.String()+"/join", nil)
	c.Params = gin.Params{{Key: "id", Value: quest.ID.String()}}
	setUserContext(c, userID)

	handler.JoinQuest(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// ClaimQuestReward Handler Tests
// ============================================================================

func TestHandler_ClaimQuestReward_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	quest := createTestQuest()
	progress := createTestQuestProgress(driverID, quest, QuestStatusCompleted)

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetQuestProgress", mock.Anything, driverID, quest.ID).Return(progress, nil)
	mockRepo.On("GetQuest", mock.Anything, quest.ID).Return(quest, nil)
	mockRepo.On("ClaimQuestReward", mock.Anything, progress.ID, quest.RewardValue).Return(nil)
	mockRepo.On("AddBonusEarned", mock.Anything, driverID, quest.RewardValue).Return(nil)
	mockRepo.On("IncrementQuestsCompleted", mock.Anything, driverID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/gamification/quests/"+quest.ID.String()+"/claim", nil)
	c.Params = gin.Params{{Key: "id", Value: quest.ID.String()}}
	setUserContext(c, userID)

	handler.ClaimQuestReward(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_ClaimQuestReward_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	questID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/gamification/quests/"+questID.String()+"/claim", nil)
	c.Params = gin.Params{{Key: "id", Value: questID.String()}}
	// Don't set user context

	handler.ClaimQuestReward(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ClaimQuestReward_InvalidQuestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/gamification/quests/invalid-uuid/claim", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID)

	handler.ClaimQuestReward(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ClaimQuestReward_QuestNotCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	quest := createTestQuest()
	progress := createTestQuestProgress(driverID, quest, QuestStatusActive) // Not completed

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetQuestProgress", mock.Anything, driverID, quest.ID).Return(progress, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/gamification/quests/"+quest.ID.String()+"/claim", nil)
	c.Params = gin.Params{{Key: "id", Value: quest.ID.String()}}
	setUserContext(c, userID)

	handler.ClaimQuestReward(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "not completed")
}

func TestHandler_ClaimQuestReward_QuestNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	questID := uuid.New()

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetQuestProgress", mock.Anything, driverID, questID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("POST", "/api/v1/driver/gamification/quests/"+questID.String()+"/claim", nil)
	c.Params = gin.Params{{Key: "id", Value: questID.String()}}
	setUserContext(c, userID)

	handler.ClaimQuestReward(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// GetAchievements Handler Tests
// ============================================================================

func TestHandler_GetAchievements_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	achievements := []*DriverAchievement{
		createTestDriverAchievement(driverID, createTestAchievement()),
	}

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetDriverAchievements", mock.Anything, driverID).Return(achievements, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/achievements", nil)
	setUserContext(c, userID)

	handler.GetAchievements(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetAchievements_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/achievements", nil)
	// Don't set user context

	handler.GetAchievements(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetAchievements_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetDriverAchievements", mock.Anything, driverID).Return([]*DriverAchievement{}, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/achievements", nil)
	setUserContext(c, userID)

	handler.GetAchievements(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetAchievements_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetDriverAchievements", mock.Anything, driverID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/achievements", nil)
	setUserContext(c, userID)

	handler.GetAchievements(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetLeaderboard Handler Tests
// ============================================================================

func TestHandler_GetLeaderboard_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	entries := []LeaderboardEntry{
		{Rank: 1, DriverID: uuid.New(), DriverName: "Top Driver", Value: 100},
		{Rank: 2, DriverID: driverID, DriverName: "Current Driver", Value: 50, IsCurrentDriver: true},
	}
	driverPosition := &LeaderboardEntry{Rank: 2, DriverID: driverID, Value: 50, IsCurrentDriver: true}

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetLeaderboard", mock.Anything, "weekly", "rides", 100).Return(entries, nil)
	mockRepo.On("GetDriverLeaderboardPosition", mock.Anything, driverID, "weekly", "rides").Return(driverPosition, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/leaderboard", nil)
	setUserContext(c, userID)

	handler.GetLeaderboard(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetLeaderboard_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/leaderboard", nil)
	// Don't set user context

	handler.GetLeaderboard(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetLeaderboard_WithQueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetLeaderboard", mock.Anything, "monthly", "earnings", 100).Return([]LeaderboardEntry{}, nil)
	mockRepo.On("GetDriverLeaderboardPosition", mock.Anything, driverID, "monthly", "earnings").Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/leaderboard?period=monthly&category=earnings", nil)
	c.Request.URL.RawQuery = "period=monthly&category=earnings"
	setUserContext(c, userID)

	handler.GetLeaderboard(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetLeaderboard_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetLeaderboard", mock.Anything, "weekly", "rides", 100).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/leaderboard", nil)
	setUserContext(c, userID)

	handler.GetLeaderboard(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetTiers Handler Tests
// ============================================================================

func TestHandler_GetTiers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	tiers := []*DriverTier{
		createTestDriverTier(),
		{
			ID:        uuid.New(),
			Name:      DriverTierBronze,
			MinRides:  10,
			IsActive:  true,
		},
	}

	mockRepo.On("GetAllTiers", mock.Anything).Return(tiers, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/tiers", nil)

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

	mockRepo.On("GetAllTiers", mock.Anything).Return([]*DriverTier{}, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/tiers", nil)

	handler.GetTiers(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetTiers_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetAllTiers", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/tiers", nil)

	handler.GetTiers(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Admin GetDriverStats Handler Tests
// ============================================================================

func TestHandler_GetDriverStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	userID := uuid.New()
	tier := createTestDriverTier()
	profile := createTestDriverGamification(driverID, tier)

	mockRepo.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
	mockRepo.On("GetAllTiers", mock.Anything).Return([]*DriverTier{tier}, nil)
	mockRepo.On("GetActiveQuests", mock.Anything, mock.Anything, mock.Anything).Return([]*DriverQuest{}, nil)
	mockRepo.On("GetDriverAchievements", mock.Anything, driverID).Return([]*DriverAchievement{}, nil)
	mockRepo.On("GetUserIDByDriverID", mock.Anything, driverID).Return(userID, nil)
	mockRepo.On("GetWeeklyStats", mock.Anything, userID).Return(&WeeklyStats{}, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/gamification/driver/"+driverID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: driverID.String()}}

	handler.GetDriverStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetDriverStats_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/admin/gamification/driver/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetDriverStats(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid driver ID")
}

func TestHandler_GetDriverStats_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	// GetOrCreateProfile fails → returns error before GetUserIDByDriverID is called
	mockRepo.On("GetDriverGamification", mock.Anything, driverID).Return(nil, errors.New("not found"))
	mockRepo.On("GetTierByName", mock.Anything, DriverTierNew).Return(nil, errors.New("tier not found"))

	c, w := setupTestContext("GET", "/api/v1/admin/gamification/driver/"+driverID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: driverID.String()}}

	handler.GetDriverStats(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Admin ResetWeeklyStats Handler Tests
// ============================================================================

func TestHandler_ResetWeeklyStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("ResetWeeklyStats", mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/gamification/reset/weekly", nil)

	handler.ResetWeeklyStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_ResetWeeklyStats_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("ResetWeeklyStats", mock.Anything).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/admin/gamification/reset/weekly", nil)

	handler.ResetWeeklyStats(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Admin ResetMonthlyStats Handler Tests
// ============================================================================

func TestHandler_ResetMonthlyStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("ResetMonthlyStats", mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/gamification/reset/monthly", nil)

	handler.ResetMonthlyStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_ResetMonthlyStats_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("ResetMonthlyStats", mock.Anything).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/admin/gamification/reset/monthly", nil)

	handler.ResetMonthlyStats(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_GetStatus_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockRepository, uuid.UUID, uuid.UUID)
		setUserID      bool
		expectedStatus int
	}{
		{
			name: "success with existing profile",
			setupMock: func(m *MockRepository, userID uuid.UUID, driverID uuid.UUID) {
				tier := createTestDriverTier()
				profile := createTestDriverGamification(driverID, tier)
				m.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
				m.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
				m.On("GetAllTiers", mock.Anything).Return([]*DriverTier{tier}, nil)
				m.On("GetActiveQuests", mock.Anything, mock.Anything, mock.Anything).Return([]*DriverQuest{}, nil)
				m.On("GetDriverAchievements", mock.Anything, driverID).Return([]*DriverAchievement{}, nil)
				m.On("GetUserIDByDriverID", mock.Anything, driverID).Return(userID, nil)
				m.On("GetWeeklyStats", mock.Anything, userID).Return(&WeeklyStats{}, nil)
			},
			setUserID:      true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthorized - no user ID",
			setupMock:      func(m *MockRepository, userID uuid.UUID, driverID uuid.UUID) {},
			setUserID:      false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "service error - tier not found",
			setupMock: func(m *MockRepository, userID uuid.UUID, driverID uuid.UUID) {
				m.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
				m.On("GetDriverGamification", mock.Anything, driverID).Return(nil, errors.New("not found"))
				m.On("GetTierByName", mock.Anything, DriverTierNew).Return(nil, errors.New("tier not found"))
			},
			setUserID:      true,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			userID := uuid.New()
			driverID := uuid.New()

			tt.setupMock(mockRepo, userID, driverID)

			c, w := setupTestContext("GET", "/api/v1/driver/gamification/status", nil)
			if tt.setUserID {
				setUserContext(c, userID)
			}

			handler.GetStatus(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_ClaimQuestReward_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		questID        string
		setupMock      func(*MockRepository, uuid.UUID, uuid.UUID, uuid.UUID)
		setUserID      bool
		expectedStatus int
	}{
		{
			name:    "success",
			questID: uuid.New().String(),
			setupMock: func(m *MockRepository, userID uuid.UUID, driverID uuid.UUID, questID uuid.UUID) {
				quest := createTestQuest()
				quest.ID = questID
				progress := createTestQuestProgress(driverID, quest, QuestStatusCompleted)
				m.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
				m.On("GetQuestProgress", mock.Anything, driverID, questID).Return(progress, nil)
				m.On("GetQuest", mock.Anything, questID).Return(quest, nil)
				m.On("ClaimQuestReward", mock.Anything, progress.ID, quest.RewardValue).Return(nil)
				m.On("AddBonusEarned", mock.Anything, driverID, quest.RewardValue).Return(nil)
				m.On("IncrementQuestsCompleted", mock.Anything, driverID).Return(nil)
			},
			setUserID:      true,
			expectedStatus: http.StatusOK,
		},
		{
			name:    "invalid quest ID",
			questID: "invalid-uuid",
			setupMock: func(m *MockRepository, userID uuid.UUID, driverID uuid.UUID, questID uuid.UUID) {
				m.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
			},
			setUserID:      true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized",
			questID:        uuid.New().String(),
			setupMock:      func(m *MockRepository, userID uuid.UUID, driverID uuid.UUID, questID uuid.UUID) {},
			setUserID:      false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:    "quest not completed",
			questID: uuid.New().String(),
			setupMock: func(m *MockRepository, userID uuid.UUID, driverID uuid.UUID, questID uuid.UUID) {
				quest := createTestQuest()
				quest.ID = questID
				progress := createTestQuestProgress(driverID, quest, QuestStatusActive)
				m.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
				m.On("GetQuestProgress", mock.Anything, driverID, questID).Return(progress, nil)
			},
			setUserID:      true,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			userID := uuid.New()
			driverID := uuid.New()
			questID, _ := uuid.Parse(tt.questID)

			tt.setupMock(mockRepo, userID, driverID, questID)

			c, w := setupTestContext("POST", "/api/v1/driver/gamification/quests/"+tt.questID+"/claim", nil)
			c.Params = gin.Params{{Key: "id", Value: tt.questID}}
			if tt.setUserID {
				setUserContext(c, userID)
			}

			handler.ClaimQuestReward(c)

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

	userID := uuid.New()
	driverID := uuid.New()
	tier := createTestDriverTier()
	profile := createTestDriverGamification(driverID, tier)

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
	mockRepo.On("GetAllTiers", mock.Anything).Return([]*DriverTier{tier}, nil)
	mockRepo.On("GetActiveQuests", mock.Anything, mock.Anything, mock.Anything).Return([]*DriverQuest{}, nil)
	mockRepo.On("GetDriverAchievements", mock.Anything, driverID).Return([]*DriverAchievement{}, nil)
	mockRepo.On("GetUserIDByDriverID", mock.Anything, driverID).Return(userID, nil)
	mockRepo.On("GetWeeklyStats", mock.Anything, userID).Return(&WeeklyStats{}, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/status", nil)
	setUserContext(c, userID)

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

	userID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetDriverGamification", mock.Anything, driverID).Return(nil, errors.New("not found"))
	mockRepo.On("GetTierByName", mock.Anything, DriverTierNew).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/status", nil)
	setUserContext(c, userID)

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

func TestHandler_GetLeaderboard_DefaultParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()

	// Should use default "weekly" and "rides"
	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)
	mockRepo.On("GetLeaderboard", mock.Anything, "weekly", "rides", 100).Return([]LeaderboardEntry{}, nil)
	mockRepo.On("GetDriverLeaderboardPosition", mock.Anything, driverID, "weekly", "rides").Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/driver/gamification/leaderboard", nil)
	setUserContext(c, userID)

	handler.GetLeaderboard(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_JoinQuest_EmptyQuestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("GetDriverIDByUserID", mock.Anything, userID).Return(driverID, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/gamification/quests//join", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID)

	handler.JoinQuest(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetDriverStats_EmptyDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/admin/gamification/driver/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.GetDriverStats(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
