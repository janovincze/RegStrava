package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/regstrava/regstrava/internal/api/middleware"
	"github.com/regstrava/regstrava/internal/domain"
	"github.com/regstrava/regstrava/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// FunderHandler handles funder-related HTTP requests
type FunderHandler struct {
	funderRepo *repository.FunderRepository
}

// NewFunderHandler creates a new funder handler
func NewFunderHandler(funderRepo *repository.FunderRepository) *FunderHandler {
	return &FunderHandler{
		funderRepo: funderRepo,
	}
}

// RegisterRequest represents a funder registration request
type RegisterRequest struct {
	Name          string `json:"name" validate:"required"`
	Email         string `json:"email" validate:"required,email"`
	Company       string `json:"company"`
	TrackFundings bool   `json:"track_fundings"`
}

// RegisterResponse represents a funder registration response
type RegisterResponse struct {
	FunderID      string `json:"funder_id"`
	Name          string `json:"name"`
	APIKey        string `json:"api_key"`
	OAuthClientID string `json:"oauth_client_id"`
	OAuthSecret   string `json:"oauth_secret"`
	CreatedAt     string `json:"created_at"`
	Message       string `json:"message"`
}

// Register handles POST /api/v1/funders/register
func (h *FunderHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Name is required")
		return
	}

	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "Email is required")
		return
	}

	// Generate API key
	apiKey, err := generateSecureToken(32)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate API key")
		return
	}

	apiKeyHash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash API key")
		return
	}

	// Generate OAuth credentials
	oauthClientID := uuid.New().String()
	oauthSecret, err := generateSecureToken(32)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate OAuth secret")
		return
	}

	oauthSecretHash, err := bcrypt.GenerateFromPassword([]byte(oauthSecret), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash OAuth secret")
		return
	}

	// Create funder
	funderID := uuid.New()
	oauthSecretHashStr := string(oauthSecretHash)
	now := time.Now()
	funder := &domain.Funder{
		ID:                    funderID,
		Name:                  req.Name,
		Email:                 &req.Email,
		Company:               &req.Company,
		APIKeyHash:            string(apiKeyHash),
		OAuthClientID:         &oauthClientID,
		OAuthSecretHash:       &oauthSecretHashStr,
		TrackFundings:         req.TrackFundings,
		RateLimitDaily:        1000,
		RateLimitMonthly:      20000,
		SubscriptionStatus:    domain.SubscriptionStatusActive,
		SubscriptionStartedAt: &now,
		CreatedAt:             now,
		IsActive:              true,
	}

	if err := h.funderRepo.Create(r.Context(), funder); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create funder")
		return
	}

	response := RegisterResponse{
		FunderID:      funderID.String(),
		Name:          req.Name,
		APIKey:        apiKey,
		OAuthClientID: oauthClientID,
		OAuthSecret:   oauthSecret,
		CreatedAt:     funder.CreatedAt.Format(time.RFC3339),
		Message:       "Save your API key and OAuth credentials securely. They will not be shown again.",
	}

	respondJSON(w, http.StatusCreated, response)
}

// GetProfile handles GET /api/v1/funders/me
func (h *FunderHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	funderID := middleware.GetFunderID(r.Context())
	if funderID == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	funder, err := h.funderRepo.FindByID(r.Context(), *funderID)
	if err != nil || funder == nil {
		respondError(w, http.StatusNotFound, "Funder not found")
		return
	}

	// Return safe profile data (no secrets)
	profile := map[string]interface{}{
		"id":                 funder.ID.String(),
		"name":               funder.Name,
		"track_fundings":     funder.TrackFundings,
		"rate_limit_daily":   funder.RateLimitDaily,
		"rate_limit_monthly": funder.RateLimitMonthly,
		"created_at":         funder.CreatedAt.Format(time.RFC3339),
		"is_active":          funder.IsActive,
	}

	if funder.OAuthClientID != nil {
		profile["oauth_client_id"] = *funder.OAuthClientID
	}

	respondJSON(w, http.StatusOK, profile)
}

