package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"HailowAuthService/internal/domain"
	redis_repository "HailowAuthService/internal/infrastructure/redis/repository"
	"HailowAuthService/internal/repository"
	s3storage "HailowAuthService/pkg/s3"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthUseCase struct {
	repo       *redis_repository.SessionRepository
	userRepo   *repository.UserRepository
	s3Client   *s3storage.S3Client
	accessTTL  time.Duration
	refreshTTL time.Duration
}

type Usecase interface {
	SignUp(ctx context.Context, input *domain.UserInfo) (*domain.User, error)
	SignIn(ctx context.Context, input *domain.UserInfo) (*domain.TokenPair, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*domain.TokenPair, error)
	ValidateToken(ctx context.Context, accessToken string) error
	ValidateTokenForUser(ctx context.Context, accessToken string, userID string) error
	Logout(ctx context.Context, refreshToken string) error
	UploadAvatar(ctx context.Context, accessToken string, userID string, avatarImage []byte, contentType string) (*domain.User, error)
	UpdateProfile(ctx context.Context, input *domain.User) (*domain.User, error)
	UpdateDeliveryInfo(ctx context.Context, input *domain.User) (*domain.User, error)
	GetProfile(ctx context.Context, userID string) (*domain.User, error)
	ResetPassword(ctx context.Context, userID string, newPassword string) error
	DeleteAccount(ctx context.Context, accessToken string, userID string) error
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

func NewAuthUsecase(userRepo *repository.UserRepository, repo *redis_repository.SessionRepository, s3Client *s3storage.S3Client) *AuthUseCase {
	return &AuthUseCase{
		repo:     repo,
		userRepo: userRepo,
		s3Client: s3Client,
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

func (u *AuthUseCase) ValidateTokenForUser(ctx context.Context, accessToken string, userID string) error {
	claims, err := u.parseAccessToken(accessToken)
	if err != nil {
		return domain.ErrUnauthorized
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("Invalid user id: %w", err)
	}

	if claims.ID != parsedUserID {
		return domain.ErrUnauthorized
	}

	_, err = u.userRepo.GetByID(ctx, parsedUserID)
	if err != nil {
		return err
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
	if key := os.Getenv("JWT_ACCESS_KEY_CUSTOMER"); key != "" {
		return []byte(key)
	}
	return []byte("secret")
}

func (u *AuthUseCase) getRefreshSigningKey() []byte {
	if key := os.Getenv("JWT_REFRESH_KEY_CUSTOMER"); key != "" {
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

func (u *AuthUseCase) UploadAvatar(ctx context.Context, accessToken string, userID string, avatarImage []byte, contentType string) (*domain.User, error) {
	if accessToken == "" {
		return nil, domain.ErrUnauthorized
	}
	if userID == "" {
		return nil, domain.ErrUserNotFound
	}
	if len(avatarImage) == 0 {
		return nil, domain.ErrAvatarImageEmpty
	}
	if u.s3Client == nil {
		return nil, domain.ErrAvatarUploadUnavailable
	}
	if contentType == "" {
		contentType = http.DetectContentType(avatarImage)
	}

	if err := u.ValidateToken(ctx, accessToken); err != nil {
		return nil, err
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("Invalid user id: %w", err)
	}

	currentUser, err := u.userRepo.GetByID(ctx, parsedUserID)
	if err != nil {
		return nil, err
	}

	avatarURL, err := u.s3Client.Upload(ctx, "avatars", fmt.Sprintf("%s/%s%s", userID, uuid.NewString(), extensionForContentType(contentType)), bytes.NewReader(avatarImage), contentType)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrAvatarUploadFailed, err)
	}

	user, oldAvatarURL, err := u.userRepo.UpdateAvatar(ctx, parsedUserID, avatarURL)
	if err != nil {
		_ = u.s3Client.Delete(ctx, avatarURL)
		return nil, err
	}

	if currentUser.AvatarURL != nil && *currentUser.AvatarURL != "" && *currentUser.AvatarURL != avatarURL {
		_ = u.s3Client.Delete(ctx, *currentUser.AvatarURL)
	}
	if oldAvatarURL != "" && oldAvatarURL != avatarURL {
		_ = u.s3Client.Delete(ctx, oldAvatarURL)
	}

	return user, nil
}

func (u *AuthUseCase) UpdateProfile(ctx context.Context, input *domain.User) (*domain.User, error) {
	if input == nil || input.ID == "" {
		return nil, domain.ErrUserNotFound
	}

	user, err := u.userRepo.UpdateProfile(ctx, input)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (u *AuthUseCase) UpdateDeliveryInfo(ctx context.Context, input *domain.User) (*domain.User, error) {
	if input == nil || input.ID == "" {
		return nil, domain.ErrUserNotFound
	}

	user, err := u.userRepo.UpdateDeliveryInfo(ctx, input)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (u *AuthUseCase) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	if userID == "" {
		return nil, domain.ErrUserNotFound
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("Invalid user id: %w", err)
	}

	user, err := u.userRepo.GetByID(ctx, parsedUserID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (u *AuthUseCase) ResetPassword(ctx context.Context, userID string, newPassword string) error {
	if userID == "" {
		return domain.ErrUserNotFound
	}
	if newPassword == "" {
		return domain.ErrInvalidCredentials
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("Invalid user id: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = u.userRepo.UpdatePassword(ctx, parsedUserID, string(hashedPassword))
	if err != nil {
		return err
	}

	return nil
}

func (u *AuthUseCase) DeleteAccount(ctx context.Context, accessToken string, userID string) error {
	if accessToken == "" {
		return domain.ErrUnauthorized
	}
	if userID == "" {
		return domain.ErrUserNotFound
	}

	if err := u.ValidateTokenForUser(ctx, accessToken, userID); err != nil {
		return err
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("Invalid user id: %w", err)
	}

	user, err := u.userRepo.GetByID(ctx, parsedUserID)
	if err != nil {
		return err
	}

	if user.AvatarURL != nil && *user.AvatarURL != "" {
		if u.s3Client != nil {
			_ = u.s3Client.Delete(ctx, *user.AvatarURL)
		}
	}

	return u.userRepo.DeleteUser(ctx, parsedUserID)
}

func extensionForContentType(contentType string) string {
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}
