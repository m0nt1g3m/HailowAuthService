package interceptors

import (
	"context"
	"time"

	"HailowAuthService/pkg/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		logger.Log.Infof("incoming gRPC request: %s", info.FullMethod)
		resp, err := handler(ctx, req)
		logger.Log.Infof("finished gRPC request: %s duration=%s error=%v", info.FullMethod, time.Since(start), err)
		return resp, err
	}
}

func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Log.Errorf("panic recovered in gRPC interceptor: %v", r)
			}
		}()
		resp, err := handler(ctx, req)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "internal server error")
		}
		return resp, nil
	}
}
