//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v83"

	"github.com/richxcame/ride-hailing/internal/auth"
	"github.com/richxcame/ride-hailing/internal/payments"
	"github.com/richxcame/ride-hailing/internal/rides"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/database"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/test/helpers"
)

const (
	authServiceKey     = "auth"
	ridesServiceKey    = "rides"
	paymentsServiceKey = "payments"
)

type errorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type apiResponse[T any] struct {
	Success bool         `json:"success"`
	Data    T            `json:"data"`
	Error   *errorInfo   `json:"error"`
	Meta    *common.Meta `json:"meta,omitempty"`
}

type serviceInstance struct {
	server  *httptest.Server
	client  *http.Client
	baseURL string
}

type authSession struct {
	User  *models.User
	Token string
}

type fakeStripeClient struct{}

var (
	services   map[string]*serviceInstance
	dbPool     *pgxpool.Pool
	phoneCount uint64
)

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("PORT", "18080")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpassword")
	os.Setenv("DB_NAME", "ride_hailing_test")
	os.Setenv("DB_SSLMODE", "disable")
	os.Setenv("JWT_SECRET", "integration-secret")
	os.Setenv("RATE_LIMIT_ENABLED", "false")
	os.Setenv("PROMOS_SERVICE_URL", "")
	os.Setenv("CB_ENABLED", "false")

	cfgAuth := mustLoadConfig("auth-service")
	cfgRides := mustLoadConfig("rides-service")
	cfgPayments := mustLoadConfig("payments-service")

	if err := logger.Init(cfgAuth.Server.Environment); err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}

	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard

	var err error
	dbPool, err = database.NewPostgresPool(&cfgAuth.Database)
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}

	services = make(map[string]*serviceInstance)
	services[authServiceKey] = startAuthService(cfgAuth)
	services[ridesServiceKey] = startRidesService(cfgRides)
	services[paymentsServiceKey] = startPaymentsService(cfgPayments)

	exitCode := m.Run()

	for _, svc := range services {
		if svc != nil && svc.server != nil {
			svc.server.Close()
		}
	}

	database.Close(dbPool)
	_ = logger.Sync()

	os.Exit(exitCode)
}

func mustLoadConfig(service string) *config.Config {
	cfg, err := config.Load(service)
	if err != nil {
		panic(fmt.Sprintf("failed to load config for %s: %v", service, err))
	}
	return cfg
}

func startAuthService(cfg *config.Config) *serviceInstance {
	repo := auth.NewRepository(dbPool)
	service := auth.NewService(repo, cfg.JWT.Secret, cfg.JWT.Expiration)
	handler := auth.NewHandler(service)

	router := newRouter(cfg.Server.ServiceName)
	handler.RegisterRoutes(router, cfg.JWT.Secret)

	server := httptest.NewServer(router)
	return &serviceInstance{server: server, client: server.Client(), baseURL: server.URL}
}

func startRidesService(cfg *config.Config) *serviceInstance {
	repo := rides.NewRepository(dbPool)
	service := rides.NewService(repo, "", nil)
	handler := rides.NewHandler(service)

	router := newRouter(cfg.Server.ServiceName)
	handler.RegisterRoutes(router, cfg.JWT.Secret, nil, cfg.RateLimit)

	server := httptest.NewServer(router)
	return &serviceInstance{server: server, client: server.Client(), baseURL: server.URL}
}

func startPaymentsService(cfg *config.Config) *serviceInstance {
	repo := payments.NewRepository(dbPool)
	service := payments.NewService(repo, &fakeStripeClient{}, &cfg.Business)
	handler := payments.NewHandler(service)

	router := newRouter(cfg.Server.ServiceName)
	handler.RegisterRoutes(router, cfg.JWT.Secret)

	server := httptest.NewServer(router)
	return &serviceInstance{server: server, client: server.Client(), baseURL: server.URL}
}

