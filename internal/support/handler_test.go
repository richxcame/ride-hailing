package support

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
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Implementation
// ============================================================================

// MockSupportRepository implements RepositoryInterface for testing
type MockSupportRepository struct {
	mock.Mock
}

func (m *MockSupportRepository) CreateTicket(ctx context.Context, ticket *Ticket) error {
	args := m.Called(ctx, ticket)
	return args.Error(0)
}

func (m *MockSupportRepository) GetTicketByID(ctx context.Context, id uuid.UUID) (*Ticket, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Ticket), args.Error(1)
}

func (m *MockSupportRepository) GetUserTickets(ctx context.Context, userID uuid.UUID, status *TicketStatus, limit, offset int) ([]TicketSummary, int, error) {
	args := m.Called(ctx, userID, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]TicketSummary), args.Int(1), args.Error(2)
}

func (m *MockSupportRepository) GetAllTickets(ctx context.Context, status *TicketStatus, priority *TicketPriority, category *TicketCategory, limit, offset int) ([]TicketSummary, int, error) {
	args := m.Called(ctx, status, priority, category, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]TicketSummary), args.Int(1), args.Error(2)
}

func (m *MockSupportRepository) UpdateTicket(ctx context.Context, id uuid.UUID, status *TicketStatus, priority *TicketPriority, assignedTo *uuid.UUID, tags []string) error {
	args := m.Called(ctx, id, status, priority, assignedTo, tags)
	return args.Error(0)
}

func (m *MockSupportRepository) GetTicketStats(ctx context.Context) (*TicketStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TicketStats), args.Error(1)
}

func (m *MockSupportRepository) CreateMessage(ctx context.Context, msg *TicketMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockSupportRepository) GetMessagesByTicket(ctx context.Context, ticketID uuid.UUID, includeInternal bool) ([]TicketMessage, error) {
	args := m.Called(ctx, ticketID, includeInternal)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TicketMessage), args.Error(1)
}

func (m *MockSupportRepository) GetFAQArticles(ctx context.Context, category *string) ([]FAQArticle, error) {
	args := m.Called(ctx, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]FAQArticle), args.Error(1)
}

func (m *MockSupportRepository) GetFAQArticleByID(ctx context.Context, id uuid.UUID) (*FAQArticle, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FAQArticle), args.Error(1)
}

