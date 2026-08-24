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
	"HailowAuthService/pkg/logger"
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
	SignIn(ctx context.Context, input *domain.UserInfo) (*domain.TokenPair, string, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*domain.TokenPair, error)
	ValidateToken(ctx context.Context, accessToken string) error
	Logout(ctx context.Context, refreshToken string) error
	UploadAvatar(ctx context.Context, userID string, avatarImage []byte, contentType string) (*domain.User, error)
	UpdateProfile(ctx context.Context, input *domain.User) (*domain.User, error)
	UpdateDeliveryInfo(ctx context.Context, input *domain.User) (*domain.User, error)
	GetProfile(ctx context.Context, userID string) (*domain.User, error)
	ResetPassword(ctx context.Context, userID string, newPassword string) error
	DeleteAccount(ctx context.Context, userID string) error
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
	accessTTL := 15 * time.Minute
	if env := os.Getenv("AUTH_ACCESS_TTL"); env != "" {
		if parsed, err := time.ParseDuration(env); err == nil && parsed > 0 {
			accessTTL = parsed
		}
	}

	refreshTTL := 7 * 24 * time.Hour
	if env := os.Getenv("AUTH_REFRESH_TTL"); env != "" {
		if parsed, err := time.ParseDuration(env); err == nil && parsed > 0 {
			refreshTTL = parsed
		}
	}
	logger.Log.Debugf("Access TTL: %s", accessTTL)
	logger.Log.Debugf("Refresh TTL: %s", refreshTTL)
	return &AuthUseCase{
		repo:       repo,
		userRepo:   userRepo,
		s3Client:   s3Client,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
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

func (u *AuthUseCase) SignIn(ctx context.Context, input *domain.UserInfo) (*domain.TokenPair, string, error) {
	if input.Email == "" || input.Password == "" {
		return nil, "", domain.ErrInvalidCredentials
	}

	user, err := u.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, "", domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, "", domain.ErrInvalidCredentials
	}

	tokens, err := u.generateTokenPair(ctx, user)
	if err != nil {
		return nil, "", err
	}

	return tokens, user.ID, nil
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

	_, err = u.userRepo.GetByID(ctx, claims.ID)
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

func (u *AuthUseCase) UploadAvatar(ctx context.Context, userID string, avatarImage []byte, contentType string) (*domain.User, error) {
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

func (u *AuthUseCase) DeleteAccount(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrUserNotFound
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
