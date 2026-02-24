package upload

import "errors"

var (
	ErrUploadNotFound  = errors.New("upload not found")
	ErrNotOwner        = errors.New("you do not own this upload")
	ErrFileTooLarge    = errors.New("file exceeds maximum allowed size")
	ErrInvalidMimeType = errors.New("file type is not allowed")
	ErrEmptyFile       = errors.New("file is empty")
)
