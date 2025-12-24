package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/regstrava/regstrava/internal/domain"
	"github.com/regstrava/regstrava/internal/repository"
)

// InvoiceService handles invoice-related business logic
type InvoiceService struct {
	repo        *repository.InvoiceRepository
	partyRepo   *repository.PartyRepository
	hashService *HashService
}

// NewInvoiceService creates a new invoice service
func NewInvoiceService(repo *repository.InvoiceRepository, partyRepo *repository.PartyRepository, hashService *HashService) *InvoiceService {
	return &InvoiceService{
		repo:        repo,
		partyRepo:   partyRepo,
		hashService: hashService,
	}
}

// CheckInvoice checks if any of the provided hashes are already funded
// Returns all matching levels instead of just the first one
func (s *InvoiceService) CheckInvoice(ctx context.Context, hashes []string) (*domain.InvoiceCheckResponse, error) {
	matchedLevels := make([]string, 0)
	details := make(map[string]domain.MatchDetail)
	var firstFundedAt *time.Time
	var firstMatchedLevel *domain.HashLevel

	levelNames := []string{"L1", "L2", "L3"}

	for i, hash := range hashes {
		invoiceHash, err := s.repo.FindByHash(ctx, hash)
		if err != nil {
			return nil, err
		}

		if invoiceHash != nil {
			// Check if expired
			if invoiceHash.ExpiresAt != nil && invoiceHash.ExpiresAt.Before(time.Now()) {
				continue // Skip expired hashes
			}

			level := s.hashService.DetermineHashLevel(i)
			levelName := levelNames[i]
			if i >= len(levelNames) {
				levelName = level.String()
			}

			matchedLevels = append(matchedLevels, levelName)
			details[levelName] = domain.MatchDetail{
				Status:       "registered",
				FirstSeen:    invoiceHash.CreatedAt,
				RegisteredAt: &invoiceHash.FundedAt,
			}

			// Keep track of first match for backward compatibility
			if firstFundedAt == nil {
				firstFundedAt = &invoiceHash.FundedAt
				firstMatchedLevel = &level
			}
		}
	}

	found := len(matchedLevels) > 0

	return &domain.InvoiceCheckResponse{
		Found:         found,
		MatchedLevels: matchedLevels,
		Details:       details,
		// Backward compatibility fields
		Funded:       found,
		MatchedLevel: firstMatchedLevel,
		FundedAt:     firstFundedAt,
	}, nil
}

// CheckInvoiceRaw checks invoice using raw data (server-side hashing)
func (s *InvoiceService) CheckInvoiceRaw(ctx context.Context, req *domain.InvoiceCheckRawRequest) (*domain.InvoiceCheckResponse, error) {
	hashes := s.hashService.GenerateHashes(req)
	return s.CheckInvoice(ctx, hashes)
}

// CheckInvoiceRawWithParty checks invoice including party-level (L0) hashes
func (s *InvoiceService) CheckInvoiceRawWithParty(ctx context.Context, req *domain.InvoiceCheckRawRequest) (*domain.InvoiceCheckResponse, error) {
	// First check document hashes (L1-L3)
	response, err := s.CheckInvoiceRaw(ctx, req)
	if err != nil {
		return nil, err
	}

	// Now check party hashes (L0) if party repository is available
	if s.partyRepo == nil {
		return response, nil
	}

	// Handle backward compatibility with deprecated fields
	supplierTaxID := req.SupplierTaxID
	if supplierTaxID == "" {
		supplierTaxID = req.IssuerTaxID
	}
	supplierCountry := req.SupplierCountry
	if supplierCountry == "" {
		supplierCountry = req.IssuerCountry
	}

	// Check supplier party hash
	if supplierTaxID != "" && supplierCountry != "" {
		supplierHash := s.hashService.GeneratePartyHash(supplierTaxID, supplierCountry)
		partyHash, err := s.partyRepo.FindByHash(ctx, supplierHash, domain.PartyTypeSupplier)
		if err != nil {
			return nil, err
		}

		if partyHash != nil {
			response.MatchedLevels = append([]string{"L0_supplier"}, response.MatchedLevels...)
			status := "checked"
			if partyHash.RegisterCount > 0 {
				status = "registered"
			}
			if response.Details == nil {
				response.Details = make(map[string]domain.MatchDetail)
			}
			response.Details["L0_supplier"] = domain.MatchDetail{
				Status:       status,
				FirstSeen:    partyHash.FirstCheckedAt,
				RegisteredAt: partyHash.FirstRegisteredAt,
			}
			response.Found = true
			response.Funded = true
		}
	}

	// Check buyer party hash
	if req.BuyerTaxID != "" && req.BuyerCountry != "" {
		buyerHash := s.hashService.GeneratePartyHash(req.BuyerTaxID, req.BuyerCountry)
		partyHash, err := s.partyRepo.FindByHash(ctx, buyerHash, domain.PartyTypeBuyer)
		if err != nil {
			return nil, err
		}

		if partyHash != nil {
			// Insert after L0_supplier if exists, otherwise at beginning
			insertPos := 0
			for i, level := range response.MatchedLevels {
				if level == "L0_supplier" {
					insertPos = i + 1
					break
				}
			}
			newLevels := make([]string, 0, len(response.MatchedLevels)+1)
			newLevels = append(newLevels, response.MatchedLevels[:insertPos]...)
			newLevels = append(newLevels, "L0_buyer")
			newLevels = append(newLevels, response.MatchedLevels[insertPos:]...)
			response.MatchedLevels = newLevels

			status := "checked"
			if partyHash.RegisterCount > 0 {
				status = "registered"
			}
			if response.Details == nil {
				response.Details = make(map[string]domain.MatchDetail)
			}
			response.Details["L0_buyer"] = domain.MatchDetail{
				Status:       status,
				FirstSeen:    partyHash.FirstCheckedAt,
				RegisteredAt: partyHash.FirstRegisteredAt,
			}
			response.Found = true
			response.Funded = true
		}
	}

	return response, nil
}

