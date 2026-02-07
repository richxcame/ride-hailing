package currency

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
	"github.com/stretchr/testify/require"
)

// ========================================
// SERVICE INTERFACE FOR HANDLER TESTS
// ========================================

// ServiceInterface defines the interface for mocking the currency service
type ServiceInterface interface {
	GetActiveCurrencies(ctx context.Context) ([]*Currency, error)
	GetCurrency(ctx context.Context, code string) (*Currency, error)
	GetExchangeRate(ctx context.Context, from, to string) (*ExchangeRate, error)
	GetAllRatesFromBase(ctx context.Context) ([]*ExchangeRate, error)
	GetBaseCurrency() string
	Convert(ctx context.Context, amount float64, from, to string) (*ConversionResult, error)
	FormatMoney(ctx context.Context, money Money) (string, error)
}

// ========================================
// SERVICE MOCK FOR HANDLER TESTS
// ========================================

type mockCurrencyService struct {
	mock.Mock
}

func (m *mockCurrencyService) GetActiveCurrencies(ctx context.Context) ([]*Currency, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Currency), args.Error(1)
}

func (m *mockCurrencyService) GetCurrency(ctx context.Context, code string) (*Currency, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Currency), args.Error(1)
}

func (m *mockCurrencyService) GetExchangeRate(ctx context.Context, from, to string) (*ExchangeRate, error) {
	args := m.Called(ctx, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ExchangeRate), args.Error(1)
}

func (m *mockCurrencyService) GetAllRatesFromBase(ctx context.Context) ([]*ExchangeRate, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ExchangeRate), args.Error(1)
}

func (m *mockCurrencyService) GetBaseCurrency() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockCurrencyService) Convert(ctx context.Context, amount float64, from, to string) (*ConversionResult, error) {
	args := m.Called(ctx, amount, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ConversionResult), args.Error(1)
}

func (m *mockCurrencyService) FormatMoney(ctx context.Context, money Money) (string, error) {
	args := m.Called(ctx, money)
	return args.String(0), args.Error(1)
}

// ========================================
// TEST HANDLER WRAPPER
// ========================================

// testHandler wraps service interface for testing
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

// ========================================
// GET CURRENCIES TESTS
// ========================================

func TestHandler_GetCurrencies_Success(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	currencies := []*Currency{
		{Code: "USD", Name: "US Dollar", Symbol: "$", DecimalPlaces: 2, IsActive: true},
		{Code: "EUR", Name: "Euro", Symbol: "\u20ac", DecimalPlaces: 2, IsActive: true},
		{Code: "GBP", Name: "British Pound", Symbol: "\u00a3", DecimalPlaces: 2, IsActive: true},
	}

	mockSvc.On("GetActiveCurrencies", mock.Anything).Return(currencies, nil)

	h := newTestHandler(mockSvc)
	router.GET("/currencies", func(c *gin.Context) {
		result, err := h.service.GetActiveCurrencies(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get currencies"}})
			return
		}
		responses := make([]*CurrencyResponse, len(result))
		for i, currency := range result {
			responses[i] = ToCurrencyResponse(currency)
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": responses})
	})

	req := httptest.NewRequest(http.MethodGet, "/currencies", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].([]interface{})
	assert.Len(t, data, 3)

	mockSvc.AssertExpectations(t)
}

func TestHandler_GetCurrencies_EmptyList(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	mockSvc.On("GetActiveCurrencies", mock.Anything).Return([]*Currency{}, nil)

	h := newTestHandler(mockSvc)
	router.GET("/currencies", func(c *gin.Context) {
		result, err := h.service.GetActiveCurrencies(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get currencies"}})
			return
		}
		responses := make([]*CurrencyResponse, len(result))
		for i, currency := range result {
			responses[i] = ToCurrencyResponse(currency)
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": responses})
	})

	req := httptest.NewRequest(http.MethodGet, "/currencies", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].([]interface{})
	assert.Len(t, data, 0)

	mockSvc.AssertExpectations(t)
}

func TestHandler_GetCurrencies_ServiceError(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	mockSvc.On("GetActiveCurrencies", mock.Anything).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/currencies", func(c *gin.Context) {
		result, err := h.service.GetActiveCurrencies(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get currencies"}})
			return
		}
		responses := make([]*CurrencyResponse, len(result))
		for i, currency := range result {
			responses[i] = ToCurrencyResponse(currency)
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": responses})
	})

	req := httptest.NewRequest(http.MethodGet, "/currencies", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))

	mockSvc.AssertExpectations(t)
}

