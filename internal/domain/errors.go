package domain

import "errors"

var (
	ErrUserAlreadyExists  = errors.New("User already exists")
	ErrUserNotFound       = errors.New("User not found")
	ErrInvalidCredentials = errors.New("Invalid credentials")
	ErrTokenNotFound      = errors.New("Token not found")
	ErrSessionNotFound    = errors.New("Session not found")
	ErrUnauthorized       = errors.New("Unauthorized")
)
