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
// New structure:
// L1: doc_type + supplier (tax_id + country) + buyer (tax_id + country)
// L2: L1 + document_id
// L3: L2 + amount + currency
func (s *HashService) GenerateHashes(req *domain.InvoiceCheckRawRequest) []string {
	hashes := make([]string, 0, 3)

	// Handle backward compatibility with deprecated fields
	documentType := req.DocumentType
	if documentType == "" {
		documentType = domain.DefaultDocumentType
	}
	documentType = s.NormalizeDocumentType(documentType)

	documentID := req.DocumentID
	if documentID == "" {
		documentID = req.InvoiceNumber // Backward compatibility
	}
	documentID = s.NormalizeInvoiceNumber(documentID)

	supplierTaxID := req.SupplierTaxID
	if supplierTaxID == "" {
		supplierTaxID = req.IssuerTaxID // Backward compatibility
	}
	supplierTaxID = s.NormalizeTaxID(supplierTaxID)

	supplierCountry := req.SupplierCountry
	if supplierCountry == "" {
		supplierCountry = req.IssuerCountry // Backward compatibility
	}
	supplierCountry = s.NormalizeCountry(supplierCountry)

	buyerTaxID := s.NormalizeTaxID(req.BuyerTaxID)
	buyerCountry := s.NormalizeCountry(req.BuyerCountry)

	// L1: doc_type + supplier (tax_id + country) + buyer (tax_id + country)
	l1Data := fmt.Sprintf("%s|%s|%s|%s|%s", documentType, supplierTaxID, supplierCountry, buyerTaxID, buyerCountry)
	hashes = append(hashes, s.Hash(l1Data))

	// L2: L1 + document_id
	if documentID != "" {
		l2Data := fmt.Sprintf("%s|%s", l1Data, documentID)
		hashes = append(hashes, s.Hash(l2Data))

		// L3: L2 + amount + currency
		if req.Amount != nil && req.Currency != "" {
			amount := s.NormalizeAmount(*req.Amount)
			currency := s.NormalizeCurrency(req.Currency)
			l3Data := fmt.Sprintf("%s|%s|%s", l2Data, amount, currency)
			hashes = append(hashes, s.Hash(l3Data))
		}
	}

	return hashes
}

// NormalizeDocumentType normalizes a document type code: uppercase
func (s *HashService) NormalizeDocumentType(docType string) string {
	return strings.ToUpper(strings.TrimSpace(docType))
}

// GenerateHashesForRegister generates hashes for registration from raw request
func (s *HashService) GenerateHashesForRegister(req *domain.InvoiceRegisterRawRequest) []string {
	checkReq := &domain.InvoiceCheckRawRequest{
		DocumentType:    req.DocumentType,
		DocumentID:      req.DocumentID,
		SupplierTaxID:   req.SupplierTaxID,
		SupplierCountry: req.SupplierCountry,
		BuyerTaxID:      req.BuyerTaxID,
		BuyerCountry:    req.BuyerCountry,
		Amount:          req.Amount,
		Currency:        req.Currency,
		// Backward compatibility
		InvoiceNumber: req.InvoiceNumber,
		IssuerTaxID:   req.IssuerTaxID,
		IssuerCountry: req.IssuerCountry,
	}
	return s.GenerateHashes(checkReq)
}

// DetermineHashLevel determines the hash level based on index (0-based)
// New levels: 1=DocType, 2=Document, 3=Full
func (s *HashService) DetermineHashLevel(index int) domain.HashLevel {
	switch index {
	case 0:
		return domain.HashLevelDocType  // L1
	case 1:
		return domain.HashLevelDocument // L2
	case 2:
		return domain.HashLevelFull     // L3
	default:
		return domain.HashLevelDocType
	}
}

// GeneratePartyHash generates a hash for a party (buyer or supplier)
// L0: tax_id + country
func (s *HashService) GeneratePartyHash(taxID, country string) string {
	normalizedTaxID := s.NormalizeTaxID(taxID)
	normalizedCountry := s.NormalizeCountry(country)
	data := fmt.Sprintf("%s|%s", normalizedTaxID, normalizedCountry)
	return s.Hash(data)
}

// GenerateAllHashes generates all hash levels including party hashes
// Returns a map of level name to hash value
func (s *HashService) GenerateAllHashes(req *domain.InvoiceCheckRawRequest) map[string]string {
	result := make(map[string]string)

	// Handle backward compatibility with deprecated fields
	supplierTaxID := req.SupplierTaxID
	if supplierTaxID == "" {
		supplierTaxID = req.IssuerTaxID
	}
	supplierCountry := req.SupplierCountry
	if supplierCountry == "" {
		supplierCountry = req.IssuerCountry
	}

	// L0: Party hashes (separate for buyer and supplier)
	if supplierTaxID != "" && supplierCountry != "" {
		result["L0_supplier"] = s.GeneratePartyHash(supplierTaxID, supplierCountry)
	}
	if req.BuyerTaxID != "" && req.BuyerCountry != "" {
		result["L0_buyer"] = s.GeneratePartyHash(req.BuyerTaxID, req.BuyerCountry)
	}

	// L1-L3: Document hashes
	docHashes := s.GenerateHashes(req)
	levels := []string{"L1", "L2", "L3"}
	for i, hash := range docHashes {
		if i < len(levels) {
			result[levels[i]] = hash
		}
	}

	return result
}
