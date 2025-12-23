package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/regstrava/regstrava/internal/domain"
)

// HashService handles invoice hashing and normalization
type HashService struct {
	hmacKey []byte
}

// NewHashService creates a new hash service with the given HMAC key
func NewHashService(hmacKey string) *HashService {
	return &HashService{
		hmacKey: []byte(hmacKey),
	}
}

// nonAlphanumericRegex matches non-alphanumeric characters
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

// NormalizeTaxID normalizes a tax ID: removes non-alphanumeric chars, uppercase
func (s *HashService) NormalizeTaxID(taxID string) string {
	cleaned := nonAlphanumericRegex.ReplaceAllString(taxID, "")
	return strings.ToUpper(cleaned)
}

// NormalizeInvoiceNumber normalizes an invoice number: trim whitespace, uppercase
func (s *HashService) NormalizeInvoiceNumber(invoiceNumber string) string {
	trimmed := strings.TrimSpace(invoiceNumber)
	return strings.ToUpper(trimmed)
}

// NormalizeAmount normalizes an amount: 2 decimal places, no thousands separator
func (s *HashService) NormalizeAmount(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}

// NormalizeCurrency normalizes a currency: ISO 4217 uppercase
func (s *HashService) NormalizeCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

// NormalizeCountry normalizes a country code: ISO 3166-1 alpha-2 uppercase
func (s *HashService) NormalizeCountry(country string) string {
	return strings.ToUpper(strings.TrimSpace(country))
}

// NormalizeDate normalizes a date to ISO 8601 format (YYYY-MM-DD)
// Assumes input is already in YYYY-MM-DD format
func (s *HashService) NormalizeDate(date string) string {
	return strings.TrimSpace(date)
}

// Hash creates an HMAC-SHA256 hash of the given data
func (s *HashService) Hash(data string) string {
	h := hmac.New(sha256.New, s.hmacKey)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateHashes generates all hash levels from raw invoice data
func (s *HashService) GenerateHashes(req *domain.InvoiceCheckRawRequest) []string {
	hashes := make([]string, 0, 4)

	invoiceNumber := s.NormalizeInvoiceNumber(req.InvoiceNumber)
	issuerTaxID := s.NormalizeTaxID(req.IssuerTaxID)
	issuerCountry := s.NormalizeCountry(req.IssuerCountry)

	// L1: Basic - invoice_number + issuer_tax_id + issuer_country
	l1Data := fmt.Sprintf("%s|%s|%s", invoiceNumber, issuerTaxID, issuerCountry)
	hashes = append(hashes, s.Hash(l1Data))

	// L2: Standard - L1 + amount + currency
	if req.Amount != nil && req.Currency != "" {
		amount := s.NormalizeAmount(*req.Amount)
		currency := s.NormalizeCurrency(req.Currency)
		l2Data := fmt.Sprintf("%s|%s|%s", l1Data, amount, currency)
		hashes = append(hashes, s.Hash(l2Data))

		// L3: Dated - L2 + invoice_date
		if req.InvoiceDate != "" {
			invoiceDate := s.NormalizeDate(req.InvoiceDate)
			l3Data := fmt.Sprintf("%s|%s", l2Data, invoiceDate)
			hashes = append(hashes, s.Hash(l3Data))

			// L4: Full - L3 + buyer_tax_id + buyer_country
			if req.BuyerTaxID != "" && req.BuyerCountry != "" {
				buyerTaxID := s.NormalizeTaxID(req.BuyerTaxID)
				buyerCountry := s.NormalizeCountry(req.BuyerCountry)
				l4Data := fmt.Sprintf("%s|%s|%s", l3Data, buyerTaxID, buyerCountry)
				hashes = append(hashes, s.Hash(l4Data))
			}
		}
	}

	return hashes
}

// GenerateHashesForRegister generates hashes for registration from raw request
func (s *HashService) GenerateHashesForRegister(req *domain.InvoiceRegisterRawRequest) []string {
	checkReq := &domain.InvoiceCheckRawRequest{
		InvoiceNumber: req.InvoiceNumber,
		IssuerTaxID:   req.IssuerTaxID,
		IssuerCountry: req.IssuerCountry,
		Amount:        req.Amount,
		Currency:      req.Currency,
		InvoiceDate:   req.InvoiceDate,
		BuyerTaxID:    req.BuyerTaxID,
		BuyerCountry:  req.BuyerCountry,
	}
	return s.GenerateHashes(checkReq)
}

// DetermineHashLevel determines the hash level based on index (0-based)
func (s *HashService) DetermineHashLevel(index int) domain.HashLevel {
	switch index {
	case 0:
		return domain.HashLevelBasic
	case 1:
		return domain.HashLevelStandard
	case 2:
		return domain.HashLevelDated
	case 3:
		return domain.HashLevelFull
	default:
		return domain.HashLevelBasic
	}
}
