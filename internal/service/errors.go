package service

import "errors"

var (
	// ErrNotFound is returned when a resource is not found
	ErrNotFound = errors.New("resource not found")

	// ErrForbidden is returned when access is denied
	ErrForbidden = errors.New("access forbidden")

	// ErrUnregisterWindowExpired is returned when trying to unregister outside the allowed window
	ErrUnregisterWindowExpired = errors.New("unregister window has expired")

	// ErrInvalidCredentials is returned when authentication fails
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrRateLimitExceeded is returned when rate limit is exceeded
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrFunderNotActive is returned when funder account is deactivated
	ErrFunderNotActive = errors.New("funder account is not active")
)
