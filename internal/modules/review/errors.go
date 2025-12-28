package review

import "errors"

var (
	ErrInvalidRequest   = errors.New("invalid_request")
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not_found")
	ErrConflict         = errors.New("conflict")
	ErrReviewNotAllowed = errors.New("review_not_allowed")
)
