package booking

import "errors"

var (
	ErrValidation              = errors.New("validation error")
	ErrNotAvailable            = errors.New("booking not available")
	ErrOverbooking             = errors.New("overbooking constraint violation")
	ErrForbidden               = errors.New("forbidden")
	ErrInvalidStatusTransition = errors.New("invalid_status_transition")
	ErrNotFound                = errors.New("not_found")
)
