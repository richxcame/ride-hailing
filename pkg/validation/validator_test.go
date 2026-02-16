package validation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ValidateEmail
// ---------------------------------------------------------------------------

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name   string
		email  string
		expect bool
	}{
		{"valid simple", "user@example.com", true},
		{"valid with dots", "first.last@example.com", true},
		{"valid with plus", "user+tag@example.com", true},
		{"valid with hyphen domain", "user@my-domain.com", true},
		{"valid subdomain", "user@sub.domain.com", true},
		{"valid with digits", "user123@example.com", true},
		{"valid with percent", "user%tag@example.com", true},
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"missing at sign", "userexample.com", false},
		{"missing domain", "user@", false},
		{"missing local part", "@example.com", false},
		{"double at", "user@@example.com", false},
		{"no TLD", "user@example", false},
		{"spaces inside", "us er@example.com", false},
		{"with leading space trimmed", " user@example.com", true},
		{"with trailing space trimmed", "user@example.com ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, ValidateEmail(tt.email))
		})
	}
}

// ---------------------------------------------------------------------------
// ValidatePhoneNumber
// ---------------------------------------------------------------------------

func TestValidatePhoneNumber(t *testing.T) {
	tests := []struct {
		name   string
		phone  string
		expect bool
	}{
		{"valid E.164 with plus", "+14155552671", true},
		{"valid without plus", "14155552671", true},
		{"valid short", "+1234", true},
		{"valid max length", "+123456789012345", true},
		{"empty string", "", false},
		{"only plus", "+", false},
		{"starts with zero", "01234567890", false},
		{"letters included", "+1415abc2671", false},
		{"too long", "+1234567890123456", false},
		{"special characters", "+1-415-555-2671", false},
		{"spaces", "+1 415 555 2671", false},
		{"with leading space trimmed", " +14155552671", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, ValidatePhoneNumber(tt.phone))
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateCoordinates
// ---------------------------------------------------------------------------

func TestValidateCoordinates(t *testing.T) {
	tests := []struct {
		name      string
		latitude       float64
		longitude       float64
		expectErr bool
		errSubstr string
	}{
		{"valid origin", 0, 0, false, ""},
		{"valid NYC", 40.7128, -74.0060, false, ""},
		{"valid max latitude", 90, 0, false, ""},
		{"valid min latitude", -90, 0, false, ""},
		{"valid max longitude", 0, 180, false, ""},
		{"valid min longitude", 0, -180, false, ""},
		{"valid boundary corners", 90, 180, false, ""},
		{"lat too high", 90.1, 0, true, "latitude"},
		{"lat too low", -90.1, 0, true, "latitude"},
		{"lon too high", 0, 180.1, true, "longitude"},
		{"lon too low", 0, -180.1, true, "longitude"},
		{"both invalid", 100, 200, true, "latitude"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCoordinates(tt.latitude, tt.longitude)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateDistance
// ---------------------------------------------------------------------------

func TestValidateDistance(t *testing.T) {
	tests := []struct {
		name      string
		distance  float64
		expectErr bool
		errSubstr string
	}{
		{"zero", 0, false, ""},
		{"normal distance", 15.5, false, ""},
		{"max allowed", 10000, false, ""},
		{"negative", -1, true, "negative"},
		{"exceeds max", 10001, true, "exceeds maximum"},
		{"very small positive", 0.001, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDistance(tt.distance)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateAmount
// ---------------------------------------------------------------------------

func TestValidateAmount(t *testing.T) {
	tests := []struct {
		name      string
		amount    float64
		expectErr bool
		errSubstr string
	}{
		{"zero", 0, false, ""},
		{"normal amount", 25.99, false, ""},
		{"max allowed", 100000, false, ""},
		{"negative", -0.01, true, "negative"},
		{"exceeds max", 100001, true, "exceeds maximum"},
		{"small fraction", 0.01, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAmount(tt.amount)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateRating
// ---------------------------------------------------------------------------

func TestValidateRating(t *testing.T) {
	tests := []struct {
		name      string
		rating    int
		expectErr bool
	}{
		{"min valid", 1, false},
		{"mid valid", 3, false},
		{"max valid", 5, false},
		{"below min", 0, true},
		{"above max", 6, true},
		{"negative", -1, true},
		{"large number", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRating(tt.rating)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "rating must be between 1 and 5")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateStringLength
// ---------------------------------------------------------------------------

func TestValidateStringLength(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		min       int
		max       int
		expectErr bool
		errSubstr string
	}{
		{"valid within range", "hello", 1, 10, false, ""},
		{"exact min", "a", 1, 10, false, ""},
		{"exact max", "abcdefghij", 1, 10, false, ""},
		{"too short", "", 1, 10, true, "at least"},
		{"too long", "abcdefghijk", 1, 10, true, "at most"},
		{"zero max means no upper bound", "a very long string here", 1, 0, false, ""},
		{"whitespace trimmed", "  ab  ", 5, 10, true, "at least"},
		{"whitespace trimmed passes", "  abcde  ", 5, 10, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStringLength(tt.s, tt.min, tt.max)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateUUID
// ---------------------------------------------------------------------------

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name   string
		uuid   string
		expect bool
	}{
		{"valid v4", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid lowercase", "abcdef01-2345-6789-abcd-ef0123456789", true},
		{"valid uppercase", "ABCDEF01-2345-6789-ABCD-EF0123456789", true},
		{"valid mixed case", "AbCdEf01-2345-6789-aBcD-Ef0123456789", true},
		{"empty", "", false},
		{"no dashes", "550e8400e29b41d4a716446655440000", false},
		{"too short", "550e8400-e29b-41d4-a716", false},
		{"wrong format", "not-a-uuid-at-all", false},
		{"extra chars", "550e8400-e29b-41d4-a716-446655440000x", false},
		{"invalid chars", "550e840g-e29b-41d4-a716-446655440000", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, ValidateUUID(tt.uuid))
		})
	}
}

// ---------------------------------------------------------------------------
// ValidatePasswordStrength
// ---------------------------------------------------------------------------

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		expectErr bool
	}{
		{"valid 8 chars", "password", false},
		{"valid long", "averylongpasswordindeed123", false},
		{"too short 7 chars", "passwor", true},
		{"too short 1 char", "a", true},
		{"empty", "", true},
		{"exactly 8 chars", "12345678", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateDateRange
// ---------------------------------------------------------------------------

func TestValidateDateRange(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		start     time.Time
		end       time.Time
		expectErr bool
	}{
		{"end after start", now, now.Add(time.Hour), false},
		{"same time", now, now, false},
		{"end before start", now.Add(time.Hour), now, true},
		{"large gap", now, now.Add(365 * 24 * time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDateRange(tt.start, tt.end)
			if tt.expectErr {
				assert.Error(t, err)
				vErr, ok := err.(*ValidationError)
				require.True(t, ok)
				_, exists := vErr.GetFieldError("date_range")
				assert.True(t, exists)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidationError methods
// ---------------------------------------------------------------------------

func TestValidationError_Error(t *testing.T) {
	ve := &ValidationError{
		Errors: map[string]string{
			"email": "email is required",
		},
	}

	assert.Contains(t, ve.Error(), "email: email is required")
}

func TestValidationError_Error_MultipleFields(t *testing.T) {
	ve := &ValidationError{
		Errors: map[string]string{
			"email":    "email is required",
			"password": "password too short",
		},
	}

	errStr := ve.Error()
	assert.Contains(t, errStr, "email: email is required")
	assert.Contains(t, errStr, "password: password too short")
}

func TestValidationError_AddError(t *testing.T) {
	ve := &ValidationError{}
	ve.AddError("field1", "error1")

	assert.NotNil(t, ve.Errors)
	msg, exists := ve.GetFieldError("field1")
	assert.True(t, exists)
	assert.Equal(t, "error1", msg)
}

func TestValidationError_AddError_NilMap(t *testing.T) {
	ve := &ValidationError{Errors: nil}
	ve.AddError("field", "message")

	assert.NotNil(t, ve.Errors)
	assert.Equal(t, "message", ve.Errors["field"])
}

func TestValidationError_HasErrors(t *testing.T) {
	ve := &ValidationError{Errors: make(map[string]string)}
	assert.False(t, ve.HasErrors())

	ve.AddError("x", "y")
	assert.True(t, ve.HasErrors())
}

func TestValidationError_GetFieldError(t *testing.T) {
	ve := &ValidationError{
		Errors: map[string]string{"name": "name is required"},
	}

	msg, exists := ve.GetFieldError("name")
	assert.True(t, exists)
	assert.Equal(t, "name is required", msg)

	_, exists = ve.GetFieldError("missing")
	assert.False(t, exists)
}

// ---------------------------------------------------------------------------
// ValidateStruct – using real request types
// ---------------------------------------------------------------------------

func TestValidateStruct_LoginRequest_Valid(t *testing.T) {
	req := LoginRequest{
		Email:    "test@example.com",
		Password: "securepassword",
	}
	assert.NoError(t, ValidateStruct(&req))
}

func TestValidateStruct_LoginRequest_MissingEmail(t *testing.T) {
	req := LoginRequest{
		Password: "securepassword",
	}
	err := ValidateStruct(&req)
	assert.Error(t, err)

	vErr, ok := err.(*ValidationError)
	require.True(t, ok)
	_, exists := vErr.GetFieldError("Email")
	assert.True(t, exists)
}

func TestValidateStruct_LoginRequest_ShortPassword(t *testing.T) {
	req := LoginRequest{
		Email:    "test@example.com",
		Password: "short",
	}
	err := ValidateStruct(&req)
	assert.Error(t, err)

	vErr, ok := err.(*ValidationError)
	require.True(t, ok)
	_, exists := vErr.GetFieldError("Password")
	assert.True(t, exists)
}

func TestValidateStruct_RegisterUserRequest_Valid(t *testing.T) {
	req := RegisterUserRequest{
		Email:       "test@example.com",
		PhoneNumber: "+14155552671",
		Password:    "securepassword",
		FirstName:   "John",
		LastName:    "Doe",
		Role:        "rider",
	}
	assert.NoError(t, ValidateStruct(&req))
}

func TestValidateStruct_RegisterUserRequest_InvalidRole(t *testing.T) {
	req := RegisterUserRequest{
		Email:       "test@example.com",
		PhoneNumber: "+14155552671",
		Password:    "securepassword",
		FirstName:   "John",
		LastName:    "Doe",
		Role:        "superadmin",
	}
	err := ValidateStruct(&req)
	assert.Error(t, err)

	vErr, ok := err.(*ValidationError)
	require.True(t, ok)
	_, exists := vErr.GetFieldError("Role")
	assert.True(t, exists)
}

func TestValidateStruct_RegisterUserRequest_InvalidPhone(t *testing.T) {
	req := RegisterUserRequest{
		Email:       "test@example.com",
		PhoneNumber: "not-a-phone",
		Password:    "securepassword",
		FirstName:   "John",
		LastName:    "Doe",
		Role:        "rider",
	}
	err := ValidateStruct(&req)
	assert.Error(t, err)
}

func TestValidateStruct_UpdateLocationRequest_Valid(t *testing.T) {
	req := UpdateLocationRequest{
		Latitude:  40.7128,
		Longitude: -74.0060,
	}
	assert.NoError(t, ValidateStruct(&req))
}

func TestValidateStruct_UpdateLocationRequest_InvalidLatitude(t *testing.T) {
	req := UpdateLocationRequest{
		Latitude:  91.0,
		Longitude: -74.0060,
	}
	err := ValidateStruct(&req)
	assert.Error(t, err)
}

func TestValidateStruct_CreatePaymentRequest_Valid(t *testing.T) {
	req := CreatePaymentRequest{
		RideID: "550e8400-e29b-41d4-a716-446655440000",
		Amount: 25.50,
		Method: "card",
	}
	assert.NoError(t, ValidateStruct(&req))
}

func TestValidateStruct_CreatePaymentRequest_InvalidMethod(t *testing.T) {
	req := CreatePaymentRequest{
		RideID: "550e8400-e29b-41d4-a716-446655440000",
		Amount: 25.50,
		Method: "bitcoin",
	}
	err := ValidateStruct(&req)
	assert.Error(t, err)
}

func TestValidateStruct_UpdateRideStatusRequest_Valid(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"requested", "requested"},
		{"accepted", "accepted"},
		{"in_progress", "in_progress"},
		{"completed", "completed"},
		{"cancelled", "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := UpdateRideStatusRequest{Status: tt.status}
			assert.NoError(t, ValidateStruct(&req))
		})
	}
}

func TestValidateStruct_UpdateRideStatusRequest_InvalidStatus(t *testing.T) {
	req := UpdateRideStatusRequest{Status: "pending"}
	err := ValidateStruct(&req)
	assert.Error(t, err)
}

func TestValidateStruct_RatingRequest_Valid(t *testing.T) {
	req := RatingRequest{
		RideID: "550e8400-e29b-41d4-a716-446655440000",
		Rating: 5,
	}
	assert.NoError(t, ValidateStruct(&req))
}

func TestValidateStruct_RatingRequest_OutOfRange(t *testing.T) {
	tests := []struct {
		name   string
		rating int
	}{
		{"zero", 0},
		{"six", 6},
		{"negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := RatingRequest{
				RideID: "550e8400-e29b-41d4-a716-446655440000",
				Rating: tt.rating,
			}
			err := ValidateStruct(&req)
			assert.Error(t, err)
		})
	}
}

func TestValidateStruct_CreateDriverRequest_Valid(t *testing.T) {
	req := CreateDriverRequest{
		UserID:        "550e8400-e29b-41d4-a716-446655440000",
		LicenseNumber: "ABC12345",
		VehicleModel:  "Toyota Camry",
		VehiclePlate:  "ABC1234",
		VehicleColor:  "White",
		VehicleYear:   2023,
	}
	assert.NoError(t, ValidateStruct(&req))
}

func TestValidateStruct_CreateDriverRequest_InvalidYear(t *testing.T) {
	req := CreateDriverRequest{
		UserID:        "550e8400-e29b-41d4-a716-446655440000",
		LicenseNumber: "ABC12345",
		VehicleModel:  "Toyota Camry",
		VehiclePlate:  "ABC1234",
		VehicleColor:  "White",
		VehicleYear:   1800, // too old
	}
	err := ValidateStruct(&req)
	assert.Error(t, err)
}

func TestValidateStruct_PaginationRequest_Valid(t *testing.T) {
	req := PaginationRequest{
		Limit:   20,
		Offset:  0,
		SortBy:  "created",
		SortDir: "desc",
	}
	assert.NoError(t, ValidateStruct(&req))
}

func TestValidateStruct_PaginationRequest_InvalidSortDir(t *testing.T) {
	req := PaginationRequest{
		Limit:   20,
		Offset:  0,
		SortDir: "up",
	}
	err := ValidateStruct(&req)
	assert.Error(t, err)
}

func TestValidateStruct_PaginationRequest_LimitTooLarge(t *testing.T) {
	req := PaginationRequest{
		Limit:  101,
		Offset: 0,
	}
	err := ValidateStruct(&req)
	assert.Error(t, err)
}

func TestValidateStruct_WalletTopUpRequest_Valid(t *testing.T) {
	req := WalletTopUpRequest{
		Amount:  50.0,
		TokenID: "tok_abc123",
	}
	assert.NoError(t, ValidateStruct(&req))
}

func TestValidateStruct_WalletTopUpRequest_AmountTooHigh(t *testing.T) {
	req := WalletTopUpRequest{
		Amount:  10001,
		TokenID: "tok_abc123",
	}
	err := ValidateStruct(&req)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// ValidateRideRequest – business rules
// ---------------------------------------------------------------------------

func TestValidateRideRequest_Valid(t *testing.T) {
	req := &CreateRideRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:   -74.0060,
		PickupAddr:  "123 Main Street, New York",
		DropoffLatitude:  40.7580,
		DropoffLongitude:  -73.9855,
		DropoffAddr: "456 Broadway, New York",
		RideType:    "economy",
	}
	assert.NoError(t, ValidateRideRequest(req))
}

func TestValidateRideRequest_SamePickupDropoff(t *testing.T) {
	req := &CreateRideRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:   -74.0060,
		PickupAddr:  "123 Main Street, New York",
		DropoffLatitude:  40.7128,
		DropoffLongitude:  -74.0060,
		DropoffAddr: "123 Main Street, New York",
		RideType:    "economy",
	}
	err := ValidateRideRequest(req)
	assert.Error(t, err)

	vErr, ok := err.(*ValidationError)
	require.True(t, ok)
	_, exists := vErr.GetFieldError("location")
	assert.True(t, exists)
}

func TestValidateRideRequest_ScheduledInPast(t *testing.T) {
	pastTime := time.Now().Add(-time.Hour)
	req := &CreateRideRequest{
		PickupLatitude:    40.7128,
		PickupLongitude:    -74.0060,
		PickupAddr:   "123 Main Street, New York",
		DropoffLatitude:   40.7580,
		DropoffLongitude:   -73.9855,
		DropoffAddr:  "456 Broadway, New York",
		RideType:     "economy",
		ScheduledFor: &pastTime,
	}
	err := ValidateRideRequest(req)
	assert.Error(t, err)

	vErr, ok := err.(*ValidationError)
	require.True(t, ok)
	_, exists := vErr.GetFieldError("scheduled_for")
	assert.True(t, exists)
}

func TestValidateRideRequest_ScheduledInFuture(t *testing.T) {
	futureTime := time.Now().Add(2 * time.Hour)
	req := &CreateRideRequest{
		PickupLatitude:    40.7128,
		PickupLongitude:    -74.0060,
		PickupAddr:   "123 Main Street, New York",
		DropoffLatitude:   40.7580,
		DropoffLongitude:   -73.9855,
		DropoffAddr:  "456 Broadway, New York",
		RideType:     "economy",
		ScheduledFor: &futureTime,
	}
	assert.NoError(t, ValidateRideRequest(req))
}

func TestValidateRideRequest_InvalidRideType(t *testing.T) {
	req := &CreateRideRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:   -74.0060,
		PickupAddr:  "123 Main Street, New York",
		DropoffLatitude:  40.7580,
		DropoffLongitude:  -73.9855,
		DropoffAddr: "456 Broadway, New York",
		RideType:    "helicopter",
	}
	err := ValidateRideRequest(req)
	assert.Error(t, err)
}

func TestValidateRideRequest_MissingRequiredFields(t *testing.T) {
	req := &CreateRideRequest{}
	err := ValidateRideRequest(req)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// contains helper
// ---------------------------------------------------------------------------

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		slice  []string
		item   string
		expect bool
	}{
		{"found exact", []string{"a", "b", "c"}, "b", true},
		{"found case insensitive", []string{"Hello", "World"}, "hello", true},
		{"not found", []string{"a", "b"}, "c", false},
		{"empty slice", []string{}, "a", false},
		{"with whitespace", []string{"rider", "driver"}, " rider ", true},
		{"empty item", []string{"a"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, contains(tt.slice, tt.item))
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateStruct_CreateRideRequest – ride type validation
// ---------------------------------------------------------------------------

func TestValidateStruct_CreateRideRequest_RideTypes(t *testing.T) {
	validTypes := []string{"economy", "premium", "xl"}

	for _, rt := range validTypes {
		t.Run("valid_"+rt, func(t *testing.T) {
			req := CreateRideRequest{
				PickupLatitude:   40.7128,
				PickupLongitude:   -74.0060,
				PickupAddr:  "123 Main Street, New York",
				DropoffLatitude:  40.7580,
				DropoffLongitude:  -73.9855,
				DropoffAddr: "456 Broadway, New York",
				RideType:    rt,
			}
			assert.NoError(t, ValidateStruct(&req))
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateStruct_UpdateProfileRequest – optional fields
// ---------------------------------------------------------------------------

func TestValidateStruct_UpdateProfileRequest_AllEmpty(t *testing.T) {
	req := UpdateProfileRequest{}
	// All fields are omitempty, so an empty struct should pass
	assert.NoError(t, ValidateStruct(&req))
}

func TestValidateStruct_UpdateProfileRequest_ValidPhone(t *testing.T) {
	req := UpdateProfileRequest{
		PhoneNumber: "+14155552671",
	}
	assert.NoError(t, ValidateStruct(&req))
}

func TestValidateStruct_UpdateProfileRequest_InvalidURL(t *testing.T) {
	req := UpdateProfileRequest{
		ProfileImage: "not-a-url",
	}
	err := ValidateStruct(&req)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Payment method validation via struct tags
// ---------------------------------------------------------------------------

func TestValidateStruct_CreatePaymentRequest_ValidMethods(t *testing.T) {
	methods := []string{"card", "wallet", "cash"}
	for _, m := range methods {
		t.Run(m, func(t *testing.T) {
			req := CreatePaymentRequest{
				RideID: "550e8400-e29b-41d4-a716-446655440000",
				Amount: 10.0,
				Method: m,
			}
			assert.NoError(t, ValidateStruct(&req))
		})
	}
}

// ---------------------------------------------------------------------------
// User role validation via struct tags
// ---------------------------------------------------------------------------

func TestValidateStruct_RegisterUserRequest_ValidRoles(t *testing.T) {
	roles := []string{"rider", "driver", "admin"}
	for _, r := range roles {
		t.Run(r, func(t *testing.T) {
			req := RegisterUserRequest{
				Email:       "test@example.com",
				PhoneNumber: "+14155552671",
				Password:    "securepassword",
				FirstName:   "John",
				LastName:    "Doe",
				Role:        r,
			}
			assert.NoError(t, ValidateStruct(&req))
		})
	}
}
