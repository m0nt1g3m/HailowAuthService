package domain

import (
	"fmt"
)

type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *AppError) Unwrap() error { return e.Err }

func (e *AppError) Is(target error) bool {
	if t, ok := target.(*AppError); ok {
		return e.Code == t.Code
	}
	return false
}

var (
	ErrUserAlreadyExists       = &AppError{Code: "user_already_exists", Message: "User already exists"}
	ErrUserNotFound            = &AppError{Code: "user_not_found", Message: "User not found"}
	ErrInvalidCredentials      = &AppError{Code: "invalid_credentials", Message: "Invalid credentials"}
	ErrTokenNotFound           = &AppError{Code: "refresh_token_not_found", Message: "refresh_token not found"}
	ErrSessionNotFound         = &AppError{Code: "session_not_found", Message: "Session not found"}
	ErrUnauthorized            = &AppError{Code: "unauthorized", Message: "Unauthorized"}
	ErrAvatarImageEmpty        = &AppError{Code: "avatar_image_empty", Message: "Avatar image is empty"}
	ErrAvatarUploadFailed      = &AppError{Code: "avatar_upload_failed", Message: "Failed to upload avatar"}
	ErrAvatarUploadUnavailable = &AppError{Code: "avatar_upload_unavailable", Message: "Avatar upload is unavailable"}
	ErrInvalidRequest          = &AppError{Code: "invalid_auth_request", Message: "Invalid auth request"}
	ErrRefreshTokenMD          = &AppError{Code: "refresh_token_not_found_md", Message: "refresh-token not found in metadata"}
	ErrUserIDNotFoundMD        = &AppError{Code: "user_id_not_found_md", Message: "user-id not found in metadata"}
	ErrPermissionDenied        = &AppError{Code: "permission_denied", Message: "Permission denied"}
	ErrMDNotFound              = &AppError{Code: "metadata_not_found", Message: "Metadata not found"}
)
