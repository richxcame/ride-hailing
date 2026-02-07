package common_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHandleServiceError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		fallbackMsg    string
		expectHandled  bool
		expectStatus   int
		expectContains string
	}{
		{
			name:          "nil error returns false",
			err:           nil,
			fallbackMsg:   "failed",
			expectHandled: false,
		},
		{
			name:           "AppError is handled",
			err:            common.NewNotFoundError("user not found", nil),
			fallbackMsg:    "failed to get user",
			expectHandled:  true,
			expectStatus:   http.StatusNotFound,
			expectContains: "user not found",
		},
		{
			name:           "regular error uses fallback",
			err:            errors.New("database error"),
			fallbackMsg:    "failed to get user",
			expectHandled:  true,
			expectStatus:   http.StatusInternalServerError,
			expectContains: "failed to get user",
		},
		{
			name:           "bad request AppError",
			err:            common.NewBadRequestError("invalid input", nil),
			fallbackMsg:    "failed",
			expectHandled:  true,
			expectStatus:   http.StatusBadRequest,
			expectContains: "invalid input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

			handled := common.HandleServiceError(c, tt.err, tt.fallbackMsg)
			assert.Equal(t, tt.expectHandled, handled)

			if tt.expectHandled {
				assert.Equal(t, tt.expectStatus, w.Code)
				assert.Contains(t, w.Body.String(), tt.expectContains)
			}
		})
	}
}

func TestParseUUIDParam(t *testing.T) {
	tests := []struct {
		name         string
		paramValue   string
		expectOK     bool
		expectStatus int
	}{
		{
			name:       "valid UUID",
			paramValue: "550e8400-e29b-41d4-a716-446655440000",
			expectOK:   true,
		},
		{
			name:         "invalid UUID",
			paramValue:   "not-a-uuid",
			expectOK:     false,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "empty UUID",
			paramValue:   "",
			expectOK:     false,
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: tt.paramValue}}
			c.Request = httptest.NewRequest(http.MethodGet, "/test/"+tt.paramValue, nil)

			id, ok := common.ParseUUIDParam(c, "id", "ride ID")
			assert.Equal(t, tt.expectOK, ok)

			if tt.expectOK {
				assert.NotEqual(t, uuid.Nil, id)
			} else {
				assert.Equal(t, tt.expectStatus, w.Code)
			}
		})
	}
}

func TestParseUUIDQuery(t *testing.T) {
	tests := []struct {
		name         string
		queryValue   string
		required     bool
		expectOK     bool
		expectNil    bool
		expectStatus int
	}{
		{
			name:       "valid UUID required",
			queryValue: "550e8400-e29b-41d4-a716-446655440000",
			required:   true,
			expectOK:   true,
			expectNil:  false,
		},
		{
			name:         "empty required",
			queryValue:   "",
			required:     true,
			expectOK:     false,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:       "empty optional",
			queryValue: "",
			required:   false,
			expectOK:   true,
			expectNil:  true,
		},
		{
			name:         "invalid UUID",
			queryValue:   "not-a-uuid",
			required:     false,
			expectOK:     false,
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			url := "/test"
			if tt.queryValue != "" {
				url += "?driver_id=" + tt.queryValue
			}
			c.Request = httptest.NewRequest(http.MethodGet, url, nil)

			id, ok := common.ParseUUIDQuery(c, "driver_id", "driver ID", tt.required)
			assert.Equal(t, tt.expectOK, ok)

			if tt.expectOK && tt.expectNil {
				assert.Equal(t, uuid.Nil, id)
			}

			if !tt.expectOK {
				assert.Equal(t, tt.expectStatus, w.Code)
			}
		})
	}
}

func TestBindJSON(t *testing.T) {
	type TestRequest struct {
		Name  string `json:"name" binding:"required"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name         string
		body         string
		expectOK     bool
		expectStatus int
	}{
		{
			name:     "valid JSON",
			body:     `{"name": "test", "value": 42}`,
			expectOK: true,
		},
		{
			name:         "missing required field",
			body:         `{"value": 42}`,
			expectOK:     false,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "invalid JSON",
			body:         `{invalid}`,
			expectOK:     false,
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			var req TestRequest
			ok := common.BindJSON(c, &req)
			assert.Equal(t, tt.expectOK, ok)

			if !tt.expectOK {
				assert.Equal(t, tt.expectStatus, w.Code)
			}
		})
	}
}

func TestBindQuery(t *testing.T) {
	type TestRequest struct {
		Page  int    `form:"page" binding:"required"`
		Limit int    `form:"limit"`
		Query string `form:"q"`
	}

	tests := []struct {
		name         string
		query        string
		expectOK     bool
		expectStatus int
	}{
		{
			name:     "valid query",
			query:    "?page=1&limit=10&q=test",
			expectOK: true,
		},
		{
			name:         "missing required field",
			query:        "?limit=10",
			expectOK:     false,
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/test"+tt.query, nil)

			var req TestRequest
			ok := common.BindQuery(c, &req)
			assert.Equal(t, tt.expectOK, ok)

			if !tt.expectOK {
				assert.Equal(t, tt.expectStatus, w.Code)
			}
		})
	}
}

func TestValidateNotEmpty(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		expectOK     bool
		expectStatus int
	}{
		{
			name:     "non-empty value",
			value:    "test",
			expectOK: true,
		},
		{
			name:         "empty value",
			value:        "",
			expectOK:     false,
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

			ok := common.ValidateNotEmpty(c, tt.value, "field")
			assert.Equal(t, tt.expectOK, ok)

			if !tt.expectOK {
				assert.Equal(t, tt.expectStatus, w.Code)
			}
		})
	}
}

func TestValidatePositive(t *testing.T) {
	tests := []struct {
		name         string
		value        float64
		expectOK     bool
		expectStatus int
	}{
		{
			name:     "positive value",
			value:    10.5,
			expectOK: true,
		},
		{
			name:         "zero value",
			value:        0,
			expectOK:     false,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "negative value",
			value:        -5,
			expectOK:     false,
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

			ok := common.ValidatePositive(c, tt.value, "amount")
			assert.Equal(t, tt.expectOK, ok)

			if !tt.expectOK {
				assert.Equal(t, tt.expectStatus, w.Code)
			}
		})
	}
}
