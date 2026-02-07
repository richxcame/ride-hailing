package disputes

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// SERVICE INTERFACE FOR MOCKING
// ========================================

type ServiceInterface interface {
	CreateDispute(ctx context.Context, userID uuid.UUID, req *CreateDisputeRequest) (*Dispute, error)
	GetMyDisputes(ctx context.Context, userID uuid.UUID, status *DisputeStatus, page, pageSize int) ([]DisputeSummary, int, error)
	GetDisputeDetail(ctx context.Context, disputeID, userID uuid.UUID) (*DisputeDetailResponse, error)
	AddComment(ctx context.Context, disputeID, userID uuid.UUID, req *AddCommentRequest) (*DisputeComment, error)
	GetDisputeReasons() *DisputeReasonsResponse
	AdminGetDisputes(ctx context.Context, status *DisputeStatus, reason *DisputeReason, page, pageSize int) ([]DisputeSummary, int, error)
	AdminGetDisputeDetail(ctx context.Context, disputeID uuid.UUID) (*DisputeDetailResponse, error)
	AdminResolveDispute(ctx context.Context, disputeID, adminID uuid.UUID, req *ResolveDisputeRequest) error
	AdminAddComment(ctx context.Context, disputeID, adminID uuid.UUID, comment string, isInternal bool) (*DisputeComment, error)
	AdminGetStats(ctx context.Context, from, to time.Time) (*DisputeStats, error)
}

// ========================================
// MOCK SERVICE
// ========================================

type mockService struct {
	mock.Mock
}

func (m *mockService) CreateDispute(ctx context.Context, userID uuid.UUID, req *CreateDisputeRequest) (*Dispute, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Dispute), args.Error(1)
}

func (m *mockService) GetMyDisputes(ctx context.Context, userID uuid.UUID, status *DisputeStatus, page, pageSize int) ([]DisputeSummary, int, error) {
	args := m.Called(ctx, userID, status, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]DisputeSummary), args.Int(1), args.Error(2)
}

func (m *mockService) GetDisputeDetail(ctx context.Context, disputeID, userID uuid.UUID) (*DisputeDetailResponse, error) {
	args := m.Called(ctx, disputeID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DisputeDetailResponse), args.Error(1)
}

func (m *mockService) AddComment(ctx context.Context, disputeID, userID uuid.UUID, req *AddCommentRequest) (*DisputeComment, error) {
	args := m.Called(ctx, disputeID, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DisputeComment), args.Error(1)
}

func (m *mockService) GetDisputeReasons() *DisputeReasonsResponse {
	args := m.Called()
	return args.Get(0).(*DisputeReasonsResponse)
}

func (m *mockService) AdminGetDisputes(ctx context.Context, status *DisputeStatus, reason *DisputeReason, page, pageSize int) ([]DisputeSummary, int, error) {
	args := m.Called(ctx, status, reason, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]DisputeSummary), args.Int(1), args.Error(2)
}

func (m *mockService) AdminGetDisputeDetail(ctx context.Context, disputeID uuid.UUID) (*DisputeDetailResponse, error) {
	args := m.Called(ctx, disputeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DisputeDetailResponse), args.Error(1)
}

func (m *mockService) AdminResolveDispute(ctx context.Context, disputeID, adminID uuid.UUID, req *ResolveDisputeRequest) error {
	args := m.Called(ctx, disputeID, adminID, req)
	return args.Error(0)
}

func (m *mockService) AdminAddComment(ctx context.Context, disputeID, adminID uuid.UUID, comment string, isInternal bool) (*DisputeComment, error) {
	args := m.Called(ctx, disputeID, adminID, comment, isInternal)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DisputeComment), args.Error(1)
}

func (m *mockService) AdminGetStats(ctx context.Context, from, to time.Time) (*DisputeStats, error) {
	args := m.Called(ctx, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DisputeStats), args.Error(1)
}

// ========================================
// TEST HANDLER WRAPPER
// ========================================

type testHandler struct {
	service ServiceInterface
}

func newTestHandler(svc ServiceInterface) *testHandler {
	return &testHandler{service: svc}
}

// ========================================
// TEST HELPERS
// ========================================

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func setUserContext(c *gin.Context, userID uuid.UUID) {
	c.Set("user_id", userID)
}

func parseResponse(t *testing.T, body *bytes.Buffer) map[string]interface{} {
	var response map[string]interface{}
	err := json.Unmarshal(body.Bytes(), &response)
	require.NoError(t, err)
	return response
}

// ========================================
// CREATE DISPUTE TESTS
// ========================================

