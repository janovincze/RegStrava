package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/regstrava/regstrava/internal/api/middleware"
	"github.com/regstrava/regstrava/internal/domain"
	"github.com/regstrava/regstrava/internal/service"
)

// InvoiceHandler handles invoice-related HTTP requests
type InvoiceHandler struct {
	invoiceService *service.InvoiceService
	hashService    *service.HashService
}

// NewInvoiceHandler creates a new invoice handler
func NewInvoiceHandler(invoiceService *service.InvoiceService, hashService *service.HashService) *InvoiceHandler {
	return &InvoiceHandler{
		invoiceService: invoiceService,
		hashService:    hashService,
	}
}

// Check handles POST /api/v1/invoices/check
func (h *InvoiceHandler) Check(w http.ResponseWriter, r *http.Request) {
	var req domain.InvoiceCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Hashes) == 0 {
		respondError(w, http.StatusBadRequest, "At least one hash is required")
		return
	}

	if len(req.Hashes) > 4 {
		respondError(w, http.StatusBadRequest, "Maximum 4 hashes allowed")
		return
	}

	result, err := h.invoiceService.CheckInvoice(r.Context(), req.Hashes)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check invoice")
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// CheckRaw handles POST /api/v1/invoices/check-raw
func (h *InvoiceHandler) CheckRaw(w http.ResponseWriter, r *http.Request) {
	var req domain.InvoiceCheckRawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields (support both new and deprecated field names)
	documentID := req.DocumentID
	if documentID == "" {
		documentID = req.InvoiceNumber // Backward compatibility
	}
	supplierTaxID := req.SupplierTaxID
	if supplierTaxID == "" {
		supplierTaxID = req.IssuerTaxID // Backward compatibility
	}
	supplierCountry := req.SupplierCountry
	if supplierCountry == "" {
		supplierCountry = req.IssuerCountry // Backward compatibility
	}

	if supplierTaxID == "" || supplierCountry == "" {
		respondError(w, http.StatusBadRequest, "supplier_tax_id and supplier_country are required")
		return
	}

	if req.BuyerTaxID == "" || req.BuyerCountry == "" {
		respondError(w, http.StatusBadRequest, "buyer_tax_id and buyer_country are required")
		return
	}

	// Use the enhanced check method that includes party-level checks
	result, err := h.invoiceService.CheckInvoiceRawWithParty(r.Context(), &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check invoice")
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// Register handles POST /api/v1/invoices/register
func (h *InvoiceHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.InvoiceRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Hashes) == 0 {
		respondError(w, http.StatusBadRequest, "At least one hash is required")
		return
	}

	if len(req.Hashes) > 4 {
		respondError(w, http.StatusBadRequest, "Maximum 4 hashes allowed")
		return
	}

	if req.FundingDate == "" {
		respondError(w, http.StatusBadRequest, "funding_date is required")
		return
	}

	// Get funder ID from context
	funderID := middleware.GetFunderID(r.Context())

	result, err := h.invoiceService.RegisterInvoice(r.Context(), &req, funderID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to register invoice")
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

// RegisterRaw handles POST /api/v1/invoices/register-raw
func (h *InvoiceHandler) RegisterRaw(w http.ResponseWriter, r *http.Request) {
	var req domain.InvoiceRegisterRawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields (support both new and deprecated field names)
	supplierTaxID := req.SupplierTaxID
	if supplierTaxID == "" {
		supplierTaxID = req.IssuerTaxID // Backward compatibility
	}
	supplierCountry := req.SupplierCountry
	if supplierCountry == "" {
		supplierCountry = req.IssuerCountry // Backward compatibility
	}

	if supplierTaxID == "" || supplierCountry == "" {
		respondError(w, http.StatusBadRequest, "supplier_tax_id and supplier_country are required")
		return
	}

	if req.BuyerTaxID == "" || req.BuyerCountry == "" {
		respondError(w, http.StatusBadRequest, "buyer_tax_id and buyer_country are required")
		return
	}

	if req.FundingDate == "" {
		respondError(w, http.StatusBadRequest, "funding_date is required")
		return
	}

	// Get funder ID from context
	funderID := middleware.GetFunderID(r.Context())

	result, err := h.invoiceService.RegisterInvoiceRaw(r.Context(), &req, funderID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to register invoice")
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

// Unregister handles DELETE /api/v1/invoices/{hash}
func (h *InvoiceHandler) Unregister(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	if hash == "" {
		respondError(w, http.StatusBadRequest, "Hash is required")
		return
	}

	// Get funder ID from context
	funderID := middleware.GetFunderID(r.Context())
	if funderID == nil {
		respondError(w, http.StatusForbidden, "Funder identification required")
		return
	}

	// Allow unregistration within 24 hours (configurable)
	maxAgeHours := 24

	err := h.invoiceService.UnregisterInvoice(r.Context(), hash, *funderID, maxAgeHours)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "Invoice not found")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			respondError(w, http.StatusForbidden, "You can only unregister invoices you registered")
			return
		}
		if errors.Is(err, service.ErrUnregisterWindowExpired) {
			respondError(w, http.StatusGone, "Unregister window has expired")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to unregister invoice")
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// GetFunderIDFromContext extracts funder ID from context (helper)
func GetFunderIDFromContext(r *http.Request) *uuid.UUID {
	return middleware.GetFunderID(r.Context())
}
