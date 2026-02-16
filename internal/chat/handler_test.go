package chat

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
	"github.com/richxcame/ride-hailing/pkg/models"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ========================================
// MOCK IMPLEMENTATIONS
// ========================================

// MockRepository is a mock implementation of RepositoryInterface for handler tests
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SaveMessage(ctx context.Context, msg *ChatMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockRepository) GetMessagesByRide(ctx context.Context, rideID uuid.UUID, limit, offset int) ([]ChatMessage, error) {
	args := m.Called(ctx, rideID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ChatMessage), args.Error(1)
}

func (m *MockRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*ChatMessage, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChatMessage), args.Error(1)
}

func (m *MockRepository) MarkMessagesDelivered(ctx context.Context, rideID, recipientID uuid.UUID) error {
	args := m.Called(ctx, rideID, recipientID)
	return args.Error(0)
}

func (m *MockRepository) MarkMessagesRead(ctx context.Context, rideID, recipientID, lastReadID uuid.UUID) error {
	args := m.Called(ctx, rideID, recipientID, lastReadID)
	return args.Error(0)
}

func (m *MockRepository) GetUnreadCount(ctx context.Context, rideID, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, rideID, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetLastMessage(ctx context.Context, rideID uuid.UUID) (*ChatMessage, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChatMessage), args.Error(1)
}

func (m *MockRepository) DeleteMessagesByRide(ctx context.Context, rideID uuid.UUID) error {
	args := m.Called(ctx, rideID)
	return args.Error(0)
}

func (m *MockRepository) GetMessageCount(ctx context.Context, rideID uuid.UUID) (int, error) {
	args := m.Called(ctx, rideID)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetQuickReplies(ctx context.Context, role string) ([]QuickReply, error) {
	args := m.Called(ctx, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]QuickReply), args.Error(1)
}

func (m *MockRepository) CreateQuickReply(ctx context.Context, qr *QuickReply) error {
	args := m.Called(ctx, qr)
	return args.Error(0)
}

func (m *MockRepository) GetActiveConversations(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

// MockHub is a mock implementation of HubInterface
type MockHub struct {
	mock.Mock
}

func (m *MockHub) SendToRide(rideID string, msg *ws.Message) {
	m.Called(rideID, msg)
}

func (m *MockHub) SendToUser(userID string, msg *ws.Message) {
	m.Called(userID, msg)
}

// ========================================
// TEST HELPERS
// ========================================

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

func setUserContext(c *gin.Context, userID uuid.UUID, role models.UserRole) {
	c.Set("user_id", userID)
	c.Set("user_role", role)
	c.Set("user_email", "test@example.com")
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestHandler(mockRepo *MockRepository, mockHub *MockHub) *Handler {
	service := NewService(mockRepo, mockHub)
	return NewHandler(service)
}

func createTestMessage(rideID, senderID uuid.UUID, role string, content string) *ChatMessage {
	now := time.Now()
	return &ChatMessage{
		ID:          uuid.New(),
		RideID:      rideID,
		SenderID:    senderID,
		SenderRole:  role,
		MessageType: MessageTypeText,
		Content:     content,
		Status:      MessageStatusSent,
		CreatedAt:   now,
	}
}

func createTestQuickReply(role, category, text string) QuickReply {
	return QuickReply{
		ID:        uuid.New(),
		Role:      role,
		Category:  category,
		Text:      text,
		SortOrder: 1,
		IsActive:  true,
	}
}

// ========================================
// SEND MESSAGE TESTS
// ========================================

func TestHandler_SendMessage_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()

	reqBody := SendMessageRequest{
		RideID:      rideID,
		MessageType: MessageTypeText,
		Content:     "Hello, driver!",
	}

	mockRepo.On("SaveMessage", mock.Anything, mock.AnythingOfType("*chat.ChatMessage")).Return(nil)
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).Return()

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockRepo.AssertExpectations(t)
	mockHub.AssertExpectations(t)
}

