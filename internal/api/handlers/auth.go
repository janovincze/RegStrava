package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/regstrava/regstrava/internal/domain"
	"github.com/regstrava/regstrava/internal/service"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Token handles POST /api/v1/oauth/token
func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	var req domain.OAuthTokenRequest

	// Support both JSON and form-urlencoded
	contentType := r.Header.Get("Content-Type")

	if contentType == "application/x-www-form-urlencoded" {
		if err := r.ParseForm(); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid form data")
			return
		}
		req.GrantType = r.FormValue("grant_type")
		req.ClientID = r.FormValue("client_id")
		req.ClientSecret = r.FormValue("client_secret")
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
	}

	// Validate grant type
	if req.GrantType != "client_credentials" {
		respondError(w, http.StatusBadRequest, "Only client_credentials grant type is supported")
		return
	}

	if req.ClientID == "" || req.ClientSecret == "" {
		respondError(w, http.StatusBadRequest, "client_id and client_secret are required")
		return
	}

	// Validate credentials
	funder, err := h.authService.ValidateOAuthCredentials(r.Context(), req.ClientID, req.ClientSecret)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate token
	tokenResp, err := h.authService.GenerateToken(funder)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	respondJSON(w, http.StatusOK, tokenResp)
}
