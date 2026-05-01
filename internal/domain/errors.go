package domain

import "errors"

// Sentinel errors let callers use errors.Is() for precise error handling
// without importing infrastructure packages.
var (
	// User errors
	ErrUserNotFound       = errors.New("user not found")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUsernameEmpty      = errors.New("username must not be empty")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
)
