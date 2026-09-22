package service

import (
	"context"
	"errors"
	"strings"

	"btsid/internal/domain"
	"btsid/internal/repository"
	"btsid/internal/security"
)

type AuthService struct {
	userRepo         repository.UserRepository
	jwtSecret        string
	accessTTLMinutes int
	refreshTTLDays   int
}

func NewAuthService(userRepo repository.UserRepository, jwtSecret string, accessTTLMinutes, refreshTTLDays int) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		jwtSecret:        jwtSecret,
		accessTTLMinutes: accessTTLMinutes,
		refreshTTLDays:   refreshTTLDays,
	}
}

// Register processes user registration according to business rules.
func (s *AuthService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.UserResponse, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return nil, domain.NewValidationError("username is required")
	}
	if req.Password == "" {
		return nil, domain.NewValidationError("password is required")
	}
	if req.PasswordConfirmation == "" {
		return nil, domain.NewValidationError("password_confirmation is required")
	}
	if req.Password != req.PasswordConfirmation {
		return nil, domain.NewValidationError("password and password_confirmation do not match")
	}

	hashedPassword, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.CreateUser(ctx, username, hashedPassword)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) && errors.Is(appErr.Code, domain.ErrConflict) {
			return nil, domain.NewConflictError("username already exists")
		}
		if errors.Is(err, domain.ErrConflict) {
			return nil, domain.NewConflictError("username already exists")
		}
		return nil, err
	}

	return &domain.UserResponse{
		Username: user.Username,
	}, nil
}

// Login authenticates a user and returns a token pair.
func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*security.TokenPair, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return nil, domain.NewValidationError("username is required")
	}
	if req.Password == "" {
		return nil, domain.NewValidationError("password is required")
	}

	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) && errors.Is(appErr.Code, domain.ErrNotFound) {
			return nil, domain.NewUnauthorizedError("invalid username or password")
		}
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewUnauthorizedError("invalid username or password")
		}
		return nil, err
	}

	if err := security.ComparePassword(user.PasswordHash, req.Password); err != nil {
		return nil, domain.NewUnauthorizedError("invalid username or password")
	}

	tokens, err := security.GenerateTokens(user.ID, user.Username, s.jwtSecret, s.accessTTLMinutes, s.refreshTTLDays)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}
