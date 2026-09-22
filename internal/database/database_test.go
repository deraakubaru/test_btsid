package database

import (
	"context"
	"testing"
	"time"
)

func TestNewPool_InvalidConnectionString(t *testing.T) {
	ctx := context.Background()
	retryCfg := RetryConfig{
		MaxAttempts:     1,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
	}

	_, err := NewPool(ctx, "invalid-connection-string", retryCfg)
	if err == nil {
		t.Fatal("expected error for invalid connection string, got nil")
	}
}

func TestNewPool_UnreachableHostBoundedRetry(t *testing.T) {
	ctx := context.Background()
	retryCfg := RetryConfig{
		MaxAttempts:     2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
	}

	start := time.Now()
	_, err := NewPool(ctx, "postgres://postgres:postgres@127.0.0.1:54321/nonexistent?sslmode=disable", retryCfg)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("expected error for unreachable host, got nil")
	}

	if duration > 15*time.Second {
		t.Errorf("retry loop took too long (%v), expected bounded retry", duration)
	}
}