func (m *MockSupportRepository) IncrementFAQViewCount(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ============================================================================
// Helper Functions
// ============================================================================

func setupSupportTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func setSupportUserContext(c *gin.Context, userID uuid.UUID, role models.UserRole) {
	c.Set("user_id", userID)
	c.Set("user_role", role)
	c.Set("user_email", "test@example.com")
}

func parseSupportResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestSupportHandler(mockRepo *MockSupportRepository) *Handler {
	service := NewService(mockRepo)
	return NewHandler(service)
}

func createTestTicket(userID uuid.UUID) *Ticket {
	return &Ticket{
		ID:           uuid.New(),
		UserID:       userID,
		TicketNumber: "TKT-123456",
		Category:     CategoryPayment,
		Priority:     PriorityMedium,
		Status:       TicketStatusOpen,
		Subject:      "Test Ticket",
		Description:  "Test ticket description",
		Tags:         []string{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func createTestTicketSummary() TicketSummary {
	return TicketSummary{
		ID:           uuid.New(),
		TicketNumber: "TKT-123456",
		Category:     CategoryPayment,
		Priority:     PriorityMedium,
		Status:       TicketStatusOpen,
		Subject:      "Test Ticket",
		MessageCount: 1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func createTestTicketMessage(ticketID, senderID uuid.UUID) *TicketMessage {
	return &TicketMessage{
		ID:          uuid.New(),
		TicketID:    ticketID,
		SenderID:    senderID,
		SenderType:  SenderUser,
		Message:     "Test message",
		Attachments: []string{},
		IsInternal:  false,
		CreatedAt:   time.Now(),
	}
}

func createTestFAQArticle() *FAQArticle {
	return &FAQArticle{
		ID:        uuid.New(),
		Category:  "payment",
		Title:     "How to pay",
		Content:   "Payment instructions...",
		SortOrder: 1,
		IsActive:  true,
		ViewCount: 100,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ============================================================================
// CreateTicket Handler Tests
// ============================================================================

func TestHandler_CreateTicket_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	reqBody := CreateTicketRequest{
		Category:    CategoryPayment,
		Subject:     "Payment Issue",
		Description: "I have a payment issue that needs to be resolved",
	}

	mockRepo.On("CreateTicket", mock.Anything, mock.AnythingOfType("*support.Ticket")).Return(nil)
	mockRepo.On("CreateMessage", mock.Anything, mock.AnythingOfType("*support.TicketMessage")).Return(nil)

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets", reqBody)
	setSupportUserContext(c, userID, models.RoleRider)

	handler.CreateTicket(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CreateTicket_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	reqBody := CreateTicketRequest{
		Category:    CategoryPayment,
		Subject:     "Payment Issue",
		Description: "I have a payment issue",
	}

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets", reqBody)
	// Don't set user context

	handler.CreateTicket(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CreateTicket_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/support/tickets", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setSupportUserContext(c, userID, models.RoleRider)

	handler.CreateTicket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateTicket_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	reqBody := CreateTicketRequest{
		Category:    CategoryPayment,
		Subject:     "Payment Issue",
		Description: "I have a payment issue that needs to be resolved",
	}

	mockRepo.On("CreateTicket", mock.Anything, mock.AnythingOfType("*support.Ticket")).Return(errors.New("database error"))

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets", reqBody)
	setSupportUserContext(c, userID, models.RoleRider)

	handler.CreateTicket(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetMyTickets Handler Tests
// ============================================================================

func TestHandler_GetMyTickets_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	tickets := []TicketSummary{createTestTicketSummary()}

	mockRepo.On("GetUserTickets", mock.Anything, userID, (*TicketStatus)(nil), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(tickets, 1, nil)

	c, w := setupSupportTestContext("GET", "/api/v1/support/tickets", nil)
	setSupportUserContext(c, userID, models.RoleRider)

	handler.GetMyTickets(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetMyTickets_WithStatusFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	tickets := []TicketSummary{createTestTicketSummary()}
	openStatus := TicketStatusOpen

	mockRepo.On("GetUserTickets", mock.Anything, userID, &openStatus, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(tickets, 1, nil)

	c, w := setupSupportTestContext("GET", "/api/v1/support/tickets?status=open", nil)
	c.Request.URL.RawQuery = "status=open"
	setSupportUserContext(c, userID, models.RoleRider)

	handler.GetMyTickets(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetMyTickets_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	c, w := setupSupportTestContext("GET", "/api/v1/support/tickets", nil)
	// Don't set user context

	handler.GetMyTickets(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyTickets_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetUserTickets", mock.Anything, userID, (*TicketStatus)(nil), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil, 0, errors.New("database error"))

	c, w := setupSupportTestContext("GET", "/api/v1/support/tickets", nil)
	setSupportUserContext(c, userID, models.RoleRider)

	handler.GetMyTickets(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetTicket Handler Tests
// ============================================================================

func TestHandler_GetTicket_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	ticket := createTestTicket(userID)

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)

	c, w := setupSupportTestContext("GET", "/api/v1/support/tickets/"+ticket.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.GetTicket(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetTicket_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()

	c, w := setupSupportTestContext("GET", "/api/v1/support/tickets/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.GetTicket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetTicket_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	ticketID := uuid.New()

	mockRepo.On("GetTicketByID", mock.Anything, ticketID).Return(nil, common.NewNotFoundError("ticket not found", nil))

	c, w := setupSupportTestContext("GET", "/api/v1/support/tickets/"+ticketID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: ticketID.String()}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.GetTicket(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetTicket_Forbidden_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	ticket := createTestTicket(otherUserID)

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)

	c, w := setupSupportTestContext("GET", "/api/v1/support/tickets/"+ticket.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.GetTicket(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_GetTicket_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	ticketID := uuid.New()

	c, w := setupSupportTestContext("GET", "/api/v1/support/tickets/"+ticketID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: ticketID.String()}}
	// Don't set user context

	handler.GetTicket(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// GetTicketMessages Handler Tests
// ============================================================================

func TestHandler_GetTicketMessages_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	ticket := createTestTicket(userID)
	messages := []TicketMessage{*createTestTicketMessage(ticket.ID, userID)}

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)
	mockRepo.On("GetMessagesByTicket", mock.Anything, ticket.ID, false).Return(messages, nil)

	c, w := setupSupportTestContext("GET", "/api/v1/support/tickets/"+ticket.ID.String()+"/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.GetTicketMessages(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetTicketMessages_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()

	c, w := setupSupportTestContext("GET", "/api/v1/support/tickets/invalid-uuid/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.GetTicketMessages(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetTicketMessages_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	ticketID := uuid.New()

	c, w := setupSupportTestContext("GET", "/api/v1/support/tickets/"+ticketID.String()+"/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: ticketID.String()}}
	// Don't set user context

	handler.GetTicketMessages(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetTicketMessages_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	ticket := createTestTicket(otherUserID)

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)

	c, w := setupSupportTestContext("GET", "/api/v1/support/tickets/"+ticket.ID.String()+"/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.GetTicketMessages(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ============================================================================
// AddMessage Handler Tests
// ============================================================================

func TestHandler_AddMessage_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	ticket := createTestTicket(userID)
	reqBody := AddMessageRequest{
		Message: "This is my follow-up message",
	}

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)
	mockRepo.On("CreateMessage", mock.Anything, mock.AnythingOfType("*support.TicketMessage")).Return(nil)

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets/"+ticket.ID.String()+"/messages", reqBody)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.AddMessage(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_AddMessage_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	reqBody := AddMessageRequest{
		Message: "This is my follow-up message",
	}

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets/invalid-uuid/messages", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.AddMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddMessage_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	ticketID := uuid.New()

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets/"+ticketID.String()+"/messages", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/support/tickets/"+ticketID.String()+"/messages", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: ticketID.String()}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.AddMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddMessage_ClosedTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	ticket := createTestTicket(userID)
	ticket.Status = TicketStatusClosed
	reqBody := AddMessageRequest{
		Message: "Trying to message closed ticket",
	}

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets/"+ticket.ID.String()+"/messages", reqBody)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.AddMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddMessage_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	ticketID := uuid.New()
	reqBody := AddMessageRequest{
		Message: "Test message",
	}

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets/"+ticketID.String()+"/messages", reqBody)
	c.Params = gin.Params{{Key: "id", Value: ticketID.String()}}
	// Don't set user context

	handler.AddMessage(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// CloseTicket Handler Tests
// ============================================================================

func TestHandler_CloseTicket_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	ticket := createTestTicket(userID)

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)
	mockRepo.On("UpdateTicket", mock.Anything, ticket.ID, mock.AnythingOfType("*support.TicketStatus"), (*TicketPriority)(nil), (*uuid.UUID)(nil), ([]string)(nil)).Return(nil)

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets/"+ticket.ID.String()+"/close", nil)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.CloseTicket(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_CloseTicket_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets/invalid-uuid/close", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.CloseTicket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CloseTicket_AlreadyClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	ticket := createTestTicket(userID)
	ticket.Status = TicketStatusClosed

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets/"+ticket.ID.String()+"/close", nil)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.CloseTicket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CloseTicket_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	ticketID := uuid.New()

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets/"+ticketID.String()+"/close", nil)
	c.Params = gin.Params{{Key: "id", Value: ticketID.String()}}
	// Don't set user context

	handler.CloseTicket(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CloseTicket_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	ticket := createTestTicket(otherUserID)

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)

	c, w := setupSupportTestContext("POST", "/api/v1/support/tickets/"+ticket.ID.String()+"/close", nil)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, userID, models.RoleRider)

	handler.CloseTicket(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ============================================================================
// GetFAQArticles Handler Tests
// ============================================================================

func TestHandler_GetFAQArticles_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	articles := []FAQArticle{*createTestFAQArticle()}

	mockRepo.On("GetFAQArticles", mock.Anything, (*string)(nil)).Return(articles, nil)

	c, w := setupSupportTestContext("GET", "/api/v1/support/faq", nil)

	handler.GetFAQArticles(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetFAQArticles_WithCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	articles := []FAQArticle{*createTestFAQArticle()}
	category := "payment"

	mockRepo.On("GetFAQArticles", mock.Anything, &category).Return(articles, nil)

	c, w := setupSupportTestContext("GET", "/api/v1/support/faq?category=payment", nil)
	c.Request.URL.RawQuery = "category=payment"

	handler.GetFAQArticles(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetFAQArticles_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	mockRepo.On("GetFAQArticles", mock.Anything, (*string)(nil)).Return(nil, errors.New("database error"))

	c, w := setupSupportTestContext("GET", "/api/v1/support/faq", nil)

	handler.GetFAQArticles(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetFAQArticle Handler Tests
// ============================================================================

func TestHandler_GetFAQArticle_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	article := createTestFAQArticle()

	mockRepo.On("GetFAQArticleByID", mock.Anything, article.ID).Return(article, nil)
	mockRepo.On("IncrementFAQViewCount", mock.Anything, article.ID).Return(nil)

	c, w := setupSupportTestContext("GET", "/api/v1/support/faq/"+article.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: article.ID.String()}}

	handler.GetFAQArticle(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetFAQArticle_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	c, w := setupSupportTestContext("GET", "/api/v1/support/faq/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetFAQArticle(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetFAQArticle_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	articleID := uuid.New()

	mockRepo.On("GetFAQArticleByID", mock.Anything, articleID).Return(nil, common.NewNotFoundError("article not found", nil))

	c, w := setupSupportTestContext("GET", "/api/v1/support/faq/"+articleID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: articleID.String()}}

	handler.GetFAQArticle(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// AdminGetTickets Handler Tests
// ============================================================================

func TestHandler_AdminGetTickets_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	tickets := []TicketSummary{createTestTicketSummary()}

	mockRepo.On("GetAllTickets", mock.Anything, (*TicketStatus)(nil), (*TicketPriority)(nil), (*TicketCategory)(nil), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(tickets, 1, nil)

	c, w := setupSupportTestContext("GET", "/api/v1/admin/support/tickets", nil)
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetTickets(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_AdminGetTickets_WithFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	tickets := []TicketSummary{createTestTicketSummary()}
	openStatus := TicketStatusOpen
	urgentPriority := PriorityUrgent

	mockRepo.On("GetAllTickets", mock.Anything, &openStatus, &urgentPriority, (*TicketCategory)(nil), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(tickets, 1, nil)

	c, w := setupSupportTestContext("GET", "/api/v1/admin/support/tickets?status=open&priority=urgent", nil)
	c.Request.URL.RawQuery = "status=open&priority=urgent"
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetTickets(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_AdminGetTickets_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()

	mockRepo.On("GetAllTickets", mock.Anything, (*TicketStatus)(nil), (*TicketPriority)(nil), (*TicketCategory)(nil), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil, 0, errors.New("database error"))

	c, w := setupSupportTestContext("GET", "/api/v1/admin/support/tickets", nil)
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetTickets(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// AdminGetTicket Handler Tests
// ============================================================================

func TestHandler_AdminGetTicket_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	userID := uuid.New()
	ticket := createTestTicket(userID)

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)

	c, w := setupSupportTestContext("GET", "/api/v1/admin/support/tickets/"+ticket.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetTicket(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_AdminGetTicket_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()

	c, w := setupSupportTestContext("GET", "/api/v1/admin/support/tickets/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetTicket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminGetTicket_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	ticketID := uuid.New()

	mockRepo.On("GetTicketByID", mock.Anything, ticketID).Return(nil, common.NewNotFoundError("ticket not found", nil))

	c, w := setupSupportTestContext("GET", "/api/v1/admin/support/tickets/"+ticketID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: ticketID.String()}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetTicket(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// AdminGetMessages Handler Tests
// ============================================================================

func TestHandler_AdminGetMessages_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	userID := uuid.New()
	ticket := createTestTicket(userID)
	messages := []TicketMessage{*createTestTicketMessage(ticket.ID, userID)}

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)
	mockRepo.On("GetMessagesByTicket", mock.Anything, ticket.ID, true).Return(messages, nil)

	c, w := setupSupportTestContext("GET", "/api/v1/admin/support/tickets/"+ticket.ID.String()+"/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetMessages(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_AdminGetMessages_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()

	c, w := setupSupportTestContext("GET", "/api/v1/admin/support/tickets/invalid-uuid/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetMessages(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// AdminReply Handler Tests
// ============================================================================

func TestHandler_AdminReply_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	userID := uuid.New()
	ticket := createTestTicket(userID)
	reqBody := AdminReplyRequest{
		Message:    "Thank you for contacting us. We will look into this.",
		IsInternal: false,
	}

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)
	mockRepo.On("CreateMessage", mock.Anything, mock.AnythingOfType("*support.TicketMessage")).Return(nil)
	mockRepo.On("UpdateTicket", mock.Anything, ticket.ID, mock.AnythingOfType("*support.TicketStatus"), (*TicketPriority)(nil), mock.AnythingOfType("*uuid.UUID"), ([]string)(nil)).Return(nil)

	c, w := setupSupportTestContext("POST", "/api/v1/admin/support/tickets/"+ticket.ID.String()+"/reply", reqBody)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminReply(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_AdminReply_InternalNote(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	userID := uuid.New()
	ticket := createTestTicket(userID)
	reqBody := AdminReplyRequest{
		Message:    "Internal note: Investigating payment system",
		IsInternal: true,
	}

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)
	mockRepo.On("CreateMessage", mock.Anything, mock.AnythingOfType("*support.TicketMessage")).Return(nil)
	mockRepo.On("UpdateTicket", mock.Anything, ticket.ID, mock.AnythingOfType("*support.TicketStatus"), (*TicketPriority)(nil), mock.AnythingOfType("*uuid.UUID"), ([]string)(nil)).Return(nil)

	c, w := setupSupportTestContext("POST", "/api/v1/admin/support/tickets/"+ticket.ID.String()+"/reply", reqBody)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminReply(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_AdminReply_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	reqBody := AdminReplyRequest{
		Message: "Test reply",
	}

	c, w := setupSupportTestContext("POST", "/api/v1/admin/support/tickets/invalid-uuid/reply", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminReply(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminReply_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	ticketID := uuid.New()

	c, w := setupSupportTestContext("POST", "/api/v1/admin/support/tickets/"+ticketID.String()+"/reply", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/support/tickets/"+ticketID.String()+"/reply", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: ticketID.String()}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminReply(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminReply_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	ticketID := uuid.New()
	reqBody := AdminReplyRequest{
		Message: "Test reply",
	}

	c, w := setupSupportTestContext("POST", "/api/v1/admin/support/tickets/"+ticketID.String()+"/reply", reqBody)
	c.Params = gin.Params{{Key: "id", Value: ticketID.String()}}
	// Don't set user context

	handler.AdminReply(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AdminReply_ClosedTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	userID := uuid.New()
	ticket := createTestTicket(userID)
	ticket.Status = TicketStatusClosed
	reqBody := AdminReplyRequest{
		Message: "Test reply",
	}

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)

	c, w := setupSupportTestContext("POST", "/api/v1/admin/support/tickets/"+ticket.ID.String()+"/reply", reqBody)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminReply(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// AdminUpdateTicket Handler Tests
// ============================================================================

func TestHandler_AdminUpdateTicket_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	userID := uuid.New()
	ticket := createTestTicket(userID)
	inProgress := TicketStatusInProgress
	reqBody := UpdateTicketRequest{
		Status: &inProgress,
	}

	mockRepo.On("GetTicketByID", mock.Anything, ticket.ID).Return(ticket, nil)
	mockRepo.On("UpdateTicket", mock.Anything, ticket.ID, &inProgress, (*TicketPriority)(nil), (*uuid.UUID)(nil), ([]string)(nil)).Return(nil)

	c, w := setupSupportTestContext("PUT", "/api/v1/admin/support/tickets/"+ticket.ID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: ticket.ID.String()}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminUpdateTicket(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_AdminUpdateTicket_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	inProgress := TicketStatusInProgress
	reqBody := UpdateTicketRequest{
		Status: &inProgress,
	}

	c, w := setupSupportTestContext("PUT", "/api/v1/admin/support/tickets/invalid-uuid", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminUpdateTicket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminUpdateTicket_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	ticketID := uuid.New()

	c, w := setupSupportTestContext("PUT", "/api/v1/admin/support/tickets/"+ticketID.String(), nil)
	c.Request = httptest.NewRequest("PUT", "/api/v1/admin/support/tickets/"+ticketID.String(), bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: ticketID.String()}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminUpdateTicket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminUpdateTicket_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	ticketID := uuid.New()
	inProgress := TicketStatusInProgress
	reqBody := UpdateTicketRequest{
		Status: &inProgress,
	}

	mockRepo.On("GetTicketByID", mock.Anything, ticketID).Return(nil, common.NewNotFoundError("ticket not found", nil))

	c, w := setupSupportTestContext("PUT", "/api/v1/admin/support/tickets/"+ticketID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: ticketID.String()}}
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminUpdateTicket(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// AdminGetStats Handler Tests
// ============================================================================

func TestHandler_AdminGetStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()
	stats := &TicketStats{
		TotalOpen:        10,
		TotalInProgress:  5,
		TotalWaiting:     3,
		TotalResolved:    100,
		TotalEscalated:   2,
		AvgResolutionMin: 45.5,
		ByCategory:       map[string]int{"payment": 50, "ride": 30},
		ByPriority:       map[string]int{"high": 10, "medium": 50},
	}

	mockRepo.On("GetTicketStats", mock.Anything).Return(stats, nil)

	c, w := setupSupportTestContext("GET", "/api/v1/admin/support/stats", nil)
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseSupportResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_AdminGetStats_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSupportRepository)
	handler := createTestSupportHandler(mockRepo)

	adminID := uuid.New()

	mockRepo.On("GetTicketStats", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupSupportTestContext("GET", "/api/v1/admin/support/stats", nil)
	setSupportUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetStats(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_CreateTicket_CategoryCases(t *testing.T) {
	tests := []struct {
		name             string
		category         TicketCategory
		expectedPriority TicketPriority
		expectedStatus   int
	}{
		{
			name:             "payment category",
			category:         CategoryPayment,
			expectedPriority: PriorityMedium,
			expectedStatus:   http.StatusCreated,
		},
		{
			name:             "safety category auto-escalates",
			category:         CategorySafety,
			expectedPriority: PriorityUrgent,
			expectedStatus:   http.StatusCreated,
		},
		{
			name:             "ride category",
			category:         CategoryRide,
			expectedPriority: PriorityMedium,
			expectedStatus:   http.StatusCreated,
		},
		{
			name:             "lost item category",
			category:         CategoryLostItem,
			expectedPriority: PriorityMedium,
			expectedStatus:   http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockSupportRepository)
			handler := createTestSupportHandler(mockRepo)
			userID := uuid.New()

			reqBody := CreateTicketRequest{
				Category:    tt.category,
				Subject:     "Test Subject",
				Description: "Test description that is long enough",
			}

			mockRepo.On("CreateTicket", mock.Anything, mock.MatchedBy(func(ticket *Ticket) bool {
				return ticket.Priority == tt.expectedPriority
			})).Return(nil)
			mockRepo.On("CreateMessage", mock.Anything, mock.AnythingOfType("*support.TicketMessage")).Return(nil)

			c, w := setupSupportTestContext("POST", "/api/v1/support/tickets", reqBody)
			setSupportUserContext(c, userID, models.RoleRider)

			handler.CreateTicket(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_AdminGetTickets_FilterCases(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		status     *TicketStatus
		priority   *TicketPriority
		category   *TicketCategory
	}{
		{
			name:     "no filters",
			queryStr: "",
			status:   nil,
			priority: nil,
			category: nil,
		},
		{
			name:     "status filter only",
			queryStr: "status=open",
			status:   func() *TicketStatus { s := TicketStatusOpen; return &s }(),
			priority: nil,
			category: nil,
		},
		{
			name:     "priority filter only",
			queryStr: "priority=urgent",
			status:   nil,
			priority: func() *TicketPriority { p := PriorityUrgent; return &p }(),
			category: nil,
		},
		{
			name:     "category filter only",
			queryStr: "category=payment",
			status:   nil,
			priority: nil,
			category: func() *TicketCategory { c := CategoryPayment; return &c }(),
		},
		{
			name:     "all filters",
			queryStr: "status=open&priority=urgent&category=safety",
			status:   func() *TicketStatus { s := TicketStatusOpen; return &s }(),
			priority: func() *TicketPriority { p := PriorityUrgent; return &p }(),
			category: func() *TicketCategory { c := CategorySafety; return &c }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockSupportRepository)
			handler := createTestSupportHandler(mockRepo)
			adminID := uuid.New()

			mockRepo.On("GetAllTickets", mock.Anything, tt.status, tt.priority, tt.category, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return([]TicketSummary{}, 0, nil)

			c, w := setupSupportTestContext("GET", "/api/v1/admin/support/tickets?"+tt.queryStr, nil)
			c.Request.URL.RawQuery = tt.queryStr
			setSupportUserContext(c, adminID, models.RoleAdmin)

			handler.AdminGetTickets(c)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