// ========================================
// GET CURRENCY BY CODE TESTS
// ========================================

func TestHandler_GetCurrency_Success(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	currency := &Currency{Code: "USD", Name: "US Dollar", Symbol: "$", DecimalPlaces: 2, IsActive: true}

	mockSvc.On("GetCurrency", mock.Anything, "USD").Return(currency, nil)

	h := newTestHandler(mockSvc)
	router.GET("/currencies/:code", func(c *gin.Context) {
		code := c.Param("code")
		if len(code) != 3 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid currency code"}})
			return
		}
		result, err := h.service.GetCurrency(c.Request.Context(), code)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"message": "currency not found"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": ToCurrencyResponse(result)})
	})

	req := httptest.NewRequest(http.MethodGet, "/currencies/USD", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "USD", data["code"])
	assert.Equal(t, "US Dollar", data["name"])
	assert.Equal(t, "$", data["symbol"])
	assert.Equal(t, float64(2), data["decimal_places"])

	mockSvc.AssertExpectations(t)
}

func TestHandler_GetCurrency_InvalidCodeTooShort(t *testing.T) {
	router := setupTestRouter()

	router.GET("/currencies/:code", func(c *gin.Context) {
		code := c.Param("code")
		if len(code) != 3 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid currency code"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/currencies/US", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetCurrency_InvalidCodeTooLong(t *testing.T) {
	router := setupTestRouter()

	router.GET("/currencies/:code", func(c *gin.Context) {
		code := c.Param("code")
		if len(code) != 3 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid currency code"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/currencies/USDD", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetCurrency_NotFound(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	mockSvc.On("GetCurrency", mock.Anything, "XYZ").Return(nil, errors.New("not found"))

	h := newTestHandler(mockSvc)
	router.GET("/currencies/:code", func(c *gin.Context) {
		code := c.Param("code")
		if len(code) != 3 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid currency code"}})
			return
		}
		result, err := h.service.GetCurrency(c.Request.Context(), code)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"message": "currency not found"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": ToCurrencyResponse(result)})
	})

	req := httptest.NewRequest(http.MethodGet, "/currencies/XYZ", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// GET EXCHANGE RATE TESTS
// ========================================

func TestHandler_GetExchangeRate_Success(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	rate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: "USD",
		ToCurrency:   "EUR",
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		Source:       "manual",
		FetchedAt:    time.Now(),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}

	mockSvc.On("GetExchangeRate", mock.Anything, "USD", "EUR").Return(rate, nil)

	h := newTestHandler(mockSvc)
	router.GET("/rate", func(c *gin.Context) {
		from := c.Query("from")
		to := c.Query("to")

		if len(from) != 3 || len(to) != 3 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid currency codes"}})
			return
		}

		result, err := h.service.GetExchangeRate(c.Request.Context(), from, to)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"message": "exchange rate not found"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": ToExchangeRateResponse(result)})
	})

	req := httptest.NewRequest(http.MethodGet, "/rate?from=USD&to=EUR", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "USD", data["from_currency"])
	assert.Equal(t, "EUR", data["to_currency"])
	assert.Equal(t, 0.85, data["rate"])

	mockSvc.AssertExpectations(t)
}

func TestHandler_GetExchangeRate_MissingFromParam(t *testing.T) {
	router := setupTestRouter()

	router.GET("/rate", func(c *gin.Context) {
		from := c.Query("from")
		to := c.Query("to")

		if len(from) != 3 || len(to) != 3 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid currency codes"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/rate?to=EUR", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetExchangeRate_MissingToParam(t *testing.T) {
	router := setupTestRouter()

	router.GET("/rate", func(c *gin.Context) {
		from := c.Query("from")
		to := c.Query("to")

		if len(from) != 3 || len(to) != 3 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid currency codes"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/rate?from=USD", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetExchangeRate_InvalidFromCode(t *testing.T) {
	router := setupTestRouter()

	router.GET("/rate", func(c *gin.Context) {
		from := c.Query("from")
		to := c.Query("to")

		if len(from) != 3 || len(to) != 3 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid currency codes"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/rate?from=US&to=EUR", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetExchangeRate_InvalidToCode(t *testing.T) {
	router := setupTestRouter()

	router.GET("/rate", func(c *gin.Context) {
		from := c.Query("from")
		to := c.Query("to")

		if len(from) != 3 || len(to) != 3 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid currency codes"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/rate?from=USD&to=EURO", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetExchangeRate_NotFound(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	mockSvc.On("GetExchangeRate", mock.Anything, "USD", "XYZ").Return(nil, errors.New("rate not found"))

	h := newTestHandler(mockSvc)
	router.GET("/rate", func(c *gin.Context) {
		from := c.Query("from")
		to := c.Query("to")

		if len(from) != 3 || len(to) != 3 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid currency codes"}})
			return
		}

		result, err := h.service.GetExchangeRate(c.Request.Context(), from, to)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"message": "exchange rate not found"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": ToExchangeRateResponse(result)})
	})

	req := httptest.NewRequest(http.MethodGet, "/rate?from=USD&to=XYZ", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetExchangeRate_SameCurrency(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	rate := &ExchangeRate{
		ID:           uuid.Nil,
		FromCurrency: "USD",
		ToCurrency:   "USD",
		Rate:         1.0,
		InverseRate:  1.0,
		Source:       "identity",
		FetchedAt:    time.Now(),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}

	mockSvc.On("GetExchangeRate", mock.Anything, "USD", "USD").Return(rate, nil)

	h := newTestHandler(mockSvc)
	router.GET("/rate", func(c *gin.Context) {
		from := c.Query("from")
		to := c.Query("to")

		if len(from) != 3 || len(to) != 3 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid currency codes"}})
			return
		}

		result, err := h.service.GetExchangeRate(c.Request.Context(), from, to)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"message": "exchange rate not found"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": ToExchangeRateResponse(result)})
	})

	req := httptest.NewRequest(http.MethodGet, "/rate?from=USD&to=USD", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["rate"])

	mockSvc.AssertExpectations(t)
}

// ========================================
// GET ALL RATES TESTS
// ========================================

func TestHandler_GetAllRates_Success(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	rates := []*ExchangeRate{
		{FromCurrency: "USD", ToCurrency: "EUR", Rate: 0.85, ValidUntil: time.Now().Add(24 * time.Hour)},
		{FromCurrency: "USD", ToCurrency: "GBP", Rate: 0.75, ValidUntil: time.Now().Add(24 * time.Hour)},
		{FromCurrency: "USD", ToCurrency: "TMT", Rate: 3.50, ValidUntil: time.Now().Add(24 * time.Hour)},
	}

	mockSvc.On("GetAllRatesFromBase", mock.Anything).Return(rates, nil)
	mockSvc.On("GetBaseCurrency").Return("USD")

	h := newTestHandler(mockSvc)
	router.GET("/rates", func(c *gin.Context) {
		result, err := h.service.GetAllRatesFromBase(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get exchange rates"}})
			return
		}

		responses := make([]*ExchangeRateResponse, len(result))
		for i, rate := range result {
			responses[i] = ToExchangeRateResponse(rate)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"base_currency": h.service.GetBaseCurrency(),
				"rates":         responses,
			},
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/rates", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "USD", data["base_currency"])

	ratesData := data["rates"].([]interface{})
	assert.Len(t, ratesData, 3)

	mockSvc.AssertExpectations(t)
}

func TestHandler_GetAllRates_EmptyList(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	mockSvc.On("GetAllRatesFromBase", mock.Anything).Return([]*ExchangeRate{}, nil)
	mockSvc.On("GetBaseCurrency").Return("USD")

	h := newTestHandler(mockSvc)
	router.GET("/rates", func(c *gin.Context) {
		result, err := h.service.GetAllRatesFromBase(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get exchange rates"}})
			return
		}

		responses := make([]*ExchangeRateResponse, len(result))
		for i, rate := range result {
			responses[i] = ToExchangeRateResponse(rate)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"base_currency": h.service.GetBaseCurrency(),
				"rates":         responses,
			},
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/rates", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	ratesData := data["rates"].([]interface{})
	assert.Len(t, ratesData, 0)

	mockSvc.AssertExpectations(t)
}

func TestHandler_GetAllRates_ServiceError(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	mockSvc.On("GetAllRatesFromBase", mock.Anything).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/rates", func(c *gin.Context) {
		result, err := h.service.GetAllRatesFromBase(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get exchange rates"}})
			return
		}

		responses := make([]*ExchangeRateResponse, len(result))
		for i, rate := range result {
			responses[i] = ToExchangeRateResponse(rate)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"base_currency": h.service.GetBaseCurrency(),
				"rates":         responses,
			},
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/rates", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// CONVERT TESTS
// ========================================

func TestHandler_Convert_Success(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	conversionResult := &ConversionResult{
		Original:     Money{Amount: 100.00, Currency: "USD"},
		Converted:    Money{Amount: 85.00, Currency: "EUR"},
		ExchangeRate: 0.85,
		ConvertedAt:  time.Now(),
	}

	mockSvc.On("Convert", mock.Anything, float64(100), "USD", "EUR").Return(conversionResult, nil)
	mockSvc.On("FormatMoney", mock.Anything, Money{Amount: 100.00, Currency: "USD"}).Return("$100.00", nil)
	mockSvc.On("FormatMoney", mock.Anything, Money{Amount: 85.00, Currency: "EUR"}).Return("\u20ac85.00", nil)

	h := newTestHandler(mockSvc)
	router.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}

		result, err := h.service.Convert(c.Request.Context(), req.Amount, req.FromCurrency, req.ToCurrency)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}

		formattedOriginal, _ := h.service.FormatMoney(c.Request.Context(), result.Original)
		formattedConverted, _ := h.service.FormatMoney(c.Request.Context(), result.Converted)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": ConvertResponse{
				OriginalAmount:     result.Original.Amount,
				OriginalCurrency:   result.Original.Currency,
				ConvertedAmount:    result.Converted.Amount,
				ConvertedCurrency:  result.Converted.Currency,
				ExchangeRate:       result.ExchangeRate,
				FormattedOriginal:  formattedOriginal,
				FormattedConverted: formattedConverted,
			},
		})
	})

	body := bytes.NewBufferString(`{"amount": 100, "from_currency": "USD", "to_currency": "EUR"}`)
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(100), data["original_amount"])
	assert.Equal(t, "USD", data["original_currency"])
	assert.Equal(t, float64(85), data["converted_amount"])
	assert.Equal(t, "EUR", data["converted_currency"])
	assert.Equal(t, 0.85, data["exchange_rate"])
	assert.Equal(t, "$100.00", data["formatted_original"])
	assert.Equal(t, "\u20ac85.00", data["formatted_converted"])

	mockSvc.AssertExpectations(t)
}

func TestHandler_Convert_SameCurrency(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	conversionResult := &ConversionResult{
		Original:     Money{Amount: 100.00, Currency: "USD"},
		Converted:    Money{Amount: 100.00, Currency: "USD"},
		ExchangeRate: 1.0,
		ConvertedAt:  time.Now(),
	}

	mockSvc.On("Convert", mock.Anything, float64(100), "USD", "USD").Return(conversionResult, nil)
	mockSvc.On("FormatMoney", mock.Anything, Money{Amount: 100.00, Currency: "USD"}).Return("$100.00", nil)

	h := newTestHandler(mockSvc)
	router.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}

		result, err := h.service.Convert(c.Request.Context(), req.Amount, req.FromCurrency, req.ToCurrency)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}

		formattedOriginal, _ := h.service.FormatMoney(c.Request.Context(), result.Original)
		formattedConverted, _ := h.service.FormatMoney(c.Request.Context(), result.Converted)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": ConvertResponse{
				OriginalAmount:     result.Original.Amount,
				OriginalCurrency:   result.Original.Currency,
				ConvertedAmount:    result.Converted.Amount,
				ConvertedCurrency:  result.Converted.Currency,
				ExchangeRate:       result.ExchangeRate,
				FormattedOriginal:  formattedOriginal,
				FormattedConverted: formattedConverted,
			},
		})
	})

	body := bytes.NewBufferString(`{"amount": 100, "from_currency": "USD", "to_currency": "USD"}`)
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["exchange_rate"])

	mockSvc.AssertExpectations(t)
}

func TestHandler_Convert_MissingAmount(t *testing.T) {
	router := setupTestRouter()

	router.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
	})

	body := bytes.NewBufferString(`{"from_currency": "USD", "to_currency": "EUR"}`)
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Convert_ZeroAmount(t *testing.T) {
	router := setupTestRouter()

	router.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
	})

	body := bytes.NewBufferString(`{"amount": 0, "from_currency": "USD", "to_currency": "EUR"}`)
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Convert_NegativeAmount(t *testing.T) {
	router := setupTestRouter()

	router.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
	})

	body := bytes.NewBufferString(`{"amount": -100, "from_currency": "USD", "to_currency": "EUR"}`)
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Convert_MissingFromCurrency(t *testing.T) {
	router := setupTestRouter()

	router.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
	})

	body := bytes.NewBufferString(`{"amount": 100, "to_currency": "EUR"}`)
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Convert_MissingToCurrency(t *testing.T) {
	router := setupTestRouter()

	router.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
	})

	body := bytes.NewBufferString(`{"amount": 100, "from_currency": "USD"}`)
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Convert_InvalidFromCurrencyLength(t *testing.T) {
	router := setupTestRouter()

	router.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
	})

	body := bytes.NewBufferString(`{"amount": 100, "from_currency": "US", "to_currency": "EUR"}`)
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Convert_InvalidToCurrencyLength(t *testing.T) {
	router := setupTestRouter()

	router.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
	})

	body := bytes.NewBufferString(`{"amount": 100, "from_currency": "USD", "to_currency": "EURO"}`)
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Convert_InvalidJSON(t *testing.T) {
	router := setupTestRouter()

	router.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
	})

	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Convert_ServiceError(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	mockSvc.On("Convert", mock.Anything, float64(100), "USD", "XYZ").Return(nil, errors.New("no exchange rate found"))

	h := newTestHandler(mockSvc)
	router.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}

		result, err := h.service.Convert(c.Request.Context(), req.Amount, req.FromCurrency, req.ToCurrency)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}

		formattedOriginal, _ := h.service.FormatMoney(c.Request.Context(), result.Original)
		formattedConverted, _ := h.service.FormatMoney(c.Request.Context(), result.Converted)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": ConvertResponse{
				OriginalAmount:     result.Original.Amount,
				OriginalCurrency:   result.Original.Currency,
				ConvertedAmount:    result.Converted.Amount,
				ConvertedCurrency:  result.Converted.Currency,
				ExchangeRate:       result.ExchangeRate,
				FormattedOriginal:  formattedOriginal,
				FormattedConverted: formattedConverted,
			},
		})
	})

	body := bytes.NewBufferString(`{"amount": 100, "from_currency": "USD", "to_currency": "XYZ"}`)
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))

	mockSvc.AssertExpectations(t)
}

