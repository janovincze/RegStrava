package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/regstrava/regstrava/internal/api/middleware"
	"github.com/regstrava/regstrava/internal/repository"
	"github.com/regstrava/regstrava/internal/service"
)

// SubscriptionHandler handles subscription-related HTTP requests
type SubscriptionHandler struct {
	subscriptionRepo *repository.SubscriptionRepository
	usageService     *service.UsageService
}

// NewSubscriptionHandler creates a new subscription handler
func NewSubscriptionHandler(subscriptionRepo *repository.SubscriptionRepository, usageService *service.UsageService) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionRepo: subscriptionRepo,
		usageService:     usageService,
	}
}

// ListTiers handles GET /api/v1/subscription-tiers
func (h *SubscriptionHandler) ListTiers(w http.ResponseWriter, r *http.Request) {
	tiers, err := h.subscriptionRepo.ListActive(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list subscription tiers")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tiers": tiers,
	})
}

// GetUsage handles GET /api/v1/funders/me/usage
func (h *SubscriptionHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	funderID := middleware.GetFunderID(r.Context())
	if funderID == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	usage, err := h.usageService.GetUsageStats(r.Context(), *funderID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get usage stats")
		return
	}

	respondJSON(w, http.StatusOK, usage)
}

// GetUsageHistory handles GET /api/v1/funders/me/usage/history
func (h *SubscriptionHandler) GetUsageHistory(w http.ResponseWriter, r *http.Request) {
	funderID := middleware.GetFunderID(r.Context())
	if funderID == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Default to 12 months of history
	months := 12

	history, err := h.usageService.GetUsageHistory(r.Context(), *funderID, months)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get usage history")
		return
	}

	respondJSON(w, http.StatusOK, history)
}

// GetSubscription handles GET /api/v1/funders/me/subscription
func (h *SubscriptionHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	funderID := middleware.GetFunderID(r.Context())
	if funderID == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tier, err := h.subscriptionRepo.GetFunderTier(r.Context(), *funderID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get subscription")
		return
	}

	if tier == nil {
		tier, _ = h.subscriptionRepo.GetDefaultTier(r.Context())
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tier":   tier,
		"status": "active",
	})
}

// RequestUpgrade handles POST /api/v1/funders/me/subscription/upgrade
func (h *SubscriptionHandler) RequestUpgrade(w http.ResponseWriter, r *http.Request) {
	funderID := middleware.GetFunderID(r.Context())
	if funderID == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		TierName string `json:"tier_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Find the requested tier
	tier, err := h.subscriptionRepo.FindByName(r.Context(), req.TierName)
	if err != nil || tier == nil {
		respondError(w, http.StatusBadRequest, "Invalid subscription tier")
		return
	}

	// For now, just return upgrade information
	// Actual upgrade would involve payment processing
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"message":     "Upgrade request received. Please contact sales for Enterprise tier or complete payment for other tiers.",
		"tier":        tier,
		"upgrade_url": "/contact-sales", // Would be a payment URL for Basic/Premium
	})
}
