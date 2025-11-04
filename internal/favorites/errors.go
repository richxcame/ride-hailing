package favorites

import "errors"

var (
	// ErrInvalidName is returned when the favorite name is invalid
	ErrInvalidName = errors.New("invalid favorite location name")

	// ErrInvalidAddress is returned when the address is invalid
	ErrInvalidAddress = errors.New("invalid address")

	// ErrInvalidCoordinates is returned when coordinates are invalid
	ErrInvalidCoordinates = errors.New("invalid coordinates")

	// ErrUnauthorized is returned when user tries to access another user's favorite
	ErrUnauthorized = errors.New("unauthorized access to favorite location")
)