func TestHandler_Convert_LargeAmount(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	conversionResult := &ConversionResult{
		Original:     Money{Amount: 1000000.00, Currency: "USD"},
		Converted:    Money{Amount: 850000.00, Currency: "EUR"},
		ExchangeRate: 0.85,
		ConvertedAt:  time.Now(),
	}

	mockSvc.On("Convert", mock.Anything, float64(1000000), "USD", "EUR").Return(conversionResult, nil)
	mockSvc.On("FormatMoney", mock.Anything, Money{Amount: 1000000.00, Currency: "USD"}).Return("$1,000,000.00", nil)
	mockSvc.On("FormatMoney", mock.Anything, Money{Amount: 850000.00, Currency: "EUR"}).Return("\u20ac850,000.00", nil)

	h := newTestHandler(mockSvc)
	router.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}

		result, err := h.service.Convert(c.Request.Context(), req.Amount, req.FromCurrency, req.ToCurrency)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}

		formattedOriginal, _ := h.service.FormatMoney(c.Request.Context(), result.Original)
		formattedConverted, _ := h.service.FormatMoney(c.Request.Context(), result.Converted)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": ConvertResponse{
				OriginalAmount:     result.Original.Amount,
				OriginalCurrency:   result.Original.Currency,
				ConvertedAmount:    result.Converted.Amount,
				ConvertedCurrency:  result.Converted.Currency,
				ExchangeRate:       result.ExchangeRate,
				FormattedOriginal:  formattedOriginal,
				FormattedConverted: formattedConverted,
			},
		})
	})

	body := bytes.NewBufferString(`{"amount": 1000000, "from_currency": "USD", "to_currency": "EUR"}`)
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	mockSvc.AssertExpectations(t)
}

