package auth

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	password := "supersecure123"
	hash, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	assert.True(t, CheckPasswordHash(password, hash), "correct password should verify")
	assert.False(t, CheckPasswordHash("wrongpassword", hash), "wrong password should fail")
}

func TestGenerateAndVerifyToken(t *testing.T) {
	os.Setenv("JWT_SECRET_KEY", "test-secret-key-at-least-32-bytes-long")
	defer os.Unsetenv("JWT_SECRET_KEY")

	userID := "user-123-abc"

	token, err := GenerateAccessToken(userID)
	require.NoError(t, err)

	claims, err := VerifyToken(token, "access")
	require.NoError(t, err)
	assert.Equal(t, userID, claims.Sub)
	assert.Equal(t, "access", claims.Type)

	// Wrong expected type must be rejected.
	_, err = VerifyToken(token, "refresh")
	require.Error(t, err)

	refToken, err := GenerateRefreshToken(userID)
	require.NoError(t, err)

	refClaims, err := VerifyToken(refToken, "refresh")
	require.NoError(t, err)
	assert.Equal(t, userID, refClaims.Sub)
	assert.Equal(t, "refresh", refClaims.Type)
}
