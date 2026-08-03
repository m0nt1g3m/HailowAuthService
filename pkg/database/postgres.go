package database

import (
	"context"
	"time"

	"HailowAuthService/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func InitDB(connectLink string) *pgxpool.Pool {
	config, err := pgxpool.ParseConfig(connectLink)
	if err != nil {
		logger.Log.Fatalf("Failed to parse config: %v", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnIdleTime = 30 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	Pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		logger.Log.Fatalf("Failed to create pool: %v", err)
	}

	if err := Pool.Ping(ctx); err != nil {
		logger.Log.Fatalf("Failed to ping pool: %v", err)
	}

	logger.Log.Info("Database initialized successfully")
	return Pool
}

func CloseDB() {
	if Pool != nil {
		Pool.Close()
		logger.Log.Info("Database closed successfully")
	}
}
