package main

import (
	"log"
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

	addr := os.Getenv("AUTH_SERVICE_ADDR")
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

	srv, err := server.Init(addr, port)
	if err != nil {
		logger.Log.Fatalf("Failed to initialize server: %v", err)
	}

	if err := srv.Run(); err != nil {
		logger.Log.Fatalf("Failed to run server: %v", err)
	}
}
