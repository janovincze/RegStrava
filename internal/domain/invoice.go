package domain

import (
	"time"

	"github.com/google/uuid"
)

// HashLevel represents the level of detail in an invoice hash
type HashLevel int

const (
	HashLevelBasic    HashLevel = 1 // invoice_number + issuer_tax_id
	HashLevelStandard HashLevel = 2 // + amount + currency
	HashLevelDated    HashLevel = 3 // + invoice_date
	HashLevelFull     HashLevel = 4 // + buyer_tax_id
)

// InvoiceHash represents a stored invoice hash in the registry
type InvoiceHash struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	HashValue string     `json:"hash_value" db:"hash_value"`
	HashLevel HashLevel  `json:"hash_level" db:"hash_level"`
	FundedAt  time.Time  `json:"funded_at" db:"funded_at"`
	FunderID  *uuid.UUID `json:"funder_id,omitempty" db:"funder_id"` // NULL if funder didn't consent
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// InvoiceCheckRequest represents a request to check if an invoice is funded
type InvoiceCheckRequest struct {
	Hashes []string `json:"hashes" validate:"required,min=1,max=4"`
}

// InvoiceCheckRawRequest represents a request with raw invoice data for server-side hashing
type InvoiceCheckRawRequest struct {
	InvoiceNumber string   `json:"invoice_number" validate:"required"`
	IssuerTaxID   string   `json:"issuer_tax_id" validate:"required"`
	Amount        *float64 `json:"amount,omitempty"`
	Currency      string   `json:"currency,omitempty"`
	InvoiceDate   string   `json:"invoice_date,omitempty"`
	BuyerTaxID    string   `json:"buyer_tax_id,omitempty"`
}

// InvoiceCheckResponse represents the response for an invoice check
type InvoiceCheckResponse struct {
	Funded       bool       `json:"funded"`
	MatchedLevel *HashLevel `json:"matched_level,omitempty"`
	FundedAt     *time.Time `json:"funded_at,omitempty"`
}

// InvoiceRegisterRequest represents a request to register an invoice as funded
type InvoiceRegisterRequest struct {
	Hashes        []string `json:"hashes" validate:"required,min=1,max=4"`
	FundingDate   string   `json:"funding_date" validate:"required"`
	TrackFunder   bool     `json:"track_funder"`
	ExpiresInDays *int     `json:"expires_in_days,omitempty"`
}

// InvoiceRegisterRawRequest represents a request with raw invoice data for registration
type InvoiceRegisterRawRequest struct {
	InvoiceNumber string   `json:"invoice_number" validate:"required"`
	IssuerTaxID   string   `json:"issuer_tax_id" validate:"required"`
	Amount        *float64 `json:"amount,omitempty"`
	Currency      string   `json:"currency,omitempty"`
	InvoiceDate   string   `json:"invoice_date,omitempty"`
	BuyerTaxID    string   `json:"buyer_tax_id,omitempty"`
	FundingDate   string   `json:"funding_date" validate:"required"`
	TrackFunder   bool     `json:"track_funder"`
	ExpiresInDays *int     `json:"expires_in_days,omitempty"`
}

// InvoiceRegisterResponse represents the response after registering an invoice
type InvoiceRegisterResponse struct {
	Success      bool      `json:"success"`
	RegisteredAt time.Time `json:"registered_at"`
	HashLevels   []int     `json:"hash_levels"`
}
