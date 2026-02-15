package lead

import "errors"

var (
	ErrLeadNotFound     = errors.New("lead not found")
	ErrAlreadyConverted = errors.New("lead already converted")
	ErrCannotConvert    = errors.New("lead cannot be converted in current status")
	ErrEmailExists      = errors.New("email already exists")
)
