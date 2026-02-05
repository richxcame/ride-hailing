package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/onboarding"
	"github.com/richxcame/ride-hailing/internal/pool"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/storage"
	"go.uber.org/zap"
)

// ---- Pool MapsService stub ----

type stubMapsService struct{}

func (s *stubMapsService) GetRoute(_ context.Context, origin, destination pool.Location) (*pool.RouteInfo, error) {
	logger.Warn("stubMapsService.GetRoute called — wire a real MapsService",
		zap.Float64("origin_lat", origin.Latitude), zap.Float64("origin_lng", origin.Longitude))
	return &pool.RouteInfo{DistanceKm: 0, DurationMinutes: 0}, nil
}

func (s *stubMapsService) GetMultiStopRoute(_ context.Context, stops []pool.Location) (*pool.MultiStopRouteInfo, error) {
	logger.Warn("stubMapsService.GetMultiStopRoute called — wire a real MapsService",
		zap.Int("stops", len(stops)))
	return &pool.MultiStopRouteInfo{TotalDistanceKm: 0, TotalDurationMinutes: 0}, nil
}

// ---- Recording Storage stub ----

type stubStorage struct{}

func (s *stubStorage) Upload(_ context.Context, key string, _ io.Reader, _ int64, _ string) (*storage.UploadResult, error) {
	logger.Warn("stubStorage.Upload called — wire a real Storage provider", zap.String("key", key))
	return nil, fmt.Errorf("storage not configured")
}

func (s *stubStorage) Download(_ context.Context, key string) (io.ReadCloser, error) {
	logger.Warn("stubStorage.Download called — wire a real Storage provider", zap.String("key", key))
	return nil, fmt.Errorf("storage not configured")
}

func (s *stubStorage) Delete(_ context.Context, key string) error {
	logger.Warn("stubStorage.Delete called — wire a real Storage provider", zap.String("key", key))
	return fmt.Errorf("storage not configured")
}

func (s *stubStorage) GetURL(key string) string { return "" }

func (s *stubStorage) GetPresignedUploadURL(_ context.Context, key, _ string, _ time.Duration) (*storage.PresignedURLResult, error) {
	return nil, fmt.Errorf("storage not configured")
}

func (s *stubStorage) GetPresignedDownloadURL(_ context.Context, key string, _ time.Duration) (*storage.PresignedURLResult, error) {
	return nil, fmt.Errorf("storage not configured")
}

func (s *stubStorage) Exists(_ context.Context, _ string) (bool, error) { return false, nil }
func (s *stubStorage) Copy(_ context.Context, _, _ string) error        { return fmt.Errorf("storage not configured") }

// ---- Onboarding DocumentService stub ----

type stubDocumentService struct{}

func (s *stubDocumentService) GetDriverDocuments(_ context.Context, driverID uuid.UUID) ([]onboarding.DocumentInfo, error) {
	logger.Warn("stubDocumentService.GetDriverDocuments called — wire a real DocumentService",
		zap.String("driver_id", driverID.String()))
	return []onboarding.DocumentInfo{}, nil
}

func (s *stubDocumentService) GetRequiredDocumentTypes(_ context.Context) ([]onboarding.DocumentTypeInfo, error) {
	logger.Warn("stubDocumentService.GetRequiredDocumentTypes called — wire a real DocumentService")
	return []onboarding.DocumentTypeInfo{}, nil
}

func (s *stubDocumentService) GetDriverVerificationStatus(_ context.Context, driverID uuid.UUID) (*onboarding.VerificationStatus, error) {
	logger.Warn("stubDocumentService.GetDriverVerificationStatus called — wire a real DocumentService",
		zap.String("driver_id", driverID.String()))
	return &onboarding.VerificationStatus{Status: "unknown"}, nil
}

// ---- PaymentSplit PaymentService stub ----

type stubPaymentService struct{}

func (s *stubPaymentService) GetRideFare(_ context.Context, rideID uuid.UUID) (float64, string, error) {
	logger.Warn("stubPaymentService.GetRideFare called — wire a real PaymentService",
		zap.String("ride_id", rideID.String()))
	return 0, "USD", fmt.Errorf("payment service not configured")
}

func (s *stubPaymentService) ProcessSplitPayment(_ context.Context, userID, rideID uuid.UUID, amount float64, method string) (uuid.UUID, error) {
	logger.Warn("stubPaymentService.ProcessSplitPayment called — wire a real PaymentService",
		zap.String("user_id", userID.String()), zap.Float64("amount", amount))
	return uuid.Nil, fmt.Errorf("payment service not configured")
}

// ---- PaymentSplit NotificationService stub ----

type stubSplitNotificationService struct{}

func (s *stubSplitNotificationService) SendSplitInvitation(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string, _ float64) error {
	logger.Warn("stubSplitNotificationService.SendSplitInvitation called — wire a real NotificationService")
	return nil
}

func (s *stubSplitNotificationService) SendSplitReminder(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ float64) error {
	return nil
}

func (s *stubSplitNotificationService) SendSplitAccepted(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (s *stubSplitNotificationService) SendSplitCompleted(_ context.Context, _ uuid.UUID) error {
	return nil
}

// ---- Subscriptions PaymentProcessor stub ----

type stubPaymentProcessor struct{}

func (s *stubPaymentProcessor) ChargeSubscription(_ context.Context, userID uuid.UUID, amount float64, currency, paymentMethod string) error {
	logger.Warn("stubPaymentProcessor.ChargeSubscription called — wire a real PaymentProcessor",
		zap.String("user_id", userID.String()), zap.Float64("amount", amount))
	return fmt.Errorf("payment processor not configured")
}

// ---- 2FA SMSSender stub ----

type stubSMSSender struct{}

func (s *stubSMSSender) SendOTP(to, otp string) (string, error) {
	logger.Warn("stubSMSSender.SendOTP called — wire a real SMSSender",
		zap.String("to", to[:3]+"***"))
	return "", fmt.Errorf("SMS sender not configured")
}

// ---- DemandForecast stubs (already nil-safe, but let's be explicit) ----
// demandforecast service nil-guards both weatherSvc and driverSvc, so nil is safe.
// No stubs needed.
