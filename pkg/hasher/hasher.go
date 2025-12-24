// Package hasher provides public hashing utilities for the RegStrava SDK
package hasher

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// Hasher provides invoice hashing functionality
type Hasher struct {
	hmacKey []byte
}

// New creates a new Hasher with the given HMAC key
func New(hmacKey string) *Hasher {
	return &Hasher{
		hmacKey: []byte(hmacKey),
	}
}

// DefaultDocumentType is the default document type code
const DefaultDocumentType = "INV"

// InvoiceData represents invoice data for hashing (new structure)
type InvoiceData struct {
	DocumentType    string   // Document type code (default: INV)
	DocumentID      string   // Invoice/document number
	SupplierTaxID   string   // Supplier's tax ID
	SupplierCountry string   // Supplier's country code (ISO 3166-1 alpha-2)
	BuyerTaxID      string   // Buyer's tax ID
	BuyerCountry    string   // Buyer's country code (ISO 3166-1 alpha-2)
	Amount          *float64 // Optional: Invoice amount
	Currency        string   // Optional: Currency code (ISO 4217)
	// Deprecated fields (for backward compatibility)
	InvoiceNumber string // Deprecated: use DocumentID
	IssuerTaxID   string // Deprecated: use SupplierTaxID
	IssuerCountry string // Deprecated: use SupplierCountry
	InvoiceDate   string // Deprecated: no longer used
}

// HashResult contains all generated hashes with their levels
type HashResult struct {
	// New hash levels (L0-L3)
	L0Supplier string `json:"l0_supplier,omitempty"` // Supplier: tax_id + country
	L0Buyer    string `json:"l0_buyer,omitempty"`    // Buyer: tax_id + country
	L1DocType  string `json:"l1_doc_type"`           // doc_type + supplier + buyer
	L2Document string `json:"l2_document,omitempty"` // L1 + document_id
	L3Full     string `json:"l3_full,omitempty"`     // L2 + amount + currency
	// Deprecated fields (for backward compatibility)
	L1Basic    string `json:"l1_basic,omitempty"`    // Deprecated
	L2Standard string `json:"l2_standard,omitempty"` // Deprecated
	L3Dated    string `json:"l3_dated,omitempty"`    // Deprecated
	L4Full     string `json:"l4_full,omitempty"`     // Deprecated
}

// nonAlphanumericRegex matches non-alphanumeric characters
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

// NormalizeTaxID normalizes a tax ID: removes non-alphanumeric chars, uppercase
func NormalizeTaxID(taxID string) string {
	cleaned := nonAlphanumericRegex.ReplaceAllString(taxID, "")
	return strings.ToUpper(cleaned)
}

// NormalizeInvoiceNumber normalizes an invoice/document number: trim whitespace, uppercase
func NormalizeInvoiceNumber(invoiceNumber string) string {
	trimmed := strings.TrimSpace(invoiceNumber)
	return strings.ToUpper(trimmed)
}

// NormalizeAmount normalizes an amount: 2 decimal places, no thousands separator
func NormalizeAmount(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}

// NormalizeCurrency normalizes a currency: ISO 4217 uppercase
func NormalizeCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

// NormalizeCountry normalizes a country code: ISO 3166-1 alpha-2 uppercase
func NormalizeCountry(country string) string {
	return strings.ToUpper(strings.TrimSpace(country))
}

// NormalizeDocumentType normalizes a document type code: uppercase
func NormalizeDocumentType(docType string) string {
	return strings.ToUpper(strings.TrimSpace(docType))
}

// NormalizeDate normalizes a date (expects YYYY-MM-DD format)
// Deprecated: no longer used in new hash levels
func NormalizeDate(date string) string {
	return strings.TrimSpace(date)
}

