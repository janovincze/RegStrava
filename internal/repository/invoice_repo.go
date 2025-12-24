package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/regstrava/regstrava/internal/domain"
)

// InvoiceRepository handles invoice hash persistence
type InvoiceRepository struct {
	db *sql.DB
}

// NewInvoiceRepository creates a new invoice repository
func NewInvoiceRepository(databaseURL string) (*InvoiceRepository, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return &InvoiceRepository{db: db}, nil
}

// Close closes the database connection
func (r *InvoiceRepository) Close() error {
	return r.db.Close()
}

// FindByHash finds an invoice hash by its hash value
func (r *InvoiceRepository) FindByHash(ctx context.Context, hashValue string) (*domain.InvoiceHash, error) {
	query := `
		SELECT id, hash_value, hash_level, COALESCE(document_type, 'INV'), funded_at, funder_id, expires_at, created_at
		FROM invoice_hashes
		WHERE hash_value = $1
	`

	var invoiceHash domain.InvoiceHash
	err := r.db.QueryRowContext(ctx, query, hashValue).Scan(
		&invoiceHash.ID,
		&invoiceHash.HashValue,
		&invoiceHash.HashLevel,
		&invoiceHash.DocumentType,
		&invoiceHash.FundedAt,
		&invoiceHash.FunderID,
		&invoiceHash.ExpiresAt,
		&invoiceHash.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find invoice hash: %w", err)
	}

	return &invoiceHash, nil
}

// Create creates a new invoice hash
func (r *InvoiceRepository) Create(ctx context.Context, invoiceHash *domain.InvoiceHash) error {
	query := `
		INSERT INTO invoice_hashes (id, hash_value, hash_level, document_type, funded_at, funder_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	docType := invoiceHash.DocumentType
	if docType == "" {
		docType = domain.DefaultDocumentType
	}

	_, err := r.db.ExecContext(ctx, query,
		invoiceHash.ID,
		invoiceHash.HashValue,
		invoiceHash.HashLevel,
		docType,
		invoiceHash.FundedAt,
		invoiceHash.FunderID,
		invoiceHash.ExpiresAt,
		invoiceHash.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create invoice hash: %w", err)
	}

	return nil
}

// Delete deletes an invoice hash by ID
func (r *InvoiceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM invoice_hashes WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete invoice hash: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("invoice hash not found")
	}

	return nil
}

// DeleteByFunderID deletes all hashes for a given funder (for account cleanup)
func (r *InvoiceRepository) DeleteByFunderID(ctx context.Context, funderID uuid.UUID) (int64, error) {
	query := `DELETE FROM invoice_hashes WHERE funder_id = $1`

	result, err := r.db.ExecContext(ctx, query, funderID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete invoice hashes: %w", err)
	}

	return result.RowsAffected()
}

// DeleteExpired deletes all expired invoice hashes
func (r *InvoiceRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM invoice_hashes WHERE expires_at IS NOT NULL AND expires_at < NOW()`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired invoice hashes: %w", err)
	}

	return result.RowsAffected()
}

// Count returns the total count of invoice hashes
func (r *InvoiceRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invoice_hashes").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count invoice hashes: %w", err)
	}
	return count, nil
}
