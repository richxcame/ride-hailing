package geo

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/logger"
	redisClient "github.com/richxcame/ride-hailing/pkg/redis"
	"go.uber.org/zap"
)

// LocationUpdate represents a buffered driver location update.
type LocationUpdate struct {
	DriverID  uuid.UUID
	Latitude  float64
	Longitude float64
	Heading   float64
	Speed     float64
	H3Cell    string
	Timestamp time.Time
}

// LocationBufferConfig configures the location batching pipeline.
type LocationBufferConfig struct {
	// FlushInterval is how often the buffer flushes to Redis.
	FlushInterval time.Duration
	// MaxBufferSize triggers a flush when the buffer reaches this size.
	MaxBufferSize int
}

// DefaultLocationBufferConfig returns sensible defaults.
func DefaultLocationBufferConfig() LocationBufferConfig {
	return LocationBufferConfig{
		FlushInterval: 500 * time.Millisecond,
		MaxBufferSize: 100,
	}
}

// LocationBuffer accumulates driver location updates and flushes them
// to Redis in batches, reducing per-update round-trip overhead.
type LocationBuffer struct {
	redis   redisClient.ClientInterface
	cfg     LocationBufferConfig
	mu      sync.Mutex
	buffer  []LocationUpdate
	stopCh  chan struct{}
	stopped bool
}

// NewLocationBuffer creates and starts a location batching pipeline.
func NewLocationBuffer(redis redisClient.ClientInterface, cfg LocationBufferConfig) *LocationBuffer {
	lb := &LocationBuffer{
		redis:  redis,
		cfg:    cfg,
		buffer: make([]LocationUpdate, 0, cfg.MaxBufferSize),
		stopCh: make(chan struct{}),
	}
	go lb.flushLoop()
	return lb
}

// Enqueue adds a location update to the buffer.
// If the buffer is full, it triggers an immediate flush.
func (lb *LocationBuffer) Enqueue(update LocationUpdate) {
	lb.mu.Lock()
	lb.buffer = append(lb.buffer, update)
	shouldFlush := len(lb.buffer) >= lb.cfg.MaxBufferSize
	lb.mu.Unlock()

	if shouldFlush {
		go lb.flush()
	}
}

// Stop stops the flush loop and flushes remaining updates.
func (lb *LocationBuffer) Stop() {
	lb.mu.Lock()
	if lb.stopped {
		lb.mu.Unlock()
		return
	}
	lb.stopped = true
	lb.mu.Unlock()
	close(lb.stopCh)
	lb.flush()
}

func (lb *LocationBuffer) flushLoop() {
	ticker := time.NewTicker(lb.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lb.flush()
		case <-lb.stopCh:
			return
		}
	}
}

func (lb *LocationBuffer) flush() {
	lb.mu.Lock()
	if len(lb.buffer) == 0 {
		lb.mu.Unlock()
		return
	}
	batch := lb.buffer
	lb.buffer = make([]LocationUpdate, 0, lb.cfg.MaxBufferSize)
	lb.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Deduplicate: keep only the latest update per driver
	latest := make(map[uuid.UUID]*LocationUpdate, len(batch))
	for i := range batch {
		u := &batch[i]
		if existing, ok := latest[u.DriverID]; !ok || u.Timestamp.After(existing.Timestamp) {
			latest[u.DriverID] = u
		}
	}

	for _, u := range latest {
		lb.writeLocation(ctx, u)
	}

	logger.Debug("location buffer flushed",
		zap.Int("batch_size", len(batch)),
		zap.Int("unique_drivers", len(latest)),
	)
}

func (lb *LocationBuffer) writeLocation(ctx context.Context, u *LocationUpdate) {
	location := &DriverLocation{
		DriverID:  u.DriverID,
		Latitude:  u.Latitude,
		Longitude: u.Longitude,
		H3Cell:    u.H3Cell,
		Heading:   u.Heading,
		Speed:     u.Speed,
		Timestamp: u.Timestamp,
	}

	data, err := json.Marshal(location)
	if err != nil {
		logger.Warn("failed to marshal buffered location", zap.Error(err))
		return
	}

	key := fmt.Sprintf("%s%s", driverLocationPrefix, u.DriverID.String())
	if err := lb.redis.SetWithExpiration(ctx, key, data, driverLocationTTL); err != nil {
		logger.Warn("failed to write buffered location", zap.String("driver_id", u.DriverID.String()), zap.Error(err))
		return
	}

	// Update geo index
	if err := lb.redis.GeoAdd(ctx, driverGeoIndexKey, u.Longitude, u.Latitude, u.DriverID.String()); err != nil {
		logger.Warn("failed to update geo index", zap.String("driver_id", u.DriverID.String()), zap.Error(err))
	}

	// Update H3 cell index
	driverIDStr := u.DriverID.String()
	prevCellKey := fmt.Sprintf("driver:h3cell:%s", driverIDStr)

	prevCell, err := lb.redis.GetString(ctx, prevCellKey)
	if err == nil && prevCell != "" && prevCell != u.H3Cell {
		oldSetKey := fmt.Sprintf("%s%s:%s", h3CellDriversPrefix, prevCell, driverIDStr)
		lb.redis.Delete(ctx, oldSetKey)
	}

	lb.redis.SetWithExpiration(ctx, prevCellKey, []byte(u.H3Cell), driverLocationTTL)
	newSetKey := fmt.Sprintf("%s%s:%s", h3CellDriversPrefix, u.H3Cell, driverIDStr)
	lb.redis.SetWithExpiration(ctx, newSetKey, []byte(driverIDStr), h3CellDriversTTL)
}
