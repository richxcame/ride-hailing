package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/pkg/storage"
	"go.uber.org/zap"
)

// ---- Storage stub (for documents service — admin only reviews, no uploads) ----

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

// ---- DriverService stub (for documents handler — admin endpoints don't use driver lookup) ----

type stubDriverService struct{}

func (s *stubDriverService) GetDriverByUserID(_ context.Context, userID uuid.UUID) (*models.Driver, error) {
	logger.Warn("stubDriverService.GetDriverByUserID called — admin endpoints don't require this",
		zap.String("user_id", userID.String()))
	return nil, fmt.Errorf("driver service not configured for admin")
}
