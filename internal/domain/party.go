package domain

import (
	"time"

	"github.com/google/uuid"
)

// PartyType represents the type of party (buyer or supplier)
type PartyType string

const (
	PartyTypeBuyer    PartyType = "buyer"
	PartyTypeSupplier PartyType = "supplier"
)

// PartyHash represents a stored party hash in the registry (L0 level)
type PartyHash struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	HashValue         string     `json:"hash_value" db:"hash_value"`
	PartyType         PartyType  `json:"party_type" db:"party_type"`
	FirstCheckedAt    time.Time  `json:"first_checked_at" db:"first_checked_at"`
	FirstRegisteredAt *time.Time `json:"first_registered_at,omitempty" db:"first_registered_at"`
	LastCheckedAt     time.Time  `json:"last_checked_at" db:"last_checked_at"`
	LastRegisteredAt  *time.Time `json:"last_registered_at,omitempty" db:"last_registered_at"`
	CheckCount        int        `json:"check_count" db:"check_count"`
	RegisterCount     int        `json:"register_count" db:"register_count"`
	FirstCheckerID    *uuid.UUID `json:"first_checker_id,omitempty" db:"first_checker_id"`
	FirstRegistererID *uuid.UUID `json:"first_registerer_id,omitempty" db:"first_registerer_id"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// PartyCheckRequest represents a request to check a party (buyer or supplier)
type PartyCheckRequest struct {
	TaxID     string    `json:"tax_id" validate:"required"`
	Country   string    `json:"country" validate:"required"`
	PartyType PartyType `json:"party_type" validate:"required,oneof=buyer supplier"`
}

// PartyCheckResponse represents the response for a party check
type PartyCheckResponse struct {
	Found              bool       `json:"found"`
	PartyType          PartyType  `json:"party_type"`
	Status             string     `json:"status"` // "new", "checked", "registered"
	FirstSeen          *time.Time `json:"first_seen,omitempty"`
	LastChecked        *time.Time `json:"last_checked,omitempty"`
	LastRegistered     *time.Time `json:"last_registered,omitempty"`
	CheckedByOthers    bool       `json:"checked_by_others"`    // Someone else checked this party
	RegisteredByOthers bool       `json:"registered_by_others"` // Someone else registered this party
}

// PartyHistoryRequest represents a request to query party history
type PartyHistoryRequest struct {
	TaxID        string    `json:"tax_id" validate:"required"`
	Country      string    `json:"country" validate:"required"`
	PartyType    PartyType `json:"party_type" validate:"required,oneof=buyer supplier"`
	LookbackDays *int      `json:"lookback_days,omitempty"` // Override default lookback
}

// PartyHistoryResponse represents the response for a party history query
type PartyHistoryResponse struct {
	Found          bool       `json:"found"`
	PartyType      PartyType  `json:"party_type"`
	FirstCheckedAt *time.Time `json:"first_checked_at,omitempty"`
	LastCheckedAt  *time.Time `json:"last_checked_at,omitempty"`
	CheckCount     int        `json:"check_count"`
	RegisterCount  int        `json:"register_count"`
	// These are only shown if the checking funder has appropriate permissions
	OtherFundersChecked    int `json:"other_funders_checked,omitempty"`
	OtherFundersRegistered int `json:"other_funders_registered,omitempty"`
}

// PartyRegisterRequest represents a request to register a party
type PartyRegisterRequest struct {
	TaxID       string    `json:"tax_id" validate:"required"`
	Country     string    `json:"country" validate:"required"`
	PartyType   PartyType `json:"party_type" validate:"required,oneof=buyer supplier"`
	TrackFunder bool      `json:"track_funder"`
}

// PartyRegisterResponse represents the response after registering a party
type PartyRegisterResponse struct {
	Success      bool      `json:"success"`
	RegisteredAt time.Time `json:"registered_at"`
	IsNew        bool      `json:"is_new"` // True if this was the first registration
}
