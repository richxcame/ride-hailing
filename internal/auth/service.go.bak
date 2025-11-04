package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

// Service handles authentication business logic
type Service struct {
	repo      *Repository
	jwtSecret string
	jwtExpiry int
}

// NewService creates a new auth service
func NewService(repo *Repository, jwtSecret string, jwtExpiry int) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		jwtExpiry: jwtExpiry,
	}
}

// Register registers a new user
func (s *Service) Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {
	// Check if user already exists
	existingUser, _ := s.repo.GetUserByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, common.NewConflictError("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
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
		return nil, common.NewInternalServerError("failed to create user")
	}

	// Clear password hash from response
	user.PasswordHash = ""

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
	// Get user by email
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
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
	token, err := s.generateToken(user)
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
		return nil, common.NewNotFoundError("user not found")
	}

	// Clear password hash
	user.PasswordHash = ""

	return user, nil
}

// UpdateProfile updates user profile
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, updates *models.User) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, common.NewNotFoundError("user not found")
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
func (s *Service) generateToken(user *models.User) (string, error) {
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
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
