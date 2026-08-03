package domain

import "errors"

var (
	ErrUserAlreadyExists       = errors.New("User already exists")
	ErrUserNotFound            = errors.New("User not found")
	ErrInvalidCredentials      = errors.New("Invalid credentials")
	ErrTokenNotFound           = errors.New("Token not found")
	ErrSessionNotFound         = errors.New("Session not found")
	ErrUnauthorized            = errors.New("Unauthorized")
	ErrAvatarImageEmpty        = errors.New("Avatar image is empty")
	ErrAvatarUploadFailed      = errors.New("Failed to upload avatar")
	ErrAvatarUploadUnavailable = errors.New("Avatar upload is unavailable")
)