func TestHandler_Convert_DecimalAmount(t *testing.T) {
	mockSvc := new(mockCurrencyService)
	router := setupTestRouter()

	conversionResult := &ConversionResult{
		Original:     Money{Amount: 99.99, Currency: "USD"},
		Converted:    Money{Amount: 84.99, Currency: "EUR"},
		ExchangeRate: 0.85,
		ConvertedAt:  time.Now(),
	}

	mockSvc.On("Convert", mock.Anything, 99.99, "USD", "EUR").Return(conversionResult, nil)
	mockSvc.On("FormatMoney", mock.Anything, Money{Amount: 99.99, Currency: "USD"}).Return("$99.99", nil)
	mockSvc.On("FormatMoney", mock.Anything, Money{Amount: 84.99, Currency: "EUR"}).Return("\u20ac84.99", nil)

	h := newTestHandler(mockSvc)
	router.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}

		result, err := h.service.Convert(c.Request.Context(), req.Amount, req.FromCurrency, req.ToCurrency)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}

		formattedOriginal, _ := h.service.FormatMoney(c.Request.Context(), result.Original)
		formattedConverted, _ := h.service.FormatMoney(c.Request.Context(), result.Converted)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": ConvertResponse{
				OriginalAmount:     result.Original.Amount,
				OriginalCurrency:   result.Original.Currency,
				ConvertedAmount:    result.Converted.Amount,
				ConvertedCurrency:  result.Converted.Currency,
				ExchangeRate:       result.ExchangeRate,
				FormattedOriginal:  formattedOriginal,
				FormattedConverted: formattedConverted,
			},
		})
	})

	body := bytes.NewBufferString(`{"amount": 99.99, "from_currency": "USD", "to_currency": "EUR"}`)
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	mockSvc.AssertExpectations(t)
}

