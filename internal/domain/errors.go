package domain

import "errors"

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenNotFound      = errors.New("token not found")
	ErrSessionNotFound    = errors.New("session not found")
	ErrUnauthorized       = errors.New("unauthorized")
)
