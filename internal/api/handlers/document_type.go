package handlers

import (
	"net/http"

	"github.com/regstrava/regstrava/internal/repository"
)

// DocumentTypeHandler handles document type-related HTTP requests
type DocumentTypeHandler struct {
	docTypeRepo *repository.DocumentTypeRepository
}

// NewDocumentTypeHandler creates a new document type handler
func NewDocumentTypeHandler(docTypeRepo *repository.DocumentTypeRepository) *DocumentTypeHandler {
	return &DocumentTypeHandler{
		docTypeRepo: docTypeRepo,
	}
}

// List handles GET /api/v1/document-types
func (h *DocumentTypeHandler) List(w http.ResponseWriter, r *http.Request) {
	docTypes, err := h.docTypeRepo.ListActive(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list document types")
		return
	}

	response := map[string]interface{}{
		"document_types": docTypes,
	}

	respondJSON(w, http.StatusOK, response)
}
