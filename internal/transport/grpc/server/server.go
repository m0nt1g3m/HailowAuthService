package server

import (
	pb "HailowAuthService/HailowProto/build/go/AuthService/v1"
	"HailowAuthService/internal/repository"
	"HailowAuthService/internal/transport/grpc/handlers"
	"HailowAuthService/internal/transport/grpc/interceptors"
	"HailowAuthService/internal/usecase/auth"
	"HailowAuthService/pkg/database"
	"HailowAuthService/pkg/logger"
	cache "HailowAuthService/pkg/redis"
	"os"

	"google.golang.org/grpc"
)

func Init(addr string, port int) (*grpc.Server, error) {
	logger.Log.Infof("initialize gRPC server on %s:%d", addr, port)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DEV_DATABASE_URL")
	}

	pool := database.InitDB(dbURL)
	userRepo := repository.NewPostgresUserRepository(pool)

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisClient, err := cache.NewRedisClient(redisAddr)
	if err != nil {
		return nil, err
	}

	authUsecase := auth.NewAuthUsecase(userRepo, redisClient)
	authHandler := handlers.NewAuthHandler(authUsecase)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(),
			interceptors.LoggingInterceptor(),
		),
	)

	pb.RegisterAuthServiceServer(grpcServer, authHandler)
	return grpcServer, nil
}
