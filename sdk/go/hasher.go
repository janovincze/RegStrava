package regstrava

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// Hasher provides invoice hashing functionality for client-side hashing
type Hasher struct {
	hmacKey []byte
}

// NewHasher creates a new Hasher with the given HMAC key
func NewHasher(hmacKey string) *Hasher {
	return &Hasher{
		hmacKey: []byte(hmacKey),
	}
}

// InvoiceData represents invoice data for hashing
type InvoiceData struct {
	InvoiceNumber string
	IssuerTaxID   string
	Amount        *float64
	Currency      string
	InvoiceDate   string
	BuyerTaxID    string
}

// HashResult contains all generated hashes with their levels
type HashResult struct {
	L1Basic    string `json:"l1_basic"`              // invoice_number + issuer_tax_id
	L2Standard string `json:"l2_standard,omitempty"` // + amount + currency
	L3Dated    string `json:"l3_dated,omitempty"`    // + invoice_date
	L4Full     string `json:"l4_full,omitempty"`     // + buyer_tax_id
}

// nonAlphanumericRegex matches non-alphanumeric characters
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

// NormalizeTaxID normalizes a tax ID: removes non-alphanumeric chars, uppercase
func NormalizeTaxID(taxID string) string {
	cleaned := nonAlphanumericRegex.ReplaceAllString(taxID, "")
	return strings.ToUpper(cleaned)
}

// NormalizeInvoiceNumber normalizes an invoice number: trim whitespace, uppercase
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

// NormalizeDate normalizes a date (expects YYYY-MM-DD format)
func NormalizeDate(date string) string {
	return strings.TrimSpace(date)
}

// Hash creates an HMAC-SHA256 hash of the given data
func (h *Hasher) Hash(data string) string {
	mac := hmac.New(sha256.New, h.hmacKey)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// GenerateHashes generates all possible hash levels from invoice data
func (h *Hasher) GenerateHashes(data *InvoiceData) *HashResult {
	result := &HashResult{}

	invoiceNumber := NormalizeInvoiceNumber(data.InvoiceNumber)
	issuerTaxID := NormalizeTaxID(data.IssuerTaxID)

	// L1: Basic - invoice_number + issuer_tax_id
	l1Data := fmt.Sprintf("%s|%s", invoiceNumber, issuerTaxID)
	result.L1Basic = h.Hash(l1Data)

	// L2: Standard - L1 + amount + currency
	if data.Amount != nil && data.Currency != "" {
		amount := NormalizeAmount(*data.Amount)
		currency := NormalizeCurrency(data.Currency)
		l2Data := fmt.Sprintf("%s|%s|%s", l1Data, amount, currency)
		result.L2Standard = h.Hash(l2Data)

		// L3: Dated - L2 + invoice_date
		if data.InvoiceDate != "" {
			invoiceDate := NormalizeDate(data.InvoiceDate)
			l3Data := fmt.Sprintf("%s|%s", l2Data, invoiceDate)
			result.L3Dated = h.Hash(l3Data)

			// L4: Full - L3 + buyer_tax_id
			if data.BuyerTaxID != "" {
				buyerTaxID := NormalizeTaxID(data.BuyerTaxID)
				l4Data := fmt.Sprintf("%s|%s", l3Data, buyerTaxID)
				result.L4Full = h.Hash(l4Data)
			}
		}
	}

	return result
}

// ToSlice converts HashResult to a slice of non-empty hashes
func (r *HashResult) ToSlice() []string {
	hashes := make([]string, 0, 4)

	if r.L1Basic != "" {
		hashes = append(hashes, r.L1Basic)
	}
	if r.L2Standard != "" {
		hashes = append(hashes, r.L2Standard)
	}
	if r.L3Dated != "" {
		hashes = append(hashes, r.L3Dated)
	}
	if r.L4Full != "" {
		hashes = append(hashes, r.L4Full)
	}

	return hashes
}
