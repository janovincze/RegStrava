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
	hashService *HashService
}

// NewInvoiceService creates a new invoice service
func NewInvoiceService(repo *repository.InvoiceRepository, hashService *HashService) *InvoiceService {
	return &InvoiceService{
		repo:        repo,
		hashService: hashService,
	}
}

// CheckInvoice checks if any of the provided hashes are already funded
func (s *InvoiceService) CheckInvoice(ctx context.Context, hashes []string) (*domain.InvoiceCheckResponse, error) {
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
			return &domain.InvoiceCheckResponse{
				Funded:       true,
				MatchedLevel: &level,
				FundedAt:     &invoiceHash.FundedAt,
			}, nil
		}
	}

	return &domain.InvoiceCheckResponse{
		Funded: false,
	}, nil
}

// CheckInvoiceRaw checks invoice using raw data (server-side hashing)
func (s *InvoiceService) CheckInvoiceRaw(ctx context.Context, req *domain.InvoiceCheckRawRequest) (*domain.InvoiceCheckResponse, error) {
	hashes := s.hashService.GenerateHashes(req)
	return s.CheckInvoice(ctx, hashes)
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
			ID:        uuid.New(),
			HashValue: hash,
			HashLevel: level,
			FundedAt:  fundingDate,
			FunderID:  storedFunderID,
			ExpiresAt: expiresAt,
			CreatedAt: time.Now(),
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

	registerReq := &domain.InvoiceRegisterRequest{
		Hashes:        hashes,
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
