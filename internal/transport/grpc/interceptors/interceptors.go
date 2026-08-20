package interceptors

import (
	"context"
	"strings"
	"time"

	"HailowAuthService/pkg/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ContextKey string

const (
	ContextUserIDKey   ContextKey = "user_id"
	ContextUserRoleKey ContextKey = "user_role"
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
		if !(strings.HasSuffix(info.FullMethod, "SignIn") || strings.HasSuffix(info.FullMethod, "SignUp")) {
			md, ok := metadata.FromIncomingContext(ctx)
			if ok {
				if userIDs := md.Get("x-user-id"); len(userIDs) > 0 && userIDs[0] != "" {
					ctx = context.WithValue(ctx, ContextUserIDKey, userIDs[0])
				} else if userIDs := md.Get("user-id"); len(userIDs) > 0 && userIDs[0] != "" {
					ctx = context.WithValue(ctx, ContextUserIDKey, userIDs[0])
				}

				if userRoles := md.Get("x-user-role"); len(userRoles) > 0 && userRoles[0] != "" {
					ctx = context.WithValue(ctx, ContextUserRoleKey, userRoles[0])
				} else if userRoles := md.Get("user-role"); len(userRoles) > 0 && userRoles[0] != "" {
					ctx = context.WithValue(ctx, ContextUserRoleKey, userRoles[0])
				}
			}
		}

		return handler(ctx, req)
	}
}
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Log.Errorf("Panic recovered in gRPC interceptor: %v", r)
			}
		}()
		resp, err := handler(ctx, req)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Internal server error")
		}
		return resp, nil
	}
}
