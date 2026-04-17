package handler

import (
	"encoding/json"
	"net/http"

	"github.com/raykavin/helix-acs/packages/auth"
)

// AuthHandler handles login and token-refresh endpoints.
type AuthHandler struct {
	jwtSvc    *auth.JWTService
	adminUser string
	adminPass string
}

// NewAuthHandler creates an AuthHandler. In production a user repository would
// replace the hardcoded admin credentials.
func NewAuthHandler(jwtSvc *auth.JWTService, adminUser, adminPass string) *AuthHandler {
	return &AuthHandler{
		jwtSvc:    jwtSvc,
		adminUser: adminUser,
		adminPass: adminPass,
	}
}

// loginRequest is the JSON body expected by Login.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginResponse is the JSON body returned by a successful Login.
type loginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// refreshRequest is the JSON body expected by Refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// refreshResponse is the JSON body returned by a successful Refresh.
type refreshResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
}

// Login handles POST /api/v1/auth/login.
// It validates credentials and returns a JWT access token plus refresh token.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	if req.Username != h.adminUser || req.Password != h.adminPass {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.jwtSvc.GenerateToken(req.Username, req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	refreshToken, err := h.jwtSvc.GenerateRefreshToken(req.Username, req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    86400,
	})
}

// Refresh handles POST /api/v1/auth/refresh.
// It validates the refresh token and issues a new access token.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	claims, err := h.jwtSvc.ValidateToken(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	token, err := h.jwtSvc.GenerateToken(claims.UserID, claims.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, refreshResponse{
		Token:     token,
		ExpiresIn: 86400,
	})
}
