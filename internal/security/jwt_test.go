package security

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test_super_secret_key"

func TestGenerateAndValidateTokens_ValidAccessAndRefresh(t *testing.T) {
	pair, err := GenerateTokens(1, "jhon_doe", testSecret, 15, 7)
	if err != nil {
		t.Fatalf("unexpected token generation error: %v", err)
	}

	if pair.AuthenticationToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected non-empty tokens in pair")
	}

	// Validate Access Token
	accessClaims, err := ValidateToken(pair.AuthenticationToken, testSecret, TokenTypeAccess)
	if err != nil {
		t.Fatalf("unexpected access token validation error: %v", err)
	}

	if accessClaims.UserID != 1 {
		t.Errorf("expected UserID 1, got %d", accessClaims.UserID)
	}
	if accessClaims.Username != "jhon_doe" {
		t.Errorf("expected Username 'jhon_doe', got '%s'", accessClaims.Username)
	}
	if accessClaims.TokenType != TokenTypeAccess {
		t.Errorf("expected TokenType 'access', got '%s'", accessClaims.TokenType)
	}

	// Validate Refresh Token
	refreshClaims, err := ValidateToken(pair.RefreshToken, testSecret, TokenTypeRefresh)
	if err != nil {
		t.Fatalf("unexpected refresh token validation error: %v", err)
	}

	if refreshClaims.UserID != 1 || refreshClaims.Username != "jhon_doe" || refreshClaims.TokenType != TokenTypeRefresh {
		t.Errorf("refresh token claims mismatch: %+v", refreshClaims)
	}
}

func TestValidateToken_MismatchedTokenType(t *testing.T) {
	pair, err := GenerateTokens(1, "jhon_doe", testSecret, 15, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Attempt to validate access token as refresh token
	_, err = ValidateToken(pair.AuthenticationToken, testSecret, TokenTypeRefresh)
	if !errors.Is(err, ErrInvalidTokenType) {
		t.Errorf("expected ErrInvalidTokenType when passing access token as refresh token, got: %v", err)
	}

	// Attempt to validate refresh token as access token
	_, err = ValidateToken(pair.RefreshToken, testSecret, TokenTypeAccess)
	if !errors.Is(err, ErrInvalidTokenType) {
		t.Errorf("expected ErrInvalidTokenType when passing refresh token as access token, got: %v", err)
	}
}

func TestValidateToken_InvalidSecret(t *testing.T) {
	pair, err := GenerateTokens(1, "jhon_doe", testSecret, 15, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = ValidateToken(pair.AuthenticationToken, "wrong_secret", TokenTypeAccess)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken for invalid secret, got: %v", err)
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Generate token expired 1 minute ago
	now := time.Now().UTC().Add(-5 * time.Minute)
	claims := &JWTClaims{
		UserID:    1,
		Username:  "jhon_doe",
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Minute)), // expired 4 mins ago
		},
	}

	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := tokenObj.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("unexpected error signing token: %v", err)
	}

	_, err = ValidateToken(tokenStr, testSecret, TokenTypeAccess)
	if !errors.Is(err, ErrExpiredToken) {
		t.Errorf("expected ErrExpiredToken, got: %v", err)
	}
}

func TestValidateToken_WrongAlgorithm(t *testing.T) {
	claims := &JWTClaims{
		UserID:    1,
		Username:  "jhon_doe",
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}

	tokenObj := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, err := tokenObj.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = ValidateToken(tokenStr, testSecret, TokenTypeAccess)
	if err == nil {
		t.Error("expected error for token signed with unsafe 'none' algorithm, got nil")
	}
}
