package handlers

import (
	"context"

	pb "HailowAuthService/HailowProto/build/go/AuthService/v1"
	"HailowAuthService/internal/domain"
	"HailowAuthService/internal/usecase/auth"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthHandler struct {
	usecase auth.Usecase
	pb.UnimplementedAuthServiceServer
}

func NewAuthHandler(usecase auth.Usecase) *AuthHandler {
	return &AuthHandler{usecase: usecase}
}

func (h *AuthHandler) SignUp(ctx context.Context, req *pb.SignUpRequest) (*pb.SignUpResponse, error) {
	user, _, err := h.usecase.SignUp(ctx, req.Email, req.Password)
	if err != nil {
		return nil, toStatus(err)
	}

	return &pb.SignUpResponse{User: mapUser(user)}, nil
}

func (h *AuthHandler) SignIn(ctx context.Context, req *pb.SignInRequest) (*pb.SignInResponse, error) {
	user, tokens, err := h.usecase.SignIn(ctx, req.Email, req.Password)
	if err != nil {
		return nil, toStatus(err)
	}

	return &pb.SignInResponse{User: mapUser(user), Tokens: mapTokenPair(tokens)}, nil
}

func (h *AuthHandler) RefreshTokens(ctx context.Context, req *pb.RefreshTokensRequest) (*pb.RefreshTokensResponse, error) {
	tokens, err := h.usecase.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		return nil, toStatus(err)
	}

	return &pb.RefreshTokensResponse{Tokens: mapTokenPair(tokens)}, nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if err := h.usecase.Logout(ctx, req.RefreshToken); err != nil {
		return nil, toStatus(err)
	}

	return &pb.LogoutResponse{Success: true}, nil
}

func (h *AuthHandler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	user, err := h.usecase.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return nil, toStatus(err)
	}

	return &pb.ValidateTokenResponse{IsValid: true, User: mapUser(user)}, nil
}

func (h *AuthHandler) GetUserSessions(ctx context.Context, req *pb.GetUserSessionsRequest) (*pb.GetUserSessionsResponse, error) {
	sessions, err := h.usecase.GetUserSessions(ctx, req.UserId)
	if err != nil {
		return nil, toStatus(err)
	}

	result := make([]*pb.Session, 0, len(sessions))
	for _, session := range sessions {
		result = append(result, mapSession(session))
	}

	return &pb.GetUserSessionsResponse{Sessions: result}, nil
}

func (h *AuthHandler) RevokeSession(ctx context.Context, req *pb.RevokeSessionRequest) (*pb.RevokeSessionResponse, error) {
	if err := h.usecase.RevokeSession(ctx, req.UserId, req.SessionId); err != nil {
		return nil, toStatus(err)
	}

	return &pb.RevokeSessionResponse{Success: true}, nil
}

func mapUser(user *domain.User) *pb.User {
	if user == nil {
		return nil
	}

	return &pb.User{
		Id:        user.ID,
		AvatarUrl: user.AvatarURL,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Role:      mapRole(user.Role),
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}
}

func mapTokenPair(tokens *domain.TokenPair) *pb.TokenPair {
	if tokens == nil {
		return nil
	}
	return &pb.TokenPair{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken}
}

func mapSession(session *domain.Session) *pb.Session {
	if session == nil {
		return nil
	}
	return &pb.Session{
		Id:        session.ID,
		UserId:    session.UserID,
		CreatedAt: timestamppb.New(session.CreatedAt),
		ExpiresAt: timestamppb.New(session.ExpiresAt),
	}
}

func mapRole(role domain.Role) pb.Role {
	switch role {
	case domain.RoleCustomer:
		return pb.Role_ROLE_CUSTOMER
	case domain.RoleSeller:
		return pb.Role_ROLE_SELLER
	case domain.RoleAdmin:
		return pb.Role_ROLE_ADMIN
	default:
		return pb.Role_ROLE_UNSPECIFIED
	}
}

func toStatus(err error) error {
	switch err {
	case domain.ErrUserAlreadyExists:
		return status.Error(codes.AlreadyExists, err.Error())
	case domain.ErrInvalidCredentials, domain.ErrUnauthorized:
		return status.Error(codes.Unauthenticated, err.Error())
	case domain.ErrTokenNotFound:
		return status.Error(codes.NotFound, err.Error())
	case domain.ErrSessionNotFound:
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
