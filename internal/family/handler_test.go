package family

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
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Implementation
// ============================================================================

// MockFamilyRepository implements RepositoryInterface for testing
type MockFamilyRepository struct {
	mock.Mock
}

func (m *MockFamilyRepository) CreateFamily(ctx context.Context, family *FamilyAccount) error {
	args := m.Called(ctx, family)
	return args.Error(0)
}

func (m *MockFamilyRepository) GetFamilyByID(ctx context.Context, id uuid.UUID) (*FamilyAccount, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FamilyAccount), args.Error(1)
}

func (m *MockFamilyRepository) GetFamilyByOwner(ctx context.Context, ownerID uuid.UUID) (*FamilyAccount, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FamilyAccount), args.Error(1)
}

func (m *MockFamilyRepository) UpdateFamily(ctx context.Context, family *FamilyAccount) error {
	args := m.Called(ctx, family)
	return args.Error(0)
}

func (m *MockFamilyRepository) IncrementSpend(ctx context.Context, familyID uuid.UUID, amount float64) error {
	args := m.Called(ctx, familyID, amount)
	return args.Error(0)
}

func (m *MockFamilyRepository) ResetMonthlySpend(ctx context.Context, dayOfMonth int) error {
	args := m.Called(ctx, dayOfMonth)
	return args.Error(0)
}

func (m *MockFamilyRepository) AddMember(ctx context.Context, member *FamilyMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockFamilyRepository) GetMembersByFamily(ctx context.Context, familyID uuid.UUID) ([]FamilyMember, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]FamilyMember), args.Error(1)
}

func (m *MockFamilyRepository) GetMemberByUserID(ctx context.Context, familyID, userID uuid.UUID) (*FamilyMember, error) {
	args := m.Called(ctx, familyID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FamilyMember), args.Error(1)
}

func (m *MockFamilyRepository) GetFamilyForUser(ctx context.Context, userID uuid.UUID) (*FamilyAccount, *FamilyMember, error) {
	args := m.Called(ctx, userID)
	var family *FamilyAccount
	var member *FamilyMember
	if args.Get(0) != nil {
		family = args.Get(0).(*FamilyAccount)
	}
	if args.Get(1) != nil {
		member = args.Get(1).(*FamilyMember)
	}
	return family, member, args.Error(2)
}

func (m *MockFamilyRepository) UpdateMember(ctx context.Context, member *FamilyMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockFamilyRepository) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	args := m.Called(ctx, memberID)
	return args.Error(0)
}

func (m *MockFamilyRepository) IncrementMemberSpend(ctx context.Context, memberID uuid.UUID, amount float64) error {
	args := m.Called(ctx, memberID, amount)
	return args.Error(0)
}

func (m *MockFamilyRepository) CreateInvite(ctx context.Context, invite *FamilyInvite) error {
	args := m.Called(ctx, invite)
	return args.Error(0)
}

func (m *MockFamilyRepository) GetInviteByID(ctx context.Context, id uuid.UUID) (*FamilyInvite, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FamilyInvite), args.Error(1)
}

func (m *MockFamilyRepository) GetPendingInvitesForUser(ctx context.Context, userID uuid.UUID, email *string) ([]FamilyInvite, error) {
	args := m.Called(ctx, userID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]FamilyInvite), args.Error(1)
}

func (m *MockFamilyRepository) UpdateInviteStatus(ctx context.Context, inviteID uuid.UUID, status InviteStatus) error {
	args := m.Called(ctx, inviteID, status)
	return args.Error(0)
}

func (m *MockFamilyRepository) LogRide(ctx context.Context, log *FamilyRideLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockFamilyRepository) GetSpendingReport(ctx context.Context, familyID uuid.UUID) ([]MemberSpend, int, float64, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Get(2).(float64), args.Error(3)
	}
	return args.Get(0).([]MemberSpend), args.Int(1), args.Get(2).(float64), args.Error(3)
}

// ============================================================================
// Helper Functions
// ============================================================================

func setupFamilyTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func setFamilyUserContext(c *gin.Context, userID uuid.UUID, role models.UserRole) {
	c.Set("user_id", userID)
	c.Set("user_role", role)
	c.Set("user_email", "test@example.com")
}

func parseFamilyResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestFamilyHandler(mockRepo *MockFamilyRepository) *Handler {
	service := NewService(mockRepo)
	return NewHandler(service)
}

func createTestFamilyAccount(ownerID uuid.UUID) *FamilyAccount {
	budget := 500.0
	return &FamilyAccount{
		ID:             uuid.New(),
		OwnerID:        ownerID,
		Name:           "Test Family",
		MonthlyBudget:  &budget,
		CurrentSpend:   100.0,
		BudgetResetDay: 1,
		Currency:       "USD",
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func createTestFamilyMember(familyID, userID uuid.UUID, role MemberRole) *FamilyMember {
	return &FamilyMember{
		ID:               uuid.New(),
		FamilyID:         familyID,
		UserID:           userID,
		Role:             role,
		DisplayName:      "Test Member",
		UseSharedPayment: true,
		RideApprovalReq:  false,
		IsActive:         true,
		JoinedAt:         time.Now(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func createTestFamilyInvite(familyID, inviterID uuid.UUID) *FamilyInvite {
	return &FamilyInvite{
		ID:        uuid.New(),
		FamilyID:  familyID,
		InviterID: inviterID,
		Role:      MemberRoleAdult,
		Status:    InviteStatusPending,
		ExpiresAt: time.Now().AddDate(0, 0, 7),
		CreatedAt: time.Now(),
	}
}

// ============================================================================
// CreateFamily Handler Tests
// ============================================================================

func TestHandler_CreateFamily_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	reqBody := CreateFamilyRequest{
		Name:     "The Test Family",
		Currency: "USD",
	}

	// Mock expectations - GetFamilyByOwner returns pgx.ErrNoRows for "not found"
	mockRepo.On("GetFamilyByOwner", mock.Anything, userID).Return(nil, pgx.ErrNoRows)
	mockRepo.On("GetFamilyForUser", mock.Anything, userID).Return(nil, nil, nil)
	mockRepo.On("CreateFamily", mock.Anything, mock.AnythingOfType("*family.FamilyAccount")).Return(nil)
	mockRepo.On("AddMember", mock.Anything, mock.AnythingOfType("*family.FamilyMember")).Return(nil)

	c, w := setupFamilyTestContext("POST", "/api/v1/family", reqBody)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.CreateFamily(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CreateFamily_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	reqBody := CreateFamilyRequest{
		Name: "The Test Family",
	}

	c, w := setupFamilyTestContext("POST", "/api/v1/family", reqBody)
	// Don't set user context

	handler.CreateFamily(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseFamilyResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_CreateFamily_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()

	c, w := setupFamilyTestContext("POST", "/api/v1/family", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/family", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.CreateFamily(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateFamily_AlreadyHasFamily(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	existingFamily := createTestFamilyAccount(userID)

	reqBody := CreateFamilyRequest{
		Name: "New Family",
	}

	mockRepo.On("GetFamilyByOwner", mock.Anything, userID).Return(existingFamily, nil)

	c, w := setupFamilyTestContext("POST", "/api/v1/family", reqBody)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.CreateFamily(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	response := parseFamilyResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_CreateFamily_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	reqBody := CreateFamilyRequest{
		Name: "The Test Family",
	}

	mockRepo.On("GetFamilyByOwner", mock.Anything, userID).Return(nil, errors.New("not found"))
	mockRepo.On("GetFamilyForUser", mock.Anything, userID).Return(nil, nil, nil)
	mockRepo.On("CreateFamily", mock.Anything, mock.AnythingOfType("*family.FamilyAccount")).Return(errors.New("database error"))

	c, w := setupFamilyTestContext("POST", "/api/v1/family", reqBody)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.CreateFamily(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetMyFamily Handler Tests
// ============================================================================

func TestHandler_GetMyFamily_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	family := createTestFamilyAccount(userID)
	member := createTestFamilyMember(family.ID, userID, MemberRoleOwner)
	members := []FamilyMember{*member}

	mockRepo.On("GetFamilyForUser", mock.Anything, userID).Return(family, member, nil)
	mockRepo.On("GetMembersByFamily", mock.Anything, family.ID).Return(members, nil)

	c, w := setupFamilyTestContext("GET", "/api/v1/family/me", nil)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.GetMyFamily(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetMyFamily_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	c, w := setupFamilyTestContext("GET", "/api/v1/family/me", nil)
	// Don't set user context

	handler.GetMyFamily(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyFamily_NotInFamily(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetFamilyForUser", mock.Anything, userID).Return(nil, nil, nil)

	c, w := setupFamilyTestContext("GET", "/api/v1/family/me", nil)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.GetMyFamily(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetMyFamily_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetFamilyForUser", mock.Anything, userID).Return(nil, nil, errors.New("database error"))

	c, w := setupFamilyTestContext("GET", "/api/v1/family/me", nil)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.GetMyFamily(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetFamily Handler Tests
// ============================================================================

func TestHandler_GetFamily_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	family := createTestFamilyAccount(userID)
	member := createTestFamilyMember(family.ID, userID, MemberRoleOwner)
	members := []FamilyMember{*member}

	mockRepo.On("GetFamilyByID", mock.Anything, family.ID).Return(family, nil)
	mockRepo.On("GetMemberByUserID", mock.Anything, family.ID, userID).Return(member, nil)
	mockRepo.On("GetMembersByFamily", mock.Anything, family.ID).Return(members, nil)

	c, w := setupFamilyTestContext("GET", "/api/v1/family/"+family.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: family.ID.String()}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.GetFamily(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetFamily_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()

	c, w := setupFamilyTestContext("GET", "/api/v1/family/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.GetFamily(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetFamily_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	familyID := uuid.New()

	mockRepo.On("GetFamilyByID", mock.Anything, familyID).Return(nil, common.NewNotFoundError("family not found", nil))

	c, w := setupFamilyTestContext("GET", "/api/v1/family/"+familyID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: familyID.String()}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.GetFamily(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetFamily_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	familyID := uuid.New()

	c, w := setupFamilyTestContext("GET", "/api/v1/family/"+familyID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: familyID.String()}}
	// Don't set user context

	handler.GetFamily(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// UpdateFamily Handler Tests
// ============================================================================

func TestHandler_UpdateFamily_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	family := createTestFamilyAccount(userID)
	newName := "Updated Family Name"
	reqBody := UpdateFamilyRequest{
		Name: &newName,
	}

	mockRepo.On("GetFamilyByID", mock.Anything, family.ID).Return(family, nil)
	mockRepo.On("UpdateFamily", mock.Anything, mock.AnythingOfType("*family.FamilyAccount")).Return(nil)

	c, w := setupFamilyTestContext("PUT", "/api/v1/family/"+family.ID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: family.ID.String()}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.UpdateFamily(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_UpdateFamily_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	family := createTestFamilyAccount(ownerID)
	newName := "Updated Family Name"
	reqBody := UpdateFamilyRequest{
		Name: &newName,
	}

	mockRepo.On("GetFamilyByID", mock.Anything, family.ID).Return(family, nil)

	c, w := setupFamilyTestContext("PUT", "/api/v1/family/"+family.ID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: family.ID.String()}}
	setFamilyUserContext(c, otherUserID, models.RoleRider)

	handler.UpdateFamily(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_UpdateFamily_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	newName := "Updated Family Name"
	reqBody := UpdateFamilyRequest{
		Name: &newName,
	}

	c, w := setupFamilyTestContext("PUT", "/api/v1/family/invalid-uuid", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.UpdateFamily(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateFamily_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	familyID := uuid.New()

	c, w := setupFamilyTestContext("PUT", "/api/v1/family/"+familyID.String(), nil)
	c.Request = httptest.NewRequest("PUT", "/api/v1/family/"+familyID.String(), bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: familyID.String()}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.UpdateFamily(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// InviteMember Handler Tests
// ============================================================================

func TestHandler_InviteMember_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	family := createTestFamilyAccount(userID)
	inviteeID := uuid.New()
	email := "invitee@example.com"
	reqBody := InviteMemberRequest{
		Email:       &email,
		Role:        MemberRoleAdult,
		DisplayName: "Invitee",
	}

	member := createTestFamilyMember(family.ID, userID, MemberRoleOwner)
	members := []FamilyMember{*member}

	mockRepo.On("GetFamilyByID", mock.Anything, family.ID).Return(family, nil)
	mockRepo.On("GetMembersByFamily", mock.Anything, family.ID).Return(members, nil)
	mockRepo.On("GetMemberByUserID", mock.Anything, family.ID, inviteeID).Return(nil, errors.New("not found"))
	mockRepo.On("CreateInvite", mock.Anything, mock.AnythingOfType("*family.FamilyInvite")).Return(nil)

	c, w := setupFamilyTestContext("POST", "/api/v1/family/"+family.ID.String()+"/invites", reqBody)
	c.Params = gin.Params{{Key: "id", Value: family.ID.String()}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.InviteMember(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_InviteMember_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	familyID := uuid.New()
	email := "invitee@example.com"
	reqBody := InviteMemberRequest{
		Email:       &email,
		Role:        MemberRoleAdult,
		DisplayName: "Invitee",
	}

	c, w := setupFamilyTestContext("POST", "/api/v1/family/"+familyID.String()+"/invites", reqBody)
	c.Params = gin.Params{{Key: "id", Value: familyID.String()}}
	// Don't set user context

	handler.InviteMember(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_InviteMember_InvalidFamilyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	email := "invitee@example.com"
	reqBody := InviteMemberRequest{
		Email:       &email,
		Role:        MemberRoleAdult,
		DisplayName: "Invitee",
	}

	c, w := setupFamilyTestContext("POST", "/api/v1/family/invalid-uuid/invites", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.InviteMember(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_InviteMember_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	family := createTestFamilyAccount(ownerID)
	email := "invitee@example.com"
	reqBody := InviteMemberRequest{
		Email:       &email,
		Role:        MemberRoleAdult,
		DisplayName: "Invitee",
	}

	mockRepo.On("GetFamilyByID", mock.Anything, family.ID).Return(family, nil)

	c, w := setupFamilyTestContext("POST", "/api/v1/family/"+family.ID.String()+"/invites", reqBody)
	c.Params = gin.Params{{Key: "id", Value: family.ID.String()}}
	setFamilyUserContext(c, otherUserID, models.RoleRider)

	handler.InviteMember(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ============================================================================
// GetPendingInvites Handler Tests
// ============================================================================

func TestHandler_GetPendingInvites_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	familyID := uuid.New()
	invite := createTestFamilyInvite(familyID, uuid.New())
	invites := []FamilyInvite{*invite}

	mockRepo.On("GetPendingInvitesForUser", mock.Anything, userID, (*string)(nil)).Return(invites, nil)

	c, w := setupFamilyTestContext("GET", "/api/v1/family/invites", nil)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.GetPendingInvites(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetPendingInvites_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	c, w := setupFamilyTestContext("GET", "/api/v1/family/invites", nil)
	// Don't set user context

	handler.GetPendingInvites(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetPendingInvites_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetPendingInvitesForUser", mock.Anything, userID, (*string)(nil)).Return([]FamilyInvite{}, nil)

	c, w := setupFamilyTestContext("GET", "/api/v1/family/invites", nil)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.GetPendingInvites(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetPendingInvites_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetPendingInvitesForUser", mock.Anything, userID, (*string)(nil)).Return(nil, errors.New("database error"))

	c, w := setupFamilyTestContext("GET", "/api/v1/family/invites", nil)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.GetPendingInvites(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// RespondToInvite Handler Tests
// ============================================================================

func TestHandler_RespondToInvite_AcceptSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	familyID := uuid.New()
	invite := createTestFamilyInvite(familyID, uuid.New())
	invite.InviteeID = &userID
	reqBody := RespondToInviteRequest{
		Accept: true,
	}

	mockRepo.On("GetInviteByID", mock.Anything, invite.ID).Return(invite, nil)
	mockRepo.On("GetFamilyForUser", mock.Anything, userID).Return(nil, nil, nil)
	mockRepo.On("GetMembersByFamily", mock.Anything, familyID).Return([]FamilyMember{}, nil)
	mockRepo.On("AddMember", mock.Anything, mock.AnythingOfType("*family.FamilyMember")).Return(nil)
	mockRepo.On("UpdateInviteStatus", mock.Anything, invite.ID, InviteStatusAccepted).Return(nil)

	c, w := setupFamilyTestContext("POST", "/api/v1/family/invites/"+invite.ID.String()+"/respond", reqBody)
	c.Params = gin.Params{{Key: "inviteId", Value: invite.ID.String()}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.RespondToInvite(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_RespondToInvite_DeclineSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	familyID := uuid.New()
	invite := createTestFamilyInvite(familyID, uuid.New())
	invite.InviteeID = &userID
	reqBody := RespondToInviteRequest{
		Accept: false,
	}

	mockRepo.On("GetInviteByID", mock.Anything, invite.ID).Return(invite, nil)
	mockRepo.On("UpdateInviteStatus", mock.Anything, invite.ID, InviteStatusDeclined).Return(nil)

	c, w := setupFamilyTestContext("POST", "/api/v1/family/invites/"+invite.ID.String()+"/respond", reqBody)
	c.Params = gin.Params{{Key: "inviteId", Value: invite.ID.String()}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.RespondToInvite(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_RespondToInvite_InvalidInviteID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	reqBody := RespondToInviteRequest{
		Accept: true,
	}

	c, w := setupFamilyTestContext("POST", "/api/v1/family/invites/invalid-uuid/respond", reqBody)
	c.Params = gin.Params{{Key: "inviteId", Value: "invalid-uuid"}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.RespondToInvite(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RespondToInvite_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	inviteID := uuid.New()
	reqBody := RespondToInviteRequest{
		Accept: true,
	}

	c, w := setupFamilyTestContext("POST", "/api/v1/family/invites/"+inviteID.String()+"/respond", reqBody)
	c.Params = gin.Params{{Key: "inviteId", Value: inviteID.String()}}
	// Don't set user context

	handler.RespondToInvite(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RespondToInvite_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	inviteID := uuid.New()
	reqBody := RespondToInviteRequest{
		Accept: true,
	}

	mockRepo.On("GetInviteByID", mock.Anything, inviteID).Return(nil, common.NewNotFoundError("invite not found", nil))

	c, w := setupFamilyTestContext("POST", "/api/v1/family/invites/"+inviteID.String()+"/respond", reqBody)
	c.Params = gin.Params{{Key: "inviteId", Value: inviteID.String()}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.RespondToInvite(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// UpdateMember Handler Tests
// ============================================================================

func TestHandler_UpdateMember_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	ownerID := uuid.New()
	family := createTestFamilyAccount(ownerID)
	member := createTestFamilyMember(family.ID, uuid.New(), MemberRoleAdult)
	useShared := false
	reqBody := UpdateMemberRequest{
		UseSharedPayment: &useShared,
	}

	ownerMember := createTestFamilyMember(family.ID, ownerID, MemberRoleOwner)
	members := []FamilyMember{*ownerMember, *member}

	mockRepo.On("GetFamilyByID", mock.Anything, family.ID).Return(family, nil)
	mockRepo.On("GetMembersByFamily", mock.Anything, family.ID).Return(members, nil)
	mockRepo.On("UpdateMember", mock.Anything, mock.AnythingOfType("*family.FamilyMember")).Return(nil)

	c, w := setupFamilyTestContext("PUT", "/api/v1/family/"+family.ID.String()+"/members/"+member.ID.String(), reqBody)
	c.Params = gin.Params{
		{Key: "id", Value: family.ID.String()},
		{Key: "memberId", Value: member.ID.String()},
	}
	setFamilyUserContext(c, ownerID, models.RoleRider)

	handler.UpdateMember(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_UpdateMember_InvalidFamilyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	memberID := uuid.New()
	useShared := false
	reqBody := UpdateMemberRequest{
		UseSharedPayment: &useShared,
	}

	c, w := setupFamilyTestContext("PUT", "/api/v1/family/invalid-uuid/members/"+memberID.String(), reqBody)
	c.Params = gin.Params{
		{Key: "id", Value: "invalid-uuid"},
		{Key: "memberId", Value: memberID.String()},
	}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.UpdateMember(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateMember_InvalidMemberID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	familyID := uuid.New()
	useShared := false
	reqBody := UpdateMemberRequest{
		UseSharedPayment: &useShared,
	}

	c, w := setupFamilyTestContext("PUT", "/api/v1/family/"+familyID.String()+"/members/invalid-uuid", reqBody)
	c.Params = gin.Params{
		{Key: "id", Value: familyID.String()},
		{Key: "memberId", Value: "invalid-uuid"},
	}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.UpdateMember(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateMember_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	family := createTestFamilyAccount(ownerID)
	memberID := uuid.New()
	useShared := false
	reqBody := UpdateMemberRequest{
		UseSharedPayment: &useShared,
	}

	mockRepo.On("GetFamilyByID", mock.Anything, family.ID).Return(family, nil)

	c, w := setupFamilyTestContext("PUT", "/api/v1/family/"+family.ID.String()+"/members/"+memberID.String(), reqBody)
	c.Params = gin.Params{
		{Key: "id", Value: family.ID.String()},
		{Key: "memberId", Value: memberID.String()},
	}
	setFamilyUserContext(c, otherUserID, models.RoleRider)

	handler.UpdateMember(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ============================================================================
// RemoveMember Handler Tests
// ============================================================================

func TestHandler_RemoveMember_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	ownerID := uuid.New()
	family := createTestFamilyAccount(ownerID)
	member := createTestFamilyMember(family.ID, uuid.New(), MemberRoleAdult)

	ownerMember := createTestFamilyMember(family.ID, ownerID, MemberRoleOwner)
	members := []FamilyMember{*ownerMember, *member}

	mockRepo.On("GetFamilyByID", mock.Anything, family.ID).Return(family, nil)
	mockRepo.On("GetMembersByFamily", mock.Anything, family.ID).Return(members, nil)
	mockRepo.On("RemoveMember", mock.Anything, member.ID).Return(nil)

	c, w := setupFamilyTestContext("DELETE", "/api/v1/family/"+family.ID.String()+"/members/"+member.ID.String(), nil)
	c.Params = gin.Params{
		{Key: "id", Value: family.ID.String()},
		{Key: "memberId", Value: member.ID.String()},
	}
	setFamilyUserContext(c, ownerID, models.RoleRider)

	handler.RemoveMember(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_RemoveMember_InvalidFamilyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	memberID := uuid.New()

	c, w := setupFamilyTestContext("DELETE", "/api/v1/family/invalid-uuid/members/"+memberID.String(), nil)
	c.Params = gin.Params{
		{Key: "id", Value: "invalid-uuid"},
		{Key: "memberId", Value: memberID.String()},
	}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.RemoveMember(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RemoveMember_InvalidMemberID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	familyID := uuid.New()

	c, w := setupFamilyTestContext("DELETE", "/api/v1/family/"+familyID.String()+"/members/invalid-uuid", nil)
	c.Params = gin.Params{
		{Key: "id", Value: familyID.String()},
		{Key: "memberId", Value: "invalid-uuid"},
	}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.RemoveMember(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RemoveMember_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	familyID := uuid.New()
	memberID := uuid.New()

	c, w := setupFamilyTestContext("DELETE", "/api/v1/family/"+familyID.String()+"/members/"+memberID.String(), nil)
	c.Params = gin.Params{
		{Key: "id", Value: familyID.String()},
		{Key: "memberId", Value: memberID.String()},
	}
	// Don't set user context

	handler.RemoveMember(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// LeaveFamily Handler Tests
// ============================================================================

func TestHandler_LeaveFamily_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	ownerID := uuid.New()
	userID := uuid.New()
	family := createTestFamilyAccount(ownerID)
	member := createTestFamilyMember(family.ID, userID, MemberRoleAdult)

	mockRepo.On("GetFamilyForUser", mock.Anything, userID).Return(family, member, nil)
	mockRepo.On("RemoveMember", mock.Anything, member.ID).Return(nil)

	c, w := setupFamilyTestContext("POST", "/api/v1/family/leave", nil)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.LeaveFamily(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_LeaveFamily_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	c, w := setupFamilyTestContext("POST", "/api/v1/family/leave", nil)
	// Don't set user context

	handler.LeaveFamily(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_LeaveFamily_NotInFamily(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetFamilyForUser", mock.Anything, userID).Return(nil, nil, nil)

	c, w := setupFamilyTestContext("POST", "/api/v1/family/leave", nil)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.LeaveFamily(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_LeaveFamily_OwnerCannotLeave(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	family := createTestFamilyAccount(userID)
	member := createTestFamilyMember(family.ID, userID, MemberRoleOwner)

	mockRepo.On("GetFamilyForUser", mock.Anything, userID).Return(family, member, nil)

	c, w := setupFamilyTestContext("POST", "/api/v1/family/leave", nil)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.LeaveFamily(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// AuthorizeRide Handler Tests
// ============================================================================

func TestHandler_AuthorizeRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	family := createTestFamilyAccount(userID)
	member := createTestFamilyMember(family.ID, userID, MemberRoleOwner)
	reqBody := struct {
		FareAmount float64 `json:"fare_amount"`
	}{
		FareAmount: 25.0,
	}

	mockRepo.On("GetFamilyForUser", mock.Anything, userID).Return(family, member, nil)

	c, w := setupFamilyTestContext("POST", "/api/v1/family/authorize-ride", reqBody)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.AuthorizeRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_AuthorizeRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	reqBody := struct {
		FareAmount float64 `json:"fare_amount"`
	}{
		FareAmount: 25.0,
	}

	c, w := setupFamilyTestContext("POST", "/api/v1/family/authorize-ride", reqBody)
	// Don't set user context

	handler.AuthorizeRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AuthorizeRide_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()

	c, w := setupFamilyTestContext("POST", "/api/v1/family/authorize-ride", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/family/authorize-ride", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.AuthorizeRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AuthorizeRide_NotInFamily(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	reqBody := struct {
		FareAmount float64 `json:"fare_amount"`
	}{
		FareAmount: 25.0,
	}

	mockRepo.On("GetFamilyForUser", mock.Anything, userID).Return(nil, nil, nil)

	c, w := setupFamilyTestContext("POST", "/api/v1/family/authorize-ride", reqBody)
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.AuthorizeRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	// User not in family - still authorized with personal payment
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
}

// ============================================================================
// GetSpendingReport Handler Tests
// ============================================================================

func TestHandler_GetSpendingReport_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()
	family := createTestFamilyAccount(userID)
	member := createTestFamilyMember(family.ID, userID, MemberRoleOwner)
	memberSpending := []MemberSpend{
		{MemberID: member.ID, DisplayName: "Owner", Amount: 100.0, RideCount: 5},
	}

	mockRepo.On("GetFamilyByID", mock.Anything, family.ID).Return(family, nil)
	mockRepo.On("GetMemberByUserID", mock.Anything, family.ID, userID).Return(member, nil)
	mockRepo.On("GetSpendingReport", mock.Anything, family.ID).Return(memberSpending, 5, 100.0, nil)

	c, w := setupFamilyTestContext("GET", "/api/v1/family/"+family.ID.String()+"/spending", nil)
	c.Params = gin.Params{{Key: "id", Value: family.ID.String()}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.GetSpendingReport(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseFamilyResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetSpendingReport_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	userID := uuid.New()

	c, w := setupFamilyTestContext("GET", "/api/v1/family/invalid-uuid/spending", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setFamilyUserContext(c, userID, models.RoleRider)

	handler.GetSpendingReport(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetSpendingReport_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	familyID := uuid.New()

	c, w := setupFamilyTestContext("GET", "/api/v1/family/"+familyID.String()+"/spending", nil)
	c.Params = gin.Params{{Key: "id", Value: familyID.String()}}
	// Don't set user context

	handler.GetSpendingReport(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetSpendingReport_NotMember(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockFamilyRepository)
	handler := createTestFamilyHandler(mockRepo)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	family := createTestFamilyAccount(ownerID)

	mockRepo.On("GetFamilyByID", mock.Anything, family.ID).Return(family, nil)
	mockRepo.On("GetMemberByUserID", mock.Anything, family.ID, otherUserID).Return(nil, common.NewForbiddenError("not a member"))

	c, w := setupFamilyTestContext("GET", "/api/v1/family/"+family.ID.String()+"/spending", nil)
	c.Params = gin.Params{{Key: "id", Value: family.ID.String()}}
	setFamilyUserContext(c, otherUserID, models.RoleRider)

	handler.GetSpendingReport(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_CreateFamily_ValidationCases(t *testing.T) {
	tests := []struct {
		name           string
		reqBody        CreateFamilyRequest
		setupMocks     func(*MockFamilyRepository, uuid.UUID)
		expectedStatus int
	}{
		{
			name: "success with budget",
			reqBody: CreateFamilyRequest{
				Name:          "Family With Budget",
				MonthlyBudget: func() *float64 { v := 1000.0; return &v }(),
				Currency:      "USD",
			},
			setupMocks: func(m *MockFamilyRepository, userID uuid.UUID) {
				m.On("GetFamilyByOwner", mock.Anything, userID).Return(nil, pgx.ErrNoRows)
				m.On("GetFamilyForUser", mock.Anything, userID).Return(nil, nil, nil)
				m.On("CreateFamily", mock.Anything, mock.AnythingOfType("*family.FamilyAccount")).Return(nil)
				m.On("AddMember", mock.Anything, mock.AnythingOfType("*family.FamilyMember")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "success without budget",
			reqBody: CreateFamilyRequest{
				Name: "Family Without Budget",
			},
			setupMocks: func(m *MockFamilyRepository, userID uuid.UUID) {
				m.On("GetFamilyByOwner", mock.Anything, userID).Return(nil, pgx.ErrNoRows)
				m.On("GetFamilyForUser", mock.Anything, userID).Return(nil, nil, nil)
				m.On("CreateFamily", mock.Anything, mock.AnythingOfType("*family.FamilyAccount")).Return(nil)
				m.On("AddMember", mock.Anything, mock.AnythingOfType("*family.FamilyMember")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockFamilyRepository)
			handler := createTestFamilyHandler(mockRepo)
			userID := uuid.New()

			tt.setupMocks(mockRepo, userID)

			c, w := setupFamilyTestContext("POST", "/api/v1/family", tt.reqBody)
			setFamilyUserContext(c, userID, models.RoleRider)

			handler.CreateFamily(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_InviteMember_RoleCases(t *testing.T) {
	tests := []struct {
		name           string
		role           MemberRole
		expectedStatus int
	}{
		{
			name:           "invite as adult",
			role:           MemberRoleAdult,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invite as teen",
			role:           MemberRoleTeen,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invite as child",
			role:           MemberRoleChild,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "cannot invite as owner",
			role:           MemberRoleOwner,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockFamilyRepository)
			handler := createTestFamilyHandler(mockRepo)

			userID := uuid.New()
			family := createTestFamilyAccount(userID)
			email := "invitee@example.com"
			reqBody := InviteMemberRequest{
				Email:       &email,
				Role:        tt.role,
				DisplayName: "Invitee",
			}

			member := createTestFamilyMember(family.ID, userID, MemberRoleOwner)
			members := []FamilyMember{*member}

			mockRepo.On("GetFamilyByID", mock.Anything, family.ID).Return(family, nil)
			mockRepo.On("GetMembersByFamily", mock.Anything, family.ID).Return(members, nil)
			if tt.role != MemberRoleOwner {
				mockRepo.On("CreateInvite", mock.Anything, mock.AnythingOfType("*family.FamilyInvite")).Return(nil)
			}

			c, w := setupFamilyTestContext("POST", "/api/v1/family/"+family.ID.String()+"/invites", reqBody)
			c.Params = gin.Params{{Key: "id", Value: family.ID.String()}}
			setFamilyUserContext(c, userID, models.RoleRider)

			handler.InviteMember(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