// RegisterInvoice registers an invoice as funded
func (s *InvoiceService) RegisterInvoice(ctx context.Context, req *domain.InvoiceRegisterRequest, funderID *uuid.UUID) (*domain.InvoiceRegisterResponse, error) {
	fundingDate, err := time.Parse("2006-01-02", req.FundingDate)
	if err != nil {
		fundingDate = time.Now()
	}

	var expiresAt *time.Time
	if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
		expires := time.Now().AddDate(0, 0, *req.ExpiresInDays)
		expiresAt = &expires
	}

	// Determine funder ID to store (only if tracking consented)
	var storedFunderID *uuid.UUID
	if req.TrackFunder && funderID != nil {
		storedFunderID = funderID
	}

	// Get document type (default to INV if not specified)
	documentType := req.DocumentType
	if documentType == "" {
		documentType = domain.DefaultDocumentType
	}

	registeredLevels := make([]int, 0, len(req.Hashes))

	for i, hash := range req.Hashes {
		// Check if already exists
		existing, err := s.repo.FindByHash(ctx, hash)
		if err != nil {
			return nil, err
		}

		if existing != nil {
			// Already registered, skip
			continue
		}

		level := s.hashService.DetermineHashLevel(i)
		invoiceHash := &domain.InvoiceHash{
			ID:           uuid.New(),
			HashValue:    hash,
			HashLevel:    level,
			DocumentType: documentType,
			FundedAt:     fundingDate,
			FunderID:     storedFunderID,
			ExpiresAt:    expiresAt,
			CreatedAt:    time.Now(),
		}

		if err := s.repo.Create(ctx, invoiceHash); err != nil {
			return nil, err
		}

		registeredLevels = append(registeredLevels, int(level))
	}

	return &domain.InvoiceRegisterResponse{
		Success:      true,
		RegisteredAt: time.Now(),
		HashLevels:   registeredLevels,
	}, nil
}

// RegisterInvoiceRaw registers invoice using raw data (server-side hashing)
func (s *InvoiceService) RegisterInvoiceRaw(ctx context.Context, req *domain.InvoiceRegisterRawRequest, funderID *uuid.UUID) (*domain.InvoiceRegisterResponse, error) {
	hashes := s.hashService.GenerateHashesForRegister(req)

	// Get document type (default to INV if not specified)
	documentType := req.DocumentType
	if documentType == "" {
		documentType = domain.DefaultDocumentType
	}

	registerReq := &domain.InvoiceRegisterRequest{
		Hashes:        hashes,
		DocumentType:  documentType,
		FundingDate:   req.FundingDate,
		TrackFunder:   req.TrackFunder,
		ExpiresInDays: req.ExpiresInDays,
	}

	return s.RegisterInvoice(ctx, registerReq, funderID)
}

// UnregisterInvoice removes an invoice from the registry
func (s *InvoiceService) UnregisterInvoice(ctx context.Context, hash string, funderID uuid.UUID, maxAgeHours int) error {
	invoiceHash, err := s.repo.FindByHash(ctx, hash)
	if err != nil {
		return err
	}

	if invoiceHash == nil {
		return ErrNotFound
	}

	// Check if the funder owns this registration
	if invoiceHash.FunderID == nil || *invoiceHash.FunderID != funderID {
		return ErrForbidden
	}

	// Check if within allowed time window
	maxAge := time.Duration(maxAgeHours) * time.Hour
	if time.Since(invoiceHash.CreatedAt) > maxAge {
		return ErrUnregisterWindowExpired
	}

	return s.repo.Delete(ctx, invoiceHash.ID)
}
