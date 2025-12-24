package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/regstrava/regstrava/internal/domain"
	"github.com/regstrava/regstrava/internal/repository"
)

// PartyService handles party-level (L0) operations
type PartyService struct {
	partyRepo   *repository.PartyRepository
	hashService *HashService
}

// NewPartyService creates a new party service
func NewPartyService(partyRepo *repository.PartyRepository, hashService *HashService) *PartyService {
	return &PartyService{
		partyRepo:   partyRepo,
		hashService: hashService,
	}
}

// CheckParty checks if a party (buyer/supplier) exists in the system
func (s *PartyService) CheckParty(ctx context.Context, req *domain.PartyCheckRequest, funderID uuid.UUID, trackFunder bool) (*domain.PartyCheckResponse, error) {
	hashValue := s.hashService.GeneratePartyHash(req.TaxID, req.Country)

	// Find existing party hash
	partyHash, err := s.partyRepo.FindByHash(ctx, hashValue, req.PartyType)
	if err != nil {
		return nil, err
	}

	response := &domain.PartyCheckResponse{
		PartyType: req.PartyType,
	}

	if partyHash == nil {
		// Party not found, create new record
		response.Found = false
		response.Status = "new"

		var funderIDPtr *uuid.UUID
		if trackFunder {
			funderIDPtr = &funderID
		}

		newPartyHash := &domain.PartyHash{
			ID:             uuid.New(),
			HashValue:      hashValue,
			PartyType:      req.PartyType,
			FirstCheckedAt: time.Now(),
			LastCheckedAt:  time.Now(),
			CheckCount:     1,
			RegisterCount:  0,
			FirstCheckerID: funderIDPtr,
			CreatedAt:      time.Now(),
		}

		if err := s.partyRepo.Create(ctx, newPartyHash); err != nil {
			return nil, err
		}

		return response, nil
	}

	// Party found
	response.Found = true
	response.FirstSeen = &partyHash.FirstCheckedAt
	response.LastChecked = &partyHash.LastCheckedAt
	response.LastRegistered = partyHash.LastRegisteredAt

	if partyHash.RegisterCount > 0 {
		response.Status = "registered"
	} else {
		response.Status = "checked"
	}

	// Check if other funders have interacted
	if partyHash.FirstCheckerID != nil && *partyHash.FirstCheckerID != funderID {
		response.CheckedByOthers = true
	}
	if partyHash.FirstRegistererID != nil && *partyHash.FirstRegistererID != funderID {
		response.RegisteredByOthers = true
	}

	// Update check count
	if err := s.partyRepo.UpdateCheck(ctx, partyHash.ID); err != nil {
		return nil, err
	}

	return response, nil
}

// RegisterParty registers a party in the system
func (s *PartyService) RegisterParty(ctx context.Context, req *domain.PartyRegisterRequest, funderID uuid.UUID) (*domain.PartyRegisterResponse, error) {
	hashValue := s.hashService.GeneratePartyHash(req.TaxID, req.Country)

	// Find existing party hash
	partyHash, err := s.partyRepo.FindByHash(ctx, hashValue, req.PartyType)
	if err != nil {
		return nil, err
	}

	var funderIDPtr *uuid.UUID
	if req.TrackFunder {
		funderIDPtr = &funderID
	}

	now := time.Now()
	response := &domain.PartyRegisterResponse{
		Success:      true,
		RegisteredAt: now,
	}

	if partyHash == nil {
		// Create new party hash with registration
		response.IsNew = true

		newPartyHash := &domain.PartyHash{
			ID:                uuid.New(),
			HashValue:         hashValue,
			PartyType:         req.PartyType,
			FirstCheckedAt:    now,
			FirstRegisteredAt: &now,
			LastCheckedAt:     now,
			LastRegisteredAt:  &now,
			CheckCount:        1,
			RegisterCount:     1,
			FirstCheckerID:    funderIDPtr,
			FirstRegistererID: funderIDPtr,
			CreatedAt:         now,
		}

		if err := s.partyRepo.Create(ctx, newPartyHash); err != nil {
			return nil, err
		}
	} else {
		// Update existing party hash
		response.IsNew = false

		if err := s.partyRepo.UpdateRegister(ctx, partyHash.ID, funderIDPtr); err != nil {
			return nil, err
		}
	}

	return response, nil
}

// QueryPartyHistory queries the history of a party within a lookback period
func (s *PartyService) QueryPartyHistory(ctx context.Context, req *domain.PartyHistoryRequest, funderID uuid.UUID, maxLookbackDays int) (*domain.PartyHistoryResponse, error) {
	hashValue := s.hashService.GeneratePartyHash(req.TaxID, req.Country)

	lookbackDays := maxLookbackDays
	if req.LookbackDays != nil && *req.LookbackDays < maxLookbackDays {
		lookbackDays = *req.LookbackDays
	}

	// Find party with recent activity
	partyHash, err := s.partyRepo.FindRecentActivity(ctx, hashValue, req.PartyType, lookbackDays, &funderID)
	if err != nil {
		return nil, err
	}

	response := &domain.PartyHistoryResponse{
		PartyType: req.PartyType,
	}

	if partyHash == nil {
		response.Found = false
		return response, nil
	}

	response.Found = true
	response.FirstCheckedAt = &partyHash.FirstCheckedAt
	response.LastCheckedAt = &partyHash.LastCheckedAt
	response.CheckCount = partyHash.CheckCount
	response.RegisterCount = partyHash.RegisterCount

	// Count activity from other funders
	since := time.Now().AddDate(0, 0, -lookbackDays)
	otherChecks, otherRegisters, err := s.partyRepo.CountOtherFunderActivity(ctx, hashValue, req.PartyType, funderID, since)
	if err != nil {
		return nil, err
	}

	response.OtherFundersChecked = otherChecks
	response.OtherFundersRegistered = otherRegisters

	return response, nil
}
