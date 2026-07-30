package server

import (
	pb "HailowAuthService/HailowProto/build/go/AuthService/v1"
	redis_repository "HailowAuthService/internal/infrastructure/redis/repository"
	"HailowAuthService/internal/repository"
	"HailowAuthService/internal/transport/grpc/handlers"
	"HailowAuthService/internal/transport/grpc/interceptors"
	"HailowAuthService/internal/usecase/auth"
	"HailowAuthService/pkg/database"
	"HailowAuthService/pkg/logger"
	cache "HailowAuthService/pkg/redis"
	"os"
	"strings"

	"google.golang.org/grpc"
)

func Init(addr string, port int) (*grpc.Server, error) {
	debug := os.Getenv("DEBUG")
	logger.Log.Infof("initialize gRPC server on %s:%d", addr, port)

	var dbURL string
	var redisAddr string

	if debug == "true" {
		dbURL = os.Getenv("DEV_DATABASE_URL")
	} else {
		dbURL = os.Getenv("DATABASE_URL")
	}

	pool := database.InitDB(dbURL)
	userRepo := repository.NewUserRepository(pool)

	if debug == "true" {
		redisAddr = strings.Join([]string{os.Getenv("DEV_REDIS_HOST"), os.Getenv("DEV_REDIS_PORT")}, ":")
	} else {
		redisAddr = strings.Join([]string{os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")}, ":")
	}

	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient, err := cache.NewRedisClient(redisAddr)
	if err != nil {
		return nil, err
	}

	sessionRepo := redis_repository.NewSessionRepository(redisClient)

	authUsecase := auth.NewAuthUsecase(userRepo, sessionRepo)
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
