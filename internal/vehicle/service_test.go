package vehicle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (m *mockRepo) CreateVehicle(ctx context.Context, v *Vehicle) error {
	args := m.Called(ctx, v)
	return args.Error(0)
}

func (m *mockRepo) GetVehicleByID(ctx context.Context, id uuid.UUID) (*Vehicle, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Vehicle), args.Error(1)
}

func (m *mockRepo) GetVehiclesByDriver(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]Vehicle, int64, error) {
	args := m.Called(ctx, driverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]Vehicle), args.Get(1).(int64), args.Error(2)
}

func (m *mockRepo) GetPrimaryVehicle(ctx context.Context, driverID uuid.UUID) (*Vehicle, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Vehicle), args.Error(1)
}

func (m *mockRepo) UpdateVehicle(ctx context.Context, v *Vehicle) error {
	args := m.Called(ctx, v)
	return args.Error(0)
}

func (m *mockRepo) UpdateVehiclePhotos(ctx context.Context, vehicleID uuid.UUID, photos *UploadVehiclePhotosRequest) error {
	args := m.Called(ctx, vehicleID, photos)
	return args.Error(0)
}

func (m *mockRepo) UpdateVehicleStatus(ctx context.Context, vehicleID uuid.UUID, status VehicleStatus, reason *string) error {
	args := m.Called(ctx, vehicleID, status, reason)
	return args.Error(0)
}

func (m *mockRepo) SetPrimaryVehicle(ctx context.Context, driverID, vehicleID uuid.UUID) error {
	args := m.Called(ctx, driverID, vehicleID)
	return args.Error(0)
}

func (m *mockRepo) RetireVehicle(ctx context.Context, vehicleID uuid.UUID) error {
	args := m.Called(ctx, vehicleID)
	return args.Error(0)
}

func (m *mockRepo) GetAllVehicles(ctx context.Context, filter *AdminVehicleFilter, limit, offset int) ([]Vehicle, int64, error) {
	args := m.Called(ctx, filter, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]Vehicle), args.Get(1).(int64), args.Error(2)
}

func (m *mockRepo) GetPendingReviewVehicles(ctx context.Context, limit, offset int) ([]Vehicle, int, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]Vehicle), args.Int(1), args.Error(2)
}

func (m *mockRepo) GetVehicleStats(ctx context.Context) (*VehicleStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*VehicleStats), args.Error(1)
}

func (m *mockRepo) GetExpiringVehicles(ctx context.Context, daysAhead int) ([]Vehicle, error) {
	args := m.Called(ctx, daysAhead)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Vehicle), args.Error(1)
}

func (m *mockRepo) CreateInspection(ctx context.Context, insp *VehicleInspection) error {
	args := m.Called(ctx, insp)
	return args.Error(0)
}

func (m *mockRepo) GetInspectionsByVehicle(ctx context.Context, vehicleID uuid.UUID) ([]VehicleInspection, error) {
	args := m.Called(ctx, vehicleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]VehicleInspection), args.Error(1)
}

func (m *mockRepo) CreateMaintenanceReminder(ctx context.Context, rem *MaintenanceReminder) error {
	args := m.Called(ctx, rem)
	return args.Error(0)
}

func (m *mockRepo) GetPendingReminders(ctx context.Context, vehicleID uuid.UUID) ([]MaintenanceReminder, error) {
	args := m.Called(ctx, vehicleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]MaintenanceReminder), args.Error(1)
}

func (m *mockRepo) CompleteReminder(ctx context.Context, reminderID uuid.UUID) error {
	args := m.Called(ctx, reminderID)
	return args.Error(0)
}

// ========================================
// TEST HELPERS
// ========================================

