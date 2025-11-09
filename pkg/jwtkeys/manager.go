package jwtkeys

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	// ErrKeyNotFound is returned when a kid cannot be resolved to a signing key.
	ErrKeyNotFound = errors.New("jwtkeys: signing key not found")
	errNoActiveKey = errors.New("jwtkeys: no active signing key available")
	errReadOnly    = errors.New("jwtkeys: manager is read-only")
)

// KeyProvider resolves signing keys for JWT verification.
type KeyProvider interface {
	ResolveKey(kid string) ([]byte, error)
	LegacyKey() []byte
}

// Config drives the behaviour of the Manager.
type Config struct {
	KeyFilePath      string
	RotationInterval time.Duration
	GracePeriod      time.Duration
	LegacySecret     string
	ReadOnly         bool
	Store            Store
}

// Manager coordinates signing key rotation and lookup.
type Manager struct {
	mu               sync.RWMutex
	store            Store
	keys             map[string]*SigningKey
	activeID         string
	rotationInterval time.Duration
	gracePeriod      time.Duration
	legacySecret     []byte
	readOnly         bool
}

// SigningKey represents a versioned JWT signing key.
type SigningKey struct {
	ID        string    `json:"id"`
	Secret    string    `json:"secret"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
}

// SecretBytes decodes the base64-encoded secret.
func (k *SigningKey) SecretBytes() ([]byte, error) {
	return base64.StdEncoding.DecodeString(k.Secret)
}

// Clone returns a copy of the signing key to avoid data races.
func (k *SigningKey) Clone() *SigningKey {
	if k == nil {
		return nil
	}
	copy := *k
	return &copy
}

// NewManager creates a new signing key manager backed by the configured keystore.
func NewManager(ctx context.Context, cfg Config) (*Manager, error) {
	if cfg.RotationInterval <= 0 {
		cfg.RotationInterval = 30 * 24 * time.Hour
	}
	if cfg.GracePeriod <= 0 {
		cfg.GracePeriod = 30 * 24 * time.Hour
	}

	var store Store
	switch {
	case cfg.Store != nil:
		store = cfg.Store
	case cfg.KeyFilePath == "":
		store = newMemoryStore()
	default:
		store = &fileStore{path: cfg.KeyFilePath}
	}

	manager := &Manager{
		store:            store,
		keys:             make(map[string]*SigningKey),
		rotationInterval: cfg.RotationInterval,
		gracePeriod:      cfg.GracePeriod,
		legacySecret:     []byte(cfg.LegacySecret),
		readOnly:         cfg.ReadOnly,
	}

	if err := manager.reloadFromStore(ctx); err != nil {
		return nil, err
	}

	if manager.activeID == "" && !manager.readOnly {
		if err := manager.seedInitialKey(ctx); err != nil {
			return nil, err
		}
	}

	return manager, nil
}

// EnsureRotation rotates the active signing key when the rotation interval elapses.
func (m *Manager) EnsureRotation(ctx context.Context) error {
	if m.readOnly {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ensureRotationLocked(ctx, time.Now())
}

// CurrentSigningKey returns the key currently used for signing operations.
func (m *Manager) CurrentSigningKey() (*SigningKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, ok := m.keys[m.activeID]
	if !ok {
		return nil, errNoActiveKey
	}

	return key.Clone(), nil
}

// ResolveKey implements KeyProvider for JWT verification.
func (m *Manager) ResolveKey(kid string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if kid == "" {
		return nil, ErrKeyNotFound
	}

	key, ok := m.keys[kid]
	if !ok || key.Revoked || time.Now().After(key.ExpiresAt) {
		return nil, ErrKeyNotFound
	}

	return key.SecretBytes()
}

// LegacyKey returns the static secret used prior to key versioning.
func (m *Manager) LegacyKey() []byte {
	return m.legacySecret
}

// StartAutoRotation periodically checks if rotation is required.
func (m *Manager) StartAutoRotation(ctx context.Context) {
	if m.readOnly {
		return
	}

	interval := m.rotationInterval / 4
	if interval <= 0 {
		interval = time.Hour
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_ = m.EnsureRotation(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// StartAutoRefresh reloads keys from the underlying store at the supplied interval.
func (m *Manager) StartAutoRefresh(ctx context.Context, interval time.Duration) {
	go func() {
		if interval <= 0 {
			interval = 5 * time.Minute
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_ = m.reloadFromStore(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// reloadFromStore refreshes the in-memory key cache.
func (m *Manager) reloadFromStore(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	keys, err := m.store.Load(ctx)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	m.keys = make(map[string]*SigningKey, len(keys))
	var latestKey *SigningKey
	for i := range keys {
		key := keys[i]
		m.keys[key.ID] = key.Clone()
		if key.Revoked {
			continue
		}
		if latestKey == nil || key.CreatedAt.After(latestKey.CreatedAt) {
			latestKey = &key
		}
	}

	if latestKey != nil {
		m.activeID = latestKey.ID
	}

	return nil
}

func (m *Manager) seedInitialKey(ctx context.Context) error {
	if m.readOnly {
		return nil
	}

	secret := m.legacySecret
	if len(secret) == 0 {
		randomSecret, err := generateSecret()
		if err != nil {
			return err
		}
		secret = randomSecret
	}

	now := time.Now()
	key := &SigningKey{
		ID:        generateKeyID(now),
		Secret:    base64.StdEncoding.EncodeToString(secret),
		CreatedAt: now,
		ExpiresAt: now.Add(m.rotationInterval + m.gracePeriod),
	}
	m.keys[key.ID] = key
	m.activeID = key.ID
	return m.persistLocked(ctx)
}

func (m *Manager) ensureRotationLocked(ctx context.Context, now time.Time) error {
	key := m.keys[m.activeID]
	if key == nil || now.Sub(key.CreatedAt) >= m.rotationInterval {
		if err := m.rotateLocked(ctx, now); err != nil {
			return err
		}
	}
	return m.pruneExpiredLocked(ctx, now)
}

func (m *Manager) rotateLocked(ctx context.Context, now time.Time) error {
	if m.readOnly {
		return errReadOnly
	}

	rawSecret, err := generateSecret()
	if err != nil {
		return err
	}

	key := &SigningKey{
		ID:        generateKeyID(now),
		Secret:    base64.StdEncoding.EncodeToString(rawSecret),
		CreatedAt: now,
		ExpiresAt: now.Add(m.rotationInterval + m.gracePeriod),
	}

	m.keys[key.ID] = key
	m.activeID = key.ID

	return m.persistLocked(ctx)
}

func (m *Manager) pruneExpiredLocked(ctx context.Context, now time.Time) error {
	if m.readOnly {
		return nil
	}

	changed := false
	for id, key := range m.keys {
		if key.Revoked || now.After(key.ExpiresAt) {
			delete(m.keys, id)
			if id == m.activeID {
				m.activeID = ""
			}
			changed = true
		}
	}

	if !changed {
		return nil
	}

	if m.activeID == "" {
		for id, key := range m.keys {
			if !key.Revoked {
				m.activeID = id
				break
			}
		}
	}

	return m.persistLocked(ctx)
}

func (m *Manager) persistLocked(ctx context.Context) error {
	if m.readOnly {
		return nil
	}

	keys := make([]SigningKey, 0, len(m.keys))
	for _, key := range m.keys {
		keys = append(keys, *key.Clone())
	}
	return m.store.Save(ctx, keys)
}

func generateKeyID(now time.Time) string {
	return fmt.Sprintf("kid_%d", now.UnixNano())
}

func generateSecret() ([]byte, error) {
	buf := make([]byte, 48) // 384 bits
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// Store abstracts persistence for signing keys.
type Store interface {
	Load(ctx context.Context) ([]SigningKey, error)
	Save(ctx context.Context, keys []SigningKey) error
}

type fileStore struct {
	path string
}

func (s *fileStore) Load(_ context.Context) ([]SigningKey, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var keys []SigningKey
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *fileStore) Save(_ context.Context, keys []SigningKey) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

type memoryStore struct {
	mu   sync.RWMutex
	keys []SigningKey
}

func newMemoryStore() *memoryStore {
	return &memoryStore{}
}

func (s *memoryStore) Load(_ context.Context) ([]SigningKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return append([]SigningKey(nil), s.keys...), nil
}

func (s *memoryStore) Save(_ context.Context, keys []SigningKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys = append([]SigningKey(nil), keys...)
	return nil
}

// StaticProvider is a helper for legacy code paths that still rely on a single secret.
type StaticProvider struct {
	secret []byte
}

// NewStaticProvider creates a KeyProvider backed by a single secret.
func NewStaticProvider(secret string) KeyProvider {
	return &StaticProvider{secret: []byte(secret)}
}

// ResolveKey implements KeyProvider by ignoring kid values.
func (p *StaticProvider) ResolveKey(string) ([]byte, error) {
	if len(p.secret) == 0 {
		return nil, ErrKeyNotFound
	}
	return p.secret, nil
}

// LegacyKey returns the static secret.
func (p *StaticProvider) LegacyKey() []byte {
	return p.secret
}
