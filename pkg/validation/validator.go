package validation

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var (
	// Validate is the global validator instance
	Validate *validator.Validate

	// Common regex patterns
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`) // E.164 format
)

func init() {
	Validate = validator.New()

	// Register custom validators
	_ = Validate.RegisterValidation("latitude", validateLatitude)
	_ = Validate.RegisterValidation("longitude", validateLongitude)
	_ = Validate.RegisterValidation("phone", validatePhone)
	_ = Validate.RegisterValidation("future", validateFutureTime)
	_ = Validate.RegisterValidation("past", validatePastTime)
	_ = Validate.RegisterValidation("ride_status", validateRideStatus)
	_ = Validate.RegisterValidation("payment_method", validatePaymentMethod)
	_ = Validate.RegisterValidation("user_role", validateUserRole)
	_ = Validate.RegisterValidation("vehicle_year", validateVehicleYear)
}

// ValidateStruct validates a struct and returns a ValidationError if validation fails
func ValidateStruct(s interface{}) error {
	err := Validate.Struct(s)
	if err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return NewValidationError(validationErrors)
		}
		return err
	}
	return nil
}

// validateLatitude checks if latitude is within valid range (-90 to 90)
func validateLatitude(fl validator.FieldLevel) bool {
	latitude := fl.Field().Float()
	return latitude >= -90.0 && latitude <= 90.0
}

// validateLongitude checks if longitude is within valid range (-180 to 180)
func validateLongitude(fl validator.FieldLevel) bool {
	longitude := fl.Field().Float()
	return longitude >= -180.0 && longitude <= 180.0
}

// validatePhone checks if phone number is in E.164 format
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	return phoneRegex.MatchString(phone)
}

// validateFutureTime checks if time is in the future
func validateFutureTime(fl validator.FieldLevel) bool {
	// This is a placeholder - implement time validation in actual usage
	// You'll need to handle *time.Time fields specifically
	return true
}

// validatePastTime checks if time is in the past
func validatePastTime(fl validator.FieldLevel) bool {
	// This is a placeholder - implement time validation in actual usage
	return true
}

// validateRideStatus checks if ride status is valid
func validateRideStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := []string{"requested", "accepted", "in_progress", "completed", "cancelled"}
	return contains(validStatuses, status)
}

// validatePaymentMethod checks if payment method is valid
func validatePaymentMethod(fl validator.FieldLevel) bool {
	method := fl.Field().String()
	validMethods := []string{"card", "wallet", "cash"}
	return contains(validMethods, method)
}

// validateUserRole checks if user role is valid
func validateUserRole(fl validator.FieldLevel) bool {
	role := fl.Field().String()
	validRoles := []string{"rider", "driver", "admin"}
	return contains(validRoles, role)
}

// validateVehicleYear checks if vehicle year is reasonable
func validateVehicleYear(fl validator.FieldLevel) bool {
	year := fl.Field().Int()
	currentYear := int64(time.Now().Year())
	return year >= 1900 && year <= currentYear+1
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	item = strings.ToLower(strings.TrimSpace(item))
	for _, s := range slice {
		if strings.ToLower(strings.TrimSpace(s)) == item {
			return true
		}
	}
	return false
}

// ValidateEmail validates email format
func ValidateEmail(email string) bool {
	email = strings.TrimSpace(email)
	return len(email) > 0 && emailRegex.MatchString(email)
}

// ValidatePhoneNumber validates phone number format
func ValidatePhoneNumber(phone string) bool {
	phone = strings.TrimSpace(phone)
	return phoneRegex.MatchString(phone)
}

// ValidateCoordinates validates latitude and longitude
func ValidateCoordinates(latitude, longitude float64) error {
	if latitude < -90.0 || latitude > 90.0 {
		return fmt.Errorf("latitude must be between -90 and 90, got: %f", latitude)
	}
	if longitude < -180.0 || longitude > 180.0 {
		return fmt.Errorf("longitude must be between -180 and 180, got: %f", longitude)
	}
	return nil
}

// ValidateDistance validates distance value (in km or miles)
func ValidateDistance(distance float64) error {
	if distance < 0 {
		return fmt.Errorf("distance cannot be negative: %f", distance)
	}
	if distance > 10000 { // Max 10,000 km seems reasonable
		return fmt.Errorf("distance exceeds maximum allowed: %f", distance)
	}
	return nil
}

// ValidateAmount validates monetary amount
func ValidateAmount(amount float64) error {
	if amount < 0 {
		return fmt.Errorf("amount cannot be negative: %f", amount)
	}
	if amount > 100000 { // Max $100,000 per transaction
		return fmt.Errorf("amount exceeds maximum allowed: %f", amount)
	}
	return nil
}

// ValidateRating validates rating value (1-5)
func ValidateRating(rating int) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5, got: %d", rating)
	}
	return nil
}

// ValidateStringLength validates string length
func ValidateStringLength(s string, min, max int) error {
	length := len(strings.TrimSpace(s))
	if length < min {
		return fmt.Errorf("string length must be at least %d characters, got: %d", min, length)
	}
	if max > 0 && length > max {
		return fmt.Errorf("string length must be at most %d characters, got: %d", max, length)
	}
	return nil
}

// ValidateUUID validates UUID format
func ValidateUUID(uuid string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidRegex.MatchString(uuid)
}