func newTestService(repo RepositoryInterface) *Service {
	return NewService(repo)
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func categoryPtr(c VehicleCategory) *VehicleCategory {
	return &c
}

// ========================================
// TESTS: RegisterVehicle
// ========================================

func TestRegisterVehicle(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	currentYear := time.Now().Year()

	tests := []struct {
		name       string
		req        *RegisterVehicleRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, v *Vehicle)
	}{
		{
			name: "success - first vehicle becomes primary",
			req: &RegisterVehicleRequest{
				Make:         "Toyota",
				Model:        "Camry",
				Year:         2022,
				Color:        "White",
				LicensePlate: "ABC123",
				Category:     VehicleCategoryComfort,
				FuelType:     FuelTypeGasoline,
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
				m.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, v *Vehicle) {
				assert.Equal(t, driverID, v.DriverID)
				assert.Equal(t, VehicleStatusPending, v.Status)
				assert.Equal(t, "Toyota", v.Make)
				assert.Equal(t, "Camry", v.Model)
				assert.Equal(t, 2022, v.Year)
				assert.True(t, v.IsActive)
				assert.True(t, v.IsPrimary, "first vehicle should be primary")
				assert.Equal(t, 4, v.MaxPassengers, "default max passengers")
			},
		},
		{
			name: "success - second vehicle not primary",
			req: &RegisterVehicleRequest{
				Make:         "Honda",
				Model:        "Accord",
				Year:         2021,
				Color:        "Black",
				LicensePlate: "XYZ789",
				Category:     VehicleCategoryEconomy,
				FuelType:     FuelTypeHybrid,
			},
			setupMocks: func(m *mockRepo) {
				existingVehicle := Vehicle{
					ID:       uuid.New(),
					DriverID: driverID,
					IsActive: true,
				}
				m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{existingVehicle}, int64(1), nil)
				m.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, v *Vehicle) {
				assert.False(t, v.IsPrimary, "second vehicle should not be primary")
			},
		},
		{
			name: "success - custom max passengers",
			req: &RegisterVehicleRequest{
				Make:          "Mercedes",
				Model:         "V-Class",
				Year:          2023,
				Color:         "Silver",
				LicensePlate:  "LUX001",
				Category:      VehicleCategoryXL,
				FuelType:      FuelTypeDiesel,
				MaxPassengers: 7,
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
				m.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, v *Vehicle) {
				assert.Equal(t, 7, v.MaxPassengers)
			},
		},
		{
			name: "error - year too old",
			req: &RegisterVehicleRequest{
				Make:         "Ford",
				Model:        "Focus",
				Year:         2005,
				Color:        "Blue",
				LicensePlate: "OLD001",
				Category:     VehicleCategoryEconomy,
				FuelType:     FuelTypeGasoline,
			},
			setupMocks: func(m *mockRepo) {},
			wantErr:    true,
			errContain: "vehicle year must be between",
		},
		{
			name: "error - year too far in future",
			req: &RegisterVehicleRequest{
				Make:         "Tesla",
				Model:        "Cybertruck",
				Year:         currentYear + 5,
				Color:        "Silver",
				LicensePlate: "FUT001",
				Category:     VehicleCategoryElectric,
				FuelType:     FuelTypeElectric,
			},
			setupMocks: func(m *mockRepo) {},
			wantErr:    true,
			errContain: "vehicle year must be between",
		},
		{
			name: "error - max vehicles exceeded",
			req: &RegisterVehicleRequest{
				Make:         "BMW",
				Model:        "X5",
				Year:         2022,
				Color:        "Black",
				LicensePlate: "MAX001",
				Category:     VehicleCategoryPremium,
				FuelType:     FuelTypeGasoline,
			},
			setupMocks: func(m *mockRepo) {
				existingVehicles := []Vehicle{
					{ID: uuid.New(), DriverID: driverID, IsActive: true},
					{ID: uuid.New(), DriverID: driverID, IsActive: true},
					{ID: uuid.New(), DriverID: driverID, IsActive: true},
				}
				m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return(existingVehicles, int64(len(existingVehicles)), nil)
			},
			wantErr:    true,
			errContain: "maximum 3 active vehicles allowed",
		},
		{
			name: "success - inactive vehicles don't count towards limit",
			req: &RegisterVehicleRequest{
				Make:         "Audi",
				Model:        "A4",
				Year:         2022,
				Color:        "Gray",
				LicensePlate: "AUD001",
				Category:     VehicleCategoryPremium,
				FuelType:     FuelTypeGasoline,
			},
			setupMocks: func(m *mockRepo) {
				existingVehicles := []Vehicle{
					{ID: uuid.New(), DriverID: driverID, IsActive: true},
					{ID: uuid.New(), DriverID: driverID, IsActive: true},
					{ID: uuid.New(), DriverID: driverID, IsActive: false}, // inactive
					{ID: uuid.New(), DriverID: driverID, IsActive: false}, // inactive
				}
				m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return(existingVehicles, int64(len(existingVehicles)), nil)
				m.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - repository get vehicles fails",
			req: &RegisterVehicleRequest{
				Make:         "Lexus",
				Model:        "ES",
				Year:         2022,
				Color:        "White",
				LicensePlate: "LEX001",
				Category:     VehicleCategoryLux,
				FuelType:     FuelTypeHybrid,
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return(nil, int64(0), errors.New("database error"))
			},
			wantErr:    true,
			errContain: "database error",
		},
		{
			name: "error - repository create fails",
			req: &RegisterVehicleRequest{
				Make:         "Porsche",
				Model:        "Cayenne",
				Year:         2023,
				Color:        "Red",
				LicensePlate: "POR001",
				Category:     VehicleCategoryLux,
				FuelType:     FuelTypeGasoline,
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
				m.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(errors.New("create failed"))
			},
			wantErr:    true,
			errContain: "create vehicle",
		},
		{
			name: "success - with all optional features",
			req: &RegisterVehicleRequest{
				Make:                "Tesla",
				Model:               "Model S",
				Year:                2023,
				Color:               "Blue",
				LicensePlate:        "TES001",
				VIN:                 stringPtr("5YJ3E1EA1JF000001"),
				Category:            VehicleCategoryElectric,
				FuelType:            FuelTypeElectric,
				MaxPassengers:       5,
				HasChildSeat:        true,
				HasWheelchairAccess: false,
				HasWifi:             true,
				HasCharger:          true,
				PetFriendly:         true,
				LuggageCapacity:     3,
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
				m.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, v *Vehicle) {
				assert.Equal(t, "5YJ3E1EA1JF000001", *v.VIN)
				assert.True(t, v.HasChildSeat)
				assert.True(t, v.HasWifi)
				assert.True(t, v.HasCharger)
				assert.True(t, v.PetFriendly)
				assert.Equal(t, 3, v.LuggageCapacity)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			vehicle, err := svc.RegisterVehicle(context.Background(), driverID, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, vehicle)
			} else {
				require.NoError(t, err)
				require.NotNil(t, vehicle)
				if tt.validate != nil {
					tt.validate(t, vehicle)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetMyVehicles
// ========================================

func TestGetMyVehicles(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		wantTotal  int64
		validate   func(t *testing.T, resp *VehicleListResponse)
	}{
		{
			name: "success - returns vehicles",
			setupMocks: func(m *mockRepo) {
				vehicles := []Vehicle{
					{ID: uuid.New(), Make: "Toyota", Model: "Camry", IsPrimary: true},
					{ID: uuid.New(), Make: "Honda", Model: "Accord", IsPrimary: false},
				}
				m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return(vehicles, int64(2), nil)
			},
			wantErr:   false,
			wantTotal: 2,
			validate: func(t *testing.T, resp *VehicleListResponse) {
				assert.Len(t, resp.Vehicles, 2)
			},
		},
		{
			name: "success - empty list",
			setupMocks: func(m *mockRepo) {
				m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
			},
			wantErr:   false,
			wantTotal: 0,
			validate: func(t *testing.T, resp *VehicleListResponse) {
				assert.NotNil(t, resp.Vehicles)
				assert.Len(t, resp.Vehicles, 0)
			},
		},
		{
			name: "error - database failure",
			setupMocks: func(m *mockRepo) {
				m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return(nil, int64(0), errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			resp, total, err := svc.GetMyVehicles(context.Background(), driverID, 20, 0)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.wantTotal, total)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetVehicle
// ========================================

func TestGetVehicle(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	otherDriverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	vehicleID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, v *Vehicle)
	}{
		{
			name: "success - returns vehicle",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
					Make:     "Toyota",
					Model:    "Camry",
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, v *Vehicle) {
				assert.Equal(t, vehicleID, v.ID)
				assert.Equal(t, "Toyota", v.Make)
			},
		},
		{
			name: "error - vehicle not found",
			setupMocks: func(m *mockRepo) {
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "vehicle not found",
		},
		{
			name: "error - not owner of vehicle",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: otherDriverID,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
		{
			name: "error - database failure",
			setupMocks: func(m *mockRepo) {
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			vehicle, err := svc.GetVehicle(context.Background(), vehicleID, driverID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, vehicle)
				if tt.validate != nil {
					tt.validate(t, vehicle)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetPrimaryVehicle
// ========================================

func TestGetPrimaryVehicle(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	vehicleID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, v *Vehicle)
	}{
		{
			name: "success - returns primary vehicle",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:        vehicleID,
					DriverID:  driverID,
					IsPrimary: true,
					Make:      "Toyota",
				}
				m.On("GetPrimaryVehicle", mock.Anything, driverID).Return(vehicle, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, v *Vehicle) {
				assert.True(t, v.IsPrimary)
			},
		},
		{
			name: "error - no primary vehicle",
			setupMocks: func(m *mockRepo) {
				m.On("GetPrimaryVehicle", mock.Anything, driverID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "no primary vehicle set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			vehicle, err := svc.GetPrimaryVehicle(context.Background(), driverID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, vehicle)
				if tt.validate != nil {
					tt.validate(t, vehicle)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: UpdateVehicle
// ========================================

func TestUpdateVehicle(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	otherDriverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	vehicleID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		req        *UpdateVehicleRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, v *Vehicle)
	}{
		{
			name: "success - update color",
			req: &UpdateVehicleRequest{
				Color: stringPtr("Red"),
			},
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
					Color:    "Blue",
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
				m.On("UpdateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, v *Vehicle) {
				assert.Equal(t, "Red", v.Color)
			},
		},
		{
			name: "success - update multiple fields",
			req: &UpdateVehicleRequest{
				Color:         stringPtr("Green"),
				Category:      categoryPtr(VehicleCategoryPremium),
				MaxPassengers: intPtr(5),
				HasChildSeat:  boolPtr(true),
				HasWifi:       boolPtr(true),
			},
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:            vehicleID,
					DriverID:      driverID,
					Color:         "Blue",
					Category:      VehicleCategoryEconomy,
					MaxPassengers: 4,
					HasChildSeat:  false,
					HasWifi:       false,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
				m.On("UpdateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, v *Vehicle) {
				assert.Equal(t, "Green", v.Color)
				assert.Equal(t, VehicleCategoryPremium, v.Category)
				assert.Equal(t, 5, v.MaxPassengers)
				assert.True(t, v.HasChildSeat)
				assert.True(t, v.HasWifi)
			},
		},
		{
			name: "error - vehicle not found",
			req: &UpdateVehicleRequest{
				Color: stringPtr("Red"),
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "vehicle not found",
		},
		{
			name: "error - not owner",
			req: &UpdateVehicleRequest{
				Color: stringPtr("Red"),
			},
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: otherDriverID,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
		{
			name: "error - update fails",
			req: &UpdateVehicleRequest{
				Color: stringPtr("Red"),
			},
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
				m.On("UpdateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(errors.New("update failed"))
			},
			wantErr:    true,
			errContain: "update vehicle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			vehicle, err := svc.UpdateVehicle(context.Background(), vehicleID, driverID, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, vehicle)
				if tt.validate != nil {
					tt.validate(t, vehicle)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: UploadPhotos
// ========================================

func TestUploadPhotos(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	otherDriverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	vehicleID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		req        *UploadVehiclePhotosRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name: "success - upload photos",
			req: &UploadVehiclePhotosRequest{
				FrontPhotoURL:        stringPtr("https://example.com/front.jpg"),
				BackPhotoURL:         stringPtr("https://example.com/back.jpg"),
				RegistrationPhotoURL: stringPtr("https://example.com/reg.jpg"),
			},
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
				m.On("UpdateVehiclePhotos", mock.Anything, vehicleID, mock.AnythingOfType("*vehicle.UploadVehiclePhotosRequest")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - vehicle not found",
			req: &UploadVehiclePhotosRequest{
				FrontPhotoURL: stringPtr("https://example.com/front.jpg"),
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "vehicle not found",
		},
		{
			name: "error - not owner",
			req: &UploadVehiclePhotosRequest{
				FrontPhotoURL: stringPtr("https://example.com/front.jpg"),
			},
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: otherDriverID,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.UploadPhotos(context.Background(), vehicleID, driverID, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: SetPrimary
// ========================================

func TestSetPrimary(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	otherDriverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	vehicleID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name: "success - set primary vehicle",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
					Status:   VehicleStatusApproved,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
				m.On("SetPrimaryVehicle", mock.Anything, driverID, vehicleID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - vehicle not found",
			setupMocks: func(m *mockRepo) {
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "vehicle not found",
		},
		{
			name: "error - not owner",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: otherDriverID,
					Status:   VehicleStatusApproved,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
		{
			name: "error - vehicle not approved",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
					Status:   VehicleStatusPending,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "only approved vehicles can be set as primary",
		},
		{
			name: "error - rejected vehicle cannot be primary",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
					Status:   VehicleStatusRejected,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "only approved vehicles can be set as primary",
		},
		{
			name: "error - suspended vehicle cannot be primary",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
					Status:   VehicleStatusSuspended,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "only approved vehicles can be set as primary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.SetPrimary(context.Background(), vehicleID, driverID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: RetireVehicle
// ========================================

func TestRetireVehicle(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	otherDriverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	vehicleID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name: "success - retire vehicle",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
				m.On("RetireVehicle", mock.Anything, vehicleID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - vehicle not found",
			setupMocks: func(m *mockRepo) {
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "vehicle not found",
		},
		{
			name: "error - not owner",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: otherDriverID,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.RetireVehicle(context.Background(), vehicleID, driverID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: ReviewVehicle (Admin Approval/Rejection)
// ========================================

func TestReviewVehicle(t *testing.T) {
	vehicleID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		req        *AdminReviewRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name: "success - approve vehicle",
			req: &AdminReviewRequest{
				Approved: true,
			},
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
					Status:   VehicleStatusPending,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
				m.On("UpdateVehicleStatus", mock.Anything, vehicleID, VehicleStatusApproved, (*string)(nil)).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - reject vehicle with reason",
			req: &AdminReviewRequest{
				Approved:        false,
				RejectionReason: stringPtr("Vehicle does not meet safety standards"),
			},
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
					Status:   VehicleStatusPending,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
				m.On("UpdateVehicleStatus", mock.Anything, vehicleID, VehicleStatusRejected, mock.AnythingOfType("*string")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - reject without reason",
			req: &AdminReviewRequest{
				Approved: false,
			},
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
					Status:   VehicleStatusPending,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "rejection reason is required",
		},
		{
			name: "error - reject with empty reason",
			req: &AdminReviewRequest{
				Approved:        false,
				RejectionReason: stringPtr(""),
			},
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
					Status:   VehicleStatusPending,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "rejection reason is required",
		},
		{
			name: "error - vehicle not found",
			req: &AdminReviewRequest{
				Approved: true,
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "vehicle not found",
		},
		{
			name: "error - vehicle not pending",
			req: &AdminReviewRequest{
				Approved: true,
			},
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
					Status:   VehicleStatusApproved,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "vehicle is not pending review",
		},
		{
			name: "error - cannot review rejected vehicle",
			req: &AdminReviewRequest{
				Approved: true,
			},
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{
					ID:       vehicleID,
					DriverID: driverID,
					Status:   VehicleStatusRejected,
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "vehicle is not pending review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.ReviewVehicle(context.Background(), vehicleID, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetPendingReviews (Admin)
// ========================================

func TestGetPendingReviews(t *testing.T) {
	tests := []struct {
		name       string
		limit      int
		offset     int
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, vehicles []Vehicle, total int)
	}{
		{
			name:   "success - returns pending vehicles",
			limit:  10,
			offset: 0,
			setupMocks: func(m *mockRepo) {
				vehicles := []Vehicle{
					{ID: uuid.New(), Make: "Toyota", Status: VehicleStatusPending},
					{ID: uuid.New(), Make: "Honda", Status: VehicleStatusPending},
				}
				m.On("GetPendingReviewVehicles", mock.Anything, 10, 0).Return(vehicles, 2, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, vehicles []Vehicle, total int) {
				assert.Len(t, vehicles, 2)
				assert.Equal(t, 2, total)
			},
		},
		{
			name:   "success - invalid limit defaults to 20",
			limit:  0,
			offset: 0,
			setupMocks: func(m *mockRepo) {
				m.On("GetPendingReviewVehicles", mock.Anything, 20, 0).Return([]Vehicle{}, 0, nil)
			},
			wantErr: false,
		},
		{
			name:   "success - negative limit defaults to 20",
			limit:  -5,
			offset: 0,
			setupMocks: func(m *mockRepo) {
				m.On("GetPendingReviewVehicles", mock.Anything, 20, 0).Return([]Vehicle{}, 0, nil)
			},
			wantErr: false,
		},
		{
			name:   "success - limit over 50 defaults to 20",
			limit:  100,
			offset: 0,
			setupMocks: func(m *mockRepo) {
				m.On("GetPendingReviewVehicles", mock.Anything, 20, 0).Return([]Vehicle{}, 0, nil)
			},
			wantErr: false,
		},
		{
			name:   "success - negative offset defaults to 0",
			limit:  10,
			offset: -5,
			setupMocks: func(m *mockRepo) {
				m.On("GetPendingReviewVehicles", mock.Anything, 10, 0).Return([]Vehicle{}, 0, nil)
			},
			wantErr: false,
		},
		{
			name:   "success - pagination with offset",
			limit:  10,
			offset: 20,
			setupMocks: func(m *mockRepo) {
				m.On("GetPendingReviewVehicles", mock.Anything, 10, 20).Return([]Vehicle{}, 50, nil)
			},
			wantErr: false,
		},
		{
			name:   "error - database failure",
			limit:  10,
			offset: 0,
			setupMocks: func(m *mockRepo) {
				m.On("GetPendingReviewVehicles", mock.Anything, 10, 0).Return(nil, 0, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			vehicles, total, err := svc.GetPendingReviews(context.Background(), tt.limit, tt.offset)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, vehicles, total)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetVehicleStats (Admin)
// ========================================

func TestGetVehicleStats(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, stats *VehicleStats)
	}{
		{
			name: "success - returns stats",
			setupMocks: func(m *mockRepo) {
				stats := &VehicleStats{
					TotalVehicles:        100,
					PendingReview:        5,
					ApprovedVehicles:     85,
					SuspendedVehicles:    10,
					ExpiringInsurance:    3,
					ExpiringRegistration: 2,
				}
				m.On("GetVehicleStats", mock.Anything).Return(stats, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, stats *VehicleStats) {
				assert.Equal(t, 100, stats.TotalVehicles)
				assert.Equal(t, 5, stats.PendingReview)
				assert.Equal(t, 85, stats.ApprovedVehicles)
				assert.Equal(t, 10, stats.SuspendedVehicles)
				assert.Equal(t, 3, stats.ExpiringInsurance)
				assert.Equal(t, 2, stats.ExpiringRegistration)
			},
		},
		{
			name: "error - database failure",
			setupMocks: func(m *mockRepo) {
				m.On("GetVehicleStats", mock.Anything).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			stats, err := svc.GetVehicleStats(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, stats)
			} else {
				require.NoError(t, err)
				require.NotNil(t, stats)
				if tt.validate != nil {
					tt.validate(t, stats)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: SuspendVehicle (Admin)
// ========================================

func TestSuspendVehicle(t *testing.T) {
	vehicleID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name       string
		reason     string
		setupMocks func(m *mockRepo)
		wantErr    bool
	}{
		{
			name:   "success - suspend vehicle",
			reason: "Safety violation",
			setupMocks: func(m *mockRepo) {
				m.On("UpdateVehicleStatus", mock.Anything, vehicleID, VehicleStatusSuspended, mock.AnythingOfType("*string")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "error - database failure",
			reason: "Safety violation",
			setupMocks: func(m *mockRepo) {
				m.On("UpdateVehicleStatus", mock.Anything, vehicleID, VehicleStatusSuspended, mock.AnythingOfType("*string")).Return(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.SuspendVehicle(context.Background(), vehicleID, tt.reason)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetExpiringDocuments (Admin)
// ========================================

func TestGetExpiringDocuments(t *testing.T) {
	tests := []struct {
		name       string
		daysAhead  int
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, vehicles []Vehicle)
	}{
		{
			name:      "success - returns expiring vehicles",
			daysAhead: 30,
			setupMocks: func(m *mockRepo) {
				vehicles := []Vehicle{
					{ID: uuid.New(), Make: "Toyota"},
					{ID: uuid.New(), Make: "Honda"},
				}
				m.On("GetExpiringVehicles", mock.Anything, 30).Return(vehicles, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, vehicles []Vehicle) {
				assert.Len(t, vehicles, 2)
			},
		},
		{
			name:      "success - daysAhead defaults to 30",
			daysAhead: 0,
			setupMocks: func(m *mockRepo) {
				m.On("GetExpiringVehicles", mock.Anything, 30).Return([]Vehicle{}, nil)
			},
			wantErr: false,
		},
		{
			name:      "success - negative daysAhead defaults to 30",
			daysAhead: -10,
			setupMocks: func(m *mockRepo) {
				m.On("GetExpiringVehicles", mock.Anything, 30).Return([]Vehicle{}, nil)
			},
			wantErr: false,
		},
		{
			name:      "success - empty result returns empty slice not nil",
			daysAhead: 30,
			setupMocks: func(m *mockRepo) {
				m.On("GetExpiringVehicles", mock.Anything, 30).Return(nil, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, vehicles []Vehicle) {
				assert.NotNil(t, vehicles)
				assert.Len(t, vehicles, 0)
			},
		},
		{
			name:      "error - database failure",
			daysAhead: 30,
			setupMocks: func(m *mockRepo) {
				m.On("GetExpiringVehicles", mock.Anything, 30).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			vehicles, err := svc.GetExpiringDocuments(context.Background(), tt.daysAhead)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, vehicles)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetMaintenanceReminders
// ========================================

func TestGetMaintenanceReminders(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	otherDriverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	vehicleID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, reminders []MaintenanceReminder)
	}{
		{
			name: "success - returns reminders",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{ID: vehicleID, DriverID: driverID}
				reminders := []MaintenanceReminder{
					{ID: uuid.New(), Type: "oil_change"},
					{ID: uuid.New(), Type: "tire_rotation"},
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
				m.On("GetPendingReminders", mock.Anything, vehicleID).Return(reminders, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, reminders []MaintenanceReminder) {
				assert.Len(t, reminders, 2)
			},
		},
		{
			name: "success - empty reminders returns empty slice",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{ID: vehicleID, DriverID: driverID}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
				m.On("GetPendingReminders", mock.Anything, vehicleID).Return(nil, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, reminders []MaintenanceReminder) {
				assert.NotNil(t, reminders)
				assert.Len(t, reminders, 0)
			},
		},
		{
			name: "error - vehicle not found",
			setupMocks: func(m *mockRepo) {
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "vehicle not found",
		},
		{
			name: "error - not owner",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{ID: vehicleID, DriverID: otherDriverID}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			reminders, err := svc.GetMaintenanceReminders(context.Background(), vehicleID, driverID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, reminders)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: CompleteMaintenance
// ========================================

func TestCompleteMaintenance(t *testing.T) {
	reminderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
	}{
		{
			name: "success - complete reminder",
			setupMocks: func(m *mockRepo) {
				m.On("CompleteReminder", mock.Anything, reminderID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - database failure",
			setupMocks: func(m *mockRepo) {
				m.On("CompleteReminder", mock.Anything, reminderID).Return(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.CompleteMaintenance(context.Background(), reminderID)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetInspections
// ========================================

func TestGetInspections(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	otherDriverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	vehicleID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, inspections []VehicleInspection)
	}{
		{
			name: "success - returns inspections",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{ID: vehicleID, DriverID: driverID}
				inspections := []VehicleInspection{
					{ID: uuid.New(), Status: "passed"},
					{ID: uuid.New(), Status: "scheduled"},
				}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
				m.On("GetInspectionsByVehicle", mock.Anything, vehicleID).Return(inspections, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, inspections []VehicleInspection) {
				assert.Len(t, inspections, 2)
			},
		},
		{
			name: "success - empty inspections returns empty slice",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{ID: vehicleID, DriverID: driverID}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
				m.On("GetInspectionsByVehicle", mock.Anything, vehicleID).Return(nil, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, inspections []VehicleInspection) {
				assert.NotNil(t, inspections)
				assert.Len(t, inspections, 0)
			},
		},
		{
			name: "error - vehicle not found",
			setupMocks: func(m *mockRepo) {
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "vehicle not found",
		},
		{
			name: "error - not owner",
			setupMocks: func(m *mockRepo) {
				vehicle := &Vehicle{ID: vehicleID, DriverID: otherDriverID}
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			inspections, err := svc.GetInspections(context.Background(), vehicleID, driverID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, inspections)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: Vehicle Status Transitions
// ========================================

func TestVehicleStatusTransitions(t *testing.T) {
	vehicleID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	t.Run("pending to approved", func(t *testing.T) {
		m := new(mockRepo)
		vehicle := &Vehicle{ID: vehicleID, DriverID: driverID, Status: VehicleStatusPending}
		m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
		m.On("UpdateVehicleStatus", mock.Anything, vehicleID, VehicleStatusApproved, (*string)(nil)).Return(nil)

		svc := newTestService(m)
		err := svc.ReviewVehicle(context.Background(), vehicleID, &AdminReviewRequest{Approved: true})

		require.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("pending to rejected", func(t *testing.T) {
		m := new(mockRepo)
		vehicle := &Vehicle{ID: vehicleID, DriverID: driverID, Status: VehicleStatusPending}
		m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
		m.On("UpdateVehicleStatus", mock.Anything, vehicleID, VehicleStatusRejected, mock.AnythingOfType("*string")).Return(nil)

		svc := newTestService(m)
		err := svc.ReviewVehicle(context.Background(), vehicleID, &AdminReviewRequest{
			Approved:        false,
			RejectionReason: stringPtr("Documents not clear"),
		})

		require.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("approved cannot be re-approved", func(t *testing.T) {
		m := new(mockRepo)
		vehicle := &Vehicle{ID: vehicleID, DriverID: driverID, Status: VehicleStatusApproved}
		m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)

		svc := newTestService(m)
		err := svc.ReviewVehicle(context.Background(), vehicleID, &AdminReviewRequest{Approved: true})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "vehicle is not pending review")
		m.AssertExpectations(t)
	})

	t.Run("suspended vehicle cannot be set as primary", func(t *testing.T) {
		m := new(mockRepo)
		vehicle := &Vehicle{ID: vehicleID, DriverID: driverID, Status: VehicleStatusSuspended}
		m.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)

		svc := newTestService(m)
		err := svc.SetPrimary(context.Background(), vehicleID, driverID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "only approved vehicles can be set as primary")
		m.AssertExpectations(t)
	})
}

// ========================================
// TESTS: Max Vehicle Limit Edge Cases
// ========================================

func TestMaxVehicleLimitEdgeCases(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	t.Run("exactly at limit - should fail", func(t *testing.T) {
		m := new(mockRepo)
		existingVehicles := []Vehicle{
			{ID: uuid.New(), IsActive: true},
			{ID: uuid.New(), IsActive: true},
			{ID: uuid.New(), IsActive: true},
		}
		m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return(existingVehicles, int64(len(existingVehicles)), nil)

		svc := newTestService(m)
		_, err := svc.RegisterVehicle(context.Background(), driverID, &RegisterVehicleRequest{
			Make:         "New",
			Model:        "Car",
			Year:         2022,
			Color:        "Red",
			LicensePlate: "NEW001",
			Category:     VehicleCategoryEconomy,
			FuelType:     FuelTypeGasoline,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "maximum 3 active vehicles allowed")
		m.AssertExpectations(t)
	})

	t.Run("one below limit - should succeed", func(t *testing.T) {
		m := new(mockRepo)
		existingVehicles := []Vehicle{
			{ID: uuid.New(), IsActive: true},
			{ID: uuid.New(), IsActive: true},
		}
		m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return(existingVehicles, int64(len(existingVehicles)), nil)
		m.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)

		svc := newTestService(m)
		vehicle, err := svc.RegisterVehicle(context.Background(), driverID, &RegisterVehicleRequest{
			Make:         "New",
			Model:        "Car",
			Year:         2022,
			Color:        "Red",
			LicensePlate: "NEW001",
			Category:     VehicleCategoryEconomy,
			FuelType:     FuelTypeGasoline,
		})

		require.NoError(t, err)
		require.NotNil(t, vehicle)
		m.AssertExpectations(t)
	})

	t.Run("mixed active and inactive - only counts active", func(t *testing.T) {
		m := new(mockRepo)
		existingVehicles := []Vehicle{
			{ID: uuid.New(), IsActive: true},
			{ID: uuid.New(), IsActive: true},
			{ID: uuid.New(), IsActive: false}, // retired
			{ID: uuid.New(), IsActive: false}, // retired
			{ID: uuid.New(), IsActive: false}, // retired
		}
		m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return(existingVehicles, int64(len(existingVehicles)), nil)
		m.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)

		svc := newTestService(m)
		vehicle, err := svc.RegisterVehicle(context.Background(), driverID, &RegisterVehicleRequest{
			Make:         "New",
			Model:        "Car",
			Year:         2022,
			Color:        "Red",
			LicensePlate: "NEW001",
			Category:     VehicleCategoryEconomy,
			FuelType:     FuelTypeGasoline,
		})

		require.NoError(t, err)
		require.NotNil(t, vehicle)
		m.AssertExpectations(t)
	})
}

// ========================================
// TESTS: Year Validation Boundaries
// ========================================

func TestYearValidationBoundaries(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	currentYear := time.Now().Year()

	tests := []struct {
		name    string
		year    int
		wantErr bool
	}{
		{
			name:    "minimum year (2010) - should succeed",
			year:    2010,
			wantErr: false,
		},
		{
			name:    "below minimum year (2009) - should fail",
			year:    2009,
			wantErr: true,
		},
		{
			name:    "current year - should succeed",
			year:    currentYear,
			wantErr: false,
		},
		{
			name:    "next year - should succeed",
			year:    currentYear + 1,
			wantErr: false,
		},
		{
			name:    "two years ahead - should fail",
			year:    currentYear + 2,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			if !tt.wantErr {
				m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
				m.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)
			}

			svc := newTestService(m)
			_, err := svc.RegisterVehicle(context.Background(), driverID, &RegisterVehicleRequest{
				Make:         "Test",
				Model:        "Car",
				Year:         tt.year,
				Color:        "Blue",
				LicensePlate: "TEST001",
				Category:     VehicleCategoryEconomy,
				FuelType:     FuelTypeGasoline,
			})

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "vehicle year must be between")
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}
