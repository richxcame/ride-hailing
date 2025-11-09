package jwtkeys

import (
	"context"
	"time"

	"github.com/richxcame/ride-hailing/pkg/config"
)

// NewManagerFromConfig builds a Manager using the shared JWT configuration.
func NewManagerFromConfig(ctx context.Context, cfg config.JWTConfig, readOnly bool) (*Manager, error) {
	return NewManager(ctx, Config{
		KeyFilePath:      cfg.KeyFile,
		RotationInterval: time.Duration(cfg.RotationHours) * time.Hour,
		GracePeriod:      time.Duration(cfg.GraceHours) * time.Hour,
		LegacySecret:     cfg.Secret,
		ReadOnly:         readOnly,
	})
}
