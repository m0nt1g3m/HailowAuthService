package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"HailowAuthService/internal/transport/grpc/server"
	"HailowAuthService/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found or error loading it:", err)
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}
	logger.InitLogger(env)

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "localhost"
	}

	portStr := os.Getenv("AUTH_SERVICE_PORT")
	if portStr == "" {
		portStr = "50001"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid PORT value '%s': %v", portStr, err)
	}

	grpcServer, err := server.Init(addr, port)
	if err != nil {
		logger.Log.Fatalf("Failed to initialize server: %v", err)
	}

	listenAddr := fmt.Sprintf("%s:%d", addr, port)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.Log.Fatalf("Failed to listen on %s: %v", listenAddr, err)
	}

	logger.Log.Infof("gRPC server started on %s", listenAddr)
	if err := grpcServer.Serve(listener); err != nil {
		logger.Log.Fatalf("gRPC server stopped with error: %v", err)
	}
}
