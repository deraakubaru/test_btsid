package security

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const BcryptCost = 10

var ErrEmptyPassword = errors.New("password cannot be empty")

// HashPassword generates a bcrypt hash of a plaintext password using cost 10.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// ComparePassword compares a bcrypt hashed password with a plaintext password.
func ComparePassword(hashedPassword, password string) error {
	if password == "" || hashedPassword == "" {
		return bcrypt.ErrMismatchedHashAndPassword
	}
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
