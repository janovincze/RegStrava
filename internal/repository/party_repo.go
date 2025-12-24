package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/regstrava/regstrava/internal/domain"
)

// PartyRepository handles party hash persistence
type PartyRepository struct {
	db *sql.DB
}

// NewPartyRepository creates a new party repository with a shared database connection
func NewPartyRepository(db *sql.DB) *PartyRepository {
	return &PartyRepository{db: db}
}

// FindByHash finds a party hash by its hash value and party type
func (r *PartyRepository) FindByHash(ctx context.Context, hashValue string, partyType domain.PartyType) (*domain.PartyHash, error) {
	query := `
		SELECT id, hash_value, party_type, first_checked_at, first_registered_at,
		       last_checked_at, last_registered_at, check_count, register_count,
		       first_checker_id, first_registerer_id, created_at
		FROM party_hashes
		WHERE hash_value = $1 AND party_type = $2
	`

	var partyHash domain.PartyHash
	err := r.db.QueryRowContext(ctx, query, hashValue, partyType).Scan(
		&partyHash.ID,
		&partyHash.HashValue,
		&partyHash.PartyType,
		&partyHash.FirstCheckedAt,
		&partyHash.FirstRegisteredAt,
		&partyHash.LastCheckedAt,
		&partyHash.LastRegisteredAt,
		&partyHash.CheckCount,
		&partyHash.RegisterCount,
		&partyHash.FirstCheckerID,
		&partyHash.FirstRegistererID,
		&partyHash.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find party hash: %w", err)
	}

	return &partyHash, nil
}

// Create creates a new party hash
func (r *PartyRepository) Create(ctx context.Context, partyHash *domain.PartyHash) error {
	query := `
		INSERT INTO party_hashes (id, hash_value, party_type, first_checked_at, first_registered_at,
		                          last_checked_at, last_registered_at, check_count, register_count,
		                          first_checker_id, first_registerer_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
		partyHash.ID,
		partyHash.HashValue,
		partyHash.PartyType,
		partyHash.FirstCheckedAt,
		partyHash.FirstRegisteredAt,
		partyHash.LastCheckedAt,
		partyHash.LastRegisteredAt,
		partyHash.CheckCount,
		partyHash.RegisterCount,
		partyHash.FirstCheckerID,
		partyHash.FirstRegistererID,
		partyHash.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create party hash: %w", err)
	}

	return nil
}

// UpdateCheck updates the party hash when it's checked
func (r *PartyRepository) UpdateCheck(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE party_hashes
		SET last_checked_at = NOW(), check_count = check_count + 1
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to update party hash check: %w", err)
	}

	return nil
}

// UpdateRegister updates the party hash when it's registered
func (r *PartyRepository) UpdateRegister(ctx context.Context, id uuid.UUID, funderID *uuid.UUID) error {
	query := `
		UPDATE party_hashes
		SET last_registered_at = NOW(),
		    register_count = register_count + 1,
		    first_registered_at = COALESCE(first_registered_at, NOW()),
		    first_registerer_id = COALESCE(first_registerer_id, $2)
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, funderID)
	if err != nil {
		return fmt.Errorf("failed to update party hash register: %w", err)
	}

	return nil
}

// FindRecentActivity finds party activity within a lookback period
func (r *PartyRepository) FindRecentActivity(ctx context.Context, hashValue string, partyType domain.PartyType, lookbackDays int, excludeFunderID *uuid.UUID) (*domain.PartyHash, error) {
	query := `
		SELECT id, hash_value, party_type, first_checked_at, first_registered_at,
		       last_checked_at, last_registered_at, check_count, register_count,
		       first_checker_id, first_registerer_id, created_at
		FROM party_hashes
		WHERE hash_value = $1
		  AND party_type = $2
		  AND (last_checked_at >= NOW() - INTERVAL '1 day' * $3
		       OR last_registered_at >= NOW() - INTERVAL '1 day' * $3)
	`

	var partyHash domain.PartyHash
	err := r.db.QueryRowContext(ctx, query, hashValue, partyType, lookbackDays).Scan(
		&partyHash.ID,
		&partyHash.HashValue,
		&partyHash.PartyType,
		&partyHash.FirstCheckedAt,
		&partyHash.FirstRegisteredAt,
		&partyHash.LastCheckedAt,
		&partyHash.LastRegisteredAt,
		&partyHash.CheckCount,
		&partyHash.RegisterCount,
		&partyHash.FirstCheckerID,
		&partyHash.FirstRegistererID,
		&partyHash.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find recent party activity: %w", err)
	}

	return &partyHash, nil
}

// CountOtherFunderActivity counts activity from other funders
func (r *PartyRepository) CountOtherFunderActivity(ctx context.Context, hashValue string, partyType domain.PartyType, excludeFunderID uuid.UUID, since time.Time) (checks int, registers int, err error) {
	// This would require a more complex schema with activity logging
	// For now, we'll return the total counts from the party hash
	partyHash, err := r.FindByHash(ctx, hashValue, partyType)
	if err != nil {
		return 0, 0, err
	}
	if partyHash == nil {
		return 0, 0, nil
	}

	// Approximate: if the first checker/registerer is not the current funder, count as "others"
	if partyHash.FirstCheckerID != nil && *partyHash.FirstCheckerID != excludeFunderID {
		checks = partyHash.CheckCount
	}
	if partyHash.FirstRegistererID != nil && *partyHash.FirstRegistererID != excludeFunderID {
		registers = partyHash.RegisterCount
	}

	return checks, registers, nil
}

// GetDB returns the underlying database connection for sharing
func (r *PartyRepository) GetDB() *sql.DB {
	return r.db
}
