# Testing Infrastructure

This directory contains the testing infrastructure for the ride-hailing backend.

## Structure

```
test/
├── helpers/        # Test helper functions and assertions
│   ├── fixtures.go      # Test data fixtures
│   └── assertions.go    # Custom assertions
├── mocks/          # Mock implementations for testing
│   └── repository.go    # Mock repository implementations
└── README.md       # This file
```

## Running Tests

### Unit Tests

Run all unit tests:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

Run tests for a specific package:

```bash
go test -v ./internal/auth/...
```

### With Test Dependencies

If tests require database or Redis, start test services first:

```bash
# Start test dependencies
docker-compose -f docker-compose.test.yml up -d

# Run tests
go test ./...

# Stop test dependencies
docker-compose -f docker-compose.test.yml down
```

### Integration Tests

Integration tests require running services. Make sure to:

1. Start test dependencies:

    ```bash
    docker-compose -f docker-compose.test.yml up -d
    ```

2. Run migrations on test database:

    ```bash
    DATABASE_URL="postgres://testuser:testpassword@localhost:5433/ride_hailing_test?sslmode=disable" \
    migrate -path migrations -database $DATABASE_URL up
    ```

3. Run integration tests:
    ```bash
    go test -tags=integration ./test/integration/...
    ```

## Test Coverage

Generate detailed coverage report:

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

### Coverage Goals

-   **Target:** 80% overall coverage
-   **Critical paths:** 90% coverage (auth, payments, rides)
-   **Utility functions:** 70% coverage

Current coverage by package:

-   `internal/auth`: 37.2% ⚠️ (Target: 80%)
-   `internal/rides`: 1.7% ⚠️ (Target: 80%)
-   `internal/payments`: 0.0% ⚠️ (Target: 80%)

## Writing Tests

### Test Structure

Follow the Arrange-Act-Assert (AAA) pattern:

```go
func TestService_MethodName_Scenario(t *testing.T) {
    // Arrange - Setup test data and mocks
    mockRepo := new(mocks.MockAuthRepository)
    service := NewService(mockRepo, "test-secret", 24)

    // Act - Execute the function under test
    result, err := service.SomeMethod(ctx, input)

    // Assert - Verify the results
    assert.NoError(t, err)
    assert.NotNil(t, result)
    mockRepo.AssertExpectations(t)
}
```

### Using Test Helpers

#### Fixtures

Create test data using fixtures:

```go
import "github.com/richxcame/ride-hailing/test/helpers"

// Create a test user
user := helpers.CreateTestUser()

// Create a test registration request
req := helpers.CreateTestRegisterRequest()
```

#### Custom Assertions

Use custom assertions for common checks:

```go
import "github.com/richxcame/ride-hailing/test/helpers"

// Assert user fields match (excluding sensitive data)
helpers.AssertUserEqual(t, expectedUser, actualUser)

// Assert password is not in response
helpers.AssertPasswordNotInResponse(t, user)

// Assert JWT token is valid
helpers.AssertValidJWT(t, token)
```

### Mocking

Use testify/mock for creating mocks:

```go
import "github.com/richxcame/ride-hailing/test/mocks"

// Create a mock repository
mockRepo := new(mocks.MockAuthRepository)

// Set expectations
mockRepo.On("GetUserByID", ctx, userID).Return(testUser, nil)

// Use in tests
service := NewService(mockRepo, "secret", 24)

// Verify expectations were met
mockRepo.AssertExpectations(t)
```

## Test Categories

### Unit Tests

-   Test individual functions and methods
-   Use mocks for dependencies
-   Fast execution
-   Location: `internal/*/service_test.go`

### Integration Tests

-   Test multiple components together
-   Use real database/redis (test containers)
-   Slower execution
-   Location: `test/integration/*_test.go`

### End-to-End Tests

-   Test complete user flows
-   Use real HTTP requests
-   Slowest execution
-   Location: `test/e2e/*_test.go`

## CI/CD Integration

Tests run automatically on:

-   Push to `main` or `develop` branches
-   Pull requests to `main` or `develop`

See `.github/workflows/test.yml` for CI configuration.

### CI Test Workflow

1. Checkout code
2. Setup Go environment
3. Install dependencies
4. Run linters (golangci-lint)
5. Run tests with coverage
6. Upload coverage to Codecov
7. Check coverage thresholds
8. Archive coverage report

## Best Practices

1. **Test naming:** Use descriptive names: `TestService_Method_Scenario`
2. **Table-driven tests:** Use for testing multiple scenarios
3. **Mock external dependencies:** Database, Redis, external APIs
4. **Use test fixtures:** Reuse common test data
5. **Test errors:** Always test error cases
6. **Test edge cases:** Boundary conditions, nil values, empty strings
7. **Keep tests fast:** Unit tests should run in milliseconds
8. **Clean up:** Use `defer` for cleanup operations
9. **Independent tests:** Tests should not depend on each other
10. **Coverage:** Aim for 80%+ coverage on business logic

## Common Patterns

### Table-Driven Tests

```go
func TestCalculateFare(t *testing.T) {
    tests := []struct {
        name     string
        distance float64
        duration int
        expected float64
    }{
        {"short ride", 2.0, 10, 5.0},
        {"medium ride", 10.0, 20, 15.0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := calculateFare(tt.distance, tt.duration)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Testing HTTP Handlers

```go
func TestHandler(t *testing.T) {
    // Create a request
    req := httptest.NewRequest("GET", "/api/users/123", nil)
    w := httptest.NewRecorder()

    // Call handler
    handler.ServeHTTP(w, req)

    // Assert response
    assert.Equal(t, http.StatusOK, w.Code)
}
```

## Troubleshooting

### Tests failing locally but passing in CI

-   Check environment variables
-   Verify database migrations are up to date
-   Check for race conditions (run with `-race` flag)

### Slow tests

-   Use mocks instead of real dependencies
-   Avoid unnecessary sleeps
-   Use test containers for integration tests only

### Flaky tests

-   Tests that randomly fail/pass
-   Usually caused by timing issues or shared state
-   Use proper synchronization
-   Ensure test isolation

## Resources

-   [Go Testing Documentation](https://golang.org/pkg/testing/)
-   [Testify Documentation](https://github.com/stretchr/testify)
-   [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
