package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

func NewPool(ctx context.Context, cfg DBConfig, logger Logger) (*pgxpool.Pool, error) {
	poolConfig, err := cfg.PoolConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	logger.Info("connecting to postgres",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
		"user", cfg.User,
	)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = pool.Ping(ctx)
		cancel()

		if err == nil {
			logger.Info("successfully connected to postgres")
			return pool, nil
		}

		logger.Error("failed to ping postgres, retrying...", "attempt", i+1, "error", err.Error())
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	pool.Close()
	return nil, fmt.Errorf("failed to connect to postgres after %d retries: %w", maxRetries, err)
}
