package profile

import "errors"

var (
	ErrProfileNotFound      = errors.New("profile not found")
	ErrProfileAlreadyExists = errors.New("profile already exists")
	ErrNotProfileOwner      = errors.New("not profile owner")
	ErrInvalidProfileType   = errors.New("invalid profile type for user role")
)
