package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/regstrava/regstrava/internal/domain"
)

// FunderRepository handles funder persistence
type FunderRepository struct {
	db *sql.DB
}

// NewFunderRepository creates a new funder repository
func NewFunderRepository(databaseURL string) (*FunderRepository, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return &FunderRepository{db: db}, nil
}

// Close closes the database connection
func (r *FunderRepository) Close() error {
	return r.db.Close()
}

// FindByID finds a funder by ID
func (r *FunderRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Funder, error) {
	query := `
		SELECT id, name, api_key_hash, oauth_client_id, oauth_secret_hash,
		       track_fundings, rate_limit_daily, rate_limit_monthly, created_at, is_active
		FROM funders
		WHERE id = $1
	`

	var funder domain.Funder
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&funder.ID,
		&funder.Name,
		&funder.APIKeyHash,
		&funder.OAuthClientID,
		&funder.OAuthSecretHash,
		&funder.TrackFundings,
		&funder.RateLimitDaily,
		&funder.RateLimitMonthly,
		&funder.CreatedAt,
		&funder.IsActive,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find funder: %w", err)
	}

	return &funder, nil
}

// FindByOAuthClientID finds a funder by OAuth client ID
func (r *FunderRepository) FindByOAuthClientID(ctx context.Context, clientID string) (*domain.Funder, error) {
	query := `
		SELECT id, name, api_key_hash, oauth_client_id, oauth_secret_hash,
		       track_fundings, rate_limit_daily, rate_limit_monthly, created_at, is_active
		FROM funders
		WHERE oauth_client_id = $1
	`

	var funder domain.Funder
	err := r.db.QueryRowContext(ctx, query, clientID).Scan(
		&funder.ID,
		&funder.Name,
		&funder.APIKeyHash,
		&funder.OAuthClientID,
		&funder.OAuthSecretHash,
		&funder.TrackFundings,
		&funder.RateLimitDaily,
		&funder.RateLimitMonthly,
		&funder.CreatedAt,
		&funder.IsActive,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find funder: %w", err)
	}

	return &funder, nil
}

// FindAll returns all funders
func (r *FunderRepository) FindAll(ctx context.Context) ([]*domain.Funder, error) {
	query := `
		SELECT id, name, api_key_hash, oauth_client_id, oauth_secret_hash,
		       track_fundings, rate_limit_daily, rate_limit_monthly, created_at, is_active
		FROM funders
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query funders: %w", err)
	}
	defer rows.Close()

	var funders []*domain.Funder
	for rows.Next() {
		var funder domain.Funder
		err := rows.Scan(
			&funder.ID,
			&funder.Name,
			&funder.APIKeyHash,
			&funder.OAuthClientID,
			&funder.OAuthSecretHash,
			&funder.TrackFundings,
			&funder.RateLimitDaily,
			&funder.RateLimitMonthly,
			&funder.CreatedAt,
			&funder.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan funder: %w", err)
		}
		funders = append(funders, &funder)
	}

	return funders, nil
}

// Create creates a new funder
func (r *FunderRepository) Create(ctx context.Context, funder *domain.Funder) error {
	query := `
		INSERT INTO funders (id, name, api_key_hash, oauth_client_id, oauth_secret_hash,
		                     track_fundings, rate_limit_daily, rate_limit_monthly, created_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		funder.ID,
		funder.Name,
		funder.APIKeyHash,
		funder.OAuthClientID,
		funder.OAuthSecretHash,
		funder.TrackFundings,
		funder.RateLimitDaily,
		funder.RateLimitMonthly,
		funder.CreatedAt,
		funder.IsActive,
	)

	if err != nil {
		return fmt.Errorf("failed to create funder: %w", err)
	}

	return nil
}

// Update updates a funder
func (r *FunderRepository) Update(ctx context.Context, funder *domain.Funder) error {
	query := `
		UPDATE funders
		SET name = $2, track_fundings = $3, rate_limit_daily = $4,
		    rate_limit_monthly = $5, is_active = $6
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		funder.ID,
		funder.Name,
		funder.TrackFundings,
		funder.RateLimitDaily,
		funder.RateLimitMonthly,
		funder.IsActive,
	)

	if err != nil {
		return fmt.Errorf("failed to update funder: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("funder not found")
	}

	return nil
}

// Delete deletes a funder by ID
func (r *FunderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM funders WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete funder: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("funder not found")
	}

	return nil
}
