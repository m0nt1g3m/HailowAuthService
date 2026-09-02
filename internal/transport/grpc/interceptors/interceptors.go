package interceptors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"HailowAuthService/internal/domain"
	"HailowAuthService/internal/transport/grpc/response/errorcode"
	"HailowAuthService/pkg/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ContextKey string

const (
	ContextUserIDKey       ContextKey = "user_id"
	ContextUserRoleKey     ContextKey = "user_role"
	ContextRefreshTokenKey ContextKey = "refresh_token"
)

func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		stCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				stCode = st.Code()
			} else {
				stCode = codes.Unknown
			}
			logger.Log.Errorf("[gRPC] %s | %s | %v | err: %v", info.FullMethod, stCode, duration, err)
		} else {
			logger.Log.Infof("[gRPC] %s | %s | %v", info.FullMethod, stCode, duration)
		}

		return resp, err
	}
}

func AuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if strings.HasSuffix(info.FullMethod, "SignIn") || strings.HasSuffix(info.FullMethod, "SignUp") || strings.HasSuffix(info.FullMethod, "ValidateToken") {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, errorcode.ToStatus(domain.ErrMDNotFound)
		}

		var userID string
		if userIDs := md.Get("x-user-id"); len(userIDs) > 0 && userIDs[0] != "" {
			userID = userIDs[0]
		} else if userIDs := md.Get("user-id"); len(userIDs) > 0 && userIDs[0] != "" {
			userID = userIDs[0]
		}

		var userRole string
		if userRoles := md.Get("x-user-role"); len(userRoles) > 0 && userRoles[0] != "" {
			userRole = userRoles[0]
		} else if userRoles := md.Get("user-role"); len(userRoles) > 0 && userRoles[0] != "" {
			userRole = userRoles[0]
		}

		if userID == "" {
			return nil, errorcode.ToStatus(domain.ErrUserIDNotFoundMD)
		}

		ctx = context.WithValue(ctx, ContextUserIDKey, userID)
		ctx = context.WithValue(ctx, ContextUserRoleKey, userRole)

		return handler(ctx, req)
	}
}

func RefreshTokenInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if strings.HasSuffix(info.FullMethod, "RefreshTokens") || strings.HasSuffix(info.FullMethod, "Logout") {
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return nil, errorcode.ToStatus(domain.ErrRefreshTokenMD)
			}

			tokens := md.Get("refresh-token")
			logger.Log.Debugf("refresh-token from metadata: %s", tokens[0])
			if len(tokens) == 0 || tokens[0] == "" {
				return nil, errorcode.ToStatus(domain.ErrRefreshTokenMD)
			}

			ctx = context.WithValue(ctx, ContextRefreshTokenKey, tokens[0])
		}

		return handler(ctx, req)
	}
}

func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Log.Errorf("Panic recovered in gRPC interceptor: %v", r)
				err = status.Errorf(codes.Internal, "Internal server error: %v", fmt.Sprint(r))
			}
		}()

		return handler(ctx, req)
	}
}
