package domain

import (
	"time"

	"github.com/google/uuid"
)

// HashLevel represents the level of detail in an invoice hash
type HashLevel int

const (
	// New hash level structure (L0-L3)
	HashLevelParty    HashLevel = 0 // L0: party only (buyer OR supplier: tax_id + country)
	HashLevelDocType  HashLevel = 1 // L1: doc_type + supplier (tax_id + country) + buyer (tax_id + country)
	HashLevelDocument HashLevel = 2 // L2: L1 + document_id
	HashLevelFull     HashLevel = 3 // L3: L2 + amount + currency
)

// HashLevelName returns the human-readable name for a hash level
func (h HashLevel) String() string {
	switch h {
	case HashLevelParty:
		return "L0_party"
	case HashLevelDocType:
		return "L1_doc_type"
	case HashLevelDocument:
		return "L2_document"
	case HashLevelFull:
		return "L3_full"
	default:
		return "unknown"
	}
}

// InvoiceHash represents a stored invoice hash in the registry
type InvoiceHash struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	HashValue    string     `json:"hash_value" db:"hash_value"`
	HashLevel    HashLevel  `json:"hash_level" db:"hash_level"`
	DocumentType string     `json:"document_type" db:"document_type"`
	FundedAt     time.Time  `json:"funded_at" db:"funded_at"`
	FunderID     *uuid.UUID `json:"funder_id,omitempty" db:"funder_id"` // NULL if funder didn't consent
	ExpiresAt    *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// InvoiceCheckRequest represents a request to check if an invoice is funded
type InvoiceCheckRequest struct {
	Hashes []string `json:"hashes" validate:"required,min=1,max=4"`
}

// InvoiceCheckRawRequest represents a request with raw invoice data for server-side hashing
type InvoiceCheckRawRequest struct {
	DocumentType   string   `json:"document_type,omitempty"`   // Document type code (default: INV)
	DocumentID     string   `json:"document_id" validate:"required"` // Invoice/document number
	SupplierTaxID  string   `json:"supplier_tax_id" validate:"required"`
	SupplierCountry string  `json:"supplier_country" validate:"required"`
	BuyerTaxID     string   `json:"buyer_tax_id" validate:"required"`
	BuyerCountry   string   `json:"buyer_country" validate:"required"`
	Amount         *float64 `json:"amount,omitempty"`
	Currency       string   `json:"currency,omitempty"`
	// Deprecated fields (kept for backward compatibility)
	InvoiceNumber string `json:"invoice_number,omitempty"` // Deprecated: use document_id
	IssuerTaxID   string `json:"issuer_tax_id,omitempty"`  // Deprecated: use supplier_tax_id
	IssuerCountry string `json:"issuer_country,omitempty"` // Deprecated: use supplier_country
	InvoiceDate   string `json:"invoice_date,omitempty"`   // Deprecated: no longer used
}

// InvoiceCheckResponse represents the response for an invoice check
type InvoiceCheckResponse struct {
	Found         bool                    `json:"found"`
	MatchedLevels []string                `json:"matched_levels,omitempty"`
	Details       map[string]MatchDetail  `json:"details,omitempty"`
	// Deprecated fields (kept for backward compatibility)
	Funded       bool       `json:"funded"`        // Deprecated: use found
	MatchedLevel *HashLevel `json:"matched_level,omitempty"` // Deprecated: use matched_levels
	FundedAt     *time.Time `json:"funded_at,omitempty"`
}

// MatchDetail provides details about a specific level match
type MatchDetail struct {
	Status       string     `json:"status"` // "checked" or "registered"
	FirstSeen    time.Time  `json:"first_seen"`
	RegisteredAt *time.Time `json:"registered_at,omitempty"`
}

// InvoiceRegisterRequest represents a request to register an invoice as funded
type InvoiceRegisterRequest struct {
	Hashes        []string `json:"hashes" validate:"required,min=1,max=4"`
	DocumentType  string   `json:"document_type,omitempty"` // Document type code (default: INV)
	FundingDate   string   `json:"funding_date" validate:"required"`
	TrackFunder   bool     `json:"track_funder"`
	ExpiresInDays *int     `json:"expires_in_days,omitempty"`
}

// InvoiceRegisterRawRequest represents a request with raw invoice data for registration
type InvoiceRegisterRawRequest struct {
	DocumentType    string   `json:"document_type,omitempty"`   // Document type code (default: INV)
	DocumentID      string   `json:"document_id" validate:"required"` // Invoice/document number
	SupplierTaxID   string   `json:"supplier_tax_id" validate:"required"`
	SupplierCountry string   `json:"supplier_country" validate:"required"`
	BuyerTaxID      string   `json:"buyer_tax_id" validate:"required"`
	BuyerCountry    string   `json:"buyer_country" validate:"required"`
	Amount          *float64 `json:"amount,omitempty"`
	Currency        string   `json:"currency,omitempty"`
	FundingDate     string   `json:"funding_date" validate:"required"`
	TrackFunder     bool     `json:"track_funder"`
	ExpiresInDays   *int     `json:"expires_in_days,omitempty"`
	// Deprecated fields (kept for backward compatibility)
	InvoiceNumber string `json:"invoice_number,omitempty"` // Deprecated: use document_id
	IssuerTaxID   string `json:"issuer_tax_id,omitempty"`  // Deprecated: use supplier_tax_id
	IssuerCountry string `json:"issuer_country,omitempty"` // Deprecated: use supplier_country
	InvoiceDate   string `json:"invoice_date,omitempty"`   // Deprecated: no longer used
}

// InvoiceRegisterResponse represents the response after registering an invoice
type InvoiceRegisterResponse struct {
	Success      bool      `json:"success"`
	RegisteredAt time.Time `json:"registered_at"`
	HashLevels   []int     `json:"hash_levels"`
}