// RegenerateAPIKey handles POST /api/v1/funders/me/regenerate-api-key
func (h *FunderHandler) RegenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	funderID := middleware.GetFunderID(r.Context())
	if funderID == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	funder, err := h.funderRepo.FindByID(r.Context(), *funderID)
	if err != nil || funder == nil {
		respondError(w, http.StatusNotFound, "Funder not found")
		return
	}

	// Generate new API key
	apiKey, err := generateSecureToken(32)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate API key")
		return
	}

	apiKeyHash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash API key")
		return
	}

	// Update funder with new API key hash
	funder.APIKeyHash = string(apiKeyHash)
	if err := h.funderRepo.UpdateAPIKey(r.Context(), funder.ID, funder.APIKeyHash); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update API key")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"api_key": apiKey,
		"message": "Save your new API key securely. It will not be shown again.",
	})
}

// UpdateProfile handles PATCH /api/v1/funders/me
func (h *FunderHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	funderID := middleware.GetFunderID(r.Context())
	if funderID == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var updates struct {
		Name          *string `json:"name"`
		TrackFundings *bool   `json:"track_fundings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	funder, err := h.funderRepo.FindByID(r.Context(), *funderID)
	if err != nil || funder == nil {
		respondError(w, http.StatusNotFound, "Funder not found")
		return
	}

	if updates.Name != nil {
		funder.Name = *updates.Name
	}
	if updates.TrackFundings != nil {
		funder.TrackFundings = *updates.TrackFundings
	}

	if err := h.funderRepo.Update(r.Context(), funder); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Profile updated successfully",
	})
}

// GetUsageStats handles GET /api/v1/funders/me/usage
func (h *FunderHandler) GetUsageStats(w http.ResponseWriter, r *http.Request) {
	funderID := middleware.GetFunderID(r.Context())
	if funderID == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get rate limit headers from context if available
	// For now, return placeholder - in production, query Redis
	funder, err := h.funderRepo.FindByID(r.Context(), *funderID)
	if err != nil || funder == nil {
		respondError(w, http.StatusNotFound, "Funder not found")
		return
	}

	usage := map[string]interface{}{
		"daily_limit":   funder.RateLimitDaily,
		"monthly_limit": funder.RateLimitMonthly,
		"message":       "Check X-RateLimit-* headers on API responses for current usage",
	}

	respondJSON(w, http.StatusOK, usage)
}

// ListFunders handles GET /api/v1/admin/funders (admin only - for future use)
func (h *FunderHandler) ListFunders(w http.ResponseWriter, r *http.Request) {
	// This would be admin-only in production
	funders, err := h.funderRepo.FindAll(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list funders")
		return
	}

	// Return safe data only
	var safeFunders []map[string]interface{}
	for _, f := range funders {
		safeFunders = append(safeFunders, map[string]interface{}{
			"id":         f.ID.String(),
			"name":       f.Name,
			"is_active":  f.IsActive,
			"created_at": f.CreatedAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, safeFunders)
}

// GetFunderByID handles GET /api/v1/admin/funders/{id}
func (h *FunderHandler) GetFunderByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid funder ID")
		return
	}

	funder, err := h.funderRepo.FindByID(r.Context(), id)
	if err != nil || funder == nil {
		respondError(w, http.StatusNotFound, "Funder not found")
		return
	}

	profile := map[string]interface{}{
		"id":                 funder.ID.String(),
		"name":               funder.Name,
		"track_fundings":     funder.TrackFundings,
		"rate_limit_daily":   funder.RateLimitDaily,
		"rate_limit_monthly": funder.RateLimitMonthly,
		"created_at":         funder.CreatedAt.Format(time.RFC3339),
		"is_active":          funder.IsActive,
	}

	respondJSON(w, http.StatusOK, profile)
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
