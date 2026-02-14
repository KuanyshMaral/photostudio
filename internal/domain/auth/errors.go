package auth

import "errors"

var (
	ErrInvalidCredentials            = errors.New("invalid credentials")
	ErrEmailAlreadyExists            = errors.New("email already exists")
	ErrUnauthorized                  = errors.New("unauthorized")
	ErrRateLimitExceeded             = errors.New("rate limit exceeded")
	ErrInvalidVerificationCode       = errors.New("invalid verification code")
	ErrInvalidVerificationCodeFormat = errors.New("invalid verification code format")
	ErrTooManyAttempts               = errors.New("too many attempts")
	ErrAccountLocked                 = errors.New("account locked")
	ErrAccountBanned                 = errors.New("account banned")
	ErrEmailNotVerified              = errors.New("email not verified")
	ErrInvalidRefreshToken           = errors.New("invalid refresh token")
	ErrRefreshTokenReused            = errors.New("refresh token reused")
)