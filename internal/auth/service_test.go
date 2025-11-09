package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/test/helpers"
	"github.com/richxcame/ride-hailing/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func newTestService(t *testing.T, repo RepositoryInterface) *Service {
	t.Helper()
	manager, err := jwtkeys.NewManager(context.Background(), jwtkeys.Config{
		RotationInterval: 365 * 24 * time.Hour,
		GracePeriod:      365 * 24 * time.Hour,
		LegacySecret:     "test-secret",
	})
	if err != nil {
		t.Fatalf("failed to create jwt manager: %v", err)
	}
	return NewService(repo, manager, 24)
}

func TestService_Register_Success(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	req := helpers.CreateTestRegisterRequest()

	// Mock expectations
	mockRepo.On("GetUserByEmail", ctx, req.Email).Return(nil, errors.New("not found"))
	mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	// Execute
	user, err := service.Register(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, req.Email, user.Email)
	assert.Equal(t, req.FirstName, user.FirstName)
	assert.Equal(t, req.LastName, user.LastName)
	assert.Equal(t, req.Role, user.Role)
	assert.True(t, user.IsActive)
	assert.False(t, user.IsVerified)
	helpers.AssertPasswordNotInResponse(t, user)
	mockRepo.AssertExpectations(t)
}

func TestService_Register_UserAlreadyExists(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	req := helpers.CreateTestRegisterRequest()
	existingUser := helpers.CreateTestUser()

	// Mock expectations
	mockRepo.On("GetUserByEmail", ctx, req.Email).Return(existingUser, nil)

	// Execute
	user, err := service.Register(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 409, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_Register_RepositoryError(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	req := helpers.CreateTestRegisterRequest()

	// Mock expectations
	mockRepo.On("GetUserByEmail", ctx, req.Email).Return(nil, errors.New("not found"))
	mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*models.User")).Return(errors.New("database error"))

	// Execute
	user, err := service.Register(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 500, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_RegisterDriver_Success(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	req := &models.RegisterRequest{
		Email:       "driver@example.com",
		Password:    "SecurePassword123!",
		PhoneNumber: "+1234567890",
		FirstName:   "John",
		LastName:    "Driver",
		Role:        "driver",
	}
	driver := &models.Driver{
		LicenseNumber: "DL123456789",
		VehicleModel:  "Toyota Camry",
		VehiclePlate:  "ABC-1234",
		VehicleColor:  "Silver",
		VehicleYear:   2020,
	}

	// Mock expectations
	mockRepo.On("GetUserByEmail", ctx, req.Email).Return(nil, errors.New("not found"))
	mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*models.User")).Return(nil)
	mockRepo.On("CreateDriver", ctx, mock.AnythingOfType("*models.Driver")).Return(nil)

	// Execute
	user, err := service.RegisterDriver(ctx, req, driver)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, req.Email, user.Email)
	assert.Equal(t, req.FirstName, user.FirstName)
	mockRepo.AssertExpectations(t)
}

func TestService_RegisterDriver_UserCreationFails(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	req := helpers.CreateTestRegisterRequest()
	req.Role = "driver"
	driver := helpers.CreateTestDriver(uuid.New())

	// Mock expectations
	mockRepo.On("GetUserByEmail", ctx, req.Email).Return(nil, errors.New("not found"))
	mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*models.User")).Return(errors.New("database error"))

	// Execute
	user, err := service.RegisterDriver(ctx, req, driver)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	mockRepo.AssertExpectations(t)
}

func TestService_RegisterDriver_DriverCreationFails(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	req := helpers.CreateTestRegisterRequest()
	req.Role = "driver"
	driver := helpers.CreateTestDriver(uuid.New())

	// Mock expectations
	mockRepo.On("GetUserByEmail", ctx, req.Email).Return(nil, errors.New("not found"))
	mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*models.User")).Return(nil)
	mockRepo.On("CreateDriver", ctx, mock.AnythingOfType("*models.Driver")).Return(errors.New("database error"))

	// Execute
	user, err := service.RegisterDriver(ctx, req, driver)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 500, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_Login_Success(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	req := helpers.CreateTestLoginRequest()
	testUser := helpers.CreateTestUser()

	// Mock expectations
	mockRepo.On("GetUserByEmail", ctx, req.Email).Return(testUser, nil)

	// Execute
	response, err := service.Login(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.User)
	assert.NotEmpty(t, response.Token)
	helpers.AssertPasswordNotInResponse(t, response.User)
	helpers.AssertValidJWT(t, response.Token)
	mockRepo.AssertExpectations(t)
}

func TestService_Login_UserNotFound(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	req := helpers.CreateTestLoginRequest()

	// Mock expectations
	mockRepo.On("GetUserByEmail", ctx, req.Email).Return(nil, errors.New("not found"))

	// Execute
	response, err := service.Login(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 401, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_Login_InactiveUser(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	req := helpers.CreateTestLoginRequest()
	testUser := helpers.CreateTestUser()
	testUser.IsActive = false

	// Mock expectations
	mockRepo.On("GetUserByEmail", ctx, req.Email).Return(testUser, nil)

	// Execute
	response, err := service.Login(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 401, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_Login_InvalidPassword(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	req := helpers.CreateTestLoginRequest()
	req.Password = "wrongpassword"
	testUser := helpers.CreateTestUser()

	// Mock expectations
	mockRepo.On("GetUserByEmail", ctx, req.Email).Return(testUser, nil)

	// Execute
	response, err := service.Login(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 401, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_GetProfile_Success(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	testUser := helpers.CreateTestUser()

	// Mock expectations
	mockRepo.On("GetUserByID", ctx, testUser.ID).Return(testUser, nil)

	// Execute
	user, err := service.GetProfile(ctx, testUser.ID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	helpers.AssertUserEqual(t, testUser, user)
	helpers.AssertPasswordNotInResponse(t, user)
	mockRepo.AssertExpectations(t)
}

func TestService_GetProfile_UserNotFound(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	userID := uuid.New()

	// Mock expectations
	mockRepo.On("GetUserByID", ctx, userID).Return(nil, errors.New("not found"))

	// Execute
	user, err := service.GetProfile(ctx, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 404, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_UpdateProfile_Success(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	testUser := helpers.CreateTestUser()
	updates := &models.User{
		FirstName:   "UpdatedFirst",
		LastName:    "UpdatedLast",
		PhoneNumber: "+9876543210",
	}

	// Mock expectations
	mockRepo.On("GetUserByID", ctx, testUser.ID).Return(testUser, nil)
	mockRepo.On("UpdateUser", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	// Execute
	updatedUser, err := service.UpdateProfile(ctx, testUser.ID, updates)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, updatedUser)
	assert.Equal(t, updates.FirstName, updatedUser.FirstName)
	assert.Equal(t, updates.LastName, updatedUser.LastName)
	assert.Equal(t, updates.PhoneNumber, updatedUser.PhoneNumber)
	helpers.AssertPasswordNotInResponse(t, updatedUser)
	mockRepo.AssertExpectations(t)
}

func TestService_UpdateProfile_UserNotFound(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	updates := &models.User{
		FirstName: "UpdatedFirst",
	}

	// Mock expectations
	mockRepo.On("GetUserByID", ctx, userID).Return(nil, errors.New("not found"))

	// Execute
	user, err := service.UpdateProfile(ctx, userID, updates)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 404, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_UpdateProfile_RepositoryError(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	testUser := helpers.CreateTestUser()
	updates := &models.User{
		FirstName: "UpdatedFirst",
	}

	// Mock expectations
	mockRepo.On("GetUserByID", ctx, testUser.ID).Return(testUser, nil)
	mockRepo.On("UpdateUser", ctx, mock.AnythingOfType("*models.User")).Return(errors.New("database error"))

	// Execute
	user, err := service.UpdateProfile(ctx, testUser.ID, updates)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 500, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_GenerateToken_ValidToken(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	testUser := helpers.CreateTestUser()

	// Execute
	tokenString, err := service.generateToken(context.Background(), testUser)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// Verify token can be parsed
	token, err := jwt.ParseWithClaims(tokenString, &middleware.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	// Verify claims
	claims, ok := token.Claims.(*middleware.Claims)
	assert.True(t, ok)
	assert.Equal(t, testUser.ID, claims.UserID)
	assert.Equal(t, testUser.Email, claims.Email)
	assert.Equal(t, testUser.Role, claims.Role)
}

func TestService_GenerateToken_ContainsCorrectClaims(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	testUser := helpers.CreateTestUser()

	// Execute
	tokenString, err := service.generateToken(context.Background(), testUser)

	// Assert
	assert.NoError(t, err)

	// Parse and verify
	token, _ := jwt.ParseWithClaims(tokenString, &middleware.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})

	claims := token.Claims.(*middleware.Claims)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	assert.True(t, claims.ExpiresAt.After(claims.IssuedAt.Time))
}

func TestPasswordHashing(t *testing.T) {
	password := "SecurePassword123!"

	// Generate hash
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)
	assert.NotEmpty(t, hashedPassword)

	// Verify correct password
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	assert.NoError(t, err)

	// Verify incorrect password
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte("wrongpassword"))
	assert.Error(t, err)
}

func TestService_UpdateProfile_PartialUpdates(t *testing.T) {
	// Setup
	mockRepo := new(mocks.MockAuthRepository)
	service := newTestService(t, mockRepo)
	ctx := context.Background()
	testUser := helpers.CreateTestUser()
	originalLastName := testUser.LastName

	// Test updating only first name
	updates := &models.User{
		FirstName: "NewFirstName",
	}

	// Mock expectations
	mockRepo.On("GetUserByID", ctx, testUser.ID).Return(testUser, nil)
	mockRepo.On("UpdateUser", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	// Execute
	updatedUser, err := service.UpdateProfile(ctx, testUser.ID, updates)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "NewFirstName", updatedUser.FirstName)
	assert.Equal(t, originalLastName, updatedUser.LastName) // Should remain unchanged
	mockRepo.AssertExpectations(t)
}