func newRouter(serviceName string) *gin.Engine {
	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestLogger(serviceName))
	router.Use(middleware.CORS())
	router.Use(middleware.Metrics(serviceName))
	router.GET("/healthz", common.HealthCheck(serviceName, "integration"))
	return router
}

func (f *fakeStripeClient) CreateCustomer(email, name string, metadata map[string]string) (*stripe.Customer, error) {
	return &stripe.Customer{ID: "cus_test"}, nil
}

func (f *fakeStripeClient) CreatePaymentIntent(amount int64, currency, customerID, description string, metadata map[string]string) (*stripe.PaymentIntent, error) {
	return &stripe.PaymentIntent{ID: "pi_test", Status: stripe.PaymentIntentStatusSucceeded}, nil
}

func (f *fakeStripeClient) ConfirmPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	return &stripe.PaymentIntent{ID: paymentIntentID, Status: stripe.PaymentIntentStatusSucceeded}, nil
}

func (f *fakeStripeClient) CapturePaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	return &stripe.PaymentIntent{ID: paymentIntentID, Status: stripe.PaymentIntentStatusSucceeded}, nil
}

func (f *fakeStripeClient) CreateRefund(chargeID string, amount *int64, reason string) (*stripe.Refund, error) {
	return &stripe.Refund{ID: "re_test"}, nil
}

func (f *fakeStripeClient) CreateTransfer(amount int64, currency, destination, description string, metadata map[string]string) (*stripe.Transfer, error) {
	return &stripe.Transfer{ID: "tr_test"}, nil
}

func (f *fakeStripeClient) GetPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	return &stripe.PaymentIntent{ID: paymentIntentID, Status: stripe.PaymentIntentStatusSucceeded}, nil
}

func (f *fakeStripeClient) CancelPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	return &stripe.PaymentIntent{ID: paymentIntentID, Status: stripe.PaymentIntentStatusCanceled}, nil
}

func TestAuthIntegration_RegisterLoginAndProfile(t *testing.T) {
	truncateTables(t)

	registerReq := helpers.CreateTestRegisterRequest()
	registerReq.Email = uniqueEmail(models.RoleRider)
	registerReq.PhoneNumber = uniquePhoneNumber()

	registerResp := doRequest[models.User](t, authServiceKey, http.MethodPost, "/api/v1/auth/register", registerReq, nil)
	require.True(t, registerResp.Success)
	require.NotEmpty(t, registerResp.Data.ID)
	require.Equal(t, registerReq.Email, registerResp.Data.Email)

	loginReq := &models.LoginRequest{Email: registerReq.Email, Password: registerReq.Password}
	loginResp := doRequest[*models.LoginResponse](t, authServiceKey, http.MethodPost, "/api/v1/auth/login", loginReq, nil)
	require.True(t, loginResp.Success)
	require.NotNil(t, loginResp.Data)
	require.Equal(t, registerReq.Email, loginResp.Data.User.Email)
	require.NotEmpty(t, loginResp.Data.Token)

	var walletCount int
	err := dbPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM wallets WHERE user_id = $1", registerResp.Data.ID).Scan(&walletCount)
	require.NoError(t, err)
	require.Equal(t, 1, walletCount)

	headers := authHeaders(loginResp.Data.Token)
	profileResp := doRequest[models.User](t, authServiceKey, http.MethodGet, "/api/v1/auth/profile", nil, headers)
	require.True(t, profileResp.Success)
	require.Equal(t, registerResp.Data.ID, profileResp.Data.ID)
	require.Equal(t, registerReq.Email, profileResp.Data.Email)

	unauthorized := doRawRequest(t, authServiceKey, http.MethodGet, "/api/v1/auth/profile", nil, nil)
	defer unauthorized.Body.Close()
	require.Equal(t, http.StatusUnauthorized, unauthorized.StatusCode)
}

