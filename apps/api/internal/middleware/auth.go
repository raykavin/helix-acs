package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/raykavin/helix-acs/packages/auth"
)

type contextKey string

// ClaimsKey is the context key under which validated JWT claims are stored.
const ClaimsKey contextKey = "claims"

// JWTAuth returns middleware that validates a Bearer token on every request.
// On success it stores the parsed claims in the request context and calls next.
// On failure it responds with HTTP 401 and a JSON error body.
func JWTAuth(jwtSvc *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, "missing Authorization header")
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(authHeader, prefix) {
				writeAuthError(w, "authorization header must use Bearer scheme")
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, prefix)
			if tokenStr == "" {
				writeAuthError(w, "bearer token is empty")
				return
			}

			claims, err := jwtSvc.ValidateToken(tokenStr)
			if err != nil {
				writeAuthError(w, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims retrieves the JWT claims stored in ctx by JWTAuth. Returns nil if
// the context does not contain claims.
func GetClaims(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(ClaimsKey).(*auth.Claims)
	return claims
}

// writeAuthError sends a JSON 401 response.
func writeAuthError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
