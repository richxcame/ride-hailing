package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/bcrypt"
)

// Service handles authentication business logic
type Service struct {
	repo       RepositoryInterface
	keyManager *jwtkeys.Manager
	jwtExpiry  int
}

// NewService creates a new auth service
func NewService(repo RepositoryInterface, keyManager *jwtkeys.Manager, jwtExpiry int) *Service {
	return &Service{
		repo:       repo,
		keyManager: keyManager,
		jwtExpiry:  jwtExpiry,
	}
}

// Register registers a new user
func (s *Service) Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {
	ctx, span := tracing.StartSpan(ctx, "auth-service", "Register")
	defer span.End()

	tracing.AddSpanAttributes(ctx,
		attribute.String("user.email", req.Email),
		attribute.String("user.role", string(req.Role)),
	)

	// Check if user already exists
	existingUser, _ := s.repo.GetUserByEmail(ctx, req.Email)
	if existingUser != nil {
		err := fmt.Errorf("user already exists")
		tracing.RecordError(ctx, err)
		return nil, common.NewConflictError("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, common.NewInternalServerError("failed to hash password")
	}

	// Create user
	user := &models.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PhoneNumber:  req.PhoneNumber,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         req.Role,
		IsActive:     true,
		IsVerified:   false,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		tracing.RecordError(ctx, err)
		return nil, common.NewInternalServerError("failed to create user")
	}

	// Auto-create driver record for driver role users
	// This ensures drivers are immediately visible in admin panel
	if user.Role == models.RoleDriver {
		driver := &models.Driver{
			ID:             uuid.New(),
			UserID:         user.ID,
			LicenseNumber:  fmt.Sprintf("PENDING-%s", user.ID.String()[:8]),
			VehicleModel:   "Not Specified",
			VehiclePlate:   fmt.Sprintf("PENDING-%s", user.ID.String()[:6]),
			VehicleColor:   "Not Specified",
			VehicleYear:    2020,
			IsAvailable:    false,
			IsOnline:       false,
			ApprovalStatus: "pending", // Requires admin approval before going online
			Rating:         5.0,        // Default new driver rating
			TotalRides:     0,
		}

		if err := s.repo.CreateDriver(ctx, driver); err != nil {
			// Log error but don't fail registration - driver can update profile later
			tracing.RecordError(ctx, err)
			tracing.AddSpanEvent(ctx, "driver_profile_creation_failed",
				attribute.String("user_id", user.ID.String()),
				attribute.String("error", err.Error()),
			)
		} else {
			tracing.AddSpanEvent(ctx, "driver_profile_created",
				attribute.String("user_id", user.ID.String()),
				attribute.String("driver_id", driver.ID.String()),
			)
		}
	}

	// Clear password hash from response
	user.PasswordHash = ""

	tracing.AddSpanAttributes(ctx, tracing.UserIDKey.String(user.ID.String()))
	tracing.AddSpanEvent(ctx, "user_registered",
		attribute.String("user_id", user.ID.String()),
		attribute.String("role", string(user.Role)),
	)

	return user, nil
}

// RegisterDriver registers a new driver with vehicle information
func (s *Service) RegisterDriver(ctx context.Context, req *models.RegisterRequest, driver *models.Driver) (*models.User, error) {
	// Register user first
	user, err := s.Register(ctx, req)
	if err != nil {
		return nil, err
	}

	// Create driver record
	driver.ID = uuid.New()
	driver.UserID = user.ID
	driver.IsAvailable = false
	driver.IsOnline = false
	driver.Rating = 0.0
	driver.TotalRides = 0

	if err := s.repo.CreateDriver(ctx, driver); err != nil {
		return nil, common.NewInternalServerError("failed to create driver profile")
	}

	return user, nil
}

// Login authenticates a user and returns a JWT token
func (s *Service) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "auth-service", "Login")
	defer span.End()

	tracing.AddSpanAttributes(ctx, attribute.String("user.email", req.Email))

	// Get user by email
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, common.NewUnauthorizedError("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, common.NewUnauthorizedError("account is inactive")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, common.NewUnauthorizedError("invalid credentials")
	}

	// Generate JWT token
	token, err := s.generateToken(ctx, user)
	if err != nil {
		return nil, common.NewInternalServerError("failed to generate token")
	}

	// Clear password hash from response
	user.PasswordHash = ""

	return &models.LoginResponse{
		User:  user,
		Token: token,
	}, nil
}

// GetProfile retrieves user profile
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, common.NewNotFoundError("user not found", nil)
	}

	// Clear password hash
	user.PasswordHash = ""

	return user, nil
}

// UpdateProfile updates user profile
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, updates *models.User) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, common.NewNotFoundError("user not found", nil)
	}

	// Update fields
	if updates.FirstName != "" {
		user.FirstName = updates.FirstName
	}
	if updates.LastName != "" {
		user.LastName = updates.LastName
	}
	if updates.PhoneNumber != "" {
		user.PhoneNumber = updates.PhoneNumber
	}
	if updates.ProfileImage != nil {
		user.ProfileImage = updates.ProfileImage
	}

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, common.NewInternalServerError("failed to update profile")
	}

	// Clear password hash
	user.PasswordHash = ""

	return user, nil
}

// generateToken generates a JWT token for a user
func (s *Service) generateToken(ctx context.Context, user *models.User) (string, error) {
	if s.keyManager == nil {
		return "", fmt.Errorf("jwt key manager is not configured")
	}

	if err := s.keyManager.EnsureRotation(ctx); err != nil {
		return "", fmt.Errorf("failed to rotate signing key: %w", err)
	}

	key, err := s.keyManager.CurrentSigningKey()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve signing key: %w", err)
	}

	secretBytes, err := key.SecretBytes()
	if err != nil {
		return "", fmt.Errorf("invalid signing key: %w", err)
	}

	claims := &middleware.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(s.jwtExpiry))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = key.ID
	tokenString, err := token.SignedString(secretBytes)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
