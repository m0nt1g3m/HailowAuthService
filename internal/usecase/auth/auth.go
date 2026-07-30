package auth

import (
	"context"
	"errors"
	"os"
	"time"

	"HailowAuthService/internal/domain"
	redis_repository "HailowAuthService/internal/infrastructure/redis/repository"
	"HailowAuthService/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthUseCase struct {
	repo       *redis_repository.SessionRepository
	userRepo   *repository.UserRepository
	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

type Usecase interface {
	SignUp(ctx context.Context, input *domain.UserInfo) (*domain.User, error)
	SignIn(ctx context.Context, input *domain.UserInfo) (*domain.TokenPair, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*domain.TokenPair, error)
	ValidateToken(ctx context.Context, accessToken string) error
	Logout(ctx context.Context, refreshToken string) error
}

type jwtClaims struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	TokenType string    `json:"token_type,omitempty"`
	jwt.RegisteredClaims
}

func NewAuthUsecase(userRepo *repository.UserRepository, repo *redis_repository.SessionRepository) *AuthUseCase {
	return &AuthUseCase{
		repo:     repo,
		userRepo: userRepo,
	}
}

func (u *AuthUseCase) SignUp(ctx context.Context, input *domain.UserInfo) (*domain.User, error) {
	if input.Email == "" || input.Password == "" {
		return nil, domain.ErrInvalidCredentials
	}

	_, err := u.userRepo.GetByEmail(ctx, input.Email)
	if err == nil {
		return nil, domain.ErrUserAlreadyExists
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	input.Password = string(hashedPassword)

	user, err := u.userRepo.CreateUser(ctx, input)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (u *AuthUseCase) SignIn(ctx context.Context, input *domain.UserInfo) (*domain.TokenPair, error) {
	if input.Email == "" || input.Password == "" {
		return nil, domain.ErrInvalidCredentials
	}

	user, err := u.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	tokens, err := u.generateTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (u *AuthUseCase) RefreshTokens(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	if refreshToken == "" {
		return nil, domain.ErrTokenNotFound
	}

	session, err := u.repo.GetSessionByToken(ctx, refreshToken)
	if err != nil {
		return nil, domain.ErrTokenNotFound
	}

	if session.IsExpired() {
		_ = u.repo.DeleteSession(ctx, refreshToken)
		return nil, domain.ErrUnauthorized
	}

	user, err := u.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	return u.generateAccessTokenOnly(user, refreshToken)
}

func (u *AuthUseCase) ValidateToken(ctx context.Context, accessToken string) error {
	claims, err := u.parseAccessToken(accessToken)
	if err != nil {
		return domain.ErrUnauthorized
	}

	_, err = u.userRepo.GetByEmail(ctx, claims.Email)
	if err != nil {
		return domain.ErrUserNotFound
	}

	return nil
}

func (u *AuthUseCase) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return domain.ErrTokenNotFound
	}
	return u.repo.DeleteSession(ctx, refreshToken)
}

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
	if key := os.Getenv("JWT_ACCESS_KEY"); key != "" {
		return []byte(key)
	}
	return []byte("secret")
}

func (u *AuthUseCase) getRefreshSigningKey() []byte {
	if key := os.Getenv("JWT_REFRESH_KEY"); key != "" {
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
