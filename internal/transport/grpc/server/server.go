package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

type Server struct {
	grpcServer  *grpc.Server
	addr        string
	port        int
	listener    net.Listener
	dbPool      *pgxpool.Pool
	redisClient *redis.Client
	s3Client    *s3storage.S3Client
}

func Init(addr string, port int) (*Server, error) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	logger.InitLogger("development")
	logger.Log.Infof("Initializing gRPC server on %s:%d", addr, port)

	debug := os.Getenv("DEBUG")
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
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
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
			return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
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

	return &Server{
		grpcServer:  grpcServer,
		addr:        addr,
		port:        port,
		dbPool:      pool,
		redisClient: redisClient,
		s3Client:    s3Client,
	}, nil
}

func (s *Server) Run() error {
	listenAddr := fmt.Sprintf("%s:%d", s.addr, s.port)
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = ln

	logger.Log.Infof("gRPC server listening on %s", ln.Addr().String())

	go func() {
		if err := s.grpcServer.Serve(ln); err != nil && err != grpc.ErrServerStopped {
			logger.Log.Errorf("gRPC server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	logger.Log.Info("Shutdown signal received")

	s.grpcServer.GracefulStop()

	if s.dbPool != nil {
		s.dbPool.Close()
		logger.Log.Info("PostgreSQL pool closed")
	}

	if s.redisClient != nil {
		if err := s.redisClient.Close(); err != nil {
			logger.Log.Errorf("Error closing Redis: %v", err)
		} else {
			logger.Log.Info("Redis client closed")
		}
	}

	logger.Log.Info("Resources closed, gRPC server stopped")
	return nil
}