func TestAuthIntegration_RegisterDuplicateEmail(t *testing.T) {
	truncateTables(t)

	registerReq := helpers.CreateTestRegisterRequest()
	registerReq.Email = uniqueEmail(models.RoleRider)
	registerReq.PhoneNumber = uniquePhoneNumber()

	first := doRequest[models.User](t, authServiceKey, http.MethodPost, "/api/v1/auth/register", registerReq, nil)
	require.True(t, first.Success)

	dup := doRawRequest(t, authServiceKey, http.MethodPost, "/api/v1/auth/register", registerReq, nil)
	defer dup.Body.Close()
	require.Equal(t, http.StatusConflict, dup.StatusCode)

	var resp apiResponse[any]
	decodeBody(t, dup, &resp)
	require.False(t, resp.Success)
	require.NotNil(t, resp.Error)
	require.Equal(t, http.StatusConflict, resp.Error.Code)
}

func TestRidesIntegration_RiderDriverLifecycle(t *testing.T) {
	truncateTables(t)

	rider := registerAndLogin(t, models.RoleRider)
	driver := registerAndLogin(t, models.RoleDriver)

	rideReq := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "1 Market St",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "12 Broadway",
	}

	rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(rider.Token))
	require.True(t, rideResp.Success)
	require.NotNil(t, rideResp.Data)
	require.Equal(t, models.RideStatusRequested, rideResp.Data.Status)
	require.Equal(t, rider.User.ID, rideResp.Data.RiderID)

	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideResp.Data.ID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(driver.Token))
	require.True(t, acceptResp.Success)
	require.Equal(t, models.RideStatusAccepted, acceptResp.Data.Status)
	require.NotNil(t, acceptResp.Data.DriverID)
	require.Equal(t, driver.User.ID, *acceptResp.Data.DriverID)

	startPath := fmt.Sprintf("/api/v1/driver/rides/%s/start", rideResp.Data.ID)
	startResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, startPath, nil, authHeaders(driver.Token))
	require.True(t, startResp.Success)
	require.Equal(t, models.RideStatusInProgress, startResp.Data.Status)

	completePath := fmt.Sprintf("/api/v1/driver/rides/%s/complete", rideResp.Data.ID)
	completeReq := map[string]float64{"actual_distance": 11.2}
	completeResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, completePath, completeReq, authHeaders(driver.Token))
	require.True(t, completeResp.Success)
	require.Equal(t, models.RideStatusCompleted, completeResp.Data.Status)
	require.NotNil(t, completeResp.Data.FinalFare)

	rideDetail := doRequest[*models.Ride](t, ridesServiceKey, http.MethodGet, fmt.Sprintf("/api/v1/rides/%s", rideResp.Data.ID), nil, authHeaders(rider.Token))
	require.True(t, rideDetail.Success)
	require.Equal(t, models.RideStatusCompleted, rideDetail.Data.Status)

	riderRides := doRequest[[]*models.Ride](t, ridesServiceKey, http.MethodGet, "/api/v1/rides", nil, authHeaders(rider.Token))
	require.True(t, riderRides.Success)
	require.Len(t, riderRides.Data, 1)
	require.Equal(t, rideResp.Data.ID, riderRides.Data[0].ID)

	unauthorizedDriver := doRawRequest(t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(rider.Token))
	defer unauthorizedDriver.Body.Close()
	require.Equal(t, http.StatusForbidden, unauthorizedDriver.StatusCode)

	var status string
	err := dbPool.QueryRow(context.Background(), "SELECT status FROM rides WHERE id = $1", rideResp.Data.ID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, string(models.RideStatusCompleted), status)
}

