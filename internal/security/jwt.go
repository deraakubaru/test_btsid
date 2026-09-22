package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var (
	ErrInvalidToken          = errors.New("invalid or tampered token")
	ErrExpiredToken          = errors.New("token has expired")
	ErrInvalidTokenType      = errors.New("invalid token type")
	ErrUnexpectedSigningAlgo = errors.New("unexpected signing algorithm, expected HS256")
)

// JWTClaims defines the standard JWT claim payload matching SPEC.md.
type JWTClaims struct {
	UserID    int64  `json:"sub"`
	Username  string `json:"username"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// TokenPair contains the generated access and refresh tokens.
type TokenPair struct {
	AuthenticationToken string `json:"authentication_token"`
	RefreshToken        string `json:"refresh_token"`
	TokenType           string `json:"token_type"`
	ExpiresIn           int64  `json:"expires_in"`
}

// GenerateTokens creates a dual JWT pair (access token and refresh token) signed with HS256.
func GenerateTokens(userID int64, username string, secret string, accessTTLMinutes, refreshTTLDays int) (*TokenPair, error) {
	if secret == "" {
		return nil, errors.New("JWT secret cannot be empty")
	}

	now := time.Now().UTC()
	accessExpiry := now.Add(time.Duration(accessTTLMinutes) * time.Minute)
	refreshExpiry := now.Add(time.Duration(refreshTTLDays) * 24 * time.Hour)

	// Access Token Claims
	accessClaims := &JWTClaims{
		UserID:    userID,
		Username:  username,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
		},
	}

	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := accessTokenObj.SignedString([]byte(secret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh Token Claims
	refreshClaims := &JWTClaims{
		UserID:    userID,
		Username:  username,
		TokenType: TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
		},
	}

	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := refreshTokenObj.SignedString([]byte(secret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AuthenticationToken: accessToken,
		RefreshToken:        refreshToken,
		TokenType:           "Bearer",
		ExpiresIn:           int64(accessTTLMinutes * 60),
	}, nil
}

// ValidateToken parses, verifies HS256 signature, checks expiration, and validates expected token_type.
func ValidateToken(tokenString string, secret string, expectedType string) (*JWTClaims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}
	if secret == "" {
		return nil, errors.New("JWT secret cannot be empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Explicitly require/validate HS256 algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Header["alg"] != jwt.SigningMethodHS256.Alg() {
			return nil, ErrUnexpectedSigningAlgo
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if expectedType != "" && claims.TokenType != expectedType {
		return nil, ErrInvalidTokenType
	}

	return claims, nil
}
