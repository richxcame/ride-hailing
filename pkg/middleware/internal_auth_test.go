package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupInternalAuthRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(InternalAPIKey())
	r.GET("/internal/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func TestInternalAPIKey_MissingEnvVar(t *testing.T) {
	t.Setenv("INTERNAL_API_KEY", "")

	r := setupInternalAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/internal/test", nil)
	req.Header.Set("X-Internal-API-Key", "some-key")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "not configured")
}

func TestInternalAPIKey_MissingHeader(t *testing.T) {
	t.Setenv("INTERNAL_API_KEY", "test-secret-key")

	r := setupInternalAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/internal/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid internal API key")
}

func TestInternalAPIKey_WrongKey(t *testing.T) {
	t.Setenv("INTERNAL_API_KEY", "test-secret-key")

	r := setupInternalAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/internal/test", nil)
	req.Header.Set("X-Internal-API-Key", "wrong-key")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid internal API key")
}

func TestInternalAPIKey_CorrectKey(t *testing.T) {
	t.Setenv("INTERNAL_API_KEY", "test-secret-key")

	r := setupInternalAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/internal/test", nil)
	req.Header.Set("X-Internal-API-Key", "test-secret-key")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}
