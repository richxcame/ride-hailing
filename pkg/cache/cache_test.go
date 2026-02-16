package cache

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// MockRedisClient implements the Redis operations needed by cache.Manager
type MockRedisClient struct {
	mu        sync.RWMutex
	data      map[string]string
	expiry    map[string]time.Time
	getError  error
	setError  error
	delError  error
	scanError error
	scanKeys  []string
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data:   make(map[string]string),
		expiry: make(map[string]time.Time),
	}
}

func (m *MockRedisClient) GetString(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getError != nil {
		return "", m.getError
	}

	// Check expiration
	if exp, ok := m.expiry[key]; ok && time.Now().After(exp) {
		return "", redis.Nil
	}

	val, ok := m.data[key]
	if !ok {
		return "", redis.Nil
	}
	return val, nil
}

func (m *MockRedisClient) SetWithExpiration(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.setError != nil {
		return m.setError
	}

	var strVal string
	switch v := value.(type) {
	case string:
		strVal = v
	case []byte:
		strVal = string(v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		strVal = string(data)
	}

	m.data[key] = strVal
	if expiration > 0 {
		m.expiry[key] = time.Now().Add(expiration)
	}
	return nil
}

func (m *MockRedisClient) Delete(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.delError != nil {
		return m.delError
	}

	for _, key := range keys {
		delete(m.data, key)
		delete(m.expiry, key)
	}
	return nil
}

type scanResult struct {
	keys   []string
	cursor uint64
	err    error
}

func (s scanResult) Result() ([]string, uint64, error) {
	return s.keys, s.cursor, s.err
}

func (m *MockRedisClient) Scan(ctx context.Context, cursor uint64, pattern string, count int64) interface{ Result() ([]string, uint64, error) } {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.scanError != nil {
		return scanResult{err: m.scanError}
	}

	if m.scanKeys != nil {
		return scanResult{keys: m.scanKeys, cursor: 0}
	}

	// Simple pattern matching for tests
	var matched []string
	for key := range m.data {
		// Very simple wildcard matching
		if matchPattern(pattern, key) {
			matched = append(matched, key)
		}
	}
	return scanResult{keys: matched, cursor: 0}
}

func matchPattern(pattern, key string) bool {
	// Simple implementation - just check prefix before *
	if len(pattern) == 0 {
		return true
	}
	if pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}
	return pattern == key
}

// MockManager wraps cache operations for testing
type MockManager struct {
	redis *MockRedisClient
}

func NewMockManager(redis *MockRedisClient) *MockManager {
	return &MockManager{redis: redis}
}

func (m *MockManager) Get(ctx context.Context, key string, result interface{}) error {
	data, err := m.redis.GetString(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), result)
}

func (m *MockManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.redis.SetWithExpiration(ctx, key, string(data), ttl)
}

func (m *MockManager) Delete(ctx context.Context, keys ...string) error {
	return m.redis.Delete(ctx, keys...)
}

// Test structures
type TestUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type TestRide struct {
	ID         string  `json:"id"`
	DriverID   string  `json:"driver_id"`
	PassengerID string `json:"passenger_id"`
	Status     string  `json:"status"`
	Fare       float64 `json:"fare"`
}

// ============== Cache Manager Tests ==============

func TestMockManager_Get_Success(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	user := TestUser{ID: "user-1", Name: "John Doe", Email: "john@example.com"}
	_ = manager.Set(ctx, "user:1", user, time.Hour)

	var result TestUser
	err := manager.Get(ctx, "user:1", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, result.ID)
	}
	if result.Name != user.Name {
		t.Errorf("expected Name %s, got %s", user.Name, result.Name)
	}
	if result.Email != user.Email {
		t.Errorf("expected Email %s, got %s", user.Email, result.Email)
	}
}

func TestMockManager_Get_CacheMiss(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	var result TestUser
	err := manager.Get(ctx, "nonexistent", &result)
	if err == nil {
		t.Fatal("expected error for cache miss")
	}
}

