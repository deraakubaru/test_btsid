package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RetryConfig holds parameters for the bounded database connection retry loop.
type RetryConfig struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
}

// DefaultRetryConfig returns a standard bounded backoff configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:     5,
		InitialInterval: 1 * time.Second,
		MaxInterval:     8 * time.Second,
	}
}

// NewPool initializes a pgxpool.Pool with a bounded retry loop and exponential backoff.
// It fails fast if all connection attempts fail or context is cancelled.
func NewPool(ctx context.Context, dbURL string, retryCfg RetryConfig) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DATABASE_URL: %w", err)
	}

	var pool *pgxpool.Pool
	interval := retryCfg.InitialInterval

	for attempt := 1; attempt <= retryCfg.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connection context cancelled: %w", ctx.Err())
		default:
		}

		log.Printf("Connecting to PostgreSQL (attempt %d/%d)...", attempt, retryCfg.MaxAttempts)

		pool, err = pgxpool.NewWithConfig(ctx, config)
		if err == nil {
			pingCtx, pingCancel := context.WithTimeout(ctx, 3*time.Second)
			err = pool.Ping(pingCtx)
			pingCancel()
			if err == nil {
				log.Println("PostgreSQL connection established successfully")
				return pool, nil
			}
			pool.Close()
		}

		log.Printf("PostgreSQL connection attempt %d failed: %v", attempt, err)

		if attempt < retryCfg.MaxAttempts {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("connection context cancelled during backoff: %w", ctx.Err())
			case <-time.After(interval):
			}
			interval *= 2
			if interval > retryCfg.MaxInterval {
				interval = retryCfg.MaxInterval
			}
		}
	}

	return nil, fmt.Errorf("failed to connect to PostgreSQL after %d attempts: %w", retryCfg.MaxAttempts, err)
}
