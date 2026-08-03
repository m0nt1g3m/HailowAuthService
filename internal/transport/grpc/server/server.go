package server

import (
	"context"
	"os"
	"strings"

	pb "HailowAuthService/HailowProto/build/go/AuthService/v1"
	redis_repository "HailowAuthService/internal/infrastructure/redis/repository"
	"HailowAuthService/internal/repository"
	"HailowAuthService/internal/transport/grpc/handlers"
	"HailowAuthService/internal/transport/grpc/interceptors"
	"HailowAuthService/internal/usecase/auth"
	"HailowAuthService/pkg/database"
	"HailowAuthService/pkg/logger"
	cache "HailowAuthService/pkg/redis"
	s3storage "HailowAuthService/pkg/s3"

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

	var s3Client *s3storage.S3Client
	if accessKeyID := os.Getenv("S3_ACCESS_KEY_ID"); accessKeyID != "" || os.Getenv("S3_SECRET_ACCESS_KEY") != "" || os.Getenv("S3_BUCKET") != "" {
		cfg := s3storage.Config{
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
			Region:          os.Getenv("S3_REGION"),
			Bucket:          os.Getenv("S3_BUCKET"),
			Endpoint:        os.Getenv("S3_ENDPOINT"),
		}
		s3Client, err = s3storage.NewS3Client(context.Background(), cfg)
		if err != nil {
			return nil, err
		}
	}

	authUsecase := auth.NewAuthUsecase(userRepo, sessionRepo, s3Client)
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
