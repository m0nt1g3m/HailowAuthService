package handlers

import (
	"context"

	pb "HailowAuthService/HailowProto/build/go/AuthService/v1"
	"HailowAuthService/internal/domain"
	"HailowAuthService/internal/transport/grpc/response/errorcode"
	"HailowAuthService/internal/usecase/auth"
	"HailowAuthService/pkg/logger"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthHandler struct {
	usecase auth.Usecase
	pb.UnimplementedAuthServiceServer
}

func NewAuthHandler(usecase auth.Usecase) *AuthHandler {
	return &AuthHandler{usecase: usecase}
}

func (h *AuthHandler) ModeratorSignUp(ctx context.Context, req *pb.ModeratorSignUpRequest) (*pb.ModeratorSignUpResponse, error) {
	input := domain.UserInfo{
		FirstName: req.GetFirstName(),
		LastName:  req.GetLastName(),
		Email:     req.GetEmail(),
		Password:  req.GetPassword(),
		Role:      domain.RoleModerator,
	}
	user, err := h.usecase.SignUp(ctx, &input)
	if err != nil {
		return nil, errorcode.ToStatus(err)
	}

	return &pb.ModeratorSignUpResponse{User: mapUser(user)}, nil
}

func (h *AuthHandler) SellerSignUp(ctx context.Context, req *pb.SellerSignUpRequest) (*pb.SellerSignUpResponse, error) {
	input := domain.UserInfo{
		FirstName: req.GetFirstName(),
		LastName:  req.GetLastName(),
		Email:     req.GetEmail(),
		Password:  req.GetPassword(),
		Role:      domain.RoleSeller,
	}
	user, err := h.usecase.SignUp(ctx, &input)
	if err != nil {
		return nil, errorcode.ToStatus(err)
	}

	return &pb.SellerSignUpResponse{User: mapUser(user)}, nil
}

func (h *AuthHandler) CustomerSignUp(ctx context.Context, req *pb.CustomerSignUpRequest) (*pb.CustomerSignUpResponse, error) {
	logger.Log.Infof("CustomerSignUp called with FirstName: %s, LastName: %s, Email: %s", req.GetFirstName(), req.GetLastName(), req.GetEmail())
	input := &domain.UserInfo{
		FirstName: req.GetFirstName(),
		LastName:  req.GetLastName(),
		Email:     req.GetEmail(),
		Password:  req.GetPassword(),
		Role:      domain.RoleCustomer,
	}
	user, err := h.usecase.SignUp(ctx, input)
	if err != nil {
		return nil, errorcode.ToStatus(err)
	}

	return &pb.CustomerSignUpResponse{User: mapUser(user)}, nil
}

func (h *AuthHandler) SignIn(ctx context.Context, req *pb.SignInRequest) (*pb.SignInResponse, error) {
	input := &domain.UserInfo{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	}
	tokens, err := h.usecase.SignIn(ctx, input)
	if err != nil {
		return nil, errorcode.ToStatus(err)
	}

	return &pb.SignInResponse{Tokens: mapTokenPair(tokens)}, nil
}

func (h *AuthHandler) RefreshTokens(ctx context.Context, req *pb.RefreshTokensRequest) (*pb.RefreshTokensResponse, error) {
	tokens, err := h.usecase.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		return nil, errorcode.ToStatus(err)
	}

	return &pb.RefreshTokensResponse{AccessToken: tokens.AccessToken}, nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if err := h.usecase.Logout(ctx, req.RefreshToken); err != nil {
		return nil, errorcode.ToStatus(err)
	}

	return &pb.LogoutResponse{Success: true}, nil
}

func (h *AuthHandler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	err := h.usecase.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return nil, errorcode.ToStatus(err)
	}

	return &pb.ValidateTokenResponse{IsValid: true}, nil
}

func mapUser(user *domain.User) *pb.User {
	if user == nil {
		return nil
	}

	return &pb.User{
		Id:        user.ID,
		Avatar:    user.Avatar,
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

func mapRole(role domain.Role) pb.Role {
	switch role {
	case domain.RoleCustomer:
		return pb.Role_ROLE_CUSTOMER
	case domain.RoleSeller:
		return pb.Role_ROLE_SELLER
	case domain.RoleModerator:
		return pb.Role_ROLE_MODERATOR
	default:
		return pb.Role_ROLE_UNSPECIFIED
	}
}
