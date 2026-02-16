package favorites

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// INTERNAL MOCK (implements RepositoryInterface within this package)
// ========================================

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) CreateFavorite(ctx context.Context, favorite *FavoriteLocation) error {
	args := m.Called(ctx, favorite)
	return args.Error(0)
}

func (m *mockRepo) GetFavoriteByID(ctx context.Context, id uuid.UUID) (*FavoriteLocation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FavoriteLocation), args.Error(1)
}

func (m *mockRepo) GetFavoritesByUser(ctx context.Context, userID uuid.UUID) ([]*FavoriteLocation, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*FavoriteLocation), args.Error(1)
}

func (m *mockRepo) UpdateFavorite(ctx context.Context, favorite *FavoriteLocation) error {
	args := m.Called(ctx, favorite)
	return args.Error(0)
}

func (m *mockRepo) DeleteFavorite(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

// ========================================
// TEST HELPERS
// ========================================

func newTestService(repo RepositoryInterface) *Service {
	return NewService(repo)
}

func createTestFavorite(userID uuid.UUID, name, address string, latitude, longitude float64) *FavoriteLocation {
	return &FavoriteLocation{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		Address:   address,
		Latitude:  latitude,
		Longitude: longitude,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ========================================
// TESTS: CreateFavoriteLocation
// ========================================

func TestCreateFavoriteLocation(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name       string
		favName    string
		address    string
		latitude   float64
		longitude  float64
		setupMocks func(m *mockRepo)
		wantErr    error
		validate   func(t *testing.T, fav *FavoriteLocation)
	}{
		{
			name:      "success - creates favorite location",
			favName:   "Home",
			address:   "123 Main St",
			latitude:  40.7128,
			longitude: -74.0060,
			setupMocks: func(m *mockRepo) {
				m.On("CreateFavorite", mock.Anything, mock.MatchedBy(func(f *FavoriteLocation) bool {
					return f.Name == "Home" && f.Address == "123 Main St" &&
						f.Latitude == 40.7128 && f.Longitude == -74.0060 &&
						f.UserID == userID
				})).Return(nil)
			},
			wantErr: nil,
			validate: func(t *testing.T, fav *FavoriteLocation) {
				assert.Equal(t, "Home", fav.Name)
				assert.Equal(t, "123 Main St", fav.Address)
				assert.Equal(t, 40.7128, fav.Latitude)
				assert.Equal(t, -74.0060, fav.Longitude)
				assert.Equal(t, userID, fav.UserID)
			},
		},
		{
			name:      "success - boundary latitude 90",
			favName:   "North Pole",
			address:   "North Pole",
			latitude:  90,
			longitude: 0,
			setupMocks: func(m *mockRepo) {
				m.On("CreateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(nil)
			},
			wantErr: nil,
			validate: func(t *testing.T, fav *FavoriteLocation) {
				assert.Equal(t, float64(90), fav.Latitude)
			},
		},
		{
			name:      "success - boundary latitude -90",
			favName:   "South Pole",
			address:   "South Pole",
			latitude:  -90,
			longitude: 0,
			setupMocks: func(m *mockRepo) {
				m.On("CreateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(nil)
			},
			wantErr: nil,
			validate: func(t *testing.T, fav *FavoriteLocation) {
				assert.Equal(t, float64(-90), fav.Latitude)
			},
		},
		{
			name:      "success - boundary longitude 180",
			favName:   "Date Line East",
			address:   "International Date Line",
			latitude:  0,
			longitude: 180,
			setupMocks: func(m *mockRepo) {
				m.On("CreateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(nil)
			},
			wantErr: nil,
			validate: func(t *testing.T, fav *FavoriteLocation) {
				assert.Equal(t, float64(180), fav.Longitude)
			},
		},
		{
			name:      "success - boundary longitude -180",
			favName:   "Date Line West",
			address:   "International Date Line",
			latitude:  0,
			longitude: -180,
			setupMocks: func(m *mockRepo) {
				m.On("CreateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(nil)
			},
			wantErr: nil,
			validate: func(t *testing.T, fav *FavoriteLocation) {
				assert.Equal(t, float64(-180), fav.Longitude)
			},
		},
		{
			name:       "error - empty name",
			favName:    "",
			address:    "123 Main St",
			latitude:   40.7128,
			longitude:  -74.0060,
			setupMocks: func(m *mockRepo) {},
			wantErr:    ErrInvalidName,
			validate:   nil,
		},
		{
			name:       "error - empty address",
			favName:    "Home",
			address:    "",
			latitude:   40.7128,
			longitude:  -74.0060,
			setupMocks: func(m *mockRepo) {},
			wantErr:    ErrInvalidAddress,
			validate:   nil,
		},
		{
			name:       "error - latitude too high (> 90)",
			favName:    "Invalid",
			address:    "123 Main St",
			latitude:   90.1,
			longitude:  0,
			setupMocks: func(m *mockRepo) {},
			wantErr:    ErrInvalidCoordinates,
			validate:   nil,
		},
		{
			name:       "error - latitude too low (< -90)",
			favName:    "Invalid",
			address:    "123 Main St",
			latitude:   -90.1,
			longitude:  0,
			setupMocks: func(m *mockRepo) {},
			wantErr:    ErrInvalidCoordinates,
			validate:   nil,
		},
		{
			name:       "error - longitude too high (> 180)",
			favName:    "Invalid",
			address:    "123 Main St",
			latitude:   0,
			longitude:  180.1,
			setupMocks: func(m *mockRepo) {},
			wantErr:    ErrInvalidCoordinates,
			validate:   nil,
		},
		{
			name:       "error - longitude too low (< -180)",
			favName:    "Invalid",
			address:    "123 Main St",
			latitude:   0,
			longitude:  -180.1,
			setupMocks: func(m *mockRepo) {},
			wantErr:    ErrInvalidCoordinates,
			validate:   nil,
		},
		{
			name:      "error - repository error",
			favName:   "Home",
			address:   "123 Main St",
			latitude:  40.7128,
			longitude: -74.0060,
			setupMocks: func(m *mockRepo) {
				m.On("CreateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).
					Return(errors.New("database error"))
			},
			wantErr:  errors.New("database error"),
			validate: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			fav, err := svc.CreateFavoriteLocation(context.Background(), userID, tt.favName, tt.address, tt.latitude, tt.longitude)

			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.wantErr, ErrInvalidName) || errors.Is(tt.wantErr, ErrInvalidAddress) || errors.Is(tt.wantErr, ErrInvalidCoordinates) {
					assert.ErrorIs(t, err, tt.wantErr)
				}
				assert.Nil(t, fav)
			} else {
				require.NoError(t, err)
				require.NotNil(t, fav)
				tt.validate(t, fav)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetFavoriteLocations
// ========================================

func TestGetFavoriteLocations(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, favs []*FavoriteLocation)
	}{
		{
			name: "success - returns user favorites",
			setupMocks: func(m *mockRepo) {
				favorites := []*FavoriteLocation{
					createTestFavorite(userID, "Home", "123 Main St", 40.7128, -74.0060),
					createTestFavorite(userID, "Work", "456 Office Ave", 40.7580, -73.9855),
				}
				m.On("GetFavoritesByUser", mock.Anything, userID).Return(favorites, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, favs []*FavoriteLocation) {
				assert.Len(t, favs, 2)
				assert.Equal(t, "Home", favs[0].Name)
				assert.Equal(t, "Work", favs[1].Name)
			},
		},
		{
			name: "success - returns empty list for user with no favorites",
			setupMocks: func(m *mockRepo) {
				m.On("GetFavoritesByUser", mock.Anything, userID).Return([]*FavoriteLocation{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, favs []*FavoriteLocation) {
				assert.Len(t, favs, 0)
			},
		},
		{
			name: "error - repository error",
			setupMocks: func(m *mockRepo) {
				m.On("GetFavoritesByUser", mock.Anything, userID).Return(nil, errors.New("database error"))
			},
			wantErr: true,
			validate: func(t *testing.T, favs []*FavoriteLocation) {
				// Not called
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			favs, err := svc.GetFavoriteLocations(context.Background(), userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, favs)
			} else {
				require.NoError(t, err)
				tt.validate(t, favs)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetFavoriteLocation
// ========================================

func TestGetFavoriteLocation(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	favoriteID := uuid.New()

	tests := []struct {
		name       string
		id         uuid.UUID
		userID     uuid.UUID
		setupMocks func(m *mockRepo)
		wantErr    error
		validate   func(t *testing.T, fav *FavoriteLocation)
	}{
		{
			name:   "success - returns favorite for owner",
			id:     favoriteID,
			userID: userID,
			setupMocks: func(m *mockRepo) {
				fav := createTestFavorite(userID, "Home", "123 Main St", 40.7128, -74.0060)
				fav.ID = favoriteID
				m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(fav, nil)
			},
			wantErr: nil,
			validate: func(t *testing.T, fav *FavoriteLocation) {
				assert.Equal(t, favoriteID, fav.ID)
				assert.Equal(t, userID, fav.UserID)
				assert.Equal(t, "Home", fav.Name)
			},
		},
		{
			name:   "error - unauthorized access (different user)",
			id:     favoriteID,
			userID: otherUserID,
			setupMocks: func(m *mockRepo) {
				fav := createTestFavorite(userID, "Home", "123 Main St", 40.7128, -74.0060)
				fav.ID = favoriteID
				m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(fav, nil)
			},
			wantErr:  ErrUnauthorized,
			validate: nil,
		},
		{
			name:   "error - favorite not found",
			id:     favoriteID,
			userID: userID,
			setupMocks: func(m *mockRepo) {
				m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(nil, errors.New("not found"))
			},
			wantErr:  errors.New("not found"),
			validate: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			fav, err := svc.GetFavoriteLocation(context.Background(), tt.id, tt.userID)

			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.wantErr, ErrUnauthorized) {
					assert.ErrorIs(t, err, ErrUnauthorized)
				}
				assert.Nil(t, fav)
			} else {
				require.NoError(t, err)
				require.NotNil(t, fav)
				tt.validate(t, fav)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: UpdateFavoriteLocation
// ========================================

func TestUpdateFavoriteLocation(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	favoriteID := uuid.New()

	tests := []struct {
		name       string
		id         uuid.UUID
		userID     uuid.UUID
		favName    string
		address    string
		latitude   float64
		longitude  float64
		setupMocks func(m *mockRepo)
		wantErr    error
		validate   func(t *testing.T, fav *FavoriteLocation)
	}{
		{
			name:      "success - updates favorite",
			id:        favoriteID,
			userID:    userID,
			favName:   "Updated Home",
			address:   "789 New St",
			latitude:  41.0,
			longitude: -75.0,
			setupMocks: func(m *mockRepo) {
				existing := createTestFavorite(userID, "Home", "123 Main St", 40.7128, -74.0060)
				existing.ID = favoriteID
				m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(existing, nil)
				m.On("UpdateFavorite", mock.Anything, mock.MatchedBy(func(f *FavoriteLocation) bool {
					return f.ID == favoriteID && f.Name == "Updated Home" &&
						f.Address == "789 New St" && f.Latitude == 41.0 && f.Longitude == -75.0
				})).Return(nil)
			},
			wantErr: nil,
			validate: func(t *testing.T, fav *FavoriteLocation) {
				assert.Equal(t, "Updated Home", fav.Name)
				assert.Equal(t, "789 New St", fav.Address)
				assert.Equal(t, 41.0, fav.Latitude)
				assert.Equal(t, -75.0, fav.Longitude)
			},
		},
		{
			name:       "error - empty name",
			id:         favoriteID,
			userID:     userID,
			favName:    "",
			address:    "789 New St",
			latitude:   41.0,
			longitude:  -75.0,
			setupMocks: func(m *mockRepo) {},
			wantErr:    ErrInvalidName,
			validate:   nil,
		},
		{
			name:       "error - empty address",
			id:         favoriteID,
			userID:     userID,
			favName:    "Home",
			address:    "",
			latitude:   41.0,
			longitude:  -75.0,
			setupMocks: func(m *mockRepo) {},
			wantErr:    ErrInvalidAddress,
			validate:   nil,
		},
		{
			name:       "error - invalid latitude (> 90)",
			id:         favoriteID,
			userID:     userID,
			favName:    "Home",
			address:    "123 Main St",
			latitude:   91.0,
			longitude:  -75.0,
			setupMocks: func(m *mockRepo) {},
			wantErr:    ErrInvalidCoordinates,
			validate:   nil,
		},
		{
			name:       "error - invalid latitude (< -90)",
			id:         favoriteID,
			userID:     userID,
			favName:    "Home",
			address:    "123 Main St",
			latitude:   -91.0,
			longitude:  -75.0,
			setupMocks: func(m *mockRepo) {},
			wantErr:    ErrInvalidCoordinates,
			validate:   nil,
		},
		{
			name:       "error - invalid longitude (> 180)",
			id:         favoriteID,
			userID:     userID,
			favName:    "Home",
			address:    "123 Main St",
			latitude:   41.0,
			longitude:  181.0,
			setupMocks: func(m *mockRepo) {},
			wantErr:    ErrInvalidCoordinates,
			validate:   nil,
		},
		{
			name:       "error - invalid longitude (< -180)",
			id:         favoriteID,
			userID:     userID,
			favName:    "Home",
			address:    "123 Main St",
			latitude:   41.0,
			longitude:  -181.0,
			setupMocks: func(m *mockRepo) {},
			wantErr:    ErrInvalidCoordinates,
			validate:   nil,
		},
		{
			name:      "error - unauthorized access",
			id:        favoriteID,
			userID:    otherUserID,
			favName:   "Updated Home",
			address:   "789 New St",
			latitude:  41.0,
			longitude: -75.0,
			setupMocks: func(m *mockRepo) {
				existing := createTestFavorite(userID, "Home", "123 Main St", 40.7128, -74.0060)
				existing.ID = favoriteID
				m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(existing, nil)
			},
			wantErr:  ErrUnauthorized,
			validate: nil,
		},
		{
			name:      "error - favorite not found",
			id:        favoriteID,
			userID:    userID,
			favName:   "Updated Home",
			address:   "789 New St",
			latitude:  41.0,
			longitude: -75.0,
			setupMocks: func(m *mockRepo) {
				m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(nil, errors.New("not found"))
			},
			wantErr:  errors.New("not found"),
			validate: nil,
		},
		{
			name:      "error - repository update error",
			id:        favoriteID,
			userID:    userID,
			favName:   "Updated Home",
			address:   "789 New St",
			latitude:  41.0,
			longitude: -75.0,
			setupMocks: func(m *mockRepo) {
				existing := createTestFavorite(userID, "Home", "123 Main St", 40.7128, -74.0060)
				existing.ID = favoriteID
				m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(existing, nil)
				m.On("UpdateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).
					Return(errors.New("update failed"))
			},
			wantErr:  errors.New("update failed"),
			validate: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			fav, err := svc.UpdateFavoriteLocation(context.Background(), tt.id, tt.userID, tt.favName, tt.address, tt.latitude, tt.longitude)

			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.wantErr, ErrInvalidName) || errors.Is(tt.wantErr, ErrInvalidAddress) ||
					errors.Is(tt.wantErr, ErrInvalidCoordinates) || errors.Is(tt.wantErr, ErrUnauthorized) {
					assert.ErrorIs(t, err, tt.wantErr)
				}
				assert.Nil(t, fav)
			} else {
				require.NoError(t, err)
				require.NotNil(t, fav)
				tt.validate(t, fav)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: DeleteFavoriteLocation
// ========================================

func TestDeleteFavoriteLocation(t *testing.T) {
	userID := uuid.New()
	favoriteID := uuid.New()

	tests := []struct {
		name       string
		id         uuid.UUID
		userID     uuid.UUID
		setupMocks func(m *mockRepo)
		wantErr    bool
	}{
		{
			name:   "success - deletes favorite",
			id:     favoriteID,
			userID: userID,
			setupMocks: func(m *mockRepo) {
				m.On("DeleteFavorite", mock.Anything, favoriteID, userID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "error - favorite not found or unauthorized",
			id:     favoriteID,
			userID: userID,
			setupMocks: func(m *mockRepo) {
				m.On("DeleteFavorite", mock.Anything, favoriteID, userID).
					Return(errors.New("favorite location not found"))
			},
			wantErr: true,
		},
		{
			name:   "error - repository error",
			id:     favoriteID,
			userID: userID,
			setupMocks: func(m *mockRepo) {
				m.On("DeleteFavorite", mock.Anything, favoriteID, userID).
					Return(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.DeleteFavoriteLocation(context.Background(), tt.id, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: Coordinate Validation Edge Cases
// ========================================

func TestCoordinateValidation(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name      string
		latitude  float64
		longitude float64
		wantErr   bool
	}{
		// Valid boundary cases
		{name: "valid - latitude 0, longitude 0", latitude: 0, longitude: 0, wantErr: false},
		{name: "valid - latitude 90, longitude 180", latitude: 90, longitude: 180, wantErr: false},
		{name: "valid - latitude -90, longitude -180", latitude: -90, longitude: -180, wantErr: false},
		{name: "valid - latitude 45.5, longitude -122.5", latitude: 45.5, longitude: -122.5, wantErr: false},

		// Invalid latitude cases
		{name: "invalid - latitude 90.0001", latitude: 90.0001, longitude: 0, wantErr: true},
		{name: "invalid - latitude -90.0001", latitude: -90.0001, longitude: 0, wantErr: true},
		{name: "invalid - latitude 100", latitude: 100, longitude: 0, wantErr: true},
		{name: "invalid - latitude -100", latitude: -100, longitude: 0, wantErr: true},

		// Invalid longitude cases
		{name: "invalid - longitude 180.0001", latitude: 0, longitude: 180.0001, wantErr: true},
		{name: "invalid - longitude -180.0001", latitude: 0, longitude: -180.0001, wantErr: true},
		{name: "invalid - longitude 200", latitude: 0, longitude: 200, wantErr: true},
		{name: "invalid - longitude -200", latitude: 0, longitude: -200, wantErr: true},

		// Both invalid
		{name: "invalid - both out of range", latitude: 100, longitude: 200, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			if !tt.wantErr {
				m.On("CreateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(nil)
			}
			svc := newTestService(m)

			_, err := svc.CreateFavoriteLocation(context.Background(), userID, "Test", "Test Address", tt.latitude, tt.longitude)

			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidCoordinates)
			} else {
				assert.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: User Ownership Checks
// ========================================

func TestUserOwnershipChecks(t *testing.T) {
	ownerID := uuid.New()
	attackerID := uuid.New()
	favoriteID := uuid.New()

	t.Run("GetFavoriteLocation - rejects access from non-owner", func(t *testing.T) {
		m := new(mockRepo)
		fav := createTestFavorite(ownerID, "Secret Place", "123 Private St", 40.0, -74.0)
		fav.ID = favoriteID
		m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(fav, nil)

		svc := newTestService(m)
		result, err := svc.GetFavoriteLocation(context.Background(), favoriteID, attackerID)

		assert.ErrorIs(t, err, ErrUnauthorized)
		assert.Nil(t, result)
		m.AssertExpectations(t)
	})

	t.Run("GetFavoriteLocation - allows access from owner", func(t *testing.T) {
		m := new(mockRepo)
		fav := createTestFavorite(ownerID, "Secret Place", "123 Private St", 40.0, -74.0)
		fav.ID = favoriteID
		m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(fav, nil)

		svc := newTestService(m)
		result, err := svc.GetFavoriteLocation(context.Background(), favoriteID, ownerID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, ownerID, result.UserID)
		m.AssertExpectations(t)
	})

	t.Run("UpdateFavoriteLocation - rejects update from non-owner", func(t *testing.T) {
		m := new(mockRepo)
		fav := createTestFavorite(ownerID, "Secret Place", "123 Private St", 40.0, -74.0)
		fav.ID = favoriteID
		m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(fav, nil)

		svc := newTestService(m)
		result, err := svc.UpdateFavoriteLocation(context.Background(), favoriteID, attackerID, "Hacked", "Evil Address", 0, 0)

		assert.ErrorIs(t, err, ErrUnauthorized)
		assert.Nil(t, result)
		m.AssertExpectations(t)
	})

	t.Run("UpdateFavoriteLocation - allows update from owner", func(t *testing.T) {
		m := new(mockRepo)
		fav := createTestFavorite(ownerID, "Secret Place", "123 Private St", 40.0, -74.0)
		fav.ID = favoriteID
		m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(fav, nil)
		m.On("UpdateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(nil)

		svc := newTestService(m)
		result, err := svc.UpdateFavoriteLocation(context.Background(), favoriteID, ownerID, "Updated", "New Address", 41.0, -75.0)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		m.AssertExpectations(t)
	})

	t.Run("DeleteFavoriteLocation - uses userID in query (repository handles auth)", func(t *testing.T) {
		m := new(mockRepo)
		// DeleteFavorite takes both id and userID, repository enforces ownership
		m.On("DeleteFavorite", mock.Anything, favoriteID, attackerID).
			Return(errors.New("favorite location not found"))

		svc := newTestService(m)
		err := svc.DeleteFavoriteLocation(context.Background(), favoriteID, attackerID)

		assert.Error(t, err)
		m.AssertExpectations(t)
	})
}

// ========================================
// TESTS: Multiple Favorites for User
// ========================================

func TestMultipleFavoritesForUser(t *testing.T) {
	userID := uuid.New()

	t.Run("user can have multiple favorites", func(t *testing.T) {
		m := new(mockRepo)
		favorites := []*FavoriteLocation{
			createTestFavorite(userID, "Home", "123 Home St", 40.0, -74.0),
			createTestFavorite(userID, "Work", "456 Work Ave", 40.5, -74.5),
			createTestFavorite(userID, "Gym", "789 Fitness Blvd", 41.0, -75.0),
			createTestFavorite(userID, "Mom's House", "321 Family Lane", 39.0, -73.0),
		}
		m.On("GetFavoritesByUser", mock.Anything, userID).Return(favorites, nil)

		svc := newTestService(m)
		result, err := svc.GetFavoriteLocations(context.Background(), userID)

		assert.NoError(t, err)
		assert.Len(t, result, 4)
		m.AssertExpectations(t)
	})
}

// ========================================
// TESTS: Service Constructor
// ========================================

func TestNewService(t *testing.T) {
	t.Run("creates service with repository", func(t *testing.T) {
		m := new(mockRepo)
		svc := NewService(m)

		assert.NotNil(t, svc)
		assert.Equal(t, m, svc.repo)
	})
}
