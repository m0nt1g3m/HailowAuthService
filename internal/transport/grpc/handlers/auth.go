package handlers

import (
	"context"
	"strings"

	pb "HailowAuthService/HailowProto/build/go/AuthService/v1"
	"HailowAuthService/internal/domain"
	"HailowAuthService/internal/transport/grpc/interceptors"
	"HailowAuthService/internal/transport/grpc/response/errorcode"
	"HailowAuthService/internal/usecase/auth"
	"HailowAuthService/pkg/logger"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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

func checkPermissions(ctx context.Context, targetUserID string) error {
	callerID, ok := ctx.Value(interceptors.ContextUserIDKey).(string)
	logger.Log.Debugf("checkOwnership: callerID=%s, targetUserID=%s", callerID, targetUserID)

	if !ok || callerID == "" {
		return status.Error(codes.Unauthenticated, "unauthorized")
	}

	// Checking the user's role from the context
	callerRole, _ := ctx.Value(interceptors.ContextUserRoleKey).(string)

	// The admin can perform any actions
	if callerRole == string(domain.RoleAdmin) {
		return nil
	}

	// A customer can only edit their own profile
	if callerID != targetUserID {
		return status.Error(codes.PermissionDenied, "Access denied")
	}

	return nil
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
		logger.Log.Errorf("AdminSignUp error: %v", err)
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
		logger.Log.Errorf("CustomerSignUp error: %v", err)
		return nil, errorcode.ToStatus(err)
	}

	return &pb.CustomerSignUpResponse{User: mapUser(user)}, nil
}

func (h *AuthHandler) SignIn(ctx context.Context, req *pb.SignInRequest) (*pb.SignInResponse, error) {
	input := &domain.UserInfo{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	}
	tokens, userID, err := h.usecase.SignIn(ctx, input)
	if err != nil {
		logger.Log.Errorf("SignIn error: %v", err)
		return nil, errorcode.ToStatus(err)
	}

	return &pb.SignInResponse{Id: userID, Tokens: mapTokenPair(tokens)}, nil
}

func (h *AuthHandler) UploadAvatar(ctx context.Context, req *pb.UploadAvatarRequest) (*pb.UploadAvatarResponse, error) {
	if err := checkPermissions(ctx, req.GetUserId()); err != nil {
		return nil, err
	}

	user, err := h.usecase.UploadAvatar(ctx, req.GetUserId(), req.GetAvatarImage(), "")
	if err != nil {
		logger.Log.Errorf("UploadAvatar error: %v", err)
		return nil, errorcode.ToStatus(err)
	}

	if user == nil || user.AvatarURL == nil {
		return &pb.UploadAvatarResponse{}, nil
	}

	return &pb.UploadAvatarResponse{AvatarUrl: *user.AvatarURL}, nil
}

func (h *AuthHandler) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	if err := checkPermissions(ctx, req.GetId()); err != nil {
		return nil, err
	}
	input := &domain.User{
		ID:          req.GetId(),
		FirstName:   req.GetFirstName(),
		LastName:    req.GetLastName(),
		Email:       req.GetEmail(),
		PhoneNumber: req.GetPhoneNumber(),
	}

	user, err := h.usecase.UpdateProfile(ctx, input)
	if err != nil {
		logger.Log.Errorf("UpdateProfile error: %v", err)
		return nil, errorcode.ToStatus(err)
	}

	return &pb.UpdateProfileResponse{User: mapUser(user)}, nil
}

func (h *AuthHandler) UpdateDeliveryInfo(ctx context.Context, req *pb.UpdateDeliveryInfoRequest) (*pb.UpdateDeliveryInfoResponse, error) {
	if err := checkPermissions(ctx, req.GetId()); err != nil {
		return nil, err
	}
	input := &domain.User{
		ID:       req.GetId(),
		City:     req.GetCity(),
		Street:   req.GetStreet(),
		Building: req.GetBuilding(),
		Porch:    req.Porch,
		Floor:    req.Floor,
		Flat:     req.Flat,
	}

	user, err := h.usecase.UpdateDeliveryInfo(ctx, input)
	if err != nil {
		logger.Log.Errorf("UpdateDeliveryInfo error: %v", err)
		return nil, errorcode.ToStatus(err)
	}

	return &pb.UpdateDeliveryInfoResponse{User: mapUser(user)}, nil
}

func (h *AuthHandler) RefreshTokens(ctx context.Context, req *pb.RefreshTokensRequest) (*pb.RefreshTokensResponse, error) {
	tokens, err := h.usecase.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		logger.Log.Errorf("RefreshTokens error: %v", err)
		return nil, errorcode.ToStatus(err)
	}

	return &pb.RefreshTokensResponse{AccessToken: tokens.AccessToken}, nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "refresh_token is not found")
	}

	tokens := md.Get("refresh_token")
	if len(tokens) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "refresh_token is not found")
	}

	refreshToken := tokens[0]
	logger.Log.Debugf("RefreshToken: %s", refreshToken)
	if err := h.usecase.Logout(ctx, refreshToken); err != nil {
		logger.Log.Errorf("Logout error: %v", err)
		return nil, errorcode.ToStatus(err)
	}

	return &pb.LogoutResponse{Success: true}, nil
}

func (h *AuthHandler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	accessToken := getAccessTokenFromContext(ctx)
	err := h.usecase.ValidateToken(ctx, accessToken)
	if err != nil {
		logger.Log.Errorf("ValidateToken error: %v", err)
		return nil, errorcode.ToStatus(err)
	}

	return &pb.ValidateTokenResponse{IsValid: true}, nil
}

func (h *AuthHandler) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	if err := checkPermissions(ctx, req.GetId()); err != nil {
		return nil, err
	}
	user, err := h.usecase.GetProfile(ctx, req.GetId())
	if err != nil {
		logger.Log.Errorf("GetProfile error: %v", err)
		return nil, errorcode.ToStatus(err)
	}

	return &pb.GetProfileResponse{User: mapUser(user)}, nil
}

func (h *AuthHandler) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.ResetPasswordResponse, error) {
	if err := checkPermissions(ctx, req.GetId()); err != nil {
		return nil, err
	}
	err := h.usecase.ResetPassword(ctx, req.GetId(), req.GetNewPassword())
	if err != nil {
		logger.Log.Errorf("ResetPassword error: %v", err)
		return nil, errorcode.ToStatus(err)
	}

	return &pb.ResetPasswordResponse{Success: true}, nil
}

func (h *AuthHandler) DeleteAccount(ctx context.Context, req *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error) {
	if err := checkPermissions(ctx, req.GetId()); err != nil {
		return nil, err
	}
	err := h.usecase.DeleteAccount(ctx, req.GetId())
	if err != nil {
		logger.Log.Errorf("DeleteAccount error: %v", err)
		return nil, errorcode.ToStatus(err)
	}

	return &pb.DeleteAccountResponse{Success: true}, nil
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

func getAccessTokenFromContext(ctx context.Context) string {
	accessToken := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get("authorization")
		if len(values) > 0 {
			accessToken = strings.TrimPrefix(values[0], "Bearer ")
		}
	}
	return accessToken
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
