package auth

import "HailowAuthService/internal/domain"

type SignUpResponse struct {
	User   *domain.User
	Tokens *domain.TokenPair
}

type SignInResponse struct {
	User   *domain.User
	Tokens *domain.TokenPair
}

type RefreshTokensResponse struct {
	Tokens *domain.TokenPair
}

type LogoutResponse struct {
	Success bool
}

type ValidateTokenResponse struct {
	User    *domain.User
	IsValid bool
}

type GetUserSessionsResponse struct {
	Sessions []*domain.Session
}

type RevokeSessionResponse struct {
	Success bool
}
