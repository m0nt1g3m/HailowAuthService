package auth

import (
	"HailowAuthService/internal/domain"
	"context"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func (u *AuthUseCase) generateTokenPair(ctx context.Context, user *domain.User) (*domain.TokenPair, error) {
	now := time.Now()

	accessClaims := jwtClaims{
		ID:        uuid.MustParse(user.ID),
		Email:     user.Email,
		Role:      string(user.Role),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	accessTokenStr, err := u.generateAccessToken(accessClaims)
	if err != nil {
		return nil, err
	}

	refreshClaims := jwtClaims{
		ID:        uuid.MustParse(user.ID),
		Email:     user.Email,
		Role:      string(user.Role),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	refreshTokenStr, err := u.generateRefreshTokenWithClaims(refreshClaims)
	if err != nil {
		return nil, err
	}

	session := &domain.RefreshSession{
		UserID:       uuid.MustParse(user.ID),
		RefreshToken: refreshTokenStr,
		CreatedAt:    now,
		ExpiresAt:    now.Add(24 * time.Hour),
	}

	if err := u.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshTokenStr,
	}, nil
}

func (u *AuthUseCase) generateAccessTokenOnly(user *domain.User, refreshToken string) (*domain.TokenPair, error) {
	now := time.Now()
	claims := jwtClaims{
		ID:        uuid.MustParse(user.ID),
		Email:     user.Email,
		Role:      string(user.Role),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	accessTokenStr, err := u.generateAccessToken(claims)
	if err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshToken,
	}, nil
}

func (u *AuthUseCase) generateAccessToken(claims jwtClaims) (string, error) {
	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return accessTokenObj.SignedString(u.getAccessSigningKey())
}

func (u *AuthUseCase) generateRefreshTokenWithClaims(claims jwtClaims) (string, error) {
	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return refreshTokenObj.SignedString(u.getRefreshSigningKey())
}

func (u *AuthUseCase) getAccessSigningKey() []byte {
	if key := os.Getenv("JWT_ACCESS_KEY_AUTH_SERVICE"); key != "" {
		return []byte(key)
	}
	return []byte("secret")
}

func (u *AuthUseCase) getRefreshSigningKey() []byte {
	if key := os.Getenv("JWT_REFRESH_KEY_AUTH_SERVICE"); key != "" {
		return []byte(key)
	}
	return []byte("secret")
}

func (u *AuthUseCase) parseAccessToken(tokenStr string) (*jwtClaims, error) {
	var claims jwtClaims
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrUnauthorized
		}
		return u.getAccessSigningKey(), nil
	})

	if err != nil || !token.Valid {
		return nil, domain.ErrUnauthorized
	}

	return &claims, nil
}
