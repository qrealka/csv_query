package numeric

import "errors"

var (
	// ErrInvalidType is returned when an invalid type conversion is attempted
	ErrInvalidType = errors.New("invalid type conversion")
)
