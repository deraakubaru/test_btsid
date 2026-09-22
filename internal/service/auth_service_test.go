package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"btsid/internal/domain"
	"btsid/internal/security"
)

type mockUserRepository struct {
	mu    sync.RWMutex
	users map[string]*domain.User
	idSeq int64
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (m *mockUserRepository) CreateUser(ctx context.Context, username, passwordHash string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[username]; exists {
		return nil, domain.NewConflictError("username already exists")
	}

	m.idSeq++
	user := &domain.User{
		ID:           m.idSeq,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	m.users[username] = user
	return user, nil
}

func (m *mockUserRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[username]
	if !exists {
		return nil, domain.NewNotFoundError("user not found")
	}
	return user, nil
}

func (m *mockUserRepository) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.NewNotFoundError("user not found")
}

func TestAuthService_Register(t *testing.T) {
	repo := newMockUserRepository()
	authService := NewAuthService(repo, "secret123", 15, 7)
	ctx := context.Background()

	t.Run("successful registration", func(t *testing.T) {
		req := domain.RegisterRequest{
			Username:             "john",
			Password:             "secret",
			PasswordConfirmation: "secret",
		}

		res, err := authService.Register(ctx, req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.Username != "john" {
			t.Errorf("expected username john, got %s", res.Username)
		}
	})

	t.Run("empty username", func(t *testing.T) {
		req := domain.RegisterRequest{
			Username:             "   ",
			Password:             "secret",
			PasswordConfirmation: "secret",
		}
		_, err := authService.Register(ctx, req)
		if err == nil {
			t.Fatal("expected error for empty username, got nil")
		}
		var appErr *domain.AppError
		if !errors.As(err, &appErr) || !errors.Is(appErr.Code, domain.ErrValidation) {
			t.Errorf("expected validation error, got %v", err)
		}
	})

	t.Run("empty password", func(t *testing.T) {
		req := domain.RegisterRequest{
			Username:             "alice",
			Password:             "",
			PasswordConfirmation: "",
		}
		_, err := authService.Register(ctx, req)
		if err == nil {
			t.Fatal("expected error for empty password, got nil")
		}
		var appErr *domain.AppError
		if !errors.As(err, &appErr) || !errors.Is(appErr.Code, domain.ErrValidation) {
			t.Errorf("expected validation error, got %v", err)
		}
	})

	t.Run("mismatched passwords", func(t *testing.T) {
		req := domain.RegisterRequest{
			Username:             "bob",
			Password:             "secret1",
			PasswordConfirmation: "secret2",
		}
		_, err := authService.Register(ctx, req)
		if err == nil {
			t.Fatal("expected error for password mismatch, got nil")
		}
		var appErr *domain.AppError
		if !errors.As(err, &appErr) || !errors.Is(appErr.Code, domain.ErrValidation) {
			t.Errorf("expected validation error, got %v", err)
		}
	})

	t.Run("duplicate username", func(t *testing.T) {
		req := domain.RegisterRequest{
			Username:             "john",
			Password:             "secret",
			PasswordConfirmation: "secret",
		}
		_, err := authService.Register(ctx, req)
		if err == nil {
			t.Fatal("expected error for duplicate username, got nil")
		}
		var appErr *domain.AppError
		if !errors.As(err, &appErr) || !errors.Is(appErr.Code, domain.ErrConflict) {
			t.Errorf("expected conflict error, got %v", err)
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	repo := newMockUserRepository()
	authService := NewAuthService(repo, "secret123", 15, 7)
	ctx := context.Background()

	// Register a user first
	_, err := authService.Register(ctx, domain.RegisterRequest{
		Username:             "john",
		Password:             "secret",
		PasswordConfirmation: "secret",
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	t.Run("successful login", func(t *testing.T) {
		tokens, err := authService.Login(ctx, domain.LoginRequest{
			Username: "john",
			Password: "secret",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if tokens.AuthenticationToken == "" || tokens.RefreshToken == "" {
			t.Error("expected non-empty tokens")
		}

		// Validate generated access token claims
		claims, err := security.ValidateToken(tokens.AuthenticationToken, "secret123", security.TokenTypeAccess)
		if err != nil {
			t.Fatalf("failed to validate generated access token: %v", err)
		}
		if claims.Username != "john" {
			t.Errorf("expected username john in claims, got %s", claims.Username)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		_, err := authService.Login(ctx, domain.LoginRequest{
			Username: "john",
			Password: "wrongpassword",
		})
		if err == nil {
			t.Fatal("expected unauthorized error for wrong password, got nil")
		}
		var appErr *domain.AppError
		if !errors.As(err, &appErr) || !errors.Is(appErr.Code, domain.ErrUnauthorized) {
			t.Errorf("expected unauthorized error, got %v", err)
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		_, err := authService.Login(ctx, domain.LoginRequest{
			Username: "nobody",
			Password: "secret",
		})
		if err == nil {
			t.Fatal("expected unauthorized error for unknown user, got nil")
		}
		var appErr *domain.AppError
		if !errors.As(err, &appErr) || !errors.Is(appErr.Code, domain.ErrUnauthorized) {
			t.Errorf("expected unauthorized error, got %v", err)
		}
	})
}