func TestHandler_SendMessage_SuccessWithImage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()
	imageURL := "https://example.com/image.jpg"

	reqBody := SendMessageRequest{
		RideID:      rideID,
		MessageType: MessageTypeImage,
		Content:     "Check this out",
		ImageURL:    &imageURL,
	}

	mockRepo.On("SaveMessage", mock.Anything, mock.AnythingOfType("*chat.ChatMessage")).Return(nil)
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).Return()

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_SendMessage_SuccessWithLocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()
	latitude := 37.7749
	longitude := -122.4194

	reqBody := SendMessageRequest{
		RideID:      rideID,
		MessageType: MessageTypeLocation,
		Content:     "I am here",
		Latitude:    &latitude,
		Longitude:   &longitude,
	}

	mockRepo.On("SaveMessage", mock.Anything, mock.AnythingOfType("*chat.ChatMessage")).Return(nil)
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).Return()

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_SendMessage_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	reqBody := SendMessageRequest{
		RideID:      uuid.New(),
		MessageType: MessageTypeText,
		Content:     "Hello",
	}

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	// Don't set user context to simulate unauthorized

	handler.SendMessage(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_SendMessage_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/chat/messages", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/chat/messages", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SendMessage_EmptyContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	reqBody := SendMessageRequest{
		RideID:      uuid.New(),
		MessageType: MessageTypeText,
		Content:     "",
	}

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SendMessage_InvalidMessageType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"ride_id":      uuid.New().String(),
		"message_type": "invalid_type",
		"content":      "Hello",
	}

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SendMessage_LocationMissingCoordinates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	reqBody := SendMessageRequest{
		RideID:      uuid.New(),
		MessageType: MessageTypeLocation,
		Content:     "I am here",
		// Missing Latitude and Longitude
	}

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "latitude and longitude")
}

func TestHandler_SendMessage_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	reqBody := SendMessageRequest{
		RideID:      uuid.New(),
		MessageType: MessageTypeText,
		Content:     "Hello",
	}

	mockRepo.On("SaveMessage", mock.Anything, mock.AnythingOfType("*chat.ChatMessage")).
		Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_SendMessage_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	// Content too long will trigger AppError
	longContent := make([]byte, 1001)
	for i := range longContent {
		longContent[i] = 'a'
	}

	reqBody := SendMessageRequest{
		RideID:      uuid.New(),
		MessageType: MessageTypeText,
		Content:     string(longContent),
	}

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "message too long")
}

func TestHandler_SendMessage_AsDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()

	reqBody := SendMessageRequest{
		RideID:      rideID,
		MessageType: MessageTypeText,
		Content:     "I am on my way!",
	}

	mockRepo.On("SaveMessage", mock.Anything, mock.MatchedBy(func(msg *ChatMessage) bool {
		return msg.SenderRole == "driver"
	})).Return(nil)
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).Return()

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_SendMessage_MissingRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	reqBody := SendMessageRequest{
		RideID:      uuid.New(),
		MessageType: MessageTypeText,
		Content:     "Hello",
	}

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	// Only set user_id, not user_role
	c.Set("user_id", userID)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ========================================
// GET CONVERSATION TESTS
// ========================================

