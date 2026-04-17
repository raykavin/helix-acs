package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the JWT payload fields for both access and refresh tokens.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWTService issues and validates HMAC-SHA256-signed JWTs.
type JWTService struct {
	secret           []byte
	expiresIn        time.Duration
	refreshExpiresIn time.Duration
}

// NewJWTService creates a JWTService with the given secret and expiry
// durations. secret must be non-empty.
func NewJWTService(secret string, expiresIn, refreshExpiresIn time.Duration) *JWTService {
	return &JWTService{
		secret:           []byte(secret),
		expiresIn:        expiresIn,
		refreshExpiresIn: refreshExpiresIn,
	}
}

// GenerateToken creates a new signed access token for the given user.
func (s *JWTService) GenerateToken(userID, username string) (string, error) {
	return s.sign(userID, username, s.expiresIn)
}

// GenerateRefreshToken creates a signed refresh token with the longer expiry.
func (s *JWTService) GenerateRefreshToken(userID, username string) (string, error) {
	return s.sign(userID, username, s.refreshExpiresIn)
}

// ValidateToken validates the token signature and expiry, then returns the
// embedded claims. Returns a descriptive error on any failure.
func (s *JWTService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return s.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	if claims.UserID == "" {
		return nil, errors.New("token missing user_id claim")
	}

	return claims, nil
}

// sign is a shared helper that builds and signs a token with a given TTL.
func (s *JWTService) sign(userID, username string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
