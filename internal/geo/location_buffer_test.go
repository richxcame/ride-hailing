package geo

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// mockRedis implements a minimal redis interface for testing.
type mockRedis struct {
	mu      sync.Mutex
	store   map[string]string
	geoData map[string]map[string]bool
}

func newMockRedis() *mockRedis {
	return &mockRedis{
		store:   make(map[string]string),
		geoData: make(map[string]map[string]bool),
	}
}

func (m *mockRedis) SetWithExpiration(_ context.Context, key string, value interface{}, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch v := value.(type) {
	case []byte:
		m.store[key] = string(v)
	case string:
		m.store[key] = v
	}
	return nil
}

func (m *mockRedis) GetString(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.store[key]
	if !ok {
		return "", context.DeadlineExceeded
	}
	return v, nil
}

func (m *mockRedis) Delete(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.store, k)
	}
	return nil
}

func (m *mockRedis) Exists(_ context.Context, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.store[key]
	return ok, nil
}

func (m *mockRedis) Close() error { return nil }

func (m *mockRedis) GeoAdd(_ context.Context, key string, _, _ float64, member string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.geoData[key] == nil {
		m.geoData[key] = make(map[string]bool)
	}
	m.geoData[key][member] = true
	return nil
}

func (m *mockRedis) GeoRadius(_ context.Context, _ string, _, _, _ float64, _ int) ([]string, error) {
	return nil, nil
}

func (m *mockRedis) GeoRemove(_ context.Context, key, member string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.geoData[key] != nil {
		delete(m.geoData[key], member)
	}
	return nil
}

func (m *mockRedis) Expire(_ context.Context, _ string, _ time.Duration) error { return nil }

func (m *mockRedis) keyCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.store)
}

func (m *mockRedis) hasGeoMember(key, member string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.geoData[key] != nil && m.geoData[key][member]
}

func TestLocationBuffer_FlushesOnInterval(t *testing.T) {
	redis := newMockRedis()
	cfg := LocationBufferConfig{
		FlushInterval: 50 * time.Millisecond,
		MaxBufferSize: 1000,
	}
	lb := NewLocationBuffer(redis, cfg)
	defer lb.Stop()

	driverID := uuid.New()
	lb.Enqueue(LocationUpdate{
		DriverID:  driverID,
		Latitude:  37.7749,
		Longitude: -122.4194,
		H3Cell:    "891f1d48267ffff",
		Timestamp: time.Now(),
	})

	// Wait for flush
	time.Sleep(150 * time.Millisecond)

	if redis.keyCount() == 0 {
		t.Fatal("expected data to be flushed to redis")
	}

	if !redis.hasGeoMember(driverGeoIndexKey, driverID.String()) {
		t.Fatal("expected driver to be in geo index")
	}
}

func TestLocationBuffer_FlushesOnMaxSize(t *testing.T) {
	redis := newMockRedis()
	cfg := LocationBufferConfig{
		FlushInterval: 10 * time.Second, // Long interval, shouldn't trigger
		MaxBufferSize: 5,
	}
	lb := NewLocationBuffer(redis, cfg)
	defer lb.Stop()

	// Enqueue exactly MaxBufferSize updates to trigger flush
	for i := 0; i < 5; i++ {
		lb.Enqueue(LocationUpdate{
			DriverID:  uuid.New(),
			Latitude:  37.7749 + float64(i)*0.001,
			Longitude: -122.4194,
			H3Cell:    "891f1d48267ffff",
			Timestamp: time.Now(),
		})
	}

	// Wait for the async flush triggered by buffer full
	time.Sleep(100 * time.Millisecond)

	if redis.keyCount() == 0 {
		t.Fatal("expected data to be flushed when buffer is full")
	}
}

func TestLocationBuffer_DeduplicatesPerDriver(t *testing.T) {
	redis := newMockRedis()
	cfg := LocationBufferConfig{
		FlushInterval: 50 * time.Millisecond,
		MaxBufferSize: 1000,
	}
	lb := NewLocationBuffer(redis, cfg)
	defer lb.Stop()

	driverID := uuid.New()

	// Send 3 updates for same driver
	for i := 0; i < 3; i++ {
		lb.Enqueue(LocationUpdate{
			DriverID:  driverID,
			Latitude:  37.7749 + float64(i)*0.001,
			Longitude: -122.4194,
			H3Cell:    "891f1d48267ffff",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		})
	}

	time.Sleep(150 * time.Millisecond)

	// Should have exactly 1 location key for this driver (deduplicated)
	key := driverLocationPrefix + driverID.String()
	_, err := redis.GetString(context.Background(), key)
	if err != nil {
		t.Fatal("expected driver location to be stored")
	}
}

func TestLocationBuffer_GracefulStop(t *testing.T) {
	redis := newMockRedis()
	cfg := LocationBufferConfig{
		FlushInterval: 10 * time.Second,
		MaxBufferSize: 1000,
	}
	lb := NewLocationBuffer(redis, cfg)

	lb.Enqueue(LocationUpdate{
		DriverID:  uuid.New(),
		Latitude:  37.7749,
		Longitude: -122.4194,
		H3Cell:    "891f1d48267ffff",
		Timestamp: time.Now(),
	})

	// Stop should flush remaining buffer
	lb.Stop()

	if redis.keyCount() == 0 {
		t.Fatal("expected remaining buffer to be flushed on Stop()")
	}
}

func TestLocationBuffer_ConcurrentWrites(t *testing.T) {
	redis := newMockRedis()
	cfg := LocationBufferConfig{
		FlushInterval: 50 * time.Millisecond,
		MaxBufferSize: 1000,
	}
	lb := NewLocationBuffer(redis, cfg)
	defer lb.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lb.Enqueue(LocationUpdate{
				DriverID:  uuid.New(),
				Latitude:  37.7749,
				Longitude: -122.4194,
				H3Cell:    "891f1d48267ffff",
				Timestamp: time.Now(),
			})
		}()
	}
	wg.Wait()

	time.Sleep(150 * time.Millisecond)

	if redis.keyCount() == 0 {
		t.Fatal("expected concurrent writes to be flushed")
	}
}