// Hash creates an HMAC-SHA256 hash of the given data
func (h *Hasher) Hash(data string) string {
	mac := hmac.New(sha256.New, h.hmacKey)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// GeneratePartyHash generates a hash for a party (buyer or supplier)
// L0: tax_id + country
func (h *Hasher) GeneratePartyHash(taxID, country string) string {
	normalizedTaxID := NormalizeTaxID(taxID)
	normalizedCountry := NormalizeCountry(country)
	data := fmt.Sprintf("%s|%s", normalizedTaxID, normalizedCountry)
	return h.Hash(data)
}

// GenerateHashes generates all possible hash levels from invoice data
// New structure:
// L0: Party hashes (buyer/supplier separately)
// L1: doc_type + supplier (tax_id + country) + buyer (tax_id + country)
// L2: L1 + document_id
// L3: L2 + amount + currency
func (h *Hasher) GenerateHashes(data *InvoiceData) *HashResult {
	result := &HashResult{}

	// Handle backward compatibility with deprecated fields
	documentType := data.DocumentType
	if documentType == "" {
		documentType = DefaultDocumentType
	}
	documentType = NormalizeDocumentType(documentType)

	documentID := data.DocumentID
	if documentID == "" {
		documentID = data.InvoiceNumber // Backward compatibility
	}
	documentID = NormalizeInvoiceNumber(documentID)

	supplierTaxID := data.SupplierTaxID
	if supplierTaxID == "" {
		supplierTaxID = data.IssuerTaxID // Backward compatibility
	}
	supplierTaxID = NormalizeTaxID(supplierTaxID)

	supplierCountry := data.SupplierCountry
	if supplierCountry == "" {
		supplierCountry = data.IssuerCountry // Backward compatibility
	}
	supplierCountry = NormalizeCountry(supplierCountry)

	buyerTaxID := NormalizeTaxID(data.BuyerTaxID)
	buyerCountry := NormalizeCountry(data.BuyerCountry)

	// L0: Party hashes
	if supplierTaxID != "" && supplierCountry != "" {
		result.L0Supplier = h.GeneratePartyHash(supplierTaxID, supplierCountry)
	}
	if buyerTaxID != "" && buyerCountry != "" {
		result.L0Buyer = h.GeneratePartyHash(buyerTaxID, buyerCountry)
	}

	// L1: doc_type + supplier (tax_id + country) + buyer (tax_id + country)
	if supplierTaxID != "" && supplierCountry != "" && buyerTaxID != "" && buyerCountry != "" {
		l1Data := fmt.Sprintf("%s|%s|%s|%s|%s", documentType, supplierTaxID, supplierCountry, buyerTaxID, buyerCountry)
		result.L1DocType = h.Hash(l1Data)

		// L2: L1 + document_id
		if documentID != "" {
			l2Data := fmt.Sprintf("%s|%s", l1Data, documentID)
			result.L2Document = h.Hash(l2Data)

			// L3: L2 + amount + currency
			if data.Amount != nil && data.Currency != "" {
				amount := NormalizeAmount(*data.Amount)
				currency := NormalizeCurrency(data.Currency)
				l3Data := fmt.Sprintf("%s|%s|%s", l2Data, amount, currency)
				result.L3Full = h.Hash(l3Data)
			}
		}
	}

	return result
}

// ToSlice converts HashResult to a slice of non-empty document hashes (L1-L3)
// Does not include party hashes (L0) as they are typically handled separately
func (r *HashResult) ToSlice() []string {
	hashes := make([]string, 0, 3)

	if r.L1DocType != "" {
		hashes = append(hashes, r.L1DocType)
	}
	if r.L2Document != "" {
		hashes = append(hashes, r.L2Document)
	}
	if r.L3Full != "" {
		hashes = append(hashes, r.L3Full)
	}

	return hashes
}

// ToAllSlice converts HashResult to a slice including all hashes (L0-L3)
func (r *HashResult) ToAllSlice() []string {
	hashes := make([]string, 0, 5)

	if r.L0Supplier != "" {
		hashes = append(hashes, r.L0Supplier)
	}
	if r.L0Buyer != "" {
		hashes = append(hashes, r.L0Buyer)
	}
	if r.L1DocType != "" {
		hashes = append(hashes, r.L1DocType)
	}
	if r.L2Document != "" {
		hashes = append(hashes, r.L2Document)
	}
	if r.L3Full != "" {
		hashes = append(hashes, r.L3Full)
	}

	return hashes
}
