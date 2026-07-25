package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"HailowAuthService/internal/domain"
	"HailowAuthService/internal/repository"
)

type Usecase interface {
	SignUp(ctx context.Context, email, password string) (*domain.User, *domain.TokenPair, error)
	SignIn(ctx context.Context, email, password string) (*domain.User, *domain.TokenPair, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*domain.TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	ValidateToken(ctx context.Context, accessToken string) (*domain.User, error)
	GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error)
	RevokeSession(ctx context.Context, userID, sessionID string) error
}

type authUsecase struct {
	mu          sync.Mutex
	userRepo    repository.UserRepository
	redisClient *redis.Client
}

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	sessionTTL      = 24 * time.Hour
)

type sessionData struct {
	Session      *domain.Session
	AccessToken  string
	RefreshToken string
}

func NewAuthUsecase(userRepo repository.UserRepository, redisClient *redis.Client) Usecase {
	return &authUsecase{
		userRepo:    userRepo,
		redisClient: redisClient,
	}
}

func (u *authUsecase) SignUp(ctx context.Context, email, password string) (*domain.User, *domain.TokenPair, error) {
	existing, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		return nil, nil, domain.ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	user := &domain.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: string(hash),
		Role:         domain.RoleCustomer,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	tokens := u.generateTokenPair()
	if err := u.addSession(ctx, user.ID, tokens); err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (u *authUsecase) SignIn(ctx context.Context, email, password string) (*domain.User, *domain.TokenPair, error) {
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	tokens := u.generateTokenPair()
	if err := u.addSession(ctx, user.ID, tokens); err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (u *authUsecase) RefreshTokens(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	sessionID, err := u.redisClient.Get(ctx, refreshKey(refreshToken)).Result()
	if err == redis.Nil {
		return nil, domain.ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}

	sessionData, err := u.loadSessionData(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sessionData == nil {
		return nil, domain.ErrTokenNotFound
	}

	if err := u.redisClient.Del(ctx, accessKey(sessionData.AccessToken), refreshKey(refreshToken)).Err(); err != nil {
		return nil, err
	}

	tokens := u.generateTokenPair()
	if err := u.updateSessionTokens(ctx, sessionData.Session.ID, tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (u *authUsecase) Logout(ctx context.Context, refreshToken string) error {
	sessionID, err := u.redisClient.Get(ctx, refreshKey(refreshToken)).Result()
	if err == redis.Nil {
		return domain.ErrTokenNotFound
	}
	if err != nil {
		return err
	}

	return u.revokeSessionByID(ctx, sessionID)
}

func (u *authUsecase) ValidateToken(ctx context.Context, accessToken string) (*domain.User, error) {
	userID, err := u.redisClient.Get(ctx, accessKey(accessToken)).Result()
	if err == redis.Nil {
		return nil, domain.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}

	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrUnauthorized
	}

	return user, nil
}

func (u *authUsecase) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	sessionIDs, err := u.redisClient.SMembers(ctx, userSessionsKey(userID)).Result()
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Session, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		sessionData, err := u.loadSessionData(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		if sessionData == nil {
			continue
		}
		result = append(result, sessionData.Session)
	}

	return result, nil
}

func (u *authUsecase) RevokeSession(ctx context.Context, userID, sessionID string) error {
	data, err := u.loadSessionData(ctx, sessionID)
	if err != nil {
		return err
	}
	if data == nil || data.Session.UserID != userID {
		return domain.ErrSessionNotFound
	}

	return u.revokeSessionByID(ctx, sessionID)
}

func (u *authUsecase) generateTokenPair() *domain.TokenPair {
	return &domain.TokenPair{
		AccessToken:  uuid.NewString(),
		RefreshToken: uuid.NewString(),
	}
}

func (u *authUsecase) addSession(ctx context.Context, userID string, tokens *domain.TokenPair) error {
	session := &domain.Session{
		ID:        uuid.NewString(),
		UserID:    userID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(sessionTTL),
	}

	sessionKey := sessionKey(session.ID)
	userSessionsKey := userSessionsKey(userID)

	_, err := u.redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, accessKey(tokens.AccessToken), userID, accessTokenTTL)
		pipe.Set(ctx, refreshKey(tokens.RefreshToken), session.ID, refreshTokenTTL)
		pipe.HSet(ctx, sessionKey, map[string]interface{}{
			"id":            session.ID,
			"user_id":       session.UserID,
			"access_token":  tokens.AccessToken,
			"refresh_token": tokens.RefreshToken,
			"created_at":    session.CreatedAt.Format(time.RFC3339Nano),
			"expires_at":    session.ExpiresAt.Format(time.RFC3339Nano),
		})
		pipe.Expire(ctx, sessionKey, sessionTTL)
		pipe.SAdd(ctx, userSessionsKey, session.ID)
		pipe.Expire(ctx, userSessionsKey, sessionTTL)
		return nil
	})
	return err
}

func (u *authUsecase) updateSessionTokens(ctx context.Context, sessionID string, tokens *domain.TokenPair) error {
	sessionData, err := u.loadSessionData(ctx, sessionID)
	if err != nil {
		return err
	}
	if sessionData == nil {
		return domain.ErrSessionNotFound
	}

	sessionKey := sessionKey(sessionID)

	_, err = u.redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, accessKey(sessionData.AccessToken))
		pipe.Set(ctx, accessKey(tokens.AccessToken), sessionData.Session.UserID, accessTokenTTL)
		pipe.Set(ctx, refreshKey(tokens.RefreshToken), sessionID, refreshTokenTTL)
		pipe.HSet(ctx, sessionKey, map[string]interface{}{
			"access_token":  tokens.AccessToken,
			"refresh_token": tokens.RefreshToken,
			"expires_at":    time.Now().UTC().Add(sessionTTL).Format(time.RFC3339Nano),
		})
		pipe.Expire(ctx, sessionKey, sessionTTL)
		return nil
	})
	return err
}

func (u *authUsecase) loadSessionData(ctx context.Context, sessionID string) (*sessionData, error) {
	sessionKey := sessionKey(sessionID)
	data, err := u.redisClient.HGetAll(ctx, sessionKey).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	createdAt, err := time.Parse(time.RFC3339Nano, data["created_at"])
	if err != nil {
		return nil, err
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, data["expires_at"])
	if err != nil {
		return nil, err
	}

	return &sessionData{
		Session: &domain.Session{
			ID:        data["id"],
			UserID:    data["user_id"],
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
		},
		AccessToken:  data["access_token"],
		RefreshToken: data["refresh_token"],
	}, nil
}

func (u *authUsecase) revokeSessionByID(ctx context.Context, sessionID string) error {
	sessionData, err := u.loadSessionData(ctx, sessionID)
	if err != nil {
		return err
	}
	if sessionData == nil {
		return domain.ErrSessionNotFound
	}

	_, err = u.redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, sessionKey(sessionID))
		pipe.Del(ctx, accessKey(sessionData.AccessToken))
		pipe.Del(ctx, refreshKey(sessionData.RefreshToken))
		pipe.SRem(ctx, userSessionsKey(sessionData.Session.UserID), sessionID)
		return nil
	})
	return err
}

func sessionKey(sessionID string) string {
	return fmt.Sprintf("auth:session:%s", sessionID)
}

func userSessionsKey(userID string) string {
	return fmt.Sprintf("auth:user_sessions:%s", userID)
}

func accessKey(token string) string {
	return fmt.Sprintf("auth:access:%s", token)
}

func refreshKey(token string) string {
	return fmt.Sprintf("auth:refresh:%s", token)
}
