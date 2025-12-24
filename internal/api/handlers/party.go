package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/regstrava/regstrava/internal/api/middleware"
	"github.com/regstrava/regstrava/internal/domain"
	"github.com/regstrava/regstrava/internal/service"
)

// PartyHandler handles party-related HTTP requests
type PartyHandler struct {
	partyService *service.PartyService
}

// NewPartyHandler creates a new party handler
func NewPartyHandler(partyService *service.PartyService) *PartyHandler {
	return &PartyHandler{
		partyService: partyService,
	}
}

// Check handles POST /api/v1/party/check
func (h *PartyHandler) Check(w http.ResponseWriter, r *http.Request) {
	var req domain.PartyCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.TaxID == "" || req.Country == "" {
		respondError(w, http.StatusBadRequest, "tax_id and country are required")
		return
	}

	if req.PartyType != domain.PartyTypeBuyer && req.PartyType != domain.PartyTypeSupplier {
		respondError(w, http.StatusBadRequest, "party_type must be 'buyer' or 'supplier'")
		return
	}

	// Get funder ID from context
	funderID := middleware.GetFunderID(r.Context())
	if funderID == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Check track_funder preference (default to false for party checks)
	trackFunder := false

	response, err := h.partyService.CheckParty(r.Context(), &req, *funderID, trackFunder)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check party")
		return
	}

	respondJSON(w, http.StatusOK, response)
}

// Register handles POST /api/v1/party/register
func (h *PartyHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.PartyRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.TaxID == "" || req.Country == "" {
		respondError(w, http.StatusBadRequest, "tax_id and country are required")
		return
	}

	if req.PartyType != domain.PartyTypeBuyer && req.PartyType != domain.PartyTypeSupplier {
		respondError(w, http.StatusBadRequest, "party_type must be 'buyer' or 'supplier'")
		return
	}

	// Get funder ID from context
	funderID := middleware.GetFunderID(r.Context())
	if funderID == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	response, err := h.partyService.RegisterParty(r.Context(), &req, *funderID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to register party")
		return
	}

	respondJSON(w, http.StatusOK, response)
}

// History handles GET /api/v1/party/history
func (h *PartyHandler) History(w http.ResponseWriter, r *http.Request) {
	var req domain.PartyHistoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.TaxID == "" || req.Country == "" {
		respondError(w, http.StatusBadRequest, "tax_id and country are required")
		return
	}

	if req.PartyType != domain.PartyTypeBuyer && req.PartyType != domain.PartyTypeSupplier {
		respondError(w, http.StatusBadRequest, "party_type must be 'buyer' or 'supplier'")
		return
	}

	// Get funder ID from context
	funderID := middleware.GetFunderID(r.Context())
	if funderID == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get funder's max lookback days (would typically come from funder settings)
	// For now, use a default of 30 days
	maxLookbackDays := 30

	response, err := h.partyService.QueryPartyHistory(r.Context(), &req, *funderID, maxLookbackDays)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to query party history")
		return
	}

	respondJSON(w, http.StatusOK, response)
}
