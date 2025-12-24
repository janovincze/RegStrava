package domain

import "time"

// DocumentType represents a configurable document type for factoring
type DocumentType struct {
	Code        string    `json:"code" db:"code"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// DefaultDocumentType is the default document type code
const DefaultDocumentType = "INV"

// DocumentTypeListResponse represents the response for listing document types
type DocumentTypeListResponse struct {
	DocumentTypes []DocumentType `json:"document_types"`
}
