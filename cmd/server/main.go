package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"HailowAuthService/internal/transport/grpc/server"

	"github.com/joho/godotenv"
)

func main() {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			slog.Warn(fmt.Sprintf("Failed to load .env: %v", err))
		}
	} else if !os.IsNotExist(err) {
		slog.Warn(fmt.Sprintf("Failed to check .env file: %v", err))
	}

	debug := false
	if os.Getenv("DEBUG") == "true" {
		debug = true
	}

	addr := os.Getenv("AUTH_SERVICE_HOST")
	if addr == "" {
		addr = "0.0.0.0"
	}

	portStr := os.Getenv("AUTH_SERVICE_PORT")
	if portStr == "" {
		portStr = "50001"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		slog.Error(fmt.Sprintf("Invalid AUTH_SERVICE_PORT value '%s': %v", portStr, err))
	}

	srv, err := server.Init(debug, addr, port)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to initialize server: %v", err))
	}

	if err := srv.Run(); err != nil {
		slog.Error(fmt.Sprintf("Failed to run server: %v", err))
	}
}
