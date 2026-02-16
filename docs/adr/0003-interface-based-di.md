# ADR-0003: Interface-Based Dependency Injection

## Status

Accepted

## Context

The ride-hailing platform requires comprehensive test coverage to ensure reliability. Key challenges:

1. **Database isolation**: Unit tests must not depend on PostgreSQL or Redis being available.
2. **External service mocking**: Tests should mock Stripe, Twilio, Firebase, and other external APIs.
3. **Fast test execution**: Tests must run quickly to support rapid development.
4. **No runtime overhead**: Production code should not incur performance penalties from DI frameworks.

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Wire (Google)** | Compile-time safety, generates code | Complex setup, overkill for this scale |
| **Uber Fx** | Feature-rich, lifecycle management | Runtime reflection, startup overhead |
| **dig (Uber)** | Lightweight runtime DI | Still has reflection overhead |
| **Interface-based DI** | Zero overhead, simple, native Go | Manual wiring, more interfaces to maintain |

## Decision

Use **interface-based dependency injection** with hand-written mocks. Each domain defines a `RepositoryInterface` that abstracts data access.

### Pattern Implementation

#### 1. Define Repository Interface

```go
// From internal/auth/repository_interface.go
type RepositoryInterface interface {
    CreateUser(ctx context.Context, user *models.User) error
    CreateDriver(ctx context.Context, driver *models.Driver) error
    GetUserByEmail(ctx context.Context, email string) (*models.User, error)
    GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
    UpdateUser(ctx context.Context, user *models.User) error
}
```

#### 2. Service Accepts Interface

```go
// From internal/auth/service.go
type Service struct {
    repo       RepositoryInterface
    keyManager jwtkeys.KeyManager
    expiration int
}

func NewService(repo RepositoryInterface, km jwtkeys.KeyManager, exp int) *Service {
    return &Service{repo: repo, keyManager: km, expiration: exp}
}
```

#### 3. Production Repository Implements Interface

```go
// From internal/auth/repository.go
type Repository struct {
    db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
    return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, user *models.User) error {
    // Actual PostgreSQL implementation
}
```

#### 4. Mock Repository for Testing

```go
// From test/mocks/repository.go
type MockAuthRepository struct {
    mock.Mock
}

func (m *MockAuthRepository) CreateUser(ctx context.Context, user *models.User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}

func (m *MockAuthRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
    args := m.Called(ctx, email)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*models.User), args.Error(1)
}
```

### Interface Catalog

The platform defines 30+ repository interfaces across domains:

| Domain | Interface File | Methods |
|--------|---------------|---------|
| auth | `internal/auth/repository_interface.go` | 5 |
| favorites | `internal/favorites/interfaces.go` | 5 |
| rides | `internal/rides/interfaces.go` | 10+ |
| payments | `internal/payments/interfaces.go` | 8 |
| mleta | `internal/mleta/service.go` | ETARepository |

### External Service Interfaces

External APIs are also abstracted:

```go
// From pkg/redis/interface.go
type ClientInterface interface {
    GetString(ctx context.Context, key string) (string, error)
    SetWithExpiration(ctx context.Context, key string, value interface{}, exp time.Duration) error
    Delete(ctx context.Context, keys ...string) error
    GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error
    GeoRadius(ctx context.Context, key string, longitude, latitude, radius float64, count int) ([]string, error)
}
```

## Consequences

### Positive

- **Zero runtime overhead**: No reflection or code generation at runtime.
- **IDE support**: Full autocomplete and refactoring support.
- **Explicit dependencies**: Constructor signatures document all dependencies.
- **Easy mocking**: `testify/mock` integrates naturally with interfaces.
- **Compile-time safety**: Interface mismatches caught at build time.

### Negative

- **Manual wiring**: `main.go` explicitly constructs dependency graph.
- **Interface maintenance**: Adding repository methods requires updating interface.
- **Mock boilerplate**: Each interface needs corresponding mock implementation.

### Test Example

```go
// From internal/auth/service_test.go
func TestService_Register_Success(t *testing.T) {
    mockRepo := new(mocks.MockAuthRepository)
    service := newTestService(t, mockRepo)
    ctx := context.Background()
    req := helpers.CreateTestRegisterRequest()

    // Setup expectations
    mockRepo.On("GetUserByEmail", mock.Anything, req.Email).Return(nil, errors.New("not found"))
    mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

    // Execute
    user, err := service.Register(ctx, req)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, user)
    mockRepo.AssertExpectations(t)
}
```

## References

- [internal/auth/repository_interface.go](/internal/auth/repository_interface.go) - Interface definition
- [internal/auth/service.go](/internal/auth/service.go) - Service using interface
- [test/mocks/repository.go](/test/mocks/repository.go) - Mock implementations
- [internal/auth/service_test.go](/internal/auth/service_test.go) - Test with mocks
- [pkg/redis/interface.go](/pkg/redis/interface.go) - Redis client interface