func TestHandler_GetConversation_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()

	messages := []ChatMessage{
		*createTestMessage(rideID, uuid.New(), "driver", "Hello"),
		*createTestMessage(rideID, userID, "rider", "Hi there"),
	}
	lastMsg := &messages[1]

	mockRepo.On("GetMessagesByRide", mock.Anything, rideID, 20, 0).Return(messages, nil)
	mockRepo.On("GetUnreadCount", mock.Anything, rideID, userID).Return(1, nil)
	mockRepo.On("GetLastMessage", mock.Anything, rideID).Return(lastMsg, nil)
	mockRepo.On("MarkMessagesDelivered", mock.Anything, rideID, userID).Return(nil)
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).Return()

	c, w := setupTestContext("GET", "/api/v1/chat/rides/"+rideID.String()+"/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetConversation(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetConversation_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	rideID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/chat/rides/"+rideID.String()+"/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context

	handler.GetConversation(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetConversation_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/chat/rides/invalid-uuid/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetConversation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid ride ID")
}

func TestHandler_GetConversation_EmptyRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/chat/rides//messages", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetConversation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetConversation_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()

	mockRepo.On("GetMessagesByRide", mock.Anything, rideID, 20, 0).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/chat/rides/"+rideID.String()+"/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetConversation(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetConversation_WithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()

	messages := []ChatMessage{}

	mockRepo.On("GetMessagesByRide", mock.Anything, rideID, 10, 5).Return(messages, nil)
	mockRepo.On("GetUnreadCount", mock.Anything, rideID, userID).Return(0, nil)
	mockRepo.On("GetLastMessage", mock.Anything, rideID).Return(nil, nil)
	mockRepo.On("MarkMessagesDelivered", mock.Anything, rideID, userID).Return(nil)

	c, w := setupTestContext("GET", "/api/v1/chat/rides/"+rideID.String()+"/messages?limit=10&offset=5", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	c.Request.URL.RawQuery = "limit=10&offset=5"
	setUserContext(c, userID, models.RoleRider)

	handler.GetConversation(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetConversation_EmptyMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()

	mockRepo.On("GetMessagesByRide", mock.Anything, rideID, 20, 0).Return(nil, nil)
	mockRepo.On("GetUnreadCount", mock.Anything, rideID, userID).Return(0, nil)
	mockRepo.On("GetLastMessage", mock.Anything, rideID).Return(nil, nil)
	mockRepo.On("MarkMessagesDelivered", mock.Anything, rideID, userID).Return(nil)

	c, w := setupTestContext("GET", "/api/v1/chat/rides/"+rideID.String()+"/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetConversation(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetConversation_AsDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	driverID := uuid.New()
	rideID := uuid.New()

	messages := []ChatMessage{
		*createTestMessage(rideID, uuid.New(), "rider", "Where are you?"),
	}

	mockRepo.On("GetMessagesByRide", mock.Anything, rideID, 20, 0).Return(messages, nil)
	mockRepo.On("GetUnreadCount", mock.Anything, rideID, driverID).Return(1, nil)
	mockRepo.On("GetLastMessage", mock.Anything, rideID).Return(&messages[0], nil)
	mockRepo.On("MarkMessagesDelivered", mock.Anything, rideID, driverID).Return(nil)
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).Return()

	c, w := setupTestContext("GET", "/api/v1/chat/rides/"+rideID.String()+"/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetConversation(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

// ========================================
// MARK AS READ TESTS
// ========================================

func TestHandler_MarkAsRead_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()
	lastReadID := uuid.New()

	reqBody := MarkReadRequest{
		RideID:     rideID,
		LastReadID: lastReadID,
	}

	mockRepo.On("MarkMessagesRead", mock.Anything, rideID, userID, lastReadID).Return(nil)
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).Return()

	c, w := setupTestContext("POST", "/api/v1/chat/read", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.MarkAsRead(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
	mockHub.AssertExpectations(t)
}

func TestHandler_MarkAsRead_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	reqBody := MarkReadRequest{
		RideID:     uuid.New(),
		LastReadID: uuid.New(),
	}

	c, w := setupTestContext("POST", "/api/v1/chat/read", reqBody)
	// Don't set user context

	handler.MarkAsRead(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_MarkAsRead_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/chat/read", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/chat/read", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.MarkAsRead(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MarkAsRead_MissingRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"last_read_id": uuid.New().String(),
	}

	c, w := setupTestContext("POST", "/api/v1/chat/read", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.MarkAsRead(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MarkAsRead_MissingLastReadID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"ride_id": uuid.New().String(),
	}

	c, w := setupTestContext("POST", "/api/v1/chat/read", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.MarkAsRead(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MarkAsRead_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()
	lastReadID := uuid.New()

	reqBody := MarkReadRequest{
		RideID:     rideID,
		LastReadID: lastReadID,
	}

	mockRepo.On("MarkMessagesRead", mock.Anything, rideID, userID, lastReadID).
		Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/chat/read", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.MarkAsRead(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_MarkAsRead_AsDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	driverID := uuid.New()
	rideID := uuid.New()
	lastReadID := uuid.New()

	reqBody := MarkReadRequest{
		RideID:     rideID,
		LastReadID: lastReadID,
	}

	mockRepo.On("MarkMessagesRead", mock.Anything, rideID, driverID, lastReadID).Return(nil)
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).Return()

	c, w := setupTestContext("POST", "/api/v1/chat/read", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.MarkAsRead(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

// ========================================
// GET ACTIVE CONVERSATIONS TESTS
// ========================================

func TestHandler_GetActiveConversations_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	mockRepo.On("GetActiveConversations", mock.Anything, userID).Return(rideIDs, nil)

	c, w := setupTestContext("GET", "/api/v1/chat/conversations", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetActiveConversations(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(3), data["count"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetActiveConversations_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	mockRepo.On("GetActiveConversations", mock.Anything, userID).Return(nil, nil)

	c, w := setupTestContext("GET", "/api/v1/chat/conversations", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetActiveConversations(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["count"])
}

func TestHandler_GetActiveConversations_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	c, w := setupTestContext("GET", "/api/v1/chat/conversations", nil)
	// Don't set user context

	handler.GetActiveConversations(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetActiveConversations_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	mockRepo.On("GetActiveConversations", mock.Anything, userID).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/chat/conversations", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetActiveConversations(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetActiveConversations_AsDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	driverID := uuid.New()
	rideIDs := []uuid.UUID{uuid.New()}

	mockRepo.On("GetActiveConversations", mock.Anything, driverID).Return(rideIDs, nil)

	c, w := setupTestContext("GET", "/api/v1/chat/conversations", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetActiveConversations(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["count"])
	mockRepo.AssertExpectations(t)
}

// ========================================
// GET QUICK REPLIES TESTS
// ========================================

func TestHandler_GetQuickReplies_Success_AsRider(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	replies := []QuickReply{
		createTestQuickReply("rider", "eta", "Where are you?"),
		createTestQuickReply("rider", "location", "I'm waiting outside"),
	}

	mockRepo.On("GetQuickReplies", mock.Anything, "rider").Return(replies, nil)

	c, w := setupTestContext("GET", "/api/v1/chat/quick-replies", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetQuickReplies(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["count"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetQuickReplies_Success_AsDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	replies := []QuickReply{
		createTestQuickReply("driver", "eta", "I'll be there in 5 minutes"),
		createTestQuickReply("driver", "arrival", "I've arrived"),
	}

	mockRepo.On("GetQuickReplies", mock.Anything, "driver").Return(replies, nil)

	c, w := setupTestContext("GET", "/api/v1/chat/quick-replies", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetQuickReplies(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["count"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetQuickReplies_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	mockRepo.On("GetQuickReplies", mock.Anything, "rider").Return([]QuickReply{}, nil)

	c, w := setupTestContext("GET", "/api/v1/chat/quick-replies", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetQuickReplies(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["count"])
}

func TestHandler_GetQuickReplies_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	c, w := setupTestContext("GET", "/api/v1/chat/quick-replies", nil)
	// Don't set user context

	handler.GetQuickReplies(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetQuickReplies_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	mockRepo.On("GetQuickReplies", mock.Anything, "rider").
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/chat/quick-replies", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetQuickReplies(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========================================
// INTEGRATION/TABLE-DRIVEN TESTS
// ========================================

func TestHandler_SendMessage_AllMessageTypes(t *testing.T) {
	latitude := 37.7749
	longitude := -122.4194
	imageURL := "https://example.com/image.jpg"

	tests := []struct {
		name        string
		messageType MessageType
		content     string
		imageURL    *string
		latitude    *float64
		longitude   *float64
		expectError bool
	}{
		{
			name:        "text message",
			messageType: MessageTypeText,
			content:     "Hello, driver!",
			expectError: false,
		},
		{
			name:        "image message",
			messageType: MessageTypeImage,
			content:     "Check this",
			imageURL:    &imageURL,
			expectError: false,
		},
		{
			name:        "location message with coordinates",
			messageType: MessageTypeLocation,
			content:     "I am here",
			latitude:    &latitude,
			longitude:   &longitude,
			expectError: false,
		},
		{
			name:        "location message without coordinates",
			messageType: MessageTypeLocation,
			content:     "I am here",
			expectError: true,
		},
		{
			name:        "empty content",
			messageType: MessageTypeText,
			content:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			mockHub := new(MockHub)
			handler := createTestHandler(mockRepo, mockHub)

			userID := uuid.New()
			rideID := uuid.New()

			reqBody := SendMessageRequest{
				RideID:      rideID,
				MessageType: tt.messageType,
				Content:     tt.content,
				ImageURL:    tt.imageURL,
				Latitude:    tt.latitude,
				Longitude:   tt.longitude,
			}

			if !tt.expectError {
				mockRepo.On("SaveMessage", mock.Anything, mock.AnythingOfType("*chat.ChatMessage")).Return(nil)
				mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).Return()
			}

			c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
			setUserContext(c, userID, models.RoleRider)

			handler.SendMessage(c)

			if tt.expectError {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			} else {
				assert.Equal(t, http.StatusCreated, w.Code)
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

func TestHandler_RolesAndAuthorization(t *testing.T) {
	tests := []struct {
		name       string
		role       models.UserRole
		setContext bool
		expectCode int
	}{
		{
			name:       "rider role",
			role:       models.RoleRider,
			setContext: true,
			expectCode: http.StatusCreated,
		},
		{
			name:       "driver role",
			role:       models.RoleDriver,
			setContext: true,
			expectCode: http.StatusCreated,
		},
		{
			name:       "admin role",
			role:       models.RoleAdmin,
			setContext: true,
			expectCode: http.StatusCreated,
		},
		{
			name:       "no authentication",
			setContext: false,
			expectCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			mockHub := new(MockHub)
			handler := createTestHandler(mockRepo, mockHub)

			userID := uuid.New()
			rideID := uuid.New()

			reqBody := SendMessageRequest{
				RideID:      rideID,
				MessageType: MessageTypeText,
				Content:     "Test message",
			}

			if tt.setContext && tt.expectCode == http.StatusCreated {
				mockRepo.On("SaveMessage", mock.Anything, mock.AnythingOfType("*chat.ChatMessage")).Return(nil)
				mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).Return()
			}

			c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
			if tt.setContext {
				setUserContext(c, userID, tt.role)
			}

			handler.SendMessage(c)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}

func TestHandler_ErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func(*MockRepository)
		expectedCode int
		endpoint     string
		method       string
		body         interface{}
	}{
		{
			name: "SendMessage database error",
			setupMock: func(m *MockRepository) {
				m.On("SaveMessage", mock.Anything, mock.AnythingOfType("*chat.ChatMessage")).
					Return(errors.New("database error"))
			},
			expectedCode: http.StatusInternalServerError,
			endpoint:     "/api/v1/chat/messages",
			method:       "POST",
			body: SendMessageRequest{
				RideID:      uuid.New(),
				MessageType: MessageTypeText,
				Content:     "Hello",
			},
		},
		{
			name: "GetConversation database error",
			setupMock: func(m *MockRepository) {
				m.On("GetMessagesByRide", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("database error"))
			},
			expectedCode: http.StatusInternalServerError,
			endpoint:     "/api/v1/chat/rides/{id}/messages",
			method:       "GET",
			body:         nil,
		},
		{
			name: "MarkAsRead database error",
			setupMock: func(m *MockRepository) {
				m.On("MarkMessagesRead", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("database error"))
			},
			expectedCode: http.StatusInternalServerError,
			endpoint:     "/api/v1/chat/read",
			method:       "POST",
			body: MarkReadRequest{
				RideID:     uuid.New(),
				LastReadID: uuid.New(),
			},
		},
		{
			name: "GetActiveConversations database error",
			setupMock: func(m *MockRepository) {
				m.On("GetActiveConversations", mock.Anything, mock.Anything).
					Return(nil, errors.New("database error"))
			},
			expectedCode: http.StatusInternalServerError,
			endpoint:     "/api/v1/chat/conversations",
			method:       "GET",
			body:         nil,
		},
		{
			name: "GetQuickReplies database error",
			setupMock: func(m *MockRepository) {
				m.On("GetQuickReplies", mock.Anything, mock.Anything).
					Return(nil, errors.New("database error"))
			},
			expectedCode: http.StatusInternalServerError,
			endpoint:     "/api/v1/chat/quick-replies",
			method:       "GET",
			body:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			mockHub := new(MockHub)
			handler := createTestHandler(mockRepo, mockHub)

			tt.setupMock(mockRepo)

			userID := uuid.New()
			rideID := uuid.New()

			c, w := setupTestContext(tt.method, tt.endpoint, tt.body)
			setUserContext(c, userID, models.RoleRider)

			// Set params for GetConversation
			if tt.endpoint == "/api/v1/chat/rides/{id}/messages" {
				c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
			}

			switch tt.endpoint {
			case "/api/v1/chat/messages":
				handler.SendMessage(c)
			case "/api/v1/chat/rides/{id}/messages":
				handler.GetConversation(c)
			case "/api/v1/chat/read":
				handler.MarkAsRead(c)
			case "/api/v1/chat/conversations":
				handler.GetActiveConversations(c)
			case "/api/v1/chat/quick-replies":
				handler.GetQuickReplies(c)
			}

			assert.Equal(t, tt.expectedCode, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

// ========================================
// WEBSOCKET NOTIFICATION TESTS
// ========================================

func TestHandler_SendMessage_WebSocketNotification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()

	reqBody := SendMessageRequest{
		RideID:      rideID,
		MessageType: MessageTypeText,
		Content:     "Hello!",
	}

	mockRepo.On("SaveMessage", mock.Anything, mock.AnythingOfType("*chat.ChatMessage")).Return(nil)

	// Capture the WebSocket message
	var capturedMsg *ws.Message
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).
		Run(func(args mock.Arguments) {
			capturedMsg = args.Get(1).(*ws.Message)
		}).Return()

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NotNil(t, capturedMsg)
	assert.Equal(t, "chat.message", capturedMsg.Type)
	assert.Equal(t, rideID.String(), capturedMsg.RideID)
	assert.Equal(t, userID.String(), capturedMsg.UserID)
	assert.Equal(t, "Hello!", capturedMsg.Data["content"])
	mockHub.AssertExpectations(t)
}

func TestHandler_MarkAsRead_WebSocketNotification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()
	lastReadID := uuid.New()

	reqBody := MarkReadRequest{
		RideID:     rideID,
		LastReadID: lastReadID,
	}

	mockRepo.On("MarkMessagesRead", mock.Anything, rideID, userID, lastReadID).Return(nil)

	// Capture the WebSocket message
	var capturedMsg *ws.Message
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).
		Run(func(args mock.Arguments) {
			capturedMsg = args.Get(1).(*ws.Message)
		}).Return()

	c, w := setupTestContext("POST", "/api/v1/chat/read", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.MarkAsRead(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, capturedMsg)
	assert.Equal(t, "chat.read", capturedMsg.Type)
	assert.Equal(t, rideID.String(), capturedMsg.Data["ride_id"])
	assert.Equal(t, userID.String(), capturedMsg.Data["read_by"])
	mockHub.AssertExpectations(t)
}

// ========================================
// EDGE CASES AND BOUNDARY TESTS
// ========================================

func TestHandler_SendMessage_MaxContentLength(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()

	// Create content exactly at max length (1000 characters)
	maxContent := make([]byte, 1000)
	for i := range maxContent {
		maxContent[i] = 'a'
	}

	reqBody := SendMessageRequest{
		RideID:      rideID,
		MessageType: MessageTypeText,
		Content:     string(maxContent),
	}

	mockRepo.On("SaveMessage", mock.Anything, mock.AnythingOfType("*chat.ChatMessage")).Return(nil)
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).Return()

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_SendMessage_ExceedsMaxContentLength(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	// Create content exceeding max length (1001 characters)
	longContent := make([]byte, 1001)
	for i := range longContent {
		longContent[i] = 'a'
	}

	reqBody := SendMessageRequest{
		RideID:      uuid.New(),
		MessageType: MessageTypeText,
		Content:     string(longContent),
	}

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SendMessage_ImageURLTooLong(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	// Create image URL exceeding max length (2049 characters)
	longURL := "https://example.com/" + string(make([]byte, 2049))

	reqBody := SendMessageRequest{
		RideID:      uuid.New(),
		MessageType: MessageTypeImage,
		Content:     "Check this",
		ImageURL:    &longURL,
	}

	c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetConversation_LargeLimitCapped(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()

	// The pagination package should cap the limit at 100
	mockRepo.On("GetMessagesByRide", mock.Anything, rideID, 100, 0).Return([]ChatMessage{}, nil)
	mockRepo.On("GetUnreadCount", mock.Anything, rideID, userID).Return(0, nil)
	mockRepo.On("GetLastMessage", mock.Anything, rideID).Return(nil, nil)
	mockRepo.On("MarkMessagesDelivered", mock.Anything, rideID, userID).Return(nil)

	c, w := setupTestContext("GET", "/api/v1/chat/rides/"+rideID.String()+"/messages?limit=1000", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	c.Request.URL.RawQuery = "limit=1000"
	setUserContext(c, userID, models.RoleRider)

	handler.GetConversation(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetConversation_NegativeOffset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()

	// Negative offset should be normalized to 0
	mockRepo.On("GetMessagesByRide", mock.Anything, rideID, 20, 0).Return([]ChatMessage{}, nil)
	mockRepo.On("GetUnreadCount", mock.Anything, rideID, userID).Return(0, nil)
	mockRepo.On("GetLastMessage", mock.Anything, rideID).Return(nil, nil)
	mockRepo.On("MarkMessagesDelivered", mock.Anything, rideID, userID).Return(nil)

	c, w := setupTestContext("GET", "/api/v1/chat/rides/"+rideID.String()+"/messages?offset=-10", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	c.Request.URL.RawQuery = "offset=-10"
	setUserContext(c, userID, models.RoleRider)

	handler.GetConversation(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

// ========================================
// HANDLER CONSTRUCTOR TESTS
// ========================================

func TestNewHandler(t *testing.T) {
	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	service := NewService(mockRepo, mockHub)

	handler := NewHandler(service)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.service)
}

// ========================================
// APP ERROR HANDLING TESTS
// ========================================

func TestHandler_SendMessage_AppErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	// Test various AppError scenarios
	tests := []struct {
		name        string
		content     string
		messageType MessageType
		expectCode  int
		expectMsg   string
	}{
		{
			name:        "empty content",
			content:     "",
			messageType: MessageTypeText,
			expectCode:  http.StatusBadRequest,
			expectMsg:   "cannot be empty",
		},
		{
			name:        "content too long",
			content:     string(make([]byte, 1001)),
			messageType: MessageTypeText,
			expectCode:  http.StatusBadRequest,
			expectMsg:   "message too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := SendMessageRequest{
				RideID:      uuid.New(),
				MessageType: tt.messageType,
				Content:     tt.content,
			}

			c, w := setupTestContext("POST", "/api/v1/chat/messages", reqBody)
			setUserContext(c, userID, models.RoleRider)

			handler.SendMessage(c)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}

// ========================================
// CONVERSATION DELIVERY NOTIFICATION TESTS
// ========================================

func TestHandler_GetConversation_DeliveryNotification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()
	rideID := uuid.New()

	messages := []ChatMessage{
		*createTestMessage(rideID, uuid.New(), "driver", "Hello"),
	}

	mockRepo.On("GetMessagesByRide", mock.Anything, rideID, 20, 0).Return(messages, nil)
	mockRepo.On("GetUnreadCount", mock.Anything, rideID, userID).Return(1, nil)
	mockRepo.On("GetLastMessage", mock.Anything, rideID).Return(&messages[0], nil)
	mockRepo.On("MarkMessagesDelivered", mock.Anything, rideID, userID).Return(nil)

	// Capture the WebSocket message
	var capturedMsg *ws.Message
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).
		Run(func(args mock.Arguments) {
			capturedMsg = args.Get(1).(*ws.Message)
		}).Return()

	c, w := setupTestContext("GET", "/api/v1/chat/rides/"+rideID.String()+"/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetConversation(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, capturedMsg)
	assert.Equal(t, "chat.delivered", capturedMsg.Type)
	assert.Equal(t, rideID.String(), capturedMsg.Data["ride_id"])
	assert.Equal(t, userID.String(), capturedMsg.Data["delivered_to"])
	mockHub.AssertExpectations(t)
}

// ========================================
// NIL HANDLING TESTS
// ========================================

func TestHandler_GetActiveConversations_NilReturnsEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockHub := new(MockHub)
	handler := createTestHandler(mockRepo, mockHub)

	userID := uuid.New()

	// Return nil from repository
	mockRepo.On("GetActiveConversations", mock.Anything, userID).Return(nil, nil)

	c, w := setupTestContext("GET", "/api/v1/chat/conversations", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetActiveConversations(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	// Should return empty array, not nil
	assert.NotNil(t, data["ride_ids"])
	assert.Equal(t, float64(0), data["count"])
}
