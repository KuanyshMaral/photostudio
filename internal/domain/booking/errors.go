package booking

import "errors"

var (
	ErrValidation              = errors.New("validation error")
	ErrNotAvailable            = errors.New("booking not available")
	ErrOverbooking             = errors.New("overbooking constraint violation")
	ErrStudioClosed            = errors.New("studio is closed on this date")
	ErrOutsideWorkingHours     = errors.New("booking outside of working hours")
	ErrForbidden               = errors.New("forbidden")
	ErrInvalidStatusTransition = errors.New("invalid_status_transition")
	ErrNotFound                = errors.New("not_found")
	ErrActivePreBookingExists  = errors.New("active_pre_booking_exists")
	ErrPreBookingConflict      = errors.New("pre_booking_conflict")
	ErrInvalidPreBookingStatus = errors.New("invalid_pre_booking_status")
)