func TestPaymentsIntegration_WalletOperations(t *testing.T) {
	truncateTables(t)

	rider := registerAndLogin(t, models.RoleRider)
	headers := authHeaders(rider.Token)

	walletResp := doRequest[*models.Wallet](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, headers)
	require.True(t, walletResp.Success)
	require.NotNil(t, walletResp.Data)
	require.Equal(t, rider.User.ID, walletResp.Data.UserID)
	require.Equal(t, 0.0, walletResp.Data.Balance)

	topUpReq := map[string]any{"amount": 25.50, "stripe_payment_method": "pm_test"}
	topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, headers)
	require.True(t, topUpResp.Success)
	require.NotNil(t, topUpResp.Data)
	require.Equal(t, "credit", topUpResp.Data.Type)
	require.InEpsilon(t, 25.50, topUpResp.Data.Amount, 1e-6)

	var txCount int
	err := dbPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM wallet_transactions WHERE wallet_id = $1", topUpResp.Data.WalletID).Scan(&txCount)
	require.NoError(t, err)
	require.Equal(t, 1, txCount)

	transactionsResp := doRequest[[]*models.WalletTransaction](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet/transactions", nil, headers)
	require.True(t, transactionsResp.Success)
	require.Len(t, transactionsResp.Data, 1)

	unauthorized := doRawRequest(t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, nil)
	defer unauthorized.Body.Close()
	require.Equal(t, http.StatusUnauthorized, unauthorized.StatusCode)
}

func registerAndLogin(t *testing.T, role models.UserRole) authSession {
	registerReq := helpers.CreateTestRegisterRequest()
	registerReq.Role = role
	registerReq.Email = uniqueEmail(role)
	registerReq.PhoneNumber = uniquePhoneNumber()

	registerResp := doRequest[models.User](t, authServiceKey, http.MethodPost, "/api/v1/auth/register", registerReq, nil)
	require.True(t, registerResp.Success)

	loginReq := &models.LoginRequest{Email: registerReq.Email, Password: registerReq.Password}
	loginResp := doRequest[*models.LoginResponse](t, authServiceKey, http.MethodPost, "/api/v1/auth/login", loginReq, nil)
	require.True(t, loginResp.Success)
	require.NotNil(t, loginResp.Data)
	require.NotNil(t, loginResp.Data.User)

	return authSession{User: loginResp.Data.User, Token: loginResp.Data.Token}
}

func truncateTables(t *testing.T) {
	t.Helper()
	tables := []string{
		"wallet_transactions",
		"payments",
		"rides",
		"drivers",
		"wallets",
		"users",
	}

	for _, table := range tables {
		_, err := dbPool.Exec(context.Background(), fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		require.NoError(t, err, "failed to truncate %s", table)
	}
}

func doRequest[T any](t *testing.T, serviceKey, method, path string, body interface{}, headers map[string]string) apiResponse[T] {
	t.Helper()
	resp := doRawRequest(t, serviceKey, method, path, body, headers)
	defer resp.Body.Close()

	require.Less(t, resp.StatusCode, 500, "unexpected server error: %d", resp.StatusCode)

	var result apiResponse[T]
	decodeBody(t, resp, &result)
	return result
}

func doRawRequest(t *testing.T, serviceKey, method, path string, body interface{}, headers map[string]string) *http.Response {
	t.Helper()

	svc, ok := services[serviceKey]
	require.True(t, ok, "service %s not registered", serviceKey)

	var payload io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		require.NoError(t, err)
		payload = bytes.NewReader(buf)
	}

	req, err := http.NewRequest(method, svc.baseURL+path, payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := svc.client.Do(req)
	require.NoError(t, err)
	return resp
}

func decodeBody[T any](t *testing.T, resp *http.Response, dest *T) {
	t.Helper()
	decoder := json.NewDecoder(resp.Body)
	require.NoError(t, decoder.Decode(dest))
}

func authHeaders(token string) map[string]string {
	return map[string]string{"Authorization": fmt.Sprintf("Bearer %s", token)}
}

func uniqueEmail(role models.UserRole) string {
	return fmt.Sprintf("integration-%s-%s@example.com", role, uuid.NewString())
}

func uniquePhoneNumber() string {
	current := atomic.AddUint64(&phoneCount, 1)
	return fmt.Sprintf("+1551%08d", current)
}