// ========================================
// RESPONSE CONVERSION TESTS
// ========================================

func TestToCurrencyResponse_NilInput(t *testing.T) {
	response := ToCurrencyResponse(nil)
	assert.Nil(t, response)
}

func TestToCurrencyResponse_ValidInput(t *testing.T) {
	currency := &Currency{
		Code:          "USD",
		Name:          "US Dollar",
		Symbol:        "$",
		DecimalPlaces: 2,
		IsActive:      true,
	}

	response := ToCurrencyResponse(currency)

	assert.NotNil(t, response)
	assert.Equal(t, "USD", response.Code)
	assert.Equal(t, "US Dollar", response.Name)
	assert.Equal(t, "$", response.Symbol)
	assert.Equal(t, 2, response.DecimalPlaces)
}

func TestToExchangeRateResponse_NilInput(t *testing.T) {
	response := ToExchangeRateResponse(nil)
	assert.Nil(t, response)
}

func TestToExchangeRateResponse_ValidInput(t *testing.T) {
	validUntil := time.Now().Add(24 * time.Hour)
	rate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: "USD",
		ToCurrency:   "EUR",
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		Source:       "manual",
		ValidUntil:   validUntil,
	}

	response := ToExchangeRateResponse(rate)

	assert.NotNil(t, response)
	assert.Equal(t, "USD", response.FromCurrency)
	assert.Equal(t, "EUR", response.ToCurrency)
	assert.Equal(t, 0.85, response.Rate)
	assert.Equal(t, validUntil, response.ValidUntil)
}