func TestMockManager_Get_Error(t *testing.T) {
	mock := NewMockRedisClient()
	mock.getError = errors.New("connection error")
	manager := NewMockManager(mock)
	ctx := context.Background()

	var result TestUser
	err := manager.Get(ctx, "user:1", &result)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "connection error" {
		t.Errorf("expected 'connection error', got %v", err)
	}
}

func TestMockManager_Get_InvalidJSON(t *testing.T) {
	mock := NewMockRedisClient()
	mock.data["invalid"] = "not valid json"
	manager := NewMockManager(mock)
	ctx := context.Background()

	var result TestUser
	err := manager.Get(ctx, "invalid", &result)
	if err == nil {
		t.Fatal("expected JSON unmarshal error")
	}
}

func TestMockManager_Set_Success(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	user := TestUser{ID: "user-1", Name: "Jane Doe", Email: "jane@example.com"}
	err := manager.Set(ctx, "user:1", user, time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify data was stored
	if _, ok := mock.data["user:1"]; !ok {
		t.Error("expected data to be stored in mock")
	}
}

func TestMockManager_Set_WithZeroTTL(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	user := TestUser{ID: "user-1"}
	err := manager.Set(ctx, "user:1", user, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify no expiry was set
	if _, ok := mock.expiry["user:1"]; ok {
		t.Error("expected no expiry for zero TTL")
	}
}

func TestMockManager_Set_Error(t *testing.T) {
	mock := NewMockRedisClient()
	mock.setError = errors.New("write error")
	manager := NewMockManager(mock)
	ctx := context.Background()

	user := TestUser{ID: "user-1"}
	err := manager.Set(ctx, "user:1", user, time.Hour)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMockManager_Set_ComplexStruct(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	ride := TestRide{
		ID:          "ride-123",
		DriverID:    "driver-456",
		PassengerID: "passenger-789",
		Status:      "in_progress",
		Fare:        25.50,
	}

	err := manager.Set(ctx, "ride:123", ride, time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var result TestRide
	err = manager.Get(ctx, "ride:123", &result)
	if err != nil {
		t.Fatalf("expected no error on get, got %v", err)
	}

	if result.Fare != ride.Fare {
		t.Errorf("expected Fare %f, got %f", ride.Fare, result.Fare)
	}
}

func TestMockManager_Delete_Success(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	// Set some data first
	_ = manager.Set(ctx, "user:1", TestUser{ID: "1"}, time.Hour)
	_ = manager.Set(ctx, "user:2", TestUser{ID: "2"}, time.Hour)

	// Delete one key
	err := manager.Delete(ctx, "user:1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify deletion
	var result TestUser
	err = manager.Get(ctx, "user:1", &result)
	if err == nil {
		t.Error("expected cache miss after deletion")
	}

	// Verify other key still exists
	err = manager.Get(ctx, "user:2", &result)
	if err != nil {
		t.Error("expected user:2 to still exist")
	}
}

func TestMockManager_Delete_MultipleKeys(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	// Set data
	_ = manager.Set(ctx, "key1", "value1", time.Hour)
	_ = manager.Set(ctx, "key2", "value2", time.Hour)
	_ = manager.Set(ctx, "key3", "value3", time.Hour)

	// Delete multiple
	err := manager.Delete(ctx, "key1", "key2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify deletions
	if _, ok := mock.data["key1"]; ok {
		t.Error("key1 should be deleted")
	}
	if _, ok := mock.data["key2"]; ok {
		t.Error("key2 should be deleted")
	}
	if _, ok := mock.data["key3"]; !ok {
		t.Error("key3 should still exist")
	}
}

func TestMockManager_Delete_NonexistentKey(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	// Delete nonexistent key should not error
	err := manager.Delete(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("expected no error for nonexistent key, got %v", err)
	}
}

func TestMockManager_Delete_Error(t *testing.T) {
	mock := NewMockRedisClient()
	mock.delError = errors.New("delete error")
	manager := NewMockManager(mock)
	ctx := context.Background()

	err := manager.Delete(ctx, "any")
	if err == nil {
		t.Fatal("expected error")
	}
}

// ============== Cache Keys Tests ==============

func TestCacheKeys_User(t *testing.T) {
	key := Keys.User("user-123")
	expected := "user:user-123"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestCacheKeys_Driver(t *testing.T) {
	key := Keys.Driver("driver-456")
	expected := "driver:driver-456"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestCacheKeys_Ride(t *testing.T) {
	key := Keys.Ride("ride-789")
	expected := "ride:ride-789"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestCacheKeys_RideHistory(t *testing.T) {
	tests := []struct {
		userID   string
		offset   int
		expected string
	}{
		{"user-1", 0, "ride_history:user-1:offset:0"},
		{"user-2", 20, "ride_history:user-2:offset:20"},
		{"user-3", 100, "ride_history:user-3:offset:100"},
	}

	for _, tc := range tests {
		key := Keys.RideHistory(tc.userID, tc.offset)
		if key != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, key)
		}
	}
}

func TestCacheKeys_DriverLocation(t *testing.T) {
	key := Keys.DriverLocation("driver-123")
	expected := "driver:location:driver-123"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestCacheKeys_NearbyDrivers(t *testing.T) {
	tests := []struct {
		latitude      float64
		longitude      float64
		radius   float64
		expected string
	}{
		{37.7749, -122.4194, 5.0, "nearby_drivers:37.774900:-122.419400:5.0"},
		{40.7128, -74.0060, 10.0, "nearby_drivers:40.712800:-74.006000:10.0"},
		{0.0, 0.0, 1.0, "nearby_drivers:0.000000:0.000000:1.0"},
	}

	for _, tc := range tests {
		key := Keys.NearbyDrivers(tc.latitude, tc.longitude, tc.radius)
		if key != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, key)
		}
	}
}

func TestCacheKeys_Wallet(t *testing.T) {
	key := Keys.Wallet("user-wallet-123")
	expected := "wallet:user-wallet-123"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestCacheKeys_PromoCode(t *testing.T) {
	key := Keys.PromoCode("SAVE20")
	expected := "promo:SAVE20"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestCacheKeys_RideType(t *testing.T) {
	key := Keys.RideType("economy")
	expected := "ride_type:economy"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestCacheKeys_DriverStats(t *testing.T) {
	key := Keys.DriverStats("driver-stats-123")
	expected := "driver:stats:driver-stats-123"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestCacheKeys_Analytics(t *testing.T) {
	tests := []struct {
		metric   string
		params   []string
		expected string
	}{
		{"daily_rides", nil, "analytics:daily_rides"},
		{"revenue", []string{"2024"}, "analytics:revenue:2024"},
		{"trips", []string{"region", "north"}, "analytics:trips:region:north"},
		{"conversion", []string{"source", "mobile", "v2"}, "analytics:conversion:source:mobile:v2"},
	}

	for _, tc := range tests {
		key := Keys.Analytics(tc.metric, tc.params...)
		if key != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, key)
		}
	}
}

// ============== Cache TTL Tests ==============

func TestCacheTTL_Short(t *testing.T) {
	expected := 5 * time.Minute
	if TTL.Short() != expected {
		t.Errorf("expected %v, got %v", expected, TTL.Short())
	}
}

func TestCacheTTL_Medium(t *testing.T) {
	expected := 15 * time.Minute
	if TTL.Medium() != expected {
		t.Errorf("expected %v, got %v", expected, TTL.Medium())
	}
}

func TestCacheTTL_Long(t *testing.T) {
	expected := 1 * time.Hour
	if TTL.Long() != expected {
		t.Errorf("expected %v, got %v", expected, TTL.Long())
	}
}

func TestCacheTTL_VeryLong(t *testing.T) {
	expected := 24 * time.Hour
	if TTL.VeryLong() != expected {
		t.Errorf("expected %v, got %v", expected, TTL.VeryLong())
	}
}

func TestCacheTTL_Permanent(t *testing.T) {
	expected := 7 * 24 * time.Hour
	if TTL.Permanent() != expected {
		t.Errorf("expected %v, got %v", expected, TTL.Permanent())
	}
}

// ============== Expiration Tests ==============

func TestCache_TTLExpiration(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	// Set with very short TTL
	_ = manager.Set(ctx, "short-lived", "value", 1*time.Millisecond)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	var result string
	err := manager.Get(ctx, "short-lived", &result)
	if err == nil {
		t.Error("expected cache miss after TTL expiration")
	}
}

func TestCache_NoExpiration(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	// Set without expiration
	_ = manager.Set(ctx, "permanent", "value", 0)

	// Should still be accessible
	var result string
	err := manager.Get(ctx, "permanent", &result)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != "value" {
		t.Errorf("expected 'value', got '%s'", result)
	}
}

// ============== Concurrent Access Tests ==============

func TestCache_ConcurrentAccess(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	var wg sync.WaitGroup
	errCh := make(chan error, 100)

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			user := TestUser{ID: string(rune(idx)), Name: "User"}
			if err := manager.Set(ctx, Keys.User(string(rune(idx))), user, time.Hour); err != nil {
				errCh <- err
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var result TestUser
			// Ignore cache miss errors, we just care about race conditions
			_ = manager.Get(ctx, Keys.User(string(rune(idx))), &result)
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent operation error: %v", err)
	}
}

func TestCache_ConcurrentDeleteAndGet(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 10; i++ {
		_ = manager.Set(ctx, Keys.User(string(rune(i))), TestUser{ID: string(rune(i))}, time.Hour)
	}

	var wg sync.WaitGroup

	// Concurrent deletes and gets
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			_ = manager.Delete(ctx, Keys.User(string(rune(idx))))
		}(i)
		go func(idx int) {
			defer wg.Done()
			var result TestUser
			_ = manager.Get(ctx, Keys.User(string(rune(idx))), &result)
		}(i)
	}

	wg.Wait()
}

// ============== Context Tests ==============

func TestCache_ContextCancellation(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Operations should still work with mock (real Redis would fail)
	var result TestUser
	_ = manager.Get(ctx, "key", &result)
	_ = manager.Set(ctx, "key", TestUser{}, time.Hour)
}

// ============== Data Type Tests ==============

func TestCache_StringValue(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	_ = manager.Set(ctx, "string-key", "simple string", time.Hour)

	var result string
	err := manager.Get(ctx, "string-key", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "simple string" {
		t.Errorf("expected 'simple string', got '%s'", result)
	}
}

func TestCache_IntValue(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	_ = manager.Set(ctx, "int-key", 42, time.Hour)

	var result int
	err := manager.Get(ctx, "int-key", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestCache_FloatValue(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	_ = manager.Set(ctx, "float-key", 3.14159, time.Hour)

	var result float64
	err := manager.Get(ctx, "float-key", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != 3.14159 {
		t.Errorf("expected 3.14159, got %f", result)
	}
}

func TestCache_BoolValue(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	_ = manager.Set(ctx, "bool-key", true, time.Hour)

	var result bool
	err := manager.Get(ctx, "bool-key", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestCache_SliceValue(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	original := []string{"a", "b", "c"}
	_ = manager.Set(ctx, "slice-key", original, time.Hour)

	var result []string
	err := manager.Get(ctx, "slice-key", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != len(original) {
		t.Errorf("expected length %d, got %d", len(original), len(result))
	}
	for i, v := range original {
		if result[i] != v {
			t.Errorf("expected result[%d] = %s, got %s", i, v, result[i])
		}
	}
}

func TestCache_MapValue(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	original := map[string]int{"one": 1, "two": 2, "three": 3}
	_ = manager.Set(ctx, "map-key", original, time.Hour)

	var result map[string]int
	err := manager.Get(ctx, "map-key", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != len(original) {
		t.Errorf("expected length %d, got %d", len(original), len(result))
	}
	for k, v := range original {
		if result[k] != v {
			t.Errorf("expected result[%s] = %d, got %d", k, v, result[k])
		}
	}
}

func TestCache_NilValue(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	// nil is marshaled as "null"
	_ = manager.Set(ctx, "nil-key", nil, time.Hour)

	var result *TestUser
	err := manager.Get(ctx, "nil-key", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestCache_EmptyStruct(t *testing.T) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	empty := TestUser{}
	_ = manager.Set(ctx, "empty-key", empty, time.Hour)

	var result TestUser
	err := manager.Get(ctx, "empty-key", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ID != "" || result.Name != "" || result.Email != "" {
		t.Error("expected empty struct")
	}
}

// ============== Pattern Matching Tests ==============

func TestPatternMatching_ExactMatch(t *testing.T) {
	if !matchPattern("user:123", "user:123") {
		t.Error("exact match should return true")
	}
}

func TestPatternMatching_WildcardMatch(t *testing.T) {
	if !matchPattern("user:*", "user:123") {
		t.Error("wildcard should match")
	}
	if !matchPattern("user:*", "user:") {
		t.Error("wildcard should match empty suffix")
	}
}

func TestPatternMatching_NoMatch(t *testing.T) {
	if matchPattern("user:*", "driver:123") {
		t.Error("different prefix should not match")
	}
}

func TestPatternMatching_EmptyPattern(t *testing.T) {
	if !matchPattern("", "anything") {
		t.Error("empty pattern should match anything")
	}
}

// ============== Table-Driven Tests ==============

func TestCacheOperations_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*MockManager, context.Context)
		key      string
		expected interface{}
		wantErr  bool
	}{
		{
			name: "get existing string",
			setup: func(m *MockManager, ctx context.Context) {
				_ = m.Set(ctx, "test-key", "test-value", time.Hour)
			},
			key:      "test-key",
			expected: "test-value",
			wantErr:  false,
		},
		{
			name:     "get nonexistent key",
			setup:    func(m *MockManager, ctx context.Context) {},
			key:      "nonexistent",
			expected: "",
			wantErr:  true,
		},
		{
			name: "get struct",
			setup: func(m *MockManager, ctx context.Context) {
				_ = m.Set(ctx, "user-key", TestUser{ID: "123", Name: "Test"}, time.Hour)
			},
			key:      "user-key",
			expected: TestUser{ID: "123", Name: "Test"},
			wantErr:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRedisClient()
			manager := NewMockManager(mock)
			ctx := context.Background()

			tc.setup(manager, ctx)

			switch expected := tc.expected.(type) {
			case string:
				var result string
				err := manager.Get(ctx, tc.key, &result)
				if (err != nil) != tc.wantErr {
					t.Errorf("wantErr = %v, got err = %v", tc.wantErr, err)
				}
				if !tc.wantErr && result != expected {
					t.Errorf("expected %v, got %v", expected, result)
				}
			case TestUser:
				var result TestUser
				err := manager.Get(ctx, tc.key, &result)
				if (err != nil) != tc.wantErr {
					t.Errorf("wantErr = %v, got err = %v", tc.wantErr, err)
				}
				if !tc.wantErr && (result.ID != expected.ID || result.Name != expected.Name) {
					t.Errorf("expected %v, got %v", expected, result)
				}
			}
		})
	}
}

// ============== Benchmark Tests ==============

func BenchmarkCacheGet(b *testing.B) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	_ = manager.Set(ctx, "bench-key", TestUser{ID: "1", Name: "Bench"}, time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result TestUser
		_ = manager.Get(ctx, "bench-key", &result)
	}
}

func BenchmarkCacheSet(b *testing.B) {
	mock := NewMockRedisClient()
	manager := NewMockManager(mock)
	ctx := context.Background()

	user := TestUser{ID: "1", Name: "Bench"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Set(ctx, "bench-key", user, time.Hour)
	}
}

func BenchmarkCacheKeys_User(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Keys.User("user-123")
	}
}

func BenchmarkCacheKeys_NearbyDrivers(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Keys.NearbyDrivers(37.7749, -122.4194, 5.0)
	}
}

func BenchmarkCacheKeys_Analytics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Keys.Analytics("daily_rides", "region", "north", "2024")
	}
}
