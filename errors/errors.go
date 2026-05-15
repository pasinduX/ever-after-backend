package errors

import "errors"

// Auth domain errors
var (
	ErrEmailTaken              = errors.New("email already in use")
	ErrInvalidCreds            = errors.New("invalid email or password")
	ErrInvalidToken            = errors.New("invalid or expired refresh token")
	ErrEmailNotVerified        = errors.New("email not verified")
	ErrInvalidVerificationCode = errors.New("invalid or expired verification code")
	ErrAlreadyVerified         = errors.New("email already verified")
)

// Wedding domain errors
var (
	ErrWeddingNotFound = errors.New("wedding not found")
	ErrForbidden       = errors.New("access denied")
)

// Upload domain errors
var (
	ErrUploadNotFound = errors.New("upload not found")
	ErrLimitReached   = errors.New("upload limit reached for this tier")
	ErrInvalidFile    = errors.New("invalid file type")
)

// Order domain errors
var (
	ErrInvalidTier = errors.New("invalid tier")
)