// ========================================
// QUERY PARAMETER PARSING TESTS
// ========================================

func TestHandler_GetExchangeRate_QueryParamVariations(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		expectedStatus int
	}{
		{"valid params", "?from=USD&to=EUR", http.StatusOK},
		{"empty from", "?from=&to=EUR", http.StatusBadRequest},
		{"empty to", "?from=USD&to=", http.StatusBadRequest},
		{"both empty", "?from=&to=", http.StatusBadRequest},
		{"no params", "", http.StatusBadRequest},
		{"only from", "?from=USD", http.StatusBadRequest},
		{"only to", "?to=EUR", http.StatusBadRequest},
		{"lowercase valid", "?from=usd&to=eur", http.StatusOK},
		{"mixed case", "?from=UsD&to=EuR", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockCurrencyService)
			router := setupTestRouter()

			if tt.expectedStatus == http.StatusOK {
				rate := &ExchangeRate{
					FromCurrency: "USD",
					ToCurrency:   "EUR",
					Rate:         0.85,
					ValidUntil:   time.Now().Add(24 * time.Hour),
				}
				mockSvc.On("GetExchangeRate", mock.Anything, mock.Anything, mock.Anything).Return(rate, nil)
			}

			h := newTestHandler(mockSvc)
			router.GET("/rate", func(c *gin.Context) {
				from := c.Query("from")
				to := c.Query("to")

				if len(from) != 3 || len(to) != 3 {
					c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid currency codes"}})
					return
				}

				result, err := h.service.GetExchangeRate(c.Request.Context(), from, to)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"message": "exchange rate not found"}})
					return
				}
				c.JSON(http.StatusOK, gin.H{"success": true, "data": ToExchangeRateResponse(result)})
			})

			req := httptest.NewRequest(http.MethodGet, "/rate"+tt.queryString, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ========================================
// ERROR HANDLING TESTS
// ========================================

func TestHandler_ServiceError_ReturnsInternalError(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*mockCurrencyService)
		path  string
	}{
		{
			name: "GetActiveCurrencies error",
			setup: func(m *mockCurrencyService) {
				m.On("GetActiveCurrencies", mock.Anything).Return(nil, errors.New("db error"))
			},
			path: "/currencies",
		},
		{
			name: "GetAllRatesFromBase error",
			setup: func(m *mockCurrencyService) {
				m.On("GetAllRatesFromBase", mock.Anything).Return(nil, errors.New("db error"))
			},
			path: "/rates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockCurrencyService)
			router := setupTestRouter()
			tt.setup(mockSvc)

			h := newTestHandler(mockSvc)

			switch tt.path {
			case "/currencies":
				router.GET(tt.path, func(c *gin.Context) {
					_, err := h.service.GetActiveCurrencies(c.Request.Context())
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
						return
					}
				})
			case "/rates":
				router.GET(tt.path, func(c *gin.Context) {
					_, err := h.service.GetAllRatesFromBase(c.Request.Context())
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
						return
					}
				})
			}

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}

