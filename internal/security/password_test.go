package security

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword_Success(t *testing.T) {
	password := "supersecret"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected hashing error: %v", err)
	}

	if hash == "" {
		t.Fatal("expected non-empty hash string")
	}

	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("failed to inspect hash cost: %v", err)
	}

	if cost != BcryptCost {
		t.Errorf("expected bcrypt cost %d, got %d", BcryptCost, cost)
	}
}

func TestComparePassword_CorrectPassword(t *testing.T) {
	password := "supersecret"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected hashing error: %v", err)
	}

	err = ComparePassword(hash, password)
	if err != nil {
		t.Errorf("expected password comparison to succeed, got error: %v", err)
	}
}

func TestComparePassword_IncorrectPassword(t *testing.T) {
	password := "supersecret"
	wrongPassword := "wrongsecret"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected hashing error: %v", err)
	}

	err = ComparePassword(hash, wrongPassword)
	if err == nil {
		t.Error("expected password comparison to fail for wrong password, got nil")
	}
}

func TestHashPassword_Empty(t *testing.T) {
	_, err := HashPassword("")
	if err == nil {
		t.Error("expected error for empty password, got nil")
	}
}
