package booking

import "errors"

var (
	ErrValidation   = errors.New("validation error")
	ErrNotAvailable = errors.New("booking not available")
	ErrOverbooking  = errors.New("overbooking constraint violation")
)