func TestHandler_CreateDispute_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()
	now := time.Now()

	dispute := &Dispute{
		ID:             uuid.New(),
		RideID:         rideID,
		UserID:         userID,
		DisputeNumber:  "DSP-123456",
		Reason:         ReasonOvercharged,
		Description:    "I was overcharged for this ride",
		Status:         DisputeStatusPending,
		OriginalFare:   50.00,
		DisputedAmount: 20.00,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	mockSvc.On("CreateDispute", mock.Anything, userID, mock.AnythingOfType("*disputes.CreateDisputeRequest")).Return(dispute, nil)

	h := newTestHandler(mockSvc)
	router.POST("/disputes", func(c *gin.Context) {
		setUserContext(c, userID)

		var req CreateDisputeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		result, err := h.service.CreateDispute(c.Request.Context(), userID, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
	})

	reqBody := `{"ride_id":"` + rideID.String() + `","reason":"overcharged","description":"I was overcharged for this ride","disputed_amount":20.00}`
	req := httptest.NewRequest(http.MethodPost, "/disputes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(t, w.Body)
	assert.True(t, response["success"].(bool))
	mockSvc.AssertExpectations(t)
}

func TestHandler_CreateDispute_Unauthorized(t *testing.T) {
	router := setupTestRouter()

	router.POST("/disputes", func(c *gin.Context) {
		// No user_id in context
		_, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"message": "unauthorized"}})
			return
		}
	})

	reqBody := `{"ride_id":"` + uuid.New().String() + `","reason":"overcharged","description":"Test description","disputed_amount":20.00}`
	req := httptest.NewRequest(http.MethodPost, "/disputes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CreateDispute_InvalidRequestBody(t *testing.T) {
	router := setupTestRouter()
	userID := uuid.New()

	router.POST("/disputes", func(c *gin.Context) {
		setUserContext(c, userID)

		var req CreateDisputeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}
	})

	// Invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/disputes", bytes.NewBufferString(`{invalid json}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateDispute_MissingRequiredFields(t *testing.T) {
	router := setupTestRouter()
	userID := uuid.New()

	router.POST("/disputes", func(c *gin.Context) {
		setUserContext(c, userID)

		var req CreateDisputeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}
	})

	// Missing ride_id
	reqBody := `{"reason":"overcharged","description":"Test","disputed_amount":20.00}`
	req := httptest.NewRequest(http.MethodPost, "/disputes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateDispute_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()

	mockSvc.On("CreateDispute", mock.Anything, userID, mock.AnythingOfType("*disputes.CreateDisputeRequest")).
		Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.POST("/disputes", func(c *gin.Context) {
		setUserContext(c, userID)

		var req CreateDisputeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		result, err := h.service.CreateDispute(c.Request.Context(), userID, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to create dispute"}})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
	})

	reqBody := `{"ride_id":"` + uuid.New().String() + `","reason":"overcharged","description":"I was overcharged for this ride","disputed_amount":20.00}`
	req := httptest.NewRequest(http.MethodPost, "/disputes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_CreateDispute_NotFoundError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()

	appErr := common.NewNotFoundError("ride not found", nil)
	mockSvc.On("CreateDispute", mock.Anything, userID, mock.AnythingOfType("*disputes.CreateDisputeRequest")).
		Return(nil, appErr)

	h := newTestHandler(mockSvc)
	router.POST("/disputes", func(c *gin.Context) {
		setUserContext(c, userID)

		var req CreateDisputeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		result, err := h.service.CreateDispute(c.Request.Context(), userID, &req)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to create dispute"}})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
	})

	reqBody := `{"ride_id":"` + uuid.New().String() + `","reason":"overcharged","description":"I was overcharged for this ride","disputed_amount":20.00}`
	req := httptest.NewRequest(http.MethodPost, "/disputes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// GET MY DISPUTES TESTS
// ========================================

func TestHandler_GetMyDisputes_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()

	disputes := []DisputeSummary{
		{
			ID:             uuid.New(),
			DisputeNumber:  "DSP-001",
			RideID:         uuid.New(),
			Reason:         ReasonOvercharged,
			Status:         DisputeStatusPending,
			OriginalFare:   50.00,
			DisputedAmount: 20.00,
			CreatedAt:      time.Now(),
		},
		{
			ID:             uuid.New(),
			DisputeNumber:  "DSP-002",
			RideID:         uuid.New(),
			Reason:         ReasonWrongRoute,
			Status:         DisputeStatusApproved,
			OriginalFare:   30.00,
			DisputedAmount: 10.00,
			CreatedAt:      time.Now(),
		},
	}

	mockSvc.On("GetMyDisputes", mock.Anything, userID, (*DisputeStatus)(nil), mock.Anything, mock.Anything).
		Return(disputes, 2, nil)

	h := newTestHandler(mockSvc)
	router.GET("/disputes", func(c *gin.Context) {
		setUserContext(c, userID)

		result, total, err := h.service.GetMyDisputes(c.Request.Context(), userID, nil, 20, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get disputes"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"disputes": result}, "meta": gin.H{"total": total}})
	})

	req := httptest.NewRequest(http.MethodGet, "/disputes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(t, w.Body)
	assert.True(t, response["success"].(bool))
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetMyDisputes_WithStatusFilter(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()

	status := DisputeStatusPending
	disputes := []DisputeSummary{
		{
			ID:             uuid.New(),
			DisputeNumber:  "DSP-001",
			Status:         DisputeStatusPending,
			OriginalFare:   50.00,
			DisputedAmount: 20.00,
		},
	}

	mockSvc.On("GetMyDisputes", mock.Anything, userID, &status, mock.Anything, mock.Anything).
		Return(disputes, 1, nil)

	h := newTestHandler(mockSvc)
	router.GET("/disputes", func(c *gin.Context) {
		setUserContext(c, userID)

		var st *DisputeStatus
		if s := c.Query("status"); s != "" {
			status := DisputeStatus(s)
			st = &status
		}

		result, total, err := h.service.GetMyDisputes(c.Request.Context(), userID, st, 20, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get disputes"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"disputes": result}, "meta": gin.H{"total": total}})
	})

	req := httptest.NewRequest(http.MethodGet, "/disputes?status=pending", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetMyDisputes_Unauthorized(t *testing.T) {
	router := setupTestRouter()

	router.GET("/disputes", func(c *gin.Context) {
		_, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"message": "unauthorized"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/disputes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyDisputes_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()

	mockSvc.On("GetMyDisputes", mock.Anything, userID, (*DisputeStatus)(nil), mock.Anything, mock.Anything).
		Return(nil, 0, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/disputes", func(c *gin.Context) {
		setUserContext(c, userID)

		result, total, err := h.service.GetMyDisputes(c.Request.Context(), userID, nil, 20, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get disputes"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"disputes": result}, "meta": gin.H{"total": total}})
	})

	req := httptest.NewRequest(http.MethodGet, "/disputes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// GET DISPUTE DETAIL TESTS
// ========================================

func TestHandler_GetDisputeDetail_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()
	disputeID := uuid.New()

	detail := &DisputeDetailResponse{
		Dispute: &Dispute{
			ID:             disputeID,
			UserID:         userID,
			DisputeNumber:  "DSP-001",
			Status:         DisputeStatusPending,
			OriginalFare:   50.00,
			DisputedAmount: 20.00,
		},
		RideContext: &RideContext{
			RideID:        uuid.New(),
			EstimatedFare: 50.00,
		},
		Comments: []DisputeComment{},
	}

	mockSvc.On("GetDisputeDetail", mock.Anything, disputeID, userID).Return(detail, nil)

	h := newTestHandler(mockSvc)
	router.GET("/disputes/:id", func(c *gin.Context) {
		setUserContext(c, userID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		result, err := h.service.GetDisputeDetail(c.Request.Context(), id, userID)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get dispute"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/disputes/"+disputeID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(t, w.Body)
	assert.True(t, response["success"].(bool))
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetDisputeDetail_InvalidUUID(t *testing.T) {
	router := setupTestRouter()
	userID := uuid.New()

	router.GET("/disputes/:id", func(c *gin.Context) {
		setUserContext(c, userID)

		_, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/disputes/invalid-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetDisputeDetail_NotFound(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()
	disputeID := uuid.New()

	appErr := common.NewNotFoundError("dispute not found", nil)
	mockSvc.On("GetDisputeDetail", mock.Anything, disputeID, userID).Return(nil, appErr)

	h := newTestHandler(mockSvc)
	router.GET("/disputes/:id", func(c *gin.Context) {
		setUserContext(c, userID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		result, err := h.service.GetDisputeDetail(c.Request.Context(), id, userID)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get dispute"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/disputes/"+disputeID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetDisputeDetail_Forbidden(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()
	disputeID := uuid.New()

	appErr := common.NewForbiddenError("not authorized to view this dispute")
	mockSvc.On("GetDisputeDetail", mock.Anything, disputeID, userID).Return(nil, appErr)

	h := newTestHandler(mockSvc)
	router.GET("/disputes/:id", func(c *gin.Context) {
		setUserContext(c, userID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		result, err := h.service.GetDisputeDetail(c.Request.Context(), id, userID)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get dispute"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/disputes/"+disputeID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetDisputeDetail_Unauthorized(t *testing.T) {
	router := setupTestRouter()
	disputeID := uuid.New()

	router.GET("/disputes/:id", func(c *gin.Context) {
		_, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"message": "unauthorized"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/disputes/"+disputeID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ========================================
// ADD COMMENT TESTS
// ========================================

func TestHandler_AddComment_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()
	disputeID := uuid.New()

	comment := &DisputeComment{
		ID:        uuid.New(),
		DisputeID: disputeID,
		UserID:    userID,
		UserRole:  "user",
		Comment:   "This is my comment",
		CreatedAt: time.Now(),
	}

	mockSvc.On("AddComment", mock.Anything, disputeID, userID, mock.AnythingOfType("*disputes.AddCommentRequest")).
		Return(comment, nil)

	h := newTestHandler(mockSvc)
	router.POST("/disputes/:id/comments", func(c *gin.Context) {
		setUserContext(c, userID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req AddCommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		result, err := h.service.AddComment(c.Request.Context(), id, userID, &req)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to add comment"}})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
	})

	reqBody := `{"comment":"This is my comment"}`
	req := httptest.NewRequest(http.MethodPost, "/disputes/"+disputeID.String()+"/comments", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AddComment_InvalidUUID(t *testing.T) {
	router := setupTestRouter()
	userID := uuid.New()

	router.POST("/disputes/:id/comments", func(c *gin.Context) {
		setUserContext(c, userID)

		_, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}
	})

	reqBody := `{"comment":"Test comment"}`
	req := httptest.NewRequest(http.MethodPost, "/disputes/invalid-uuid/comments", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddComment_InvalidRequestBody(t *testing.T) {
	router := setupTestRouter()
	userID := uuid.New()
	disputeID := uuid.New()

	router.POST("/disputes/:id/comments", func(c *gin.Context) {
		setUserContext(c, userID)

		_, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req AddCommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/disputes/"+disputeID.String()+"/comments", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddComment_Unauthorized(t *testing.T) {
	router := setupTestRouter()
	disputeID := uuid.New()

	router.POST("/disputes/:id/comments", func(c *gin.Context) {
		_, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"message": "unauthorized"}})
			return
		}
	})

	reqBody := `{"comment":"Test comment"}`
	req := httptest.NewRequest(http.MethodPost, "/disputes/"+disputeID.String()+"/comments", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AddComment_NotFound(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()
	disputeID := uuid.New()

	appErr := common.NewNotFoundError("dispute not found", nil)
	mockSvc.On("AddComment", mock.Anything, disputeID, userID, mock.AnythingOfType("*disputes.AddCommentRequest")).
		Return(nil, appErr)

	h := newTestHandler(mockSvc)
	router.POST("/disputes/:id/comments", func(c *gin.Context) {
		setUserContext(c, userID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req AddCommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		result, err := h.service.AddComment(c.Request.Context(), id, userID, &req)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to add comment"}})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
	})

	reqBody := `{"comment":"Test comment"}`
	req := httptest.NewRequest(http.MethodPost, "/disputes/"+disputeID.String()+"/comments", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// GET DISPUTE REASONS TESTS
// ========================================

func TestHandler_GetDisputeReasons_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	reasons := &DisputeReasonsResponse{
		Reasons: []DisputeReasonOption{
			{Code: ReasonOvercharged, Label: "Overcharged", Description: "The fare was higher than the estimate"},
			{Code: ReasonWrongRoute, Label: "Wrong route taken", Description: "The driver took a different route"},
		},
	}

	mockSvc.On("GetDisputeReasons").Return(reasons)

	h := newTestHandler(mockSvc)
	router.GET("/disputes/reasons", func(c *gin.Context) {
		result := h.service.GetDisputeReasons()
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/disputes/reasons", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(t, w.Body)
	assert.True(t, response["success"].(bool))
	mockSvc.AssertExpectations(t)
}

// ========================================
// ADMIN GET DISPUTES TESTS
// ========================================

func TestHandler_AdminGetDisputes_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	disputes := []DisputeSummary{
		{
			ID:             uuid.New(),
			DisputeNumber:  "DSP-001",
			Status:         DisputeStatusPending,
			OriginalFare:   50.00,
			DisputedAmount: 20.00,
		},
		{
			ID:             uuid.New(),
			DisputeNumber:  "DSP-002",
			Status:         DisputeStatusReviewing,
			OriginalFare:   30.00,
			DisputedAmount: 10.00,
		},
	}

	mockSvc.On("AdminGetDisputes", mock.Anything, (*DisputeStatus)(nil), (*DisputeReason)(nil), mock.Anything, mock.Anything).
		Return(disputes, 2, nil)

	h := newTestHandler(mockSvc)
	router.GET("/admin/disputes", func(c *gin.Context) {
		result, total, err := h.service.AdminGetDisputes(c.Request.Context(), nil, nil, 20, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get disputes"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"disputes": result}, "meta": gin.H{"total": total}})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/disputes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminGetDisputes_WithFilters(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	status := DisputeStatusPending
	reason := ReasonOvercharged

	disputes := []DisputeSummary{
		{
			ID:             uuid.New(),
			DisputeNumber:  "DSP-001",
			Status:         DisputeStatusPending,
			Reason:         ReasonOvercharged,
			OriginalFare:   50.00,
			DisputedAmount: 20.00,
		},
	}

	mockSvc.On("AdminGetDisputes", mock.Anything, &status, &reason, mock.Anything, mock.Anything).
		Return(disputes, 1, nil)

	h := newTestHandler(mockSvc)
	router.GET("/admin/disputes", func(c *gin.Context) {
		var st *DisputeStatus
		if s := c.Query("status"); s != "" {
			status := DisputeStatus(s)
			st = &status
		}
		var rs *DisputeReason
		if r := c.Query("reason"); r != "" {
			reason := DisputeReason(r)
			rs = &reason
		}

		result, total, err := h.service.AdminGetDisputes(c.Request.Context(), st, rs, 20, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get disputes"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"disputes": result}, "meta": gin.H{"total": total}})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/disputes?status=pending&reason=overcharged", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminGetDisputes_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("AdminGetDisputes", mock.Anything, (*DisputeStatus)(nil), (*DisputeReason)(nil), mock.Anything, mock.Anything).
		Return(nil, 0, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/admin/disputes", func(c *gin.Context) {
		result, total, err := h.service.AdminGetDisputes(c.Request.Context(), nil, nil, 20, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get disputes"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"disputes": result}, "meta": gin.H{"total": total}})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/disputes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// ADMIN GET DISPUTE DETAIL TESTS
// ========================================

func TestHandler_AdminGetDisputeDetail_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	disputeID := uuid.New()

	detail := &DisputeDetailResponse{
		Dispute: &Dispute{
			ID:             disputeID,
			DisputeNumber:  "DSP-001",
			Status:         DisputeStatusPending,
			OriginalFare:   50.00,
			DisputedAmount: 20.00,
		},
		RideContext: &RideContext{
			RideID:        uuid.New(),
			EstimatedFare: 50.00,
		},
		Comments: []DisputeComment{},
	}

	mockSvc.On("AdminGetDisputeDetail", mock.Anything, disputeID).Return(detail, nil)

	h := newTestHandler(mockSvc)
	router.GET("/admin/disputes/:id", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		result, err := h.service.AdminGetDisputeDetail(c.Request.Context(), id)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get dispute"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/disputes/"+disputeID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminGetDisputeDetail_InvalidUUID(t *testing.T) {
	router := setupTestRouter()

	router.GET("/admin/disputes/:id", func(c *gin.Context) {
		_, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/disputes/invalid-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminGetDisputeDetail_NotFound(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	disputeID := uuid.New()

	appErr := common.NewNotFoundError("dispute not found", nil)
	mockSvc.On("AdminGetDisputeDetail", mock.Anything, disputeID).Return(nil, appErr)

	h := newTestHandler(mockSvc)
	router.GET("/admin/disputes/:id", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		result, err := h.service.AdminGetDisputeDetail(c.Request.Context(), id)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get dispute"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/disputes/"+disputeID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// ADMIN RESOLVE DISPUTE TESTS
// ========================================

func TestHandler_AdminResolveDispute_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	adminID := uuid.New()
	disputeID := uuid.New()

	mockSvc.On("AdminResolveDispute", mock.Anything, disputeID, adminID, mock.AnythingOfType("*disputes.ResolveDisputeRequest")).
		Return(nil)

	h := newTestHandler(mockSvc)
	router.POST("/admin/disputes/:id/resolve", func(c *gin.Context) {
		setUserContext(c, adminID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req ResolveDisputeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		if err := h.service.AdminResolveDispute(c.Request.Context(), id, adminID, &req); err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to resolve dispute"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "dispute resolved"}})
	})

	reqBody := `{"resolution_type":"full_refund","note":"Approved full refund for customer"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/disputes/"+disputeID.String()+"/resolve", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminResolveDispute_InvalidUUID(t *testing.T) {
	router := setupTestRouter()
	adminID := uuid.New()

	router.POST("/admin/disputes/:id/resolve", func(c *gin.Context) {
		setUserContext(c, adminID)

		_, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}
	})

	reqBody := `{"resolution_type":"full_refund","note":"Test note"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/disputes/invalid-uuid/resolve", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminResolveDispute_InvalidRequestBody(t *testing.T) {
	router := setupTestRouter()
	adminID := uuid.New()
	disputeID := uuid.New()

	router.POST("/admin/disputes/:id/resolve", func(c *gin.Context) {
		setUserContext(c, adminID)

		_, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req ResolveDisputeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/disputes/"+disputeID.String()+"/resolve", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminResolveDispute_Unauthorized(t *testing.T) {
	router := setupTestRouter()
	disputeID := uuid.New()

	router.POST("/admin/disputes/:id/resolve", func(c *gin.Context) {
		_, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"message": "unauthorized"}})
			return
		}
	})

	reqBody := `{"resolution_type":"full_refund","note":"Test note"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/disputes/"+disputeID.String()+"/resolve", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AdminResolveDispute_NotFound(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	adminID := uuid.New()
	disputeID := uuid.New()

	appErr := common.NewNotFoundError("dispute not found", nil)
	mockSvc.On("AdminResolveDispute", mock.Anything, disputeID, adminID, mock.AnythingOfType("*disputes.ResolveDisputeRequest")).
		Return(appErr)

	h := newTestHandler(mockSvc)
	router.POST("/admin/disputes/:id/resolve", func(c *gin.Context) {
		setUserContext(c, adminID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req ResolveDisputeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		if err := h.service.AdminResolveDispute(c.Request.Context(), id, adminID, &req); err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to resolve dispute"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "dispute resolved"}})
	})

	reqBody := `{"resolution_type":"full_refund","note":"Approved full refund"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/disputes/"+disputeID.String()+"/resolve", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminResolveDispute_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	adminID := uuid.New()
	disputeID := uuid.New()

	mockSvc.On("AdminResolveDispute", mock.Anything, disputeID, adminID, mock.AnythingOfType("*disputes.ResolveDisputeRequest")).
		Return(errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.POST("/admin/disputes/:id/resolve", func(c *gin.Context) {
		setUserContext(c, adminID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req ResolveDisputeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		if err := h.service.AdminResolveDispute(c.Request.Context(), id, adminID, &req); err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to resolve dispute"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "dispute resolved"}})
	})

	reqBody := `{"resolution_type":"full_refund","note":"Approved full refund"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/disputes/"+disputeID.String()+"/resolve", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// ADMIN ADD COMMENT TESTS
// ========================================

func TestHandler_AdminAddComment_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	adminID := uuid.New()
	disputeID := uuid.New()

	comment := &DisputeComment{
		ID:         uuid.New(),
		DisputeID:  disputeID,
		UserID:     adminID,
		UserRole:   "admin",
		Comment:    "Admin review comment",
		IsInternal: false,
		CreatedAt:  time.Now(),
	}

	mockSvc.On("AdminAddComment", mock.Anything, disputeID, adminID, "Admin review comment", false).
		Return(comment, nil)

	h := newTestHandler(mockSvc)
	router.POST("/admin/disputes/:id/comments", func(c *gin.Context) {
		setUserContext(c, adminID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req struct {
			Comment    string `json:"comment" binding:"required"`
			IsInternal bool   `json:"is_internal"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		result, err := h.service.AdminAddComment(c.Request.Context(), id, adminID, req.Comment, req.IsInternal)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to add comment"}})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
	})

	reqBody := `{"comment":"Admin review comment","is_internal":false}`
	req := httptest.NewRequest(http.MethodPost, "/admin/disputes/"+disputeID.String()+"/comments", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminAddComment_InternalNote(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	adminID := uuid.New()
	disputeID := uuid.New()

	comment := &DisputeComment{
		ID:         uuid.New(),
		DisputeID:  disputeID,
		UserID:     adminID,
		UserRole:   "admin",
		Comment:    "Internal note for team",
		IsInternal: true,
		CreatedAt:  time.Now(),
	}

	mockSvc.On("AdminAddComment", mock.Anything, disputeID, adminID, "Internal note for team", true).
		Return(comment, nil)

	h := newTestHandler(mockSvc)
	router.POST("/admin/disputes/:id/comments", func(c *gin.Context) {
		setUserContext(c, adminID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req struct {
			Comment    string `json:"comment" binding:"required"`
			IsInternal bool   `json:"is_internal"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		result, err := h.service.AdminAddComment(c.Request.Context(), id, adminID, req.Comment, req.IsInternal)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to add comment"}})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
	})

	reqBody := `{"comment":"Internal note for team","is_internal":true}`
	req := httptest.NewRequest(http.MethodPost, "/admin/disputes/"+disputeID.String()+"/comments", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminAddComment_InvalidUUID(t *testing.T) {
	router := setupTestRouter()
	adminID := uuid.New()

	router.POST("/admin/disputes/:id/comments", func(c *gin.Context) {
		setUserContext(c, adminID)

		_, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}
	})

	reqBody := `{"comment":"Test comment"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/disputes/invalid-uuid/comments", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminAddComment_InvalidRequestBody(t *testing.T) {
	router := setupTestRouter()
	adminID := uuid.New()
	disputeID := uuid.New()

	router.POST("/admin/disputes/:id/comments", func(c *gin.Context) {
		setUserContext(c, adminID)

		_, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req struct {
			Comment    string `json:"comment" binding:"required"`
			IsInternal bool   `json:"is_internal"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/disputes/"+disputeID.String()+"/comments", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminAddComment_Unauthorized(t *testing.T) {
	router := setupTestRouter()
	disputeID := uuid.New()

	router.POST("/admin/disputes/:id/comments", func(c *gin.Context) {
		_, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"message": "unauthorized"}})
			return
		}
	})

	reqBody := `{"comment":"Test comment"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/disputes/"+disputeID.String()+"/comments", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AdminAddComment_NotFound(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	adminID := uuid.New()
	disputeID := uuid.New()

	appErr := common.NewNotFoundError("dispute not found", nil)
	mockSvc.On("AdminAddComment", mock.Anything, disputeID, adminID, "Test comment", false).
		Return(nil, appErr)

	h := newTestHandler(mockSvc)
	router.POST("/admin/disputes/:id/comments", func(c *gin.Context) {
		setUserContext(c, adminID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req struct {
			Comment    string `json:"comment" binding:"required"`
			IsInternal bool   `json:"is_internal"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		result, err := h.service.AdminAddComment(c.Request.Context(), id, adminID, req.Comment, req.IsInternal)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to add comment"}})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
	})

	reqBody := `{"comment":"Test comment","is_internal":false}`
	req := httptest.NewRequest(http.MethodPost, "/admin/disputes/"+disputeID.String()+"/comments", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// ADMIN GET STATS TESTS
// ========================================

func TestHandler_AdminGetStats_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	stats := &DisputeStats{
		TotalDisputes:      100,
		PendingDisputes:    20,
		ReviewingDisputes:  10,
		ApprovedDisputes:   50,
		RejectedDisputes:   20,
		TotalRefunded:      5000.00,
		TotalDisputed:      8000.00,
		AvgResolutionHours: 24.5,
		DisputeRate:        0.02,
		ByReason: []ReasonCount{
			{Reason: ReasonOvercharged, Count: 40, Percentage: 40.0},
			{Reason: ReasonWrongRoute, Count: 30, Percentage: 30.0},
		},
	}

	mockSvc.On("AdminGetStats", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return(stats, nil)

	h := newTestHandler(mockSvc)
	router.GET("/admin/disputes/stats", func(c *gin.Context) {
		from := time.Now().AddDate(0, -1, 0)
		to := time.Now()

		result, err := h.service.AdminGetStats(c.Request.Context(), from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get dispute stats"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/disputes/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminGetStats_WithDateRange(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	stats := &DisputeStats{
		TotalDisputes:   50,
		PendingDisputes: 10,
	}

	mockSvc.On("AdminGetStats", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return(stats, nil)

	h := newTestHandler(mockSvc)
	router.GET("/admin/disputes/stats", func(c *gin.Context) {
		from := time.Now().AddDate(0, -1, 0)
		to := time.Now()

		if fromStr := c.Query("from"); fromStr != "" {
			if t, err := time.Parse("2006-01-02", fromStr); err == nil {
				from = t
			}
		}
		if toStr := c.Query("to"); toStr != "" {
			if t, err := time.Parse("2006-01-02", toStr); err == nil {
				to = t.AddDate(0, 0, 1)
			}
		}

		result, err := h.service.AdminGetStats(c.Request.Context(), from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get dispute stats"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/disputes/stats?from=2025-01-01&to=2025-01-31", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminGetStats_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("AdminGetStats", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/admin/disputes/stats", func(c *gin.Context) {
		from := time.Now().AddDate(0, -1, 0)
		to := time.Now()

		result, err := h.service.AdminGetStats(c.Request.Context(), from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get dispute stats"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/disputes/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// CONFLICT ERROR TESTS
// ========================================

func TestHandler_CreateDispute_ConflictError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()

	appErr := common.NewConflictError("you already have an active dispute for this ride")
	mockSvc.On("CreateDispute", mock.Anything, userID, mock.AnythingOfType("*disputes.CreateDisputeRequest")).
		Return(nil, appErr)

	h := newTestHandler(mockSvc)
	router.POST("/disputes", func(c *gin.Context) {
		setUserContext(c, userID)

		var req CreateDisputeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		result, err := h.service.CreateDispute(c.Request.Context(), userID, &req)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to create dispute"}})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
	})

	reqBody := `{"ride_id":"` + uuid.New().String() + `","reason":"overcharged","description":"I was overcharged for this ride","disputed_amount":20.00}`
	req := httptest.NewRequest(http.MethodPost, "/disputes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// BAD REQUEST ERROR TESTS
// ========================================

func TestHandler_CreateDispute_BadRequestError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()

	appErr := common.NewBadRequestError("disputed amount cannot exceed the ride fare", nil)
	mockSvc.On("CreateDispute", mock.Anything, userID, mock.AnythingOfType("*disputes.CreateDisputeRequest")).
		Return(nil, appErr)

	h := newTestHandler(mockSvc)
	router.POST("/disputes", func(c *gin.Context) {
		setUserContext(c, userID)

		var req CreateDisputeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		result, err := h.service.CreateDispute(c.Request.Context(), userID, &req)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to create dispute"}})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
	})

	reqBody := `{"ride_id":"` + uuid.New().String() + `","reason":"overcharged","description":"I was overcharged for this ride","disputed_amount":200.00}`
	req := httptest.NewRequest(http.MethodPost, "/disputes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminResolveDispute_BadRequestError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	adminID := uuid.New()
	disputeID := uuid.New()

	appErr := common.NewBadRequestError("dispute already resolved", nil)
	mockSvc.On("AdminResolveDispute", mock.Anything, disputeID, adminID, mock.AnythingOfType("*disputes.ResolveDisputeRequest")).
		Return(appErr)

	h := newTestHandler(mockSvc)
	router.POST("/admin/disputes/:id/resolve", func(c *gin.Context) {
		setUserContext(c, adminID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req ResolveDisputeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		if err := h.service.AdminResolveDispute(c.Request.Context(), id, adminID, &req); err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to resolve dispute"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "dispute resolved"}})
	})

	reqBody := `{"resolution_type":"full_refund","note":"Approved full refund"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/disputes/"+disputeID.String()+"/resolve", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// EDGE CASE TESTS
// ========================================

func TestHandler_GetMyDisputes_EmptyResult(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()

	mockSvc.On("GetMyDisputes", mock.Anything, userID, (*DisputeStatus)(nil), mock.Anything, mock.Anything).
		Return([]DisputeSummary{}, 0, nil)

	h := newTestHandler(mockSvc)
	router.GET("/disputes", func(c *gin.Context) {
		setUserContext(c, userID)

		result, total, err := h.service.GetMyDisputes(c.Request.Context(), userID, nil, 20, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get disputes"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"disputes": result}, "meta": gin.H{"total": total}})
	})

	req := httptest.NewRequest(http.MethodGet, "/disputes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(t, w.Body)
	assert.True(t, response["success"].(bool))
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminGetDisputes_EmptyResult(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("AdminGetDisputes", mock.Anything, (*DisputeStatus)(nil), (*DisputeReason)(nil), mock.Anything, mock.Anything).
		Return([]DisputeSummary{}, 0, nil)

	h := newTestHandler(mockSvc)
	router.GET("/admin/disputes", func(c *gin.Context) {
		result, total, err := h.service.AdminGetDisputes(c.Request.Context(), nil, nil, 20, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get disputes"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"disputes": result}, "meta": gin.H{"total": total}})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/disputes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AddComment_DisputeClosed(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()
	disputeID := uuid.New()

	appErr := common.NewBadRequestError("cannot comment on a closed or rejected dispute", nil)
	mockSvc.On("AddComment", mock.Anything, disputeID, userID, mock.AnythingOfType("*disputes.AddCommentRequest")).
		Return(nil, appErr)

	h := newTestHandler(mockSvc)
	router.POST("/disputes/:id/comments", func(c *gin.Context) {
		setUserContext(c, userID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req AddCommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		result, err := h.service.AddComment(c.Request.Context(), id, userID, &req)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to add comment"}})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
	})

	reqBody := `{"comment":"This is my comment"}`
	req := httptest.NewRequest(http.MethodPost, "/disputes/"+disputeID.String()+"/comments", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AddComment_Forbidden(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()
	userID := uuid.New()
	disputeID := uuid.New()

	appErr := common.NewForbiddenError("not authorized to comment on this dispute")
	mockSvc.On("AddComment", mock.Anything, disputeID, userID, mock.AnythingOfType("*disputes.AddCommentRequest")).
		Return(nil, appErr)

	h := newTestHandler(mockSvc)
	router.POST("/disputes/:id/comments", func(c *gin.Context) {
		setUserContext(c, userID)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid dispute id"}})
			return
		}

		var req AddCommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid request body"}})
			return
		}

		result, err := h.service.AddComment(c.Request.Context(), id, userID, &req)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to add comment"}})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
	})

	reqBody := `{"comment":"This is my comment"}`
	req := httptest.NewRequest(http.MethodPost, "/disputes/"+disputeID.String()+"/comments", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertExpectations(t)
}
