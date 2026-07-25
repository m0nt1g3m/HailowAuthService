package auth

type SignUpRequest struct {
	Email    string
	Password string
}

type SignInRequest struct {
	Email    string
	Password string
}

type RefreshTokensRequest struct {
	RefreshToken string
}

type LogoutRequest struct {
	RefreshToken string
}

type ValidateTokenRequest struct {
	AccessToken string
}

type GetUserSessionsRequest struct {
	UserID string
}

type RevokeSessionRequest struct {
	UserID    string
	SessionID string
}
