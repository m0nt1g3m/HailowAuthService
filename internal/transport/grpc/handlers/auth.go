package handlers

import (
	"context"
	"strings"

	pb "HailowAuthService/HailowProto/build/go/AuthService/v1"
	"HailowAuthService/internal/domain"
	"HailowAuthService/internal/transport/grpc/response/errorcode"
	"HailowAuthService/internal/usecase/auth"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthHandler struct {
	usecase auth.Usecase
	pb.UnimplementedAuthServiceServer
}

func NewAuthHandler(usecase auth.Usecase) *AuthHandler {
	return &AuthHandler{usecase: usecase}
}

func (h *AuthHandler) AdminSignUp(ctx context.Context, req *pb.AdminSignUpRequest) (*pb.AdminSignUpResponse, error) {
	input := domain.UserInfo{
		FirstName:   req.GetFirstName(),
		LastName:    req.GetLastName(),
		Email:       req.GetEmail(),
		PhoneNumber: req.GetPhoneNumber(),
		City:        req.GetCity(),
		Street:      req.GetStreet(),
		Building:    req.GetBuilding(),
		Password:    req.GetPassword(),
		Role:        domain.RoleAdmin,
	}
	user, err := h.usecase.SignUp(ctx, &input)
	if err != nil {
		return nil, errorcode.ToStatus(err)
	}

	return &pb.AdminSignUpResponse{User: mapUser(user)}, nil
}

func (h *AuthHandler) CustomerSignUp(ctx context.Context, req *pb.CustomerSignUpRequest) (*pb.CustomerSignUpResponse, error) {
	input := &domain.UserInfo{
		FirstName:   req.GetFirstName(),
		LastName:    req.GetLastName(),
		Email:       req.GetEmail(),
		PhoneNumber: req.GetPhoneNumber(),
		City:        req.GetCity(),
		Street:      req.GetStreet(),
		Building:    req.GetBuilding(),
		Password:    req.GetPassword(),
		Role:        domain.RoleCustomer,
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

func (h *AuthHandler) UploadAvatar(ctx context.Context, req *pb.UploadAvatarRequest) (*pb.UploadAvatarResponse, error) {
	accessToken := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get("authorization")
		if len(values) > 0 {
			accessToken = strings.TrimPrefix(values[0], "Bearer ")
		}
	}

	user, err := h.usecase.UploadAvatar(ctx, accessToken, req.GetUserId(), req.GetAvatarImage(), "")
	if err != nil {
		return nil, errorcode.ToStatus(err)
	}

	if user == nil || user.AvatarURL == nil {
		return &pb.UploadAvatarResponse{}, nil
	}

	return &pb.UploadAvatarResponse{AvatarUrl: *user.AvatarURL}, nil
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
		Id:          user.ID,
		AvatarUrl:   user.AvatarURL,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		PhoneNumber: user.PhoneNumber,
		City:        user.City,
		Street:      user.Street,
		Building:    user.Building,
		Flat:        user.Flat,
		Porch:       user.Porch,
		Floor:       user.Floor,
		Role:        mapRole(user.Role),
		CreatedAt:   timestamppb.New(user.CreatedAt),
		UpdatedAt:   timestamppb.New(user.UpdatedAt),
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
	case domain.RoleAdmin:
		return pb.Role_ROLE_ADMIN
	default:
		return pb.Role_ROLE_UNSPECIFIED
	}
}