// ========================================
// CURRENCY CONVERSION LOGIC TESTS
// ========================================

func TestHandler_Convert_MultiCurrencyConversions(t *testing.T) {
	testCases := []struct {
		name         string
		amount       float64
		from         string
		to           string
		rate         float64
		expectedAmt  float64
	}{
		{"USD to EUR", 100.00, "USD", "EUR", 0.85, 85.00},
		{"EUR to USD", 100.00, "EUR", "USD", 1.18, 118.00},
		{"USD to GBP", 100.00, "USD", "GBP", 0.75, 75.00},
		{"USD to TMT", 100.00, "USD", "TMT", 3.50, 350.00},
		{"TMT to USD", 350.00, "TMT", "USD", 0.2857, 99.99},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mockCurrencyService)
			router := setupTestRouter()

			conversionResult := &ConversionResult{
				Original:     Money{Amount: tc.amount, Currency: tc.from},
				Converted:    Money{Amount: tc.expectedAmt, Currency: tc.to},
				ExchangeRate: tc.rate,
				ConvertedAt:  time.Now(),
			}

			mockSvc.On("Convert", mock.Anything, tc.amount, tc.from, tc.to).Return(conversionResult, nil)
			mockSvc.On("FormatMoney", mock.Anything, mock.Anything).Return("formatted", nil)

			h := newTestHandler(mockSvc)
			router.POST("/convert", func(c *gin.Context) {
				var req ConvertRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
					return
				}

				result, err := h.service.Convert(c.Request.Context(), req.Amount, req.FromCurrency, req.ToCurrency)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
					return
				}

				formattedOriginal, _ := h.service.FormatMoney(c.Request.Context(), result.Original)
				formattedConverted, _ := h.service.FormatMoney(c.Request.Context(), result.Converted)

				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data": ConvertResponse{
						OriginalAmount:     result.Original.Amount,
						OriginalCurrency:   result.Original.Currency,
						ConvertedAmount:    result.Converted.Amount,
						ConvertedCurrency:  result.Converted.Currency,
						ExchangeRate:       result.ExchangeRate,
						FormattedOriginal:  formattedOriginal,
						FormattedConverted: formattedConverted,
					},
				})
			})

			body := bytes.NewBufferString(`{"amount": ` + formatFloat(tc.amount) + `, "from_currency": "` + tc.from + `", "to_currency": "` + tc.to + `"}`)
			req := httptest.NewRequest(http.MethodPost, "/convert", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.True(t, response["success"].(bool))

			data := response["data"].(map[string]interface{})
			assert.Equal(t, tc.expectedAmt, data["converted_amount"])
			assert.Equal(t, tc.rate, data["exchange_rate"])

			mockSvc.AssertExpectations(t)
		})
	}
}

// Helper function to format float for JSON
func formatFloat(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

// ========================================
// REGISTER ROUTES TEST
// ========================================

func TestHandler_RegisterRoutes(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, "USD")
	handler := NewHandler(service)

	router := gin.New()
	rg := router.Group("/api/v1")
	handler.RegisterRoutes(rg)

	routes := router.Routes()

	expectedRoutes := map[string]bool{
		"GET/api/v1/currency/currencies":      false,
		"GET/api/v1/currency/currencies/:code": false,
		"GET/api/v1/currency/rates":           false,
		"GET/api/v1/currency/rate":            false,
		"POST/api/v1/currency/convert":        false,
	}

	for _, route := range routes {
		key := route.Method + route.Path
		if _, ok := expectedRoutes[key]; ok {
			expectedRoutes[key] = true
		}
	}

	for route, found := range expectedRoutes {
		assert.True(t, found, "Expected route %s to be registered", route)
	}
}
