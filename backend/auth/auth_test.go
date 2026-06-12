package auth

import (
	"os"
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "supersecure123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Fatal("Hashed password cannot be empty")
	}

	if !CheckPasswordHash(password, hash) {
		t.Error("Password check failed on correct password")
	}

	if CheckPasswordHash("wrongpassword", hash) {
		t.Error("Password check succeeded on wrong password")
	}
}

func TestGenerateAndVerifyToken(t *testing.T) {
	os.Setenv("JWT_SECRET_KEY", "test-secret-key-at-least-32-bytes-long")
	defer os.Unsetenv("JWT_SECRET_KEY")

	userID := "user-123-abc"

	// 1. Access Token Testing
	token, err := GenerateAccessToken(userID)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	claims, err := VerifyToken(token, "access")
	if err != nil {
		t.Fatalf("Failed to verify access token: %v", err)
	}

	if claims.Sub != userID {
		t.Errorf("Expected subject to be %s, got %s", userID, claims.Sub)
	}

	if claims.Type != "access" {
		t.Errorf("Expected token type to be access, got %s", claims.Type)
	}

	// 2. Rejecting wrong token type
	_, err = VerifyToken(token, "refresh")
	if err == nil {
		t.Error("VerifyToken succeeded for wrong expected type")
	}

	// 3. Refresh Token Testing
	refToken, err := GenerateRefreshToken(userID)
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	refClaims, err := VerifyToken(refToken, "refresh")
	if err != nil {
		t.Fatalf("Failed to verify refresh token: %v", err)
	}

	if refClaims.Sub != userID {
		t.Errorf("Expected subject to be %s, got %s", userID, refClaims.Sub)
	}

	if refClaims.Type != "refresh" {
		t.Errorf("Expected token type to be refresh, got %s", refClaims.Type)
	}
}
