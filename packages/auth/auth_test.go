package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-for-unit-tests"

func newTestJWTService() *JWTService {
	return NewJWTService(testSecret, 24*time.Hour, 168*time.Hour)
}

// GenerateToken + ValidateToken (happy path)

func TestJWTGenerateAndValidate(t *testing.T) {
	svc := newTestJWTService()

	token, err := svc.GenerateToken("user-001", "alice")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Token should have three dot-separated parts (header.payload.sig)
	parts := strings.Split(token, ".")
	assert.Len(t, parts, 3, "JWT must have three parts")

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)

	assert.Equal(t, "user-001", claims.UserID)
	assert.Equal(t, "alice", claims.Username)
	assert.False(t, claims.ExpiresAt.IsZero(), "ExpiresAt must be set")
	assert.True(t, claims.ExpiresAt.After(time.Now()), "token should not be expired")
}

// Expired token

func TestJWTExpiredToken(t *testing.T) {
	// Create a service with a 1 ms access-token TTL.
	svc := NewJWTService(testSecret, time.Millisecond, 168*time.Hour)

	token, err := svc.GenerateToken("user-002", "bob")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Wait long enough for the token to expire.
	time.Sleep(10 * time.Millisecond)

	_, err = svc.ValidateToken(token)
	assert.Error(t, err, "expired token should return an error")
	assert.Contains(t, err.Error(), "parse token")
}

// Invalid signature (tampered token)

func TestJWTInvalidSignature(t *testing.T) {
	svc := newTestJWTService()

	token, err := svc.GenerateToken("user-003", "carol")
	require.NoError(t, err)

	// Tamper with the signature part (last segment).
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	parts[2] = parts[2] + "tampered"
	tampered := strings.Join(parts, ".")

	_, err = svc.ValidateToken(tampered)
	assert.Error(t, err, "tampered token should fail validation")
}

// Refresh token

func TestJWTRefreshToken(t *testing.T) {
	svc := newTestJWTService()

	refreshToken, err := svc.GenerateRefreshToken("user-004", "dave")
	require.NoError(t, err)
	require.NotEmpty(t, refreshToken)

	claims, err := svc.ValidateToken(refreshToken)
	require.NoError(t, err)
	require.NotNil(t, claims)

	assert.Equal(t, "user-004", claims.UserID)
	assert.Equal(t, "dave", claims.Username)

	// Refresh token should expire later than an access token would.
	accessExpiry := time.Now().Add(24 * time.Hour)
	assert.True(t, claims.ExpiresAt.After(accessExpiry),
		"refresh token expiry should be beyond the access token expiry")
}

// Wrong secret

func TestJWTWrongSecret(t *testing.T) {
	svcA := NewJWTService("secret-A", 24*time.Hour, 168*time.Hour)
	svcB := NewJWTService("secret-B", 24*time.Hour, 168*time.Hour)

	token, err := svcA.GenerateToken("user-005", "eve")
	require.NoError(t, err)

	_, err = svcB.ValidateToken(token)
	assert.Error(t, err, "token signed with a different secret must fail")
}

// Empty user_id claim

func TestJWTEmptyUserID(t *testing.T) {
	svc := newTestJWTService()

	// sign() is not exported, but we can use GenerateToken with an empty id.
	token, err := svc.GenerateToken("", "nobody")
	require.NoError(t, err)

	_, err = svc.ValidateToken(token)
	assert.Error(t, err, "token with empty user_id should be rejected")
	assert.Contains(t, err.Error(), "user_id")
}
